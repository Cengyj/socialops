import { describe, expect, it } from 'vitest'

import { createAccountWorkbenchErrorMessages } from '../accountWorkbenchErrorMessages'

describe('account workbench error messages', () => {
  it('maps backend account workbench error codes to translated safe messages', () => {
    const messages = createAccountWorkbenchErrorMessages(key => `translated:${key}`)

    expect(messages).toMatchObject({
      SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE',
      SOCIAL_ACCOUNT_INPUT_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED',
      SOCIAL_ACCOUNT_PASSWORD_REQUIRED: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED',
      SOCIAL_ACCOUNT_IMPORT_INCOMPLETE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE',
      SOCIAL_ACCOUNT_DUPLICATE: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_DUPLICATE',
      SOCIAL_ACCOUNT_NOT_FOUND: 'translated:admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND',
      SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID: 'translated:accountWorkbench.edit.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID',
      SOCIAL_IP_SERVICE_UNAVAILABLE: 'translated:proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE',
      SOCIAL_IP_NOT_AVAILABLE: 'translated:accountWorkbench.proxy.errors.SOCIAL_IP_NOT_AVAILABLE',
      GLOBAL_PROXY_NOT_AVAILABLE: 'translated:accountWorkbench.execution.errors.GLOBAL_PROXY_NOT_AVAILABLE',
      GLOBAL_PROXY_SERVICE_UNAVAILABLE: 'translated:accountWorkbench.execution.errors.GLOBAL_PROXY_SERVICE_UNAVAILABLE',
      SOCIAL_IP_NOT_FOUND: 'translated:accountWorkbench.proxy.errors.SOCIAL_IP_NOT_FOUND',
      SOCIAL_IP_POOL_EMPTY: 'translated:accountWorkbench.proxy.errors.SOCIAL_IP_POOL_EMPTY',
      SOCIAL_IP_REQUIRED: 'translated:accountWorkbench.proxy.errors.SOCIAL_IP_REQUIRED',
      SOCIAL_IP_ASSIGNMENT_MODE_INVALID: 'translated:accountWorkbench.proxy.errors.SOCIAL_IP_ASSIGNMENT_MODE_INVALID',
    })
  })
})
