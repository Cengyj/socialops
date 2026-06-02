package service

import (
	"context"
	"net/http"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

// AdminService is the generic SaaS admin surface used by preserved handlers.
// The concrete implementation is restored in later batches; this skeleton keeps
// route and handler wiring SocialOps-compatible without reintroducing AI account
// or gateway administration.
type AdminService interface {
	ListUsers(ctx context.Context, page, pageSize int, filters UserListFilters, sortBy, sortOrder string) ([]User, int64, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, input *CreateUserInput) (*User, error)
	UpdateUser(ctx context.Context, id int64, input *UpdateUserInput) (*User, error)
	DeleteUser(ctx context.Context, id int64) error
	UpdateUserBalance(ctx context.Context, userID int64, balance float64, operation string, notes string) (*User, error)
	BatchUpdateConcurrency(ctx context.Context, userIDs []int64, value int, mode string) (int, error)
	GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int, sortBy, sortOrder string) ([]APIKey, int64, error)
	GetUserUsageStats(ctx context.Context, userID int64, period string) (any, error)
	GetUserRPMStatus(ctx context.Context, userID int64) (*UserRPMStatus, error)
	GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]RedeemCode, int64, float64, error)
	BindUserAuthIdentity(ctx context.Context, userID int64, input AdminBindAuthIdentityInput) (*AdminBoundAuthIdentity, error)
	ReplaceUserGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (*ReplaceUserGroupResult, error)
	ListRedeemCodes(ctx context.Context, page, pageSize int, codeType, status, search string, sortBy, sortOrder string) ([]RedeemCode, int64, error)
	GetRedeemCode(ctx context.Context, id int64) (*RedeemCode, error)
	GenerateRedeemCodes(ctx context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error)
	DeleteRedeemCode(ctx context.Context, id int64) error
	BatchDeleteRedeemCodes(ctx context.Context, ids []int64) (int64, error)
	ExpireRedeemCode(ctx context.Context, id int64) (*RedeemCode, error)
	AdminUpdateAPIKeyGroupID(ctx context.Context, keyID int64, groupID *int64) (*AdminUpdateAPIKeyGroupIDResult, error)
	AdminResetAPIKeyRateLimitUsage(ctx context.Context, keyID int64) (*APIKey, error)
}

type CreateUserInput struct {
	Email         string
	Password      string
	Username      string
	Notes         string
	Balance       float64
	Concurrency   int
	RPMLimit      int
	AllowedGroups []int64
}

type UpdateUserInput struct {
	Email         string
	Password      string
	Username      *string
	Notes         *string
	Balance       *float64
	Concurrency   *int
	RPMLimit      *int
	Status        string
	AllowedGroups *[]int64
	GroupRates    map[int64]*float64
}

type AdminBindAuthIdentityInput struct {
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
	Issuer          *string
	Metadata        map[string]any
	Channel         *AdminBindAuthIdentityChannelInput
}

type AdminBindAuthIdentityChannelInput struct {
	Channel        string
	ChannelAppID   string
	ChannelSubject string
	Metadata       map[string]any
}

type AdminBoundAuthIdentity struct {
	UserID          int64                          `json:"user_id"`
	ProviderType    string                         `json:"provider_type"`
	ProviderKey     string                         `json:"provider_key"`
	ProviderSubject string                         `json:"provider_subject"`
	VerifiedAt      *time.Time                     `json:"verified_at,omitempty"`
	Issuer          *string                        `json:"issuer,omitempty"`
	Metadata        map[string]any                 `json:"metadata,omitempty"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	Channel         *AdminBoundAuthIdentityChannel `json:"channel,omitempty"`
}

type AdminBoundAuthIdentityChannel struct {
	Channel        string         `json:"channel"`
	ChannelAppID   string         `json:"channel_app_id"`
	ChannelSubject string         `json:"channel_subject"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type AdminUpdateAPIKeyGroupIDResult struct {
	APIKey                 *APIKey
	AutoGrantedGroupAccess bool
	GrantedGroupID         *int64
	GrantedGroupName       string
}

type ReplaceUserGroupResult struct {
	MigratedKeys int64
}

type UserRPMStatus struct {
	UserRPMUsed  int                  `json:"user_rpm_used"`
	UserRPMLimit int                  `json:"user_rpm_limit"`
	PerGroup     []UserGroupRPMStatus `json:"per_group"`
}

type UserGroupRPMStatus struct {
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
	Used      int    `json:"used"`
	Limit     int    `json:"limit"`
	Source    string `json:"source"`
}

type GenerateRedeemCodesInput struct {
	Count        int
	Type         string
	Value        float64
	GroupID      *int64
	ValidityDays int
	ExpiresAt    *time.Time
}

var ErrAdminServiceNotConfigured = infraerrors.New(http.StatusNotImplemented, "ADMIN_SERVICE_NOT_CONFIGURED", "admin operation is not available yet")

type adminServiceSkeleton struct{}

func NewAdminServiceSkeleton() AdminService {
	return &adminServiceSkeleton{}
}

func (s *adminServiceSkeleton) ListUsers(context.Context, int, int, UserListFilters, string, string) ([]User, int64, error) {
	return nil, 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetUser(context.Context, int64) (*User, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) CreateUser(_ context.Context, input *CreateUserInput) (*User, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) UpdateUser(_ context.Context, id int64, input *UpdateUserInput) (*User, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) DeleteUser(context.Context, int64) error {
	return ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) UpdateUserBalance(_ context.Context, userID int64, balance float64, _ string, _ string) (*User, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) BatchUpdateConcurrency(_ context.Context, userIDs []int64, _ int, _ string) (int, error) {
	return 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetUserAPIKeys(context.Context, int64, int, int, string, string) ([]APIKey, int64, error) {
	return nil, 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetUserUsageStats(_ context.Context, userID int64, period string) (any, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetUserRPMStatus(context.Context, int64) (*UserRPMStatus, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetUserBalanceHistory(context.Context, int64, int, int, string) ([]RedeemCode, int64, float64, error) {
	return nil, 0, 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) BindUserAuthIdentity(_ context.Context, userID int64, input AdminBindAuthIdentityInput) (*AdminBoundAuthIdentity, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) ReplaceUserGroup(context.Context, int64, int64, int64) (*ReplaceUserGroupResult, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) ListRedeemCodes(context.Context, int, int, string, string, string, string, string) ([]RedeemCode, int64, error) {
	return nil, 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GetRedeemCode(context.Context, int64) (*RedeemCode, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) GenerateRedeemCodes(_ context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) DeleteRedeemCode(context.Context, int64) error {
	return ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) BatchDeleteRedeemCodes(_ context.Context, ids []int64) (int64, error) {
	return 0, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) ExpireRedeemCode(_ context.Context, id int64) (*RedeemCode, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) AdminUpdateAPIKeyGroupID(_ context.Context, keyID int64, groupID *int64) (*AdminUpdateAPIKeyGroupIDResult, error) {
	return nil, ErrAdminServiceNotConfigured
}

func (s *adminServiceSkeleton) AdminResetAPIKeyRateLimitUsage(_ context.Context, keyID int64) (*APIKey, error) {
	return nil, ErrAdminServiceNotConfigured
}
