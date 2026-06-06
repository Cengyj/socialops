import { apiClient } from '../client'
import type { SocialAccount, SocialAccountBatchResult } from '../accountWorkbench'
import type { PaginatedResponse } from '@/types'

const BASE = '/admin/total-accounts'

const unwrapData = async <T>(request: Promise<{ data: T }>): Promise<T> => {
  const { data } = await request
  return data
}

export const totalAccountsAPI = {
  list(params?: {
    page?: number
    page_size?: number
    platform?: string
    account_status?: string
    task_status?: string
    unassigned?: boolean
  }): Promise<PaginatedResponse<SocialAccount>> {
    return unwrapData(apiClient.get<PaginatedResponse<SocialAccount>>(BASE, { params }))
  },

  assign(id: number, userId: number): Promise<SocialAccount> {
    return unwrapData(apiClient.post<SocialAccount>(`${BASE}/${id}/assign`, { user_id: userId }))
  },

  reclaim(id: number): Promise<SocialAccount> {
    return unwrapData(apiClient.post<SocialAccount>(`${BASE}/${id}/reclaim`))
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
}

export default totalAccountsAPI
