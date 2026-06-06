//go:build unit

package service

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialIPServiceDoesNotExposeAdminWideProxyListing(t *testing.T) {
	raw, err := os.ReadFile("social_ip_service.go")
	require.NoError(t, err)

	source := string(raw)
	require.NotContains(t, source, "ListForAdmin")
	require.NotContains(t, source, "execution proxies across users")
	require.NotContains(t, strings.ReplaceAll(source, " ", ""), "UserID*int64")
}
