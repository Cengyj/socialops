package admin

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	maxSocialAccountImportFileBytes = 2 << 20
	maxSocialAccountImportRecords   = 5000
)

// SocialAccountAdminHandler handles admin social account management.
type SocialAccountAdminHandler struct {
	svc      *service.SocialAccountService
	ipSvc    *service.SocialIPService
	billing  *service.SocialBillingService
	executor *service.SocialTaskExecutor
}

// NewSocialAccountAdminHandler creates a new SocialAccountAdminHandler.
func NewSocialAccountAdminHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor) *SocialAccountAdminHandler {
	return &SocialAccountAdminHandler{svc: svc, ipSvc: ipSvc, billing: billing, executor: executor}
}

type socialAdminTaskRequest struct {
	AccountIDs       []int64 `json:"account_ids" binding:"required"`
	Action           string  `json:"action" binding:"required"`
	Target           *string `json:"target"`
	Content          *string `json:"content"`
	ProxyID          *int64  `json:"proxy_id"`
	ClientRequestID  string  `json:"client_request_id"`
	BillingRequestID string  `json:"billing_request_id"`
}

// List returns a paginated list of social accounts.
// GET /api/v1/admin/social-accounts
func (h *SocialAccountAdminHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	filters := service.SocialAccountListFilters{
		Platform:      c.Query("platform"),
		AccountStatus: c.Query("account_status"),
		TaskStatus:    c.Query("task_status"),
		Source:        c.Query("source"),
	}
	if c.Query("unassigned") == "true" {
		filters.UnassignedOnly = true
	}

	accounts, result, err := h.svc.List(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, accounts, result.Total, page, pageSize)
}

// GetByID returns a social account by ID.
// GET /api/v1/admin/social-accounts/:id
func (h *SocialAccountAdminHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	account, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Create creates a new social account.
// POST /api/v1/admin/social-accounts
func (h *SocialAccountAdminHandler) Create(c *gin.Context) {
	var input service.CreateSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.Create(c.Request.Context(), &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Register fails closed until a real platform registrar is configured.
// POST /api/v1/admin/social-accounts/register
func (h *SocialAccountAdminHandler) Register(c *gin.Context) {
	response.ErrorFrom(c, infraerrors.ServiceUnavailable(
		"SOCIAL_ACCOUNT_REGISTRAR_NOT_CONFIGURED",
		"social account registrar is not configured; no account was created",
	))
}

// Update updates a social account.
// PUT /api/v1/admin/social-accounts/:id
func (h *SocialAccountAdminHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var input service.UpdateSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.Update(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Delete deletes a social account.
// DELETE /api/v1/admin/social-accounts/:id
func (h *SocialAccountAdminHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// Assign assigns a social account to a user.
// POST /api/v1/admin/social-accounts/:id/assign
func (h *SocialAccountAdminHandler) Assign(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.Assign(c.Request.Context(), id, req.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Reclaim removes the user assignment from a social account.
// POST /api/v1/admin/social-accounts/:id/reclaim
func (h *SocialAccountAdminHandler) Reclaim(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	account, err := h.svc.Reclaim(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// SetDefaultProxy stores or clears an assigned account's default execution proxy.
// PUT /api/v1/admin/social-accounts/:id/default-proxy
func (h *SocialAccountAdminHandler) SetDefaultProxy(c *gin.Context) {
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
		account, err := h.svc.GetByID(c.Request.Context(), id)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if account.AssignedUserID == nil {
			response.ErrorFrom(c, service.ErrSocialAccountNotAssigned)
			return
		}
		ip, err := h.ipSvc.GetByIDForUser(c.Request.Context(), *req.ProxyID, *account.AssignedUserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if ip.Status != service.SocialIPStatusOnline {
			response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP must be online"))
			return
		}
		proxy = ip
	}
	account, err := h.svc.SetDefaultProxyForAdmin(c.Request.Context(), id, proxy)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// GetStats returns summary statistics.
// GET /api/v1/admin/social-accounts/stats
func (h *SocialAccountAdminHandler) GetStats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// EstimateTask estimates successful-execution billing for selected assigned accounts.
// POST /api/v1/admin/social-accounts/tasks/estimate
func (h *SocialAccountAdminHandler) EstimateTask(c *gin.Context) {
	var req socialAdminTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := normalizeAndValidateAdminTaskRequest(&req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	counts, err := h.validateAdminTaskAccounts(c, req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	estimates := make(map[int64]*service.SocialBillingEstimate, len(counts))
	for userID, count := range counts {
		estimate, err := h.billing.Estimate(c.Request.Context(), userID, count)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		estimates[userID] = estimate
	}
	response.Success(c, gin.H{"action": req.Action, "estimates": estimates})
}

// SubmitTask submits social execution tasks from the admin account workbench.
// POST /api/v1/admin/social-accounts/tasks
func (h *SocialAccountAdminHandler) SubmitTask(c *gin.Context) {
	var req socialAdminTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := normalizeAndValidateAdminTaskRequest(&req); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.ClientRequestID)
	}

	type pendingTask struct {
		accountID     int64
		userID        int64
		proxyID       *int64
		proxySnapshot *string
	}
	pending := make([]pendingTask, 0, len(req.AccountIDs))
	logs := make([]any, 0, len(req.AccountIDs))
	counts := make(map[int64]int)

	for _, accountID := range req.AccountIDs {
		account, err := h.svc.GetByID(c.Request.Context(), accountID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if account.AssignedUserID == nil {
			response.ErrorFrom(c, service.ErrSocialAccountNotAssigned)
			return
		}
		if account.AccountStatus != service.SocialAccountStatusAvailable {
			response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_AVAILABLE", "account is not available for execution"))
			return
		}
		ownerID := *account.AssignedUserID
		if idempotencyKey != "" {
			existing, err := h.svc.FindTaskLogByIdempotency(c.Request.Context(), ownerID, accountID, req.Action, idempotencyKey)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			if existing != nil {
				logs = append(logs, existing)
				continue
			}
		}

		taskProxyID, taskProxySnapshot, err := h.resolveAdminTaskProxy(c, ownerID, req.ProxyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if taskProxySnapshot == nil && account.BoundIP != nil && strings.TrimSpace(*account.BoundIP) != "" {
			taskProxyID, taskProxySnapshot, err = h.resolveAdminAccountDefaultProxy(c, ownerID, account)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
		}

		counts[ownerID]++
		pending = append(pending, pendingTask{
			accountID:     accountID,
			userID:        ownerID,
			proxyID:       taskProxyID,
			proxySnapshot: taskProxySnapshot,
		})
	}

	estimates := make(map[int64]*service.SocialBillingEstimate, len(counts))
	for userID, count := range counts {
		estimate, err := h.billing.EnsureCanAfford(c.Request.Context(), userID, count)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		estimates[userID] = estimate
	}

	taskLogIDs := make([]int64, 0, len(pending))
	for _, task := range pending {
		log, err := h.svc.CreateTaskLog(c.Request.Context(), &service.CreateSocialTaskLogInput{
			AccountID:        task.accountID,
			UserID:           task.userID,
			Action:           req.Action,
			Target:           req.Target,
			Content:          req.Content,
			Status:           service.SocialTaskLogStatusPending,
			ProxyID:          task.proxyID,
			ProxySnapshot:    task.proxySnapshot,
			BillingRequestID: adminOptionalString(req.BillingRequestID),
			IdempotencyKey:   adminOptionalString(idempotencyKey),
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
					logs = replaceAdminTaskLog(logs, log)
				}
			}
			failedClosed = len(taskLogIDs)
		} else {
			enqueued = h.executor.EnqueueBatch(taskLogIDs)
			if enqueued < len(taskLogIDs) {
				message := "social platform executor queue is full; task was not charged"
				for _, taskLogID := range taskLogIDs[enqueued:] {
					if log, err := h.svc.MarkTaskLogFailedNotCharged(c.Request.Context(), taskLogID, message); err == nil {
						logs = replaceAdminTaskLog(logs, log)
					}
				}
				failedClosed = len(taskLogIDs) - enqueued
			}
		}
	}

	response.Success(c, gin.H{"submitted": len(logs), "enqueued": enqueued, "failed_closed": failedClosed, "billing_estimates": estimates, "logs": logs})
}

// ListTaskLogs returns recent social execution logs for administrators.
// GET /api/v1/admin/social-accounts/tasks
func (h *SocialAccountAdminHandler) ListTaskLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	logs, result, err := h.svc.ListAllTaskLogs(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, logs, result.Total, page, pageSize)
}

// BatchDelete deletes multiple social accounts.
// POST /api/v1/admin/social-accounts/batch-delete
func (h *SocialAccountAdminHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	for _, id := range req.IDs {
		if err := h.svc.Delete(c.Request.Context(), id); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	response.Success(c, gin.H{"deleted": len(req.IDs)})
}

// Import imports social accounts from a CSV or JSON file.
// POST /api/v1/admin/social-accounts/import
func (h *SocialAccountAdminHandler) Import(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxSocialAccountImportFileBytes {
		response.BadRequest(c, "social account import file is too large")
		return
	}

	var inputs []*service.CreateSocialAccountInput

	contentType := strings.ToLower(header.Header.Get("Content-Type"))
	limitedFile := io.LimitReader(file, maxSocialAccountImportFileBytes+1)
	if strings.HasPrefix(contentType, "application/json") || strings.HasSuffix(strings.ToLower(header.Filename), ".json") {
		// JSON import
		var records []struct {
			Name          string  `json:"name"`
			Platform      string  `json:"platform"`
			AccountID     *string `json:"account_id"`
			Password      *string `json:"password"`
			Phone         *string `json:"phone"`
			Email         *string `json:"email"`
			EmailPassword *string `json:"email_password"`
			BoundIP       *string `json:"bound_ip"`
			Remark        *string `json:"remark"`
		}
		if err := json.NewDecoder(limitedFile).Decode(&records); err != nil {
			response.BadRequest(c, "invalid JSON: "+err.Error())
			return
		}
		if len(records) > maxSocialAccountImportRecords {
			response.BadRequest(c, "social account import record limit exceeded")
			return
		}
		for _, r := range records {
			inputs = append(inputs, &service.CreateSocialAccountInput{
				Name:          r.Name,
				Platform:      r.Platform,
				AccountID:     r.AccountID,
				Password:      r.Password,
				Phone:         r.Phone,
				Email:         r.Email,
				EmailPassword: r.EmailPassword,
				Source:        service.SocialAccountSourceFileUpload,
				BoundIP:       r.BoundIP,
				Remark:        r.Remark,
			})
		}
	} else {
		// CSV import: name,platform,account_id,password,phone,email,email_password,bound_ip,remark
		reader := csv.NewReader(limitedFile)
		// Skip header row
		if _, err := reader.Read(); err != nil {
			response.BadRequest(c, "empty or invalid CSV")
			return
		}
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				response.BadRequest(c, "CSV parse error: "+err.Error())
				return
			}
			if len(record) < 2 {
				continue
			}
			if len(inputs) >= maxSocialAccountImportRecords {
				response.BadRequest(c, "social account import record limit exceeded")
				return
			}
			input := &service.CreateSocialAccountInput{
				Name:     record[0],
				Platform: record[1],
				Source:   service.SocialAccountSourceFileUpload,
			}
			if len(record) > 2 && record[2] != "" {
				v := record[2]
				input.AccountID = &v
			}
			if len(record) > 3 && record[3] != "" {
				v := record[3]
				input.Password = &v
			}
			if len(record) > 4 && record[4] != "" {
				v := record[4]
				input.Phone = &v
			}
			if len(record) > 5 && record[5] != "" {
				v := record[5]
				input.Email = &v
			}
			if len(record) > 6 && record[6] != "" {
				v := record[6]
				input.EmailPassword = &v
			}
			if len(record) > 7 && record[7] != "" {
				v := record[7]
				input.BoundIP = &v
			}
			if len(record) > 8 && record[8] != "" {
				v := record[8]
				input.Remark = &v
			}
			inputs = append(inputs, input)
		}
	}

	result, err := h.svc.ImportPoolAccounts(c.Request.Context(), inputs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Export exports social accounts as CSV.
// GET /api/v1/admin/social-accounts/export
func (h *SocialAccountAdminHandler) Export(c *gin.Context) {
	accounts, _, err := h.svc.List(c.Request.Context(), pagination.PaginationParams{Page: 1, PageSize: 10000}, service.SocialAccountListFilters{})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=social_accounts.csv")

	writer := csv.NewWriter(c.Writer)
	// Header
	if err := writer.Write([]string{"platform", "name", "account_id", "password", "phone", "email", "email_password", "bound_ip", "account_status", "task_status", "source", "remark", "created_at", "updated_at"}); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	for _, a := range accounts {
		row := []string{
			a.Platform,
			a.Name,
			ptrStr(a.AccountID),
			ptrStr(a.Password),
			ptrStr(a.Phone),
			ptrStr(a.Email),
			ptrStr(a.EmailPassword),
			ptrStr(a.BoundIP),
			a.AccountStatus,
			a.TaskStatus,
			a.Source,
			ptrStr(a.Remark),
			a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if err := writer.Write(row); err != nil {
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

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func normalizeAndValidateAdminTaskRequest(req *socialAdminTaskRequest) error {
	if req == nil {
		return infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	action, ok := service.NormalizeSocialTaskAction(req.Action)
	if !ok {
		return service.ErrSocialTaskUnsupportedAction
	}
	req.Action = action
	if len(req.AccountIDs) == 0 {
		return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNTS_REQUIRED", "at least one social account is required")
	}
	for _, accountID := range req.AccountIDs {
		if accountID <= 0 {
			return infraerrors.BadRequest("SOCIAL_TASK_ACCOUNT_ID_INVALID", "social account id must be positive")
		}
	}
	req.AccountIDs = service.UniqueInt64sPreserveOrder(req.AccountIDs)
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

func (h *SocialAccountAdminHandler) validateAdminTaskAccounts(c *gin.Context, accountIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int)
	for _, accountID := range accountIDs {
		account, err := h.svc.GetByID(c.Request.Context(), accountID)
		if err != nil {
			return nil, err
		}
		if account.AssignedUserID == nil {
			return nil, service.ErrSocialAccountNotAssigned
		}
		if account.AccountStatus != service.SocialAccountStatusAvailable {
			return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_AVAILABLE", "account is not available for execution")
		}
		counts[*account.AssignedUserID]++
	}
	return counts, nil
}

func (h *SocialAccountAdminHandler) resolveAdminTaskProxy(c *gin.Context, ownerID int64, requestedProxyID *int64) (*int64, *string, error) {
	if requestedProxyID == nil {
		return nil, nil, nil
	}
	ip, err := h.ipSvc.GetByIDForUser(c.Request.Context(), *requestedProxyID, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if ip.Status != service.SocialIPStatusOnline {
		return nil, nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP must be online")
	}
	snapshot := service.SocialIPTaskSnapshot(ip)
	return requestedProxyID, &snapshot, nil
}

func (h *SocialAccountAdminHandler) resolveAdminAccountDefaultProxy(c *gin.Context, ownerID int64, account *service.SocialAccount) (*int64, *string, error) {
	defaultProxyID, ok := service.SocialIPIDFromSnapshot(strings.TrimSpace(*account.BoundIP))
	if !ok {
		return nil, nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "default social IP snapshot is stale")
	}
	return h.resolveAdminTaskProxy(c, ownerID, &defaultProxyID)
}

func adminOptionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func replaceAdminTaskLog(logs []any, replacement *service.SocialTaskLog) []any {
	if replacement == nil {
		return logs
	}
	for i, item := range logs {
		if existing, ok := item.(*service.SocialTaskLog); ok && existing.ID == replacement.ID {
			logs[i] = replacement
			break
		}
	}
	return logs
}
