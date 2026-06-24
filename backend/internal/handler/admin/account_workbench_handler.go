package admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/Wei-Shaw/socialops/internal/handler/socialaccountcsv"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	maxSocialAccountImportFileBytes = 2 << 20
	maxSocialAccountImportRecords   = 5000
)

// AccountWorkbenchAdminHandler handles admin social account management.
type AccountWorkbenchAdminHandler struct {
	svc       *service.SocialAccountService
	ipSvc     *service.SocialIPService
	billing   *service.SocialBillingService
	executor  *service.SocialTaskExecutor
	workbench *service.AccountWorkbenchService
}

// NewAccountWorkbenchAdminHandler creates a new AccountWorkbenchAdminHandler.
func NewAccountWorkbenchAdminHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor) *AccountWorkbenchAdminHandler {
	return &AccountWorkbenchAdminHandler{svc: svc, ipSvc: ipSvc, billing: billing, executor: executor, workbench: service.NewAccountWorkbenchService(svc, ipSvc, billing, executor)}
}

type socialAdminTaskRequest struct {
	AccountIDs       []int64 `json:"account_ids" binding:"required"`
	Action           string  `json:"action"`
	Target           *string `json:"target"`
	Content          *string `json:"content"`
	ClientRequestID  string  `json:"client_request_id"`
	BillingRequestID string  `json:"billing_request_id"`
}

// List returns a paginated list of social accounts.
// GET /api/v1/admin/accounts
func (h *AccountWorkbenchAdminHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	filters := service.SocialAccountListFilters{
		Platform:      c.Query("platform"),
		AccountStatus: c.Query("account_status"),
		TaskStatus:    c.Query("task_status"),
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
// GET /api/v1/admin/accounts/:id
func (h *AccountWorkbenchAdminHandler) GetByID(c *gin.Context) {
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
// POST /api/v1/admin/accounts
func (h *AccountWorkbenchAdminHandler) Create(c *gin.Context) {
	var input service.CreateSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, adminAccountWorkbenchInputRequiredError())
		return
	}
	account, err := h.svc.Create(c.Request.Context(), &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Update updates a social account.
// PUT /api/v1/admin/accounts/:id
func (h *AccountWorkbenchAdminHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var input service.UpdateSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, adminAccountWorkbenchInputRequiredError())
		return
	}
	input.Name = nil
	input.PlatformUserID = nil
	account, err := h.svc.Update(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Delete deletes a social account.
// DELETE /api/v1/admin/accounts/:id
func (h *AccountWorkbenchAdminHandler) Delete(c *gin.Context) {
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

// GetStats returns summary statistics.
// GET /api/v1/admin/accounts/stats
func (h *AccountWorkbenchAdminHandler) GetStats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// SubmitTask submits social execution tasks from the admin account workbench.
func (h *AccountWorkbenchAdminHandler) SubmitTask(c *gin.Context) {
	var req socialAdminTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, adminAccountWorkbenchInputRequiredError())
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.ClientRequestID)
	}
	result, err := h.workbench.SubmitTask(c.Request.Context(), &service.AccountWorkbenchTaskInput{
		Mode:             service.AccountWorkbenchTaskModeAdmin,
		AccountIDs:       req.AccountIDs,
		Action:           req.Action,
		Target:           req.Target,
		Content:          req.Content,
		IdempotencyKey:   idempotencyKey,
		BillingRequestID: adminOptionalString(req.BillingRequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"submitted": result.Submitted, "enqueued": result.Enqueued, "failed_closed": result.FailedClosed, "logs": result.Logs})
}

// BatchDelete deletes multiple social accounts.
// POST /api/v1/admin/accounts/batch-delete
func (h *AccountWorkbenchAdminHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, adminAccountWorkbenchInputRequiredError())
		return
	}
	result, err := h.svc.BatchDelete(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// StoreWorkbench uploads selected workbench staging accounts into the total pool.
// POST /api/v1/admin/accounts/store-workbench
func (h *AccountWorkbenchAdminHandler) StoreWorkbench(c *gin.Context) {
	var req struct {
		AccountIDs []int64 `json:"account_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, adminAccountWorkbenchInputRequiredError())
		return
	}
	result, err := h.svc.StoreWorkbenchAccounts(c.Request.Context(), req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Import imports social accounts from a JSON, CSV, or XLSX file.
// POST /api/v1/admin/accounts/import
func (h *AccountWorkbenchAdminHandler) Import(c *gin.Context) {
	importPoolAccountsFromRequest(c, h.svc)
}

func importPoolAccountsFromRequest(c *gin.Context, svc *service.SocialAccountService) {
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

	content, err := io.ReadAll(io.LimitReader(file, maxSocialAccountImportFileBytes+1))
	if err != nil {
		response.BadRequest(c, "failed to read social account import file")
		return
	}
	if len(content) > maxSocialAccountImportFileBytes {
		response.BadRequest(c, "social account import file is too large")
		return
	}

	platform := strings.TrimSpace(c.PostForm("platform"))
	if platform == "" {
		platform = "x_twitter"
	}
	inputs, err := parseSocialAccountImportFile(header.Filename, header.Header.Get("Content-Type"), content, platform)
	if err != nil {
		respondSocialAccountImportError(c, err)
		return
	}

	if svc == nil {
		response.ErrorFrom(c, socialAccountServiceUnavailableError())
		return
	}
	result, err := svc.ImportPoolAccounts(c.Request.Context(), inputs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Export exports social accounts as CSV.
// GET /api/v1/admin/accounts/export
func (h *AccountWorkbenchAdminHandler) Export(c *gin.Context) {
	accounts, _, err := h.svc.List(c.Request.Context(), pagination.PaginationParams{Page: 1, PageSize: 10000}, service.SocialAccountListFilters{})
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

func adminAccountWorkbenchInputRequiredError() error {
	return infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
}

func adminAccountImportRequiredError() error {
	return infraerrors.BadRequest("SOCIAL_ACCOUNT_IMPORT_REQUIRED", "social account import input is required")
}

func socialAccountServiceUnavailableError() error {
	return infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE", "social account service is unavailable")
}

func respondSocialAccountImportError(c *gin.Context, err error) {
	if infraerrors.Reason(err) != "" {
		response.ErrorFrom(c, err)
		return
	}
	response.BadRequest(c, err.Error())
}

var socialAccountImportHeaderAliases = map[string][]string{
	"name":                   {"name", "username", "screen_name", "account", "账号", "用户名"},
	"password":               {"password", "密码"},
	"phone":                  {"phone", "mobile", "手机号", "手机"},
	"email":                  {"email", "mail", "邮箱", "邮箱账号"},
	"email_password":         {"email_password", "emailpassword", "mail_password", "mailpassword", "邮箱密码"},
	"two_factor":             {"two_factor", "twofactor", "2fa", "totp", "otp", "两步验证", "二次验证码"},
	"backup_code":            {"backup_code", "backupcode", "backup", "备份码"},
	"email_client_id":        {"email_client_id", "emailclientid", "client_id", "clientid", "邮箱客户端id", "邮箱clientid", "客户端id"},
	"email_token":            {"email_token", "emailtoken", "mail_token", "mailtoken", "token", "邮箱令牌", "邮箱token", "邮箱授权码"},
	"registration_ip":        {"registration_ip", "registrationip", "register_ip", "registerip", "bound_ip", "boundip", "注册ip"},
	"auth_cookie":            {"auth_cookie", "authcookie", "cookie", "认证cookie"},
	"execution_auth":         {"execution_auth", "execution" + "auth", "执行凭证"},
	"default_proxy_snapshot": {"default_proxy_snapshot", "defaultproxysnapshot", "默认代理快照"},
	"remark":                 {"remark", "note", "备注"},
}

func parseSocialAccountImportFile(filename, contentType string, content []byte, platform string) ([]*service.CreateSocialAccountInput, error) {
	normalizedFilename := strings.ToLower(strings.TrimSpace(filename))
	normalizedContentType := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasSuffix(normalizedFilename, ".json") || strings.HasPrefix(normalizedContentType, "application/json"):
		return parseSocialAccountImportJSON(content, platform)
	case strings.HasSuffix(normalizedFilename, ".xlsx") || strings.Contains(normalizedContentType, "spreadsheetml"):
		return parseSocialAccountImportXLSX(content, platform)
	case strings.HasSuffix(normalizedFilename, ".xls"):
		return nil, errors.New("old .xls social account imports are not supported; please use .xlsx, .csv, or .json")
	default:
		return parseSocialAccountImportCSV(content, platform)
	}
}

func parseSocialAccountImportJSON(content []byte, platform string) ([]*service.CreateSocialAccountInput, error) {
	var records []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&records); err != nil {
		return nil, adminAccountImportRequiredError()
	}
	if len(records) > maxSocialAccountImportRecords {
		return nil, errors.New("social account import record limit exceeded")
	}
	inputs := make([]*service.CreateSocialAccountInput, 0, len(records))
	for _, record := range records {
		values := make(map[string]string, len(record))
		for key, raw := range record {
			if field := socialAccountImportFieldForHeader(key); field != "" {
				values[field] = valueToImportString(raw)
			}
		}
		inputs = append(inputs, socialAccountInputFromImportValues(values, platform))
	}
	return inputs, nil
}

func parseSocialAccountImportCSV(content []byte, platform string) ([]*service.CreateSocialAccountInput, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, adminAccountImportRequiredError()
	}
	return socialAccountInputsFromTabularRows(rows, platform)
}

func socialAccountInputsFromFixedXLSXRows(rows [][]string, platform string) ([]*service.CreateSocialAccountInput, error) {
	rawRows := make([][]string, 0, len(rows))
	trimmedRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		raw := make([]string, len(row))
		trimmed := make([]string, len(row))
		hasValue := false
		for i, cell := range row {
			raw[i] = cell
			trimmed[i] = strings.TrimSpace(cell)
			if trimmed[i] != "" {
				hasValue = true
			}
		}
		if hasValue {
			rawRows = append(rawRows, raw)
			trimmedRows = append(trimmedRows, trimmed)
		}
	}
	if len(rawRows) == 0 {
		return nil, nil
	}

	records := rawRows
	if socialAccountImportRowLooksLikeHeader(trimmedRows[0]) {
		records = rawRows[1:]
	}
	if len(records) > maxSocialAccountImportRecords {
		return nil, errors.New("social account import record limit exceeded")
	}

	inputs := make([]*service.CreateSocialAccountInput, 0, len(records))
	for _, row := range records {
		inputs = append(inputs, socialAccountInputFromFixedXLSXRow(row, platform))
	}
	return inputs, nil
}

func parseSocialAccountImportXLSX(content []byte, platform string) ([]*service.CreateSocialAccountInput, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, adminAccountImportRequiredError()
	}
	defer func() { _ = workbook.Close() }()
	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("empty XLSX")
	}
	rows, err := workbook.GetRows(sheets[0])
	if err != nil {
		return nil, adminAccountImportRequiredError()
	}
	return socialAccountInputsFromFixedXLSXRows(rows, platform)
}

func socialAccountInputsFromTabularRows(rows [][]string, platform string) ([]*service.CreateSocialAccountInput, error) {
	rawRows := make([][]string, 0, len(rows))
	trimmedRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		raw := make([]string, len(row))
		trimmed := make([]string, len(row))
		hasValue := false
		for i, cell := range row {
			raw[i] = cell
			trimmed[i] = strings.TrimSpace(cell)
			if trimmed[i] != "" {
				hasValue = true
			}
		}
		if hasValue {
			rawRows = append(rawRows, raw)
			trimmedRows = append(trimmedRows, trimmed)
		}
	}
	if len(rawRows) == 0 {
		return nil, nil
	}

	headerIndex := make(map[string]int)
	for index, header := range trimmedRows[0] {
		if field := socialAccountImportFieldForHeader(header); field != "" {
			if _, exists := headerIndex[field]; !exists {
				headerIndex[field] = index
			}
		}
	}

	hasHeader := len(headerIndex) > 0
	records := rawRows
	if hasHeader {
		records = rawRows[1:]
	}
	if len(records) > maxSocialAccountImportRecords {
		return nil, errors.New("social account import record limit exceeded")
	}

	inputs := make([]*service.CreateSocialAccountInput, 0, len(records))
	for _, row := range records {
		inputs = append(inputs, socialAccountInputFromImportRow(row, headerIndex, hasHeader, platform))
	}
	return inputs, nil
}

func socialAccountInputFromImportRow(row []string, headerIndex map[string]int, hasHeader bool, platform string) *service.CreateSocialAccountInput {
	values := map[string]string{}
	fallback := map[string]int{
		"name":                   0,
		"password":               1,
		"phone":                  2,
		"email":                  3,
		"email_password":         4,
		"auth_cookie":            5,
		"execution_auth":         6,
		"default_proxy_snapshot": 7,
		"remark":                 8,
	}
	for field, fallbackIndex := range fallback {
		index := fallbackIndex
		if hasHeader {
			mapped, ok := headerIndex[field]
			if !ok {
				continue
			}
			index = mapped
		}
		if index >= 0 && index < len(row) {
			values[field] = row[index]
		}
	}
	if hasHeader {
		for field, index := range headerIndex {
			if index >= 0 && index < len(row) {
				values[field] = row[index]
			}
		}
	}
	return socialAccountInputFromImportValues(values, platform)
}

func socialAccountInputFromFixedXLSXRow(row []string, platform string) *service.CreateSocialAccountInput {
	values := map[string]string{}
	fixedColumns := map[string]int{
		"name":            0,
		"password":        1,
		"two_factor":      2,
		"phone":           3,
		"email":           4,
		"email_password":  5,
		"email_client_id": 6,
		"email_token":     7,
	}
	for field, index := range fixedColumns {
		if index >= 0 && index < len(row) {
			values[field] = row[index]
		}
	}
	return socialAccountInputFromImportValues(values, platform)
}

func socialAccountInputFromImportValues(values map[string]string, platform string) *service.CreateSocialAccountInput {
	input := &service.CreateSocialAccountInput{
		Name:     strings.TrimSpace(values["name"]),
		Platform: strings.TrimSpace(platform),
	}
	if input.Platform == "" {
		input.Platform = "x_twitter"
	}
	assignOptionalString := func(field string, setter func(*string)) {
		if value := strings.TrimSpace(values[field]); value != "" {
			setter(&value)
		}
	}
	assignOptionalDeliveryString := func(field string, setter func(*string)) {
		if strings.TrimSpace(values[field]) != "" {
			value := values[field]
			setter(&value)
		}
	}
	assignOptionalDeliveryString("password", func(value *string) { input.Password = value })
	assignOptionalString("phone", func(value *string) { input.Phone = value })
	assignOptionalString("email", func(value *string) { input.Email = value })
	assignOptionalDeliveryString("email_password", func(value *string) { input.EmailPassword = value })
	assignOptionalDeliveryString("two_factor", func(value *string) { input.TwoFactor = value })
	assignOptionalDeliveryString("backup_code", func(value *string) { input.BackupCode = value })
	assignOptionalDeliveryString("email_client_id", func(value *string) { input.EmailClientID = value })
	assignOptionalDeliveryString("email_token", func(value *string) { input.EmailToken = value })
	assignOptionalString("registration_ip", func(value *string) { input.RegistrationIP = value })
	assignOptionalDeliveryString("auth_cookie", func(value *string) { input.AuthCookie = value })
	assignOptionalString("execution_auth", func(value *string) { input.ExecutionAuth = value })
	assignOptionalString("default_proxy_snapshot", func(value *string) { input.DefaultProxySnapshot = value })
	assignOptionalDeliveryString("remark", func(value *string) { input.Remark = value })
	return input
}

func socialAccountImportFieldForHeader(header string) string {
	normalized := normalizeSocialAccountImportHeader(header)
	for field, aliases := range socialAccountImportHeaderAliases {
		for _, alias := range aliases {
			if normalized == normalizeSocialAccountImportHeader(alias) {
				return field
			}
		}
	}
	return ""
}

func socialAccountImportRowLooksLikeHeader(row []string) bool {
	fields := map[string]bool{}
	for _, header := range row {
		if field := socialAccountImportFieldForHeader(header); field != "" {
			fields[field] = true
		}
	}
	return fields["name"] && (fields["password"] || fields["two_factor"] || fields["email"])
}

func normalizeSocialAccountImportHeader(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func valueToImportString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return jsonStringForImportValue(v)
	}
}

func jsonStringForImportValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func adminOptionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
