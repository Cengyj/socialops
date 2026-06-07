//go:build !unit

package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

type settingRepoStub struct {
	values map[string]string
	err    error
}

func (s *settingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *settingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			result[key] = v
		}
	}
	return result, nil
}

func (s *settingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type defaultSubscriptionAssignerStub struct {
	calls []AssignSubscriptionInput
	err   error
}

func (s *defaultSubscriptionAssignerStub) AssignOrExtendSubscription(_ context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	if input != nil {
		s.calls = append(s.calls, *input)
	}
	if s.err != nil {
		return nil, false, s.err
	}
	if input == nil {
		return &UserSubscription{}, false, nil
	}
	return &UserSubscription{UserID: input.UserID, GroupID: input.GroupID}, false, nil
}

type mockUserRepo struct {
	updateBalanceErr error
	updateBalanceFn  func(ctx context.Context, id int64, amount float64) error
	getByIDUser      *User
	getByIDErr       error
}

func (m *mockUserRepo) Create(context.Context, *User) error { return nil }

func (m *mockUserRepo) GetByID(context.Context, int64) (*User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if m.getByIDUser != nil {
		cloned := *m.getByIDUser
		return &cloned, nil
	}
	return &User{}, nil
}

func (m *mockUserRepo) GetByEmail(context.Context, string) (*User, error) { return &User{}, nil }
func (m *mockUserRepo) GetFirstAdmin(context.Context) (*User, error)      { return &User{}, nil }
func (m *mockUserRepo) Update(context.Context, *User) error               { return nil }
func (m *mockUserRepo) Delete(context.Context, int64) error               { return nil }
func (m *mockUserRepo) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	return nil, nil
}
func (m *mockUserRepo) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}
func (m *mockUserRepo) DeleteUserAvatar(context.Context, int64) error { return nil }
func (m *mockUserRepo) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockUserRepo) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockUserRepo) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	return map[int64]*time.Time{}, nil
}
func (m *mockUserRepo) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}
func (m *mockUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	if m.updateBalanceFn != nil {
		return m.updateBalanceFn(ctx, id, amount)
	}
	return m.updateBalanceErr
}
func (m *mockUserRepo) DeductBalance(context.Context, int64, float64) error { return nil }
func (m *mockUserRepo) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (m *mockUserRepo) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (m *mockUserRepo) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	return nil, nil
}
func (m *mockUserRepo) UnbindUserAuthProvider(context.Context, int64, string) error {
	return nil
}
func (m *mockUserRepo) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (m *mockUserRepo) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (m *mockUserRepo) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (m *mockUserRepo) UpdateTotpSecret(context.Context, int64, *string) error { return nil }
func (m *mockUserRepo) EnableTotp(context.Context, int64) error                { return nil }
func (m *mockUserRepo) DisableTotp(context.Context, int64) error               { return nil }

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
