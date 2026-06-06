package dto

import (
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAvailableSubscriptionPlansFromEntBuildsQuotaPackageContract(t *testing.T) {
	plan := &dbent.SubscriptionPlan{
		ID:              11,
		GroupID:         22,
		Platform:        "",
		Name:            "X Starter",
		Description:     "starter execution quota",
		Price:           19,
		ValidityDays:    1,
		ValidityUnit:    "months",
		Features:        "Login\n\nFollow\n Like ",
		ForSale:         true,
		SortOrder:       3,
		MonthlyLimitUsd: floatPtr(120),
		DailyLimitUsd:   floatPtr(10),
	}
	inactivePlan := &dbent.SubscriptionPlan{
		ID:              12,
		GroupID:         23,
		Name:            "Inactive",
		Price:           19,
		ValidityDays:    1,
		ValidityUnit:    "months",
		MonthlyLimitUsd: floatPtr(120),
	}
	groupInfo := map[int64]service.PlanGroupInfo{
		22: {
			Platform:         "x_twitter",
			Name:             "X Execution Pool",
			Status:           service.StatusActive,
			SubscriptionType: service.SubscriptionTypeSubscription,
			WeeklyLimitUSD:   floatPtr(50),
		},
		23: {
			Platform:         "x_twitter",
			Name:             "Inactive Pool",
			Status:           "inactive",
			SubscriptionType: service.SubscriptionTypeSubscription,
		},
	}

	plans := AvailableSubscriptionPlansFromEnt([]*dbent.SubscriptionPlan{plan, inactivePlan}, groupInfo)

	require.Len(t, plans, 1)
	require.Equal(t, int64(11), plans[0].ID)
	require.Equal(t, "x_twitter", plans[0].Platform)
	require.Equal(t, "X Execution Pool", plans[0].GroupName)
	require.NotNil(t, plans[0].QuotaUSD)
	require.Equal(t, 120.0, *plans[0].QuotaUSD)
	require.NotNil(t, plans[0].MonthlyLimitUSD)
	require.Equal(t, 120.0, *plans[0].MonthlyLimitUSD)
	require.NotNil(t, plans[0].DailyLimitUSD)
	require.Equal(t, 10.0, *plans[0].DailyLimitUSD)
	require.NotNil(t, plans[0].WeeklyLimitUSD)
	require.Equal(t, 50.0, *plans[0].WeeklyLimitUSD)
	require.Equal(t, []string{"Login", "Follow", "Like"}, plans[0].Features)
	require.True(t, plans[0].ForSale)
	require.Equal(t, 3, plans[0].SortOrder)
}

func TestAdminSubscriptionPlansFromEntKeepsInternalBindingMetadata(t *testing.T) {
	plan := &dbent.SubscriptionPlan{
		ID:              11,
		GroupID:         22,
		Name:            "X Starter",
		Price:           19,
		ValidityDays:    1,
		ValidityUnit:    "months",
		MonthlyLimitUsd: floatPtr(120),
	}

	plans := AdminSubscriptionPlansFromEnt([]*dbent.SubscriptionPlan{plan}, map[int64]service.PlanGroupInfo{
		22: {
			Platform:         "x_twitter",
			Name:             "X Execution Pool",
			Status:           service.StatusActive,
			SubscriptionType: service.SubscriptionTypeSubscription,
		},
	})

	require.Len(t, plans, 1)
	require.Equal(t, service.StatusActive, plans[0].GroupStatus)
	require.Equal(t, service.SubscriptionTypeSubscription, plans[0].SubscriptionType)
	require.Equal(t, "x_twitter", plans[0].GroupPlatform)
}

func TestUserSubscriptionFromServiceReturnsEffectiveQuotaLimits(t *testing.T) {
	sub := &service.UserSubscription{
		ID:              41,
		UserID:          7,
		GroupID:         22,
		PlanName:        "X Starter",
		PlanPlatform:    "x_twitter",
		DailyUsageUSD:   1,
		WeeklyUsageUSD:  2,
		MonthlyUsageUSD: 3,
		Group: &service.Group{
			ID:              22,
			Name:            "X Execution Pool",
			Platform:        "x_twitter",
			DailyLimitUSD:   floatPtr(10),
			WeeklyLimitUSD:  floatPtr(50),
			MonthlyLimitUSD: floatPtr(120),
		},
	}

	out := UserSubscriptionFromService(sub)

	require.NotNil(t, out)
	require.NotNil(t, out.QuotaUSD)
	require.Equal(t, 120.0, *out.QuotaUSD)
	require.NotNil(t, out.DailyLimitUSD)
	require.Equal(t, 10.0, *out.DailyLimitUSD)
	require.NotNil(t, out.WeeklyLimitUSD)
	require.Equal(t, 50.0, *out.WeeklyLimitUSD)
	require.NotNil(t, out.MonthlyLimitUSD)
	require.Equal(t, 120.0, *out.MonthlyLimitUSD)
}

func TestUserSubscriptionFromServiceReturnsEffectivePlatform(t *testing.T) {
	sub := &service.UserSubscription{
		ID:           43,
		UserID:       7,
		GroupID:      22,
		PlanName:     "Legacy X Starter",
		PlanPlatform: "",
		Group: &service.Group{
			ID:       22,
			Name:     "X Execution Pool",
			Platform: "x_twitter",
		},
	}

	out := UserSubscriptionFromService(sub)

	require.NotNil(t, out)
	require.Equal(t, "x_twitter", out.PlanPlatform)
}

func TestUserSubscriptionFromServiceKeepsPlanLimitOverrides(t *testing.T) {
	sub := &service.UserSubscription{
		ID:              42,
		UserID:          7,
		GroupID:         22,
		PlanName:        "X Growth",
		PlanPlatform:    "x_twitter",
		DailyLimitUSD:   floatPtr(6),
		WeeklyLimitUSD:  floatPtr(28),
		MonthlyLimitUSD: floatPtr(100),
		Group: &service.Group{
			ID:              22,
			Name:            "X Execution Pool",
			Platform:        "x_twitter",
			DailyLimitUSD:   floatPtr(10),
			WeeklyLimitUSD:  floatPtr(50),
			MonthlyLimitUSD: floatPtr(120),
		},
	}

	out := UserSubscriptionFromService(sub)

	require.NotNil(t, out)
	require.NotNil(t, out.DailyLimitUSD)
	require.Equal(t, 6.0, *out.DailyLimitUSD)
	require.NotNil(t, out.WeeklyLimitUSD)
	require.Equal(t, 28.0, *out.WeeklyLimitUSD)
	require.NotNil(t, out.MonthlyLimitUSD)
	require.Equal(t, 100.0, *out.MonthlyLimitUSD)
}

func floatPtr(value float64) *float64 {
	return &value
}
