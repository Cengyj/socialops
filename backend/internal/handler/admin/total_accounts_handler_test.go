//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTotalAccountsHandlerBatchOperationsReturnCommercialSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "total-batch-user@example.com")
	unassignedA := client.SocialAccount.Create().
		SetName("@total_batch_a").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_batch_a").
		SetIdentityKind("username").
		SetIdentityKey("total_batch_a").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	unassignedB := client.SocialAccount.Create().
		SetName("@total_batch_b").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_batch_b").
		SetIdentityKind("username").
		SetIdentityKey("total_batch_b").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assigned := client.SocialAccount.Create().
		SetName("@total_batch_assigned").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_batch_assigned").
		SetIdentityKind("username").
		SetIdentityKey("total_batch_assigned").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(`{"id":1}`).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	alreadyUnassigned := client.SocialAccount.Create().
		SetName("@total_batch_already_unassigned").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_batch_already_unassigned").
		SetIdentityKind("username").
		SetIdentityKey("total_batch_already_unassigned").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountServiceWithCredentialEncryptor(client, adminExecutionAuthEncryptor{}))

	assignRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-assign", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedB.ID)+`,`+formatAdminID(assigned.ID)+`],"user_id":`+formatAdminID(user.ID)+`}`), handler.BatchAssign)
	require.Equal(t, http.StatusOK, assignRec.Code)
	assignBody := decodeBatchSummary(t, assignRec.Body.Bytes())
	require.Equal(t, float64(4), assignBody["total"])
	require.Equal(t, float64(2), assignBody["succeeded"])
	require.Equal(t, float64(2), assignBody["skipped"])
	require.Equal(t, float64(0), assignBody["failed"])
	require.NotEmpty(t, assignBody["items"])
	require.Contains(t, batchItemReasons(t, assignBody), "duplicate_in_batch")

	reclaimRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-reclaim", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(assigned.ID)+`,`+formatAdminID(alreadyUnassigned.ID)+`,0]}`), handler.BatchReclaim)
	require.Equal(t, http.StatusOK, reclaimRec.Code)
	reclaimBody := decodeBatchSummary(t, reclaimRec.Body.Bytes())
	require.Equal(t, float64(5), reclaimBody["total"])
	require.Equal(t, float64(2), reclaimBody["succeeded"])
	require.Equal(t, float64(3), reclaimBody["skipped"])
	require.Equal(t, float64(0), reclaimBody["failed"])
	require.Contains(t, batchItemReasons(t, reclaimBody), "already_unassigned")
	require.Contains(t, batchItemReasons(t, reclaimBody), "duplicate_in_batch")
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).AssignedUserID)
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).DefaultProxySnapshot)

	deleteRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-delete", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedB.ID)+`,0]}`), handler.BatchDelete)
	require.Equal(t, http.StatusOK, deleteRec.Code)
	deleteBody := decodeBatchSummary(t, deleteRec.Body.Bytes())
	require.Equal(t, float64(4), deleteBody["total"])
	require.Equal(t, float64(2), deleteBody["succeeded"])
	require.Equal(t, float64(2), deleteBody["skipped"])
	require.Equal(t, float64(0), deleteBody["failed"])
	require.Contains(t, batchItemReasons(t, deleteBody), "duplicate_in_batch")
	_, err := client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), unassignedA.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), unassignedB.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestTotalAccountsHandlerListIncludesAssignedUserEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "total-list-owner@example.com")
	account := client.SocialAccount.Create().
		SetName("@total_list_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_list_owner").
		SetIdentityKind("username").
		SetIdentityKey("total_list_owner").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountServiceWithCredentialEncryptor(client, adminExecutionAuthEncryptor{}))

	rec := invokeTotalAccountsJSON(t, http.MethodGet, "/api/v1/admin/total-accounts?page=1&page_size=20", nil, handler.List)
	require.Equal(t, http.StatusOK, rec.Code)
	items := decodePaginatedItems(t, rec.Body.Bytes())
	require.Len(t, items, 1)
	require.Equal(t, float64(account.ID), items[0]["id"])
	require.Equal(t, float64(user.ID), items[0]["assigned_user_id"])
	require.Equal(t, "total-list-owner@example.com", items[0]["assigned_user_email"])
}

func TestTotalAccountsHandlerListAppliesSearchFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	owner := createProxyAdminUser(t, ctx, client, "total-search-owner@example.com")
	match := client.SocialAccount.Create().
		SetName("@total_search_match").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_search_match").
		SetIdentityKind("username").
		SetIdentityKey("total_search_match").
		SetAssignedUserID(owner.ID).
		SetPassword("pool-delivery-secret").
		SetDefaultProxySnapshot(`{"endpoint":"http://total-search-proxy.example:8080"}`).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@total_search_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_search_other").
		SetIdentityKind("username").
		SetIdentityKey("total_search_other").
		SetPassword("other-delivery-secret").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountServiceWithCredentialEncryptor(client, adminExecutionAuthEncryptor{}))

	for _, query := range []string{"total-search-owner", "pool-delivery-secret", "total-search-proxy", "#" + formatAdminID(match.ID)} {
		rec := invokeTotalAccountsJSON(t, http.MethodGet, "/api/v1/admin/total-accounts?page=1&page_size=20&assigned=true&search="+query, nil, handler.List)

		require.Equal(t, http.StatusOK, rec.Code)
		items := decodePaginatedItems(t, rec.Body.Bytes())
		require.Len(t, items, 1)
		require.Equal(t, float64(match.ID), items[0]["id"])
		require.Equal(t, "total-search-owner@example.com", items[0]["assigned_user_email"])
	}
}

func TestTotalAccountsHandlerListTrimsStatusAndSearchFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	match := client.SocialAccount.Create().
		SetName("@total_trimmed_filter").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_trimmed_filter").
		SetIdentityKind("username").
		SetIdentityKey("total_trimmed_filter").
		SetPassword("trimmed-filter-secret").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@total_trimmed_filter_wrong_status").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_trimmed_filter_wrong_status").
		SetIdentityKind("username").
		SetIdentityKey("total_trimmed_filter_wrong_status").
		SetPassword("trimmed-filter-secret").
		SetAccountStatus(service.SocialAccountStatusLimited).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	rec := invokeTotalAccountsJSON(t, http.MethodGet, "/api/v1/admin/total-accounts?page=1&page_size=20&search=%20trimmed-filter-secret%20&account_status=%20available%20&task_status=%20stored%20&platform=%20x_twitter%20", nil, handler.List)

	require.Equal(t, http.StatusOK, rec.Code)
	requireSinglePaginatedAdminID(t, rec.Body.Bytes(), match.ID)
}

func TestTotalAccountsHandlerImportCreatesTotalPoolAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))
	csv := "name,password,two_factor,email,remark\n" +
		"@total_import_route,account-secret,totp-secret,total-import@example.test,imported through total pool\n"
	body, contentType := multipartBody(t, "total-accounts.csv", "text/csv", csv)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/total-accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	summary := decodeBatchSummary(t, rec.Body.Bytes())
	require.Equal(t, float64(1), summary["created"])
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("total_import_route")).
		OnlyX(ctx)
	require.Nil(t, imported.AssignedUserID)
	require.Equal(t, service.SocialTaskStatusStored, imported.TaskStatus)
	require.Equal(t, "imported through total pool", *imported.Remark)
}

func TestTotalAccountsHandlerImportRejectsMalformedJSONWithStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newProxyAdminTestClient(t)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))
	body, contentType := multipartBody(t, "total-accounts.json", "application/json", `{"name":`)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/total-accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	requireStructuredSocialAccountImportError(t, rec)
}

func TestTotalAccountsHandlerUpdateEditsOnlyTotalPoolMutableFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "total-update-owner@example.com")
	pool := client.SocialAccount.Create().
		SetName("@total_update_pool").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_update_pool").
		SetIdentityKind("username").
		SetIdentityKey("total_update_pool").
		SetPlatformUserID("immutable-platform-id").
		SetAssignedUserID(user.ID).
		SetPassword("old-password").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	staging := client.SocialAccount.Create().
		SetName("@total_update_staging").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_update_staging").
		SetIdentityKind("username").
		SetIdentityKey("total_update_staging").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusNotStored).
		SetTaskStatus(service.SocialTaskStatusPending).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	updateRec := invokeTotalAccountsJSON(t, http.MethodPut, "/api/v1/admin/total-accounts/"+formatAdminID(pool.ID), []byte(`{
		"name":"@should_not_change",
		"platform_user_id":"should-not-change",
		"password":"new-password",
		"email":"updated@example.test",
		"account_status":"limited",
		"remark":"updated from total pool"
	}`), handler.Update)

	require.Equal(t, http.StatusOK, updateRec.Code)
	updated := client.SocialAccount.GetX(ctx, pool.ID)
	require.Equal(t, "@total_update_pool", updated.Name)
	require.Equal(t, "immutable-platform-id", *updated.PlatformUserID)
	require.Equal(t, "new-password", *updated.Password)
	require.Equal(t, "updated@example.test", *updated.Email)
	require.Equal(t, service.SocialAccountStatusLimited, updated.AccountStatus)
	require.Equal(t, "updated from total pool", *updated.Remark)

	stagingRec := invokeTotalAccountsJSON(t, http.MethodPut, "/api/v1/admin/total-accounts/"+formatAdminID(staging.ID), []byte(`{"password":"must-not-change"}`), handler.Update)
	require.Equal(t, http.StatusNotFound, stagingRec.Code)
	require.Nil(t, client.SocialAccount.GetX(ctx, staging.ID).Password)
}

func TestTotalAccountsHandlerOwnershipOperationsRejectWorkbenchStagingAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	owner := createProxyAdminUser(t, ctx, client, "total-staging-owner@example.com")
	target := createProxyAdminUser(t, ctx, client, "total-staging-target@example.com")
	staging := client.SocialAccount.Create().
		SetName("@total_staging_guard").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_staging_guard").
		SetIdentityKind("username").
		SetIdentityKey("total_staging_guard").
		SetAssignedUserID(owner.ID).
		SetAccountStatus(service.SocialAccountStatusNotStored).
		SetTaskStatus(service.SocialTaskStatusPending).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	assignRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/"+formatAdminID(staging.ID)+"/assign", []byte(`{"user_id":`+formatAdminID(target.ID)+`}`), handler.Assign)
	require.Equal(t, http.StatusNotFound, assignRec.Code)
	require.Equal(t, owner.ID, *client.SocialAccount.GetX(ctx, staging.ID).AssignedUserID)

	reclaimRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/"+formatAdminID(staging.ID)+"/reclaim", []byte(`{}`), handler.Reclaim)
	require.Equal(t, http.StatusNotFound, reclaimRec.Code)
	require.Equal(t, owner.ID, *client.SocialAccount.GetX(ctx, staging.ID).AssignedUserID)
}

func TestTotalAccountsHandlerRejectsInvalidPathIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newProxyAdminTestClient(t)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	cases := []struct {
		name string
		path string
		body []byte
		fn   gin.HandlerFunc
	}{
		{
			name: "update",
			path: "/api/v1/admin/total-accounts/not-a-number",
			body: []byte(`{"password":"must-not-parse"}`),
			fn:   handler.Update,
		},
		{
			name: "assign",
			path: "/api/v1/admin/total-accounts/not-a-number/assign",
			body: []byte(`{"user_id":1}`),
			fn:   handler.Assign,
		},
		{
			name: "reclaim",
			path: "/api/v1/admin/total-accounts/not-a-number/reclaim",
			body: []byte(`{}`),
			fn:   handler.Reclaim,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeTotalAccountsJSON(t, http.MethodPost, tt.path, tt.body, tt.fn)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "invalid id")
		})
	}
}

func TestTotalAccountsHandlerReportsServiceUnavailableWhenDependencyIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTotalAccountsHandler(nil)

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
		fn     gin.HandlerFunc
	}{
		{
			name:   "list",
			method: http.MethodGet,
			path:   "/api/v1/admin/total-accounts?page=1&page_size=20",
			fn:     handler.List,
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/api/v1/admin/total-accounts/1",
			body:   []byte(`{"password":"updated-secret"}`),
			fn:     handler.Update,
		},
		{
			name:   "export",
			method: http.MethodGet,
			path:   "/api/v1/admin/total-accounts/export",
			fn:     handler.Export,
		},
		{
			name:   "assign",
			method: http.MethodPost,
			path:   "/api/v1/admin/total-accounts/1/assign",
			body:   []byte(`{"user_id":1}`),
			fn:     handler.Assign,
		},
		{
			name:   "reclaim",
			method: http.MethodPost,
			path:   "/api/v1/admin/total-accounts/1/reclaim",
			fn:     handler.Reclaim,
		},
		{
			name:   "batch assign",
			method: http.MethodPost,
			path:   "/api/v1/admin/total-accounts/batch-assign",
			body:   []byte(`{"ids":[1],"user_id":1}`),
			fn:     handler.BatchAssign,
		},
		{
			name:   "batch reclaim",
			method: http.MethodPost,
			path:   "/api/v1/admin/total-accounts/batch-reclaim",
			body:   []byte(`{"ids":[1]}`),
			fn:     handler.BatchReclaim,
		},
		{
			name:   "batch delete",
			method: http.MethodPost,
			path:   "/api/v1/admin/total-accounts/batch-delete",
			body:   []byte(`{"ids":[1]}`),
			fn:     handler.BatchDelete,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeTotalAccountsJSON(t, tt.method, tt.path, tt.body, tt.fn)

			requireTotalAccountServiceUnavailableError(t, rec)
		})
	}

	body, contentType := multipartBody(t, "total-accounts.csv", "text/csv", "@total_service_unavailable,account-secret")
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/total-accounts/import", body)
	ginCtx.Request.Header.Set("Content-Type", contentType)

	handler.Import(ginCtx)

	requireTotalAccountServiceUnavailableError(t, rec)
}

func TestTotalAccountsHandlerInputBindingErrorsAreStructured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newProxyAdminTestClient(t)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	cases := []struct {
		name string
		path string
		body []byte
		fn   gin.HandlerFunc
	}{
		{
			name: "update malformed json",
			path: "/api/v1/admin/total-accounts/1",
			body: []byte(`{"password":`),
			fn:   handler.Update,
		},
		{
			name: "update wrong field type",
			path: "/api/v1/admin/total-accounts/1",
			body: []byte(`{"password":123}`),
			fn:   handler.Update,
		},
		{
			name: "assign malformed json",
			path: "/api/v1/admin/total-accounts/1/assign",
			body: []byte(`{"user_id":`),
			fn:   handler.Assign,
		},
		{
			name: "assign wrong field type",
			path: "/api/v1/admin/total-accounts/1/assign",
			body: []byte(`{"user_id":"bad"}`),
			fn:   handler.Assign,
		},
		{
			name: "batch assign malformed json",
			path: "/api/v1/admin/total-accounts/batch-assign",
			body: []byte(`{"ids":`),
			fn:   handler.BatchAssign,
		},
		{
			name: "batch assign wrong field type",
			path: "/api/v1/admin/total-accounts/batch-assign",
			body: []byte(`{"ids":[1],"user_id":"bad"}`),
			fn:   handler.BatchAssign,
		},
		{
			name: "batch reclaim malformed json",
			path: "/api/v1/admin/total-accounts/batch-reclaim",
			body: []byte(`{"ids":`),
			fn:   handler.BatchReclaim,
		},
		{
			name: "batch reclaim wrong field type",
			path: "/api/v1/admin/total-accounts/batch-reclaim",
			body: []byte(`{"ids":"not-array"}`),
			fn:   handler.BatchReclaim,
		},
		{
			name: "batch delete malformed json",
			path: "/api/v1/admin/total-accounts/batch-delete",
			body: []byte(`{"ids":`),
			fn:   handler.BatchDelete,
		},
		{
			name: "batch delete wrong field type",
			path: "/api/v1/admin/total-accounts/batch-delete",
			body: []byte(`{"ids":"not-array"}`),
			fn:   handler.BatchDelete,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeTotalAccountsJSON(t, http.MethodPost, tt.path, tt.body, tt.fn)

			requireStructuredTotalAccountInputError(t, rec)
		})
	}
}

func TestTotalAccountsHandlerBatchAssignRejectsMissingTargetUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	account := client.SocialAccount.Create().
		SetName("@total_batch_missing_user").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_batch_missing_user").
		SetIdentityKind("username").
		SetIdentityKey("total_batch_missing_user").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	rec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-assign", []byte(`{"ids":[`+formatAdminID(account.ID)+`],"user_id":404404}`), handler.BatchAssign)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeBatchSummary(t, rec.Body.Bytes())
	require.Equal(t, float64(1), body["total"])
	require.Equal(t, float64(0), body["succeeded"])
	require.Equal(t, float64(0), body["skipped"])
	require.Equal(t, float64(1), body["failed"])
	require.Contains(t, batchItemReasons(t, body), "target_user_not_found")
	require.Contains(t, batchErrors(t, body), "target user not found")
	require.Nil(t, client.SocialAccount.GetX(ctx, account.ID).AssignedUserID)
}

func TestTotalAccountsHandlerExportUsesTotalPoolAndPreservesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	user := createProxyAdminUser(t, ctx, client, "total-export-staging-owner@example.com")
	selectedAccount := client.SocialAccount.Create().
		SetName("@total_export_cookie").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_export_cookie").
		SetIdentityKind("username").
		SetIdentityKey("total_export_cookie").
		SetPassword("account-secret").
		SetEmailPassword("email-secret").
		SetAuthCookie("ct0=export; auth_token=export").
		SetExecutionAuth("encrypted-total-export-execution-auth-ciphertext").
		SetDefaultProxySnapshot(`{"id":301,"endpoint":"http://proxy.example:8080"}`).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@total_export_cookie_unselected").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_export_cookie_unselected").
		SetIdentityKind("username").
		SetIdentityKey("total_export_cookie_unselected").
		SetPassword("unselected-secret").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@total_export_limited").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_export_limited").
		SetIdentityKind("username").
		SetIdentityKey("total_export_limited").
		SetPassword("limited-secret").
		SetAccountStatus(service.SocialAccountStatusLimited).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@workbench_staging_export").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_staging_export").
		SetIdentityKind("username").
		SetIdentityKey("workbench_staging_export").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusNotStored).
		SetTaskStatus(service.SocialTaskStatusPending).
		SaveX(ctx)
	handler := NewTotalAccountsHandler(service.NewSocialAccountServiceWithCredentialEncryptor(client, adminExecutionAuthEncryptor{}))

	rec := invokeTotalAccountsJSON(t, http.MethodGet, "/api/v1/admin/total-accounts/export?search=total_export_cookie&account_status=available&unassigned=true&account_ids="+strconv.FormatInt(selectedAccount.ID, 10), nil, handler.Export)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "total_accounts.csv")
	rows, err := csv.NewReader(bytes.NewReader(rec.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)
	header := rows[0]
	require.Contains(t, header, "auth_cookie")
	require.Contains(t, header, "execution_auth")
	require.Contains(t, header, "default_proxy_snapshot")

	nameIndex := csvHeaderIndex(t, header, "name")
	rowByName := make(map[string][]string)
	for _, row := range rows[1:] {
		rowByName[row[nameIndex]] = row
	}
	exported := rowByName["@total_export_cookie"]
	require.NotNil(t, exported)
	require.Equal(t, "account-secret", exported[csvHeaderIndex(t, header, "password")])
	require.Equal(t, "email-secret", exported[csvHeaderIndex(t, header, "email_password")])
	require.Equal(t, "ct0=export; auth_token=export", exported[csvHeaderIndex(t, header, "auth_cookie")])
	require.NotContains(t, exported[csvHeaderIndex(t, header, "execution_auth")], "secret-access")
	require.NotContains(t, exported[csvHeaderIndex(t, header, "execution_auth")], "token_secret")
	require.Equal(t, "encrypted-total-export-execution-auth-ciphertext", exported[csvHeaderIndex(t, header, "execution_auth")])
	require.Equal(t, `{"id":301,"endpoint":"http://proxy.example:8080"}`, exported[csvHeaderIndex(t, header, "default_proxy_snapshot")])
	require.NotContains(t, rowByName, "@total_export_limited")
	require.NotContains(t, rowByName, "@workbench_staging_export")
	require.NotContains(t, rowByName, "@total_export_cookie_unselected")

	stagingCount, err := client.SocialAccount.Query().
		Where(socialaccount.NameEQ("@workbench_staging_export")).
		Count(mixins.SkipSoftDelete(ctx))
	require.NoError(t, err)
	require.Equal(t, 1, stagingCount)
}

type adminExecutionAuthEncryptor struct{}

func (adminExecutionAuthEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (adminExecutionAuthEncryptor) Decrypt(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return "", errors.New("execution auth ciphertext is not encrypted")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func invokeTotalAccountsJSON(t *testing.T, method, path string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	ginCtx.Request = httptest.NewRequest(method, path, reader)
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	if id := totalAccountsPathID(path); id != "" {
		ginCtx.Params = gin.Params{{Key: "id", Value: id}}
	}
	fn(ginCtx)
	return rec
}

func totalAccountsPathID(path string) string {
	const prefix = "/api/v1/admin/total-accounts/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return ""
	}
	id, _, _ := strings.Cut(rest, "/")
	return id
}

func decodeBatchSummary(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Equal(t, 0, resp.Code)
	return resp.Data
}

func decodePaginatedItems(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Equal(t, 0, resp.Code)
	return resp.Data.Items
}

func requireSinglePaginatedAdminID(t *testing.T, raw []byte, want int64) {
	t.Helper()
	items := decodePaginatedItems(t, raw)
	require.Len(t, items, 1)
	require.Equal(t, float64(want), items[0]["id"])
}

func requireStructuredTotalAccountInputError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "SOCIAL_ACCOUNT_INPUT_REQUIRED")
	require.Contains(t, body, "social account input is required")
	require.NotContains(t, body, "unexpected EOF")
	require.NotContains(t, body, "invalid character")
	require.NotContains(t, body, "cannot unmarshal")
}

func requireTotalAccountServiceUnavailableError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE")
	require.Contains(t, body, "social account service is unavailable")
}

func batchItemReasons(t *testing.T, body map[string]any) []string {
	t.Helper()
	rawItems, ok := body["items"].([]any)
	require.True(t, ok)
	reasons := make([]string, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		require.True(t, ok)
		if reason, ok := item["reason"].(string); ok && reason != "" {
			reasons = append(reasons, reason)
		}
	}
	return reasons
}

func batchErrors(t *testing.T, body map[string]any) []string {
	t.Helper()
	rawErrors, ok := body["errors"].([]any)
	require.True(t, ok)
	errors := make([]string, 0, len(rawErrors))
	for _, rawError := range rawErrors {
		err, ok := rawError.(string)
		require.True(t, ok)
		errors = append(errors, err)
	}
	return errors
}

func csvHeaderIndex(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, value := range header {
		if value == name {
			return i
		}
	}
	require.Failf(t, "missing csv header", "%s", name)
	return -1
}

func formatAdminID(id int64) string {
	return strconv.FormatInt(id, 10)
}
