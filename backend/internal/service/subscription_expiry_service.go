package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

type SubscriptionExpiryService struct {
	userSubRepo              UserSubscriptionRepository
	interval                 time.Duration
	settingRepo              SettingRepository
	notificationEmailService *NotificationEmailService
}

func NewSubscriptionExpiryService(userSubRepo UserSubscriptionRepository, interval time.Duration) *SubscriptionExpiryService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &SubscriptionExpiryService{userSubRepo: userSubRepo, interval: interval}
}

func (s *SubscriptionExpiryService) SetSettingRepository(settingRepo SettingRepository) {
	s.settingRepo = settingRepo
}

func (s *SubscriptionExpiryService) SetNotificationEmailService(notificationEmailService *NotificationEmailService) {
	s.notificationEmailService = notificationEmailService
}

func (s *SubscriptionExpiryService) Stop() {}

func (s *SubscriptionExpiryService) expiryReminderEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return true
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeySubscriptionExpiryNotifyEnabled)
	if err != nil {
		return err == ErrSettingNotFound
	}
	return value != "false"
}

func (s *SubscriptionExpiryService) sendExpiryReminders(ctx context.Context) {
	if s == nil || !s.expiryReminderEnabled(ctx) || s.userSubRepo == nil {
		return
	}
	_, _, _ = s.userSubRepo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 100}, nil, nil, nil, "", "", "", "")
}
