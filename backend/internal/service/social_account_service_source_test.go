//go:build unit

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialAccountServiceDoesNotKeepWorkbenchToPoolUploadShortcut(t *testing.T) {
	raw, err := os.ReadFile("social_account_service.go")
	require.NoError(t, err)

	source := string(raw)
	require.NotContains(t, source, "StoreWorkbenchAccounts")
	require.NotContains(t, source, "SocialAccountStoreResult")
	require.NotContains(t, source, "workbench accounts as stored")
}
