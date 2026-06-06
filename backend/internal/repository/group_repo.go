package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dbent "github.com/Wei-Shaw/socialops/ent"
	dbgroup "github.com/Wei-Shaw/socialops/ent/group"
	"github.com/Wei-Shaw/socialops/ent/userallowedgroup"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type groupRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewGroupRepository(client *dbent.Client, sqlDB *sql.DB) service.GroupRepository {
	return &groupRepository{client: client, sql: sqlDB}
}

func (r *groupRepository) Create(ctx context.Context, groupIn *service.Group) error {
	if groupIn == nil {
		return nil
	}
	builder := r.client.Group.Create().
		SetName(groupIn.Name).
		SetDescription(groupIn.Description).
		SetPlatform(groupPlatformOrDefault(groupIn.Platform)).
		SetRateMultiplier(groupRateOrDefault(groupIn.RateMultiplier)).
		SetIsExclusive(groupIn.IsExclusive).
		SetStatus(groupStatusOrDefault(groupIn.Status)).
		SetSubscriptionType(groupSubscriptionTypeOrDefault(groupIn.SubscriptionType)).
		SetNillableDailyLimitUsd(groupIn.DailyLimitUSD).
		SetNillableWeeklyLimitUsd(groupIn.WeeklyLimitUSD).
		SetNillableMonthlyLimitUsd(groupIn.MonthlyLimitUSD).
		SetDefaultValidityDays(groupValidityOrDefault(groupIn.DefaultValidityDays)).
		SetSortOrder(groupIn.SortOrder).
		SetRpmLimit(groupIn.RPMLimit)

	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrGroupExists)
	}
	applyGroupEntityToService(groupIn, created)
	return nil
}

func (r *groupRepository) GetByID(ctx context.Context, id int64) (*service.Group, error) {
	return r.GetByIDLite(ctx, id)
}

func (r *groupRepository) GetByIDLite(ctx context.Context, id int64) (*service.Group, error) {
	if r.client == nil {
		return r.getByIDLiteSQL(ctx, id)
	}
	m, err := r.client.Group.Query().
		Where(dbgroup.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGroupNotFound, nil)
	}
	return groupEntityToService(m), nil
}

func (r *groupRepository) getByIDLiteSQL(ctx context.Context, id int64) (*service.Group, error) {
	if r.sql == nil {
		return nil, service.ErrGroupNotFound
	}
	var (
		description sql.NullString
		daily       sql.NullFloat64
		weekly      sql.NullFloat64
		monthly     sql.NullFloat64
		out         service.Group
	)
	err := scanSingleRow(ctx, r.sql, `
SELECT id, name, description, platform, rate_multiplier, is_exclusive, status,
       subscription_type, daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
       default_validity_days, sort_order, rpm_limit, created_at, updated_at
FROM groups
WHERE id = $1
`, []any{id},
		&out.ID,
		&out.Name,
		&description,
		&out.Platform,
		&out.RateMultiplier,
		&out.IsExclusive,
		&out.Status,
		&out.SubscriptionType,
		&daily,
		&weekly,
		&monthly,
		&out.DefaultValidityDays,
		&out.SortOrder,
		&out.RPMLimit,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrGroupNotFound
		}
		return nil, err
	}
	if description.Valid {
		out.Description = description.String
	}
	out.DailyLimitUSD = groupNullFloat64Ptr(daily)
	out.WeeklyLimitUSD = groupNullFloat64Ptr(weekly)
	out.MonthlyLimitUSD = groupNullFloat64Ptr(monthly)
	out.Hydrated = true
	return &out, nil
}

func (r *groupRepository) Update(ctx context.Context, groupIn *service.Group) error {
	if groupIn == nil {
		return nil
	}
	builder := r.client.Group.UpdateOneID(groupIn.ID).
		SetName(groupIn.Name).
		SetDescription(groupIn.Description).
		SetPlatform(groupPlatformOrDefault(groupIn.Platform)).
		SetRateMultiplier(groupRateOrDefault(groupIn.RateMultiplier)).
		SetIsExclusive(groupIn.IsExclusive).
		SetStatus(groupStatusOrDefault(groupIn.Status)).
		SetSubscriptionType(groupSubscriptionTypeOrDefault(groupIn.SubscriptionType)).
		SetDefaultValidityDays(groupValidityOrDefault(groupIn.DefaultValidityDays)).
		SetSortOrder(groupIn.SortOrder).
		SetRpmLimit(groupIn.RPMLimit)

	if groupIn.DailyLimitUSD != nil {
		builder.SetDailyLimitUsd(*groupIn.DailyLimitUSD)
	} else {
		builder.ClearDailyLimitUsd()
	}
	if groupIn.WeeklyLimitUSD != nil {
		builder.SetWeeklyLimitUsd(*groupIn.WeeklyLimitUSD)
	} else {
		builder.ClearWeeklyLimitUsd()
	}
	if groupIn.MonthlyLimitUSD != nil {
		builder.SetMonthlyLimitUsd(*groupIn.MonthlyLimitUSD)
	} else {
		builder.ClearMonthlyLimitUsd()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrGroupNotFound, service.ErrGroupExists)
	}
	groupIn.UpdatedAt = updated.UpdatedAt
	groupIn.Hydrated = true
	return nil
}

func (r *groupRepository) Delete(ctx context.Context, id int64) error {
	affected, err := r.client.Group.Delete().Where(dbgroup.IDEQ(id)).Exec(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrGroupNotFound, nil)
	}
	if affected == 0 {
		return service.ErrGroupNotFound
	}
	return nil
}

func (r *groupRepository) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return nil, err
	}

	client := r.client
	opCtx := ctx
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		client = tx.Client()
		opCtx = dbent.NewTxContext(ctx, tx)
	} else if existingTx := dbent.TxFromContext(ctx); existingTx != nil {
		client = existingTx.Client()
	}

	var userIDs []int64
	err = client.UserAllowedGroup.Query().
		Where(userallowedgroup.GroupIDEQ(id)).
		Select(userallowedgroup.FieldUserID).
		Scan(opCtx, &userIDs)
	if err != nil {
		return nil, err
	}
	if _, err := client.UserAllowedGroup.Delete().
		Where(userallowedgroup.GroupIDEQ(id)).
		Exec(opCtx); err != nil {
		return nil, err
	}
	if r.sql != nil {
		if _, err := txAwareSQLExecutor(opCtx, r.sql, r.client).ExecContext(opCtx, "DELETE FROM user_group_rates WHERE group_id = $1", id); err != nil {
			return nil, err
		}
	}
	affected, err := client.Group.Delete().Where(dbgroup.IDEQ(id)).Exec(opCtx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGroupNotFound, nil)
	}
	if affected == 0 {
		return nil, service.ErrGroupNotFound
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	return userIDs, nil
}

func (r *groupRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", "", "", "", nil)
}

func (r *groupRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, subscriptionType, search string, isExclusive *bool) ([]service.Group, *pagination.PaginationResult, error) {
	q := r.client.Group.Query()
	if strings.TrimSpace(platform) != "" {
		q = q.Where(dbgroup.PlatformEQ(strings.TrimSpace(platform)))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where(dbgroup.StatusEQ(strings.TrimSpace(status)))
	}
	if strings.TrimSpace(subscriptionType) != "" {
		q = q.Where(dbgroup.SubscriptionTypeEQ(strings.TrimSpace(subscriptionType)))
	}
	if strings.TrimSpace(search) != "" {
		needle := strings.TrimSpace(search)
		q = q.Where(dbgroup.Or(
			dbgroup.NameContainsFold(needle),
			dbgroup.DescriptionContainsFold(needle),
		))
	}
	if isExclusive != nil {
		q = q.Where(dbgroup.IsExclusiveEQ(*isExclusive))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	groupsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range groupListOrder(params) {
		groupsQuery = groupsQuery.Order(order)
	}
	rows, err := groupsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]service.Group, 0, len(rows))
	for i := range rows {
		out = append(out, *groupEntityToService(rows[i]))
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *groupRepository) ListActive(ctx context.Context) ([]service.Group, error) {
	return r.ListActiveByPlatform(ctx, "")
}

func (r *groupRepository) ListActiveByPlatform(ctx context.Context, platform string) ([]service.Group, error) {
	q := r.client.Group.Query().
		Where(
			dbgroup.StatusEQ(service.StatusActive),
			dbgroup.SubscriptionTypeEQ(service.SubscriptionTypeSubscription),
		)
	if strings.TrimSpace(platform) != "" {
		q = q.Where(dbgroup.PlatformEQ(strings.TrimSpace(platform)))
	}
	rows, err := q.Order(dbent.Asc(dbgroup.FieldSortOrder), dbent.Asc(dbgroup.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.Group, 0, len(rows))
	for i := range rows {
		out = append(out, *groupEntityToService(rows[i]))
	}
	return out, nil
}

func (r *groupRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	return r.client.Group.Query().
		Where(dbgroup.NameEQ(strings.TrimSpace(name))).
		Exist(ctx)
}

func (r *groupRepository) UpdateSortOrders(ctx context.Context, updates []service.GroupSortOrderUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return err
	}
	client := r.client
	opCtx := ctx
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		client = tx.Client()
		opCtx = dbent.NewTxContext(ctx, tx)
	} else if existingTx := dbent.TxFromContext(ctx); existingTx != nil {
		client = existingTx.Client()
	}
	for _, update := range updates {
		if update.ID <= 0 {
			continue
		}
		if _, err := client.Group.Update().
			Where(dbgroup.IDEQ(update.ID)).
			SetSortOrder(update.SortOrder).
			Save(opCtx); err != nil {
			return err
		}
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}

func groupListOrder(params pagination.PaginationParams) []func(*entsql.Selector) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderAsc)

	field := dbgroup.FieldSortOrder
	switch sortBy {
	case "id":
		field = dbgroup.FieldID
	case "name":
		field = dbgroup.FieldName
	case "status":
		field = dbgroup.FieldStatus
	case "platform":
		field = dbgroup.FieldPlatform
	case "subscription_type":
		field = dbgroup.FieldSubscriptionType
	case "created_at":
		field = dbgroup.FieldCreatedAt
	case "updated_at":
		field = dbgroup.FieldUpdatedAt
	case "sort_order", "":
		field = dbgroup.FieldSortOrder
	}

	if sortOrder == pagination.SortOrderDesc {
		return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(dbgroup.FieldID)}
	}
	return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(dbgroup.FieldID)}
}

func groupEntityToService(m *dbent.Group) *service.Group {
	if m == nil {
		return nil
	}
	out := &service.Group{
		ID:                  m.ID,
		Name:                m.Name,
		Description:         groupDerefString(m.Description),
		Platform:            m.Platform,
		RateMultiplier:      m.RateMultiplier,
		IsExclusive:         m.IsExclusive,
		Status:              m.Status,
		Hydrated:            true,
		SubscriptionType:    m.SubscriptionType,
		DailyLimitUSD:       m.DailyLimitUsd,
		WeeklyLimitUSD:      m.WeeklyLimitUsd,
		MonthlyLimitUSD:     m.MonthlyLimitUsd,
		DefaultValidityDays: m.DefaultValidityDays,
		SortOrder:           m.SortOrder,
		RPMLimit:            m.RpmLimit,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
	return out
}

func groupDerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func groupNullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	out := value.Float64
	return &out
}

func applyGroupEntityToService(dst *service.Group, src *dbent.Group) {
	if dst == nil || src == nil {
		return
	}
	*dst = *groupEntityToService(src)
}

func groupPlatformOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "social"
	}
	return strings.TrimSpace(value)
}

func groupStatusOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return service.StatusActive
	}
	return strings.TrimSpace(value)
}

func groupSubscriptionTypeOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return service.SubscriptionTypeSubscription
	}
	return strings.TrimSpace(value)
}

func groupRateOrDefault(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func groupValidityOrDefault(value int) int {
	if value <= 0 {
		return 30
	}
	return value
}
