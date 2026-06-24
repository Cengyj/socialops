package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/domain"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/Wei-Shaw/socialops/internal/service"
)

type usageLogRepository struct {
	sql sqlQueryer
}

func NewUsageLogRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageLogRepository {
	return &usageLogRepository{sql: sqlDB}
}

func (r *usageLogRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	where, args := socialUsageWhereWithAlias(filters, "stl")

	countQuery := "SELECT COUNT(*) FROM social_task_logs stl" + where
	var total int64
	if err := scanSingleRow(ctx, r.sql, countQuery, args, &total); err != nil {
		return nil, nil, err
	}

	query := `
SELECT stl.id,
       stl.user_id,
       stl.social_account_id,
       ` + socialUsagePlatformColumn("sa") + ` AS platform,
       COALESCE(sa.name, '') AS account_name,
       stl.action AS operation,
       stl.status,
       1 AS quantity,
       ` + socialUsageCostColumn("stl") + ` AS cost,
       stl.charge_status,
       stl.charge_source,
       stl.target,
       stl.content,
       stl.payload,
       stl.template_snapshot,
       stl.result_message,
       stl.proxy_snapshot,
       stl.billing_request_id,
       stl.idempotency_key,
       stl.created_at,
       stl.executed_at AS completed_at
FROM social_task_logs stl
LEFT JOIN social_accounts sa ON sa.id = stl.social_account_id` + where + socialUsageOrderWithAlias(params, "stl") + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	queryArgs := append(append([]any{}, args...), params.Limit(), params.Offset())

	rows, err := r.sql.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.UsageLog, 0, params.Limit())
	for rows.Next() {
		item, err := scanUsageLogRow(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *usageLogRepository) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	where, args := socialUsageWhere(filters)
	query := `
SELECT COUNT(*),
       COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
       ` + socialUsageCostSum("") + `
FROM social_task_logs` + where

	stats := &usagestats.UsageStats{}
	if err := scanSingleRow(
		ctx,
		r.sql,
		query,
		args,
		&stats.TotalOperations,
		&stats.SuccessCount,
		&stats.FailedCount,
		&stats.TotalCharged,
	); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *usageLogRepository) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	now := time.Now().UTC()
	todayStart := utcDayStart(now)
	hourStart := now.Truncate(time.Hour)
	recentStart := now.Add(-5 * time.Minute)
	stats := &usagestats.DashboardStats{
		StatsUpdatedAt: now.Format(time.RFC3339),
		StatsStale:     false,
	}

	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN created_at >= $1 THEN 1 ELSE 0 END), 0)
		   FROM users
		  WHERE deleted_at IS NULL`,
		[]any{todayStart},
		&stats.TotalUsers,
		&stats.TodayNewUsers,
	); err != nil {
		return nil, err
	}

	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN account_status = 'available' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN account_status IN ('invalid', 'not_stored') THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN account_status = 'limited' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN account_status = 'pending_check' THEN 1 ELSE 0 END), 0)
		   FROM social_accounts`,
		nil,
		&stats.TotalAccounts,
		&stats.NormalAccounts,
		&stats.ErrorAccounts,
		&stats.RateLimitAccounts,
		&stats.OverloadAccounts,
	); err != nil {
		return nil, err
	}

	var recentOperations int64
	activity := socialUsageActivityColumn("")
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(DISTINCT CASE WHEN `+activity+` >= $1 AND `+activity+` <= $4 THEN user_id END),
		        COUNT(DISTINCT CASE WHEN `+activity+` >= $2 AND `+activity+` <= $4 THEN user_id END),
		        COUNT(*),
		        `+socialUsageCostSum("")+`,
		        COALESCE(SUM(CASE WHEN `+activity+` >= $1 AND `+activity+` <= $4 THEN 1 ELSE 0 END), 0),
		        `+socialUsageWindowCostSum("", 1, 4)+`,
		        COALESCE(SUM(CASE WHEN `+activity+` >= $3 AND `+activity+` <= $4 THEN 1 ELSE 0 END), 0)
		   FROM social_task_logs
		  WHERE `+socialUsageFinalStatusPredicate("")+``,
		[]any{todayStart, hourStart, recentStart, now},
		&stats.ActiveUsers,
		&stats.HourlyActiveUsers,
		&stats.TotalOperations,
		&stats.TotalCharged,
		&stats.TodayOperations,
		&stats.TodayCharged,
		&recentOperations,
	); err != nil {
		return nil, err
	}

	stats.RecentOperationsPerMinute = recentOperations / 5
	return stats, nil
}

func (r *usageLogRepository) GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	now := time.Now().UTC()
	todayStart := utcDayStart(now)
	recentStart := now.Add(-5 * time.Minute)
	stats := &usagestats.UserDashboardStats{}

	var recentOperations int64
	activity := socialUsageActivityColumn("")
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*),
		        `+socialUsageCostSum("")+`,
		        COALESCE(SUM(CASE WHEN `+activity+` >= $2 AND `+activity+` <= $4 THEN 1 ELSE 0 END), 0),
		        `+socialUsageWindowCostSum("", 2, 4)+`,
		        COALESCE(SUM(CASE WHEN `+activity+` >= $3 AND `+activity+` <= $4 THEN 1 ELSE 0 END), 0)
		   FROM social_task_logs
		  WHERE user_id = $1 AND `+socialUsageFinalStatusPredicate("")+``,
		[]any{userID, todayStart, recentStart, now},
		&stats.TotalOperations,
		&stats.TotalCharged,
		&stats.TodayOperations,
		&stats.TodayCharged,
		&recentOperations,
	); err != nil {
		return nil, err
	}
	stats.RecentOperationsPerMinute = recentOperations / 5

	byPlatform, err := r.getUserDashboardStatsByPlatform(ctx, userID, todayStart, now)
	if err != nil {
		return nil, err
	}
	stats.ByPlatform = byPlatform
	return stats, nil
}

func (r *usageLogRepository) getUserDashboardStatsByPlatform(ctx context.Context, userID int64, todayStart, now time.Time) ([]usagestats.PlatformDashboardStats, error) {
	activity := socialUsageActivityColumn("stl")
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT `+socialUsagePlatformColumn("sa")+`,
		        COUNT(*),
		        `+socialUsageCostSum("stl")+`,
		        COALESCE(SUM(CASE WHEN `+activity+` >= $2 AND `+activity+` <= $3 THEN 1 ELSE 0 END), 0),
		        `+socialUsageWindowCostSum("stl", 2, 3)+`
		   FROM social_task_logs stl
		   LEFT JOIN social_accounts sa ON sa.id = stl.social_account_id
		  WHERE stl.user_id = $1 AND `+socialUsageFinalStatusPredicate("stl")+`
		  GROUP BY `+socialUsagePlatformColumn("sa")+`
		  ORDER BY `+socialUsagePlatformColumn("sa")+` ASC`,
		userID,
		todayStart,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]usagestats.PlatformDashboardStats, 0)
	for rows.Next() {
		var item usagestats.PlatformDashboardStats
		if err := rows.Scan(
			&item.Platform,
			&item.TotalOperations,
			&item.TotalCharged,
			&item.TodayOperations,
			&item.TodayCharged,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *usageLogRepository) GetUsageTrend(ctx context.Context, start, end time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	if !end.After(start) {
		return []usagestats.TrendDataPoint{}, nil
	}

	activity := socialUsageActivityColumn("")
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT `+activity+`, `+socialUsageCostColumn("")+`
		   FROM social_task_logs
		  WHERE `+activity+` >= $1 AND `+activity+` <= $2
		    AND `+socialUsageFinalStatusPredicate("")+`
		  ORDER BY `+activity+` ASC`,
		start.UTC(),
		end.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	byDate := map[string]*usagestats.TrendDataPoint{}
	for rows.Next() {
		var rawActivityAt any
		var cost float64
		if err := rows.Scan(&rawActivityAt, &cost); err != nil {
			return nil, err
		}
		activityAt, err := scanUsageActivityTime(rawActivityAt)
		if err != nil {
			return nil, err
		}
		date := trendBucket(activityAt, granularity)
		point := byDate[date]
		if point == nil {
			point = &usagestats.TrendDataPoint{Date: date}
			byDate[date] = point
		}
		point.Operations++
		point.Charged += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]usagestats.TrendDataPoint, 0, len(byDate))
	for _, point := range byDate {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
	return points, nil
}

func (r *usageLogRepository) GetUserUsageTrend(ctx context.Context, start, end time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	if !end.After(start) {
		return []usagestats.UserUsageTrendPoint{}, nil
	}

	activity := socialUsageActivityColumn("stl")
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT `+activity+`,
		        stl.user_id,
		        COALESCE(u.email, ''),
		        COALESCE(u.username, ''),
		        `+socialUsageCostColumn("stl")+`
		   FROM social_task_logs stl
		   LEFT JOIN users u ON u.id = stl.user_id AND u.deleted_at IS NULL
		  WHERE `+activity+` >= $1 AND `+activity+` <= $2
		    AND `+socialUsageFinalStatusPredicate("stl")+`
		  ORDER BY `+activity+` ASC`,
		start.UTC(),
		end.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	type userTrendKey struct {
		date   string
		userID int64
	}
	type userTrendTotal struct {
		userID     int64
		charged    float64
		operations int64
	}
	byUserDate := map[userTrendKey]*usagestats.UserUsageTrendPoint{}
	byUserTotal := map[int64]*userTrendTotal{}
	for rows.Next() {
		var rawActivityAt any
		var userID int64
		var email string
		var username string
		var cost float64
		if err := rows.Scan(&rawActivityAt, &userID, &email, &username, &cost); err != nil {
			return nil, err
		}
		activityAt, err := scanUsageActivityTime(rawActivityAt)
		if err != nil {
			return nil, err
		}
		date := trendBucket(activityAt, granularity)
		key := userTrendKey{date: date, userID: userID}
		point := byUserDate[key]
		if point == nil {
			point = &usagestats.UserUsageTrendPoint{Date: date, UserID: userID, Email: email, Username: username}
			byUserDate[key] = point
		}
		point.Operations++
		point.Charged += cost

		total := byUserTotal[userID]
		if total == nil {
			total = &userTrendTotal{userID: userID}
			byUserTotal[userID] = total
		}
		total.operations++
		total.charged += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var topUsers map[int64]struct{}
	if limit > 0 && len(byUserTotal) > limit {
		totals := make([]userTrendTotal, 0, len(byUserTotal))
		for _, total := range byUserTotal {
			totals = append(totals, *total)
		}
		sort.Slice(totals, func(i, j int) bool {
			if totals[i].charged != totals[j].charged {
				return totals[i].charged > totals[j].charged
			}
			if totals[i].operations != totals[j].operations {
				return totals[i].operations > totals[j].operations
			}
			return totals[i].userID < totals[j].userID
		})
		topUsers = make(map[int64]struct{}, limit)
		for i := 0; i < limit; i++ {
			topUsers[totals[i].userID] = struct{}{}
		}
	}

	points := make([]usagestats.UserUsageTrendPoint, 0, len(byUserDate))
	for _, point := range byUserDate {
		if topUsers != nil {
			if _, ok := topUsers[point.UserID]; !ok {
				continue
			}
		}
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].Date != points[j].Date {
			return points[i].Date < points[j].Date
		}
		if points[i].Charged != points[j].Charged {
			return points[i].Charged > points[j].Charged
		}
		if points[i].Operations != points[j].Operations {
			return points[i].Operations > points[j].Operations
		}
		return points[i].UserID < points[j].UserID
	})
	return points, nil
}

func (r *usageLogRepository) GetUserUsageTrendByUserID(ctx context.Context, userID int64, start, end time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	if !end.After(start) {
		return []usagestats.TrendDataPoint{}, nil
	}
	filters := usagestats.UsageLogFilters{
		UserID:    userID,
		StartTime: ptrTime(start.UTC()),
		EndTime:   ptrTime(end.UTC()),
	}
	where, args := socialUsageWhere(filters)
	activity := socialUsageActivityColumn("")
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT `+activity+`, `+socialUsageCostColumn("")+`
		   FROM social_task_logs`+where+`
		  ORDER BY `+activity+` ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	byDate := map[string]*usagestats.TrendDataPoint{}
	for rows.Next() {
		var rawActivityAt any
		var cost float64
		if err := rows.Scan(&rawActivityAt, &cost); err != nil {
			return nil, err
		}
		activityAt, err := scanUsageActivityTime(rawActivityAt)
		if err != nil {
			return nil, err
		}
		date := trendBucket(activityAt, granularity)
		point := byDate[date]
		if point == nil {
			point = &usagestats.TrendDataPoint{Date: date}
			byDate[date] = point
		}
		point.Operations++
		point.Charged += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]usagestats.TrendDataPoint, 0, len(byDate))
	for _, point := range byDate {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
	return points, nil
}

func (r *usageLogRepository) GetUserSpendingRanking(ctx context.Context, start, end time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	if !end.After(start) {
		return &usagestats.UserSpendingRankingResponse{Ranking: []usagestats.UserSpendingRankingItem{}}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	resp := &usagestats.UserSpendingRankingResponse{Ranking: []usagestats.UserSpendingRankingItem{}}
	activity := socialUsageActivityColumn("")
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*), `+socialUsageCostSum("")+`
		   FROM social_task_logs
		  WHERE `+activity+` >= $1 AND `+activity+` <= $2
		    AND `+socialUsageFinalStatusPredicate("")+``,
		[]any{start.UTC(), end.UTC()},
		&resp.TotalOperations,
		&resp.TotalCharged,
	); err != nil {
		return nil, err
	}
	activity = socialUsageActivityColumn("stl")
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT stl.user_id,
		        COALESCE(u.email, ''),
		        COUNT(*),
		        `+socialUsageCostSum("stl")+`
		   FROM social_task_logs stl
		   LEFT JOIN users u ON u.id = stl.user_id AND u.deleted_at IS NULL
		  WHERE `+activity+` >= $1 AND `+activity+` <= $2
		    AND `+socialUsageFinalStatusPredicate("stl")+`
		  GROUP BY stl.user_id, COALESCE(u.email, '')
		  HAVING `+socialUsageCostSum("stl")+` > 0
		  ORDER BY `+socialUsageCostSum("stl")+` DESC, COUNT(*) DESC, stl.user_id ASC
		  LIMIT $3`,
		start.UTC(),
		end.UTC(),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item usagestats.UserSpendingRankingItem
		if err := rows.Scan(&item.UserID, &item.Email, &item.Operations, &item.Charged); err != nil {
			return nil, err
		}
		resp.Ranking = append(resp.Ranking, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *usageLogRepository) GetByID(ctx context.Context, id, userID int64) (*service.UsageLog, error) {
	filters := usagestats.UsageLogFilters{UserID: userID}
	where, args := socialUsageWhereWithAlias(filters, "stl")
	args = append(args, id)
	idPredicate := fmt.Sprintf("stl.id = $%d", len(args))
	if where == "" {
		where = " WHERE " + idPredicate
	} else {
		where += " AND " + idPredicate
	}
	query := `
SELECT stl.id,
       stl.user_id,
       stl.social_account_id,
       ` + socialUsagePlatformColumn("sa") + ` AS platform,
       COALESCE(sa.name, '') AS account_name,
       stl.action AS operation,
       stl.status,
       1 AS quantity,
       ` + socialUsageCostColumn("stl") + ` AS cost,
       stl.charge_status,
       stl.charge_source,
       stl.target,
       stl.content,
       stl.payload,
       stl.template_snapshot,
       stl.result_message,
       stl.proxy_snapshot,
       stl.billing_request_id,
       stl.idempotency_key,
       stl.created_at,
       stl.executed_at AS completed_at
FROM social_task_logs stl
LEFT JOIN social_accounts sa ON sa.id = stl.social_account_id` + where

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrUsageLogNotFound
	}
	item, err := scanUsageLogDetailRow(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &item, nil
}

func utcDayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func trendBucket(t time.Time, granularity string) string {
	t = t.UTC()
	switch strings.ToLower(strings.TrimSpace(granularity)) {
	case "hour":
		return t.Format("2006-01-02 15:00")
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}

func socialUsageWhere(filters usagestats.UsageLogFilters) (string, []any) {
	return socialUsageWhereWithAlias(filters, "")
}

func socialUsageWhereWithAlias(filters usagestats.UsageLogFilters, alias string) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	column := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}
	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	if filters.UserID > 0 {
		add(column("user_id")+" = $%d", filters.UserID)
	}
	operation := strings.TrimSpace(filters.Operation)
	if operation != "" {
		add(column("action")+" = $%d", operation)
	}
	if strings.TrimSpace(filters.Status) != "" {
		status := strings.ToLower(strings.TrimSpace(filters.Status))
		if status != service.SocialTaskLogStatusSuccess && status != service.SocialTaskLogStatusFailed {
			clauses = append(clauses, "1 = 0")
		} else {
			add(column("status")+" = $%d", status)
		}
	} else {
		clauses = append(clauses, socialUsageFinalStatusPredicate(alias))
	}
	if platform := strings.TrimSpace(filters.Platform); platform != "" {
		platform = service.NormalizePlanPlatform(platform)
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM social_accounts sa_filter WHERE sa_filter.id = %s AND LOWER(COALESCE(NULLIF(sa_filter.platform_key, ''), sa_filter.platform, '')) = $%d)",
			column("social_account_id"),
			len(args)+1,
		))
		args = append(args, strings.ToLower(platform))
	}
	if accountName := strings.TrimSpace(filters.AccountName); accountName != "" {
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM social_accounts sa_filter WHERE sa_filter.id = %s AND LOWER(COALESCE(sa_filter.name, '')) LIKE $%d)",
			column("social_account_id"),
			len(args)+1,
		))
		args = append(args, "%"+strings.ToLower(accountName)+"%")
	}
	if filters.StartTime != nil {
		add(socialUsageActivityColumn(alias)+" >= $%d", *filters.StartTime)
	}
	if filters.EndTime != nil {
		add(socialUsageActivityColumn(alias)+" <= $%d", *filters.EndTime)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func socialUsageOrder(params pagination.PaginationParams) string {
	return socialUsageOrderWithAlias(params, "")
}

func socialUsageOrderWithAlias(params pagination.PaginationParams, alias string) string {
	direction := "DESC"
	if params.NormalizedSortOrder(pagination.SortOrderDesc) == pagination.SortOrderAsc {
		direction = "ASC"
	}
	idColumn := "id"
	taskColumn := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}
	accountAlias := ""
	if alias != "" {
		idColumn = alias + ".id"
		accountAlias = "sa"
	}
	defaultOrder := " ORDER BY " + socialUsageActivityColumn(alias) + " " + direction + ", " + idColumn + " " + direction
	switch strings.ToLower(strings.TrimSpace(params.SortBy)) {
	case "platform":
		if accountAlias == "" {
			return defaultOrder
		}
		return " ORDER BY " + socialUsagePlatformColumn(accountAlias) + " " + direction + ", " + idColumn + " " + direction
	case "account", "account_name":
		if accountAlias == "" {
			return defaultOrder
		}
		return " ORDER BY COALESCE(" + socialUsageAccountColumn(accountAlias, "name") + ", '') " + direction + ", " + idColumn + " " + direction
	case "status", "result":
		return " ORDER BY " + taskColumn("status") + " " + direction + ", " + idColumn + " " + direction
	case "cost":
		return " ORDER BY cost " + direction + ", " + idColumn + " " + direction
	case "operation":
		return " ORDER BY operation " + direction + ", " + idColumn + " " + direction
	default:
		return defaultOrder
	}
}

func socialUsageActivityColumn(alias string) string {
	if alias == "" {
		return "COALESCE(executed_at, created_at)"
	}
	return "COALESCE(" + alias + ".executed_at, " + alias + ".created_at)"
}

func socialUsageFinalStatusPredicate(alias string) string {
	column := "status"
	if alias != "" {
		column = alias + ".status"
	}
	return column + " IN ('success', 'failed')"
}

func socialUsagePlatformColumn(alias string) string {
	return "COALESCE(NULLIF(" + socialUsageAccountColumn(alias, "platform_key") + ", ''), " + socialUsageAccountColumn(alias, "platform") + ", '')"
}

func socialUsageAccountColumn(alias string, name string) string {
	if alias == "" {
		return name
	}
	return alias + "." + name
}

func socialUsageCostColumn(alias string) string {
	column := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}
	return "CASE WHEN " + column("status") + " = 'success' AND " + column("charge_status") + " = 'charged' THEN COALESCE(" + column("charged_amount") + ", 0) ELSE 0 END"
}

func socialUsageCostSum(alias string) string {
	return "COALESCE(SUM(" + socialUsageCostColumn(alias) + "), 0)"
}

func socialUsageWindowCostSum(alias string, startArg, endArg int) string {
	activity := socialUsageActivityColumn(alias)
	return fmt.Sprintf(
		"COALESCE(SUM(CASE WHEN %s >= $%d AND %s <= $%d THEN %s ELSE 0 END), 0)",
		activity,
		startArg,
		activity,
		endArg,
		socialUsageCostColumn(alias),
	)
}

func scanUsageActivityTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseUsageActivityTimeString(v)
	case []byte:
		return parseUsageActivityTimeString(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported social usage activity time type %T", value)
	}
}

func parseUsageActivityTimeString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05",
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid social usage activity time %q", value)
}

type usageLogScanner interface {
	Scan(dest ...any) error
}

func scanUsageLogRow(row usageLogScanner) (service.UsageLog, error) {
	var item service.UsageLog
	var payloadRaw []byte
	var templateSnapshotRaw []byte
	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.SocialAccountID,
		&item.Platform,
		&item.AccountName,
		&item.Operation,
		&item.Status,
		&item.Quantity,
		&item.Cost,
		&item.ChargeStatus,
		&item.ChargeSource,
		&item.Target,
		&item.Content,
		&payloadRaw,
		&templateSnapshotRaw,
		&item.ResultMessage,
		&item.ProxySnapshot,
		&item.BillingRequestID,
		&item.IdempotencyKey,
		&item.CreatedAt,
		&item.CompletedAt,
	); err != nil {
		return service.UsageLog{}, err
	}
	if payload, err := decodeSocialTaskPayloadJSON(payloadRaw); err != nil {
		return service.UsageLog{}, err
	} else {
		item.Payload = payload
	}
	if snapshot, err := decodeSocialTaskTemplateSnapshotJSON(templateSnapshotRaw); err != nil {
		return service.UsageLog{}, err
	} else {
		item.TemplateSnapshot = snapshot
	}
	return item, nil
}

func scanUsageLogDetailRow(row usageLogScanner) (service.UsageLog, error) {
	var item service.UsageLog
	var payloadRaw []byte
	var templateSnapshotRaw []byte
	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.SocialAccountID,
		&item.Platform,
		&item.AccountName,
		&item.Operation,
		&item.Status,
		&item.Quantity,
		&item.Cost,
		&item.ChargeStatus,
		&item.ChargeSource,
		&item.Target,
		&item.Content,
		&payloadRaw,
		&templateSnapshotRaw,
		&item.ResultMessage,
		&item.ProxySnapshot,
		&item.BillingRequestID,
		&item.IdempotencyKey,
		&item.CreatedAt,
		&item.CompletedAt,
	); err != nil {
		return service.UsageLog{}, err
	}
	if payload, err := decodeSocialTaskPayloadJSON(payloadRaw); err != nil {
		return service.UsageLog{}, err
	} else {
		item.Payload = payload
	}
	if snapshot, err := decodeSocialTaskTemplateSnapshotJSON(templateSnapshotRaw); err != nil {
		return service.UsageLog{}, err
	} else {
		item.TemplateSnapshot = snapshot
	}
	return item, nil
}

func decodeSocialTaskPayloadJSON(raw []byte) (*domain.SocialTaskPayload, error) {
	if len(bytesTrimSpace(raw)) == 0 || string(bytesTrimSpace(raw)) == "null" {
		return nil, nil
	}
	var payload domain.SocialTaskPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload.IsZero() {
		return nil, nil
	}
	return &payload, nil
}

func decodeSocialTaskTemplateSnapshotJSON(raw []byte) (*domain.SocialTaskTemplateSnapshot, error) {
	if len(bytesTrimSpace(raw)) == 0 || string(bytesTrimSpace(raw)) == "null" {
		return nil, nil
	}
	var snapshot domain.SocialTaskTemplateSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.IsZero() {
		return nil, nil
	}
	return &snapshot, nil
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}
