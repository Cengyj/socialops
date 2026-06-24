import { describe, expect, it } from 'vitest'
import {
  ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT,
  useAccountOperationResults,
} from '../useAccountOperationResults'

describe('useAccountOperationResults', () => {
  function createResults(options?: { previewLimit?: number }) {
    return useAccountOperationResults({
      previewLimit: options?.previewLimit,
      batchImportResultSummaryText: result => result ? `imported ${result.imported}` : '',
      storeWorkbenchResultSummaryText: result => result ? `store ${result.succeeded}/${result.failed}` : '',
      proxyAssignmentResultSummaryText: result => result ? `proxy ${result.succeeded}/${result.failed}` : '',
    })
  }

  it('previews results and clears them with keep semantics', () => {
    const results = createResults({ previewLimit: 2 })

    results.batchImportResult.value = {
      total: 3,
      succeeded: 2,
      imported: 2,
      skipped: 1,
      failed: 0,
      duplicates: 1,
      errors: [],
      items: [
        { name: 'one', status: 'succeeded' },
        { name: 'two', status: 'succeeded' },
        { name: 'three', status: 'skipped' },
      ],
      accounts: [],
    }
    results.storeWorkbenchResult.value = {
      total: 3,
      succeeded: 2,
      skipped: 0,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 102, name: 'x-main-102', status: 'succeeded' },
        { id: 103, name: 'x-main-103', status: 'failed' },
      ],
    }
    results.proxyAssignmentResult.value = {
      total: 3,
      succeeded: 1,
      skipped: 1,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 102, name: 'x-main-102', status: 'skipped' },
        { id: 103, name: 'x-main-103', status: 'failed' },
      ],
    }

    expect(results.batchImportResultPreviewItems.value).toHaveLength(2)
    expect(results.remainingBatchImportResultItemCount.value).toBe(1)
    expect(results.batchImportResultSummary.value).toBe('imported 2')
    expect(results.storeWorkbenchResultPreviewItems.value).toHaveLength(2)
    expect(results.remainingStoreWorkbenchResultItemCount.value).toBe(1)
    expect(results.storeWorkbenchResultSummary.value).toBe('store 2/1')
    expect(results.proxyAssignmentResultPreviewItems.value).toHaveLength(2)
    expect(results.remainingProxyAssignmentResultItemCount.value).toBe(1)
    expect(results.proxyAssignmentResultSummary.value).toBe('proxy 1/1')

    results.clearAccountOperationResults({ keep: ['proxyAssignment'] })

    expect(results.batchImportResult.value).toBeNull()
    expect(results.storeWorkbenchResult.value).toBeNull()
    expect(results.proxyAssignmentResult.value).not.toBeNull()

    results.clearAccountOperationResults()

    expect(results.proxyAssignmentResult.value).toBeNull()
  })

  it('uses the shared default preview limit for operation result panels', () => {
    const results = createResults()
    const items = Array.from({ length: ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT + 2 }, (_, index) => ({
      id: index + 1,
      name: `account-${index + 1}`,
      status: index % 2 === 0 ? 'succeeded' : 'failed',
    }))

    results.proxyAssignmentResult.value = {
      total: items.length,
      succeeded: ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT,
      skipped: 0,
      failed: 2,
      items,
    }

    expect(results.proxyAssignmentResultPreviewItems.value).toHaveLength(ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT)
    expect(results.remainingProxyAssignmentResultItemCount.value).toBe(2)
    expect(results.proxyAssignmentResultPreviewItems.value.at(-1)?.name)
      .toBe(`account-${ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT}`)
  })

  it('returns empty previews and summaries before any operation result is set', () => {
    const results = createResults()

    expect(results.batchImportResultPreviewItems.value).toEqual([])
    expect(results.remainingBatchImportResultItemCount.value).toBe(0)
    expect(results.batchImportResultSummary.value).toBe('')
    expect(results.storeWorkbenchResultPreviewItems.value).toEqual([])
    expect(results.remainingStoreWorkbenchResultItemCount.value).toBe(0)
    expect(results.storeWorkbenchResultSummary.value).toBe('')
    expect(results.proxyAssignmentResultPreviewItems.value).toEqual([])
    expect(results.remainingProxyAssignmentResultItemCount.value).toBe(0)
    expect(results.proxyAssignmentResultSummary.value).toBe('')
  })
})
