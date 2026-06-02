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

type SocialBillingEstimate struct {
	UnitPrice                  float64 `json:"unit_price"`
	ActionCount                int     `json:"action_count"`
	EstimatedTotal             float64 `json:"estimated_total"`
	SubscriptionAllowance      float64 `json:"subscription_allowance"`
	SubscriptionEstimatedUsage float64 `json:"subscription_estimated_usage"`
	WalletRequired             float64 `json:"wallet_required"`
	WalletBalance              float64 `json:"wallet_balance"`
	CanAfford                  bool    `json:"can_afford"`
	ChargeOnSuccessOnly        bool    `json:"charge_on_success_only"`
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

func (s *SocialBillingService) Estimate(ctx context.Context, userID int64, actionCount int) (*SocialBillingEstimate, error) {
	if actionCount < 0 {
		actionCount = 0
	}
	total := roundSocialAmount(float64(actionCount) * SocialTaskUnitPrice)
	estimate := &SocialBillingEstimate{
		UnitPrice:           SocialTaskUnitPrice,
		ActionCount:         actionCount,
		EstimatedTotal:      total,
		ChargeOnSuccessOnly: true,
	}
	if total == 0 {
		estimate.CanAfford = true
		return estimate, nil
	}
	if s == nil || s.userRepo == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_BILLING_UNAVAILABLE", "social billing service is unavailable")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	estimate.WalletBalance = roundSocialAmount(user.Balance)

	allowance, err := s.availableSubscriptionAllowance(ctx, userID, total)
	if err != nil {
		return nil, err
	}
	estimate.SubscriptionAllowance = roundSocialAmount(allowance)
	estimate.SubscriptionEstimatedUsage = roundSocialAmount(math.Min(total, allowance))
	estimate.WalletRequired = roundSocialAmount(math.Max(0, total-estimate.SubscriptionEstimatedUsage))
	estimate.CanAfford = estimate.WalletRequired <= estimate.WalletBalance+1e-9
	return estimate, nil
}

func (s *SocialBillingService) EnsureCanAfford(ctx context.Context, userID int64, actionCount int) (*SocialBillingEstimate, error) {
	estimate, err := s.Estimate(ctx, userID, actionCount)
	if err != nil {
		return nil, err
	}
	if !estimate.CanAfford {
		return estimate, ErrSocialTaskInsufficientFunds.WithMetadata(map[string]string{
			"estimated_total": fmt.Sprintf("%.2f", estimate.EstimatedTotal),
			"wallet_required": fmt.Sprintf("%.2f", estimate.WalletRequired),
			"wallet_balance":  fmt.Sprintf("%.2f", estimate.WalletBalance),
		})
	}
	return estimate, nil
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
	if amount <= 0 {
		return &SocialBillingChargeResult{Amount: 0}, nil
	}
	if s == nil || entClient == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_BILLING_UNAVAILABLE", "social billing service is unavailable")
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

	plan, subscriptionAmount, walletAmount, err := planSocialTaskChargeInTx(ctx, tx, userID, amount)
	if err != nil {
		return nil, err
	}

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
			predicates = append(predicates, usersubscription.DailyUsageUsdLTE(roundSocialAmount(*charge.dailyLimit-charge.amount)))
		}
		if charge.weeklyLimit != nil {
			predicates = append(predicates, usersubscription.WeeklyUsageUsdLTE(roundSocialAmount(*charge.weeklyLimit-charge.amount)))
		}
		if charge.monthlyLimit != nil {
			predicates = append(predicates, usersubscription.MonthlyUsageUsdLTE(roundSocialAmount(*charge.monthlyLimit-charge.amount)))
		}
		updated, err := tx.UserSubscription.Update().
			Where(predicates...).
			AddDailyUsageUsd(charge.amount).
			AddWeeklyUsageUsd(charge.amount).
			AddMonthlyUsageUsd(charge.amount).
			Save(ctx)
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
		billingParts = append(billingParts, fmt.Sprintf("wallet:%d:%d:%.2f", userID, time.Now().UnixNano(), walletAmount))
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
		SetExecutedAt(time.Now()).
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

func (s *SocialBillingService) availableSubscriptionAllowance(ctx context.Context, userID int64, needed float64) (float64, error) {
	if s == nil || s.userSubRepo == nil || s.groupRepo == nil || needed <= 0 {
		return 0, nil
	}
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	total := 0.0
	for i := range subs {
		allowance, err := s.subscriptionAllowance(ctx, &subs[i], needed-total)
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
}

func planSocialTaskChargeInTx(ctx context.Context, tx *dbent.Tx, userID int64, amount float64) ([]socialTaskChargeAllocation, float64, float64, error) {
	now := time.Now()
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
		allowance := socialTaskEntSubscriptionAllowance(sub, remaining)
		allowance = roundSocialAmount(allowance)
		if allowance <= 0 {
			continue
		}
		plan = append(plan, socialTaskChargeAllocation{
			subID:        sub.ID,
			groupID:      sub.GroupID,
			amount:       allowance,
			dailyLimit:   copyFloat64Ptr(sub.Edges.Group.DailyLimitUsd),
			weeklyLimit:  copyFloat64Ptr(sub.Edges.Group.WeeklyLimitUsd),
			monthlyLimit: copyFloat64Ptr(sub.Edges.Group.MonthlyLimitUsd),
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

func socialTaskEntSubscriptionAllowance(sub *dbent.UserSubscription, needed float64) float64 {
	if sub == nil || sub.Edges.Group == nil || needed <= 0 {
		return 0
	}
	group := sub.Edges.Group
	if group.Status != StatusActive || group.SubscriptionType != SubscriptionTypeSubscription {
		return 0
	}
	remaining := math.Inf(1)
	if group.DailyLimitUsd != nil {
		remaining = math.Min(remaining, math.Max(0, *group.DailyLimitUsd-sub.DailyUsageUsd))
	}
	if group.WeeklyLimitUsd != nil {
		remaining = math.Min(remaining, math.Max(0, *group.WeeklyLimitUsd-sub.WeeklyUsageUsd))
	}
	if group.MonthlyLimitUsd != nil {
		remaining = math.Min(remaining, math.Max(0, *group.MonthlyLimitUsd-sub.MonthlyUsageUsd))
	}
	if math.IsInf(remaining, 1) {
		return needed
	}
	return math.Min(needed, remaining)
}

func (s *SocialBillingService) subscriptionAllowance(ctx context.Context, sub *UserSubscription, needed float64) (float64, error) {
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

	remaining := math.Inf(1)
	if group.HasDailyLimit() {
		remaining = math.Min(remaining, math.Max(0, *group.DailyLimitUSD-sub.DailyUsageUSD))
	}
	if group.HasWeeklyLimit() {
		remaining = math.Min(remaining, math.Max(0, *group.WeeklyLimitUSD-sub.WeeklyUsageUSD))
	}
	if group.HasMonthlyLimit() {
		remaining = math.Min(remaining, math.Max(0, *group.MonthlyLimitUSD-sub.MonthlyUsageUSD))
	}
	if math.IsInf(remaining, 1) {
		return needed, nil
	}
	return math.Min(needed, remaining), nil
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
