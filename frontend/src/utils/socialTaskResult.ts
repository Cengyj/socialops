export function formatSafeUserTaskResult(value?: string | null, fallback = '-'): string {
  const message = String(value || '').replace(/\s+/g, ' ').trim()
  if (!message) return fallback
  const maxLength = 160
  return message.length > maxLength ? `${message.slice(0, maxLength)}...` : message
}
