import { describe, expect, it } from 'vitest'

import {
  ACCOUNT_WORKBENCH_MAX_TEMPLATE_POOL_VALUES,
  ACCOUNT_WORKBENCH_MAX_TEMPLATE_VALUE_LENGTH,
  actionRequiresDefaultTaskTemplate,
  countTemplateMediaRefs,
  countTemplateProfileFields,
  hasTemplateMediaRef,
  normalizedTemplatePoolValues,
  templatePoolValuesValid,
  workbenchTaskTemplateDisabled,
  type AccountWorkbenchTaskTemplateLike,
} from '../accountWorkbenchTaskTemplate'

function template(type: string, params: AccountWorkbenchTaskTemplateLike['params']): AccountWorkbenchTaskTemplateLike {
  return { type, params }
}

describe('account workbench task template validation', () => {
  it('keeps direct login actions independent from default templates', () => {
    expect(actionRequiresDefaultTaskTemplate('login')).toBe(false)
    expect(actionRequiresDefaultTaskTemplate('login_check')).toBe(false)
    expect(actionRequiresDefaultTaskTemplate('follow')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('post')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('like')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('retweet')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('update_profile')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('update_avatar')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('update_banner')).toBe(true)
    expect(actionRequiresDefaultTaskTemplate('')).toBe(false)
    expect(actionRequiresDefaultTaskTemplate(' quote_post ')).toBe(false)
  })

  it('normalizes pool values and rejects empty, excessive, or oversized target pools', () => {
    expect(normalizedTemplatePoolValues([' @a ', '', ' @b '])).toEqual(['@a', '@b'])
    expect(templatePoolValuesValid(['x'.repeat(ACCOUNT_WORKBENCH_MAX_TEMPLATE_VALUE_LENGTH + 1)])).toBe(false)
    expect(templatePoolValuesValid(Array.from({ length: ACCOUNT_WORKBENCH_MAX_TEMPLATE_POOL_VALUES + 1 }, (_, index) => `@target_${index}`))).toBe(false)

    expect(workbenchTaskTemplateDisabled(template('follow', { targets: ['   '] }))).toBe(true)
    expect(workbenchTaskTemplateDisabled(template('follow', { targets: [' @target '] }))).toBe(false)
  })

  it('requires post templates to have valid text or image media and keeps videos fail-closed', () => {
    expect(workbenchTaskTemplateDisabled(template('post', { contents: ['   '], media: [] }))).toBe(true)
    expect(workbenchTaskTemplateDisabled(template('post', { contents: ['hello'], media: [] }))).toBe(false)
    expect(workbenchTaskTemplateDisabled(template('post', {
      media: [{ source: 'library', storage_key: 'social-task/42/post.png', content_type: 'image/png' }],
    }))).toBe(false)
    expect(workbenchTaskTemplateDisabled(template('post', {
      media: [{ source: 'library', storage_key: 'media/post.png', content_type: 'image/png' }],
    }))).toBe(true)
    expect(workbenchTaskTemplateDisabled(template('post', {
      media: [{ source: 'library', storage_key: 'social-task/42/post.mp4', content_type: 'video/mp4' }],
    }))).toBe(true)
    expect(workbenchTaskTemplateDisabled(template('post', {
      contents: ['hello'],
      media: Array.from({ length: 5 }, (_, index) => ({
        source: 'library',
        storage_key: `social-task/42/post-${index + 1}.png`,
        content_type: 'image/png',
      })),
    }))).toBe(true)
  })

  it('requires profile fields and exact avatar or banner dimensions', () => {
    expect(countTemplateProfileFields({ display_name: 'Ops', location: '  ' })).toBe(1)
    expect(workbenchTaskTemplateDisabled(template('update_profile', { profile: { display_name: 'Ops' } }))).toBe(false)
    expect(workbenchTaskTemplateDisabled(template('update_profile', { profile: { location: '  ' } }))).toBe(true)

    expect(workbenchTaskTemplateDisabled(template('update_avatar', {
      avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 400, height: 400 },
    }))).toBe(false)
    expect(workbenchTaskTemplateDisabled(template('update_avatar', {
      avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 399, height: 400 },
    }))).toBe(true)
    expect(workbenchTaskTemplateDisabled(template('update_banner', {
      banner: { source: 'library', storage_key: 'social-task/42/banner.png', content_type: 'image/png', width: 1500, height: 500 },
    }))).toBe(false)
    expect(workbenchTaskTemplateDisabled(template('update_banner', {
      banner: { source: 'library', storage_key: 'social-task/42/banner.png', content_type: 'image/png', width: 1500, height: 499 },
    }))).toBe(true)
  })

  it('counts only populated media refs and disables templates on unsupported platforms', () => {
    expect(hasTemplateMediaRef({ file_name: 'post.png' })).toBe(true)
    expect(hasTemplateMediaRef({ file_name: '   ' })).toBe(false)
    expect(countTemplateMediaRefs([{ file_name: 'post.png' }, { file_name: '   ' }])).toBe(1)
    expect(workbenchTaskTemplateDisabled(template('follow', { targets: ['@target'] }), { platformUnsupported: true })).toBe(true)
  })
})
