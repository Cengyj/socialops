const unsafeTaskResultPatterns = [
  /https?:\/\//i,
  /\bauthorization\b/i,
  /\bbearer\s+[a-z0-9._-]+/i,
  /\btoken\s*[:=]/i,
  /\bsecret\b/i,
  /\bcookie\b/i,
  /\bpassword\b/i,
  /\bproxy\b/i,
  /\bstack\b/i,
  /\btrace[_-]?id\b/i,
  /\bexception\b/i,
  /\bpanic\b/i,
]

export function formatSafeUserTaskResult(value?: string | null, fallback = '-'): string {
  const message = String(value || '').replace(/\s+/g, ' ').trim()
  if (!message) return fallback
  if (unsafeTaskResultPatterns.some(pattern => pattern.test(message))) {
    return fallback
  }
  const maxLength = 160
  return message.length > maxLength ? `${message.slice(0, maxLength)}...` : message
}
