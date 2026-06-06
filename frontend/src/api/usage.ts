import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface UsageLog {
  id: number
  user_id: number
  api_key_id?: number | null
  group_id?: number | null
  social_account_id: number
  platform: string
  account_name: string
  operation: string
  status: string
  quantity: number
  cost: number
  charge_status: string
  result_message?: string | null
  created_at: string
  completed_at?: string | null
}

export interface UsageStats {
  total_requests?: number
  total_operations?: number
  total_quantity?: number
  total_tokens?: number
  total_cost?: number
  total_actual_cost?: number
  success_count?: number
  failed_count?: number
  [key: string]: number | undefined
}

export interface PlatformDashboardStats {
  platform: string
  total_requests?: number
  total_actual_cost?: number
  today_requests?: number
  today_actual_cost?: number
}

export interface UserDashboardStats {
  total_requests?: number
  today_requests?: number
  total_actual_cost?: number
  today_actual_cost?: number
  average_duration_ms?: number
  rpm?: number
  by_platform?: PlatformDashboardStats[]
}

export interface DashboardTrendPoint {
  date: string
  requests?: number
  actual_cost?: number
  cost?: number
}

export interface DashboardTrendParams {
  granularity?: 'day' | 'hour'
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

function normalizeTrend(data: DashboardTrendPoint[] | { trend?: DashboardTrendPoint[] } | null | undefined): DashboardTrendPoint[] {
  if (Array.isArray(data)) return data
  return data?.trend ?? []
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

  async getDashboardStats(): Promise<UserDashboardStats> {
    const { data } = await apiClient.get<UserDashboardStats>('/usage/dashboard/stats')
    return data
  },

  async getDashboardTrend(params?: DashboardTrendParams): Promise<DashboardTrendPoint[]> {
    const { data } = await apiClient.get<DashboardTrendPoint[] | { trend?: DashboardTrendPoint[] }>(
      '/usage/dashboard/trend',
      { params }
    )
    return normalizeTrend(data)
  },
}

export default usageAPI
