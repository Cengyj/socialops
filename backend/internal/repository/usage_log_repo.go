package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
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
	where, args := socialUsageWhere(filters)

	countQuery := "SELECT COUNT(*) FROM social_task_logs" + where
	var total int64
	if err := scanSingleRow(ctx, r.sql, countQuery, args, &total); err != nil {
		return nil, nil, err
	}

	query := `
SELECT id,
       user_id,
       NULL AS api_key_id,
       NULL AS group_id,
       action AS operation,
       status,
       1 AS quantity,
       COALESCE(charged_amount, 0) AS cost,
       created_at,
       executed_at AS completed_at
FROM social_task_logs` + where + socialUsageOrder(params) + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
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
       0,
       0,
       0,
       COUNT(*),
       COALESCE(SUM(charged_amount), 0),
       COALESCE(SUM(charged_amount), 0),
       0
FROM social_task_logs` + where

	stats := &usagestats.UsageStats{}
	if err := scanSingleRow(
		ctx,
		r.sql,
		query,
		args,
		&stats.TotalRequests,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheTokens,
		&stats.TotalTokens,
		&stats.TotalCost,
		&stats.TotalActualCost,
		&stats.AverageDurationMs,
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
		        COALESCE(SUM(CASE WHEN status = $1 THEN 1 ELSE 0 END), 0)
		   FROM api_keys
		  WHERE deleted_at IS NULL`,
		[]any{service.StatusActive},
		&stats.TotalAPIKeys,
		&stats.ActiveAPIKeys,
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
		   FROM social_accounts
		  WHERE deleted_at IS NULL`,
		nil,
		&stats.TotalAccounts,
		&stats.NormalAccounts,
		&stats.ErrorAccounts,
		&stats.RateLimitAccounts,
		&stats.OverloadAccounts,
	); err != nil {
		return nil, err
	}

	var recentRequests int64
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(DISTINCT CASE WHEN created_at >= $1 THEN user_id END),
		        COUNT(DISTINCT CASE WHEN created_at >= $2 THEN user_id END),
		        COUNT(*),
		        COALESCE(SUM(COALESCE(charged_amount, 0)), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $1 THEN COALESCE(charged_amount, 0) ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $3 THEN 1 ELSE 0 END), 0)
		   FROM social_task_logs`,
		[]any{todayStart, hourStart, recentStart},
		&stats.ActiveUsers,
		&stats.HourlyActiveUsers,
		&stats.TotalRequests,
		&stats.TotalActualCost,
		&stats.TodayRequests,
		&stats.TodayActualCost,
		&recentRequests,
	); err != nil {
		return nil, err
	}

	stats.TotalTokens = stats.TotalRequests
	stats.TotalCost = stats.TotalActualCost
	stats.TodayTokens = stats.TodayRequests
	stats.TodayCost = stats.TodayActualCost
	stats.Rpm = recentRequests / 5
	stats.Tpm = stats.Rpm
	return stats, nil
}

func (r *usageLogRepository) GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	now := time.Now().UTC()
	todayStart := utcDayStart(now)
	recentStart := now.Add(-5 * time.Minute)
	stats := &usagestats.UserDashboardStats{}

	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN status = $2 THEN 1 ELSE 0 END), 0)
		   FROM api_keys
		  WHERE user_id = $1 AND deleted_at IS NULL`,
		[]any{userID, service.StatusActive},
		&stats.TotalAPIKeys,
		&stats.ActiveAPIKeys,
	); err != nil {
		return nil, err
	}

	var recentRequests int64
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*),
		        COALESCE(SUM(COALESCE(charged_amount, 0)), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $2 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $2 THEN COALESCE(charged_amount, 0) ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN created_at >= $3 THEN 1 ELSE 0 END), 0)
		   FROM social_task_logs
		  WHERE user_id = $1`,
		[]any{userID, todayStart, recentStart},
		&stats.TotalRequests,
		&stats.TotalActualCost,
		&stats.TodayRequests,
		&stats.TodayActualCost,
		&recentRequests,
	); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalRequests
	stats.TotalCost = stats.TotalActualCost
	stats.TodayTokens = stats.TodayRequests
	stats.TodayCost = stats.TodayActualCost
	stats.Rpm = recentRequests / 5
	stats.Tpm = stats.Rpm

	byPlatform, err := r.getUserDashboardStatsByPlatform(ctx, userID, todayStart)
	if err != nil {
		return nil, err
	}
	stats.ByPlatform = byPlatform
	return stats, nil
}

func (r *usageLogRepository) getUserDashboardStatsByPlatform(ctx context.Context, userID int64, todayStart time.Time) ([]usagestats.PlatformDashboardStats, error) {
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT COALESCE(sa.platform, ''),
		        COUNT(*),
		        COALESCE(SUM(COALESCE(stl.charged_amount, 0)), 0),
		        COALESCE(SUM(CASE WHEN stl.created_at >= $2 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN stl.created_at >= $2 THEN COALESCE(stl.charged_amount, 0) ELSE 0 END), 0)
		   FROM social_task_logs stl
		   LEFT JOIN social_accounts sa ON sa.id = stl.social_account_id
		  WHERE stl.user_id = $1
		  GROUP BY COALESCE(sa.platform, '')
		  ORDER BY COALESCE(sa.platform, '') ASC`,
		userID,
		todayStart,
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
			&item.TotalRequests,
			&item.TotalActualCost,
			&item.TodayRequests,
			&item.TodayActualCost,
		); err != nil {
			return nil, err
		}
		item.TotalTokens = item.TotalRequests
		item.TodayTokens = item.TodayRequests
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *usageLogRepository) GetUsageTrend(ctx context.Context, start, end time.Time, granularity string, requestType *int16, stream *bool) ([]usagestats.TrendDataPoint, error) {
	if requestType != nil || stream != nil || !end.After(start) {
		return []usagestats.TrendDataPoint{}, nil
	}

	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT created_at, COALESCE(charged_amount, 0)
		   FROM social_task_logs
		  WHERE created_at >= $1 AND created_at <= $2
		  ORDER BY created_at ASC`,
		start.UTC(),
		end.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	byDate := map[string]*usagestats.TrendDataPoint{}
	for rows.Next() {
		var createdAt time.Time
		var cost float64
		if err := rows.Scan(&createdAt, &cost); err != nil {
			return nil, err
		}
		date := trendBucket(createdAt, granularity)
		point := byDate[date]
		if point == nil {
			point = &usagestats.TrendDataPoint{Date: date}
			byDate[date] = point
		}
		point.Requests++
		point.TotalTokens++
		point.Cost += cost
		point.ActualCost += cost
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

	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT stl.created_at,
		        stl.user_id,
		        COALESCE(u.email, ''),
		        COALESCE(u.username, ''),
		        COALESCE(stl.charged_amount, 0)
		   FROM social_task_logs stl
		   LEFT JOIN users u ON u.id = stl.user_id AND u.deleted_at IS NULL
		  WHERE stl.created_at >= $1 AND stl.created_at <= $2
		  ORDER BY stl.created_at ASC`,
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
	byUserDate := map[userTrendKey]*usagestats.UserUsageTrendPoint{}
	for rows.Next() {
		var createdAt time.Time
		var userID int64
		var email string
		var username string
		var cost float64
		if err := rows.Scan(&createdAt, &userID, &email, &username, &cost); err != nil {
			return nil, err
		}
		date := trendBucket(createdAt, granularity)
		key := userTrendKey{date: date, userID: userID}
		point := byUserDate[key]
		if point == nil {
			point = &usagestats.UserUsageTrendPoint{Date: date, UserID: userID, Email: email, Username: username}
			byUserDate[key] = point
		}
		point.Requests++
		point.Tokens++
		point.Cost += cost
		point.ActualCost += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]usagestats.UserUsageTrendPoint, 0, len(byUserDate))
	for _, point := range byUserDate {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].Date != points[j].Date {
			return points[i].Date < points[j].Date
		}
		if points[i].ActualCost != points[j].ActualCost {
			return points[i].ActualCost > points[j].ActualCost
		}
		if points[i].Requests != points[j].Requests {
			return points[i].Requests > points[j].Requests
		}
		return points[i].UserID < points[j].UserID
	})
	if limit > 0 && len(points) > limit {
		points = points[:limit]
	}
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
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT created_at, COALESCE(charged_amount, 0)
		   FROM social_task_logs`+where+`
		  ORDER BY created_at ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	byDate := map[string]*usagestats.TrendDataPoint{}
	for rows.Next() {
		var createdAt time.Time
		var cost float64
		if err := rows.Scan(&createdAt, &cost); err != nil {
			return nil, err
		}
		date := trendBucket(createdAt, granularity)
		point := byDate[date]
		if point == nil {
			point = &usagestats.TrendDataPoint{Date: date}
			byDate[date] = point
		}
		point.Requests++
		point.TotalTokens++
		point.Cost += cost
		point.ActualCost += cost
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
	if err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT COUNT(*), COALESCE(SUM(COALESCE(charged_amount, 0)), 0)
		   FROM social_task_logs
		  WHERE created_at >= $1 AND created_at <= $2`,
		[]any{start.UTC(), end.UTC()},
		&resp.TotalRequests,
		&resp.TotalActualCost,
	); err != nil {
		return nil, err
	}
	resp.TotalTokens = resp.TotalRequests

	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT stl.user_id,
		        COALESCE(u.email, ''),
		        COUNT(*),
		        COALESCE(SUM(COALESCE(stl.charged_amount, 0)), 0)
		   FROM social_task_logs stl
		   LEFT JOIN users u ON u.id = stl.user_id AND u.deleted_at IS NULL
		  WHERE stl.created_at >= $1 AND stl.created_at <= $2
		  GROUP BY stl.user_id, COALESCE(u.email, '')
		  ORDER BY COALESCE(SUM(COALESCE(stl.charged_amount, 0)), 0) DESC, COUNT(*) DESC, stl.user_id ASC
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
		if err := rows.Scan(&item.UserID, &item.Email, &item.Requests, &item.ActualCost); err != nil {
			return nil, err
		}
		item.Tokens = item.Requests
		resp.Ranking = append(resp.Ranking, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *usageLogRepository) GetByID(ctx context.Context, id, userID int64) (*service.UsageLog, error) {
	filters := usagestats.UsageLogFilters{UserID: userID}
	where, args := socialUsageWhere(filters)
	args = append(args, id)
	idPredicate := fmt.Sprintf("id = $%d", len(args))
	if where == "" {
		where = " WHERE " + idPredicate
	} else {
		where += " AND " + idPredicate
	}
	query := `
SELECT id,
       user_id,
       NULL AS api_key_id,
       NULL AS group_id,
       action AS operation,
       status,
       1 AS quantity,
       COALESCE(charged_amount, 0) AS cost,
       created_at,
       executed_at AS completed_at
FROM social_task_logs` + where

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
	item, err := scanUsageLogRow(rows)
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
	clauses := make([]string, 0)
	args := make([]any, 0)
	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	if filters.UserID > 0 {
		add("user_id = $%d", filters.UserID)
	}
	if filters.APIKeyID > 0 {
		clauses = append(clauses, "1 = 0")
	}
	if filters.AccountID > 0 {
		add("social_account_id = $%d", filters.AccountID)
	}
	if filters.GroupID > 0 {
		clauses = append(clauses, "1 = 0")
	}
	if strings.TrimSpace(filters.Model) != "" {
		add("action = $%d", strings.TrimSpace(filters.Model))
	}
	if filters.RequestType != nil {
		clauses = append(clauses, "1 = 0")
	}
	if filters.Stream != nil {
		clauses = append(clauses, "1 = 0")
	}
	if filters.BillingType != nil {
		clauses = append(clauses, "1 = 0")
	}
	if strings.TrimSpace(filters.BillingMode) != "" {
		clauses = append(clauses, "1 = 0")
	}
	if filters.StartTime != nil {
		add("created_at >= $%d", *filters.StartTime)
	}
	if filters.EndTime != nil {
		add("created_at <= $%d", *filters.EndTime)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func socialUsageOrder(params pagination.PaginationParams) string {
	direction := "DESC"
	if params.NormalizedSortOrder(pagination.SortOrderDesc) == pagination.SortOrderAsc {
		direction = "ASC"
	}
	switch strings.ToLower(strings.TrimSpace(params.SortBy)) {
	case "cost":
		return " ORDER BY cost " + direction + ", id " + direction
	case "operation", "model":
		return " ORDER BY operation " + direction + ", id " + direction
	default:
		return " ORDER BY created_at " + direction + ", id " + direction
	}
}

type usageLogScanner interface {
	Scan(dest ...any) error
}

func scanUsageLogRow(row usageLogScanner) (service.UsageLog, error) {
	var item service.UsageLog
	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.APIKeyID,
		&item.GroupID,
		&item.Operation,
		&item.Status,
		&item.Quantity,
		&item.Cost,
		&item.CreatedAt,
		&item.CompletedAt,
	); err != nil {
		return service.UsageLog{}, err
	}
	return item, nil
}
