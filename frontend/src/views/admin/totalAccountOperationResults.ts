import type { Ref } from 'vue'
import { useOperationResultPreview } from '@/utils/operationResults'
import {
  socialAccountBatchResultToastParams,
  type AccountWorkbenchBatchItemLike,
  type AccountWorkbenchBatchResultLike,
} from '@/utils/accountWorkbenchBatchResult'

export const TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT = 8

export interface TotalAccountOperationResultPreviewOptions<TResult, TItem> {
  result: Ref<TResult | null>
  items: (result: TResult) => readonly TItem[] | null | undefined
  previewLimit?: number
}

export interface TotalAccountBatchOperationSummaryState {
  result: Pick<AccountWorkbenchBatchResultLike, 'total' | 'succeeded' | 'failed' | 'skipped'>
  targetUser?: string | null
}

export function useTotalAccountOperationResultPreview<
  TResult,
  TItem extends AccountWorkbenchBatchItemLike,
>(options: TotalAccountOperationResultPreviewOptions<TResult, TItem>) {
  const previewLimit = options.previewLimit ?? TOTAL_ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT
  return useOperationResultPreview({ ...options, previewLimit })
}

export function totalAccountBatchOperationSummaryParams(state: TotalAccountBatchOperationSummaryState) {
  return socialAccountBatchResultToastParams(
    state.result,
    state.targetUser ? { user: state.targetUser } : {},
  )
}
