import { apiClient } from '../client'
import type {
  AdminGroup,
  CreateGroupRequest,
  PaginatedResponse,
  UpdateGroupRequest
} from '@/types'

export interface GroupListFilters {
  platform?: string
  status?: 'active' | 'inactive'
  subscription_type?: 'standard' | 'subscription'
  is_exclusive?: boolean
  search?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(
  page = 1,
  pageSize = 20,
  filters?: GroupListFilters,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<AdminGroup>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminGroup>>('/admin/groups', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

export async function getAll(filters?: GroupListFilters): Promise<AdminGroup[]> {
  const response = await list(1, 1000, {
    sort_by: 'sort_order',
    sort_order: 'asc',
    ...filters
  })
  return response.items
}

export async function getById(id: number): Promise<AdminGroup> {
  const { data } = await apiClient.get<AdminGroup>(`/admin/groups/${id}`)
  return data
}

export async function create(payload: CreateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.post<AdminGroup>('/admin/groups', payload)
  return data
}

export async function update(id: number, payload: UpdateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.put<AdminGroup>(`/admin/groups/${id}`, payload)
  return data
}

export async function deleteGroup(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}`)
  return data
}

const groupsAPI = {
  list,
  getAll,
  getById,
  create,
  update,
  delete: deleteGroup
}

export default groupsAPI
