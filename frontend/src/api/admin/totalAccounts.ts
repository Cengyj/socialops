import { apiClient } from '../client'
import { unwrapData } from '../utils'
import type { SocialAccount, SocialAccountBatchResult, UpdateSocialAccountRequest } from '../accountWorkbench'
import type { PaginatedResponse } from '@/types'

const BASE = '/admin/total-accounts'

export interface TotalAccountListParams {
  page?: number
  page_size?: number
  platform?: string
  account_status?: string
  task_status?: string
  search?: string
  assigned?: boolean
  unassigned?: boolean
}

export type TotalAccountExportParams = Omit<TotalAccountListParams, 'page' | 'page_size'> & {
  account_ids?: number[]
}

export interface TotalAccountImportResult extends SocialAccountBatchResult {
  created: number
  duplicates: number
  errors: string[]
}

export const totalAccountsAPI = {
  list(params?: TotalAccountListParams): Promise<PaginatedResponse<SocialAccount>> {
    return unwrapData(apiClient.get<PaginatedResponse<SocialAccount>>(BASE, { params }))
  },

  batchAssign(ids: number[], userId: number): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${BASE}/batch-assign`, { ids, user_id: userId }))
  },

  batchReclaim(ids: number[]): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${BASE}/batch-reclaim`, { ids }))
  },

  batchDelete(ids: number[]): Promise<SocialAccountBatchResult> {
    return unwrapData(apiClient.post<SocialAccountBatchResult>(`${BASE}/batch-delete`, { ids }))
  },

  update(id: number, data: UpdateSocialAccountRequest): Promise<SocialAccount> {
    return unwrapData(apiClient.put<SocialAccount>(`${BASE}/${id}`, data))
  },

  importAccounts(file: File, platform = 'x_twitter'): Promise<TotalAccountImportResult> {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('platform', platform)
    return unwrapData(apiClient.post<TotalAccountImportResult>(`${BASE}/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }))
  },

  exportAccounts(params?: TotalAccountExportParams): Promise<Blob> {
    const { account_ids: accountIds, ...rest } = params ?? {}
    const exportParams = params
      ? { ...rest, ...(accountIds?.length ? { account_ids: accountIds.join(',') } : {}) }
      : undefined
    const config = exportParams ? { params: exportParams, responseType: 'blob' as const } : { responseType: 'blob' as const }
    return unwrapData<Blob>(apiClient.get<Blob>(`${BASE}/export`, config))
  },
}

export default totalAccountsAPI
