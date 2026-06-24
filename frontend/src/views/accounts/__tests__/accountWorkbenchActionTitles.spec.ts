import { describe, expect, it } from 'vitest'

import {
  accountBatchImportButtonTitle,
  accountBatchImportCancelButtonTitle,
  accountBatchProxyButtonTitle,
  accountClearSelectionButtonTitle,
  accountDeleteCancelButtonTitle,
  accountDeleteSelectedButtonTitle,
  accountDetailCloseButtonTitle,
  accountEditCancelButtonTitle,
  accountEditSaveButtonLabel,
  accountEditSaveButtonTitle,
  accountExecutionCancelButtonTitle,
  accountExecutionConfirmButtonTitle,
  accountExecutionStartButtonTitle,
  accountExportButtonTitle,
  accountProxyCancelButtonTitle,
  accountRefreshButtonTitle,
  accountRowDeleteButtonTitle,
  accountRowEditButtonTitle,
  accountRowProxyButtonTitle,
  accountStoreWorkbenchCancelButtonTitle,
} from '../accountWorkbenchActionTitles'

const messages: Record<string, string> = {
  'common.cancel': 'Cancel',
  'common.close': 'Close',
  'common.edit': 'Edit',
  'common.processing': 'Processing',
  'common.refresh': 'Refresh',
  'common.save': 'Save',
  'common.saving': 'Saving',
  'accountWorkbench.deleteOne': 'Delete account',
  'accountWorkbench.deleteSelected': 'Delete selected',
  'accountWorkbench.execution.confirmSubmit': 'Confirm submission',
  'accountWorkbench.execution.start': 'Start execution',
  'accountWorkbench.exportAccounts': 'Export accounts',
  'accountWorkbench.exportSelectedAccounts': 'Export selected accounts',
  'accountWorkbench.import.batchAction': 'Batch import',
  'accountWorkbench.proxy.action': 'Proxy',
  'accountWorkbench.proxy.batchAction': 'Batch proxy',
  'admin.socialAccountWorkbench.executionBar.clear': 'Clear selection',
  'admin.socialAccountWorkbench.executionBar.selectionRequired': 'Select at least one account first.',
}

function t(key: string) {
  return messages[key] ?? key
}

describe('account workbench action titles', () => {
  it('keeps idle toolbar titles aligned with existing actions', () => {
    expect(accountRefreshButtonTitle(t, { loading: false })).toBe('Refresh')
    expect(accountBatchImportButtonTitle(t, { locked: false })).toBe('Batch import')
    expect(accountBatchImportCancelButtonTitle(t, { processing: false })).toBe('Cancel')
    expect(accountExportButtonTitle(t, { loading: false, exporting: false, selectedCount: 0 })).toBe('Export accounts')
    expect(accountExportButtonTitle(t, { loading: false, exporting: false, selectedCount: 2 })).toBe('Export selected accounts')
    expect(accountBatchProxyButtonTitle(t, { loading: false, savingProxy: false, selectedCount: 2 })).toBe('Batch proxy')
    expect(accountClearSelectionButtonTitle(t, { loading: false, selectedCount: 2 })).toBe('Clear selection')
    expect(accountDeleteSelectedButtonTitle(t, { loading: false, deleting: false, selectedCount: 2 })).toBe('Delete selected')
  })

  it('explains disabled toolbar states without changing action availability rules', () => {
    expect(accountRefreshButtonTitle(t, { loading: true })).toBe('Processing')
    expect(accountBatchImportButtonTitle(t, { locked: true })).toBe('Processing')
    expect(accountBatchImportCancelButtonTitle(t, { processing: true })).toBe('Processing')
    expect(accountExportButtonTitle(t, { loading: false, exporting: true, selectedCount: 2 })).toBe('Processing')
    expect(accountBatchProxyButtonTitle(t, { loading: false, savingProxy: false, selectedCount: 0 })).toBe('Select at least one account first.')
    expect(accountClearSelectionButtonTitle(t, { loading: false, selectedCount: 0 })).toBe('Select at least one account first.')
    expect(accountDeleteSelectedButtonTitle(t, { loading: false, deleting: true, selectedCount: 1 })).toBe('Processing')
    expect(accountDeleteCancelButtonTitle(t, { deleting: false })).toBe('Cancel')
    expect(accountDeleteCancelButtonTitle(t, { deleting: true })).toBe('Processing')
    expect(accountDetailCloseButtonTitle(t, { refreshingExecutionAuth: false })).toBe('Close')
    expect(accountDetailCloseButtonTitle(t, { refreshingExecutionAuth: true })).toBe('Processing')
    expect(accountStoreWorkbenchCancelButtonTitle(t, { storing: false })).toBe('Cancel')
    expect(accountStoreWorkbenchCancelButtonTitle(t, { storing: true })).toBe('Processing')
  })

  it('keeps row action titles specific to the active row operation', () => {
    expect(accountRowProxyButtonTitle(t, { locked: false, savingProxy: false })).toBe('Proxy')
    expect(accountRowProxyButtonTitle(t, { locked: false, savingProxy: true })).toBe('Processing')
    expect(accountProxyCancelButtonTitle(t, { savingProxy: false })).toBe('Cancel')
    expect(accountProxyCancelButtonTitle(t, { savingProxy: true })).toBe('Saving')
    expect(accountRowEditButtonTitle(t, { locked: false, saving: false })).toBe('Edit')
    expect(accountRowEditButtonTitle(t, { locked: false, saving: true })).toBe('Saving')
    expect(accountRowEditButtonTitle(t, { locked: true, saving: false })).toBe('Processing')
    expect(accountRowDeleteButtonTitle(t, { locked: false, deleting: false })).toBe('Delete account')
    expect(accountRowDeleteButtonTitle(t, { locked: true, deleting: false })).toBe('Processing')
  })

  it('keeps edit save titles aligned with the existing disabled reason priority', () => {
    expect(accountEditCancelButtonTitle(t, { saving: false })).toBe('Cancel')
    expect(accountEditCancelButtonTitle(t, { saving: true })).toBe('Saving')
    expect(accountEditSaveButtonTitle(t, { saving: false, locked: false, disabledReason: '' })).toBe('Save')
    expect(accountEditSaveButtonTitle(t, { saving: true, locked: false, disabledReason: 'No changes' })).toBe('Saving')
    expect(accountEditSaveButtonTitle(t, { saving: false, locked: true, disabledReason: 'No changes' })).toBe('Processing')
    expect(accountEditSaveButtonTitle(t, { saving: false, locked: false, disabledReason: 'No changes' })).toBe('No changes')
    expect(accountEditSaveButtonLabel(t, { saving: false })).toBe('Save')
    expect(accountEditSaveButtonLabel(t, { saving: true })).toBe('Saving')
  })

  it('keeps execution start titles aligned with the existing processing and disabled states', () => {
    expect(accountExecutionStartButtonTitle(t, { submitting: false, locked: false, disabledReason: '' })).toBe('Start execution')
    expect(accountExecutionStartButtonTitle(t, { submitting: true, locked: false, disabledReason: 'Select accounts' })).toBe('Processing')
    expect(accountExecutionStartButtonTitle(t, { submitting: false, locked: true, disabledReason: 'Select accounts' })).toBe('Processing')
    expect(accountExecutionStartButtonTitle(t, { submitting: false, locked: false, disabledReason: 'Select accounts' })).toBe('Select accounts')
    expect(accountExecutionCancelButtonTitle(t, { submitting: false })).toBe('Cancel')
    expect(accountExecutionCancelButtonTitle(t, { submitting: true })).toBe('Processing')
    expect(accountExecutionConfirmButtonTitle(t, { submitting: false })).toBe('Confirm submission')
    expect(accountExecutionConfirmButtonTitle(t, { submitting: true })).toBe('Processing')
  })
})
