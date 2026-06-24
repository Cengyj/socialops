package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadForBootstrapReadsUpdateProxyURLFromEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	const proxyURL = "socks5://127.0.0.1:1080"
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))
	t.Setenv("UPDATE_PROXY_URL", proxyURL)

	cfg, err := LoadForBootstrap()
	if err != nil {
		t.Fatalf("LoadForBootstrap() error: %v", err)
	}
	if cfg.Update.ProxyURL != proxyURL {
		t.Fatalf("update.proxy_url from env = %q, want %q", cfg.Update.ProxyURL, proxyURL)
	}
}

func TestLoadForBootstrapAppliesHistoricalWeChatOAuthEnvFallbacks(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setEnvForEnvContractTest(t, map[string]string{
		"WECHAT_OAUTH_OPEN_APP_ID":           "wx-historical-open",
		"WECHAT_OAUTH_OPEN_APP_SECRET":       "historical-open-secret",
		"WECHAT_OAUTH_FRONTEND_REDIRECT_URL": "/auth/wechat/historical-callback",
	}, weChatEnvContractKeys...)

	cfg, err := LoadForBootstrap()
	if err != nil {
		t.Fatalf("LoadForBootstrap() error: %v", err)
	}

	if !cfg.WeChat.Enabled {
		t.Fatal("historical WeChat OAuth env should enable WeChat Connect")
	}
	if !cfg.WeChat.OpenEnabled {
		t.Fatal("historical WeChat OAuth open credentials should enable open login")
	}
	if cfg.WeChat.OpenAppID != "wx-historical-open" {
		t.Fatalf("wechat open_app_id = %q, want historical env value", cfg.WeChat.OpenAppID)
	}
	if cfg.WeChat.OpenAppSecret != "historical-open-secret" {
		t.Fatalf("wechat open_app_secret = %q, want historical env value", cfg.WeChat.OpenAppSecret)
	}
	if cfg.WeChat.Mode != "open" {
		t.Fatalf("wechat mode = %q, want open", cfg.WeChat.Mode)
	}
	if cfg.WeChat.Scopes != "snsapi_login" {
		t.Fatalf("wechat scopes = %q, want snsapi_login", cfg.WeChat.Scopes)
	}
	if cfg.WeChat.FrontendRedirectURL != "/auth/wechat/historical-callback" {
		t.Fatalf("wechat frontend_redirect_url = %q, want historical env value", cfg.WeChat.FrontendRedirectURL)
	}
}

func TestLoadForBootstrapPrefersCurrentWeChatConnectEnvOverHistoricalOAuthEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setEnvForEnvContractTest(t, map[string]string{
		"WECHAT_CONNECT_ENABLED":               "true",
		"WECHAT_CONNECT_OPEN_ENABLED":          "true",
		"WECHAT_CONNECT_OPEN_APP_ID":           "wx-current-open",
		"WECHAT_CONNECT_OPEN_APP_SECRET":       "current-open-secret",
		"WECHAT_CONNECT_FRONTEND_REDIRECT_URL": "/auth/wechat/current-callback",
		"WECHAT_OAUTH_OPEN_APP_ID":             "wx-historical-open",
		"WECHAT_OAUTH_OPEN_APP_SECRET":         "historical-open-secret",
		"WECHAT_OAUTH_FRONTEND_REDIRECT_URL":   "/auth/wechat/historical-callback",
	}, weChatEnvContractKeys...)

	cfg, err := LoadForBootstrap()
	if err != nil {
		t.Fatalf("LoadForBootstrap() error: %v", err)
	}

	if cfg.WeChat.OpenAppID != "wx-current-open" {
		t.Fatalf("wechat open_app_id = %q, want current env value", cfg.WeChat.OpenAppID)
	}
	if cfg.WeChat.OpenAppSecret != "current-open-secret" {
		t.Fatalf("wechat open_app_secret = %q, want current env value", cfg.WeChat.OpenAppSecret)
	}
	if cfg.WeChat.FrontendRedirectURL != "/auth/wechat/current-callback" {
		t.Fatalf("wechat frontend_redirect_url = %q, want current env value", cfg.WeChat.FrontendRedirectURL)
	}
}

var weChatEnvContractKeys = []string{
	"WECHAT_CONNECT_ENABLED",
	"WECHAT_CONNECT_APP_ID",
	"WECHAT_CONNECT_APP_SECRET",
	"WECHAT_CONNECT_OPEN_APP_ID",
	"WECHAT_CONNECT_OPEN_APP_SECRET",
	"WECHAT_CONNECT_MP_APP_ID",
	"WECHAT_CONNECT_MP_APP_SECRET",
	"WECHAT_CONNECT_MOBILE_APP_ID",
	"WECHAT_CONNECT_MOBILE_APP_SECRET",
	"WECHAT_CONNECT_OPEN_ENABLED",
	"WECHAT_CONNECT_MP_ENABLED",
	"WECHAT_CONNECT_MOBILE_ENABLED",
	"WECHAT_CONNECT_MODE",
	"WECHAT_CONNECT_SCOPES",
	"WECHAT_CONNECT_REDIRECT_URL",
	"WECHAT_CONNECT_FRONTEND_REDIRECT_URL",
	"WECHAT_OAUTH_OPEN_APP_ID",
	"WECHAT_OAUTH_OPEN_APP_SECRET",
	"WECHAT_OAUTH_MP_APP_ID",
	"WECHAT_OAUTH_MP_APP_SECRET",
	"WECHAT_OAUTH_FRONTEND_REDIRECT_URL",
}

func setEnvForEnvContractTest(t *testing.T, values map[string]string, resetKeys ...string) {
	t.Helper()

	type envState struct {
		value   string
		present bool
	}

	original := make(map[string]envState, len(resetKeys)+len(values))
	remember := func(key string) {
		if _, ok := original[key]; ok {
			return
		}
		value, present := os.LookupEnv(key)
		original[key] = envState{value: value, present: present}
	}

	for _, key := range resetKeys {
		remember(key)
	}
	for key := range values {
		remember(key)
	}

	for key := range original {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("set env %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for key, state := range original {
			if state.present {
				_ = os.Setenv(key, state.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}
