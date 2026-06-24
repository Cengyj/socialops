import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import TemplateEditorActions from '../TemplateEditorActions.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'common.delete': 'Delete',
      'common.processing': 'Processing',
      'common.saving': 'Saving',
      'taskSettings.alreadyDefault': 'This template is already the default.',
      'taskSettings.copy': 'Copy',
      'taskSettings.save': 'Save',
      'taskSettings.savedConfigs.selectTemplateFirst': 'Select a saved template first.',
      'taskSettings.setDefault': 'Set default',
      'taskSettings.validate': 'Validate',
    }[key] ?? key),
  }),
}))

function mountActions(props = {}) {
  return mount(TemplateEditorActions, {
    props: {
      canSave: true,
      hasSelectedTemplate: true,
      isDefault: false,
      operation: null,
      saveDisabledReason: '',
      saving: false,
      ...props,
    },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span data-testid="icon-stub" :data-icon="name" />' },
      },
    },
  })
}

describe('TemplateEditorActions', () => {
  it('renders the existing editor actions without adding extra buttons', () => {
    const wrapper = mountActions()

    expect(wrapper.get('[data-testid="editor-template-actions"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="save-template-button"]').text()).toContain('Save')
    expect(wrapper.get('[data-testid="validation-button"]').text()).toContain('Validate')
    expect(wrapper.get('[data-testid="copy-template-button"]').text()).toContain('Copy')
    expect(wrapper.get('[data-testid="set-default-button"]').text()).toContain('Set default')
    expect(wrapper.get('[data-testid="delete-template-button"]').text()).toContain('Delete')
    expect(wrapper.findAll('button')).toHaveLength(5)
    expect(wrapper.findAll('[data-testid="icon-stub"]').map(icon => icon.attributes('data-icon')))
      .toEqual(['check', 'shield', 'copy', 'checkCircle', 'trash'])
  })

  it('keeps pending labels scoped to the current operation', () => {
    const wrapper = mountActions({ operation: 'copy', saving: true })

    expect(wrapper.get('[data-testid="copy-template-button"]').text()).toContain('Processing')
    expect(wrapper.get('[data-testid="copy-template-button"]').attributes('title')).toBe('Processing')
    expect(wrapper.get('[data-testid="save-template-button"]').attributes('title')).toBe('Processing')
    expect(wrapper.get('[data-testid="save-template-button"]').text()).toContain('Save')
    expect(wrapper.get('[data-testid="save-template-button"]').text()).not.toContain('Saving')
    expect(wrapper.get('[data-testid="set-default-button"]').attributes('disabled')).toBeDefined()
  })

  it('keeps validation pending feedback scoped to the existing validate action', () => {
    const wrapper = mountActions({ operation: 'validate', saving: true })

    const validationButton = wrapper.get('[data-testid="validation-button"]')
    expect(validationButton.text()).toContain('Processing')
    expect(validationButton.attributes('aria-label')).toBe('Processing')
    expect(validationButton.attributes('title')).toBe('Processing')
    expect(validationButton.attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="save-template-button"]').text()).toContain('Save')
    expect(wrapper.get('[data-testid="copy-template-button"]').text()).toContain('Copy')
  })

  it('keeps editor action labels inspectable and constrained on narrow layouts', () => {
    const wrapper = mountActions()

    for (const [testId, label, title] of [
      ['save-template-button', 'Save', undefined],
      ['validation-button', 'Validate', 'Validate'],
      ['copy-template-button', 'Copy', 'Copy'],
      ['set-default-button', 'Set default', 'Set default'],
      ['delete-template-button', 'Delete', 'Delete'],
    ] as const) {
      const button = wrapper.get(`[data-testid="${testId}"]`)
      expect(button.attributes('aria-label')).toBe(label)
      expect(button.attributes('title')).toBe(title)
      expect(button.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
      expect(button.get('span.min-w-0.truncate').exists()).toBe(true)
    }
  })

  it('preserves disabled reasons and emits existing action events', async () => {
    const wrapper = mountActions({
      canSave: false,
      hasSelectedTemplate: false,
      saveDisabledReason: 'Name is required',
    })

    expect(wrapper.get('[data-testid="save-template-button"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="save-template-button"]').attributes('title')).toBe('Name is required')
    expect(wrapper.get('[data-testid="copy-template-button"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="copy-template-button"]').attributes('title')).toBe('Select a saved template first.')
    expect(wrapper.get('[data-testid="set-default-button"]').attributes('title')).toBe('Select a saved template first.')
    expect(wrapper.get('[data-testid="delete-template-button"]').attributes('title')).toBe('Select a saved template first.')
    expect(wrapper.get('[data-testid="delete-template-button"]').attributes('aria-label')).toBe('Select a saved template first.')

    await wrapper.setProps({ canSave: true, hasSelectedTemplate: true, saveDisabledReason: '' })
    await wrapper.get('[data-testid="save-template-button"]').trigger('click')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await wrapper.get('[data-testid="copy-template-button"]').trigger('click')
    await wrapper.get('[data-testid="set-default-button"]').trigger('click')
    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')

    expect(wrapper.emitted('save')).toHaveLength(1)
    expect(wrapper.emitted('validate')).toHaveLength(1)
    expect(wrapper.emitted('copy')).toHaveLength(1)
    expect(wrapper.emitted('setDefault')).toHaveLength(1)
    expect(wrapper.emitted('delete')).toHaveLength(1)
  })

  it('explains when the selected template is already the default', () => {
    const wrapper = mountActions({ isDefault: true })
    const setDefaultButton = wrapper.get('[data-testid="set-default-button"]')

    expect(setDefaultButton.attributes('disabled')).toBeDefined()
    expect(setDefaultButton.attributes('title')).toBe('This template is already the default.')
  })

  it('keeps the disabled delete action reason available to assistive labels while saving', () => {
    const wrapper = mountActions({ saving: true })
    const deleteButton = wrapper.get('[data-testid="delete-template-button"]')

    expect(deleteButton.attributes('disabled')).toBeDefined()
    expect(deleteButton.attributes('title')).toBe('Processing')
    expect(deleteButton.attributes('aria-label')).toBe('Processing')
    expect(deleteButton.text()).toContain('Delete')
  })
})
