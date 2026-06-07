//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type backupHandlerSettingRepo struct {
	mu   sync.Mutex
	data map[string]string
}

func newBackupHandlerSettingRepo() *backupHandlerSettingRepo {
	return &backupHandlerSettingRepo{data: make(map[string]string)}
}

func (r *backupHandlerSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.data[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *backupHandlerSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.data[key], nil
}

func (r *backupHandlerSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	return nil
}

func (r *backupHandlerSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.data[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (r *backupHandlerSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range values {
		r.data[key] = value
	}
	return nil
}

func (r *backupHandlerSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make(map[string]string, len(r.data))
	for key, value := range r.data {
		values[key] = value
	}
	return values, nil
}

func (r *backupHandlerSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, key)
	return nil
}

type backupHandlerEncryptor struct{}

func (backupHandlerEncryptor) Encrypt(plaintext string) (string, error) {
	return "ENC:" + plaintext, nil
}

func (backupHandlerEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "ENC:"), nil
}

type backupHandlerDumper struct{}

func (backupHandlerDumper) Dump(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("select 1;"))), nil
}

func (backupHandlerDumper) Restore(context.Context, io.Reader) error {
	return nil
}

type backupHandlerObjectStore struct{}

func (backupHandlerObjectStore) Upload(_ context.Context, _ string, body io.Reader, _ string) (int64, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func (backupHandlerObjectStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unexpected download: %s", key)
}

func (backupHandlerObjectStore) Delete(context.Context, string) error {
	return nil
}

func (backupHandlerObjectStore) PresignURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (backupHandlerObjectStore) HeadBucket(context.Context) error {
	return nil
}

func TestBackupHandler_CreateBackupRejectsMalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newBackupHandlerSettingRepo()
	svc := service.NewBackupService(
		repo,
		&config.Config{
			Database: config.DatabaseConfig{
				DBName: "socialops_test",
			},
		},
		backupHandlerEncryptor{},
		func(context.Context, *service.BackupS3Config) (service.BackupObjectStore, error) {
			return backupHandlerObjectStore{}, nil
		},
		backupHandlerDumper{},
	)
	_, err := svc.UpdateS3Config(context.Background(), service.BackupS3Config{
		Bucket:          "backups",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		Prefix:          "backups",
	})
	require.NoError(t, err)

	handler := NewBackupHandler(svc, nil)
	router := gin.New()
	router.POST("/api/v1/admin/backups", handler.CreateBackup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid request")

	records, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestBackupHandler_CreateBackupAllowsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newBackupHandlerSettingRepo()
	svc := service.NewBackupService(
		repo,
		&config.Config{
			Database: config.DatabaseConfig{
				DBName: "socialops_test",
			},
		},
		backupHandlerEncryptor{},
		func(context.Context, *service.BackupS3Config) (service.BackupObjectStore, error) {
			return backupHandlerObjectStore{}, nil
		},
		backupHandlerDumper{},
	)
	_, err := svc.UpdateS3Config(context.Background(), service.BackupS3Config{
		Bucket:          "backups",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		Prefix:          "backups",
	})
	require.NoError(t, err)

	handler := NewBackupHandler(svc, nil)
	router := gin.New()
	router.POST("/api/v1/admin/backups", handler.CreateBackup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var envelope apiEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
}
