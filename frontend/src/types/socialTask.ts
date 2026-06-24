export const DIRECT_SOCIAL_TASK_ACTIONS = ['login', 'login_check'] as const

export const PARAMETER_SOCIAL_TASK_ACTIONS = [
  'follow',
  'post',
  'like',
  'retweet',
  'update_profile',
  'update_avatar',
  'update_banner',
] as const

export const EXECUTABLE_SOCIAL_TASK_ACTIONS = [
  ...DIRECT_SOCIAL_TASK_ACTIONS,
  ...PARAMETER_SOCIAL_TASK_ACTIONS,
] as const

export type DirectSocialTaskAction = typeof DIRECT_SOCIAL_TASK_ACTIONS[number]
export type ParameterSocialTaskAction = typeof PARAMETER_SOCIAL_TASK_ACTIONS[number]
export type ExecutableSocialTaskAction = typeof EXECUTABLE_SOCIAL_TASK_ACTIONS[number]

export interface SocialTaskMediaRef {
  source?: string
  storage_key?: string
  url?: string
  content_type?: string
  file_name?: string
  sha256?: string
  byte_size?: number
  width?: number
  height?: number
}

export interface SocialProfileUpdateParams {
  display_name?: string
  screen_name?: string
  description?: string
  location?: string
  url?: string
}

export interface SocialPostPayload {
  text?: string
  quote_post_url?: string
  media?: SocialTaskMediaRef[]
}

export interface SocialTaskPayload {
  target?: string
  post?: SocialPostPayload
  profile?: SocialProfileUpdateParams
  avatar?: SocialTaskMediaRef
  banner?: SocialTaskMediaRef
}

export interface SocialTaskTemplateParams {
  targets?: string[]
  contents?: string[]
  quote_post_url?: string
  media?: SocialTaskMediaRef[]
  profile?: SocialProfileUpdateParams
  avatar?: SocialTaskMediaRef
  banner?: SocialTaskMediaRef
}

export interface SocialTaskTemplateSnapshot {
  template_id?: string
  template_name?: string
  template_type?: string
  params?: SocialTaskTemplateParams
}
