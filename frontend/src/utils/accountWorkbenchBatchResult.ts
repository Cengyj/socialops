export interface AccountWorkbenchBatchItemLike {
  id?: number
  name?: string | null
  status?: string
  reason?: string | null
  error?: string | null
}

export interface AccountWorkbenchBatchResultLike {
  total?: number
  succeeded: number
  skipped: number
  failed: number
  items?: AccountWorkbenchBatchItemLike[]
}

export type SocialAccountBatchResultToastTone = 'success' | 'warning' | 'error'
type Translate = (key: string, fallback?: string) => string

export interface SocialAccountBatchResultToastHandlers {
  showError: (message: string) => void
  showSuccess: (message: string) => void
  showWarning: (message: string) => void
}

export interface SocialAccountBatchResultToastPayload {
  succeeded: number
  failed: number
  skipped: number
  summary: string
  successMessage: string
  preferWarning?: boolean
}

export function socialAccountBatchResultSummaryParams(
  result: Pick<AccountWorkbenchBatchResultLike, 'total' | 'succeeded' | 'failed' | 'skipped'>,
) {
  return {
    total: Number(result.total || 0),
    succeeded: Number(result.succeeded || 0),
    failed: Number(result.failed || 0),
    skipped: Number(result.skipped || 0),
  }
}

export function socialAccountBatchDeleteResultSummaryParams(
  result: Pick<AccountWorkbenchBatchResultLike, 'total' | 'succeeded' | 'failed' | 'skipped'> & { removed?: number | null },
) {
  return {
    ...socialAccountBatchResultSummaryParams(result),
    removed: Number(result.removed ?? result.succeeded ?? 0),
  }
}

export function socialAccountBatchResultToastParams(
  result: Pick<AccountWorkbenchBatchResultLike, 'total' | 'succeeded' | 'failed' | 'skipped'>,
  extraParams: Record<string, string | number> = {},
) {
  return {
    count: Number(result.succeeded || 0),
    ...socialAccountBatchResultSummaryParams(result),
    ...extraParams,
  }
}

export function normalizeSocialAccountBatchResultStatus(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

export function socialAccountBatchResultFallbackText(value?: string | null) {
  const trimmed = String(value || '').trim()
  return trimmed || '-'
}

export function socialAccountBatchResultStatusLabel(
  value: string | null | undefined,
  keyPrefix: string,
  translate: Translate,
) {
  const normalized = normalizeSocialAccountBatchResultStatus(value)
  return translate(`${keyPrefix}.${normalized}`, socialAccountBatchResultFallbackText(value))
}

export function formatSocialAccountBatchResultName(name: string | null | undefined, fallback: string) {
  const trimmed = String(name || '').trim()
  return trimmed || fallback
}

export function formatSocialAccountBatchResultItemLabel(
  item: Pick<AccountWorkbenchBatchItemLike, 'id' | 'name'>,
  fallback: string,
) {
  return formatSocialAccountBatchResultName(item.name, item.id ? `#${item.id}` : fallback)
}

export function socialAccountBatchResultToastTone(
  result: Pick<AccountWorkbenchBatchResultLike, 'succeeded' | 'failed' | 'skipped'>,
): SocialAccountBatchResultToastTone {
  const succeeded = Number(result.succeeded || 0)
  const failed = Number(result.failed || 0)
  const skipped = Number(result.skipped || 0)
  const hasProblem = failed > 0 || skipped > 0

  if (succeeded === 0 && !hasProblem) return 'error'
  if (succeeded === 0 && hasProblem) return 'error'
  if (hasProblem) return 'warning'
  return 'success'
}

export function showSocialAccountBatchResultToast(
  payload: SocialAccountBatchResultToastPayload,
  handlers: SocialAccountBatchResultToastHandlers,
) {
  const tone = socialAccountBatchResultToastTone(payload)
  if (tone === 'error') {
    handlers.showError(payload.summary)
    return tone
  }
  if (tone === 'warning' || payload.preferWarning) {
    handlers.showWarning(payload.summary)
    return 'warning'
  }
  handlers.showSuccess(payload.successMessage)
  return tone
}

export function collectSucceededBatchItemIds(result: AccountWorkbenchBatchResultLike, requestedIds: number[]) {
  const succeededIds = new Set(
    (result.items ?? [])
      .filter(item => normalizeSocialAccountBatchResultStatus(item.status) === 'succeeded' && typeof item.id === 'number')
      .map(item => item.id as number),
  )
  const fullySucceededWithoutItemIds = succeededIds.size === 0
    && result.succeeded === requestedIds.length
    && result.failed === 0
    && result.skipped === 0
  return fullySucceededWithoutItemIds ? new Set(requestedIds) : succeededIds
}

export const socialAccountBatchResultReasonTranslationKeys: Record<string, string> = {
  invalid_id: 'accountWorkbench.batchResultReasons.invalidId',
  invalid_input: 'accountWorkbench.batchResultReasons.invalidInput',
  duplicate_in_batch: 'accountWorkbench.batchResultReasons.duplicateInBatch',
  duplicate_in_database: 'accountWorkbench.batchResultReasons.duplicateInDatabase',
  account_not_found: 'accountWorkbench.batchResultReasons.accountNotFound',
  account_not_assigned: 'accountWorkbench.batchResultReasons.accountNotAssigned',
  not_assigned: 'accountWorkbench.batchResultReasons.alreadyUnassigned',
  proxy_not_available: 'accountWorkbench.batchResultReasons.proxyNotAvailable',
  assign_failed: 'accountWorkbench.batchResultReasons.assignFailed',
  not_found: 'accountWorkbench.batchResultReasons.notFound',
  already_stored: 'accountWorkbench.batchResultReasons.alreadyStored',
  invalid_credentials: 'accountWorkbench.batchResultReasons.invalidCredentials',
  already_assigned: 'accountWorkbench.batchResultReasons.alreadyAssigned',
  already_unassigned: 'accountWorkbench.batchResultReasons.alreadyUnassigned',
  target_user_not_found: 'accountWorkbench.batchResultReasons.targetUserNotFound',
  reclaim_failed: 'accountWorkbench.batchResultReasons.reclaimFailed',
  delete_failed: 'accountWorkbench.batchResultReasons.deleteFailed',
  create_failed: 'accountWorkbench.batchResultReasons.createFailed',
  load_failed: 'accountWorkbench.batchResultReasons.loadFailed',
  upload_failed: 'accountWorkbench.batchResultReasons.uploadFailed',
  operation_failed: 'accountWorkbench.batchResultReasons.operationFailed',
  state_changed: 'accountWorkbench.batchResultReasons.stateChanged',
}

const socialAccountBatchResultErrorTranslationKeys: Record<string, string> = {
  'invalid credentials': socialAccountBatchResultReasonTranslationKeys.invalid_credentials,
  'missing platform or name': socialAccountBatchResultReasonTranslationKeys.invalid_input,
  'account import requires account, password, and 2fa, email, or cookie': socialAccountBatchResultReasonTranslationKeys.invalid_input,
  'nil input': socialAccountBatchResultReasonTranslationKeys.invalid_input,
  'duplicate account in import batch': socialAccountBatchResultReasonTranslationKeys.duplicate_in_batch,
  'duplicate account in total pool': socialAccountBatchResultReasonTranslationKeys.duplicate_in_database,
  'account could not be assigned': socialAccountBatchResultReasonTranslationKeys.assign_failed,
  'account proxy could not be assigned': socialAccountBatchResultReasonTranslationKeys.assign_failed,
  'account could not be reclaimed': socialAccountBatchResultReasonTranslationKeys.reclaim_failed,
  'account could not be deleted': socialAccountBatchResultReasonTranslationKeys.delete_failed,
  'account could not be uploaded': socialAccountBatchResultReasonTranslationKeys.upload_failed,
  'account is already assigned': socialAccountBatchResultReasonTranslationKeys.already_assigned,
  'account is already unassigned': socialAccountBatchResultReasonTranslationKeys.already_unassigned,
  'target user not found': socialAccountBatchResultReasonTranslationKeys.target_user_not_found,
  'account is not a workbench staging account': socialAccountBatchResultReasonTranslationKeys.already_stored,
}

export function socialAccountBatchResultReasonTranslationKey(reason?: string | null) {
  const normalized = String(reason || '').trim().toLowerCase()
  return socialAccountBatchResultReasonTranslationKeys[normalized]
}

function socialAccountBatchResultErrorTranslationKey(error?: string | null) {
  const normalized = String(error || '').trim().toLowerCase()
  return socialAccountBatchResultErrorTranslationKeys[normalized]
}

export function socialAccountBatchResultItemMessage(
  item: Pick<AccountWorkbenchBatchItemLike, 'reason' | 'error'>,
  translate: (key: string) => string,
) {
  const reasonTranslationKey = socialAccountBatchResultReasonTranslationKey(item.reason)
  if (reasonTranslationKey) return translate(reasonTranslationKey)
  const reason = String(item.reason || '').trim()
  const error = String(item.error || '').trim()
  const errorTranslationKey = socialAccountBatchResultErrorTranslationKey(error)
  if (errorTranslationKey) return translate(errorTranslationKey)
  if (reason || error) return translate('accountWorkbench.batchResultReasons.operationFailed')
  return socialAccountBatchResultFallbackText()
}

export function socialAccountBatchResultRowToneClass(value?: string | null) {
  const normalized = normalizeSocialAccountBatchResultStatus(value)
  if (normalized === 'succeeded') return 'bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-100 dark:ring-emerald-900/50'
  if (normalized === 'skipped' || normalized === 'duplicate') return 'bg-amber-50 text-amber-800 ring-1 ring-amber-200 dark:bg-amber-950/30 dark:text-amber-100 dark:ring-amber-900/50'
  if (normalized === 'failed') return 'bg-red-50 text-red-800 ring-1 ring-red-200 dark:bg-red-950/40 dark:text-red-200 dark:ring-red-900/60'
  return 'bg-gray-50 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600'
}
