package repository

import (
	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/service"
)

func userEntityToService(u *dbent.User) *service.User {
	if u == nil {
		return nil
	}
	out := &service.User{
		ID:                         u.ID,
		Email:                      u.Email,
		Username:                   u.Username,
		Notes:                      u.Notes,
		PasswordHash:               u.PasswordHash,
		Role:                       u.Role,
		Balance:                    u.Balance,
		Concurrency:                u.Concurrency,
		Status:                     u.Status,
		TokenVersion:               u.TokenVersion,
		SignupSource:               u.SignupSource,
		LastLoginAt:                u.LastLoginAt,
		LastActiveAt:               u.LastActiveAt,
		TotpSecretEncrypted:        u.TotpSecretEncrypted,
		TotpEnabled:                u.TotpEnabled,
		TotpEnabledAt:              u.TotpEnabledAt,
		BalanceNotifyEnabled:       u.BalanceNotifyEnabled,
		BalanceNotifyThresholdType: u.BalanceNotifyThresholdType,
		BalanceNotifyThreshold:     u.BalanceNotifyThreshold,
		TotalRecharged:             u.TotalRecharged,
		RPMLimit:                   u.RpmLimit,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
	}
	if u.BalanceNotifyExtraEmails != "" && u.BalanceNotifyExtraEmails != "[]" {
		out.BalanceNotifyExtraEmails = service.ParseNotifyEmails(u.BalanceNotifyExtraEmails)
	}
	return out
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
