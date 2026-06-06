/**
 * Admin Payment API endpoints
 * Handles payment management operations for administrators
 */

import { apiClient } from '../client'
import type {
  DashboardStats,
  PaymentOrder,
  SubscriptionPlan,
  ProviderInstance
} from '@/types/payment'
import type { BasePaginationResponse } from '@/types'

/** Admin-facing payment config returned by GET /admin/payment/config */
export interface AdminPaymentConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  daily_limit: number
  order_timeout_minutes: number
  max_pending_orders: number
  enabled_payment_types: string[]
  balance_disabled: boolean
  balance_recharge_multiplier: number
  recharge_fee_rate: number
  load_balance_strategy: string
  product_name_prefix: string
  product_name_suffix: string
  help_image_url: string
  help_text: string
  stripe_publishable_key?: string
  cancel_rate_limit_enabled: boolean
  cancel_rate_limit_max: number
  cancel_rate_limit_window: number
  cancel_rate_limit_unit: string
  cancel_rate_limit_window_mode: string
  alipay_force_qrcode: boolean
  payment_visible_method_alipay_source?: string
  payment_visible_method_wxpay_source?: string
  payment_visible_method_alipay_enabled?: boolean
  payment_visible_method_wxpay_enabled?: boolean
}

/** Fields accepted by PUT /admin/payment/config (all optional via pointer semantics) */
export interface UpdatePaymentConfigRequest {
  enabled?: boolean
  min_amount?: number
  max_amount?: number
  daily_limit?: number
  order_timeout_minutes?: number
  max_pending_orders?: number
  enabled_payment_types?: string[]
  balance_disabled?: boolean
  balance_recharge_multiplier?: number
  recharge_fee_rate?: number
  load_balance_strategy?: string
  product_name_prefix?: string
  product_name_suffix?: string
  help_image_url?: string
  help_text?: string
  cancel_rate_limit_enabled?: boolean
  cancel_rate_limit_max?: number
  cancel_rate_limit_window?: number
  cancel_rate_limit_unit?: string
  cancel_rate_limit_window_mode?: string
  alipay_force_qrcode?: boolean
  payment_visible_method_alipay_source?: string
  payment_visible_method_wxpay_source?: string
  payment_visible_method_alipay_enabled?: boolean
  payment_visible_method_wxpay_enabled?: boolean
}

export interface AdminPaymentOrderDetail {
  order: PaymentOrder
  auditLogs?: PaymentAuditLog[]
  audit_logs?: PaymentAuditLog[]
}

export interface PaymentAuditLog {
  id: number
  action: string
  detail: string | null
  operator: string | null
  created_at: string
}

export interface SubscriptionPlanPayload {
  group_id?: number
  platform?: string
  name?: string
  description?: string
  price?: number
  original_price?: number
  validity_days?: number
  validity_unit?: string
  quota_usd?: number
  daily_limit_usd?: number
  weekly_limit_usd?: number
  product_name?: string
  features?: string
  for_sale?: boolean
  sort_order?: number
}

export const adminPaymentAPI = {
  // ==================== Config ====================

  /** Get payment configuration (admin view) */
  getConfig() {
    return apiClient.get<AdminPaymentConfig>('/admin/payment/config')
  },

  /** Update payment configuration */
  updateConfig(data: UpdatePaymentConfigRequest) {
    return apiClient.put('/admin/payment/config', data)
  },

  // ==================== Dashboard ====================

  /** Get payment dashboard statistics */
  getDashboard(days?: number) {
    return apiClient.get<DashboardStats>('/admin/payment/dashboard', {
      params: days ? { days } : undefined
    })
  },

  // ==================== Orders ====================

  /** Get all orders (paginated, with filters) */
  getOrders(params?: {
    page?: number
    page_size?: number
    status?: string
    payment_type?: string
    user_id?: number
    keyword?: string
    start_date?: string
    end_date?: string
    order_type?: string
  }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>('/admin/payment/orders', { params })
  },

  /** Get a specific order by ID */
  getOrder(id: number) {
    return apiClient.get<AdminPaymentOrderDetail>(`/admin/payment/orders/${id}`)
  },

  /** Cancel an order (admin) */
  cancelOrder(id: number) {
    return apiClient.post(`/admin/payment/orders/${id}/cancel`)
  },

  /** Retry recharge for a failed order */
  retryRecharge(id: number) {
    return apiClient.post(`/admin/payment/orders/${id}/retry`)
  },

  /** Process a refund */
  refundOrder(id: number, data: { amount: number; reason: string; deduct_balance?: boolean; force?: boolean }) {
    return apiClient.post(`/admin/payment/orders/${id}/refund`, data)
  },

  // ==================== Subscription Plans ====================

  /** Get all subscription plans */
  getPlans() {
    return apiClient.get<SubscriptionPlan[]>('/admin/payment/plans')
  },

  /** Create a subscription plan */
  createPlan(data: SubscriptionPlanPayload) {
    return apiClient.post<SubscriptionPlan>('/admin/payment/plans', data)
  },

  /** Update a subscription plan */
  updatePlan(id: number, data: SubscriptionPlanPayload) {
    return apiClient.put<SubscriptionPlan>(`/admin/payment/plans/${id}`, data)
  },

  /** Delete a subscription plan */
  deletePlan(id: number) {
    return apiClient.delete(`/admin/payment/plans/${id}`)
  },

  // ==================== Provider Instances ====================

  /** Get all provider instances */
  getProviders() {
    return apiClient.get<ProviderInstance[]>('/admin/payment/providers')
  },

  /** Create a provider instance */
  createProvider(data: Partial<ProviderInstance>) {
    return apiClient.post<ProviderInstance>('/admin/payment/providers', data)
  },

  /** Update a provider instance */
  updateProvider(id: number, data: Partial<ProviderInstance>) {
    return apiClient.put<ProviderInstance>(`/admin/payment/providers/${id}`, data)
  },

  /** Delete a provider instance */
  deleteProvider(id: number) {
    return apiClient.delete(`/admin/payment/providers/${id}`)
  }
}

export default adminPaymentAPI
