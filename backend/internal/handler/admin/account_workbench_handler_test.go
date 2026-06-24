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
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
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
	executionAuthSecret := "encrypted-admin-list-execution-auth-ciphertext"
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
	handler := newEncryptedAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})

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

func TestSocialAccountAdminCreatePreservesDeliveryFieldsAndDefaultProxySnapshot(t *testing.T) {
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
		"two_factor": "  totp-secret  ",
		"backup_code": "  backup-secret  ",
		"email_client_id": "  client-id  ",
		"email_token": "  mail-token  ",
		"auth_cookie": "  ct0=create; auth_token=create  ",
		"default_proxy_snapshot": `+strconv.Quote(defaultProxySnapshot)+`
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.Create(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID                   int64   `json:"id"`
			TwoFactor            *string `json:"two_factor"`
			BackupCode           *string `json:"backup_code"`
			EmailClientID        *string `json:"email_client_id"`
			EmailToken           *string `json:"email_token"`
			AuthCookie           *string `json:"auth_cookie"`
			DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "  totp-secret  ", requireStringPtr(t, resp.Data.TwoFactor))
	require.Equal(t, "  backup-secret  ", requireStringPtr(t, resp.Data.BackupCode))
	require.Equal(t, "  client-id  ", requireStringPtr(t, resp.Data.EmailClientID))
	require.Equal(t, "  mail-token  ", requireStringPtr(t, resp.Data.EmailToken))
	require.Equal(t, "  ct0=create; auth_token=create  ", requireStringPtr(t, resp.Data.AuthCookie))
	require.Equal(t, defaultProxySnapshot, requireStringPtr(t, resp.Data.DefaultProxySnapshot))

	stored, err := client.SocialAccount.Get(ctx, resp.Data.ID)
	require.NoError(t, err)
	require.Equal(t, "  totp-secret  ", requireStringPtr(t, stored.TwoFactor))
	require.Equal(t, "  backup-secret  ", requireStringPtr(t, stored.BackupCode))
	require.Equal(t, "  client-id  ", requireStringPtr(t, stored.EmailClientID))
	require.Equal(t, "  mail-token  ", requireStringPtr(t, stored.EmailToken))
	require.Equal(t, "  ct0=create; auth_token=create  ", requireStringPtr(t, stored.AuthCookie))
	require.Equal(t, defaultProxySnapshot, requireStringPtr(t, stored.DefaultProxySnapshot))
}

func TestSocialAccountAdminUpdateKeepsIdentityReadOnlyAndUpdatesRegistrationIP(t *testing.T) {
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
		"two_factor": "  admin-2fa  ",
		"backup_code": "  admin-backup  ",
		"email_client_id": "  admin-client  ",
		"email_token": "  admin-token  ",
		"auth_cookie": "  ct0=admin; auth_token=admin  ",
		"remark": "mutable note"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(account.ID, 10)}}

	handler.Update(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	responseBody := rec.Body.String()
	require.Contains(t, responseBody, `"two_factor":"  admin-2fa  "`)
	require.Contains(t, responseBody, `"backup_code":"  admin-backup  "`)
	require.Contains(t, responseBody, `"email_client_id":"  admin-client  "`)
	require.Contains(t, responseBody, `"email_token":"  admin-token  "`)
	require.Contains(t, responseBody, `"auth_cookie":"  ct0=admin; auth_token=admin  "`)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@admin_readonly", stored.Name)
	require.Equal(t, "admin_readonly", stored.NameKey)
	require.Equal(t, originalIdentityKey, stored.IdentityKey)
	require.Equal(t, "rest-admin-1", requireStringPtr(t, stored.PlatformUserID))
	require.Equal(t, "203.0.113.30", requireStringPtr(t, stored.RegistrationIP))
	require.Equal(t, "new-secret", requireStringPtr(t, stored.Password))
	require.Equal(t, "  admin-2fa  ", requireStringPtr(t, stored.TwoFactor))
	require.Equal(t, "  admin-backup  ", requireStringPtr(t, stored.BackupCode))
	require.Equal(t, "  admin-client  ", requireStringPtr(t, stored.EmailClientID))
	require.Equal(t, "  admin-token  ", requireStringPtr(t, stored.EmailToken))
	require.Equal(t, "  ct0=admin; auth_token=admin  ", requireStringPtr(t, stored.AuthCookie))
	require.Equal(t, "mutable note", requireStringPtr(t, stored.Remark))
}

func TestSocialAccountAdminInputBindingErrorsAreStructured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchAdminHandler(nil, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		pathID string
		body   []byte
		call   gin.HandlerFunc
	}{
		{
			name:   "create malformed json",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts",
			body:   []byte(`{"name":`),
			call:   handler.Create,
		},
		{
			name:   "create wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts",
			body:   []byte(`{"name":123,"platform":"x_twitter"}`),
			call:   handler.Create,
		},
		{
			name:   "update malformed json",
			method: http.MethodPut,
			path:   "/api/v1/admin/accounts/1",
			pathID: "1",
			body:   []byte(`{"password":`),
			call:   handler.Update,
		},
		{
			name:   "update wrong field type",
			method: http.MethodPut,
			path:   "/api/v1/admin/accounts/1",
			pathID: "1",
			body:   []byte(`{"password":123}`),
			call:   handler.Update,
		},
		{
			name:   "submit task malformed json",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/tasks",
			body:   []byte(`{"account_ids":`),
			call:   handler.SubmitTask,
		},
		{
			name:   "submit task wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/tasks",
			body:   []byte(`{"account_ids":"bad","action":"follow"}`),
			call:   handler.SubmitTask,
		},
		{
			name:   "batch delete malformed json",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/batch-delete",
			body:   []byte(`{"ids":`),
			call:   handler.BatchDelete,
		},
		{
			name:   "batch delete wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/batch-delete",
			body:   []byte(`{"ids":"bad"}`),
			call:   handler.BatchDelete,
		},
		{
			name:   "store workbench malformed json",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/store-workbench",
			body:   []byte(`{"account_ids":`),
			call:   handler.StoreWorkbench,
		},
		{
			name:   "store workbench wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/store-workbench",
			body:   []byte(`{"account_ids":"bad"}`),
			call:   handler.StoreWorkbench,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeAdminAccountWorkbenchJSON(t, tt.method, tt.path, tt.pathID, tt.body, tt.call)

			requireStructuredAdminAccountWorkbenchInputError(t, rec)
		})
	}
}

func TestSocialAccountAdminBatchDeleteHardDeletesAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "admin-batch-delete@example.com")
	account := client.SocialAccount.Create().
		SetName("@admin_batch_delete").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("admin_batch_delete").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLoginCheck).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	ledgerRequestID := "social-task:" + strconv.FormatInt(log.ID, 10) + ":wallet"
	ledger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(ledgerRequestID).
		SetModel("social-action").
		SetActualCost(0.05).
		SetTotalCost(0.05).
		SetBillingType(2).
		SaveX(ctx)
	unrelatedLedgerRequestID := "social-task:" + strconv.FormatInt(log.ID+999, 10) + ":wallet"
	unrelatedLedger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(unrelatedLedgerRequestID).
		SetModel("social-action").
		SetActualCost(0.1).
		SetTotalCost(0.1).
		SetBillingType(2).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("admin-batch-delete-proxy").
		SetBoundSocialAccountID(account.ID).
		SaveX(ctx)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 0},
	}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-delete", bytes.NewBufferString(`{"ids":[`+strconv.FormatInt(account.ID, 10)+`,0]}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeBatchSummary(t, rec.Body.Bytes())
	require.Equal(t, float64(2), body["total"])
	require.Equal(t, float64(1), body["succeeded"])
	require.Equal(t, float64(1), body["skipped"])
	require.Equal(t, float64(0), body["failed"])
	_, err := client.SocialAccount.Get(ctx, account.ID)
	require.True(t, dbent.IsNotFound(err), "admin batch delete must physically remove the account")
	logExists, err := client.SocialTaskLog.Query().Where(socialtasklog.IDEQ(log.ID)).Exist(ctx)
	require.NoError(t, err)
	require.False(t, logExists)
	ledgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(ledger.ID), usagelog.RequestIDEQ(ledgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, ledgerExists)
	unrelatedLedgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(unrelatedLedger.ID), usagelog.RequestIDEQ(unrelatedLedgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, unrelatedLedgerExists)
	storedProxy := client.SocialIP.GetX(ctx, proxy.ID)
	require.Nil(t, storedProxy.BoundSocialAccountID)
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

func TestSocialAccountAdminImportRejectsOldXLSFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchAdminHandler(nil, nil, nil, nil)

	body, contentType := multipartBody(t, "accounts.xls", "application/vnd.ms-excel", "not-a-supported-workbook")
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "old .xls social account imports are not supported")
	require.NotContains(t, rec.Body.String(), "leg"+"acy")
}

func TestSocialAccountAdminImportRejectsMalformedJSONWithStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchAdminHandler(nil, nil, nil, nil)

	body, contentType := multipartBody(t, "accounts.json", "application/json", `{"name":`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ctx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ctx)

	requireStructuredSocialAccountImportError(t, rec)
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
		"@admin_import_bound,\"  account-secret  \",+1000000,user@example.com,\"  mail-secret  \",\"  totp-secret  \",\"  backup-1  \",\"  client-id  \",\"  mail-token  \",127.0.0.1,\"  ct0=csv; auth_token=csv  \",encrypted-admin-import-execution-auth-ciphertext,\"" + strings.ReplaceAll(defaultProxySnapshot, `"`, `""`) + "\",\"  delivery remark  \"\n"
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
	require.Equal(t, "  account-secret  ", *imported.Password)
	require.Equal(t, "  mail-secret  ", *imported.EmailPassword)
	require.Equal(t, "  totp-secret  ", *imported.TwoFactor)
	require.Equal(t, "  backup-1  ", *imported.BackupCode)
	require.Equal(t, "  client-id  ", *imported.EmailClientID)
	require.Equal(t, "  mail-token  ", *imported.EmailToken)
	require.Equal(t, "127.0.0.1", *imported.RegistrationIP)
	require.Equal(t, "  ct0=csv; auth_token=csv  ", *imported.AuthCookie)
	require.Equal(t, "encrypted-admin-import-execution-auth-ciphertext", *imported.ExecutionAuth)
	require.Equal(t, "  delivery remark  ", *imported.Remark)
}

func TestSocialAccountAdminImportJSONPreservesDeliveryFieldWhitespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	jsonBody := `[{
		"name": "@admin_json_bound",
		"platform": "x_twitter",
		"password": "  json-secret  ",
		"two_factor": "  json-2fa  ",
		"auth_cookie": "  ct0=json; auth_token=json  ",
		"remark": "  json delivery note  "
	}]`
	body, contentType := multipartBody(t, "accounts.json", "application/json", jsonBody)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"created":1`)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("admin_json_bound")).
		OnlyX(ctx)
	require.Equal(t, "  json-secret  ", requireStringPtr(t, imported.Password))
	require.Equal(t, "  json-2fa  ", requireStringPtr(t, imported.TwoFactor))
	require.Equal(t, "  ct0=json; auth_token=json  ", requireStringPtr(t, imported.AuthCookie))
	require.Equal(t, "  json delivery note  ", requireStringPtr(t, imported.Remark))
}

func TestSocialAccountAdminImportDeduplicatesChineseXLSXByUsernameFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	xlsxBytes := minimalXLSX(t, [][]string{
		{"账号", "密码", "2FA", "手机号", "邮箱账号", "邮箱密码", "邮箱 Client ID", "邮箱 Token"},
		{"@Admin_XLSX_Dedupe", "account-secret", "TOTP-SECRET", "+15550001111", "mail@example.test", "mail-secret", "client-id", "mail-token"},
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
	require.Equal(t, "+15550001111", requireStringPtr(t, imported.Phone))
	require.Equal(t, "mail@example.test", requireStringPtr(t, imported.Email))
	require.Equal(t, "mail-secret", requireStringPtr(t, imported.EmailPassword))
	require.Equal(t, "client-id", requireStringPtr(t, imported.EmailClientID))
	require.Equal(t, "mail-token", requireStringPtr(t, imported.EmailToken))
	require.Nil(t, imported.BackupCode)
	require.Nil(t, imported.RegistrationIP)
	require.Nil(t, imported.AuthCookie)

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

func TestSocialAccountAdminImportXLSXUsesFixedColumnsInsteadOfHeaderNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := newAccountWorkbenchAdminHandlerForTest(client, &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{}})
	xlsxBytes := minimalXLSX(t, [][]string{
		{"密码", "账号", "2FA", "手机号", "邮箱账号", "邮箱密码", "邮箱 Client ID", "邮箱 Token"},
		{"@Admin_XLSX_Fixed", "account-secret", "TOTP-SECRET", "+15550002222", "fixed@example.test", "mail-secret", "client-id", "mail-token"},
	})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	body, contentType := multipartBytes(t, "accounts.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"created":1`)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("admin_xlsx_fixed")).
		OnlyX(ctx)
	require.Equal(t, "account-secret", requireStringPtr(t, imported.Password))
	require.Equal(t, "TOTP-SECRET", requireStringPtr(t, imported.TwoFactor))
	require.Equal(t, "+15550002222", requireStringPtr(t, imported.Phone))
	require.Equal(t, "fixed@example.test", requireStringPtr(t, imported.Email))
	require.Equal(t, "mail-secret", requireStringPtr(t, imported.EmailPassword))
	require.Equal(t, "client-id", requireStringPtr(t, imported.EmailClientID))
	require.Equal(t, "mail-token", requireStringPtr(t, imported.EmailToken))
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
		SetExecutionAuth("encrypted-admin-export-execution-auth-ciphertext").
		SaveX(ctx)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/export", nil)

	handler.Export(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "auth_cookie")
	require.Contains(t, body, "ct0=export; auth_token=export")
	require.Contains(t, body, "encrypted-admin-export-execution-auth-ciphertext")
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

func TestSocialAccountAdminRejectsUnsupportedActionBeforeBilling(t *testing.T) {
	cases := map[string]string{
		"blank":               "",
		"removed_tweet_alias": "tweet",
		"removed_dm_alias":    "dm",
		"message":             "message",
		"unsupported":         "unsupported_action",
	}
	for name, action := range cases {
		t.Run(name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx := context.Background()
			client := newProxyAdminTestClient(t)
			user := createProxyAdminUser(t, ctx, client, name+"-admin-unsupported-action@example.com")
			account := client.SocialAccount.Create().
				SetName("@admin_" + name + "_unsupported_action").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("admin_" + name + "_unsupported_action").
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
				"action": "`+action+`",
				"target": "@target",
				"content": "hello"
			}`))
			ginCtx.Request.Header.Set("Content-Type", "application/json")

			handler.SubmitTask(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "SOCIAL_TASK_UNSUPPORTED_ACTION")
			require.Zero(t, userRepo.deductCalls)
			count, err := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestSocialAccountAdminRejectsMissingActionWithUnsupportedActionCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "missing-admin-task-action@example.com")
	userRepo := &socialAccountAdminBillingUserRepo{users: map[int64]*service.User{
		user.ID: {ID: user.ID, Balance: 1.0},
	}}
	handler := newAccountWorkbenchAdminHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [1],
		"target": "@target",
		"content": "hello"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_UNSUPPORTED_ACTION")
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

func invokeAdminAccountWorkbenchJSON(t *testing.T, method, path, pathID string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	if pathID != "" {
		ginCtx.Params = gin.Params{{Key: "id", Value: pathID}}
	}
	fn(ginCtx)
	return rec
}

func requireStructuredAdminAccountWorkbenchInputError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "SOCIAL_ACCOUNT_INPUT_REQUIRED")
	require.Contains(t, body, "social account input is required")
	require.NotContains(t, body, "unexpected EOF")
	require.NotContains(t, body, "invalid character")
	require.NotContains(t, body, "cannot unmarshal")
}

func requireStructuredSocialAccountImportError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "SOCIAL_ACCOUNT_IMPORT_REQUIRED")
	require.Contains(t, body, "social account import input is required")
	require.NotContains(t, body, "invalid JSON")
	require.NotContains(t, body, "invalid character")
	require.NotContains(t, body, "unexpected EOF")
	require.NotContains(t, body, "cannot unmarshal")
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

func newEncryptedAccountWorkbenchAdminHandlerForTest(client *dbent.Client, userRepo *socialAccountAdminBillingUserRepo) *AccountWorkbenchAdminHandler {
	subRepo := &socialAccountAdminSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewAccountWorkbenchAdminHandler(
		service.NewSocialAccountServiceWithCredentialEncryptor(client, adminExecutionAuthEncryptor{}),
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
