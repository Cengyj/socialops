import { computed, type ComputedRef } from 'vue'

export interface ListPreview<T> {
  items: T[]
  remainingCount: number
}

export interface ListPreviewRefs<T> {
  items: ComputedRef<T[]>
  remainingCount: ComputedRef<number>
}

export function createListPreview<T>(items: readonly T[] | null | undefined, limit: number): ListPreview<T> {
  const safeItems = items ?? []
  const safeLimit = Math.max(0, Math.floor(limit))
  const previewItems = safeItems.slice(0, safeLimit)
  return {
    items: previewItems,
    remainingCount: Math.max(0, safeItems.length - previewItems.length),
  }
}

export function useListPreview<T>(
  items: () => readonly T[] | null | undefined,
  limit: number,
): ListPreviewRefs<T> {
  const preview = computed(() => createListPreview(items(), limit))

  return {
    items: computed(() => preview.value.items),
    remainingCount: computed(() => preview.value.remainingCount),
  }
}
