package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Reason  string          `json:"reason"`
	Data    json.RawMessage `json:"data"`
}

func TestDataManagementHandler_AgentHealthAlways200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewDataManagementServiceWithOptions(filepath.Join(t.TempDir(), "missing.sock"), 50*time.Millisecond)
	h := NewDataManagementHandler(svc)

	r := gin.New()
	r.GET("/api/v1/admin/data-management/agent/health", h.GetAgentHealth)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/data-management/agent/health", nil)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var envelope apiEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)

	var data struct {
		Enabled    bool   `json:"enabled"`
		Reason     string `json:"reason"`
		SocketPath string `json:"socket_path"`
	}
	require.NoError(t, json.Unmarshal(envelope.Data, &data))
	require.False(t, data.Enabled)
	require.Equal(t, service.DataManagementDeprecatedReason, data.Reason)
	require.Equal(t, svc.SocketPath(), data.SocketPath)
}

func TestDataManagementHandler_NonHealthRouteReturns503WhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewDataManagementServiceWithOptions(filepath.Join(t.TempDir(), "missing.sock"), 50*time.Millisecond)
	h := NewDataManagementHandler(svc)

	r := gin.New()
	r.GET("/api/v1/admin/data-management/config", h.GetConfig)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/data-management/config", nil)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var envelope apiEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, http.StatusServiceUnavailable, envelope.Code)
	require.Equal(t, service.DataManagementDeprecatedReason, envelope.Reason)
}

func TestDataManagementHandler_WriteRoutesReturn503BeforeParsingBodyWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewDataManagementServiceWithOptions(filepath.Join(t.TempDir(), "missing.sock"), 50*time.Millisecond)
	h := NewDataManagementHandler(svc)

	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*gin.Engine)
	}{
		{
			name:   "update config",
			method: http.MethodPut,
			path:   "/api/v1/admin/data-management/config",
			setup: func(r *gin.Engine) {
				r.PUT("/api/v1/admin/data-management/config", h.UpdateConfig)
			},
		},
		{
			name:   "test s3",
			method: http.MethodPost,
			path:   "/api/v1/admin/data-management/s3/test",
			setup: func(r *gin.Engine) {
				r.POST("/api/v1/admin/data-management/s3/test", h.TestS3)
			},
		},
		{
			name:   "create backup job",
			method: http.MethodPost,
			path:   "/api/v1/admin/data-management/backups",
			setup: func(r *gin.Engine) {
				r.POST("/api/v1/admin/data-management/backups", h.CreateBackupJob)
			},
		},
		{
			name:   "create source profile",
			method: http.MethodPost,
			path:   "/api/v1/admin/data-management/sources/postgres/profiles",
			setup: func(r *gin.Engine) {
				r.POST("/api/v1/admin/data-management/sources/:source_type/profiles", h.CreateSourceProfile)
			},
		},
		{
			name:   "update source profile",
			method: http.MethodPut,
			path:   "/api/v1/admin/data-management/sources/postgres/profiles/default",
			setup: func(r *gin.Engine) {
				r.PUT("/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id", h.UpdateSourceProfile)
			},
		},
		{
			name:   "create s3 profile",
			method: http.MethodPost,
			path:   "/api/v1/admin/data-management/s3/profiles",
			setup: func(r *gin.Engine) {
				r.POST("/api/v1/admin/data-management/s3/profiles", h.CreateS3Profile)
			},
		},
		{
			name:   "update s3 profile",
			method: http.MethodPut,
			path:   "/api/v1/admin/data-management/s3/profiles/default",
			setup: func(r *gin.Engine) {
				r.PUT("/api/v1/admin/data-management/s3/profiles/:profile_id", h.UpdateS3Profile)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			tt.setup(r)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("{"))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(rec, req)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)

			var envelope apiEnvelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
			require.Equal(t, http.StatusServiceUnavailable, envelope.Code)
			require.Equal(t, service.DataManagementDeprecatedReason, envelope.Reason)

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, "{", string(body))
		})
	}
}

func TestNormalizeBackupIdempotencyKey(t *testing.T) {
	require.Equal(t, "from-header", normalizeBackupIdempotencyKey("from-header", "from-body"))
	require.Equal(t, "from-body", normalizeBackupIdempotencyKey(" ", " from-body "))
	require.Equal(t, "", normalizeBackupIdempotencyKey("", ""))
}
