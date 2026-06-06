/**
 * Admin Redeem Codes API endpoints
 * Handles redeem code generation and management for administrators
 */

import { apiClient } from '../client'
import type {
  RedeemCode,
  GenerateRedeemCodesRequest,
  BatchUpdateRedeemCodeFields,
  RedeemCodeType,
  PaginatedResponse
} from '@/types'

export interface GenerateRedeemCodesInput {
  count: number
  type: RedeemCodeType
  value: number
  plan_id?: number | null
  group_id?: number | null
  validity_days?: number
  expires_in_days?: number | null
}

/**
 * List all redeem codes with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters
 * @returns Paginated list of redeem codes
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    type?: RedeemCodeType
    status?: 'active' | 'used' | 'expired' | 'unused' | 'disabled'
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<RedeemCode>> {
  const { data } = await apiClient.get<PaginatedResponse<RedeemCode>>('/admin/redeem-codes', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get redeem code by ID
 * @param id - Redeem code ID
 * @returns Redeem code details
 */
export async function getById(id: number): Promise<RedeemCode> {
  const { data } = await apiClient.get<RedeemCode>(`/admin/redeem-codes/${id}`)
  return data
}

/**
 * Generate new redeem codes
 * @param input - Code generation payload
 * @returns Array of generated redeem codes
 */
export async function generate(input: GenerateRedeemCodesInput): Promise<RedeemCode[]> {
  const payload: GenerateRedeemCodesRequest = {
    count: input.count,
    type: input.type,
    value: input.value
  }

  if (input.type === 'subscription') {
    if (input.plan_id != null) {
      payload.plan_id = input.plan_id
    }
    if (input.group_id != null) {
      payload.group_id = input.group_id
    }
    if (input.validity_days && input.validity_days > 0) {
      payload.validity_days = input.validity_days
    }
  }
  if (input.expires_in_days && input.expires_in_days > 0) {
    payload.expires_in_days = input.expires_in_days
  }

  const { data } = await apiClient.post<RedeemCode[]>('/admin/redeem-codes/generate', payload)
  return data
}

/**
 * Delete redeem code
 * @param id - Redeem code ID
 * @returns Success confirmation
 */
export async function deleteCode(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/redeem-codes/${id}`)
  return data
}

/**
 * Batch delete redeem codes
 * @param ids - Array of redeem code IDs
 * @returns Success confirmation
 */
export async function batchDelete(ids: number[]): Promise<{
  deleted: number
  message: string
}> {
  const { data } = await apiClient.post<{
    deleted: number
    message: string
  }>('/admin/redeem-codes/batch-delete', { ids })
  return data
}

/**
 * Batch update selected redeem code fields
 * @param ids - Array of redeem code IDs
 * @param fields - Field collection to update
 * @returns Updated count
 */
export async function batchUpdate(
  ids: number[],
  fields: BatchUpdateRedeemCodeFields
): Promise<{
  updated: number
  message: string
}> {
  const { data } = await apiClient.post<{
    updated: number
    message: string
  }>('/admin/redeem-codes/batch-update', { ids, fields })
  return data
}

/**
 * Expire redeem code
 * @param id - Redeem code ID
 * @returns Updated redeem code
 */
export async function expire(id: number): Promise<RedeemCode> {
  const { data } = await apiClient.post<RedeemCode>(`/admin/redeem-codes/${id}/expire`)
  return data
}

/**
 * Get redeem code statistics
 * @returns Statistics about redeem codes
 */
export async function getStats(): Promise<{
  total_codes: number
  active_codes: number
  used_codes: number
  expired_codes: number
  total_value_distributed: number
  by_type: Record<RedeemCodeType, number>
}> {
  const { data } = await apiClient.get<{
    total_codes: number
    active_codes: number
    used_codes: number
    expired_codes: number
    total_value_distributed: number
    by_type: Record<RedeemCodeType, number>
  }>('/admin/redeem-codes/stats')
  return data
}

/**
 * Export redeem codes to CSV
 * @param filters - Optional filters
 * @returns CSV data as blob
 */
export async function exportCodes(filters?: {
  type?: RedeemCodeType
  status?: 'used' | 'expired' | 'unused' | 'disabled'
  search?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}): Promise<Blob> {
  const response = await apiClient.get('/admin/redeem-codes/export', {
    params: filters,
    responseType: 'blob'
  })
  return response.data
}

export const redeemAPI = {
  list,
  getById,
  generate,
  delete: deleteCode,
  batchDelete,
  batchUpdate,
  expire,
  getStats,
  exportCodes
}

export default redeemAPI
