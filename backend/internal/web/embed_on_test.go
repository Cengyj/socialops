//go:build embed

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type testPublicSettingsProvider struct{}

func (testPublicSettingsProvider) GetPublicSettingsForInjection(context.Context) (any, error) {
	return map[string]string{"site_name": "SocialOps"}, nil
}

func TestInjectSiteTitleUsesSocialOpsProductTitle(t *testing.T) {
	html := []byte("<html><head><title>Placeholder</title></head><body></body></html>")
	settings := []byte(`{"site_name":"SocialOps"}`)

	rendered := string(injectSiteTitle(html, settings))

	require.Contains(t, rendered, "<title>SocialOps - Social Account Operations Platform</title>")
	require.NotContains(t, rendered, "AI API Gateway")
}

func TestFrontendMiddlewareDoesNotServeSPAForNonGetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewFrontendServer(testPublicSettingsProvider{})
	require.NoError(t, err)

	router := gin.New()
	router.Use(server.Middleware())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-accounts/tasks", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotContains(t, rec.Body.String(), "<!doctype html>")
}

func TestFrontendMiddlewareDoesNotServeSPAForRemovedAIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewFrontendServer(testPublicSettingsProvider{})
	require.NoError(t, err)

	router := gin.New()
	router.Use(server.Middleware())

	for _, path := range []string{
		"/v1/chat/completions",
		"/v1beta/models",
		"/antigravity/v1/messages",
		"/sora/v1/jobs",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusNotFound, rec.Code)
			require.NotContains(t, rec.Body.String(), "<!doctype html>")
		})
	}
}
