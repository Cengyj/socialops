package config

import (
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
