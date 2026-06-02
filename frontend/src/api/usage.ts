import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface UsageLog {
  id: number
  user_id: number
  api_key_id?: number | null
  group_id?: number | null
  operation: string
  status: string
  quantity: number
  cost: number
  created_at: string
  completed_at?: string | null
}

export interface UsageStats {
  total_requests?: number
  total_operations?: number
  total_quantity?: number
  total_cost?: number
  success_count?: number
  failed_count?: number
  [key: string]: number | undefined
}

export interface UsageQueryParams {
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: string
  start_date?: string
  end_date?: string
  operation?: string
  status?: string
}

export const usageAPI = {
  async query(params?: UsageQueryParams): Promise<PaginatedResponse<UsageLog>> {
    const { data } = await apiClient.get<PaginatedResponse<UsageLog>>('/usage', { params })
    return data
  },

  async list(params?: UsageQueryParams): Promise<PaginatedResponse<UsageLog>> {
    const { data } = await apiClient.get<PaginatedResponse<UsageLog>>('/usage', { params })
    return data
  },

  async getById(id: number): Promise<UsageLog> {
    const { data } = await apiClient.get<UsageLog>(`/usage/${id}`)
    return data
  },

  async getStats(params?: Pick<UsageQueryParams, 'start_date' | 'end_date' | 'operation' | 'status'>): Promise<UsageStats> {
    const { data } = await apiClient.get<UsageStats>('/usage/stats', { params })
    return data
  },

  async getStatsByDateRange(startDate?: string, endDate?: string): Promise<UsageStats> {
    const { data } = await apiClient.get<UsageStats>('/usage/stats', {
      params: {
        start_date: startDate,
        end_date: endDate,
      },
    })
    return data
  },
}

export default usageAPI
