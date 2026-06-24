import { describe, expect, it } from 'vitest'

import {
  parseOAuthCallbackFragment,
  readOAuthFragmentLogin,
  readOAuthFragmentPendingToken
} from '../oauthCallbackFragment'

describe('oauth callback fragment helpers', () => {
  it('parses hash fragments and extracts token completion payloads', () => {
    const params = parseOAuthCallbackFragment(
      '#access_token=fragment-access-token&refresh_token=fragment-refresh-token&expires_in=3600&token_type=Bearer'
    )

    expect(readOAuthFragmentLogin(params)).toEqual({
      access_token: 'fragment-access-token',
      refresh_token: 'fragment-refresh-token',
      expires_in: 3600,
      token_type: 'Bearer'
    })
  })

  it('returns null for non-login fragments and extracts pending oauth tokens', () => {
    const params = parseOAuthCallbackFragment('#error=invitation_required&pending_oauth_token=fragment-pending-token')

    expect(readOAuthFragmentLogin(params)).toBeNull()
    expect(readOAuthFragmentPendingToken(params)).toBe('fragment-pending-token')
  })
})
