package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/socialops/internal/domain"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type UsageHandler struct {
	usageService *service.UsageService
}

type usageStatsResponse struct {
	TotalOperations int64   `json:"total_operations"`
	SuccessCount    int64   `json:"success_count"`
	FailedCount     int64   `json:"failed_count"`
	TotalCharged    float64 `json:"total_charged"`
}

type userDashboardStatsResponse struct {
	TotalOperations           int64                            `json:"total_operations"`
	TodayOperations           int64                            `json:"today_operations"`
	TotalCharged              float64                          `json:"total_charged"`
	TodayCharged              float64                          `json:"today_charged"`
	RecentOperationsPerMinute int64                            `json:"recent_operations_per_minute"`
	ByPlatform                []platformDashboardStatsResponse `json:"by_platform,omitempty"`
}

type platformDashboardStatsResponse struct {
	Platform        string  `json:"platform"`
	TotalOperations int64   `json:"total_operations"`
	TotalCharged    float64 `json:"total_charged"`
	TodayOperations int64   `json:"today_operations"`
	TodayCharged    float64 `json:"today_charged"`
}

type dashboardTrendPointResponse struct {
	Date       string  `json:"date"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"`
}

func NewUsageHandler(usageService *service.UsageService) *UsageHandler {
	return &UsageHandler{usageService: usageService}
}

func (h *UsageHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	filters, err := usageLogFiltersFromQuery(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items, result, err := h.usageService.List(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items = sanitizeUsageLogResults(items)
	if result == nil {
		response.Paginated(c, items, int64(len(items)), page, pageSize)
		return
	}
	response.Paginated(c, items, result.Total, page, pageSize)
}

func (h *UsageHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid usage ID")
		return
	}
	item, err := h.usageService.GetByID(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeUsageLogDetail(item))
}

func (h *UsageHandler) PreviewTaskMedia(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.usageService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("USAGE_SERVICE_UNAVAILABLE", "usage service is unavailable"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid usage ID")
		return
	}
	index := -1
	if rawIndex := strings.TrimSpace(c.Query("index")); rawIndex != "" {
		parsed, err := strconv.Atoi(rawIndex)
		if err != nil {
			response.ErrorFrom(c, infraerrors.BadRequest("USAGE_TASK_MEDIA_LOCATOR_INVALID", "usage task media locator is invalid"))
			return
		}
		index = parsed
	}
	resolved, err := h.usageService.PreviewTaskMedia(c.Request.Context(), id, subject.UserID, service.UsageTaskMediaLocator{
		Scope:   c.Query("scope"),
		Section: c.Query("section"),
		Index:   index,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Disposition", inlineMediaContentDisposition(resolved.FileName))
	c.Data(http.StatusOK, resolved.ContentType, resolved.Body)
}

func (h *UsageHandler) Stats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	filters, err := usageLogFiltersFromQuery(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	stats, err := h.usageService.Stats(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, usageStatsResponseFromUsageStats(stats))
}

func (h *UsageHandler) DashboardStats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	stats, err := h.usageService.GetUserDashboardStats(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userDashboardStatsResponseFromUsageStats(stats))
}

func (h *UsageHandler) DashboardTrend(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	start, end := parseUsageWindow()
	trend, err := h.usageService.GetUserUsageTrendByUserID(c.Request.Context(), subject.UserID, start, end, c.DefaultQuery("granularity", "day"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dashboardTrendResponseFromUsageTrend(trend))
}

func parseUsageWindow() (time.Time, time.Time) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)
	return start, end
}

func usageStatsResponseFromUsageStats(stats *usagestats.UsageStats) usageStatsResponse {
	if stats == nil {
		return usageStatsResponse{}
	}
	return usageStatsResponse{
		TotalOperations: stats.TotalOperations,
		SuccessCount:    stats.SuccessCount,
		FailedCount:     stats.FailedCount,
		TotalCharged:    stats.TotalCharged,
	}
}

func userDashboardStatsResponseFromUsageStats(stats *usagestats.UserDashboardStats) userDashboardStatsResponse {
	if stats == nil {
		return userDashboardStatsResponse{}
	}
	out := userDashboardStatsResponse{
		TotalOperations:           stats.TotalOperations,
		TodayOperations:           stats.TodayOperations,
		TotalCharged:              stats.TotalCharged,
		TodayCharged:              stats.TodayCharged,
		RecentOperationsPerMinute: stats.RecentOperationsPerMinute,
	}
	if len(stats.ByPlatform) > 0 {
		out.ByPlatform = make([]platformDashboardStatsResponse, 0, len(stats.ByPlatform))
		for _, item := range stats.ByPlatform {
			out.ByPlatform = append(out.ByPlatform, platformDashboardStatsResponse{
				Platform:        item.Platform,
				TotalOperations: item.TotalOperations,
				TotalCharged:    item.TotalCharged,
				TodayOperations: item.TodayOperations,
				TodayCharged:    item.TodayCharged,
			})
		}
	}
	return out
}

func dashboardTrendResponseFromUsageTrend(trend []usagestats.TrendDataPoint) []dashboardTrendPointResponse {
	if len(trend) == 0 {
		return []dashboardTrendPointResponse{}
	}
	out := make([]dashboardTrendPointResponse, 0, len(trend))
	for _, point := range trend {
		out = append(out, dashboardTrendPointResponse{
			Date:       point.Date,
			Operations: point.Operations,
			Charged:    point.Charged,
		})
	}
	return out
}

func sanitizeUsageLogResults(items []service.UsageLog) []service.UsageLog {
	if len(items) == 0 {
		return items
	}
	sanitized := make([]service.UsageLog, len(items))
	copy(sanitized, items)
	for i := range sanitized {
		sanitized[i].ResultMessage = shortUserTaskResult(sanitized[i].ResultMessage)
		sanitized[i].ChargeSource = sanitizeUsageDetailTextPtr(sanitized[i].ChargeSource)
		sanitized[i].Target = sanitizeUsageDetailTextPtr(sanitized[i].Target)
		sanitized[i].Content = sanitizeUsageDetailTextPtr(sanitized[i].Content)
		sanitized[i].Payload = sanitizeUsageTaskPayload(sanitized[i].Payload)
		sanitized[i].TemplateSnapshot = sanitizeUsageTaskTemplateSnapshot(sanitized[i].TemplateSnapshot)
		sanitized[i].ProxySnapshot = sanitizeUsageProxySnapshot(sanitized[i].ProxySnapshot)
		sanitized[i].BillingRequestID = sanitizeUsageDetailTextPtr(sanitized[i].BillingRequestID)
		sanitized[i].IdempotencyKey = sanitizeUsageDetailTextPtr(sanitized[i].IdempotencyKey)
	}
	return sanitized
}

func sanitizeUsageLogDetail(item *service.UsageLog) *service.UsageLog {
	if item == nil {
		return nil
	}
	sanitized := *item
	sanitized.ResultMessage = shortUserTaskResult(item.ResultMessage)
	sanitized.ChargeSource = sanitizeUsageDetailTextPtr(item.ChargeSource)
	sanitized.Target = sanitizeUsageDetailTextPtr(item.Target)
	sanitized.Content = sanitizeUsageDetailTextPtr(item.Content)
	sanitized.Payload = sanitizeUsageTaskPayload(item.Payload)
	sanitized.TemplateSnapshot = sanitizeUsageTaskTemplateSnapshot(item.TemplateSnapshot)
	sanitized.ProxySnapshot = sanitizeUsageProxySnapshot(item.ProxySnapshot)
	sanitized.BillingRequestID = sanitizeUsageDetailTextPtr(item.BillingRequestID)
	sanitized.IdempotencyKey = sanitizeUsageDetailTextPtr(item.IdempotencyKey)
	return &sanitized
}

func sanitizeUsageTaskPayload(payload *domain.SocialTaskPayload) *domain.SocialTaskPayload {
	if payload == nil {
		return nil
	}
	sanitized := &domain.SocialTaskPayload{
		Target:  sanitizeUsageDetailText(payload.Target),
		Post:    sanitizeUsagePostPayload(payload.Post),
		Profile: sanitizeUsageProfileUpdateParams(payload.Profile),
		Avatar:  sanitizeUsageMediaRefPtr(payload.Avatar),
		Banner:  sanitizeUsageMediaRefPtr(payload.Banner),
	}
	if sanitized.IsZero() {
		return nil
	}
	return sanitized
}

func sanitizeUsagePostPayload(post *domain.SocialPostPayload) *domain.SocialPostPayload {
	if post == nil {
		return nil
	}
	sanitized := &domain.SocialPostPayload{
		Text:         sanitizeUsageDetailText(post.Text),
		QuotePostURL: sanitizeUsageDetailText(post.QuotePostURL),
		Media:        sanitizeUsageMediaRefs(post.Media),
	}
	if sanitized.IsZero() {
		return nil
	}
	return sanitized
}

func sanitizeUsageProfileUpdateParams(profile *domain.SocialProfileUpdateParams) *domain.SocialProfileUpdateParams {
	if profile == nil {
		return nil
	}
	sanitized := &domain.SocialProfileUpdateParams{
		DisplayName: sanitizeUsageDetailText(profile.DisplayName),
		ScreenName:  sanitizeUsageDetailText(profile.ScreenName),
		Description: sanitizeUsageDetailText(profile.Description),
		Location:    sanitizeUsageDetailText(profile.Location),
		URL:         sanitizeUsageDetailText(profile.URL),
	}
	if sanitized.IsZero() {
		return nil
	}
	return sanitized
}

func sanitizeUsageTaskTemplateSnapshot(snapshot *domain.SocialTaskTemplateSnapshot) *domain.SocialTaskTemplateSnapshot {
	if snapshot == nil {
		return nil
	}
	sanitized := &domain.SocialTaskTemplateSnapshot{
		TemplateID:   sanitizeUsageDetailText(snapshot.TemplateID),
		TemplateName: sanitizeUsageDetailText(snapshot.TemplateName),
		TemplateType: sanitizeUsageDetailText(snapshot.TemplateType),
		Params:       sanitizeUsageTaskTemplateParams(snapshot.Params),
	}
	if sanitized.IsZero() {
		return nil
	}
	return sanitized
}

func sanitizeUsageTaskTemplateParams(params domain.SocialTaskTemplateParams) domain.SocialTaskTemplateParams {
	return domain.SocialTaskTemplateParams{
		Targets:      sanitizeUsageDetailTextSlice(params.Targets),
		Contents:     sanitizeUsageDetailTextSlice(params.Contents),
		QuotePostURL: sanitizeUsageDetailText(params.QuotePostURL),
		Media:        sanitizeUsageMediaRefs(params.Media),
		Profile:      sanitizeUsageProfileUpdateParams(params.Profile),
		Avatar:       sanitizeUsageMediaRefPtr(params.Avatar),
		Banner:       sanitizeUsageMediaRefPtr(params.Banner),
	}
}

func sanitizeUsageMediaRefs(items []domain.SocialTaskMediaRef) []domain.SocialTaskMediaRef {
	if len(items) == 0 {
		return nil
	}
	sanitized := make([]domain.SocialTaskMediaRef, 0, len(items))
	for _, item := range items {
		safe := sanitizeUsageMediaRef(item)
		if safe.IsZero() {
			continue
		}
		sanitized = append(sanitized, safe)
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func sanitizeUsageMediaRefPtr(item *domain.SocialTaskMediaRef) *domain.SocialTaskMediaRef {
	if item == nil {
		return nil
	}
	safe := sanitizeUsageMediaRef(*item)
	if safe.IsZero() {
		return nil
	}
	return &safe
}

func sanitizeUsageMediaRef(item domain.SocialTaskMediaRef) domain.SocialTaskMediaRef {
	source := sanitizeUsageDetailText(item.Source)
	if strings.EqualFold(strings.TrimSpace(source), "library") {
		source = "inline"
	}
	return domain.SocialTaskMediaRef{
		Source:      source,
		ContentType: sanitizeUsageDetailText(item.ContentType),
		FileName:    sanitizeUsageDetailText(item.FileName),
		ByteSize:    item.ByteSize,
		Width:       item.Width,
		Height:      item.Height,
	}
}

func sanitizeUsageDetailTextPtr(value *string) *string {
	if value == nil {
		return nil
	}
	sanitized := sanitizeUsageDetailText(*value)
	if sanitized == "" {
		return nil
	}
	return &sanitized
}

func sanitizeUsageProxySnapshot(value *string) *string {
	if value == nil {
		return nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		if parsed, parseErr := url.Parse(raw); parseErr == nil && parsed.Scheme != "" && parsed.Host != "" {
			parsed.User = nil
			sanitized := sanitizeUsageDetailText(parsed.String())
			if sanitized == "" {
				return nil
			}
			return &sanitized
		}
		sanitized := sanitizeUsageProxySnapshotString(raw)
		if sanitized == "" {
			return nil
		}
		return &sanitized
	}
	payload = sanitizeUsageProxySnapshotMap(payload)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return sanitizeUsageDetailTextPtr(value)
	}
	sanitized := sanitizeUsageDetailText(string(encoded))
	if sanitized == "" {
		return nil
	}
	return &sanitized
}

func sanitizeUsageProxySnapshotMap(payload map[string]any) map[string]any {
	sanitized := make(map[string]any, len(payload))
	for key, value := range payload {
		if isSensitiveUsageProxySnapshotKey(key) {
			continue
		}
		sanitizedValue := sanitizeUsageProxySnapshotValue(value)
		if sanitizedValue == nil {
			continue
		}
		sanitized[key] = sanitizedValue
	}
	return sanitized
}

func sanitizeUsageProxySnapshotValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeUsageProxySnapshotMap(typed)
	case []any:
		sanitized := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitizedItem := sanitizeUsageProxySnapshotValue(item)
			if sanitizedItem == nil {
				continue
			}
			sanitized = append(sanitized, sanitizedItem)
		}
		if len(sanitized) == 0 {
			return nil
		}
		return sanitized
	case string:
		return sanitizeUsageProxySnapshotString(typed)
	default:
		return value
	}
}

func sanitizeUsageProxySnapshotString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.User = nil
		value = parsed.String()
	}
	if containsSensitiveUsageProxySnapshotText(value) {
		return ""
	}
	return sanitizeUsageDetailText(value)
}

func containsSensitiveUsageProxySnapshotText(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "bearer ") ||
		strings.Contains(normalized, "password=") ||
		strings.Contains(normalized, "passwd=") ||
		strings.Contains(normalized, "proxy_password=") ||
		strings.Contains(normalized, "proxypassword=") ||
		strings.Contains(normalized, "username=") ||
		strings.Contains(normalized, "proxy_username=") ||
		strings.Contains(normalized, "proxyusername=") ||
		strings.Contains(normalized, "token=") ||
		strings.Contains(normalized, "access_token=") ||
		strings.Contains(normalized, "accesstoken=") ||
		strings.Contains(normalized, "cookie=") ||
		strings.Contains(normalized, "auth_cookie=") ||
		strings.Contains(normalized, "authcookie=") ||
		strings.Contains(normalized, "secret=")
}

func isSensitiveUsageProxySnapshotKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "user", "username", "proxy_user", "proxy_username",
		"proxyuser", "proxyusername",
		"pass", "password", "proxy_pass", "proxy_password",
		"proxypass", "proxypassword",
		"auth", "authorization", "proxy_auth", "credentials", "credential",
		"proxyauth",
		"cookie", "cookies", "auth_cookie", "authcookie",
		"token", "access_token", "accesstoken", "refresh_token", "refreshtoken", "secret":
		return true
	default:
		return false
	}
}

func sanitizeUsageDetailTextSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sanitized := make([]string, 0, len(values))
	for _, value := range values {
		safe := sanitizeUsageDetailText(value)
		if safe == "" {
			continue
		}
		sanitized = append(sanitized, safe)
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func sanitizeUsageDetailText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	const maxUsageDetailTextLen = 2048
	runes := []rune(value)
	if len(runes) > maxUsageDetailTextLen {
		value = string(runes[:maxUsageDetailTextLen])
	}
	return value
}

func usageLogFiltersFromQuery(c *gin.Context, userID int64) (usagestats.UsageLogFilters, error) {
	status := normalizeUsageQueryValue(c.Query("status"))
	operation := normalizeUsageQueryValue(c.Query("operation"))
	filters := usagestats.UsageLogFilters{
		UserID:      userID,
		Operation:   operation,
		Platform:    normalizeUsageQueryValue(c.Query("platform")),
		AccountName: normalizeUsageQueryValue(firstUsageQuery(c, "account", "account_name")),
		Status:      status,
	}
	if start, err := parseUsageQueryTime(c.Query("start_date"), false); err != nil {
		return filters, err
	} else if start != nil {
		filters.StartTime = start
	}
	if end, err := parseUsageQueryTime(c.Query("end_date"), true); err != nil {
		return filters, err
	} else if end != nil {
		filters.EndTime = end
	}
	return filters, nil
}

func firstUsageQuery(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(c.Query(name)); value != "" {
			return value
		}
	}
	return ""
}

func normalizeUsageQueryValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseUsageQueryTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		utc := parsed.UTC()
		return &utc, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, infraerrors.BadRequest("USAGE_DATE_INVALID", "invalid usage date filter")
	}
	parsed = parsed.UTC()
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
