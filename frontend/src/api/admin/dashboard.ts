/**
 * Admin dashboard API endpoints.
 * Values are interpreted as SocialOps platform activity and task usage.
 */

import { apiClient } from '../client'

export interface AdminDashboardStats {
  total_users?: number
  today_new_users?: number
  active_users?: number
  today_requests?: number
}

const dashboardAPI = {
  async getStats(): Promise<AdminDashboardStats> {
    const { data } = await apiClient.get<AdminDashboardStats>('/admin/dashboard/stats')
    return data
  },
}

export default dashboardAPI
