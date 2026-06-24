import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'
import type { ExecutableSocialTaskAction, ParameterSocialTaskAction, SocialProfileUpdateParams } from '@/types/socialTask'
import { isFinalUsageStatus } from '@/utils/usageRecords'
export type {
  ExecutableSocialTaskAction,
  ParameterSocialTaskAction,
  SocialProfileUpdateParams,
} from '@/types/socialTask'

export interface UsageTaskMediaRef {
  source?: string
  content_type?: string
  file_name?: string
  byte_size?: number
  width?: number
  height?: number
}

export interface UsagePostPayload {
  text?: string
  quote_post_url?: string
  media?: UsageTaskMediaRef[]
}

export interface UsageTaskPayload {
  target?: string
  post?: UsagePostPayload
  profile?: SocialProfileUpdateParams
  avatar?: UsageTaskMediaRef
  banner?: UsageTaskMediaRef
}

export interface UsageTaskTemplateParams {
  targets?: string[]
  contents?: string[]
  quote_post_url?: string
  media?: UsageTaskMediaRef[]
  profile?: SocialProfileUpdateParams
  avatar?: UsageTaskMediaRef
  banner?: UsageTaskMediaRef
}

export interface UsageTaskTemplateSnapshot {
  template_id?: string
  template_name?: string
  template_type?: ParameterSocialTaskAction
  params?: UsageTaskTemplateParams
}

export interface UsageLog {
  id: number
  user_id: number
  social_account_id: number
  platform: string
  account_name: string
  operation: ExecutableSocialTaskAction
  status: string
  quantity: number
  cost: number
  charge_status: string
  charge_source?: string | null
  target?: string | null
  content?: string | null
  payload?: UsageTaskPayload | null
  template_snapshot?: UsageTaskTemplateSnapshot | null
  result_message?: string | null
  proxy_snapshot?: string | null
  billing_request_id?: string | null
  idempotency_key?: string | null
  created_at: string
  completed_at?: string | null
}

export interface UsageStats {
  total_operations?: number
  success_count?: number
  failed_count?: number
  total_charged?: number
}

export interface PlatformDashboardStats {
  platform: string
  total_operations?: number
  total_charged?: number
  today_operations?: number
  today_charged?: number
}

export interface UserDashboardStats {
  total_operations?: number
  today_operations?: number
  total_charged?: number
  today_charged?: number
  recent_operations_per_minute?: number
  by_platform?: PlatformDashboardStats[]
}

export interface DashboardTrendPoint {
  date: string
  operations?: number
  charged?: number
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
  operation?: ExecutableSocialTaskAction
  platform?: string
  account?: string
  status?: string
}

export interface UsageTaskMediaPreviewLocator {
  scope: 'payload' | 'template'
  section: 'post' | 'avatar' | 'banner'
  index?: number
}

const unwrapData = async <T>(request: Promise<{ data: T }>): Promise<T> => {
  const { data } = await request
  return data
}

function sanitizeDashboardStats(data: UserDashboardStats | null | undefined): UserDashboardStats {
  const raw = data ?? {}
  return {
    total_operations: raw.total_operations,
    today_operations: raw.today_operations,
    total_charged: raw.total_charged,
    today_charged: raw.today_charged,
    recent_operations_per_minute: raw.recent_operations_per_minute,
    by_platform: sanitizePlatformDashboardStats(raw.by_platform),
  }
}

function sanitizePlatformDashboardStats(items?: PlatformDashboardStats[] | null): PlatformDashboardStats[] | undefined {
  const normalized = (items ?? []).map(item => ({
    platform: item.platform,
    total_operations: item.total_operations,
    total_charged: item.total_charged,
    today_operations: item.today_operations,
    today_charged: item.today_charged,
  }))
  return normalized.length > 0 ? normalized : undefined
}

function sanitizeTrend(data: DashboardTrendPoint[] | null | undefined): DashboardTrendPoint[] {
  const points = Array.isArray(data) ? data : []
  return points.map(point => ({
    date: point.date,
    operations: point.operations,
    charged: point.charged,
  }))
}

function sanitizeUsagePage(data: PaginatedResponse<UsageLog>): PaginatedResponse<UsageLog> {
  const rawItems = data.items ?? []
  return {
    ...data,
    items: rawItems
      .filter(item => isFinalUsageStatus(item.status))
      .map(sanitizeUsageLog),
  }
}

function sanitizeUsageStats(raw: UsageStats): UsageStats {
  return {
    total_operations: raw.total_operations,
    success_count: raw.success_count,
    failed_count: raw.failed_count,
    total_charged: raw.total_charged,
  }
}

function sanitizeUsageLog(raw: UsageLog): UsageLog {
  return {
    id: raw.id,
    user_id: raw.user_id,
    social_account_id: raw.social_account_id,
    platform: raw.platform,
    account_name: raw.account_name,
    operation: raw.operation,
    status: raw.status,
    quantity: raw.quantity,
    cost: raw.cost,
    charge_status: raw.charge_status,
    charge_source: raw.charge_source,
    target: raw.target,
    content: raw.content,
    payload: sanitizeUsagePayload(raw.payload),
    template_snapshot: sanitizeUsageTemplateSnapshot(raw.template_snapshot),
    result_message: raw.result_message,
    proxy_snapshot: raw.proxy_snapshot,
    billing_request_id: raw.billing_request_id,
    idempotency_key: raw.idempotency_key,
    created_at: raw.created_at,
    completed_at: raw.completed_at,
  }
}

function sanitizeUsagePayload(payload?: UsageTaskPayload | null): UsageTaskPayload | null | undefined {
  if (!payload) return payload
  return {
    target: payload.target,
    post: payload.post ? {
      text: payload.post.text,
      quote_post_url: payload.post.quote_post_url,
      media: sanitizeUsageMediaRefs(payload.post.media),
    } : undefined,
    profile: payload.profile,
    avatar: sanitizeUsageMediaRef(payload.avatar),
    banner: sanitizeUsageMediaRef(payload.banner),
  }
}

function sanitizeUsageTemplateSnapshot(snapshot?: UsageTaskTemplateSnapshot | null): UsageTaskTemplateSnapshot | null | undefined {
  if (!snapshot) return snapshot
  return {
    template_id: snapshot.template_id,
    template_name: snapshot.template_name,
    template_type: snapshot.template_type,
    params: snapshot.params ? {
      targets: snapshot.params.targets,
      contents: snapshot.params.contents,
      quote_post_url: snapshot.params.quote_post_url,
      media: sanitizeUsageMediaRefs(snapshot.params.media),
      profile: snapshot.params.profile,
      avatar: sanitizeUsageMediaRef(snapshot.params.avatar),
      banner: sanitizeUsageMediaRef(snapshot.params.banner),
    } : undefined,
  }
}

function sanitizeUsageMediaRefs(items?: UsageTaskMediaRef[] | null): UsageTaskMediaRef[] | undefined {
  const sanitized = (items ?? [])
    .map(item => sanitizeUsageMediaRef(item))
    .filter((item): item is UsageTaskMediaRef => Boolean(item))
  return sanitized.length > 0 ? sanitized : undefined
}

function sanitizeUsageMediaRef(item?: UsageTaskMediaRef | null): UsageTaskMediaRef | undefined {
  if (!item) return undefined
  return {
    source: item.source,
    content_type: item.content_type,
    file_name: item.file_name,
    byte_size: item.byte_size,
    width: item.width,
    height: item.height,
  }
}

export const usageAPI = {
  async list(params?: UsageQueryParams): Promise<PaginatedResponse<UsageLog>> {
    const { data } = await apiClient.get<PaginatedResponse<UsageLog>>('/usage', { params })
    return sanitizeUsagePage(data)
  },

  async getById(id: number): Promise<UsageLog> {
    const { data } = await apiClient.get<UsageLog>(`/usage/${id}`)
    return sanitizeUsageLog(data)
  },

  previewTaskMedia(id: number, locator: UsageTaskMediaPreviewLocator): Promise<Blob> {
    return unwrapData(apiClient.get<Blob>(`/usage/${id}/media`, {
      params: locator,
      responseType: 'blob',
    }))
  },

  async getStats(params?: Pick<UsageQueryParams, 'start_date' | 'end_date' | 'operation' | 'platform' | 'account' | 'status'>): Promise<UsageStats> {
    const { data } = await apiClient.get<UsageStats>('/usage/stats', { params })
    return sanitizeUsageStats(data)
  },

  async getDashboardStats(): Promise<UserDashboardStats> {
    const { data } = await apiClient.get<UserDashboardStats>('/usage/dashboard/stats')
    return sanitizeDashboardStats(data)
  },

  async getDashboardTrend(params?: DashboardTrendParams): Promise<DashboardTrendPoint[]> {
    const { data } = await apiClient.get<DashboardTrendPoint[]>(
      '/usage/dashboard/trend',
      { params }
    )
    return sanitizeTrend(data)
  },
}

export default usageAPI
