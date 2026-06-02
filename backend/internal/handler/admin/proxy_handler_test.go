//go:build unit

package admin

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestProxyAdminListCanFilterUserOwnedExecutionProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	owner := createProxyAdminUser(t, ctx, client, "proxy-owner@example.com")
	other := createProxyAdminUser(t, ctx, client, "proxy-other@example.com")
	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("owner execution proxy").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("other execution proxy").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies?user_id="+formatProxyAdminID(owner.ID), nil)

	handler.List(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "owner execution proxy")
	require.NotContains(t, rec.Body.String(), "other execution proxy")
}

func TestProxyAdminConnectivityTestIsFreeAndDoesNotCreateTaskLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	owner := createProxyAdminUser(t, ctx, client, "proxy-test@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("free connectivity check").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyAdminID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/"+formatProxyAdminID(ip.ID)+"/test", nil)

	handler.Test(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"unknown"`)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestProxyAdminDeleteClearsDefaultProxySnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyAdminTestClient(t)
	owner := createProxyAdminUser(t, ctx, client, "proxy-delete-owner@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("delete me").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusOnline).
		SaveX(ctx)
	ipSvc := service.NewSocialIPService(client)
	proxy, err := ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := service.SocialIPTaskSnapshot(proxy)
	account := client.SocialAccount.Create().
		SetName("delete_proxy_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("delete_proxy_account").
		SetAssignedUserID(owner.ID).
		SetBoundIP(snapshot).
		SaveX(ctx)

	handler := NewProxyHandler(ipSvc, service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyAdminID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/proxies/"+formatProxyAdminID(ip.ID), nil)

	handler.Delete(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.BoundIP)
}

func newProxyAdminTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func createProxyAdminUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetRole("user").
		SetStatus("active").
		SaveX(ctx)
}

func formatProxyAdminID(id int64) string {
	return strconv.FormatInt(id, 10)
}
