type Translate = (key: string) => string

const processingKey = 'common.processing'

export function proxyRefreshButtonTitle(t: Translate, state: { loading: boolean }) {
  return state.loading ? t(processingKey) : t('common.refresh')
}

export function proxyTestSelectedButtonTitle(t: Translate, state: { loading: boolean; testing: boolean; selectedCount: number }) {
  if (state.loading || state.testing) return t(processingKey)
  if (state.selectedCount === 0) return t('proxies.selection.noneSelected')
  return t('proxies.testSelected')
}

export function proxyTestAllButtonTitle(t: Translate, state: { loading: boolean; testing: boolean; hasProxies: boolean }) {
  if (state.loading || state.testing) return t(processingKey)
  if (!state.hasProxies) return t('proxies.noProxiesToTest')
  return t('proxies.testAll')
}

export function proxyCreateButtonTitle(t: Translate, state: { testing: boolean }) {
  return state.testing ? t(processingKey) : t('proxies.addProxy')
}

export function proxyClearTestResultsButtonTitle(t: Translate, state: { disabled: boolean; loading?: boolean; testing?: boolean }) {
  if (state.loading) return t('common.loading')
  if (state.testing || state.disabled) return t(processingKey)
  return t('common.clear')
}

export function proxyRowTestButtonTitle(t: Translate, state: { loading: boolean; testing: boolean }) {
  return state.loading || state.testing ? t(processingKey) : t('proxies.test')
}

export function proxyRowEditButtonTitle(t: Translate, state: { loading: boolean; testing: boolean }) {
  return state.loading || state.testing ? t(processingKey) : t('common.edit')
}

export function proxyRowDeleteButtonTitle(t: Translate, state: { loading: boolean; testing: boolean }) {
  return state.loading || state.testing ? t(processingKey) : t('common.delete')
}

export function proxyFormCancelButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t('common.saving') : t('common.cancel')
}

export function proxyFormSubmitButtonTitle(t: Translate, state: { saving: boolean; disabledReason: string }) {
  if (state.saving) return t('common.saving')
  if (state.disabledReason) return state.disabledReason
  return t('common.confirm')
}

export function proxyFormSubmitButtonLabel(t: Translate, state: { saving: boolean }) {
  return state.saving ? t('common.saving') : t('common.confirm')
}

export function proxyDeleteCancelButtonTitle(t: Translate, state: { deleting: boolean }) {
  return state.deleting ? t(processingKey) : t('common.cancel')
}

export function proxyDeleteConfirmButtonTitle(
  t: Translate,
  state: { deleting: boolean; locked: boolean },
) {
  if (state.deleting || state.locked) return t(processingKey)
  return t('common.delete')
}
