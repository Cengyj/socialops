package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type adminUserGroupContractUserRepo struct {
	UserRepository
	user    *User
	updated []*User
	created []*User
	nextID  int64
}

func (r *adminUserGroupContractUserRepo) Create(_ context.Context, user *User) error {
	if user == nil {
		return ErrInvalidInput
	}
	copied := *user
	if copied.ID == 0 {
		if r.nextID == 0 {
			r.nextID = 1
		}
		copied.ID = r.nextID
		r.nextID++
	}
	if user.AllowedGroups != nil {
		copied.AllowedGroups = append([]int64(nil), user.AllowedGroups...)
	}
	*user = copied
	r.created = append(r.created, &copied)
	r.user = &copied
	return nil
}

func (r *adminUserGroupContractUserRepo) GetByID(context.Context, int64) (*User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	copied := *r.user
	if r.user.AllowedGroups != nil {
		copied.AllowedGroups = append([]int64(nil), r.user.AllowedGroups...)
	}
	return &copied, nil
}

func (r *adminUserGroupContractUserRepo) Update(_ context.Context, user *User) error {
	copied := *user
	if user.AllowedGroups != nil {
		copied.AllowedGroups = append([]int64(nil), user.AllowedGroups...)
	}
	r.updated = append(r.updated, &copied)
	r.user = &copied
	return nil
}

type adminUserGroupContractAuthInvalidator struct {
	userIDs []int64
}

func (s *adminUserGroupContractAuthInvalidator) InvalidateAuthCacheByKey(context.Context, string) {}

func (s *adminUserGroupContractAuthInvalidator) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *adminUserGroupContractAuthInvalidator) InvalidateAuthCacheByGroupID(context.Context, int64) {
}

type adminUserGroupContractRateRepo struct {
	UserGroupRateRepository
	syncUserID int64
	syncRates  map[int64]*float64
	syncCalls  int
}

func (r *adminUserGroupContractRateRepo) SyncUserGroupRates(_ context.Context, userID int64, rates map[int64]*float64) error {
	r.syncUserID = userID
	r.syncRates = rates
	r.syncCalls++
	return nil
}

func TestAdminServiceUpdateUserAllowedGroupsInvalidatesAuthCache(t *testing.T) {
	ctx := context.Background()
	userRepo := &adminUserGroupContractUserRepo{
		user: &User{
			ID:            42,
			Email:         "user@example.com",
			Role:          RoleUser,
			Status:        StatusActive,
			Concurrency:   2,
			AllowedGroups: []int64{1},
		},
	}
	invalidator := &adminUserGroupContractAuthInvalidator{}
	svc := &adminServiceImpl{
		userRepo:             userRepo,
		authCacheInvalidator: invalidator,
	}
	nextGroups := []int64{2, 3}

	updated, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{AllowedGroups: &nextGroups})

	require.NoError(t, err)
	require.Equal(t, []int64{2, 3}, updated.AllowedGroups)
	require.Equal(t, []int64{42}, invalidator.userIDs)
}

func TestAdminServiceUpdateUserGroupRatesInvalidatesAuthCacheAfterSync(t *testing.T) {
	ctx := context.Background()
	userRepo := &adminUserGroupContractUserRepo{
		user: &User{
			ID:          42,
			Email:       "user@example.com",
			Role:        RoleUser,
			Status:      StatusActive,
			Concurrency: 2,
		},
	}
	invalidator := &adminUserGroupContractAuthInvalidator{}
	rateRepo := &adminUserGroupContractRateRepo{}
	svc := &adminServiceImpl{
		userRepo:             userRepo,
		userGroupRateRepo:    rateRepo,
		authCacheInvalidator: invalidator,
	}
	rate := 1.25
	rates := map[int64]*float64{9: &rate}

	_, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{GroupRates: rates})

	require.NoError(t, err)
	require.Equal(t, 1, rateRepo.syncCalls)
	require.Equal(t, int64(42), rateRepo.syncUserID)
	require.Same(t, &rate, rateRepo.syncRates[9])
	require.Equal(t, []int64{42}, invalidator.userIDs)
}

func TestAdminServiceUpdateUserPasswordIncrementsTokenVersion(t *testing.T) {
	ctx := context.Background()
	user := &User{
		ID:           42,
		Email:        "user@example.com",
		Role:         RoleUser,
		Status:       StatusActive,
		Concurrency:  2,
		TokenVersion: 7,
	}
	require.NoError(t, user.SetPassword("old-password"))
	userRepo := &adminUserGroupContractUserRepo{user: user}
	invalidator := &adminUserGroupContractAuthInvalidator{}
	svc := &adminServiceImpl{
		userRepo:             userRepo,
		authCacheInvalidator: invalidator,
	}

	updated, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{Password: "new-password"})

	require.NoError(t, err)
	require.Equal(t, int64(8), updated.TokenVersion)
	require.True(t, updated.CheckPassword("new-password"))
	require.False(t, updated.CheckPassword("old-password"))
	require.Equal(t, []int64{42}, invalidator.userIDs)
}

func TestAdminServiceCreateUserDefaultSubscriptionsPreservePlanID(t *testing.T) {
	ctx := context.Background()
	userRepo := &adminUserGroupContractUserRepo{nextID: 77}
	assigner := &defaultSubscriptionAssignerStub{}
	settingSvc := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyDefaultSubscriptions: `[{"plan_id":301,"validity_days":30}]`,
	}}, nil)
	svc := &adminServiceImpl{
		userRepo:           userRepo,
		settingService:     settingSvc,
		defaultSubAssigner: assigner,
	}

	user, err := svc.CreateUser(ctx, &CreateUserInput{
		Email:       "created@example.com",
		Password:    "password",
		Concurrency: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, userRepo.created, 1)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(77), assigner.calls[0].UserID)
	require.Zero(t, assigner.calls[0].GroupID)
	require.NotNil(t, assigner.calls[0].PlanID)
	require.Equal(t, int64(301), *assigner.calls[0].PlanID)
	require.Equal(t, 30, assigner.calls[0].ValidityDays)
}
