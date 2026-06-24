type Translate = (key: string) => string

const savingKey = 'common.saving'
const poolEmptyKey = 'taskSettings.pool.empty'

export function parameterPoolImportButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t(savingKey) : t('taskSettings.importFile')
}

export function parameterPoolViewAllButtonTitle(t: Translate, state: { saving: boolean; valueCount: number }) {
  if (state.saving) return t(savingKey)
  if (state.valueCount === 0) return t(poolEmptyKey)
  return t('taskSettings.viewAll')
}

export function parameterPoolDedupeButtonTitle(t: Translate, state: { saving: boolean; duplicateCount: number }) {
  if (state.saving) return t(savingKey)
  if (state.duplicateCount === 0) return t('taskSettings.pool.noDuplicates')
  return t('taskSettings.dedupe')
}

export function parameterPoolClearButtonTitle(t: Translate, state: { saving: boolean; canClear: boolean }) {
  if (state.saving) return t(savingKey)
  if (!state.canClear) return t(poolEmptyKey)
  return t('taskSettings.clearValues')
}
