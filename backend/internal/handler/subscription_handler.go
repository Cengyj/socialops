package handler

import (
	"time"

	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type SubscriptionSummaryItem struct {
	ID              int64    `json:"id"`
	GroupName       string   `json:"group_name"`
	Status          string   `json:"status"`
	DailyProgress   *float64 `json:"daily_progress"`
	WeeklyProgress  *float64 `json:"weekly_progress"`
	MonthlyProgress *float64 `json:"monthly_progress"`
	ExpiresAt       *string  `json:"expires_at"`
	DaysRemaining   *int     `json:"days_remaining"`
}

type SubscriptionSummaryResponse struct {
	ActiveCount   int                       `json:"active_count"`
	Subscriptions []SubscriptionSummaryItem `json:"subscriptions"`
}

type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	subscriptions, err := h.subscriptionService.ListUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.UserSubscription, 0, len(subscriptions))
	for i := range subscriptions {
		out = append(out, *dto.UserSubscriptionFromService(&subscriptions[i]))
	}
	response.Success(c, out)
}

func (h *SubscriptionHandler) GetActive(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.UserSubscription, 0, len(subscriptions))
	for i := range subscriptions {
		out = append(out, *dto.UserSubscriptionFromService(&subscriptions[i]))
	}
	response.Success(c, out)
}

func (h *SubscriptionHandler) GetProgress(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]gin.H, 0, len(subscriptions))
	for i := range subscriptions {
		progress, err := h.subscriptionService.GetSubscriptionProgress(c.Request.Context(), subscriptions[i].ID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		items = append(items, gin.H{
			"subscription": dto.UserSubscriptionFromService(&subscriptions[i]),
			"progress":     progress,
		})
	}
	response.Success(c, items)
}

func (h *SubscriptionHandler) GetSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]SubscriptionSummaryItem, 0, len(subscriptions))
	for i := range subscriptions {
		items = append(items, subscriptionSummaryItemFromService(&subscriptions[i]))
	}
	response.Success(c, SubscriptionSummaryResponse{
		ActiveCount:   len(items),
		Subscriptions: items,
	})
}

func subscriptionSummaryItemFromService(sub *service.UserSubscription) SubscriptionSummaryItem {
	if sub == nil {
		return SubscriptionSummaryItem{}
	}
	return SubscriptionSummaryItem{
		ID:              sub.ID,
		GroupName:       sub.EffectiveDisplayName(sub.Group),
		Status:          sub.Status,
		DailyProgress:   subscriptionUsageProgress(sub.DailyUsageUSD, sub.EffectiveDailyLimitUSD(sub.Group)),
		WeeklyProgress:  subscriptionUsageProgress(sub.WeeklyUsageUSD, sub.EffectiveWeeklyLimitUSD(sub.Group)),
		MonthlyProgress: subscriptionUsageProgress(sub.MonthlyUsageUSD, sub.EffectiveMonthlyLimitUSD(sub.Group)),
		ExpiresAt:       subscriptionSummaryTime(sub.ExpiresAt),
		DaysRemaining:   subscriptionSummaryDaysRemaining(sub),
	}
}

func subscriptionUsageProgress(used float64, limit *float64) *float64 {
	if limit == nil || *limit <= 0 {
		return nil
	}
	progress := used / *limit * 100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	return &progress
}

func subscriptionSummaryTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func subscriptionSummaryDaysRemaining(sub *service.UserSubscription) *int {
	if sub == nil || sub.ExpiresAt.IsZero() {
		return nil
	}
	days := sub.DaysRemaining()
	return &days
}
