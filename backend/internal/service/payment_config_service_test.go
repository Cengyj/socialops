package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/payment"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestPcParseFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		defaultVal float64
		expected   float64
	}{
		{"empty string returns default", "", 1.0, 1.0},
		{"valid float", "3.14", 0, 3.14},
		{"valid integer as float", "42", 0, 42.0},
		{"invalid string returns default", "notanumber", 9.99, 9.99},
		{"zero value", "0", 5.0, 0},
		{"negative value", "-10.5", 0, -10.5},
		{"very large value", "99999999.99", 0, 99999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pcParseFloat(tt.input, tt.defaultVal)
			if got != tt.expected {
				t.Fatalf("pcParseFloat(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestPcParseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		defaultVal int
		expected   int
	}{
		{"empty string returns default", "", 30, 30},
		{"valid int", "10", 0, 10},
		{"invalid string returns default", "abc", 5, 5},
		{"float string returns default", "3.14", 0, 0},
		{"zero value", "0", 99, 0},
		{"negative value", "-1", 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pcParseInt(tt.input, tt.defaultVal)
			if got != tt.expected {
				t.Fatalf("pcParseInt(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestParsePaymentConfig(t *testing.T) {
	t.Parallel()

	svc := &PaymentConfigService{}

	t.Run("empty vals uses defaults", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{})
		if cfg.Enabled {
			t.Fatal("expected Enabled=false by default")
		}
		if cfg.MinAmount != 1 {
			t.Fatalf("expected MinAmount=1, got %v", cfg.MinAmount)
		}
		if cfg.MaxAmount != 0 {
			t.Fatalf("expected MaxAmount=0 (no limit), got %v", cfg.MaxAmount)
		}
		if cfg.OrderTimeoutMin != 30 {
			t.Fatalf("expected OrderTimeoutMin=30, got %v", cfg.OrderTimeoutMin)
		}
		if cfg.MaxPendingOrders != 3 {
			t.Fatalf("expected MaxPendingOrders=3, got %v", cfg.MaxPendingOrders)
		}
		if cfg.LoadBalanceStrategy != payment.DefaultLoadBalanceStrategy {
			t.Fatalf("expected LoadBalanceStrategy=%s, got %q", payment.DefaultLoadBalanceStrategy, cfg.LoadBalanceStrategy)
		}
		if len(cfg.EnabledTypes) != 0 {
			t.Fatalf("expected empty EnabledTypes, got %v", cfg.EnabledTypes)
		}
	})

	t.Run("all values populated", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingPaymentEnabled:      "true",
			SettingMinRechargeAmount:   "5.00",
			SettingMaxRechargeAmount:   "1000.00",
			SettingDailyRechargeLimit:  "5000.00",
			SettingOrderTimeoutMinutes: "15",
			SettingMaxPendingOrders:    "5",
			SettingEnabledPaymentTypes: "alipay,wxpay,stripe",
			SettingBalancePayDisabled:  "true",
			SettingLoadBalanceStrategy: "least_amount",
			SettingProductNamePrefix:   "PRE",
			SettingProductNameSuffix:   "SUF",
		}
		cfg := svc.parsePaymentConfig(vals)

		if !cfg.Enabled {
			t.Fatal("expected Enabled=true")
		}
		if cfg.MinAmount != 5 {
			t.Fatalf("MinAmount = %v, want 5", cfg.MinAmount)
		}
		if cfg.MaxAmount != 1000 {
			t.Fatalf("MaxAmount = %v, want 1000", cfg.MaxAmount)
		}
		if cfg.DailyLimit != 5000 {
			t.Fatalf("DailyLimit = %v, want 5000", cfg.DailyLimit)
		}
		if cfg.OrderTimeoutMin != 15 {
			t.Fatalf("OrderTimeoutMin = %v, want 15", cfg.OrderTimeoutMin)
		}
		if cfg.MaxPendingOrders != 5 {
			t.Fatalf("MaxPendingOrders = %v, want 5", cfg.MaxPendingOrders)
		}
		if len(cfg.EnabledTypes) != 3 {
			t.Fatalf("EnabledTypes len = %d, want 3", len(cfg.EnabledTypes))
		}
		if cfg.EnabledTypes[0] != "alipay" || cfg.EnabledTypes[1] != "wxpay" || cfg.EnabledTypes[2] != "stripe" {
			t.Fatalf("EnabledTypes = %v, want [alipay wxpay stripe]", cfg.EnabledTypes)
		}
		if !cfg.BalanceDisabled {
			t.Fatal("expected BalanceDisabled=true")
		}
		if cfg.LoadBalanceStrategy != "least_amount" {
			t.Fatalf("LoadBalanceStrategy = %q, want %q", cfg.LoadBalanceStrategy, "least_amount")
		}
		if cfg.ProductNamePrefix != "PRE" {
			t.Fatalf("ProductNamePrefix = %q, want %q", cfg.ProductNamePrefix, "PRE")
		}
		if cfg.ProductNameSuffix != "SUF" {
			t.Fatalf("ProductNameSuffix = %q, want %q", cfg.ProductNameSuffix, "SUF")
		}
	})

	t.Run("enabled types with spaces are trimmed", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingEnabledPaymentTypes: " alipay , wxpay ",
		}
		cfg := svc.parsePaymentConfig(vals)
		if len(cfg.EnabledTypes) != 2 {
			t.Fatalf("EnabledTypes len = %d, want 2", len(cfg.EnabledTypes))
		}
		if cfg.EnabledTypes[0] != "alipay" || cfg.EnabledTypes[1] != "wxpay" {
			t.Fatalf("EnabledTypes = %v, want [alipay wxpay]", cfg.EnabledTypes)
		}
	})

	t.Run("enabled types are normalized to visible methods and deduplicated", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingEnabledPaymentTypes: "alipay_direct, alipay, wxpay_direct, wxpay",
		}
		cfg := svc.parsePaymentConfig(vals)
		if len(cfg.EnabledTypes) != 2 {
			t.Fatalf("EnabledTypes len = %d, want 2", len(cfg.EnabledTypes))
		}
		if cfg.EnabledTypes[0] != "alipay" || cfg.EnabledTypes[1] != "wxpay" {
			t.Fatalf("EnabledTypes = %v, want [alipay wxpay]", cfg.EnabledTypes)
		}
	})

	t.Run("empty enabled types string", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingEnabledPaymentTypes: "",
		}
		cfg := svc.parsePaymentConfig(vals)
		if len(cfg.EnabledTypes) != 0 {
			t.Fatalf("expected empty EnabledTypes for empty string, got %v", cfg.EnabledTypes)
		}
	})
}

func TestGetBasePaymentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{payment.TypeEasyPay, payment.TypeEasyPay},
		{payment.TypeStripe, payment.TypeStripe},
		{payment.TypeCard, payment.TypeStripe},
		{payment.TypeLink, payment.TypeStripe},
		{payment.TypeAlipay, payment.TypeAlipay},
		{payment.TypeAlipayDirect, payment.TypeAlipay},
		{payment.TypeWxpay, payment.TypeWxpay},
		{payment.TypeWxpayDirect, payment.TypeWxpay},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := payment.GetBasePaymentType(tt.input)
			if got != tt.expected {
				t.Fatalf("GetBasePaymentType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyVisibleMethodRoutingToEnabledTypes(t *testing.T) {
	t.Parallel()

	base := []string{"alipay", "wxpay", "stripe"}
	vals := map[string]string{
		SettingPaymentVisibleMethodAlipayEnabled: "true",
		SettingPaymentVisibleMethodAlipaySource:  VisibleMethodSourceOfficialAlipay,
		SettingPaymentVisibleMethodWxpayEnabled:  "true",
		SettingPaymentVisibleMethodWxpaySource:   VisibleMethodSourceOfficialWechat,
	}
	available := map[string]bool{
		VisibleMethodSourceOfficialAlipay: true,
		VisibleMethodSourceOfficialWechat: false,
	}

	got := applyVisibleMethodRoutingToEnabledTypes(base, vals, available)
	want := []string{"alipay", "stripe"}
	if len(got) != len(want) {
		t.Fatalf("applyVisibleMethodRoutingToEnabledTypes len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("applyVisibleMethodRoutingToEnabledTypes[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestApplyVisibleMethodRoutingAddsConfiguredVisibleMethod(t *testing.T) {
	t.Parallel()

	base := []string{"stripe"}
	vals := map[string]string{
		SettingPaymentVisibleMethodAlipayEnabled: "true",
		SettingPaymentVisibleMethodAlipaySource:  VisibleMethodSourceEasyPayAlipay,
	}
	available := map[string]bool{
		VisibleMethodSourceEasyPayAlipay: true,
	}

	got := applyVisibleMethodRoutingToEnabledTypes(base, vals, available)
	want := []string{"stripe", "alipay"}
	if len(got) != len(want) {
		t.Fatalf("applyVisibleMethodRoutingToEnabledTypes len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("applyVisibleMethodRoutingToEnabledTypes[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildVisibleMethodSourceAvailability(t *testing.T) {
	t.Parallel()

	instances := []*dbent.PaymentProviderInstance{
		{ProviderKey: payment.TypeAlipay, SupportedTypes: "alipay"},
		{ProviderKey: payment.TypeEasyPay, SupportedTypes: "wxpay_direct, alipay"},
		{ProviderKey: payment.TypeWxpay, SupportedTypes: "wxpay_direct"},
	}

	got := buildVisibleMethodSourceAvailability(instances)
	if !got[VisibleMethodSourceOfficialAlipay] {
		t.Fatalf("expected %q to be available", VisibleMethodSourceOfficialAlipay)
	}
	if !got[VisibleMethodSourceEasyPayAlipay] {
		t.Fatalf("expected %q to be available", VisibleMethodSourceEasyPayAlipay)
	}
	if !got[VisibleMethodSourceOfficialWechat] {
		t.Fatalf("expected %q to be available", VisibleMethodSourceOfficialWechat)
	}
	if !got[VisibleMethodSourceEasyPayWechat] {
		t.Fatalf("expected %q to be available", VisibleMethodSourceEasyPayWechat)
	}
}

func TestGetPaymentConfigKeepsStoredEnabledTypes(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	_, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeEasyPay).
		SetName("EasyPay Alipay").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create easypay instance: %v", err)
	}

	svc := &PaymentConfigService{
		entClient: client,
		settingRepo: &paymentConfigSettingRepoStub{
			values: map[string]string{
				SettingEnabledPaymentTypes: "alipay,wxpay,stripe",
			},
		},
	}

	cfg, err := svc.GetPaymentConfig(ctx)
	if err != nil {
		t.Fatalf("GetPaymentConfig returned error: %v", err)
	}

	want := []string{payment.TypeAlipay, payment.TypeWxpay, payment.TypeStripe}
	if len(cfg.EnabledTypes) != len(want) {
		t.Fatalf("EnabledTypes len = %d, want %d (%v)", len(cfg.EnabledTypes), len(want), cfg.EnabledTypes)
	}
	for i := range want {
		if cfg.EnabledTypes[i] != want[i] {
			t.Fatalf("EnabledTypes[%d] = %q, want %q (full=%v)", i, cfg.EnabledTypes[i], want[i], cfg.EnabledTypes)
		}
	}
}

func TestPaymentConfigServiceGetGroupInfoMapUsesSocialOpsGroups(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	dailyLimit := 12.5
	weeklyLimit := 70.0

	group, err := client.Group.Create().
		SetName("X Subscription").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetRateMultiplier(1.25).
		SetDailyLimitUsd(dailyLimit).
		SetWeeklyLimitUsd(weeklyLimit).
		Save(ctx)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(group.ID).
		SetName("X Monthly").
		SetPrice(19.99).
		SetValidityDays(30).
		SetValidityUnit("days").
		Save(ctx)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	svc := &PaymentConfigService{entClient: client}
	info := svc.GetGroupInfoMap(ctx, []*dbent.SubscriptionPlan{plan, nil, plan})
	got, ok := info[group.ID]
	if !ok {
		t.Fatalf("expected group info for id %d, got %v", group.ID, info)
	}
	if got.Name != "X Subscription" || got.Platform != "x_twitter" {
		t.Fatalf("group info name/platform = %q/%q", got.Name, got.Platform)
	}
	if got.Status != StatusActive || got.SubscriptionType != SubscriptionTypeSubscription {
		t.Fatalf("group info status/type = %q/%q", got.Status, got.SubscriptionType)
	}
	if got.RateMultiplier != 1.25 {
		t.Fatalf("rate multiplier = %v, want 1.25", got.RateMultiplier)
	}
	if got.DailyLimitUSD == nil || *got.DailyLimitUSD != dailyLimit {
		t.Fatalf("daily limit = %v, want %v", got.DailyLimitUSD, dailyLimit)
	}
	if got.WeeklyLimitUSD == nil || *got.WeeklyLimitUSD != weeklyLimit {
		t.Fatalf("weekly limit = %v, want %v", got.WeeklyLimitUSD, weeklyLimit)
	}
}

func TestPaymentConfigServiceCreatePlanRejectsNonSubscriptionGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	group, err := client.Group.Create().
		SetName("Standard Social").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeStandard).
		Save(ctx)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	svc := &PaymentConfigService{entClient: client}
	groupID := group.ID
	_, err = svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Invalid Standard Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err == nil {
		t.Fatal("expected CreatePlan to reject non-subscription group")
	}
	if !strings.Contains(err.Error(), "group is not a subscription type") {
		t.Fatalf("CreatePlan error = %v", err)
	}
}

func TestPaymentConfigServiceCreatePlanRejectsGroupPlatformMismatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}

	_, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Platform:        "instagram",
		Name:            "Mismatched Platform Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err == nil {
		t.Fatal("expected CreatePlan to reject mismatched group and plan platform")
	}
	if !strings.Contains(err.Error(), "platform") {
		t.Fatalf("CreatePlan error = %v", err)
	}
}

func TestPaymentConfigServiceCreatePlanFindsLegacyTwitterSubscriptionGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	group, err := client.Group.Create().
		SetName("Legacy Twitter Subscription").
		SetPlatform("twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy subscription group: %v", err)
	}

	svc := &PaymentConfigService{entClient: client}
	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		Platform:        "x_twitter",
		Name:            "X Compatible Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if plan.GroupID != group.ID {
		t.Fatalf("plan group id = %d, want legacy twitter group %d", plan.GroupID, group.ID)
	}
	if plan.Platform != "x_twitter" {
		t.Fatalf("plan platform = %q, want x_twitter", plan.Platform)
	}
}

func TestPaymentConfigServiceCreatePlanRequiresQuota(t *testing.T) {
	ctx := context.Background()
	svc := &PaymentConfigService{entClient: newPaymentConfigServiceTestClient(t)}
	groupID := int64(1)

	_, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:      &groupID,
		Name:         "Missing Quota Plan",
		Price:        9.99,
		ValidityDays: 30,
		ValidityUnit: "days",
	})
	if err == nil {
		t.Fatal("expected CreatePlan to reject missing quota")
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Fatalf("CreatePlan error = %v", err)
	}
}

func TestPaymentConfigServiceCreatePlanSavesQuotaAndGuardrails(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}
	zero := 0.0

	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Quota Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
		DailyLimitUSD:   paymentConfigFloatPtr(10),
		WeeklyLimitUSD:  &zero,
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if plan.MonthlyLimitUsd == nil || *plan.MonthlyLimitUsd != 100 {
		t.Fatalf("monthly quota = %v, want 100", plan.MonthlyLimitUsd)
	}
	if plan.DailyLimitUsd == nil || *plan.DailyLimitUsd != 10 {
		t.Fatalf("daily guardrail = %v, want 10", plan.DailyLimitUsd)
	}
	if plan.WeeklyLimitUsd != nil {
		t.Fatalf("weekly guardrail = %v, want nil", plan.WeeklyLimitUsd)
	}
}

func TestPaymentConfigServiceCreatePlanAcceptsSemanticQuotaField(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}

	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:      &groupID,
		Name:         "Semantic Quota Plan",
		Price:        9.99,
		ValidityDays: 30,
		ValidityUnit: "days",
		QuotaUSD:     paymentConfigFloatPtr(150),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if plan.MonthlyLimitUsd == nil || *plan.MonthlyLimitUsd != 150 {
		t.Fatalf("monthly quota = %v, want 150", plan.MonthlyLimitUsd)
	}
}

func TestPaymentConfigServiceCreatePlanRejectsConflictingQuotaFields(t *testing.T) {
	ctx := context.Background()
	svc := &PaymentConfigService{entClient: newPaymentConfigServiceTestClient(t)}
	groupID := int64(1)

	_, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Conflicting Quota Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		QuotaUSD:        paymentConfigFloatPtr(150),
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err == nil {
		t.Fatal("expected CreatePlan to reject conflicting quota fields")
	}
	if !strings.Contains(err.Error(), "quota_usd") {
		t.Fatalf("CreatePlan error = %v", err)
	}
}

func TestPaymentConfigServiceCreatePlanRejectsInvalidGuardrail(t *testing.T) {
	ctx := context.Background()
	svc := &PaymentConfigService{entClient: newPaymentConfigServiceTestClient(t)}
	groupID := int64(1)

	_, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Invalid Guardrail Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
		DailyLimitUSD:   paymentConfigFloatPtr(120),
	})
	if err == nil {
		t.Fatal("expected CreatePlan to reject guardrail over quota")
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Fatalf("CreatePlan error = %v", err)
	}
}

func TestPaymentConfigServiceUpdatePlanClearsGuardrailsWithZero(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}
	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Clear Guardrails Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
		DailyLimitUSD:   paymentConfigFloatPtr(10),
		WeeklyLimitUSD:  paymentConfigFloatPtr(50),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}

	updated, err := svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{
		DailyLimitUSD:  paymentConfigFloatPtr(0),
		WeeklyLimitUSD: paymentConfigFloatPtr(0),
	})
	if err != nil {
		t.Fatalf("UpdatePlan returned error: %v", err)
	}
	if updated.DailyLimitUsd != nil {
		t.Fatalf("daily guardrail = %v, want nil", updated.DailyLimitUsd)
	}
	if updated.WeeklyLimitUsd != nil {
		t.Fatalf("weekly guardrail = %v, want nil", updated.WeeklyLimitUsd)
	}
	if updated.MonthlyLimitUsd == nil || *updated.MonthlyLimitUsd != 100 {
		t.Fatalf("monthly quota = %v, want 100", updated.MonthlyLimitUsd)
	}
}

func TestPaymentConfigServiceUpdatePlanRejectsInvalidQuotaPatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}
	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Invalid Patch Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
		DailyLimitUSD:   paymentConfigFloatPtr(30),
		WeeklyLimitUSD:  paymentConfigFloatPtr(80),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}

	_, err = svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{MonthlyLimitUSD: paymentConfigFloatPtr(0)})
	if err == nil {
		t.Fatal("expected UpdatePlan to reject zero quota")
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Fatalf("UpdatePlan zero quota error = %v", err)
	}

	_, err = svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{WeeklyLimitUSD: paymentConfigFloatPtr(20)})
	if err == nil {
		t.Fatal("expected UpdatePlan to reject weekly guardrail below current daily guardrail")
	}
	if !strings.Contains(err.Error(), "weekly") {
		t.Fatalf("UpdatePlan weekly guardrail error = %v", err)
	}
}

func TestPaymentConfigServiceUpdatePlanAcceptsSemanticQuotaField(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}
	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Update Semantic Quota Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}

	updated, err := svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{QuotaUSD: paymentConfigFloatPtr(180)})
	if err != nil {
		t.Fatalf("UpdatePlan returned error: %v", err)
	}
	if updated.MonthlyLimitUsd == nil || *updated.MonthlyLimitUsd != 180 {
		t.Fatalf("monthly quota = %v, want 180", updated.MonthlyLimitUsd)
	}
}

func TestPaymentConfigServiceUpdatePlanRejectsGroupPlatformMismatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	groupID := createPaymentConfigSubscriptionGroup(t, ctx, client)
	svc := &PaymentConfigService{entClient: client}
	plan, err := svc.CreatePlan(ctx, CreatePlanRequest{
		GroupID:         &groupID,
		Name:            "Consistent Plan",
		Price:           9.99,
		ValidityDays:    30,
		ValidityUnit:    "days",
		MonthlyLimitUSD: paymentConfigFloatPtr(100),
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}

	instagram := "instagram"
	_, err = svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{
		GroupID:  &groupID,
		Platform: &instagram,
	})
	if err == nil {
		t.Fatal("expected UpdatePlan to reject mismatched group and plan platform")
	}
	if !strings.Contains(err.Error(), "platform") {
		t.Fatalf("UpdatePlan error = %v", err)
	}
}

func newPaymentConfigServiceTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	dbName := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()),
	)
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type paymentConfigSettingRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *paymentConfigSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, nil
}
func (s *paymentConfigSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *paymentConfigSettingRepoStub) Set(context.Context, string, string) error { return nil }
func (s *paymentConfigSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}
func (s *paymentConfigSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	s.updates = make(map[string]string, len(values))
	for key, value := range values {
		s.updates[key] = value
		if s.values == nil {
			s.values = map[string]string{}
		}
		s.values[key] = value
	}
	return nil
}
func (s *paymentConfigSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *paymentConfigSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestUpdatePaymentConfig_PersistsVisibleMethodRouting(t *testing.T) {
	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}

	alipayEnabled := true
	wxpayEnabled := false
	err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		VisibleMethodAlipayEnabled: &alipayEnabled,
		VisibleMethodAlipaySource:  paymentConfigStrPtr(VisibleMethodSourceEasyPayAlipay),
		VisibleMethodWxpayEnabled:  &wxpayEnabled,
		VisibleMethodWxpaySource:   paymentConfigStrPtr(VisibleMethodSourceOfficialWechat),
	})
	if err != nil {
		t.Fatalf("UpdatePaymentConfig returned error: %v", err)
	}

	if repo.values[SettingPaymentVisibleMethodAlipayEnabled] != "true" {
		t.Fatalf("alipay enabled = %q, want true", repo.values[SettingPaymentVisibleMethodAlipayEnabled])
	}
	if repo.values[SettingPaymentVisibleMethodAlipaySource] != VisibleMethodSourceEasyPayAlipay {
		t.Fatalf("alipay source = %q, want %q", repo.values[SettingPaymentVisibleMethodAlipaySource], VisibleMethodSourceEasyPayAlipay)
	}
	if repo.values[SettingPaymentVisibleMethodWxpayEnabled] != "false" {
		t.Fatalf("wxpay enabled = %q, want false", repo.values[SettingPaymentVisibleMethodWxpayEnabled])
	}
	if repo.values[SettingPaymentVisibleMethodWxpaySource] != VisibleMethodSourceOfficialWechat {
		t.Fatalf("wxpay source = %q, want %q", repo.values[SettingPaymentVisibleMethodWxpaySource], VisibleMethodSourceOfficialWechat)
	}
}

func TestUpdatePaymentConfig_PreservesUnspecifiedPatchFields(t *testing.T) {
	repo := &paymentConfigSettingRepoStub{values: map[string]string{
		SettingEnabledPaymentTypes: "alipay,wxpay",
		SettingHelpText:            "existing payment help",
		SettingMaxPendingOrders:    "5",
	}}
	svc := &PaymentConfigService{settingRepo: repo}

	enabled := true
	err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("UpdatePaymentConfig returned error: %v", err)
	}

	if repo.values[SettingPaymentEnabled] != "true" {
		t.Fatalf("payment enabled = %q, want true", repo.values[SettingPaymentEnabled])
	}
	if repo.values[SettingEnabledPaymentTypes] != "alipay,wxpay" {
		t.Fatalf("enabled payment types = %q, want existing value", repo.values[SettingEnabledPaymentTypes])
	}
	if repo.values[SettingHelpText] != "existing payment help" {
		t.Fatalf("help text = %q, want existing value", repo.values[SettingHelpText])
	}
	if repo.values[SettingMaxPendingOrders] != "5" {
		t.Fatalf("max pending orders = %q, want existing value", repo.values[SettingMaxPendingOrders])
	}
}

func paymentConfigStrPtr(value string) *string {
	return &value
}

func paymentConfigFloatPtr(value float64) *float64 {
	return &value
}

func createPaymentConfigSubscriptionGroup(t *testing.T, ctx context.Context, client *dbent.Client) int64 {
	t.Helper()
	group, err := client.Group.Create().
		SetName("X Subscription").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		Save(ctx)
	if err != nil {
		t.Fatalf("create subscription group: %v", err)
	}
	return group.ID
}
