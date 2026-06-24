import { describe, expect, it } from 'vitest'

import {
  templateDeleteCancelButtonTitle,
  templateDeleteConfirmButtonTitle,
  templateEditorAddPostMediaButtonTitle,
  templateEditorCopyButtonLabel,
  templateEditorCopyButtonTitle,
  templateEditorDeleteButtonTitle,
  templateEditorRemovePostMediaButtonTitle,
  templateEditorSaveButtonLabel,
  templateEditorSaveButtonTitle,
  templateEditorSetDefaultButtonLabel,
  templateEditorSetDefaultButtonTitle,
  templateEditorValidateButtonLabel,
  templateEditorValidateButtonTitle,
} from '../templateEditorActionTitles'

const messages: Record<string, string> = {
  'common.cancel': 'Cancel',
  'common.delete': 'Delete',
  'common.processing': 'Processing',
  'common.saving': 'Saving',
  'taskSettings.alreadyDefault': 'This template is already the default.',
  'taskSettings.copy': 'Copy',
  'taskSettings.media.addPostImage': 'Add media',
  'taskSettings.media.removePostImage': 'Remove media',
  'taskSettings.save': 'Save',
  'taskSettings.savedConfigs.selectTemplateFirst': 'Select a saved template first.',
  'taskSettings.setDefault': 'Set default',
  'taskSettings.validation.postMediaTooMany': 'Post templates can contain at most {max} media items.',
  'taskSettings.validate': 'Validate',
}

function t(key: string, params?: Record<string, unknown>) {
  const message = messages[key] ?? key
  return message.replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`))
}

describe('template editor action titles', () => {
  it('keeps idle editor action titles aligned with existing actions', () => {
    expect(templateEditorSaveButtonTitle(t, { operation: null, saveDisabledReason: '', saving: false })).toBeUndefined()
    expect(templateEditorSaveButtonLabel(t, { operation: null })).toBe('Save')
    expect(templateEditorValidateButtonTitle(t, { saving: false })).toBe('Validate')
    expect(templateEditorValidateButtonLabel(t, { operation: null })).toBe('Validate')
    expect(templateEditorCopyButtonTitle(t, { saving: false, hasSelectedTemplate: true })).toBe('Copy')
    expect(templateEditorCopyButtonLabel(t, { operation: null })).toBe('Copy')
    expect(templateEditorSetDefaultButtonTitle(t, { saving: false, hasSelectedTemplate: true, isDefault: false })).toBe('Set default')
    expect(templateEditorSetDefaultButtonLabel(t, { operation: null })).toBe('Set default')
    expect(templateEditorDeleteButtonTitle(t, { saving: false, hasSelectedTemplate: true })).toBe('Delete')
    expect(templateDeleteConfirmButtonTitle(t, { deleting: false })).toBe('Delete')
    expect(templateEditorAddPostMediaButtonTitle(t, { saving: false, mediaCount: 1, maxMediaItems: 4 })).toBe('Add media')
    expect(templateEditorRemovePostMediaButtonTitle(t, { saving: false })).toBe('Remove media')
  })

  it('keeps save title priority aligned with current editor state', () => {
    expect(templateEditorSaveButtonTitle(t, { operation: 'save', saveDisabledReason: 'Name is required', saving: true })).toBe('Saving')
    expect(templateEditorSaveButtonTitle(t, { operation: null, saveDisabledReason: 'Name is required', saving: true })).toBe('Name is required')
    expect(templateEditorSaveButtonTitle(t, { operation: null, saveDisabledReason: '', saving: true })).toBe('Processing')
    expect(templateEditorSaveButtonLabel(t, { operation: 'save' })).toBe('Saving')
    expect(templateEditorValidateButtonLabel(t, { operation: 'validate' })).toBe('Processing')
    expect(templateEditorCopyButtonLabel(t, { operation: 'copy' })).toBe('Processing')
    expect(templateEditorSetDefaultButtonLabel(t, { operation: 'default' })).toBe('Processing')
  })

  it('explains unavailable selected-template actions without changing availability rules', () => {
    expect(templateEditorCopyButtonTitle(t, { saving: false, hasSelectedTemplate: false })).toBe('Select a saved template first.')
    expect(templateEditorSetDefaultButtonTitle(t, { saving: false, hasSelectedTemplate: false, isDefault: false })).toBe('Select a saved template first.')
    expect(templateEditorDeleteButtonTitle(t, { saving: false, hasSelectedTemplate: false })).toBe('Select a saved template first.')
    expect(templateEditorSetDefaultButtonTitle(t, { saving: false, hasSelectedTemplate: true, isDefault: true })).toBe('This template is already the default.')
  })

  it('prioritizes busy titles for actions locked during saving', () => {
    expect(templateEditorValidateButtonTitle(t, { saving: true })).toBe('Processing')
    expect(templateEditorCopyButtonTitle(t, { saving: true, hasSelectedTemplate: false })).toBe('Processing')
    expect(templateEditorSetDefaultButtonTitle(t, { saving: true, hasSelectedTemplate: false, isDefault: true })).toBe('Processing')
    expect(templateEditorDeleteButtonTitle(t, { saving: true, hasSelectedTemplate: false })).toBe('Processing')
    expect(templateDeleteCancelButtonTitle(t, { saving: true })).toBe('Processing')
    expect(templateDeleteConfirmButtonTitle(t, { deleting: true })).toBe('Processing')
    expect(templateEditorAddPostMediaButtonTitle(t, { saving: true, mediaCount: 4, maxMediaItems: 4 })).toBe('Saving')
    expect(templateEditorRemovePostMediaButtonTitle(t, { saving: true })).toBe('Saving')
  })

  it('keeps delete confirmation dialog titles aligned with the existing idle labels', () => {
    expect(templateDeleteCancelButtonTitle(t, { saving: false })).toBe('Cancel')
    expect(templateDeleteConfirmButtonTitle(t, { deleting: false })).toBe('Delete')
  })

  it('keeps post media add title aligned with the existing capacity limit', () => {
    expect(templateEditorAddPostMediaButtonTitle(t, { saving: false, mediaCount: 4, maxMediaItems: 4 }))
      .toBe('Post templates can contain at most 4 media items.')
  })
})
