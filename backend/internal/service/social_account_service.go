package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/predicate"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
	"github.com/Wei-Shaw/socialops/ent/user"
	"github.com/Wei-Shaw/socialops/internal/domain"
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

const (
	userImportReasonMatchedTotalPool   = "matched_total_pool"
	userImportReasonStagedNotStored    = "staged_not_stored"
	userImportReasonInvalidInput       = "invalid_input"
	userImportReasonDuplicateInBatch   = "duplicate_in_batch"
	userImportReasonDuplicateInPool    = "duplicate_in_database"
	userImportReasonAlreadyInWorkbench = "already_in_workbench"
	userImportReasonAlreadyAssigned    = "already_assigned"
	userImportReasonAmbiguousPoolMatch = "ambiguous_total_pool_match"
	userImportReasonImportFailed       = "import_failed"
)

// SocialTaskStatus constants
const (
	SocialTaskStatusPending        = "pending"
	SocialTaskStatusRegistering    = "registering"
	SocialTaskStatusImporting      = "importing"
	SocialTaskStatusParsing        = "parsing"
	SocialTaskStatusStored         = "stored"
	SocialTaskStatusFailed         = "failed"
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
	ErrSocialTaskMixedPlatforms       = infraerrors.BadRequest("SOCIAL_TASK_MIXED_PLATFORMS", "selected social accounts must belong to the same platform for one task")
	ErrSocialTaskIdempotencyConflict  = infraerrors.Conflict("SOCIAL_TASK_IDEMPOTENCY_CONFLICT", "idempotency key was already used for a different social task")
	ErrSocialTaskAccountBusy          = infraerrors.Conflict("SOCIAL_TASK_ACCOUNT_BUSY", "account already has an active task")
	ErrSocialAccountNotAssigned       = infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_ASSIGNED", "social account is not assigned to this user")
	ErrSocialAccountDefaultProxyRoute = infraerrors.BadRequest("SOCIAL_ACCOUNT_DEFAULT_PROXY_ROUTE_REQUIRED", "use the default proxy endpoint to set an account execution proxy")
	ErrSocialIPOwnerMismatch          = infraerrors.BadRequest("SOCIAL_IP_OWNER_MISMATCH", "social IP does not belong to the account owner")
)

// SocialAccount represents a social media account.
type SocialAccount struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	Platform             string    `json:"platform"`
	Username             string    `json:"username,omitempty"`
	IdentityKind         string    `json:"identity_kind"`
	IdentityKey          string    `json:"identity_key"`
	PlatformUserID       *string   `json:"platform_user_id,omitempty"`
	Password             *string   `json:"password,omitempty"`
	Phone                *string   `json:"phone,omitempty"`
	Email                *string   `json:"email,omitempty"`
	EmailPassword        *string   `json:"email_password,omitempty"`
	TwoFactor            *string   `json:"two_factor,omitempty"`
	BackupCode           *string   `json:"backup_code,omitempty"`
	EmailClientID        *string   `json:"email_client_id,omitempty"`
	EmailToken           *string   `json:"email_token,omitempty"`
	RegistrationIP       *string   `json:"registration_ip,omitempty"`
	AuthCookie           *string   `json:"auth_cookie,omitempty"`
	ExecutionAuth        *string   `json:"execution_auth,omitempty"`
	AccountStatus        string    `json:"account_status"`
	TaskStatus           string    `json:"task_status"`
	TaskMessage          *string   `json:"task_message,omitempty"`
	DefaultProxySnapshot *string   `json:"default_proxy_snapshot,omitempty"`
	AssignedUserID       *int64    `json:"assigned_user_id,omitempty"`
	AssignedUserEmail    *string   `json:"assigned_user_email,omitempty"`
	Remark               *string   `json:"remark,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// SocialTaskLog represents a task execution log entry.
type SocialTaskLog struct {
	ID               int64                              `json:"id"`
	SocialAccountID  int64                              `json:"social_account_id"`
	UserID           int64                              `json:"user_id"`
	Action           string                             `json:"action"`
	Target           *string                            `json:"target,omitempty"`
	Content          *string                            `json:"content,omitempty"`
	Payload          *domain.SocialTaskPayload          `json:"payload,omitempty"`
	TemplateSnapshot *domain.SocialTaskTemplateSnapshot `json:"template_snapshot,omitempty"`
	Status           string                             `json:"status"`
	ResultMessage    *string                            `json:"result_message,omitempty"`
	Price            float64                            `json:"price"`
	ChargedAmount    float64                            `json:"charged_amount"`
	ChargeStatus     string                             `json:"charge_status"`
	ChargeSource     *string                            `json:"charge_source,omitempty"`
	ProxyID          *int64                             `json:"proxy_id,omitempty"`
	ProxySnapshot    *string                            `json:"proxy_snapshot,omitempty"`
	BillingRequestID *string                            `json:"billing_request_id,omitempty"`
	IdempotencyKey   *string                            `json:"idempotency_key,omitempty"`
	ExecutedAt       *time.Time                         `json:"executed_at,omitempty"`
	CreatedAt        time.Time                          `json:"created_at"`
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
	AccountIDs     []int64
	AssignedOnly   bool
	UnassignedOnly bool
	Search         string
	TotalPoolOnly  bool
	IncludeOwner   bool
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

type UserSocialAccountDeleteResult struct {
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
	Payload          *domain.SocialTaskPayload
	TemplateSnapshot *domain.SocialTaskTemplateSnapshot
	Status           string
	ResultMessage    *string
	ProxyID          *int64
	ProxySnapshot    *string
	BillingRequestID *string
	IdempotencyKey   *string
}

// SocialTaskLogListFilters contains user-workbench task log filters.
type SocialTaskLogListFilters struct {
	UserID     int64
	LogIDs     []int64
	AccountIDs []int64
	Statuses   []string
	Limit      int
}

// SocialAccountService handles social account management.
type SocialAccountService struct {
	entClient           *dbent.Client
	taskMedia           *SocialTaskMediaService
	credentialEncryptor ExecutionAuthEncryptor
}

// NewSocialAccountService creates a new SocialAccountService.
func NewSocialAccountService(entClient *dbent.Client) *SocialAccountService {
	return &SocialAccountService{
		entClient: entClient,
		taskMedia: NewSocialTaskMediaService(entClient),
	}
}

func NewSocialAccountServiceWithCredentialEncryptor(entClient *dbent.Client, encryptor ExecutionAuthEncryptor) *SocialAccountService {
	svc := NewSocialAccountService(entClient)
	return svc.WithCredentialEncryptor(encryptor)
}

func (s *SocialAccountService) WithCredentialEncryptor(encryptor ExecutionAuthEncryptor) *SocialAccountService {
	if s == nil {
		return nil
	}
	s.credentialEncryptor = encryptor
	return s
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

	if err := s.applyCreateSocialAccountFields(q, input); err != nil {
		return nil, err
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return s.socialAccountFromEnt(ent), nil
}

func (s *SocialAccountService) applyCreateSocialAccountFields(q *dbent.SocialAccountCreate, input *CreateSocialAccountInput) error {
	if q == nil || input == nil {
		return nil
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
	if input.TwoFactor != nil && strings.TrimSpace(*input.TwoFactor) != "" {
		q.SetTwoFactor(*input.TwoFactor)
	}
	if input.BackupCode != nil && strings.TrimSpace(*input.BackupCode) != "" {
		q.SetBackupCode(*input.BackupCode)
	}
	if input.EmailClientID != nil && strings.TrimSpace(*input.EmailClientID) != "" {
		q.SetEmailClientID(*input.EmailClientID)
	}
	if input.EmailToken != nil && strings.TrimSpace(*input.EmailToken) != "" {
		q.SetEmailToken(*input.EmailToken)
	}
	if value := trimPtr(input.RegistrationIP); value != "" {
		q.SetRegistrationIP(value)
	}
	if input.AuthCookie != nil && strings.TrimSpace(*input.AuthCookie) != "" {
		q.SetAuthCookie(*input.AuthCookie)
	}
	if value := trimPtr(input.ExecutionAuth); value != "" {
		normalized, err := s.normalizeTwitterExecutionAuth(value, input.Name)
		if err != nil {
			return err
		}
		value = normalized
		q.SetExecutionAuth(value)
	}
	if value := trimPtr(input.DefaultProxySnapshot); value != "" {
		q.SetDefaultProxySnapshot(value)
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}
	return nil
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
	return s.socialAccountFromEnt(ent), nil
}

// List returns a paginated list of social accounts.
func (s *SocialAccountService) List(ctx context.Context, params pagination.PaginationParams, filters SocialAccountListFilters) ([]*SocialAccount, *pagination.PaginationResult, error) {
	filters = normalizeSocialAccountListFilters(filters)
	q := s.entClient.SocialAccount.Query()

	if filters.TotalPoolOnly {
		q = q.Where(totalPoolVisibleAccountPredicate())
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
	if filters.Search != "" {
		q = q.Where(socialaccount.Or(socialAccountSearchPredicates(filters.Search, true)...))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	if filters.IncludeOwner {
		q = q.WithAssignedUser()
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
		accounts[i] = s.socialAccountFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return accounts, result, nil
}

func parseSocialAccountIDSearch(search string) (int64, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(search), "#"))
	if trimmed == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func socialAccountSearchPredicates(search string, includeAssignedUser bool) []predicate.SocialAccount {
	trimmed := strings.TrimSpace(search)
	predicates := []predicate.SocialAccount{
		socialaccount.NameContainsFold(trimmed),
		socialaccount.PlatformContainsFold(trimmed),
		socialaccount.PlatformKeyContainsFold(trimmed),
		socialaccount.PlatformUserIDContainsFold(trimmed),
		socialaccount.IdentityKeyContainsFold(trimmed),
		socialaccount.PasswordContainsFold(trimmed),
		socialaccount.PhoneContainsFold(trimmed),
		socialaccount.EmailContainsFold(trimmed),
		socialaccount.EmailPasswordContainsFold(trimmed),
		socialaccount.TwoFactorContainsFold(trimmed),
		socialaccount.BackupCodeContainsFold(trimmed),
		socialaccount.EmailClientIDContainsFold(trimmed),
		socialaccount.EmailTokenContainsFold(trimmed),
		socialaccount.RegistrationIPContainsFold(trimmed),
		socialaccount.AuthCookieContainsFold(trimmed),
		socialaccount.DefaultProxySnapshotContainsFold(trimmed),
		socialaccount.TaskMessageContainsFold(trimmed),
		socialaccount.RemarkContainsFold(trimmed),
	}
	if includeAssignedUser {
		predicates = append(predicates, socialaccount.HasAssignedUserWith(user.EmailContainsFold(trimmed)))
	}
	if idSearch, ok := parseSocialAccountIDSearch(trimmed); ok {
		predicates = append(predicates, socialaccount.IDEQ(idSearch))
	}
	return predicates
}

// ListTotalPool returns records that are governed from the admin total account
// pool. User workbench staging imports remain visible from /accounts only until
// an admin explicitly uploads them into the pool.
func (s *SocialAccountService) ListTotalPool(ctx context.Context, params pagination.PaginationParams, filters SocialAccountListFilters) ([]*SocialAccount, *pagination.PaginationResult, error) {
	filters.TotalPoolOnly = true
	filters.IncludeOwner = true
	return s.List(ctx, params, filters)
}

// UpdateTotalPool updates an admin-governed total-pool account and rejects
// transient workbench staging records.
func (s *SocialAccountService) UpdateTotalPool(ctx context.Context, id int64, input *UpdateSocialAccountInput) (*SocialAccount, error) {
	exists, err := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.IDEQ(id),
			totalPoolVisibleAccountPredicate(),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
	}
	if input == nil {
		return s.Update(ctx, id, input)
	}
	updateInput := *input
	updateInput.Name = nil
	updateInput.PlatformUserID = nil
	return s.Update(ctx, id, &updateInput)
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
		setOptionalDeliveryString(input.TwoFactor, func(value string) {
			q.SetTwoFactor(value)
		}, func() {
			q.ClearTwoFactor()
		})
	}
	if input.BackupCode != nil {
		setOptionalDeliveryString(input.BackupCode, func(value string) {
			q.SetBackupCode(value)
		}, func() {
			q.ClearBackupCode()
		})
	}
	if input.EmailClientID != nil {
		setOptionalDeliveryString(input.EmailClientID, func(value string) {
			q.SetEmailClientID(value)
		}, func() {
			q.ClearEmailClientID()
		})
	}
	if input.EmailToken != nil {
		setOptionalDeliveryString(input.EmailToken, func(value string) {
			q.SetEmailToken(value)
		}, func() {
			q.ClearEmailToken()
		})
	}
	if input.RegistrationIP != nil {
		if value := trimPtr(input.RegistrationIP); value != "" {
			q.SetRegistrationIP(value)
		} else {
			q.ClearRegistrationIP()
		}
	}
	if input.AuthCookie != nil {
		setOptionalDeliveryString(input.AuthCookie, func(value string) {
			q.SetAuthCookie(value)
		}, func() {
			q.ClearAuthCookie()
		})
	}
	if input.ExecutionAuth != nil {
		if value := trimPtr(input.ExecutionAuth); value != "" {
			normalized, err := s.normalizeTwitterExecutionAuth(value, nextName)
			if err != nil {
				return nil, err
			}
			value = normalized
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
	return s.socialAccountFromEnt(ent), nil
}

// Delete permanently removes a social account from the account pool.
func (s *SocialAccountService) Delete(ctx context.Context, id int64) error {
	affected, err := s.hardDeleteAccounts(ctx, socialaccount.IDEQ(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
	}
	return nil
}

// DeleteTotalPool permanently removes an admin-governed total-pool account.
func (s *SocialAccountService) DeleteTotalPool(ctx context.Context, id int64) error {
	affected, err := s.hardDeleteAccounts(ctx, socialaccount.IDEQ(id), totalPoolVisibleAccountPredicate())
	if err != nil {
		return err
	}
	if affected == 0 {
		return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
	}
	return nil
}

func (s *SocialAccountService) hardDeleteAccounts(ctx context.Context, predicates ...predicate.SocialAccount) (int, error) {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return hardDeleteSocialAccountsWithClient(ctx, tx.Client(), predicates...)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	affected, err := hardDeleteSocialAccountsWithClient(ctx, tx.Client(), predicates...)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

func hardDeleteSocialAccountsWithClient(ctx context.Context, client *dbent.Client, predicates ...predicate.SocialAccount) (int, error) {
	ids, err := client.SocialAccount.Query().
		Where(predicates...).
		IDs(ctx)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	taskLogIDs, err := client.SocialTaskLog.Query().
		Where(socialtasklog.SocialAccountIDIn(ids...)).
		IDs(ctx)
	if err != nil {
		return 0, err
	}
	if len(taskLogIDs) > 0 {
		usagePredicates := make([]predicate.UsageLog, 0, len(taskLogIDs))
		for _, taskLogID := range taskLogIDs {
			usagePredicates = append(usagePredicates, usagelog.RequestIDHasPrefix(fmt.Sprintf("social-task:%d:", taskLogID)))
		}
		if _, err := client.UsageLog.Delete().
			Where(
				usagelog.ModelEQ(socialUsageLedgerModel),
				usagelog.APIKeyIDIsNil(),
				usagelog.Or(usagePredicates...),
			).
			Exec(ctx); err != nil {
			return 0, err
		}
		if _, err := client.SocialTaskLog.Delete().
			Where(socialtasklog.IDIn(taskLogIDs...)).
			Exec(ctx); err != nil {
			return 0, err
		}
	}
	if _, err := client.SocialIP.Update().
		Where(socialip.BoundSocialAccountIDIn(ids...)).
		ClearBoundSocialAccountID().
		Save(ctx); err != nil {
		return 0, err
	}
	return client.SocialAccount.Delete().
		Where(socialaccount.IDIn(ids...)).
		Exec(ctx)
}

func socialAccountClientFromContext(ctx context.Context, entClient *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return entClient
}

// Assign assigns a social account to a user.
func (s *SocialAccountService) Assign(ctx context.Context, accountID, userID int64) (*SocialAccount, error) {
	return s.assign(ctx, accountID, userID)
}

// AssignTotalPool assigns an admin-governed total-pool social account to a user.
func (s *SocialAccountService) AssignTotalPool(ctx context.Context, accountID, userID int64) (*SocialAccount, error) {
	return s.assign(ctx, accountID, userID, totalPoolVisibleAccountPredicate())
}

func (s *SocialAccountService) assign(ctx context.Context, accountID, userID int64, scopes ...predicate.SocialAccount) (*SocialAccount, error) {
	current, err := s.entClient.SocialAccount.Query().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID)}, scopes...)...).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, err
	}
	if current.AssignedUserID != nil {
		return nil, ErrSocialAccountAlreadyAssigned
	}
	if err := s.ensureAssignableTargetUser(ctx, userID); err != nil {
		return nil, err
	}
	updated, err := s.entClient.SocialAccount.Update().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID), socialaccount.AssignedUserIDIsNil()}, scopes...)...).
		SetAssignedUserID(userID).
		ClearDefaultProxySnapshot().
		SetTaskStatus(SocialTaskStatusStored).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		latest, latestErr := s.entClient.SocialAccount.Query().
			Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID)}, scopes...)...).
			Only(ctx)
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
	ent, err := s.entClient.SocialAccount.Query().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID)}, scopes...)...).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return s.socialAccountFromEnt(ent), nil
}

func (s *SocialAccountService) ensureAssignableTargetUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrUserNotFound
	}
	exists, err := s.entClient.User.Query().
		Where(user.IDEQ(userID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}

// Reclaim removes the user assignment from a social account.
func (s *SocialAccountService) Reclaim(ctx context.Context, accountID int64) (*SocialAccount, error) {
	account, _, err := s.reclaim(ctx, accountID)
	return account, err
}

// ReclaimTotalPool removes the user assignment from an admin-governed
// total-pool account.
func (s *SocialAccountService) ReclaimTotalPool(ctx context.Context, accountID int64) (*SocialAccount, error) {
	account, _, err := s.reclaim(ctx, accountID, totalPoolVisibleAccountPredicate())
	return account, err
}

func (s *SocialAccountService) reclaim(ctx context.Context, accountID int64, scopes ...predicate.SocialAccount) (*SocialAccount, bool, error) {
	current, err := s.entClient.SocialAccount.Query().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID)}, scopes...)...).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, false, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, false, err
	}
	if current.AssignedUserID == nil {
		return s.socialAccountFromEnt(current), false, nil
	}
	ownerID := *current.AssignedUserID
	updated, err := s.entClient.SocialAccount.Update().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID), socialaccount.AssignedUserIDEQ(ownerID)}, scopes...)...).
		ClearAssignedUserID().
		ClearDefaultProxySnapshot().
		Save(ctx)
	if err != nil {
		return nil, false, err
	}
	if updated == 0 {
		return nil, false, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Query().
		Where(append([]predicate.SocialAccount{socialaccount.IDEQ(accountID)}, scopes...)...).
		Only(ctx)
	if err != nil {
		return nil, false, err
	}
	return s.socialAccountFromEnt(ent), true, nil
}

func (s *SocialAccountService) BatchAssign(ctx context.Context, accountIDs []int64, userID int64) (*SocialAccountBatchResult, error) {
	return s.batchAssign(ctx, accountIDs, userID, s.Assign)
}

func (s *SocialAccountService) BatchAssignTotalPool(ctx context.Context, accountIDs []int64, userID int64) (*SocialAccountBatchResult, error) {
	return s.batchAssign(ctx, accountIDs, userID, s.AssignTotalPool)
}

func (s *SocialAccountService) batchAssign(ctx context.Context, accountIDs []int64, userID int64, assign func(context.Context, int64, int64) (*SocialAccount, error)) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	seen := newSocialAccountBatchIDTracker(len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 || userID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_input", "account could not be assigned")
			continue
		}
		if !seen.record(accountID) {
			addSocialAccountBatchSkipped(result, accountID, "", "duplicate_in_batch", "account could not be assigned")
			continue
		}
		account, err := assign(ctx, accountID, userID)
		if err != nil {
			if err == ErrSocialAccountAlreadyAssigned {
				result.Skipped++
				result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "skipped", Reason: "already_assigned", Error: "account is already assigned"})
				continue
			}
			if err == ErrUserNotFound {
				addSocialAccountBatchFailure(result, accountID, "", "target_user_not_found", "target user not found")
				continue
			}
			addSocialAccountBatchFailure(result, accountID, "", "assign_failed", totalAccountBatchAssignErrorMessage())
			continue
		}
		addSocialAccountBatchSuccess(result, account.ID, account.Name)
	}
	return result, nil
}

func (s *SocialAccountService) BatchReclaim(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	return s.batchReclaim(ctx, accountIDs, func(ctx context.Context, accountID int64) (*SocialAccount, bool, error) {
		return s.reclaim(ctx, accountID)
	})
}

func (s *SocialAccountService) BatchReclaimTotalPool(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	return s.batchReclaim(ctx, accountIDs, func(ctx context.Context, accountID int64) (*SocialAccount, bool, error) {
		return s.reclaim(ctx, accountID, totalPoolVisibleAccountPredicate())
	})
}

func (s *SocialAccountService) batchReclaim(ctx context.Context, accountIDs []int64, reclaim func(context.Context, int64) (*SocialAccount, bool, error)) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	seen := newSocialAccountBatchIDTracker(len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_id", "account could not be reclaimed")
			continue
		}
		if !seen.record(accountID) {
			addSocialAccountBatchSkipped(result, accountID, "", "duplicate_in_batch", "account could not be reclaimed")
			continue
		}
		account, reclaimed, err := reclaim(ctx, accountID)
		if err != nil {
			addSocialAccountBatchFailure(result, accountID, "", "reclaim_failed", totalAccountBatchReclaimErrorMessage())
			continue
		}
		if !reclaimed {
			addSocialAccountBatchSkipped(result, account.ID, account.Name, "already_unassigned", "account is already unassigned")
			continue
		}
		addSocialAccountBatchSuccess(result, account.ID, account.Name)
	}
	return result, nil
}

func (s *SocialAccountService) BatchDelete(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	return s.batchDelete(ctx, accountIDs, s.Delete)
}

func (s *SocialAccountService) BatchDeleteTotalPool(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	return s.batchDelete(ctx, accountIDs, s.DeleteTotalPool)
}

func (s *SocialAccountService) batchDelete(ctx context.Context, accountIDs []int64, deleteAccount func(context.Context, int64) error) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	seen := newSocialAccountBatchIDTracker(len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_id", "account could not be deleted")
			continue
		}
		if !seen.record(accountID) {
			addSocialAccountBatchSkipped(result, accountID, "", "duplicate_in_batch", "account could not be deleted")
			continue
		}
		if err := deleteAccount(ctx, accountID); err != nil {
			addSocialAccountBatchFailure(result, accountID, "", "delete_failed", totalAccountBatchDeleteErrorMessage())
			continue
		}
		addSocialAccountBatchSuccess(result, accountID, "")
	}
	return result, nil
}

func (s *SocialAccountService) StoreWorkbenchAccounts(ctx context.Context, accountIDs []int64) (*SocialAccountBatchResult, error) {
	result := &SocialAccountBatchResult{Total: len(accountIDs)}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchSkipped(result, accountID, "", "invalid_id", "account could not be uploaded")
			continue
		}
		account, err := s.entClient.SocialAccount.Get(ctx, accountID)
		if err != nil {
			if dbent.IsNotFound(err) {
				addSocialAccountBatchSkipped(result, accountID, "", "not_found", "account could not be uploaded")
				continue
			}
			addSocialAccountBatchFailure(result, accountID, "", "load_failed", err.Error())
			continue
		}
		if !isWorkbenchStagingSocialAccount(account) {
			result.Skipped++
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "skipped", Reason: "already_stored", Error: "account is not a workbench staging account"})
			continue
		}
		next := socialAccountFromEnt(account)
		if err := validateMutableSocialAccountCredentials(next); err != nil {
			addSocialAccountBatchFailure(result, account.ID, account.Name, "invalid_credentials", "account could not be uploaded")
			continue
		}
		updated, err := s.entClient.SocialAccount.Update().
			Where(
				socialaccount.IDEQ(account.ID),
				workbenchStagingAccountPredicate(),
			).
			SetAccountStatus(SocialAccountStatusPendingCheck).
			SetTaskStatus(SocialTaskStatusStored).
			ClearTaskMessage().
			Save(ctx)
		if err != nil {
			addSocialAccountBatchFailure(result, account.ID, account.Name, "upload_failed", err.Error())
			continue
		}
		if updated == 0 {
			addSocialAccountBatchSkipped(result, account.ID, account.Name, "state_changed", "account could not be uploaded")
			continue
		}
		addSocialAccountBatchSuccess(result, account.ID, account.Name)
	}
	return result, nil
}

func isWorkbenchStagingSocialAccount(account *dbent.SocialAccount) bool {
	return account != nil &&
		account.AssignedUserID != nil &&
		account.AccountStatus == SocialAccountStatusNotStored &&
		account.TaskStatus == SocialTaskStatusPending
}

func addSocialAccountBatchFailure(result *SocialAccountBatchResult, id int64, name, reason, message string) {
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

func addSocialAccountBatchSuccess(result *SocialAccountBatchResult, id int64, name string) {
	if result == nil {
		return
	}
	result.Succeeded++
	result.Items = append(result.Items, SocialAccountBatchItemResult{ID: id, Name: name, Status: "succeeded"})
}

type socialAccountBatchIDTracker map[int64]struct{}

func newSocialAccountBatchIDTracker(size int) socialAccountBatchIDTracker {
	return make(socialAccountBatchIDTracker, size)
}

func (tracker socialAccountBatchIDTracker) record(id int64) bool {
	if _, ok := tracker[id]; ok {
		return false
	}
	tracker[id] = struct{}{}
	return true
}

func (s *SocialAccountService) SetDefaultProxyForUser(ctx context.Context, accountID, userID int64, ip *SocialIP) (*SocialAccount, error) {
	account, err := s.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.AssignedUserID == nil || *account.AssignedUserID != userID {
		return nil, ErrSocialAccountNotAssigned
	}
	snapshot, err := defaultProxySnapshotForOwner(ip, userID)
	if err != nil {
		return nil, err
	}
	return s.setDefaultProxySnapshotForUser(ctx, accountID, userID, snapshot)
}

func (s *SocialAccountService) BatchSetDefaultProxyForUser(ctx context.Context, userID int64, accountIDs []int64, mode string, ip *SocialIP, pool []*SocialIP) (*SocialAccountBatchResult, error) {
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

	seen := newSocialAccountBatchIDTracker(len(accountIDs))
	randomIndex := 0
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			addSocialAccountBatchFailure(result, accountID, "", "invalid_id", "account proxy could not be assigned")
			continue
		}
		if !seen.record(accountID) {
			addSocialAccountBatchSkipped(result, accountID, "", "duplicate_in_batch", "account proxy could not be assigned")
			continue
		}
		account, err := s.GetByID(ctx, accountID)
		if err != nil {
			addSocialAccountBatchFailure(result, accountID, "", "account_not_found", "account proxy could not be assigned")
			continue
		}
		if account.AssignedUserID == nil || *account.AssignedUserID != userID {
			addSocialAccountBatchFailure(result, accountID, "", "account_not_assigned", "account proxy could not be assigned")
			continue
		}
		var snapshot *string
		switch mode {
		case DefaultProxyAssignmentSpecific:
			if specificProxyUnavailable {
				addSocialAccountBatchFailure(result, accountID, account.Name, "proxy_not_available", "account proxy could not be assigned")
				continue
			}
			snapshot = specificSnapshot
		case DefaultProxyAssignmentRandom:
			snapshot = randomSnapshots[randomIndex%len(randomSnapshots)]
			randomIndex++
		case DefaultProxyAssignmentClear:
			snapshot = nil
		}
		if _, err := s.setDefaultProxySnapshotForUser(ctx, accountID, userID, snapshot); err != nil {
			addSocialAccountBatchFailure(result, accountID, account.Name, "assign_failed", "account proxy could not be assigned")
			continue
		}
		addSocialAccountBatchSuccess(result, accountID, account.Name)
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
	if current.AssignedUserID == nil || int64(*current.AssignedUserID) != userID {
		return nil, ErrSocialAccountNotAssigned
	}
	next := socialAccountFromEnt(current)
	q := s.entClient.SocialAccount.Update().
		Where(
			socialaccount.IDEQ(accountID),
			socialaccount.AssignedUserIDEQ(userID),
		)

	if input.Password != nil {
		value := *input.Password
		if strings.TrimSpace(value) == "" {
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
		setOptionalDeliveryString(input.EmailPassword, func(value string) {
			q.SetEmailPassword(value)
			next.EmailPassword = &value
		}, func() {
			q.ClearEmailPassword()
			next.EmailPassword = nil
		})
	}
	if input.TwoFactor != nil {
		setOptionalDeliveryString(input.TwoFactor, func(value string) {
			q.SetTwoFactor(value)
			next.TwoFactor = &value
		}, func() {
			q.ClearTwoFactor()
			next.TwoFactor = nil
		})
	}
	if input.BackupCode != nil {
		setOptionalDeliveryString(input.BackupCode, func(value string) {
			q.SetBackupCode(value)
			next.BackupCode = &value
		}, func() {
			q.ClearBackupCode()
			next.BackupCode = nil
		})
	}
	if input.EmailClientID != nil {
		setOptionalDeliveryString(input.EmailClientID, func(value string) {
			q.SetEmailClientID(value)
			next.EmailClientID = &value
		}, func() {
			q.ClearEmailClientID()
			next.EmailClientID = nil
		})
	}
	if input.EmailToken != nil {
		setOptionalDeliveryString(input.EmailToken, func(value string) {
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
		setOptionalDeliveryString(input.AuthCookie, func(value string) {
			q.SetAuthCookie(value)
			next.AuthCookie = &value
		}, func() {
			q.ClearAuthCookie()
			next.AuthCookie = nil
		})
	}
	if input.ExecutionAuth != nil {
		executionAuth := input.ExecutionAuth
		if strings.TrimSpace(*executionAuth) != "" {
			normalized, err := s.normalizeTwitterExecutionAuth(*executionAuth, current.Name)
			if err != nil {
				return nil, err
			}
			executionAuth = &normalized
		}
		setOptionalDeliveryString(executionAuth, func(value string) {
			q.SetExecutionAuth(value)
			next.ExecutionAuth = &value
		}, func() {
			q.ClearExecutionAuth()
			next.ExecutionAuth = nil
		})
	}
	if input.Remark != nil {
		setOptionalDeliveryString(input.Remark, func(value string) {
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
	updated, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		exists, err := s.entClient.SocialAccount.Query().
			Where(socialaccount.IDEQ(accountID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.socialAccountFromEnt(ent), nil
}

func defaultProxySnapshotForOwner(ip *SocialIP, ownerID int64) (*string, error) {
	if ip == nil {
		return nil, nil
	}
	if ip.UserID != ownerID {
		return nil, ErrSocialIPOwnerMismatch
	}
	if err := EnsureSocialIPUsableForExecution(ip); err != nil {
		return nil, err
	}
	snapshot := SocialIPTaskSnapshot(ip)
	return &snapshot, nil
}

func (s *SocialAccountService) setDefaultProxySnapshotForUser(ctx context.Context, accountID, userID int64, snapshot *string) (*SocialAccount, error) {
	q := s.entClient.SocialAccount.Update().
		Where(
			socialaccount.IDEQ(accountID),
			socialaccount.AssignedUserIDEQ(userID),
		)
	if snapshot == nil || strings.TrimSpace(*snapshot) == "" {
		q.ClearDefaultProxySnapshot()
	} else {
		trimmed := strings.TrimSpace(*snapshot)
		q.SetDefaultProxySnapshot(trimmed)
	}
	updated, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		exists, err := s.entClient.SocialAccount.Query().
			Where(socialaccount.IDEQ(accountID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return nil, ErrSocialAccountAssignmentChanged
	}
	ent, err := s.entClient.SocialAccount.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.socialAccountFromEnt(ent), nil
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

	if err := s.applyCreateSocialAccountFields(q, input); err != nil {
		return nil, err
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return s.socialAccountFromEnt(ent), nil
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
		return nil, ErrSocialAccountImportAmbiguous
	}

	matches := make([]*dbent.SocialAccount, 0, len(allMatches))
	for _, account := range allMatches {
		if account.AssignedUserID != nil && *account.AssignedUserID == userID {
			return nil, ErrSocialAccountAlreadyAssigned
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
	if err := s.applyUserImportFields(ent, input); err != nil {
		return nil, err
	}
	saved, err := ent.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return s.socialAccountFromEnt(saved), nil
}

func (s *SocialAccountService) BatchImportForUser(ctx context.Context, userID int64, inputs []*UserImportSocialAccountInput) (*UserSocialAccountImportResult, error) {
	result := &UserSocialAccountImportResult{Total: len(inputs)}
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input == nil {
			result.Skipped++
			result.Failed++
			message := userAccountBatchImportReasonMessage(userImportReasonInvalidInput)
			result.Errors = append(result.Errors, message)
			result.Items = append(result.Items, SocialAccountBatchItemResult{Status: "failed", Reason: userImportReasonInvalidInput, Error: message})
			continue
		}
		if err := validateUserImportCredentials(input); err != nil {
			result.Skipped++
			result.Failed++
			message := userAccountBatchImportReasonMessage(userImportReasonInvalidInput)
			result.Errors = append(result.Errors, message)
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "failed", Reason: userImportReasonInvalidInput, Error: message})
			continue
		}
		platform := normalizeSocialPlatform(input.Platform)
		identity := socialAccountBusinessIdentity(platform, input.Name)
		if key := socialAccountDedupKeyFromIdentity(identity); key != "" {
			if _, ok := seen[key]; ok {
				result.Skipped++
				result.Duplicates++
				message := userAccountBatchImportReasonMessage(userImportReasonDuplicateInBatch)
				result.Errors = append(result.Errors, message)
				result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: "duplicate", Reason: userImportReasonDuplicateInBatch, Error: message})
				continue
			}
			seen[key] = struct{}{}
		}
		account, err := s.importUserWorkbenchAccount(ctx, userID, input)
		if err != nil {
			status, reason, message := s.classifyUserAccountBatchImportFailure(ctx, userID, input, err)
			result.Skipped++
			if status == "duplicate" {
				result.Duplicates++
			} else {
				result.Failed++
			}
			result.Items = append(result.Items, SocialAccountBatchItemResult{Name: strings.TrimSpace(input.Name), Status: status, Reason: reason, Error: message})
			result.Errors = append(result.Errors, message)
			continue
		}
		result.Imported++
		result.Succeeded++
		result.Accounts = append(result.Accounts, account)
		result.Items = append(result.Items, SocialAccountBatchItemResult{ID: account.ID, Name: account.Name, Status: "succeeded", Reason: userAccountBatchImportSuccessReason(account)})
	}
	return result, nil
}

func (s *SocialAccountService) classifyUserAccountBatchImportFailure(ctx context.Context, userID int64, input *UserImportSocialAccountInput, err error) (status, reason, message string) {
	switch {
	case errors.Is(err, ErrSocialAccountDuplicate):
		reason = userImportReasonDuplicateInPool
		status = "duplicate"
	case errors.Is(err, ErrSocialAccountAlreadyAssigned):
		reason = s.userAccountImportAssignmentConflictReason(ctx, userID, input)
		status = "duplicate"
	case errors.Is(err, ErrSocialAccountImportAmbiguous):
		reason = userImportReasonAmbiguousPoolMatch
		status = "failed"
	case errors.Is(err, ErrSocialAccountImportIncomplete), infraerrors.IsBadRequest(err):
		reason = userImportReasonInvalidInput
		status = "failed"
	default:
		reason = userImportReasonImportFailed
		status = "failed"
	}
	return status, reason, userAccountBatchImportReasonMessage(reason)
}

func (s *SocialAccountService) userAccountImportAssignmentConflictReason(ctx context.Context, userID int64, input *UserImportSocialAccountInput) string {
	if input == nil {
		return userImportReasonAlreadyAssigned
	}
	identity := socialAccountBusinessIdentity(normalizeSocialPlatform(input.Platform), input.Name)
	matches, err := s.findAccountsByBusinessIdentity(ctx, identity)
	if err != nil {
		return userImportReasonAlreadyAssigned
	}
	for _, account := range matches {
		if account.AssignedUserID != nil && *account.AssignedUserID == userID {
			return userImportReasonAlreadyInWorkbench
		}
	}
	return userImportReasonAlreadyAssigned
}

func userAccountBatchImportSuccessReason(account *SocialAccount) string {
	if account != nil && account.AccountStatus == SocialAccountStatusNotStored && account.TaskStatus == SocialTaskStatusPending {
		return userImportReasonStagedNotStored
	}
	return userImportReasonMatchedTotalPool
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

func setOptionalDeliveryString(value *string, set func(string), clear func()) {
	if value == nil {
		return
	}
	if strings.TrimSpace(*value) == "" {
		clear()
		return
	}
	set(*value)
}

func (s *SocialAccountService) applyUserImportFields(q *dbent.SocialAccountCreate, input *UserImportSocialAccountInput) error {
	if q == nil || input == nil {
		return nil
	}
	if value := trimPtr(input.PlatformUserID); value != "" {
		q.SetPlatformUserID(value)
	}
	if input.Password != nil && strings.TrimSpace(*input.Password) != "" {
		q.SetPassword(*input.Password)
	}
	if value := trimPtr(input.Phone); value != "" {
		q.SetPhone(value)
	}
	if value := trimPtr(input.Email); value != "" {
		q.SetEmail(value)
	}
	if input.EmailPassword != nil && strings.TrimSpace(*input.EmailPassword) != "" {
		q.SetEmailPassword(*input.EmailPassword)
	}
	if input.TwoFactor != nil && strings.TrimSpace(*input.TwoFactor) != "" {
		q.SetTwoFactor(*input.TwoFactor)
	}
	if input.BackupCode != nil && strings.TrimSpace(*input.BackupCode) != "" {
		q.SetBackupCode(*input.BackupCode)
	}
	if input.EmailClientID != nil && strings.TrimSpace(*input.EmailClientID) != "" {
		q.SetEmailClientID(*input.EmailClientID)
	}
	if input.EmailToken != nil && strings.TrimSpace(*input.EmailToken) != "" {
		q.SetEmailToken(*input.EmailToken)
	}
	if value := trimPtr(input.RegistrationIP); value != "" {
		q.SetRegistrationIP(value)
	}
	if input.AuthCookie != nil && strings.TrimSpace(*input.AuthCookie) != "" {
		q.SetAuthCookie(*input.AuthCookie)
	}
	if input.ExecutionAuth != nil && strings.TrimSpace(*input.ExecutionAuth) != "" {
		normalized, err := s.normalizeTwitterExecutionAuth(*input.ExecutionAuth, input.Name)
		if err != nil {
			return err
		}
		q.SetExecutionAuth(normalized)
	}
	if input.Remark != nil && strings.TrimSpace(*input.Remark) != "" {
		q.SetRemark(*input.Remark)
	}
	return nil
}

func (s *SocialAccountService) normalizeTwitterExecutionAuth(value, screenName string) (string, error) {
	var encryptor ExecutionAuthEncryptor
	if s != nil {
		encryptor = s.credentialEncryptor
	}
	return normalizeTwitterExecutionAuthForEncryptedStorage(value, screenName, encryptor)
}

func (s *SocialAccountService) socialAccountFromEnt(e *dbent.SocialAccount) *SocialAccount {
	account := socialAccountFromEnt(e)
	if account != nil && account.ExecutionAuth != nil && looksLikePlainExecutionAuthPayload(*account.ExecutionAuth) {
		account.ExecutionAuth = nil
	}
	return account
}

func looksLikePlainExecutionAuthPayload(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "{")
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
func (s *SocialAccountService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, filterArgs ...SocialAccountListFilters) ([]*SocialAccount, *pagination.PaginationResult, error) {
	filters := SocialAccountListFilters{}
	if len(filterArgs) > 0 {
		filters = filterArgs[0]
	}
	q := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.AssignedUserIDEQ(userID),
		)
	q = applySocialAccountListFilters(q, filters)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Asc(socialaccount.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	accounts := make([]*SocialAccount, len(ents))
	for i, e := range ents {
		accounts[i] = s.socialAccountFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return accounts, result, nil
}

func applySocialAccountListFilters(q *dbent.SocialAccountQuery, filters SocialAccountListFilters) *dbent.SocialAccountQuery {
	filters = normalizeSocialAccountListFilters(filters)
	if platform := normalizeSocialPlatform(filters.Platform); platform != "" {
		q = q.Where(socialaccount.PlatformKeyEQ(platform))
	}
	if filters.AccountStatus != "" {
		if filters.AccountStatus == SocialAccountStatusInvalid {
			q = q.Where(socialaccount.Or(
				socialaccount.AccountStatusEQ(SocialAccountStatusInvalid),
				socialaccount.AccountStatusEQ(SocialAccountStatusNotStored),
			))
		} else {
			q = q.Where(socialaccount.AccountStatusEQ(filters.AccountStatus))
		}
	}
	if filters.TaskStatus != "" {
		q = q.Where(socialaccount.TaskStatusEQ(filters.TaskStatus))
	}
	if len(filters.AccountIDs) > 0 {
		q = q.Where(socialaccount.IDIn(filters.AccountIDs...))
	}
	if filters.Search != "" {
		q = q.Where(socialaccount.Or(socialAccountSearchPredicates(filters.Search, false)...))
	}
	return q
}

func normalizeSocialAccountListFilters(filters SocialAccountListFilters) SocialAccountListFilters {
	filters.Platform = strings.TrimSpace(filters.Platform)
	filters.AccountStatus = strings.TrimSpace(filters.AccountStatus)
	filters.TaskStatus = strings.TrimSpace(filters.TaskStatus)
	filters.Search = strings.TrimSpace(filters.Search)
	return filters
}

// ListAllByUserForExport returns all social accounts assigned to a user for CSV export.
func (s *SocialAccountService) ListAllByUserForExport(ctx context.Context, userID int64, filterArgs ...SocialAccountListFilters) ([]*SocialAccount, error) {
	filters := SocialAccountListFilters{}
	if len(filterArgs) > 0 {
		filters = filterArgs[0]
	}
	q := s.entClient.SocialAccount.Query().
		Where(socialaccount.AssignedUserIDEQ(userID))
	q = applySocialAccountListFilters(q, filters)
	ents, err := q.
		Order(dbent.Desc(socialaccount.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]*SocialAccount, len(ents))
	for i, e := range ents {
		accounts[i] = s.socialAccountFromEnt(e)
	}
	return accounts, nil
}

// DeleteForUser deletes an account assigned to the current user from the account pool.
func (s *SocialAccountService) DeleteForUser(ctx context.Context, userID, accountID int64) error {
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
	affected, err := s.hardDeleteAccounts(ctx, socialaccount.IDEQ(accountID), socialaccount.AssignedUserIDEQ(userID))
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSocialAccountAssignmentChanged
	}
	return nil
}

func (s *SocialAccountService) BatchDeleteForUser(ctx context.Context, userID int64, accountIDs []int64) (*UserSocialAccountDeleteResult, error) {
	result := &UserSocialAccountDeleteResult{Total: len(accountIDs)}
	seen := newSocialAccountBatchIDTracker(len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			result.Failed++
			result.Errors = append(result.Errors, userAccountBatchDeleteErrorMessage())
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "failed", Reason: "invalid_id", Error: userAccountBatchDeleteErrorMessage()})
			continue
		}
		if !seen.record(accountID) {
			result.Skipped++
			result.Items = append(result.Items, SocialAccountBatchItemResult{ID: accountID, Status: "skipped", Reason: "duplicate_in_batch", Error: userAccountBatchDeleteErrorMessage()})
			continue
		}
		if err := s.DeleteForUser(ctx, userID, accountID); err != nil {
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

func userAccountBatchImportReasonMessage(reason string) string {
	switch reason {
	case userImportReasonMatchedTotalPool:
		return "matched an existing total-pool account"
	case userImportReasonStagedNotStored:
		return "staged as a not-stored workbench account"
	case userImportReasonInvalidInput:
		return "account import data is invalid"
	case userImportReasonDuplicateInBatch:
		return "account is duplicated in this import batch"
	case userImportReasonDuplicateInPool:
		return "account already exists in the total account pool"
	case userImportReasonAlreadyInWorkbench:
		return "account already exists in your workbench"
	case userImportReasonAlreadyAssigned:
		return "account is already assigned to a workbench"
	case userImportReasonAmbiguousPoolMatch:
		return "multiple total-pool accounts match this username"
	case userImportReasonImportFailed:
		fallthrough
	default:
		return userAccountBatchImportErrorMessage()
	}
}

func userAccountBatchDeleteErrorMessage() string {
	return "account could not be deleted"
}

func totalAccountBatchAssignErrorMessage() string {
	return "account could not be assigned"
}

func totalAccountBatchReclaimErrorMessage() string {
	return "account could not be reclaimed"
}

func totalAccountBatchDeleteErrorMessage() string {
	return "account could not be deleted"
}

// CreateTaskLog creates a task execution log entry.
func (s *SocialAccountService) CreateTaskLog(ctx context.Context, input *CreateSocialTaskLogInput) (*SocialTaskLog, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	if dbent.TxFromContext(ctx) != nil {
		return s.createTaskLogWithPreparedMedia(ctx, input)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	log, err := s.createTaskLogWithPreparedMedia(txCtx, input)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return log, nil
}

func (s *SocialAccountService) createTaskLogWithPreparedMedia(ctx context.Context, input *CreateSocialTaskLogInput) (*SocialTaskLog, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_INPUT_REQUIRED", "social task input is required")
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = SocialTaskLogStatusPending
	}
	client := socialTaskMediaClientFromContext(ctx, s.entClient)
	payload := cloneSocialTaskPayload(input.Payload)
	snapshot := cloneSocialTaskTemplateSnapshot(input.TemplateSnapshot)
	if s.taskMedia != nil {
		var err error
		payload, snapshot, err = s.taskMedia.MaterializeTaskLogMedia(ctx, input.UserID, payload, snapshot)
		if err != nil {
			return nil, err
		}
	}
	create := client.SocialTaskLog.Create().
		SetSocialAccountID(input.AccountID).
		SetUserID(input.UserID).
		SetAction(input.Action).
		SetStatus(status).
		SetPrice(SocialTaskPriceForAction(input.Action)).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetNillableTarget(input.Target).
		SetNillableContent(input.Content).
		SetNillableResultMessage(input.ResultMessage).
		SetNillableProxyID(input.ProxyID).
		SetNillableProxySnapshot(input.ProxySnapshot).
		SetNillableBillingRequestID(input.BillingRequestID).
		SetNillableIdempotencyKey(input.IdempotencyKey)
	if payload != nil {
		create.SetPayload(*payload)
	}
	if snapshot != nil {
		create.SetTemplateSnapshot(*snapshot)
	}
	ent, err := create.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) && input.IdempotencyKey != nil {
			existing, findErr := s.FindTaskLogByIdempotency(ctx, input.UserID, input.AccountID, input.Action, *input.IdempotencyKey)
			if findErr != nil {
				return nil, findErr
			}
			if existing != nil {
				if !socialTaskLogMatchesIdempotentInput(existing, &CreateSocialTaskLogInput{
					AccountID:        input.AccountID,
					UserID:           input.UserID,
					Action:           input.Action,
					Target:           input.Target,
					Content:          input.Content,
					Payload:          payload,
					TemplateSnapshot: snapshot,
				}) {
					return nil, ErrSocialTaskIdempotencyConflict
				}
				return existing, nil
			}
		}
		if isSocialTaskActiveAccountConstraintError(err) {
			return nil, ErrSocialTaskAccountBusy
		}
		return nil, err
	}
	return socialTaskLogFromEnt(ent), nil
}

func isSocialTaskActiveAccountConstraintError(err error) bool {
	if !dbent.IsConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_social_task_logs_one_active_per_account") ||
		(strings.Contains(message, "unique") &&
			strings.Contains(message, "social_task_logs") &&
			strings.Contains(message, "social_account_id"))
}

func socialTaskLogMatchesIdempotentInput(log *SocialTaskLog, input *CreateSocialTaskLogInput) bool {
	if log == nil || input == nil {
		return false
	}
	if log.UserID != input.UserID || log.SocialAccountID != input.AccountID || log.Action != input.Action {
		return false
	}
	if !socialTaskStringPtrEqual(log.Target, input.Target) || !socialTaskStringPtrEqual(log.Content, input.Content) {
		return false
	}
	if !socialTaskPayloadEqual(log.Payload, input.Payload) {
		return false
	}
	return socialTaskTemplateSnapshotEqual(log.TemplateSnapshot, input.TemplateSnapshot)
}

func socialTaskStringPtrEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func socialTaskPayloadEqual(left, right *domain.SocialTaskPayload) bool {
	left = cloneSocialTaskPayload(left)
	right = cloneSocialTaskPayload(right)
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftRaw) == string(rightRaw)
}

func socialTaskTemplateSnapshotEqual(left, right *domain.SocialTaskTemplateSnapshot) bool {
	left = cloneSocialTaskTemplateSnapshot(left)
	right = cloneSocialTaskTemplateSnapshot(right)
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftRaw) == string(rightRaw)
}

func (s *SocialAccountService) MarkTaskLogFailedNotCharged(ctx context.Context, taskLogID int64, message string) (*SocialTaskLog, error) {
	return s.markTaskLogWithStatusFailedNotCharged(ctx, taskLogID, SocialTaskLogStatusPending, message, nil)
}

func (s *SocialAccountService) MarkStaleRunningTaskLogFailedNotCharged(ctx context.Context, taskLogID int64, staleBefore time.Time, message string) (*SocialTaskLog, error) {
	return s.markTaskLogWithStatusFailedNotCharged(ctx, taskLogID, SocialTaskLogStatusRunning, message, []predicate.SocialTaskLog{
		socialtasklog.UpdatedAtLTE(staleBefore),
	})
}

func (s *SocialAccountService) markTaskLogWithStatusFailedNotCharged(ctx context.Context, taskLogID int64, expectedStatus, message string, extraPredicates []predicate.SocialTaskLog) (*SocialTaskLog, error) {
	if dbent.TxFromContext(ctx) != nil {
		return s.markTaskLogWithStatusFailedNotChargedInTx(ctx, taskLogID, expectedStatus, message, extraPredicates)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	log, err := s.markTaskLogWithStatusFailedNotChargedInTx(txCtx, taskLogID, expectedStatus, message, extraPredicates)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return log, nil
}

func (s *SocialAccountService) markTaskLogWithStatusFailedNotChargedInTx(ctx context.Context, taskLogID int64, expectedStatus, message string, extraPredicates []predicate.SocialTaskLog) (*SocialTaskLog, error) {
	client := socialAccountClientFromContext(ctx, s.entClient)
	now := time.Now()
	predicates := []predicate.SocialTaskLog{
		socialtasklog.IDEQ(taskLogID),
		socialtasklog.StatusEQ(expectedStatus),
		socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
	}
	predicates = append(predicates, extraPredicates...)
	updated, err := client.SocialTaskLog.Update().
		Where(predicates...).
		SetStatus(SocialTaskLogStatusFailed).
		SetResultMessage(message).
		SetExecutedAt(now).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		ClearChargeSource().
		ClearBillingRequestID().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		exists, err := client.SocialTaskLog.Query().Where(socialtasklog.IDEQ(taskLogID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, infraerrors.NotFound("SOCIAL_TASK_LOG_NOT_FOUND", "social task log not found")
		}
		return nil, infraerrors.Conflict("SOCIAL_TASK_LOG_FINALIZED", "social task log was already finalized")
	}
	ent, err := client.SocialTaskLog.Get(ctx, taskLogID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("SOCIAL_TASK_LOG_NOT_FOUND", "social task log not found")
		}
		return nil, err
	}
	if ent.SocialAccountID > 0 {
		accountUpdate := client.SocialAccount.UpdateOneID(ent.SocialAccountID).
			SetTaskStatus(SocialTaskStatusStored)
		if strings.TrimSpace(message) != "" {
			accountUpdate.SetTaskMessage(message)
		} else {
			accountUpdate.ClearTaskMessage()
		}
		if _, err := accountUpdate.Save(ctx); err != nil {
			return nil, err
		}
	}
	return socialTaskLogFromEnt(ent), nil
}

func (s *SocialAccountService) FindTaskLogByIdempotency(ctx context.Context, userID, accountID int64, action, key string) (*SocialTaskLog, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	client := socialAccountClientFromContext(ctx, s.entClient)
	ent, err := client.SocialTaskLog.Query().
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

// HasActiveTaskLogForAccount reports whether an account already has an unfinished task.
func (s *SocialAccountService) HasActiveTaskLogForAccount(ctx context.Context, userID, accountID int64) (bool, error) {
	if s == nil || s.entClient == nil {
		return false, infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE", "social account service is unavailable")
	}
	if userID <= 0 || accountID <= 0 {
		return false, infraerrors.BadRequest("SOCIAL_TASK_ACCOUNT_ID_INVALID", "social task account id must be positive")
	}
	client := socialAccountClientFromContext(ctx, s.entClient)
	return client.SocialTaskLog.Query().
		Where(
			socialtasklog.UserIDEQ(userID),
			socialtasklog.SocialAccountIDEQ(accountID),
			socialtasklog.StatusIn(SocialTaskLogStatusPending, SocialTaskLogStatusRunning),
		).
		Exist(ctx)
}

// ListTaskLogsForUser returns recent task logs owned by a workbench user.
func (s *SocialAccountService) ListTaskLogsForUser(ctx context.Context, filters SocialTaskLogListFilters) ([]*SocialTaskLog, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE", "social account service is unavailable")
	}
	if filters.UserID <= 0 {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_USER_REQUIRED", "task log owner is required")
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := s.entClient.SocialTaskLog.Query().
		Where(socialtasklog.UserIDEQ(filters.UserID))
	if ids := uniquePositiveInt64s(filters.LogIDs); len(ids) > 0 {
		q = q.Where(socialtasklog.IDIn(ids...))
	}
	if ids := uniquePositiveInt64s(filters.AccountIDs); len(ids) > 0 {
		q = q.Where(socialtasklog.SocialAccountIDIn(ids...))
	}
	if statuses := normalizeSocialTaskLogStatuses(filters.Statuses); len(statuses) > 0 {
		q = q.Where(socialtasklog.StatusIn(statuses...))
	}
	ents, err := q.
		Order(dbent.Desc(socialtasklog.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	logs := make([]*SocialTaskLog, 0, len(ents))
	for _, ent := range ents {
		logs = append(logs, socialTaskLogFromEnt(ent))
	}
	return logs, nil
}

func uniquePositiveInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeSocialTaskLogStatuses(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	valid := map[string]struct{}{
		SocialTaskLogStatusPending: {},
		SocialTaskLogStatusRunning: {},
		SocialTaskLogStatusSuccess: {},
		SocialTaskLogStatusFailed:  {},
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		status := strings.ToLower(strings.TrimSpace(value))
		if _, ok := valid[status]; !ok {
			continue
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		result = append(result, status)
	}
	return result
}

// GetStats returns summary statistics for social accounts.
func (s *SocialAccountService) GetStats(ctx context.Context) (map[string]int, error) {
	total, _ := s.entClient.SocialAccount.Query().
		Where(totalPoolVisibleAccountPredicate()).Count(ctx)
	stored, _ := s.entClient.SocialAccount.Query().
		Where(
			totalPoolVisibleAccountPredicate(),
			socialaccount.TaskStatusEQ(SocialTaskStatusStored),
		).Count(ctx)
	available, _ := s.entClient.SocialAccount.Query().
		Where(
			totalPoolVisibleAccountPredicate(),
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

func totalPoolVisibleAccountPredicate() predicate.SocialAccount {
	return socialaccount.Not(workbenchStagingAccountPredicate())
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
			totalPoolVisibleAccountPredicate(),
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
	case SocialTaskActionLogin:
		return SocialTaskActionLogin, true
	case SocialTaskActionLoginCheck:
		return SocialTaskActionLoginCheck, true
	case SocialTaskActionFollow:
		return SocialTaskActionFollow, true
	case SocialTaskActionPost:
		return SocialTaskActionPost, true
	case SocialTaskActionLike:
		return SocialTaskActionLike, true
	case SocialTaskActionRetweet:
		return SocialTaskActionRetweet, true
	case SocialTaskActionUpdateProfile:
		return SocialTaskActionUpdateProfile, true
	case SocialTaskActionUpdateAvatar:
		return SocialTaskActionUpdateAvatar, true
	case SocialTaskActionUpdateBanner:
		return SocialTaskActionUpdateBanner, true
	default:
		return "", false
	}
}

func IsBillableSocialTaskAction(action string) bool {
	return SocialTaskPriceForAction(action) > 0
}

func SocialTaskPriceForAction(action string) float64 {
	normalized, ok := NormalizeSocialTaskAction(action)
	if !ok {
		return 0
	}
	if normalized == SocialTaskActionLogin {
		return SocialTaskUnitPrice
	}
	return 0
}

func EnsureExecutableSocialTaskAction(action string) error {
	_, ok := NormalizeSocialTaskAction(action)
	if !ok {
		return ErrSocialTaskUnsupportedAction
	}
	return nil
}

func socialAccountFromEnt(e *dbent.SocialAccount) *SocialAccount {
	platform := normalizeSocialPlatform(e.PlatformKey)
	if platform == "" {
		platform = normalizeSocialPlatform(e.Platform)
	}
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
	if e.AuthCookie != nil && strings.TrimSpace(*e.AuthCookie) != "" {
		a.AuthCookie = e.AuthCookie
	}
	if e.ExecutionAuth != nil && strings.TrimSpace(*e.ExecutionAuth) != "" {
		a.ExecutionAuth = e.ExecutionAuth
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
	if e.Edges.AssignedUser != nil {
		if email := strings.TrimSpace(e.Edges.AssignedUser.Email); email != "" {
			a.AssignedUserEmail = &email
		}
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
	if !e.Payload.IsZero() {
		payload := e.Payload
		l.Payload = &payload
	}
	if !e.TemplateSnapshot.IsZero() {
		snapshot := e.TemplateSnapshot
		l.TemplateSnapshot = &snapshot
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

func cloneSocialTaskPayload(payload *domain.SocialTaskPayload) *domain.SocialTaskPayload {
	if payload == nil || payload.IsZero() {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var cloned domain.SocialTaskPayload
	if err := json.Unmarshal(raw, &cloned); err != nil || cloned.IsZero() {
		return nil
	}
	return &cloned
}

func cloneSocialTaskTemplateSnapshot(snapshot *domain.SocialTaskTemplateSnapshot) *domain.SocialTaskTemplateSnapshot {
	if snapshot == nil || snapshot.IsZero() {
		return nil
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil
	}
	var cloned domain.SocialTaskTemplateSnapshot
	if err := json.Unmarshal(raw, &cloned); err != nil || cloned.IsZero() {
		return nil
	}
	return &cloned
}
