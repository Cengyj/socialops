import { formatSafeUserTaskResult } from '@/utils/socialTaskResult'

export type TemplateValidationTranslateFn = (
  key: string,
  params?: Record<string, string | number>,
) => string

export function normalizeTemplateValidationErrors(
  errors: readonly unknown[] | null | undefined,
  t: TemplateValidationTranslateFn,
) {
  return (errors ?? [])
    .map(error => String(error ?? '').trim())
    .filter(Boolean)
    .map(error => formatTemplateValidationError(error, t))
}

export function formatTemplateValidationError(
  error: string,
  t: TemplateValidationTranslateFn,
) {
  const normalized = error.toLowerCase()
  const valueLimit = error.match(/cannot exceed (\d+) (?:items|characters)/i)
  const dimensionLimit = error.match(/(avatar|banner) image must be (\d+)x(\d+) pixels/i)

  if (normalized === 'template name is required' || normalized === 'task template name is required') {
    return t('taskSettings.validation.nameRequired')
  }
  if (normalized === 'template is required') return t('taskSettings.validation.templateRequired')
  if (normalized === 'unsupported task template type' || normalized === 'unsupported social task action') {
    return t('taskSettings.validation.unsupportedType')
  }
  if (normalized === 'target list is required') return t('taskSettings.validation.targetsRequired')
  if (normalized === 'post template requires content pool or media') return t('taskSettings.validation.postConfigurationRequired')
  if (normalized === 'profile settings are required') return t('taskSettings.validation.profileRequired')
  if (normalized === 'avatar media is required') return t('taskSettings.validation.avatarRequired')
  if (normalized === 'banner media is required') return t('taskSettings.validation.bannerRequired')
  if (normalized.startsWith('target list cannot exceed') || normalized.startsWith('content pool cannot exceed')) {
    return t('taskSettings.validation.tooManyValues', { max: valueLimit?.[1] ?? 500 })
  }
  if (normalized.startsWith('target item cannot exceed') || normalized.startsWith('content item cannot exceed')) {
    return t('taskSettings.validation.valueTooLong', { max: valueLimit?.[1] ?? 2048 })
  }
  if (normalized.startsWith('post media cannot exceed')) {
    return t('taskSettings.validation.postMediaTooMany', { max: valueLimit?.[1] ?? 4 })
  }
  if (normalized.includes('video media is not supported')) return t('taskSettings.validation.postVideoUnavailable')
  if (normalized.includes('media source is not supported')) return t('taskSettings.validation.mediaSourceUnsupported')
  if (normalized.endsWith('media is invalid')) return t('taskSettings.validation.mediaInvalid')
  if (normalized === 'post media content type is not supported' || normalized.includes('media must be an image')) {
    return t('taskSettings.validation.postMediaTypeUnsupported')
  }
  if (dimensionLimit?.[1] === 'avatar') {
    return t('taskSettings.validation.avatarDimensions', { width: dimensionLimit[2], height: dimensionLimit[3] })
  }
  if (dimensionLimit?.[1] === 'banner') {
    return t('taskSettings.validation.bannerDimensions', { width: dimensionLimit[2], height: dimensionLimit[3] })
  }
  return formatSafeUserTaskResult(error, t('taskSettings.validation.invalid'))
}
