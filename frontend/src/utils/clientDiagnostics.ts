export interface ClientDiagnosticEntry {
  context: string
  at: string
  message?: string
  status?: number
  code?: string
}

declare global {
  interface Window {
    __SOCIALOPS_CLIENT_DIAGNOSTICS__?: ClientDiagnosticEntry[]
  }
}

const MAX_DIAGNOSTICS = 80
const SENSITIVE_VALUE_PATTERN =
  /((?:authorization|cookie|token|secret|password|proxy_password)\s*[:=]\s*)([^&\s,;]+)/gi
const BEARER_PATTERN = /(Bearer\s+)[A-Za-z0-9._~+/=-]+/gi

function redactSensitiveText(value: string): string {
  return value
    .replace(BEARER_PATTERN, '$1[redacted]')
    .replace(SENSITIVE_VALUE_PATTERN, '$1[redacted]')
    .slice(0, 240)
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : null
}

function firstString(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return redactSensitiveText(value.trim())
    }
  }
  return undefined
}

function firstNumber(...values: unknown[]): number | undefined {
  for (const value of values) {
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value.trim()) {
      const parsed = Number(value)
      if (Number.isFinite(parsed)) return parsed
    }
  }
  return undefined
}

function extractDiagnostic(error: unknown): Omit<ClientDiagnosticEntry, 'context' | 'at'> {
  const err = asRecord(error)
  const response = asRecord(err?.response)
  const data = asRecord(response?.data)

  return {
    message: firstString(err?.message, data?.code, data?.reason, data?.message),
    status: firstNumber(err?.status, response?.status),
    code: firstString(err?.code, data?.code, data?.reason),
  }
}

export function recordClientDiagnostic(context: string, error?: unknown): void {
  if (typeof window === 'undefined') return

  const diagnostics = window.__SOCIALOPS_CLIENT_DIAGNOSTICS__ ?? []
  diagnostics.push({
    context,
    at: new Date().toISOString(),
    ...extractDiagnostic(error),
  })

  if (diagnostics.length > MAX_DIAGNOSTICS) {
    diagnostics.splice(0, diagnostics.length - MAX_DIAGNOSTICS)
  }

  window.__SOCIALOPS_CLIENT_DIAGNOSTICS__ = diagnostics
}
