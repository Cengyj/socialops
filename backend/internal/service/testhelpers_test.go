//go:build unit

package service

import (
	"context"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

// testPtrFloat64 returns a pointer to the given float64 value.
func testPtrFloat64(v float64) *float64 { return &v }

// testPtrInt returns a pointer to the given int value.
func testPtrInt(v int) *int { return &v }

// testPtrString returns a pointer to the given string value.
func testPtrString(v string) *string { return &v }

// testPtrBool returns a pointer to the given bool value.
func testPtrBool(v bool) *bool { return &v }

type userRepoStub struct {
	UserRepository
	user          *User
	createErr     error
	getErr        error
	updateErr     error
	deleteErr     error
	exists        bool
	existsErr     error
	nextID        int64
	created       []*User
	updated       []*User
	deletedIDs    []int64
	usersByEmail  map[string]*User
	getByEmailErr error
}

func (s *userRepoStub) Create(_ context.Context, user *User) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.nextID != 0 && user.ID == 0 {
		user.ID = s.nextID
	}
	s.created = append(s.created, user)
	if s.usersByEmail == nil {
		s.usersByEmail = make(map[string]*User)
	}
	s.usersByEmail[user.Email] = user
	s.user = user
	return nil
}

func (s *userRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}

func (s *userRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	if s.getByEmailErr != nil {
		return nil, s.getByEmailErr
	}
	if s.usersByEmail != nil {
		if user, ok := s.usersByEmail[email]; ok {
			return user, nil
		}
	}
	if s.user != nil && s.user.Email == email {
		return s.user, nil
	}
	return nil, ErrUserNotFound
}

func (s *userRepoStub) Update(_ context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, user)
	if s.usersByEmail == nil {
		s.usersByEmail = make(map[string]*User)
	}
	s.usersByEmail[user.Email] = user
	s.user = user
	return nil
}

func (s *userRepoStub) Delete(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	return s.deleteErr
}

func (s *userRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	if s.existsErr != nil {
		return false, s.existsErr
	}
	return s.exists, nil
}

func (s *userRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(_ context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(_ context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

type ensureEmailCall struct {
	userID int64
	email  string
}

type replaceEmailCall struct {
	userID   int64
	oldEmail string
	newEmail string
}

type emailSyncRepoStub struct {
	UserRepository

	user         *User
	updateCalls  int
	updated      []*User
	ensureCalls  []ensureEmailCall
	replaceCalls []replaceEmailCall
	ensureErr    error
	replaceErr   error
}

func (s *emailSyncRepoStub) GetByID(_ context.Context, _ int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	cloned := *s.user
	return &cloned, nil
}

func (s *emailSyncRepoStub) Update(_ context.Context, user *User) error {
	s.updateCalls++
	s.updated = append(s.updated, user)
	s.user = user
	return nil
}

func (s *emailSyncRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

func (s *emailSyncRepoStub) EnsureEmailAuthIdentity(_ context.Context, userID int64, email string) error {
	s.ensureCalls = append(s.ensureCalls, ensureEmailCall{userID: userID, email: email})
	return s.ensureErr
}

func (s *emailSyncRepoStub) ReplaceEmailAuthIdentity(_ context.Context, userID int64, oldEmail, newEmail string) error {
	s.replaceCalls = append(s.replaceCalls, replaceEmailCall{
		userID:   userID,
		oldEmail: oldEmail,
		newEmail: newEmail,
	})
	return s.replaceErr
}

type redeemRepoStub struct {
	deleteErrByID map[int64]error
	deletedIDs    []int64

	batchUpdateIDs    []int64
	batchUpdateFields RedeemCodeBatchUpdateFields
	batchUpdateResult int64
	batchUpdateErr    error
	batchUpdateCalled bool
}

func (s *redeemRepoStub) Create(context.Context, *RedeemCode) error {
	panic("unexpected Create call")
}

func (s *redeemRepoStub) CreateBatch(context.Context, []RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (s *redeemRepoStub) GetByID(context.Context, int64) (*RedeemCode, error) {
	panic("unexpected GetByID call")
}

func (s *redeemRepoStub) GetByCode(context.Context, string) (*RedeemCode, error) {
	panic("unexpected GetByCode call")
}

func (s *redeemRepoStub) Update(context.Context, *RedeemCode) error {
	panic("unexpected Update call")
}

func (s *redeemRepoStub) BatchUpdate(_ context.Context, ids []int64, fields RedeemCodeBatchUpdateFields) (int64, error) {
	s.batchUpdateCalled = true
	s.batchUpdateIDs = append([]int64(nil), ids...)
	s.batchUpdateFields = fields
	if s.batchUpdateErr != nil {
		return 0, s.batchUpdateErr
	}
	if s.batchUpdateResult != 0 {
		return s.batchUpdateResult, nil
	}
	return int64(len(ids)), nil
}

func (s *redeemRepoStub) Delete(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	if s.deleteErrByID != nil {
		if err, ok := s.deleteErrByID[id]; ok {
			return err
		}
	}
	return nil
}

func (s *redeemRepoStub) Use(context.Context, int64, int64) error {
	panic("unexpected Use call")
}

func (s *redeemRepoStub) List(context.Context, pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *redeemRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *redeemRepoStub) ListByUser(context.Context, int64, int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (s *redeemRepoStub) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (s *redeemRepoStub) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}
