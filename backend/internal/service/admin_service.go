package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/authidentity"
	"github.com/Wei-Shaw/socialops/ent/authidentitychannel"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

const (
	adminBalanceAdjustmentType     = "admin_balance"
	adminConcurrencyAdjustmentType = "admin_concurrency"
)

type adminServiceImpl struct {
	userRepo             UserRepository
	groupRepo            GroupRepository
	apiKeyRepo           APIKeyRepository
	redeemCodeRepo       RedeemCodeRepository
	userGroupRateRepo    UserGroupRateRepository
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	entClient            *dbent.Client
	settingService       *SettingService
	defaultSubAssigner   DefaultSubscriptionAssigner
	userSubRepo          UserSubscriptionRepository
}

func NewAdminService(
	userRepo UserRepository,
	groupRepo GroupRepository,
	apiKeyRepo APIKeyRepository,
	redeemCodeRepo RedeemCodeRepository,
	userGroupRateRepo UserGroupRateRepository,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	entClient *dbent.Client,
	settingService *SettingService,
	defaultSubAssigner DefaultSubscriptionAssigner,
	userSubRepo UserSubscriptionRepository,
) AdminService {
	return &adminServiceImpl{
		userRepo:             userRepo,
		groupRepo:            groupRepo,
		apiKeyRepo:           apiKeyRepo,
		redeemCodeRepo:       redeemCodeRepo,
		userGroupRateRepo:    userGroupRateRepo,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
		entClient:            entClient,
		settingService:       settingService,
		defaultSubAssigner:   defaultSubAssigner,
		userSubRepo:          userSubRepo,
	}
}

func (s *adminServiceImpl) ListUsers(ctx context.Context, page, pageSize int, filters UserListFilters, sortBy, sortOrder string) ([]User, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder}
	users, result, err := s.userRepo.ListWithFilters(ctx, params, filters)
	if err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return users, result.Total, nil
	}

	userIDs := make([]int64, 0, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
	}
	if latest, err := s.userRepo.GetLatestUsedAtByUserIDs(ctx, userIDs); err == nil {
		for i := range users {
			users[i].LastUsedAt = latest[users[i].ID]
		}
	}
	s.loadUserGroupRates(ctx, users)
	return users, result.Total, nil
}

func (s *adminServiceImpl) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if latest, err := s.userRepo.GetLatestUsedAtByUserID(ctx, id); err == nil {
		user.LastUsedAt = latest
	}
	if s.userGroupRateRepo != nil {
		if rates, err := s.userGroupRateRepo.GetByUserID(ctx, id); err == nil {
			user.GroupRates = rates
		}
	}
	return user, nil
}

func (s *adminServiceImpl) CreateUser(ctx context.Context, input *CreateUserInput) (*User, error) {
	if input == nil {
		return nil, ErrInvalidInput
	}
	user := &User{
		Email:         strings.TrimSpace(input.Email),
		Username:      strings.TrimSpace(input.Username),
		Notes:         input.Notes,
		Role:          RoleUser,
		Balance:       input.Balance,
		Concurrency:   input.Concurrency,
		RPMLimit:      input.RPMLimit,
		Status:        StatusActive,
		AllowedGroups: input.AllowedGroups,
	}
	if err := user.SetPassword(input.Password); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.assignDefaultSubscriptions(ctx, user.ID)
	return user, nil
}

func (s *adminServiceImpl) UpdateUser(ctx context.Context, id int64, input *UpdateUserInput) (*User, error) {
	if input == nil {
		return nil, ErrInvalidInput
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	oldConcurrency := user.Concurrency
	oldStatus := user.Status
	oldRPMLimit := user.RPMLimit

	if user.Role == RoleAdmin && input.Status == StatusDisabled {
		return nil, infraerrors.BadRequest("ADMIN_DISABLE_FORBIDDEN", "cannot disable admin user")
	}
	if strings.TrimSpace(input.Email) != "" {
		user.Email = strings.TrimSpace(input.Email)
	}
	if input.Password != "" {
		if err := user.SetPassword(input.Password); err != nil {
			return nil, err
		}
		user.TokenVersion++
	}
	if input.Username != nil {
		user.Username = strings.TrimSpace(*input.Username)
	}
	if input.Notes != nil {
		user.Notes = *input.Notes
	}
	if input.Balance != nil {
		user.Balance = *input.Balance
	}
	if input.Concurrency != nil {
		user.Concurrency = *input.Concurrency
	}
	if input.RPMLimit != nil {
		user.RPMLimit = *input.RPMLimit
	}
	if strings.TrimSpace(input.Status) != "" {
		user.Status = strings.TrimSpace(input.Status)
	}
	if input.AllowedGroups != nil {
		user.AllowedGroups = *input.AllowedGroups
	}
	if input.GroupRates != nil {
		for groupID, rate := range input.GroupRates {
			if groupID <= 0 {
				return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be greater than zero")
			}
			if rate != nil && *rate <= 0 {
				return nil, infraerrors.BadRequest("INVALID_GROUP_RATE", "rate_multiplier must be greater than zero")
			}
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	if input.GroupRates != nil && s.userGroupRateRepo != nil {
		if err := s.userGroupRateRepo.SyncUserGroupRates(ctx, user.ID, input.GroupRates); err != nil {
			return nil, err
		}
	}
	authCacheNeedsInvalidation := oldConcurrency != user.Concurrency ||
		oldStatus != user.Status ||
		oldRPMLimit != user.RPMLimit ||
		input.Password != "" ||
		input.AllowedGroups != nil ||
		(input.GroupRates != nil && s.userGroupRateRepo != nil)
	if s.authCacheInvalidator != nil && authCacheNeedsInvalidation {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)
	}
	if oldConcurrency != user.Concurrency && s.redeemCodeRepo != nil {
		_ = s.createAdjustmentRecord(ctx, user.ID, adminConcurrencyAdjustmentType, float64(user.Concurrency-oldConcurrency), "")
	}
	return user, nil
}

func (s *adminServiceImpl) DeleteUser(ctx context.Context, id int64) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.Role == RoleAdmin {
		return infraerrors.BadRequest("ADMIN_DELETE_FORBIDDEN", "cannot delete admin user")
	}
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, id)
	}
	return nil
}

func (s *adminServiceImpl) UpdateUserBalance(ctx context.Context, userID int64, balance float64, operation string, notes string) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	oldBalance := user.Balance
	switch operation {
	case "set":
		user.Balance = balance
	case "add":
		user.Balance += balance
	case "subtract":
		user.Balance -= balance
	default:
		return nil, infraerrors.BadRequest("INVALID_BALANCE_OPERATION", "operation must be set, add, or subtract")
	}
	if user.Balance < 0 {
		return nil, infraerrors.BadRequest("BALANCE_NEGATIVE", "balance cannot be negative")
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	diff := user.Balance - oldBalance
	if diff != 0 {
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
		if s.billingCacheService != nil {
			_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
		}
		_ = s.createAdjustmentRecord(ctx, userID, adminBalanceAdjustmentType, diff, notes)
	}
	return user, nil
}

func (s *adminServiceImpl) BatchUpdateConcurrency(ctx context.Context, userIDs []int64, value int, mode string) (int, error) {
	cleaned := cleanPositiveIDs(userIDs)
	if len(cleaned) == 0 {
		return 0, nil
	}
	beforeByUserID := s.snapshotUserConcurrency(ctx, cleaned)
	var (
		affected int
		err      error
	)
	switch mode {
	case "set":
		affected, err = s.userRepo.BatchSetConcurrency(ctx, cleaned, value)
	case "add":
		affected, err = s.userRepo.BatchAddConcurrency(ctx, cleaned, value)
	default:
		return 0, infraerrors.BadRequest("INVALID_CONCURRENCY_MODE", "mode must be set or add")
	}
	if err != nil {
		return 0, err
	}
	if s.authCacheInvalidator != nil {
		for _, userID := range cleaned {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
	}
	s.createBatchConcurrencyAdjustmentRecords(ctx, beforeByUserID, value, mode)
	return affected, nil
}

func (s *adminServiceImpl) GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int, sortBy, sortOrder string) ([]APIKey, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder}
	keys, result, err := s.apiKeyRepo.ListByUserID(ctx, userID, params, APIKeyListFilters{})
	if err != nil {
		return nil, 0, err
	}
	return keys, result.Total, nil
}

func (s *adminServiceImpl) GetUserUsageStats(ctx context.Context, userID int64, period string) (any, error) {
	if s.entClient == nil {
		return map[string]any{"period": period, "total_requests": 0, "total_cost": 0.0}, nil
	}
	var totalRequests int64
	var chargedAmount float64
	rows, err := s.entClient.QueryContext(ctx, `
SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'success' AND charge_status = 'charged' THEN charged_amount ELSE 0 END), 0)::double precision
FROM social_task_logs
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		if err := rows.Scan(&totalRequests, &chargedAmount); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"period":         period,
		"total_requests": totalRequests,
		"total_cost":     chargedAmount,
	}, nil
}

func (s *adminServiceImpl) GetUserRPMStatus(ctx context.Context, userID int64) (*UserRPMStatus, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	keys, _, err := s.GetUserAPIKeys(ctx, userID, 1, 1000, "id", "asc")
	if err != nil {
		return nil, err
	}
	groupIDs := make(map[int64]struct{})
	for i := range keys {
		if keys[i].GroupID != nil && *keys[i].GroupID > 0 {
			groupIDs[*keys[i].GroupID] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(groupIDs))
	for id := range groupIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	perGroup := make([]UserGroupRPMStatus, 0, len(ids))
	for _, groupID := range ids {
		entry := UserGroupRPMStatus{GroupID: groupID, Source: "group"}
		if s.groupRepo != nil {
			if group, err := s.groupRepo.GetByIDLite(ctx, groupID); err == nil && group != nil {
				entry.GroupName = group.Name
				entry.Limit = group.RPMLimit
			}
		}
		if s.userGroupRateRepo != nil {
			if override, err := s.userGroupRateRepo.GetRPMOverrideByUserAndGroup(ctx, userID, groupID); err == nil && override != nil {
				entry.Limit = *override
				entry.Source = "override"
			}
		}
		perGroup = append(perGroup, entry)
	}
	return &UserRPMStatus{UserRPMLimit: user.RPMLimit, PerGroup: perGroup}, nil
}

func (s *adminServiceImpl) GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]RedeemCode, int64, float64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	if codeType == RedeemTypeAffiliateBalance {
		codes, total, err := s.listAffiliateBalanceHistory(ctx, userID, params)
		if err != nil {
			return nil, 0, 0, err
		}
		totalRecharged, err := s.redeemCodeRepo.SumPositiveBalanceByUser(ctx, userID)
		if err != nil {
			return nil, 0, 0, err
		}
		return codes, total, totalRecharged, nil
	}
	if codeType == "" {
		return s.getAllUserBalanceHistory(ctx, userID, params)
	}
	codes, result, err := s.redeemCodeRepo.ListByUserPaginated(ctx, userID, params, codeType)
	if err != nil {
		return nil, 0, 0, err
	}
	totalRecharged, err := s.redeemCodeRepo.SumPositiveBalanceByUser(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	return codes, result.Total, totalRecharged, nil
}

func (s *adminServiceImpl) BindUserAuthIdentity(ctx context.Context, userID int64, input AdminBindAuthIdentityInput) (*AdminBoundAuthIdentity, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "user_id must be greater than zero")
	}
	if s.entClient == nil {
		return nil, infraerrors.InternalServer("AUTH_IDENTITY_BIND_UNAVAILABLE", "auth identity binding is unavailable")
	}
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	providerType := normalizeAdminProviderType(input.ProviderType)
	providerKey := strings.TrimSpace(input.ProviderKey)
	providerSubject := strings.TrimSpace(input.ProviderSubject)
	if providerType == "" || providerKey == "" || providerSubject == "" {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "provider_type, provider_key, and provider_subject are required")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	identity, err := tx.AuthIdentity.Query().
		Where(
			authidentity.ProviderTypeEQ(providerType),
			authidentity.ProviderKeyEQ(providerKey),
			authidentity.ProviderSubjectEQ(providerSubject),
		).
		Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	verifiedAt := time.Now().UTC()
	if identity != nil && identity.UserID != userID {
		return nil, infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
	}
	if identity == nil {
		create := tx.AuthIdentity.Create().
			SetUserID(userID).
			SetProviderType(providerType).
			SetProviderKey(providerKey).
			SetProviderSubject(providerSubject).
			SetVerifiedAt(verifiedAt)
		if input.Issuer != nil && strings.TrimSpace(*input.Issuer) != "" {
			create.SetIssuer(strings.TrimSpace(*input.Issuer))
		}
		if input.Metadata != nil {
			create.SetMetadata(cloneStringAnyMap(input.Metadata))
		}
		identity, err = create.Save(ctx)
	} else {
		update := tx.AuthIdentity.UpdateOneID(identity.ID).SetVerifiedAt(verifiedAt)
		if input.Issuer != nil && strings.TrimSpace(*input.Issuer) != "" {
			update.SetIssuer(strings.TrimSpace(*input.Issuer))
		}
		if input.Metadata != nil {
			update.SetMetadata(cloneStringAnyMap(input.Metadata))
		}
		identity, err = update.Save(ctx)
	}
	if err != nil {
		return nil, err
	}

	var channel *dbent.AuthIdentityChannel
	if input.Channel != nil {
		channel, err = bindAdminAuthIdentityChannel(ctx, tx, identity.ID, providerType, providerKey, input.Channel)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return adminBoundAuthIdentityFromEnt(identity, channel), nil
}

func (s *adminServiceImpl) ReplaceUserGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (*ReplaceUserGroupResult, error) {
	if oldGroupID == newGroupID {
		return nil, infraerrors.BadRequest("SAME_GROUP", "old and new group must be different")
	}
	newGroup, err := s.groupRepo.GetByID(ctx, newGroupID)
	if err != nil {
		return nil, err
	}
	if !newGroup.IsActive() || !newGroup.IsExclusive || newGroup.IsSubscriptionType() {
		return nil, infraerrors.BadRequest("INVALID_TARGET_GROUP", "target group must be an active exclusive standard group")
	}
	if s.entClient == nil {
		return nil, infraerrors.InternalServer("GROUP_REPLACE_UNAVAILABLE", "group replacement requires transaction support")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	opCtx := dbent.NewTxContext(ctx, tx)

	if err := s.userRepo.AddGroupToAllowedGroups(opCtx, userID, newGroupID); err != nil {
		return nil, err
	}
	migrated, err := s.apiKeyRepo.UpdateGroupIDByUserAndGroup(opCtx, userID, oldGroupID, newGroupID)
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.RemoveGroupFromUserAllowedGroups(opCtx, userID, oldGroupID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return &ReplaceUserGroupResult{MigratedKeys: migrated}, nil
}

func (s *adminServiceImpl) ListRedeemCodes(ctx context.Context, page, pageSize int, codeType, status, search string, sortBy, sortOrder string) ([]RedeemCode, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder}
	codes, result, err := s.redeemCodeRepo.ListWithFilters(ctx, params, codeType, status, search)
	if err != nil {
		return nil, 0, err
	}
	return codes, result.Total, nil
}

func (s *adminServiceImpl) GetRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	return s.redeemCodeRepo.GetByID(ctx, id)
}

func (s *adminServiceImpl) GenerateRedeemCodes(ctx context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error) {
	if input == nil {
		return nil, ErrInvalidInput
	}
	if input.Count <= 0 {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_COUNT", "count must be greater than zero")
	}
	if input.Count > 1000 {
		return nil, infraerrors.BadRequest("INVALID_REDEEM_COUNT", "count cannot exceed 1000")
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return nil, ErrRedeemCodeExpired
	}
	codeType, err := normalizeRedeemCodeCreationType(input.Type)
	if err != nil {
		return nil, err
	}
	redeemValue := input.Value
	if codeType == RedeemTypeSubscription || codeType == RedeemTypeInvitation {
		redeemValue = 0
	}
	var subscriptionTarget *generateRedeemSubscriptionTarget
	if codeType == RedeemTypeSubscription {
		target, err := s.resolveGenerateRedeemSubscriptionTarget(ctx, input)
		if err != nil {
			return nil, err
		}
		subscriptionTarget = target
	}
	codes := make([]RedeemCode, 0, input.Count)
	for i := 0; i < input.Count; i++ {
		value, err := GenerateRedeemCode()
		if err != nil {
			return nil, err
		}
		code := RedeemCode{
			Code:         value,
			Type:         codeType,
			Value:        redeemValue,
			Status:       StatusUnused,
			GroupID:      input.GroupID,
			PlanID:       input.PlanID,
			ValidityDays: input.ValidityDays,
			ExpiresAt:    input.ExpiresAt,
		}
		if subscriptionTarget != nil {
			code.GroupID = subscriptionTarget.GroupID
			code.PlanID = subscriptionTarget.PlanID
			code.ValidityDays = subscriptionTarget.ValidityDays
		}
		if err := s.redeemCodeRepo.Create(ctx, &code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

type generateRedeemSubscriptionTarget struct {
	GroupID      *int64
	PlanID       *int64
	ValidityDays int
}

func (s *adminServiceImpl) resolveGenerateRedeemSubscriptionTarget(ctx context.Context, input *GenerateRedeemCodesInput) (*generateRedeemSubscriptionTarget, error) {
	if input.PlanID != nil && *input.PlanID <= 0 {
		return nil, infraerrors.BadRequest("PLAN_REQUIRED", "plan_id must be positive")
	}
	if input.GroupID != nil && *input.GroupID <= 0 {
		return nil, infraerrors.BadRequest("GROUP_REQUIRED", "group_id must be positive")
	}

	if input.PlanID != nil {
		if s.entClient == nil {
			return nil, infraerrors.InternalServer("PLAN_CATALOG_UNAVAILABLE", "subscription plan catalog is unavailable")
		}
		plan, err := s.entClient.SubscriptionPlan.Get(ctx, *input.PlanID)
		if err != nil {
			return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
		}
		groupID := plan.GroupID
		if input.GroupID != nil && *input.GroupID != groupID {
			return nil, infraerrors.BadRequest("PLAN_GROUP_MISMATCH", "plan_id does not match group_id")
		}
		if err := s.validateRedeemSubscriptionGroup(ctx, groupID); err != nil {
			return nil, err
		}
		planID := plan.ID
		validityDays := input.ValidityDays
		if validityDays == 0 {
			validityDays = plan.ValidityDays
		}
		if validityDays == 0 {
			validityDays = 30
		}
		return &generateRedeemSubscriptionTarget{GroupID: &groupID, PlanID: &planID, ValidityDays: validityDays}, nil
	}

	if input.GroupID == nil {
		return nil, infraerrors.BadRequest("SUBSCRIPTION_PACKAGE_REQUIRED", "plan_id or group_id is required for subscription redeem codes")
	}
	if err := s.validateRedeemSubscriptionGroup(ctx, *input.GroupID); err != nil {
		return nil, err
	}
	validityDays := input.ValidityDays
	if validityDays == 0 {
		validityDays = 30
	}
	return &generateRedeemSubscriptionTarget{GroupID: input.GroupID, ValidityDays: validityDays}, nil
}

func (s *adminServiceImpl) validateRedeemSubscriptionGroup(ctx context.Context, groupID int64) error {
	if s.groupRepo == nil {
		return infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable")
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	if !group.IsSubscriptionType() {
		return ErrGroupNotSubscriptionType
	}
	return nil
}

func (s *adminServiceImpl) DeleteRedeemCode(ctx context.Context, id int64) error {
	code, err := s.redeemCodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := ensureRedeemCodeDeletable(code); err != nil {
		return err
	}
	return s.redeemCodeRepo.Delete(ctx, id)
}

func (s *adminServiceImpl) BatchDeleteRedeemCodes(ctx context.Context, ids []int64) (int64, error) {
	cleaned := cleanPositiveIDs(ids)
	deletableIDs := make([]int64, 0, len(cleaned))
	for _, id := range cleaned {
		code, err := s.redeemCodeRepo.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrRedeemCodeNotFound) {
				continue
			}
			return 0, err
		}
		if err := ensureRedeemCodeDeletable(code); err != nil {
			return 0, err
		}
		deletableIDs = append(deletableIDs, id)
	}

	var deleted int64
	for _, id := range deletableIDs {
		if err := s.redeemCodeRepo.Delete(ctx, id); err != nil {
			return deleted, err
		} else {
			deleted++
		}
	}
	return deleted, nil
}

func (s *adminServiceImpl) ExpireRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	code, err := s.redeemCodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	code.Status = StatusExpired
	if err := s.redeemCodeRepo.Update(ctx, code); err != nil {
		return nil, err
	}
	return code, nil
}

func (s *adminServiceImpl) AdminUpdateAPIKeyGroupID(ctx context.Context, keyID int64, groupID *int64) (*AdminUpdateAPIKeyGroupIDResult, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	result := &AdminUpdateAPIKeyGroupIDResult{APIKey: apiKey}
	if groupID == nil {
		return result, nil
	}
	if *groupID < 0 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
	}
	if *groupID == 0 {
		apiKey.GroupID = nil
		apiKey.Group = nil
	} else {
		group, err := s.groupRepo.GetByID(ctx, *groupID)
		if err != nil {
			return nil, err
		}
		if !group.IsActive() {
			return nil, infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
		}
		if group.IsSubscriptionType() {
			if s.userSubRepo == nil {
				return nil, infraerrors.InternalServer("SUBSCRIPTION_REPOSITORY_UNAVAILABLE", "subscription repository is unavailable")
			}
			if _, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, apiKey.UserID, *groupID); err != nil {
				if errors.Is(err, ErrSubscriptionNotFound) {
					return nil, infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
				}
				return nil, err
			}
		}
		gid := *groupID
		apiKey.GroupID = &gid
		apiKey.Group = group
		if group.IsExclusive && !group.IsSubscriptionType() {
			if err := s.userRepo.AddGroupToAllowedGroups(ctx, apiKey.UserID, gid); err != nil {
				return nil, err
			}
			result.AutoGrantedGroupAccess = true
			result.GrantedGroupID = &gid
			result.GrantedGroupName = group.Name
		}
	}
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}
	result.APIKey = apiKey
	return result, nil
}

func (s *adminServiceImpl) AdminResetAPIKeyRateLimitUsage(ctx context.Context, keyID int64) (*APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	apiKey.Usage5h = 0
	apiKey.Usage1d = 0
	apiKey.Usage7d = 0
	apiKey.Window5hStart = nil
	apiKey.Window1dStart = nil
	apiKey.Window7dStart = nil
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateAPIKeyRateLimit(ctx, apiKey.ID)
	}
	return apiKey, nil
}

func (s *adminServiceImpl) AdminUpdateAPIKeyGroupAndRateLimitUsage(ctx context.Context, keyID int64, groupID *int64, resetRateLimitUsage bool) (*AdminUpdateAPIKeyGroupIDResult, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	result := &AdminUpdateAPIKeyGroupIDResult{APIKey: apiKey}

	if groupID != nil {
		if *groupID < 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
		}
		if *groupID == 0 {
			apiKey.GroupID = nil
			apiKey.Group = nil
		} else {
			group, err := s.groupRepo.GetByID(ctx, *groupID)
			if err != nil {
				return nil, err
			}
			if !group.IsActive() {
				return nil, infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
			}
			if group.IsSubscriptionType() {
				if s.userSubRepo == nil {
					return nil, infraerrors.InternalServer("SUBSCRIPTION_REPOSITORY_UNAVAILABLE", "subscription repository is unavailable")
				}
				if _, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, apiKey.UserID, *groupID); err != nil {
					if errors.Is(err, ErrSubscriptionNotFound) {
						return nil, infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
					}
					return nil, err
				}
			}
			gid := *groupID
			apiKey.GroupID = &gid
			apiKey.Group = group
			if group.IsExclusive && !group.IsSubscriptionType() {
				if err := s.userRepo.AddGroupToAllowedGroups(ctx, apiKey.UserID, gid); err != nil {
					return nil, err
				}
				result.AutoGrantedGroupAccess = true
				result.GrantedGroupID = &gid
				result.GrantedGroupName = group.Name
			}
		}
	}

	if resetRateLimitUsage {
		apiKey.Usage5h = 0
		apiKey.Usage1d = 0
		apiKey.Usage7d = 0
		apiKey.Window5hStart = nil
		apiKey.Window1dStart = nil
		apiKey.Window7dStart = nil
	}
	if groupID == nil && !resetRateLimitUsage {
		return result, nil
	}
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}
	if resetRateLimitUsage && s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateAPIKeyRateLimit(ctx, apiKey.ID)
	}
	result.APIKey = apiKey
	return result, nil
}

func (s *adminServiceImpl) assignDefaultSubscriptions(ctx context.Context, userID int64) {
	if s.settingService == nil || s.defaultSubAssigner == nil || userID <= 0 {
		return
	}
	for _, item := range s.settingService.GetDefaultSubscriptions(ctx) {
		if (item.GroupID <= 0 && item.PlanID <= 0) || item.ValidityDays == 0 {
			continue
		}
		_, _, _ = s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      item.GroupID,
			PlanID:       positiveInt64Ptr(item.PlanID),
			ValidityDays: item.ValidityDays,
			Notes:        "auto assigned by default user subscriptions setting",
		})
	}
}

func (s *adminServiceImpl) loadUserGroupRates(ctx context.Context, users []User) {
	if s.userGroupRateRepo == nil {
		return
	}
	for i := range users {
		if rates, err := s.userGroupRateRepo.GetByUserID(ctx, users[i].ID); err == nil {
			users[i].GroupRates = rates
		}
	}
}

func (s *adminServiceImpl) createAdjustmentRecord(ctx context.Context, userID int64, codeType string, value float64, notes string) error {
	if s.redeemCodeRepo == nil || value == 0 {
		return nil
	}
	code, err := GenerateRedeemCode()
	if err != nil {
		return err
	}
	now := time.Now()
	record := &RedeemCode{
		Code:   code,
		Type:   codeType,
		Value:  value,
		Status: StatusUsed,
		UsedBy: &userID,
		UsedAt: &now,
		Notes:  notes,
	}
	return s.redeemCodeRepo.Create(ctx, record)
}

func cleanPositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *adminServiceImpl) snapshotUserConcurrency(ctx context.Context, userIDs []int64) map[int64]int {
	out := make(map[int64]int, len(userIDs))
	if s == nil || s.userRepo == nil {
		return out
	}
	for _, userID := range userIDs {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user == nil {
			continue
		}
		out[userID] = user.Concurrency
	}
	return out
}

func (s *adminServiceImpl) createBatchConcurrencyAdjustmentRecords(ctx context.Context, beforeByUserID map[int64]int, value int, mode string) {
	for userID, before := range beforeByUserID {
		after := before
		switch mode {
		case "set":
			after = value
			if after < 0 {
				after = 0
			}
		case "add":
			after = before + value
			if after < 0 {
				after = 0
			}
		default:
			continue
		}
		diff := after - before
		if diff == 0 {
			continue
		}
		_ = s.createAdjustmentRecord(ctx, userID, adminConcurrencyAdjustmentType, float64(diff), "")
	}
}

func normalizeAdminProviderType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "email", "github", "google", "linuxdo", "oidc", "wechat", "dingtalk":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func bindAdminAuthIdentityChannel(ctx context.Context, tx *dbent.Tx, identityID int64, providerType, providerKey string, input *AdminBindAuthIdentityChannelInput) (*dbent.AuthIdentityChannel, error) {
	channel := strings.TrimSpace(input.Channel)
	appID := strings.TrimSpace(input.ChannelAppID)
	subject := strings.TrimSpace(input.ChannelSubject)
	if channel == "" || appID == "" || subject == "" {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "channel, channel_app_id, and channel_subject are required")
	}

	existing, err := tx.AuthIdentityChannel.Query().
		Where(
			authidentitychannel.ProviderTypeEQ(providerType),
			authidentitychannel.ProviderKeyEQ(providerKey),
			authidentitychannel.ChannelEQ(channel),
			authidentitychannel.ChannelAppIDEQ(appID),
			authidentitychannel.ChannelSubjectEQ(subject),
		).
		Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	if existing != nil && existing.IdentityID != identityID {
		return nil, infraerrors.Conflict("AUTH_IDENTITY_CHANNEL_CONFLICT", "auth identity channel already belongs to another identity")
	}
	if existing == nil {
		create := tx.AuthIdentityChannel.Create().
			SetIdentityID(identityID).
			SetProviderType(providerType).
			SetProviderKey(providerKey).
			SetChannel(channel).
			SetChannelAppID(appID).
			SetChannelSubject(subject)
		if input.Metadata != nil {
			create.SetMetadata(cloneStringAnyMap(input.Metadata))
		}
		return create.Save(ctx)
	}
	update := tx.AuthIdentityChannel.UpdateOneID(existing.ID)
	if input.Metadata != nil {
		update.SetMetadata(cloneStringAnyMap(input.Metadata))
	}
	return update.Save(ctx)
}

func adminBoundAuthIdentityFromEnt(identity *dbent.AuthIdentity, channel *dbent.AuthIdentityChannel) *AdminBoundAuthIdentity {
	if identity == nil {
		return nil
	}
	out := &AdminBoundAuthIdentity{
		UserID:          identity.UserID,
		ProviderType:    identity.ProviderType,
		ProviderKey:     identity.ProviderKey,
		ProviderSubject: identity.ProviderSubject,
		VerifiedAt:      identity.VerifiedAt,
		Issuer:          identity.Issuer,
		Metadata:        cloneStringAnyMap(identity.Metadata),
		CreatedAt:       identity.CreatedAt,
		UpdatedAt:       identity.UpdatedAt,
	}
	if channel != nil {
		out.Channel = &AdminBoundAuthIdentityChannel{
			Channel:        channel.Channel,
			ChannelAppID:   channel.ChannelAppID,
			ChannelSubject: channel.ChannelSubject,
			Metadata:       cloneStringAnyMap(channel.Metadata),
			CreatedAt:      channel.CreatedAt,
			UpdatedAt:      channel.UpdatedAt,
		}
	}
	return out
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
