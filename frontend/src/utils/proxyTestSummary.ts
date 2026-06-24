import { normalizeProxyStatus } from './proxyStatus'

export interface ProxyTestResultLike {
  status?: string | null
}

export interface ProxyTestResultSummary {
  total: number
  online: number
  offline: number
  unknown: number
}

export function proxyTestResultSummary(results: readonly ProxyTestResultLike[]): ProxyTestResultSummary {
  return results.reduce<ProxyTestResultSummary>((summary, result) => {
    summary.total += 1
    summary[normalizeProxyStatus(result.status)] += 1
    return summary
  }, {
    total: 0,
    online: 0,
    offline: 0,
    unknown: 0,
  })
}
