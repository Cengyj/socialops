type Translate = (key: string, params?: Record<string, unknown>) => string

const processingKey = 'common.processing'
const selectedTemplateRequiredKey = 'taskSettings.savedConfigs.selectTemplateFirst'

export type TemplateEditorTitleOperation = 'validate' | 'save' | 'copy' | 'default' | 'delete'

export function templateEditorSaveButtonTitle(
  t: Translate,
  state: { operation: TemplateEditorTitleOperation | null; saveDisabledReason: string; saving: boolean },
) {
  if (state.operation === 'save') return t('common.saving')
  if (state.saveDisabledReason) return state.saveDisabledReason
  if (state.saving) return t(processingKey)
  return undefined
}

export function templateEditorSaveButtonLabel(t: Translate, state: { operation: TemplateEditorTitleOperation | null }) {
  return state.operation === 'save' ? t('common.saving') : t('taskSettings.save')
}

export function templateEditorValidateButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t(processingKey) : t('taskSettings.validate')
}

export function templateEditorValidateButtonLabel(t: Translate, state: { operation: TemplateEditorTitleOperation | null }) {
  return state.operation === 'validate' ? t(processingKey) : t('taskSettings.validate')
}

export function templateEditorCopyButtonTitle(t: Translate, state: { saving: boolean; hasSelectedTemplate: boolean }) {
  if (state.saving) return t(processingKey)
  if (!state.hasSelectedTemplate) return t(selectedTemplateRequiredKey)
  return t('taskSettings.copy')
}

export function templateEditorCopyButtonLabel(t: Translate, state: { operation: TemplateEditorTitleOperation | null }) {
  return state.operation === 'copy' ? t(processingKey) : t('taskSettings.copy')
}

export function templateEditorSetDefaultButtonTitle(
  t: Translate,
  state: { saving: boolean; hasSelectedTemplate: boolean; isDefault: boolean },
) {
  if (state.saving) return t(processingKey)
  if (!state.hasSelectedTemplate) return t(selectedTemplateRequiredKey)
  if (state.isDefault) return t('taskSettings.alreadyDefault')
  return t('taskSettings.setDefault')
}

export function templateEditorSetDefaultButtonLabel(t: Translate, state: { operation: TemplateEditorTitleOperation | null }) {
  return state.operation === 'default' ? t(processingKey) : t('taskSettings.setDefault')
}

export function templateEditorDeleteButtonTitle(t: Translate, state: { saving: boolean; hasSelectedTemplate: boolean }) {
  if (state.saving) return t(processingKey)
  if (!state.hasSelectedTemplate) return t(selectedTemplateRequiredKey)
  return t('common.delete')
}

export function templateDeleteCancelButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t(processingKey) : t('common.cancel')
}

export function templateDeleteConfirmButtonTitle(t: Translate, state: { deleting: boolean }) {
  return state.deleting ? t(processingKey) : t('common.delete')
}

export function templateEditorAddPostMediaButtonTitle(
  t: Translate,
  state: { saving: boolean; mediaCount: number; maxMediaItems: number },
) {
  if (state.saving) return t('common.saving')
  if (state.mediaCount >= state.maxMediaItems) {
    return t('taskSettings.validation.postMediaTooMany', { max: state.maxMediaItems })
  }
  return t('taskSettings.media.addPostImage')
}

export function templateEditorRemovePostMediaButtonTitle(t: Translate, state: { saving: boolean }) {
  return state.saving ? t('common.saving') : t('taskSettings.media.removePostImage')
}
