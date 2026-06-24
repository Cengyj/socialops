export type NormalizedProxyStatus = 'online' | 'offline' | 'unknown'

export function normalizeProxyStatus(value: unknown): NormalizedProxyStatus {
  const normalized = String(value ?? '').trim().toLowerCase()
  if (normalized === 'online' || normalized === 'offline' || normalized === 'unknown') return normalized
  return 'unknown'
}
