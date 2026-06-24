import { describe, expect, it } from 'vitest'

import {
  collectSucceededBatchItemIds,
  formatSocialAccountBatchResultItemLabel,
  formatSocialAccountBatchResultName,
  normalizeSocialAccountBatchResultStatus,
  socialAccountBatchDeleteResultSummaryParams,
  socialAccountBatchResultFallbackText,
  socialAccountBatchResultItemMessage,
  socialAccountBatchResultReasonTranslationKey,
  socialAccountBatchResultRowToneClass,
  socialAccountBatchResultStatusLabel,
  socialAccountBatchResultSummaryParams,
  socialAccountBatchResultToastParams,
  socialAccountBatchResultToastTone,
  showSocialAccountBatchResultToast,
} from '../accountWorkbenchBatchResult'

describe('account workbench batch result helpers', () => {
  it('normalizes shared batch result statuses and fallback text', () => {
    expect(normalizeSocialAccountBatchResultStatus(' SUCCEEDED ')).toBe('succeeded')
    expect(normalizeSocialAccountBatchResultStatus(null)).toBe('')
    expect(socialAccountBatchResultFallbackText(' unknown_status ')).toBe('unknown_status')
    expect(socialAccountBatchResultFallbackText('  ')).toBe('-')
  })

  it('formats shared batch result item labels without leaking padded names', () => {
    expect(formatSocialAccountBatchResultName('  fresh_ops  ', '-')).toBe('fresh_ops')
    expect(formatSocialAccountBatchResultName('  ', 'current row')).toBe('current row')
    expect(formatSocialAccountBatchResultItemLabel({ id: 501, name: '  fresh_ops  ' }, '-')).toBe('fresh_ops')
    expect(formatSocialAccountBatchResultItemLabel({ id: 502, name: '  ' }, '-')).toBe('#502')
    expect(formatSocialAccountBatchResultItemLabel({ name: '  ' }, '#3')).toBe('#3')
  })

  it('formats shared batch result statuses with a caller-owned i18n namespace', () => {
    const t = (key: string, fallback = '-') => key === 'accounts.statuses.succeeded' ? 'Success' : fallback

    expect(socialAccountBatchResultStatusLabel(' SUCCEEDED ', 'accounts.statuses', t)).toBe('Success')
    expect(socialAccountBatchResultStatusLabel(' waiting_for_review ', 'accounts.statuses', t)).toBe('waiting_for_review')
    expect(socialAccountBatchResultStatusLabel('  ', 'accounts.statuses', t)).toBe('-')
  })

  it('builds shared batch result summary params for caller-owned translations', () => {
    expect(socialAccountBatchResultSummaryParams({
      total: 5,
      succeeded: 2,
      failed: 1,
      skipped: 2,
    })).toEqual({
      total: 5,
      succeeded: 2,
      failed: 1,
      skipped: 2,
    })
    expect(socialAccountBatchResultSummaryParams({
      succeeded: 2,
      failed: 0,
      skipped: 0,
    })).toEqual({
      total: 0,
      succeeded: 2,
      failed: 0,
      skipped: 0,
    })
  })

  it('builds shared batch toast params with count and caller-owned extras', () => {
    expect(socialAccountBatchResultToastParams({
      total: 5,
      succeeded: 2,
      failed: 1,
      skipped: 2,
    }, { user: 'ops@example.test' })).toEqual({
      count: 2,
      total: 5,
      succeeded: 2,
      failed: 1,
      skipped: 2,
      user: 'ops@example.test',
    })

    expect(socialAccountBatchResultToastParams({
      succeeded: 0,
      failed: 1,
      skipped: 0,
    })).toEqual({
      count: 0,
      total: 0,
      succeeded: 0,
      failed: 1,
      skipped: 0,
    })
  })

  it('builds delete summary params while preserving the existing removed wording', () => {
    expect(socialAccountBatchDeleteResultSummaryParams({
      total: 4,
      succeeded: 3,
      removed: 2,
      failed: 1,
      skipped: 0,
    })).toEqual({
      total: 4,
      succeeded: 3,
      removed: 2,
      failed: 1,
      skipped: 0,
    })

    expect(socialAccountBatchDeleteResultSummaryParams({
      total: 2,
      succeeded: 2,
      failed: 0,
      skipped: 0,
    })).toEqual({
      total: 2,
      succeeded: 2,
      removed: 2,
      failed: 0,
      skipped: 0,
    })
  })

  it('collects only succeeded item ids from mixed batch results', () => {
    const ids = collectSucceededBatchItemIds({
      succeeded: 2,
      failed: 1,
      skipped: 1,
      items: [
        { id: 101, status: 'succeeded' },
        { id: 202, status: 'failed' },
        { id: 303, status: 'skipped' },
        { id: 404, status: 'Succeeded' },
      ],
    }, [101, 202, 303, 404])

    expect([...ids].sort((a, b) => a - b)).toEqual([101, 404])
  })

  it('falls back to requested ids only for complete success responses without item ids', () => {
    const ids = collectSucceededBatchItemIds({
      succeeded: 2,
      failed: 0,
      skipped: 0,
      items: [],
    }, [101, 202])

    expect([...ids].sort((a, b) => a - b)).toEqual([101, 202])
  })

  it('does not guess locally changed rows for partial success responses without succeeded item ids', () => {
    const ids = collectSucceededBatchItemIds({
      succeeded: 1,
      failed: 1,
      skipped: 0,
      items: [
        { status: 'succeeded' },
        { id: 202, status: 'failed' },
      ],
    }, [101, 202])

    expect([...ids]).toEqual([])
  })

  it('normalizes shared batch result reason translation keys', () => {
    expect(socialAccountBatchResultReasonTranslationKey(' INVALID_INPUT ')).toBe('accountWorkbench.batchResultReasons.invalidInput')
    expect(socialAccountBatchResultReasonTranslationKey('not_assigned')).toBe('accountWorkbench.batchResultReasons.alreadyUnassigned')
    expect(socialAccountBatchResultReasonTranslationKey(' upload_failed ')).toBe('accountWorkbench.batchResultReasons.uploadFailed')
    expect(socialAccountBatchResultReasonTranslationKey('operation_failed')).toBe('accountWorkbench.batchResultReasons.operationFailed')
    expect(socialAccountBatchResultReasonTranslationKey('state_changed')).toBe('accountWorkbench.batchResultReasons.stateChanged')
    expect(socialAccountBatchResultReasonTranslationKey('unknown_reason')).toBeUndefined()
  })

  it('formats shared batch item messages with translated reasons before safe fallbacks', () => {
    const t = (key: string) => `translated:${key}`

    expect(socialAccountBatchResultItemMessage({ reason: ' INVALID_INPUT ', error: 'raw detail' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.invalidInput')
    expect(socialAccountBatchResultItemMessage({ reason: 'upload_failed', error: 'database timeout' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.uploadFailed')
    expect(socialAccountBatchResultItemMessage({ reason: 'unknown_reason', error: 'row failed' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.operationFailed')
    expect(socialAccountBatchResultItemMessage({ reason: ' unknown_reason ', error: '  ' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.operationFailed')
    expect(socialAccountBatchResultItemMessage({}, t)).toBe('-')
  })

  it('formats known raw batch item errors with existing translated reason messages', () => {
    const t = (key: string) => `translated:${key}`

    expect(socialAccountBatchResultItemMessage({ error: ' invalid credentials ' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.invalidCredentials')
    expect(socialAccountBatchResultItemMessage({ error: 'duplicate account in import batch' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.duplicateInBatch')
    expect(socialAccountBatchResultItemMessage({ error: 'duplicate account in total pool' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.duplicateInDatabase')
    expect(socialAccountBatchResultItemMessage({ error: 'missing platform or name' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.invalidInput')
    expect(socialAccountBatchResultItemMessage({ error: 'account could not be assigned' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.assignFailed')
    expect(socialAccountBatchResultItemMessage({ error: 'account proxy could not be assigned' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.assignFailed')
    expect(socialAccountBatchResultItemMessage({ error: 'account could not be reclaimed' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.reclaimFailed')
    expect(socialAccountBatchResultItemMessage({ error: 'account could not be deleted' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.deleteFailed')
    expect(socialAccountBatchResultItemMessage({ error: 'account could not be uploaded' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.uploadFailed')
    expect(socialAccountBatchResultItemMessage({ error: 'unexpected row failure' }, t))
      .toBe('translated:accountWorkbench.batchResultReasons.operationFailed')
  })

  it('uses one shared tone map for social account batch result rows', () => {
    expect(socialAccountBatchResultRowToneClass('Succeeded')).toContain('emerald')
    expect(socialAccountBatchResultRowToneClass('duplicate')).toContain('amber')
    expect(socialAccountBatchResultRowToneClass('skipped')).toContain('amber')
    expect(socialAccountBatchResultRowToneClass('failed')).toContain('red')
    expect(socialAccountBatchResultRowToneClass('unknown')).toContain('gray')
  })

  it('classifies batch result toast severity consistently across account pages', () => {
    expect(socialAccountBatchResultToastTone({ succeeded: 2, failed: 0, skipped: 0 })).toBe('success')
    expect(socialAccountBatchResultToastTone({ succeeded: 1, failed: 1, skipped: 0 })).toBe('warning')
    expect(socialAccountBatchResultToastTone({ succeeded: 1, failed: 0, skipped: 1 })).toBe('warning')
    expect(socialAccountBatchResultToastTone({ succeeded: 0, failed: 1, skipped: 0 })).toBe('error')
    expect(socialAccountBatchResultToastTone({ succeeded: 0, failed: 0, skipped: 1 })).toBe('error')
    expect(socialAccountBatchResultToastTone({ succeeded: 0, failed: 0, skipped: 0 })).toBe('error')
  })

  it('routes shared batch result toasts through the same severity policy', () => {
    const calls: string[] = []
    const handlers = {
      showError: (message: string) => calls.push(`error:${message}`),
      showSuccess: (message: string) => calls.push(`success:${message}`),
      showWarning: (message: string) => calls.push(`warning:${message}`),
    }

    expect(showSocialAccountBatchResultToast({
      succeeded: 2,
      failed: 0,
      skipped: 0,
      summary: 'summary ok',
      successMessage: 'saved ok',
    }, handlers)).toBe('success')
    expect(showSocialAccountBatchResultToast({
      succeeded: 1,
      failed: 1,
      skipped: 0,
      summary: 'summary partial',
      successMessage: 'saved partial',
    }, handlers)).toBe('warning')
    expect(showSocialAccountBatchResultToast({
      succeeded: 0,
      failed: 1,
      skipped: 0,
      summary: 'summary failed',
      successMessage: 'saved failed',
    }, handlers)).toBe('error')
    expect(showSocialAccountBatchResultToast({
      succeeded: 0,
      failed: 0,
      skipped: 0,
      summary: 'summary empty',
      successMessage: 'saved empty',
    }, handlers)).toBe('error')
    expect(showSocialAccountBatchResultToast({
      succeeded: 2,
      failed: 0,
      skipped: 0,
      summary: 'summary duplicate',
      successMessage: 'saved duplicate',
      preferWarning: true,
    }, handlers)).toBe('warning')

    expect(calls).toEqual([
      'success:saved ok',
      'warning:summary partial',
      'error:summary failed',
      'error:summary empty',
      'warning:summary duplicate',
    ])
  })
})
