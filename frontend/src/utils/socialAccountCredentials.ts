export type SocialAccountCredentialPreviewKey = 'authCookie' | 'executionAuth'

export interface ParsedSocialAccountCredential {
  raw: string
  parsed: Record<string, unknown> | null
  validOAuth: boolean
  hasAccessToken: boolean
  hasTokenSecret: boolean
  rawCookie: boolean
}

export interface SocialAccountCredentialPreview {
  key: SocialAccountCredentialPreviewKey
  label: string
  description: string
  meta: string[]
  copyable: boolean
  copyTitle: string
}

export interface SocialAccountCredentialPreviewText {
  empty: string
  length: (count: number) => string
  encryptedStored: string
  oauthReady: string
  rawCookieDetected: string
  oauthPartial: string
  jsonDetected: string
  loginRefreshRequired: string
}

export function buildSocialAccountCredentialPreview(options: {
  key: SocialAccountCredentialPreviewKey
  raw: string
  authCookie?: string
  label: string
  description: string
  copyTitle: string
  text: SocialAccountCredentialPreviewText
}): SocialAccountCredentialPreview {
  const isExecutionAuth = options.key === 'executionAuth'
  const parsed = parseSocialAccountCredential(options.raw)
  const authCookieParsed = parseSocialAccountCredential(options.authCookie ?? '')
  const hasStoredExecutionAuth = isExecutionAuth && parsed.raw !== ''
  return {
    key: options.key,
    label: options.label,
    description: options.description,
    meta: [
      ...socialAccountCredentialMeta(parsed, options.text, {
        encryptedStored: hasStoredExecutionAuth,
      }),
      ...socialAccountCredentialSourceMeta(options.key, parsed, authCookieParsed, options.text),
    ],
    copyable: parsed.raw !== '',
    copyTitle: options.copyTitle,
  }
}

export function parseSocialAccountCredential(rawValue?: string | null): ParsedSocialAccountCredential {
  const raw = String(rawValue ?? '').trim()
  const parsed = parseCredentialObject(raw)
  const unsupportedEnvelope = hasUnsupportedCredentialEnvelope(parsed)
  const hasAccessToken = !unsupportedEnvelope && hasCredentialString(parsed, 'access_token')
  const hasTokenSecret = !unsupportedEnvelope && hasCredentialString(parsed, 'token_secret')
  return {
    raw,
    parsed,
    validOAuth: hasAccessToken && hasTokenSecret,
    hasAccessToken,
    hasTokenSecret,
    rawCookie: looksLikeRawCookie(raw),
  }
}

export function socialAccountCredentialCharacterCountText(rawValue: string | null | undefined, text: SocialAccountCredentialPreviewText) {
  const raw = String(rawValue ?? '').trim()
  if (!raw) return text.empty
  return text.length(raw.length)
}

function socialAccountCredentialSourceMeta(
  key: SocialAccountCredentialPreviewKey,
  parsed: ParsedSocialAccountCredential,
  authCookieParsed: ParsedSocialAccountCredential,
  text: SocialAccountCredentialPreviewText,
) {
  if (key !== 'executionAuth') return []
  if (parsed.raw) return []
  if (!authCookieParsed.raw) return []
  if (!parsed.validOAuth && authCookieParsed.rawCookie) {
    return [text.loginRefreshRequired]
  }
  return []
}

function socialAccountCredentialMeta(
  parsed: ParsedSocialAccountCredential,
  text: SocialAccountCredentialPreviewText,
  options: { encryptedStored?: boolean } = {},
) {
  if (!parsed.raw) {
    return [text.empty]
  }
  const meta = [socialAccountCredentialCharacterCountText(parsed.raw, text)]
  if (options.encryptedStored) {
    meta.push(text.encryptedStored)
  } else if (parsed.validOAuth) {
    meta.push(text.oauthReady)
  } else if (parsed.rawCookie) {
    meta.push(text.rawCookieDetected)
  } else if (parsed.hasAccessToken || parsed.hasTokenSecret) {
    meta.push(text.oauthPartial)
  } else if (parsed.parsed) {
    meta.push(text.jsonDetected)
  }
  return meta
}

function parseCredentialObject(raw: string): Record<string, unknown> | null {
  if (!raw) return null
  const candidates = [raw]
  try {
    if (typeof globalThis.atob === 'function') {
      candidates.push(globalThis.atob(raw))
    }
  } catch {
    // Raw JSON is still parsed below.
  }
  for (const candidate of candidates) {
    try {
      const parsed = JSON.parse(candidate)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>
      }
    } catch {
      // Keep trying candidates.
    }
  }
  return null
}

function hasCredentialString(parsed: Record<string, unknown> | null, key: string) {
  if (!parsed) return false
  return credentialStringValue(parsed, key) !== ''
}

function credentialStringValue(parsed: Record<string, unknown>, key: string) {
  const value = parsed[key]
  return typeof value === 'string' ? value.trim() : ''
}

function hasUnsupportedCredentialEnvelope(parsed: Record<string, unknown> | null) {
  if (!parsed) return false
  return typeof parsed.kind === 'string'
    || typeof parsed['encr' + 'yption'] === 'string'
}

function looksLikeRawCookie(raw: string) {
  if (!raw || raw.trim().startsWith('{')) return false
  return /(?:^|[;\s])(ct0|auth_token|twid)=/i.test(raw)
}
