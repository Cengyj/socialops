package dto

import "github.com/Wei-Shaw/socialops/internal/service"

func UserFromServiceShallow(u *service.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:                         u.ID,
		Email:                      u.Email,
		Username:                   u.Username,
		Role:                       u.Role,
		Balance:                    u.Balance,
		Concurrency:                u.Concurrency,
		Status:                     u.Status,
		AllowedGroups:              u.AllowedGroups,
		LastActiveAt:               u.LastActiveAt,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
		BalanceNotifyEnabled:       u.BalanceNotifyEnabled,
		BalanceNotifyThresholdType: u.BalanceNotifyThresholdType,
		BalanceNotifyThreshold:     u.BalanceNotifyThreshold,
		BalanceNotifyExtraEmails:   NotifyEmailEntriesFromService(u.BalanceNotifyExtraEmails),
		TotalRecharged:             u.TotalRecharged,
		RPMLimit:                   u.RPMLimit,
	}
}

func UserFromService(u *service.User) *User {
	if u == nil {
		return nil
	}
	out := UserFromServiceShallow(u)
	if len(u.APIKeys) > 0 {
		out.APIKeys = make([]APIKey, 0, len(u.APIKeys))
		for i := range u.APIKeys {
			k := u.APIKeys[i]
			out.APIKeys = append(out.APIKeys, *APIKeyFromService(&k))
		}
	}
	if len(u.Subscriptions) > 0 {
		out.Subscriptions = make([]UserSubscription, 0, len(u.Subscriptions))
		for i := range u.Subscriptions {
			s := u.Subscriptions[i]
			out.Subscriptions = append(out.Subscriptions, *UserSubscriptionFromService(&s))
		}
	}
	return out
}

func UserFromServiceAdmin(u *service.User) *AdminUser {
	if u == nil {
		return nil
	}
	base := UserFromService(u)
	if base == nil {
		return nil
	}
	return &AdminUser{
		User:       *base,
		Notes:      u.Notes,
		LastUsedAt: u.LastUsedAt,
		GroupRates: u.GroupRates,
	}
}

func APIKeyFromService(k *service.APIKey) *APIKey {
	if k == nil {
		return nil
	}
	out := &APIKey{
		ID:            k.ID,
		UserID:        k.UserID,
		Key:           k.Key,
		Name:          k.Name,
		GroupID:       k.GroupID,
		Status:        k.Status,
		IPWhitelist:   k.IPWhitelist,
		IPBlacklist:   k.IPBlacklist,
		LastUsedAt:    k.LastUsedAt,
		Quota:         k.Quota,
		QuotaUsed:     k.QuotaUsed,
		ExpiresAt:     k.ExpiresAt,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
		RateLimit5h:   k.RateLimit5h,
		RateLimit1d:   k.RateLimit1d,
		RateLimit7d:   k.RateLimit7d,
		Usage5h:       k.EffectiveUsage5h(),
		Usage1d:       k.EffectiveUsage1d(),
		Usage7d:       k.EffectiveUsage7d(),
		Window5hStart: k.Window5hStart,
		Window1dStart: k.Window1dStart,
		Window7dStart: k.Window7dStart,
		User:          UserFromServiceShallow(k.User),
		Group:         GroupFromServiceShallow(k.Group),
	}
	if k.Window5hStart != nil && !service.IsWindowExpired(k.Window5hStart, service.RateLimitWindow5h) {
		t := k.Window5hStart.Add(service.RateLimitWindow5h)
		out.Reset5hAt = &t
	}
	if k.Window1dStart != nil && !service.IsWindowExpired(k.Window1dStart, service.RateLimitWindow1d) {
		t := k.Window1dStart.Add(service.RateLimitWindow1d)
		out.Reset1dAt = &t
	}
	if k.Window7dStart != nil && !service.IsWindowExpired(k.Window7dStart, service.RateLimitWindow7d) {
		t := k.Window7dStart.Add(service.RateLimitWindow7d)
		out.Reset7dAt = &t
	}
	return out
}

func GroupFromServiceShallow(g *service.Group) *Group {
	if g == nil {
		return nil
	}
	out := groupFromServiceBase(g)
	return &out
}

func GroupFromService(g *service.Group) *Group {
	if g == nil {
		return nil
	}
	return GroupFromServiceShallow(g)
}

func GroupFromServiceAdmin(g *service.Group) *AdminGroup {
	if g == nil {
		return nil
	}
	return &AdminGroup{
		Group:                   groupFromServiceBase(g),
		AccountCount:            g.AccountCount,
		ActiveAccountCount:      g.ActiveAccountCount,
		RateLimitedAccountCount: g.RateLimitedAccountCount,
		SortOrder:               g.SortOrder,
	}
}

func groupFromServiceBase(g *service.Group) Group {
	return Group{
		ID:               g.ID,
		Name:             g.Name,
		Description:      g.Description,
		Platform:         g.Platform,
		RateMultiplier:   g.RateMultiplier,
		IsExclusive:      g.IsExclusive,
		Status:           g.Status,
		SubscriptionType: g.SubscriptionType,
		DailyLimitUSD:    g.DailyLimitUSD,
		WeeklyLimitUSD:   g.WeeklyLimitUSD,
		MonthlyLimitUSD:  g.MonthlyLimitUSD,
		RPMLimit:         g.RPMLimit,
		CreatedAt:        g.CreatedAt,
		UpdatedAt:        g.UpdatedAt,
	}
}

func RedeemCodeFromService(rc *service.RedeemCode) *RedeemCode {
	if rc == nil {
		return nil
	}
	out := redeemCodeFromServiceBase(rc)
	return &out
}

func RedeemCodeFromServiceAdmin(rc *service.RedeemCode) *AdminRedeemCode {
	if rc == nil {
		return nil
	}
	return &AdminRedeemCode{
		RedeemCode: redeemCodeFromServiceBase(rc),
		Notes:      rc.Notes,
	}
}

func redeemCodeFromServiceBase(rc *service.RedeemCode) RedeemCode {
	var notes *string
	if rc.Type == service.RedeemTypeBalance || rc.Type == service.RedeemTypeConcurrency {
		notes = &rc.Notes
	}
	return RedeemCode{
		ID:           rc.ID,
		Code:         rc.Code,
		Type:         rc.Type,
		Value:        rc.Value,
		Status:       rc.Status,
		UsedBy:       rc.UsedBy,
		UsedAt:       rc.UsedAt,
		CreatedAt:    rc.CreatedAt,
		ExpiresAt:    rc.ExpiresAt,
		GroupID:      rc.GroupID,
		PlanID:       rc.PlanID,
		ValidityDays: rc.ValidityDays,
		Notes:        notes,
		User:         UserFromServiceShallow(rc.User),
		Group:        GroupFromServiceShallow(rc.Group),
	}
}

func SettingFromService(s *service.Setting) *Setting {
	if s == nil {
		return nil
	}
	return &Setting{
		ID:        s.ID,
		Key:       s.Key,
		Value:     s.Value,
		UpdatedAt: s.UpdatedAt,
	}
}

func UserSubscriptionFromService(sub *service.UserSubscription) *UserSubscription {
	if sub == nil {
		return nil
	}
	out := userSubscriptionFromServiceBase(sub)
	return &out
}

func UserSubscriptionFromServiceAdmin(sub *service.UserSubscription) *AdminUserSubscription {
	if sub == nil {
		return nil
	}
	return &AdminUserSubscription{
		UserSubscription: userSubscriptionFromServiceBase(sub),
		AssignedBy:       sub.AssignedBy,
		AssignedAt:       sub.AssignedAt,
		Notes:            sub.Notes,
		AssignedByUser:   UserFromServiceShallow(sub.AssignedByUser),
	}
}

func userSubscriptionFromServiceBase(sub *service.UserSubscription) UserSubscription {
	return UserSubscription{
		ID:                 sub.ID,
		UserID:             sub.UserID,
		GroupID:            sub.GroupID,
		PlanID:             sub.PlanID,
		PlanName:           sub.PlanName,
		PlanPlatform:       sub.EffectivePlatform(sub.Group),
		QuotaUSD:           service.PlanQuotaUSD(sub.EffectiveMonthlyLimitUSD(sub.Group)),
		DailyLimitUSD:      sub.EffectiveDailyLimitUSD(sub.Group),
		WeeklyLimitUSD:     sub.EffectiveWeeklyLimitUSD(sub.Group),
		MonthlyLimitUSD:    sub.EffectiveMonthlyLimitUSD(sub.Group),
		StartsAt:           sub.StartsAt,
		ExpiresAt:          sub.ExpiresAt,
		Status:             sub.Status,
		DailyWindowStart:   sub.DailyWindowStart,
		WeeklyWindowStart:  sub.WeeklyWindowStart,
		MonthlyWindowStart: sub.MonthlyWindowStart,
		DailyUsageUSD:      sub.DailyUsageUSD,
		WeeklyUsageUSD:     sub.WeeklyUsageUSD,
		MonthlyUsageUSD:    sub.MonthlyUsageUSD,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
		User:               UserFromServiceShallow(sub.User),
		Group:              GroupFromServiceShallow(sub.Group),
	}
}

func PromoCodeFromService(pc *service.PromoCode) *PromoCode {
	if pc == nil {
		return nil
	}
	return &PromoCode{
		ID:          pc.ID,
		Code:        pc.Code,
		BonusAmount: pc.BonusAmount,
		MaxUses:     pc.MaxUses,
		UsedCount:   pc.UsedCount,
		Status:      pc.Status,
		ExpiresAt:   pc.ExpiresAt,
		Notes:       pc.Notes,
		CreatedAt:   pc.CreatedAt,
		UpdatedAt:   pc.UpdatedAt,
	}
}

func PromoCodeUsageFromService(u *service.PromoCodeUsage) *PromoCodeUsage {
	if u == nil {
		return nil
	}
	return &PromoCodeUsage{
		ID:          u.ID,
		PromoCodeID: u.PromoCodeID,
		UserID:      u.UserID,
		BonusAmount: u.BonusAmount,
		UsedAt:      u.UsedAt,
		User:        UserFromServiceShallow(u.User),
	}
}
