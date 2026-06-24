//go:build unit

package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/payment"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type paymentRefundUserRepoStub struct {
	UserRepository

	user    *User
	deducts []float64
	updates []float64
}

func (r *paymentRefundUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	cloned := *r.user
	return &cloned, nil
}

func (r *paymentRefundUserRepoStub) DeductBalance(_ context.Context, _ int64, amount float64) error {
	r.deducts = append(r.deducts, amount)
	if r.user != nil {
		r.user.Balance -= amount
	}
	return nil
}

func (r *paymentRefundUserRepoStub) UpdateBalance(_ context.Context, _ int64, amount float64) error {
	r.updates = append(r.updates, amount)
	if r.user != nil {
		r.user.Balance += amount
	}
	return nil
}

type paymentRefundBillingCacheRecorder struct {
	mu                       sync.Mutex
	userBalanceInvalidations []int64
}

func (r *paymentRefundBillingCacheRecorder) GetUserBalance(context.Context, int64) (float64, error) {
	return 0, errors.New("unexpected GetUserBalance call")
}

func (r *paymentRefundBillingCacheRecorder) SetUserBalance(context.Context, int64, float64) error {
	return nil
}

func (r *paymentRefundBillingCacheRecorder) DeductUserBalance(context.Context, int64, float64) error {
	return nil
}

func (r *paymentRefundBillingCacheRecorder) InvalidateUserBalance(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userBalanceInvalidations = append(r.userBalanceInvalidations, userID)
	return nil
}

func (r *paymentRefundBillingCacheRecorder) GetSubscriptionCache(context.Context, int64, int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("unexpected GetSubscriptionCache call")
}

func (r *paymentRefundBillingCacheRecorder) SetSubscriptionCache(context.Context, int64, int64, *SubscriptionCacheData) error {
	return nil
}

func (r *paymentRefundBillingCacheRecorder) UpdateSubscriptionUsage(context.Context, int64, int64, float64) error {
	return nil
}

func (r *paymentRefundBillingCacheRecorder) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	return nil
}

func (r *paymentRefundBillingCacheRecorder) invalidationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.userBalanceInvalidations)
}

func newPaymentRefundCacheAwareRedeemService(cache BillingCache) *RedeemService {
	return &RedeemService{
		billingCacheService: &BillingCacheService{cache: cache},
	}
}

func TestValidateRefundRequestRejectsHistoricalGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-historical@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-historical-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-HISTORICAL-ORDER").
		SetOutTradeNo("socialops_refund_historical_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-historical-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	_, err = svc.validateRefundRequest(ctx, order.ID, user.ID)
	require.Error(t, err)
	require.Equal(t, "USER_REFUND_DISABLED", infraerrors.Reason(err))
}

func TestValidateRefundRequestRequiresProviderRefundEnabled(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-disabled@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-disabled-user").
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-user-refund-disabled").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(false).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-DISABLED-ORDER").
		SetOutTradeNo("socialops_refund_disabled_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-disabled-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(strconv.FormatInt(instance.ID, 10)).
		SetProviderKey(payment.TypeAlipay).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	_, err = svc.validateRefundRequest(ctx, order.ID, user.ID)
	require.Error(t, err)
	require.Equal(t, "USER_REFUND_DISABLED", infraerrors.Reason(err))
}

func TestPrepareRefundRejectsHistoricalGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-historical-admin@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-historical-admin-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-admin-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(188).
		SetPayAmount(188).
		SetFeeRate(0).
		SetRechargeCode("REFUND-HISTORICAL-ADMIN-ORDER").
		SetOutTradeNo("socialops_refund_historical_admin_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-historical-admin-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	plan, result, err := svc.PrepareRefund(ctx, order.ID, 0, "", false, false)
	require.Nil(t, plan)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "REFUND_DISABLED", infraerrors.Reason(err))
}

func TestGwRefundRejectsAlipayMerchantIdentitySnapshotMismatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-snapshot-mismatch@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-snapshot-mismatch-user").
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-mismatch-instance").
		SetConfig(encryptWebhookProviderConfig(t, map[string]string{
			"appId":      "runtime-alipay-app",
			"privateKey": "runtime-private-key",
		})).
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	instID := strconv.FormatInt(inst.ID, 10)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-SNAPSHOT-MISMATCH-ORDER").
		SetOutTradeNo("socialops_refund_snapshot_mismatch_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-refund-snapshot-mismatch").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(instID).
		SetProviderKey(payment.TypeAlipay).
		SetProviderSnapshot(map[string]any{
			"schema_version":       2,
			"provider_instance_id": instID,
			"provider_key":         payment.TypeAlipay,
			"merchant_app_id":      "expected-alipay-app",
		}).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient:    client,
		loadBalancer: newWebhookProviderTestLoadBalancer(client),
	}

	err = svc.gwRefund(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.Amount,
		Reason:        "snapshot mismatch",
	})
	require.ErrorContains(t, err, "alipay app_id mismatch")
}

func TestCalculateGatewayRefundAmountUsesCurrencyPrecision(t *testing.T) {
	require.InDelta(t, 6.173, calculateGatewayRefundAmount(100, 12.345, 50, "KWD"), 1e-12)
	require.InDelta(t, 12.345, calculateGatewayRefundAmount(100, 12.345, 100, "KWD"), 1e-12)
	require.InDelta(t, 52, calculateGatewayRefundAmount(100, 103, 50, "JPY"), 1e-12)
}

func TestFormatGatewayRefundAmountUsesOrderCurrency(t *testing.T) {
	order := &dbent.PaymentOrder{
		ProviderSnapshot: map[string]any{
			"currency": "KWD",
		},
	}

	require.Equal(t, "12.345", formatGatewayRefundAmount(12.345, order))
}

func TestValidateRefundProviderResponseAcceptsPending(t *testing.T) {
	require.NoError(t, validateRefundProviderResponse(&payment.RefundResponse{Status: payment.ProviderStatusPending}))
	require.NoError(t, validateRefundProviderResponse(&payment.RefundResponse{Status: payment.ProviderStatusSuccess}))
	require.Error(t, validateRefundProviderResponse(&payment.RefundResponse{Status: payment.ProviderStatusFailed}))
	require.Error(t, validateRefundProviderResponse(nil))
}

func TestPrepareRefundAllowsRemainingAmountForPartiallyRefundedOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-partial-remaining@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-partial-remaining-user").
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-partial-remaining").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(100).
		SetPayAmount(103).
		SetFeeRate(3).
		SetRechargeCode("REFUND-PARTIAL-REMAINING").
		SetOutTradeNo("socialops_refund_partial_remaining").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-partial-remaining").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPartiallyRefunded).
		SetRefundAmount(40).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(strconv.FormatInt(instance.ID, 10)).
		SetProviderKey(payment.TypeAlipay).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}

	plan, early, err := svc.PrepareRefund(ctx, order.ID, 30, "continue partial refund", false, false)
	require.NoError(t, err)
	require.Nil(t, early)
	require.NotNil(t, plan)
	require.Equal(t, 30.0, plan.RefundAmount)
	require.InDelta(t, 30.9, plan.GatewayAmount, 1e-12)

	plan, early, err = svc.PrepareRefund(ctx, order.ID, 70, "too much", false, false)
	require.Nil(t, plan)
	require.Nil(t, early)
	require.Error(t, err)
	require.Equal(t, "REFUND_AMOUNT_EXCEEDED", infraerrors.Reason(err))
}

func TestExecuteRefundCumulatesPartiallyRefundedAmount(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-partial-cumulative@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-partial-cumulative-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(100).
		SetPayAmount(103).
		SetFeeRate(3).
		SetRechargeCode("REFUND-PARTIAL-CUMULATIVE").
		SetOutTradeNo("socialops_refund_partial_cumulative").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPartiallyRefunded).
		SetRefundAmount(40).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}

	result, err := svc.ExecuteRefund(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  30,
		GatewayAmount: 30.9,
		Reason:        "continue partial refund",
	})
	require.NoError(t, err)
	require.True(t, result.Success)

	updated, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusPartiallyRefunded, updated.Status)
	require.Equal(t, 70.0, updated.RefundAmount)
	require.Equal(t, "continue partial refund", *updated.RefundReason)
}

func TestExecuteRefundInvalidatesBalanceCachesAfterSuccessfulDeduction(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-cache-success@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-cache-success-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-CACHE-SUCCESS").
		SetOutTradeNo("socialops_refund_cache_success").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	cache := &paymentRefundBillingCacheRecorder{}
	userRepo := &paymentRefundUserRepoStub{
		user: &User{
			ID:      user.ID,
			Balance: 100,
		},
	}
	svc := &PaymentService{
		entClient:     client,
		userRepo:      userRepo,
		redeemService: newPaymentRefundCacheAwareRedeemService(cache),
	}

	result, err := svc.ExecuteRefund(ctx, &RefundPlan{
		OrderID:         order.ID,
		Order:           order,
		RefundAmount:    order.Amount,
		GatewayAmount:   order.PayAmount,
		Reason:          "cache invalidation",
		DeductionType:   payment.DeductionTypeBalance,
		BalanceToDeduct: order.Amount,
	})
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, []float64{order.Amount}, userRepo.deducts)

	require.Eventually(t, func() bool {
		return cache.invalidationCount() == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestRollbackRefundInvalidatesBalanceCachesAfterSuccessfulRollback(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-cache-rollback@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-cache-rollback-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-CACHE-ROLLBACK").
		SetOutTradeNo("socialops_refund_cache_rollback").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-rollback-cache").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusRefunding).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	cache := &paymentRefundBillingCacheRecorder{}
	userRepo := &paymentRefundUserRepoStub{
		user: &User{
			ID:      user.ID,
			Balance: 12,
		},
	}
	svc := &PaymentService{
		entClient:     client,
		userRepo:      userRepo,
		redeemService: newPaymentRefundCacheAwareRedeemService(cache),
	}

	ok := svc.RollbackRefund(ctx, &RefundPlan{
		OrderID:         order.ID,
		Order:           order,
		DeductionType:   payment.DeductionTypeBalance,
		BalanceToDeduct: 50,
	}, errors.New("gateway timeout"))
	require.True(t, ok)
	require.Equal(t, []float64{50}, userRepo.updates)

	require.Eventually(t, func() bool {
		return cache.invalidationCount() == 1
	}, 2*time.Second, 10*time.Millisecond)
}
