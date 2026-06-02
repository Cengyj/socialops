import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { UsageLog, UsageQueryParams, UsageStats } from '../usage'

export interface UsageCleanupTask {
  id: number
  status: string
  start_time?: string | null
  end_time?: string | null
  deleted_rows?: number
  created_at?: string
}

export interface CreateUsageCleanupTaskRequest {
  start_time?: string
  end_time?: string
  user_id?: number
  api_key_id?: number
  operation?: string
  status?: string
}

export const adminUsageAPI = {
  async list(params?: UsageQueryParams & { user_id?: number; api_key_id?: number }): Promise<PaginatedResponse<UsageLog>> {
    const { data } = await apiClient.get<PaginatedResponse<UsageLog>>('/admin/usage', { params })
    return data
  },

  async getStats(params?: UsageQueryParams & { user_id?: number; api_key_id?: number }): Promise<UsageStats> {
    const { data } = await apiClient.get<UsageStats>('/admin/usage/stats', { params })
    return data
  },

  async searchUsers(query: string): Promise<Array<{ id: number; email: string; username: string }>> {
    const { data } = await apiClient.get<Array<{ id: number; email: string; username: string }>>('/admin/usage/search-users', { params: { q: query } })
    return data
  },

  async searchAPIKeys(query: string): Promise<Array<{ id: number; name: string; user_id: number }>> {
    const { data } = await apiClient.get<Array<{ id: number; name: string; user_id: number }>>('/admin/usage/search-api-keys', { params: { q: query } })
    return data
  },

  async listCleanupTasks(params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<UsageCleanupTask>> {
    const { data } = await apiClient.get<PaginatedResponse<UsageCleanupTask>>('/admin/usage/cleanup-tasks', { params })
    return data
  },

  async createCleanupTask(data: CreateUsageCleanupTaskRequest): Promise<{ created: boolean; message?: string }> {
    const response = await apiClient.post<{ created: boolean; message?: string }>('/admin/usage/cleanup-tasks', data)
    return response.data
  },

  async cancelCleanupTask(id: number): Promise<{ canceled: boolean }> {
    const { data } = await apiClient.post<{ canceled: boolean }>(`/admin/usage/cleanup-tasks/${id}/cancel`)
    return data
  },
}

export default adminUsageAPI
