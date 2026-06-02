import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface RiskControlStatus {
  enabled: boolean
  status?: string
  message?: string
}

export interface RiskControlConfig {
  enabled: boolean
}

export interface RiskControlLog {
  id: number
  scope?: string
  target?: string
  status?: string
  message?: string
  created_at?: string
}

export const adminRiskControlAPI = {
  async getStatus(): Promise<RiskControlStatus> {
    const { data } = await apiClient.get<RiskControlStatus>('/admin/risk-control/status')
    return data
  },

  async getConfig(): Promise<RiskControlConfig> {
    const { data } = await apiClient.get<RiskControlConfig>('/admin/risk-control/config')
    return data
  },

  async listLogs(params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<RiskControlLog>> {
    const { data } = await apiClient.get<PaginatedResponse<RiskControlLog>>('/admin/risk-control/logs', { params })
    return data
  },
}

export default adminRiskControlAPI
