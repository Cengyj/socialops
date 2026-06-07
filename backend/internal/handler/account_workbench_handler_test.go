//go:build unit

package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
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
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`",
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
	require.Contains(t, rec.Body.String(), "暂不可用")
	require.NotContains(t, rec.Body.String(), "not configured")
	require.NotContains(t, rec.Body.String(), "executor")
	require.NotContains(t, rec.Body.String(), "executor queue")
	require.NotContains(t, rec.Body.String(), "queue is")
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
	require.Contains(t, *logs[0].ResultMessage, "not configured")
	require.NotNil(t, logs[0].IdempotencyKey)
	require.Equal(t, "g008-submit-1", *logs[0].IdempotencyKey)
	require.Nil(t, logs[0].BillingRequestID)
	require.NotNil(t, logs[0].ProxySnapshot)
	require.Contains(t, *logs[0].ProxySnapshot, proxyEndpoint)
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
			want:    "该平台动作暂不可用，本次未扣费",
			forbidden: []string{
				"executor",
				"not configured",
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
			message: "unsupported action message on x_twitter",
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
			message: "video media is not implemented yet",
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
			name:    "unknown internal detail",
			message: `upstream response body {"error":"secret","headers":{"authorization":"Bearer abc"}} trace_id=trace-123 request_id=req-456`,
			want:    "任务执行失败，本次未扣费",
			forbidden: []string{
				"response body",
				"authorization",
				"Bearer abc",
				"trace-123",
				"req-456",
			},
		},
		{
			name:    "success-looking sensitive upstream detail",
			message: `follow succeeded https://upstream.example/callback?trace_id=trace-123 authorization=Bearer abc`,
			want:    "任务已完成，详细结果已隐藏",
			forbidden: []string{
				"https://upstream.example",
				"trace-123",
				"authorization",
				"Bearer abc",
			},
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
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`",
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
	require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_REQUIRED")
	require.Zero(t, userRepo.deductCalls)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestAccountWorkbenchHandlerSubmitTaskAcceptsTemplateOnlyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "submit-template-only@example.com")
	account := client.SocialAccount.Create().
		SetName("@submit_template_only").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("submit_template_only").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	assignHandlerDefaultProxy(t, ctx, client, user.ID, account.ID, "submit template only proxy")
	tmpl, err := service.NewTaskSettingsService(client).SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "template-only login check",
		Type: service.SocialTaskActionLoginCheck,
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"template_id": "`+tmpl.ID+`",
		"client_request_id": "template-only-submit"
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
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(first.ID)+`, `+formatID(second.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`",
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
			QuotePostURL: "https://x.com/openai/status/1",
			Media: []service.SocialTaskMediaRef{
				{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "handler-post.png",
					URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 640, 640),
				},
			},
		},
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"template_id": "`+tmpl.ID+`",
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
	require.Equal(t, "https://x.com/openai/status/1", body.Data.Logs[0].Payload.Post.QuotePostURL)
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
	require.Equal(t, "https://x.com/openai/status/1", body.Data.Logs[0].TemplateSnapshot.Params.QuotePostURL)
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
	require.Equal(t, "https://x.com/openai/status/1", logs[0].Payload.Post.QuotePostURL)
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
	require.Equal(t, "https://x.com/openai/status/1", logs[0].TemplateSnapshot.Params.QuotePostURL)
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
	})
	require.NoError(t, err)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTest(client, userRepo)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"template_id": "`+tmpl.ID+`",
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
				Name:   tc.templateName,
				Type:   tc.taskType,
				Params: tc.params,
			})
			require.NoError(t, err)
			userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
			handler := newAccountWorkbenchHandlerForTest(client, userRepo)

			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
				"account_ids": [`+formatID(account.ID)+`],
				"template_id": "`+tmpl.ID+`",
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(xAccount.ID)+`, `+formatID(instagramAccount.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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

func TestAccountWorkbenchHandlerRejectsUnavailableMessageActionBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "message-unavailable@example.com")
	account := client.SocialAccount.Create().
		SetName("@message_unavailable").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("message_unavailable").
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}}
	handler := newAccountWorkbenchHandlerForTestWithExecutor(client, userRepo)
	messageTemplateID := "malicious_message_template"
	now := time.Now().UTC().Format(time.RFC3339)
	client.Setting.Create().
		SetKey("socialops:task_settings:user:" + formatID(user.ID)).
		SetValue(`{"templates":[{"id":"` + messageTemplateID + `","name":"message","type":"message","params":{"targets":["@target"],"contents":["hello"]},"is_default":false,"created_at":"` + now + `","updated_at":"` + now + `"}]}`).
		SaveX(ctx)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+messageTemplateID+`",
		"client_request_id": "message-unavailable"
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
		SetValue(`{"templates":[{"id":"` + templateID + `","name":"oversized","type":"post","params":{"contents":["` + strings.Repeat("a", 2049) + `"]},"is_default":false,"created_at":"` + now + `","updated_at":"` + now + `"}]}`).
		SaveX(ctx)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"template_id": "`+templateID+`",
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

func TestAccountWorkbenchHandlerRejectsInvalidStoredProfileMediaTemplatesBeforeBilling(t *testing.T) {
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
					IsDefault: false,
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
				"template_id": "`+tc.templateID+`",
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
			expectedMessage: "video media is not implemented yet",
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
			expectedMessage: "post media #1 media source is not supported yet",
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
					IsDefault: false,
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
				"template_id": "`+tc.templateID+`",
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
			expectedMessage: "avatar media source is not supported yet",
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
			expectedMessage: "banner media source is not supported yet",
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
					IsDefault: false,
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
				"template_id": "`+tc.templateID+`",
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`, `+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [-1, `+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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
	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", bytes.NewBufferString(`{
		"account_ids": [`+formatID(account.ID)+`],
		"action": "login_check",
		"template_id": "`+tmpl.ID+`"
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
	proxySnapshot := "http://user:pass@proxy.local:8080"
	accountID := "pool-account-id"
	password := "pool-secret"
	phone := "+15550000001"
	email := "safe-list@example.com"
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=list; auth_token=list"
	executionAuthSecret := `{"access_token":"list","token_secret":"secret"}`

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
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

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
	require.Contains(t, body, `"execution_auth":"{\"access_token\":\"list\",\"token_secret\":\"secret\"}"`)
	require.Contains(t, body, `"default_proxy_snapshot":"http://user:pass@proxy.local:8080"`)
	require.Contains(t, body, `"default_proxy_configured":true`)
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
		SetExecutionAuth("sensitive-cookie").
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 1.0}})

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
	require.Contains(t, body, `"execution_auth":"sensitive-cookie"`)
}

func TestAccountWorkbenchHandlerSetDefaultProxyIncludesDeliveryFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "safe-proxy@example.com")
	password := "pool-secret"
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=proxy; auth_token=proxy"
	executionAuthSecret := "execution-secret"
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
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

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
	require.Contains(t, body, `"execution_auth":"execution-secret"`)
	require.Contains(t, body, endpoint)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Contains(t, *stored.DefaultProxySnapshot, endpoint)
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

	tmpl := saveHandlerTaskTemplate(t, ctx, client, user.ID, service.SocialTaskActionFollow, service.TaskTemplateParams{Targets: []string{"@target"}})
	taskBody := []byte(`{"account_ids":[` + formatID(otherAccount.ID) + `],"action":"login_check","template_id":"` + tmpl.ID + `"}`)
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

	require.Nil(t, client.SocialAccount.GetX(ctx, otherAccount.ID).UserWorkbenchDeletedAt)
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
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	body := []byte(`{"name":"@renamed","platform_user_id":"fake-rest","registration_ip":"203.0.113.10","password":"new-password","email":" new@example.com ","two_factor":"","auth_cookie":"ct0=new; auth_token=new","account_status":"invalid","task_status":"manual_review","default_proxy_snapshot":"proxy","remark":"editable note"}`)
	rec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPut, "/api/v1/accounts/"+formatID(account.ID), body, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: formatID(account.ID)}}
		handler.UpdateMyAccount(c)
	})

	require.Equal(t, http.StatusOK, rec.Code)
	responseBody := rec.Body.String()
	require.Contains(t, responseBody, `"name":"@handler_identity"`)
	require.Contains(t, responseBody, `"platform_user_id":"rest-123"`)
	require.Contains(t, responseBody, `"registration_ip":"198.51.100.20"`)
	require.Contains(t, responseBody, `"password":"new-password"`)
	require.Contains(t, responseBody, `"email":"new@example.com"`)
	require.NotContains(t, responseBody, `"two_factor"`)
	require.Contains(t, responseBody, `"auth_cookie":"ct0=new; auth_token=new"`)
	require.Contains(t, responseBody, `"remark":"editable note"`)
	require.Contains(t, responseBody, `"account_status":"available"`)
	require.Contains(t, responseBody, `"task_status":"stored"`)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@handler_identity", stored.Name)
	require.Equal(t, "handler_identity", stored.NameKey)
	require.Equal(t, original.IdentityKind, stored.IdentityKind)
	require.Equal(t, original.IdentityKey, stored.IdentityKey)
	require.NotNil(t, stored.PlatformUserID)
	require.Equal(t, "rest-123", *stored.PlatformUserID)
	require.NotNil(t, stored.RegistrationIP)
	require.Equal(t, "198.51.100.20", *stored.RegistrationIP)
	require.Equal(t, service.SocialAccountStatusAvailable, stored.AccountStatus)
	require.Equal(t, service.SocialTaskStatusStored, stored.TaskStatus)

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
	emailPassword := "mail-secret"
	authCookieSecret := "ct0=export; auth_token=export"
	executionAuthSecret := "execution-secret"
	proxySnapshot := "http://user:pass@proxy.local:8080"

	client.SocialAccount.Create().
		SetName("@safe_export").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("safe_export").
		SetPassword(password).
		SetEmailPassword(emailPassword).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetDefaultProxySnapshot(proxySnapshot).
		SetAssignedUserID(user.ID).
		SetAccountStatus(service.SocialAccountStatusAvailable).
		SetTaskStatus(service.SocialTaskStatusStored).
		SaveX(ctx)
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	rec := invokeSocialHandlerAsUser(t, user.ID, http.MethodGet, "/api/v1/accounts/export", nil, handler.ExportMyAccounts)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "@safe_export")
	require.Contains(t, body, "account_status")
	require.Contains(t, body, "password")
	require.Contains(t, body, "email_password")
	require.Contains(t, body, "auth_cookie")
	require.Contains(t, body, "execution_auth")
	require.Contains(t, body, "default_proxy_snapshot")
	require.Contains(t, body, password)
	require.Contains(t, body, emailPassword)
	require.Contains(t, body, authCookieSecret)
	require.Contains(t, body, executionAuthSecret)
	require.Contains(t, body, proxySnapshot)
}

func TestAccountWorkbenchHandlerDeleteMyAccountHidesOnlyCurrentUserAccount(t *testing.T) {
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
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.DeletedAt)
	require.NotNil(t, stored.AssignedUserID)
	require.Equal(t, user.ID, int64(*stored.AssignedUserID))
	require.NotNil(t, stored.UserWorkbenchDeletedAt)
	require.Nil(t, client.SocialAccount.GetX(ctx, otherAccount.ID).UserWorkbenchDeletedAt)

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
	require.Nil(t, client.SocialAccount.GetX(ctx, otherAccount.ID).UserWorkbenchDeletedAt)
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
	executionAuthSecret := "execution-secret"
	hidden := client.SocialAccount.Create().
		SetName("@handler_hidden").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_hidden").
		SetPassword(password).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
		SetAssignedUserID(user.ID).
		SetUserWorkbenchDeletedAt(time.Now()).
		SaveX(ctx)
	fresh := client.SocialAccount.Create().
		SetName("@handler_fresh").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("handler_fresh").
		SetPassword(password).
		SetAuthCookie(authCookieSecret).
		SetExecutionAuth(executionAuthSecret).
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
	handler := newAccountWorkbenchHandlerForTest(client, &socialHandlerBillingUserRepo{user: &service.User{ID: user.ID, Balance: 0}})

	importBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@handler_hidden","password":"typed-secret","two_factor":"JBSWY3DPEHPK3PXP"},{"platform":"x_twitter","name":"@handler_fresh","password":"typed-secret","auth_cookie":"ct0=typed; auth_token=typed"}]}`)
	importRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", importBody, handler.BatchImportMyAccounts)

	require.Equal(t, http.StatusOK, importRec.Code)
	require.Contains(t, importRec.Body.String(), `"total":2`)
	require.Contains(t, importRec.Body.String(), `"succeeded":2`)
	require.Contains(t, importRec.Body.String(), `"imported":2`)
	require.Contains(t, importRec.Body.String(), `"failed":0`)
	require.Contains(t, importRec.Body.String(), `"duplicates":0`)
	require.Contains(t, importRec.Body.String(), `"items":[`)
	require.Contains(t, importRec.Body.String(), `"status":"succeeded"`)
	require.Contains(t, importRec.Body.String(), `"id":`+formatID(hidden.ID))
	require.Contains(t, importRec.Body.String(), `"id":`+formatID(fresh.ID))
	require.Contains(t, importRec.Body.String(), `"password":"pool-secret"`)
	require.Contains(t, importRec.Body.String(), `"auth_cookie":"ct0=batch; auth_token=batch"`)
	require.Contains(t, importRec.Body.String(), `"execution_auth":"execution-secret"`)

	deleteBody := []byte(`{"ids":[` + formatID(hidden.ID) + `,` + formatID(fresh.ID) + `]}`)
	deleteRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-delete", deleteBody, handler.BatchDeleteMyAccounts)

	require.Equal(t, http.StatusOK, deleteRec.Code)
	require.Contains(t, deleteRec.Body.String(), `"removed":2`)
	require.NotNil(t, client.SocialAccount.GetX(ctx, hidden.ID).UserWorkbenchDeletedAt)
	require.NotNil(t, client.SocialAccount.GetX(ctx, fresh.ID).UserWorkbenchDeletedAt)

	missingImportBody := []byte(`{"accounts":[{"platform":"x_twitter","name":"@missing_secret_token","password":"account-secret","email":"mail@example.com","email_password":"mail-secret","email_client_id":"client-id","email_token":"mail-token"}]}`)
	missingImportRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-import", missingImportBody, handler.BatchImportMyAccounts)
	require.Equal(t, http.StatusOK, missingImportRec.Code)
	require.Contains(t, missingImportRec.Body.String(), `"succeeded":1`)
	require.Contains(t, missingImportRec.Body.String(), `"imported":1`)
	require.Contains(t, missingImportRec.Body.String(), `"skipped":0`)
	require.Contains(t, missingImportRec.Body.String(), `"failed":0`)
	require.Contains(t, missingImportRec.Body.String(), `"duplicates":0`)
	require.Contains(t, missingImportRec.Body.String(), `"items":[`)
	require.Contains(t, missingImportRec.Body.String(), `"name":"@missing_secret_token"`)
	require.Contains(t, missingImportRec.Body.String(), `"password":"account-secret"`)
	require.Contains(t, missingImportRec.Body.String(), `"email":"mail@example.com"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_password":"mail-secret"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_client_id":"client-id"`)
	require.Contains(t, missingImportRec.Body.String(), `"email_token":"mail-token"`)
	require.NotContains(t, missingImportRec.Body.String(), `"remark":"Email Client ID: client-id`)
	require.Contains(t, missingImportRec.Body.String(), `"account_status":"not_stored"`)
	require.NotContains(t, missingImportRec.Body.String(), "SOCIAL_ACCOUNT_POOL_MATCH_NOT_FOUND")

	failedDeleteBody := []byte(`{"ids":[` + formatID(otherAccount.ID) + `,0]}`)
	failedDeleteRec := invokeJSONSocialHandlerAsUser(t, user.ID, http.MethodPost, "/api/v1/accounts/batch-delete", failedDeleteBody, handler.BatchDeleteMyAccounts)
	require.Equal(t, http.StatusOK, failedDeleteRec.Code)
	require.Contains(t, failedDeleteRec.Body.String(), `"skipped":2`)
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
		"execution-secret",
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
		Name:   "handler task template",
		Type:   taskType,
		Params: params,
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
