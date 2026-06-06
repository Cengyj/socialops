import { apiClient } from './client'

export type TaskTemplateType = 'login_check' | 'post' | 'like' | 'retweet' | 'follow'

export interface TaskTemplateParams {
  targets?: string[]
  contents?: string[]
}

export interface TaskTemplate {
  id: string
  name: string
  type: TaskTemplateType
  params: TaskTemplateParams
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface TaskTemplateInput {
  id?: string
  name: string
  type: TaskTemplateType
  params: TaskTemplateParams
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
