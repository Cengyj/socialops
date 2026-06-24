import { apiClient } from '../client'
import { unwrapData } from '../utils'
import type { ProxyType } from '../proxies'
import type { PaginatedResponse } from '@/types'

const BASE = '/admin/global-proxies'

export interface AdminGlobalProxy {
  id: number
  name: string
  ip_type: ProxyType
  endpoint?: string | null
  status: string
  latency_ms?: number | null
  last_check_at?: string | null
  last_used_at?: string | null
  remark?: string | null
  created_at: string
  updated_at: string
}

export interface AdminGlobalProxyPayload {
  name?: string
  ip_type?: ProxyType
  endpoint?: string
  remark?: string
}

export interface AdminGlobalProxyCheckResult {
  id: number
  status: string
  latency_ms: number
  error?: string
}

export interface AdminGlobalProxyListParams {
  page?: number
  page_size?: number
  status?: string
  ip_type?: ProxyType
  search?: string
}

export const adminGlobalProxiesAPI = {
  list(params?: AdminGlobalProxyListParams): Promise<PaginatedResponse<AdminGlobalProxy>> {
    return unwrapData(apiClient.get<PaginatedResponse<AdminGlobalProxy>>(BASE, { params }))
  },

  create(data: AdminGlobalProxyPayload & { name: string }): Promise<AdminGlobalProxy> {
    return unwrapData(apiClient.post<AdminGlobalProxy>(BASE, data))
  },

  update(id: number, data: AdminGlobalProxyPayload): Promise<AdminGlobalProxy> {
    return unwrapData(apiClient.put<AdminGlobalProxy>(`${BASE}/${id}`, data))
  },

  delete(id: number): Promise<void> {
    return unwrapData(apiClient.delete<void>(`${BASE}/${id}`))
  },

  test(id: number): Promise<AdminGlobalProxyCheckResult> {
    return unwrapData(apiClient.post<AdminGlobalProxyCheckResult>(`${BASE}/${id}/test`))
  },

  testAll(): Promise<AdminGlobalProxyCheckResult[]> {
    return unwrapData(apiClient.post<AdminGlobalProxyCheckResult[]>(`${BASE}/test`))
  },
}

export default adminGlobalProxiesAPI
