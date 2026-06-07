//go:build unit

package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type apiKeyHandlerUpdateRepoStub struct {
	apiKey  *service.APIKey
	updated *service.APIKey
}

func (s *apiKeyHandlerUpdateRepoStub) Create(context.Context, *service.APIKey) error {
	panic("unexpected Create call")
}

func (s *apiKeyHandlerUpdateRepoStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	if s.apiKey == nil {
		return nil, service.ErrAPIKeyNotFound
	}
	clone := *s.apiKey
	return &clone, nil
}

func (s *apiKeyHandlerUpdateRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}

func (s *apiKeyHandlerUpdateRepoStub) GetByKey(context.Context, string) (*service.APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *apiKeyHandlerUpdateRepoStub) GetByKeyForAuth(context.Context, string) (*service.APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *apiKeyHandlerUpdateRepoStub) Update(_ context.Context, key *service.APIKey) error {
	clone := *key
	s.updated = &clone
	return nil
}

func (s *apiKeyHandlerUpdateRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *apiKeyHandlerUpdateRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (s *apiKeyHandlerUpdateRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}

func (s *apiKeyHandlerUpdateRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *apiKeyHandlerUpdateRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected ExistsByKey call")
}

func (s *apiKeyHandlerUpdateRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *apiKeyHandlerUpdateRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]service.APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *apiKeyHandlerUpdateRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *apiKeyHandlerUpdateRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *apiKeyHandlerUpdateRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *apiKeyHandlerUpdateRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *apiKeyHandlerUpdateRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *apiKeyHandlerUpdateRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *apiKeyHandlerUpdateRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *apiKeyHandlerUpdateRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *apiKeyHandlerUpdateRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *apiKeyHandlerUpdateRepoStub) GetRateLimitData(context.Context, int64) (*service.APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

func newAPIKeyHandlerUpdateContext(body []byte, repo *apiKeyHandlerUpdateRepoStub) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/keys/10", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "10"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	handler := NewAPIKeyHandler(service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil))
	handler.Update(c)
	return c, recorder
}

func TestAPIKeyHandlerUpdatePreservesIPACLWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &apiKeyHandlerUpdateRepoStub{
		apiKey: &service.APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-existing",
			Name:        "old name",
			Status:      service.StatusActive,
			IPWhitelist: []string{"10.0.0.0/8"},
			IPBlacklist: []string{"192.0.2.1"},
		},
	}

	_, recorder := newAPIKeyHandlerUpdateContext([]byte(`{"name":"new name"}`), repo)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repo.updated)
	require.Equal(t, []string{"10.0.0.0/8"}, repo.updated.IPWhitelist)
	require.Equal(t, []string{"192.0.2.1"}, repo.updated.IPBlacklist)
}

func TestAPIKeyHandlerUpdateClearsIPACLWhenEmptyArraysProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &apiKeyHandlerUpdateRepoStub{
		apiKey: &service.APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-existing",
			Name:        "old name",
			Status:      service.StatusActive,
			IPWhitelist: []string{"10.0.0.0/8"},
			IPBlacklist: []string{"192.0.2.1"},
		},
	}

	_, recorder := newAPIKeyHandlerUpdateContext([]byte(`{"ip_whitelist":[],"ip_blacklist":[]}`), repo)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repo.updated)
	require.Empty(t, repo.updated.IPWhitelist)
	require.Empty(t, repo.updated.IPBlacklist)
}
