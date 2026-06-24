type Translate = (key: string) => string

const processingKey = 'common.processing'
const selectionRequiredKey = 'admin.socialAccountWorkbench.executionBar.selectionRequired'

export function accountRefreshButtonTitle(t: Translate, state: { loading: boolean }) {
  return state.loading ? t(processingKey) : t('common.refresh')
}

export function accountBatchImportButtonTitle(t: Translate, state: { locked: boolean }) {
  return state.locked ? t(processingKey) : t('accountWorkbench.import.batchAction')
}

export function accountBatchImportCancelButtonTitle(t: Translate, state: { processing: boolean }) {
  return state.processing ? t(processingKey) : t('common.cancel')
}

export function accountExportButtonTitle(t: Translate, state: { loading: boolean; exporting: boolean; selectedCount: number }) {
  if (state.loading || state.exporting) return t(processingKey)
  if (state.selectedCount > 0) return t('accountWorkbench.exportSelectedAccounts')
  return t('accountWorkbench.exportAccounts')
}

export function accountBatchProxyButtonTitle(t: Translate, state: { loading: boolean; savingProxy: boolean; selectedCount: number }) {
  if (state.loading || state.savingProxy) return t(processingKey)
  if (state.selectedCount === 0) return t(selectionRequiredKey)
  return t('accountWorkbench.proxy.batchAction')
}

export function accountClearSelectionButtonTitle(t: Translate, state: { loading: boolean; selectedCount: number }) {
  if (state.loading) return t(processingKey)
  if (state.selectedCount === 0) return t(selectionRequiredKey)
  return t('admin.socialAccountWorkbench.executionBar.clear')
}

export function accountDeleteSelectedButtonTitle(t: Translate, state: { loading: boolean; deleting: boolean; selectedCount: number }) {
  if (state.loading || state.deleting) return t(processingKey)
  if (state.selectedCount === 0) return t(selectionRequiredKey)
  return t('accountWorkbench.deleteSelected')
}

export function accountDeleteCancelButtonTitle(t: Translate, state: { deleting: boolean }) {
  return state.deleting ? t(processingKey) : t('common.cancel')
}

export function accountDetailCloseButtonTitle(t: Translate, state: { refreshingExecutionAuth: boolean }) {
  return state.refreshingExecutionAuth ? t(processingKey) : t('common.close')
}

export function accountStoreWorkbenchCancelButtonTitle(t: Translate, state: { storing: boolean }) {
  return state.storing ? t(processingKey) : t('common.cancel')
}

export function accountRowProxyButtonTitle(t: Translate, state: { locked: boolean; savingProxy: boolean }) {
  return state.locked || state.savingProxy ? t(processingKey) : t('accountWorkbench.proxy.action')
}

export function accountProxyCancelButtonTitle(t: Translate, state: { savingProxy: boolean }) {
  return state.savingProxy ? t('common.saving') : t('common.cancel')
}

export function accountRowEditButtonTitle(t: Translate, state: { locked: boolean; saving: boolean }) {
  if (state.saving) return t('common.saving')
  if (state.locked) return t(processingKey)
  return t('common.edit')
}

export function accountRowDeleteButtonTitle(t: Translate, state: { locked: boolean; deleting: boolean }) {
  return state.locked || state.deleting ? t(processingKey) : t('accountWorkbench.deleteOne')
}

export function accountEditSaveButtonTitle(t: Translate, state: { saving: boolean; locked: boolean; disabledReason: string }) {
  if (state.saving) return t('common.saving')
  if (state.locked) return t(processingKey)
  return state.disabledReason || t('common.save')
}

export function accountEditSaveButtonLabel(t: Translate, state: { saving: boolean }) {
  return state.saving ? t('common.saving') : t('common.save')
}

export function accountEditCancelButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t('common.saving') : t('common.cancel')
}

export function accountExecutionStartButtonTitle(t: Translate, state: { submitting: boolean; locked: boolean; disabledReason: string }) {
  if (state.submitting || state.locked) return t(processingKey)
  return state.disabledReason || t('accountWorkbench.execution.start')
}

export function accountExecutionCancelButtonTitle(t: Translate, state: { submitting: boolean }) {
  return state.submitting ? t(processingKey) : t('common.cancel')
}

export function accountExecutionConfirmButtonTitle(t: Translate, state: { submitting: boolean }) {
  return state.submitting ? t(processingKey) : t('accountWorkbench.execution.confirmSubmit')
}
