import { describe, expect, it } from 'vitest'

import {
  isActiveWorkbenchTaskLog,
  normalizeWorkbenchAccountStatus,
  normalizeKnownWorkbenchAccountStatus,
  normalizeWorkbenchTaskLogStatus,
  normalizeWorkbenchTaskStatus,
  presentationWorkbenchAccountStatus,
  totalPoolAccountStatusBadgeClass,
  workbenchAccountStatusBadgeClass,
  workbenchStatusFallbackText,
  workbenchTaskMessagePanelClass,
  workbenchTaskStatusBadgeClass,
} from '../accountWorkbenchStatus'

describe('account workbench status presentation', () => {
  it('normalizes account and task statuses before UI decisions', () => {
    expect(normalizeWorkbenchAccountStatus(' Available ')).toBe('available')
    expect(normalizeWorkbenchTaskStatus(' Running ')).toBe('running')
  })

  it('normalizes task log statuses before activity decisions', () => {
    expect(normalizeWorkbenchTaskLogStatus({ status: ' Pending ' })).toBe('pending')
    expect(normalizeWorkbenchTaskLogStatus({ status: ' SUCCESS ' })).toBe('success')
    expect(normalizeWorkbenchTaskLogStatus(null)).toBe('')

    expect(isActiveWorkbenchTaskLog({ status: ' RUNNING ' })).toBe(true)
    expect(isActiveWorkbenchTaskLog({ status: 'pending' })).toBe(true)
    expect(isActiveWorkbenchTaskLog({ status: 'success' })).toBe(false)
    expect(isActiveWorkbenchTaskLog(null)).toBe(false)
  })

  it('normalizes known account statuses with a stable fallback for pool views', () => {
    expect(normalizeKnownWorkbenchAccountStatus(' Available ')).toBe('available')
    expect(normalizeKnownWorkbenchAccountStatus(' LIMITED ')).toBe('limited')
    expect(normalizeKnownWorkbenchAccountStatus(' custom_backend_status ')).toBe('not_stored')
    expect(normalizeKnownWorkbenchAccountStatus(' ', 'pending_check')).toBe('pending_check')
  })

  it('trims unknown status fallback labels without changing empty fallbacks', () => {
    expect(workbenchStatusFallbackText(' Needs Review ')).toBe('Needs Review')
    expect(workbenchStatusFallbackText('  ')).toBe('-')
    expect(workbenchStatusFallbackText(null)).toBe('-')
  })

  it('keeps not-stored workbench accounts under the invalid filter presentation', () => {
    expect(presentationWorkbenchAccountStatus('not_stored')).toBe('invalid')
    expect(workbenchAccountStatusBadgeClass('not_stored')).toBe('badge-danger')
  })

  it('maps account status badges consistently for table and detail views', () => {
    expect(workbenchAccountStatusBadgeClass('available')).toBe('badge-success')
    expect(workbenchAccountStatusBadgeClass('limited')).toBe('badge-danger')
    expect(workbenchAccountStatusBadgeClass('pending_check')).toBe('badge-warning')
  })

  it('keeps total-pool badge tones aligned with account health semantics', () => {
    expect(totalPoolAccountStatusBadgeClass('available')).toBe('badge-success')
    expect(totalPoolAccountStatusBadgeClass('pending_check')).toBe('badge-warning')
    expect(totalPoolAccountStatusBadgeClass('limited')).toBe('badge-danger')
    expect(totalPoolAccountStatusBadgeClass('custom_backend_status')).toBe('badge-danger')
  })

  it('maps task status badges and message panel tones from the same status semantics', () => {
    expect(workbenchTaskStatusBadgeClass('stored')).toBe('badge-success')
    expect(workbenchTaskStatusBadgeClass('manual_review')).toBe('badge-danger')
    expect(workbenchTaskStatusBadgeClass('pending')).toBe('badge-warning')

    expect(workbenchTaskMessagePanelClass('stored')).toContain('border-emerald-200')
    expect(workbenchTaskMessagePanelClass('ip_unavailable')).toContain('border-red-200')
    expect(workbenchTaskMessagePanelClass('manual_review')).toContain('border-red-200')
    expect(workbenchTaskMessagePanelClass('pending')).toContain('border-amber-200')
  })
})
