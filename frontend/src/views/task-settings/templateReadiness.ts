import type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  SocialTaskTemplateParams,
  TaskTemplateType,
} from '@/api/taskSettings'
import { socialPostMediaRefsSupported, socialTaskMediaRefExecutable, unsupportedSocialPostMediaKind } from '@/utils/socialTaskMediaValidation'
import { hasMediaRef, mediaDimensionsEqual, normalizeMediaRefs } from './taskMedia'
import {
  MAX_TEMPLATE_POOL_VALUES,
  MAX_TEMPLATE_VALUE_LENGTH,
  normalizeTemplatePoolValues,
  templatePoolValuesValid,
} from './templatePool'

export type ParameterTaskTemplateType = Extract<
  TaskTemplateType,
  'follow' | 'like' | 'retweet' | 'post' | 'update_profile' | 'update_avatar' | 'update_banner'
>

export interface TaskTemplateReadinessInput {
  type: TaskTemplateType | string
  params: SocialTaskTemplateParams
}

export interface TaskTemplateSaveReadinessInput {
  name: string
  type: ParameterTaskTemplateType
  targetValues: readonly string[]
  contentValues: readonly string[]
  postMedia: SocialTaskMediaRef[]
  profile?: SocialProfileUpdateParams
  avatar?: SocialTaskMediaRef
  banner?: SocialTaskMediaRef
}

export type TaskTemplateReadinessTranslateFn = (key: string, params?: Record<string, unknown>) => string

export const TASK_SETTINGS_MAX_POST_MEDIA_ITEMS = 4
export const TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH = 400
export const TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT = 400
export const TASK_SETTINGS_REQUIRED_BANNER_IMAGE_WIDTH = 1500
export const TASK_SETTINGS_REQUIRED_BANNER_IMAGE_HEIGHT = 500
export const TASK_SETTINGS_PARAMETER_TASK_TYPES: ParameterTaskTemplateType[] = [
  'follow',
  'like',
  'retweet',
  'post',
  'update_profile',
  'update_avatar',
  'update_banner',
]
export const TASK_SETTINGS_TARGET_TYPES: ReadonlySet<ParameterTaskTemplateType> = new Set(['follow', 'like', 'retweet'])

export function countProfileFields(profile?: SocialProfileUpdateParams) {
  if (!profile) return 0
  return [
    profile.display_name,
    profile.screen_name,
    profile.description,
    profile.location,
    profile.url,
  ].filter(value => String(value || '').trim() !== '').length
}

export function resolveTaskTemplateSaveDisabledReason(
  input: TaskTemplateSaveReadinessInput,
  t: TaskTemplateReadinessTranslateFn,
) {
  if (input.name.trim() === '') return t('taskSettings.validation.nameRequired')
  if (TASK_SETTINGS_TARGET_TYPES.has(input.type) && input.targetValues.length === 0) {
    return t('taskSettings.validation.targetsRequired')
  }
  if (input.type === 'post') {
    const media = normalizeMediaRefs(input.postMedia, 'post-image')
    if (input.contentValues.length === 0 && media.length === 0) {
      return t('taskSettings.validation.postConfigurationRequired')
    }
    if (media.length > TASK_SETTINGS_MAX_POST_MEDIA_ITEMS) {
      return t('taskSettings.validation.postMediaTooMany', { max: TASK_SETTINGS_MAX_POST_MEDIA_ITEMS })
    }
    const unsupportedMediaKind = unsupportedSocialPostMediaKind(media)
    if (unsupportedMediaKind === 'video') return t('taskSettings.validation.postVideoUnavailable')
    if (unsupportedMediaKind === 'source') return t('taskSettings.validation.mediaSourceUnsupported')
    if (unsupportedMediaKind === 'type') return t('taskSettings.validation.postMediaTypeUnsupported')
  }
  if (input.type === 'update_profile' && countProfileFields(input.profile) === 0) {
    return t('taskSettings.validation.profileRequired')
  }
  if (input.type === 'update_avatar' && !hasMediaRef(input.avatar)) {
    return t('taskSettings.validation.avatarRequired')
  }
  if (input.type === 'update_banner' && !hasMediaRef(input.banner)) {
    return t('taskSettings.validation.bannerRequired')
  }
  if (input.type === 'update_avatar' && !socialTaskMediaRefExecutable(input.avatar)) {
    return t('taskSettings.validation.mediaSourceUnsupported')
  }
  if (input.type === 'update_banner' && !socialTaskMediaRefExecutable(input.banner)) {
    return t('taskSettings.validation.mediaSourceUnsupported')
  }
  if (input.type === 'update_avatar' && !mediaDimensionsEqual(
    input.avatar,
    TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
    TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT,
  )) {
    return t('taskSettings.validation.avatarDimensions', {
      width: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
      height: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT,
    })
  }
  if (input.type === 'update_banner' && !mediaDimensionsEqual(
    input.banner,
    TASK_SETTINGS_REQUIRED_BANNER_IMAGE_WIDTH,
    TASK_SETTINGS_REQUIRED_BANNER_IMAGE_HEIGHT,
  )) {
    return t('taskSettings.validation.bannerDimensions', {
      width: TASK_SETTINGS_REQUIRED_BANNER_IMAGE_WIDTH,
      height: TASK_SETTINGS_REQUIRED_BANNER_IMAGE_HEIGHT,
    })
  }

  const poolValues = TASK_SETTINGS_TARGET_TYPES.has(input.type)
    ? input.targetValues
    : input.type === 'post'
      ? input.contentValues
      : []
  if (poolValues.length > MAX_TEMPLATE_POOL_VALUES) {
    return t('taskSettings.validation.tooManyValues', { max: MAX_TEMPLATE_POOL_VALUES })
  }
  if (poolValues.some(value => Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH)) {
    return t('taskSettings.validation.valueTooLong', { max: MAX_TEMPLATE_VALUE_LENGTH })
  }
  return ''
}

export function isTaskTemplateUsable(template: TaskTemplateReadinessInput) {
  if (TASK_SETTINGS_TARGET_TYPES.has(template.type as ParameterTaskTemplateType)) {
    const values = normalizeTemplatePoolValues(template.params.targets)
    return values.length > 0 && templatePoolValuesValid(values)
  }
  if (template.type === 'post') {
    const values = normalizeTemplatePoolValues(template.params.contents)
    const media = normalizeMediaRefs(template.params.media ?? [], 'post-image')
    return (values.length > 0 || media.length > 0)
      && media.length <= TASK_SETTINGS_MAX_POST_MEDIA_ITEMS
      && templatePoolValuesValid(values)
      && socialPostMediaRefsSupported(media)
  }
  if (template.type === 'update_profile') {
    return countProfileFields(template.params.profile) > 0
  }
  if (template.type === 'update_avatar') {
    return socialTaskMediaRefExecutable(template.params.avatar)
      && mediaDimensionsEqual(
        template.params.avatar,
        TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
        TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT,
      )
  }
  if (template.type === 'update_banner') {
    return socialTaskMediaRefExecutable(template.params.banner)
      && mediaDimensionsEqual(
        template.params.banner,
        TASK_SETTINGS_REQUIRED_BANNER_IMAGE_WIDTH,
        TASK_SETTINGS_REQUIRED_BANNER_IMAGE_HEIGHT,
      )
  }
  return false
}
