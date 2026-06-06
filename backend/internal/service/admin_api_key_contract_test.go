package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type adminAPIKeyContractRepo struct {
	APIKeyRepository
	key         *APIKey
	updateErr   error
	updateCalls int
}

func (r *adminAPIKeyContractRepo) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if r.key == nil || r.key.ID != id {
		return nil, ErrAPIKeyNotFound
	}
	copied := *r.key
	if r.key.GroupID != nil {
		gid := *r.key.GroupID
		copied.GroupID = &gid
	}
	return &copied, nil
}

func (r *adminAPIKeyContractRepo) Update(_ context.Context, key *APIKey) error {
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	copied := *key
	if key.GroupID != nil {
		gid := *key.GroupID
		copied.GroupID = &gid
	}
	r.key = &copied
	return nil
}

type adminAPIKeyContractGroupRepo struct {
	GroupRepository
	group *Group
}

func (r *adminAPIKeyContractGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	if r.group == nil || r.group.ID != id {
		return nil, ErrGroupNotFound
	}
	copied := *r.group
	return &copied, nil
}

func TestAdminServiceUpdateAPIKeyGroupAndRateLimitUsageDoesNotPersistPartialStateWhenUpdateFails(t *testing.T) {
	ctx := context.Background()
	oldGroupID := int64(1)
	nextGroupID := int64(2)
	now := time.Now().UTC()
	updateErr := errors.New("api key update failed")
	apiKeyRepo := &adminAPIKeyContractRepo{
		key: &APIKey{
			ID:            10,
			UserID:        42,
			Key:           "sk-contract",
			Name:          "contract",
			Status:        StatusActive,
			GroupID:       &oldGroupID,
			Usage5h:       1.2,
			Usage1d:       3.4,
			Usage7d:       5.6,
			Window5hStart: &now,
			Window1dStart: &now,
			Window7dStart: &now,
		},
		updateErr: updateErr,
	}
	svc := &adminServiceImpl{
		apiKeyRepo: apiKeyRepo,
		groupRepo: &adminAPIKeyContractGroupRepo{
			group: &Group{
				ID:               nextGroupID,
				Name:             "standard",
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
			},
		},
	}

	result, err := svc.AdminUpdateAPIKeyGroupAndRateLimitUsage(ctx, 10, &nextGroupID, true)

	require.Nil(t, result)
	require.ErrorIs(t, err, updateErr)
	require.Equal(t, 1, apiKeyRepo.updateCalls)
	require.NotNil(t, apiKeyRepo.key.GroupID)
	require.Equal(t, oldGroupID, *apiKeyRepo.key.GroupID)
	require.InDelta(t, 1.2, apiKeyRepo.key.Usage5h, 0.000001)
	require.InDelta(t, 3.4, apiKeyRepo.key.Usage1d, 0.000001)
	require.InDelta(t, 5.6, apiKeyRepo.key.Usage7d, 0.000001)
	require.NotNil(t, apiKeyRepo.key.Window5hStart)
	require.NotNil(t, apiKeyRepo.key.Window1dStart)
	require.NotNil(t, apiKeyRepo.key.Window7dStart)
}
