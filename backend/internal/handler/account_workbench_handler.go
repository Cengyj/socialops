package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/socialops/internal/handler/socialaccountcsv"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountWorkbenchHandler handles user-facing social account operations.
type AccountWorkbenchHandler struct {
	svc           *service.SocialAccountService
	ipSvc         *service.SocialIPService
	globalProxies *service.GlobalProxyService
	billing       *service.SocialBillingService
	executor      *service.SocialTaskExecutor
	workbench     *service.AccountWorkbenchService
	templates     *service.TaskSettingsService
}

// NewAccountWorkbenchHandler creates a new AccountWorkbenchHandler.
func NewAccountWorkbenchHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor, templates *service.TaskSettingsService) *AccountWorkbenchHandler {
	return NewAccountWorkbenchHandlerWithGlobalProxies(svc, ipSvc, nil, billing, executor, templates)
}

// NewAccountWorkbenchHandlerWithGlobalProxies creates a user workbench handler
// with access to administrator-managed global fallback proxies.
func NewAccountWorkbenchHandlerWithGlobalProxies(svc *service.SocialAccountService, ipSvc *service.SocialIPService, globalProxies *service.GlobalProxyService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor, templates *service.TaskSettingsService) *AccountWorkbenchHandler {
	return &AccountWorkbenchHandler{svc: svc, ipSvc: ipSvc, globalProxies: globalProxies, billing: billing, executor: executor, workbench: service.NewAccountWorkbenchServiceWithGlobalProxies(svc, ipSvc, globalProxies, billing, executor), templates: templates}
}

type socialTaskRequest struct {
	AccountIDs      []int64 `json:"account_ids" binding:"required"`
	Action          string  `json:"action"`
	ClientRequestID string  `json:"client_request_id"`
}

type batchImportMyAccountsRequest struct {
	Accounts []*service.UserImportSocialAccountInput `json:"accounts" binding:"required"`
}

type batchDeleteMyAccountsRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

type batchDefaultProxyRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required"`
	Mode       string  `json:"mode" binding:"required"`
	ProxyID    *int64  `json:"proxy_id"`
}

type updateMyAccountRequest struct {
	Password       *string `json:"password"`
	Phone          *string `json:"phone"`
	Email          *string `json:"email"`
	EmailPassword  *string `json:"email_password"`
	TwoFactor      *string `json:"two_factor"`
	BackupCode     *string `json:"backup_code"`
	EmailClientID  *string `json:"email_client_id"`
	EmailToken     *string `json:"email_token"`
	RegistrationIP *string `json:"registration_ip"`
	AuthCookie     *string `json:"auth_cookie"`
	ExecutionAuth  *string `json:"execution_auth"`
	Remark         *string `json:"remark"`
}

type userSocialAccountResponse struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	Platform               string    `json:"platform"`
	Username               string    `json:"username,omitempty"`
	PlatformUserID         *string   `json:"platform_user_id,omitempty"`
	Password               *string   `json:"password,omitempty"`
	Phone                  *string   `json:"phone,omitempty"`
	Email                  *string   `json:"email,omitempty"`
	EmailPassword          *string   `json:"email_password,omitempty"`
	TwoFactor              *string   `json:"two_factor,omitempty"`
	BackupCode             *string   `json:"backup_code,omitempty"`
	EmailClientID          *string   `json:"email_client_id,omitempty"`
	EmailToken             *string   `json:"email_token,omitempty"`
	RegistrationIP         *string   `json:"registration_ip,omitempty"`
	AuthCookie             *string   `json:"auth_cookie,omitempty"`
	ExecutionAuth          *string   `json:"execution_auth,omitempty"`
	AccountStatus          string    `json:"account_status"`
	TaskStatus             string    `json:"task_status"`
	TaskMessage            *string   `json:"task_message,omitempty"`
	DefaultProxySnapshot   *string   `json:"default_proxy_snapshot,omitempty"`
	Remark                 *string   `json:"remark,omitempty"`
	DefaultProxyConfigured bool      `json:"default_proxy_configured"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type userSocialTaskLogResponse struct {
	ID               int64                               `json:"id"`
	SocialAccountID  int64                               `json:"social_account_id"`
	Action           string                              `json:"action"`
	Platform         string                              `json:"platform"`
	AccountName      string                              `json:"account_name"`
	Status           string                              `json:"status"`
	Target           *string                             `json:"target,omitempty"`
	Content          *string                             `json:"content,omitempty"`
	Payload          *service.SocialTaskPayload          `json:"payload,omitempty"`
	TemplateSnapshot *service.SocialTaskTemplateSnapshot `json:"template_snapshot,omitempty"`
	ResultMessage    *string                             `json:"result_message,omitempty"`
	Charged          bool                                `json:"charged"`
	ChargedAmount    float64                             `json:"charged_amount"`
	ChargeStatus     string                              `json:"charge_status"`
	ExecutedAt       *time.Time                          `json:"executed_at,omitempty"`
	CreatedAt        time.Time                           `json:"created_at"`
}

// ListMyAccounts returns social accounts assigned to the current user.
// GET /api/v1/accounts
func (h *AccountWorkbenchHandler) ListMyAccounts(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := accountWorkbenchFiltersFromQuery(c)

	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	accounts, result, err := svc.ListByUser(c.Request.Context(), subject.UserID, params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, userSocialAccountResponsesFromService(accounts), result.Total, page, pageSize)
}

// ListTaskLogs returns recent/current task logs for the current user's workbench.
// GET /api/v1/accounts/tasks
func (h *AccountWorkbenchHandler) ListTaskLogs(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	filters, err := accountWorkbenchTaskLogFiltersFromQuery(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	logs, err := svc.ListTaskLogsForUser(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"logs": h.userTaskLogResponses(c.Request.Context(), logs)})
}

// BatchImportMyAccounts binds or restores existing total-pool accounts for the current user.
// POST /api/v1/accounts/batch-import
func (h *AccountWorkbenchHandler) BatchImportMyAccounts(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	var req batchImportMyAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	result, err := svc.BatchImportForUser(c.Request.Context(), subject.UserID, req.Accounts)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"total":      result.Total,
		"succeeded":  result.Succeeded,
		"imported":   result.Imported,
		"skipped":    result.Skipped,
		"failed":     result.Failed,
		"duplicates": result.Duplicates,
		"errors":     result.Errors,
		"items":      result.Items,
		"accounts":   userSocialAccountResponsesFromService(result.Accounts),
	})
}

// UpdateMyAccount updates mutable credential fields for an assigned account.
// PUT /api/v1/accounts/:id
func (h *AccountWorkbenchHandler) UpdateMyAccount(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	id, ok := accountWorkbenchPathID(c)
	if !ok {
		return
	}
	var req updateMyAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	account, err := svc.UpdateForUser(c.Request.Context(), id, subject.UserID, &service.UpdateSocialAccountInput{
		Password:       req.Password,
		Phone:          req.Phone,
		Email:          req.Email,
		EmailPassword:  req.EmailPassword,
		TwoFactor:      req.TwoFactor,
		BackupCode:     req.BackupCode,
		EmailClientID:  req.EmailClientID,
		EmailToken:     req.EmailToken,
		RegistrationIP: req.RegistrationIP,
		AuthCookie:     req.AuthCookie,
		ExecutionAuth:  req.ExecutionAuth,
		Remark:         req.Remark,
	})
	if err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		if respondUserAccountEditError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userSocialAccountResponseFromService(account))
}

// DeleteMyAccount deletes an assigned account from the account pool.
// DELETE /api/v1/accounts/:id
func (h *AccountWorkbenchHandler) DeleteMyAccount(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	id, ok := accountWorkbenchPathID(c)
	if !ok {
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	if err := svc.DeleteForUser(c.Request.Context(), subject.UserID, id); err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// BatchDeleteMyAccounts deletes assigned accounts from the account pool.
// POST /api/v1/accounts/batch-delete
func (h *AccountWorkbenchHandler) BatchDeleteMyAccounts(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	var req batchDeleteMyAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	result, err := svc.BatchDeleteForUser(c.Request.Context(), subject.UserID, req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ExportMyAccounts exports the current user's assigned accounts with delivery fields.
// GET /api/v1/accounts/export
func (h *AccountWorkbenchHandler) ExportMyAccounts(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	filters := accountWorkbenchFiltersFromQuery(c)
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	accounts, err := svc.ListAllByUserForExport(c.Request.Context(), subject.UserID, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=accounts.csv")
	if err := socialaccountcsv.WriteDeliveryExport(c.Writer, accounts); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// SubmitTask submits a batch task for social accounts.
func (h *AccountWorkbenchHandler) SubmitTask(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	var req socialTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.ClientRequestID)
	}
	input := &service.AccountWorkbenchTaskInput{
		Mode:           service.AccountWorkbenchTaskModeUser,
		UserID:         subject.UserID,
		AccountIDs:     req.AccountIDs,
		Action:         req.Action,
		IdempotencyKey: idempotencyKey,
	}
	if h == nil {
		response.ErrorFrom(c, taskTemplateServiceUnavailableError())
		return
	}
	if err := h.templates.ApplyDefaultTemplateToTaskInput(c.Request.Context(), subject.UserID, input); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	workbench, ok := h.taskSubmitService(c)
	if !ok {
		return
	}
	result, err := workbench.SubmitTask(c.Request.Context(), input)
	if err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	publicLogs := h.userTaskLogResponses(c.Request.Context(), result.Logs)
	response.Success(c, gin.H{"submitted": len(publicLogs), "enqueued": result.Enqueued, "failed_closed": result.FailedClosed, "logs": publicLogs})
}

// SetDefaultProxy stores or clears the default execution proxy snapshot for an assigned account.
// PUT /api/v1/accounts/:id/default-proxy
func (h *AccountWorkbenchHandler) SetDefaultProxy(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	id, ok := accountWorkbenchPathID(c)
	if !ok {
		return
	}
	var req struct {
		ProxyID *int64 `json:"proxy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}
	var proxy *service.SocialIP
	if req.ProxyID != nil && *req.ProxyID > 0 {
		ipSvc, ok := h.proxyService(c)
		if !ok {
			return
		}
		ip, err := ipSvc.GetByIDForUser(c.Request.Context(), *req.ProxyID, subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		proxy = ip
	}
	account, err := svc.SetDefaultProxyForUser(c.Request.Context(), id, subject.UserID, proxy)
	if err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userSocialAccountResponseFromService(account))
}

// BatchSetDefaultProxy assigns, clears, or randomly distributes current-user online proxies.
// POST /api/v1/accounts/default-proxy
func (h *AccountWorkbenchHandler) BatchSetDefaultProxy(c *gin.Context) {
	subject, ok := accountWorkbenchAuthSubject(c)
	if !ok {
		return
	}
	var req batchDefaultProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, accountWorkbenchInputRequiredError())
		return
	}
	svc, ok := h.accountService(c)
	if !ok {
		return
	}

	var proxy *service.SocialIP
	var pool []*service.SocialIP
	var err error
	switch strings.TrimSpace(req.Mode) {
	case service.DefaultProxyAssignmentSpecific:
		if req.ProxyID == nil || *req.ProxyID <= 0 {
			response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_REQUIRED", "proxy is required for this assignment"))
			return
		}
		ipSvc, ok := h.proxyService(c)
		if !ok {
			return
		}
		proxy, err = ipSvc.GetByIDForUser(c.Request.Context(), *req.ProxyID, subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	case service.DefaultProxyAssignmentRandom:
		ipSvc, ok := h.proxyService(c)
		if !ok {
			return
		}
		pool, err = ipSvc.ListUsableByUser(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	case service.DefaultProxyAssignmentClear:
	default:
		response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_ASSIGNMENT_MODE_INVALID", "proxy assignment mode is invalid"))
		return
	}

	result, err := svc.BatchSetDefaultProxyForUser(c.Request.Context(), subject.UserID, req.AccountIDs, req.Mode, proxy, pool)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func accountWorkbenchAuthSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func accountWorkbenchPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func accountWorkbenchFiltersFromQuery(c *gin.Context) service.SocialAccountListFilters {
	return service.SocialAccountListFilters{
		Search:        c.Query("search"),
		Platform:      c.Query("platform"),
		AccountStatus: c.Query("account_status"),
		TaskStatus:    c.Query("task_status"),
		AccountIDs:    parseInt64ListQuery(c, "account_ids"),
	}
}

func accountWorkbenchTaskLogFiltersFromQuery(c *gin.Context, userID int64) (service.SocialTaskLogListFilters, error) {
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return service.SocialTaskLogListFilters{}, infraerrors.BadRequest("SOCIAL_TASK_LOG_LIMIT_INVALID", "task log limit is invalid")
		}
		limit = parsed
	}
	return service.SocialTaskLogListFilters{
		UserID:     userID,
		LogIDs:     parseInt64ListQuery(c, "log_ids"),
		AccountIDs: parseInt64ListQuery(c, "account_ids"),
		Statuses:   parseStringListQuery(c, "statuses"),
		Limit:      limit,
	}, nil
}

func respondUserAccountScopeError(c *gin.Context, err error) bool {
	if !errors.Is(err, service.ErrSocialAccountNotAssigned) {
		return false
	}
	response.ErrorFrom(c, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found"))
	return true
}

func respondUserAccountEditError(c *gin.Context, err error) bool {
	if !errors.Is(err, service.ErrSocialAccountExecutionAuthInvalid) {
		return false
	}
	response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID", "account execution auth is invalid"))
	return true
}

func accountWorkbenchInputRequiredError() error {
	return infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
}

func (h *AccountWorkbenchHandler) accountService(c *gin.Context) (*service.SocialAccountService, bool) {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, accountServiceUnavailableError())
		return nil, false
	}
	return h.svc, true
}

func (h *AccountWorkbenchHandler) proxyService(c *gin.Context) (*service.SocialIPService, bool) {
	if h == nil || h.ipSvc == nil {
		response.ErrorFrom(c, proxyServiceUnavailableError())
		return nil, false
	}
	return h.ipSvc, true
}

func (h *AccountWorkbenchHandler) taskSubmitService(c *gin.Context) (*service.AccountWorkbenchService, bool) {
	if h == nil || h.workbench == nil {
		response.ErrorFrom(c, socialTaskServiceUnavailableError())
		return nil, false
	}
	return h.workbench, true
}

func accountServiceUnavailableError() error {
	return infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE", "social account service is unavailable")
}

func socialTaskServiceUnavailableError() error {
	return infraerrors.ServiceUnavailable("SOCIAL_TASK_SERVICE_UNAVAILABLE", "social task service is unavailable")
}

func userSocialAccountResponsesFromService(accounts []*service.SocialAccount) []*userSocialAccountResponse {
	items := make([]*userSocialAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		items = append(items, userSocialAccountResponseFromService(account))
	}
	return items
}

func userSocialAccountResponseFromService(account *service.SocialAccount) *userSocialAccountResponse {
	if account == nil {
		return nil
	}
	return &userSocialAccountResponse{
		ID:                     account.ID,
		Name:                   account.Name,
		Platform:               account.Platform,
		Username:               account.Username,
		PlatformUserID:         account.PlatformUserID,
		Password:               account.Password,
		Phone:                  account.Phone,
		Email:                  account.Email,
		EmailPassword:          account.EmailPassword,
		TwoFactor:              account.TwoFactor,
		BackupCode:             account.BackupCode,
		EmailClientID:          account.EmailClientID,
		EmailToken:             account.EmailToken,
		RegistrationIP:         account.RegistrationIP,
		AuthCookie:             account.AuthCookie,
		ExecutionAuth:          account.ExecutionAuth,
		AccountStatus:          account.AccountStatus,
		TaskStatus:             account.TaskStatus,
		TaskMessage:            userSocialAccountTaskMessage(account),
		DefaultProxySnapshot:   account.DefaultProxySnapshot,
		Remark:                 account.Remark,
		DefaultProxyConfigured: userSocialAccountDefaultProxyConfigured(account),
		CreatedAt:              account.CreatedAt,
		UpdatedAt:              account.UpdatedAt,
	}
}

func userSocialAccountTaskMessage(account *service.SocialAccount) *string {
	if account == nil {
		return nil
	}
	return shortUserTaskResult(account.TaskMessage)
}

func userSocialAccountDefaultProxyConfigured(account *service.SocialAccount) bool {
	if account == nil || account.DefaultProxySnapshot == nil {
		return false
	}
	return service.SocialIPSnapshotUsable(*account.DefaultProxySnapshot)
}

func parseInt64ListQuery(c *gin.Context, key string) []int64 {
	parts := parseStringListQuery(c, key)
	if len(parts) == 0 {
		return nil
	}
	values := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil || parsed <= 0 {
			continue
		}
		values = append(values, parsed)
	}
	return values
}

func parseStringListQuery(c *gin.Context, key string) []string {
	if c == nil {
		return nil
	}
	rawValues := c.QueryArray(key)
	if len(rawValues) == 0 {
		raw := strings.TrimSpace(c.Query(key))
		if raw != "" {
			rawValues = []string{raw}
		}
	}
	if len(rawValues) == 0 {
		return nil
	}
	values := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				values = append(values, trimmed)
			}
		}
	}
	return values
}

func (h *AccountWorkbenchHandler) userTaskLogResponses(ctx context.Context, logs []*service.SocialTaskLog) []*userSocialTaskLogResponse {
	items := make([]*userSocialTaskLogResponse, 0, len(logs))
	accounts := make(map[int64]*service.SocialAccount)
	missing := make(map[int64]bool)
	for _, log := range logs {
		if log == nil {
			continue
		}
		account := accounts[log.SocialAccountID]
		if account == nil && !missing[log.SocialAccountID] && h != nil && h.svc != nil {
			found, err := h.svc.GetByID(ctx, log.SocialAccountID)
			if err == nil {
				account = found
				accounts[log.SocialAccountID] = found
			} else {
				missing[log.SocialAccountID] = true
			}
		}
		items = append(items, userTaskLogResponseFromService(log, account))
	}
	return items
}

func userTaskLogResponseFromService(log *service.SocialTaskLog, account *service.SocialAccount) *userSocialTaskLogResponse {
	item := &userSocialTaskLogResponse{
		ID:               log.ID,
		SocialAccountID:  log.SocialAccountID,
		Action:           log.Action,
		Status:           log.Status,
		Target:           sanitizeUsageDetailTextPtr(log.Target),
		Content:          sanitizeUsageDetailTextPtr(log.Content),
		Payload:          sanitizeUsageTaskPayload(log.Payload),
		TemplateSnapshot: sanitizeUsageTaskTemplateSnapshot(log.TemplateSnapshot),
		ResultMessage:    shortUserTaskResult(log.ResultMessage),
		Charged:          userTaskLogCharged(log),
		ChargedAmount:    log.ChargedAmount,
		ChargeStatus:     log.ChargeStatus,
		ExecutedAt:       log.ExecutedAt,
		CreatedAt:        log.CreatedAt,
	}
	applyUserTaskLogAccount(item, account)
	return item
}

func userTaskLogCharged(log *service.SocialTaskLog) bool {
	return log != nil && log.ChargeStatus == service.SocialTaskChargeStatusCharged && log.ChargedAmount > 0
}

func applyUserTaskLogAccount(item *userSocialTaskLogResponse, account *service.SocialAccount) {
	if item == nil || account == nil {
		return
	}
	item.Platform = account.Platform
	item.AccountName = account.Name
}

const (
	userTaskResultCompletedHidden = "任务已完成，详细结果已隐藏"
	userTaskResultFailedNoCharge  = "任务执行失败，本次未扣费"
	userTaskResultAvatarSize      = "头像图片尺寸必须为 400x400，本次未扣费"
	userTaskResultBannerSize      = "背景图图片尺寸必须为 1500x500，本次未扣费"
	userTaskResultMediaAsset      = "任务媒体资源不可用，本次未扣费"
	userTaskResultMediaUpload     = "平台媒体上传失败，本次未扣费"
	userTaskResultMediaSource     = "媒体引用暂未开放，本次未扣费"
	userTaskResultPostVideo       = "视频发帖媒体暂未开放，本次未扣费"
	userTaskResultPostMediaType   = "发帖媒体类型暂不支持，本次未扣费"
	userTaskResultChallenge       = "账号需要额外验证，本次未扣费"
	userTaskResultPasswordInvalid = "密码错误，本次未扣费"
	userTaskResultLoginConfig     = "登录依赖服务未配置，本次未扣费"
	userTaskResultTimeout         = "任务执行超时，本次未扣费"
)

func shortUserTaskResult(value *string) *string {
	if value == nil {
		return nil
	}
	message := strings.Join(strings.Fields(*value), " ")
	if message == "" {
		return nil
	}
	if safe := safeUserTaskResultMessage(message); safe != "" {
		return &safe
	}
	return cappedUserTaskResult(message)
}

func cappedUserTaskResult(message string) *string {
	const maxUserTaskResultLen = 160
	runes := []rune(message)
	if len(runes) > maxUserTaskResultLen {
		message = string(runes[:maxUserTaskResultLen])
	}
	return &message
}

func safeUserTaskResultMessage(message string) string {
	normalizedExact := strings.Join(strings.Fields(message), " ")
	switch normalizedExact {
	case userTaskResultCompletedHidden,
		userTaskResultFailedNoCharge,
		"任务队列繁忙，本次未扣费",
		userTaskResultTimeout,
		"执行已完成，但扣费确认异常，请联系管理员处理",
		"账号认证信息不可用，本次未扣费",
		"执行代理不可用，本次未扣费",
		"全局代理不可用，本次未扣费",
		"平台网络请求失败，本次未扣费",
		userTaskResultPasswordInvalid,
		userTaskResultLoginConfig,
		"该动作暂不支持，本次未扣费",
		"任务参数不完整，本次未扣费",
		"执行目标不存在，本次未扣费",
		userTaskResultChallenge,
		"账号状态或频率受限，本次未扣费",
		"内容或目标状态不符合平台要求，本次未扣费",
		"平台拒绝执行，本次未扣费",
		userTaskResultAvatarSize,
		userTaskResultBannerSize,
		userTaskResultMediaAsset,
		userTaskResultMediaUpload,
		userTaskResultMediaSource,
		userTaskResultPostVideo,
		userTaskResultPostMediaType:
		return normalizedExact
	}

	if known, ok := service.KnownTwitterTaskFailureMessage(message); ok {
		return known
	}
	if service.IsTwitterPlatformFailureMessage(message) {
		return ""
	}

	normalized := strings.ToLower(message)
	if known := knownUserTaskFailureMessage(normalized); known != "" {
		return known
	}
	switch {
	case strings.Contains(normalized, "wrong password") ||
		strings.Contains(normalized, "incorrect password") ||
		strings.Contains(normalized, "invalid password") ||
		strings.Contains(normalized, "password is incorrect") ||
		strings.Contains(normalized, "password you entered is incorrect") ||
		strings.Contains(normalized, "密码错误"):
		return userTaskResultPasswordInvalid
	case strings.Contains(normalized, "avatar image must be 400x400 pixels"):
		return userTaskResultAvatarSize
	case strings.Contains(normalized, "banner image must be 1500x500 pixels"):
		return userTaskResultBannerSize
	case strings.Contains(normalized, "media asset is unavailable"):
		return userTaskResultMediaAsset
	case strings.Contains(normalized, "twitter media upload returned invalid media id"),
		strings.Contains(normalized, "twitter media upload returned invalid response"),
		strings.Contains(normalized, "twitter media upload returned no media id"),
		strings.Contains(normalized, "twitter media upload returned processing failed"),
		strings.Contains(normalized, "twitter media upload returned processing timeout"):
		return userTaskResultMediaUpload
	case strings.Contains(normalized, "media source is not supported for socialops execution"):
		return userTaskResultMediaSource
	case strings.Contains(normalized, "video media is not supported for socialops execution"):
		return userTaskResultPostVideo
	case strings.Contains(normalized, "post media content type is not supported"):
		return userTaskResultPostMediaType
	case strings.Contains(normalized, "queue is full"):
		return "任务队列繁忙，本次未扣费"
	case strings.Contains(normalized, "device fingerprint provider is not configured") ||
		strings.Contains(normalized, "twitter login is not configured"):
		return userTaskResultLoginConfig
	case strings.Contains(normalized, "billing failed"):
		return "执行已完成，但扣费确认异常，请联系管理员处理"
	case strings.Contains(normalized, "auth cookie") ||
		strings.Contains(normalized, "authentication failed") ||
		strings.Contains(normalized, "oauth") ||
		strings.Contains(normalized, "token"):
		return "账号认证信息不可用，本次未扣费"
	case strings.Contains(normalized, "global proxy"):
		return "全局代理不可用，本次未扣费"
	case strings.Contains(normalized, "proxy"):
		return "执行代理不可用，本次未扣费"
	case strings.Contains(normalized, "network") ||
		strings.Contains(normalized, "timeout") ||
		strings.Contains(normalized, "connection"):
		return "平台网络请求失败，本次未扣费"
	case strings.Contains(normalized, "unsupported action"):
		return "该动作暂不支持，本次未扣费"
	case strings.Contains(normalized, "target is required") ||
		strings.Contains(normalized, "content is required"):
		return "任务参数不完整，本次未扣费"
	case strings.Contains(normalized, "target not found"):
		return "执行目标不存在，本次未扣费"
	case strings.Contains(normalized, "challenge required") ||
		strings.Contains(normalized, "captcha challenge") ||
		strings.Contains(normalized, "verification required") ||
		strings.Contains(normalized, "additional verification") ||
		strings.Contains(normalized, "confirm your identity"):
		return userTaskResultChallenge
	case strings.Contains(normalized, "rate limit") ||
		strings.Contains(normalized, "too frequent") ||
		strings.Contains(normalized, "follow limit") ||
		strings.Contains(normalized, "suspended") ||
		strings.Contains(normalized, "locked"):
		return "账号状态或频率受限，本次未扣费"
	case strings.Contains(normalized, "content is too long") ||
		strings.Contains(normalized, "duplicate") ||
		strings.Contains(normalized, "already") ||
		strings.Contains(normalized, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费"
	case strings.Contains(normalized, "access denied"):
		return "平台拒绝执行，本次未扣费"
	default:
		return ""
	}
}

func knownUserTaskFailureMessage(normalized string) string {
	switch {
	case strings.Contains(normalized, "wrong password") ||
		strings.Contains(normalized, "incorrect password") ||
		strings.Contains(normalized, "invalid password") ||
		strings.Contains(normalized, "password is incorrect") ||
		strings.Contains(normalized, "password you entered is incorrect") ||
		strings.Contains(normalized, "密码错误"):
		return userTaskResultPasswordInvalid
	case strings.Contains(normalized, "could not find your account") ||
		strings.Contains(normalized, "account not found") ||
		strings.Contains(normalized, "user not found"):
		return "账号不存在，本次未扣费"
	case strings.Contains(normalized, "challenge required") ||
		strings.Contains(normalized, "captcha challenge") ||
		strings.Contains(normalized, "verification required") ||
		strings.Contains(normalized, "verify login") ||
		strings.Contains(normalized, "additional verification") ||
		strings.Contains(normalized, "confirm your identity"):
		return userTaskResultChallenge
	case strings.Contains(normalized, "suspended") ||
		strings.Contains(normalized, "locked") ||
		strings.Contains(normalized, "automated") ||
		strings.Contains(normalized, "rate limit") ||
		strings.Contains(normalized, "too frequent") ||
		strings.Contains(normalized, "follow limit"):
		return "账号状态或频率受限，本次未扣费"
	case strings.Contains(normalized, "content is too long") ||
		strings.Contains(normalized, "duplicate") ||
		strings.Contains(normalized, "already") ||
		strings.Contains(normalized, "restricted"):
		return "内容或目标状态不符合平台要求，本次未扣费"
	case strings.Contains(normalized, "authentication failed"):
		return "账号认证信息不可用，本次未扣费"
	}
	return ""
}
