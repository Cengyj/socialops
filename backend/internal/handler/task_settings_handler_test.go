//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTaskSettingsHandlerValidateTemplateRequiresAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTaskSettingsHandler(nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task-settings/templates/validate", strings.NewReader(`{
		"name": "follow targets",
		"type": "follow",
		"params": {"targets":["@target"]}
	}`))
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	handler.ValidateTemplate(ginCtx)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "unauthorized")
}
