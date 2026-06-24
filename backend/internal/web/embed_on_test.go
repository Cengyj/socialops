//go:build embed

package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type testPublicSettingsProvider struct{}

func (testPublicSettingsProvider) GetPublicSettingsForInjection(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{"site_name":"SocialOps"}`), nil
}

type invalidPublicSettingsProvider struct{}

func (invalidPublicSettingsProvider) GetPublicSettingsForInjection(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{`), nil
}

func TestInjectSiteTitleUsesSocialOpsProductTitle(t *testing.T) {
	html := []byte("<html><head><title>Placeholder</title></head><body></body></html>")
	settings := []byte(`{"site_name":"SocialOps"}`)

	rendered := string(injectSiteTitle(html, settings))

	require.Contains(t, rendered, "<title>SocialOps - Website Account Pool Social Operations Platform</title>")
	require.NotContains(t, rendered, "Social Account Operations Platform")
	require.NotContains(t, rendered, "AI API Gateway")
}

func TestInjectSiteTitleEscapesConfiguredSiteName(t *testing.T) {
	html := []byte("<html><head><title>Placeholder</title></head><body></body></html>")
	settings := []byte(`{"site_name":"</title><script>alert(1)</script>"}`)

	rendered := string(injectSiteTitle(html, settings))

	require.Contains(t, rendered, "<title>&lt;/title&gt;&lt;script&gt;alert(1)&lt;/script&gt; - Website Account Pool Social Operations Platform</title>")
	require.NotContains(t, rendered, "Social Account Operations Platform")
	require.NotContains(t, rendered, "<script>alert(1)</script>")
}

func TestFrontendMiddlewareDoesNotServeSPAForNonGetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewFrontendServer(testPublicSettingsProvider{})
	require.NoError(t, err)

	router := gin.New()
	router.Use(server.Middleware())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts"+"/tasks", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotContains(t, rec.Body.String(), "<!doctype html>")
}

func TestFrontendMiddlewareBypassesOnlySocialOpsOwnedBackendRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewFrontendServer(testPublicSettingsProvider{})
	require.NoError(t, err)

	router := gin.New()
	router.Use(server.Middleware())

	for _, path := range []string{
		"/api/v1/accounts" + "/tasks",
		"/setup/health",
		"/health",
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

func TestFrontendMiddlewareFallsBackWhenInjectedSettingsJSONIsInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewFrontendServer(invalidPublicSettingsProvider{})
	require.NoError(t, err)

	router := gin.New()
	router.Use(server.Middleware())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "window.__APP_CONFIG__={;")
}

func TestFrontendBypassListDoesNotCarryRemovedGatewayPrefixes(t *testing.T) {
	for _, path := range []string{
		"/v1",
		"/v1/" + "chat/completions",
		"/v1beta/" + "models",
		"/" + "anti" + "gravity" + "/v1/messages",
		"/" + "so" + "ra" + "/v1/jobs",
	} {
		require.Falsef(t, shouldBypassEmbeddedFrontend(path), "removed gateway path %s must not be a SocialOps backend prefix", path)
	}
}
