package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemUpdateCacheStub struct {
	data string
}

func (s *systemUpdateCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *systemUpdateCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type systemReleaseClientStub struct{}

func (systemReleaseClientStub) FetchLatestRelease(context.Context, string) (*service.GitHubRelease, error) {
	return &service.GitHubRelease{
		TagName: "v1.0.1",
		Name:    "v1.0.1",
		Assets:  []service.GitHubAsset{},
	}, nil
}

func (systemReleaseClientStub) DownloadFile(context.Context, string, string, int64) error {
	return errors.New("download should not run in handler test")
}

func (systemReleaseClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, errors.New("checksum should not run in handler test")
}

func TestSystemHandler_PerformUpdateUsesConfiguredOperationLock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateSvc := service.NewUpdateService(&systemUpdateCacheStub{}, systemReleaseClientStub{}, "1.0.0", "release")
	lockSvc := service.NewSystemOperationLockService(newMemoryIdempotencyRepoStub(), service.DefaultIdempotencyConfig())
	h := NewSystemHandler(updateSvc, lockSvc)

	router := gin.New()
	router.POST("/api/v1/admin/system/update", h.PerformUpdate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "system-update-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), service.ErrIdempotencyStoreUnavail.Error())
}

func TestSystemHandler_PerformUpdateFailsClosedWhenOperationLockStoreMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateSvc := service.NewUpdateService(&systemUpdateCacheStub{}, systemReleaseClientStub{}, "1.0.0", "release")
	h := NewSystemHandler(updateSvc, service.NewSystemOperationLockService(nil, service.DefaultIdempotencyConfig()))

	router := gin.New()
	router.POST("/api/v1/admin/system/update", h.PerformUpdate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "system-update-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "IDEMPOTENCY_STORE_UNAVAILABLE")
}
