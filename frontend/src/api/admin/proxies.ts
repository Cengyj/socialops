import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface AdminProxy {
  id: number
  user_id: number
  name: string
  ip_type: string
  endpoint?: string | null
  status: string
  latency_ms?: number | null
  last_check_at?: string | null
  remark?: string | null
  created_at: string
  updated_at: string
}

export interface AdminProxyPayload {
  user_id?: number
  name?: string
  ip_type?: string
  endpoint?: string
  remark?: string
}

export interface AdminProxyCheckResult {
  id: number
  status: string
  latency_ms: number
  error?: string
}

const BASE = '/admin/proxies'

const proxiesAPI = {
  list(params?: {
    page?: number
    page_size?: number
    user_id?: number
    status?: string
    ip_type?: string
    search?: string
  }): Promise<PaginatedResponse<AdminProxy>> {
    return apiClient.get(BASE, { params })
  },

  create(data: AdminProxyPayload & { user_id: number; name: string }): Promise<AdminProxy> {
    return apiClient.post(BASE, data)
  },

  update(id: number, data: AdminProxyPayload): Promise<AdminProxy> {
    return apiClient.put(`${BASE}/${id}`, data)
  },

  delete(id: number): Promise<void> {
    return apiClient.delete(`${BASE}/${id}`)
  },

  test(id: number): Promise<AdminProxyCheckResult> {
    return apiClient.post(`${BASE}/${id}/test`)
  },
}

export default proxiesAPI
