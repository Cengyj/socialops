export const WORKBENCH_ACCOUNT_STATUSES = ['pending_check', 'available', 'limited', 'invalid', 'not_stored'] as const

export type WorkbenchAccountStatus = typeof WORKBENCH_ACCOUNT_STATUSES[number]

export function normalizeWorkbenchAccountStatus(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

export function normalizeKnownWorkbenchAccountStatus(
  value?: string | null,
  fallback: WorkbenchAccountStatus = 'not_stored',
): WorkbenchAccountStatus {
  const normalized = normalizeWorkbenchAccountStatus(value)
  return WORKBENCH_ACCOUNT_STATUSES.includes(normalized as WorkbenchAccountStatus)
    ? normalized as WorkbenchAccountStatus
    : fallback
}

export function workbenchStatusFallbackText(value?: string | null) {
  return String(value || '').trim() || '-'
}

export function presentationWorkbenchAccountStatus(value?: string | null) {
  const normalized = normalizeWorkbenchAccountStatus(value)
  return normalized === 'not_stored' ? 'invalid' : normalized
}

export function workbenchAccountStatusBadgeClass(value?: string | null) {
  const normalized = presentationWorkbenchAccountStatus(value)
  if (normalized === 'available') return 'badge-success'
  if (['invalid', 'suspended', 'limited'].includes(normalized)) return 'badge-danger'
  return 'badge-warning'
}

export function totalPoolAccountStatusBadgeClass(value?: string | null) {
  const normalized = normalizeKnownWorkbenchAccountStatus(value)
  if (normalized === 'available') return 'badge-success'
  if (normalized === 'pending_check') return 'badge-warning'
  if (normalized === 'limited') return 'badge-danger'
  return 'badge-danger'
}

export function normalizeWorkbenchTaskStatus(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

export function normalizeWorkbenchTaskLogStatus(log?: { status?: string | null } | null) {
  return normalizeWorkbenchTaskStatus(log?.status)
}

export function isActiveWorkbenchTaskLog(log?: { status?: string | null } | null) {
  const normalized = normalizeWorkbenchTaskLogStatus(log)
  return normalized === 'pending' || normalized === 'running'
}

export function workbenchTaskStatusBadgeClass(value?: string | null) {
  const normalized = normalizeWorkbenchTaskStatus(value)
  if (['stored', 'success'].includes(normalized)) return 'badge-success'
  if (['register_failed', 'risk_rejected', 'ip_unavailable', 'manual_review', 'failed'].includes(normalized)) return 'badge-danger'
  return 'badge-warning'
}

export function workbenchTaskMessagePanelClass(value?: string | null) {
  const normalized = normalizeWorkbenchTaskStatus(value)
  if (['success', 'stored'].includes(normalized)) return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (['failed', 'register_failed', 'risk_rejected', 'ip_unavailable', 'manual_review'].includes(normalized)) return 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300'
  return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300'
}
