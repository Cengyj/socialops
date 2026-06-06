interface APIErrorLike {
  code?: number | string
  reason?: string
  message?: string
  response?: {
    data?: {
      code?: number | string
      error?: string
      detail?: string
      message?: string
      reason?: string
    }
  }
}

function extractErrorMessage(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  return err.response?.data?.detail || err.response?.data?.message || err.message || ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
  }
): string {
  const { fallback } = options
  const message = extractErrorMessage(error)
  return message || fallback
}

function extractErrorCode(error: unknown): string | undefined {
  const err = (error || {}) as APIErrorLike
  const code = err.reason ?? err.code ?? err.response?.data?.reason ?? err.response?.data?.error ?? err.response?.data?.code
  return code == null ? undefined : String(code)
}

export function buildSafeAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
    messages?: Record<string, string>
  }
): string {
  const code = extractErrorCode(error)
  if (code && options.messages?.[code]) {
    return options.messages[code]
  }
  return options.fallback
}
