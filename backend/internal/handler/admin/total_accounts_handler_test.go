//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
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
	handler := NewTotalAccountsHandler(service.NewSocialAccountService(client))

	assignRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-assign", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedB.ID)+`,`+formatAdminID(assigned.ID)+`],"user_id":`+formatAdminID(user.ID)+`}`), handler.BatchAssign)
	require.Equal(t, http.StatusOK, assignRec.Code)
	assignBody := decodeBatchSummary(t, assignRec.Body.Bytes())
	require.Equal(t, float64(3), assignBody["total"])
	require.Equal(t, float64(2), assignBody["succeeded"])
	require.Equal(t, float64(1), assignBody["skipped"])
	require.Equal(t, float64(0), assignBody["failed"])
	require.NotEmpty(t, assignBody["items"])

	reclaimRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-reclaim", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(assigned.ID)+`,0]}`), handler.BatchReclaim)
	require.Equal(t, http.StatusOK, reclaimRec.Code)
	reclaimBody := decodeBatchSummary(t, reclaimRec.Body.Bytes())
	require.Equal(t, float64(3), reclaimBody["total"])
	require.Equal(t, float64(2), reclaimBody["succeeded"])
	require.Equal(t, float64(1), reclaimBody["skipped"])
	require.Equal(t, float64(0), reclaimBody["failed"])
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).AssignedUserID)
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).DefaultProxySnapshot)

	deleteRec := invokeTotalAccountsJSON(t, http.MethodPost, "/api/v1/admin/total-accounts/batch-delete", []byte(`{"ids":[`+formatAdminID(unassignedA.ID)+`,`+formatAdminID(unassignedB.ID)+`,0]}`), handler.BatchDelete)
	require.Equal(t, http.StatusOK, deleteRec.Code)
	deleteBody := decodeBatchSummary(t, deleteRec.Body.Bytes())
	require.Equal(t, float64(3), deleteBody["total"])
	require.Equal(t, float64(2), deleteBody["succeeded"])
	require.Equal(t, float64(1), deleteBody["skipped"])
	require.Equal(t, float64(0), deleteBody["failed"])
	require.NotNil(t, client.SocialAccount.GetX(mixins.SkipSoftDelete(ctx), unassignedA.ID).DeletedAt)
	require.NotNil(t, client.SocialAccount.GetX(mixins.SkipSoftDelete(ctx), unassignedB.ID).DeletedAt)
}

func invokeTotalAccountsJSON(t *testing.T, method, path string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	fn(ginCtx)
	return rec
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

func formatAdminID(id int64) string {
	return strconv.FormatInt(id, 10)
}
