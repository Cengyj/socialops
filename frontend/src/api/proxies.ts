import { apiClient } from './client'
import { unwrapData } from './utils'
import type { PaginatedResponse } from '@/types'

export type ProxyType = 'residential' | 'static' | 'dynamic' | 'mobile' | 'datacenter'

export interface UserProxy {
  id: number
  user_id: number
  name: string
  ip_type: ProxyType
  endpoint?: string | null
  status: string
  latency_ms?: number | null
  last_check_at?: string | null
  remark?: string | null
  created_at: string
  updated_at: string
}

export interface ProxyPayload {
  name?: string
  ip_type?: ProxyType
  endpoint?: string
  remark?: string
}

export interface ProxyCheckResult {
  id: number
  status: string
  latency_ms: number
  error?: string
}

const BASE = '/proxies'

const proxiesAPI = {
  list(params?: {
    page?: number
    page_size?: number
    status?: string
    ip_type?: ProxyType
    search?: string
  }): Promise<PaginatedResponse<UserProxy>> {
    return unwrapData(apiClient.get<PaginatedResponse<UserProxy>>(BASE, { params }))
  },

  listUsable(): Promise<UserProxy[]> {
    return unwrapData(apiClient.get<UserProxy[]>(`${BASE}/usable`))
  },

  create(data: ProxyPayload & { name: string }): Promise<UserProxy> {
    return unwrapData(apiClient.post<UserProxy>(BASE, data))
  },

  update(id: number, data: ProxyPayload): Promise<UserProxy> {
    return unwrapData(apiClient.put<UserProxy>(`${BASE}/${id}`, data))
  },

  delete(id: number): Promise<void> {
    return unwrapData(apiClient.delete<void>(`${BASE}/${id}`))
  },

  test(id: number): Promise<ProxyCheckResult> {
    return unwrapData(apiClient.post<ProxyCheckResult>(`${BASE}/${id}/test`))
  },

  testAll(): Promise<ProxyCheckResult[]> {
    return unwrapData(apiClient.post<ProxyCheckResult[]>(`${BASE}/test`))
  },
}

export default proxiesAPI
