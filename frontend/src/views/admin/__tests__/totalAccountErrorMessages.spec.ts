import { describe, expect, it } from 'vitest'
import { createTotalAccountErrorMessages } from '../totalAccountErrorMessages'

describe('createTotalAccountErrorMessages', () => {
  it('maps total-account backend reason codes to localized messages', () => {
    const messages = createTotalAccountErrorMessages(key => `translated:${key}`)

    expect(messages).toEqual({
      SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE',
      SOCIAL_ACCOUNT_INPUT_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED',
      SOCIAL_ACCOUNT_NAME_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NAME_REQUIRED',
      SOCIAL_ACCOUNT_PLATFORM_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PLATFORM_REQUIRED',
      SOCIAL_ACCOUNT_IDENTITY_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IDENTITY_REQUIRED',
      SOCIAL_ACCOUNT_PASSWORD_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED',
      SOCIAL_ACCOUNT_IMPORT_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_REQUIRED',
      SOCIAL_ACCOUNT_IMPORT_INCOMPLETE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE',
      SOCIAL_ACCOUNT_DUPLICATE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_DUPLICATE',
      SOCIAL_ACCOUNT_NOT_FOUND: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND',
      SOCIAL_ACCOUNT_ALREADY_ASSIGNED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ALREADY_ASSIGNED',
      SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED',
      SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID',
      USER_NOT_FOUND: 'translated:admin.socialAccountWorkbench.errors.USER_NOT_FOUND',
    })
  })
})
