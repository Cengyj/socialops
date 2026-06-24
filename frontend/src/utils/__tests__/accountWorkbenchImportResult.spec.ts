import { describe, expect, it } from 'vitest'

import {
  accountWorkbenchImportResultItemMessage,
  accountWorkbenchImportResultReasonTranslationKey,
  accountWorkbenchImportResultStatusLabel,
  accountWorkbenchImportResultSummaryParams,
} from '../accountWorkbenchImportResult'

describe('account workbench import result helpers', () => {
  it('normalizes import result statuses with existing labels', () => {
    const t = (key: string) => `translated:${key}`

    expect(accountWorkbenchImportResultStatusLabel(' SUCCEEDED ', t)).toBe('translated:common.success')
    expect(accountWorkbenchImportResultStatusLabel('duplicate', t)).toBe('translated:accountWorkbench.import.status.duplicate')
    expect(accountWorkbenchImportResultStatusLabel('skipped', t)).toBe('translated:accountWorkbench.import.status.skipped')
    expect(accountWorkbenchImportResultStatusLabel('failed', t)).toBe('translated:common.error')
    expect(accountWorkbenchImportResultStatusLabel(' queued ', t)).toBe('queued')
  })

  it('normalizes import result reason translation keys', () => {
    expect(accountWorkbenchImportResultReasonTranslationKey(' MATCHED_TOTAL_POOL '))
      .toBe('accountWorkbench.import.resultReasons.matchedTotalPool')
    expect(accountWorkbenchImportResultReasonTranslationKey('staged_not_stored'))
      .toBe('accountWorkbench.import.resultReasons.stagedNotStored')
    expect(accountWorkbenchImportResultReasonTranslationKey('ambiguous_total_pool_match'))
      .toBe('accountWorkbench.import.resultReasons.ambiguousTotalPoolMatch')
    expect(accountWorkbenchImportResultReasonTranslationKey('unknown_reason')).toBeUndefined()
  })

  it('builds import summary params without merging user and admin copy keys', () => {
    expect(accountWorkbenchImportResultSummaryParams({
      total: 4,
      succeeded: 2,
      imported: 2,
      skipped: 1,
      failed: 1,
      duplicates: 1,
    })).toEqual({
      total: 4,
      succeeded: 2,
      imported: 2,
      created: 2,
      skipped: 1,
      failed: 1,
      duplicates: 1,
    })
    expect(accountWorkbenchImportResultSummaryParams({
      total: 3,
      succeeded: 1,
      created: 1,
      skipped: 1,
      failed: 1,
      duplicates: 0,
    })).toEqual({
      total: 3,
      succeeded: 1,
      imported: 1,
      created: 1,
      skipped: 1,
      failed: 1,
      duplicates: 0,
    })
  })

  it('formats import result messages with translated reasons before safe fallbacks', () => {
    const t = (key: string) => `translated:${key}`

    expect(accountWorkbenchImportResultItemMessage({ reason: ' duplicate_in_database ', error: 'raw detail' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.duplicateInDatabase')
    expect(accountWorkbenchImportResultItemMessage({ reason: 'unknown_reason', error: 'row failed' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.importFailed')
    expect(accountWorkbenchImportResultItemMessage({ reason: ' unknown_reason ', error: '  ' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.importFailed')
    expect(accountWorkbenchImportResultItemMessage({}, t)).toBe('-')
  })

  it('formats known raw import result errors with existing translated reason messages', () => {
    const t = (key: string) => `translated:${key}`

    expect(accountWorkbenchImportResultItemMessage({ error: ' account already exists in your workbench ' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.alreadyInWorkbench')
    expect(accountWorkbenchImportResultItemMessage({ error: 'account already exists in the total account pool' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.duplicateInDatabase')
    expect(accountWorkbenchImportResultItemMessage({ error: 'multiple total-pool accounts match this username' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.ambiguousTotalPoolMatch')
    expect(accountWorkbenchImportResultItemMessage({ error: 'account could not be imported' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.importFailed')
    expect(accountWorkbenchImportResultItemMessage({ error: 'unexpected import failure' }, t))
      .toBe('translated:accountWorkbench.import.resultReasons.importFailed')
  })
})
