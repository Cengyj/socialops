import { ref } from 'vue'
import { describe, expect, it } from 'vitest'

import {
  TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT,
  totalAccountBatchOperationSummaryParams,
  useTotalAccountOperationResultPreview,
} from '../totalAccountOperationResults'
import type { SocialAccountBatchItemResult, SocialAccountBatchResult } from '@/api/accountWorkbench'

describe('total account operation result helpers', () => {
  it('uses one preview limit for batch and import result rows', () => {
    const result = ref<SocialAccountBatchResult | null>({
      total: 10,
      succeeded: 8,
      skipped: 1,
      failed: 1,
      errors: [],
      items: Array.from({ length: TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT + 3 }, (_, index) => ({
        id: index + 1,
        status: 'succeeded',
      })),
    })
    const preview = useTotalAccountOperationResultPreview<SocialAccountBatchResult, SocialAccountBatchItemResult>({
      result,
      items: value => value.items,
    })

    expect(preview.items.value).toHaveLength(TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT)
    expect(preview.items.value[0]?.id).toBe(1)
    expect(preview.items.value.at(-1)?.id).toBe(TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT)
    expect(preview.remainingCount.value).toBe(3)

    result.value = null
    expect(preview.items.value).toEqual([])
    expect(preview.remainingCount.value).toBe(0)
  })

  it('keeps target-user summary params attached only for assignment results', () => {
    const result = {
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
    }

    expect(totalAccountBatchOperationSummaryParams({
      result,
      targetUser: 'owner@example.test',
    })).toEqual({
      count: 1,
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      user: 'owner@example.test',
    })

    expect(totalAccountBatchOperationSummaryParams({ result })).toEqual({
      count: 1,
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
    })
  })
})
