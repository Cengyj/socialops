import { describe, expect, it } from 'vitest'

import {
  TASK_SETTINGS_MAX_POST_MEDIA_ITEMS,
  TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT,
  TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
  countProfileFields,
  isTaskTemplateUsable,
  resolveTaskTemplateSaveDisabledReason,
} from '../templateReadiness'

const t = (key: string, params?: Record<string, unknown>) => ({
  'taskSettings.validation.avatarDimensions': `Avatar image must be exactly ${params?.width}x${params?.height} pixels.`,
  'taskSettings.validation.avatarRequired': 'Upload one avatar image before saving this template.',
  'taskSettings.validation.bannerRequired': 'Upload one banner image before saving this template.',
  'taskSettings.validation.mediaSourceUnsupported': 'Saved media references are not available for execution yet.',
  'taskSettings.validation.nameRequired': 'Enter a template name before saving.',
  'taskSettings.validation.postConfigurationRequired': 'Add post text or at least one media item before saving this template.',
  'taskSettings.validation.postMediaTooMany': `Post templates can contain at most ${params?.max} media items.`,
  'taskSettings.validation.postMediaTypeUnsupported': 'Only image media is supported right now.',
  'taskSettings.validation.postVideoUnavailable': 'Video post media is not supported for execution right now.',
  'taskSettings.validation.profileRequired': 'Add at least one profile field before saving this template.',
  'taskSettings.validation.targetsRequired': 'Add at least one target before saving this template.',
  'taskSettings.validation.tooManyValues': `A template parameter pool can contain at most ${params?.max} items.`,
  'taskSettings.validation.valueTooLong': `Each template parameter can contain at most ${params?.max} characters.`,
}[key] ?? key)

describe('template readiness helpers', () => {
  it('counts profile update fields after trimming blank values', () => {
    expect(countProfileFields({
      display_name: ' Northwind ',
      screen_name: '',
      description: '  ',
      location: 'Singapore',
      url: undefined,
    })).toBe(2)
    expect(countProfileFields(undefined)).toBe(0)
  })

  it('explains editor save readiness with the same user-facing validation messages', () => {
    expect(resolveTaskTemplateSaveDisabledReason({
      name: ' ',
      type: 'follow',
      targetValues: ['@northwind'],
      contentValues: [],
      postMedia: [],
    }, t)).toBe('Enter a template name before saving.')

    expect(resolveTaskTemplateSaveDisabledReason({
      name: 'Follow list',
      type: 'follow',
      targetValues: [],
      contentValues: [],
      postMedia: [],
    }, t)).toBe('Add at least one target before saving this template.')

    expect(resolveTaskTemplateSaveDisabledReason({
      name: 'Video post',
      type: 'post',
      targetValues: [],
      contentValues: ['hello video'],
      postMedia: [{ source: 'inline', url: 'data:video/mp4;base64,AAA', content_type: 'video/mp4' }],
    }, t)).toBe('Video post media is not supported for execution right now.')

    expect(resolveTaskTemplateSaveDisabledReason({
      name: 'Long post',
      type: 'post',
      targetValues: [],
      contentValues: ['x'.repeat(2049)],
      postMedia: [],
    }, t)).toBe('Each template parameter can contain at most 2048 characters.')

    expect(resolveTaskTemplateSaveDisabledReason({
      name: 'Avatar refresh',
      type: 'update_avatar',
      targetValues: [],
      contentValues: [],
      postMedia: [],
      avatar: {
        source: 'inline',
        url: 'data:image/png;base64,AAA',
        content_type: 'image/png',
        width: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
        height: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT - 1,
      },
    }, t)).toBe('Avatar image must be exactly 400x400 pixels.')
  })

  it('requires valid normalized target pools for target-based templates', () => {
    expect(isTaskTemplateUsable({
      type: 'follow',
      params: { targets: [' @northwind ', '', '@socialops'] },
    })).toBe(true)

    expect(isTaskTemplateUsable({
      type: 'like',
      params: { targets: [] },
    })).toBe(false)

    expect(isTaskTemplateUsable({
      type: 'retweet',
      params: { targets: ['x'.repeat(2049)] },
    })).toBe(false)
  })

  it('keeps post templates executable when they have content or supported image media', () => {
    expect(isTaskTemplateUsable({
      type: 'post',
      params: { contents: ['hello'] },
    })).toBe(true)

    expect(isTaskTemplateUsable({
      type: 'post',
      params: {
        media: [{
          source: 'inline',
          url: 'data:image/png;base64,AAA',
          content_type: 'image/png',
        }],
      },
    })).toBe(true)

    expect(isTaskTemplateUsable({
      type: 'post',
      params: {
        media: Array.from({ length: TASK_SETTINGS_MAX_POST_MEDIA_ITEMS + 1 }, () => ({
          source: 'inline',
          url: 'data:image/png;base64,AAA',
          content_type: 'image/png',
        })),
      },
    })).toBe(false)

    expect(isTaskTemplateUsable({
      type: 'post',
      params: {
        media: [{
          source: 'inline',
          url: 'data:video/mp4;base64,AAA',
          content_type: 'video/mp4',
        }],
      },
    })).toBe(false)
  })

  it('requires at least one profile field for profile update templates', () => {
    expect(isTaskTemplateUsable({
      type: 'update_profile',
      params: { profile: { description: 'Operator account' } },
    })).toBe(true)

    expect(isTaskTemplateUsable({
      type: 'update_profile',
      params: { profile: { description: '  ' } },
    })).toBe(false)
  })

  it('requires executable avatar media with exact dimensions', () => {
    expect(isTaskTemplateUsable({
      type: 'update_avatar',
      params: {
        avatar: {
          source: 'inline',
          url: 'data:image/png;base64,AAA',
          content_type: 'image/png',
          width: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
          height: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT,
        },
      },
    })).toBe(true)

    expect(isTaskTemplateUsable({
      type: 'update_avatar',
      params: {
        avatar: {
          source: 'inline',
          url: 'data:image/png;base64,AAA',
          content_type: 'image/png',
          width: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_WIDTH,
          height: TASK_SETTINGS_REQUIRED_AVATAR_IMAGE_HEIGHT - 1,
        },
      },
    })).toBe(false)
  })
})
