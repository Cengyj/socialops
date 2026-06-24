import { describe, expect, it } from 'vitest'

import {
  proxyClearTestResultsButtonTitle,
  proxyCreateButtonTitle,
  proxyDeleteCancelButtonTitle,
  proxyDeleteConfirmButtonTitle,
  proxyFormCancelButtonTitle,
  proxyFormSubmitButtonLabel,
  proxyFormSubmitButtonTitle,
  proxyRefreshButtonTitle,
  proxyRowDeleteButtonTitle,
  proxyRowEditButtonTitle,
  proxyRowTestButtonTitle,
  proxyTestAllButtonTitle,
  proxyTestSelectedButtonTitle,
} from '../proxyActionTitles'

const messages: Record<string, string> = {
  'common.cancel': 'Cancel',
  'common.clear': 'Clear',
  'common.confirm': 'Confirm',
  'common.delete': 'Delete',
  'common.edit': 'Edit',
  'common.loading': 'Loading',
  'common.processing': 'Processing',
  'common.refresh': 'Refresh',
  'common.saving': 'Saving',
  'proxies.addProxy': 'Add proxy',
  'proxies.noProxiesToTest': 'No proxies to test.',
  'proxies.selection.noneSelected': 'Select at least one proxy to test.',
  'proxies.test': 'Test',
  'proxies.testAll': 'Test all',
  'proxies.testSelected': 'Test selected',
}

function t(key: string) {
  return messages[key] ?? key
}

describe('proxy action titles', () => {
  it('keeps idle toolbar titles aligned with existing proxy actions', () => {
    expect(proxyRefreshButtonTitle(t, { loading: false })).toBe('Refresh')
    expect(proxyTestSelectedButtonTitle(t, { loading: false, testing: false, selectedCount: 1 })).toBe('Test selected')
    expect(proxyTestAllButtonTitle(t, { loading: false, testing: false, hasProxies: true })).toBe('Test all')
    expect(proxyCreateButtonTitle(t, { testing: false })).toBe('Add proxy')
    expect(proxyClearTestResultsButtonTitle(t, { disabled: false })).toBe('Clear')
  })

  it('explains disabled toolbar states without changing availability rules', () => {
    expect(proxyRefreshButtonTitle(t, { loading: true })).toBe('Processing')
    expect(proxyTestSelectedButtonTitle(t, { loading: true, testing: false, selectedCount: 1 })).toBe('Processing')
    expect(proxyTestSelectedButtonTitle(t, { loading: false, testing: true, selectedCount: 1 })).toBe('Processing')
    expect(proxyTestSelectedButtonTitle(t, { loading: false, testing: false, selectedCount: 0 })).toBe('Select at least one proxy to test.')
    expect(proxyTestAllButtonTitle(t, { loading: false, testing: true, hasProxies: true })).toBe('Processing')
    expect(proxyTestAllButtonTitle(t, { loading: false, testing: false, hasProxies: false })).toBe('No proxies to test.')
    expect(proxyCreateButtonTitle(t, { testing: true })).toBe('Processing')
    expect(proxyClearTestResultsButtonTitle(t, { disabled: true })).toBe('Processing')
    expect(proxyClearTestResultsButtonTitle(t, { disabled: true, loading: true })).toBe('Loading')
    expect(proxyClearTestResultsButtonTitle(t, { disabled: true, testing: true })).toBe('Processing')
  })

  it('keeps row action titles specific to the active row operation', () => {
    expect(proxyRowTestButtonTitle(t, { loading: false, testing: false })).toBe('Test')
    expect(proxyRowEditButtonTitle(t, { loading: false, testing: false })).toBe('Edit')
    expect(proxyRowDeleteButtonTitle(t, { loading: false, testing: false })).toBe('Delete')
    expect(proxyRowTestButtonTitle(t, { loading: true, testing: false })).toBe('Processing')
    expect(proxyRowEditButtonTitle(t, { loading: false, testing: true })).toBe('Processing')
    expect(proxyRowDeleteButtonTitle(t, { loading: true, testing: true })).toBe('Processing')
  })

  it('keeps proxy dialog titles aligned with save and delete blocking states', () => {
    expect(proxyFormCancelButtonTitle(t, { saving: false })).toBe('Cancel')
    expect(proxyFormCancelButtonTitle(t, { saving: true })).toBe('Saving')
    expect(proxyFormSubmitButtonTitle(t, { saving: false, disabledReason: '' })).toBe('Confirm')
    expect(proxyFormSubmitButtonTitle(t, { saving: false, disabledReason: 'Enter a proxy name.' })).toBe('Enter a proxy name.')
    expect(proxyFormSubmitButtonTitle(t, { saving: true, disabledReason: 'Enter a proxy name.' })).toBe('Saving')
    expect(proxyFormSubmitButtonLabel(t, { saving: false })).toBe('Confirm')
    expect(proxyFormSubmitButtonLabel(t, { saving: true })).toBe('Saving')
    expect(proxyDeleteCancelButtonTitle(t, { deleting: false })).toBe('Cancel')
    expect(proxyDeleteCancelButtonTitle(t, { deleting: true })).toBe('Processing')
    expect(proxyDeleteConfirmButtonTitle(t, { deleting: false, locked: false })).toBe('Delete')
    expect(proxyDeleteConfirmButtonTitle(t, { deleting: true, locked: false })).toBe('Processing')
    expect(proxyDeleteConfirmButtonTitle(t, { deleting: false, locked: true })).toBe('Processing')
  })
})
