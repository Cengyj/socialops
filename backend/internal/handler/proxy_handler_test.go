//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/proxies?search=%20owner%20&status=%20online%20&ip_type=%20residential%20", nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.List(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "qa owner proxy")
	require.Contains(t, rec.Body.String(), endpoint)
	require.NotContains(t, rec.Body.String(), "qa other proxy")
}

func TestProxyFiltersFromQueryKeepsListFilterFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/proxies?search=tokyo&status=online&ip_type=residential", nil)

	filters := proxyFiltersFromQuery(ginCtx)

	require.Equal(t, service.SocialIPListFilters{
		Search: "tokyo",
		Status: "online",
		IPType: "residential",
	}, filters)
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

func TestProxyHandlerCreateReturnsStructuredNameRequiredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-create-name-owner@example.com")
	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))

	for _, body := range []string{
		`{"ip_type":"residential"}`,
		`{"name":"   ","ip_type":"residential"}`,
	} {
		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies", strings.NewReader(body))
		ginCtx.Request.Header.Set("Content-Type", "application/json")
		ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

		handler.Create(ginCtx)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "SOCIAL_IP_NAME_REQUIRED")
		require.Contains(t, rec.Body.String(), "social IP name is required")
	}
	require.Equal(t, 0, client.SocialIP.Query().CountX(ctx))
}

func TestProxyHandlerCreateUpdateReturnStructuredInputErrorForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-invalid-json-owner@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("original proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)
	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	initialCount := client.SocialIP.Query().CountX(ctx)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		fn     gin.HandlerFunc
		params gin.Params
	}{
		{
			name:   "create malformed json",
			method: http.MethodPost,
			path:   "/api/v1/proxies",
			body:   `{"name":`,
			fn:     handler.Create,
		},
		{
			name:   "create wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/proxies",
			body:   `{"name":123,"ip_type":"residential"}`,
			fn:     handler.Create,
		},
		{
			name:   "update malformed json",
			method: http.MethodPut,
			path:   "/api/v1/proxies/" + formatProxyHandlerID(ip.ID),
			body:   `{"name":`,
			fn:     handler.Update,
			params: gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}},
		},
		{
			name:   "update wrong field type",
			method: http.MethodPut,
			path:   "/api/v1/proxies/" + formatProxyHandlerID(ip.ID),
			body:   `{"name":123}`,
			fn:     handler.Update,
			params: gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Params = tt.params
			ginCtx.Request = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

			tt.fn(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "SOCIAL_IP_INPUT_REQUIRED")
			require.Contains(t, rec.Body.String(), "social IP input is required")
			require.NotContains(t, rec.Body.String(), "unexpected EOF")
			require.NotContains(t, rec.Body.String(), "invalid character")
			require.NotContains(t, rec.Body.String(), "cannot unmarshal")
			require.Equal(t, initialCount, client.SocialIP.Query().CountX(ctx))
			stored := client.SocialIP.GetX(ctx, ip.ID)
			require.Equal(t, "original proxy", stored.Name)
			require.Equal(t, owner.ID, stored.UserID)
		})
	}
}

func TestProxyHandlerUpdateRejectsUserIDAndKeepsOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-update-owner@example.com")
	other := createProxyHandlerUser(t, ctx, client, "proxy-update-other@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("owned proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusUnknown).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), strings.NewReader(`{
		"user_id": `+formatProxyHandlerID(other.ID)+`,
		"name": "owner injection"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Update(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_USER_ID_NOT_ACCEPTED")
	require.Contains(t, rec.Body.String(), "user_id is not accepted")
	stored := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, owner.ID, stored.UserID)
	require.Equal(t, "owned proxy", stored.Name)

	rec = httptest.NewRecorder()
	ginCtx, _ = gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), strings.NewReader(`{
		"user_id": null,
		"name": "null owner injection"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Update(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_USER_ID_NOT_ACCEPTED")
	require.Contains(t, rec.Body.String(), "user_id is not accepted")
	stored = client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, owner.ID, stored.UserID)
	require.Equal(t, "owned proxy", stored.Name)
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
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), strings.NewReader(`{"name":"cross-user update"}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Update(ginCtx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_NOT_FOUND")
	stored := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, other.ID, stored.UserID)
	require.Equal(t, "other proxy", stored.Name)

	rec = httptest.NewRecorder()
	ginCtx, _ = gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID), nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Delete(ginCtx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotNil(t, client.SocialIP.GetX(ctx, ip.ID))
}

func TestProxyHandlerRejectsInvalidPathIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-invalid-id-owner@example.com")
	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		fn     gin.HandlerFunc
	}{
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/api/v1/proxies/not-a-number",
			body:   `{"name":"must-not-parse"}`,
			fn:     handler.Update,
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/api/v1/proxies/not-a-number",
			fn:     handler.Delete,
		},
		{
			name:   "test",
			method: http.MethodPost,
			path:   "/api/v1/proxies/not-a-number/test",
			fn:     handler.Test,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Params = gin.Params{{Key: "id", Value: "not-a-number"}}
			ginCtx.Request = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

			tt.fn(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "invalid id")
		})
	}
}

func TestProxyHandlerReportsServiceUnavailableWhenDependenciesAreMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProxyHandler(nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		params gin.Params
		fn     gin.HandlerFunc
	}{
		{
			name:   "list",
			method: http.MethodGet,
			path:   "/api/v1/proxies",
			fn:     handler.List,
		},
		{
			name:   "list usable",
			method: http.MethodGet,
			path:   "/api/v1/proxies/usable",
			fn:     handler.ListUsable,
		},
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/api/v1/proxies",
			body:   `{"name":"proxy","ip_type":"residential"}`,
			fn:     handler.Create,
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/api/v1/proxies/1",
			body:   `{"name":"proxy"}`,
			params: gin.Params{{Key: "id", Value: "1"}},
			fn:     handler.Update,
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/api/v1/proxies/1",
			params: gin.Params{{Key: "id", Value: "1"}},
			fn:     handler.Delete,
		},
		{
			name:   "test",
			method: http.MethodPost,
			path:   "/api/v1/proxies/1/test",
			params: gin.Params{{Key: "id", Value: "1"}},
			fn:     handler.Test,
		},
		{
			name:   "test all",
			method: http.MethodPost,
			path:   "/api/v1/proxies/test",
			fn:     handler.TestAll,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Params = tt.params
			ginCtx.Request = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				ginCtx.Request.Header.Set("Content-Type", "application/json")
			}
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

			tt.fn(ginCtx)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			require.Contains(t, rec.Body.String(), "SOCIAL_IP_SERVICE_UNAVAILABLE")
			require.Contains(t, rec.Body.String(), "social IP service is unavailable")
		})
	}
}

func TestProxyHandlerDeleteClearsCurrentUserDefaultProxySnapshotsAndTaskLogReferences(t *testing.T) {
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
	taskLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(owner.ID).
		SetAction("follow").
		SetStatus(service.SocialTaskLogStatusPending).
		SetProxyID(ip.ID).
		SetProxySnapshot(snapshot).
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
	storedTaskLog := client.SocialTaskLog.GetX(ctx, taskLog.ID)
	require.Nil(t, storedTaskLog.ProxyID)
	require.Equal(t, snapshot, *storedTaskLog.ProxySnapshot)
	_, err = client.SocialIP.Get(ctx, ip.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestProxyHandlerTestCurrentUserProxyUpdatesStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-test-owner@example.com")
	ip := client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("test me").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatProxyHandlerID(ip.ID)}}
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies/"+formatProxyHandlerID(ip.ID)+"/test", nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.Test(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"unknown"`)
	stored := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, owner.ID, stored.UserID)
	require.Equal(t, service.SocialIPStatusUnknown, stored.Status)
	require.Nil(t, stored.LatencyMs)
	require.NotNil(t, stored.LastCheckAt)
}

func TestProxyHandlerTestAllFailsWhenStatusPersistenceFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newProxyHandlerTestClient(t)
	owner := createProxyHandlerUser(t, ctx, client, "proxy-test-all-persist-owner@example.com")
	client.SocialIP.Create().
		SetUserID(owner.ID).
		SetName("persist failure proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)
	persistErr := errors.New("proxy status persistence failed")
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) {
				return nil, persistErr
			}
			return next.Mutate(ctx, m)
		})
	})

	handler := NewProxyHandler(service.NewSocialIPService(client), service.NewSocialIPChecker(client))
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", nil)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: owner.ID})

	handler.TestAll(ginCtx)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "internal error")
	require.NotContains(t, rec.Body.String(), "proxy status persistence failed")
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
