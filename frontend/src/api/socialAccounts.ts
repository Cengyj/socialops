/**
 * User Social Accounts API
 * Handles social account operations for regular users
 */

import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface UserSocialAccount {
  id: number
  name: string
  platform: string
  account_id?: string | null
  password?: string | null
  phone?: string | null
  email?: string | null
  email_password?: string | null
  account_status: string
  task_status: string
  task_message?: string | null
  source: string
  bound_ip?: string | null
  assigned_user_id?: number | null
  remark?: string | null
  created_at: string
  updated_at: string
}

export interface SocialBillingEstimate {
  unit_price: number
  action_count: number
  estimated_total: number
  subscription_allowance: number
  subscription_estimated_usage: number
  wallet_required: number
  wallet_balance: number
  can_afford: boolean
  charge_on_success_only: boolean
}

export interface SocialTaskLog {
  id: number
  social_account_id: number
  user_id: number
  action: string
  target?: string | null
  content?: string | null
  status: string
  result_message?: string | null
  price: number
  charged_amount: number
  charge_status: string
  charge_source?: string | null
  proxy_id?: number | null
  proxy_snapshot?: string | null
  billing_request_id?: string | null
  idempotency_key?: string | null
  executed_at?: string | null
  created_at: string
}

export interface ImportSocialAccountRequest {
  platform?: string
  name: string
}

export interface SubmitTaskRequest {
  account_ids: number[]
  action: string
  target?: string
  content?: string
  proxy_id?: number
  client_request_id?: string
  billing_request_id?: string
}

export interface SubmitTaskResponse {
  submitted: number
  enqueued: number
  failed_closed?: number
  billing_estimate?: SocialBillingEstimate
  logs: SocialTaskLog[]
}

const socialAccountsAPI = {
  // Social accounts
  listMyAccounts(params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<UserSocialAccount>> {
    return apiClient.get('/social-accounts', { params })
  },

  importMyAccount(data: ImportSocialAccountRequest): Promise<UserSocialAccount> {
    return apiClient.post('/social-accounts/import', data)
  },

  exportMyAccounts(): Promise<Blob> {
    return apiClient.get('/social-accounts/export', { responseType: 'blob' })
  },

  estimateTask(data: SubmitTaskRequest): Promise<SocialBillingEstimate> {
    return apiClient.post('/social-accounts/tasks/estimate', data)
  },

  submitTask(data: SubmitTaskRequest): Promise<SubmitTaskResponse> {
    return apiClient.post('/social-accounts/tasks', data)
  },

  listMyTaskLogs(params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<SocialTaskLog>> {
    return apiClient.get('/social-accounts/tasks', { params })
  },

  setDefaultProxy(accountId: number, proxyId?: number | null): Promise<UserSocialAccount> {
    return apiClient.put(`/social-accounts/${accountId}/default-proxy`, { proxy_id: proxyId ?? null })
  },
}

export default socialAccountsAPI
