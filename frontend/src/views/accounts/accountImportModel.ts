import type { ImportSocialAccountRequest } from '@/api/accountWorkbench'

export interface AccountImportPreviewMessages {
  duplicateMessage: string
  missingAccountMessage: string
  missingPasswordMessage: string
  missingCredentialMessage: string
}

export interface AccountImportPreviewRow {
  account: ImportSocialAccountRequest
  error: string
  rowNumber?: number
}

export type AccountImportPreviewRowStatus = 'format_valid' | 'needs_data' | 'batch_duplicate'

export interface AccountImportPreviewStatusRow extends AccountImportPreviewRow {
  valid: boolean
  status: AccountImportPreviewRowStatus
}

export interface AccountImportPreviewInputRow {
  account: ImportSocialAccountRequest
  rowNumber?: number
}

export interface AccountImportCredentialSummaryLabels {
  password: string
  twoFactor: string
  email: string
  authCookie: string
  executionAuth: string
  fallback?: string
}

export interface SocialAccountImportTextPreviewRow extends AccountImportPreviewStatusRow {
  rowNumber: number
}

export type SocialAccountImportWorkbookCell = string | number | boolean | null

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

export function buildAccountImportPreview(
  accounts: ImportSocialAccountRequest[],
  messages: AccountImportPreviewMessages,
): AccountImportPreview {
  const rows = buildAccountImportPreviewRows(
    accounts.map((account, index) => ({ account, rowNumber: index + 1 })),
    messages,
  )
  const validRows: AccountImportPreviewRow[] = rows.filter(row => row.valid)
  const invalidRows: AccountImportPreviewRow[] = rows.filter(row => !row.valid)
  const duplicateCount = rows.filter(row => row.status === 'batch_duplicate').length

  return {
    validRows,
    invalidRows,
    duplicateCount,
    invalidCount: invalidRows.length,
    submittableCount: validRows.length,
  }
}

export function buildAccountImportPreviewRows(
  rows: AccountImportPreviewInputRow[],
  messages: AccountImportPreviewMessages,
): AccountImportPreviewStatusRow[] {
  const seen = new Set<string>()
  return rows.map((row) => {
    const normalized = normalizeAccountImportRequest(row.account)
    let error = validateAccountImportRequest(normalized, messages)
    let status: AccountImportPreviewRowStatus = error ? 'needs_data' : 'format_valid'
    const key = socialAccountImportDedupKey(normalized)
    if (!error && key) {
      if (seen.has(key)) {
        error = messages.duplicateMessage
        status = 'batch_duplicate'
      } else {
        seen.add(key)
      }
    }
    return {
      account: normalized,
      error,
      rowNumber: row.rowNumber,
      valid: !error,
      status,
    }
  })
}

export function parseSocialAccountImportTextRows(
  raw: string,
  fallbackPlatform: string,
  messages: AccountImportPreviewMessages,
): SocialAccountImportTextPreviewRow[] {
  return buildAccountImportPreviewRows(raw
    .split(/\r?\n/)
    .map((line, index) => ({ line, rowNumber: index + 1 }))
    .filter(item => trimImportValue(item.line))
    .filter((item, index) => index > 0 || !looksLikeSocialAccountImportHeader(splitSocialAccountImportLine(item.line)))
    .map((item) => ({ account: parseSocialAccountImportLine(item.line, fallbackPlatform), rowNumber: item.rowNumber })), messages)
    .map((row, index) => ({
      ...row,
      rowNumber: row.rowNumber ?? index + 1,
    }))
}

export function socialAccountImportWorkbookRowsToText(rows: SocialAccountImportWorkbookCell[][]): string {
  if (rows.length === 0) return ''
  const dataRows = looksLikeSocialAccountImportHeader(rows[0].map(value => String(value || ''))) ? rows.slice(1) : rows
  return dataRows
    .map((row) => {
      if (!row.some(cell => trimImportValue(String(cell || '')))) return ''
      const fixedValues = Object.fromEntries(socialAccountImportFixedWorkbookHeaders.map((header, index) => [header, String(row[index] ?? '')]))
      return socialAccountImportOrderedHeaders.map(header => String(fixedValues[header] || '')).join('\t')
    })
    .filter(Boolean)
    .join('\n')
}

export function normalizeAccountImportRequest(account: ImportSocialAccountRequest): ImportSocialAccountRequest {
  const normalized: ImportSocialAccountRequest = {
    platform: normalizeSocialImportPlatform(account.platform || 'x_twitter'),
    name: trimImportValue(account.name),
  }
  setImportDeliveryField(normalized, 'password', account.password)
  setImportField(normalized, 'phone', trimImportValue(account.phone))
  setImportField(normalized, 'email', trimImportValue(account.email))
  setImportDeliveryField(normalized, 'email_password', account.email_password)
  setImportDeliveryField(normalized, 'auth_cookie', account.auth_cookie)
  setImportDeliveryField(normalized, 'execution_auth', account.execution_auth)
  setImportDeliveryField(normalized, 'two_factor', account.two_factor)
  setImportDeliveryField(normalized, 'backup_code', account.backup_code)
  setImportDeliveryField(normalized, 'email_client_id', account.email_client_id)
  setImportDeliveryField(normalized, 'email_token', account.email_token)
  setImportField(normalized, 'registration_ip', trimImportValue(account.registration_ip))
  setImportDeliveryField(normalized, 'remark', account.remark)
  return normalized
}

export function validateAccountImportRequest(account: ImportSocialAccountRequest, messages: AccountImportPreviewMessages): string {
  if (!trimImportValue(account.name)) return messages.missingAccountMessage
  if (!trimImportValue(account.password)) return messages.missingPasswordMessage
  const hasTwoFactor = !!trimImportValue(account.two_factor)
  const hasAuthCookie = !!trimImportValue(account.auth_cookie)
  const hasEmail = !!trimImportValue(account.email) && (!!trimImportValue(account.email_password) || !!trimImportValue(account.email_token))
  if (!hasTwoFactor && !hasEmail && !hasAuthCookie) return messages.missingCredentialMessage
  return ''
}

export function accountImportCredentialSummary(
  account: ImportSocialAccountRequest,
  labels: AccountImportCredentialSummaryLabels,
) {
  const parts = []
  if (trimImportValue(account.password)) parts.push(labels.password)
  if (trimImportValue(account.two_factor)) parts.push(labels.twoFactor)
  if (trimImportValue(account.email)) parts.push(labels.email)
  if (trimImportValue(account.auth_cookie)) parts.push(labels.authCookie)
  if (trimImportValue(account.execution_auth)) parts.push(labels.executionAuth)
  return parts.join(' · ') || labels.fallback || '-'
}

function trimImportValue(value?: string | null): string {
  return String(value || '').trim()
}

function parseSocialAccountImportLine(line: string, fallbackPlatform: string): ImportSocialAccountRequest {
  const columns = splitSocialAccountImportLine(line)
  const [
    name = '',
    password = '',
    twoFactor = '',
    backupCode = '',
    email = '',
    emailPassword = '',
    emailClientID = '',
    emailToken = '',
    registrationIP = '',
    authCookie = '',
    executionAuth = '',
    phone = '',
    remark = '',
  ] = columns
  return normalizeAccountImportRequest({
    platform: fallbackPlatform || 'x_twitter',
    name,
    password,
    phone,
    email,
    email_password: emailPassword,
    auth_cookie: authCookie,
    execution_auth: executionAuth,
    two_factor: twoFactor,
    backup_code: backupCode,
    email_client_id: emailClientID,
    email_token: emailToken,
    registration_ip: registrationIP,
    remark,
  })
}

function splitSocialAccountImportLine(line: string) {
  const delimiter = line.includes('\t') ? /\t/ : /[,，]/
  const parts = line.split(delimiter)
  if (parts.length > 1) return parts
  return line.split(/\s+/).map(part => part.trim()).filter(Boolean)
}

function looksLikeSocialAccountImportHeader(columns: string[]) {
  const normalized = columns.map(normalizeSocialAccountImportHeader)
  return normalized.includes('name') && (normalized.includes('password') || normalized.includes('two_factor') || normalized.includes('auth_cookie') || normalized.includes('execution_auth'))
}

function normalizeSocialAccountImportHeader(header: string) {
  const value = header.trim().toLowerCase().replace(/\s+/g, '_')
  const aliases: Record<string, string> = {
    account: 'name',
    username: 'name',
    user_name: 'name',
    name: 'name',
    '账号': 'name',
    '用户名': 'name',
    password: 'password',
    pass: 'password',
    '密码': 'password',
    phone: 'phone',
    mobile: 'phone',
    '手机号': 'phone',
    '手机': 'phone',
    two_factor: 'two_factor',
    twofa: 'two_factor',
    '2fa': 'two_factor',
    '二次验证': 'two_factor',
    '两步验证': 'two_factor',
    backup_code: 'backup_code',
    backup: 'backup_code',
    '备份码': 'backup_code',
    email: 'email',
    email_account: 'email',
    mail: 'email',
    '邮箱': 'email',
    '邮箱账号': 'email',
    email_password: 'email_password',
    mail_password: 'email_password',
    '邮箱密码': 'email_password',
    email_client_id: 'email_client_id',
    client_id: 'email_client_id',
    '邮箱客户端id': 'email_client_id',
    '邮箱客户端ID': 'email_client_id',
    '邮箱_client_id': 'email_client_id',
    '邮箱_client_ID': 'email_client_id',
    email_token: 'email_token',
    mail_token: 'email_token',
    token: 'email_token',
    '邮箱令牌': 'email_token',
    '邮箱_token': 'email_token',
    registration_ip: 'registration_ip',
    register_ip: 'registration_ip',
    ip: 'registration_ip',
    '注册ip': 'registration_ip',
    '注册IP': 'registration_ip',
    auth_cookie: 'auth_cookie',
    authcookie: 'auth_cookie',
    cookie: 'auth_cookie',
    cookies: 'auth_cookie',
    '登录cookie': 'auth_cookie',
    '认证cookie': 'auth_cookie',
    '授权cookie': 'auth_cookie',
    execution_auth: 'execution_auth',
    remark: 'remark',
    note: 'remark',
    '备注': 'remark',
  }
  return aliases[value] || value
}

const socialAccountImportOrderedHeaders = [
  'name',
  'password',
  'two_factor',
  'backup_code',
  'email',
  'email_password',
  'email_client_id',
  'email_token',
  'registration_ip',
  'auth_cookie',
  'execution_auth',
  'phone',
  'remark',
]

const socialAccountImportFixedWorkbookHeaders = [
  'name',
  'password',
  'two_factor',
  'phone',
  'email',
  'email_password',
  'email_client_id',
  'email_token',
]

function setImportField<K extends keyof ImportSocialAccountRequest>(target: ImportSocialAccountRequest, key: K, value: string) {
  if (value) {
    target[key] = value as ImportSocialAccountRequest[K]
  }
}

function setImportDeliveryField<K extends keyof ImportSocialAccountRequest>(target: ImportSocialAccountRequest, key: K, value?: string | null) {
  const raw = String(value ?? '')
  if (raw.trim()) {
    target[key] = raw as ImportSocialAccountRequest[K]
  }
}
