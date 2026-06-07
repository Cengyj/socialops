//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type apiKeyUpdateRepoStub struct {
	apiKey    *APIKey
	created   *APIKey
	updated   *APIKey
	updateErr error
}

func (s *apiKeyUpdateRepoStub) Create(_ context.Context, key *APIKey) error {
	clone := *key
	if clone.ID == 0 {
		clone.ID = 101
	}
	s.created = &clone
	*key = clone
	return nil
}

func (s *apiKeyUpdateRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	if s.apiKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *s.apiKey
	return &clone, nil
}

func (s *apiKeyUpdateRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}

func (s *apiKeyUpdateRepoStub) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *apiKeyUpdateRepoStub) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *apiKeyUpdateRepoStub) Update(_ context.Context, key *APIKey) error {
	clone := *key
	s.updated = &clone
	return s.updateErr
}

func (s *apiKeyUpdateRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *apiKeyUpdateRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (s *apiKeyUpdateRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}

func (s *apiKeyUpdateRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *apiKeyUpdateRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	return false, nil
}

func (s *apiKeyUpdateRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *apiKeyUpdateRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *apiKeyUpdateRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *apiKeyUpdateRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *apiKeyUpdateRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *apiKeyUpdateRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *apiKeyUpdateRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *apiKeyUpdateRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *apiKeyUpdateRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *apiKeyUpdateRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *apiKeyUpdateRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *apiKeyUpdateRepoStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

func TestAPIKeyService_UpdatePreservesIPACLWhenOmitted(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		apiKey: &APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-existing",
			Name:        "old name",
			Status:      StatusActive,
			IPWhitelist: []string{"10.0.0.0/8"},
			IPBlacklist: []string{"192.0.2.1"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	newName := "new name"

	key, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{Name: &newName})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Equal(t, newName, key.Name)
	require.Equal(t, []string{"10.0.0.0/8"}, repo.updated.IPWhitelist)
	require.Equal(t, []string{"192.0.2.1"}, repo.updated.IPBlacklist)
}

func TestAPIKeyService_UpdateClearsIPACLWhenEmptyArraysProvided(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		apiKey: &APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-existing",
			Name:        "old name",
			Status:      StatusActive,
			IPWhitelist: []string{"10.0.0.0/8"},
			IPBlacklist: []string{"192.0.2.1"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	emptyWhitelist := []string{}
	emptyBlacklist := []string{}

	key, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{
		IPWhitelist: &emptyWhitelist,
		IPBlacklist: &emptyBlacklist,
	})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Empty(t, key.IPWhitelist)
	require.Empty(t, key.IPBlacklist)
	require.Empty(t, repo.updated.IPWhitelist)
	require.Empty(t, repo.updated.IPBlacklist)
}

type apiKeyGroupRepoStub struct {
	groups map[int64]*Group
}

func (s *apiKeyGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := s.groups[id]; ok {
		clone := *group
		return &clone, nil
	}
	return nil, ErrGroupNotFound
}

func (s *apiKeyGroupRepoStub) Create(context.Context, *Group) error { panic("unexpected Create call") }
func (s *apiKeyGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	panic("unexpected GetByIDLite call")
}
func (s *apiKeyGroupRepoStub) Update(context.Context, *Group) error { panic("unexpected Update call") }
func (s *apiKeyGroupRepoStub) Delete(context.Context, int64) error  { panic("unexpected Delete call") }
func (s *apiKeyGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}
func (s *apiKeyGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *apiKeyGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *apiKeyGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	panic("unexpected ListActive call")
}
func (s *apiKeyGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected ListActiveByPlatform call")
}
func (s *apiKeyGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName call")
}
func (s *apiKeyGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

type apiKeyUserSubRepoStub struct {
	active map[int64]*UserSubscription
}

func (s *apiKeyUserSubRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if sub, ok := s.active[groupID]; ok && sub.UserID == userID {
		clone := *sub
		return &clone, nil
	}
	return nil, ErrSubscriptionNotFound
}

func (s *apiKeyUserSubRepoStub) Create(context.Context, *UserSubscription) error {
	panic("unexpected Create call")
}
func (s *apiKeyUserSubRepoStub) GetByID(context.Context, int64) (*UserSubscription, error) {
	panic("unexpected GetByID call")
}
func (s *apiKeyUserSubRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetByUserIDAndGroupID call")
}
func (s *apiKeyUserSubRepoStub) Update(context.Context, *UserSubscription) error {
	panic("unexpected Update call")
}
func (s *apiKeyUserSubRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *apiKeyUserSubRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListByUserID call")
}
func (s *apiKeyUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListActiveByUserID call")
}
func (s *apiKeyUserSubRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}
func (s *apiKeyUserSubRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *apiKeyUserSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	panic("unexpected ExistsByUserIDAndGroupID call")
}
func (s *apiKeyUserSubRepoStub) ExtendExpiry(context.Context, int64, time.Time) error {
	panic("unexpected ExtendExpiry call")
}
func (s *apiKeyUserSubRepoStub) UpdateStatus(context.Context, int64, string) error {
	panic("unexpected UpdateStatus call")
}
func (s *apiKeyUserSubRepoStub) UpdateNotes(context.Context, int64, string) error {
	panic("unexpected UpdateNotes call")
}
func (s *apiKeyUserSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error {
	panic("unexpected ActivateWindows call")
}
func (s *apiKeyUserSubRepoStub) ResetDailyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetDailyUsage call")
}
func (s *apiKeyUserSubRepoStub) ResetWeeklyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetWeeklyUsage call")
}
func (s *apiKeyUserSubRepoStub) ResetMonthlyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetMonthlyUsage call")
}
func (s *apiKeyUserSubRepoStub) IncrementUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementUsage call")
}
func (s *apiKeyUserSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	panic("unexpected BatchUpdateExpiredStatus call")
}

func TestAPIKeyService_CreatePreservesAllowedGroupBinding(t *testing.T) {
	groupID := int64(22)
	repo := &apiKeyUpdateRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo: &userRepoStub{user: &User{
			ID:            7,
			AllowedGroups: []int64{groupID},
		}},
		groupRepo: &apiKeyGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, Status: StatusActive, IsExclusive: true},
		}},
	}

	key, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:      "scoped key",
		GroupID:   &groupID,
		CustomKey: testPtrString("sk-scoped-key-123"),
	})

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.NotNil(t, key.GroupID)
	require.Equal(t, groupID, *key.GroupID)
	require.Equal(t, groupID, *repo.created.GroupID)
}

func TestAPIKeyService_CreateRejectsDisallowedExclusiveGroup(t *testing.T) {
	groupID := int64(22)
	repo := &apiKeyUpdateRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &userRepoStub{user: &User{ID: 7}},
		groupRepo: &apiKeyGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, Status: StatusActive, IsExclusive: true},
		}},
	}

	key, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:      "blocked key",
		GroupID:   &groupID,
		CustomKey: testPtrString("sk-blocked-key-123"),
	})

	require.ErrorIs(t, err, ErrGroupNotAllowed)
	require.Nil(t, key)
	require.Nil(t, repo.created)
}

func TestAPIKeyService_UpdateRejectsDisallowedGroupBinding(t *testing.T) {
	groupID := int64(22)
	repo := &apiKeyUpdateRepoStub{
		apiKey: &APIKey{
			ID:     10,
			UserID: 7,
			Key:    "sk-existing",
			Name:   "old name",
			Status: StatusActive,
		},
	}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &userRepoStub{user: &User{ID: 7}},
		groupRepo: &apiKeyGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, Status: StatusActive, IsExclusive: true},
		}},
	}

	key, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{GroupID: &groupID})

	require.ErrorIs(t, err, ErrGroupNotAllowed)
	require.Nil(t, key)
	require.Nil(t, repo.updated)
}

func TestAPIKeyService_UpdateAllowsActiveSubscriptionGroup(t *testing.T) {
	groupID := int64(33)
	repo := &apiKeyUpdateRepoStub{
		apiKey: &APIKey{
			ID:     10,
			UserID: 7,
			Key:    "sk-existing",
			Name:   "old name",
			Status: StatusActive,
		},
	}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &userRepoStub{user: &User{ID: 7}},
		groupRepo: &apiKeyGroupRepoStub{groups: map[int64]*Group{
			groupID: {ID: groupID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
		}},
		userSubRepo: &apiKeyUserSubRepoStub{active: map[int64]*UserSubscription{
			groupID: {UserID: 7, GroupID: groupID, Status: SubscriptionStatusActive},
		}},
	}

	key, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{GroupID: &groupID})

	require.NoError(t, err)
	require.NotNil(t, key.GroupID)
	require.Equal(t, groupID, *key.GroupID)
	require.NotNil(t, repo.updated)
	require.Equal(t, groupID, *repo.updated.GroupID)
}
