package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountWorkbenchHandler handles user-facing social account operations.
type AccountWorkbenchHandler struct {
	svc       *service.SocialAccountService
	ipSvc     *service.SocialIPService
	billing   *service.SocialBillingService
	executor  *service.SocialTaskExecutor
	workbench *service.AccountWorkbenchService
	templates *service.TaskSettingsService
}

// NewAccountWorkbenchHandler creates a new AccountWorkbenchHandler.
func NewAccountWorkbenchHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor, templates *service.TaskSettingsService) *AccountWorkbenchHandler {
	return &AccountWorkbenchHandler{svc: svc, ipSvc: ipSvc, billing: billing, executor: executor, workbench: service.NewAccountWorkbenchService(svc, ipSvc, billing, executor), templates: templates}
}

type socialTaskRequest struct {
	AccountIDs      []int64 `json:"account_ids" binding:"required"`
	TemplateID      string  `json:"template_id"`
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
	Name                 *string `json:"name"`
	Platform             *string `json:"platform"`
	PlatformUserID       *string `json:"platform_user_id"`
	Password             *string `json:"password"`
	Phone                *string `json:"phone"`
	Email                *string `json:"email"`
	EmailPassword        *string `json:"email_password"`
	TwoFactor            *string `json:"two_factor"`
	BackupCode           *string `json:"backup_code"`
	EmailClientID        *string `json:"email_client_id"`
	EmailToken           *string `json:"email_token"`
	AuthCookie           *string `json:"auth_cookie"`
	ExecutionAuth        *string `json:"execution_auth"`
	AccountStatus        *string `json:"account_status"`
	TaskStatus           *string `json:"task_status"`
	TaskMessage          *string `json:"task_message"`
	DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
	Remark               *string `json:"remark"`
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	accounts, result, err := h.svc.ListByUser(c.Request.Context(), subject.UserID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, userSocialAccountResponsesFromService(accounts), result.Total, page, pageSize)
}

// BatchImportMyAccounts binds or restores existing total-pool accounts for the current user.
// POST /api/v1/accounts/batch-import
func (h *AccountWorkbenchHandler) BatchImportMyAccounts(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req batchImportMyAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.BatchImportForUser(c.Request.Context(), subject.UserID, req.Accounts)
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateMyAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.UpdateForUser(c.Request.Context(), id, subject.UserID, &service.UpdateSocialAccountInput{
		Password:      req.Password,
		Phone:         req.Phone,
		Email:         req.Email,
		EmailPassword: req.EmailPassword,
		TwoFactor:     req.TwoFactor,
		BackupCode:    req.BackupCode,
		EmailClientID: req.EmailClientID,
		EmailToken:    req.EmailToken,
		AuthCookie:    req.AuthCookie,
		ExecutionAuth: req.ExecutionAuth,
		Remark:        req.Remark,
	})
	if err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userSocialAccountResponseFromService(account))
}

// DeleteMyAccount hides an assigned account from the current user's workbench.
// DELETE /api/v1/accounts/:id
func (h *AccountWorkbenchHandler) DeleteMyAccount(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.RemoveFromUserWorkbench(c.Request.Context(), subject.UserID, id); err != nil {
		if respondUserAccountScopeError(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": 1})
}

// BatchDeleteMyAccounts hides assigned accounts from the current user's workbench.
// POST /api/v1/accounts/batch-delete
func (h *AccountWorkbenchHandler) BatchDeleteMyAccounts(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req batchDeleteMyAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.BatchRemoveFromUserWorkbench(c.Request.Context(), subject.UserID, req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ExportMyAccounts exports the current user's assigned accounts with delivery fields.
// GET /api/v1/accounts/export
func (h *AccountWorkbenchHandler) ExportMyAccounts(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	accounts, _, err := h.svc.ListByUser(c.Request.Context(), subject.UserID, pagination.PaginationParams{Page: 1, PageSize: 10000})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=accounts.csv")
	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{"platform", "username", "name", "platform_user_id", "password", "phone", "email", "email_password", "two_factor", "backup_code", "email_client_id", "email_token", "registration_ip", "auth_cookie", "execution_auth", "default_proxy_snapshot", "account_status", "task_status", "remark", "created_at", "updated_at"}); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	for _, a := range accounts {
		if err := writer.Write([]string{
			a.Platform,
			a.Username,
			a.Name,
			ptrString(a.PlatformUserID),
			ptrString(a.Password),
			ptrString(a.Phone),
			ptrString(a.Email),
			ptrString(a.EmailPassword),
			ptrString(a.TwoFactor),
			ptrString(a.BackupCode),
			ptrString(a.EmailClientID),
			ptrString(a.EmailToken),
			ptrString(a.RegistrationIP),
			ptrString(a.AuthCookie),
			ptrString(a.ExecutionAuth),
			ptrString(a.DefaultProxySnapshot),
			a.AccountStatus,
			a.TaskStatus,
			ptrString(a.Remark),
			a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// SubmitTask submits a batch task for social accounts.
func (h *AccountWorkbenchHandler) SubmitTask(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req socialTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.ClientRequestID)
	}
	if strings.TrimSpace(req.TemplateID) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("TASK_TEMPLATE_REQUIRED", "task template is required"))
		return
	}
	if h.templates == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("TASK_TEMPLATE_SERVICE_UNAVAILABLE", "task template service is unavailable"))
		return
	}
	tmpl, err := h.templates.GetTemplate(c.Request.Context(), subject.UserID, req.TemplateID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if result := service.ValidateTaskTemplate(tmpl); !result.Valid {
		response.ErrorFrom(c, infraerrors.BadRequest("TASK_TEMPLATE_INVALID", strings.Join(result.Errors, "; ")))
		return
	}
	result, err := h.workbench.SubmitTask(c.Request.Context(), &service.AccountWorkbenchTaskInput{
		Mode:        service.AccountWorkbenchTaskModeUser,
		UserID:      subject.UserID,
		AccountIDs:  req.AccountIDs,
		Action:      tmpl.Type,
		Target:      nil,
		Content:     nil,
		TargetPool:  tmpl.Params.Targets,
		ContentPool: tmpl.Params.Contents,
		Payload:     socialTaskPayloadFromTemplate(tmpl),
		TemplateSnapshot: &service.SocialTaskTemplateSnapshot{
			TemplateID:   tmpl.ID,
			TemplateName: tmpl.Name,
			TemplateType: tmpl.Type,
			Params:       tmpl.Params,
		},
		IdempotencyKey: idempotencyKey,
	})
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

func socialTaskPayloadFromTemplate(tmpl *service.TaskTemplate) *service.SocialTaskPayload {
	if tmpl == nil {
		return nil
	}
	switch tmpl.Type {
	case service.SocialTaskActionPost:
		payload := &service.SocialTaskPayload{
			Post: &service.SocialPostPayload{
				QuotePostURL: tmpl.Params.QuotePostURL,
				Media:        append([]service.SocialTaskMediaRef(nil), tmpl.Params.Media...),
			},
		}
		if payload.IsZero() {
			return nil
		}
		return payload
	case service.SocialTaskActionUpdateProfile:
		if tmpl.Params.Profile == nil {
			return nil
		}
		profile := *tmpl.Params.Profile
		payload := &service.SocialTaskPayload{Profile: &profile}
		if payload.IsZero() {
			return nil
		}
		return payload
	case service.SocialTaskActionUpdateAvatar:
		if tmpl.Params.Avatar == nil {
			return nil
		}
		avatar := *tmpl.Params.Avatar
		payload := &service.SocialTaskPayload{Avatar: &avatar}
		if payload.IsZero() {
			return nil
		}
		return payload
	case service.SocialTaskActionUpdateBanner:
		if tmpl.Params.Banner == nil {
			return nil
		}
		banner := *tmpl.Params.Banner
		payload := &service.SocialTaskPayload{Banner: &banner}
		if payload.IsZero() {
			return nil
		}
		return payload
	default:
		return nil
	}
}

// SetDefaultProxy stores or clears the default execution proxy snapshot for an assigned account.
// PUT /api/v1/accounts/:id/default-proxy
func (h *AccountWorkbenchHandler) SetDefaultProxy(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		ProxyID *int64 `json:"proxy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var proxy *service.SocialIP
	if req.ProxyID != nil && *req.ProxyID > 0 {
		ip, err := h.ipSvc.GetByIDForUser(c.Request.Context(), *req.ProxyID, subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if err := ensureSocialIPUsable(ip); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		proxy = ip
	}
	account, err := h.svc.SetDefaultProxyForUser(c.Request.Context(), id, subject.UserID, proxy)
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req batchDefaultProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
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
		proxy, err = h.ipSvc.GetByIDForUser(c.Request.Context(), *req.ProxyID, subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	case service.DefaultProxyAssignmentRandom:
		pool, err = h.ipSvc.ListUsableByUser(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	case service.DefaultProxyAssignmentClear:
	default:
		response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_ASSIGNMENT_MODE_INVALID", "proxy assignment mode is invalid"))
		return
	}

	result, err := h.svc.BatchSetDefaultProxyForUser(c.Request.Context(), subject.UserID, req.AccountIDs, req.Mode, proxy, pool)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func respondUserAccountScopeError(c *gin.Context, err error) bool {
	if !errors.Is(err, service.ErrSocialAccountNotAssigned) {
		return false
	}
	response.ErrorFrom(c, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found"))
	return true
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
		TaskMessage:            shortUserTaskResult(account.TaskMessage),
		DefaultProxySnapshot:   account.DefaultProxySnapshot,
		Remark:                 account.Remark,
		DefaultProxyConfigured: strings.TrimSpace(ptrString(account.DefaultProxySnapshot)) != "",
		CreatedAt:              account.CreatedAt,
		UpdatedAt:              account.UpdatedAt,
	}
}

func ensureSocialIPUsable(ip *service.SocialIP) error {
	if ip == nil {
		return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP is not available")
	}
	if ip.Status != service.SocialIPStatusOnline {
		return infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP must pass a connectivity test before execution")
	}
	return nil
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
		Charged:          log.ChargeStatus == service.SocialTaskChargeStatusCharged && log.ChargedAmount > 0,
		ChargedAmount:    log.ChargedAmount,
		ChargeStatus:     log.ChargeStatus,
		ExecutedAt:       log.ExecutedAt,
		CreatedAt:        log.CreatedAt,
	}
	if account != nil {
		item.Platform = account.Platform
		item.AccountName = account.Name
	}
	return item
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
	if containsUnsafeUserTaskResultDetail(message) {
		if isSafeUserTaskSuccessMessage(message) {
			generic := userTaskResultCompletedHidden
			return &generic
		}
		generic := userTaskResultFailedNoCharge
		return &generic
	}
	if !isSafeUserTaskSuccessMessage(message) {
		generic := userTaskResultFailedNoCharge
		return &generic
	}
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
		"该平台动作暂不可用，本次未扣费",
		"执行已完成，但扣费确认异常，请联系管理员处理",
		"账号认证信息不可用，本次未扣费",
		"执行代理不可用，本次未扣费",
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

	normalized := strings.ToLower(message)
	switch {
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
	case strings.Contains(normalized, "media source is not supported yet"):
		return userTaskResultMediaSource
	case strings.Contains(normalized, "video media is not implemented yet"):
		return userTaskResultPostVideo
	case strings.Contains(normalized, "post media content type is not supported"):
		return userTaskResultPostMediaType
	case strings.Contains(normalized, "queue is full"):
		return "任务队列繁忙，本次未扣费"
	case strings.Contains(normalized, "not configured") ||
		strings.Contains(normalized, "executor is not available") ||
		strings.Contains(normalized, "executor queue"):
		return "该平台动作暂不可用，本次未扣费"
	case strings.Contains(normalized, "billing failed"):
		return "执行已完成，但扣费确认异常，请联系管理员处理"
	case strings.Contains(normalized, "auth cookie") ||
		strings.Contains(normalized, "authentication failed") ||
		strings.Contains(normalized, "oauth") ||
		strings.Contains(normalized, "token"):
		return "账号认证信息不可用，本次未扣费"
	case strings.Contains(normalized, "proxy"):
		return "执行代理不可用，本次未扣费"
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

func containsUnsafeUserTaskResultDetail(message string) bool {
	normalized := strings.ToLower(message)
	unsafeMarkers := []string{
		"http://",
		"https://",
		"authorization",
		"bearer ",
		"token",
		"secret",
		"cookie",
		"password",
		"proxy",
		"response body",
		"stack",
		"trace",
		"trace_id",
		"request_id",
		"exception",
		"panic",
	}
	for _, marker := range unsafeMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isSafeUserTaskSuccessMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return normalized == "ok" ||
		strings.Contains(normalized, "succeeded") ||
		strings.Contains(normalized, "success")
}
