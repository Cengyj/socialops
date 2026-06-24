import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import TemplateSummaryPanel, { type TemplateSummaryRow } from '../TemplateSummaryPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) => ({
      'taskSettings.summary.executionHint': 'Accounts execution reads saved templates.',
      'taskSettings.summary.title': 'Execution summary',
      'taskSettings.status.ready': 'Ready',
      'taskSettings.validation.avatarDimensions': `Avatar image must be exactly ${params?.width}x${params?.height} pixels.`,
      'taskSettings.validation.avatarRequired': 'Upload one avatar image before saving this template.',
      'taskSettings.validation.invalid': 'Template has issues',
      'taskSettings.validation.mediaInvalid': 'This image cannot be read. Upload it again before saving.',
      'taskSettings.validation.mediaSourceUnsupported': 'Saved media references are not available for execution yet.',
      'taskSettings.validation.postConfigurationRequired': 'Add post text or at least one media item before saving this template.',
      'taskSettings.validation.postMediaTooMany': `Post templates can contain at most ${params?.max} media items.`,
      'taskSettings.validation.postMediaTypeUnsupported': 'Only image media is supported right now.',
      'taskSettings.validation.postVideoUnavailable': 'Video post media is not supported for execution right now.',
      'taskSettings.validation.profileRequired': 'Add at least one profile field before saving this template.',
      'taskSettings.validation.nameRequired': 'Enter a template name before saving.',
      'taskSettings.validation.targetsRequired': 'Add at least one target before saving this template.',
      'taskSettings.validation.templateRequired': 'Template information is missing. Refresh and try again.',
      'taskSettings.validation.tooManyValues': `A template parameter pool can contain at most ${params?.max} items.`,
      'taskSettings.validation.unsupportedType': 'This task type is not supported. Choose another task type.',
      'taskSettings.validation.valid': 'Template is complete',
      'taskSettings.validation.valueTooLong': `Each template parameter can contain at most ${params?.max} characters.`,
    }[key] ?? key),
  }),
}))

const rows: TemplateSummaryRow[] = [
  { key: 'type', label: 'Type', value: 'Follow' },
  { key: 'targets', label: 'Targets', value: 2 },
  { key: 'default', label: 'Default', value: 'No' },
]

function mountPanel(props = {}) {
  return mount(TemplateSummaryPanel, {
    props: {
      rows,
      saveDisabledReason: '',
      validationResult: null,
      ...props,
    },
  })
}

describe('TemplateSummaryPanel', () => {
  it('renders existing summary rows without adding actions', () => {
    const wrapper = mountPanel()

    expect(wrapper.get('[data-testid="template-summary-panel"]').text()).toContain('Execution summary')
    expect(wrapper.get('[data-testid="template-summary-panel"]').text()).toContain('Type')
    expect(wrapper.get('[data-testid="template-summary-panel"]').text()).toContain('Follow')
    expect(wrapper.get('[data-testid="template-summary-panel"]').text()).toContain('Targets')
    expect(wrapper.get('[data-testid="template-summary-panel"]').text()).toContain('2')
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('keeps long summary values readable without adding actions', () => {
    const longValue = 'stage108_template_summary_value_with_really_long_unbroken_identifier_0123456789abcdef'
    const wrapper = mountPanel({
      rows: [
        { key: 'long', label: 'Current template', value: longValue },
      ],
    })

    const value = wrapper.get('[data-testid="template-summary-panel"] span[title]')
    expect(value.text()).toBe(longValue)
    expect(value.attributes('title')).toBe(longValue)
    expect(value.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words', 'text-right']))
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('renders the existing ready fallback and execution hint before server validation runs', () => {
    const wrapper = mountPanel()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Ready')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('Template is complete')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Accounts execution reads saved templates.')
    const readyFallback = wrapper.get('[title="Ready"]')
    expect(readyFallback.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(readyFallback.attributes('role')).toBe('status')
    expect(readyFallback.attributes('aria-live')).toBe('polite')
    expect(readyFallback.attributes('aria-atomic')).toBe('true')
    const executionHint = wrapper.get('[title="Accounts execution reads saved templates."]')
    expect(executionHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(executionHint.attributes('role')).toBe('status')
    expect(executionHint.attributes('aria-live')).toBe('polite')
    expect(executionHint.attributes('aria-atomic')).toBe('true')
  })

  it('renders save-disabled reasons before validation has run', () => {
    const reason = 'Targets are required'
    const wrapper = mountPanel({ saveDisabledReason: reason })

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Targets are required')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('Template is complete')
    const warning = wrapper.get('[data-testid="template-validation-panel"] div[title]')
    expect(warning.attributes('title')).toBe(reason)
    expect(warning.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(warning.attributes('role')).toBe('status')
    expect(warning.attributes('aria-live')).toBe('polite')
    expect(warning.attributes('aria-atomic')).toBe('true')
  })

  it('renders explicit validation results and errors when present', () => {
    const wrapper = mountPanel({
      saveDisabledReason: 'Targets are required',
      validationResult: {
        valid: false,
        type: 'follow',
        targets: 0,
        contents: 0,
        errors: ['Add at least one target', 'Remove empty rows'],
      },
    })

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template has issues')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Add at least one target')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Remove empty rows')
  })

  it('normalizes validation result errors before display', () => {
    const wrapper = mountPanel({
      validationResult: {
        valid: false,
        type: 'follow',
        targets: 0,
        contents: 0,
        errors: ['  Add at least one target  ', '   ', ' Remove empty rows '],
      },
    })

    const errors = wrapper.findAll('[data-testid="template-validation-panel"] li')
      .map(item => item.text())
    expect(errors).toEqual(['Add at least one target', 'Remove empty rows'])
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('  Add at least one target  ')
  })

  it('keeps long validation errors readable', () => {
    const longError = 'stage108_template_validation_error_with_really_long_unbroken_backend_detail_0123456789abcdef'
    const wrapper = mountPanel({
      validationResult: {
        valid: false,
        type: 'follow',
        targets: 0,
        contents: 0,
        errors: [longError],
      },
    })

    const error = wrapper.get('[data-testid="template-validation-panel"] li')
    expect(error.text()).toBe(longError)
    expect(error.attributes('title')).toBe(longError)
    expect(error.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })

  it('maps known backend validation errors to task-settings friendly messages', () => {
    const wrapper = mountPanel({
      validationResult: {
        valid: false,
        type: 'post',
        targets: 0,
        contents: 0,
        errors: [
          'template name is required',
          'task template name is required',
          'template is required',
          'unsupported task template type',
          'unsupported social task action',
          'target list is required',
          'content pool cannot exceed 500 items',
          'content item cannot exceed 2048 characters',
          'post template requires content pool or media',
          'post media cannot exceed 4 items',
          'video media is not supported for SocialOps execution',
          'post media #1 media source is not supported for SocialOps execution',
          'post media content type is not supported',
          'profile settings are required',
          'avatar media is required',
          'avatar media is invalid',
          'avatar image must be 400x400 pixels',
          'Unmapped backend detail',
        ],
      },
    })

    const panel = wrapper.get('[data-testid="template-validation-panel"]').text()

    expect(panel).toContain('Enter a template name before saving.')
    expect(panel).toContain('Template information is missing. Refresh and try again.')
    expect(panel).toContain('This task type is not supported. Choose another task type.')
    expect(panel).toContain('Add at least one target before saving this template.')
    expect(panel).toContain('A template parameter pool can contain at most 500 items.')
    expect(panel).toContain('Each template parameter can contain at most 2048 characters.')
    expect(panel).toContain('Add post text or at least one media item before saving this template.')
    expect(panel).toContain('Post templates can contain at most 4 media items.')
    expect(panel).toContain('Video post media is not supported for execution right now.')
    expect(panel).toContain('Saved media references are not available for execution yet.')
    expect(panel).toContain('Only image media is supported right now.')
    expect(panel).toContain('Add at least one profile field before saving this template.')
    expect(panel).toContain('Upload one avatar image before saving this template.')
    expect(panel).toContain('This image cannot be read. Upload it again before saving.')
    expect(panel).toContain('Avatar image must be exactly 400x400 pixels.')
    expect(panel).toContain('Unmapped backend detail')
    expect(panel).not.toContain('template name is required')
    expect(panel).not.toContain('task template name is required')
    expect(panel).not.toContain('unsupported task template type')
    expect(panel).not.toContain('unsupported social task action')
    expect(panel).not.toContain('target list is required')
    expect(panel).not.toContain('content pool cannot exceed 500 items')
    expect(panel).not.toContain('post media #1 media source is not supported for SocialOps execution')
    expect(panel).not.toContain('avatar media is invalid')
  })

  it('renders explicit valid validation results', () => {
    const wrapper = mountPanel({
      validationResult: {
        valid: true,
        type: 'follow',
        targets: 1,
        contents: 0,
        errors: [],
      },
    })

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')
    const validationResultPanel = wrapper.get('[data-testid="template-validation-panel"] > div')
    expect(validationResultPanel.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(validationResultPanel.attributes('role')).toBe('status')
    expect(validationResultPanel.attributes('aria-live')).toBe('polite')
    expect(validationResultPanel.attributes('aria-atomic')).toBe('true')
  })
})
