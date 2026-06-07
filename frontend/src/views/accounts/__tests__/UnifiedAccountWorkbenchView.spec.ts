import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import UnifiedAccountWorkbenchView from '../UnifiedAccountWorkbenchView.vue'

const {
  listMyAccounts,
  batchImportMyAccounts,
  deleteMyAccount,
  batchDeleteMyAccounts,
  updateMyAccount,
  submitTask,
  setDefaultProxy,
  batchSetDefaultProxy,
  exportMyAccounts,
  listUsable,
  listTemplates,
  previewMedia,
  showError,
  showWarning,
  showSuccess,
  recordClientDiagnostic,
} = vi.hoisted(() => ({
  listMyAccounts: vi.fn(),
  batchImportMyAccounts: vi.fn(),
  deleteMyAccount: vi.fn(),
  batchDeleteMyAccounts: vi.fn(),
  updateMyAccount: vi.fn(),
  submitTask: vi.fn(),
  setDefaultProxy: vi.fn(),
  batchSetDefaultProxy: vi.fn(),
  exportMyAccounts: vi.fn(),
  listUsable: vi.fn(),
  listTemplates: vi.fn(),
  previewMedia: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

const originalCreateObjectURL = globalThis.URL.createObjectURL
const originalRevokeObjectURL = globalThis.URL.revokeObjectURL

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.cancel': 'Cancel',
  'common.close': 'Close',
  'common.confirm': 'Confirm',
  'common.none': 'None',
  'common.no': 'No',
  'common.processing': 'Processing',
  'common.refresh': 'Refresh',
  'common.retry': 'Retry',
  'common.save': 'Save',
  'common.saving': 'Saving',
  'common.success': 'Success',
  'common.unknown': 'Unknown',
  'common.yes': 'Yes',
  'common.error': 'Error',
  'accountWorkbench.failedToLoad': 'Failed to load',
  'accountWorkbench.dependencyLoadWarning': 'Dependencies unavailable',
  'accountWorkbench.searchPlaceholder': 'Search accounts',
  'accountWorkbench.filters.all': 'All',
  'accountWorkbench.filters.allPlatforms': 'All platforms',
  'accountWorkbench.filters.clear': 'Clear filters',
  'accountWorkbench.exportAccounts': 'Export accounts',
  'accountWorkbench.deleteSelected': 'Delete selected',
  'accountWorkbench.deleteOne': 'Delete account',
  'accountWorkbench.noResults.title': 'No results',
  'accountWorkbench.noResults.description': 'Adjust your filters.',
  'accountWorkbench.empty.title': 'No accounts',
  'accountWorkbench.empty.description': 'Import accounts to get started.',
  'accountWorkbench.selection.selectedCount': '{count} selected',
  'accountWorkbench.proxy.batchAction': 'Batch proxy',
  'accountWorkbench.proxy.action': 'Proxy',
  'accountWorkbench.proxy.title': 'Proxy',
  'accountWorkbench.proxy.batchTitle': 'Batch proxy',
  'accountWorkbench.proxy.modeSpecific': 'Specific proxy',
  'accountWorkbench.proxy.modeSpecificHint': 'Assign one proxy.',
  'accountWorkbench.proxy.modeRandom': 'Random proxy',
  'accountWorkbench.proxy.modeRandomHint': 'Spread accounts randomly.',
  'accountWorkbench.proxy.modeClear': 'Clear proxy',
  'accountWorkbench.proxy.modeClearHint': 'Remove the default proxy snapshot.',
  'accountWorkbench.proxy.hint': 'Choose an online proxy.',
  'accountWorkbench.proxy.configured': 'Configured',
  'accountWorkbench.proxy.notConfigured': 'Not configured',
  'accountWorkbench.proxy.summaryTotal': 'Total',
  'accountWorkbench.proxy.summaryMode': 'Mode',
  'accountWorkbench.proxy.summaryOnline': 'Online',
  'accountWorkbench.proxy.summarySelected': 'Selected proxy',
  'accountWorkbench.proxy.resultTitle': 'Proxy result',
  'accountWorkbench.proxy.resultRowsMore': '{count} more',
  'accountWorkbench.proxy.savedWithSummary': 'Proxy saved',
  'accountWorkbench.proxy.selectPlaceholder': 'Select proxy',
  'accountWorkbench.proxy.noOnlineProxies': 'No online proxies',
  'accountWorkbench.proxy.selectAccountsFirst': 'Select accounts first',
  'accountWorkbench.proxy.selectOnlineProxyFirst': 'Select an online proxy first',
  'accountWorkbench.proxy.failed': 'Proxy failed',
  'accountWorkbench.proxy.apply': 'Apply proxy',
  'accountWorkbench.proxy.currentAccount': 'Current account: {name}',
  'accountWorkbench.proxy.batchHint': 'Apply to {count} accounts.',
  'accountWorkbench.proxy.resultSummary': 'Total {total}, succeeded {succeeded}, failed {failed}, skipped {skipped}.',
  'accountWorkbench.proxy.dependenciesUnavailable': 'Proxy dependency unavailable',
  'accountWorkbench.proxy.modes.specific': 'Specific',
  'accountWorkbench.proxy.modes.random': 'Random',
  'accountWorkbench.proxy.modes.clear': 'Clear',
  'accountWorkbench.import.batchAction': 'Batch import',
  'accountWorkbench.import.batchTitle': 'Batch import',
  'accountWorkbench.import.batchHint': 'Paste text or choose a file.',
  'accountWorkbench.import.defaultPlatform': 'Platform',
  'accountWorkbench.import.fileLabel': 'File',
  'accountWorkbench.import.fileEmpty': 'No file selected',
  'accountWorkbench.import.clearSource': 'Clear source',
  'accountWorkbench.import.previewScopeUser': 'Preview is scoped to your workbench.',
  'accountWorkbench.import.submitReady': 'Ready',
  'accountWorkbench.import.pendingCount': 'Pending',
  'accountWorkbench.import.invalidCount': 'Invalid',
  'accountWorkbench.import.previewTitle': 'Import preview',
  'accountWorkbench.import.previewMeta': 'Valid {valid} / invalid {invalid}',
  'accountWorkbench.import.resultTitle': 'Import result',
  'accountWorkbench.import.resultRowsMore': '{count} more',
  'accountWorkbench.import.batchSuccess': 'Imported {count}',
  'accountWorkbench.import.batchFailed': 'Import failed',
  'accountWorkbench.import.errors.accountRequired': 'Account required',
  'accountWorkbench.import.errors.passwordRequired': 'Password required',
  'accountWorkbench.import.errors.credentialRequired': 'Credential required',
  'accountWorkbench.import.errors.invalidExecutionAuth': 'Execution auth must include access_token and token_secret',
  'accountWorkbench.import.errors.duplicateAccount': 'Duplicate account',
  'accountWorkbench.import.errors.duplicateInWorkbench': 'Already exists',
  'accountWorkbench.import.errors.unsupportedFile': 'Unsupported file',
  'accountWorkbench.import.errors.fileReadFailed': 'File read failed',
  'accountWorkbench.import.status.batchDuplicate': 'Batch duplicate',
  'accountWorkbench.import.status.existingWorkbenchDuplicate': 'Existing duplicate',
  'accountWorkbench.import.status.needsData': 'Needs data',
  'accountWorkbench.import.status.duplicate': 'Duplicate',
  'accountWorkbench.import.status.skipped': 'Skipped',
  'accountWorkbench.import.credentials.password': 'Password',
  'accountWorkbench.import.credentials.twoFactor': '2FA',
  'accountWorkbench.import.credentials.email': 'Email',
  'accountWorkbench.import.credentials.authCookie': 'Auth cookie',
  'accountWorkbench.import.credentials.executionAuth': 'Execution auth',
  'accountWorkbench.accountStatus.available': 'Available',
  'accountWorkbench.accountStatus.pending_check': 'Pending check',
  'accountWorkbench.accountStatus.limited': 'Limited',
  'accountWorkbench.accountStatus.invalid': 'Invalid',
  'accountWorkbench.accountStatus.not_stored': 'Not stored',
  'accountWorkbench.taskStatus.idle': 'Idle',
  'accountWorkbench.taskStatus.stored': 'Stored',
  'accountWorkbench.stats.assigned': 'Assigned',
  'accountWorkbench.stats.assignedMeta': 'Assigned accounts',
  'accountWorkbench.stats.executable': 'Executable',
  'accountWorkbench.stats.executableMeta': 'Ready accounts',
  'accountWorkbench.stats.selected': 'Selected',
  'accountWorkbench.stats.selectedMeta': 'Current selection',
  'accountWorkbench.stats.abnormal': 'Abnormal',
  'accountWorkbench.stats.abnormalMeta': 'Needs attention',
  'accountWorkbench.columns.name': 'Name',
  'accountWorkbench.columns.platform': 'Platform',
  'accountWorkbench.columns.accountStatus': 'Account status',
  'accountWorkbench.columns.taskStatus': 'Task status',
  'accountWorkbench.columns.proxy': 'Proxy',
  'accountWorkbench.columns.updatedAt': 'Updated at',
  'accountWorkbench.columns.username': 'Username',
  'accountWorkbench.columns.platformUserId': 'Platform user ID',
  'accountWorkbench.actions.follow': 'Follow',
  'accountWorkbench.actions.like': 'Like',
  'accountWorkbench.actions.retweet': 'Retweet',
  'accountWorkbench.actions.post': 'Post',
  'accountWorkbench.actions.update_profile': 'Update profile',
  'accountWorkbench.actions.update_avatar': 'Update avatar',
  'accountWorkbench.actions.update_banner': 'Update banner',
  'accountWorkbench.actions.message': 'Message',
  'accountWorkbench.actions.messageUnavailable': 'Message unavailable',
  'accountWorkbench.execution.noTemplates': 'No templates',
  'accountWorkbench.execution.defaultTemplate': 'Default',
  'accountWorkbench.execution.start': 'Submit task',
  'accountWorkbench.execution.confirmTitle': 'Confirm task execution',
  'accountWorkbench.execution.confirmHint': 'Submit {count} account(s) with template {template}.',
  'accountWorkbench.execution.confirmSubmit': 'Submit task',
  'accountWorkbench.execution.templateType': 'Template type',
  'accountWorkbench.execution.targets': 'Targets',
  'accountWorkbench.execution.contents': 'Contents',
  'accountWorkbench.execution.profileFields': 'Profile fields',
  'accountWorkbench.execution.media': 'Media',
  'accountWorkbench.execution.templateDetails': 'Template details',
  'accountWorkbench.execution.accountSummary': 'Account summary',
  'accountWorkbench.execution.loginCheckSummary': 'No extra parameters.',
  'accountWorkbench.execution.targetPoolSummary': '{count} target(s)',
  'accountWorkbench.execution.contentPoolSummary': '{count} content item(s)',
  'accountWorkbench.execution.postRichSummary': '{count} content item(s), {media} media item(s), quote link {quote}.',
  'accountWorkbench.execution.profileSummary': '{count} profile field(s)',
  'accountWorkbench.execution.avatarSummary': '1 avatar image ready.',
  'accountWorkbench.execution.bannerSummary': '1 banner image ready.',
  'accountWorkbench.execution.resultSummary': 'Submitted {submitted}; queued {enqueued}; failed {failed}.',
  'accountWorkbench.execution.failureNoChargeSummary': 'Failed tasks were not charged.',
  'accountWorkbench.execution.resultRows': 'Result rows',
  'accountWorkbench.execution.resultRowsMore': '{count} more',
  'accountWorkbench.execution.taskSummaryNoDetails': 'No details',
  'accountWorkbench.execution.taskStatuses.pending': 'Pending',
  'accountWorkbench.execution.taskStatuses.running': 'Running',
  'accountWorkbench.execution.taskStatuses.success': 'Success',
  'accountWorkbench.execution.taskStatuses.failed': 'Failed',
  'accountWorkbench.execution.chargeStatuses.charged': 'Charged',
  'accountWorkbench.execution.chargeStatuses.not_charged': 'Not charged',
  'accountWorkbench.execution.chargeStatuses.charge_failed': 'Charge failed',
  'accountWorkbench.execution.selectAccountsFirst': 'Select accounts first.',
  'accountWorkbench.execution.nonExecutableSelected': 'Non-executable account selected.',
  'accountWorkbench.execution.mixedPlatforms': 'Mixed platforms are not allowed.',
  'accountWorkbench.execution.platformUnavailable': 'Platform unavailable.',
  'accountWorkbench.execution.templateRequired': 'Template required.',
  'accountWorkbench.execution.templatesUnavailable': 'Templates unavailable.',
  'accountWorkbench.execution.templateInvalid': 'Template invalid.',
  'accountWorkbench.execution.submitFailed': 'Submit failed',
  'accountWorkbench.execution.submitted': 'Submitted {count}',
  'accountWorkbench.execution.templatePlaceholder': 'Select a task template',
  'accountWorkbench.deleteDialog.title': 'Delete accounts',
  'accountWorkbench.deleteDialog.accountSummary': 'Account summary',
  'accountWorkbench.deleteDialog.accountSummaryMore': '{count} more account(s)',
  'accountWorkbench.deleteDialog.impactHint': 'Deletion cannot be undone.',
  'accountWorkbench.deleteDialog.batchHint': 'Delete {count} accounts.',
  'accountWorkbench.deleteDialog.singleHint': 'Delete {name}.',
  'accountWorkbench.deleteDialog.confirmSingle': 'Delete account',
  'accountWorkbench.deleteDialog.confirmBatch': 'Delete {count} accounts',
  'accountWorkbench.deleteSuccess': 'Deleted {count}',
  'accountWorkbench.deleteFailed': 'Delete failed',
  'accountWorkbench.batchDeleteSuccess': 'Deleted {count}',
  'accountWorkbench.sections.managementHint': 'Manage SocialOps account credentials here.',
  'accountWorkbench.detailSections.identity': 'Identity',
  'accountWorkbench.detailSections.credentials': 'Credentials',
  'accountWorkbench.detailSections.operations': 'Operations',
  'accountWorkbench.edit.title': 'Edit account',
  'accountWorkbench.edit.identityTitle': 'Identity',
  'accountWorkbench.edit.identityHint': 'Identity fields are read-only.',
  'accountWorkbench.edit.saved': 'Saved',
  'accountWorkbench.edit.failed': 'Save failed',
  'admin.socialAccountWorkbench.executionBar.clear': 'Clear selection',
  'admin.socialAccountWorkbench.columns.id': 'ID',
  'admin.socialAccountWorkbench.columns.registrationIp': 'Registration IP',
  'admin.socialAccountWorkbench.columns.defaultProxySnapshot': 'Default proxy snapshot',
  'admin.socialAccountWorkbench.columns.password': 'Password',
  'admin.socialAccountWorkbench.columns.phone': 'Phone',
  'admin.socialAccountWorkbench.columns.email': 'Email',
  'admin.socialAccountWorkbench.columns.emailPassword': 'Email password',
  'admin.socialAccountWorkbench.columns.twoFactor': '2FA',
  'admin.socialAccountWorkbench.columns.backupCode': 'Backup code',
  'admin.socialAccountWorkbench.columns.emailClientId': 'Email client ID',
  'admin.socialAccountWorkbench.columns.emailToken': 'Email token',
  'admin.socialAccountWorkbench.columns.authCookie': 'Auth cookie',
  'admin.socialAccountWorkbench.columns.executionAuth': 'Execution auth',
  'admin.socialAccountWorkbench.detailTitle': 'Account details',
  'admin.socialAccountWorkbench.form.password': 'Password',
  'admin.socialAccountWorkbench.form.phone': 'Phone',
  'admin.socialAccountWorkbench.form.email': 'Email',
  'admin.socialAccountWorkbench.form.emailPassword': 'Email password',
  'admin.socialAccountWorkbench.form.twoFactor': '2FA',
  'admin.socialAccountWorkbench.form.backupCode': 'Backup code',
  'admin.socialAccountWorkbench.form.emailClientId': 'Email client ID',
  'admin.socialAccountWorkbench.form.emailToken': 'Email token',
  'admin.socialAccountWorkbench.form.authCookie': 'Auth cookie',
  'admin.socialAccountWorkbench.form.executionAuth': 'Execution auth',
  'admin.socialAccountWorkbench.form.remark': 'Remark',
}

vi.mock('@/api/accountWorkbench', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/accountWorkbench')>()
  return {
    ...actual,
    default: {
      ...actual.default,
      listMyAccounts,
      batchImportMyAccounts,
      deleteMyAccount,
      batchDeleteMyAccounts,
      updateMyAccount,
      submitTask,
      setDefaultProxy,
      batchSetDefaultProxy,
      exportMyAccounts,
    },
  }
})

vi.mock('@/api/proxies', () => ({
  default: {
    listUsable,
  },
}))

vi.mock('@/api/taskSettings', () => ({
  default: {
    listTemplates,
    previewMedia,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showWarning,
    showSuccess,
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        let value = messages[key] ?? key
        Object.entries(params ?? {}).forEach(([name, replacement]) => {
          value = value.replace(`{${name}}`, String(replacement))
        })
        return value
      },
    }),
  }
})

const SearchInputStub = defineComponent({
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  methods: {
    emitValue(event: Event) {
      this.$emit('update:modelValue', (event.target as HTMLInputElement).value)
    },
  },
  template: `
    <input
      :value="modelValue"
      :placeholder="placeholder"
      data-testid="search-input-stub"
      @input="emitValue"
    />
  `,
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: null },
    options: { type: Array, default: () => [] },
    placeholder: { type: String, default: '' },
    emptyText: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  methods: {
    emitValue(event: Event) {
      const rawValue = (event.target as HTMLSelectElement).value
      this.$emit('update:modelValue', rawValue === '__null__' ? null : rawValue)
    },
  },
  template: `
    <select
      v-bind="$attrs"
      data-testid="select-stub"
      :value="modelValue == null ? '__null__' : String(modelValue)"
      @change="emitValue"
    >
      <option value="__null__">{{ placeholder || emptyText }}</option>
      <option
        v-for="option in options"
        :key="String(option.value)"
        :value="option.value == null ? '__null__' : String(option.value)"
        :disabled="Boolean(option.disabled)"
      >
        {{ option.label }}
      </option>
    </select>
  `,
})

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
  },
  methods: {
    rowId(row: unknown) {
      return (row as { id?: number | string } | null)?.id ?? ''
    },
    rowValue(row: unknown, key: string) {
      const record = row as Record<string, unknown> | null
      return record?.[key]
    },
  },
  template: `
    <div data-testid="data-table-stub">
      <div data-testid="data-table-header">
        <slot name="header-select" />
      </div>
      <div v-for="row in data" :key="rowId(row)" :data-testid="\`account-row-\${rowId(row)}\`">
        <slot name="cell-select" :row="row" />
        <slot name="cell-name" :row="row" />
        <slot name="cell-platform" :row="row" :value="rowValue(row, 'platform')" />
        <slot name="cell-accountStatus" :row="row" :value="rowValue(row, 'accountStatus')" />
        <slot name="cell-taskStatus" :row="row" :value="rowValue(row, 'taskStatus')" />
        <slot name="cell-proxy" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
      <slot v-if="!loading && data.length === 0" name="empty" />
    </div>
  `,
})

function buildAccount(id = 101) {
  return {
    id,
    name: `x-main-${id}`,
    platform: 'x_twitter',
    username: `x-main-${id}`,
    platform_user_id: `uid-${id}`,
    password: 'secret',
    account_status: 'available',
    task_status: 'idle',
    default_proxy_configured: true,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  }
}

function buildPostTemplate() {
  return {
    id: 'post-template',
    name: 'Post template',
    type: 'post',
    is_default: true,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    params: {
      contents: ['Launch update'],
      media: [{
        source: 'library',
        storage_key: 'social-task/42/post.png',
        file_name: 'post.png',
        content_type: 'image/png',
        width: 1200,
        height: 675,
      }],
    },
  }
}

function buildVideoPostTemplate() {
  return {
    id: 'video-post-template',
    name: 'Video post template',
    type: 'post',
    is_default: true,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    params: {
      contents: ['Launch video update'],
      media: [{
        source: 'library',
        storage_key: 'social-task/42/post-video.mp4',
        file_name: 'post-video.mp4',
        content_type: 'video/mp4',
      }],
    },
  }
}

function buildAvatarTemplate(
  storageKey = 'social-task/42/avatar.png',
  id = 'avatar-template',
  width = 400,
  height = 400,
) {
  return {
    id,
    name: id === 'stale-avatar-template' ? 'Stale avatar template' : 'Avatar template',
    type: 'update_avatar',
    is_default: false,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    params: {
      avatar: {
        source: 'library',
        storage_key: storageKey,
        file_name: 'avatar.png',
        content_type: 'image/png',
        width,
        height,
      },
    },
  }
}

function buildBannerTemplate(
  storageKey = 'social-task/42/banner.png',
  id = 'banner-template',
  width = 1500,
  height = 500,
) {
  return {
    id,
    name: 'Banner template',
    type: 'update_banner',
    is_default: false,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    params: {
      banner: {
        source: 'library',
        storage_key: storageKey,
        file_name: 'banner.png',
        content_type: 'image/png',
        width,
        height,
      },
    },
  }
}

function mountView() {
  return mount(UnifiedAccountWorkbenchView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
        BaseDialog: { props: ['show', 'title'], template: '<section v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>' },
        DataTable: DataTableStub,
        SearchInput: SearchInputStub,
        Select: SelectStub,
        Icon: true,
      },
    },
  })
}

async function waitForWorkbench() {
  await flushPromises()
  await flushPromises()
}

function findTemplateSelect(wrapper: ReturnType<typeof mount>) {
  const select = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('Post template') || node.text().includes('Avatar template'))
  expect(select).toBeTruthy()
  return select!
}

function findExecutionStartButton(wrapper: ReturnType<typeof mount>) {
  const button = wrapper.findAll('button').find(node => node.text().includes('Submit task'))
  expect(button).toBeTruthy()
  return button!
}

describe('accounts UnifiedAccountWorkbenchView', () => {
  beforeEach(() => {
    listMyAccounts.mockReset()
    batchImportMyAccounts.mockReset()
    deleteMyAccount.mockReset()
    batchDeleteMyAccounts.mockReset()
    updateMyAccount.mockReset()
    submitTask.mockReset()
    setDefaultProxy.mockReset()
    batchSetDefaultProxy.mockReset()
    exportMyAccounts.mockReset()
    listUsable.mockReset()
    listTemplates.mockReset()
    previewMedia.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    showSuccess.mockReset()
    recordClientDiagnostic.mockReset()

    let blobURLCounter = 0
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(() => `blob:workbench-preview-${++blobURLCounter}`),
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(),
    })

    listMyAccounts.mockResolvedValue({
      items: [buildAccount()],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listUsable.mockResolvedValue([])
    previewMedia.mockResolvedValue(new Blob(['preview'], { type: 'image/png' }))
  })

  afterEach(() => {
    globalThis.URL.createObjectURL = originalCreateObjectURL
    globalThis.URL.revokeObjectURL = originalRevokeObjectURL
  })

  it('previews stored post media in the selected-template summary and confirm dialog', async () => {
    listTemplates.mockResolvedValue([buildPostTemplate()])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/post.png')

    const toolbarPreview = wrapper.get('[data-testid="selected-template-preview-post-0"]')
    expect(toolbarPreview.attributes('src')).toBe('blob:workbench-preview-1')
    expect(wrapper.text()).not.toContain('social-task/42/post.png')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmPreview = wrapper.get('[data-testid="execution-confirm-preview-post-0"]')
    expect(confirmPreview.attributes('src')).toBe('blob:workbench-preview-1')
    expect(wrapper.text()).toContain('Confirm task execution')
  })

  it('submits media-only post templates from the workbench when stored image media is present', async () => {
    listTemplates.mockResolvedValue([{
      id: 'media-only-post-template',
      name: 'Media only post template',
      type: 'post',
      is_default: true,
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-01T00:00:00Z',
      params: {
        contents: [],
        media: [{
          source: 'library',
          storage_key: 'social-task/42/media-only-post.png',
          file_name: 'media-only-post.png',
          content_type: 'image/png',
          width: 1200,
          height: 675,
        }],
      },
    }])
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7001,
        social_account_id: 101,
        action: 'post',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: '2026-06-07T00:00:00Z',
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/media-only-post.png')
    expect(wrapper.get('[data-testid="selected-template-preview-post-0"]').attributes('src')).toBe('blob:workbench-preview-1')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      template_id: 'media-only-post-template',
    })
    expect(typeof submitTask.mock.calls[0]?.[0]?.client_request_id).toBe('string')
    expect(String(submitTask.mock.calls[0]?.[0]?.client_request_id || '')).not.toBe('')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('treats stored single-mp4 post templates as selectable and previews them in the summary and confirm dialog', async () => {
    previewMedia.mockResolvedValue(new Blob(['video'], { type: 'video/mp4' }))
    listTemplates.mockResolvedValue([buildVideoPostTemplate()])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/post-video.mp4')

    const toolbarPreview = wrapper.get('[data-testid="selected-template-preview-post-0"]')
    expect(toolbarPreview.element.tagName).toBe('VIDEO')
    expect(toolbarPreview.attributes('src')).toBe('blob:workbench-preview-1')
    expect(wrapper.text()).toContain('1 media item(s)')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmPreview = wrapper.get('[data-testid="execution-confirm-preview-post-0"]')
    expect(confirmPreview.element.tagName).toBe('VIDEO')
    expect(confirmPreview.attributes('src')).toBe('blob:workbench-preview-1')
    expect(wrapper.text()).toContain('Confirm task execution')
  })

  it('previews stored avatar and banner templates, skips stale external refs, and revokes blob urls on switch', async () => {
    listTemplates.mockResolvedValue([
      {
        ...buildAvatarTemplate(),
        is_default: true,
      },
      buildBannerTemplate(),
      buildAvatarTemplate('media/avatar.png', 'stale-avatar-template'),
    ])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(wrapper.get('[data-testid="selected-template-preview-avatar"]').attributes('src')).toBe('blob:workbench-preview-1')

    const templateSelect = findTemplateSelect(wrapper)
    await templateSelect.setValue('banner-template')
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/banner.png')
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalledWith('blob:workbench-preview-1')
    expect(wrapper.get('[data-testid="selected-template-preview-banner"]').attributes('src')).toBe('blob:workbench-preview-2')

    await templateSelect.setValue('stale-avatar-template')
    await waitForWorkbench()

    expect(previewMedia).not.toHaveBeenCalledWith('media/avatar.png')
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalledWith('blob:workbench-preview-2')
    expect(wrapper.find('[data-testid="selected-template-preview-avatar"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="selected-template-preview-banner"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('media/avatar.png')
  })

  it('keeps avatar and banner templates selectable when stored images need normalization', async () => {
    listTemplates.mockResolvedValue([
      {
        ...buildAvatarTemplate('social-task/42/avatar.png', 'avatar-needs-normalization', 300, 300),
        is_default: true,
      },
      buildBannerTemplate('social-task/42/banner.png', 'banner-needs-normalization', 1200, 500),
    ])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(wrapper.get('[data-testid="selected-template-preview-avatar"]').attributes('src')).toBe('blob:workbench-preview-1')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')

    const templateSelect = findTemplateSelect(wrapper)
    await templateSelect.setValue('banner-needs-normalization')
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/banner.png')
    expect(wrapper.get('[data-testid="selected-template-preview-banner"]').attributes('src')).toBe('blob:workbench-preview-2')
  })

  it('blocks batch import preview rows when Twitter execution_auth is invalid', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@bad_auth\tpw\ttotp\t\t\t\t\t\t\tct0=ok\t{"access_token":"only"}')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Execution auth must include access_token and token_secret')
    expect(wrapper.text()).toContain('No')
    expect(wrapper.text()).toContain('Invalid1')
    expect(wrapper.text()).not.toContain('Not stored')
  })
})
