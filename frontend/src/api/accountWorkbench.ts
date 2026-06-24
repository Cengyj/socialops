/**
 * Unified account workbench API.
 * User endpoints live at /accounts. Admin-only workbench capabilities live at
 * /admin/accounts; ownership transfer remains in the total account pool UI.
 */

import { apiClient } from './client'
import { unwrapData } from './utils'
import type { PaginatedResponse } from '@/types'
import type { ExecutableSocialTaskAction, SocialTaskPayload, SocialTaskTemplateSnapshot } from '@/types/socialTask'
export type {
  DirectSocialTaskAction,
  ExecutableSocialTaskAction,
  ParameterSocialTaskAction,
  SocialPostPayload,
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  SocialTaskPayload,
  SocialTaskTemplateParams,
  SocialTaskTemplateSnapshot,
} from '@/types/socialTask'

export interface UserSocialAccount {
  id: number
  name: string
  platform: string
  username?: string | null
  identity_kind?: string | null
  identity_key?: string | null
  platform_user_id?: string | null
  password?: string | null
  phone?: string | null
  email?: string | null
  email_password?: string | null
  two_factor?: string | null
  backup_code?: string | null
  email_client_id?: string | null
  email_token?: string | null
  registration_ip?: string | null
  auth_cookie?: string | null
  execution_auth?: string | null
  account_status: string
  task_status: string
  task_message?: string | null
  default_proxy_snapshot?: string | null
  remark?: string | null
  default_proxy_configured?: boolean
  created_at: string
  updated_at: string
}

export interface SocialAccount extends UserSocialAccount {
  assigned_user_id?: number | null
  assigned_user_email?: string | null
}

export interface SocialTaskLog {
  id: number
  social_account_id: number
  action: ExecutableSocialTaskAction
  platform?: string | null
  account_name?: string | null
  status: string
  target?: string | null
  content?: string | null
  payload?: SocialTaskPayload | null
  template_snapshot?: SocialTaskTemplateSnapshot | null
  result_message?: string | null
  charged?: boolean
  charged_amount: number
  charge_status: string
  executed_at?: string | null
  created_at: string
}

export interface AdminSocialTaskLog extends SocialTaskLog {
  user_id: number
  price: number
  charge_source?: string | null
  proxy_id?: number | null
  proxy_snapshot?: string | null
  billing_request_id?: string | null
  idempotency_key?: string | null
}

export interface ImportSocialAccountRequest {
  platform?: string
  name: string
  platform_user_id?: string
  password?: string
  phone?: string
  email?: string
  email_password?: string
  auth_cookie?: string
  execution_auth?: string
  two_factor?: string
  backup_code?: string
  email_client_id?: string
  email_token?: string
  registration_ip?: string
  remark?: string
}

export interface BatchImportSocialAccountResponse {
  total: number
  succeeded: number
  imported: number
  skipped: number
  failed: number
  duplicates: number
  errors: string[]
  items: SocialAccountBatchItemResult[]
  accounts: UserSocialAccount[]
}

export interface BatchDeleteSocialAccountResponse {
  total: number
  succeeded: number
  removed: number
  skipped: number
  failed: number
  errors: string[]
  items: SocialAccountBatchItemResult[]
}

export interface SocialAccountBatchItemResult {
  id?: number
  name?: string
  status: string
  reason?: string
  error?: string
}

export interface SubmitTaskRequest {
  account_ids: number[]
  action: ExecutableSocialTaskAction
  client_request_id?: string
}

export interface AdminSubmitTaskRequest {
  account_ids: number[]
  action: ExecutableSocialTaskAction
  target?: string
  content?: string
  client_request_id?: string
}

export interface SubmitTaskResponse {
  submitted: number
  enqueued: number
  failed_closed?: number
  logs: SocialTaskLog[]
}

export interface ListTaskLogsParams {
  account_ids?: number[]
  statuses?: string[]
  limit?: number
}

export interface ListTaskLogsResponse {
  logs: SocialTaskLog[]
}

export interface CreateSocialAccountRequest {
  name: string
  platform: string
  platform_user_id?: string
  password?: string
  phone?: string
  email?: string
  email_password?: string
  two_factor?: string
  backup_code?: string
  email_client_id?: string
  email_token?: string
  registration_ip?: string
  auth_cookie?: string
  execution_auth?: string
  default_proxy_snapshot?: string
  remark?: string
}

export interface UpdateSocialAccountRequest {
  name?: string
  platform_user_id?: string
  password?: string
  phone?: string
  email?: string
  email_password?: string
  two_factor?: string
  backup_code?: string
  email_client_id?: string
  email_token?: string
  registration_ip?: string
  auth_cookie?: string
  execution_auth?: string
  account_status?: string
  task_status?: string
  task_message?: string
  default_proxy_snapshot?: string
  remark?: string
}

export interface UpdateMySocialAccountRequest {
  password?: string
  phone?: string
  email?: string
  email_password?: string
  two_factor?: string
  backup_code?: string
  email_client_id?: string
  email_token?: string
  registration_ip?: string
  auth_cookie?: string
  execution_auth?: string
  remark?: string
}

export interface SocialAccountStats {
  total: number
  stored: number
  available: number
}

export interface MyAccountListParams {
  page?: number
  page_size?: number
  search?: string
  platform?: string
  account_status?: string
  task_status?: string
}

export type MyAccountExportParams = Omit<MyAccountListParams, 'page' | 'page_size'> & {
  account_ids?: number[]
}

export interface SocialAccountBatchResult {
  total: number
  succeeded: number
  skipped: number
  failed: number
  duplicates?: number
  errors?: string[]
  items: SocialAccountBatchItemResult[]
}

export type DefaultProxyAssignmentMode = 'specific' | 'random' | 'clear'

export interface DefaultProxyAssignmentRequest {
  account_ids: number[]
  mode: DefaultProxyAssignmentMode
  proxy_id?: number | null
}

export interface AdminSubmitTaskResponse {
  submitted: number
  enqueued: number
  failed_closed?: number
  logs: AdminSocialTaskLog[]
}

const USER_BASE = '/accounts'
const ADMIN_BASE = '/admin/accounts'

const accountWorkbenchAPI = {
  listMyAccounts(params?: MyAccountListParams): Promise<PaginatedResponse<UserSocialAccount>> {
    return unwrapData(apiClient.get<PaginatedResponse<UserSocialAccount>>(USER_BASE, { params }))
  },

  batchImportMyAccounts(accounts: ImportSocialAccountRequest[]): Promise<BatchImportSocialAccountResponse> {
    return unwrapData(apiClient.post<BatchImportSocialAccountResponse>(`${USER_BASE}/batch-import`, { accounts }))
  },

  updateMyAccount(accountId: number, data: UpdateMySocialAccountRequest): Promise<UserSocialAccount> {
    return unwrapData(apiClient.put<UserSocialAccount>(`${USER_BASE}/${accountId}`, data))
  },

  deleteMyAccount(accountId: number): Promise<void> {
    return unwrapData(apiClient.delete<void>(`${USER_BASE}/${accountId}`))
  },

  batchDeleteMyAccounts(ids: number[]): Promise<BatchDeleteSocialAccountResponse> {
    return unwrapData(apiClient.post<BatchDeleteSocialAccountResponse>(`${USER_BASE}/batch-delete`, { ids }))
  },

  exportMyAccounts(params?: MyAccountExportParams): Promise<Blob> {
    const { account_ids: accountIds, ...rest } = params ?? {}
    const exportParams = params
      ? { ...rest, ...(accountIds?.length ? { account_ids: accountIds.join(',') } : {}) }
      : undefined
    const config = exportParams ? { params: exportParams, responseType: 'blob' as const } : { responseType: 'blob' as const }
    return unwrapData(apiClient.get<Blob>(`${USER_BASE}/export`, config))
  },

  submitTask(data: SubmitTaskRequest): Promise<SubmitTaskResponse> {
    return unwrapData(apiClient.post<SubmitTaskResponse>(`${USER_BASE}/tasks`, data))
  },

  listTaskLogs(params?: ListTaskLogsParams): Promise<ListTaskLogsResponse> {
    return unwrapData(apiClient.get<ListTaskLogsResponse>(`${USER_BASE}/tasks`, {
      params: {
        ...params,
        account_ids: params?.account_ids?.join(','),
        statuses: params?.statuses?.join(','),
      },
    }))
  },

  setDefaultProxy(accountId: number, proxyId?: number | null): Promise<UserSocialAccount> {
    return unwrapData(apiClient.put<UserSocialAccount>(`${USER_BASE}/${accountId}/default-proxy`, { proxy_id: proxyId ?? null }))
  },

  batchSetDefaultProxy(data: DefaultProxyAssignmentRequest): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${USER_BASE}/default-proxy`, data))
  },
}

export const accountWorkbenchAdminAPI = {
  list(params?: {
    page?: number
    page_size?: number
    platform?: string
    account_status?: string
    task_status?: string
    unassigned?: boolean
  }): Promise<PaginatedResponse<SocialAccount>> {
    return unwrapData(apiClient.get<PaginatedResponse<SocialAccount>>(ADMIN_BASE, { params }))
  },

  getById(id: number): Promise<SocialAccount> {
    return unwrapData(apiClient.get<SocialAccount>(`${ADMIN_BASE}/${id}`))
  },

  create(data: CreateSocialAccountRequest): Promise<SocialAccount> {
    return unwrapData(apiClient.post<SocialAccount>(ADMIN_BASE, data))
  },

  update(id: number, data: UpdateSocialAccountRequest): Promise<SocialAccount> {
    return unwrapData(apiClient.put<SocialAccount>(`${ADMIN_BASE}/${id}`, data))
  },

  delete(id: number): Promise<void> {
    return unwrapData(apiClient.delete<void>(`${ADMIN_BASE}/${id}`))
  },

  batchDelete(ids: number[]): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${ADMIN_BASE}/batch-delete`, { ids }))
  },

  storeWorkbenchAccounts(ids: number[]): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${ADMIN_BASE}/store-workbench`, { account_ids: ids }))
  },

  getStats(): Promise<SocialAccountStats> {
    return unwrapData(apiClient.get<SocialAccountStats>(`${ADMIN_BASE}/stats`))
  },

  importAccounts(file: File, platform = 'x_twitter'): Promise<{ total: number; succeeded: number; created: number; skipped: number; failed: number; duplicates: number; errors: string[]; items: SocialAccountBatchItemResult[] }> {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('platform', platform)
    return unwrapData(apiClient.post<{ total: number; succeeded: number; created: number; skipped: number; failed: number; duplicates: number; errors: string[]; items: SocialAccountBatchItemResult[] }>(`${ADMIN_BASE}/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }))
  },

  exportAccounts(): Promise<Blob> {
    return unwrapData(apiClient.get<Blob>(`${ADMIN_BASE}/export`, { responseType: 'blob' }))
  },

  submitTask(data: AdminSubmitTaskRequest): Promise<AdminSubmitTaskResponse> {
    return unwrapData(apiClient.post<AdminSubmitTaskResponse>(`${ADMIN_BASE}/tasks`, data))
  },

}

export default accountWorkbenchAPI
