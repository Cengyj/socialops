import { describe, expect, it } from 'vitest'

import {
  removeSelectedIds,
  retainExistingSelectedIds,
  selectedRowsById,
  toggleSelectedId,
  toggleVisibleSelectedIds,
  visibleSelectionState,
} from '../selection'

describe('selection helpers', () => {
  it('retains selected ids that still exist in the current rows', () => {
    expect(retainExistingSelectedIds([1, 2, 3], [
      { id: 1 },
      { id: 3 },
    ])).toEqual([1, 3])
  })

  it('deduplicates retained ids so refreshed selection counts stay stable', () => {
    expect(retainExistingSelectedIds([1, 1, 2, 3, 2], [
      { id: 1 },
      { id: 2 },
    ])).toEqual([1, 2])
  })

  it('removes selected ids by array or set without mutating inputs', () => {
    const selected = [1, 2, 3]
    const removed = new Set([2, 4])

    expect(removeSelectedIds(selected, removed)).toEqual([1, 3])
    expect(removeSelectedIds(selected, [1, 3])).toEqual([2])
    expect(selected).toEqual([1, 2, 3])
  })

  it('deduplicates remaining ids after removals', () => {
    expect(removeSelectedIds([1, 1, 2, 3, 3], [2])).toEqual([1, 3])
  })

  it('returns a fresh selected id list when there is nothing to remove', () => {
    const selected = [1, 2]
    const next = removeSelectedIds(selected, [])

    expect(next).toEqual([1, 2])
    expect(next).not.toBe(selected)
  })

  it('returns selected rows in current row order and ignores duplicate selected ids', () => {
    const rows = [
      { id: 3, name: 'third' },
      { id: 1, name: 'first' },
      { id: 2, name: 'second' },
    ]

    expect(selectedRowsById(rows, [1, 1, 3, 99])).toEqual([
      { id: 3, name: 'third' },
      { id: 1, name: 'first' },
    ])
  })

  it('toggles a selected id without mutating the original list', () => {
    const selected = [1, 2]

    expect(toggleSelectedId(selected, 2)).toEqual([1])
    expect(toggleSelectedId(selected, 3)).toEqual([1, 2, 3])
    expect(selected).toEqual([1, 2])
  })

  it('toggles visible ids consistently for current-page selection', () => {
    const selected = [1, 2]

    expect(toggleVisibleSelectedIds(selected, [2, 3], false)).toEqual([1, 2, 3])
    expect(toggleVisibleSelectedIds(selected, [2], true)).toEqual([1])
    expect(selected).toEqual([1, 2])
  })

  it('reports current-page visible selection state', () => {
    expect(visibleSelectionState([], [])).toEqual({ allSelected: false, someSelected: false })
    expect(visibleSelectionState([1, 3], [1, 2, 3])).toEqual({ allSelected: false, someSelected: true })
    expect(visibleSelectionState([1, 2, 3, 99], [1, 2, 3])).toEqual({ allSelected: true, someSelected: false })
    expect(visibleSelectionState([99], [1, 2, 3])).toEqual({ allSelected: false, someSelected: false })
  })
})
