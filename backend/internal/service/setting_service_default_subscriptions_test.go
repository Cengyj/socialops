package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/stretchr/testify/require"
)

type defaultSubscriptionPlanReaderStub struct {
	bindings map[int64]DefaultSubscriptionPlanBinding
	err      error
}

func (s *defaultSubscriptionPlanReaderStub) GetDefaultSubscriptionPlan(_ context.Context, id int64) (DefaultSubscriptionPlanBinding, error) {
	if s.err != nil {
		return DefaultSubscriptionPlanBinding{}, s.err
	}
	if binding, ok := s.bindings[id]; ok {
		return binding, nil
	}
	return DefaultSubscriptionPlanBinding{}, ErrDefaultSubPlanInvalid
}

type defaultSubscriptionGroupReaderStub struct {
	groups map[int64]*Group
}

func (s *defaultSubscriptionGroupReaderStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := s.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func TestSettingServiceValidateDefaultSubscriptionPackagesAllowsPlanOnly(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})
	svc.SetDefaultSubscriptionPlanReader(&defaultSubscriptionPlanReaderStub{
		bindings: map[int64]DefaultSubscriptionPlanBinding{
			101: {ID: 101, GroupID: 11},
		},
	})

	err := svc.validateDefaultSubscriptionPackages(context.Background(), []DefaultSubscriptionSetting{
		{PlanID: 101, ValidityDays: 30},
	})

	require.NoError(t, err)
}

func TestSettingServiceValidateDefaultSubscriptionPackagesRejectsPlanGroupMismatch(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})
	svc.SetDefaultSubscriptionPlanReader(&defaultSubscriptionPlanReaderStub{
		bindings: map[int64]DefaultSubscriptionPlanBinding{
			101: {ID: 101, GroupID: 11},
		},
	})

	err := svc.validateDefaultSubscriptionPackages(context.Background(), []DefaultSubscriptionSetting{
		{PlanID: 101, GroupID: 12, ValidityDays: 30},
	})

	require.ErrorIs(t, err, ErrDefaultSubPlanInvalid)
}

func TestSettingServiceValidateDefaultSubscriptionPackagesRejectsDuplicatePlans(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})
	svc.SetDefaultSubscriptionPlanReader(&defaultSubscriptionPlanReaderStub{
		bindings: map[int64]DefaultSubscriptionPlanBinding{
			101: {ID: 101, GroupID: 11},
		},
	})

	err := svc.validateDefaultSubscriptionPackages(context.Background(), []DefaultSubscriptionSetting{
		{PlanID: 101, ValidityDays: 30},
		{PlanID: 101, ValidityDays: 60},
	})

	require.ErrorIs(t, err, ErrDefaultSubPlanDuplicate)
}

func TestSettingServiceValidateDefaultSubscriptionPackagesKeepsLegacyGroupOnlyCompatibility(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(&defaultSubscriptionGroupReaderStub{
		groups: map[int64]*Group{
			11: {ID: 11, SubscriptionType: SubscriptionTypeSubscription},
		},
	})

	err := svc.validateDefaultSubscriptionPackages(context.Background(), []DefaultSubscriptionSetting{
		{GroupID: 11, ValidityDays: 30},
	})

	require.NoError(t, err)
}
