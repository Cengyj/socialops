import { PARAMETER_SOCIAL_TASK_ACTIONS } from '@/types/socialTask'
import type { SocialProfileUpdateParams, SocialTaskMediaRef, SocialTaskTemplateParams } from '@/types/socialTask'
import { socialPostMediaRefsSupported, socialTaskMediaRefExecutable } from './socialTaskMediaValidation'

export const ACCOUNT_WORKBENCH_MAX_TEMPLATE_POOL_VALUES = 500
export const ACCOUNT_WORKBENCH_MAX_TEMPLATE_VALUE_LENGTH = 2048
export const ACCOUNT_WORKBENCH_REQUIRED_AVATAR_IMAGE_WIDTH = 400
export const ACCOUNT_WORKBENCH_REQUIRED_AVATAR_IMAGE_HEIGHT = 400
export const ACCOUNT_WORKBENCH_REQUIRED_BANNER_IMAGE_WIDTH = 1500
export const ACCOUNT_WORKBENCH_REQUIRED_BANNER_IMAGE_HEIGHT = 500

export interface AccountWorkbenchTaskTemplateLike {
  type: string
  params: SocialTaskTemplateParams
}

export function actionRequiresDefaultTaskTemplate(action?: string | null) {
  const normalized = String(action || '').trim()
  return (PARAMETER_SOCIAL_TASK_ACTIONS as readonly string[]).includes(normalized)
}

export function workbenchTaskTemplateDisabled(
  template?: AccountWorkbenchTaskTemplateLike | null,
  options: { platformUnsupported?: boolean } = {},
) {
  if (!template) return true
  if (options.platformUnsupported) return true
  if (template.type === 'follow' || template.type === 'like' || template.type === 'retweet') {
    const targets = normalizedTemplatePoolValues(template.params.targets)
    return !templatePoolValuesValid(targets) || targets.length === 0
  }
  if (template.type === 'post') {
    const contents = normalizedTemplatePoolValues(template.params.contents)
    const mediaCount = countTemplateMediaRefs(template.params.media)
    return !templatePoolValuesValid(contents)
      || (contents.length === 0 && mediaCount === 0)
      || !templateMediaRefsValid(template.params.media, 4)
      || !socialPostMediaRefsSupported(template.params.media)
  }
  if (template.type === 'update_profile') {
    return countTemplateProfileFields(template.params.profile) === 0
  }
  if (template.type === 'update_avatar') {
    return !socialTaskMediaRefExecutable(template.params.avatar)
      || !templateExactImageDimensionsValid(
        template.params.avatar,
        ACCOUNT_WORKBENCH_REQUIRED_AVATAR_IMAGE_WIDTH,
        ACCOUNT_WORKBENCH_REQUIRED_AVATAR_IMAGE_HEIGHT,
      )
  }
  if (template.type === 'update_banner') {
    return !socialTaskMediaRefExecutable(template.params.banner)
      || !templateExactImageDimensionsValid(
        template.params.banner,
        ACCOUNT_WORKBENCH_REQUIRED_BANNER_IMAGE_WIDTH,
        ACCOUNT_WORKBENCH_REQUIRED_BANNER_IMAGE_HEIGHT,
      )
  }
  return true
}

export function normalizedTemplatePoolValues(values?: string[]) {
  return (values ?? []).map(value => value.trim()).filter(Boolean)
}

export function templatePoolValuesValid(values?: string[]) {
  const normalized = normalizedTemplatePoolValues(values)
  if (normalized.length > ACCOUNT_WORKBENCH_MAX_TEMPLATE_POOL_VALUES) return false
  return normalized.every(value => Array.from(value).length <= ACCOUNT_WORKBENCH_MAX_TEMPLATE_VALUE_LENGTH)
}

export function countTemplateProfileFields(profile?: SocialProfileUpdateParams) {
  if (!profile) return 0
  return [
    profile.display_name,
    profile.screen_name,
    profile.description,
    profile.location,
    profile.url,
  ].filter(value => String(value || '').trim() !== '').length
}

export function countTemplateMediaRefs(items?: SocialTaskMediaRef[]) {
  return (items ?? []).filter(item => hasTemplateMediaRef(item)).length
}

export function templateMediaRefsValid(items?: SocialTaskMediaRef[], maxCount = 4) {
  const validCount = countTemplateMediaRefs(items)
  return validCount === (items ?? []).length && validCount <= maxCount
}

export function templateExactImageDimensionsValid(item: SocialTaskMediaRef | undefined, requiredWidth: number, requiredHeight: number) {
  if (!item) return false
  return Number(item.width) === requiredWidth && Number(item.height) === requiredHeight
}

export function hasTemplateMediaRef(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  return [
    item.source,
    item.storage_key,
    item.url,
    item.content_type,
    item.file_name,
    item.sha256,
  ].some(value => String(value || '').trim() !== '')
}
