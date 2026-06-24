export interface SelectableIDRow {
  id: number
}

export function retainExistingSelectedIds<T extends SelectableIDRow>(
  selectedIds: readonly number[],
  rows: readonly T[],
) {
  const existingIds = new Set(rows.map(row => row.id))
  return uniqueSelectedIds(selectedIds.filter(id => existingIds.has(id)))
}

export function removeSelectedIds(
  selectedIds: readonly number[],
  idsToRemove: ReadonlySet<number> | readonly number[],
) {
  const removedIds = idsToRemove instanceof Set ? idsToRemove : new Set(idsToRemove)
  if (removedIds.size === 0) return uniqueSelectedIds(selectedIds)
  return uniqueSelectedIds(selectedIds.filter(id => !removedIds.has(id)))
}

export function selectedRowsById<T extends SelectableIDRow>(
  rows: readonly T[],
  selectedIds: readonly number[],
) {
  if (rows.length === 0 || selectedIds.length === 0) return []
  const selectedIdSet = new Set(selectedIds)
  return rows.filter(row => selectedIdSet.has(row.id))
}

export function toggleSelectedId(
  selectedIds: readonly number[],
  id: number,
) {
  return selectedIds.includes(id)
    ? selectedIds.filter(selectedId => selectedId !== id)
    : [...selectedIds, id]
}

export function toggleVisibleSelectedIds(
  selectedIds: readonly number[],
  visibleIds: readonly number[],
  allVisibleSelected: boolean,
) {
  if (allVisibleSelected) return removeSelectedIds(selectedIds, visibleIds)
  return Array.from(new Set([...selectedIds, ...visibleIds]))
}

export function visibleSelectionState(
  selectedIds: readonly number[],
  visibleIds: readonly number[],
) {
  if (visibleIds.length === 0) {
    return { allSelected: false, someSelected: false }
  }

  const selectedIdSet = new Set(selectedIds)
  const selectedVisibleCount = visibleIds.filter(id => selectedIdSet.has(id)).length

  return {
    allSelected: selectedVisibleCount === visibleIds.length,
    someSelected: selectedVisibleCount > 0 && selectedVisibleCount < visibleIds.length,
  }
}

function uniqueSelectedIds(selectedIds: readonly number[]) {
  return Array.from(new Set(selectedIds))
}
