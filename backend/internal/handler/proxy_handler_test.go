//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestProxyHandlerListOnlyCurrentUserAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-owner@example.com")
	other := createProxyHandlerUser(t, ctx, client, "proxy-other@example.com")
	endpoint := "http://proxy.example.test:8080"
	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("qa owner proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetEndpoint(endpoint).
		SetStatus(service.SocialIPStatusOnline).
		SetRemark("visible").
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("qa other proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusOnline).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/proxies?search=owner&status=online&ip_type=residential", nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.List(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "qa owner proxy")
	require.Contains(t, rec.Body.String(), endpoint)
	require.NotContains(t, rec.Body.String(), "qa other proxy")
}

func TestProxyHandlerCreateUsesJWTSubjectAndRejectsUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-create-owner@example.com")
	other := createProxyHandlerUser(t, ctx, client, "proxy-create-other@example.com")
	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies", strings.NewReader(`{
		"user_id": `+formatProxyHandlerID(other.ID)+`,
		"name": "bad owner injection"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Create(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_USER_ID_NOT_ACCEPTED")
	require.Contains(t, rec.Body.String(), "user_id is not accepted")

	rec = httptest.NewRecorder()
	ginCtx, _ = gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies", strings.NewReader(`{
		"user_id": null,
		"name": "null owner injection",
		"ip_type": "residential"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Create(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_USER_ID_NOT_ACCEPTED")
	require.Contains(t, rec.Body.String(), "user_id is not accepted")

	rec = httptest.NewRecorder()
	ginCtx, _ = gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies", strings.NewReader(`{
		"name": "jwt owned proxy",
		"ip_type": "residential"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Create(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data struct {
			UserID int64  `json:"user_id"`
			Name   string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, owner.ID, resp.Data.UserID)
	require.Equal(t, "jwt owned proxy", resp.Data.Name)
}

func TestProxyHandlerCrossUserOperationsAreNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-scope-owner@example.com")
	other := createProxyHandlerUser(t, ctx, client, "proxy-scope-other@example.com")
	ip := client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("other proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID)+"/test", nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Test(ginCtx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_NOT_FOUND")

	rec = httptest.NewRecorder()
	ginCtx, _ = gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Delete(ginCtx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotNil(t, client.SocialIP.GetX(ctx, ip.ID))
}

func TestProxyHandlerDeleteClearsCurrentUserDefaultProxySnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-delete-owner@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("delete me").
		SetIPType(service.SocialIPTypeResidential).
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
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)

	handler := NewProxyHandler(ipSvc, service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Delete(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.DefaultProxySnapshot)
}

func newProxyHandlerTestClient(t *testing.T) *dbent.Client {
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

func createProxyHandlerUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetRole("user").
		SetStatus("active").
		SaveX(ctx)
}

func formatProxyHandlerID(id int64) string {
	return strconv.FormatInt(id, 10)
}
