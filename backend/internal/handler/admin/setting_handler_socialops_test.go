//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminSettingHandler_GetSettings_DoesNotExposeAIGatewaySettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		"enable_model_fallback":                    "true",
		"ops_monitoring_enabled":                   "true",
		"ops_realtime_monitoring_enabled":          "true",
		"channel_monitor_enabled":                  "true",
		"channel_monitor_default_interval_seconds": "60",
		"available_channels_enabled":               "true",
		"enable_identity_patch":                    "true",
		"identity_patch_prompt":                    "legacy AI prompt",
		"enable_fingerprint_unification":           "true",
		"enable_metadata_passthrough":              "true",
		"enable_cch_signing":                       "true",
		"rewrite_message_cache_control":            "true",
		"web_search_emulation_config":              `{"enabled":true,"providers":[{"type":"search"}]}`,
		"allow_ungrouped_key_scheduling":           "true",
	}}
	handler := NewSettingHandler(
		service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)

	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	forbidden := []string{
		"enable_model_fallback",
		"ops_monitoring_enabled",
		"ops_realtime_monitoring_enabled",
		"ops_query_mode_default",
		"ops_metrics_interval_seconds",
		"channel_monitor_enabled",
		"channel_monitor_default_interval_seconds",
		"available_channels_enabled",
		"enable_identity_patch",
		"identity_patch_prompt",
		"enable_fingerprint_unification",
		"enable_metadata_passthrough",
		"enable_cch_signing",
		"rewrite_message_cache_control",
		"web_search_emulation_enabled",
		"allow_ungrouped_key_scheduling",
	}
	for _, key := range forbidden {
		require.NotContainsf(t, resp.Data, key, "admin settings must not expose AI gateway setting %s", key)
	}
}

func TestAdminSettingHandler_UpdateSettings_IgnoresAIGatewaySettingsAndPreservesPlatformSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		"enable_identity_patch":                   "true",
		"identity_patch_prompt":                   "legacy AI prompt",
		"allow_ungrouped_key_scheduling":          "true",
		service.SettingKeyEmailVerifyEnabled:      "true",
		service.SettingKeyPasswordResetEnabled:    "true",
		service.SettingKeyTotpEnabled:             "true",
		service.SettingKeySMTPHost:                "smtp.example.com",
		service.SettingKeySMTPPort:                "465",
		service.SettingKeySMTPUsername:            "mailer",
		service.SettingKeySMTPPassword:            "smtp-secret",
		service.SettingKeySMTPFrom:                "noreply@example.com",
		service.SettingKeySMTPFromName:            "SocialOps Mail",
		service.SettingKeySMTPUseTLS:              "true",
		service.SettingKeyGitHubOAuthEnabled:      "true",
		service.SettingKeyGitHubOAuthClientID:     "github-client",
		service.SettingKeyGitHubOAuthClientSecret: "github-secret",
	}}
	handler := NewSettingHandler(
		service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader([]byte(`{
		"site_name": "SocialOps",
		"default_concurrency": 5
	}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	for _, key := range []string{
		"enable_identity_patch",
		"identity_patch_prompt",
		"allow_ungrouped_key_scheduling",
	} {
		require.NotContainsf(t, repo.lastUpdates, key, "AI gateway setting %s must not be rewritten by SocialOps settings update", key)
	}
	require.Equal(t, "true", repo.values["enable_identity_patch"])
	require.Equal(t, "legacy AI prompt", repo.values["identity_patch_prompt"])
	require.Equal(t, "true", repo.values["allow_ungrouped_key_scheduling"])
	require.Equal(t, "true", repo.values[service.SettingKeyEmailVerifyEnabled])
	require.Equal(t, "true", repo.values[service.SettingKeyPasswordResetEnabled])
	require.Equal(t, "true", repo.values[service.SettingKeyTotpEnabled])
	require.Equal(t, "smtp.example.com", repo.values[service.SettingKeySMTPHost])
	require.Equal(t, "465", repo.values[service.SettingKeySMTPPort])
	require.Equal(t, "mailer", repo.values[service.SettingKeySMTPUsername])
	require.Equal(t, "smtp-secret", repo.values[service.SettingKeySMTPPassword])
	require.Equal(t, "noreply@example.com", repo.values[service.SettingKeySMTPFrom])
	require.Equal(t, "SocialOps Mail", repo.values[service.SettingKeySMTPFromName])
	require.Equal(t, "true", repo.values[service.SettingKeySMTPUseTLS])
	require.Equal(t, "true", repo.values[service.SettingKeyGitHubOAuthEnabled])
	require.Equal(t, "github-client", repo.values[service.SettingKeyGitHubOAuthClientID])
	require.Equal(t, "github-secret", repo.values[service.SettingKeyGitHubOAuthClientSecret])
}
