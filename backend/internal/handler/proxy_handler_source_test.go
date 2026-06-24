//go:build unit

package handler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyHandlerDelegatesListFilterNormalizationToService(t *testing.T) {
	source, err := os.ReadFile("proxy_handler.go")
	require.NoError(t, err)

	text := string(source)
	require.Contains(t, text, `Status: c.Query("status")`)
	require.Contains(t, text, `IPType: c.Query("ip_type")`)
	require.Contains(t, text, `Search: c.Query("search")`)
	require.NotContains(t, text, `strings.TrimSpace(c.Query("status"))`)
	require.NotContains(t, text, `strings.TrimSpace(c.Query("ip_type"))`)
	require.NotContains(t, text, `strings.TrimSpace(c.Query("search"))`)
}
