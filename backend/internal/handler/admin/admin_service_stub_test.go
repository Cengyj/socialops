package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/socialops/internal/service"
)

type stubAdminService struct {
	users             []service.User
	redeems           []service.RedeemCode
	getUserErr        error
	boundAuthIdentity *service.AdminBindAuthIdentityInput

	lastListUsers struct {
		page      int
		pageSize  int
		filters   service.UserListFilters
		sortBy    string
		sortOrder string
		calls     int
	}
	lastListRedeemCodes struct {
		codeType  string
		status    string
		search    string
		sortBy    string
		sortOrder string
		calls     int
	}
}

func newStubAdminService() *stubAdminService {
	now := time.Now().UTC()
	user := service.User{
		ID:        1,
		Email:     "user@example.com",
		Role:      service.RoleUser,
		Status:    service.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	redeem := service.RedeemCode{
		ID:        5,
		Code:      "R-TEST",
		Type:      service.RedeemTypeBalance,
		Value:     10,
		Status:    service.StatusUnused,
		CreatedAt: now,
	}
	return &stubAdminService{
		users:   []service.User{user},
		redeems: []service.RedeemCode{redeem},
	}
}

func (s *stubAdminService) ListUsers(_ context.Context, page, pageSize int, filters service.UserListFilters, sortBy, sortOrder string) ([]service.User, int64, error) {
	s.lastListUsers.page = page
	s.lastListUsers.pageSize = pageSize
	s.lastListUsers.filters = filters
	s.lastListUsers.sortBy = sortBy
	s.lastListUsers.sortOrder = sortOrder
	s.lastListUsers.calls++
	return s.users, int64(len(s.users)), nil
}

func (s *stubAdminService) GetUser(_ context.Context, id int64) (*service.User, error) {
	if s.getUserErr != nil {
		return nil, s.getUserErr
	}
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i], nil
		}
	}
	return &service.User{ID: id, Email: "user@example.com", Role: service.RoleUser, Status: service.StatusActive}, nil
}

func (s *stubAdminService) CreateUser(_ context.Context, input *service.CreateUserInput) (*service.User, error) {
	return &service.User{ID: 100, Email: input.Email, Role: service.RoleUser, Status: service.StatusActive}, nil
}

func (s *stubAdminService) UpdateUser(_ context.Context, id int64, input *service.UpdateUserInput) (*service.User, error) {
	user := service.User{ID: id, Email: input.Email, Role: service.RoleUser, Status: service.StatusActive}
	if input.Username != nil {
		user.Username = *input.Username
	}
	return &user, nil
}

func (s *stubAdminService) DeleteUser(context.Context, int64) error { return nil }

func (s *stubAdminService) UpdateUserBalance(_ context.Context, userID int64, balance float64, _ string, _ string) (*service.User, error) {
	return &service.User{ID: userID, Balance: balance, Role: service.RoleUser, Status: service.StatusActive}, nil
}

func (s *stubAdminService) BatchUpdateConcurrency(_ context.Context, userIDs []int64, _ int, _ string) (int, error) {
	return len(userIDs), nil
}

func (s *stubAdminService) GetUserUsageStats(_ context.Context, userID int64) (*service.AdminUserUsageStats, error) {
	return &service.AdminUserUsageStats{TotalOperations: userID, TotalCharged: float64(userID) / 10}, nil
}

func (s *stubAdminService) GetUserBalanceHistory(context.Context, int64, int, int, string) ([]service.RedeemCode, int64, float64, error) {
	return s.redeems, int64(len(s.redeems)), 100, nil
}

func (s *stubAdminService) BindUserAuthIdentity(_ context.Context, userID int64, input service.AdminBindAuthIdentityInput) (*service.AdminBoundAuthIdentity, error) {
	copied := input
	s.boundAuthIdentity = &copied
	now := time.Now().UTC()
	return &service.AdminBoundAuthIdentity{
		UserID:          userID,
		ProviderType:    input.ProviderType,
		ProviderKey:     input.ProviderKey,
		ProviderSubject: input.ProviderSubject,
		Issuer:          input.Issuer,
		Metadata:        input.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *stubAdminService) ListRedeemCodes(_ context.Context, _ int, _ int, codeType, status, search string, sortBy, sortOrder string) ([]service.RedeemCode, int64, error) {
	s.lastListRedeemCodes.codeType = codeType
	s.lastListRedeemCodes.status = status
	s.lastListRedeemCodes.search = search
	s.lastListRedeemCodes.sortBy = sortBy
	s.lastListRedeemCodes.sortOrder = sortOrder
	s.lastListRedeemCodes.calls++
	return s.redeems, int64(len(s.redeems)), nil
}

func (s *stubAdminService) GetRedeemCode(_ context.Context, id int64) (*service.RedeemCode, error) {
	return &service.RedeemCode{ID: id, Code: "R-TEST", Status: service.StatusUnused}, nil
}

func (s *stubAdminService) GenerateRedeemCodes(context.Context, *service.GenerateRedeemCodesInput) ([]service.RedeemCode, error) {
	return s.redeems, nil
}

func (s *stubAdminService) DeleteRedeemCode(context.Context, int64) error { return nil }

func (s *stubAdminService) BatchDeleteRedeemCodes(_ context.Context, ids []int64) (int64, error) {
	return int64(len(ids)), nil
}

func (s *stubAdminService) ExpireRedeemCode(_ context.Context, id int64) (*service.RedeemCode, error) {
	return &service.RedeemCode{ID: id, Code: "R-TEST", Status: service.StatusExpired}, nil
}
