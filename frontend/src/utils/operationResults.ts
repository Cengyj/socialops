import { useListPreview } from './listPreview'

type OperationResultSource<TResult> = {
  readonly value: TResult | null | undefined
}

export interface OperationResultPreviewOptions<TResult, TItem> {
  result: OperationResultSource<TResult>
  items: (result: TResult) => readonly TItem[] | null | undefined
  previewLimit: number
}

export function useOperationResultPreview<TResult, TItem>(options: OperationResultPreviewOptions<TResult, TItem>) {
  return useListPreview(
    () => options.result.value ? options.items(options.result.value) : undefined,
    options.previewLimit,
  )
}

type NullableResultRef = {
  value: unknown | null
}

export function clearOperationResultRefs<TKey extends string>(
  refs: Record<TKey, NullableResultRef>,
  options?: { keep?: readonly TKey[] },
) {
  const keep = new Set(options?.keep ?? [])
  for (const key of Object.keys(refs) as TKey[]) {
    if (!keep.has(key)) refs[key].value = null
  }
}
