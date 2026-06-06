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
  total_requests?: number
  today_requests?: number
  total_actual_cost?: number
  today_actual_cost?: number
  average_duration_ms?: number
  rpm?: number
  stats_updated_at?: string
  stats_stale?: boolean
}

export interface AdminDashboardTrendPoint {
  date: string
  requests?: number
  actual_cost?: number
  cost?: number
}

export interface AdminUserUsageTrendPoint {
  date: string
  user_id: number
  email?: string
  username?: string
  requests?: number
  actual_cost?: number
}

export interface AdminUserSpendingRankingItem {
  user_id: number
  email?: string
  username?: string
  actual_cost?: number
  requests?: number
}

export interface AdminUserSpendingRankingResponse {
  ranking: AdminUserSpendingRankingItem[]
  total_actual_cost?: number
  total_requests?: number
}

export interface AdminDashboardTrendParams {
  granularity?: 'day' | 'hour'
  limit?: number
}

function normalizeTrend(
  data: AdminDashboardTrendPoint[] | { trend?: AdminDashboardTrendPoint[] } | null | undefined
): AdminDashboardTrendPoint[] {
  if (Array.isArray(data)) return data
  return data?.trend ?? []
}

function normalizeUserTrend(
  data: AdminUserUsageTrendPoint[] | { trend?: AdminUserUsageTrendPoint[] } | null | undefined
): AdminUserUsageTrendPoint[] {
  if (Array.isArray(data)) return data
  return data?.trend ?? []
}

function normalizeRanking(
  data: AdminUserSpendingRankingResponse | AdminUserSpendingRankingItem[] | null | undefined
): AdminUserSpendingRankingResponse {
  if (Array.isArray(data)) {
    return { ranking: data }
  }
  return {
    ranking: data?.ranking ?? [],
    total_actual_cost: data?.total_actual_cost ?? 0,
    total_requests: data?.total_requests ?? 0,
  }
}

const dashboardAPI = {
  async getStats(): Promise<AdminDashboardStats> {
    const { data } = await apiClient.get<AdminDashboardStats>('/admin/dashboard/stats')
    return data
  },

  async getUsageTrend(params?: AdminDashboardTrendParams): Promise<AdminDashboardTrendPoint[]> {
    const { data } = await apiClient.get<AdminDashboardTrendPoint[] | { trend?: AdminDashboardTrendPoint[] }>(
      '/admin/dashboard/trend',
      { params }
    )
    return normalizeTrend(data)
  },

  async getUserUsageTrend(params?: AdminDashboardTrendParams): Promise<AdminUserUsageTrendPoint[]> {
    const { data } = await apiClient.get<AdminUserUsageTrendPoint[] | { trend?: AdminUserUsageTrendPoint[] }>(
      '/admin/dashboard/users-trend',
      { params }
    )
    return normalizeUserTrend(data)
  },

  async getUserSpendingRanking(params?: AdminDashboardTrendParams): Promise<AdminUserSpendingRankingResponse> {
    const { data } = await apiClient.get<AdminUserSpendingRankingResponse | AdminUserSpendingRankingItem[]>(
      '/admin/dashboard/users-ranking',
      { params }
    )
    return normalizeRanking(data)
  },
}

export default dashboardAPI
