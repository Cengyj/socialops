package handler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskSettingsHandlerDelegatesTemplateValidationToService(t *testing.T) {
	source, err := os.ReadFile("task_settings_handler.go")
	require.NoError(t, err)

	text := string(source)
	require.Contains(t, text, "svc.ValidateTemplateInput")
	require.NotContains(t, text, "service.ValidateTaskTemplateInput")
}

func TestTaskSettingsHandlerUsesSharedServiceUnavailableGuard(t *testing.T) {
	source, err := os.ReadFile("task_settings_handler.go")
	require.NoError(t, err)

	text := string(source)
	require.Contains(t, text, "func (h *TaskSettingsHandler) taskSettingsService")
	require.Contains(t, text, "taskTemplateServiceUnavailableError")
	require.Contains(t, text, "TASK_TEMPLATE_SERVICE_UNAVAILABLE")
	require.NotContains(t, text, "h.svc.ListTemplates")
	require.NotContains(t, text, "h.svc.SaveTemplate")
	require.NotContains(t, text, "h.svc.DeleteTemplate")
	require.NotContains(t, text, "h.svc.CopyTemplate")
	require.NotContains(t, text, "h.svc.SetDefaultTemplate")
	require.NotContains(t, text, "h.svc.PreviewTemplateMedia")
}
