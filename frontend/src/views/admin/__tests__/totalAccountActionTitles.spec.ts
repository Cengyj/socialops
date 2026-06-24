import { describe, expect, it } from 'vitest'

import {
  totalAccountAssignBackButtonTitle,
  totalAccountAssignConfirmButtonTitle,
  totalAccountAssignReviewButtonTitle,
  totalAccountAssignSelectedButtonTitle,
  totalAccountClearSelectionButtonTitle,
  totalAccountDeleteConfirmButtonTitle,
  totalAccountDeleteSelectedButtonTitle,
  totalAccountDialogCancelButtonTitle,
  totalAccountEditSubmitButtonTitle,
  totalAccountExportButtonTitle,
  totalAccountImportButtonTitle,
  totalAccountReclaimConfirmButtonTitle,
  totalAccountReclaimSelectedButtonTitle,
  totalAccountRefreshButtonTitle,
  totalAccountRowDetailButtonTitle,
  totalAccountRowEditButtonTitle,
  totalAccountSubmitButtonLabel,
} from '../totalAccountActionTitles'

const messages: Record<string, string> = {
  'common.cancel': 'Cancel',
  'common.confirm': 'Confirm',
  'common.edit': 'Edit',
  'common.processing': 'Processing',
  'common.refresh': 'Refresh',
  'admin.socialAccountWorkbench.actions.assign': 'Assign',
  'admin.socialAccountWorkbench.actions.delete': 'Delete',
  'admin.socialAccountWorkbench.actions.reclaim': 'Reclaim',
  'admin.socialAccountWorkbench.assignDialog.backToSelect': 'Back',
  'admin.socialAccountWorkbench.assignDialog.confirm': 'Confirm assign',
  'admin.socialAccountWorkbench.assignDialog.reviewButton': 'Review assignment',
  'admin.socialAccountWorkbench.deleteDialog.confirm': 'Delete selected',
  'admin.socialAccountWorkbench.executionBar.clear': 'Clear',
  'admin.socialAccountWorkbench.executionBar.noAssignedSelection': 'Select at least one assigned account to return to the pool.',
  'admin.socialAccountWorkbench.executionBar.selectionRequired': 'Select at least one account first.',
  'admin.socialAccountWorkbench.reclaimDialog.confirm': 'Reclaim selected',
  'admin.socialAccountWorkbench.rowActions.detail': 'Account detail',
  'admin.socialAccountWorkbench.toolbar.exportRecords': 'Export',
  'admin.socialAccountWorkbench.toolbar.exportSelectedRecords': 'Export selected',
  'admin.socialAccountWorkbench.toolbar.importAccounts': 'Import',
  'admin.socialAccountWorkbench.toasts.assignRequiresUnassigned': '{count} selected account(s) are already assigned.',
  'admin.socialAccountWorkbench.toasts.selectTargetUser': 'Please choose a target user.',
}

function t(key: string, params?: Record<string, unknown>) {
  const message = messages[key] ?? key
  return message.replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`))
}

describe('total account action titles', () => {
  it('keeps idle toolbar and row titles aligned with existing actions', () => {
    expect(totalAccountRefreshButtonTitle(t, { loading: false })).toBe('Refresh')
    expect(totalAccountImportButtonTitle(t, { loading: false, importing: false })).toBe('Import')
    expect(totalAccountExportButtonTitle(t, { loading: false, exporting: false, selectedCount: 0 })).toBe('Export')
    expect(totalAccountExportButtonTitle(t, { loading: false, exporting: false, selectedCount: 2 })).toBe('Export selected')
    expect(totalAccountRowDetailButtonTitle(t, { loading: false })).toBe('Account detail')
    expect(totalAccountRowEditButtonTitle(t, { loading: false })).toBe('Edit')
  })

  it('keeps busy toolbar and row titles in the existing processing state', () => {
    expect(totalAccountRefreshButtonTitle(t, { loading: true })).toBe('Processing')
    expect(totalAccountImportButtonTitle(t, { loading: false, importing: true })).toBe('Processing')
    expect(totalAccountExportButtonTitle(t, { loading: true, exporting: false, selectedCount: 2 })).toBe('Processing')
    expect(totalAccountRowDetailButtonTitle(t, { loading: true })).toBe('Processing')
    expect(totalAccountRowEditButtonTitle(t, { loading: true })).toBe('Processing')
  })

  it('explains disabled batch actions without changing selection rules', () => {
    expect(totalAccountClearSelectionButtonTitle(t, { loading: false, hasSelection: false })).toBe('Select at least one account first.')
    expect(totalAccountClearSelectionButtonTitle(t, { loading: false, hasSelection: true })).toBe('Clear')
    expect(totalAccountAssignSelectedButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: false,
      canAssign: false,
      selectedAssignedCount: 0,
    })).toBe('Select at least one account first.')
    expect(totalAccountAssignSelectedButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: true,
      canAssign: false,
      selectedAssignedCount: 2,
    })).toBe('2 selected account(s) are already assigned.')
    expect(totalAccountAssignSelectedButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: true,
      canAssign: true,
      selectedAssignedCount: 0,
    })).toBe('Assign')
    expect(totalAccountReclaimSelectedButtonTitle(t, {
      loading: false,
      reclaiming: false,
      hasSelection: true,
      canReclaim: false,
    })).toBe('Select at least one assigned account to return to the pool.')
    expect(totalAccountReclaimSelectedButtonTitle(t, {
      loading: false,
      reclaiming: false,
      hasSelection: true,
      canReclaim: true,
    })).toBe('Reclaim')
    expect(totalAccountDeleteSelectedButtonTitle(t, { loading: false, deleting: false, hasSelection: false })).toBe('Select at least one account first.')
    expect(totalAccountDeleteSelectedButtonTitle(t, { loading: false, deleting: false, hasSelection: true })).toBe('Delete')
  })

  it('prioritizes busy states over selection explanations for batch actions', () => {
    expect(totalAccountClearSelectionButtonTitle(t, { loading: true, hasSelection: false })).toBe('Processing')
    expect(totalAccountAssignSelectedButtonTitle(t, {
      loading: false,
      assigning: true,
      hasSelection: false,
      canAssign: false,
      selectedAssignedCount: 0,
    })).toBe('Processing')
    expect(totalAccountReclaimSelectedButtonTitle(t, {
      loading: true,
      reclaiming: false,
      hasSelection: false,
      canReclaim: false,
    })).toBe('Processing')
    expect(totalAccountDeleteSelectedButtonTitle(t, { loading: false, deleting: true, hasSelection: false })).toBe('Processing')
  })

  it('keeps dialog titles aligned with processing and validation states', () => {
    expect(totalAccountDialogCancelButtonTitle(t, { processing: false })).toBe('Cancel')
    expect(totalAccountDialogCancelButtonTitle(t, { processing: true })).toBe('Processing')
    expect(totalAccountAssignBackButtonTitle(t, { assigning: false })).toBe('Back')
    expect(totalAccountAssignBackButtonTitle(t, { assigning: true })).toBe('Processing')
    expect(totalAccountAssignConfirmButtonTitle(t, { assigning: false })).toBe('Confirm assign')
    expect(totalAccountAssignConfirmButtonTitle(t, { assigning: true })).toBe('Processing')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: true,
      hasTargetUser: true,
      canAssign: true,
      selectedAssignedCount: 0,
    })).toBe('Review assignment')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: false,
      hasTargetUser: false,
      canAssign: false,
      selectedAssignedCount: 0,
    })).toBe('Select at least one account first.')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: true,
      hasTargetUser: false,
      canAssign: true,
      selectedAssignedCount: 0,
    })).toBe('Please choose a target user.')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: false,
      assigning: false,
      hasSelection: true,
      hasTargetUser: true,
      canAssign: false,
      selectedAssignedCount: 2,
    })).toBe('2 selected account(s) are already assigned.')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: true,
      assigning: false,
      hasSelection: false,
      hasTargetUser: false,
      canAssign: false,
      selectedAssignedCount: 0,
    })).toBe('Processing')
    expect(totalAccountAssignReviewButtonTitle(t, {
      loading: false,
      assigning: true,
      hasSelection: true,
      hasTargetUser: false,
      canAssign: true,
      selectedAssignedCount: 0,
    })).toBe('Processing')
    expect(totalAccountEditSubmitButtonTitle(t, { saving: false, disabledReason: '' })).toBe('Confirm')
    expect(totalAccountEditSubmitButtonTitle(t, { saving: false, disabledReason: 'No changes to save.' })).toBe('No changes to save.')
    expect(totalAccountEditSubmitButtonTitle(t, { saving: true, disabledReason: 'No changes to save.' })).toBe('Processing')
    expect(totalAccountSubmitButtonLabel(t, { processing: false })).toBe('Confirm')
    expect(totalAccountSubmitButtonLabel(t, { processing: true })).toBe('Processing')
    expect(totalAccountReclaimConfirmButtonTitle(t, { reclaiming: false })).toBe('Reclaim selected')
    expect(totalAccountReclaimConfirmButtonTitle(t, { reclaiming: true })).toBe('Processing')
    expect(totalAccountDeleteConfirmButtonTitle(t, { deleting: false })).toBe('Delete selected')
    expect(totalAccountDeleteConfirmButtonTitle(t, { deleting: true })).toBe('Processing')
  })
})
