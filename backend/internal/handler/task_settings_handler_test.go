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
