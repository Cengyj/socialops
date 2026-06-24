import type { OAuthTokenResponse } from '@/api/auth'

export function parseOAuthCallbackFragment(rawHash?: string): URLSearchParams {
  const raw = rawHash ?? (typeof window !== 'undefined' ? window.location.hash : '')
  const hash = raw.startsWith('#') ? raw.slice(1) : raw
  return new URLSearchParams(hash)
}

export function readOAuthFragmentLogin(params: URLSearchParams): OAuthTokenResponse | null {
  const accessToken = params.get('access_token')?.trim() || ''
  if (!accessToken) {
    return null
  }

  const completion: OAuthTokenResponse = {
    access_token: accessToken
  }
  const refreshToken = params.get('refresh_token')?.trim() || ''
  if (refreshToken) {
    completion.refresh_token = refreshToken
  }
  const expiresIn = Number.parseInt(params.get('expires_in')?.trim() || '', 10)
  if (Number.isFinite(expiresIn) && expiresIn > 0) {
    completion.expires_in = expiresIn
  }
  const tokenType = params.get('token_type')?.trim() || ''
  if (tokenType) {
    completion.token_type = tokenType
  }
  return completion
}

export function readOAuthFragmentPendingToken(params: URLSearchParams): string {
  return params.get('pending_oauth_token')?.trim() || ''
}
