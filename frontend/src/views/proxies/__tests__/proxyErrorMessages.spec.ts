import { describe, expect, it } from 'vitest'

import { createProxyErrorMessages } from '../proxyErrorMessages'

describe('proxy error messages', () => {
  it('centralizes structured proxy backend errors for user-facing feedback', () => {
    const messages = createProxyErrorMessages(key => `translated:${key}`)

    expect(messages).toEqual({
      SOCIAL_IP_SERVICE_UNAVAILABLE: 'translated:proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE',
      SOCIAL_IP_INPUT_REQUIRED: 'translated:proxies.errors.SOCIAL_IP_INPUT_REQUIRED',
      SOCIAL_IP_NAME_REQUIRED: 'translated:proxies.errors.SOCIAL_IP_NAME_REQUIRED',
      SOCIAL_IP_TYPE_INVALID: 'translated:proxies.errors.SOCIAL_IP_TYPE_INVALID',
      INVALID_PROXY_ENDPOINT: 'translated:proxies.errors.INVALID_PROXY_ENDPOINT',
      SOCIAL_IP_NOT_FOUND: 'translated:proxies.errors.SOCIAL_IP_NOT_FOUND',
      SOCIAL_IP_OWNER_NOT_FOUND: 'translated:proxies.errors.SOCIAL_IP_OWNER_NOT_FOUND',
      SOCIAL_IP_USER_ID_NOT_ACCEPTED: 'translated:proxies.errors.SOCIAL_IP_USER_ID_NOT_ACCEPTED',
    })
  })
})
