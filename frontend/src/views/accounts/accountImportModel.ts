import type { ImportSocialAccountRequest } from '@/api/accountWorkbench'

export interface AccountImportPreviewMessages {
  duplicateMessage: string
  missingAccountMessage: string
  missingPasswordMessage: string
  missingCredentialMessage: string
  invalidExecutionAuthMessage: string
}

export interface AccountImportPreviewRow {
  account: ImportSocialAccountRequest
  error: string
  rowNumber?: number
}

export interface AccountImportPreview {
  validRows: AccountImportPreviewRow[]
  invalidRows: AccountImportPreviewRow[]
  duplicateCount: number
  invalidCount: number
  submittableCount: number
}

export function normalizeSocialImportPlatform(value?: string | null): string {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[-/\s]+/g, '_')
  if (['twitter', 'x', 'x_twitter', 'twitter_x'].includes(normalized)) return 'x_twitter'
  return normalized
}

export function normalizeSocialImportUsername(value?: string | null): string {
  return String(value || '').trim().toLowerCase().replace(/^@+/, '').trim()
}

export function socialAccountImportDedupKey(account: Pick<ImportSocialAccountRequest, 'platform' | 'name'>): string {
  const platform = normalizeSocialImportPlatform(account.platform || 'x_twitter')
  const username = normalizeSocialImportUsername(account.name)
  if (!platform || !username) return ''
  return `${platform}\u0000username\u0000${username}`
}

export function validateTwitterExecutionAuth(raw?: string | null): boolean {
  const text = String(raw || '').trim()
  if (!text) return false
  const parsed = parseExecutionAuthObject(text)
  if (!parsed) return false
  return typeof parsed.access_token === 'string'
    && parsed.access_token.trim() !== ''
    && typeof parsed.token_secret === 'string'
    && parsed.token_secret.trim() !== ''
}

export function buildAccountImportPreview(
  accounts: ImportSocialAccountRequest[],
  messages: AccountImportPreviewMessages,
): AccountImportPreview {
  const seen = new Set<string>()
  const validRows: AccountImportPreviewRow[] = []
  const invalidRows: AccountImportPreviewRow[] = []
  let duplicateCount = 0

  accounts.forEach((account, index) => {
    const normalized = normalizeAccountImportRequest(account)
    let error = validateAccountImportRequest(normalized, messages)
    const key = socialAccountImportDedupKey(normalized)
    if (!error && key) {
      if (seen.has(key)) {
        duplicateCount += 1
        error = messages.duplicateMessage
      } else {
        seen.add(key)
      }
    }
    const row: AccountImportPreviewRow = { account: normalized, error, rowNumber: index + 1 }
    if (error) {
      invalidRows.push(row)
    } else {
      validRows.push(row)
    }
  })

  return {
    validRows,
    invalidRows,
    duplicateCount,
    invalidCount: invalidRows.length,
    submittableCount: validRows.length,
  }
}

export function normalizeAccountImportRequest(account: ImportSocialAccountRequest): ImportSocialAccountRequest {
  const normalized: ImportSocialAccountRequest = {
    platform: normalizeSocialImportPlatform(account.platform || 'x_twitter'),
    name: trimImportValue(account.name),
  }
  setImportField(normalized, 'password', trimImportValue(account.password))
  setImportField(normalized, 'phone', trimImportValue(account.phone))
  setImportField(normalized, 'email', trimImportValue(account.email))
  setImportField(normalized, 'email_password', trimImportValue(account.email_password))
  setImportField(normalized, 'auth_cookie', trimImportValue(account.auth_cookie))
  setImportField(normalized, 'execution_auth', trimImportValue(account.execution_auth))
  setImportField(normalized, 'two_factor', trimImportValue(account.two_factor))
  setImportField(normalized, 'backup_code', trimImportValue(account.backup_code))
  setImportField(normalized, 'email_client_id', trimImportValue(account.email_client_id))
  setImportField(normalized, 'email_token', trimImportValue(account.email_token))
  setImportField(normalized, 'registration_ip', trimImportValue(account.registration_ip))
  setImportField(normalized, 'remark', trimImportValue(account.remark))
  return normalized
}

export function validateAccountImportRequest(account: ImportSocialAccountRequest, messages: AccountImportPreviewMessages): string {
  if (!trimImportValue(account.name)) return messages.missingAccountMessage
  if (!trimImportValue(account.password)) return messages.missingPasswordMessage
  const hasTwoFactor = !!trimImportValue(account.two_factor)
  const hasAuthCookie = !!trimImportValue(account.auth_cookie)
  const hasExecutionAuth = !!trimImportValue(account.execution_auth)
  const hasEmail = !!trimImportValue(account.email) && (!!trimImportValue(account.email_password) || !!trimImportValue(account.email_token))
  if (!hasTwoFactor && !hasEmail && !hasAuthCookie) return messages.missingCredentialMessage
  if (normalizeSocialImportPlatform(account.platform) === 'x_twitter' && hasExecutionAuth && !validateTwitterExecutionAuth(account.execution_auth)) {
    return messages.invalidExecutionAuthMessage
  }
  return ''
}

function trimImportValue(value?: string | null): string {
  return String(value || '').trim()
}

function setImportField<K extends keyof ImportSocialAccountRequest>(target: ImportSocialAccountRequest, key: K, value: string) {
  if (value) {
    target[key] = value as ImportSocialAccountRequest[K]
  }
}

function parseExecutionAuthObject(raw: string): Record<string, unknown> | null {
  const candidates = [raw]
  try {
    candidates.push(globalThis.atob(raw))
  } catch {
    // Not base64. JSON parsing below still handles the raw input.
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
