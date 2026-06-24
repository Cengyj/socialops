import { ref } from 'vue'
import { describe, expect, it } from 'vitest'

import { clearOperationResultRefs, useOperationResultPreview } from '../operationResults'

describe('operation result helpers', () => {
  it('previews result items with a shared list-preview shape', () => {
    const result = ref<{ items: Array<{ id: number }> } | null>({
      items: [{ id: 1 }, { id: 2 }, { id: 3 }],
    })

    const preview = useOperationResultPreview({
      result,
      items: value => value.items,
      previewLimit: 2,
    })

    expect(preview.items.value).toEqual([{ id: 1 }, { id: 2 }])
    expect(preview.remainingCount.value).toBe(1)

    result.value = null
    expect(preview.items.value).toEqual([])
    expect(preview.remainingCount.value).toBe(0)
  })

  it('supports non-nullable array result refs used by inline operation rows', () => {
    const result = ref([{ id: 1 }, { id: 2 }, { id: 3 }])

    const preview = useOperationResultPreview({
      result,
      items: value => value,
      previewLimit: 2,
    })

    expect(preview.items.value).toEqual([{ id: 1 }, { id: 2 }])
    expect(preview.remainingCount.value).toBe(1)

    result.value = []
    expect(preview.items.value).toEqual([])
    expect(preview.remainingCount.value).toBe(0)
  })

  it('clears nullable operation result refs while preserving explicit keep keys', () => {
    const importResult = ref<{ total: number } | null>({ total: 1 })
    const batchResult = ref<{ total: number } | null>({ total: 2 })

    clearOperationResultRefs({
      importResult,
      batchResult,
    }, { keep: ['batchResult'] })

    expect(importResult.value).toBeNull()
    expect(batchResult.value).toEqual({ total: 2 })

    clearOperationResultRefs({ importResult, batchResult })

    expect(importResult.value).toBeNull()
    expect(batchResult.value).toBeNull()
  })
})
