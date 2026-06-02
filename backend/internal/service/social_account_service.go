package service

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
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

// SocialAccountSource constants
const (
	SocialAccountSourceRegistered   = "registered"
	SocialAccountSourceManualImport = "manual_import"
	SocialAccountSourceFileUpload   = "file_upload"
)

var (
	ErrSocialAccountAlreadyAssigned   = infraerrors.Conflict("SOCIAL_ACCOUNT_ALREADY_ASSIGNED", "social account is already assigned")
	ErrSocialAccountAssignmentChanged = infraerrors.Conflict("SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED", "social account assignment changed; retry with the latest state")
	ErrSocialAccountDuplicate         = infraerrors.Conflict("SOCIAL_ACCOUNT_DUPLICATE", "social account already exists in the total pool")
	ErrSocialAccountImportNotFound    = infraerrors.NotFound("SOCIAL_ACCOUNT_POOL_MATCH_NOT_FOUND", "no unassigned total-pool social account matches this platform username")
	ErrSocialAccountImportAmbiguous   = infraerrors.Conflict("SOCIAL_ACCOUNT_POOL_MATCH_AMBIGUOUS", "multiple unassigned total-pool social accounts match this username")
	ErrSocialTaskUnsupportedAction    = infraerrors.BadRequest("SOCIAL_TASK_UNSUPPORTED_ACTION", "unsupported social task action")
	ErrSocialAccountNotAssigned       = infraerrors.BadRequest("SOCIAL_ACCOUNT_NOT_ASSIGNED", "social account is not assigned to this user")
	ErrSocialAccountDefaultProxyRoute = infraerrors.BadRequest("SOCIAL_ACCOUNT_DEFAULT_PROXY_ROUTE_REQUIRED", "use the default proxy endpoint to set an account execution proxy")
	ErrSocialIPOwnerMismatch          = infraerrors.BadRequest("SOCIAL_IP_OWNER_MISMATCH", "social IP does not belong to the account owner")
)

// SocialAccount represents a social media account.
type SocialAccount struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Platform       string    `json:"platform"`
	AccountID      *string   `json:"account_id,omitempty"`
	Password       *string   `json:"password,omitempty"`
	Phone          *string   `json:"phone,omitempty"`
	Email          *string   `json:"email,omitempty"`
	EmailPassword  *string   `json:"email_password,omitempty"`
	AccountStatus  string    `json:"account_status"`
	TaskStatus     string    `json:"task_status"`
	TaskMessage    *string   `json:"task_message,omitempty"`
	Source         string    `json:"source"`
	BoundIP        *string   `json:"bound_ip,omitempty"`
	AssignedUserID *int64    `json:"assigned_user_id,omitempty"`
	Remark         *string   `json:"remark,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	Name          string  `json:"name" binding:"required"`
	Platform      string  `json:"platform" binding:"required"`
	AccountID     *string `json:"account_id"`
	Password      *string `json:"password"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	EmailPassword *string `json:"email_password"`
	Source        string  `json:"source"`
	BoundIP       *string `json:"bound_ip"`
	Remark        *string `json:"remark"`
}

// UpdateSocialAccountInput is the input for updating a social account.
type UpdateSocialAccountInput struct {
	Name          *string `json:"name"`
	AccountID     *string `json:"account_id"`
	Password      *string `json:"password"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	EmailPassword *string `json:"email_password"`
	AccountStatus *string `json:"account_status"`
	TaskStatus    *string `json:"task_status"`
	TaskMessage   *string `json:"task_message"`
	BoundIP       *string `json:"bound_ip"`
	Remark        *string `json:"remark"`
}

// SocialAccountListFilters holds filters for listing social accounts.
type SocialAccountListFilters struct {
	Platform       string
	AccountStatus  string
	TaskStatus     string
	Source         string
	AssignedOnly   bool
	UnassignedOnly bool
	Search         string
}

type UserImportSocialAccountInput struct {
	Platform string `json:"platform"`
	Name     string `json:"name" binding:"required"`
}

type SocialAccountImportResult struct {
	Total   int      `json:"total"`
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

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
	nameKey := normalizeSocialUsername(name)
	exists, err := s.poolAccountExists(ctx, platform, nameKey)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSocialAccountDuplicate
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = SocialAccountSourceManualImport
	}
	q := s.entClient.SocialAccount.Create().
		SetName(name).
		SetPlatform(platform).
		SetPlatformKey(platform).
		SetNameKey(nameKey).
		SetSource(source).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending)

	if input.AccountID != nil {
		q.SetAccountID(strings.TrimSpace(*input.AccountID))
	}
	if input.Password != nil {
		q.SetPassword(*input.Password)
	}
	if input.AccountID != nil {
		q.SetAccountID(strings.TrimSpace(*input.AccountID))
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
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
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

	if filters.Platform != "" {
		q = q.Where(socialaccount.PlatformEqualFold(strings.TrimSpace(filters.Platform)))
	}
	if filters.AccountStatus != "" {
		q = q.Where(socialaccount.AccountStatusEQ(filters.AccountStatus))
	}
	if filters.TaskStatus != "" {
		q = q.Where(socialaccount.TaskStatusEQ(filters.TaskStatus))
	}
	if filters.Source != "" {
		q = q.Where(socialaccount.SourceEQ(filters.Source))
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
			socialaccount.AccountIDContainsFold(search),
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

// Update updates a social account.
func (s *SocialAccountService) Update(ctx context.Context, id int64, input *UpdateSocialAccountInput) (*SocialAccount, error) {
	q := s.entClient.SocialAccount.UpdateOneID(id)

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		current, err := s.entClient.SocialAccount.Get(ctx, id)
		if err != nil {
			if dbent.IsNotFound(err) {
				return nil, infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
			}
			return nil, err
		}
		nameKey := normalizeSocialUsername(name)
		exists, err := s.poolAccountExistsExcept(ctx, current.PlatformKey, nameKey, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrSocialAccountDuplicate
		}
		q.SetName(name).SetNameKey(nameKey)
	}
	if input.AccountID != nil {
		q.SetAccountID(strings.TrimSpace(*input.AccountID))
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
	if input.AccountStatus != nil {
		q.SetAccountStatus(*input.AccountStatus)
	}
	if input.TaskStatus != nil {
		q.SetTaskStatus(*input.TaskStatus)
	}
	if input.TaskMessage != nil {
		q.SetTaskMessage(*input.TaskMessage)
	}
	if input.BoundIP != nil {
		if strings.TrimSpace(*input.BoundIP) != "" {
			return nil, ErrSocialAccountDefaultProxyRoute
		}
		q.ClearBoundIP()
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
	err := s.entClient.SocialAccount.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("SOCIAL_ACCOUNT_NOT_FOUND", "social account not found")
		}
		return err
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
		ClearBoundIP().
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
	return s.setDefaultProxySnapshot(ctx, accountID, snapshot)
}

func (s *SocialAccountService) SetDefaultProxyForAdmin(ctx context.Context, accountID int64, ip *SocialIP) (*SocialAccount, error) {
	account, err := s.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if ip == nil {
		return s.setDefaultProxySnapshot(ctx, accountID, nil)
	}
	if account.AssignedUserID == nil {
		return nil, ErrSocialAccountNotAssigned
	}
	snapshot, err := defaultProxySnapshotForOwner(ip, *account.AssignedUserID)
	if err != nil {
		return nil, err
	}
	return s.setDefaultProxySnapshot(ctx, accountID, snapshot)
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
	snapshot := SocialIPTaskSnapshot(ip)
	return &snapshot, nil
}

func (s *SocialAccountService) setDefaultProxySnapshot(ctx context.Context, accountID int64, snapshot *string) (*SocialAccount, error) {
	q := s.entClient.SocialAccount.UpdateOneID(accountID)
	if snapshot == nil || strings.TrimSpace(*snapshot) == "" {
		q.ClearBoundIP()
	} else {
		q.SetBoundIP(strings.TrimSpace(*snapshot))
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
	for _, input := range inputs {
		if input == nil {
			result.Skipped++
			result.Errors = append(result.Errors, "nil input")
			continue
		}
		platform := normalizeSocialPlatform(input.Platform)
		name := normalizeSocialUsername(input.Name)
		if platform == "" || name == "" {
			result.Skipped++
			result.Errors = append(result.Errors, "missing platform or name")
			continue
		}
		exists, err := s.poolAccountExists(ctx, platform, name)
		if err != nil {
			return result, err
		}
		if exists {
			result.Skipped++
			continue
		}
		input.Platform = platform
		if strings.TrimSpace(input.Source) == "" {
			input.Source = SocialAccountSourceFileUpload
		}
		if _, err := s.createPoolAccount(ctx, input); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, strings.TrimSpace(input.Name)+": "+err.Error())
			continue
		}
		result.Created++
	}
	return result, nil
}

func (s *SocialAccountService) createPoolAccount(ctx context.Context, input *CreateSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	name := strings.TrimSpace(input.Name)
	platform := normalizeSocialPlatform(input.Platform)
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = SocialAccountSourceManualImport
	}
	q := s.entClient.SocialAccount.Create().
		SetName(name).
		SetPlatform(platform).
		SetPlatformKey(platform).
		SetNameKey(normalizeSocialUsername(name)).
		SetSource(source).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending)

	if input.AccountID != nil {
		q.SetAccountID(strings.TrimSpace(*input.AccountID))
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
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}

	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

// ImportForUser binds an existing unassigned total-pool account by platform username.
// It never creates accounts from user-provided input.
func (s *SocialAccountService) ImportForUser(ctx context.Context, userID int64, input *UserImportSocialAccountInput) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_IMPORT_REQUIRED", "social account import input is required")
	}
	platform := normalizeSocialPlatform(input.Platform)
	name := normalizeSocialUsername(input.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}

	allMatches, err := s.findAccountsByNormalizedName(ctx, platform, name)
	if err != nil {
		return nil, err
	}
	if len(allMatches) == 0 {
		return nil, ErrSocialAccountImportNotFound
	}
	if platform == "" && len(allMatches) > 1 {
		return nil, ErrSocialAccountImportAmbiguous
	}

	matches := make([]*dbent.SocialAccount, 0, len(allMatches))
	for _, account := range allMatches {
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

// ListByUser returns social accounts assigned to a specific user.
func (s *SocialAccountService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]*SocialAccount, *pagination.PaginationResult, error) {
	q := s.entClient.SocialAccount.Query().
		Where(
			socialaccount.AssignedUserIDEQ(int64(userID)),
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

// ListTaskLogs returns task logs for a user.
func (s *SocialAccountService) ListTaskLogs(ctx context.Context, userID int64, params pagination.PaginationParams) ([]*SocialTaskLog, *pagination.PaginationResult, error) {
	q := s.entClient.SocialTaskLog.Query().
		Where(socialtasklog.UserIDEQ(userID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialtasklog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	logs := make([]*SocialTaskLog, len(ents))
	for i, e := range ents {
		logs[i] = socialTaskLogFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return logs, result, nil
}

// ListAllTaskLogs returns recent task logs across users for the admin workbench.
func (s *SocialAccountService) ListAllTaskLogs(ctx context.Context, params pagination.PaginationParams) ([]*SocialTaskLog, *pagination.PaginationResult, error) {
	q := s.entClient.SocialTaskLog.Query()

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	ents, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(socialtasklog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	logs := make([]*SocialTaskLog, len(ents))
	for i, e := range ents {
		logs[i] = socialTaskLogFromEnt(e)
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return logs, result, nil
}

// GetStats returns summary statistics for social accounts.
func (s *SocialAccountService) GetStats(ctx context.Context) (map[string]int, error) {
	total, _ := s.entClient.SocialAccount.Query().Count(ctx)
	stored, _ := s.entClient.SocialAccount.Query().
		Where(socialaccount.TaskStatusEQ(SocialTaskStatusStored)).Count(ctx)
	available, _ := s.entClient.SocialAccount.Query().
		Where(socialaccount.AccountStatusEQ(SocialAccountStatusAvailable)).Count(ctx)

	return map[string]int{
		"total":     total,
		"stored":    stored,
		"available": available,
	}, nil
}

func (s *SocialAccountService) poolAccountExists(ctx context.Context, platform, normalizedName string) (bool, error) {
	return s.poolAccountExistsExcept(ctx, platform, normalizedName, 0)
}

func (s *SocialAccountService) poolAccountExistsExcept(ctx context.Context, platform, normalizedName string, exceptID int64) (bool, error) {
	accounts, err := s.findAccountsByNormalizedName(ctx, platform, normalizedName)
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

func (s *SocialAccountService) findAccountsByNormalizedName(ctx context.Context, platform, normalizedName string) ([]*dbent.SocialAccount, error) {
	normalizedName = normalizeSocialUsername(normalizedName)
	if normalizedName == "" {
		return nil, nil
	}
	q := s.entClient.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ(normalizedName))
	if platform = normalizeSocialPlatform(platform); platform != "" {
		q = q.Where(socialaccount.PlatformKeyEQ(platform))
	}
	accounts, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func normalizeSocialPlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func normalizeSocialUsername(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.TrimLeft(normalized, "@")
	return strings.TrimSpace(normalized)
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
	default:
		return "", false
	}
}

func IsBillableSocialTaskAction(action string) bool {
	_, ok := NormalizeSocialTaskAction(action)
	return ok
}

func socialAccountFromEnt(e *dbent.SocialAccount) *SocialAccount {
	a := &SocialAccount{
		ID:            int64(e.ID),
		Name:          e.Name,
		Platform:      e.Platform,
		AccountStatus: e.AccountStatus,
		TaskStatus:    e.TaskStatus,
		Source:        e.Source,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
	if e.AccountID != nil {
		a.AccountID = e.AccountID
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
	if e.TaskMessage != nil {
		a.TaskMessage = e.TaskMessage
	}
	if e.BoundIP != nil {
		a.BoundIP = e.BoundIP
	}
	if e.AssignedUserID != nil {
		uid := int64(*e.AssignedUserID)
		a.AssignedUserID = &uid
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
