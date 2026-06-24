package handler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountWorkbenchHandlerDelegatesProxyUsabilityRulesToService(t *testing.T) {
	source, err := os.ReadFile("account_workbench_handler.go")
	require.NoError(t, err)

	require.NotContains(t, string(source), "EnsureSocialIPUsableForExecution")
}

func TestAccountWorkbenchHandlerDelegatesDefaultTemplateRulesToService(t *testing.T) {
	source, err := os.ReadFile("account_workbench_handler.go")
	require.NoError(t, err)

	text := string(source)
	require.Contains(t, text, "ApplyDefaultTemplateToTaskInput")
	require.NotContains(t, text, "NormalizeSocialTaskAction")
	require.NotContains(t, text, "GetDefaultTemplate")
	require.NotContains(t, text, "ValidateTaskTemplate")
	require.NotContains(t, text, "TASK_DEFAULT_TEMPLATE_REQUIRED")
	require.NotContains(t, text, "TASK_TEMPLATE_INVALID")
}

func TestAccountWorkbenchHandlerUsesSharedDependencyGuards(t *testing.T) {
	source, err := os.ReadFile("account_workbench_handler.go")
	require.NoError(t, err)

	text := string(source)
	require.Contains(t, text, "func (h *AccountWorkbenchHandler) accountService")
	require.Contains(t, text, "SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE")
	require.Contains(t, text, "func (h *AccountWorkbenchHandler) proxyService")
	require.Contains(t, text, "proxyServiceUnavailableError()")
	require.Contains(t, text, "func (h *AccountWorkbenchHandler) taskSubmitService")
	require.Contains(t, text, "SOCIAL_TASK_SERVICE_UNAVAILABLE")
	require.Contains(t, text, "svc.ListByUser")
	require.NotContains(t, text, "h.svc.ListByUser")
	require.NotContains(t, text, "h.svc.BatchImportForUser")
	require.NotContains(t, text, "h.svc.BatchDeleteForUser")
	require.NotContains(t, text, "h.ipSvc.GetByIDForUser")
	require.NotContains(t, text, "h.ipSvc.ListUsableByUser")
}
