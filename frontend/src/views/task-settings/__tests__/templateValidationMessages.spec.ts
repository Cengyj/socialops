import { describe, expect, it } from 'vitest'

import {
  formatTemplateValidationError,
  normalizeTemplateValidationErrors,
  type TemplateValidationTranslateFn,
} from '../templateValidationMessages'

const t: TemplateValidationTranslateFn = (key, params) => ({
  'taskSettings.validation.avatarDimensions': `Avatar image must be exactly ${params?.width}x${params?.height} pixels.`,
  'taskSettings.validation.avatarRequired': 'Upload one avatar image before saving this template.',
  'taskSettings.validation.bannerDimensions': `Banner image must be exactly ${params?.width}x${params?.height} pixels.`,
  'taskSettings.validation.bannerRequired': 'Upload one banner image before saving this template.',
  'taskSettings.validation.invalid': 'Template has issues',
  'taskSettings.validation.mediaInvalid': 'This image cannot be read. Upload it again before saving.',
  'taskSettings.validation.mediaSourceUnsupported': 'Saved media references are not available for execution yet.',
  'taskSettings.validation.nameRequired': 'Enter a template name before saving.',
  'taskSettings.validation.postConfigurationRequired': 'Add post text or at least one media item before saving this template.',
  'taskSettings.validation.postMediaTooMany': `Post templates can contain at most ${params?.max} media items.`,
  'taskSettings.validation.postMediaTypeUnsupported': 'Only image media is supported right now.',
  'taskSettings.validation.postVideoUnavailable': 'Video post media is not supported for execution right now.',
  'taskSettings.validation.profileRequired': 'Add at least one profile field before saving this template.',
  'taskSettings.validation.targetsRequired': 'Add at least one target before saving this template.',
  'taskSettings.validation.templateRequired': 'Template information is missing. Refresh and try again.',
  'taskSettings.validation.tooManyValues': `A template parameter pool can contain at most ${params?.max} items.`,
  'taskSettings.validation.unsupportedType': 'This task type is not supported. Choose another task type.',
  'taskSettings.validation.valueTooLong': `Each template parameter can contain at most ${params?.max} characters.`,
}[key] ?? key)

describe('task-settings template validation messages', () => {
  it('maps known backend validation strings to existing friendly task-settings messages', () => {
    expect(formatTemplateValidationError('template name is required', t)).toBe('Enter a template name before saving.')
    expect(formatTemplateValidationError('task template name is required', t)).toBe('Enter a template name before saving.')
    expect(formatTemplateValidationError('template is required', t)).toBe('Template information is missing. Refresh and try again.')
    expect(formatTemplateValidationError('unsupported task template type', t)).toBe('This task type is not supported. Choose another task type.')
    expect(formatTemplateValidationError('unsupported social task action', t)).toBe('This task type is not supported. Choose another task type.')
    expect(formatTemplateValidationError('target list is required', t)).toBe('Add at least one target before saving this template.')
    expect(formatTemplateValidationError('post template requires content pool or media', t)).toBe('Add post text or at least one media item before saving this template.')
    expect(formatTemplateValidationError('profile settings are required', t)).toBe('Add at least one profile field before saving this template.')
    expect(formatTemplateValidationError('avatar media is required', t)).toBe('Upload one avatar image before saving this template.')
    expect(formatTemplateValidationError('banner media is required', t)).toBe('Upload one banner image before saving this template.')
  })

  it('keeps backend limits readable while preserving the reported max values', () => {
    expect(formatTemplateValidationError('target list cannot exceed 500 items', t)).toBe('A template parameter pool can contain at most 500 items.')
    expect(formatTemplateValidationError('content pool cannot exceed 400 items', t)).toBe('A template parameter pool can contain at most 400 items.')
    expect(formatTemplateValidationError('target item cannot exceed 2048 characters', t)).toBe('Each template parameter can contain at most 2048 characters.')
    expect(formatTemplateValidationError('content item cannot exceed 1024 characters', t)).toBe('Each template parameter can contain at most 1024 characters.')
    expect(formatTemplateValidationError('post media cannot exceed 4 items', t)).toBe('Post templates can contain at most 4 media items.')
  })

  it('maps media validation details without exposing backend phrasing', () => {
    expect(formatTemplateValidationError('video media is not supported for SocialOps execution', t)).toBe('Video post media is not supported for execution right now.')
    expect(formatTemplateValidationError('post media #1 media source is not supported for SocialOps execution', t)).toBe('Saved media references are not available for execution yet.')
    expect(formatTemplateValidationError('post media content type is not supported', t)).toBe('Only image media is supported right now.')
    expect(formatTemplateValidationError('banner media must be an image', t)).toBe('Only image media is supported right now.')
    expect(formatTemplateValidationError('avatar media is invalid', t)).toBe('This image cannot be read. Upload it again before saving.')
    expect(formatTemplateValidationError('banner media is invalid', t)).toBe('This image cannot be read. Upload it again before saving.')
    expect(formatTemplateValidationError('avatar image must be 400x400 pixels', t)).toBe('Avatar image must be exactly 400x400 pixels.')
    expect(formatTemplateValidationError('banner image must be 1500x500 pixels', t)).toBe('Banner image must be exactly 1500x500 pixels.')
  })

  it('normalizes validation result errors before display and preserves unmapped messages', () => {
    expect(normalizeTemplateValidationErrors(['  template name is required  ', '', null, 'Custom validation detail'], t)).toEqual([
      'Enter a template name before saving.',
      'Custom validation detail',
    ])
  })

  it('hides sensitive unmapped validation details behind the existing invalid fallback', () => {
    expect(formatTemplateValidationError('execution_auth refresh failed token=secret', t)).toBe('Template has issues')
    expect(formatTemplateValidationError('backend stack trace auth_cookie=ct0-secret', t)).toBe('Template has issues')
  })
})
