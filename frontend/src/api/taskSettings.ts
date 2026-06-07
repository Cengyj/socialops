import { apiClient } from './client'
import type { SocialTaskTemplateParams } from '@/types/socialTask'
export type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  SocialTaskTemplateParams,
} from '@/types/socialTask'

export type TaskTemplateType =
  | 'login'
  | 'login_check'
  | 'post'
  | 'like'
  | 'retweet'
  | 'follow'
  | 'update_profile'
  | 'update_avatar'
  | 'update_banner'

export interface TaskTemplate {
  id: string
  name: string
  type: TaskTemplateType
  params: SocialTaskTemplateParams
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface TaskTemplateInput {
  id?: string
  name: string
  type: TaskTemplateType
  params: SocialTaskTemplateParams
  is_default?: boolean
}

export interface TaskTemplateValidationResult {
  valid: boolean
  type: string
  targets: number
  contents: number
  errors: string[]
}

const BASE = '/task-settings/templates'

const unwrapData = async <T>(request: Promise<{ data: T }>): Promise<T> => {
  const { data } = await request
  return data
}

const taskSettingsAPI = {
  async listTemplates(): Promise<TaskTemplate[]> {
    const templates = await unwrapData(apiClient.get<TaskTemplate[] | null>(BASE))
    return Array.isArray(templates) ? templates : []
  },

  saveTemplate(data: TaskTemplateInput): Promise<TaskTemplate> {
    return unwrapData(apiClient.post<TaskTemplate>(BASE, data))
  },

  previewMedia(storageKey: string): Promise<Blob> {
    return unwrapData(apiClient.get<Blob>('/task-settings/media', {
      params: { storage_key: storageKey },
      responseType: 'blob',
    }))
  },

  validateTemplate(data: TaskTemplateInput): Promise<TaskTemplateValidationResult> {
    return unwrapData(apiClient.post<TaskTemplateValidationResult>(`${BASE}/validate`, data))
  },

  deleteTemplate(id: string): Promise<void> {
    return unwrapData(apiClient.delete<void>(`${BASE}/${encodeURIComponent(id)}`))
  },

  copyTemplate(id: string): Promise<TaskTemplate> {
    return unwrapData(apiClient.post<TaskTemplate>(`${BASE}/${encodeURIComponent(id)}/copy`))
  },

  setDefaultTemplate(id: string): Promise<TaskTemplate> {
    return unwrapData(apiClient.post<TaskTemplate>(`${BASE}/${encodeURIComponent(id)}/default`))
  },
}

export default taskSettingsAPI
