//go:build unit

package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestAccountWorkbenchHandlerSubmitTaskFailsClosedWithoutCharging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-task@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	proxyEndpoint := "http://8.8.8.8:8080"
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("submit proxy").
		SetIPType("residential").
		SetEndpoint(proxyEndpoint).
		SetStatus(service.SocialIPStatusOnline).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	proxySvc := service.NewSocialIPService(client)
	defaultProxy, err := proxySvc.GetByID(ctx, proxy.ID)
	require.NoError(t, err)
	_, err = service.NewSocialAccountService(client).SetDefaultProxyForUser(ctx, account.ID, user.ID, defaultProxy)
	require.NoError(t, err)
	templateSvc := service.NewTaskSettingsService(client)
	tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "follow target",
		Type: service.SocialTaskActionFollow,
		Params: service.TaskTemplateParams{
			Targets: []string{"@target"},
		},
		IsDefault: true,
	})
	require.NoError(t, err)
	require.Equal(t, service.SocialTaskActionFollow, tmpl.Type)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"client_request_id": "g008-submit-1"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	require.Contains(t, rec.Body.String(), `"platform":"x_twitter"`)
	require.Contains(t, rec.Body.String(), `"account_name":"@submit_task"`)
	require.Contains(t, rec.Body.String(), `"charged":false`)
	require.Contains(t, rec.Body.String(), "social platform executor queue is not configured; task was not charged")
	require.NotContains(t, rec.Body.String(), "proxy_snapshot")
	require.NotContains(t, rec.Body.String(), "proxy_id")
	require.NotContains(t, rec.Body.String(), "billing_request_id")
	require.NotContains(t, rec.Body.String(), "idempotency_key")
	require.NotContains(t, rec.Body.String(), proxyEndpoint)
	require.Zero(t, userRepo.deductCalls)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)

	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, service.SocialTaskLogStatusFailed, logs[0].Status)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, logs[0].ChargeStatus)
	require.Zero(t, logs[0].ChargedAmount)
	require.Nil(t, logs[0].ChargeSource)
	require.NotNil(t, logs[0].ResultMessage)
	require.Equal(t, "social platform executor queue is not configured; task was not charged", *logs[0].ResultMessage)
	require.NotNil(t, logs[0].IdempotencyKey)
	require.Equal(t, "g008-submit-1", *logs[0].IdempotencyKey)
	require.Nil(t, logs[0].BillingRequestID)
	require.NotNil(t, logs[0].ProxySnapshot)
	require.Contains(t, *logs[0].ProxySnapshot, proxyEndpoint)
}

func TestAccountWorkbenchHandlerRoutesRequireAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchHandler(nil, nil, nil, nil, nil)

	tests := []struct {
		name   string
		method string
		target string
		call   func(*gin.Context)
	}{
		{
			name:   "list accounts",
			method: http.MethodGet,
			target: "/api/v1/accounts",
			call:   handler.ListMyAccounts,
		},
		{
			name:   "list task logs",
			method: http.MethodGet,
			target: "/api/v1/accounts/tasks",
			call:   handler.ListTaskLogs,
		},
		{
			name:   "batch import",
			method: http.MethodPost,
			target: "/api/v1/accounts/batch-import",
			call:   handler.BatchImportMyAccounts,
		},
		{
			name:   "update account",
			method: http.MethodPut,
			target: "/api/v1/accounts/1",
			call:   handler.UpdateMyAccount,
		},
		{
			name:   "delete account",
			method: http.MethodDelete,
			target: "/api/v1/accounts/1",
			call:   handler.DeleteMyAccount,
		},
		{
			name:   "batch delete",
			method: http.MethodPost,
			target: "/api/v1/accounts/batch-delete",
			call:   handler.BatchDeleteMyAccounts,
		},
		{
			name:   "export accounts",
			method: http.MethodGet,
			target: "/api/v1/accounts/export",
			call:   handler.ExportMyAccounts,
		},
		{
			name:   "submit task",
			method: http.MethodPost,
			target: "/api/v1/accounts/tasks",
			call:   handler.SubmitTask,
		},
		{
			name:   "set default proxy",
			method: http.MethodPut,
			target: "/api/v1/accounts/1/default-proxy",
			call:   handler.SetDefaultProxy,
		},
		{
			name:   "batch set default proxy",
			method: http.MethodPost,
			target: "/api/v1/accounts/default-proxy",
			call:   handler.BatchSetDefaultProxy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(tt.method, tt.target, nil)

			tt.call(ginCtx)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.Contains(t, rec.Body.String(), "unauthorized")
		})
	}
}

func TestAccountWorkbenchHandlerRejectsInvalidPathIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchHandler(nil, nil, nil, nil, nil)

	tests := []struct {
		name   string
		method string
		target string
		call   func(*gin.Context)
	}{
		{
			name:   "update account",
			method: http.MethodPut,
			target: "/api/v1/accounts/not-a-number",
			call:   handler.UpdateMyAccount,
		},
		{
			name:   "delete account",
			method: http.MethodDelete,
			target: "/api/v1/accounts/not-a-number",
			call:   handler.DeleteMyAccount,
		},
		{
			name:   "set default proxy",
			method: http.MethodPut,
			target: "/api/v1/accounts/not-a-number/default-proxy",
			call:   handler.SetDefaultProxy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(tt.method, tt.target, nil)
			ginCtx.Params = gin.Params{{Key: "id", Value: "not-a-number"}}
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

			tt.call(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "invalid id")
		})
	}
}

func TestAccountWorkbenchHandlerReportsAccountServiceUnavailableWhenDependencyIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchHandler(nil, nil, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		pathID string
		body   []byte
		call   func(*gin.Context)
	}{
		{name: "list accounts", method: http.MethodGet, path: "/api/v1/accounts", call: handler.ListMyAccounts},
		{name: "list task logs", method: http.MethodGet, path: "/api/v1/accounts/tasks", call: handler.ListTaskLogs},
		{name: "batch import", method: http.MethodPost, path: "/api/v1/accounts/batch-import", body: []byte(`{"accounts":[{"name":"@service_unavailable","platform":"x_twitter"}]}`), call: handler.BatchImportMyAccounts},
		{name: "update account", method: http.MethodPut, path: "/api/v1/accounts/1", pathID: "1", body: []byte(`{"password":"updated-secret"}`), call: handler.UpdateMyAccount},
		{name: "delete account", method: http.MethodDelete, path: "/api/v1/accounts/1", pathID: "1", call: handler.DeleteMyAccount},
		{name: "batch delete", method: http.MethodPost, path: "/api/v1/accounts/batch-delete", body: []byte(`{"ids":[1]}`), call: handler.BatchDeleteMyAccounts},
		{name: "export accounts", method: http.MethodGet, path: "/api/v1/accounts/export", call: handler.ExportMyAccounts},
		{name: "clear default proxy", method: http.MethodPut, path: "/api/v1/accounts/1/default-proxy", pathID: "1", body: []byte(`{"proxy_id":null}`), call: handler.SetDefaultProxy},
		{name: "batch clear default proxy", method: http.MethodPost, path: "/api/v1/accounts/default-proxy", body: []byte(`{"account_ids":[1],"mode":"clear"}`), call: handler.BatchSetDefaultProxy},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeJSONSocialHandlerAsUserWithPathID(t, 7, tt.method, tt.path, tt.pathID, tt.body, tt.call)

			requireAccountWorkbenchServiceUnavailableError(t, rec, "SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE")
		})
	}
}

func TestAccountWorkbenchHandlerReportsProxyServiceUnavailableForDefaultProxyDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newAccountWorkbenchHandlerTestClient(t)
	handler := NewAccountWorkbenchHandler(service.NewSocialAccountService(client), nil, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		pathID string
		body   []byte
		call   func(*gin.Context)
	}{
		{name: "single specific proxy", method: http.MethodPut, path: "/api/v1/accounts/1/default-proxy", pathID: "1", body: []byte(`{"proxy_id":1}`), call: handler.SetDefaultProxy},
		{name: "batch specific proxy", method: http.MethodPost, path: "/api/v1/accounts/default-proxy", body: []byte(`{"account_ids":[1],"mode":"specific","proxy_id":1}`), call: handler.BatchSetDefaultProxy},
		{name: "batch random proxy", method: http.MethodPost, path: "/api/v1/accounts/default-proxy", body: []byte(`{"account_ids":[1],"mode":"random"}`), call: handler.BatchSetDefaultProxy},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeJSONSocialHandlerAsUserWithPathID(t, 7, tt.method, tt.path, tt.pathID, tt.body, tt.call)

			requireAccountWorkbenchServiceUnavailableError(t, rec, "SOCIAL_IP_SERVICE_UNAVAILABLE")
		})
	}
}

func TestAccountWorkbenchHandlerReportsTaskDependenciesUnavailableWithoutChangingLoginTemplateSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchHandler(nil, nil, nil, nil, nil)

	followRec := invokeJSONSocialHandlerAsUser(t, 7, http.MethodPost, "/api/v1/accounts/tasks", []byte(`{"account_ids":[1],"action":"follow"}`), handler.SubmitTask)
	requireAccountWorkbenchServiceUnavailableError(t, followRec, "TASK_TEMPLATE_SERVICE_UNAVAILABLE")

	loginRec := invokeJSONSocialHandlerAsUser(t, 7, http.MethodPost, "/api/v1/accounts/tasks", []byte(`{"account_ids":[1],"action":"login"}`), handler.SubmitTask)
	requireAccountWorkbenchServiceUnavailableError(t, loginRec, "SOCIAL_TASK_SERVICE_UNAVAILABLE")
	require.NotContains(t, loginRec.Body.String(), "TASK_TEMPLATE_SERVICE_UNAVAILABLE")
}

func TestAccountWorkbenchHandlerInputBindingErrorsAreStructured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountWorkbenchHandler(nil, nil, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		pathID string
		body   []byte
		call   func(*gin.Context)
	}{
		{
			name:   "batch import malformed json",
			method: http.MethodPost,
			path:   "/api/v1/accounts/batch-import",
			body:   []byte(`{"accounts":`),
			call:   handler.BatchImportMyAccounts,
		},
		{
			name:   "batch import wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/accounts/batch-import",
			body:   []byte(`{"accounts":"bad"}`),
			call:   handler.BatchImportMyAccounts,
		},
		{
			name:   "update account malformed json",
			method: http.MethodPut,
			path:   "/api/v1/accounts/1",
			pathID: "1",
			body:   []byte(`{"password":`),
			call:   handler.UpdateMyAccount,
		},
		{
			name:   "update account wrong field type",
			method: http.MethodPut,
			path:   "/api/v1/accounts/1",
			pathID: "1",
			body:   []byte(`{"password":123}`),
			call:   handler.UpdateMyAccount,
		},
		{
			name:   "batch delete malformed json",
			method: http.MethodPost,
			path:   "/api/v1/accounts/batch-delete",
			body:   []byte(`{"ids":`),
			call:   handler.BatchDeleteMyAccounts,
		},
		{
			name:   "batch delete wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/accounts/batch-delete",
			body:   []byte(`{"ids":"bad"}`),
			call:   handler.BatchDeleteMyAccounts,
		},
		{
			name:   "submit task malformed json",
			method: http.MethodPost,
			path:   "/api/v1/accounts/tasks",
			body:   []byte(`{"account_ids":`),
			call:   handler.SubmitTask,
		},
		{
			name:   "submit task wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/accounts/tasks",
			body:   []byte(`{"account_ids":"bad","action":"follow"}`),
			call:   handler.SubmitTask,
		},
		{
			name:   "set default proxy malformed json",
			method: http.MethodPut,
			path:   "/api/v1/accounts/1/default-proxy",
			pathID: "1",
			body:   []byte(`{"proxy_id":`),
			call:   handler.SetDefaultProxy,
		},
		{
			name:   "set default proxy wrong field type",
			method: http.MethodPut,
			path:   "/api/v1/accounts/1/default-proxy",
			pathID: "1",
			body:   []byte(`{"proxy_id":"bad"}`),
			call:   handler.SetDefaultProxy,
		},
		{
			name:   "batch default proxy malformed json",
			method: http.MethodPost,
			path:   "/api/v1/accounts/default-proxy",
			body:   []byte(`{"account_ids":[1],"mode":`),
			call:   handler.BatchSetDefaultProxy,
		},
		{
			name:   "batch default proxy wrong field type",
			method: http.MethodPost,
			path:   "/api/v1/accounts/default-proxy",
			body:   []byte(`{"account_ids":"bad","mode":"specific"}`),
			call:   handler.BatchSetDefaultProxy,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := invokeJSONSocialHandlerAsUserWithPathID(t, 7, tt.method, tt.path, tt.pathID, tt.body, tt.call)

			requireStructuredAccountWorkbenchInputError(t, rec)
		})
	}
}

func TestAccountWorkbenchFiltersFromQueryKeepsListAndExportFilterFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/accounts?search=northwind&platform=x_twitter&account_status=available&task_status=stored&account_ids=101,%20102,bad", nil)

	filters := accountWorkbenchFiltersFromQuery(ginCtx)

	require.Equal(t, service.SocialAccountListFilters{
		Search:        "northwind",
		Platform:      "x_twitter",
		AccountStatus: "available",
		TaskStatus:    "stored",
		AccountIDs:    []int64{101, 102},
	}, filters)
}

func TestAccountWorkbenchTaskLogFiltersFromQueryParsesPollingFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/accounts/tasks?log_ids=1,%202,bad,3&account_ids=7,0,-1,9&statuses=pending,running&statuses=success&limit=25",
		nil,
	)

	filters, err := accountWorkbenchTaskLogFiltersFromQuery(ginCtx, 42)

	require.NoError(t, err)
	require.Equal(t, service.SocialTaskLogListFilters{
		UserID:     42,
		LogIDs:     []int64{1, 2, 3},
		AccountIDs: []int64{7, 9},
		Statuses:   []string{"pending", "running", "success"},
		Limit:      25,
	}, filters)
}

func TestAccountWorkbenchTaskLogFiltersFromQueryDefaultsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/accounts/tasks", nil)

	filters, err := accountWorkbenchTaskLogFiltersFromQuery(ginCtx, 42)

	require.NoError(t, err)
	require.Equal(t, 50, filters.Limit)
}

func TestAccountWorkbenchTaskLogFiltersFromQueryRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, target := range []string{
		"/api/v1/accounts/tasks?limit=0",
		"/api/v1/accounts/tasks?limit=not-a-number",
	} {
		t.Run(target, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodGet, target, nil)

			_, err := accountWorkbenchTaskLogFiltersFromQuery(ginCtx, 42)

			require.Error(t, err)
			require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
			require.Equal(t, "SOCIAL_TASK_LOG_LIMIT_INVALID", infraerrors.Reason(err))
		})
	}
}

func TestShortUserTaskResultSanitizesExecutionDetails(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		want      string
		forbidden []string
	}{
		{
			name:    "executor unavailable",
			message: "social platform executor queue is not configured; task was not charged",
			want:    "social platform executor queue is not configured; task was not charged",
		},
		{
			name:    "login dependency not configured",
			message: "twitter device fingerprint provider is not configured",
			want:    "登录依赖服务未配置，本次未扣费",
			forbidden: []string{
				"device fingerprint",
				"not configured",
			},
		},
		{
			name:    "password invalid",
			message: "twitter password is incorrect",
			want:    "密码错误，本次未扣费",
			forbidden: []string{
				"password",
			},
		},
		{
			name:    "auth material",
			message: "auth cookie missing for account; token=secret-token-value",
			want:    "账号认证信息不可用，本次未扣费",
			forbidden: []string{
				"auth cookie",
				"secret-token-value",
			},
		},
		{
			name:    "proxy detail",
			message: "proxy http://user:password@127.0.0.1:8080 is offline",
			want:    "执行代理不可用，本次未扣费",
			forbidden: []string{
				"user:password",
				"127.0.0.1:8080",
			},
		},
		{
			name:    "unsupported action",
			message: "unsupported action unsupported_action on x_twitter",
			want:    "该动作暂不支持，本次未扣费",
			forbidden: []string{
				"unsupported action",
				"x_twitter",
			},
		},
		{
			name:    "avatar dimensions",
			message: "avatar image must be 400x400 pixels",
			want:    "头像图片尺寸必须为 400x400，本次未扣费",
		},
		{
			name:    "banner dimensions",
			message: "banner image must be 1500x500 pixels",
			want:    "背景图图片尺寸必须为 1500x500，本次未扣费",
		},
		{
			name:    "media asset unavailable",
			message: "post media #1 media asset is unavailable",
			want:    "任务媒体资源不可用，本次未扣费",
		},
		{
			name:    "media upload failure",
			message: "twitter media upload returned no media id",
			want:    "平台媒体上传失败，本次未扣费",
		},
		{
			name:    "media processing failure",
			message: "twitter media upload returned processing failed",
			want:    "平台媒体上传失败，本次未扣费",
		},
		{
			name:    "media processing timeout",
			message: "twitter media upload returned processing timeout",
			want:    "平台媒体上传失败，本次未扣费",
		},
		{
			name:    "post video unavailable",
			message: "video media is not supported for SocialOps execution",
			want:    "视频发帖媒体暂未开放，本次未扣费",
		},
		{
			name:    "post media type unsupported",
			message: "post media content type is not supported",
			want:    "发帖媒体类型暂不支持，本次未扣费",
		},
		{
			name:    "challenge required",
			message: "additional verification required",
			want:    "账号需要额外验证，本次未扣费",
		},
		{
			name:    "safe avatar dimensions are preserved",
			message: "头像图片尺寸必须为 400x400，本次未扣费",
			want:    "头像图片尺寸必须为 400x400，本次未扣费",
		},
		{
			name:    "safe banner dimensions are preserved",
			message: "背景图图片尺寸必须为 1500x500，本次未扣费",
			want:    "背景图图片尺寸必须为 1500x500，本次未扣费",
		},
		{
			name:    "safe media asset unavailable message is preserved",
			message: "任务媒体资源不可用，本次未扣费",
			want:    "任务媒体资源不可用，本次未扣费",
		},
		{
			name:    "safe media upload failure message is preserved",
			message: "平台媒体上传失败，本次未扣费",
			want:    "平台媒体上传失败，本次未扣费",
		},
		{
			name:    "safe challenge required message is preserved",
			message: "账号需要额外验证，本次未扣费",
			want:    "账号需要额外验证，本次未扣费",
		},
		{
			name:    "safe post video message is preserved",
			message: "视频发帖媒体暂未开放，本次未扣费",
			want:    "视频发帖媒体暂未开放，本次未扣费",
		},
		{
			name:    "safe unsupported post media type message is preserved",
			message: "发帖媒体类型暂不支持，本次未扣费",
			want:    "发帖媒体类型暂不支持，本次未扣费",
		},
		{
			name:    "safe stale task timeout message is preserved",
			message: "任务执行超时，本次未扣费",
			want:    "任务执行超时，本次未扣费",
		},
		{
			name:    "raw platform business error is preserved",
			message: "twitter error 399: Sorry, we could not find your account.",
			want:    "账号不存在，本次未扣费",
		},
		{
			name:    "twitter 399 wrong password prefers password classification",
			message: "twitter error 399: Wrong password!",
			want:    "密码错误，本次未扣费",
		},
		{
			name:    "twitter 399 without exact business message remains raw",
			message: "twitter error 399",
			want:    "twitter error 399",
		},
		{
			name:    "twitter 399 with unknown business message remains raw",
			message: "twitter error 399: Password checkpoint required.",
			want:    "twitter error 399: Password checkpoint required.",
		},
		{
			name:    "raw platform password business error is preserved",
			message: "twitter login error: The password you entered is incorrect.",
			want:    "密码错误，本次未扣费",
		},
		{
			name:    "twitter login error with unknown business message remains raw",
			message: "twitter login error: Password checkpoint required.",
			want:    "twitter login error: Password checkpoint required.",
		},
		{
			name:    "raw platform error with sensitive value is hidden",
			message: "twitter error 89: token=secret-token-value",
			want:    "twitter error 89: token=secret-token-value",
		},
		{
			name:    "unknown internal detail",
			message: `upstream response body {"error":"secret","headers":{"authorization":"Bearer abc"}} trace_id=trace-123 request_id=req-456`,
			want:    `upstream response body {"error":"secret","headers":{"authorization":"Bearer abc"}} trace_id=trace-123 request_id=req-456`,
		},
		{
			name:    "success-looking sensitive upstream detail",
			message: `follow succeeded https://upstream.example/callback?trace_id=trace-123 authorization=Bearer abc`,
			want:    `follow succeeded https://upstream.example/callback?trace_id=trace-123 authorization=Bearer abc`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortUserTaskResult(&tt.message)
			require.NotNil(t, got)
			require.Equal(t, tt.want, *got)
			for _, forbidden := range tt.forbidden {
				require.NotContains(t, *got, forbidden)
			}
		})
	}
}

func TestShortUserTaskResultKeepsSafeSuccessSummary(t *testing.T) {
	message := "follow succeeded"

	got := shortUserTaskResult(&message)

	require.NotNil(t, got)
	require.Equal(t, "follow succeeded", *got)
}

func TestUserTaskLogResponseFromServiceAppliesAccountAndSanitizesStructuredFields(t *testing.T) {
	message := "auth cookie missing for account; token=secret-token-value"
	target := "https://x.com/northwind"
	content := "hello from template"
	executedAt := time.Date(2026, 6, 21, 9, 30, 0, 0, time.UTC)
	createdAt := time.Date(2026, 6, 21, 9, 29, 0, 0, time.UTC)

	resp := userTaskLogResponseFromService(&service.SocialTaskLog{
		ID:              77,
		SocialAccountID: 12,
		Action:          service.SocialTaskActionPost,
		Target:          &target,
		Content:         &content,
		Payload: &service.SocialTaskPayload{
			Post: &service.SocialPostPayload{
				Text: "hello from template",
				Media: []service.SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "social-task/12/post.png",
					URL:         "https://media.example/private/post.png",
					ContentType: "image/png",
					FileName:    "post.png",
					Width:       640,
					Height:      640,
				}},
			},
		},
		TemplateSnapshot: &service.SocialTaskTemplateSnapshot{
			TemplateID:   "tmpl-post",
			TemplateName: "Post template",
			TemplateType: service.SocialTaskActionPost,
			Params: service.TaskTemplateParams{
				Media: []service.SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "social-task/12/template.png",
					URL:         "https://media.example/private/template.png",
					ContentType: "image/png",
					FileName:    "template.png",
				}},
			},
		},
		Status:        service.SocialTaskLogStatusSuccess,
		ResultMessage: &message,
		ChargedAmount: 0.2,
		ChargeStatus:  service.SocialTaskChargeStatusCharged,
		ExecutedAt:    &executedAt,
		CreatedAt:     createdAt,
	}, &service.SocialAccount{
		ID:       12,
		Name:     "@northwind",
		Platform: "x_twitter",
	})

	require.Equal(t, int64(77), resp.ID)
	require.Equal(t, int64(12), resp.SocialAccountID)
	require.Equal(t, "x_twitter", resp.Platform)
	require.Equal(t, "@northwind", resp.AccountName)
	require.True(t, resp.Charged)
	require.InEpsilon(t, 0.2, resp.ChargedAmount, 0.000001)
	require.Equal(t, service.SocialTaskChargeStatusCharged, resp.ChargeStatus)
	require.Equal(t, executedAt, *resp.ExecutedAt)
	require.Equal(t, createdAt, resp.CreatedAt)
	require.NotNil(t, resp.ResultMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *resp.ResultMessage)
	require.NotNil(t, resp.Payload)
	require.NotNil(t, resp.Payload.Post)
	require.Len(t, resp.Payload.Post.Media, 1)
	require.Equal(t, "inline", resp.Payload.Post.Media[0].Source)
	require.Equal(t, "image/png", resp.Payload.Post.Media[0].ContentType)
	require.Equal(t, "post.png", resp.Payload.Post.Media[0].FileName)
	require.Empty(t, resp.Payload.Post.Media[0].StorageKey)
	require.Empty(t, resp.Payload.Post.Media[0].URL)
	require.NotNil(t, resp.TemplateSnapshot)
	require.Len(t, resp.TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "inline", resp.TemplateSnapshot.Params.Media[0].Source)
	require.Equal(t, "template.png", resp.TemplateSnapshot.Params.Media[0].FileName)
	require.Empty(t, resp.TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, resp.TemplateSnapshot.Params.Media[0].URL)
}

func TestUserTaskLogChargedRequiresChargedStatusAndPositiveAmount(t *testing.T) {
	tests := []struct {
		name string
		log  *service.SocialTaskLog
		want bool
	}{
		{
			name: "charged with positive amount",
			log:  &service.SocialTaskLog{ChargeStatus: service.SocialTaskChargeStatusCharged, ChargedAmount: 0.1},
			want: true,
		},
		{
			name: "charged status without amount",
			log:  &service.SocialTaskLog{ChargeStatus: service.SocialTaskChargeStatusCharged, ChargedAmount: 0},
			want: false,
		},
		{
			name: "amount without charged status",
			log:  &service.SocialTaskLog{ChargeStatus: service.SocialTaskChargeStatusNotCharged, ChargedAmount: 0.1},
			want: false,
		},
		{
			name: "nil log",
			log:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, userTaskLogCharged(tt.log))
		})
	}
}

func TestUserSocialAccountResponseFromServiceKeepsDeliveryFieldsAndPublicStatus(t *testing.T) {
	platformUserID := "pool-account-id"
	password := "pool-secret"
	phone := "+15550000001"
	email := "account@example.test"
	emailPassword := "mail-secret"
	twoFactor := "totp-secret"
	backupCode := "backup-code"
	emailClientID := "mail-client"
	emailToken := "mail-token"
	registrationIP := "203.0.113.10"
	authCookie := "ct0=list; auth_token=list"
	executionAuth := "encrypted-execution-auth-ciphertext"
	taskMessage := `upstream response body {"error":"secret"} authorization=Bearer abc token=secret-token`
	proxySnapshot := `{"id":301,"name":"delivery proxy","ip_type":"residential","endpoint":"http://proxy.local:8080","status":"online"}`
	remark := "operator note"
	createdAt := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 21, 10, 30, 0, 0, time.UTC)

	resp := userSocialAccountResponseFromService(&service.SocialAccount{
		ID:                   12,
		Name:                 "@delivery",
		Platform:             "x_twitter",
		Username:             "delivery",
		PlatformUserID:       &platformUserID,
		Password:             &password,
		Phone:                &phone,
		Email:                &email,
		EmailPassword:        &emailPassword,
		TwoFactor:            &twoFactor,
		BackupCode:           &backupCode,
		EmailClientID:        &emailClientID,
		EmailToken:           &emailToken,
		RegistrationIP:       &registrationIP,
		AuthCookie:           &authCookie,
		ExecutionAuth:        &executionAuth,
		AccountStatus:        service.SocialAccountStatusAvailable,
		TaskStatus:           service.SocialTaskStatusManualReview,
		TaskMessage:          &taskMessage,
		DefaultProxySnapshot: &proxySnapshot,
		Remark:               &remark,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	})

	require.Equal(t, int64(12), resp.ID)
	require.Equal(t, "@delivery", resp.Name)
	require.Equal(t, "x_twitter", resp.Platform)
	require.Equal(t, "delivery", resp.Username)
	require.Equal(t, platformUserID, *resp.PlatformUserID)
	require.Equal(t, password, *resp.Password)
	require.Equal(t, phone, *resp.Phone)
	require.Equal(t, email, *resp.Email)
	require.Equal(t, emailPassword, *resp.EmailPassword)
	require.Equal(t, twoFactor, *resp.TwoFactor)
	require.Equal(t, backupCode, *resp.BackupCode)
	require.Equal(t, emailClientID, *resp.EmailClientID)
	require.Equal(t, emailToken, *resp.EmailToken)
	require.Equal(t, registrationIP, *resp.RegistrationIP)
	require.Equal(t, authCookie, *resp.AuthCookie)
	require.Equal(t, executionAuth, *resp.ExecutionAuth)
	require.Equal(t, service.SocialAccountStatusAvailable, resp.AccountStatus)
	require.Equal(t, service.SocialTaskStatusManualReview, resp.TaskStatus)
	require.NotNil(t, resp.TaskMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *resp.TaskMessage)
	require.Equal(t, proxySnapshot, *resp.DefaultProxySnapshot)
	require.True(t, resp.DefaultProxyConfigured)
	require.Equal(t, remark, *resp.Remark)
	require.Equal(t, createdAt, resp.CreatedAt)
	require.Equal(t, updatedAt, resp.UpdatedAt)
}

func TestUserSocialAccountDefaultProxyConfiguredRequiresUsableSnapshot(t *testing.T) {
	onlineSnapshot := `{"id":301,"endpoint":"http://proxy.local:8080","status":"online"}`
	offlineSnapshot := `{"id":301,"endpoint":"http://proxy.local:8080","status":"offline"}`
	emptyEndpointSnapshot := `{"id":301,"endpoint":"","status":"online"}`

	require.True(t, userSocialAccountDefaultProxyConfigured(&service.SocialAccount{DefaultProxySnapshot: &onlineSnapshot}))
	require.False(t, userSocialAccountDefaultProxyConfigured(&service.SocialAccount{DefaultProxySnapshot: &offlineSnapshot}))
	require.False(t, userSocialAccountDefaultProxyConfigured(&service.SocialAccount{DefaultProxySnapshot: &emptyEndpointSnapshot}))
	require.False(t, userSocialAccountDefaultProxyConfigured(&service.SocialAccount{}))
	require.False(t, userSocialAccountDefaultProxyConfigured(nil))
}

func TestAccountWorkbenchHandlerListTaskLogsFiltersCurrentUserActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "task-log-owner@example.com")
	otherUser := createSocialHandlerUser(t, ctx, client, "task-log-other@example.com")
	account := client.SocialAccount.Create().
		SetName("@task_log_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_log_owner").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("@task_log_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_log_other").
		SetAssignedUserID(otherUser.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	pendingLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLogin).
		SetStatus(service.SocialTaskLogStatusPending).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	failedLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLoginCheck).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	otherLog := client.SocialTaskLog.Create().
		SetSocialAccountID(otherAccount.ID).
		SetUserID(otherUser.ID).
		SetAction(service.SocialTaskActionLogin).
		SetStatus(service.SocialTaskLogStatusPending).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/accounts/tasks?log_ids="+formatID(pendingLog.ID)+","+formatID(failedLog.ID)+","+formatID(otherLog.ID)+"&account_ids="+formatID(account.ID)+"&statuses=pending,running",
		nil,
	)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.ListTaskLogs(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":`+formatID(pendingLog.ID))
	require.Contains(t, rec.Body.String(), `"status":"pending"`)
	require.NotContains(t, rec.Body.String(), `"id":`+formatID(failedLog.ID))
	require.NotContains(t, rec.Body.String(), `"id":`+formatID(otherLog.ID))
	require.NotContains(t, rec.Body.String(), "@task_log_other")
}

func TestAccountWorkbenchHandlerSubmitTaskUsesExecutorQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-queue@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_queue").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_queue").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "submit queue proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
	templateSvc := service.NewTaskSettingsService(client)
	tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "follow queue target",
		Type: service.SocialTaskActionFollow,
		Params: service.TaskTemplateParams{
			Targets: []string{"@target"},
		},
		IsDefault: true,
	})
	require.NoError(t, err)
	require.Equal(t, service.SocialTaskActionFollow, tmpl.Type)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"client_request_id": "g008-submit-queue"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"enqueued":1`)
	require.Contains(t, rec.Body.String(), `"failed_closed":0`)
	require.Zero(t, userRepo.deductCalls)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)

	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, service.SocialTaskLogStatusPending, logs[0].Status)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, logs[0].ChargeStatus)
	require.Zero(t, logs[0].ChargedAmount)
}

func TestAccountWorkbenchHandlerSubmitTaskRequiresTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-requires-template@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_requires_template").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_requires_template").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow",
		"target": "@target",
		"client_request_id": "direct-user-submit"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "TASK_DEFAULT_TEMPLATE_REQUIRED")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerSubmitLoginCheckDoesNotRequireDefaultTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-login-check-direct@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_login_check_direct").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_login_check_direct").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "submit login check direct proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	handler.templates = nil

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"client_request_id": "direct-login-check-submit"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, service.SocialTaskActionLoginCheck, logs[0].Action)
	require.True(t, logs[0].TemplateSnapshot.IsZero())
	require.Zero(t, userRepo.deductCalls)
}

func TestAccountWorkbenchHandlerSubmitTaskExpandsSavedTemplateParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-template@example.com")
	first := client.SocialAccount.Create().
		SetName("@template_first").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("template_first").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	second := client.SocialAccount.Create().
		SetName("@template_second").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("template_second").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, first.ID, "template first proxy")
	assignHandlerDefaultProxy(t, ctx, client, user.ID, second.ID, "template second proxy")
	templateSvc := service.NewTaskSettingsService(client)
	tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "follow targets",
		Type: service.SocialTaskActionFollow,
		Params: service.TaskTemplateParams{
			Targets: []string{"@target_one", "@target_two"},
		},
		IsDefault: true,
	})
	require.NoError(t, err)
	require.Equal(t, service.SocialTaskActionFollow, tmpl.Type)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(first.ID)+`, `+formatID(second.ID)+`],
		"action": "follow",
		"target": "@request_body_target_must_be_ignored",
		"content": "request body content must be ignored",
		"client_request_id": "g008-submit-template"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, service.SocialTaskActionFollow, logs[0].Action)
	require.NotNil(t, logs[0].Target)
	require.Equal(t, "@target_one", *logs[0].Target)
	require.Nil(t, logs[0].Content)
	require.NotNil(t, logs[1].Target)
	require.Equal(t, "@target_two", *logs[1].Target)
	require.Nil(t, logs[1].Content)
	require.Zero(t, userRepo.deductCalls)
}

func TestAccountWorkbenchHandlerSubmitTaskCapturesStructuredTemplateSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-structured-template@example.com")
	account := client.SocialAccount.Create().
		SetName("@structured_template").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("structured_template").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "structured template proxy")
	templateSvc := service.NewTaskSettingsService(client)
	tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "structured post",
		Type: service.SocialTaskActionPost,
		Params: service.TaskTemplateParams{
			Contents:     []string{"hello structured handler"},
			QuotePostURL: "https://x.com/northwind/status/1",
			Media: []service.SocialTaskMediaRef{
				{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "handler-post.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 640, 640),
				},
			},
		},
		IsDefault: true,
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "post",
		"client_request_id": "structured-template-submit"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Logs []struct {
				Action  string  `json:"action"`
				Target  *string `json:"target"`
				Content *string `json:"content"`
				Payload *struct {
					Post *struct {
						Text         string `json:"text"`
						QuotePostURL string `json:"quote_post_url"`
						Media        []struct {
							Source      string `json:"source"`
							StorageKey  string `json:"storage_key"`
							URL         string `json:"url"`
							ContentType string `json:"content_type"`
							FileName    string `json:"file_name"`
						} `json:"media"`
					} `json:"post"`
				} `json:"payload"`
				TemplateSnapshot *struct {
					TemplateID   string `json:"template_id"`
					TemplateName string `json:"template_name"`
					TemplateType string `json:"template_type"`
					Params       struct {
						Contents     []string `json:"contents"`
						QuotePostURL string   `json:"quote_post_url"`
						Media        []struct {
							Source      string `json:"source"`
							StorageKey  string `json:"storage_key"`
							URL         string `json:"url"`
							ContentType string `json:"content_type"`
							FileName    string `json:"file_name"`
						} `json:"media"`
					} `json:"params"`
				} `json:"template_snapshot"`
			} `json:"logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Logs, 1)
	require.Equal(t, service.SocialTaskActionPost, body.Data.Logs[0].Action)
	require.Nil(t, body.Data.Logs[0].Target)
	require.NotNil(t, body.Data.Logs[0].Content)
	require.Equal(t, "hello structured handler", *body.Data.Logs[0].Content)
	require.NotNil(t, body.Data.Logs[0].Payload)
	require.NotNil(t, body.Data.Logs[0].Payload.Post)
	require.Equal(t, "hello structured handler", body.Data.Logs[0].Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/1", body.Data.Logs[0].Payload.Post.QuotePostURL)
	require.Len(t, body.Data.Logs[0].Payload.Post.Media, 1)
	require.Equal(t, "inline", body.Data.Logs[0].Payload.Post.Media[0].Source)
	require.Equal(t, "image/png", body.Data.Logs[0].Payload.Post.Media[0].ContentType)
	require.Equal(t, "handler-post.png", body.Data.Logs[0].Payload.Post.Media[0].FileName)
	require.Empty(t, body.Data.Logs[0].Payload.Post.Media[0].StorageKey)
	require.Empty(t, body.Data.Logs[0].Payload.Post.Media[0].URL)
	require.NotNil(t, body.Data.Logs[0].TemplateSnapshot)
	require.Equal(t, tmpl.ID, body.Data.Logs[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "structured post", body.Data.Logs[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, body.Data.Logs[0].TemplateSnapshot.TemplateType)
	require.Equal(t, []string{"hello structured handler"}, body.Data.Logs[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "https://x.com/northwind/status/1", body.Data.Logs[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, body.Data.Logs[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "inline", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].Source)
	require.Equal(t, "image/png", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].ContentType)
	require.Equal(t, "handler-post.png", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].FileName)
	require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Media[0].URL)
	require.NotContains(t, rec.Body.String(), "data:image/png;base64")
	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].Payload.Post)
	require.Equal(t, "hello structured handler", logs[0].Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/1", logs[0].Payload.Post.QuotePostURL)
	require.Len(t, logs[0].Payload.Post.Media, 1)
	require.Equal(t, "library", logs[0].Payload.Post.Media[0].Source)
	require.Equal(t, "image/png", logs[0].Payload.Post.Media[0].ContentType)
	require.Equal(t, "handler-post.png", logs[0].Payload.Post.Media[0].FileName)
	require.NotEmpty(t, logs[0].Payload.Post.Media[0].StorageKey)
	require.Empty(t, logs[0].Payload.Post.Media[0].URL)
	require.Equal(t, tmpl.ID, logs[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "structured post", logs[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, logs[0].TemplateSnapshot.TemplateType)
	require.Equal(t, []string{"hello structured handler"}, logs[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "https://x.com/northwind/status/1", logs[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, logs[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "library", logs[0].TemplateSnapshot.Params.Media[0].Source)
	require.NotEmpty(t, logs[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, logs[0].TemplateSnapshot.Params.Media[0].URL)
	require.Zero(t, userRepo.deductCalls)
}

func TestAccountWorkbenchHandlerSubmitTaskCapturesMediaOnlyStructuredTemplateSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-media-only-template@example.com")
	account := client.SocialAccount.Create().
		SetName("@media_only_template").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("media_only_template").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "media only template proxy")
	templateSvc := service.NewTaskSettingsService(client)
	tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "media only post",
		Type: service.SocialTaskActionPost,
		Params: service.TaskTemplateParams{
			Media: []service.SocialTaskMediaRef{
				{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "handler-media-only-post.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 640, 640),
				},
			},
		},
		IsDefault: true,
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "post",
		"client_request_id": "media-only-template-submit"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Logs []struct {
				Action  string  `json:"action"`
				Target  *string `json:"target"`
				Content *string `json:"content"`
				Payload *struct {
					Post *struct {
						Text         string `json:"text"`
						QuotePostURL string `json:"quote_post_url"`
						Media        []struct {
							Source      string `json:"source"`
							StorageKey  string `json:"storage_key"`
							URL         string `json:"url"`
							ContentType string `json:"content_type"`
							FileName    string `json:"file_name"`
						} `json:"media"`
					} `json:"post"`
				} `json:"payload"`
				TemplateSnapshot *struct {
					TemplateID   string `json:"template_id"`
					TemplateName string `json:"template_name"`
					TemplateType string `json:"template_type"`
					Params       struct {
						Contents     []string `json:"contents"`
						QuotePostURL string   `json:"quote_post_url"`
						Media        []struct {
							Source      string `json:"source"`
							StorageKey  string `json:"storage_key"`
							URL         string `json:"url"`
							ContentType string `json:"content_type"`
							FileName    string `json:"file_name"`
						} `json:"media"`
					} `json:"params"`
				} `json:"template_snapshot"`
			} `json:"logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Logs, 1)
	require.Equal(t, service.SocialTaskActionPost, body.Data.Logs[0].Action)
	require.Nil(t, body.Data.Logs[0].Target)
	require.Nil(t, body.Data.Logs[0].Content)
	require.NotNil(t, body.Data.Logs[0].Payload)
	require.NotNil(t, body.Data.Logs[0].Payload.Post)
	require.Equal(t, "", body.Data.Logs[0].Payload.Post.Text)
	require.Equal(t, "", body.Data.Logs[0].Payload.Post.QuotePostURL)
	require.Len(t, body.Data.Logs[0].Payload.Post.Media, 1)
	require.Equal(t, "inline", body.Data.Logs[0].Payload.Post.Media[0].Source)
	require.Equal(t, "image/png", body.Data.Logs[0].Payload.Post.Media[0].ContentType)
	require.Equal(t, "handler-media-only-post.png", body.Data.Logs[0].Payload.Post.Media[0].FileName)
	require.Empty(t, body.Data.Logs[0].Payload.Post.Media[0].StorageKey)
	require.Empty(t, body.Data.Logs[0].Payload.Post.Media[0].URL)
	require.NotNil(t, body.Data.Logs[0].TemplateSnapshot)
	require.Equal(t, tmpl.ID, body.Data.Logs[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "media only post", body.Data.Logs[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, body.Data.Logs[0].TemplateSnapshot.TemplateType)
	require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "", body.Data.Logs[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, body.Data.Logs[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "inline", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].Source)
	require.Equal(t, "image/png", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].ContentType)
	require.Equal(t, "handler-media-only-post.png", body.Data.Logs[0].TemplateSnapshot.Params.Media[0].FileName)
	require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Media[0].URL)
	require.NotContains(t, rec.Body.String(), "data:image/png;base64")

	logs, err := client.SocialTaskLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].Payload.Post)
	require.Equal(t, "", logs[0].Payload.Post.Text)
	require.Equal(t, "", logs[0].Payload.Post.QuotePostURL)
	require.Len(t, logs[0].Payload.Post.Media, 1)
	require.Equal(t, "library", logs[0].Payload.Post.Media[0].Source)
	require.Equal(t, "image/png", logs[0].Payload.Post.Media[0].ContentType)
	require.Equal(t, "handler-media-only-post.png", logs[0].Payload.Post.Media[0].FileName)
	require.NotEmpty(t, logs[0].Payload.Post.Media[0].StorageKey)
	require.Empty(t, logs[0].Payload.Post.Media[0].URL)
	require.Equal(t, tmpl.ID, logs[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "media only post", logs[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, logs[0].TemplateSnapshot.TemplateType)
	require.Empty(t, logs[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "", logs[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, logs[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "library", logs[0].TemplateSnapshot.Params.Media[0].Source)
	require.NotEmpty(t, logs[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, logs[0].TemplateSnapshot.Params.Media[0].URL)
	require.Zero(t, userRepo.deductCalls)
}

func TestAccountWorkbenchHandlerSubmitTaskCapturesStructuredProfileMediaTemplateSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	testCases := []struct {
		name         string
		email        string
		templateName string
		taskType     string
		params       service.TaskTemplateParams
		assertLog    func(t *testing.T, log *dbent.SocialTaskLog)
	}{
		{
			name:         "profile update",
			email:        "submit-structured-profile@example.com",
			templateName: "structured profile",
			taskType:     service.SocialTaskActionUpdateProfile,
			params: service.TaskTemplateParams{
				Profile: &service.SocialProfileUpdateParams{
					DisplayName: "Northwind Ops",
					Description: "Operator account",
					Location:    "Singapore",
				},
			},
			assertLog: func(t *testing.T, log *dbent.SocialTaskLog) {
				require.NotNil(t, log.Payload.Profile)
				require.Equal(t, "Northwind Ops", log.Payload.Profile.DisplayName)
				require.Equal(t, "Operator account", log.Payload.Profile.Description)
				require.Equal(t, "Singapore", log.Payload.Profile.Location)
				require.NotNil(t, log.TemplateSnapshot.Params.Profile)
				require.Equal(t, "Northwind Ops", log.TemplateSnapshot.Params.Profile.DisplayName)
				require.Equal(t, "Operator account", log.TemplateSnapshot.Params.Profile.Description)
				require.Equal(t, "Singapore", log.TemplateSnapshot.Params.Profile.Location)
			},
		},
		{
			name:         "avatar update",
			email:        "submit-structured-avatar@example.com",
			templateName: "structured avatar",
			taskType:     service.SocialTaskActionUpdateAvatar,
			params: service.TaskTemplateParams{
				Avatar: &service.SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "handler-avatar.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 400, 400),
				},
			},
			assertLog: func(t *testing.T, log *dbent.SocialTaskLog) {
				require.NotNil(t, log.Payload.Avatar)
				require.Equal(t, "library", log.Payload.Avatar.Source)
				require.Equal(t, "image/png", log.Payload.Avatar.ContentType)
				require.Equal(t, "handler-avatar.png", log.Payload.Avatar.FileName)
				require.Equal(t, 400, log.Payload.Avatar.Width)
				require.Equal(t, 400, log.Payload.Avatar.Height)
				require.NotEmpty(t, log.Payload.Avatar.StorageKey)
				require.Empty(t, log.Payload.Avatar.URL)
				require.NotNil(t, log.TemplateSnapshot.Params.Avatar)
				require.Equal(t, "library", log.TemplateSnapshot.Params.Avatar.Source)
				require.Equal(t, "image/png", log.TemplateSnapshot.Params.Avatar.ContentType)
				require.Equal(t, "handler-avatar.png", log.TemplateSnapshot.Params.Avatar.FileName)
				require.Equal(t, 400, log.TemplateSnapshot.Params.Avatar.Width)
				require.Equal(t, 400, log.TemplateSnapshot.Params.Avatar.Height)
				require.NotEmpty(t, log.TemplateSnapshot.Params.Avatar.StorageKey)
				require.Empty(t, log.TemplateSnapshot.Params.Avatar.URL)
			},
		},
		{
			name:         "banner update",
			email:        "submit-structured-banner@example.com",
			templateName: "structured banner",
			taskType:     service.SocialTaskActionUpdateBanner,
			params: service.TaskTemplateParams{
				Banner: &service.SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "handler-banner.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 1500, 500),
				},
			},
			assertLog: func(t *testing.T, log *dbent.SocialTaskLog) {
				require.NotNil(t, log.Payload.Banner)
				require.Equal(t, "library", log.Payload.Banner.Source)
				require.Equal(t, "image/png", log.Payload.Banner.ContentType)
				require.Equal(t, "handler-banner.png", log.Payload.Banner.FileName)
				require.Equal(t, 1500, log.Payload.Banner.Width)
				require.Equal(t, 500, log.Payload.Banner.Height)
				require.NotEmpty(t, log.Payload.Banner.StorageKey)
				require.Empty(t, log.Payload.Banner.URL)
				require.NotNil(t, log.TemplateSnapshot.Params.Banner)
				require.Equal(t, "library", log.TemplateSnapshot.Params.Banner.Source)
				require.Equal(t, "image/png", log.TemplateSnapshot.Params.Banner.ContentType)
				require.Equal(t, "handler-banner.png", log.TemplateSnapshot.Params.Banner.FileName)
				require.Equal(t, 1500, log.TemplateSnapshot.Params.Banner.Width)
				require.Equal(t, 500, log.TemplateSnapshot.Params.Banner.Height)
				require.NotEmpty(t, log.TemplateSnapshot.Params.Banner.StorageKey)
				require.Empty(t, log.TemplateSnapshot.Params.Banner.URL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newAccountWorkbenchHandlerTestClient(t)
			user := createSocialHandlerUser(t, ctx, client, tc.email)
			account := client.SocialAccount.Create().
				SetName("@structured_profile_media").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("structured_profile_media").
				SetAssignedUserID(user.ID).
				SetAccountStatus(service.SocialAccountStatusAvailable).
				SetTaskStatus(service.SocialTaskStatusStored).
				SaveX(ctx)
			assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, tc.name+" proxy")
			templateSvc := service.NewTaskSettingsService(client)
			tmpl, err := templateSvc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
				Name:      tc.templateName,
				Type:      tc.taskType,
				Params:    tc.params,
				IsDefault: true,
			})
			require.NoError(t, err)
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTest(client, userRepo)

			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
				"account_ids": [`+formatID(account.ID)+`],
				"action": "`+tc.taskType+`",
				"client_request_id": "`+strings.ReplaceAll(tc.taskType, "_", "-")+`-submit"
			}`))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

			handler.SubmitTask(ginCtx)

			require.Equal(t, http.StatusOK, rec.Code)
			var body struct {
				Data struct {
					Logs []struct {
						Action  string  `json:"action"`
						Target  *string `json:"target"`
						Content *string `json:"content"`
						Payload *struct {
							Profile *struct {
								DisplayName string `json:"display_name"`
								Description string `json:"description"`
								Location    string `json:"location"`
							} `json:"profile"`
							Avatar *struct {
								Source      string `json:"source"`
								StorageKey  string `json:"storage_key"`
								URL         string `json:"url"`
								ContentType string `json:"content_type"`
								FileName    string `json:"file_name"`
								Width       int    `json:"width"`
								Height      int    `json:"height"`
							} `json:"avatar"`
							Banner *struct {
								Source      string `json:"source"`
								StorageKey  string `json:"storage_key"`
								URL         string `json:"url"`
								ContentType string `json:"content_type"`
								FileName    string `json:"file_name"`
								Width       int    `json:"width"`
								Height      int    `json:"height"`
							} `json:"banner"`
						} `json:"payload"`
						TemplateSnapshot *struct {
							TemplateType string `json:"template_type"`
							Params       struct {
								Profile *struct {
									DisplayName string `json:"display_name"`
									Description string `json:"description"`
									Location    string `json:"location"`
								} `json:"profile"`
								Avatar *struct {
									Source      string `json:"source"`
									StorageKey  string `json:"storage_key"`
									URL         string `json:"url"`
									ContentType string `json:"content_type"`
									FileName    string `json:"file_name"`
									Width       int    `json:"width"`
									Height      int    `json:"height"`
								} `json:"avatar"`
								Banner *struct {
									Source      string `json:"source"`
									StorageKey  string `json:"storage_key"`
									URL         string `json:"url"`
									ContentType string `json:"content_type"`
									FileName    string `json:"file_name"`
									Width       int    `json:"width"`
									Height      int    `json:"height"`
								} `json:"banner"`
							} `json:"params"`
						} `json:"template_snapshot"`
					} `json:"logs"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Len(t, body.Data.Logs, 1)
			require.Equal(t, tc.taskType, body.Data.Logs[0].Action)
			require.Nil(t, body.Data.Logs[0].Target)
			require.Nil(t, body.Data.Logs[0].Content)
			require.NotNil(t, body.Data.Logs[0].Payload)
			require.NotNil(t, body.Data.Logs[0].TemplateSnapshot)
			require.Equal(t, tc.taskType, body.Data.Logs[0].TemplateSnapshot.TemplateType)

			switch tc.taskType {
			case service.SocialTaskActionUpdateProfile:
				require.NotNil(t, body.Data.Logs[0].Payload.Profile)
				require.Equal(t, "Northwind Ops", body.Data.Logs[0].Payload.Profile.DisplayName)
				require.Equal(t, "Operator account", body.Data.Logs[0].Payload.Profile.Description)
				require.Equal(t, "Singapore", body.Data.Logs[0].Payload.Profile.Location)
				require.NotNil(t, body.Data.Logs[0].TemplateSnapshot.Params.Profile)
				require.Equal(t, "Northwind Ops", body.Data.Logs[0].TemplateSnapshot.Params.Profile.DisplayName)
			case service.SocialTaskActionUpdateAvatar:
				require.NotNil(t, body.Data.Logs[0].Payload.Avatar)
				require.Equal(t, "inline", body.Data.Logs[0].Payload.Avatar.Source)
				require.Equal(t, "image/png", body.Data.Logs[0].Payload.Avatar.ContentType)
				require.Equal(t, "handler-avatar.png", body.Data.Logs[0].Payload.Avatar.FileName)
				require.Equal(t, 400, body.Data.Logs[0].Payload.Avatar.Width)
				require.Equal(t, 400, body.Data.Logs[0].Payload.Avatar.Height)
				require.Empty(t, body.Data.Logs[0].Payload.Avatar.StorageKey)
				require.Empty(t, body.Data.Logs[0].Payload.Avatar.URL)
				require.NotNil(t, body.Data.Logs[0].TemplateSnapshot.Params.Avatar)
				require.Equal(t, "inline", body.Data.Logs[0].TemplateSnapshot.Params.Avatar.Source)
				require.Equal(t, 400, body.Data.Logs[0].TemplateSnapshot.Params.Avatar.Width)
				require.Equal(t, 400, body.Data.Logs[0].TemplateSnapshot.Params.Avatar.Height)
				require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Avatar.StorageKey)
			case service.SocialTaskActionUpdateBanner:
				require.NotNil(t, body.Data.Logs[0].Payload.Banner)
				require.Equal(t, "inline", body.Data.Logs[0].Payload.Banner.Source)
				require.Equal(t, "image/png", body.Data.Logs[0].Payload.Banner.ContentType)
				require.Equal(t, "handler-banner.png", body.Data.Logs[0].Payload.Banner.FileName)
				require.Equal(t, 1500, body.Data.Logs[0].Payload.Banner.Width)
				require.Equal(t, 500, body.Data.Logs[0].Payload.Banner.Height)
				require.Empty(t, body.Data.Logs[0].Payload.Banner.StorageKey)
				require.Empty(t, body.Data.Logs[0].Payload.Banner.URL)
				require.NotNil(t, body.Data.Logs[0].TemplateSnapshot.Params.Banner)
				require.Equal(t, "inline", body.Data.Logs[0].TemplateSnapshot.Params.Banner.Source)
				require.Equal(t, 1500, body.Data.Logs[0].TemplateSnapshot.Params.Banner.Width)
				require.Equal(t, 500, body.Data.Logs[0].TemplateSnapshot.Params.Banner.Height)
				require.Empty(t, body.Data.Logs[0].TemplateSnapshot.Params.Banner.StorageKey)
			}
			logs, err := client.SocialTaskLog.Query().All(ctx)
			require.NoError(t, err)
			require.Len(t, logs, 1)
			require.Nil(t, logs[0].Target)
			require.Nil(t, logs[0].Content)
			require.Equal(t, tmpl.ID, logs[0].TemplateSnapshot.TemplateID)
			require.Equal(t, tc.templateName, logs[0].TemplateSnapshot.TemplateName)
			require.Equal(t, tc.taskType, logs[0].TemplateSnapshot.TemplateType)
			tc.assertLog(t, logs[0])
			require.Zero(t, userRepo.deductCalls)
		})
	}
}

func TestAccountWorkbenchHandlerRejectsMixedPlatformBatchBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "mixed-platform-task@example.com")
	xAccount := client.SocialAccount.Create().
		SetName("@mixed_x").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("mixed_x").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	instagramAccount := client.SocialAccount.Create().
		SetName("@mixed_instagram").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("mixed_instagram").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, xAccount.ID, "mixed x proxy")
	assignHandlerDefaultProxy(t, ctx, client, user.ID, instagramAccount.ID, "mixed instagram proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(xAccount.ID)+`, `+formatID(instagramAccount.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_MIXED_PLATFORMS")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsUnsupportedActionBeforeBilling(t *testing.T) {
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
			client := newAccountWorkbenchHandlerTestClient(t)
			user := createSocialHandlerUser(t, ctx, client, name+"-unsupported-action@example.com")
			account := client.SocialAccount.Create().
				SetName("@" + name + "_unsupported_action").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(name + "_unsupported_action").
				SetAssignedUserID(user.ID).
				SetAccountStatus(service.SocialAccountStatusAvailable).
				SetTaskStatus(service.SocialTaskStatusStored).
				SaveX(ctx)
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)

			body := `{
				"account_ids": [` + formatID(account.ID) + `],
				"action": "` + action + `",
				"client_request_id": "unsupported-action"
			}`
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(body))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

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

func TestAccountWorkbenchHandlerRejectsMissingActionWithUnsupportedActionCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "missing-task-action@example.com")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [1],
		"client_request_id": "missing-action"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_UNSUPPORTED_ACTION")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsInvalidStoredTemplateBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "invalid-template-submit@example.com")
	account := client.SocialAccount.Create().
		SetName("@invalid_template_submit").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("invalid_template_submit").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "invalid template proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
	templateID := "oversized_stored_template"
	now := time.Now().UTC().Format(time.RFC3339)
	client.Setting.Create().
		SetKey("socialops:task_settings:user:" + formatID(user.ID)).
		SetValue(`{"templates":[{"id":"` + templateID + `","name":"oversized","type":"post","params":{"contents":["` + strings.Repeat("a", 2049) + `"]},"is_default":true,"created_at":"` + now + `","updated_at":"` + now + `"}]}`).
		SaveX(ctx)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "post",
		"client_request_id": "invalid-stored-template"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_INVALID")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsStoredProfileMediaTemplatesWithInvalidDimensionsBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	testCases := []struct {
		name            string
		email           string
		templateID      string
		templateName    string
		templateType    string
		params          service.TaskTemplateParams
		expectedMessage string
	}{
		{
			name:         "avatar dimensions prefer actual inline image size",
			email:        "invalid-avatar-template-submit@example.com",
			templateID:   "stale_avatar_template",
			templateName: "stale avatar",
			templateType: service.SocialTaskActionUpdateAvatar,
			params: service.TaskTemplateParams{
				Avatar: &service.SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "avatar.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 300, 300),
					Width:       400,
					Height:      400,
				},
			},
			expectedMessage: "avatar image must be 400x400 pixels",
		},
		{
			name:         "banner dimensions prefer actual inline image size",
			email:        "invalid-banner-template-submit@example.com",
			templateID:   "stale_banner_template",
			templateName: "stale banner",
			templateType: service.SocialTaskActionUpdateBanner,
			params: service.TaskTemplateParams{
				Banner: &service.SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "banner.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 1400, 500),
					Width:       1500,
					Height:      500,
				},
			},
			expectedMessage: "banner image must be 1500x500 pixels",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newAccountWorkbenchHandlerTestClient(t)
			user := createSocialHandlerUser(t, ctx, client, tc.email)
			account := client.SocialAccount.Create().
				SetName("@invalid_profile_media_template").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("invalid_profile_media_template").
				SetAssignedUserID(user.ID).
				SetAccountStatus(service.SocialAccountStatusAvailable).
				SetTaskStatus(service.SocialTaskStatusStored).
				SaveX(ctx)
			assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, tc.name+" proxy")
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
			now := time.Now().UTC()
			doc := struct {
				Templates []*service.TaskTemplate `json:"templates"`
			}{
				Templates: []*service.TaskTemplate{{
					ID:        tc.templateID,
					Name:      tc.templateName,
					Type:      tc.templateType,
					Params:    tc.params,
					IsDefault: true,
					CreatedAt: now,
					UpdatedAt: now,
				}},
			}
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			client.Setting.Create().
				SetKey("socialops:task_settings:user:" + formatID(user.ID)).
				SetValue(string(raw)).
				SaveX(ctx)

			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
				"account_ids": [`+formatID(account.ID)+`],
				"action": "`+tc.templateType+`",
				"client_request_id": "`+tc.templateID+`-submit"
			}`))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

			handler.SubmitTask(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_INVALID")
			require.Contains(t, rec.Body.String(), tc.expectedMessage)
			require.Zero(t, userRepo.deductCalls)

			count, err := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestAccountWorkbenchHandlerRejectsInvalidStoredPostMediaTemplatesBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	testCases := []struct {
		name            string
		email           string
		templateID      string
		templateName    string
		params          service.TaskTemplateParams
		expectedMessage string
	}{
		{
			name:         "video media is blocked before submit",
			email:        "invalid-post-video-template-submit@example.com",
			templateID:   "stale_post_video_template",
			templateName: "stale post video",
			params: service.TaskTemplateParams{
				Contents: []string{"hello video"},
				Media: []service.SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "video/mp4",
					FileName:    "clip.mp4",
					URL:         "data:video/mp4;base64,QUJD",
				}},
			},
			expectedMessage: "video media is not supported for SocialOps execution",
		},
		{
			name:         "unsupported media type is blocked before submit",
			email:        "invalid-post-media-type-template-submit@example.com",
			templateID:   "stale_post_file_template",
			templateName: "stale post file",
			params: service.TaskTemplateParams{
				Contents: []string{"hello file"},
				Media: []service.SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "application/pdf",
					FileName:    "spec.pdf",
					URL:         "data:application/pdf;base64,QUJD",
				}},
			},
			expectedMessage: "post media content type is not supported",
		},
		{
			name:         "non-inline media source is blocked before submit",
			email:        "invalid-post-media-source-template-submit@example.com",
			templateID:   "stale_post_library_template",
			templateName: "stale post library",
			params: service.TaskTemplateParams{
				Contents: []string{"hello library"},
				Media: []service.SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "media/post-image.jpg",
					ContentType: "image/jpeg",
				}},
			},
			expectedMessage: "post media #1 media source is not supported for SocialOps execution",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newAccountWorkbenchHandlerTestClient(t)
			user := createSocialHandlerUser(t, ctx, client, tc.email)
			account := client.SocialAccount.Create().
				SetName("@invalid_post_media_template").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("invalid_post_media_template").
				SetAssignedUserID(user.ID).
				SetAccountStatus(service.SocialAccountStatusAvailable).
				SetTaskStatus(service.SocialTaskStatusStored).
				SaveX(ctx)
			assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, tc.name+" proxy")
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
			now := time.Now().UTC()
			doc := struct {
				Templates []*service.TaskTemplate `json:"templates"`
			}{
				Templates: []*service.TaskTemplate{{
					ID:        tc.templateID,
					Name:      tc.templateName,
					Type:      service.SocialTaskActionPost,
					Params:    tc.params,
					IsDefault: true,
					CreatedAt: now,
					UpdatedAt: now,
				}},
			}
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			client.Setting.Create().
				SetKey("socialops:task_settings:user:" + formatID(user.ID)).
				SetValue(string(raw)).
				SaveX(ctx)

			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
				"account_ids": [`+formatID(account.ID)+`],
				"action": "post",
				"client_request_id": "`+tc.templateID+`-submit"
			}`))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

			handler.SubmitTask(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_INVALID")
			require.Contains(t, rec.Body.String(), tc.expectedMessage)
			require.Zero(t, userRepo.deductCalls)

			count, err := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestAccountWorkbenchHandlerRejectsInvalidStoredProfileMediaSourcesBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	testCases := []struct {
		name            string
		email           string
		templateID      string
		templateName    string
		templateType    string
		params          service.TaskTemplateParams
		expectedMessage string
	}{
		{
			name:         "avatar library media source is blocked before submit",
			email:        "invalid-avatar-media-source-template-submit@example.com",
			templateID:   "stale_avatar_library_template",
			templateName: "stale avatar library",
			templateType: service.SocialTaskActionUpdateAvatar,
			params: service.TaskTemplateParams{
				Avatar: &service.SocialTaskMediaRef{
					Source:      "library",
					StorageKey:  "media/avatar.jpg",
					ContentType: "image/jpeg",
					Width:       400,
					Height:      400,
				},
			},
			expectedMessage: "avatar media source is not supported for SocialOps execution",
		},
		{
			name:         "banner library media source is blocked before submit",
			email:        "invalid-banner-media-source-template-submit@example.com",
			templateID:   "stale_banner_library_template",
			templateName: "stale banner library",
			templateType: service.SocialTaskActionUpdateBanner,
			params: service.TaskTemplateParams{
				Banner: &service.SocialTaskMediaRef{
					Source:      "library",
					StorageKey:  "media/banner.jpg",
					ContentType: "image/jpeg",
					Width:       1500,
					Height:      500,
				},
			},
			expectedMessage: "banner media source is not supported for SocialOps execution",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newAccountWorkbenchHandlerTestClient(t)
			user := createSocialHandlerUser(t, ctx, client, tc.email)
			account := client.SocialAccount.Create().
				SetName("@invalid_profile_media_source_template").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("invalid_profile_media_source_template").
				SetAssignedUserID(user.ID).
				SetAccountStatus(service.SocialAccountStatusAvailable).
				SetTaskStatus(service.SocialTaskStatusStored).
				SaveX(ctx)
			assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, tc.name+" proxy")
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
			now := time.Now().UTC()
			doc := struct {
				Templates []*service.TaskTemplate `json:"templates"`
			}{
				Templates: []*service.TaskTemplate{{
					ID:        tc.templateID,
					Name:      tc.templateName,
					Type:      tc.templateType,
					Params:    tc.params,
					IsDefault: true,
					CreatedAt: now,
					UpdatedAt: now,
				}},
			}
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			client.Setting.Create().
				SetKey("socialops:task_settings:user:" + formatID(user.ID)).
				SetValue(string(raw)).
				SaveX(ctx)

			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
				"account_ids": [`+formatID(account.ID)+`],
				"action": "`+tc.templateType+`",
				"client_request_id": "`+tc.templateID+`-submit"
			}`))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

			handler.SubmitTask(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_INVALID")
			require.Contains(t, rec.Body.String(), tc.expectedMessage)
			require.Zero(t, userRepo.deductCalls)

			count, err := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestAccountWorkbenchHandlerSubmitTaskDeduplicatesAccountIDsWithoutIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-dedupe@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_dedupe").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_dedupe").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "submit dedupe proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`, `+formatID(account.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"submitted":1`)
	require.Contains(t, rec.Body.String(), `"failed_closed":1`)
	require.Zero(t, userRepo.deductCalls)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAccountWorkbenchHandlerSubmitTaskRejectsNonPositiveAccountIDWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-invalid-id@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_invalid_id").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_invalid_id").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [-1, `+formatID(account.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_ACCOUNT_ID_INVALID")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsUnavailableAccountWithoutLogOrCharge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "limited-task@example.com")
	account := client.SocialAccount.Create().
		SetName("@limited_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("limited_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusLimited).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "account is not available")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsInsufficientFundsWithoutLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "no-funds@example.com")
	account := client.SocialAccount.Create().
		SetName("@no_funds").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("no_funds").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "no funds proxy")
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_TASK_INSUFFICIENT_FUNDS")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerRejectsStaleDefaultProxyWithoutLogOrCharge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "offline-default-proxy@example.com")
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("offline default proxy").
		SetIPType("residential").
		SetStatus(service.SocialIPStatusOffline).
		SaveX(ctx)
	snapshotIP := &service.SocialIP{ID: ip.ID, UserID: user.ID, Name: ip.Name, IPType: ip.IPType, Status: ip.Status}
	snapshot := service.SocialIPTaskSnapshot(snapshotIP)
	account := client.SocialAccount.Create().
		SetName("@offline_proxy_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("offline_proxy_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "follow"
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SubmitTask(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_NOT_AVAILABLE")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerListsOnlyCurrentUserResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	owner := createSocialHandlerUser(t, ctx, client, "list-owner@example.com")
	other := createSocialHandlerUser(t, ctx, client, "list-other@example.com")
	ownAccount := client.SocialAccount.Create().
		SetName("@own_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("own_account").
		SetAssignedUserID(owner.ID).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@other_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("other_account").
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	client.SocialIP.Create().
		SetUserID(other.ID).
		SetName("other proxy").
		SetIPType("residential").
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: owner.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, owner.ID, http.MethodGet, "/api/v1/accounts", nil, handler.ListMyAccounts)
	require.Equal(t, http.StatusOK, rec.Code)
	requireSinglePaginatedID(t, rec.Body.Bytes(), ownAccount.ID)
}

func TestAccountWorkbenchHandlerListMyAccountsIncludesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "safe-list@example.com")
	proxySnapshot := `{"id":301,"name":"delivery proxy","ip_type":"residential","endpoint":"http://user:pass@proxy.local:8080","status":"online"}`
	accountID := "pool-account-id"
	password := "pool-secret"
	phone := "+15550000001"
	email := "safe-list@example.com"
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=list; auth_token=list"
	executionAuthSecret := "encrypted-list-execution-auth-ciphertext"

	account := client.SocialAccount.Create().
		SetName("@safe_list").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("safe_list").
		SetPlatformUserID(accountID).
		SetPassword(password).
		SetPhone(phone).
		SetEmail(email).
		SetEmailPassword(emailPassword).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetDefaultProxySnapshot(proxySnapshot).
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts", nil, handler.ListMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	requireSinglePaginatedID(t, rec.Body.Bytes(), account.ID)
	body := rec.Body.String()
	require.Contains(t, body, "@safe_list")
	require.Contains(t, body, `"platform_user_id":"pool-account-id"`)
	require.Contains(t, body, `"password":"pool-secret"`)
	require.Contains(t, body, `"phone":"+15550000001"`)
	require.Contains(t, body, `"email":"safe-list@example.com"`)
	require.Contains(t, body, `"email_password":"mail-secret"`)
	require.Contains(t, body, `"auth_cookie":"ct0=list; auth_token=list"`)
	require.Contains(t, body, `"execution_auth":"encrypted-list-execution-auth-ciphertext"`)
	require.NotContains(t, body, "access_token")
	require.NotContains(t, body, "token_secret")
	require.Contains(t, body, `http://user:pass@proxy.local:8080`)
	require.Contains(t, body, `"default_proxy_configured":true`)
}

func TestAccountWorkbenchHandlerListMyAccountsTreatsStaleProxySnapshotsAsNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "stale-proxy-list@example.com")
	staleSnapshots := []string{
		`http://proxy.local:8080`,
		`{"id":301,"name":"offline proxy","ip_type":"residential","endpoint":"http://proxy.local:8080","status":"offline"}`,
		`{"id":302,"name":"empty endpoint","ip_type":"residential","endpoint":"","status":"online"}`,
	}
	for index, snapshot := range staleSnapshots {
		client.SocialAccount.Create().
			SetName("@stale_proxy_" + strconv.Itoa(index)).
			SetPlatform("x_twitter").
			SetPlatformKey("x_twitter").
			SetNameKey("stale_proxy_" + strconv.Itoa(index)).
			SetDefaultProxySnapshot(snapshot).
			SetAssignedUserID(user.ID).
			SetAccountStatus(service.SocialAccountStatusAvailable).
			SetTaskStatus(service.SocialTaskStatusStored).
			SaveX(ctx)
	}
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts", nil, handler.ListMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				DefaultProxySnapshot   *string `json:"default_proxy_snapshot"`
				DefaultProxyConfigured bool    `json:"default_proxy_configured"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, len(staleSnapshots))
	for _, item := range resp.Data.Items {
		require.NotNil(t, item.DefaultProxySnapshot)
		require.False(t, item.DefaultProxyConfigured)
	}
}

func TestAccountWorkbenchHandlerListMyAccountsAppliesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "filtered-list@example.com")
	other := createSocialHandlerUser(t, ctx, client, "filtered-list-other@example.com")
	match := client.SocialAccount.Create().
		SetName("@filtered_match").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("filtered_match").
		SetAssignedUserID(user.ID).
		SetPassword("list-filter-secret").
		SetDefaultProxySnapshot(`{"id":301,"endpoint":"http://list-filter-proxy.example:8080"}`).
		SetAccountStatus(service.SocialAccountStatusNotStored).
		SetTaskStatus(service.SocialTaskStatusPending).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@filtered_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("filtered_other").
		SetAssignedUserID(user.ID).
		SetPassword("other-secret").
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@filtered_cross_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("filtered_cross_owner").
		SetAssignedUserID(other.ID).
		SetPassword("list-filter-secret").
		SetAccountStatus(service.SocialAccountStatusNotStored).
		SetTaskStatus(service.SocialTaskStatusPending).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts?search=list-filter-secret&platform=x_twitter&account_status=invalid&task_status=pending", nil, handler.ListMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	requireSinglePaginatedID(t, rec.Body.Bytes(), match.ID)
	body := rec.Body.String()
	require.Contains(t, body, `"password":"list-filter-secret"`)
	require.Contains(t, body, `"default_proxy_snapshot":"{\"id\":301,\"endpoint\":\"http://list-filter-proxy.example:8080\"}"`)
	require.NotContains(t, body, "@filtered_other")
	require.NotContains(t, body, "@filtered_cross_owner")
}

func TestAccountWorkbenchHandlerListMyAccountsNormalizesInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "list-pagination@example.com")
	account := client.SocialAccount.Create().
		SetName("@list_pagination").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("list_pagination").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts?page=0&page_size=0", nil, handler.ListMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
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
	require.Equal(t, account.ID, resp.Data.Items[0].ID)
}

func TestAccountWorkbenchHandlerListMyAccountsSanitizesTaskMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "account-task-message@example.com")
	internalMessage := `upstream response body {"error":"secret","headers":{"authorization":"Bearer abc"}} token=secret-token proxy=http://user:pass@127.0.0.1:8080 trace_id=trace-123`
	client.SocialAccount.Create().
		SetName("@task_message").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_message").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusInvalid).
		SetTaskStatus(service.SocialTaskStatusManualReview).
		SetTaskMessage(internalMessage).
		SetAuthCookie("ct0=sensitive; auth_token=sensitive").
		SetExecutionAuth("encrypted-task-message-execution-auth-ciphertext").
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts", nil, handler.ListMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "账号认证信息不可用，本次未扣费")
	require.NotContains(t, body, "upstream response body")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "Bearer abc")
	require.NotContains(t, body, "secret-token")
	require.NotContains(t, body, "user:pass")
	require.NotContains(t, body, "127.0.0.1")
	require.NotContains(t, body, "trace-123")
	require.Contains(t, body, `"auth_cookie":"ct0=sensitive; auth_token=sensitive"`)
	require.Contains(t, body, `"execution_auth":"encrypted-task-message-execution-auth-ciphertext"`)
}

func TestAccountWorkbenchHandlerSetDefaultProxyIncludesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "safe-proxy@example.com")
	password := "pool-secret"
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=proxy; auth_token=proxy"
	executionAuthSecret := "encrypted-proxy-execution-auth-ciphertext"
	endpoint := "http://user:pass@proxy.local:8080"
	account := client.SocialAccount.Create().
		SetName("@safe_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("safe_proxy").
		SetPassword(password).
		SetEmailPassword(emailPassword).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("safe proxy").
		SetIPType("residential").
		SetEndpoint(endpoint).
		SetStatus(service.SocialIPStatusOnline).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/accounts/"+formatID(account.ID)+"/default-proxy", bytes.NewBufferString(`{"proxy_id":`+formatID(proxy.ID)+`}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SetDefaultProxy(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "@safe_proxy")
	require.Contains(t, body, `"password":"pool-secret"`)
	require.Contains(t, body, `"email_password":"mail-secret"`)
	require.Contains(t, body, `"auth_cookie":"ct0=proxy; auth_token=proxy"`)
	require.Contains(t, body, `"execution_auth":"encrypted-proxy-execution-auth-ciphertext"`)
	require.Contains(t, body, endpoint)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Contains(t, *stored.DefaultProxySnapshot, endpoint)
}

func TestAccountWorkbenchHandlerSetDefaultProxyRejectsOnlineProxyWithoutEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "missing-endpoint-default-proxy@example.com")
	account := client.SocialAccount.Create().
		SetName("@missing_endpoint_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("missing_endpoint_proxy").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("missing endpoint proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetStatus(service.SocialIPStatusOnline).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/accounts/"+formatID(account.ID)+"/default-proxy", bytes.NewBufferString(`{"proxy_id":`+formatID(proxy.ID)+`}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SetDefaultProxy(ginCtx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_NOT_AVAILABLE")
	require.Contains(t, rec.Body.String(), "social IP endpoint is required for execution")
	require.Nil(t, client.SocialAccount.GetX(ctx, account.ID).DefaultProxySnapshot)
}

func TestAccountWorkbenchHandlerSetDefaultProxyClearsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "clear-proxy@example.com")
	account := client.SocialAccount.Create().
		SetName("@clear_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("clear_proxy").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(`{"id":99,"name":"old","endpoint":"http://old-proxy.example:8080","status":"online"}`).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/accounts/"+formatID(account.ID)+"/default-proxy", bytes.NewBufferString(`{"proxy_id":null}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.SetDefaultProxy(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `"default_proxy_configured":false`)
	require.NotContains(t, body, `"default_proxy_snapshot"`)
	require.NotContains(t, body, "old-proxy.example")

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.DefaultProxySnapshot)
}

func TestAccountWorkbenchHandlerBatchSetDefaultProxyRejectsInvalidModeWithContractCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "invalid-proxy-mode@example.com")
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeJSONSocialHandlerAsUser(
		t,
		user.ID,
		http.MethodPost,
		"/api/v1/accounts/default-proxy",
		[]byte(`{"account_ids":[1],"mode":"debug-all-users"}`),
		handler.BatchSetDefaultProxy,
	)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_ASSIGNMENT_MODE_INVALID")
	require.Contains(t, rec.Body.String(), "proxy assignment mode is invalid")
}

func TestAccountWorkbenchHandlerBatchSetDefaultProxyRequiresProxyIDWithContractCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "missing-proxy-id@example.com")
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeJSONSocialHandlerAsUser(
		t,
		user.ID,
		http.MethodPost,
		"/api/v1/accounts/default-proxy",
		[]byte(`{"account_ids":[1],"mode":"specific"}`),
		handler.BatchSetDefaultProxy,
	)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_REQUIRED")
	require.Contains(t, rec.Body.String(), "proxy is required for this assignment")
}

func TestAccountWorkbenchHandlerBatchSetDefaultProxyRejectsRandomWithoutOnlinePoolWithContractCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "empty-random-proxy-pool@example.com")
	account := client.SocialAccount.Create().
		SetName("@empty_random_proxy_pool").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("empty_random_proxy_pool").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeJSONSocialHandlerAsUser(
		t,
		user.ID,
		http.MethodPost,
		"/api/v1/accounts/default-proxy",
		[]byte(`{"account_ids":[`+formatID(account.ID)+`],"mode":"random"}`),
		handler.BatchSetDefaultProxy,
	)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "SOCIAL_IP_POOL_EMPTY")
	require.Contains(t, rec.Body.String(), "no online proxy is available for assignment")
}

func TestAccountWorkbenchHandlerBatchSetDefaultProxyReturnsRowFailureForOfflineSpecificProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "offline-batch-proxy@example.com")
	account := client.SocialAccount.Create().
		SetName("@offline_batch_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("offline_batch_proxy").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("offline batch proxy").
		SetIPType(service.SocialIPTypeResidential).
		SetEndpoint("http://8.8.8.8:8080").
		SetStatus(service.SocialIPStatusOffline).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeJSONSocialHandlerAsUser(
		t,
		user.ID,
		http.MethodPost,
		"/api/v1/accounts/default-proxy",
		[]byte(`{"account_ids":[`+formatID(account.ID)+`],"mode":"specific","proxy_id":`+formatID(proxy.ID)+`}`),
		handler.BatchSetDefaultProxy,
	)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `"total":1`)
	require.Contains(t, body, `"succeeded":0`)
	require.Contains(t, body, `"failed":1`)
	require.Contains(t, body, `"skipped":0`)
	require.Contains(t, body, `"status":"failed"`)
	require.Contains(t, body, `"reason":"proxy_not_available"`)
	require.Contains(t, body, "account proxy could not be assigned")
	require.NotContains(t, body, "SOCIAL_IP_NOT_AVAILABLE")
}

func TestAccountWorkbenchHandlerHidesCrossUserAccountScopeErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "scope-user@example.com")
	other := createSocialHandlerUser(t, ctx, client, "scope-other@example.com")
	otherAccount := client.SocialAccount.Create().
		SetName("@scope_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("scope_other").
		SetAssignedUserID(other.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

	saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})
	taskBody := []byte(`{"account_ids":[` + formatID(otherAccount.ID) + `],"action":"follow"}`)
	submitRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts"+"/tasks", taskBody, handler.SubmitTask)
	requireUserAccountNotFoundResponse(t, submitRec, otherAccount.ID, "@scope_other")

	proxyRec := httptest.NewRecorder()
	proxyCtx, _ := gin.CreateTestContext(proxyRec)
	proxyCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/accounts/"+formatID(otherAccount.ID)+"/default-proxy", bytes.NewBufferString(`{"proxy_id":null}`))
	proxyCtx.Request.Header.Set("Content-Type", "application/json")
	proxyCtx.Params = gin.Params{{Key: "id", Value: formatID(otherAccount.ID)}}
	proxyCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})
	handler.SetDefaultProxy(proxyCtx)
	requireUserAccountNotFoundResponse(t, proxyRec, otherAccount.ID, "@scope_other")

	batchProxyBody := []byte(`{"account_ids":[` + formatID(otherAccount.ID) + `],"mode":"clear"}`)
	batchProxyRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/default-proxy", batchProxyBody, handler.BatchSetDefaultProxy)
	require.Equal(t, http.StatusOK, batchProxyRec.Code)
	require.Contains(t, batchProxyRec.Body.String(), `"failed":1`)
	require.Contains(t, batchProxyRec.Body.String(), `"reason":"account_not_assigned"`)
	require.NotContains(t, batchProxyRec.Body.String(), "@scope_other")
	require.NotContains(t, batchProxyRec.Body.String(), "SOCIAL_ACCOUNT_NOT_ASSIGNED")

	taskLogCount, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, taskLogCount)
}

func TestAccountWorkbenchHandlerUpdateMyAccountKeepsIdentityReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "update-workbench@example.com")
	other := createSocialHandlerUser(t, ctx, client, "update-workbench-other@example.com")
	account := client.SocialAccount.Create().
		SetName("@handler_identity").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_identity").
		SetIdentityKind("username").
		SetIdentityKey("x_twitter:username:handler_identity").
		SetPlatformUserID("rest-123").
		SetPassword("old-password").
		SetEmail("old@example.com").
		SetTwoFactor("old-2fa").
		SetRegistrationIP("198.51.100.20").
		SetAuthCookie("ct0=old; auth_token=old").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	original := client.SocialAccount.GetX(ctx, account.ID)
	otherAccount := client.SocialAccount.Create().
		SetName("@handler_other_identity").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_other_identity").
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	body := []byte(`{"name":"@renamed","platform_user_id":"fake-rest","registration_ip":"  203.0.113.10  ","password":"  new-password  ","email":" new@example.com ","email_password":"  mail-secret  ","two_factor":"  totp-secret  ","backup_code":"  backup-code  ","email_client_id":"  mail-client  ","email_token":"  mail-token  ","auth_cookie":"  ct0=new; auth_token=new  ","execution_auth":"  encrypted-new-execution-auth  ","account_status":"invalid","task_status":"manual_review","default_proxy_snapshot":"proxy","remark":"  editable note  "}`)
	rec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPut, "/api/v1/accounts/"+formatID(account.ID), body, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
		handler.UpdateMyAccount(c)
	})

	require.Equal(t, http.StatusOK, rec.Code)
	responseBody := rec.Body.String()
	require.Contains(t, responseBody, `"name":"@handler_identity"`)
	require.Contains(t, responseBody, `"platform_user_id":"rest-123"`)
	require.Contains(t, responseBody, `"registration_ip":"203.0.113.10"`)
	require.Contains(t, responseBody, `"password":"  new-password  "`)
	require.Contains(t, responseBody, `"email":"new@example.com"`)
	require.Contains(t, responseBody, `"email_password":"  mail-secret  "`)
	require.Contains(t, responseBody, `"two_factor":"  totp-secret  "`)
	require.Contains(t, responseBody, `"backup_code":"  backup-code  "`)
	require.Contains(t, responseBody, `"email_client_id":"  mail-client  "`)
	require.Contains(t, responseBody, `"email_token":"  mail-token  "`)
	require.Contains(t, responseBody, `"auth_cookie":"  ct0=new; auth_token=new  "`)
	require.Contains(t, responseBody, `"execution_auth":"encrypted-new-execution-auth"`)
	require.Contains(t, responseBody, `"remark":"  editable note  "`)
	require.Contains(t, responseBody, `"account_status":"available"`)
	require.Contains(t, responseBody, `"task_status":"stored"`)
	require.NotContains(t, responseBody, `"default_proxy_snapshot"`)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@handler_identity", stored.Name)
	require.Equal(t, "handler_identity", stored.NameKey)
	require.Equal(t, original.IdentityKind, stored.IdentityKind)
	require.Equal(t, original.IdentityKey, stored.IdentityKey)
	require.NotNil(t, stored.PlatformUserID)
	require.Equal(t, "rest-123", *stored.PlatformUserID)
	require.NotNil(t, stored.RegistrationIP)
	require.Equal(t, "203.0.113.10", *stored.RegistrationIP)
	require.NotNil(t, stored.Password)
	require.Equal(t, "  new-password  ", *stored.Password)
	require.NotNil(t, stored.Email)
	require.Equal(t, "new@example.com", *stored.Email)
	require.NotNil(t, stored.EmailPassword)
	require.Equal(t, "  mail-secret  ", *stored.EmailPassword)
	require.NotNil(t, stored.TwoFactor)
	require.Equal(t, "  totp-secret  ", *stored.TwoFactor)
	require.NotNil(t, stored.BackupCode)
	require.Equal(t, "  backup-code  ", *stored.BackupCode)
	require.NotNil(t, stored.EmailClientID)
	require.Equal(t, "  mail-client  ", *stored.EmailClientID)
	require.NotNil(t, stored.EmailToken)
	require.Equal(t, "  mail-token  ", *stored.EmailToken)
	require.NotNil(t, stored.AuthCookie)
	require.Equal(t, "  ct0=new; auth_token=new  ", *stored.AuthCookie)
	require.NotNil(t, stored.ExecutionAuth)
	require.Equal(t, "encrypted-new-execution-auth", *stored.ExecutionAuth)
	require.NotNil(t, stored.Remark)
	require.Equal(t, "  editable note  ", *stored.Remark)
	require.Equal(t, service.SocialAccountStatusAvailable, stored.AccountStatus)
	require.Equal(t, service.SocialTaskStatusStored, stored.TaskStatus)
	require.Nil(t, stored.DefaultProxySnapshot)

	clearTwoFactorRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPut, "/api/v1/accounts/"+formatID(account.ID), []byte(`{"two_factor":" "}`), func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
		handler.UpdateMyAccount(c)
	})
	require.Equal(t, http.StatusOK, clearTwoFactorRec.Code)
	require.NotContains(t, clearTwoFactorRec.Body.String(), `"two_factor"`)
	require.Nil(t, client.SocialAccount.GetX(ctx, account.ID).TwoFactor)

	invalidExecutionAuthRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPut, "/api/v1/accounts/"+formatID(account.ID), []byte(`{"password":"partially-written-password","execution_auth":"{\"access_token\":\"access\"}"}`), func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
		handler.UpdateMyAccount(c)
	})
	require.Equal(t, http.StatusBadRequest, invalidExecutionAuthRec.Code)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"reason":"SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID"`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"message":"account execution auth is invalid"`)
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "access_token")
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "token_secret")
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "twitter execution auth")
	storedAfterInvalidExecutionAuth := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAfterInvalidExecutionAuth.Password)
	require.Equal(t, "  new-password  ", *storedAfterInvalidExecutionAuth.Password)
	require.NotNil(t, storedAfterInvalidExecutionAuth.ExecutionAuth)
	require.Equal(t, *stored.ExecutionAuth, *storedAfterInvalidExecutionAuth.ExecutionAuth)

	crossUserRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPut, "/api/v1/accounts/"+formatID(otherAccount.ID), []byte(`{"remark":"cross"}`), func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: formatID(otherAccount.ID)}}
		handler.UpdateMyAccount(c)
	})
	requireUserAccountNotFoundResponse(t, crossUserRec, otherAccount.ID, "@handler_other_identity")
	require.Nil(t, client.SocialAccount.GetX(ctx, otherAccount.ID).Remark)
}

func TestAccountWorkbenchHandlerExportMyAccountsIncludesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "safe-export@example.com")
	password := "pool-secret"
	phone := "+15550004444"
	email := "export@example.com"
	emailPassword := "mail-secret"
	twoFactor := "JBSWY3DPEHPK3PXP"
	backupCode := "backup-1"
	emailClientID := "client-id"
	emailToken := "mail-token"
	registrationIP := "198.51.100.44"
	authCookieSecret := "ct0=export; auth_token=export"
	executionAuthSecret := "encrypted-export-execution-auth-ciphertext"
	proxySnapshot := "http://user:pass@proxy.local:8080"
	remark := "delivery export note"

	client.SocialAccount.Create().
		SetName("@safe_export").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("safe_export").
		SetPassword(password).
		SetPhone(phone).
		SetEmail(email).
		SetEmailPassword(emailPassword).
		SetTwoFactor(twoFactor).
		SetBackupCode(backupCode).
		SetEmailClientID(emailClientID).
		SetEmailToken(emailToken).
		SetRegistrationIP(registrationIP).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetDefaultProxySnapshot(proxySnapshot).
		SetRemark(remark).
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	for i := 0; i < 1001; i++ {
		name := "bulk_export_" + strconv.Itoa(i)
		client.SocialAccount.Create().
			SetName("@" + name).
			SetPlatform("x_twitter").
			SetPlatformKey("x_twitter").
			SetNameKey(name).
			SetPassword("bulk-secret").
			SetAssignedUserID(user.ID).
			SetAccountStatus(service.SocialAccountStatusAvailable).
			SetTaskStatus(service.SocialTaskStatusStored).
			SaveX(ctx)
	}
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts/export", nil, handler.ExportMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "@safe_export")
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1003)
	require.GreaterOrEqual(t, len(records), 2)
	header := records[0]
	require.Equal(t, []string{"platform", "username", "name", "platform_user_id", "password", "phone", "email", "email_password", "two_factor", "backup_code", "email_client_id", "email_token", "registration_ip", "auth_cookie", "execution_auth", "default_proxy_snapshot", "account_status", "task_status", "remark", "created_at", "updated_at"}, header)
	var safeRecord []string
	for _, record := range records[1:] {
		if len(record) > 2 && record[2] == "@safe_export" {
			safeRecord = record
			break
		}
	}
	require.NotNil(t, safeRecord)
	require.Len(t, safeRecord, len(header))
	exported := make(map[string]string, len(header))
	for index, name := range header {
		exported[name] = safeRecord[index]
	}
	require.Equal(t, "x_twitter", exported["platform"])
	require.Equal(t, "@safe_export", exported["name"])
	require.Equal(t, password, exported["password"])
	require.Equal(t, phone, exported["phone"])
	require.Equal(t, email, exported["email"])
	require.Equal(t, emailPassword, exported["email_password"])
	require.Equal(t, twoFactor, exported["two_factor"])
	require.Equal(t, backupCode, exported["backup_code"])
	require.Equal(t, emailClientID, exported["email_client_id"])
	require.Equal(t, emailToken, exported["email_token"])
	require.Equal(t, registrationIP, exported["registration_ip"])
	require.Equal(t, authCookieSecret, exported["auth_cookie"])
	require.Equal(t, executionAuthSecret, exported["execution_auth"])
	require.Equal(t, proxySnapshot, exported["default_proxy_snapshot"])
	require.Equal(t, service.SocialAccountStatusAvailable, exported["account_status"])
	require.Equal(t, service.SocialTaskStatusStored, exported["task_status"])
	require.Equal(t, remark, exported["remark"])
	require.Contains(t, body, "@bulk_export_1000")
}

func TestAccountWorkbenchHandlerExportMyAccountsAppliesListFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "filtered-export@example.com")
	otherUser := createSocialHandlerUser(t, ctx, client, "filtered-export-other@example.com")

	create := func(name, platform, accountStatus, taskStatus string, ownerID int64) {
		normalizedName := strings.TrimPrefix(name, "@")
		client.SocialAccount.Create().
			SetName(name).
			SetPlatform(platform).
			SetPlatformKey(platform).
			SetNameKey(normalizedName).
			SetAssignedUserID(ownerID).
			SetAccountStatus(accountStatus).
			SetTaskStatus(taskStatus).
			SaveX(ctx)
	}
	create("@filtered_export_keep", "x_twitter", service.SocialAccountStatusAvailable, service.SocialTaskStatusStored, user.ID)
	create("@filtered_export_wrong_platform", "instagram", service.SocialAccountStatusAvailable, service.SocialTaskStatusStored, user.ID)
	create("@filtered_export_wrong_status", "x_twitter", service.SocialAccountStatusLimited, service.SocialTaskStatusStored, user.ID)
	create("@filtered_export_wrong_task", "x_twitter", service.SocialAccountStatusAvailable, service.SocialTaskStatusPending, user.ID)
	create("@filtered_export_other_owner", "x_twitter", service.SocialAccountStatusAvailable, service.SocialTaskStatusStored, otherUser.ID)

	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts/export?search=filtered_export&platform=x_twitter&account_status=available&task_status=stored", nil, handler.ExportMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	records, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(records), 2)
	header := records[0]
	nameIndex := -1
	for index, name := range header {
		if name == "name" {
			nameIndex = index
			break
		}
	}
	require.NotEqual(t, -1, nameIndex)
	require.Len(t, records, 2)
	require.Equal(t, "@filtered_export_keep", records[1][nameIndex])
	body := rec.Body.String()
	require.NotContains(t, body, "@filtered_export_wrong_platform")
	require.NotContains(t, body, "@filtered_export_wrong_status")
	require.NotContains(t, body, "@filtered_export_wrong_task")
	require.NotContains(t, body, "@filtered_export_other_owner")
}

func TestAccountWorkbenchHandlerExportMyAccountsAppliesSelectedAccountIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "selected-export@example.com")
	otherUser := createSocialHandlerUser(t, ctx, client, "selected-export-other@example.com")

	create := func(name string, ownerID int64) int64 {
		normalizedName := strings.TrimPrefix(name, "@")
		return client.SocialAccount.Create().
			SetName(name).
			SetPlatform("x_twitter").
			SetPlatformKey("x_twitter").
			SetNameKey(normalizedName).
			SetAssignedUserID(ownerID).
			SetAccountStatus(service.SocialAccountStatusAvailable).
			SetTaskStatus(service.SocialTaskStatusStored).
			SaveX(ctx).
			ID
	}
	selectedID := create("@selected_export_keep", user.ID)
	unselectedID := create("@selected_export_skip", user.ID)
	otherID := create("@selected_export_other_owner", otherUser.ID)

	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	target := fmt.Sprintf("/api/v1/accounts/export?account_ids=%d,%d", selectedID, otherID)
	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, target, nil, handler.ExportMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "@selected_export_keep")
	require.NotContains(t, body, "@selected_export_skip")
	require.NotContains(t, body, "@selected_export_other_owner")
	require.NotContains(t, body, strconv.FormatInt(unselectedID, 10))
}

func TestAccountWorkbenchHandlerDeleteMyAccountDeletesOnlyCurrentUserAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "delete-workbench@example.com")
	other := createSocialHandlerUser(t, ctx, client, "delete-workbench-other@example.com")
	account := client.SocialAccount.Create().
		SetName("@delete_workbench").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("delete_workbench").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	taskLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLoginCheck).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetPrice(service.SocialTaskUnitPrice).
		SaveX(ctx)
	ledgerRequestID := "social-task:" + formatID(taskLog.ID) + ":wallet"
	ledger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(ledgerRequestID).
		SetModel("social-action").
		SetTotalCost(service.SocialTaskUnitPrice).
		SetActualCost(service.SocialTaskUnitPrice).
		SaveX(ctx)
	unrelatedLedgerRequestID := "social-task:" + formatID(taskLog.ID+999) + ":wallet"
	unrelatedLedger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(unrelatedLedgerRequestID).
		SetModel("social-action").
		SetTotalCost(service.SocialTaskUnitPrice).
		SetActualCost(service.SocialTaskUnitPrice).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("delete workbench bound proxy").
		SetBoundSocialAccountID(account.ID).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("@delete_workbench_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("delete_workbench_other").
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/"+formatID(account.ID), nil)
	ginCtx.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.DeleteMyAccount(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	var deleteResp map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleteResp))
	require.JSONEq(t, "0", string(deleteResp["code"]))
	if data, ok := deleteResp["data"]; ok {
		require.Equal(t, "null", strings.TrimSpace(string(data)))
	}
	require.NotContains(t, rec.Body.String(), `"deleted"`)
	_, err := client.SocialAccount.Get(ctx, account.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), account.ID)
	require.True(t, dbent.IsNotFound(err), "deleted account must be physically removed")
	taskLogExists, err := client.SocialTaskLog.Query().
		Where(socialtasklog.IDEQ(taskLog.ID), socialtasklog.SocialAccountIDEQ(account.ID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, taskLogExists, "deleted account task logs must be physically removed")
	ledgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(ledger.ID), usagelog.RequestIDEQ(ledgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, ledgerExists, "deleted account usage projection rows must be removed")
	unrelatedLedgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(unrelatedLedger.ID), usagelog.RequestIDEQ(unrelatedLedgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, unrelatedLedgerExists, "unrelated usage projection rows must be retained")
	storedProxy := client.SocialIP.Query().
		Where(socialip.IDEQ(proxy.ID)).
		OnlyX(ctx)
	require.Nil(t, storedProxy.BoundSocialAccountID)
	require.Equal(t, otherAccount.ID, client.SocialAccount.GetX(ctx, otherAccount.ID).ID)

	listRec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts", nil, handler.ListMyAccounts)
	require.Equal(t, http.StatusOK, listRec.Code)
	requirePaginatedIDs(t, listRec.Body.Bytes(), []int64{})

	crossUserRec := httptest.NewRecorder()
	crossUserCtx, _ := gin.CreateTestContext(crossUserRec)
	crossUserCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/"+formatID(otherAccount.ID), nil)
	crossUserCtx.Params = gin.Params{{Key: "id", Value: formatID(otherAccount.ID)}}
	crossUserCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.DeleteMyAccount(crossUserCtx)

	require.Equal(t, http.StatusNotFound, crossUserRec.Code)
	require.Equal(t, otherAccount.ID, client.SocialAccount.GetX(ctx, otherAccount.ID).ID)
	require.NotContains(t, crossUserRec.Body.String(), "SOCIAL_ACCOUNT_NOT_ASSIGNED")
	require.NotContains(t, crossUserRec.Body.String(), formatID(otherAccount.ID))
	require.NotContains(t, crossUserRec.Body.String(), "@delete_workbench_other")
}

func TestAccountWorkbenchHandlerBatchImportAndDeleteSanitizesResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "batch-handler@example.com")
	other := createSocialHandlerUser(t, ctx, client, "batch-handler-other@example.com")
	password := "pool-secret"
	authCookieSecret := "ct0=batch; auth_token=batch"
	executionAuthSecret := "encrypted-batch-execution-auth-ciphertext"
	removed := client.SocialAccount.Create().
		SetName("@handler_removed").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_removed").
		SetPassword(password).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	require.NoError(t, service.NewSocialAccountService(client).DeleteForUser(ctx, user.ID, removed.ID))
	poolDefaultProxySnapshot := `{"id":999,"name":"pool-proxy","endpoint":"http://pool-proxy.example:8080","status":"online"}`
	fresh := client.SocialAccount.Create().
		SetName("@handler_fresh").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_fresh").
		SetPassword(password).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetDefaultProxySnapshot(poolDefaultProxySnapshot).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("@handler_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_other").
		SetPassword(password).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	handler := newEncryptedAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	importBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@handler_removed","password":"typed-secret","two_factor":"JBSWY3DPEHPK3PXP"},{"platform":"x_twitter","name":"@handler_fresh","password":"typed-secret","auth_cookie":"ct0=typed; auth_token=typed"}]}`)
	importRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", importBody, handler.BatchImportMyAccounts)

	require.Equal(t, http.StatusOK, importRec.Code)
	require.Contains(t, importRec.Body.String(), `"total":2`)
	require.Contains(t, importRec.Body.String(), `"succeeded":2`)
	require.Contains(t, importRec.Body.String(), `"imported":2`)
	require.Contains(t, importRec.Body.String(), `"failed":0`)
	require.Contains(t, importRec.Body.String(), `"duplicates":0`)
	require.Contains(t, importRec.Body.String(), `"items":[`)
	require.Contains(t, importRec.Body.String(), `"status":"succeeded"`)
	require.NotContains(t, importRec.Body.String(), `"id":`+formatID(removed.ID))
	require.Contains(t, importRec.Body.String(), `"id":`+formatID(fresh.ID))
	require.Contains(t, importRec.Body.String(), `"password":"pool-secret"`)
	require.Contains(t, importRec.Body.String(), `"auth_cookie":"ct0=batch; auth_token=batch"`)
	require.Contains(t, importRec.Body.String(), `"execution_auth":"encrypted-batch-execution-auth-ciphertext"`)
	require.Contains(t, importRec.Body.String(), `"default_proxy_configured":false`)
	require.NotContains(t, importRec.Body.String(), `"default_proxy_configured":true`)
	require.NotContains(t, importRec.Body.String(), "pool-proxy.example")
	require.Nil(t, client.SocialAccount.GetX(ctx, fresh.ID).DefaultProxySnapshot)
	importedRemoved := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("handler_removed")).
		OnlyX(ctx)
	require.NotEqual(t, removed.ID, importedRemoved.ID)

	deleteBody := []byte(`{"ids":[` + formatID(importedRemoved.ID) + `,` + formatID(fresh.ID) + `,` + formatID(fresh.ID) + `]}`)
	deleteRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-delete", deleteBody, handler.BatchDeleteMyAccounts)

	require.Equal(t, http.StatusOK, deleteRec.Code)
	require.Contains(t, deleteRec.Body.String(), `"total":3`)
	require.Contains(t, deleteRec.Body.String(), `"removed":2`)
	require.Contains(t, deleteRec.Body.String(), `"skipped":1`)
	require.Contains(t, deleteRec.Body.String(), `"reason":"duplicate_in_batch"`)
	_, err := client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), importedRemoved.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), fresh.ID)
	require.True(t, dbent.IsNotFound(err))

	missingImportBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@missing_secret_token","password":"account-secret","phone":"+15550003333","email":"mail@example.com","email_password":"mail-secret","email_client_id":"client-id","email_token":"mail-token","remark":"fresh response note"}]}`)
	missingImportRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", missingImportBody, handler.BatchImportMyAccounts)
	require.Equal(t, http.StatusOK, missingImportRec.Code)
	require.Contains(t, missingImportRec.Body.String(), `"succeeded":1`)
	require.Contains(t, missingImportRec.Body.String(), `"imported":1`)
	require.Contains(t, missingImportRec.Body.String(), `"skipped":0`)
	require.Contains(t, missingImportRec.Body.String(), `"failed":0`)
	require.Contains(t, missingImportRec.Body.String(), `"duplicates":0`)
	require.Contains(t, missingImportRec.Body.String(), `"items":[`)
	require.Contains(t, missingImportRec.Body.String(), `"reason":"staged_not_stored"`)
	require.Contains(t, missingImportRec.Body.String(), `"name":"@missing_secret_token"`)
	require.Contains(t, missingImportRec.Body.String(), `"password":"account-secret"`)
	require.Contains(t, missingImportRec.Body.String(), `"phone":"+15550003333"`)
	require.Contains(t, missingImportRec.Body.String(), `"email":"mail@example.com"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_password":"mail-secret"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_client_id":"client-id"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_token":"mail-token"`)
	require.Contains(t, missingImportRec.Body.String(), `"remark":"fresh response note"`)
	require.NotContains(t, missingImportRec.Body.String(), `"remark":"Email Client ID: client-id`)
	require.Contains(t, missingImportRec.Body.String(), `"account_status":"not_stored"`)
	require.NotContains(t, missingImportRec.Body.String(), "SOCIAL_ACCOUNT_POOL_MATCH_NOT_FOUND")

	whitespaceImportBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@handler_whitespace","password":"  account-secret  ","email":"  whitespace@example.com  ","email_password":"  mail-secret  ","two_factor":"  totp-secret  ","backup_code":"  backup-code  ","email_client_id":"  mail-client  ","email_token":"  mail-token  ","auth_cookie":"  ct0=whitespace; auth_token=whitespace  ","execution_auth":"  encrypted-import-execution-auth  ","registration_ip":"  203.0.113.44  ","remark":"  fresh response note  "}]}`)
	whitespaceImportRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", whitespaceImportBody, handler.BatchImportMyAccounts)
	require.Equal(t, http.StatusOK, whitespaceImportRec.Code)
	require.Contains(t, whitespaceImportRec.Body.String(), `"succeeded":1`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"password":"  account-secret  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"email":"whitespace@example.com"`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"email_password":"  mail-secret  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"two_factor":"  totp-secret  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"backup_code":"  backup-code  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"email_client_id":"  mail-client  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"email_token":"  mail-token  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"auth_cookie":"  ct0=whitespace; auth_token=whitespace  "`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"execution_auth":"encrypted-import-execution-auth"`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"registration_ip":"203.0.113.44"`)
	require.Contains(t, whitespaceImportRec.Body.String(), `"remark":"  fresh response note  "`)

	invalidExecutionAuthBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@handler_invalid_execution_auth","password":"typed-secret","auth_cookie":"ct0=typed; auth_token=typed","execution_auth":"{\"access_token\":\"access\"}"}]}`)
	invalidExecutionAuthRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", invalidExecutionAuthBody, handler.BatchImportMyAccounts)
	require.Equal(t, http.StatusOK, invalidExecutionAuthRec.Code)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"total":1`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"succeeded":0`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"imported":0`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"failed":1`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"reason":"invalid_input"`)
	require.Contains(t, invalidExecutionAuthRec.Body.String(), `"error":"account import data is invalid"`)
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "access_token")
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "token_secret")
	require.NotContains(t, invalidExecutionAuthRec.Body.String(), "twitter execution auth")
	invalidExecutionAuthExists, err := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("handler_invalid_execution_auth")).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, invalidExecutionAuthExists)

	failedDeleteBody := []byte(`{"ids":[` + formatID(otherAccount.ID) + `,0]}`)
	failedDeleteRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-delete", failedDeleteBody, handler.BatchDeleteMyAccounts)
	require.Equal(t, http.StatusOK, failedDeleteRec.Code)
	require.Contains(t, failedDeleteRec.Body.String(), `"skipped":0`)
	require.Contains(t, failedDeleteRec.Body.String(), `"failed":2`)
	require.Contains(t, failedDeleteRec.Body.String(), `"status":"failed"`)
	require.Contains(t, failedDeleteRec.Body.String(), "account could not be deleted")
	require.NotContains(t, failedDeleteRec.Body.String(), "error: code=")
	require.NotContains(t, failedDeleteRec.Body.String(), "SOCIAL_ACCOUNT_NOT_ASSIGNED")
	require.NotContains(t, failedDeleteRec.Body.String(), formatID(otherAccount.ID)+":")
	requireNoDeliveryFieldsInFailedBatchResponse(t, failedDeleteRec.Body.String())
}

func newAccountWorkbenchHandlerForTest(client *dbent.Client, userRepo *socialHandlerBillingUserRepo) *AccountWorkbenchHandler {
	subRepo := &socialHandlerSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewAccountWorkbenchHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		nil,
		service.NewTaskSettingsService(client),
	)
}

func newEncryptedAccountWorkbenchHandlerForTest(client *dbent.Client, userRepo *socialHandlerBillingUserRepo) *AccountWorkbenchHandler {
	subRepo := &socialHandlerSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	return NewAccountWorkbenchHandler(
		service.NewSocialAccountServiceWithCredentialEncryptor(client, handlerExecutionAuthEncryptor{}),
		service.NewSocialIPService(client),
		billing,
		nil,
		service.NewTaskSettingsService(client),
	)
}

type handlerExecutionAuthEncryptor struct{}

func (handlerExecutionAuthEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (handlerExecutionAuthEncryptor) Decrypt(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return "", errors.New("execution auth ciphertext is not encrypted")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func newAccountWorkbenchHandlerForTestWithExecutor(client *dbent.Client, userRepo *socialHandlerBillingUserRepo) *AccountWorkbenchHandler {
	subRepo := &socialHandlerSubscriptionRepo{}
	billing := service.NewSocialBillingService(userRepo, subRepo, nil, nil)
	executor := service.NewSocialTaskExecutor(client, billing, service.SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 10, MinIntervalMs: 1})
	executor.RegisterPlatformExecutor("x_twitter", service.NewTwitterExecutor().WithMediaResolver(service.NewSocialTaskMediaService(client)))
	return NewAccountWorkbenchHandler(
		service.NewSocialAccountService(client),
		service.NewSocialIPService(client),
		billing,
		executor,
		service.NewTaskSettingsService(client),
	)
}

func newAccountWorkbenchHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS social_task_media_assets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	storage_provider TEXT NOT NULL DEFAULT 'inline',
	storage_key TEXT NOT NULL,
	url TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT '',
	file_name TEXT NOT NULL DEFAULT '',
	sha256 TEXT NOT NULL DEFAULT '',
	byte_size INTEGER NOT NULL DEFAULT 0,
	width INTEGER NOT NULL DEFAULT 0,
	height INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(user_id, storage_key)
)`)
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func createSocialHandlerUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed-password").
		SaveX(ctx)
}

func invokeSocialHandlerAsUser(t *testing.T, userID int64, method, path string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	fn(ginCtx)
	return rec
}

func invokeJSONSocialHandlerAsUser(t *testing.T, userID int64, method, path string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	fn(ginCtx)
	return rec
}

func invokeJSONSocialHandlerAsUserWithPathID(t *testing.T, userID int64, method, path, pathID string, body []byte, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	if pathID != "" {
		ginCtx.Params = gin.Params{{Key: "id", Value: pathID}}
	}
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	fn(ginCtx)
	return rec
}

func requireStructuredAccountWorkbenchInputError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "SOCIAL_ACCOUNT_INPUT_REQUIRED")
	require.Contains(t, body, "social account input is required")
	require.NotContains(t, body, "unexpected EOF")
	require.NotContains(t, body, "invalid character")
	require.NotContains(t, body, "cannot unmarshal")
}

func requireAccountWorkbenchServiceUnavailableError(t *testing.T, rec *httptest.ResponseRecorder, reason string) {
	t.Helper()
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, reason)
	require.Contains(t, body, "service is unavailable")
}

func requireSinglePaginatedID(t *testing.T, raw []byte, want int64) {
	t.Helper()
	requirePaginatedIDs(t, raw, []int64{want})
}

func requirePaginatedIDs(t *testing.T, raw []byte, want []int64) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, len(want))
	got := make([]int64, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		got = append(got, item.ID)
	}
	require.ElementsMatch(t, want, got)
}

func requireNoDeliveryFieldsInFailedBatchResponse(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{
		`"password"`,
		`"email_password"`,
		`"auth_cookie"`,
		`"execution_auth"`,
		`"default_proxy_snapshot"`,
		`"assigned_user_id"`,
		",password,",
		",email_password,",
		",auth_cookie,",
		",execution_auth,",
		",default_proxy_snapshot,",
		",assigned_user_id,",
		"pool-secret",
		"mail-secret",
		"ct0=batch; auth_token=batch",
		"encrypted-batch-execution-auth-ciphertext",
		"http://user:pass@proxy.local:8080",
	} {
		require.NotContains(t, body, forbidden)
	}
}

func requireUserAccountNotFoundResponse(t *testing.T, rec *httptest.ResponseRecorder, accountID int64, accountName string) {
	t.Helper()
	require.Equal(t, http.StatusNotFound, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "social account not found")
	require.NotContains(t, body, "SOCIAL_ACCOUNT_NOT_ASSIGNED")
	require.NotContains(t, body, formatID(accountID))
	require.NotContains(t, body, accountName)
}

func saveHandlerTaskTemplate(t *testing.T, ctx context.Context, client *dbent.Client, userID int64, taskType string, params service.TaskTemplateParams) *service.TaskTemplate {
	t.Helper()
	tmpl, err := service.NewTaskSettingsService(client).SaveTemplate(ctx, userID, &service.TaskTemplateInput{
		Name:      "handler task template",
		Type:      taskType,
		Params:    params,
		IsDefault: true,
	})
	require.NoError(t, err)
	return tmpl
}

func assignHandlerDefaultProxy(t *testing.T, ctx context.Context, client *dbent.Client, userID, accountID int64, name string) {
	t.Helper()
	endpoint := "http://8.8.8.8:8080"
	ipSvc := service.NewSocialIPService(client)
	ip, err := ipSvc.Create(ctx, &service.CreateSocialIPInput{
		UserID:   userID,
		Name:     name,
		IPType:   service.SocialIPTypeResidential,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(service.SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	_, err = service.NewSocialAccountService(client).SetDefaultProxyForUser(ctx, accountID, userID, ip)
	require.NoError(t, err)
}

func inlinePNGDataURLForHandlerTemplateValidation(t *testing.T, width, height int) string {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	require.NoError(t, png.Encode(&buf, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

type socialHandlerBillingUserRepo struct {
	service.UserRepository
	user        *service.User
	deductCalls int
}

func (r *socialHandlerBillingUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, service.ErrUserNotFound
	}
	out := *r.user
	return &out, nil
}

func (r *socialHandlerBillingUserRepo) DeductBalance(_ context.Context, id int64, amount float64) error {
	if r.user == nil || r.user.ID != id {
		return service.ErrUserNotFound
	}
	r.deductCalls++
	r.user.Balance -= amount
	return nil
}

type socialHandlerSubscriptionRepo struct {
	service.UserSubscriptionRepository
	subs []service.UserSubscription
}

func (r *socialHandlerSubscriptionRepo) ListActiveByUserID(_ context.Context, userID int64) ([]service.UserSubscription, error) {
	out := make([]service.UserSubscription, 0, len(r.subs))
	for _, sub := range r.subs {
		if sub.UserID == userID && sub.Status == service.SubscriptionStatusActive {
			out = append(out, sub)
		}
	}
	return out, nil
}

func (r *socialHandlerSubscriptionRepo) IncrementUsage(_ context.Context, _ int64, _ float64) error {
	return nil
}
