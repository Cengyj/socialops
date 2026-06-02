/**
 * Admin Social Accounts API
 * Handles social account pool management for administrators
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { SocialTaskLog, SubmitTaskRequest } from '../socialAccounts'

export interface SocialAccount {
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

export interface CreateSocialAccountRequest {
  name: string
  platform: string
  account_id?: string
  password?: string
  phone?: string
  email?: string
  email_password?: string
  source?: string
  remark?: string
}

export interface UpdateSocialAccountRequest {
  name?: string
  account_id?: string
  password?: string
  phone?: string
  email?: string
  email_password?: string
  account_status?: string
  task_status?: string
  task_message?: string
  remark?: string
}

export interface SocialAccountStats {
  total: number
  stored: number
  available: number
}

export interface AdminSocialBillingEstimate {
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

export interface AdminSocialTaskEstimateResponse {
  action: string
  estimates: Record<string, AdminSocialBillingEstimate>
}

export interface AdminSubmitTaskResponse {
  submitted: number
  enqueued: number
  failed_closed?: number
  billing_estimates?: Record<string, AdminSocialBillingEstimate>
  logs: SocialTaskLog[]
}

const BASE = '/admin/social-accounts'

const socialAccountsAdminAPI = {
  list(params?: {
    page?: number
    page_size?: number
    platform?: string
    account_status?: string
    task_status?: string
    source?: string
    unassigned?: boolean
  }): Promise<PaginatedResponse<SocialAccount>> {
    return apiClient.get(BASE, { params })
  },

  getById(id: number): Promise<SocialAccount> {
    return apiClient.get(`${BASE}/${id}`)
  },

  create(data: CreateSocialAccountRequest): Promise<SocialAccount> {
    return apiClient.post(BASE, data)
  },

  update(id: number, data: UpdateSocialAccountRequest): Promise<SocialAccount> {
    return apiClient.put(`${BASE}/${id}`, data)
  },

  delete(id: number): Promise<void> {
    return apiClient.delete(`${BASE}/${id}`)
  },

  assign(id: number, userId: number): Promise<SocialAccount> {
    return apiClient.post(`${BASE}/${id}/assign`, { user_id: userId })
  },

  reclaim(id: number): Promise<SocialAccount> {
    return apiClient.post(`${BASE}/${id}/reclaim`)
  },

  setDefaultProxy(id: number, proxyId?: number | null): Promise<SocialAccount> {
    return apiClient.put(`${BASE}/${id}/default-proxy`, { proxy_id: proxyId ?? null })
  },

  register(data: CreateSocialAccountRequest): Promise<SocialAccount> {
    return apiClient.post(`${BASE}/register`, data)
  },

  batchDelete(ids: number[]): Promise<{ deleted: number }> {
    return apiClient.post(`${BASE}/batch-delete`, { ids })
  },

  getStats(): Promise<SocialAccountStats> {
    return apiClient.get(`${BASE}/stats`)
  },

  importAccounts(file: File): Promise<{ total: number; created: number; skipped: number; errors: string[] }> {
    const formData = new FormData()
    formData.append('file', file)
    return apiClient.post(`${BASE}/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  exportAccounts(): Promise<Blob> {
    return apiClient.get(`${BASE}/export`, { responseType: 'blob' })
  },

  estimateTask(data: SubmitTaskRequest): Promise<AdminSocialTaskEstimateResponse> {
    return apiClient.post(`${BASE}/tasks/estimate`, data)
  },

  submitTask(data: SubmitTaskRequest): Promise<AdminSubmitTaskResponse> {
    return apiClient.post(`${BASE}/tasks`, data)
  },

  listTaskLogs(params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<SocialTaskLog>> {
    return apiClient.get(`${BASE}/tasks`, { params })
  },
}

export default socialAccountsAdminAPI
