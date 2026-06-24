//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTaskSettingsHandlerValidateTemplateRequiresAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task-settings/templates/validate", strings.NewReader(`{
		"name": "follow targets",
		"type": "follow",
		"params": {"targets":["@target"]}
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.ValidateTemplate(ginCtx)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "unauthorized")
}

func TestTaskSettingsHandlerTemplateRoutesRequireAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		call   func(*gin.Context)
	}{
		{
			name:   "list templates",
			method: http.MethodGet,
			target: "/api/v1/task-settings/templates",
			call:   handler.ListTemplates,
		},
		{
			name:   "save template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates",
			body:   `{}`,
			call:   handler.SaveTemplate,
		},
		{
			name:   "delete template",
			method: http.MethodDelete,
			target: "/api/v1/task-settings/templates/tmpl",
			call:   handler.DeleteTemplate,
		},
		{
			name:   "copy template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/tmpl/copy",
			call:   handler.CopyTemplate,
		},
		{
			name:   "set default template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/tmpl/default",
			call:   handler.SetDefaultTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.body != "" {
				ginCtx.Request.Header.Set("Content-Type", "application/json")
			}

			tt.call(ginCtx)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.Contains(t, rec.Body.String(), "unauthorized")
		})
	}
}

func TestTaskSettingsHandlerTemplateInputBindingErrorsAreStructured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		call   func(*gin.Context)
	}{
		{
			name:   "save malformed json",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates",
			body:   `{"name":`,
			call:   handler.SaveTemplate,
		},
		{
			name:   "save wrong field type",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates",
			body:   `{"name":123,"type":"follow","params":{"targets":["@target"]}}`,
			call:   handler.SaveTemplate,
		},
		{
			name:   "validate malformed json",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/validate",
			body:   `{"name":`,
			call:   handler.ValidateTemplate,
		},
		{
			name:   "validate wrong field type",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/validate",
			body:   `{"name":123,"type":"follow","params":{"targets":["@target"]}}`,
			call:   handler.ValidateTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			ginCtx.Request.Header.Set("Content-Type", "application/json")
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

			tt.call(ginCtx)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_INPUT_REQUIRED")
			require.Contains(t, rec.Body.String(), "task template input is required")
			require.NotContains(t, rec.Body.String(), "unexpected EOF")
			require.NotContains(t, rec.Body.String(), "invalid character")
			require.NotContains(t, rec.Body.String(), "cannot unmarshal")
		})
	}
}

func TestTaskSettingsHandlerTemplateRoutesReportServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		param  string
		call   func(*gin.Context)
	}{
		{
			name:   "list templates",
			method: http.MethodGet,
			target: "/api/v1/task-settings/templates",
			call:   handler.ListTemplates,
		},
		{
			name:   "save template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates",
			body:   `{"name":"follow targets","type":"follow","params":{"targets":["@target"]}}`,
			call:   handler.SaveTemplate,
		},
		{
			name:   "validate template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/validate",
			body:   `{"name":"follow targets","type":"follow","params":{"targets":["@target"]}}`,
			call:   handler.ValidateTemplate,
		},
		{
			name:   "delete template",
			method: http.MethodDelete,
			target: "/api/v1/task-settings/templates/tmpl_1",
			param:  "tmpl_1",
			call:   handler.DeleteTemplate,
		},
		{
			name:   "copy template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/tmpl_1/copy",
			param:  "tmpl_1",
			call:   handler.CopyTemplate,
		},
		{
			name:   "set default template",
			method: http.MethodPost,
			target: "/api/v1/task-settings/templates/tmpl_1/default",
			param:  "tmpl_1",
			call:   handler.SetDefaultTemplate,
		},
		{
			name:   "preview template media",
			method: http.MethodGet,
			target: "/api/v1/task-settings/media?storage_key=social-task%2F42%2Favatar.png",
			call:   handler.PreviewTemplateMedia,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.body != "" {
				ginCtx.Request.Header.Set("Content-Type", "application/json")
			}
			if tt.param != "" {
				ginCtx.Params = gin.Params{{Key: "id", Value: tt.param}}
			}
			ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

			tt.call(ginCtx)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			require.Contains(t, rec.Body.String(), "TASK_TEMPLATE_SERVICE_UNAVAILABLE")
			require.Contains(t, rec.Body.String(), "task template service is unavailable")
		})
	}
}

func TestTaskSettingsHandlerPreviewTemplateMediaRequiresAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/task-settings/media?storage_key=social-task%2F42%2Favatar.png", nil)

	handler.PreviewTemplateMedia(ginCtx)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "unauthorized")
}

func TestTaskSettingsHandlerPreviewTemplateMediaStreamsOwnedTaskAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAccountWorkbenchHandlerTestClient(t)
	user := createSocialHandlerUser(t, ctx, client, "task-media-preview@example.com")
	svc := service.NewTaskSettingsService(client)

	saved, err := svc.SaveTemplate(ctx, user.ID, &service.TaskTemplateInput{
		Name: "Preview avatar",
		Type: service.SocialTaskActionUpdateAvatar,
		Params: service.TaskTemplateParams{
			Avatar: &service.SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "preview-avatar.png",
				URL:         inlinePNGDataURLForHandlerTemplateValidation(t, 400, 400),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, saved.Params.Avatar)

	handler := NewTaskSettingsHandler(svc)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})
	ginCtx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/task-settings/media?storage_key="+url.QueryEscape(saved.Params.Avatar.StorageKey),
		nil,
	)

	handler.PreviewTemplateMedia(ginCtx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), "preview-avatar.png")
	require.NotEmpty(t, rec.Body.Bytes())
}

func TestInlineMediaContentDispositionSanitizesPreviewFileName(t *testing.T) {
	disposition := inlineMediaContentDisposition("..\\private/evil\"\r\nX-Storage-Key: social-task/42/avatar.png\r\n.png")

	require.Contains(t, disposition, "inline")
	require.Contains(t, disposition, "filename=")
	require.NotContains(t, disposition, "\r")
	require.NotContains(t, disposition, "\n")
	require.NotContains(t, disposition, "X-Storage-Key")
	require.NotContains(t, disposition, "social-task/42")
	require.NotContains(t, disposition, "..")
	require.NotContains(t, disposition, "\\")
}
