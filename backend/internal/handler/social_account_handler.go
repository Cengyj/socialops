package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// SocialAccountHandler handles user-facing social account operations.
type SocialAccountHandler struct {
	svc      *service.SocialAccountService
	ipSvc    *service.SocialIPService
	billing  *service.SocialBillingService
	executor *service.SocialTaskExecutor
}

// NewSocialAccountHandler creates a new SocialAccountHandler.
func NewSocialAccountHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor) *SocialAccountHandler {
	return &SocialAccountHandler{svc: svc, ipSvc: ipSvc, billing: billing, executor: executor}
}

type socialTaskRequest struct {
	AccountIDs       []int64 `json:"account_ids" binding:"required"`
	Action           string  `json:"action" binding:"required"`
	Target           *string `json:"target"`
	Content          *string `json:"content"`
	ProxyID          *int64  `json:"proxy_id"`
	ClientRequestID  string  `json:"client_request_id"`
	BillingRequestID string  `json:"billing_request_id"`
}

// ListMyAccounts returns social accounts assigned to the current user.
// GET /api/v1/social-accounts
func (h *SocialAccountHandler) ListMyAccounts(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	accounts, result, err := h.svc.ListByUser(c.Request.Context(), subject.UserID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, accounts, result.Total, page, pageSize)
}

// ImportMyAccount binds an existing total-pool account to the current user.
// POST /api/v1/social-accounts/import
func (h *SocialAccountHandler) ImportMyAccount(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var input service.UserImportSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.ImportForUser(c.Request.Context(), subject.UserID, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// ExportMyAccounts exports the current user's assigned accounts with full credentials.
// GET /api/v1/social-accounts/export
func (h *SocialAccountHandler) ExportMyAccounts(c *gin.Context) {
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
	c.Header("Content-Disposition", "attachment; filename=social_accounts.csv")
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"platform", "name", "account_id", "password", "phone", "email", "email_password", "bound_ip", "account_status", "task_status", "source", "remark", "created_at", "updated_at"})
	for _, a := range accounts {
		_ = writer.Write([]string{
			a.Platform,
			a.Name,
			ptrString(a.AccountID),
			ptrString(a.Password),
			ptrString(a.Phone),
			ptrString(a.Email),
			ptrString(a.EmailPassword),
			ptrString(a.BoundIP),
			a.AccountStatus,
			a.TaskStatus,
			a.Source,
			ptrString(a.Remark),
			a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writer.Flush()
	c.Status(http.StatusOK)
}

// EstimateTask estimates successful-execution billing without charging.
// POST /api/v1/social-accounts/tasks/estimate
func (h *SocialAccountHandler) EstimateTask(c *gin.Context) {
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
	var actionOK bool
	req.Action, actionOK = service.NormalizeSocialTaskAction(req.Action)
	if !actionOK {
		response.ErrorFrom(c, service.ErrSocialTaskUnsupportedAction)
		return
	}
	if err := validateSocialTaskRequest(req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	req.AccountIDs = service.UniqueInt64sPreserveOrder(req.AccountIDs)
	if _, err := h.validateTaskAccounts(c, subject.UserID, req.AccountIDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	estimate, err := h.billing.Estimate(c.Request.Context(), subject.UserID, len(req.AccountIDs))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, estimate)
}

// SubmitTask submits a batch task for social accounts.
// POST /api/v1/social-accounts/tasks
func (h *SocialAccountHandler) SubmitTask(c *gin.Context) {
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
	var actionOK bool
	req.Action, actionOK = service.NormalizeSocialTaskAction(req.Action)
	if !actionOK {
		response.ErrorFrom(c, service.ErrSocialTaskUnsupportedAction)
		return
	}
	if err := validateSocialTaskRequest(req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	req.AccountIDs = service.UniqueInt64sPreserveOrder(req.AccountIDs)

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.ClientRequestID)
	}

	proxyID, proxySnapshot, err := h.resolveTaskProxy(c, subject.UserID, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	logs := make([]any, 0, len(req.AccountIDs))
	type pendingTask struct {
		accountID     int64
		proxyID       *int64
		proxySnapshot *string
	}
	pending := make([]pendingTask, 0, len(req.AccountIDs))
	for _, accountID := range req.AccountIDs {
		account, err := h.svc.GetByID(c.Request.Context(), accountID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if account.AssignedUserID == nil || *account.AssignedUserID != subject.UserID {
			response.BadRequest(c, "account not assigned to you")
			return
		}
		if account.AccountStatus != service.SocialAccountStatusAvailable {
			response.BadRequest(c, "account is not available for execution")
			return
		}
		if idempotencyKey != "" {
			existing, err := h.svc.FindTaskLogByIdempotency(c.Request.Context(), subject.UserID, accountID, req.Action, idempotencyKey)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			if existing != nil {
				logs = append(logs, existing)
				continue
			}
		}

		taskProxyID := proxyID
		taskProxySnapshot := proxySnapshot
		if taskProxySnapshot == nil && account.BoundIP != nil && strings.TrimSpace(*account.BoundIP) != "" {
			taskProxyID, taskProxySnapshot, err = h.resolveAccountDefaultProxy(c, subject.UserID, account)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
		}
		pending = append(pending, pendingTask{
			accountID:     accountID,
			proxyID:       taskProxyID,
			proxySnapshot: taskProxySnapshot,
		})
	}

	estimate, err := h.billing.EnsureCanAfford(c.Request.Context(), subject.UserID, len(pending))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	taskLogIDs := make([]int64, 0, len(pending))
	for _, task := range pending {
		log, err := h.svc.CreateTaskLog(c.Request.Context(), &service.CreateSocialTaskLogInput{
			AccountID:        task.accountID,
			UserID:           subject.UserID,
			Action:           req.Action,
			Target:           req.Target,
			Content:          req.Content,
			Status:           service.SocialTaskLogStatusPending,
			ProxyID:          task.proxyID,
			ProxySnapshot:    task.proxySnapshot,
			BillingRequestID: optionalString(req.BillingRequestID),
			IdempotencyKey:   optionalString(idempotencyKey),
		})
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		logs = append(logs, log)
		taskLogIDs = append(taskLogIDs, log.ID)
	}

	enqueued := 0
	failedClosed := 0
	if len(taskLogIDs) > 0 {
		if h.executor == nil {
			message := "social platform executor queue is not configured; task was not charged"
			for _, taskLogID := range taskLogIDs {
				if log, err := h.svc.MarkTaskLogFailedNotCharged(c.Request.Context(), taskLogID, message); err == nil {
					logs = replaceTaskLog(logs, log)
				}
			}
			failedClosed = len(taskLogIDs)
		} else {
			enqueued = h.executor.EnqueueBatch(taskLogIDs)
			if enqueued < len(taskLogIDs) {
				message := "social platform executor queue is full; task was not charged"
				for _, taskLogID := range taskLogIDs[enqueued:] {
					if log, err := h.svc.MarkTaskLogFailedNotCharged(c.Request.Context(), taskLogID, message); err == nil {
						logs = replaceTaskLog(logs, log)
					}
				}
				failedClosed = len(taskLogIDs) - enqueued
			}
		}
	}

	response.Success(c, gin.H{"submitted": len(logs), "enqueued": enqueued, "failed_closed": failedClosed, "billing_estimate": estimate, "logs": logs})
}

// ListMyTaskLogs returns task execution logs for the current user.
// GET /api/v1/social-accounts/tasks
func (h *SocialAccountHandler) ListMyTaskLogs(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	logs, result, err := h.svc.ListTaskLogs(c.Request.Context(), subject.UserID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, logs, result.Total, page, pageSize)
}

// SetDefaultProxy stores or clears the default execution proxy snapshot for an assigned account.
// PUT /api/v1/social-accounts/:id/default-proxy
func (h *SocialAccountHandler) SetDefaultProxy(c *gin.Context) {
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
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *SocialAccountHandler) validateTaskAccounts(c *gin.Context, userID int64, accountIDs []int64) ([]*service.SocialAccount, error) {
	accounts := make([]*service.SocialAccount, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		account, err := h.svc.GetByID(c.Request.Context(), accountID)
		if err != nil {
			return nil, err
		}
		if account.AssignedUserID == nil || *account.AssignedUserID != userID {
			return nil, service.ErrSocialAccountNotAssigned
		}
		if account.AccountStatus != service.SocialAccountStatusAvailable {
			return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_AVAILABLE", "account is not available for execution")
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func validateSocialTaskRequest(req socialTaskRequest) error {
	if len(req.AccountIDs) == 0 {
		return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNTS_REQUIRED", "at least one social account is required")
	}
	for _, accountID := range req.AccountIDs {
		if accountID <= 0 {
			return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNT_ID_INVALID", "social account id must be positive")
		}
	}
	if !service.IsBillableSocialTaskAction(req.Action) {
		return service.ErrSocialTaskUnsupportedAction
	}
	switch req.Action {
	case service.SocialTaskActionMessage:
		if req.Target == nil || strings.TrimSpace(*req.Target) == "" {
			return infraerrors.BadRequest("SOCIAL_TASK_TARGET_REQUIRED", "target is required for messages")
		}
		if req.Content == nil || strings.TrimSpace(*req.Content) == "" {
			return infraerrors.BadRequest("SOCIAL_TASK_CONTENT_REQUIRED", "content is required for messages")
		}
	case service.SocialTaskActionFollow, service.SocialTaskActionLike:
		if req.Target == nil || strings.TrimSpace(*req.Target) == "" {
			return infraerrors.BadRequest("SOCIAL_TASK_TARGET_REQUIRED", "target is required for this action")
		}
	case service.SocialTaskActionPost:
		if req.Content == nil || strings.TrimSpace(*req.Content) == "" {
			return infraerrors.BadRequest("SOCIAL_TASK_CONTENT_REQUIRED", "content is required for posts")
		}
	}
	return nil
}

func (h *SocialAccountHandler) resolveTaskProxy(c *gin.Context, userID int64, requestedProxyID *int64) (*int64, *string, error) {
	if requestedProxyID == nil {
		return nil, nil, nil
	}
	ip, err := h.ipSvc.GetByIDForUser(c.Request.Context(), *requestedProxyID, userID)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureSocialIPUsable(ip); err != nil {
		return nil, nil, err
	}
	snapshot := service.SocialIPTaskSnapshot(ip)
	return requestedProxyID, &snapshot, nil
}

func (h *SocialAccountHandler) resolveAccountDefaultProxy(c *gin.Context, userID int64, account *service.SocialAccount) (*int64, *string, error) {
	if account == nil || account.BoundIP == nil || strings.TrimSpace(*account.BoundIP) == "" {
		return nil, nil, nil
	}
	defaultProxyID, ok := service.SocialIPIDFromSnapshot(strings.TrimSpace(*account.BoundIP))
	if !ok {
		return nil, nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "default social IP snapshot is stale")
	}
	ip, err := h.ipSvc.GetByIDForUser(c.Request.Context(), defaultProxyID, userID)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureSocialIPUsable(ip); err != nil {
		return nil, nil, err
	}
	snapshot := service.SocialIPTaskSnapshot(ip)
	return &defaultProxyID, &snapshot, nil
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

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func replaceTaskLog(logs []any, replacement *service.SocialTaskLog) []any {
	if replacement == nil {
		return logs
	}
	for i, item := range logs {
		if existing, ok := item.(*service.SocialTaskLog); ok && existing.ID == replacement.ID {
			logs[i] = replacement
			return logs
		}
	}
	return logs
}
