package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/payment"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles user-facing SaaS payment requests.
type PaymentHandler struct {
	paymentService *service.PaymentService
	configService  *service.PaymentConfigService
}

// NewPaymentHandler creates a user-facing payment handler.
func NewPaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, configService: configService}
}

func (h *PaymentHandler) GetPaymentConfig(c *gin.Context) {
	cfg, err := h.configService.GetPaymentConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *PaymentHandler) GetPlans(c *gin.Context) {
	plans, err := h.configService.ListPlansForSale(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groupInfo := h.configService.GetGroupInfoMap(c.Request.Context(), plans)
	response.Success(c, dto.AvailableSubscriptionPlansFromEnt(plans, groupInfo))
}

func (h *PaymentHandler) GetCheckoutInfo(c *gin.Context) {
	ctx := c.Request.Context()
	limitsResp, err := h.configService.GetAvailableMethodLimits(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	cfg, err := h.configService.GetPaymentConfig(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	plans, err := h.configService.ListPlansForSale(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groupInfo := h.configService.GetGroupInfoMap(ctx, plans)
	planList := dto.AvailableSubscriptionPlansFromEnt(plans, groupInfo)

	response.Success(c, checkoutInfoResponse{
		Methods:                   limitsResp.Methods,
		GlobalMin:                 limitsResp.GlobalMin,
		GlobalMax:                 limitsResp.GlobalMax,
		Plans:                     planList,
		BalanceDisabled:           cfg.BalanceDisabled,
		BalanceRechargeMultiplier: cfg.BalanceRechargeMultiplier,
		RechargeFeeRate:           cfg.RechargeFeeRate,
		HelpText:                  cfg.HelpText,
		HelpImageURL:              cfg.HelpImageURL,
		StripePublishableKey:      cfg.StripePublishableKey,
		AlipayForceQRCode:         cfg.AlipayForceQRCode,
	})
}

type checkoutInfoResponse struct {
	Methods                   map[string]service.MethodLimits `json:"methods"`
	GlobalMin                 float64                         `json:"global_min"`
	GlobalMax                 float64                         `json:"global_max"`
	Plans                     []dto.SubscriptionPlan          `json:"plans"`
	BalanceDisabled           bool                            `json:"balance_disabled"`
	BalanceRechargeMultiplier float64                         `json:"balance_recharge_multiplier"`
	RechargeFeeRate           float64                         `json:"recharge_fee_rate"`
	HelpText                  string                          `json:"help_text"`
	HelpImageURL              string                          `json:"help_image_url"`
	StripePublishableKey      string                          `json:"stripe_publishable_key"`
	AlipayForceQRCode         bool                            `json:"alipay_force_qrcode"`
}

func (h *PaymentHandler) GetLimits(c *gin.Context) {
	resp, err := h.configService.GetAvailableMethodLimits(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, resp)
}

type CreateOrderRequest struct {
	Amount            float64 `json:"amount"`
	PaymentType       string  `json:"payment_type" binding:"required"`
	OpenID            string  `json:"openid"`
	WechatResumeToken string  `json:"wechat_resume_token"`
	ReturnURL         string  `json:"return_url"`
	PaymentSource     string  `json:"payment_source"`
	OrderType         string  `json:"order_type"`
	PlanID            int64   `json:"plan_id"`
	IsMobile          *bool   `json:"is_mobile,omitempty"`
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.WechatResumeToken) != "" {
		claims, err := h.paymentService.ParseWeChatPaymentResumeToken(req.WechatResumeToken)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if err := applyWeChatPaymentResumeClaims(&req, claims); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	mobile := isMobile(c)
	if req.IsMobile != nil {
		mobile = *req.IsMobile
	}
	result, err := h.paymentService.CreateOrder(c.Request.Context(), service.CreateOrderRequest{
		UserID:          subject.UserID,
		Amount:          req.Amount,
		PaymentType:     req.PaymentType,
		OpenID:          req.OpenID,
		ClientIP:        c.ClientIP(),
		IsMobile:        mobile,
		IsWeChatBrowser: isWeChatBrowser(c),
		SrcHost:         c.Request.Host,
		SrcURL:          c.Request.Referer(),
		ReturnURL:       req.ReturnURL,
		PaymentSource:   req.PaymentSource,
		OrderType:       req.OrderType,
		PlanID:          req.PlanID,
		Locale:          c.GetHeader("Accept-Language"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func applyWeChatPaymentResumeClaims(req *CreateOrderRequest, claims *service.WeChatPaymentResumeClaims) error {
	if req == nil || claims == nil {
		return infraerrors.BadRequest("INVALID_WECHAT_PAYMENT_RESUME_TOKEN", "wechat payment resume context is missing")
	}
	openid := strings.TrimSpace(claims.OpenID)
	if openid == "" {
		return infraerrors.BadRequest("INVALID_WECHAT_PAYMENT_RESUME_TOKEN", "wechat payment resume token missing openid")
	}

	paymentType := service.NormalizeVisibleMethod(claims.PaymentType)
	if paymentType == "" {
		paymentType = payment.TypeWxpay
	}
	if req.PaymentType != "" {
		requestPaymentType := service.NormalizeVisibleMethod(req.PaymentType)
		if requestPaymentType != "" && requestPaymentType != paymentType {
			return infraerrors.BadRequest("INVALID_WECHAT_PAYMENT_RESUME_TOKEN", "wechat payment resume token payment type mismatch")
		}
	}
	req.PaymentType = paymentType
	req.OpenID = openid

	if strings.TrimSpace(claims.Amount) != "" {
		amount, err := strconv.ParseFloat(strings.TrimSpace(claims.Amount), 64)
		if err != nil || amount <= 0 {
			return infraerrors.BadRequest("INVALID_WECHAT_PAYMENT_RESUME_TOKEN", fmt.Sprintf("invalid resume amount: %s", claims.Amount))
		}
		req.Amount = amount
	}
	if claims.OrderType != "" {
		req.OrderType = claims.OrderType
	}
	if claims.PlanID > 0 {
		req.PlanID = claims.PlanID
	}
	return nil
}

func (h *PaymentHandler) GetMyOrders(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	orders, total, err := h.paymentService.GetUserOrders(c.Request.Context(), subject.UserID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.Query("status"),
		OrderType:   c.Query("order_type"),
		PaymentType: c.Query("payment_type"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizePaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

func (h *PaymentHandler) GetOrder(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}
	order, err := h.paymentService.GetOrder(c.Request.Context(), orderID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizePaymentOrderForResponse(order))
}

func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}
	msg, err := h.paymentService.CancelOrder(c.Request.Context(), orderID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": msg})
}

type RefundRequestBody struct {
	Reason string `json:"reason"`
}

func (h *PaymentHandler) RequestRefund(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}
	var req RefundRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.paymentService.RequestRefund(c.Request.Context(), orderID, subject.UserID, req.Reason); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "refund requested"})
}

func (h *PaymentHandler) GetRefundEligibleProviders(c *gin.Context) {
	ids, err := h.configService.GetUserRefundEligibleInstanceIDs(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"provider_instance_ids": ids})
}

type VerifyOrderRequest struct {
	OutTradeNo string `json:"out_trade_no" binding:"required"`
}

type ResolveOrderByResumeTokenRequest struct {
	ResumeToken string `json:"resume_token" binding:"required"`
}

func (h *PaymentHandler) VerifyOrder(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var req VerifyOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	order, err := h.paymentService.VerifyOrderByOutTradeNo(c.Request.Context(), req.OutTradeNo, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizePaymentOrderForResponse(order))
}

type PublicOrderResult struct {
	ID                  int64      `json:"id"`
	OutTradeNo          string     `json:"out_trade_no"`
	Amount              float64    `json:"amount"`
	PayAmount           float64    `json:"pay_amount"`
	FeeRate             float64    `json:"fee_rate"`
	Currency            string     `json:"currency"`
	PaymentType         string     `json:"payment_type"`
	OrderType           string     `json:"order_type"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	RefundAmount        float64    `json:"refund_amount"`
	RefundReason        *string    `json:"refund_reason,omitempty"`
	RefundRequestedAt   *time.Time `json:"refund_requested_at,omitempty"`
	RefundRequestedBy   *string    `json:"refund_requested_by,omitempty"`
	RefundRequestReason *string    `json:"refund_request_reason,omitempty"`
	PlanID              *int64     `json:"plan_id,omitempty"`
}

func buildPublicOrderResult(order *dbent.PaymentOrder) PublicOrderResult {
	return PublicOrderResult{
		ID:                  order.ID,
		OutTradeNo:          order.OutTradeNo,
		Amount:              order.Amount,
		PayAmount:           order.PayAmount,
		FeeRate:             order.FeeRate,
		Currency:            service.PaymentOrderCurrency(order),
		PaymentType:         order.PaymentType,
		OrderType:           order.OrderType,
		Status:              order.Status,
		CreatedAt:           order.CreatedAt,
		ExpiresAt:           order.ExpiresAt,
		PaidAt:              order.PaidAt,
		CompletedAt:         order.CompletedAt,
		RefundAmount:        order.RefundAmount,
		RefundReason:        order.RefundReason,
		RefundRequestedAt:   order.RefundRequestedAt,
		RefundRequestedBy:   order.RefundRequestedBy,
		RefundRequestReason: order.RefundRequestReason,
		PlanID:              order.PlanID,
	}
}

func (h *PaymentHandler) VerifyOrderPublic(c *gin.Context) {
	var req VerifyOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	order, err := h.paymentService.VerifyOrderPublic(c.Request.Context(), req.OutTradeNo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildPublicOrderResult(order))
}

func (h *PaymentHandler) ResolveOrderPublicByResumeToken(c *gin.Context) {
	var req ResolveOrderByResumeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	order, err := h.paymentService.GetPublicOrderByResumeToken(c.Request.Context(), req.ResumeToken)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildPublicOrderResult(order))
}

func requireAuth(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func isMobile(c *gin.Context) bool {
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	for _, keyword := range []string{"mobile", "android", "iphone", "ipad", "ipod"} {
		if strings.Contains(ua, keyword) {
			return true
		}
	}
	return false
}

type PaymentOrderResult struct {
	ID                  int64      `json:"id"`
	UserID              int64      `json:"user_id"`
	Amount              float64    `json:"amount"`
	PayAmount           float64    `json:"pay_amount"`
	FeeRate             float64    `json:"fee_rate"`
	Currency            string     `json:"currency"`
	PaymentType         string     `json:"payment_type"`
	OutTradeNo          string     `json:"out_trade_no"`
	Status              string     `json:"status"`
	OrderType           string     `json:"order_type"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	RefundAmount        float64    `json:"refund_amount"`
	RefundReason        *string    `json:"refund_reason,omitempty"`
	RefundRequestedAt   *time.Time `json:"refund_requested_at,omitempty"`
	RefundRequestedBy   *string    `json:"refund_requested_by,omitempty"`
	RefundRequestReason *string    `json:"refund_request_reason,omitempty"`
	PlanID              *int64     `json:"plan_id,omitempty"`
	ProviderInstanceID  *string    `json:"provider_instance_id,omitempty"`
}

func sanitizePaymentOrdersForResponse(orders []*dbent.PaymentOrder) []PaymentOrderResult {
	out := make([]PaymentOrderResult, 0, len(orders))
	for _, order := range orders {
		if item := sanitizePaymentOrderForResponse(order); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func sanitizePaymentOrderForResponse(order *dbent.PaymentOrder) *PaymentOrderResult {
	if order == nil {
		return nil
	}
	return &PaymentOrderResult{
		ID:                  order.ID,
		UserID:              order.UserID,
		Amount:              order.Amount,
		PayAmount:           order.PayAmount,
		FeeRate:             order.FeeRate,
		Currency:            service.PaymentOrderCurrency(order),
		PaymentType:         order.PaymentType,
		OutTradeNo:          order.OutTradeNo,
		Status:              order.Status,
		OrderType:           order.OrderType,
		CreatedAt:           order.CreatedAt,
		ExpiresAt:           order.ExpiresAt,
		PaidAt:              order.PaidAt,
		CompletedAt:         order.CompletedAt,
		RefundAmount:        order.RefundAmount,
		RefundReason:        order.RefundReason,
		RefundRequestedAt:   order.RefundRequestedAt,
		RefundRequestedBy:   order.RefundRequestedBy,
		RefundRequestReason: order.RefundRequestReason,
		PlanID:              order.PlanID,
		ProviderInstanceID:  order.ProviderInstanceID,
	}
}

func isWeChatBrowser(c *gin.Context) bool {
	return strings.Contains(strings.ToLower(c.GetHeader("User-Agent")), "micromessenger")
}
