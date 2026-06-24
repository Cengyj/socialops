//go:build unit

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialAccountServiceKeepsSelectedWorkbenchUploadToPool(t *testing.T) {
	raw, err := os.ReadFile("social_account_service.go")
	require.NoError(t, err)

	source := string(raw)
	require.Contains(t, source, "StoreWorkbenchAccounts")
	require.Contains(t, source, "workbenchStagingAccountPredicate()")
	require.Contains(t, source, "SetAccountStatus(SocialAccountStatusPendingCheck)")
	require.Contains(t, source, "SetTaskStatus(SocialTaskStatusStored)")
	require.NotContains(t, source, "SocialAccountStoreResult")
}
