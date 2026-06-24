import {
  normalizeSocialAccountBatchResultStatus,
  socialAccountBatchResultFallbackText,
} from '@/utils/accountWorkbenchBatchResult'

type Translate = (key: string, fallback?: string) => string

interface AccountWorkbenchImportResultItemLike {
  reason?: string | null
  error?: string | null
}

interface AccountWorkbenchImportResultSummaryLike {
  total?: number | null
  succeeded?: number | null
  imported?: number | null
  created?: number | null
  skipped?: number | null
  failed?: number | null
  duplicates?: number | null
}

export const accountWorkbenchImportResultReasonTranslationKeys: Record<string, string> = {
  matched_total_pool: 'accountWorkbench.import.resultReasons.matchedTotalPool',
  staged_not_stored: 'accountWorkbench.import.resultReasons.stagedNotStored',
  duplicate_in_batch: 'accountWorkbench.import.resultReasons.duplicateInBatch',
  duplicate_in_database: 'accountWorkbench.import.resultReasons.duplicateInDatabase',
  already_in_workbench: 'accountWorkbench.import.resultReasons.alreadyInWorkbench',
  already_assigned: 'accountWorkbench.import.resultReasons.alreadyAssigned',
  ambiguous_total_pool_match: 'accountWorkbench.import.resultReasons.ambiguousTotalPoolMatch',
  invalid_input: 'accountWorkbench.import.resultReasons.invalidInput',
  import_failed: 'accountWorkbench.import.resultReasons.importFailed',
}

const accountWorkbenchImportResultErrorTranslationKeys: Record<string, string> = {
  'matched an existing total-pool account': accountWorkbenchImportResultReasonTranslationKeys.matched_total_pool,
  'staged as a not-stored workbench account': accountWorkbenchImportResultReasonTranslationKeys.staged_not_stored,
  'account import data is invalid': accountWorkbenchImportResultReasonTranslationKeys.invalid_input,
  'account is duplicated in this import batch': accountWorkbenchImportResultReasonTranslationKeys.duplicate_in_batch,
  'account already exists in the total account pool': accountWorkbenchImportResultReasonTranslationKeys.duplicate_in_database,
  'account already exists in your workbench': accountWorkbenchImportResultReasonTranslationKeys.already_in_workbench,
  'account is already assigned to a workbench': accountWorkbenchImportResultReasonTranslationKeys.already_assigned,
  'multiple total-pool accounts match this username': accountWorkbenchImportResultReasonTranslationKeys.ambiguous_total_pool_match,
  'account could not be imported': accountWorkbenchImportResultReasonTranslationKeys.import_failed,
}

function importResultCount(value: unknown) {
  const count = Number(value)
  return Number.isFinite(count) ? count : 0
}

export function accountWorkbenchImportResultSummaryParams(result: AccountWorkbenchImportResultSummaryLike) {
  const imported = result.imported ?? result.created
  const created = result.created ?? result.imported
  return {
    total: importResultCount(result.total),
    succeeded: importResultCount(result.succeeded),
    imported: importResultCount(imported),
    created: importResultCount(created),
    skipped: importResultCount(result.skipped),
    failed: importResultCount(result.failed),
    duplicates: importResultCount(result.duplicates),
  }
}

export function accountWorkbenchImportResultReasonTranslationKey(reason?: string | null) {
  const normalized = String(reason || '').trim().toLowerCase()
  return accountWorkbenchImportResultReasonTranslationKeys[normalized]
}

function accountWorkbenchImportResultErrorTranslationKey(error?: string | null) {
  const normalized = String(error || '').trim().toLowerCase()
  return accountWorkbenchImportResultErrorTranslationKeys[normalized]
}

export function accountWorkbenchImportResultStatusLabel(value: string | null | undefined, translate: Translate) {
  const normalized = normalizeSocialAccountBatchResultStatus(value)
  if (normalized === 'succeeded') return translate('common.success')
  if (normalized === 'duplicate') return translate('accountWorkbench.import.status.duplicate')
  if (normalized === 'skipped') return translate('accountWorkbench.import.status.skipped')
  if (normalized === 'failed') return translate('common.error')
  return socialAccountBatchResultFallbackText(value)
}

export function accountWorkbenchImportResultItemMessage(
  item: AccountWorkbenchImportResultItemLike,
  translate: Translate,
) {
  const reasonTranslationKey = accountWorkbenchImportResultReasonTranslationKey(item.reason)
  if (reasonTranslationKey) return translate(reasonTranslationKey)
  const reason = String(item.reason || '').trim()
  const error = String(item.error || '').trim()
  const errorTranslationKey = accountWorkbenchImportResultErrorTranslationKey(error)
  if (errorTranslationKey) return translate(errorTranslationKey)
  if (reason || error) return translate(accountWorkbenchImportResultReasonTranslationKeys.import_failed)
  return socialAccountBatchResultFallbackText()
}
