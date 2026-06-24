import { apiClient } from '../client'

export interface AdminDashboardStats {
  total_users?: number
  today_new_users?: number
  active_users?: number
  hourly_active_users?: number
  total_accounts?: number
  normal_accounts?: number
  error_accounts?: number
  ratelimit_accounts?: number
  overload_accounts?: number
  total_operations?: number
  today_operations?: number
  total_charged?: number
  today_charged?: number
  average_duration_ms?: number
  recent_operations_per_minute?: number
  stats_updated_at?: string
  stats_stale?: boolean
}

export interface AdminDashboardTrendPoint {
  date: string
  operations?: number
  charged?: number
}

export interface AdminUserUsageTrendPoint {
  date: string
  user_id: number
  email?: string
  username?: string
  operations?: number
  charged?: number
}

export interface AdminUserSpendingRankingItem {
  user_id: number
  email?: string
  charged?: number
  operations?: number
}

export interface AdminUserSpendingRankingResponse {
  ranking: AdminUserSpendingRankingItem[]
  total_charged?: number
  total_operations?: number
}

export interface AdminDashboardTrendParams {
  granularity?: 'day' | 'hour'
  limit?: number
}

const dashboardAPI = {
  async getStats(): Promise<AdminDashboardStats> {
    const { data } = await apiClient.get<AdminDashboardStats>('/admin/dashboard/stats')
    return data
  },

  async getUsageTrend(params?: AdminDashboardTrendParams): Promise<AdminDashboardTrendPoint[]> {
    const { data } = await apiClient.get<AdminDashboardTrendPoint[]>(
      '/admin/dashboard/trend',
      { params }
    )
    return data
  },

  async getUserUsageTrend(params?: AdminDashboardTrendParams): Promise<AdminUserUsageTrendPoint[]> {
    const { data } = await apiClient.get<AdminUserUsageTrendPoint[]>(
      '/admin/dashboard/users-trend',
      { params }
    )
    return data
  },

  async getUserSpendingRanking(params?: AdminDashboardTrendParams): Promise<AdminUserSpendingRankingResponse> {
    const { data } = await apiClient.get<AdminUserSpendingRankingResponse>(
      '/admin/dashboard/users-ranking',
      { params }
    )
    return data
  },
}

export default dashboardAPI
