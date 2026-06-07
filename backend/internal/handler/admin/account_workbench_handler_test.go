//go:build unit

package admin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSocialAccountAdminListNormalizesInvalidPaginationAndPreservesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	accountID := "pool-account-id"
	password := "pool-secret"
	phone := "+15550000001"
	email := "pool@example.com"
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=admin-list; auth_token=admin-list"
	executionAuthSecret := "execution-secret"
	defaultProxySnapshot := `{"id":2,"endpoint":"http://proxy.local:8080"}`
	remark := "delivery remark"
	account := client.SocialAccount.Create().
		SetName("@admin_pool_list").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_pool_list").
		SetPlatformUserID(accountID).
		SetPassword(password).
		SetPhone(phone).
		SetEmail(email).
		SetEmailPassword(emailPassword).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetDefaultProxySnapshot(defaultProxySnapshot).
		SetRemark(remark).
		SaveX(ctx)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=0&page_size=0", nil)

	handler.List(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID                   int64   `json:"id"`
				PlatformUserID       *string `json:"platform_user_id"`
				Password             *string `json:"password"`
				Phone                *string `json:"phone"`
				Email                *string `json:"email"`
				EmailPassword        *string `json:"email_password"`
				AuthCookie           *string `json:"auth_cookie"`
				ExecutionAuth        *string `json:"execution_auth"`
				DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
				Remark               *string `json:"remark"`
			} `json:"items"`
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Pages    int   `json:"pages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Equal(t, 1, resp.Data.Page)
	require.Equal(t, 20, resp.Data.PageSize)
	require.Equal(t, 1, resp.Data.Pages)
	require.Len(t, resp.Data.Items, 1)
	item := resp.Data.Items[0]
	require.Equal(t, account.ID, item.ID)
	require.Equal(t, accountID, requireStringPtr(t, item.PlatformUserID))
	require.Equal(t, password, requireStringPtr(t, item.Password))
	require.Equal(t, phone, requireStringPtr(t, item.Phone))
	require.Equal(t, email, requireStringPtr(t, item.Email))
	require.Equal(t, emailPassword, requireStringPtr(t, item.EmailPassword))
	require.Equal(t, authCookieSecret, requireStringPtr(t, item.AuthCookie))
	require.Equal(t, executionAuthSecret, requireStringPtr(t, item.ExecutionAuth))
	require.Equal(t, defaultProxySnapshot, requireStringPtr(t, item.DefaultProxySnapshot))
	require.Equal(t, remark, requireStringPtr(t, item.Remark))
}

func TestSocialAccountAdminCreatePreservesDefaultProxySnapshotField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	defaultProxySnapshot := `{"id":3,"endpoint":"http://create-proxy.local:8080"}`

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewBufferString(`{
		"name": "@admin_pool_create",
		"platform": "x_twitter",
		"password": "account-secret",
		"two_factor": "totp-secret",
		"default_proxy_snapshot": `+strconv.Quote(defaultProxySnapshot)+`
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.Create(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID                   int64   `json:"id"`
			DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, defaultProxySnapshot, requireStringPtr(t, resp.Data.DefaultProxySnapshot))

	stored, err := client.SocialAccount.Get(ctx, resp.Data.ID)
	require.NoError(t, err)
	require.Equal(t, defaultProxySnapshot, requireStringPtr(t, stored.DefaultProxySnapshot))
}

func TestSocialAccountAdminUpdateKeepsIdentityAndRegistrationIPReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	account := client.SocialAccount.Create().
		SetName("@admin_readonly").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_readonly").
		SetIdentityKind("username").
		SetIdentityKey("x_twitter:username:admin_readonly").
		SetPlatformUserID("rest-admin-1").
		SetRegistrationIP("198.51.100.30").
		SetPassword("old-secret").
		SaveX(ctx)
	originalIdentityKey := account.IdentityKey

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/"+strconv.FormatInt(account.ID, 10), bytes.NewBufferString(`{
		"name": "@admin_renamed",
		"platform_user_id": "fake-rest",
		"registration_ip": "203.0.113.30",
		"password": "new-secret",
		"remark": "mutable note"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(account.ID, 10)}}

	handler.Update(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@admin_readonly", stored.Name)
	require.Equal(t, "admin_readonly", stored.NameKey)
	require.Equal(t, originalIdentityKey, stored.IdentityKey)
	require.Equal(t, "rest-admin-1", requireStringPtr(t, stored.PlatformUserID))
	require.Equal(t, "198.51.100.30", requireStringPtr(t, stored.RegistrationIP))
	require.Equal(t, "new-secret", requireStringPtr(t, stored.Password))
	require.Equal(t, "mutable note", requireStringPtr(t, stored.Remark))
}

func TestSocialAccountAdminImportRejectsOversizedFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchAdminHandler(nil, nil, nil, nil)

	body, contentType := multipartBody(t, "huge.csv", "text/csv", strings.Repeat("x", maxSocialAccountImportFileBytes+1))
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "too large")
}

func TestSocialAccountAdminImportRejectsTooManyCSVRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchAdminHandler(nil, nil, nil, nil)

	var csv strings.Builder
	csv.WriteString("name,platform\n")
	for i := 0; i <= maxSocialAccountImportRecords; i++ {
		csv.WriteString("@user")
		csv.WriteString(strconv.Itoa(i))
		csv.WriteString(",x_twitter\n")
	}
	body, contentType := multipartBody(t, "accounts.csv", "text/csv", csv.String())
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "record limit exceeded")
}

func TestSocialAccountAdminImportPreservesStructuredCredentialFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	defaultProxySnapshot := `{"id":123,"name":"default-proxy","endpoint":"http://proxy.local:8080"}`
	csv := "name,password,phone,email,email_password,two_factor,backup_code,email_client_id,email_token,registration_ip,auth_cookie,execution_auth,default_proxy_snapshot,remark\n" +
		"@admin_import_bound,account-secret,+1000000,user@example.com,mail-secret,totp-secret,backup-1,client-id,mail-token,127.0.0.1,ct0=csv; auth_token=csv,execution-secret,\"" + strings.ReplaceAll(defaultProxySnapshot, `"`, `""`) + "\",delivery remark\n"
	body, contentType := multipartBody(t, "accounts.csv", "text/csv", csv)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"created":1`)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("admin_import_bound")).
		OnlyX(ctx)
	require.NotNil(t, imported.DefaultProxySnapshot)
	require.Equal(t, defaultProxySnapshot, *imported.DefaultProxySnapshot)
	require.Nil(t, imported.PlatformUserID)
	require.Equal(t, "account-secret", *imported.Password)
	require.Equal(t, "mail-secret", *imported.EmailPassword)
	require.Equal(t, "totp-secret", *imported.TwoFactor)
	require.Equal(t, "backup-1", *imported.BackupCode)
	require.Equal(t, "client-id", *imported.EmailClientID)
	require.Equal(t, "mail-token", *imported.EmailToken)
	require.Equal(t, "127.0.0.1", *imported.RegistrationIP)
	require.Equal(t, "ct0=csv; auth_token=csv", *imported.AuthCookie)
	require.Equal(t, "execution-secret", *imported.ExecutionAuth)
}

func TestSocialAccountAdminImportDeduplicatesChineseXLSXByUsernameFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	xlsxBytes := minimalXLSX(t, [][]string{
		{"账号", "密码", "2FA", "备份码", "邮箱账号", "邮箱密码", "邮箱客户端ID", "邮箱令牌", "注册IP", "Cookie"},
		{"@Admin_XLSX_Dedupe", "account-secret", "TOTP-SECRET", "backup-1", "mail@example.test", "mail-secret", "client-id", "mail-token", "127.0.0.1", "ct0=xlsx; auth_token=xlsx"},
	})

	firstRec := httptest.NewRecorder()
	firstCtx, _ := gin.CreateTestContext(firstRec)
	firstBody, firstContentType := multipartBytes(t, "accounts.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
	firstCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", firstBody)
	firstCtx.Request.Header.Set("Content-Type", firstContentType)

	handler.Import(firstCtx)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Contains(t, firstRec.Body.String(), `"created":1`)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("admin_xlsx_dedupe")).
		OnlyX(ctx)
	require.Equal(t, "x_twitter", imported.PlatformKey)
	require.Equal(t, "username", imported.IdentityKind)
	require.Equal(t, "admin_xlsx_dedupe", imported.IdentityKey)
	require.Nil(t, imported.PlatformUserID)
	require.Equal(t, "account-secret", requireStringPtr(t, imported.Password))
	require.Equal(t, "TOTP-SECRET", requireStringPtr(t, imported.TwoFactor))
	require.Equal(t, "backup-1", requireStringPtr(t, imported.BackupCode))
	require.Equal(t, "mail-secret", requireStringPtr(t, imported.EmailPassword))
	require.Equal(t, "client-id", requireStringPtr(t, imported.EmailClientID))
	require.Equal(t, "mail-token", requireStringPtr(t, imported.EmailToken))
	require.Equal(t, "127.0.0.1", requireStringPtr(t, imported.RegistrationIP))
	require.Equal(t, "ct0=xlsx; auth_token=xlsx", requireStringPtr(t, imported.AuthCookie))

	secondRec := httptest.NewRecorder()
	secondCtx, _ := gin.CreateTestContext(secondRec)
	secondBody, secondContentType := multipartBytes(t, "accounts.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
	secondCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", secondBody)
	secondCtx.Request.Header.Set("Content-Type", secondContentType)

	handler.Import(secondCtx)

	require.Equal(t, http.StatusOK, secondRec.Code)
	require.Contains(t, secondRec.Body.String(), `"created":0`)
	require.Contains(t, secondRec.Body.String(), `"skipped":1`)
	require.Contains(t, secondRec.Body.String(), `"duplicates":1`)
	require.Contains(t, secondRec.Body.String(), `"reason":"duplicate_in_database"`)
	count, err := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("admin_xlsx_dedupe")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSocialAccountAdminExportIncludesAuthCookieField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})

	client.SocialAccount.Create().
		SetName("@admin_export_cookie").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_export_cookie").
		SetPassword("account-secret").
		SetAuthCookie("ct0=export; auth_token=export").
		SetExecutionAuth("execution-secret").
		SaveX(ctx)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/export", nil)

	handler.Export(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "auth_cookie")
	require.Contains(t, body, "ct0=export; auth_token=export")
	require.Contains(t, body, "execution-secret")
}

func TestSocialAccountAdminSubmitTaskDeduplicatesAccountIDsWithoutIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-submit-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_submit_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_submit_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newAccountWorkbenchAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+strconv.FormatInt(account.ID, 10)+`, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"submitted":1`)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	require.Zero(t, userRepo.deductCalls)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSocialAccountAdminRejectsMixedPlatformBatchBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-mixed-platform@example.com")
	xAccount := client.SocialAccount.Create().
		SetName("@admin_mixed_x").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_mixed_x").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	instagramAccount := client.SocialAccount.Create().
		SetName("@admin_mixed_instagram").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("admin_mixed_instagram").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newAccountWorkbenchAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+strconv.FormatInt(xAccount.ID, 10)+`, `+strconv.FormatInt(instagramAccount.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_MIXED_PLATFORMS")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountAdminRejectsUnavailableMessageActionBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-message-unavailable@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_message_unavailable").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_message_unavailable").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newAccountWorkbenchAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+strconv.FormatInt(account.ID, 10)+`],
		"action": "message",
		"target": "@target",
		"content": "hello"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACTION_UNAVAILABLE")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSocialAccountAdminSubmitTaskRejectsNonPositiveAccountIDWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-submit-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_submit_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_submit_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newAccountWorkbenchAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [-1, `+strconv.FormatInt(account.ID, 10)+`],
		"action": "follow",
		"target": "@target"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func requireStringPtr(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func multipartBody(t *testing.T, filename, contentType, content string) (*bytes.Buffer, string) {
	return multipartBytes(t, filename, contentType, []byte(content))
}

func multipartBytes(t *testing.T, filename, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func minimalXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var body bytes.Buffer
	archive := zip.NewWriter(&body)
	writeZipFile(t, archive, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`)
	writeZipFile(t, archive, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`)
	writeZipFile(t, archive, "xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="accounts" sheetId="1" r:id="rId1"/></sheets>
</workbook>`)
	writeZipFile(t, archive, "xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`)
	writeZipFile(t, archive, "xl/worksheets/sheet1.xml", minimalSheetXML(rows))
	require.NoError(t, archive.Close())
	return body.Bytes()
}

func writeZipFile(t *testing.T, archive *zip.Writer, name, content string) {
	t.Helper()
	file, err := archive.Create(name)
	require.NoError(t, err)
	_, err = file.Write([]byte(content))
	require.NoError(t, err)
}

func minimalSheetXML(rows [][]string) string {
	var out strings.Builder
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		out.WriteString(`<row r="`)
		out.WriteString(strconv.Itoa(rowIndex + 1))
		out.WriteString(`">`)
		for columnIndex, value := range row {
			out.WriteString(`<c r="`)
			out.WriteString(excelColumnName(columnIndex))
			out.WriteString(strconv.Itoa(rowIndex + 1))
			out.WriteString(`" t="inlineStr"><is><t>`)
			out.WriteString(xmlEscape(value))
			out.WriteString(`</t></is></c>`)
		}
		out.WriteString(`</row>`)
	}
	out.WriteString(`</sheetData></worksheet>`)
	return out.String()
}

func excelColumnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return replacer.Replace(value)
}

func newAccountWorkbenchAdminHandlerForTest(client *dbent.Client, userRepo *socialAccountAdminBillingUserRepo) *AccountWorkbenchAdminHandler {
	subRepo := &socialAccountAdminSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewAccountWorkbenchAdminHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		nil,
	)
}

type socialAccountAdminBillingUserRepo struct {
	service.UserRepository
	users       map[int64]*service.User
	deductCalls int
}

func (r *socialAccountAdminBillingUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	user := r.users[id]
	if user == nil {
		return nil, service.ErrUserNotFound
	}
	out := *user
	return &out, nil
}

func (r *socialAccountAdminBillingUserRepo) DeductBalance(_ context.Context, id int64, amount float64) error {
	user := r.users[id]
	if user == nil {
		return service.ErrUserNotFound
	}
	r.deductCalls++
	user.Balance -= amount
	return nil
}

type socialAccountAdminSubscriptionRepo struct {
	service.UserSubscriptionRepository
}

func (r *socialAccountAdminSubscriptionRepo) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

func (r *socialAccountAdminSubscriptionRepo) IncrementUsage(context.Context, int64, float64) error {
	return nil
}
