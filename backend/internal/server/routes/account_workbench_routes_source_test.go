//go:build unit

package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserAccountWorkbenchRoutesDoNotExposeSingleAccountImport(t *testing.T) {
	source, err := os.ReadFile("user.go")
	require.NoError(t, err)

	text := string(source)
	require.NotContains(t, text, `accounts.POST("`+"/import"+`"`)
	require.NotContains(t, text, "h.AccountWorkbench."+"Import"+"MyAccount")
	require.Contains(t, text, `accounts.POST("/batch-import"`)
	require.Contains(t, text, "BatchImportMyAccounts")
	require.Contains(t, text, `accounts.PUT("/:id", h.AccountWorkbench.UpdateMyAccount)`)
}
