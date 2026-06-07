import { defineComponent } from 'vue'
import { flushPromises, mount, VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import TaskSettingsView from '../TaskSettingsView.vue'

const { listTemplates, saveTemplate, copyTemplate, setDefaultTemplate, deleteTemplate, previewMedia, showError } = vi.hoisted(() => ({
  listTemplates: vi.fn(),
  saveTemplate: vi.fn(),
  copyTemplate: vi.fn(),
  setDefaultTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  previewMedia: vi.fn(),
  showError: vi.fn(),
}))

let imageUploadValue = 'data:image/png;base64,QUJD'

vi.mock('@/api/taskSettings', () => ({
  default: {
    listTemplates,
    saveTemplate,
    validateTemplate: vi.fn(),
    copyTemplate,
    setDefaultTemplate,
    deleteTemplate,
    previewMedia,
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

const ImageUploadStub = defineComponent({
  props: {
    modelValue: { type: String, default: '' },
    previewSrc: { type: String, default: '' },
    hasValue: { type: Boolean, default: undefined },
    uploadLabel: { type: String, default: '' },
    removeLabel: { type: String, default: '' },
    hint: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(_props, { emit }) {
    return {
      emitSet: () => emit('update:modelValue', imageUploadValue),
      emitClear: () => emit('update:modelValue', ''),
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
    >
      <button type="button" data-testid="image-upload-set" @click="emitSet">set</button>
      <button type="button" data-testid="image-upload-clear" @click="emitClear">clear</button>
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
    'taskSettings.summary.profileFields': 'Profile fields',
    'taskSettings.summary.quotePost': 'Quote link',
    'taskSettings.summary.media': 'Media',
    'taskSettings.summary.executionHint': 'Selected templates are submitted with task requests.',
    'taskSettings.media.postImages': 'Post media',
    'taskSettings.media.postImagesHint': 'Attach up to 4 images or 1 MP4 video for post templates.',
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
    'taskSettings.filters.all': 'All types',
    'taskSettings.validate': 'Validate',
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
    'taskSettings.typeRequirements.like': 'Targets: Tweet URL or Tweet ID',
    'taskSettings.typeRequirements.retweet': 'Targets: Tweet URL or Tweet ID',
    'taskSettings.typeRequirements.post': 'Contents: one post text per line',
    'taskSettings.typeRequirements.update_profile': 'Profile fields: at least one field is required',
    'taskSettings.typeRequirements.update_avatar': 'Avatar: upload one image',
    'taskSettings.typeRequirements.update_banner': 'Banner: upload one image',
    'taskSettings.validation.nameRequired': 'Name is required',
    'taskSettings.validation.targetsRequired': 'Targets are required',
  'taskSettings.validation.postConfigurationRequired': 'Add post text or at least one media item before saving.',
    'taskSettings.validation.postVideoUnavailable': 'Only one MP4 video is supported for post media right now',
    'taskSettings.validation.postMediaTypeUnsupported': 'Only image media or a single MP4 video is supported right now',
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
        BaseDialog: { props: ['show', 'title'], template: '<section v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>' },
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
    copyTemplate.mockReset()
    setDefaultTemplate.mockReset()
    deleteTemplate.mockReset()
    previewMedia.mockReset()
    showError.mockReset()
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
        quote_post_url: 'https://x.com/openai/status/1',
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
    await wrapper.get('[data-testid="quote-post-url-input"]').setValue('https://x.com/openai/status/1')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Rich post',
      type: 'post',
      params: {
        contents: ['hello world'],
        quote_post_url: 'https://x.com/openai/status/1',
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

  it('keeps quote-only post templates blocked until text or executable media is present', async () => {
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Quote only post')
    await wrapper.get('[data-testid="quote-post-url-input"]').setValue('https://x.com/openai/status/1')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Add post text or at least one media item before saving.')
  })

  it('allows saving post templates when attached media is a single mp4 video', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-video-post',
      name: 'Video post',
      type: 'post',
      params: {
        contents: ['hello video'],
        media: [{
          source: 'inline',
          url: 'data:video/mp4;base64,QUJD',
          content_type: 'video/mp4',
          file_name: 'post-image-1.mp4',
        }],
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    imageUploadValue = 'data:video/mp4;base64,QUJD'
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'Video post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello video')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')

    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Video post',
      type: 'post',
      params: {
        contents: ['hello video'],
        media: [{
          source: 'inline',
          url: 'data:video/mp4;base64,QUJD',
          content_type: 'video/mp4',
          file_name: 'post-image-1.mp4',
        }],
      },
      is_default: true,
    })
  })

  it('blocks saving post templates when attached media is not an image or supported mp4 video', async () => {
    imageUploadValue = 'data:application/pdf;base64,QUJD'
    const wrapper = await mountLoadedView()

    await chooseType(wrapper, 'post')
    await enterName(wrapper, 'PDF post')
    await wrapper.get('[data-testid="content-pool-textarea"]').setValue('hello file')
    await wrapper.get('[data-testid="add-post-media-button"]').trigger('click')
    await wrapper.get('[data-testid="post-media-item-0"]').get('[data-testid="image-upload-set"]').trigger('click')

    expect(saveButton(wrapper).attributes('disabled')).toBeDefined()
    expect(saveButton(wrapper).attributes('title')).toContain('Only image media or a single MP4 video is supported right now')
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
    expect(saveButton(wrapper).attributes('title')).toContain('Only one MP4 video is supported for post media right now')
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

  it('treats internal social-task media refs as ready templates instead of stale saved-media placeholders', async () => {
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
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-task-video-asset"]').text()).toContain('Ready')
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

  it('allows saving avatar templates when the uploaded image needs normalization', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-avatar-normalized',
      name: 'Avatar refresh',
      type: 'update_avatar',
      params: {
        avatar: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'avatar-image.png', width: 300, height: 300 },
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()
    installImageDimensionMock([{ width: 300, height: 300 }])

    await chooseType(wrapper, 'update_avatar')
    await enterName(wrapper, 'Avatar refresh')
    await wrapper.get('[data-testid="avatar-editor"]').get('[data-testid="image-upload-set"]').trigger('click')
    await flushPromises()
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
    expect(saveButton(wrapper).attributes('title')).toBeUndefined()

    await saveButton(wrapper).trigger('click')

    expect(saveTemplate).toHaveBeenCalledWith({
      id: undefined,
      name: 'Avatar refresh',
      type: 'update_avatar',
      params: {
        avatar: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'avatar-image.png', width: 300, height: 300 },
      },
      is_default: true,
    })
  })

  it('allows saving banner templates when the uploaded image needs normalization', async () => {
    saveTemplate.mockResolvedValue({
      id: 'tmpl-banner-normalized',
      name: 'Banner refresh',
      type: 'update_banner',
      params: {
        banner: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'banner-image.png', width: 1200, height: 500 },
      },
      is_default: true,
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T00:00:00Z',
    })
    const wrapper = await mountLoadedView()
    installImageDimensionMock([{ width: 1200, height: 500 }])

    await chooseType(wrapper, 'update_banner')
    await enterName(wrapper, 'Banner refresh')
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
        banner: { source: 'inline', url: 'data:image/png;base64,QUJD', content_type: 'image/png', file_name: 'banner-image.png', width: 1200, height: 500 },
      },
      is_default: true,
    })
  })

  it('keeps saved avatar and banner templates usable when inline images need normalization', async () => {
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
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-avatar-inline"]').text()).toContain('Ready')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()

    await chooseType(wrapper, 'update_banner')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-banner-inline"]').text()).toContain('Ready')
    expect(saveButton(wrapper).attributes('disabled')).toBeUndefined()
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
