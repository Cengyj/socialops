import { computed, ref } from 'vue'
import type { BatchImportSocialAccountResponse, SocialAccountBatchResult } from '@/api/accountWorkbench'
import { clearOperationResultRefs, useOperationResultPreview } from '@/utils/operationResults'

export type AccountOperationResultKey = 'batchImport' | 'storeWorkbench' | 'proxyAssignment'
export const ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT = 12

type SocialAccountResultSummarySource = Pick<SocialAccountBatchResult, 'total' | 'succeeded' | 'failed' | 'skipped'>

interface UseAccountOperationResultsOptions {
  previewLimit?: number
  batchImportResultSummaryText: (result: BatchImportSocialAccountResponse | null | undefined) => string
  storeWorkbenchResultSummaryText: (result: SocialAccountResultSummarySource | null | undefined) => string
  proxyAssignmentResultSummaryText: (result: SocialAccountResultSummarySource | null | undefined) => string
}

export function useAccountOperationResults(options: UseAccountOperationResultsOptions) {
  const previewLimit = options.previewLimit ?? ACCOUNT_OPERATION_RESULT_PREVIEW_LIMIT
  const proxyAssignmentResult = ref<SocialAccountBatchResult | null>(null)
  const storeWorkbenchResult = ref<SocialAccountBatchResult | null>(null)
  const batchImportResult = ref<BatchImportSocialAccountResponse | null>(null)

  const storeWorkbenchResultPreview = useOperationResultPreview({
    result: storeWorkbenchResult,
    items: result => result.items,
    previewLimit,
  })
  const storeWorkbenchResultPreviewItems = storeWorkbenchResultPreview.items
  const remainingStoreWorkbenchResultItemCount = storeWorkbenchResultPreview.remainingCount
  const storeWorkbenchResultSummary = computed(() => options.storeWorkbenchResultSummaryText(storeWorkbenchResult.value))

  const batchImportResultPreview = useOperationResultPreview({
    result: batchImportResult,
    items: result => result.items,
    previewLimit,
  })
  const batchImportResultPreviewItems = batchImportResultPreview.items
  const remainingBatchImportResultItemCount = batchImportResultPreview.remainingCount
  const batchImportResultSummary = computed(() => options.batchImportResultSummaryText(batchImportResult.value))

  const proxyAssignmentResultPreview = useOperationResultPreview({
    result: proxyAssignmentResult,
    items: result => result.items,
    previewLimit,
  })
  const proxyAssignmentResultPreviewItems = proxyAssignmentResultPreview.items
  const remainingProxyAssignmentResultItemCount = proxyAssignmentResultPreview.remainingCount
  const proxyAssignmentResultSummary = computed(() => options.proxyAssignmentResultSummaryText(proxyAssignmentResult.value))

  function clearAccountOperationResults(clearOptions?: { keep?: AccountOperationResultKey[] }) {
    clearOperationResultRefs<AccountOperationResultKey>({
      batchImport: batchImportResult,
      storeWorkbench: storeWorkbenchResult,
      proxyAssignment: proxyAssignmentResult,
    }, clearOptions)
  }

  return {
    batchImportResult,
    batchImportResultPreviewItems,
    batchImportResultSummary,
    clearAccountOperationResults,
    proxyAssignmentResult,
    proxyAssignmentResultPreviewItems,
    proxyAssignmentResultSummary,
    remainingBatchImportResultItemCount,
    remainingProxyAssignmentResultItemCount,
    remainingStoreWorkbenchResultItemCount,
    storeWorkbenchResult,
    storeWorkbenchResultPreviewItems,
    storeWorkbenchResultSummary,
  }
}
