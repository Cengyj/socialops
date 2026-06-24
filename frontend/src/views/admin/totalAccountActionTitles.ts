type Translate = (key: string, params?: Record<string, unknown>) => string

const processingKey = 'common.processing'
const selectionRequiredKey = 'admin.socialAccountWorkbench.executionBar.selectionRequired'

export function totalAccountRefreshButtonTitle(t: Translate, state: { loading: boolean }) {
  return state.loading ? t(processingKey) : t('common.refresh')
}

export function totalAccountImportButtonTitle(t: Translate, state: { loading: boolean; importing: boolean }) {
  return state.loading || state.importing ? t(processingKey) : t('admin.socialAccountWorkbench.toolbar.importAccounts')
}

export function totalAccountExportButtonTitle(t: Translate, state: { loading: boolean; exporting: boolean; selectedCount: number }) {
  if (state.loading || state.exporting) return t(processingKey)
  if (state.selectedCount > 0) return t('admin.socialAccountWorkbench.toolbar.exportSelectedRecords')
  return t('admin.socialAccountWorkbench.toolbar.exportRecords')
}

export function totalAccountRowDetailButtonTitle(t: Translate, state: { loading: boolean }) {
  return state.loading ? t(processingKey) : t('admin.socialAccountWorkbench.rowActions.detail')
}

export function totalAccountRowEditButtonTitle(t: Translate, state: { loading: boolean }) {
  return state.loading ? t(processingKey) : t('common.edit')
}

export function totalAccountClearSelectionButtonTitle(t: Translate, state: { loading: boolean; hasSelection: boolean }) {
  if (state.loading) return t(processingKey)
  if (!state.hasSelection) return t(selectionRequiredKey)
  return t('admin.socialAccountWorkbench.executionBar.clear')
}

export function totalAccountAssignSelectedButtonTitle(
  t: Translate,
  state: { loading: boolean; assigning: boolean; hasSelection: boolean; canAssign: boolean; selectedAssignedCount: number },
) {
  if (state.loading || state.assigning) return t(processingKey)
  if (!state.hasSelection) return t(selectionRequiredKey)
  if (!state.canAssign) {
    return t('admin.socialAccountWorkbench.toasts.assignRequiresUnassigned', { count: state.selectedAssignedCount })
  }
  return t('admin.socialAccountWorkbench.actions.assign')
}

export function totalAccountReclaimSelectedButtonTitle(
  t: Translate,
  state: { loading: boolean; reclaiming: boolean; hasSelection: boolean; canReclaim: boolean },
) {
  if (state.loading || state.reclaiming) return t(processingKey)
  if (!state.hasSelection) return t(selectionRequiredKey)
  if (!state.canReclaim) return t('admin.socialAccountWorkbench.executionBar.noAssignedSelection')
  return t('admin.socialAccountWorkbench.actions.reclaim')
}

export function totalAccountDeleteSelectedButtonTitle(t: Translate, state: { loading: boolean; deleting: boolean; hasSelection: boolean }) {
  if (state.loading || state.deleting) return t(processingKey)
  if (!state.hasSelection) return t(selectionRequiredKey)
  return t('admin.socialAccountWorkbench.actions.delete')
}

export function totalAccountDialogCancelButtonTitle(t: Translate, state: { processing: boolean }) {
  return state.processing ? t(processingKey) : t('common.cancel')
}

export function totalAccountAssignBackButtonTitle(t: Translate, state: { assigning: boolean }) {
  return state.assigning ? t(processingKey) : t('admin.socialAccountWorkbench.assignDialog.backToSelect')
}

export function totalAccountAssignConfirmButtonTitle(t: Translate, state: { assigning: boolean }) {
  return state.assigning ? t(processingKey) : t('admin.socialAccountWorkbench.assignDialog.confirm')
}

export function totalAccountAssignReviewButtonTitle(
  t: Translate,
  state: {
    loading: boolean
    assigning: boolean
    hasSelection: boolean
    hasTargetUser: boolean
    canAssign: boolean
    selectedAssignedCount: number
  },
) {
  if (state.loading || state.assigning) return t(processingKey)
  if (!state.hasSelection) return t(selectionRequiredKey)
  if (!state.canAssign) {
    return t('admin.socialAccountWorkbench.toasts.assignRequiresUnassigned', { count: state.selectedAssignedCount })
  }
  if (!state.hasTargetUser) return t('admin.socialAccountWorkbench.toasts.selectTargetUser')
  return t('admin.socialAccountWorkbench.assignDialog.reviewButton')
}

export function totalAccountEditSubmitButtonTitle(t: Translate, state: { saving: boolean; disabledReason: string }) {
  if (state.saving) return t(processingKey)
  if (state.disabledReason) return state.disabledReason
  return t('common.confirm')
}

export function totalAccountSubmitButtonLabel(t: Translate, state: { processing: boolean }) {
  return state.processing ? t(processingKey) : t('common.confirm')
}

export function totalAccountReclaimConfirmButtonTitle(t: Translate, state: { reclaiming: boolean }) {
  return state.reclaiming ? t(processingKey) : t('admin.socialAccountWorkbench.reclaimDialog.confirm')
}

export function totalAccountDeleteConfirmButtonTitle(t: Translate, state: { deleting: boolean }) {
  return state.deleting ? t(processingKey) : t('admin.socialAccountWorkbench.deleteDialog.confirm')
}
