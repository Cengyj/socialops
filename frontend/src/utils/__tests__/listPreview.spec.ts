import { ref } from 'vue'
import { describe, expect, it } from 'vitest'

import { createListPreview, useListPreview } from '../listPreview'

describe('list preview helpers', () => {
  it('returns visible items and hidden count for a bounded preview', () => {
    const preview = createListPreview([1, 2, 3, 4], 2)

    expect(preview.items).toEqual([1, 2])
    expect(preview.remainingCount).toBe(2)
  })

  it('handles empty or missing lists without negative hidden counts', () => {
    expect(createListPreview(undefined, 8)).toEqual({ items: [], remainingCount: 0 })
    expect(createListPreview([], 8)).toEqual({ items: [], remainingCount: 0 })
    expect(createListPreview([1], 8)).toEqual({ items: [1], remainingCount: 0 })
  })

  it('normalizes invalid limits to an empty preview', () => {
    const preview = createListPreview(['a', 'b'], -1)

    expect(preview.items).toEqual([])
    expect(preview.remainingCount).toBe(2)
  })

  it('keeps preview refs in sync with a reactive item getter', () => {
    const items = ref(['a', 'b', 'c'])
    const preview = useListPreview(() => items.value, 2)

    expect(preview.items.value).toEqual(['a', 'b'])
    expect(preview.remainingCount.value).toBe(1)

    items.value = ['x']

    expect(preview.items.value).toEqual(['x'])
    expect(preview.remainingCount.value).toBe(0)
  })
})
