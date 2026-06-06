import { flushPromises, mount, VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import TaskSettingsView from '../TaskSettingsView.vue'

const { listTemplates, saveTemplate, copyTemplate, setDefaultTemplate, deleteTemplate, showError } = vi.hoisted(() => ({
  listTemplates: vi.fn(),
  saveTemplate: vi.fn(),
  copyTemplate: vi.fn(),
  setDefaultTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/taskSettings', () => ({
  default: {
    listTemplates,
    saveTemplate,
    validateTemplate: vi.fn(),
    copyTemplate,
    setDefaultTemplate,
    deleteTemplate,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.loading': 'Loading',
    'common.retry': 'Retry',
    'common.delete': 'Delete',
    'common.close': 'Close',
    'common.cancel': 'Cancel',
    'common.processing': 'Processing',
    'common.saving': 'Saving',
    'common.all': 'All',
    'taskSettings.templateList': 'Templates',
    'taskSettings.templateListHint': 'Saved execution parameter templates',
    'taskSettings.title': 'Task Settings',
    'taskSettings.description': 'Maintain reusable execution parameter templates for account tasks.',
    'taskSettings.newTemplate': 'New template',
    'taskSettings.empty.title': 'No templates yet',
    'taskSettings.empty.description': 'Create a template before submitting tasks.',
    'taskSettings.editTitle': 'Edit template',
    'taskSettings.createTitle': 'Create template',
    'taskSettings.editorHint': 'Maintain reusable execution parameters.',
    'taskSettings.defaultBadge': 'Default',
    'taskSettings.copy': 'Copy',
    'taskSettings.setDefault': 'Set default',
    'taskSettings.defaultImpact': 'Only one default template is active per task type.',
    'taskSettings.form.name': 'Name',
    'taskSettings.form.namePlaceholder': 'Template name',
    'taskSettings.form.type': 'Type',
    'taskSettings.form.targets': 'Targets',
    'taskSettings.form.contents': 'Contents',
    'taskSettings.form.contentsPlaceholder': 'One content per line',
    'taskSettings.form.followTargetsPlaceholder': 'One target per line',
    'taskSettings.form.tweetTargetsPlaceholder': 'One tweet per line',
    'taskSettings.importFile': 'Import',
    'taskSettings.viewAll': 'View all',
    'taskSettings.clearValues': 'Clear',
    'taskSettings.dedupe': 'Deduplicate',
    'taskSettings.pool.title': 'Parameter Pool',
    'taskSettings.pool.valid': 'Valid',
    'taskSettings.pool.emptyLines': 'Empty lines',
    'taskSettings.pool.duplicates': 'Duplicates',
    'taskSettings.pool.tooLong': 'Too long',
    'taskSettings.pool.remaining': 'Remaining',
    'taskSettings.pool.emptyLinesHint': '{count} empty line(s) will be ignored before saving.',
    'taskSettings.pool.duplicateHint': '{count} duplicate value(s) detected. You can deduplicate before saving.',
    'taskSettings.summary.title': 'Summary',
    'taskSettings.summary.type': 'Type',
    'taskSettings.summary.targets': 'Targets',
    'taskSettings.summary.contents': 'Contents',
    'taskSettings.summary.executionHint': 'Selected templates are submitted with task requests.',
    'taskSettings.media.postTextOnlyHint': 'Post templates currently configure text content only. Image or video attachments are not connected to this template.',
    'taskSettings.stats.total': 'Total',
    'taskSettings.stats.totalMeta': 'Saved {type} templates',
    'taskSettings.stats.defaults': 'Defaults',
    'taskSettings.stats.defaultsMeta': 'Default for current {type} flow',
    'taskSettings.stats.unusable': 'Unusable',
    'taskSettings.stats.unusableMeta': 'Current {type} templates needing attention',
    'taskSettings.savedConfigs.title': 'Saved configs',
    'taskSettings.savedConfigs.description': 'Pick an existing config for the current task type.',
    'taskSettings.savedConfigs.emptyTitle': 'No saved config for this type',
    'taskSettings.savedConfigs.emptyDescription': 'Fill the current settings and save it.',
    'taskSettings.savedConfigs.newForType': 'New {type} config',
    'taskSettings.savedConfigs.allTypesHint': 'All task types are listed here.',
    'taskSettings.savedConfigs.unsavedTitle': 'Unsaved template',
    'taskSettings.savedConfigs.unsavedDescription': 'Save it before account execution can use it.',
    'taskSettings.savedConfigs.readyState': 'Ready',
    'taskSettings.savedConfigs.needsInputState': 'Needs input',
    'taskSettings.filters.all': 'All types',
    'taskSettings.validate': 'Validate',
    'taskSettings.save': 'Save',
    'taskSettings.types.follow': 'Follow',
    'taskSettings.types.like': 'Like',
    'taskSettings.types.retweet': 'Retweet',
    'taskSettings.types.post': 'Post',
    'taskSettings.typeDescriptions.follow': 'Follow target users.',
    'taskSettings.typeDescriptions.like': 'Like tweets.',
    'taskSettings.typeDescriptions.retweet': 'Retweet tweets.',
    'taskSettings.typeDescriptions.post': 'Publish text content.',
    'taskSettings.typeRequirements.follow': 'Targets: user ID, username, or profile URL',
    'taskSettings.typeRequirements.like': 'Targets: Tweet URL or Tweet ID',
    'taskSettings.typeRequirements.retweet': 'Targets: Tweet URL or Tweet ID',
    'taskSettings.typeRequirements.post': 'Contents: one post text per line',
    'taskSettings.validation.nameRequired': 'Name is required',
    'taskSettings.validation.targetsRequired': 'Targets are required',
    'taskSettings.validation.contentsRequired': 'Contents are required',
    'taskSettings.validation.tooManyValues': 'Too many values: max {max}',
    'taskSettings.validation.valueTooLong': 'Value is too long: max {max}',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, paramsOrFallback?: Record<string, string | number> | string) => {
        const params = typeof paramsOrFallback === 'object' ? paramsOrFallback : {}
        const fallback = typeof paramsOrFallback === 'string' ? paramsOrFallback : key
        return (messages[key] ?? fallback).replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? `{${name}}`))
      },
    }),
  }
})

function mountView() {
  return mount(TaskSettingsView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        BaseDialog: { props: ['show', 'title'], template: '<section v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>' },
        Icon: true,
      },
    },
  })
}

async function mountLoadedView(templates = []) {
  listTemplates.mockResolvedValue(templates)
  const wrapper = mountView()
  await flushPromises()
  return wrapper
}

async function chooseType(wrapper: VueWrapper, type: string) {
  await wrapper.get(`[data-testid="task-type-${type}"]`).trigger('click')
}

async function enterName(wrapper: VueWrapper, name = 'Reusable template') {
  await wrapper.get('[data-testid="template-name-input"]').setValue(name)
}

function saveButton(wrapper: VueWrapper) {
  return wrapper.get('[data-testid="save-template-button"]')
}

describe('TaskSettingsView', () => {
  beforeEach(() => {
    listTemplates.mockReset()
    saveTemplate.mockReset()
    copyTemplate.mockReset()
    setDefaultTemplate.mockReset()
    deleteTemplate.mockReset()
    showError.mockReset()
  })

  it('treats a null template list response as an empty list instead of blanking the page', async () => {
    listTemplates.mockResolvedValue(null)

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('h1').exists()).toBe(false)
    expect(wrapper.find('[data-testid="editor-template-actions"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="task-type-follow"]').text()).toContain('Follow')
    expect(wrapper.text()).toContain('No templates yet')
    expect(showError).not.toHaveBeenCalled()
  })

  it('keeps save and validation in the editor header without duplicating new or bottom save actions', async () => {
    const wrapper = await mountLoadedView()
    const editorActions = wrapper.get('[data-testid="editor-template-actions"]')

    expect(wrapper.find('h1').exists()).toBe(false)
    expect(editorActions.text()).not.toContain('New template')
    expect(editorActions.get('[data-testid="validation-button"]').text()).toContain('Validate')
    expect(editorActions.get('[data-testid="save-template-button"]').text()).toContain('Save')
    expect(wrapper.find('[data-testid="save-template-button-secondary"]').exists()).toBe(false)
  })

  it('filters login checks out of task-settings templates', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-login',
        name: 'Login readiness',
        type: 'login_check',
        params: { targets: ['ignored'], contents: ['ignored'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-follow',
        name: 'Follow operators',
        type: 'follow',
        params: { targets: ['@openai'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.find('[data-testid="task-type-login_check"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-login"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="login-check-zero-params"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Follow operators')
    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Targets')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@openai')
  })

  it('saves follow templates with target params only', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-follow-new',
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@openai'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@openai')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@openai'] },
      is_default: true,
    })
  })

  it.each(['follow', 'like', 'retweet'] as const)('shows a targets pool and requires a target for %s templates', async (type) => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, type)
    await enterName(wrapper)

    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Targets')
    expect(wrapper.find('[data-testid="target-pool-textarea"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="content-pool-textarea"]').exists()).toBe(false)
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Targets are required')
  })

  it('shows a contents pool and requires content for post templates', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper)

    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Contents')
    expect(wrapper.get('[data-testid="post-text-only-hint"]').text()).toContain('text content only')
    expect(wrapper.find('[data-testid="content-pool-textarea"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="target-pool-textarea"]').exists()).toBe(false)
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Contents are required')
  })

  it('blocks saving when a parameter pool exceeds 500 values', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper)
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue(
      Array.from({ length: 501 }, (_, index) => `target-${index + 1}`).join('\n'),
    )

    expect(wrapper.get('[data-testid="pool-capacity"]').text()).toContain('Too many values')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Too many values')
  })

  it('blocks saving when any single parameter exceeds 2048 characters', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper)
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('x'.repeat(2049))

    expect(wrapper.get('[data-testid="pool-too-long"]').text()).toContain('1')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Value is too long')
  })

  it('shows ignored blank lines before saving pasted parameter pools', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper)
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@openai\n\n@socialops\n   ')

    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="pool-empty-lines"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').text()).toContain('2 empty line(s) will be ignored')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('identifies duplicate parameter values before saving', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'like')
    await enterName(wrapper)
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('https://x.com/post/1\nhttps://x.com/post/1\nhttps://x.com/post/2')

    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('3')
    expect(wrapper.get('[data-testid="pool-duplicates"]').text()).toContain('1')
    expect(wrapper.text()).toContain('1 duplicate value(s) detected')
    expect(wrapper.get('[data-testid="dedupe-button"]').text()).toContain('Deduplicate')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('keeps copy, delete, and set-default actions available for saved templates', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow',
        name: 'Follow batch',
        type: 'follow',
        params: { targets: ['openai'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.find('[data-testid="copy-template-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="set-default-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="delete-template-button"]').exists()).toBe(true)
  })

  it('scopes the saved template list to the currently selected task type', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow',
        name: 'Follow operators',
        type: 'follow',
        params: { targets: ['@openai'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-post-empty',
        name: 'Post draft pool',
        type: 'post',
        params: { contents: [] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    const cards = wrapper.findAll('[data-template-card="saved"]')
    expect(cards).toHaveLength(1)
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('Saved Follow templates')
    expect(wrapper.get('[data-testid="template-stats-defaults"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Ready')
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-post-empty"]').exists()).toBe(false)

    await wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').trigger('click')

    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Targets')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@openai')

    await chooseType(wrapper, 'post')

    expect(wrapper.findAll('[data-template-card="saved"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-follow"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-empty"]').text()).toContain('Needs input')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('Saved Post templates')
    expect(wrapper.get('[data-testid="template-stats-unusable"]').text()).toContain('1')

    await chooseType(wrapper, 'retweet')

    expect(wrapper.findAll('[data-template-card="saved"]')).toHaveLength(0)
    expect(wrapper.get('[data-testid="active-type-empty-state"]').text()).toContain('No saved config for this type')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('0')
  })
})
