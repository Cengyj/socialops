package service

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTotpServiceDoesNotLogSecretMaterial(t *testing.T) {
	source, err := os.ReadFile("totp_service.go")
	require.NoError(t, err)

	content := string(source)
	for _, forbidden := range []string{
		"secret_prefix",
		"decrypted_prefix",
		"totp_complete_setup_before_encrypt",
		"totp_complete_setup_verified",
		"totp_verify_decrypted",
	} {
		require.NotContains(t, content, forbidden)
	}
	require.False(t, strings.Contains(content, `slog.Debug("totp_verify_result"`) &&
		strings.Contains(content, `"secret_len"`), "totp verify result logs must not include decrypted secret metadata")
}
