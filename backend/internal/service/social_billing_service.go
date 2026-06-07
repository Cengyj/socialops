package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/predicate"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/user"
	"github.com/Wei-Shaw/socialops/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

var ErrSocialTaskInsufficientFunds = infraerrors.BadRequest("SOCIAL_TASK_INSUFFICIENT_FUNDS", "insufficient subscription allowance and wallet balance for social task")

var errSocialTaskChargeRace = errors.New("social task charge plan changed")

const (
	socialUsageLedgerModel                  = "social-action"
	socialUsageBillingTypeSubscription int8 = 1
	socialUsageBillingTypeWallet       int8 = 2
)

type SocialBillingAffordability struct {
	UnitPrice             float64 `json:"unit_price"`
	ActionCount           int     `json:"action_count"`
	RequiredTotal         float64 `json:"required_total"`
	SubscriptionAllowance float64 `json:"subscription_allowance"`
	SubscriptionUsage     float64 `json:"subscription_usage"`
	WalletRequired        float64 `json:"wallet_required"`
	WalletBalance         float64 `json:"wallet_balance"`
	CanAfford             bool    `json:"can_afford"`
	ChargeOnSuccessOnly   bool    `json:"charge_on_success_only"`
}

type SocialBillingChargeResult struct {
	Amount             float64
	Source             string
	BillingRequestID   string
	SubscriptionAmount float64
	WalletAmount       float64
}

type SocialBillingService struct {
	userRepo            UserRepository
	userSubRepo         UserSubscriptionRepository
	groupRepo           GroupRepository
	billingCacheService *BillingCacheService
}

func NewSocialBillingService(userRepo UserRepository, userSubRepo UserSubscriptionRepository, groupRepo GroupRepository, billingCacheService *BillingCacheService) *SocialBillingService {
	return &SocialBillingService{
		userRepo:            userRepo,
		userSubRepo:         userSubRepo,
		groupRepo:           groupRepo,
		billingCacheService: billingCacheService,
	}
}

func (s *SocialBillingService) CheckAffordability(ctx context.Context, userID int64, actionCount int) (*SocialBillingAffordability, error) {
	return s.CheckAffordabilityForPlatform(ctx, userID, "", actionCount)
}

func (s *SocialBillingService) CheckAffordabilityForPlatform(ctx context.Context, userID int64, platform string, actionCount int) (*SocialBillingAffordability, error) {
	if actionCount < 0 {
		actionCount = 0
	}
	total := roundSocialAmount(float64(actionCount) * SocialTaskUnitPrice)
	check := &SocialBillingAffordability{
		UnitPrice:           SocialTaskUnitPrice,
		ActionCount:         actionCount,
		RequiredTotal:       total,
		ChargeOnSuccessOnly: true,
	}
	if total == 0 {
		check.CanAfford = true
		return check, nil
	}
	if s == nil || s.userRepo == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_BILLING_UNAVAILABLE", "social billing service is unavailable")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	check.WalletBalance = roundSocialAmount(user.Balance)

	allowance, err := s.availableSubscriptionAllowance(ctx, userID, platform, total)
	if err != nil {
		return nil, err
	}
	check.SubscriptionAllowance = roundSocialAmount(allowance)
	check.SubscriptionUsage = roundSocialAmount(math.Min(total, allowance))
	check.WalletRequired = roundSocialAmount(math.Max(0, total-check.SubscriptionUsage))
	check.CanAfford = check.WalletRequired <= check.WalletBalance+1e-9
	return check, nil
}

func (s *SocialBillingService) EnsureCanAfford(ctx context.Context, userID int64, actionCount int) (*SocialBillingAffordability, error) {
	return s.EnsureCanAffordForPlatform(ctx, userID, "", actionCount)
}

func (s *SocialBillingService) EnsureCanAffordForPlatform(ctx context.Context, userID int64, platform string, actionCount int) (*SocialBillingAffordability, error) {
	check, err := s.CheckAffordabilityForPlatform(ctx, userID, platform, actionCount)
	if err != nil {
		return nil, err
	}
	if !check.CanAfford {
		return check, ErrSocialTaskInsufficientFunds.WithMetadata(map[string]string{
			"required_total":  fmt.Sprintf("%.2f", check.RequiredTotal),
			"wallet_required": fmt.Sprintf("%.2f", check.WalletRequired),
			"wallet_balance":  fmt.Sprintf("%.2f", check.WalletBalance),
		})
	}
	return check, nil
}

func (s *SocialBillingService) ChargeSuccessfulAction(ctx context.Context, userID int64, amount float64) (*SocialBillingChargeResult, error) {
	amount = roundSocialAmount(amount)
	if amount <= 0 {
		return &SocialBillingChargeResult{Amount: 0}, nil
	}
	return nil, infraerrors.ServiceUnavailable(
		"SOCIAL_BILLING_DIRECT_CHARGE_DISABLED",
		"social task charges must be finalized by the task executor transaction",
	)
}

// FinalizeSuccessfulTask is the SocialOps settlement primitive for a successful
// account action. It keeps subscription usage, wallet fallback, task status, and
// billing cache invalidation in one atomic path so task execution never mutates
// balance/quota semantics directly. If the generic subscription or wallet
// semantics change, update this function and its success-only tests in the same
// patch; task execution should continue to call only this finalizer.
func (s *SocialBillingService) FinalizeSuccessfulTask(ctx context.Context, entClient *dbent.Client, taskLogID, userID int64, amount float64, result string) (*SocialBillingChargeResult, error) {
	for attempt := 0; attempt < 3; attempt++ {
		charge, err := s.finalizeSuccessfulTaskOnce(ctx, entClient, taskLogID, userID, amount, result)
		if errors.Is(err, errSocialTaskChargeRace) {
			continue
		}
		return charge, err
	}
	return nil, errSocialTaskChargeRace
}

func (s *SocialBillingService) finalizeSuccessfulTaskOnce(ctx context.Context, entClient *dbent.Client, taskLogID, userID int64, amount float64, result string) (*SocialBillingChargeResult, error) {
	amount = roundSocialAmount(amount)
	if s == nil || entClient == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_BILLING_UNAVAILABLE", "social billing service is unavailable")
	}
	if amount <= 0 {
		return finalizeZeroAmountSuccessfulTask(ctx, entClient, taskLogID, userID, result)
	}

	tx, err := entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	taskLog, err := tx.SocialTaskLog.Query().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.UserIDEQ(userID),
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
		).
		WithSocialAccount().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, errSocialTaskChargeRace
		}
		return nil, err
	}
	platform := socialTaskBillingPlatform(taskLog)
	if platform == "" {
		return nil, infraerrors.BadRequest("SOCIAL_TASK_PLATFORM_REQUIRED", "social task platform is required for billing")
	}

	plan, subscriptionAmount, walletAmount, err := planSocialTaskChargeInTx(ctx, tx, userID, platform, amount)
	if err != nil {
		return nil, err
	}

	settledAt := time.Now()
	billingParts := make([]string, 0, len(plan)+1)
	for _, charge := range plan {
		if charge.amount <= 0 {
			continue
		}
		predicates := []predicate.UserSubscription{
			usersubscription.IDEQ(charge.subID),
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(time.Now()),
		}
		if charge.dailyLimit != nil {
			if !charge.activateDailyWindow && !charge.resetDailyWindow {
				predicates = append(predicates, usersubscription.DailyUsageUsdLTE(roundSocialAmount(*charge.dailyLimit-charge.amount)))
			}
		}
		if charge.weeklyLimit != nil {
			if !charge.activateWeeklyWindow && !charge.resetWeeklyWindow {
				predicates = append(predicates, usersubscription.WeeklyUsageUsdLTE(roundSocialAmount(*charge.weeklyLimit-charge.amount)))
			}
		}
		if charge.monthlyLimit != nil {
			if !charge.activateMonthlyWindow && !charge.resetMonthlyWindow {
				predicates = append(predicates, usersubscription.MonthlyUsageUsdLTE(roundSocialAmount(*charge.monthlyLimit-charge.amount)))
			}
		}
		if charge.activateDailyWindow {
			predicates = append(predicates, usersubscription.DailyWindowStartIsNil())
		} else if charge.resetDailyWindow {
			predicates = append(predicates, usersubscription.DailyWindowStartLTE(*charge.resetDailyBefore))
		}
		if charge.activateWeeklyWindow {
			predicates = append(predicates, usersubscription.WeeklyWindowStartIsNil())
		} else if charge.resetWeeklyWindow {
			predicates = append(predicates, usersubscription.WeeklyWindowStartLTE(*charge.resetWeeklyBefore))
		}
		if charge.activateMonthlyWindow {
			predicates = append(predicates, usersubscription.MonthlyWindowStartIsNil())
		} else if charge.resetMonthlyWindow {
			predicates = append(predicates, usersubscription.MonthlyWindowStartLTE(*charge.resetMonthlyBefore))
		}
		update := tx.UserSubscription.Update().Where(predicates...)
		if charge.activateDailyWindow || charge.resetDailyWindow {
			update.SetDailyUsageUsd(charge.amount)
			update.SetDailyWindowStart(charge.windowStart)
		} else {
			update.AddDailyUsageUsd(charge.amount)
		}
		if charge.activateWeeklyWindow || charge.resetWeeklyWindow {
			update.SetWeeklyUsageUsd(charge.amount)
			update.SetWeeklyWindowStart(charge.windowStart)
		} else {
			update.AddWeeklyUsageUsd(charge.amount)
		}
		if charge.activateMonthlyWindow || charge.resetMonthlyWindow {
			update.SetMonthlyUsageUsd(charge.amount)
			update.SetMonthlyWindowStart(charge.windowStart)
		} else {
			update.AddMonthlyUsageUsd(charge.amount)
		}
		updated, err := update.Save(ctx)
		if err != nil {
			return nil, err
		}
		if updated == 0 {
			return nil, errSocialTaskChargeRace
		}
		billingParts = append(billingParts, fmt.Sprintf("subscription:%d:%.2f", charge.subID, charge.amount))
	}
	if walletAmount > 0 {
		updated, err := tx.User.Update().
			Where(user.IDEQ(userID), user.BalanceGTE(walletAmount)).
			AddBalance(-walletAmount).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		if updated == 0 {
			return nil, ErrSocialTaskInsufficientFunds
		}
		billingParts = append(billingParts, fmt.Sprintf("wallet:%d:%d:%.2f", userID, settledAt.UnixNano(), walletAmount))
	}

	if err := createSocialTaskUsageLedgerInTx(ctx, tx, taskLog, userID, plan, walletAmount, settledAt); err != nil {
		return nil, err
	}

	source := chargeSourceForAmounts(subscriptionAmount, walletAmount)
	update := tx.SocialTaskLog.Update().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.UserIDEQ(userID),
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
		).
		SetStatus(SocialTaskLogStatusSuccess).
		SetResultMessage(result).
		SetExecutedAt(settledAt).
		SetChargedAmount(amount).
		SetChargeStatus(SocialTaskChargeStatusCharged).
		SetBillingRequestID(strings.Join(billingParts, ","))
	if source != "" {
		update.SetChargeSource(source)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		return nil, errSocialTaskChargeRace
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true

	s.queueBillingCacheWrites(userID, plan, walletAmount)

	return &SocialBillingChargeResult{
		Amount:             amount,
		Source:             source,
		BillingRequestID:   strings.Join(billingParts, ","),
		SubscriptionAmount: subscriptionAmount,
		WalletAmount:       walletAmount,
	}, nil
}

func finalizeZeroAmountSuccessfulTask(ctx context.Context, entClient *dbent.Client, taskLogID, userID int64, result string) (*SocialBillingChargeResult, error) {
	settledAt := time.Now()
	updated, err := entClient.SocialTaskLog.Update().
		Where(
			socialtasklog.IDEQ(taskLogID),
			socialtasklog.UserIDEQ(userID),
			socialtasklog.StatusEQ(SocialTaskLogStatusRunning),
			socialtasklog.ChargeStatusEQ(SocialTaskChargeStatusNotCharged),
		).
		SetStatus(SocialTaskLogStatusSuccess).
		SetResultMessage(result).
		SetExecutedAt(settledAt).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		ClearChargeSource().
		ClearBillingRequestID().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated == 0 {
		return nil, errSocialTaskChargeRace
	}
	return &SocialBillingChargeResult{Amount: 0}, nil
}

func createSocialTaskUsageLedgerInTx(ctx context.Context, tx *dbent.Tx, taskLog *dbent.SocialTaskLog, userID int64, plan []socialTaskChargeAllocation, walletAmount float64, settledAt time.Time) error {
	if tx == nil || taskLog == nil {
		return nil
	}
	for _, charge := range plan {
		if charge.amount <= 0 {
			continue
		}
		if err := createSocialTaskUsageLedgerRow(ctx, tx, socialTaskUsageLedgerRow{
			userID:         userID,
			taskLogID:      taskLog.ID,
			source:         SocialTaskChargeSourceSubscription,
			refID:          charge.subID,
			groupID:        charge.groupID,
			subscriptionID: charge.subID,
			amount:         charge.amount,
			billingType:    socialUsageBillingTypeSubscription,
			settledAt:      settledAt,
		}); err != nil {
			return err
		}
	}
	if walletAmount <= 0 {
		return nil
	}
	return createSocialTaskUsageLedgerRow(ctx, tx, socialTaskUsageLedgerRow{
		userID:      userID,
		taskLogID:   taskLog.ID,
		source:      SocialTaskChargeSourceWallet,
		amount:      walletAmount,
		billingType: socialUsageBillingTypeWallet,
		settledAt:   settledAt,
	})
}

type socialTaskUsageLedgerRow struct {
	userID         int64
	taskLogID      int64
	source         string
	refID          int64
	groupID        int64
	subscriptionID int64
	amount         float64
	billingType    int8
	settledAt      time.Time
}

func createSocialTaskUsageLedgerRow(ctx context.Context, tx *dbent.Tx, row socialTaskUsageLedgerRow) error {
	amount := roundSocialAmount(row.amount)
	if tx == nil || row.userID <= 0 || row.taskLogID <= 0 || amount <= 0 {
		return nil
	}
	create := tx.UsageLog.Create().
		SetUserID(row.userID).
		SetRequestID(socialTaskUsageLedgerRequestID(row.taskLogID, row.source, row.refID)).
		SetModel(socialUsageLedgerModel).
		SetTotalCost(amount).
		SetActualCost(amount).
		SetBillingType(row.billingType).
		SetCreatedAt(row.settledAt)
	if row.groupID > 0 {
		create.SetGroupID(row.groupID)
	}
	if row.subscriptionID > 0 {
		create.SetSubscriptionID(row.subscriptionID)
	}
	_, err := create.Save(ctx)
	return err
}

func socialTaskUsageLedgerRequestID(taskLogID int64, source string, refID int64) string {
	source = strings.TrimSpace(source)
	if source == "" {
		source = "charge"
	}
	if refID > 0 {
		return fmt.Sprintf("social-task:%d:%s:%d", taskLogID, source, refID)
	}
	return fmt.Sprintf("social-task:%d:%s", taskLogID, source)
}

func (s *SocialBillingService) queueBillingCacheWrites(userID int64, plan []socialTaskChargeAllocation, walletAmount float64) {
	if s == nil || s.billingCacheService == nil {
		return
	}
	for _, charge := range plan {
		s.billingCacheService.QueueUpdateSubscriptionUsage(userID, charge.groupID, charge.amount)
	}
	if walletAmount > 0 {
		s.billingCacheService.QueueDeductBalance(userID, walletAmount)
	}
}

func (s *SocialBillingService) availableSubscriptionAllowance(ctx context.Context, userID int64, platform string, needed float64) (float64, error) {
	if s == nil || s.userSubRepo == nil || s.groupRepo == nil || needed <= 0 {
		return 0, nil
	}
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	total := 0.0
	now := time.Now()
	for i := range subs {
		current := serviceSubscriptionWithCurrentWindows(&subs[i], now)
		allowance, err := s.subscriptionAllowance(ctx, current, needed-total, platform)
		if err != nil {
			return 0, err
		}
		total += allowance
		if total >= needed {
			return needed, nil
		}
	}
	return total, nil
}

type socialTaskChargeAllocation struct {
	subID        int64
	groupID      int64
	amount       float64
	dailyLimit   *float64
	weeklyLimit  *float64
	monthlyLimit *float64

	activateDailyWindow   bool
	activateWeeklyWindow  bool
	activateMonthlyWindow bool
	resetDailyWindow      bool
	resetWeeklyWindow     bool
	resetMonthlyWindow    bool
	resetDailyBefore      *time.Time
	resetWeeklyBefore     *time.Time
	resetMonthlyBefore    *time.Time
	windowStart           time.Time
}

func planSocialTaskChargeInTx(ctx context.Context, tx *dbent.Tx, userID int64, platform string, amount float64) ([]socialTaskChargeAllocation, float64, float64, error) {
	now := time.Now()
	windowStart := startOfDay(now)
	subs, err := tx.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(now),
		).
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	remaining := amount
	plan := make([]socialTaskChargeAllocation, 0, len(subs))
	for _, sub := range subs {
		if sub.Edges.Group == nil {
			continue
		}
		if !socialTaskEntSubscriptionCoversPlatform(sub, platform) {
			continue
		}
		windowState := socialTaskEntSubscriptionWindowState(sub, now)
		allowance := socialTaskEntSubscriptionAllowance(sub, remaining, windowState)
		allowance = roundSocialAmount(allowance)
		if allowance <= 0 {
			continue
		}
		plan = append(plan, socialTaskChargeAllocation{
			subID:        sub.ID,
			groupID:      sub.GroupID,
			amount:       allowance,
			dailyLimit:   copyFloat64Ptr(subscriptionEntDailyLimit(sub)),
			weeklyLimit:  copyFloat64Ptr(subscriptionEntWeeklyLimit(sub)),
			monthlyLimit: copyFloat64Ptr(subscriptionEntMonthlyLimit(sub)),

			activateDailyWindow:   windowState.activateDailyWindow,
			activateWeeklyWindow:  windowState.activateWeeklyWindow,
			activateMonthlyWindow: windowState.activateMonthlyWindow,
			resetDailyWindow:      windowState.resetDailyWindow,
			resetWeeklyWindow:     windowState.resetWeeklyWindow,
			resetMonthlyWindow:    windowState.resetMonthlyWindow,
			resetDailyBefore:      copyTimePtr(windowState.resetDailyBefore),
			resetWeeklyBefore:     copyTimePtr(windowState.resetWeeklyBefore),
			resetMonthlyBefore:    copyTimePtr(windowState.resetMonthlyBefore),
			windowStart:           windowStart,
		})
		remaining = roundSocialAmount(remaining - allowance)
		if remaining <= 0 {
			break
		}
	}

	subscriptionAmount := 0.0
	for _, charge := range plan {
		subscriptionAmount += charge.amount
	}
	subscriptionAmount = roundSocialAmount(subscriptionAmount)
	walletAmount := roundSocialAmount(math.Max(0, amount-subscriptionAmount))
	if walletAmount > 0 {
		u, err := tx.User.Query().Where(user.IDEQ(userID)).Only(ctx)
		if err != nil {
			return nil, 0, 0, err
		}
		if u.Balance+1e-9 < walletAmount {
			return nil, 0, 0, ErrSocialTaskInsufficientFunds
		}
	}
	return plan, subscriptionAmount, walletAmount, nil
}

func copyFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}

func copyTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}

func copyTimePtrValue(v time.Time) *time.Time {
	copied := v
	return &copied
}

type socialTaskSubscriptionWindowState struct {
	dailyUsage   float64
	weeklyUsage  float64
	monthlyUsage float64

	activateDailyWindow   bool
	activateWeeklyWindow  bool
	activateMonthlyWindow bool
	resetDailyWindow      bool
	resetWeeklyWindow     bool
	resetMonthlyWindow    bool
	resetDailyBefore      *time.Time
	resetWeeklyBefore     *time.Time
	resetMonthlyBefore    *time.Time
}

func socialTaskEntSubscriptionWindowState(sub *dbent.UserSubscription, now time.Time) socialTaskSubscriptionWindowState {
	state := socialTaskSubscriptionWindowState{}
	if sub == nil {
		return state
	}
	state.dailyUsage = sub.DailyUsageUsd
	state.weeklyUsage = sub.WeeklyUsageUsd
	state.monthlyUsage = sub.MonthlyUsageUsd
	if sub.DailyWindowStart == nil {
		state.activateDailyWindow = true
		state.dailyUsage = 0
	} else if socialTaskEntSubscriptionDailyWindowExpired(sub, now) {
		state.resetDailyWindow = true
		state.resetDailyBefore = copyTimePtrValue(now.Add(-24 * time.Hour))
		state.dailyUsage = 0
	}
	if sub.WeeklyWindowStart == nil {
		state.activateWeeklyWindow = true
		state.weeklyUsage = 0
	} else if socialTaskEntSubscriptionWindowExpired(sub.WeeklyWindowStart, now, 7*24*time.Hour) {
		state.resetWeeklyWindow = true
		state.resetWeeklyBefore = copyTimePtrValue(now.Add(-7 * 24 * time.Hour))
		state.weeklyUsage = 0
	}
	if sub.MonthlyWindowStart == nil {
		state.activateMonthlyWindow = true
		state.monthlyUsage = 0
	} else if socialTaskEntSubscriptionWindowExpired(sub.MonthlyWindowStart, now, 30*24*time.Hour) {
		state.resetMonthlyWindow = true
		state.resetMonthlyBefore = copyTimePtrValue(now.Add(-30 * 24 * time.Hour))
		state.monthlyUsage = 0
	}
	return state
}

func socialTaskEntSubscriptionDailyWindowExpired(sub *dbent.UserSubscription, now time.Time) bool {
	if sub == nil || sub.DailyWindowStart == nil || socialTaskEntSubscriptionHasOneTimeDailyQuota(sub) {
		return false
	}
	return socialTaskEntSubscriptionWindowExpired(sub.DailyWindowStart, now, 24*time.Hour)
}

func socialTaskEntSubscriptionHasOneTimeDailyQuota(sub *dbent.UserSubscription) bool {
	if sub == nil || sub.StartsAt.IsZero() || sub.ExpiresAt.IsZero() {
		return false
	}
	return !sub.ExpiresAt.After(sub.StartsAt.AddDate(0, 0, 1))
}

func socialTaskEntSubscriptionWindowExpired(windowStart *time.Time, now time.Time, duration time.Duration) bool {
	if windowStart == nil {
		return false
	}
	return !now.Before(windowStart.Add(duration))
}

func socialTaskEntSubscriptionAllowance(sub *dbent.UserSubscription, needed float64, windowState socialTaskSubscriptionWindowState) float64 {
	if sub == nil || sub.Edges.Group == nil || needed <= 0 {
		return 0
	}
	group := sub.Edges.Group
	if group.Status != StatusActive || group.SubscriptionType != SubscriptionTypeSubscription {
		return 0
	}
	remaining := math.Inf(1)
	if limit := subscriptionEntDailyLimit(sub); limit != nil {
		remaining = math.Min(remaining, math.Max(0, *limit-windowState.dailyUsage))
	}
	if limit := subscriptionEntWeeklyLimit(sub); limit != nil {
		remaining = math.Min(remaining, math.Max(0, *limit-windowState.weeklyUsage))
	}
	if limit := subscriptionEntMonthlyLimit(sub); limit != nil {
		remaining = math.Min(remaining, math.Max(0, *limit-windowState.monthlyUsage))
	}
	if math.IsInf(remaining, 1) {
		return needed
	}
	return math.Min(needed, remaining)
}

func (s *SocialBillingService) subscriptionAllowance(ctx context.Context, sub *UserSubscription, needed float64, platform string) (float64, error) {
	if sub == nil || !sub.IsActive() || needed <= 0 {
		return 0, nil
	}
	group := sub.Group
	if group == nil || !group.Hydrated {
		loaded, err := s.groupRepo.GetByID(ctx, sub.GroupID)
		if err != nil {
			return 0, err
		}
		group = loaded
	}
	if group == nil || !group.IsActive() || !group.IsSubscriptionType() {
		return 0, nil
	}
	if !subscriptionCoversPlatform(sub, group, platform) {
		return 0, nil
	}

	remaining := math.Inf(1)
	if limit := sub.EffectiveDailyLimitUSD(group); limit != nil && *limit > 0 {
		remaining = math.Min(remaining, math.Max(0, *limit-sub.DailyUsageUSD))
	}
	if limit := sub.EffectiveWeeklyLimitUSD(group); limit != nil && *limit > 0 {
		remaining = math.Min(remaining, math.Max(0, *limit-sub.WeeklyUsageUSD))
	}
	if limit := sub.EffectiveMonthlyLimitUSD(group); limit != nil && *limit > 0 {
		remaining = math.Min(remaining, math.Max(0, *limit-sub.MonthlyUsageUSD))
	}
	if math.IsInf(remaining, 1) {
		return needed, nil
	}
	return math.Min(needed, remaining), nil
}

func serviceSubscriptionWithCurrentWindows(sub *UserSubscription, now time.Time) *UserSubscription {
	if sub == nil {
		return nil
	}
	current := *sub
	windowStart := startOfDay(now)
	if current.DailyWindowStart == nil {
		current.DailyUsageUSD = 0
	} else if serviceSubscriptionDailyWindowExpired(&current, now) {
		current.DailyUsageUSD = 0
		current.DailyWindowStart = &windowStart
	}
	if current.WeeklyWindowStart == nil {
		current.WeeklyUsageUSD = 0
	} else if serviceSubscriptionWindowExpired(current.WeeklyWindowStart, now, 7*24*time.Hour) {
		current.WeeklyUsageUSD = 0
		current.WeeklyWindowStart = &windowStart
	}
	if current.MonthlyWindowStart == nil {
		current.MonthlyUsageUSD = 0
	} else if serviceSubscriptionWindowExpired(current.MonthlyWindowStart, now, 30*24*time.Hour) {
		current.MonthlyUsageUSD = 0
		current.MonthlyWindowStart = &windowStart
	}
	return &current
}

func serviceSubscriptionDailyWindowExpired(sub *UserSubscription, now time.Time) bool {
	if sub == nil || sub.DailyWindowStart == nil || sub.HasOneTimeDailyQuota() {
		return false
	}
	return serviceSubscriptionWindowExpired(sub.DailyWindowStart, now, 24*time.Hour)
}

func serviceSubscriptionWindowExpired(windowStart *time.Time, now time.Time, duration time.Duration) bool {
	if windowStart == nil {
		return false
	}
	return !now.Before(windowStart.Add(duration))
}

func subscriptionCoversPlatform(sub *UserSubscription, group *Group, platform string) bool {
	requested := normalizePlanPlatform(platform)
	if requested == "" {
		return true
	}
	if sub == nil {
		return false
	}
	return normalizePlanPlatform(sub.EffectivePlatform(group)) == requested
}

func socialTaskEntSubscriptionCoversPlatform(sub *dbent.UserSubscription, platform string) bool {
	requested := normalizePlanPlatform(platform)
	if requested == "" {
		return true
	}
	if sub == nil {
		return false
	}
	effective := strings.TrimSpace(sub.PlanPlatform)
	if effective == "" && sub.Edges.Group != nil {
		effective = sub.Edges.Group.Platform
	}
	return normalizePlanPlatform(effective) == requested
}

func socialTaskBillingPlatform(taskLog *dbent.SocialTaskLog) string {
	if taskLog == nil || taskLog.Edges.SocialAccount == nil {
		return ""
	}
	account := taskLog.Edges.SocialAccount
	if platform := normalizePlanPlatform(account.PlatformKey); platform != "" {
		return platform
	}
	return normalizePlanPlatform(account.Platform)
}

func subscriptionEntDailyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.DailyLimitUsd != nil {
		return positiveEntLimit(sub.DailyLimitUsd)
	}
	if sub.Edges.Group != nil {
		return positiveEntLimit(sub.Edges.Group.DailyLimitUsd)
	}
	return nil
}

func subscriptionEntWeeklyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.WeeklyLimitUsd != nil {
		return positiveEntLimit(sub.WeeklyLimitUsd)
	}
	if sub.Edges.Group != nil {
		return positiveEntLimit(sub.Edges.Group.WeeklyLimitUsd)
	}
	return nil
}

func subscriptionEntMonthlyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.MonthlyLimitUsd != nil {
		return positiveEntLimit(sub.MonthlyLimitUsd)
	}
	if sub.Edges.Group != nil {
		return positiveEntLimit(sub.Edges.Group.MonthlyLimitUsd)
	}
	return nil
}

func positiveEntLimit(limit *float64) *float64 {
	if limit == nil || math.IsNaN(*limit) || math.IsInf(*limit, 0) || *limit <= 0 {
		return nil
	}
	return limit
}

func roundSocialAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

func chargeSourceForAmounts(subscriptionAmount, walletAmount float64) string {
	switch {
	case subscriptionAmount > 0 && walletAmount > 0:
		return SocialTaskChargeSourceMixed
	case subscriptionAmount > 0:
		return SocialTaskChargeSourceSubscription
	case walletAmount > 0:
		return SocialTaskChargeSourceWallet
	default:
		return ""
	}
}
