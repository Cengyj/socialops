package handler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthHandlerDoesNotLogTemporaryAuthTokens(t *testing.T) {
	source, err := os.ReadFile("auth_handler.go")
	require.NoError(t, err)

	require.NotContains(t, string(source), "temp_token_prefix")
}
