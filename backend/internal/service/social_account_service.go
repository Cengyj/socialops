package service

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/predicate"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/socialidentity"
)

// SocialAccountStatus constants
const (
	SocialAccountStatusPendingCheck = "pending_check"
	SocialAccountStatusAvailable    = "available"
	SocialAccountStatusLimited      = "limited"
	SocialAccountStatusInvalid      = "invalid"
	SocialAccountStatusNotStored    = "not_stored"
)

// SocialTaskStatus constants
const (
	SocialTaskStatusPending        = "pending"
	SocialTaskStatusRegistering    = "registering"
	SocialTaskStatusImporting      = "importing"
	SocialTaskStatusParsing        = "parsing"
	SocialTaskStatusStored         = "stored"
	SocialTaskStatusRegisterFailed = "register_failed"
	SocialTaskStatusRiskRejected   = "risk_rejected"
	SocialTaskStatusDuplicate      = "duplicate"
	SocialTaskStatusIPUnavailable  = "ip_unavailable"
	SocialTaskStatusManualReview   = "manual_review"
)

// Social account identity constants.
const (
	SocialAccountIdentityUsername = socialidentity.KindUsername
)

// Social task billing constants.
const (
	SocialTaskUnitPrice                = 0.1
	SocialTaskChargeStatusNotCharged   = "not_charged"
	SocialTaskChargeStatusCharged      = "charged"
	SocialTaskChargeStatusRefunded     = "refunded"
	SocialTaskChargeSourceSubscription = "subscription"
	SocialTaskChargeSourceWallet       = "wallet"
	SocialTaskChargeSourceMixed        = "mixed"
)

var (
	ErrSocialAccountAlreadyAssigned   = infraerrors.Conflict("SOCIAL_ACCOUNT_ALREADY_ASSIGNED", "social account is already assigned")
	ErrSocialAccountAssignmentChanged = infraerrors.Conflict("SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED", "social account assignment changed; retry with the latest state")
	ErrSocialAccountDuplicate         = infraerrors.Conflict("SOCIAL_ACCOUNT_DUPLICATE", "social account already exists in the total pool")
	ErrSocialAccountImportNotFound    = infraerrors.NotFound("SOCIAL_ACCOUNT_POOL_MATCH_NOT_FOUND", "no unassigned total-pool social account matches this platform username")
	ErrSocialAccountImportAmbiguous   = infraerrors.Conflict("SOCIAL_ACCOUNT_POOL_MATCH_AMBIGUOUS", "multiple unassigned total-pool social accounts match this username")
	ErrSocialAccountImportIncomplete  = infraerrors.BadRequest("SOCIAL_ACCOUNT_IMPORT_INCOMPLETE", "account import requires account, password, and 2fa, email, or cookie")
	ErrSocialTaskUnsupportedAction    = infraerrors.BadRequest("SOCIAL_TASK_UNSUPPORTED_ACTION", "unsupported social task action")
	ErrSocialTaskActionUnavailable    = infraerrors.BadRequest("SOCIAL_TASK_ACTION_UNAVAILABLE", "social task action is not connected yet")
	ErrSocialTaskMixedPlatforms       = infraerrors.BadRequest("SOCIAL_TASK_MIXED_PLATFORMS", "selected social accounts must belong to the same platform for one task")
	ErrSocialAccountNotAssigned       = infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_ASSIGNED", "social account is not assigned to this user")
	ErrSocialAccountDefaultProxyRoute = infraerrors.BadRequest("SOCIAL_ACCOUNT_DEFAULT_PROXY_ROUTE_REQUIRED", "use the default proxy endpoint to set an account execution proxy")
	ErrSocialIPOwnerMismatch          = infraerrors.BadRequest("SOCIAL_IP_OWNER_MISMATCH", "social IP does not belong to the account owner")
)

// SocialAccount represents a social media account.
type SocialAccount struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Platform               string     `json:"platform"`
	Username               string     `json:"username,omitempty"`
	IdentityKind           string     `json:"identity_kind"`
	IdentityKey            string     `json:"identity_key"`
	PlatformUserID         *string    `json:"platform_user_id,omitempty"`
	Password               *string    `json:"password,omitempty"`
	Phone                  *string    `json:"phone,omitempty"`
	Email                  *string    `json:"email,omitempty"`
	EmailPassword          *string    `json:"email_password,omitempty"`
	TwoFactor              *string    `json:"two_factor,omitempty"`
	BackupCode             *string    `json:"backup_code,omitempty"`
	EmailClientID          *string    `json:"email_client_id,omitempty"`
	EmailToken             *string    `json:"email_token,omitempty"`
	RegistrationIP         *string    `json:"registration_ip,omitempty"`
	AuthCookie             *string    `json:"auth_cookie,omitempty"`
	ExecutionAuth          *string    `json:"execution_auth,omitempty"`
	AccountStatus          string     `json:"account_status"`
	TaskStatus             string     `json:"task_status"`
	TaskMessage            *string    `json:"task_message,omitempty"`
	DefaultProxySnapshot   *string    `json:"default_proxy_snapshot,omitempty"`
	AssignedUserID         *int64     `json:"assigned_user_id,omitempty"`
	UserWorkbenchDeletedAt *time.Time `json:"user_workbench_deleted_at,omitempty"`
	Remark                 *string    `json:"remark,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// SocialTaskLog represents a task execution log entry.
type SocialTaskLog struct {
	ID               int64      `json:"id"`
	SocialAccountID  int64      `json:"social_account_id"`
	UserID           int64      `json:"user_id"`
	Action           string     `json:"action"`
	Target           *string    `json:"target,omitempty"`
	Content          *string    `json:"content,omitempty"`
	Status           string     `json:"status"`
	ResultMessage    *string    `json:"result_message,omitempty"`
	Price            float64    `json:"price"`
	ChargedAmount    float64    `json:"charged_amount"`
	ChargeStatus     string     `json:"charge_status"`
	ChargeSource     *string    `json:"charge_source,omitempty"`
	ProxyID          *int64     `json:"proxy_id,omitempty"`
	ProxySnapshot    *string    `json:"proxy_snapshot,omitempty"`
	BillingRequestID *string    `json:"billing_request_id,omitempty"`
	IdempotencyKey   *string    `json:"idempotency_key,omitempty"`
	ExecutedAt       *time.Time `json:"executed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreateSocialAccountInput is the input for creating a social account.
type CreateSocialAccountInput struct {
	Name                 string  `json:"name" binding:"required"`
	Platform             string  `json:"platform" binding:"required"`
	PlatformUserID       *string `json:"platform_user_id"`
	Password             *string `json:"password"`
	Phone                *string `json:"phone"`
	Email                *string `json:"email"`
	EmailPassword        *string `json:"email_password"`
	TwoFactor            *string `json:"two_factor"`
	BackupCode           *string `json:"backup_code"`
	EmailClientID        *string `json:"email_client_id"`
	EmailToken           *string `json:"email_token"`
	RegistrationIP       *string `json:"registration_ip"`
	AuthCookie           *string `json:"auth_cookie"`
	ExecutionAuth        *string `json:"execution_auth"`
	DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
	Remark               *string `json:"remark"`
}

// UpdateSocialAccountInput is the input for updating a social account.
type UpdateSocialAccountInput struct {
	Name                 *string `json:"name"`
	PlatformUserID       *string `json:"platform_user_id"`
	Password             *string `json:"password"`
	Phone                *string `json:"phone"`
	Email                *string `json:"email"`
	EmailPassword        *string `json:"email_password"`
	TwoFactor            *string `json:"two_factor"`
	BackupCode           *string `json:"backup_code"`
	EmailClientID        *string `json:"email_client_id"`
	EmailToken           *string `json:"email_token"`
	RegistrationIP       *string `json:"registration_ip"`
	AuthCookie           *string `json:"auth_cookie"`
	ExecutionAuth        *string `json:"execution_auth"`
	AccountStatus        *string `json:"account_status"`
	TaskStatus           *string `json:"task_status"`
	TaskMessage          *string `json:"task_message"`
	DefaultProxySnapshot *string `json:"default_proxy_snapshot"`
	Remark               *string `json:"remark"`
}

// SocialAccountListFilters holds filters for listing social accounts.
type SocialAccountListFilters struct {
	Platform       string
	AccountStatus  string
	TaskStatus     string
	AssignedOnly   bool
	UnassignedOnly bool
	Search         string
	TotalPoolOnly  bool
}

type UserImportSocialAccountInput struct {
	Platform       string  `json:"platform"`
	Name           string  `json:"name" binding:"required"`
	PlatformUserID *string `json:"platform_user_id"`
	Password       *string `json:"password"`
	Phone          *string `json:"phone"`
	Email          *string `json:"email"`
	EmailPassword  *string `json:"email_password"`
	AuthCookie     *string `json:"auth_cookie"`
	ExecutionAuth  *string `json:"execution_auth"`
	TwoFactor      *string `json:"two_factor"`
	BackupCode     *string `json:"backup_code"`
	EmailClientID  *string `json:"email_client_id"`
	EmailToken     *string `json:"email_token"`
	RegistrationIP *string `json:"registration_ip"`
	Remark         *string `json:"remark"`
}

type SocialAccountBatchItemResult struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

type SocialAccountImportResult struct {
	Total      int                            `json:"total"`
	Succeeded  int                            `json:"succeeded"`
	Created    int                            `json:"created"`
	Skipped    int                            `json:"skipped"`
	Failed     int                            `json:"failed"`
	Duplicates int                            `json:"duplicates"`
	Errors     []string                       `json:"errors"`
	Items      []SocialAccountBatchItemResult `json:"items"`
}

type UserSocialAccountImportResult struct {
	Total      int                            `json:"total"`
	Succeeded  int                            `json:"succeeded"`
	Imported   int                            `json:"imported"`
	Skipped    int                            `json:"skipped"`
	Failed     int                            `json:"failed"`
	Duplicates int                            `json:"duplicates"`
	Errors     []string                       `json:"errors"`
	Items      []SocialAccountBatchItemResult `json:"items"`
	Accounts   []*SocialAccount               `json:"accounts"`
}

type UserSocialAccountRemoveResult struct {
	Total     int                            `json:"total"`
	Succeeded int                            `json:"succeeded"`
	Removed   int                            `json:"removed"`
	Skipped   int                            `json:"skipped"`
	Failed    int                            `json:"failed"`
	Errors    []string                       `json:"errors"`
	Items     []SocialAccountBatchItemResult `json:"items"`
}

type SocialAccountBatchResult struct {
	Total      int                            `json:"total"`
	Succeeded  int                            `json:"succeeded"`
	Skipped    int                            `json:"skipped"`
	Failed     int                            `json:"failed"`
	Duplicates int                            `json:"duplicates"`
	Errors     []string                       `json:"errors"`
	Items      []SocialAccountBatchItemResult `json:"items"`
}

const (
	DefaultProxyAssignmentSpecific = "specific"
	DefaultProxyAssignmentRandom   = "random"
	DefaultProxyAssignmentClear    = "clear"
)

type CreateSocialTaskLogInput struct {
	AccountID        int64
	UserID           int64
	Action           string
	Target           *string
	Content          *string
	Status           string
	ResultMessage    *string
	ProxyID          *int64
	ProxySnapshot    *string
	BillingRequestID *string
	IdempotencyKey   *string
}

// SocialAccountService handles social account management.
type SocialAccountService struct {
	entClient *dbent.Client
}

// NewSocialAccountService creates a new SocialAccountService.
func NewSocialAccountService(entClient *dbent.Client) *SocialAccountService {
	return &SocialAccountService{entClient: entClient}
}

// Create creates a new social account.
func (s *SocialAccountService) Create(ctx context.Context, input *CreateSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	name := strings.TrimSpace(input.Name)
	platform := normalizeSocialPlatform(input.Platform)
	if name == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	if platform == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_PLATFORM_REQUIRED", "social account platform is required")
	}
	if err := validateCreateSocialAccountCredentials(input); err != nil {
		return nil, err
	}
	identity := socialAccountBusinessIdentity(platform, name)
	if identity.IdentityKey == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_IDENTITY_REQUIRED", "social account identity is required")
	}
	exists, err := s.poolAccountExists(ctx, identity)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSocialAccountDuplicate
	}
	q := s.entClient.SocialAccount.Create().
		SetName(name).
		SetPlatform(platform).
		SetPlatformKey(identity.PlatformKey).
		SetNameKey(identity.NameKey).
		SetIdentityKind(identity.IdentityKind).
		SetIdentityKey(identity.IdentityKey).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored)

	applyCreateSocialAccountFields(q, input)

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func applyCreateSocialAccountFields(q *dbent.SocialAccountCreate, input *CreateSocialAccountInput) {
	if q == nil || input == nil {
		return
	}
	if value := trimPtr(input.PlatformUserID); value != "" {
		q.SetPlatformUserID(value)
	}
	if input.Password != nil {
		q.SetPassword(*input.Password)
	}
	if input.Phone != nil {
		q.SetPhone(strings.TrimSpace(*input.Phone))
	}
	if input.Email != nil {
		q.SetEmail(strings.TrimSpace(*input.Email))
	}
	if input.EmailPassword != nil {
		q.SetEmailPassword(*input.EmailPassword)
	}
	if value := trimPtr(input.TwoFactor); value != "" {
		q.SetTwoFactor(value)
	}
	if value := trimPtr(input.BackupCode); value != "" {
		q.SetBackupCode(value)
	}
	if value := trimPtr(input.EmailClientID); value != "" {
		q.SetEmailClientID(value)
	}
	if value := trimPtr(input.EmailToken); value != "" {
		q.SetEmailToken(value)
	}
	if value := trimPtr(input.RegistrationIP); value != "" {
		q.SetRegistrationIP(value)
	}
	if value := trimPtr(input.AuthCookie); value != "" {
		q.SetAuthCookie(value)
	}
	if value := trimPtr(input.ExecutionAuth); value != "" {
		q.SetExecutionAuth(value)
	}
	if value := trimPtr(input.DefaultProxySnapshot); value != "" {
		q.SetDefaultProxySnapshot(value)
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}
}

// GetByID returns a social account by ID.
func (s *SocialAccountService) GetByID(ctx context.Context, id int64) (*SocialAccount, error) {
	ent, err := s.entClient.SocialAccount.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// List returns a paginated list of social accounts.
func (s *SocialAccountService) List(ctx context.Context, params pagination.PaginationParams, filters SocialAccountListFilters) ([]*SocialAccount, *pagination.PaginationResult, error) {
	q := s.entClient.SocialAccount.Query()

	if filters.TotalPoolOnly {
		q = q.Where(socialaccount.Not(workbenchStagingAccountPredicate()))
	}
	if platform := normalizeSocialPlatform(filters.Platform); platform != "" {
		q = q.Where(socialaccount.PlatformKeyEQ(platform))
	}
	if filters.AccountStatus != "" {
		q = q.Where(socialaccount.AccountStatusEQ(filters.AccountStatus))
	}
	if filters.TaskStatus != "" {
		q = q.Where(socialaccount.TaskStatusEQ(filters.TaskStatus))
	}
	if filters.AssignedOnly {
		q = q.Where(socialaccount.AssignedUserIDNotNil())
	}
	if filters.UnassignedOnly {
		q = q.Where(socialaccount.AssignedUserIDIsNil())
	}
	if strings.TrimSpace(filters.Search) != "" {
		search := strings.TrimSpace(filters.Search)
		q = q.Where(socialaccount.Or(
			socialaccount.NameContainsFold(search),
			socialaccount.PlatformContainsFold(search),
			socialaccount.PlatformUserIDContainsFold(search),
			socialaccount.IdentityKeyContainsFold(search),
			socialaccount.EmailContainsFold(search),
			socialaccount.PhoneContainsFold(search),
		))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialaccount.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	accounts := make([]*SocialAccount, len(ents))
	for i, e := range ents {
		accounts[i] = socialAccountFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return accounts, result, nil
}

// ListTotalPool returns records that are governed from the admin total account
// pool. User workbench staging imports remain visible from /accounts only until
// an admin explicitly uploads them into the pool.
func (s *SocialAccountService) ListTotalPool(ctx context.Context, params pagination.PaginationParams, filters SocialAccountListFilters) ([]*SocialAccount, *pagination.PaginationResult, error) {
	filters.TotalPoolOnly = true
	return s.List(ctx, params, filters)
}

// Update updates a social account.
func (s *SocialAccountService) Update(ctx context.Context, id int64, input *UpdateSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	current, err := s.entClient.SocialAccount.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	q := s.entClient.SocialAccount.UpdateOneID(id)
	nextName := current.Name
	identityMayChange := false

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
		}
		nextName = name
		identityMayChange = true
	}
	if input.PlatformUserID != nil {
		if value := trimPtr(input.PlatformUserID); value != "" {
			q.SetPlatformUserID(value)
		} else {
			q.ClearPlatformUserID()
		}
	}
	if identityMayChange {
		identity := socialAccountBusinessIdentity(current.PlatformKey, nextName)
		exists, err := s.poolAccountExistsExcept(ctx, identity, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrSocialAccountDuplicate
		}
		q.SetName(nextName).
			SetNameKey(identity.NameKey).
			SetPlatformKey(identity.PlatformKey).
			SetIdentityKind(identity.IdentityKind).
			SetIdentityKey(identity.IdentityKey)
	}
	if input.Password != nil {
		q.SetPassword(*input.Password)
	}
	if input.Phone != nil {
		q.SetPhone(strings.TrimSpace(*input.Phone))
	}
	if input.Email != nil {
		q.SetEmail(strings.TrimSpace(*input.Email))
	}
	if input.EmailPassword != nil {
		q.SetEmailPassword(*input.EmailPassword)
	}
	if input.TwoFactor != nil {
		if value := trimPtr(input.TwoFactor); value != "" {
			q.SetTwoFactor(value)
		} else {
			q.ClearTwoFactor()
		}
	}
	if input.BackupCode != nil {
		if value := trimPtr(input.BackupCode); value != "" {
			q.SetBackupCode(value)
		} else {
			q.ClearBackupCode()
		}
	}
	if input.EmailClientID != nil {
		if value := trimPtr(input.EmailClientID); value != "" {
			q.SetEmailClientID(value)
		} else {
			q.ClearEmailClientID()
		}
	}
	if input.EmailToken != nil {
		if value := trimPtr(input.EmailToken); value != "" {
			q.SetEmailToken(value)
		} else {
			q.ClearEmailToken()
		}
	}
	if input.RegistrationIP != nil {
		if value := trimPtr(input.RegistrationIP); value != "" {
			q.SetRegistrationIP(value)
		} else {
			q.ClearRegistrationIP()
		}
	}
	if input.AuthCookie != nil {
		if value := trimPtr(input.AuthCookie); value != "" {
			q.SetAuthCookie(value)
		} else {
			q.ClearAuthCookie()
		}
	}
	if input.ExecutionAuth != nil {
		if value := trimPtr(input.ExecutionAuth); value != "" {
			q.SetExecutionAuth(value)
		} else {
			q.ClearExecutionAuth()
		}
	}
	if input.AccountStatus != nil {
		q.SetAccountStatus(*input.AccountStatus)
	}
	if input.TaskStatus != nil {
		q.SetTaskStatus(*input.TaskStatus)
	}
	if input.TaskMessage != nil {
		q.SetTaskMessage(*input.TaskMessage)
	}
	if input.DefaultProxySnapshot != nil {
		if trimPtr(input.DefaultProxySnapshot) != "" {
			return nil, ErrSocialAccountDefaultProxyRoute
		}
		q.ClearDefaultProxySnapshot()
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// Delete soft-deletes a social account.
func (s *SocialAccountService) Delete(ctx context.Context, id int64) error {
	affected, err := s.entClient.SocialAccount.Update().
		Where(socialaccount.IDEQ(id), socialaccount.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return err
	}
	if affected == 0 {
		exists, err := s.entClient.SocialAccount.Query().
			Where(socialaccount.IDEQ(id)).
			Exist(mixins.SkipSoftDelete(ctx))
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
	}
	return nil
}

// Assign assigns a social account to a user.
func (s *SocialAccountService) Assign(ctx context.Context, accountID, userID int64) (*SocialAccount, error) {
	current, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	if current.AssignedUserID != nil {
		return nil, ErrSocialAccountAlreadyAssigned
	}
	updated, err := s.entClient.SocialAccount.Update().
		Where(socialaccount.IDEQ(accountID), socialaccount.AssignedUserIDIsNil()).
		SetAssignedUserID(userID).
		SetTaskStatus(SocialTaskStatusStored).
		ClearUserWorkbenchDeletedAt().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		latest, latestErr := s.entClient.SocialAccount.Get(ctx, accountID)
		if latestErr != nil {
			if dbent.IsNotFound(latestErr) {
				return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
			}
			return nil, latestErr
		}
		if latest.AssignedUserID != nil {
			return nil, ErrSocialAccountAlreadyAssigned
		}
		return nil, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// Reclaim removes the user assignment from a social account.
func (s *SocialAccountService) Reclaim(ctx context.Context, accountID int64) (*SocialAccount, error) {
	current, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	if current.AssignedUserID == nil {
		return socialAccountFromEnt(current), nil
	}
	ownerID := *current.AssignedUserID
	updated, err := s.entClient.SocialAccount.Update().
		Where(socialaccount.IDEQ(accountID), socialaccount.AssignedUserIDEQ(ownerID)).
		ClearAssignedUserID().
		ClearDefaultProxySnapshot().
		ClearUserWorkbenchDeletedAt().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		return nil, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func (s *SocialAccountService) BatchAssign(ctx context.Context, accountIDs []int64, userID int64) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	for _, accountID := range accountIDs {
		if accountID <= 0 || userID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_input", "account could not be assigned")
			continue
		}
		account, err := s.Assign(ctx, accountID, userID)
		if err != nil {
			if err == ErrSocialAccountAlreadyAssigned {
				result.Skipped++
				result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "skipped", Reason: "already_assigned", Error: "account is already assigned"})
				continue
			}
			addSocialAccountBatchFailure(result, accountID, "", "assign_failed", err.Error())
			continue
		}
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "succeeded"})
	}
	return result, nil
}

func (s *SocialAccountService) BatchReclaim(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_id", "account could not be reclaimed")
			continue
		}
		account, err := s.Reclaim(ctx, accountID)
		if err != nil {
			addSocialAccountBatchFailure(result, accountID, "", "reclaim_failed", err.Error())
			continue
		}
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "succeeded"})
	}
	return result, nil
}

func (s *SocialAccountService) BatchDelete(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_id", "account could not be deleted")
			continue
		}
		if err := s.Delete(ctx, accountID); err != nil {
			addSocialAccountBatchFailure(result, accountID, "", "delete_failed", err.Error())
			continue
		}
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "succeeded"})
	}
	return result, nil
}

func addSocialAccountBatchFailure(result *SocialAccountBatchResult, id int64, name, reason, message string) {
	if result == nil {
		return
	}
	result.Skipped++
	result.Failed++
	result.Errors = append(result.Errors, message)
	result.Items = append(result.Items, SocialAccountBatchItemResult{ID: id, Name: name, Status: "failed", Reason: reason, Error: message})
}

func addSocialAccountBatchFailedOnly(result *SocialAccountBatchResult, id int64, name, reason, message string) {
	if result == nil {
		return
	}
	result.Failed++
	result.Errors = append(result.Errors, message)
	result.Items = append(result.Items, SocialAccountBatchItemResult{ID: id, Name: name, Status: "failed", Reason: reason, Error: message})
}

func addSocialAccountBatchSkipped(result *SocialAccountBatchResult, id int64, name, reason, message string) {
	if result == nil {
		return
	}
	result.Skipped++
	result.Items = append(result.Items, SocialAccountBatchItemResult{ID: id, Name: name, Status: "skipped", Reason: reason, Error: message})
}

func (s *SocialAccountService) SetDefaultProxyForUser(ctx context.Context, accountID, userID int64, ip *SocialIP) (*SocialAccount, error) {
	account, err := s.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.AssignedUserID == nil || *account.AssignedUserID != userID || account.UserWorkbenchDeletedAt != nil {
		return nil, ErrSocialAccountNotAssigned
	}
	snapshot, err := defaultProxySnapshotForOwner(ip, userID)
	if err != nil {
		return nil, err
	}
	return s.setDefaultProxySnapshot(ctx, accountID, snapshot)
}

func (s *SocialAccountService) BatchSetDefaultProxyForUser(ctx context.Context, userID int64, accountIDs []int64, mode string, ip *SocialIP, pool []*SocialIP) (*SocialAccountBatchResult, error) {
	accountIDs = UniqueInt64sPreserveOrder(accountIDs)
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = DefaultProxyAssignmentSpecific
	}
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	if len(accountIDs) == 0 {
		return result, nil
	}

	var specificSnapshot *string
	var specificProxyUnavailable bool
	if mode == DefaultProxyAssignmentSpecific {
		snapshot, err := defaultProxySnapshotForOwner(ip, userID)
		if err != nil {
			if err == ErrSocialIPOwnerMismatch {
				return nil, err
			}
			specificProxyUnavailable = true
		} else if snapshot == nil {
			return nil, infraerrors.BadRequest("SOCIAL_IP_REQUIRED", "proxy is required for this assignment")
		} else {
			specificSnapshot = snapshot
		}
	}

	randomSnapshots := make([]*string, 0)
	if mode == DefaultProxyAssignmentRandom {
		shuffled := shuffleSocialIPs(pool)
		for _, candidate := range shuffled {
			snapshot, err := defaultProxySnapshotForOwner(candidate, userID)
			if err != nil {
				continue
			}
			randomSnapshots = append(randomSnapshots, snapshot)
		}
		if len(randomSnapshots) == 0 {
			return nil, infraerrors.BadRequest("SOCIAL_IP_POOL_EMPTY", "no online proxy is available for assignment")
		}
	}
	if mode != DefaultProxyAssignmentSpecific && mode != DefaultProxyAssignmentRandom && mode != DefaultProxyAssignmentClear {
		return nil, infraerrors.BadRequest("SOCIAL_IP_ASSIGNMENT_MODE_INVALID", "proxy assignment mode is invalid")
	}

	for index, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchFailedOnly(result, accountID, "", "invalid_id", "account proxy could not be assigned")
			continue
		}
		account, err := s.GetByID(ctx, accountID)
		if err != nil {
			addSocialAccountBatchFailedOnly(result, accountID, "", "account_not_found", "account proxy could not be assigned")
			continue
		}
		if account.AssignedUserID == nil || *account.AssignedUserID != userID || account.UserWorkbenchDeletedAt != nil {
			addSocialAccountBatchFailedOnly(result, accountID, account.Name, "account_not_visible", "account proxy could not be assigned")
			continue
		}
		var snapshot *string
		switch mode {
		case DefaultProxyAssignmentSpecific:
			if specificProxyUnavailable {
				addSocialAccountBatchFailedOnly(result, accountID, account.Name, "proxy_not_available", "account proxy could not be assigned")
				continue
			}
			snapshot = specificSnapshot
		case DefaultProxyAssignmentRandom:
			snapshot = randomSnapshots[index%len(randomSnapshots)]
		case DefaultProxyAssignmentClear:
			snapshot = nil
		}
		if _, err := s.setDefaultProxySnapshot(ctx, accountID, snapshot); err != nil {
			addSocialAccountBatchFailedOnly(result, accountID, account.Name, "assign_failed", "account proxy could not be assigned")
			continue
		}
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Name: account.Name, Status: "succeeded"})
	}
	return result, nil
}

func shuffleSocialIPs(ips []*SocialIP) []*SocialIP {
	if len(ips) < 2 {
		return ips
	}
	shuffled := append([]*SocialIP(nil), ips...)
	seed := time.Now().UnixNano()
	for i := len(shuffled) - 1; i > 0; i-- {
		seed = seed*1664525 + 1013904223
		j := int(seed % int64(i+1))
		if j < 0 {
			j = -j
		}
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

func (s *SocialAccountService) UpdateForUser(ctx context.Context, accountID, userID int64, input *UpdateSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	current, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	if current.AssignedUserID == nil || int64(*current.AssignedUserID) != userID || current.UserWorkbenchDeletedAt != nil {
		return nil, ErrSocialAccountNotAssigned
	}
	next := socialAccountFromEnt(current)
	q := s.entClient.SocialAccount.UpdateOneID(accountID)

	if input.Password != nil {
		value := strings.TrimSpace(*input.Password)
		if value == "" {
			return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_PASSWORD_REQUIRED", "social account password is required")
		}
		q.SetPassword(value)
		next.Password = &value
	}
	if input.Phone != nil {
		setOptionalString(input.Phone, func(value string) {
			q.SetPhone(value)
			next.Phone = &value
		}, func() {
			q.ClearPhone()
			next.Phone = nil
		})
	}
	if input.Email != nil {
		setOptionalString(input.Email, func(value string) {
			q.SetEmail(value)
			next.Email = &value
		}, func() {
			q.ClearEmail()
			next.Email = nil
		})
	}
	if input.EmailPassword != nil {
		setOptionalString(input.EmailPassword, func(value string) {
			q.SetEmailPassword(value)
			next.EmailPassword = &value
		}, func() {
			q.ClearEmailPassword()
			next.EmailPassword = nil
		})
	}
	if input.TwoFactor != nil {
		setOptionalString(input.TwoFactor, func(value string) {
			q.SetTwoFactor(value)
			next.TwoFactor = &value
		}, func() {
			q.ClearTwoFactor()
			next.TwoFactor = nil
		})
	}
	if input.BackupCode != nil {
		setOptionalString(input.BackupCode, func(value string) {
			q.SetBackupCode(value)
			next.BackupCode = &value
		}, func() {
			q.ClearBackupCode()
			next.BackupCode = nil
		})
	}
	if input.EmailClientID != nil {
		setOptionalString(input.EmailClientID, func(value string) {
			q.SetEmailClientID(value)
			next.EmailClientID = &value
		}, func() {
			q.ClearEmailClientID()
			next.EmailClientID = nil
		})
	}
	if input.EmailToken != nil {
		setOptionalString(input.EmailToken, func(value string) {
			q.SetEmailToken(value)
			next.EmailToken = &value
		}, func() {
			q.ClearEmailToken()
			next.EmailToken = nil
		})
	}
	if input.RegistrationIP != nil {
		setOptionalString(input.RegistrationIP, func(value string) {
			q.SetRegistrationIP(value)
			next.RegistrationIP = &value
		}, func() {
			q.ClearRegistrationIP()
			next.RegistrationIP = nil
		})
	}
	if input.AuthCookie != nil {
		setOptionalString(input.AuthCookie, func(value string) {
			q.SetAuthCookie(value)
			next.AuthCookie = &value
		}, func() {
			q.ClearAuthCookie()
			next.AuthCookie = nil
		})
	}
	if input.ExecutionAuth != nil {
		setOptionalString(input.ExecutionAuth, func(value string) {
			q.SetExecutionAuth(value)
			next.ExecutionAuth = &value
		}, func() {
			q.ClearExecutionAuth()
			next.ExecutionAuth = nil
		})
	}
	if input.Remark != nil {
		setOptionalString(input.Remark, func(value string) {
			q.SetRemark(value)
			next.Remark = &value
		}, func() {
			q.ClearRemark()
			next.Remark = nil
		})
	}
	if err := validateMutableSocialAccountCredentials(next); err != nil {
		return nil, err
	}
	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func defaultProxySnapshotForOwner(ip *SocialIP, ownerID int64) (*string, error) {
	if ip == nil {
		return nil, nil
	}
	if ip.UserID != ownerID {
		return nil, ErrSocialIPOwnerMismatch
	}
	if ip.Status != SocialIPStatusOnline {
		return nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP must pass a connectivity test before execution")
	}
	if strings.TrimSpace(stringValue(ip.Endpoint)) == "" {
		return nil, infraerrors.BadRequest("SOCIAL_IP_NOT_AVAILABLE", "social IP endpoint is required for execution")
	}
	snapshot := SocialIPTaskSnapshot(ip)
	return &snapshot, nil
}

func (s *SocialAccountService) setDefaultProxySnapshot(ctx context.Context, accountID int64, snapshot *string) (*SocialAccount, error) {
	q := s.entClient.SocialAccount.UpdateOneID(accountID)
	if snapshot == nil || strings.TrimSpace(*snapshot) == "" {
		q.ClearDefaultProxySnapshot()
	} else {
		trimmed := strings.TrimSpace(*snapshot)
		q.SetDefaultProxySnapshot(trimmed)
	}
	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// ImportPoolAccounts imports admin-provided accounts into the total pool and
// skips duplicates by normalized platform plus username.
func (s *SocialAccountService) ImportPoolAccounts(ctx context.Context, inputs []*CreateSocialAccountInput) (*SocialAccountImportResult, error) {
	result := &SocialAccountImportResult{Total: len(inputs)}
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input == nil {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, "nil input")
			result.Items = append(result.Items, SocialAccountBatchItemResult{Status: "failed", Reason: "invalid_input", Error: "nil input"})
			continue
		}
		platform := normalizeSocialPlatform(input.Platform)
		name := normalizeSocialUsername(input.Name)
		if platform == "" || name == "" {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, "missing platform or name")
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: "invalid_input", Error: "missing platform or name"})
			continue
		}
		if err := validateCreateSocialAccountCredentials(input); err != nil {
			result.Skipped++
			result.Failed++
			message := "account import requires account, password, and 2fa, email, or cookie"
			result.Errors = append(result.Errors, message)
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: "invalid_input", Error: message})
			continue
		}
		identity := socialAccountBusinessIdentity(platform, input.Name)
		if key := socialAccountDedupKeyFromIdentity(identity); key != "" {
			if _, ok := seen[key]; ok {
				result.Skipped++
				result.Duplicates++
				message := "duplicate account in import batch"
				result.Errors = append(result.Errors, message)
				result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "duplicate", Reason: "duplicate_in_batch", Error: message})
				continue
			}
			seen[key] = struct{}{}
		}
		exists, err := s.poolAccountExists(ctx, identity)
		if err != nil {
			return result, err
		}
		if exists {
			result.Skipped++
			result.Duplicates++
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "duplicate", Reason: "duplicate_in_database", Error: "duplicate account in total pool"})
			continue
		}
		input.Platform = platform
		account, err := s.createPoolAccount(ctx, input)
		if err != nil {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, strings.TrimSpace(input.Name)+": "+err.Error())
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: "create_failed", Error: err.Error()})
			continue
		}
		result.Created++
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "succeeded"})
	}
	return result, nil
}

func (s *SocialAccountService) createPoolAccount(ctx context.Context, input *CreateSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	if err := validateCreateSocialAccountCredentials(input); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	platform := normalizeSocialPlatform(input.Platform)
	identity := socialAccountBusinessIdentity(platform, name)
	q := s.entClient.SocialAccount.Create().
		SetName(name).
		SetPlatform(platform).
		SetPlatformKey(identity.PlatformKey).
		SetNameKey(identity.NameKey).
		SetIdentityKind(identity.IdentityKind).
		SetIdentityKey(identity.IdentityKey).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored)

	applyCreateSocialAccountFields(q, input)

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// importUserWorkbenchAccount applies one batch-import row by binding an
// existing total-pool account or creating a not_stored workbench staging record.
func (s *SocialAccountService) importUserWorkbenchAccount(ctx context.Context, userID int64, input *UserImportSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_IMPORT_REQUIRED", "social account import input is required")
	}
	if err := validateUserImportCredentials(input); err != nil {
		return nil, err
	}
	platform := normalizeSocialPlatform(input.Platform)
	name := normalizeSocialUsername(input.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	identity := socialAccountBusinessIdentity(platform, input.Name)

	allMatches, err := s.findAccountsByBusinessIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}
	if len(allMatches) == 0 {
		return s.createUserWorkbenchAccount(ctx, userID, input, identity)
	}
	if platform == "" && len(allMatches) > 1 {
		if restored, ok, err := s.restorePlatformlessUserWorkbenchAccount(ctx, userID, allMatches); ok || err != nil {
			return restored, err
		}
		return nil, ErrSocialAccountImportAmbiguous
	}

	matches := make([]*dbent.SocialAccount, 0, len(allMatches))
	for _, account := range allMatches {
		if account.AssignedUserID != nil && *account.AssignedUserID == userID {
			if account.UserWorkbenchDeletedAt == nil {
				return nil, ErrSocialAccountAlreadyAssigned
			}
			return s.restoreUserWorkbenchAccount(ctx, userID, account.ID)
		}
		if account.AssignedUserID == nil {
			matches = append(matches, account)
		}
	}
	if len(matches) == 0 {
		return nil, ErrSocialAccountAlreadyAssigned
	}
	if len(matches) > 1 {
		return nil, ErrSocialAccountImportAmbiguous
	}
	return s.Assign(ctx, matches[0].ID, userID)
}

func (s *SocialAccountService) createUserWorkbenchAccount(ctx context.Context, userID int64, input *UserImportSocialAccountInput, identity socialidentity.BusinessIdentity) (*SocialAccount, error) {
	if userID <= 0 {
		return nil, ErrSocialAccountNotAssigned
	}
	if identity.NameKey == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	if identity.PlatformKey == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_PLATFORM_REQUIRED", "social account platform is required")
	}
	displayName := strings.TrimSpace(input.Name)
	if displayName == "" {
		displayName = identity.NameKey
	}
	ent := s.entClient.SocialAccount.Create().
		SetName(displayName).
		SetPlatform(identity.PlatformKey).
		SetPlatformKey(identity.PlatformKey).
		SetNameKey(identity.NameKey).
		SetIdentityKind(identity.IdentityKind).
		SetIdentityKey(identity.IdentityKey).
		SetAssignedUserID(userID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending)
	applyUserImportFields(ent, input)
	saved, err := ent.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(saved), nil
}

func (s *SocialAccountService) restorePlatformlessUserWorkbenchAccount(ctx context.Context, userID int64, allMatches []*dbent.SocialAccount) (*SocialAccount, bool, error) {
	var restorable *dbent.SocialAccount
	for _, account := range allMatches {
		if account.AssignedUserID == nil {
			return nil, false, nil
		}
		if *account.AssignedUserID != userID {
			continue
		}
		if account.UserWorkbenchDeletedAt == nil {
			return nil, false, nil
		}
		if restorable != nil {
			return nil, false, nil
		}
		restorable = account
	}
	if restorable == nil {
		return nil, false, nil
	}
	account, err := s.restoreUserWorkbenchAccount(ctx, userID, restorable.ID)
	return account, true, err
}

func (s *SocialAccountService) restoreUserWorkbenchAccount(ctx context.Context, userID, accountID int64) (*SocialAccount, error) {
	restored, err := s.entClient.SocialAccount.Update().
		Where(
			socialaccount.IDEQ(accountID),
			socialaccount.AssignedUserIDEQ(userID),
		).
		ClearUserWorkbenchDeletedAt().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if restored == 0 {
		return nil, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func (s *SocialAccountService) BatchImportForUser(ctx context.Context, userID int64, inputs []*UserImportSocialAccountInput) (*UserSocialAccountImportResult, error) {
	result := &UserSocialAccountImportResult{Total: len(inputs)}
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input == nil {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchImportErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{Status: "failed", Reason: "invalid_input", Error: userAccountBatchImportErrorMessage()})
			continue
		}
		if err := validateUserImportCredentials(input); err != nil {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchImportErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: "invalid_input", Error: userAccountBatchImportErrorMessage()})
			continue
		}
		platform := normalizeSocialPlatform(input.Platform)
		identity := socialAccountBusinessIdentity(platform, input.Name)
		if key := socialAccountDedupKeyFromIdentity(identity); key != "" {
			if _, ok := seen[key]; ok {
				result.Skipped++
				result.Duplicates++
				result.Errors = append(result.Errors, userAccountBatchImportErrorMessage())
				result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "duplicate", Reason: "duplicate_in_batch", Error: userAccountBatchImportErrorMessage()})
				continue
			}
			seen[key] = struct{}{}
		}
		account, err := s.importUserWorkbenchAccount(ctx, userID, input)
		if err != nil {
			result.Skipped++
			if err == ErrSocialAccountDuplicate || err == ErrSocialAccountAlreadyAssigned {
				result.Duplicates++
				result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "duplicate", Reason: "duplicate_in_database", Error: userAccountBatchImportErrorMessage()})
			} else {
				result.Failed++
				result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: "import_failed", Error: userAccountBatchImportErrorMessage()})
			}
			result.Errors = append(result.Errors, userAccountBatchImportErrorMessage())
			continue
		}
		result.Imported++
		result.Succeeded++
		result.Accounts = append(result.Accounts, account)
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "succeeded"})
	}
	return result, nil
}

func validateUserImportCredentials(input *UserImportSocialAccountInput) error {
	if input == nil {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_IMPORT_REQUIRED", "social account import input is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	if trimPtr(input.Password) == "" {
		return ErrSocialAccountImportIncomplete
	}
	hasTwoFactor := trimPtr(input.TwoFactor) != ""
	hasAuthCookie := trimPtr(input.AuthCookie) != ""
	hasEmail := trimPtr(input.Email) != "" && (trimPtr(input.EmailPassword) != "" || trimPtr(input.EmailToken) != "")
	if !hasTwoFactor && !hasEmail && !hasAuthCookie {
		return ErrSocialAccountImportIncomplete
	}
	return nil
}

func validateCreateSocialAccountCredentials(input *CreateSocialAccountInput) error {
	if input == nil {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	if trimPtr(input.Password) == "" {
		return ErrSocialAccountImportIncomplete
	}
	hasTwoFactor := trimPtr(input.TwoFactor) != ""
	hasAuthCookie := trimPtr(input.AuthCookie) != ""
	hasEmail := trimPtr(input.Email) != "" && (trimPtr(input.EmailPassword) != "" || trimPtr(input.EmailToken) != "")
	if !hasTwoFactor && !hasEmail && !hasAuthCookie {
		return ErrSocialAccountImportIncomplete
	}
	return nil
}

func validateMutableSocialAccountCredentials(account *SocialAccount) error {
	if account == nil {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	if trimPtr(account.Password) == "" {
		return infraerrors.BadRequest("SOCIAL_ACCOUNT_PASSWORD_REQUIRED", "social account password is required")
	}
	hasTwoFactor := trimPtr(account.TwoFactor) != ""
	hasAuthCookie := trimPtr(account.AuthCookie) != ""
	hasEmail := trimPtr(account.Email) != "" && (trimPtr(account.EmailPassword) != "" || trimPtr(account.EmailToken) != "")
	if !hasTwoFactor && !hasEmail && !hasAuthCookie {
		return ErrSocialAccountImportIncomplete
	}
	return nil
}

func setOptionalString(value *string, set func(string), clear func()) {
	if value == nil {
		return
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		clear()
		return
	}
	set(trimmed)
}

func applyUserImportFields(q *dbent.SocialAccountCreate, input *UserImportSocialAccountInput) {
	if q == nil || input == nil {
		return
	}
	if value := trimPtr(input.PlatformUserID); value != "" {
		q.SetPlatformUserID(value)
	}
	if value := trimPtr(input.Password); value != "" {
		q.SetPassword(value)
	}
	if value := trimPtr(input.Phone); value != "" {
		q.SetPhone(value)
	}
	if value := trimPtr(input.Email); value != "" {
		q.SetEmail(value)
	}
	if value := trimPtr(input.EmailPassword); value != "" {
		q.SetEmailPassword(value)
	}
	if value := trimPtr(input.TwoFactor); value != "" {
		q.SetTwoFactor(value)
	}
	if value := trimPtr(input.BackupCode); value != "" {
		q.SetBackupCode(value)
	}
	if value := trimPtr(input.EmailClientID); value != "" {
		q.SetEmailClientID(value)
	}
	if value := trimPtr(input.EmailToken); value != "" {
		q.SetEmailToken(value)
	}
	if value := trimPtr(input.RegistrationIP); value != "" {
		q.SetRegistrationIP(value)
	}
	if value := trimPtr(input.AuthCookie); value != "" {
		q.SetAuthCookie(value)
	}
	if value := trimPtr(input.ExecutionAuth); value != "" {
		q.SetExecutionAuth(value)
	}
	if value := trimPtr(input.Remark); value != "" {
		q.SetRemark(value)
	}
}

func trimPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstTrimmed(values ...*string) string {
	for _, value := range values {
		if trimmed := trimPtr(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// ListByUser returns social accounts assigned to a specific user.
func (s *SocialAccountService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]*SocialAccount, *pagination.PaginationResult, error) {
	q := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.AssignedUserIDEQ(int64(userID)),
			socialaccount.UserWorkbenchDeletedAtIsNil(),
		)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialaccount.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	accounts := make([]*SocialAccount, len(ents))
	for i, e := range ents {
		accounts[i] = socialAccountFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return accounts, result, nil
}

// RemoveFromUserWorkbench hides an assigned account from a user's workbench
// without deleting, reclaiming, or reassigning the total-pool account.
func (s *SocialAccountService) RemoveFromUserWorkbench(ctx context.Context, userID, accountID int64) error {
	account, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return err
	}
	if account.AssignedUserID == nil || *account.AssignedUserID != userID {
		return ErrSocialAccountNotAssigned
	}
	if account.UserWorkbenchDeletedAt != nil {
		return nil
	}
	updated, err := s.entClient.SocialAccount.Update().
		Where(
			socialaccount.IDEQ(accountID),
			socialaccount.AssignedUserIDEQ(userID),
			socialaccount.UserWorkbenchDeletedAtIsNil(),
		).
		SetUserWorkbenchDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if updated == 0 {
		latest, latestErr := s.entClient.SocialAccount.Get(ctx, accountID)
		if latestErr != nil {
			if dbent.IsNotFound(latestErr) {
				return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
			}
			return latestErr
		}
		if latest.AssignedUserID != nil && *latest.AssignedUserID == userID && latest.UserWorkbenchDeletedAt != nil {
			return nil
		}
		return ErrSocialAccountAssignmentChanged
	}
	return nil
}

func (s *SocialAccountService) BatchRemoveFromUserWorkbench(ctx context.Context, userID int64, accountIDs []int64) (*UserSocialAccountRemoveResult, error) {
	accountIDs = UniqueInt64sPreserveOrder(accountIDs)
	result := &UserSocialAccountRemoveResult{Total: len(accountIDs)}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchDeleteErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "failed", Reason: "invalid_id", Error: userAccountBatchDeleteErrorMessage()})
			continue
		}
		visible, err := s.entClient.SocialAccount.Query().
			Where(
				socialaccount.IDEQ(accountID),
				socialaccount.AssignedUserIDEQ(userID),
				socialaccount.UserWorkbenchDeletedAtIsNil(),
			).
			Exist(ctx)
		if err != nil {
			return result, err
		}
		if !visible {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchDeleteErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "failed", Reason: "not_visible", Error: userAccountBatchDeleteErrorMessage()})
			continue
		}
		if err := s.RemoveFromUserWorkbench(ctx, userID, accountID); err != nil {
			result.Skipped++
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchDeleteErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "failed", Reason: "delete_failed", Error: userAccountBatchDeleteErrorMessage()})
			continue
		}
		result.Removed++
		result.Succeeded++
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "succeeded"})
	}
	return result, nil
}

func userAccountBatchImportErrorMessage() string {
	return "account could not be imported"
}

func userAccountBatchDeleteErrorMessage() string {
	return "account could not be deleted"
}

// CreateTaskLog creates a task execution log entry.
func (s *SocialAccountService) CreateTaskLog(ctx context.Context, input *CreateSocialTaskLogInput) (*SocialTaskLog, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = SocialTaskLogStatusPending
	}
	ent, err := s.entClient.SocialTaskLog.Create().
		SetSocialAccountID(input.AccountID).
		SetUserID(input.UserID).
		SetAction(input.Action).
		SetStatus(status).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetNillableTarget(input.Target).
		SetNillableContent(input.Content).
		SetNillableResultMessage(input.ResultMessage).
		SetNillableProxyID(input.ProxyID).
		SetNillableProxySnapshot(input.ProxySnapshot).
		SetNillableBillingRequestID(input.BillingRequestID).
		SetNillableIdempotencyKey(input.IdempotencyKey).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) && input.IdempotencyKey != nil {
			existing, findErr := s.FindTaskLogByIdempotency(ctx, input.UserID, input.AccountID, input.Action, *input.IdempotencyKey)
			if findErr != nil {
				return nil, findErr
			}
			if existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return socialTaskLogFromEnt(ent), nil
}

func (s *SocialAccountService) MarkTaskLogFailedNotCharged(ctx context.Context, taskLogID int64, message string) (*SocialTaskLog, error) {
	now := time.Now()
	ent, err := s.entClient.SocialTaskLog.UpdateOneID(taskLogID).
		SetStatus(SocialTaskLogStatusFailed).
		SetResultMessage(message).
		SetExecutedAt(now).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		ClearChargeSource().
		ClearBillingRequestID().
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_TASK_LOG_NOT_FOUND", "social task log not found")
		}
		return nil, err
	}
	return socialTaskLogFromEnt(ent), nil
}

func (s *SocialAccountService) FindTaskLogByIdempotency(ctx context.Context, userID, accountID int64, action, key string) (*SocialTaskLog, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	ent, err := s.entClient.SocialTaskLog.Query().
		Where(
			socialtasklog.UserIDEQ(userID),
			socialtasklog.SocialAccountIDEQ(accountID),
			socialtasklog.ActionEQ(action),
			socialtasklog.IdempotencyKeyEQ(key),
		).
		Order(dbent.Desc(socialtasklog.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return socialTaskLogFromEnt(ent), nil
}

// GetStats returns summary statistics for social accounts.
func (s *SocialAccountService) GetStats(ctx context.Context) (map[string]int, error) {
	total, _ := s.entClient.SocialAccount.Query().
		Where(socialaccount.Not(workbenchStagingAccountPredicate())).Count(ctx)
	stored, _ := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.Not(workbenchStagingAccountPredicate()),
			socialaccount.TaskStatusEQ(SocialTaskStatusStored),
		).Count(ctx)
	available, _ := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.Not(workbenchStagingAccountPredicate()),
			socialaccount.AccountStatusEQ(SocialAccountStatusAvailable),
		).Count(ctx)

	return map[string]int{
		"total":     total,
		"stored":    stored,
		"available": available,
	}, nil
}

func workbenchStagingAccountPredicate() predicate.SocialAccount {
	return socialaccount.And(
		socialaccount.AssignedUserIDNotNil(),
		socialaccount.AccountStatusEQ(SocialAccountStatusNotStored),
		socialaccount.TaskStatusEQ(SocialTaskStatusPending),
	)
}

func (s *SocialAccountService) poolAccountExists(ctx context.Context, identity socialidentity.BusinessIdentity) (bool, error) {
	return s.poolAccountExistsExcept(ctx, identity, 0)
}

func (s *SocialAccountService) poolAccountExistsExcept(ctx context.Context, identity socialidentity.BusinessIdentity, exceptID int64) (bool, error) {
	accounts, err := s.findAccountsByBusinessIdentity(ctx, identity)
	if err != nil {
		return false, err
	}
	for _, account := range accounts {
		if exceptID > 0 && account.ID == exceptID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *SocialAccountService) totalPoolAccountExistsExcept(ctx context.Context, identity socialidentity.BusinessIdentity, exceptID int64) (bool, error) {
	if identity.IdentityKey == "" {
		return false, nil
	}
	q := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.PlatformKeyEQ(identity.PlatformKey),
			socialaccount.NameKeyEQ(identity.NameKey),
			socialaccount.Not(workbenchStagingAccountPredicate()),
		)
	if exceptID > 0 {
		q = q.Where(socialaccount.IDNEQ(exceptID))
	}
	return q.Exist(ctx)
}

func (s *SocialAccountService) findAccountsByBusinessIdentity(ctx context.Context, identity socialidentity.BusinessIdentity) ([]*dbent.SocialAccount, error) {
	if identity.IdentityKey == "" {
		return nil, nil
	}
	q := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.NameKeyEQ(identity.NameKey),
		)
	if identity.PlatformKey != "" {
		q = q.Where(socialaccount.PlatformKeyEQ(identity.PlatformKey))
	}
	accounts, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func normalizeSocialPlatform(platform string) string {
	return socialidentity.NormalizePlatform(platform)
}

func normalizeSocialUsername(name string) string {
	return socialidentity.NormalizeUsername(name)
}

func socialAccountBusinessIdentity(platform, name string) socialidentity.BusinessIdentity {
	return socialidentity.Build(platform, name)
}

func socialAccountDedupKeyFromIdentity(identity socialidentity.BusinessIdentity) string {
	if identity.PlatformKey == "" || identity.IdentityKey == "" {
		return ""
	}
	return identity.PlatformKey + "\x00" + identity.IdentityKind + "\x00" + identity.IdentityKey
}

func socialAccountDedupKey(platform, name string, platformUserID *string) string {
	return socialidentity.DedupKey(platform, name)
}

func NormalizeSocialTaskAction(action string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case SocialTaskActionLoginCheck:
		return SocialTaskActionLoginCheck, true
	case SocialTaskActionFollow:
		return SocialTaskActionFollow, true
	case SocialTaskActionMessage, legacySocialTaskActionDM:
		return SocialTaskActionMessage, true
	case SocialTaskActionPost, legacySocialTaskActionTweet:
		return SocialTaskActionPost, true
	case SocialTaskActionLike:
		return SocialTaskActionLike, true
	case SocialTaskActionRetweet:
		return SocialTaskActionRetweet, true
	default:
		return "", false
	}
}

func IsBillableSocialTaskAction(action string) bool {
	_, ok := NormalizeSocialTaskAction(action)
	return ok
}

func EnsureExecutableSocialTaskAction(action string) error {
	normalized, ok := NormalizeSocialTaskAction(action)
	if !ok {
		return ErrSocialTaskUnsupportedAction
	}
	if normalized == SocialTaskActionMessage {
		return ErrSocialTaskActionUnavailable
	}
	return nil
}

func socialAccountFromEnt(e *dbent.SocialAccount) *SocialAccount {
	platform := normalizeSocialPlatform(e.Platform)
	if platform == "" {
		platform = e.Platform
	}
	a := &SocialAccount{
		ID:            int64(e.ID),
		Name:          e.Name,
		Platform:      platform,
		Username:      normalizeSocialUsername(e.NameKey),
		IdentityKind:  e.IdentityKind,
		IdentityKey:   e.IdentityKey,
		AccountStatus: e.AccountStatus,
		TaskStatus:    e.TaskStatus,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
	if a.Username == "" {
		a.Username = normalizeSocialUsername(e.Name)
	}
	if value := trimPtr(e.PlatformUserID); value != "" {
		a.PlatformUserID = &value
	}
	if e.Password != nil {
		a.Password = e.Password
	}
	if e.Phone != nil {
		a.Phone = e.Phone
	}
	if e.Email != nil {
		a.Email = e.Email
	}
	if e.EmailPassword != nil {
		a.EmailPassword = e.EmailPassword
	}
	if e.TwoFactor != nil {
		a.TwoFactor = e.TwoFactor
	}
	if e.BackupCode != nil {
		a.BackupCode = e.BackupCode
	}
	if e.EmailClientID != nil {
		a.EmailClientID = e.EmailClientID
	}
	if e.EmailToken != nil {
		a.EmailToken = e.EmailToken
	}
	if e.RegistrationIP != nil {
		a.RegistrationIP = e.RegistrationIP
	}
	if value := trimPtr(e.AuthCookie); value != "" {
		a.AuthCookie = &value
	}
	if value := trimPtr(e.ExecutionAuth); value != "" {
		a.ExecutionAuth = &value
	}
	if e.TaskMessage != nil {
		a.TaskMessage = e.TaskMessage
	}
	if value := trimPtr(e.DefaultProxySnapshot); value != "" {
		a.DefaultProxySnapshot = &value
	}
	if e.AssignedUserID != nil {
		uid := int64(*e.AssignedUserID)
		a.AssignedUserID = &uid
	}
	if e.UserWorkbenchDeletedAt != nil {
		a.UserWorkbenchDeletedAt = e.UserWorkbenchDeletedAt
	}
	if e.Remark != nil {
		a.Remark = e.Remark
	}
	return a
}

func socialTaskLogFromEnt(e *dbent.SocialTaskLog) *SocialTaskLog {
	l := &SocialTaskLog{
		ID:              int64(e.ID),
		SocialAccountID: int64(e.SocialAccountID),
		UserID:          int64(e.UserID),
		Action:          e.Action,
		Status:          e.Status,
		Price:           e.Price,
		ChargedAmount:   e.ChargedAmount,
		ChargeStatus:    e.ChargeStatus,
		CreatedAt:       e.CreatedAt,
	}
	if e.Target != nil {
		l.Target = e.Target
	}
	if e.Content != nil {
		l.Content = e.Content
	}
	if e.ResultMessage != nil {
		l.ResultMessage = e.ResultMessage
	}
	if e.ChargeSource != nil {
		l.ChargeSource = e.ChargeSource
	}
	if e.ProxyID != nil {
		id := int64(*e.ProxyID)
		l.ProxyID = &id
	}
	if e.ProxySnapshot != nil {
		l.ProxySnapshot = e.ProxySnapshot
	}
	if e.BillingRequestID != nil {
		l.BillingRequestID = e.BillingRequestID
	}
	if e.IdempotencyKey != nil {
		l.IdempotencyKey = e.IdempotencyKey
	}
	if e.ExecutedAt != nil {
		l.ExecutedAt = e.ExecutedAt
	}
	return l
}
