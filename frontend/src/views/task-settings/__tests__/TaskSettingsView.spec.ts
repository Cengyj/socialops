import { defineComponent, nextTick } from 'vue'
import { flushPromises, mount, VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import TaskSettingsView from '../TaskSettingsView.vue'

const { listTemplates, saveTemplate, validateTemplate, copyTemplate, setDefaultTemplate, deleteTemplate, previewMedia, showError, showSuccess, showWarning, recordClientDiagnostic } = vi.hoisted(() => ({
  listTemplates: vi.fn(),
  saveTemplate: vi.fn(),
  validateTemplate: vi.fn(),
  copyTemplate: vi.fn(),
  setDefaultTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  previewMedia: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

let imageUploadValue = 'data:image/png;base64,QUJD'

vi.mock('@/api/taskSettings', () => ({
  default: {
    listTemplates,
    saveTemplate,
    validateTemplate,
    copyTemplate,
    setDefaultTemplate,
    deleteTemplate,
    previewMedia,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning,
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic,
}))

const ImageUploadStub = defineComponent({
  props: {
    modelValue: { type: String, default: '' },
    previewSrc: { type: String, default: '' },
    hasValue: { type: Boolean, default: undefined },
    maxSize: { type: Number, default: undefined },
    uploadLabel: { type: String, default: '' },
    removeLabel: { type: String, default: '' },
    hint: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return {
      emitSet: () => {
        if (!props.disabled) emit('update:modelValue', imageUploadValue)
      },
      emitClear: () => {
        if (!props.disabled) emit('update:modelValue', '')
      },
    }
  },
  template: `
    <div
      data-testid="image-upload-stub"
      :data-upload-label="uploadLabel"
      :data-remove-label="removeLabel"
      :data-hint="hint"
      :data-value="modelValue"
      :data-preview-src="previewSrc"
      :data-has-value="String(hasValue)"
      :data-max-size="String(maxSize)"
      :data-disabled="String(disabled)"
    >
      <button type="button" data-testid="image-upload-set" :disabled="disabled" @click="emitSet">set</button>
      <button type="button" data-testid="image-upload-clear" :disabled="disabled" @click="emitClear">clear</button>
    </div>
  `,
})

const originalImage = globalThis.Image
const originalCreateObjectURL = globalThis.URL.createObjectURL
const originalRevokeObjectURL = globalThis.URL.revokeObjectURL

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
    'taskSettings.failedToLoad': 'Failed to load task templates',
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
    'taskSettings.deleteDialog.title': 'Delete template',
    'taskSettings.deleteDialog.description': 'Delete template {name}? This action cannot be undone.',
    'taskSettings.deleteDialog.warning': 'Accounts can only execute saved templates. Delete this template only if it is no longer used for task submissions.',
    'taskSettings.form.name': 'Name',
    'taskSettings.form.namePlaceholder': 'Template name',
    'taskSettings.form.type': 'Type',
    'taskSettings.form.targets': 'Targets',
    'taskSettings.form.contents': 'Contents',
    'taskSettings.form.quotePostUrl': 'Quote URL',
    'taskSettings.form.quotePostUrlPlaceholder': 'https://x.com/example/status/123',
    'taskSettings.form.profileDisplayName': 'Display name',
    'taskSettings.form.profileDisplayNamePlaceholder': 'Northwind Ops',
    'taskSettings.form.profileScreenName': 'Screen name',
    'taskSettings.form.profileScreenNamePlaceholder': 'northwind_ops',
    'taskSettings.form.profileDescription': 'Description',
    'taskSettings.form.profileDescriptionPlaceholder': 'Operator account',
    'taskSettings.form.profileLocation': 'Location',
    'taskSettings.form.profileLocationPlaceholder': 'Singapore',
    'taskSettings.form.profileUrl': 'Website',
    'taskSettings.form.profileUrlPlaceholder': 'https://example.com',
    'taskSettings.form.contentsPlaceholder': 'One content per line',
    'taskSettings.form.followTargetsPlaceholder': 'One target per line',
    'taskSettings.form.postTargetsPlaceholder': 'One target post per line',
    'taskSettings.importFile': 'Import',
    'taskSettings.imported': 'Imported {count} item(s).',
    'taskSettings.importEmpty': 'No usable values were found in the selected file.',
    'taskSettings.importFailed': 'Failed to import the selected file. Choose a readable text file and try again.',
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
    'taskSettings.pool.empty': 'No values in this pool.',
    'taskSettings.pool.noDuplicates': 'No duplicate values to remove.',
    'taskSettings.summary.title': 'Summary',
    'taskSettings.summary.type': 'Type',
    'taskSettings.summary.targets': 'Targets',
    'taskSettings.summary.contents': 'Contents',
    'taskSettings.summary.profileFields': 'Profile fields',
    'taskSettings.summary.quotePost': 'Quote link',
    'taskSettings.summary.media': 'Media',
    'taskSettings.summary.executionHint': 'Selected templates are submitted with task requests.',
    'taskSettings.media.postImages': 'Post media',
    'taskSettings.media.postImagesHint': 'Attach up to 4 images for post templates.',
    'taskSettings.media.postImagesEmpty': 'No post media attached yet.',
    'taskSettings.media.postImageCount': '{count} / {max} media item(s) used',
    'taskSettings.media.postImageItem': 'Media {index}',
    'taskSettings.media.addPostImage': 'Add media',
    'taskSettings.media.uploadPostImage': 'Upload media',
    'taskSettings.media.removePostImage': 'Remove media',
    'taskSettings.media.avatarHint': 'Upload the avatar image to apply during execution.',
    'taskSettings.media.bannerHint': 'Upload the banner image to apply during execution.',
    'taskSettings.media.uploadAvatar': 'Upload avatar',
    'taskSettings.media.removeAvatar': 'Remove avatar',
    'taskSettings.media.uploadBanner': 'Upload banner',
    'taskSettings.media.removeBanner': 'Remove banner',
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
    'taskSettings.status.ready': 'Ready',
    'taskSettings.filters.all': 'All types',
    'taskSettings.validate': 'Validate',
    'taskSettings.saveFailed': 'Failed to save template',
    'taskSettings.copyFailed': 'Failed to copy template',
    'taskSettings.deleteFailed': 'Failed to delete template',
    'taskSettings.errors.serviceUnavailable': 'Task template service is temporarily unavailable. Try again later.',
    'taskSettings.errors.templateMissing': 'Template information is missing. Refresh and try again.',
    'taskSettings.errors.templateInvalid': 'The template is incomplete. Fix the validation issues and try again.',
    'taskSettings.errors.templateNotFound': 'The template no longer exists or was deleted. Refresh the list and try again.',
    'taskSettings.errors.storeInvalid': 'Template data cannot be read right now. Refresh and try again.',
    'taskSettings.errors.mediaServiceUnavailable': 'Template media service is temporarily unavailable. Try again later.',
    'taskSettings.errors.mediaMissing': 'Template media information is missing. Upload it again and retry.',
    'taskSettings.validation.unsupportedType': 'This task type is not supported. Choose another task type.',
    'taskSettings.save': 'Save',
    'taskSettings.types.follow': 'Follow',
    'taskSettings.types.like': 'Like',
    'taskSettings.types.retweet': 'Retweet',
    'taskSettings.types.post': 'Post',
    'taskSettings.types.update_profile': 'Update profile',
    'taskSettings.types.update_avatar': 'Update avatar',
    'taskSettings.types.update_banner': 'Update banner',
    'taskSettings.typeDescriptions.follow': 'Follow target users.',
    'taskSettings.typeDescriptions.like': 'Like tweets.',
    'taskSettings.typeDescriptions.retweet': 'Retweet tweets.',
    'taskSettings.typeDescriptions.post': 'Publish text content.',
    'taskSettings.typeDescriptions.update_profile': 'Update profile metadata such as name, bio, location, and website.',
    'taskSettings.typeDescriptions.update_avatar': 'Refresh the profile avatar with a prepared image.',
    'taskSettings.typeDescriptions.update_banner': 'Refresh the profile banner with a prepared image.',
    'taskSettings.typeRequirements.follow': 'Targets: user ID, username, or profile URL',
    'taskSettings.typeRequirements.like': 'Targets: post URL or post ID',
    'taskSettings.typeRequirements.retweet': 'Targets: post URL or post ID',
    'taskSettings.typeRequirements.post': 'Contents: one post text per line',
    'taskSettings.typeRequirements.update_profile': 'Profile fields: at least one field is required',
    'taskSettings.typeRequirements.update_avatar': 'Avatar: upload one image',
    'taskSettings.typeRequirements.update_banner': 'Banner: upload one image',
    'taskSettings.validation.nameRequired': 'Name is required',
    'taskSettings.validation.targetsRequired': 'Targets are required',
    'taskSettings.validation.valid': 'Template is complete',
    'taskSettings.validation.invalid': 'Template has issues',
  'taskSettings.validation.postConfigurationRequired': 'Add post text or at least one media item before saving.',
    'taskSettings.validation.postMediaTooMany': 'Post templates can contain at most {max} media items.',
    'taskSettings.validation.postVideoUnavailable': 'Video post media is not supported for execution right now',
    'taskSettings.validation.postMediaTypeUnsupported': 'Only image media is supported right now',
    'taskSettings.validation.mediaSourceUnsupported': 'Saved media references are not available for execution yet.',
    'taskSettings.validation.profileRequired': 'Profile settings are required',
    'taskSettings.validation.avatarRequired': 'Avatar media is required',
    'taskSettings.validation.bannerRequired': 'Banner media is required',
    'taskSettings.validation.avatarDimensions': 'Avatar image must be exactly {width}x{height} pixels',
    'taskSettings.validation.bannerDimensions': 'Banner image must be exactly {width}x{height} pixels',
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
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<section v-if="show" role="dialog"><button type="button" aria-label="Close modal" @click="$emit(\'close\')"></button><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
        },
        Icon: true,
        ImageUpload: ImageUploadStub,
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

async function clickLatestDialogClose(wrapper: ReturnType<typeof mount>) {
  const closeButtons = wrapper.findAll('button[aria-label="Close modal"]')
  expect(closeButtons.length, 'dialog close buttons').toBeGreaterThan(0)
  await closeButtons[closeButtons.length - 1].trigger('click')
}

function paragraphWithText(wrapper: VueWrapper, text: string) {
  const paragraph = wrapper.findAll('p').find(node => node.text() === text)
  if (!paragraph) {
    throw new Error(`Expected paragraph with text: ${text}`)
  }
  return paragraph
}

function installImageDimensionMock(dimensions: Array<{ width: number; height: number }>) {
  class MockImage {
    naturalWidth = 0
    naturalHeight = 0
    onload: (() => void) | null = null
    onerror: (() => void) | null = null

    set src(_value: string) {
      const next = dimensions.shift() ?? { width: 0, height: 0 }
      this.naturalWidth = next.width
      this.naturalHeight = next.height
      this.onload?.()
    }
  }

  vi.stubGlobal('Image', MockImage)
}

describe('TaskSettingsView', () => {
  beforeEach(() => {
    listTemplates.mockReset()
    saveTemplate.mockReset()
    validateTemplate.mockReset()
    copyTemplate.mockReset()
    setDefaultTemplate.mockReset()
    deleteTemplate.mockReset()
    previewMedia.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    recordClientDiagnostic.mockReset()
    imageUploadValue = 'data:image/png;base64,QUJD'
    previewMedia.mockResolvedValue(new Blob(['preview'], { type: 'image/png' }))
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(() => 'blob:task-media-preview'),
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    globalThis.Image = originalImage
    globalThis.URL.createObjectURL = originalCreateObjectURL
    globalThis.URL.revokeObjectURL = originalRevokeObjectURL
  })

  it('keeps task-template load errors readable in the existing retry panel', async () => {
    listTemplates.mockRejectedValue({})

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load task templates')
    expect(wrapper.text().match(/Failed to load task templates/g)).toHaveLength(1)
    expect(wrapper.find('p[title="Failed to load task templates"]').exists()).toBe(false)
    const retryButton = wrapper.get('button[aria-label="Retry"]')
    expect(retryButton.attributes('title')).toBe('Retry')
    expect(retryButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'shrink-0', 'justify-center']))
    expect(retryButton.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))

    listTemplates.mockResolvedValueOnce([])
    await retryButton.trigger('click')
    await flushPromises()

    expect(listTemplates).toHaveBeenCalledTimes(2)
    expect(wrapper.find('p[title="Failed to load task templates"]').exists()).toBe(false)
  })

  it('maps task-template service availability load errors to the existing retry panel', async () => {
    const loadErrorMessage = 'Task template service is temporarily unavailable. Try again later.'
    listTemplates.mockRejectedValue({ reason: 'TASK_TEMPLATE_SERVICE_UNAVAILABLE', message: 'task template service is unavailable' })

    const wrapper = mountView()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(loadErrorMessage)
    expect(wrapper.text()).toContain('Failed to load task templates')
    const errorMessage = wrapper.findAll('p').find(node => node.text() === loadErrorMessage)
    expect(errorMessage).toBeTruthy()
    expect(errorMessage!.attributes('title')).toBe(loadErrorMessage)
  })

  it('keeps long parameter values and footer actions readable in the existing view-all dialog', async () => {
    const longValue = 'stage119_parameter_value_with_really_long_unbroken_identifier_0123456789abcdef'
    const wrapper = await mountLoadedView()

    await wrapper.get('[data-testid="target-pool-textarea"]').setValue(longValue)
    await wrapper.get('[data-testid="view-all-button"]').trigger('click')
    await flushPromises()

    const dialog = wrapper.get('section[role="dialog"]')
    const value = dialog.get(`span[title="${longValue}"]`)
    expect(value.text()).toBe(longValue)
    expect(value.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-all']))

    const closeButton = dialog.get('button[aria-label="Close"]')
    expect(closeButton.attributes('title')).toBe('Close')
    expect(closeButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(closeButton.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))

    await closeButton.trigger('click')
    expect(wrapper.find('section[role="dialog"]').exists()).toBe(false)
  })

  it('keeps media section hints readable and inspectable', async () => {
    const wrapper = await mountLoadedView()
    const cases = [
      { type: 'post', hint: 'Attach up to 4 images for post templates.' },
      { type: 'update_avatar', hint: 'Upload the avatar image to apply during execution.' },
      { type: 'update_banner', hint: 'Upload the banner image to apply during execution.' },
    ]

    for (const item of cases) {
      await chooseType(wrapper, item.type)
      const hint = paragraphWithText(wrapper, item.hint)

      expect(hint.attributes('title')).toBe(item.hint)
      expect(hint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    }
  })

  it('keeps active requirement hints readable and inspectable', async () => {
    const wrapper = await mountLoadedView()
    const cases = [
      { type: 'follow', hint: 'Targets: user ID, username, or profile URL' },
      { type: 'update_profile', hint: 'Profile fields: at least one field is required' },
    ]

    for (const item of cases) {
      await chooseType(wrapper, item.type)
      const hints = wrapper.findAll('p')
        .filter(node => node.text() === item.hint && node.attributes('title') === item.hint)

      expect(hints.length).toBeGreaterThan(0)
      expect(hints[0].classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    }
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

  it('ignores stale validation responses after editable fields change', async () => {
    let resolveValidation: (value: unknown) => void = () => {}
    validateTemplate.mockImplementationOnce(() => new Promise(resolve => {
      resolveValidation = resolve
    }))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@before-edit')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('')

    resolveValidation({
      valid: true,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: [],
    })
    await flushPromises()

    expect(validateTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Validation target',
      type: 'follow',
      params: { targets: ['@before-edit'] },
      is_default: true,
    })
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Targets are required')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('taskSettings.validation.valid')
    expect(showSuccess).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
  })

  it('returns to the ready fallback instead of showing stale valid validation after editable fields change', async () => {
    validateTemplate.mockResolvedValueOnce({
      valid: true,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: [],
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')

    await enterName(wrapper, 'Validation target edited')

    const panelText = wrapper.get('[data-testid="template-validation-panel"]').text()
    expect(panelText).toContain('Ready')
    expect(panelText).not.toContain('Template is complete')
  })

  it('uses a warning toast when validation completes with template issues', async () => {
    validateTemplate.mockResolvedValueOnce({
      valid: false,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: ['server-only validation issue'],
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('server-only validation issue')
    expect(showWarning).toHaveBeenCalledWith('Template has issues')
    expect(showSuccess).not.toHaveBeenCalledWith('Template has issues')
    expect(showError).not.toHaveBeenCalled()
  })

  it('shows validation as processing and avoids duplicate validation requests', async () => {
    let resolveValidation: (value: unknown) => void = () => {}
    validateTemplate.mockImplementationOnce(() => new Promise(resolve => {
      resolveValidation = resolve
    }))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await wrapper.vm.$nextTick()

    const pendingValidationButton = wrapper.get('[data-testid="validation-button"]')
    expect(validateTemplate).toHaveBeenCalledTimes(1)
    expect(pendingValidationButton.text()).toContain('Processing')
    expect(pendingValidationButton.attributes('aria-label')).toBe('Processing')
    expect(pendingValidationButton.attributes('title')).toBe('Processing')
    expect(pendingValidationButton.attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()

    await pendingValidationButton.trigger('click')
    expect(validateTemplate).toHaveBeenCalledTimes(1)

    resolveValidation({
      valid: true,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: [],
    })
    await flushPromises()

    const idleValidationButton = wrapper.get('[data-testid="validation-button"]')
    expect(idleValidationButton.text()).toContain('Validate')
    expect(idleValidationButton.attributes('aria-label')).toBe('Validate')
    expect(idleValidationButton.attributes('title')).toBe('Validate')
    expect(idleValidationButton.attributes('disabled')).toBeUndefined()
  })

  it('clears previous validation details when starting a new validation request', async () => {
    let resolveValidation: (value: unknown) => void = () => {}
    validateTemplate
      .mockResolvedValueOnce({
        valid: false,
        type: 'follow',
        targets: 1,
        contents: 0,
        errors: ['server-only validation issue'],
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveValidation = resolve
      }))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('server-only validation issue')

    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(validateTemplate).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('server-only validation issue')

    resolveValidation({
      valid: true,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: [],
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="validation-button"]').text()).toContain('Validate')
  })

  it('locks parameter-pool actions while a template operation is pending', async () => {
    let resolveValidation: (value: unknown) => void = () => {}
    validateTemplate.mockImplementationOnce(() => new Promise(resolve => {
      resolveValidation = resolve
    }))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Validation target')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind\n@northwind\n@southwind')
    await wrapper.vm.$nextTick()

    for (const testId of ['import-button', 'view-all-button', 'dedupe-button', 'clear-pool-button']) {
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('disabled')).toBeUndefined()
    }
    expect(wrapper.find('input[type="file"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await wrapper.vm.$nextTick()

    for (const testId of ['import-button', 'view-all-button', 'dedupe-button', 'clear-pool-button']) {
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('disabled')).toBeDefined()
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('aria-label')).toBe('Saving')
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('title')).toBe('Saving')
    }
    expect(wrapper.find('input[type="file"]').attributes('disabled')).toBeDefined()

    resolveValidation({
      valid: true,
      type: 'follow',
      targets: 2,
      contents: 0,
      errors: [],
    })
    await flushPromises()

    for (const testId of ['import-button', 'view-all-button', 'dedupe-button', 'clear-pool-button']) {
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('disabled')).toBeUndefined()
    }
    expect(wrapper.find('input[type="file"]').attributes('disabled')).toBeUndefined()
  })

  it('filters unsupported task types out of task-settings templates', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-unsupported-zero',
        name: 'Unsupported zero parameter action',
        type: 'unsupported_zero_parameter_action',
        params: {},
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-unsupported-target',
        name: 'Unsupported target action',
        type: 'unsupported_target_action',
        params: { targets: ['ignored'], contents: ['ignored'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-follow',
        name: 'Follow operators',
        type: 'follow',
        params: { targets: ['@northwind'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.find('[data-testid="saved-template-card-tmpl-unsupported-zero"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-unsupported-target"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Follow operators')
    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Targets')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@northwind')
  })

  it('normalizes supported task template type variants before filtering and editing', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-noisy-post',
        name: 'Noisy post template',
        type: ' POST ',
        params: { contents: ['hello from list'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-unsupported-target',
        name: 'Unsupported target action',
        type: ' unsupported_target_action ',
        params: { targets: ['ignored'], contents: ['ignored'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')

    expect(wrapper.find('[data-testid="saved-template-card-tmpl-unsupported-target"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-noisy-post"]').text()).toContain('Noisy post template')
    expect(wrapper.text()).toContain('Post')
    expect(wrapper.text()).not.toContain(' POST ')
    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Contents')
    expect((wrapper.get('[data-testid="content-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('hello from list')
  })

  it('trims backend template names before saved cards and editor fields display them', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow-padded-name',
        name: '  Padded follow template  ',
        type: 'follow',
        params: { targets: ['@northwind'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    const card = wrapper.get('[data-testid="saved-template-card-tmpl-follow-padded-name"]')

    expect(card.text()).toContain('Padded follow template')
    expect(card.element.textContent).not.toContain('  Padded follow template  ')
    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value)
      .toBe('Padded follow template')
  })

  it('trims backend template pool values before editor textareas display them', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow-padded-targets',
        name: 'Padded follow targets',
        type: 'follow',
        params: { targets: ['  @northwind  ', '', '  @southwind  '] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-post-padded-contents',
        name: 'Padded post contents',
        type: 'post',
        params: { contents: ['  hello world  ', '   ', '  second post  '] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value)
      .toBe('@northwind\n@southwind')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value)
      .not.toContain('  @northwind  ')

    await chooseType(wrapper, 'post')

    expect((wrapper.get('[data-testid="content-pool-textarea"]').element as HTMLTextAreaElement).value)
      .toBe('hello world\nsecond post')
    expect((wrapper.get('[data-testid="content-pool-textarea"]').element as HTMLTextAreaElement).value)
      .not.toContain('  hello world  ')
  })

  it('removes a deleted template locally even when the follow-up reload fails', async () => {
    listTemplates
      .mockResolvedValueOnce([
        {
          id: 'tmpl-post-delete',
          name: 'Delete me',
          type: 'post',
          params: { contents: ['delete me'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 'tmpl-post-fallback',
          name: 'Fallback post',
          type: 'post',
          params: { contents: ['fallback text'] },
          is_default: false,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ])
      .mockRejectedValueOnce(new Error('reload failed'))
    deleteTemplate.mockResolvedValue(undefined)

    const wrapper = mountView()
    await flushPromises()
    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Delete me')
    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')
    const confirmDeleteButton = wrapper.findAll('section[role="dialog"] button').find(button => button.text().includes('Delete'))
    expect(confirmDeleteButton, 'confirm delete button').toBeTruthy()
    await confirmDeleteButton!.trigger('click')
    await flushPromises()

    expect(deleteTemplate).toHaveBeenCalledWith('tmpl-post-delete')
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-post-delete"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-fallback"]').text()).toContain('Fallback post')
    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Fallback post')
    expect(wrapper.text()).not.toContain('Delete me')
    expect(wrapper.find('section[role="dialog"]').exists()).toBe(false)
    expect(showError).toHaveBeenCalled()
  })

  it('keeps long template names readable in the delete confirmation dialog', async () => {
    const longName = 'stage105-mobile-task-template-delete-name-with-a-very-long-unbroken-identifier-0123456789abcdef'
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow-long-delete',
        name: longName,
        type: 'follow',
        params: { targets: ['@northwind'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')

    const expectedDescription = `Delete template ${longName}? This action cannot be undone.`
    const description = wrapper.findAll('[title]').find(item => item.attributes('title') === expectedDescription)
    expect(description, 'delete description should keep the full long template name').toBeTruthy()
    expect(description!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    const warning = wrapper.findAll('[title]').find(item => item.attributes('title') === 'Accounts can only execute saved templates. Delete this template only if it is no longer used for task submissions.')
    expect(warning, 'delete warning should expose full text').toBeTruthy()
    expect(warning!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(warning!.attributes('role')).toBe('status')
    expect(warning!.attributes('aria-live')).toBe('polite')
    expect(warning!.attributes('aria-atomic')).toBe('true')
    const name = wrapper.get(`dd[title="${longName}"]`)
    expect(name.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    expect(name.text()).toBe(longName)

    const cancelButton = wrapper.get('section[role="dialog"] button[aria-label="Cancel"]')
    expect(cancelButton.attributes('title')).toBe('Cancel')
    expect(cancelButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(cancelButton.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))

    const deleteButton = wrapper.get('section[role="dialog"] button[aria-label="Delete"]')
    expect(deleteButton.attributes('title')).toBe('Delete')
    expect(deleteButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(deleteButton.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
  })

  it('keeps the delete confirmation open while delete is pending', async () => {
    let resolveDelete: () => void = () => {}
    listTemplates
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-pending-delete',
          name: 'Pending delete',
          type: 'follow',
          params: { targets: ['@northwind'] },
          is_default: false,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ])
      .mockResolvedValueOnce([])
    deleteTemplate.mockReturnValue(new Promise<void>(resolve => {
      resolveDelete = resolve
    }))

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')
    await wrapper.get('section[role="dialog"] button[aria-label="Delete"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(deleteTemplate).toHaveBeenCalledWith('tmpl-follow-pending-delete')
    expect(wrapper.get('section[role="dialog"]').text()).toContain('Pending delete')
    expect(wrapper.get('section[role="dialog"] button[aria-label="Processing"]').attributes('disabled')).toBeDefined()
    const cancelButton = wrapper.findAll('section[role="dialog"] button')
      .find(button => button.text().includes('Cancel'))
    expect(cancelButton, 'delete cancel button should stay visible while pending').toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('aria-label')).toBe('Processing')
    expect(cancelButton!.attributes('title')).toBe('Processing')

    await clickLatestDialogClose(wrapper)

    expect(wrapper.get('section[role="dialog"]').text()).toContain('Pending delete')

    resolveDelete()
    await flushPromises()

    expect(wrapper.find('section[role="dialog"]').exists()).toBe(false)
  })

  it('keeps delete failures visible in the confirmation dialog without exposing backend details', async () => {
    listTemplates.mockResolvedValueOnce([
      {
        id: 'tmpl-follow-delete-error',
        name: 'Delete error follow',
        type: 'follow',
        params: { targets: ['@northwind'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])
    deleteTemplate.mockRejectedValue({
      reason: 'TASK_TEMPLATE_SERVICE_UNAVAILABLE',
      message: 'task template service is unavailable token=secret',
    })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')
    await wrapper.get('section[role="dialog"] button[aria-label="Delete"]').trigger('click')
    await flushPromises()

    const dialog = wrapper.get('section[role="dialog"]')
    const alert = dialog.get('[role="alert"]')
    expect(deleteTemplate).toHaveBeenCalledWith('tmpl-follow-delete-error')
    expect(alert.text()).toBe('Task template service is temporarily unavailable. Try again later.')
    expect(alert.attributes('title')).toBe('Task template service is temporarily unavailable. Try again later.')
    expect(alert.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(showError).toHaveBeenCalledWith('Task template service is temporarily unavailable. Try again later.')
    expect(dialog.text()).toContain('Delete error follow')
    expect(dialog.text()).not.toContain('token=secret')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('token=secret')
  })

  it('refreshes the delete confirmation target when a template reload updates it', async () => {
    listTemplates
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-default',
          name: 'Default follow',
          type: 'follow',
          params: { targets: ['@default'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 'tmpl-follow-delete',
          name: 'Old follow delete',
          type: 'follow',
          params: { targets: ['@old'] },
          is_default: false,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ])
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-default',
          name: 'Default follow',
          type: 'follow',
          params: { targets: ['@default'] },
          is_default: false,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 'tmpl-follow-delete',
          name: 'Updated follow delete',
          type: 'follow',
          params: { targets: ['@updated', '@second'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:01Z',
        },
      ])
    setDefaultTemplate.mockResolvedValue({
      id: 'tmpl-follow-delete',
      name: 'Updated follow delete from action',
      type: 'follow',
      params: { targets: ['@from-action'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="saved-template-card-tmpl-follow-delete"]').trigger('click')
    await wrapper.get('[data-testid="delete-template-button"]').trigger('click')

    expect(wrapper.get('section[role="dialog"]').text()).toContain('Old follow delete')

    await wrapper.get('[data-testid="set-default-button"]').trigger('click')
    await flushPromises()

    const dialog = wrapper.get('section[role="dialog"]')
    expect(setDefaultTemplate).toHaveBeenCalledWith('tmpl-follow-delete')
    expect(dialog.text()).toContain('Updated follow delete')
    expect(dialog.text()).not.toContain('Old follow delete')
    expect(dialog.get('dd[title="Updated follow delete"]').text()).toBe('Updated follow delete')
  })

  it('keeps supported templates with malformed params visible as needing input', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow-malformed',
        name: 'Malformed follow params',
        type: 'follow',
        params: null,
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-malformed"]').text()).toContain('Malformed follow params')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-malformed"]').text()).toContain('Needs input')
    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Targets')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Targets are required')
    expect(showError).not.toHaveBeenCalled()
  })

  it('saves follow templates with target params only', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-follow-new',
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@northwind'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@northwind'] },
      is_default: true,
    })
  })

  it('maps known task-template save errors without exposing backend details', async () => {
    saveTemplate.mockRejectedValue({
      reason: 'TASK_TEMPLATE_INVALID',
      message: 'target list cannot exceed 500 items; internal detail',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Invalid follow')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('The template is incomplete. Fix the validation issues and try again.')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('target list cannot exceed')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('internal detail')
  })

  it('maps task-template service availability save errors without exposing backend details', async () => {
    saveTemplate.mockRejectedValue({
      reason: 'TASK_TEMPLATE_SERVICE_UNAVAILABLE',
      message: 'task template service is unavailable token=secret',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Service unavailable follow')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Task template service is temporarily unavailable. Try again later.')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('token=secret')
    const editorError = wrapper.get('[title="Task template service is temporarily unavailable. Try again later."]')
    expect(editorError.text()).toBe('Task template service is temporarily unavailable. Try again later.')
    expect(editorError.attributes('role')).toBe('alert')
    expect(editorError.attributes('aria-live')).toBe('assertive')
    expect(editorError.attributes('aria-atomic')).toBe('true')
    expect(editorError.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.text()).not.toContain('token=secret')

    await wrapper.get('[data-testid="template-name-input"]').setValue('Service unavailable follow retry')
    await flushPromises()

    expect(wrapper.find('[title="Task template service is temporarily unavailable. Try again later."]').exists()).toBe(false)
  })

  it('maps unsupported task action save errors to the existing task-type recovery message', async () => {
    saveTemplate.mockRejectedValue({
      reason: 'SOCIAL_TASK_UNSUPPORTED_ACTION',
      message: 'unsupported social task action internal=legacy-action',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Unsupported action follow')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('This task type is not supported. Choose another task type.')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('legacy-action')
  })

  it('keeps unknown task-template operation errors on safe fallback messages', async () => {
    saveTemplate.mockRejectedValue(new Error('database stack trace token=secret'))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Unknown failure')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to save template')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('database stack trace')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('token=secret')
  })

  it('selects the refreshed saved template after saving when the list payload differs from the save response', async () => {
    listTemplates
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-new',
          name: 'Daily follows from list',
          type: 'follow',
          params: { targets: ['@from-list'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:02Z',
        },
      ])
    saveTemplate.mockResolvedValue({
      id: 'tmpl-follow-new',
      name: 'Daily follows from save',
      type: 'follow',
      params: { targets: ['@from-save'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })
    const wrapper = mountView()
    await flushPromises()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value).toBe('Daily follows from list')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@from-list')
  })

  it('keeps a saved template visible locally when the follow-up reload fails', async () => {
    listTemplates
      .mockResolvedValueOnce([])
      .mockRejectedValueOnce(new Error('reload failed after save'))
    saveTemplate.mockResolvedValue({
      id: 'tmpl-follow-new',
      name: 'Daily follows from save',
      type: 'follow',
      params: { targets: ['@from-save'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })
    const wrapper = mountView()
    await flushPromises()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@northwind'] },
      is_default: true,
    })
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-new"]').text()).toContain('Daily follows from save')
    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Daily follows from save')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@from-save')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('1')
    expect(showSuccess).toHaveBeenCalledWith('taskSettings.saved')
    expect(showError).toHaveBeenCalled()
  })

  it('keeps a new default template visible locally when the follow-up reload fails', async () => {
    listTemplates
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-old-default',
          name: 'Old default',
          type: 'follow',
          params: { targets: ['@old'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 'tmpl-follow-new-default',
          name: 'New default',
          type: 'follow',
          params: { targets: ['@new'] },
          is_default: false,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ])
      .mockRejectedValueOnce(new Error('reload failed after default'))
    setDefaultTemplate.mockResolvedValue({
      id: 'tmpl-follow-new-default',
      name: 'New default',
      type: 'follow',
      params: { targets: ['@new'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="saved-template-card-tmpl-follow-new-default"]').trigger('click')
    await wrapper.get('[data-testid="set-default-button"]').trigger('click')
    await flushPromises()

    expect(setDefaultTemplate).toHaveBeenCalledWith('tmpl-follow-new-default')
    expect(wrapper.get('[data-testid="template-stats-defaults"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-new-default"]').text()).toContain('Default')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-old-default"]').text()).not.toContain('Default')
    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'New default')
    expect(showSuccess).toHaveBeenCalledWith('taskSettings.defaultSaved')
    expect(showError).toHaveBeenCalled()
  })

  it('keeps a copied template visible locally when the follow-up reload fails', async () => {
    listTemplates
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-source',
          name: 'Source follow',
          type: 'follow',
          params: { targets: ['@source'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ])
      .mockRejectedValueOnce(new Error('reload failed after copy'))
    copyTemplate.mockResolvedValue({
      id: 'tmpl-follow-copy',
      name: 'Source follow copy',
      type: 'follow',
      params: { targets: ['@copy'] },
      is_default: false,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="copy-template-button"]').trigger('click')
    await flushPromises()

    expect(copyTemplate).toHaveBeenCalledWith('tmpl-follow-source')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-source"]').text()).toContain('Source follow')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-copy"]').text()).toContain('Source follow copy')
    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Source follow copy')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@copy')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('2')
    expect(showSuccess).toHaveBeenCalledWith('taskSettings.copied')
    expect(showError).toHaveBeenCalled()
  })

  it('ignores stale template list responses after a newer save refresh completes', async () => {
    let resolveInitialLoad: (value: unknown) => void = () => {}
    listTemplates
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveInitialLoad = resolve
      }))
      .mockResolvedValueOnce([
        {
          id: 'tmpl-follow-new',
          name: 'Daily follows from refresh',
          type: 'follow',
          params: { targets: ['@from-refresh'] },
          is_default: true,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:02Z',
        },
      ])
    saveTemplate.mockResolvedValue({
      id: 'tmpl-follow-new',
      name: 'Daily follows from save',
      type: 'follow',
      params: { targets: ['@from-save'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:01Z',
    })
    const wrapper = mountView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await flushPromises()

    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value).toBe('Daily follows from refresh')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@from-refresh')

    resolveInitialLoad([
      {
        id: 'tmpl-follow-stale',
        name: 'Stale initial template',
        type: 'follow',
        params: { targets: ['@stale-initial'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])
    await flushPromises()

    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value).toBe('Daily follows from refresh')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@from-refresh')
    expect(wrapper.find('[data-testid="saved-template-card-tmpl-follow-stale"]').exists()).toBe(false)
  })

  it('does not overwrite editor changes when a pending template load finishes', async () => {
    let resolveInitialLoad: (value: unknown) => void = () => {}
    listTemplates.mockImplementationOnce(() => new Promise(resolve => {
      resolveInitialLoad = resolve
    }))

    const wrapper = mountView()
    await enterName(wrapper, 'Draft follow')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@draft')

    resolveInitialLoad([
      {
        id: 'tmpl-follow-loaded',
        name: 'Loaded follow',
        type: 'follow',
        params: { targets: ['@loaded'] },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])
    await flushPromises()

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-loaded"]').text()).toContain('Loaded follow')
    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value).toBe('Draft follow')
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@draft')
  })

  it('does not force-select a saved template when the editor context changes before save refresh completes', async () => {
    let resolveSave: (template: unknown) => void = () => {}
    const savedTemplate = {
      id: 'tmpl-follow-new',
      name: 'Daily follows from refresh',
      type: 'follow',
      params: { targets: ['@from-refresh'] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:02Z',
    }
    listTemplates
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([savedTemplate])
    saveTemplate.mockImplementationOnce(() => new Promise(resolve => {
      resolveSave = resolve
    }))
    const wrapper = mountView()
    await flushPromises()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Daily follows')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await saveButton(wrapper).trigger('click')
    await chooseType(wrapper, 'post')

    resolveSave(savedTemplate)
    await flushPromises()

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Daily follows',
      type: 'follow',
      params: { targets: ['@northwind'] },
      is_default: true,
    })
    expect(wrapper.find('[data-testid="content-pool-textarea"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="target-pool-textarea"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="template-name-input"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.get('[data-testid="active-type-empty-state"]').text()).toContain('No saved config for this type')
    expect(wrapper.text()).not.toContain('Daily follows from refresh')
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

  it('ignores stale parameter file imports when the editor switches pools before the file is read', async () => {
    let resolveText!: (value: string) => void
    const file = new File(['pending'], 'targets.txt', { type: 'text/plain' })
    Object.defineProperty(file, 'text', {
      configurable: true,
      value: vi.fn(() => new Promise(resolve => {
        resolveText = resolve
      })),
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Follow import')
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [file],
    })
    await input.trigger('change')

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Post import')
    resolveText('@stale-follow-target')
    await flushPromises()

    expect(wrapper.find('[data-testid="target-pool-textarea"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="content-pool-textarea"]').element as HTMLTextAreaElement).value)
      .not.toContain('@stale-follow-target')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
  })

  it('keeps existing values and validation when an imported parameter file has no usable rows', async () => {
    validateTemplate.mockResolvedValueOnce({
      valid: true,
      type: 'follow',
      targets: 1,
      contents: 0,
      errors: [],
    })
    const file = new File(['pending'], 'empty-targets.txt', { type: 'text/plain' })
    Object.defineProperty(file, 'text', {
      configurable: true,
      value: vi.fn(() => Promise.resolve('   \n\n  ')),
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Follow import')
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')
    showSuccess.mockClear()
    showWarning.mockClear()

    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [file],
    })
    await input.trigger('change')
    await flushPromises()

    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@northwind')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')
    expect(showWarning).toHaveBeenCalledWith('No usable values were found in the selected file.')
    expect(showSuccess).not.toHaveBeenCalled()
  })

  it('keeps parameter file read failures friendly without exposing raw local errors', async () => {
    const rawError = new Error('local file read failed: token=secret')
    const file = new File(['pending'], 'targets.txt', { type: 'text/plain' })
    const retryFile = new File(['@northwind'], 'retry-targets.txt', { type: 'text/plain' })
    Object.defineProperty(file, 'text', {
      configurable: true,
      value: vi.fn(() => Promise.reject(rawError)),
    })
    Object.defineProperty(retryFile, 'text', {
      configurable: true,
      value: vi.fn(() => Promise.resolve('@northwind')),
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper, 'Follow import')
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [file],
    })
    await input.trigger('change')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to import the selected file. Choose a readable text file and try again.')
    const inlineError = wrapper.get('[role="alert"]')
    expect(inlineError.text()).toBe('Failed to import the selected file. Choose a readable text file and try again.')
    expect(inlineError.attributes('title')).toBe('Failed to import the selected file. Choose a readable text file and try again.')
    expect(inlineError.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.text()).not.toContain('token=secret')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('token=secret')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('task_settings.import_file', rawError)
    expect(showSuccess).not.toHaveBeenCalled()

    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [retryFile],
    })
    await input.trigger('change')
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value)
      .toContain('@northwind')
  })

  it('shows a contents pool and requires some structured post input for post templates', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper)

    expect(wrapper.get('[data-testid="parameter-pool-manager"]').text()).toContain('Contents')
    expect(wrapper.get('[data-testid="quote-post-url-input"]').attributes('placeholder')).toContain('https://x.com')
    expect(wrapper.get('[data-testid="post-media-empty"]').text()).toContain('No post media attached yet')
    expect(wrapper.find('[data-testid="content-pool-textarea"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="target-pool-textarea"]').exists()).toBe(false)
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Add post text or at least one media item before saving.')
  })

  it('saves post templates with quote links and structured media refs', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-post-rich',
      name: 'Rich post',
      type: 'post',
      params: {
        contents: ['hello world'],
        quote_post_url: 'https://x.com/northwind/status/1',
        media: [{ source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'post-image-1.png' }],
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Rich post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello world')
    await wrapper.get('[data-testid="quote-post-url-input"]').setValue('https://x.com/northwind/status/1')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    expect(wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-stub"]').attributes('data-max-size')).toBe('2097152')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Rich post',
      type: 'post',
      params: {
        contents: ['hello world'],
        quote_post_url: 'https://x.com/northwind/status/1',
        media: [{ source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'post-image-1.png' }],
      },
      is_default: true,
    })
  })

  it('saves media-only post templates without requiring text content', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-post-media-only',
      name: 'Media only post',
      type: 'post',
      params: {
        media: [{ source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'post-image-1.png' }],
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Media only post')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')

    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()

    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Media only post',
      type: 'post',
      params: {
        contents: [],
        media: [{ source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'post-image-1.png' }],
      },
      is_default: true,
    })
  })

  it('keeps post media action labels inspectable and constrained on narrow layouts', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')

    const addButton = wrapper.get('[data-testid="add-post-media-button"]')
    expect(addButton.text()).toContain('Add media')
    expect(addButton.attributes('aria-label')).toBe('Add media')
    expect(addButton.attributes('title')).toBe('Add media')
    expect(addButton.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
    expect(addButton.get('span.min-w-0.truncate').exists()).toBe(true)

    await addButton.trigger('click')

    const removeButton = wrapper.get('[data-testid="remove-post-media-button-0"]')
    expect(wrapper.findAll('[data-testid^="post-media-item-"]')).toHaveLength(1)
    expect(removeButton.text()).toContain('Remove media')
    expect(removeButton.attributes('aria-label')).toBe('Remove media')
    expect(removeButton.attributes('title')).toBe('Remove media')
    expect(removeButton.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
    expect(removeButton.get('span.min-w-0.truncate').exists()).toBe(true)

    await removeButton.trigger('click')

    expect(wrapper.find('[data-testid="post-media-item-0"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="post-media-empty"]').text()).toContain('No post media attached yet.')
  })

  it('locks post media add and remove actions while a template operation is pending', async () => {
    let resolveSave: (value: unknown) => void = () => {}
    saveTemplate.mockImplementationOnce(() => new Promise(resolve => {
      resolveSave = resolve
    }))
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Busy media post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello world')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')

    expect(wrapper.findAll('[data-testid^="post-media-item-"]')).toHaveLength(1)

    await saveButton(wrapper).trigger('click')
    await nextTick()

    const addButton = wrapper.get('[data-testid="add-post-media-button"]')
    const removeButton = wrapper.get('[data-testid="remove-post-media-button-0"]')
    const upload = wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-stub"]')
    const uploadComponent = wrapper.findComponent(ImageUploadStub)
    expect(uploadComponent.exists()).toBe(true)
    expect(addButton.attributes('disabled')).toBeDefined()
    expect(addButton.attributes('aria-label')).toBe('Saving')
    expect(addButton.attributes('title')).toBe('Saving')
    expect(removeButton.attributes('disabled')).toBeDefined()
    expect(removeButton.attributes('aria-label')).toBe('Saving')
    expect(removeButton.attributes('title')).toBe('Saving')
    expect(upload.attributes('data-disabled')).toBe('true')
    expect(upload.get('[data-testid="image-upload-set"]').attributes('disabled')).toBeDefined()
    expect(upload.get('[data-testid="image-upload-clear"]').attributes('disabled')).toBeDefined()

    await removeButton.trigger('click')
    await upload.get('[data-testid="image-upload-set"]').trigger('click')
    await upload.get('[data-testid="image-upload-clear"]').trigger('click')
    uploadComponent.vm.$emit('update:modelValue', imageUploadValue)
    await nextTick()

    expect(wrapper.findAll('[data-testid^="post-media-item-"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-stub"]').attributes('data-value')).toBe('')

    resolveSave({
      id: 'tmpl-post-busy-media',
      name: 'Busy media post',
      type: 'post',
      params: { contents: ['hello world'], media: [] },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    await flushPromises()
  })

  it('keeps saved media-only post templates marked as ready when they carry executable image media', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-media-only',
        name: 'Media only post',
        type: 'post',
        params: {
          media: [{ source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'post-image-1.png' }],
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-media-only"]').text()).toContain('Ready')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('marks stored templates with only blank pool values as needing input', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow-blank',
        name: 'Blank follow pool',
        type: 'follow',
        params: { targets: [' ', ''] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-post-blank',
        name: 'Blank post pool',
        type: 'post',
        params: { contents: ['   '] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow-blank"]').text()).toContain('Needs input')

    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-blank"]').text()).toContain('Needs input')
  })

  it('marks stored post templates with more than four media items as needing input', async () => {
    const media = Array.from({ length: 5 }, (_, index) => ({
      source: 'library',
      storage_key: `social-task/42/post-${index + 1}.png`,
      content_type: 'image/png',
      file_name: `post-${index + 1}.png`,
    }))
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-too-many-media',
        name: 'Too many media post',
        type: 'post',
        params: {
          contents: ['hello world'],
          media,
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-too-many-media"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Post templates can contain at most 4 media items')
    expect(wrapper.get('[data-testid="add-post-media-button"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="add-post-media-button"]').attributes('aria-label')).toBe('Post templates can contain at most 4 media items.')
    expect(wrapper.get('[data-testid="add-post-media-button"]').attributes('title')).toBe('Post templates can contain at most 4 media items.')
  })

  it('keeps quote-only post templates blocked until text or executable media is present', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Quote only post')
    await wrapper.get('[data-testid="quote-post-url-input"]').setValue('https://x.com/northwind/status/1')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Add post text or at least one media item before saving.')
  })

  it('blocks saving post templates when attached media is mp4 video', async () => {
    imageUploadValue = 'data:video/mp4;base64,QUJD'
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Video post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello video')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Video post media is not supported for execution right now')
    await saveButton(wrapper).trigger('click')
    expect(saveTemplate).not.toHaveBeenCalled()
  })

  it('blocks saving post templates when attached media is not an image', async () => {
    imageUploadValue = 'data:application/pdf;base64,QUJD'
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'PDF post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello file')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Only image media is supported right now')
  })

  it('marks stored mixed image and video post templates as needing input', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-mixed-media',
        name: 'Mixed media post',
        type: 'post',
        params: {
          contents: ['hello mixed'],
          media: [
            { source: 'library', storage_key: 'social-task/42/post-image.jpg', content_type: 'image/jpeg' },
            { source: 'library', storage_key: 'social-task/42/post-video.mp4', content_type: 'video/mp4' },
          ],
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-mixed-media"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Video post media is not supported for execution right now')
  })

  it('marks stale post templates with non-inline media refs as needing input and shows the executable-media boundary', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-library',
        name: 'Library post',
        type: 'post',
        params: {
          contents: ['hello world'],
          media: [{ source: 'library', storage_key: 'media/post.jpg', content_type: 'image/jpeg' }],
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')

    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-library"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Saved media references are not available for execution yet.')
  })

  it('treats internal social-task image refs as ready templates while keeping stored video refs blocked', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-task-asset',
        name: 'Stored post',
        type: 'post',
        params: {
          contents: ['hello world'],
          media: [{ source: 'library', storage_key: 'social-task/42/post.jpg', content_type: 'image/jpeg' }],
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-post-task-video-asset',
        name: 'Stored video post',
        type: 'post',
        params: {
          contents: ['hello video world'],
          media: [{ source: 'library', storage_key: 'social-task/42/post-video.mp4', content_type: 'video/mp4' }],
        },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-avatar-task-asset',
        name: 'Stored avatar',
        type: 'update_avatar',
        params: {
          avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 400, height: 400 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-banner-task-asset',
        name: 'Stored banner',
        type: 'update_banner',
        params: {
          banner: { source: 'library', storage_key: 'social-task/42/banner.png', content_type: 'image/png', width: 1500, height: 500 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-task-asset"]').text()).toContain('Ready')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-task-video-asset"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()

    await chooseType(wrapper, 'update_avatar')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-avatar-task-asset"]').text()).toContain('Ready')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()

    await chooseType(wrapper, 'update_banner')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-banner-task-asset"]').text()).toContain('Ready')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('loads blob previews for stored social-task media without rewriting the template payload back to inline data', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-avatar-task-asset',
        name: 'Stored avatar',
        type: 'update_avatar',
        params: {
          avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 400, height: 400 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'update_avatar')
    await flushPromises()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>).toHaveBeenCalled()
    const upload = wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-stub"]')
    expect(upload.attributes('data-value')).toBe('')
    expect(upload.attributes('data-preview-src')).toBe('blob:task-media-preview')
    expect(upload.attributes('data-has-value')).toBe('true')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('records stored media preview failures while keeping saved media usable', async () => {
    const previewError = new Error('preview service failed token=secret')
    previewMedia.mockRejectedValue(previewError)
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-avatar-preview-failed',
        name: 'Stored avatar preview failed',
        type: 'update_avatar',
        params: {
          avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 400, height: 400 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'update_avatar')
    await flushPromises()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('task_settings.preview_media', previewError)
    const upload = wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-stub"]')
    expect(upload.attributes('data-preview-src')).toBe('')
    expect(upload.attributes('data-has-value')).toBe('true')
    expect(wrapper.text()).not.toContain('preview service failed')
    expect(wrapper.text()).not.toContain('token=secret')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('ignores stale stored media preview responses after switching templates', async () => {
    let resolveFirstPreview: ((blob: Blob) => void) | undefined
    previewMedia.mockImplementation((storageKey: string) => {
      if (storageKey === 'social-task/42/first.png') {
        return new Promise<Blob>(resolve => {
          resolveFirstPreview = resolve
        })
      }
      return Promise.resolve(new Blob(['second'], { type: 'image/png' }))
    })
    ;(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>)
      .mockReset()
      .mockReturnValueOnce('blob:second-preview')
      .mockReturnValueOnce('blob:first-late-preview')
    const revokeObjectURL = globalThis.URL.revokeObjectURL as unknown as ReturnType<typeof vi.fn>

    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-post-first',
        name: 'First stored post',
        type: 'post',
        params: {
          contents: ['first'],
          media: [{ source: 'library', storage_key: 'social-task/42/first.png', content_type: 'image/png' }],
        },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-post-second',
        name: 'Second stored post',
        type: 'post',
        params: {
          contents: ['second'],
          media: [{ source: 'library', storage_key: 'social-task/42/second.png', content_type: 'image/png' }],
        },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'post')
    expect(previewMedia).toHaveBeenCalledWith('social-task/42/first.png')

    await wrapper.get('[data-testid="saved-template-card-tmpl-post-second"]').trigger('click')
    await flushPromises()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/second.png')
    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Second stored post')
    expect(wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-stub"]').attributes('data-preview-src')).toBe('blob:second-preview')

    resolveFirstPreview?.(new Blob(['first'], { type: 'image/png' }))
    await flushPromises()

    expect(wrapper.get('[data-testid="template-name-input"]').element).toHaveProperty('value', 'Second stored post')
    expect(wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-stub"]').attributes('data-preview-src')).toBe('blob:second-preview')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:first-late-preview')
  })

  it('ignores stale stored media preview responses after clearing media in the editor', async () => {
    let resolvePreview: ((blob: Blob) => void) | undefined
    previewMedia.mockImplementation((storageKey: string) => {
      if (storageKey === 'social-task/42/avatar.png') {
        return new Promise<Blob>(resolve => {
          resolvePreview = resolve
        })
      }
      return Promise.resolve(new Blob(['preview'], { type: 'image/png' }))
    })
    ;(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>)
      .mockReset()
      .mockReturnValueOnce('blob:stale-avatar-preview')
    const revokeObjectURL = globalThis.URL.revokeObjectURL as unknown as ReturnType<typeof vi.fn>

    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-avatar-task-asset',
        name: 'Stored avatar',
        type: 'update_avatar',
        params: {
          avatar: { source: 'library', storage_key: 'social-task/42/avatar.png', content_type: 'image/png', width: 400, height: 400 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'update_avatar')
    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')

    const upload = wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-stub"]')
    await upload.get('[data-testid="image-upload-clear"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-stub"]').attributes('data-has-value')).toBe('false')

    resolvePreview?.(new Blob(['avatar'], { type: 'image/png' }))
    await flushPromises()

    const clearedUpload = wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-stub"]')
    expect(clearedUpload.attributes('data-preview-src')).toBe('')
    expect(clearedUpload.attributes('data-has-value')).toBe('false')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:stale-avatar-preview')
  })

  it('saves update_profile templates with structured profile params only', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-profile',
      name: 'Profile refresh',
      type: 'update_profile',
      params: {
        profile: {
          display_name: 'Northwind Ops',
          screen_name: 'northwind_ops',
          description: 'Operator account',
          location: 'Singapore',
          url: 'https://example.com',
        },
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'update_profile')
    await enterName(wrapper, 'Profile refresh')
    await wrapper.get('[data-testid="profile-display-name-input"]').setValue('Northwind Ops')
    await wrapper.get('[data-testid="profile-screen-name-input"]').setValue('northwind_ops')
    await wrapper.get('[data-testid="profile-description-textarea"]').setValue('Operator account')
    await wrapper.get('[data-testid="profile-location-input"]').setValue('Singapore')
    await wrapper.get('[data-testid="profile-url-input"]').setValue('https://example.com')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Profile refresh',
      type: 'update_profile',
      params: {
        profile: {
          display_name: 'Northwind Ops',
          screen_name: 'northwind_ops',
          description: 'Operator account',
          location: 'Singapore',
          url: 'https://example.com',
        },
      },
      is_default: true,
    })
  })

  it('requires avatar media for update_avatar templates and saves the structured media ref once uploaded', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-avatar',
      name: 'Avatar refresh',
      type: 'update_avatar',
      params: {
        avatar: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'avatar-image.png', width: 400, height: 400 },
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()
    installImageDimensionMock([{ width: 400, height: 400 }])

    await chooseType(wrapper, 'update_avatar')
    await enterName(wrapper, 'Avatar refresh')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Avatar media is required')

    await wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-set"]').trigger('click')
    await flushPromises()
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Avatar refresh',
      type: 'update_avatar',
      params: {
        avatar: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'avatar-image.png', width: 400, height: 400 },
      },
      is_default: true,
    })
  })

  it('blocks avatar and banner templates when uploaded image dimensions do not match execution requirements', async () => {
    const wrapper = await mountLoadedView()
    installImageDimensionMock([
      { width: 300, height: 300 },
      { width: 1200, height: 500 },
    ])

    await chooseType(wrapper, 'update_avatar')
    await enterName(wrapper, 'Avatar refresh')
    await wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-set"]').trigger('click')
    await flushPromises()
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Avatar image must be exactly 400x400 pixels')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).not.toHaveBeenCalled()

    await chooseType(wrapper, 'update_banner')
    await enterName(wrapper, 'Banner refresh')
    await wrapper.get('[data-testid="banner-editor"]').get('[data-testid="image-upload-set"]').trigger('click')
    await flushPromises()
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Banner image must be exactly 1500x500 pixels')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).not.toHaveBeenCalled()
  })

  it('requires banner media for update_banner templates and saves the structured media ref once uploaded', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-banner',
      name: 'Banner refresh',
      type: 'update_banner',
      params: {
        banner: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'banner-image.png', width: 1500, height: 500 },
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()
    installImageDimensionMock([{ width: 1500, height: 500 }])

    await chooseType(wrapper, 'update_banner')
    await enterName(wrapper, 'Banner refresh')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Banner media is required')

    expect(wrapper.get('[data-testid="banner-editor"]').get('[data-testid="image-upload-stub"]').attributes('data-max-size')).toBe('2097152')
    await wrapper.get('[data-testid="banner-editor"]').get('[data-testid="image-upload-set"]').trigger('click')
    await flushPromises()
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
    expect(saveButton(wrapper).attributes('title')).toBeUndefined()

    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Banner refresh',
      type: 'update_banner',
      params: {
        banner: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'banner-image.png', width: 1500, height: 500 },
      },
      is_default: true,
    })
  })

  it('marks saved avatar and banner templates with wrong dimensions as needing input', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-avatar-inline',
        name: 'Inline avatar',
        type: 'update_avatar',
        params: {
          avatar: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'avatar-inline.png', width: 300, height: 300 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-banner-inline',
        name: 'Inline banner',
        type: 'update_banner',
        params: {
          banner: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'banner-inline.png', width: 1200, height: 500 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'update_avatar')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-avatar-inline"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Avatar image must be exactly 400x400 pixels')

    await chooseType(wrapper, 'update_banner')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-banner-inline"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Banner image must be exactly 1500x500 pixels')
  })

  it('marks stale avatar and banner templates with non-inline media refs as needing input and shows the executable-media boundary', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-avatar-library',
        name: 'Library avatar',
        type: 'update_avatar',
        params: {
          avatar: { source: 'library', storage_key: 'media/avatar.jpg', content_type: 'image/jpeg', width: 400, height: 400 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
      {
        id: 'tmpl-banner-library',
        name: 'Library banner',
        type: 'update_banner',
        params: {
          banner: { source: 'library', storage_key: 'media/banner.jpg', content_type: 'image/jpeg', width: 1500, height: 500 },
        },
        is_default: true,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await chooseType(wrapper, 'update_avatar')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-avatar-library"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Saved media references are not available for execution yet.')

    await chooseType(wrapper, 'update_banner')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-banner-library"]').text()).toContain('Needs input')
    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Saved media references are not available for execution yet.')
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
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('@northwind\n\n@socialops\n   ')

    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="pool-empty-lines"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').text()).toContain('2 empty line(s) will be ignored')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('lets users clear ignored-only parameter pool text', async () => {
    validateTemplate.mockResolvedValueOnce({
      valid: false,
      type: 'follow',
      targets: 0,
      contents: 0,
      errors: [{ field: 'targets', message: 'Targets are required' }],
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'follow')
    await enterName(wrapper)
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue(',,,\n   ')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="pool-empty-lines"]').text()).toContain('5')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template has issues')

    const clearButton = wrapper.get('[data-testid="clear-pool-button"]')
    expect(clearButton.attributes('disabled')).toBeUndefined()
    await clearButton.trigger('click')
    await flushPromises()

    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).not.toContain('Template has issues')
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

  it('does not report a zero-count dedupe as a successful change', async () => {
    validateTemplate.mockResolvedValueOnce({
      valid: true,
      type: 'like',
      targets: 2,
      contents: 0,
      errors: [],
    })
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'like')
    await enterName(wrapper)
    await wrapper.get('[data-testid="target-pool-textarea"]').setValue('https://x.com/post/1\nhttps://x.com/post/2')
    await wrapper.get('[data-testid="validation-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')
    showSuccess.mockClear()

    const dedupeButton = wrapper.get('[data-testid="dedupe-button"]')
    expect(dedupeButton.attributes('disabled')).toBeDefined()
    expect(dedupeButton.attributes('aria-label')).toBe('No duplicate values to remove.')
    expect(dedupeButton.attributes('title')).toBe('No duplicate values to remove.')
    await dedupeButton.trigger('click')

    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value)
      .toBe('https://x.com/post/1\nhttps://x.com/post/2')
    expect(wrapper.get('[data-testid="template-validation-panel"]').text()).toContain('Template is complete')
    expect(showSuccess).not.toHaveBeenCalledWith('Removed 0 duplicate value(s).')
  })

  it('keeps parameter-pool action labels inspectable and constrained on narrow layouts', async () => {
    const wrapper = await mountLoadedView()

    const actions = [
      ['import-button', 'Import', 'Import', false],
      ['view-all-button', 'View all', 'No values in this pool.', true],
      ['dedupe-button', 'Deduplicate', 'No duplicate values to remove.', true],
      ['clear-pool-button', 'Clear', 'No values in this pool.', true],
    ] as const

    expect(actions.map(([testId]) => wrapper.get(`[data-testid="${testId}"]`).exists())).toEqual([true, true, true, true])

    for (const [testId, label, title, disabled] of actions) {
      const button = wrapper.get(`[data-testid="${testId}"]`)
      expect(button.text()).toContain(label)
      expect(button.attributes('aria-label')).toBe(title)
      expect(button.attributes('title')).toBe(title)
      expect(button.attributes('disabled') !== undefined).toBe(disabled)
      expect(button.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
      expect(button.get('span.min-w-0.truncate').exists()).toBe(true)
    }
  })

  it('keeps copy, delete, and set-default actions available for saved templates', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow',
        name: 'Follow batch',
        type: 'follow',
        params: { targets: ['northwind'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    expect(wrapper.find('[data-testid="copy-template-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="set-default-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="delete-template-button"]').exists()).toBe(true)
  })

  it('keeps pending operation labels scoped to the action in progress', async () => {
    let resolveCopy: (template: unknown) => void = () => {}
    copyTemplate.mockReturnValue(new Promise(resolve => {
      resolveCopy = resolve
    }))
    const copiedTemplate = {
      id: 'tmpl-follow-copy',
      name: 'Follow batch copy',
      type: 'follow',
      params: { targets: ['northwind'] },
      is_default: false,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    }
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow',
        name: 'Follow batch',
        type: 'follow',
        params: { targets: ['northwind'] },
        is_default: false,
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T00:00:00Z',
      },
    ])

    await wrapper.get('[data-testid="copy-template-button"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-testid="copy-template-button"]').text()).toContain('Processing')
    expect(saveButton(wrapper).text()).toContain('Save')
    expect(saveButton(wrapper).text()).not.toContain('Saving')
    expect(wrapper.get('[data-testid="set-default-button"]').attributes('disabled')).toBeDefined()

    resolveCopy(copiedTemplate)
    await flushPromises()

    expect(wrapper.get('[data-testid="copy-template-button"]').text()).toContain('Copy')
  })

  it('scopes the saved template list to the currently selected task type', async () => {
    const wrapper = await mountLoadedView([
      {
        id: 'tmpl-follow',
        name: 'Follow operators',
        type: 'follow',
        params: { targets: ['@northwind'] },
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
    expect((wrapper.get('[data-testid="target-pool-textarea"]').element as HTMLTextAreaElement).value).toBe('@northwind')

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
