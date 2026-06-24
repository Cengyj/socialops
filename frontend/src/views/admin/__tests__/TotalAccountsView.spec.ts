import { defineComponent, nextTick } from 'vue'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { DOMWrapper, VueWrapper } from '@vue/test-utils'

import TotalAccountsView from '../TotalAccountsView.vue'

enableAutoUnmount(afterEach)

const {
  listUsers,
  listTotalAccounts,
  batchAssign,
  batchReclaim,
  batchDelete,
  exportTotalAccounts,
  importAccounts,
  updateAccount,
  showSuccess,
  showWarning,
  showError,
  recordClientDiagnostic,
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  listTotalAccounts: vi.fn(),
  batchAssign: vi.fn(),
  batchReclaim: vi.fn(),
  batchDelete: vi.fn(),
  exportTotalAccounts: vi.fn(),
  importAccounts: vi.fn(),
  updateAccount: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

const originalClipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
const originalSecureContextDescriptor = Object.getOwnPropertyDescriptor(window, 'isSecureContext')
const writeClipboard = vi.fn()

const messages: Record<string, string> = {
  'common.close': 'Close',
  'common.cancel': 'Cancel',
  'common.edit': 'Edit',
  'common.confirm': 'Confirm',
  'common.processing': 'Processing',
  'common.refresh': 'Refresh',
  'common.retry': 'Retry',
  'common.success': 'Success',
  'common.error': 'Error',
  'accountWorkbench.credentials.title': 'Credential Details',
  'accountWorkbench.credentials.previewHint': 'Long credential values are summarized by character count here.',
  'accountWorkbench.credentials.authCookieDescription': 'Login auth cookie.',
  'accountWorkbench.credentials.executionAuthDescription': 'Encrypted execution auth.',
  'accountWorkbench.credentials.copy': 'Copy',
  'accountWorkbench.credentials.copyRaw': 'Copy raw {field}',
  'accountWorkbench.credentials.copied': 'Credential copied.',
  'accountWorkbench.credentials.copyFailed': 'Copy failed',
  'accountWorkbench.credentials.emptyCopy': 'Empty field',
  'accountWorkbench.credentials.empty': 'Not configured',
  'accountWorkbench.credentials.length': '{count} chars',
  'accountWorkbench.credentials.encryptedStored': 'Encrypted value stored',
  'accountWorkbench.credentials.oauthReady': 'OAuth complete',
  'accountWorkbench.credentials.rawCookieDetected': 'Raw cookie',
  'accountWorkbench.credentials.oauthPartial': 'OAuth incomplete',
  'accountWorkbench.credentials.jsonDetected': 'JSON detected',
  'accountWorkbench.credentials.loginRefreshRequired': 'Login refresh required to capture execution auth',
  'accountWorkbench.proxy.configured': 'Configured',
  'accountWorkbench.proxy.notConfigured': 'Not configured',
  'accountWorkbench.detailSections.identity': 'Identity',
  'accountWorkbench.detailSections.credentials': 'Credentials',
  'accountWorkbench.detailSections.operations': 'Operations',
  'accountWorkbench.edit.identityTitle': 'Account identity',
  'accountWorkbench.edit.identityHint': 'Identity fields are controlled by the total pool record.',
  'accountWorkbench.columns.username': 'Username',
  'admin.socialAccountWorkbench.detailTitle': 'Account detail',
  'admin.socialAccountWorkbench.editTitle': 'Edit account',
  'admin.socialAccountWorkbench.failedToLoad': 'Failed to load',
  'admin.socialAccountWorkbench.searchPlaceholder': 'Search accounts',
  'admin.socialAccountWorkbench.tabs.poolDescription': 'Total account pool',
  'admin.socialAccountWorkbench.stats.total': 'Total',
  'admin.socialAccountWorkbench.stats.available': 'Available',
  'admin.socialAccountWorkbench.stats.assigned': 'Assigned',
  'admin.socialAccountWorkbench.stats.unassigned': 'Unassigned',
  'admin.socialAccountWorkbench.columns.id': 'ID',
  'admin.socialAccountWorkbench.columns.account': 'Account',
  'admin.socialAccountWorkbench.columns.platform': 'Platform',
  'admin.socialAccountWorkbench.columns.platformUserId': 'Platform ID',
  'admin.socialAccountWorkbench.columns.password': 'Password',
  'admin.socialAccountWorkbench.columns.phone': 'Phone',
  'admin.socialAccountWorkbench.columns.email': 'Email',
  'admin.socialAccountWorkbench.columns.emailPassword': 'Email password',
  'admin.socialAccountWorkbench.columns.twoFactor': '2FA',
  'admin.socialAccountWorkbench.columns.backupCode': 'Backup code',
  'admin.socialAccountWorkbench.columns.emailClientId': 'Email client ID',
  'admin.socialAccountWorkbench.columns.emailToken': 'Email token',
  'admin.socialAccountWorkbench.columns.authCookie': 'Auth cookie',
  'admin.socialAccountWorkbench.columns.registrationIp': 'Registration IP',
  'admin.socialAccountWorkbench.columns.executionAuth': 'Execution auth',
  'admin.socialAccountWorkbench.columns.defaultProxySnapshot': 'Default proxy',
  'admin.socialAccountWorkbench.columns.accountStatus': 'Account status',
  'admin.socialAccountWorkbench.columns.assignedUser': 'Assigned user',
  'admin.socialAccountWorkbench.columns.createdAt': 'Created at',
  'admin.socialAccountWorkbench.columns.actions': 'Actions',
  'admin.socialAccountWorkbench.rowActions.detail': 'Account detail',
  'admin.socialAccountWorkbench.form.password': 'Password',
  'admin.socialAccountWorkbench.form.phone': 'Phone',
  'admin.socialAccountWorkbench.form.email': 'Email',
  'admin.socialAccountWorkbench.form.emailPassword': 'Email password',
  'admin.socialAccountWorkbench.form.twoFactor': '2FA',
  'admin.socialAccountWorkbench.form.backupCode': 'Backup code',
  'admin.socialAccountWorkbench.form.emailClientId': 'Email client ID',
  'admin.socialAccountWorkbench.form.emailToken': 'Email token',
  'admin.socialAccountWorkbench.form.registrationIp': 'Registration IP',
  'admin.socialAccountWorkbench.form.authCookie': 'Auth cookie',
  'admin.socialAccountWorkbench.form.executionAuth': 'Execution auth',
  'admin.socialAccountWorkbench.form.accountStatus': 'Account status',
  'admin.socialAccountWorkbench.form.remark': 'Remark',
  'admin.socialAccountWorkbench.saved': 'Saved',
  'admin.socialAccountWorkbench.saveFailed': 'Failed to save account',
  'admin.socialAccountWorkbench.noChanges': 'No changes to save.',
  'admin.socialAccountWorkbench.assignFailed': 'Failed to assign accounts',
  'admin.socialAccountWorkbench.reclaimFailed': 'Failed to reclaim accounts',
  'admin.socialAccountWorkbench.deleteFailed': 'Failed to delete accounts',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE': 'Account service is temporarily unavailable. Try again later.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED': 'Account details are incomplete. Check the form and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NAME_REQUIRED': 'Enter an account name.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PLATFORM_REQUIRED': 'Choose an account platform.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IDENTITY_REQUIRED': 'Account identity is incomplete. Check the account name and platform.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED': 'Enter the account password.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_REQUIRED': 'Choose a valid import file and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE': 'Account delivery is incomplete. Provide a password and at least one login credential: 2FA, complete email credentials, or auth cookie.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_DUPLICATE': 'This account already exists in the total account pool.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND': 'The account no longer exists or was updated. Refresh the list and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ALREADY_ASSIGNED': 'This account has already been assigned to a user. Refresh the list and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED': 'The account assignment changed. Refresh the list and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID': 'Execution auth format is invalid.',
  'admin.socialAccountWorkbench.errors.USER_NOT_FOUND': 'The target user no longer exists or is unavailable. Refresh the user list and try again.',
  'admin.socialAccountWorkbench.assignment.all': 'All assignments',
  'admin.socialAccountWorkbench.assignment.assigned': 'Assigned',
  'admin.socialAccountWorkbench.assignment.unassigned': 'Unassigned',
  'admin.socialAccountWorkbench.accountStatus.available': 'Available',
  'admin.socialAccountWorkbench.accountStatus.pending_check': 'Pending',
  'admin.socialAccountWorkbench.accountStatus.limited': 'Limited',
  'admin.socialAccountWorkbench.accountStatus.invalid': 'Invalid',
  'admin.socialAccountWorkbench.accountStatus.not_stored': 'Not stored',
  'admin.socialAccountWorkbench.filters.allAccountStatus': 'All statuses',
  'admin.socialAccountWorkbench.filters.allAssignments': 'All assignments',
  'admin.socialAccountWorkbench.filters.assigned': 'Assigned',
  'admin.socialAccountWorkbench.filters.unassigned': 'Unassigned',
  'admin.socialAccountWorkbench.filters.clear': 'Clear filters',
  'admin.socialAccountWorkbench.empty.title': 'No accounts',
  'admin.socialAccountWorkbench.empty.description': 'Import accounts to get started.',
  'admin.socialAccountWorkbench.noResults.title': 'No results',
  'admin.socialAccountWorkbench.noResults.description': 'Adjust filters.',
  'admin.socialAccountWorkbench.toolbar.importAccounts': 'Import',
  'admin.socialAccountWorkbench.toolbar.exportRecords': 'Export',
  'admin.socialAccountWorkbench.toolbar.exportSelectedRecords': 'Export selected',
  'admin.socialAccountWorkbench.exportFailed': 'Export Failed',
  'admin.socialAccountWorkbench.importFailed': 'Import Failed',
  'admin.socialAccountWorkbench.imported': 'Imported {count} accounts.',
  'admin.socialAccountWorkbench.executionBar.selectedCount': '{count} selected',
  'admin.socialAccountWorkbench.executionBar.clear': 'Clear',
  'admin.socialAccountWorkbench.executionBar.selectionRequired': 'Select at least one account first.',
  'admin.socialAccountWorkbench.executionBar.noAssignedSelection': 'Select at least one assigned account to return to the pool.',
  'admin.socialAccountWorkbench.executionBar.accountSummaryMore': '{count} more',
  'admin.socialAccountWorkbench.actions.assign': 'Assign',
  'admin.socialAccountWorkbench.actions.reclaim': 'Reclaim',
  'admin.socialAccountWorkbench.actions.delete': 'Delete',
  'admin.socialAccountWorkbench.assignDialog.title': 'Assign accounts',
  'admin.socialAccountWorkbench.assignDialog.hint': 'Assign {count} accounts.',
  'admin.socialAccountWorkbench.assignDialog.accountSummary': 'Account summary',
  'admin.socialAccountWorkbench.assignDialog.accountSummaryMore': '{count} more',
  'admin.socialAccountWorkbench.assignDialog.targetUser': 'Target user',
  'admin.socialAccountWorkbench.assignDialog.searchPlaceholder': 'Search users',
  'admin.socialAccountWorkbench.assignDialog.userListLabel': 'Users',
  'admin.socialAccountWorkbench.assignDialog.userStatus.active': 'Active',
  'admin.socialAccountWorkbench.assignDialog.assignedCountLabel': '{count} visible assigned',
  'admin.socialAccountWorkbench.assignDialog.noUsersFound': 'No users found',
  'admin.socialAccountWorkbench.assignDialog.userLoadFailed': 'Failed to load target users.',
  'admin.socialAccountWorkbench.assignDialog.selectedUserLabel': 'Selected user',
  'admin.socialAccountWorkbench.assignDialog.noSelectedUserPrompt': 'Select a user.',
  'admin.socialAccountWorkbench.assignDialog.reviewButton': 'Review assignment',
  'admin.socialAccountWorkbench.assignDialog.confirmTitle': 'Confirm assignment',
  'admin.socialAccountWorkbench.assignDialog.confirmHint': 'Assign {count} accounts to {user}.',
  'admin.socialAccountWorkbench.assignDialog.impactHint': 'Assigned accounts move to the user workspace.',
  'admin.socialAccountWorkbench.assignDialog.backToSelect': 'Back',
  'admin.socialAccountWorkbench.assignDialog.confirm': 'Confirm assign',
  'admin.socialAccountWorkbench.reclaimDialog.title': 'Reclaim accounts',
  'admin.socialAccountWorkbench.reclaimDialog.hint': 'Reclaim {count} accounts.',
  'admin.socialAccountWorkbench.reclaimDialog.assignedImpact': 'Assigned accounts will be reclaimed.',
  'admin.socialAccountWorkbench.reclaimDialog.unassignedImpact': 'Unassigned accounts stay in the pool.',
  'admin.socialAccountWorkbench.reclaimDialog.accountSummary': 'Account summary',
  'admin.socialAccountWorkbench.reclaimDialog.confirm': 'Confirm reclaim',
  'admin.socialAccountWorkbench.deleteDialog.title': 'Delete accounts',
  'admin.socialAccountWorkbench.deleteDialog.hint': 'Delete {count} accounts.',
  'admin.socialAccountWorkbench.deleteDialog.accountSummary': 'Account summary',
  'admin.socialAccountWorkbench.deleteDialog.impactHint': 'Related task logs and proxy references are cleaned.',
  'admin.socialAccountWorkbench.deleteDialog.confirm': 'Confirm delete',
  'admin.socialAccountWorkbench.batchResult.dismiss': 'Clear batch operation result',
  'admin.socialAccountWorkbench.batchResult.rowsMore': '{count} more result(s)',
  'admin.socialAccountWorkbench.batchResult.statuses.succeeded': 'Succeeded',
  'admin.socialAccountWorkbench.batchResult.statuses.skipped': 'Skipped',
  'admin.socialAccountWorkbench.batchResult.statuses.failed': 'Failed',
  'admin.socialAccountWorkbench.batchResult.statuses.duplicate': 'Duplicate',
  'admin.socialAccountWorkbench.importResult.dismiss': 'Clear import result',
  'admin.socialAccountWorkbench.importResult.summary': 'Import result: total {total}, imported {created}, skipped {skipped}, failed {failed}, duplicates {duplicates}.',
  'accountWorkbench.import.status.duplicate': 'Duplicate',
  'accountWorkbench.import.status.skipped': 'Skipped',
  'accountWorkbench.import.resultReasons.duplicateInDatabase': 'Already exists in the total account pool',
  'accountWorkbench.import.resultReasons.invalidInput': 'Import data is invalid',
  'accountWorkbench.import.resultReasons.importFailed': 'Could not import this account',
  'admin.socialAccountWorkbench.toasts.assigned': 'Assigned {count} accounts to {user}.',
  'admin.socialAccountWorkbench.toasts.assignedResult': 'Assignment result: total {total}, succeeded {succeeded}, skipped {skipped}, failed {failed}. Target user: {user}.',
  'admin.socialAccountWorkbench.toasts.assignRequiresUnassigned': 'Assign only supports unassigned accounts. {count} selected accounts are already assigned; adjust the selection and try again.',
  'admin.socialAccountWorkbench.toasts.selectTargetUser': 'Please choose a target user.',
  'admin.socialAccountWorkbench.toasts.reclaimed': 'Reclaimed {count} accounts and marked them unassigned.',
  'admin.socialAccountWorkbench.toasts.reclaimedResult': 'Reclaim result: total {total}, succeeded {succeeded}, skipped {skipped}, failed {failed}.',
  'admin.socialAccountWorkbench.toasts.deleted': 'Deleted {count} accounts.',
  'admin.socialAccountWorkbench.toasts.deletedResult': 'Delete result: total {total}, succeeded {succeeded}, skipped {skipped}, failed {failed}.',
  'accountWorkbench.batchResultReasons.invalidId': 'Invalid account ID',
  'accountWorkbench.batchResultReasons.invalidInput': 'Invalid input',
  'accountWorkbench.batchResultReasons.duplicateInBatch': 'Duplicate in batch',
  'accountWorkbench.batchResultReasons.duplicateInDatabase': 'Already exists in the total account pool',
  'accountWorkbench.batchResultReasons.accountNotFound': 'Account not found',
  'accountWorkbench.batchResultReasons.accountNotAssigned': 'Account not assigned',
  'accountWorkbench.batchResultReasons.proxyNotAvailable': 'Selected proxy unavailable',
  'accountWorkbench.batchResultReasons.assignFailed': 'Assignment failed',
  'accountWorkbench.batchResultReasons.notFound': 'Record not found',
  'accountWorkbench.batchResultReasons.alreadyStored': 'Already stored',
  'accountWorkbench.batchResultReasons.invalidCredentials': 'Invalid credentials',
  'accountWorkbench.batchResultReasons.alreadyAssigned': 'Already assigned',
  'accountWorkbench.batchResultReasons.alreadyUnassigned': 'Already unassigned',
  'accountWorkbench.batchResultReasons.targetUserNotFound': 'Target user not found',
  'accountWorkbench.batchResultReasons.reclaimFailed': 'Reclaim failed',
  'accountWorkbench.batchResultReasons.deleteFailed': 'Delete failed',
  'accountWorkbench.batchResultReasons.createFailed': 'Create failed',
  'accountWorkbench.batchResultReasons.operationFailed': 'Could not process this account',
}

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

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: { list: listUsers },
    totalAccounts: {
      list: listTotalAccounts,
      batchAssign,
      batchReclaim,
      batchDelete,
      exportAccounts: exportTotalAccounts,
      importAccounts,
      update: updateAccount,
    },
    accountWorkbench: {
      exportAccounts: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showWarning,
    showError,
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic,
}))

const SearchInputStub = defineComponent({
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  template: '<input data-testid="search-input-stub" :value="modelValue" :placeholder="placeholder" @input="$emit(\'update:modelValue\', $event.target.value)" />',
})

const SelectStub = defineComponent({
  props: {
    modelValue: { type: [String, Number], default: '' },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  template: '<select data-testid="select-stub" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>',
})

const DataTableStub = defineComponent({
  props: {
    columns: { type: Array, default: () => [] },
    data: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    defaultSortKey: { type: String, default: undefined },
    defaultSortOrder: { type: String, default: undefined },
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
      <div v-for="row in data" :key="rowId(row)" :data-testid="\`total-account-row-\${rowId(row)}\`">
        <slot name="cell-select" :row="row" />
        <slot name="cell-id" :row="row" :value="rowValue(row, 'id')" />
        <slot name="cell-account" :row="row" />
        <slot name="cell-platform" :row="row" :value="rowValue(row, 'platform')" />
        <slot name="cell-email" :row="row" :value="rowValue(row, 'email')" />
        <slot name="cell-defaultProxySnapshot" :row="row" />
        <slot name="cell-accountStatus" :row="row" :value="rowValue(row, 'accountStatus')" />
        <slot name="cell-assignedUser" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
      <slot v-if="!loading && data.length === 0" name="empty" />
    </div>
  `,
})

function mountView() {
  return mount(TotalAccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<section v-if="show" role="dialog"><button type="button" aria-label="Close modal" @click="$emit(\'close\')">Close modal</button><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
        },
        DataTable: DataTableStub,
        SearchInput: SearchInputStub,
        Select: SelectStub,
        Icon: true,
      },
    },
  })
}

async function waitForView() {
  await flushPromises()
  await nextTick()
  await flushPromises()
  await nextTick()
}

async function waitForCondition(assertion: () => void) {
  let lastError: unknown
  const deadline = Date.now() + 1000
  while (Date.now() < deadline) {
    try {
      assertion()
      return
    } catch (error) {
      lastError = error
      await waitForView()
      await new Promise(resolve => setTimeout(resolve, 0))
    }
  }
  try {
    assertion()
  } catch {
    throw lastError
  }
}

function totalAccountFixture(overrides: Record<string, unknown> = {}) {
  return {
    id: 501,
    name: 'northwind_ops',
    platform: 'x_twitter',
    username: 'northwind_ops',
    platform_user_id: 'uid-501',
    password: 'plain-password',
    phone: '+15550001',
    email: 'ops@example.com',
    email_password: 'plain-email-password',
    two_factor: '123456',
    backup_code: 'backup',
    email_client_id: 'client-id',
    email_token: 'email-token',
    registration_ip: '203.0.113.10',
    auth_cookie: 'ct0=raw-cookie; auth_token=raw-token',
    execution_auth: 'encrypted-execution-auth-ciphertext',
    default_proxy_snapshot: 'proxy-1',
    account_status: 'available',
    task_status: 'idle',
    task_message: '',
    assigned_user_id: null,
    assigned_user_email: '',
    remark: 'remark',
    created_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

function userFixture() {
  return {
    id: 7,
    email: 'owner@example.com',
    username: 'owner',
    role: 'user',
    status: 'active',
  }
}

function batchResult(overrides: Record<string, unknown> = {}) {
  return {
    total: 1,
    succeeded: 1,
    skipped: 0,
    failed: 0,
    errors: [],
    items: [{ id: 501, status: 'succeeded' }],
    ...overrides,
  }
}

type ButtonSearchWrapper = VueWrapper | DOMWrapper<Element>

async function clickButtonByText(wrapper: ButtonSearchWrapper, text: string) {
  const button = wrapper.findAll('button').find(item => item.text().includes(text))
  expect(button, `button containing "${text}"`).toBeTruthy()
  await button!.trigger('click')
}

function getButtonByAriaLabel(wrapper: ReturnType<typeof mount>, label: string) {
  return wrapper.get(`button[aria-label="${label}"]`)
}

function getButtonByText(wrapper: ButtonSearchWrapper, text: string) {
  const button = wrapper.findAll('button').find(item => item.text().includes(text))
  expect(button, `button containing "${text}"`).toBeTruthy()
  return button!
}

async function clickLatestDialogClose(wrapper: ReturnType<typeof mount>) {
  const closeButtons = wrapper.findAll('button[aria-label="Close modal"]')
  expect(closeButtons.length, 'dialog close buttons').toBeGreaterThan(0)
  await closeButtons[closeButtons.length - 1].trigger('click')
}

function getEmptyStateButton(wrapper: ReturnType<typeof mount>, label: string) {
  const button = wrapper.findAll('.py-8.text-center button')
    .find(item => item.attributes('aria-label') === label)
  expect(button, `empty-state button "${label}"`).toBeTruthy()
  return button!
}

function expectConstrainedActionButton(button: ReturnType<typeof getButtonByAriaLabel>, label: string) {
  expect(button.attributes('aria-label')).toBe(label)
  expect(button.attributes('title')).toBe(label)
  expect(button.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
  const text = button.findAll('span').find(node => node.text() === label)
  expect(text, `button text "${label}"`).toBeTruthy()
  expect(text!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
}

async function triggerImportFile(wrapper: ReturnType<typeof mount>, file: File) {
  const input = wrapper.get('input[type="file"]')
  Object.defineProperty(input.element, 'files', {
    configurable: true,
    value: [file],
  })
  await input.trigger('change')
  return input
}

describe('admin TotalAccountsView credential preview', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    Object.defineProperty(window, 'isSecureContext', {
      configurable: true,
      value: true,
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: writeClipboard },
    })
    listUsers.mockResolvedValue({ items: [] })
  })

  afterEach(() => {
    if (originalClipboardDescriptor) {
      Object.defineProperty(navigator, 'clipboard', originalClipboardDescriptor)
    }
    if (originalSecureContextDescriptor) {
      Object.defineProperty(window, 'isSecureContext', originalSecureContextDescriptor)
    }
  })

  it('keeps total-pool load errors readable in the existing retry panel', async () => {
    const loadErrorMessage = 'The account no longer exists or was updated. Refresh the list and try again.'
    listTotalAccounts.mockRejectedValue({ reason: 'SOCIAL_ACCOUNT_NOT_FOUND' })

    const wrapper = mountView()
    await waitForView()

    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_accounts', expect.any(Object))
    expect(showError).toHaveBeenCalledWith(loadErrorMessage)
    expect(wrapper.text()).toContain('Failed to load')
    expect(wrapper.text()).toContain('Retry')
    const errorMessage = wrapper.findAll('p').find(node => node.text() === loadErrorMessage)
    expect(errorMessage).toBeTruthy()
    expect(errorMessage!.attributes('title')).toBe(loadErrorMessage)
    expect(errorMessage!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expectConstrainedActionButton(getButtonByAriaLabel(wrapper, 'Retry'), 'Retry')
  })

  it('maps total-pool service availability load errors to the existing retry panel', async () => {
    const loadErrorMessage = 'Account service is temporarily unavailable. Try again later.'
    listTotalAccounts.mockRejectedValue({ reason: 'SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE' })

    const wrapper = mountView()
    await waitForView()

    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_accounts', expect.any(Object))
    expect(showError).toHaveBeenCalledWith(loadErrorMessage)
    expect(wrapper.text()).toContain('Failed to load')
    const errorMessage = wrapper.findAll('p').find(node => node.text() === loadErrorMessage)
    expect(errorMessage).toBeTruthy()
    expect(errorMessage!.attributes('title')).toBe(loadErrorMessage)
  })

  it('shows default proxy state in list rows while keeping the raw snapshot inspectable', async () => {
    const snapshot = '{"id":301,"endpoint":"http://proxy.example:8080"}'
    listTotalAccounts.mockResolvedValue({
      items: [
        totalAccountFixture({ default_proxy_snapshot: snapshot }),
        totalAccountFixture({ id: 502, name: 'southwind_ops', username: 'southwind_ops', default_proxy_snapshot: '' }),
      ],
    })

    const wrapper = mountView()
    await waitForView()

    const configuredRow = wrapper.get('[data-testid="total-account-row-501"]')
    expect(configuredRow.text()).toContain('Configured')
    expect(configuredRow.text()).not.toContain('http://proxy.example:8080')
    const configuredProxyCell = configuredRow.findAll('span').find(node => node.attributes('title') === snapshot)
    expect(configuredProxyCell).toBeTruthy()
    expect(configuredProxyCell!.text()).toBe('Configured')

    const emptyRow = wrapper.get('[data-testid="total-account-row-502"]')
    expect(emptyRow.text()).toContain('Not configured')

    await configuredRow.get('button').trigger('click')
    await waitForView()

    expect(wrapper.text()).toContain('Account detail')
    expect(wrapper.text()).toContain('http://proxy.example:8080')
  })

  it('keeps edit identity hints readable and inspectable', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const editWrapper = mountView()
    await waitForView()

    await editWrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await waitForView()

    const identityHint = editWrapper.get('[title="Identity fields are controlled by the total pool record."]')
    expect(identityHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })

  it('keeps edit dialog footer actions inspectable and constrained on narrow layouts', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const editWrapper = mountView()
    await waitForView()
    await editWrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await waitForView()

    const editDialog = editWrapper.get('section[role="dialog"]')
    const editCancel = editDialog.get('button[aria-label="Cancel"]')
    expect(editCancel.attributes('title')).toBe('Cancel')
    expect(editCancel.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(editCancel.findAll('span').find(node => node.text() === 'Cancel')?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))

    const editConfirm = editDialog.get('button[aria-label="Confirm"]')
    expect(editConfirm.attributes('title')).toBe('No changes to save.')
    expect(editConfirm.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(editConfirm.findAll('span').find(node => node.text() === 'Confirm')?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(editConfirm.attributes('disabled')).toBeDefined()
    await editWrapper.get('#total-account-edit-email').setValue('changed@example.test')
    await nextTick()
    const changedEditConfirm = editDialog.get('button[aria-label="Confirm"]')
    expect(changedEditConfirm.attributes('title')).toBe('Confirm')
    expect(changedEditConfirm.attributes('disabled')).toBeUndefined()
  })

  it('keeps trimmed contact-only total-pool edit changes disabled while preserving delivery field edits', async () => {
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        password: 'pool-password',
        phone: '+15550001111',
        email: 'pool@example.test',
        registration_ip: '203.0.113.10',
        auth_cookie: 'ct0=pool; auth_token=pool',
        execution_auth: 'encrypted-pool-execution-auth',
        account_status: 'available',
        remark: 'pool note',
      })],
    })
    updateAccount.mockResolvedValue(totalAccountFixture({
      password: '  pool-password  ',
      phone: '+15550001111',
      email: 'pool@example.test',
      registration_ip: '203.0.113.10',
      auth_cookie: 'ct0=pool; auth_token=pool',
      execution_auth: 'encrypted-pool-execution-auth',
      account_status: 'available',
      remark: 'pool note',
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await waitForView()

    await wrapper.get('#total-account-edit-phone').setValue('  +15550001111  ')
    await wrapper.get('#total-account-edit-email').setValue('  pool@example.test  ')
    await wrapper.get('#total-account-edit-registration-ip').setValue('  203.0.113.10  ')
    await nextTick()

    const unchangedConfirm = wrapper.get('section[role="dialog"] button[aria-label="Confirm"]')
    expect(unchangedConfirm.attributes('disabled')).toBeDefined()
    expect(unchangedConfirm.attributes('title')).toBe('No changes to save.')
    await unchangedConfirm.trigger('click')
    await waitForView()

    expect(updateAccount).not.toHaveBeenCalled()

    await wrapper.get('#total-account-edit-password').setValue('  pool-password  ')
    await nextTick()

    const changedConfirm = wrapper.get('section[role="dialog"] button[aria-label="Confirm"]')
    expect(changedConfirm.attributes('disabled')).toBeUndefined()
    expect(changedConfirm.attributes('title')).toBe('Confirm')
    await changedConfirm.trigger('click')
    await waitForView()

    expect(updateAccount).toHaveBeenCalledWith(501, expect.objectContaining({
      password: '  pool-password  ',
      phone: '+15550001111',
      email: 'pool@example.test',
      registration_ip: '203.0.113.10',
      auth_cookie: 'ct0=pool; auth_token=pool',
      execution_auth: 'encrypted-pool-execution-auth',
      account_status: 'available',
      remark: 'pool note',
    }))
  })

  it('keeps edit dialog open while submit is pending', async () => {
    const updatedAccount = totalAccountFixture({
      email: 'pending-edit@example.test',
    })
    let resolveUpdate!: (value: ReturnType<typeof totalAccountFixture>) => void
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [updatedAccount] })
    updateAccount.mockReturnValueOnce(new Promise((resolve) => {
      resolveUpdate = resolve
    }))

    const editWrapper = mountView()
    await waitForView()

    await editWrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await editWrapper.get('#total-account-edit-email').setValue('pending-edit@example.test')
    await clickButtonByText(editWrapper, 'Confirm')
    await nextTick()

    expect(updateAccount).toHaveBeenCalledWith(501, expect.objectContaining({
      email: 'pending-edit@example.test',
    }))
    expect(editWrapper.get('#total-account-edit-email').attributes('disabled')).toBeDefined()
    expect(editWrapper.get('section[role="dialog"] button[aria-label="Processing"]').attributes('title')).toBe('Processing')
    expect(editWrapper.get('section[role="dialog"] button[aria-label="Processing"]').attributes('title')).toBe('Processing')
    await clickLatestDialogClose(editWrapper)
    await nextTick()
    expect(editWrapper.text()).toContain('Edit account')

    resolveUpdate(updatedAccount)
    await waitForView()

    expect(editWrapper.text()).not.toContain('Edit account')
    editWrapper.unmount()
  })

  it('summarizes long delivery credentials like accounts while copying full stored values', async () => {
    const emailToken = 'email-token-1234567890-secret'
    const authCookie = 'ct0=raw-cookie; auth_token=raw-token'
    const executionAuth = 'encrypted-execution-auth-ciphertext'
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        email_token: emailToken,
        auth_cookie: authCookie,
        execution_auth: executionAuth,
      })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] button').trigger('click')

    const credentialPreview = wrapper.get('[data-testid="total-account-credential-preview"]')
    expect(credentialPreview.text()).toContain('Credential Details')
    expect(credentialPreview.text()).toContain('Encrypted value stored')
    expect(credentialPreview.text()).not.toContain('Refresh')
    expect(wrapper.get('[data-testid="total-account-email-token-preview"]').text()).toContain(`${emailToken.length} chars`)
    expect(wrapper.get('[data-testid="total-account-email-token-preview"]').text()).not.toContain(emailToken)
    expect(wrapper.get('[data-testid="total-account-credential-authCookie"]').text()).toContain(`${authCookie.length} chars`)
    expect(wrapper.get('[data-testid="total-account-credential-authCookie"]').text()).not.toContain(authCookie)
    expect(wrapper.get('[data-testid="total-account-credential-executionAuth"]').text()).toContain(`${executionAuth.length} chars`)
    expect(wrapper.get('[data-testid="total-account-credential-executionAuth"]').text()).not.toContain(executionAuth)
    expect(wrapper.find('[data-testid="total-account-credential-authCookie-value"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="total-account-credential-executionAuth-value"]').exists()).toBe(false)

    await wrapper.get('[data-testid="total-account-email-token-copy"]').trigger('click')
    expect(writeClipboard).toHaveBeenLastCalledWith(emailToken)
    await wrapper.get('[data-testid="total-account-credential-authCookie-copy"]').trigger('click')
    expect(writeClipboard).toHaveBeenLastCalledWith(authCookie)
    await wrapper.get('[data-testid="total-account-credential-executionAuth-copy"]').trigger('click')
    expect(writeClipboard).toHaveBeenLastCalledWith(executionAuth)
    expect(showSuccess).toHaveBeenCalledWith('Credential copied.')
  })

  it('closes the assign confirmation when refreshed accounts no longer contain the selection', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await waitForView()
    expect(wrapper.text()).toContain('1 selected')
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    expect(wrapper.text()).toContain('Confirm assignment')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('Confirm assignment')
      expect(wrapper.text()).not.toContain('Assign accounts')
    })
    expect(batchAssign).not.toHaveBeenCalled()
  })

  it('closes the assign confirmation when a refreshed selected account is already assigned', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({
        items: [totalAccountFixture({
          assigned_user_id: 7,
          assigned_user_email: 'owner@example.com',
        })],
      })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    expect(wrapper.text()).toContain('Confirm assignment')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      const row = wrapper.get('[data-testid="total-account-row-501"]')
      expect(row.text()).toContain('owner@example.com')
      expect(wrapper.text()).not.toContain('Confirm assignment')
      expect(wrapper.text()).not.toContain('Assign accounts')
      const assignButton = getButtonByText(wrapper, 'Assign')
      expect(assignButton.attributes('aria-label')).toBe('Assign only supports unassigned accounts. 1 selected accounts are already assigned; adjust the selection and try again.')
      expect(assignButton.attributes('title')).toBe('Assign only supports unassigned accounts. 1 selected accounts are already assigned; adjust the selection and try again.')
      expect(assignButton.attributes('disabled')).toBeDefined()
    })
    expect(batchAssign).not.toHaveBeenCalled()
  })

  it('locks old total-pool row and assignment actions while a list refresh is pending', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    expect(wrapper.text()).toContain('Confirm assignment')

    await getButtonByAriaLabel(wrapper, 'Refresh').trigger('click')
    await wrapper.vm.$nextTick()

    const row = wrapper.get('[data-testid="total-account-row-501"]')
    expect((row.get('input[type="checkbox"]').element as HTMLInputElement).disabled).toBe(true)
    const rowActionButtons = row.findAll('.flex.items-center.justify-start.gap-2 button')
    expect(rowActionButtons).toHaveLength(2)
    const detailButton = rowActionButtons[0]
    const editButton = rowActionButtons[1]
    expect(editButton.attributes('disabled')).toBeDefined()
    expect(editButton.attributes('aria-label')).toBe('Processing')
    expect(editButton.attributes('title')).toBe('Processing')
    expect(detailButton.attributes('disabled')).toBeDefined()
    expect(detailButton.attributes('aria-label')).toBe('Processing')
    expect(detailButton.attributes('title')).toBe('Processing')

    const confirmAssignButton = wrapper.findAll('button').find(item => item.text().includes('Confirm assign'))
    expect(confirmAssignButton, 'confirm assignment button').toBeTruthy()
    expect(confirmAssignButton!.attributes('disabled')).toBeDefined()
    await confirmAssignButton!.trigger('click')
    await row.get('input[type="checkbox"]').trigger('change')
    await editButton.trigger('click')

    expect(batchAssign).not.toHaveBeenCalled()
    expect(updateAccount).not.toHaveBeenCalled()

    resolveRefresh({ items: [totalAccountFixture()] })
    await waitForView()
  })

  it('locks total-pool export while a list refresh is pending', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForView()

    await getButtonByAriaLabel(wrapper, 'Refresh').trigger('click')
    await wrapper.vm.$nextTick()

    const exportButton = getButtonByText(wrapper, 'Export')
    expect(exportButton.attributes('disabled')).toBeDefined()
    expect(exportButton.attributes('aria-label')).toBe('Processing')
    expect(exportButton.attributes('title')).toBe('Processing')

    await exportButton.trigger('click')
    expect(exportTotalAccounts).not.toHaveBeenCalled()

    resolveRefresh({ items: [totalAccountFixture()] })
    await waitForView()

    expect(getButtonByAriaLabel(wrapper, 'Export').attributes('disabled')).toBeUndefined()
  })

  it('locks total-pool import while a list refresh is pending', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForView()

    await getButtonByAriaLabel(wrapper, 'Refresh').trigger('click')
    await wrapper.vm.$nextTick()

    const importButton = getButtonByText(wrapper, 'Import')
    expect(importButton.attributes('disabled')).toBeDefined()
    expect(importButton.attributes('aria-label')).toBe('Processing')
    expect(importButton.attributes('title')).toBe('Processing')

    await importButton.trigger('click')
    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalledWith('Import Failed')

    resolveRefresh({ items: [totalAccountFixture()] })
    await waitForView()

    expect(getButtonByAriaLabel(wrapper, 'Import').attributes('disabled')).toBeUndefined()
  })

  it('keeps batch confirmation dialogs open while requests are submitting', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
    let resolveAssign!: (value: ReturnType<typeof batchResult>) => void
    batchAssign.mockReturnValueOnce(new Promise((resolve) => {
      resolveAssign = resolve
    }))

    const assignWrapper = mountView()
    await waitForView()

    await assignWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(assignWrapper, 'Assign')
    await clickButtonByText(assignWrapper, 'owner@example.com')
    await clickButtonByText(assignWrapper, 'Review assignment')
    await clickButtonByText(assignWrapper, 'Confirm assign')
    await nextTick()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    const pendingBackButton = getButtonByText(assignWrapper, 'Back')
    expect(pendingBackButton.attributes('disabled')).toBeDefined()
    expect(pendingBackButton.attributes('aria-label')).toBe('Processing')
    expect(pendingBackButton.attributes('title')).toBe('Processing')
    await clickLatestDialogClose(assignWrapper)
    await nextTick()
    expect(assignWrapper.text()).toContain('Confirm assignment')

    resolveAssign(batchResult())
    await waitForView()
    expect(assignWrapper.text()).not.toContain('Confirm assignment')
    assignWrapper.unmount()

    vi.clearAllMocks()
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [totalAccountFixture({ assigned_user_id: 7, assigned_user_email: 'owner@example.com' })],
      })
      .mockResolvedValueOnce({
        items: [totalAccountFixture({ assigned_user_id: 7, assigned_user_email: 'owner@example.com' })],
      })
    let resolveReclaim!: (value: ReturnType<typeof batchResult>) => void
    batchReclaim.mockReturnValueOnce(new Promise((resolve) => {
      resolveReclaim = resolve
    }))

    const reclaimWrapper = mountView()
    await waitForView()

    await reclaimWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(reclaimWrapper, 'Reclaim')
    await clickButtonByText(reclaimWrapper, 'Confirm reclaim')
    await nextTick()

    expect(batchReclaim).toHaveBeenCalledWith([501])
    const reclaimCancelButton = getButtonByText(reclaimWrapper, 'Cancel')
    expect(reclaimCancelButton.attributes('disabled')).toBeDefined()
    expect(reclaimCancelButton.attributes('aria-label')).toBe('Processing')
    expect(reclaimCancelButton.attributes('title')).toBe('Processing')
    await clickLatestDialogClose(reclaimWrapper)
    await nextTick()
    expect(reclaimWrapper.text()).toContain('Reclaim accounts')

    resolveReclaim(batchResult())
    await waitForView()
    expect(reclaimWrapper.text()).not.toContain('Reclaim accounts')
    reclaimWrapper.unmount()

    vi.clearAllMocks()
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
    let resolveDelete!: (value: ReturnType<typeof batchResult>) => void
    batchDelete.mockReturnValueOnce(new Promise((resolve) => {
      resolveDelete = resolve
    }))

    const deleteWrapper = mountView()
    await waitForView()

    await deleteWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(deleteWrapper, 'Delete')
    await clickButtonByText(deleteWrapper, 'Confirm delete')
    await nextTick()

    expect(batchDelete).toHaveBeenCalledWith([501])
    const deleteCancelButton = getButtonByText(deleteWrapper, 'Cancel')
    expect(deleteCancelButton.attributes('disabled')).toBeDefined()
    expect(deleteCancelButton.attributes('aria-label')).toBe('Processing')
    expect(deleteCancelButton.attributes('title')).toBe('Processing')
    await clickLatestDialogClose(deleteWrapper)
    await nextTick()
    expect(deleteWrapper.text()).toContain('Delete accounts')

    resolveDelete(batchResult())
    await waitForView()
    expect(deleteWrapper.text()).not.toContain('Delete accounts')
    deleteWrapper.unmount()
  })

  it('keeps disabled batch toolbar actions inert when no total-pool rows are selected', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const wrapper = mountView()
    await waitForView()

    const assignButton = wrapper.findAll('button').find(item => item.text().includes('Assign'))
    const reclaimButton = wrapper.findAll('button').find(item => item.text().includes('Reclaim'))
    const deleteButton = wrapper.findAll('button').find(item => item.text().includes('Delete'))
    expect(assignButton, 'toolbar assign button').toBeTruthy()
    expect(reclaimButton, 'toolbar reclaim button').toBeTruthy()
    expect(deleteButton, 'toolbar delete button').toBeTruthy()
    expect(assignButton!.attributes('disabled')).toBeDefined()
    expect(reclaimButton!.attributes('disabled')).toBeDefined()
    expect(deleteButton!.attributes('disabled')).toBeDefined()
    expect(assignButton!.attributes('title')).toBe('Select at least one account first.')
    expect(reclaimButton!.attributes('title')).toBe('Select at least one account first.')
    expect(deleteButton!.attributes('title')).toBe('Select at least one account first.')

    await assignButton!.trigger('click')
    await reclaimButton!.trigger('click')
    await deleteButton!.trigger('click')
    await waitForView()

    expect(wrapper.text()).not.toContain('Assign accounts')
    expect(wrapper.text()).not.toContain('Reclaim accounts')
    expect(wrapper.text()).not.toContain('Delete accounts')
    expect(batchAssign).not.toHaveBeenCalled()
    expect(batchReclaim).not.toHaveBeenCalled()
    expect(batchDelete).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
  })

  it('keeps total-pool toolbar action labels inspectable and constrained on narrow layouts', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.findAll('button').some(button => button.attributes('aria-label') === 'Create')).toBe(false)

    const labels = ['Refresh', 'Import', 'Export']
    for (const label of labels) {
      const button = getButtonByAriaLabel(wrapper, label)
      expect(button.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))

      const text = button.get('span')
      expect(text.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    }

    for (const label of ['Refresh', 'Import', 'Export']) {
      const button = getButtonByAriaLabel(wrapper, label)
      expect(button.attributes('title')).toBe(label)
      expect(button.attributes('disabled')).toBeUndefined()
    }

    for (const label of ['Assign', 'Reclaim', 'Delete']) {
      const button = getButtonByText(wrapper, label)
      expect(button.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
      expect(button.attributes('aria-label')).toBe('Select at least one account first.')
      expect(button.attributes('title')).toBe('Select at least one account first.')
      expect(button.attributes('disabled')).toBeDefined()
    }
  })

  it('keeps bulk dialog footer actions inspectable and constrained on narrow layouts', async () => {
    const findDialogs = (wrapper: ReturnType<typeof mountView>, title: string) =>
      wrapper.findAll('section[role="dialog"]').filter(section => section.text().includes(title))
    const findDialog = (wrapper: ReturnType<typeof mountView>, title: string) => {
      const dialog = findDialogs(wrapper, title)[0]
      expect(dialog, `dialog "${title}"`).toBeTruthy()
      return dialog!
    }

    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const assignWrapper = mountView()
    await waitForView()
    await assignWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(assignWrapper, 'Assign')
    await waitForView()

    const assignDialog = findDialog(assignWrapper, 'Assign accounts')
    expectConstrainedActionButton(assignDialog.get('button[aria-label="Cancel"]'), 'Cancel')
    const reviewButton = getButtonByText(assignDialog, 'Review assignment')
    expect(reviewButton.attributes('aria-label')).toBe('Please choose a target user.')
    expect(reviewButton.attributes('title')).toBe('Please choose a target user.')
    expect(reviewButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    const reviewText = reviewButton.findAll('span').find(node => node.text() === 'Review assignment')
    expect(reviewText, 'button text "Review assignment"').toBeTruthy()
    expect(reviewText!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(reviewButton.attributes('disabled')).toBeDefined()

    await clickButtonByText(assignWrapper, 'owner@example.com')
    await clickButtonByText(assignWrapper, 'Review assignment')
    await waitForView()

    const assignConfirmDialog = findDialog(assignWrapper, 'Confirm assignment')
    expect(findDialogs(assignWrapper, 'Assign accounts')).toHaveLength(0)
    expect(findDialogs(assignWrapper, 'Confirm assignment')).toHaveLength(1)
    expectConstrainedActionButton(assignConfirmDialog.get('button[aria-label="Back"]'), 'Back')
    const confirmAssignButton = assignConfirmDialog.get('button[aria-label="Confirm assign"]')
    expectConstrainedActionButton(confirmAssignButton, 'Confirm assign')
    expect(confirmAssignButton.attributes('disabled')).toBeUndefined()
    await assignConfirmDialog.get('button[aria-label="Back"]').trigger('click')
    await waitForView()

    const returnedAssignDialog = findDialog(assignWrapper, 'Assign accounts')
    expect(findDialogs(assignWrapper, 'Confirm assignment')).toHaveLength(0)
    expect(getButtonByText(returnedAssignDialog, 'Review assignment').attributes('disabled')).toBeUndefined()

    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        assigned_user_id: 7,
        assigned_user_email: 'owner@example.com',
      })],
    })

    const reclaimWrapper = mountView()
    await waitForView()
    await reclaimWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(reclaimWrapper, 'Reclaim')
    await waitForView()

    const reclaimDialog = findDialog(reclaimWrapper, 'Reclaim accounts')
    expectConstrainedActionButton(reclaimDialog.get('button[aria-label="Cancel"]'), 'Cancel')
    const confirmReclaimButton = reclaimDialog.get('button[aria-label="Confirm reclaim"]')
    expectConstrainedActionButton(confirmReclaimButton, 'Confirm reclaim')
    expect(confirmReclaimButton.attributes('disabled')).toBeUndefined()

    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const deleteWrapper = mountView()
    await waitForView()
    await deleteWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(deleteWrapper, 'Delete')
    await waitForView()

    const deleteDialog = findDialog(deleteWrapper, 'Delete accounts')
    expectConstrainedActionButton(deleteDialog.get('button[aria-label="Cancel"]'), 'Cancel')
    const confirmDeleteButton = deleteDialog.get('button[aria-label="Confirm delete"]')
    expectConstrainedActionButton(confirmDeleteButton, 'Confirm delete')
    expect(confirmDeleteButton.attributes('disabled')).toBeUndefined()
  })

  it('keeps long account and user labels readable in assignment review dialogs', async () => {
    const longAccount = 'stage106_total_account_assignment_name_with_really_long_unbroken_identifier_0123456789abcdef'
    const longEmail = 'stage106-owner-with-a-very-long-unbroken-email-local-part-0123456789abcdef@example.test'
    listUsers.mockResolvedValue({ items: [{ ...userFixture(), email: longEmail }] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture({ name: longAccount })] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')

    const assignHint = wrapper.get('[title="Assign 1 accounts."]')
    expect(assignHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(assignHint.attributes('role')).toBe('status')
    expect(assignHint.attributes('aria-live')).toBe('polite')
    expect(assignHint.attributes('aria-atomic')).toBe('true')
    const assignAccount = wrapper.get(`span[title="${longAccount}"]`)
    expect(assignAccount.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    const assignUser = wrapper.get(`div[title="${longEmail}"]`)
    expect(assignUser.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))

    const userButton = wrapper.findAll('button').find(button => button.text().includes(longEmail))
    expect(userButton, 'long target user button').toBeTruthy()
    await userButton!.trigger('click')
    await clickButtonByText(wrapper, 'Review assignment')

    const assignConfirmHint = wrapper.get(`[title="Assign 1 accounts to ${longEmail}."]`)
    expect(assignConfirmHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(assignConfirmHint.attributes('role')).toBe('status')
    expect(assignConfirmHint.attributes('aria-live')).toBe('polite')
    expect(assignConfirmHint.attributes('aria-atomic')).toBe('true')
    const assignImpactHint = wrapper.get('[title="Assigned accounts move to the user workspace."]')
    expect(assignImpactHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(assignImpactHint.attributes('role')).toBe('status')
    expect(assignImpactHint.attributes('aria-live')).toBe('polite')
    expect(assignImpactHint.attributes('aria-atomic')).toBe('true')
    const accountLabels = wrapper.findAll(`span[title="${longAccount}"]`)
    expect(accountLabels.some(label => label.classes().includes('break-all') && label.classes().includes('sm:truncate'))).toBe(true)
    const userLabels = wrapper.findAll(`div[title="${longEmail}"]`)
    expect(userLabels.some(label => label.classes().includes('break-all') && label.classes().includes('sm:truncate'))).toBe(true)
  })

  it('labels target-user assignment counts as visible-scope counts', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({
      items: [
        totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: null, assigned_user_email: '' }),
      ],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await waitForView()

    const assignDialog = wrapper.findAll('section[role="dialog"]').find(section => section.text().includes('Assign accounts'))
    expect(assignDialog, 'assign dialog').toBeTruthy()
    expect(assignDialog!.text()).toContain('0 visible assigned')

    await clickButtonByText(wrapper, 'owner@example.com')
    expect(assignDialog!.text()).toContain('0 visible assigned')
    await clickButtonByText(wrapper, 'Review assignment')
    await waitForView()

    const confirmDialog = wrapper.findAll('section[role="dialog"]').find(section => section.text().includes('Confirm assignment'))
    expect(confirmDialog, 'assign confirmation dialog').toBeTruthy()
    expect(confirmDialog!.text()).toContain('0 visible assigned')
  })

  it('keeps long account and owner labels readable in reclaim and delete confirmations', async () => {
    const longAccount = 'stage106_total_account_reclaim_delete_name_with_really_long_unbroken_identifier_0123456789abcdef'
    const longOwner = 'stage106-assigned-owner-with-a-very-long-unbroken-email-local-part-0123456789abcdef@example.test'
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        name: longAccount,
        assigned_user_id: 42,
        assigned_user_email: longOwner,
      })],
    })

    const reclaimWrapper = mountView()
    await waitForView()
    await reclaimWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(reclaimWrapper, 'Reclaim')

    const reclaimHint = reclaimWrapper.get('[title="Reclaim 1 accounts."]')
    expect(reclaimHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(reclaimHint.attributes('role')).toBe('status')
    expect(reclaimHint.attributes('aria-live')).toBe('polite')
    expect(reclaimHint.attributes('aria-atomic')).toBe('true')
    const reclaimAccount = reclaimWrapper.get(`span[title="${longAccount}"]`)
    expect(reclaimAccount.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    const reclaimOwnerLabels = reclaimWrapper.findAll(`span[title="${longOwner}"]`)
    expect(reclaimOwnerLabels.some(label => label.classes().includes('break-all') && label.classes().includes('sm:truncate'))).toBe(true)

    const deleteWrapper = mountView()
    await waitForView()
    await deleteWrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(deleteWrapper, 'Delete')

    const deleteHint = deleteWrapper.get('[title="Delete 1 accounts."]')
    expect(deleteHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(deleteHint.attributes('role')).toBe('status')
    expect(deleteHint.attributes('aria-live')).toBe('polite')
    expect(deleteHint.attributes('aria-atomic')).toBe('true')
    const deleteImpactHint = deleteWrapper.get('[title="Related task logs and proxy references are cleaned."]')
    expect(deleteImpactHint.attributes('role')).toBe('status')
    expect(deleteImpactHint.attributes('aria-live')).toBe('polite')
    expect(deleteImpactHint.attributes('aria-atomic')).toBe('true')
    const deleteAccount = deleteWrapper.get(`span[title="${longAccount}"]`)
    expect(deleteAccount.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    const deleteOwnerLabels = deleteWrapper.findAll(`span[title="${longOwner}"]`)
    expect(deleteOwnerLabels.some(label => label.classes().includes('break-all') && label.classes().includes('sm:truncate'))).toBe(true)
  })

  it('uses assigned user email from total pool records when the user list is incomplete', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        assigned_user_id: 42,
        assigned_user_email: 'pool-owner@example.com',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-501"]')
    expect(row.text()).toContain('pool-owner@example.com')
    expect(row.text()).not.toContain('#42')
    const ownerBadge = row.get('span[title="pool-owner@example.com"]')
    expect(ownerBadge.classes()).toEqual(expect.arrayContaining(['badge', 'min-w-0', 'max-w-full', 'truncate', 'badge-primary']))
  })

  it('defaults total-pool rows to ID ascending order', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [
        totalAccountFixture({ id: 501 }),
        totalAccountFixture({ id: 777, name: 'southwind_ops' }),
      ],
    })

    const wrapper = mountView()
    await waitForView()

    const table = wrapper.getComponent(DataTableStub)
    expect(table.props('defaultSortKey')).toBe('id')
    expect(table.props('defaultSortOrder')).toBe('asc')
  })

  it('renders limited total-pool account status as a danger badge and keeps ID out of the account subtitle', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 777,
        name: 'southwind_ops',
        username: 'southwind_ops',
        account_status: 'limited',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-777"]')
    const limitedBadge = row.findAll('span').find(node => node.text() === 'Limited')
    expect(limitedBadge).toBeTruthy()
    expect(limitedBadge!.classes()).toEqual(expect.arrayContaining(['badge', 'badge-danger']))
    expect(row.text()).not.toContain('#777')
    expect(row.text()).toContain('southwind_ops')
  })

  it('trims backend total-pool account names before row, detail, and edit identity surfaces display them', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 901,
        name: '  padded_pool_account  ',
        username: 'padded_pool_account',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-901"]')
    expect(row.element.textContent).toContain('padded_pool_account')
    expect(row.element.textContent).not.toContain('  padded_pool_account  ')

    await row.get('button').trigger('click')
    await waitForView()
    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('padded_pool_account')
    expect(detailDialog.element.textContent).not.toContain('  padded_pool_account  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-901"] button[title="Edit"]').trigger('click')
    await waitForView()

    const editIdentity = wrapper.get('[data-testid="total-account-edit-identity"]')
    expect(editIdentity.text()).toContain('padded_pool_account')
    expect(editIdentity.element.textContent).not.toContain('  padded_pool_account  ')
  })

  it('trims backend total-pool usernames before row, detail, and edit identity surfaces display them', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 903,
        name: 'pool_account',
        username: '  padded_pool_user  ',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-903"]')
    expect(row.element.textContent).toContain('padded_pool_user')
    expect(row.element.textContent).not.toContain('  padded_pool_user  ')

    await row.get('button').trigger('click')
    await waitForView()
    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('padded_pool_user')
    expect(detailDialog.element.textContent).not.toContain('  padded_pool_user  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-903"] button[title="Edit"]').trigger('click')
    await waitForView()

    const editIdentity = wrapper.get('[data-testid="total-account-edit-identity"]')
    expect(editIdentity.text()).toContain('padded_pool_user')
    expect(editIdentity.element.textContent).not.toContain('  padded_pool_user  ')
  })

  it('trims backend total-pool platform user IDs before detail and edit identity surfaces display them', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 904,
        platform_user_id: '  uid-stage57  ',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-904"]')
    await row.get('button').trigger('click')
    await waitForView()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('uid-stage57')
    expect(detailDialog.element.textContent).not.toContain('  uid-stage57  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-904"] button[title="Edit"]').trigger('click')
    await waitForView()

    const editIdentity = wrapper.get('[data-testid="total-account-edit-identity"]')
    expect(editIdentity.text()).toContain('uid-stage57')
    expect(editIdentity.element.textContent).not.toContain('  uid-stage57  ')
  })

  it('trims backend total-pool registration IPs before detail and edit identity surfaces display them', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 905,
        registration_ip: '  203.0.113.59  ',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-905"]')
    await row.get('button').trigger('click')
    await waitForView()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('203.0.113.59')
    expect(detailDialog.element.textContent).not.toContain('  203.0.113.59  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-905"] button[title="Edit"]').trigger('click')
    await waitForView()

    const editIdentity = wrapper.get('[data-testid="total-account-edit-identity"]')
    expect(editIdentity.text()).toContain('203.0.113.59')
    expect(editIdentity.element.textContent).not.toContain('  203.0.113.59  ')
  })

  it('hides invalid total-pool created-at values instead of showing Invalid Date in details', async () => {
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 906,
        created_at: 'not-a-date',
      })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-906"] button').trigger('click')
    await waitForView()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('Created at')
    expect(detailDialog.text()).toContain('-')
    expect(detailDialog.text()).not.toContain('Invalid Date')
  })

  it('trims backend total-pool task messages and keeps long details readable', async () => {
    const longTaskMessage = 'stage131_total_pool_task_message_with_a_very_long_unbroken_backend_detail_0123456789abcdef'
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 902,
        account_status: 'limited',
        task_message: `  ${longTaskMessage}  `,
      })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-902"] button').trigger('click')
    await waitForView()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain(longTaskMessage)
    expect(detailDialog.element.textContent).not.toContain(`  ${longTaskMessage}  `)
    const message = wrapper.get(`[title="${longTaskMessage}"]`)
    expect(message.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(message.attributes('role')).toBe('status')
    expect(message.attributes('aria-live')).toBe('polite')
    expect(message.attributes('aria-atomic')).toBe('true')
  })

  it('tones total-pool task messages from normalized task status instead of account status', async () => {
    const taskMessage = 'proxy unavailable during registration'
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 904,
        account_status: 'available',
        task_status: ' IP_UNAVAILABLE ',
        task_message: taskMessage,
      })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-904"] button').trigger('click')
    await waitForView()

    const message = wrapper.get(`[title="${taskMessage}"]`)
    expect(message.classes()).toEqual(expect.arrayContaining(['border-red-200', 'bg-red-50', 'text-red-700']))
    expect(message.classes()).not.toContain('border-emerald-200')
  })

  it('shows stored total-pool snapshots with explicit execution failures as failed messages', async () => {
    const taskMessage = '任务参数不完整，本次未扣费'
    listUsers.mockResolvedValue({ items: [] })
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({
        id: 907,
        account_status: 'available',
        task_status: 'stored',
        task_message: taskMessage,
      })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-907"] button').trigger('click')
    await waitForView()

    const message = wrapper.get(`[title="${taskMessage}"]`)
    expect(message.classes()).toEqual(expect.arrayContaining(['border-red-200', 'bg-red-50', 'text-red-700']))
    expect(message.classes()).not.toContain('border-emerald-200')
  })

  it('keeps target user search stable when optional user fields are missing', async () => {
    listUsers.mockResolvedValue({
      items: [{
        id: 7,
        email: 'owner@example.com',
        status: 'active',
      }],
    })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await wrapper.get('input[placeholder="Search users"]').setValue('no-match')
    await waitForView()

    expect(wrapper.text()).toContain('No users found')
    expect(wrapper.text()).not.toContain('owner@example.com')
  })

  it('shows an inline assignment user-list error instead of an empty result when target users fail to load', async () => {
    listUsers.mockRejectedValue(new Error('user list unavailable'))
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await waitForView()

    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_users', expect.any(Error))
    expect(wrapper.text()).toContain('Failed to load target users.')
    const userLoadError = wrapper.get('[title="Failed to load target users."]')
    expect(userLoadError.attributes('role')).toBe('alert')
    expect(userLoadError.attributes('aria-live')).toBe('assertive')
    expect(userLoadError.attributes('aria-atomic')).toBe('true')
    expect(wrapper.text()).not.toContain('No users found')
  })

  it('filters total-pool rows by the visible account id token', async () => {
    listTotalAccounts.mockResolvedValue({
      items: [
        totalAccountFixture({
          id: 501,
          name: 'northwind_ops',
        }),
        totalAccountFixture({
          id: 777,
          name: 'southwind_ops',
          email: 'southwind@example.com',
        }),
      ],
    })

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="total-account-row-777"]').exists()).toBe(true)

    await wrapper.get('[data-testid="search-input-stub"]').setValue('#777')

    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="total-account-row-777"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="total-account-row-777"]').text()).toContain('southwind_ops')
  })

  it('reloads total-pool rows through backend filters when list filters change', async () => {
    const northwind = totalAccountFixture({ id: 501, name: 'northwind_ops', account_status: 'available' })
    const southwind = totalAccountFixture({ id: 777, name: 'southwind_ops', account_status: 'limited', assigned_user_id: 7, assigned_user_email: 'owner@example.com', default_proxy_snapshot: '' })
    listTotalAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === '#777' && params.account_status === 'limited' && params.assigned === true) {
        return Promise.resolve({ items: [southwind] })
      }
      return Promise.resolve({ items: [northwind, southwind] })
    })

    const wrapper = mountView()
    await waitForView()
    expect(listTotalAccounts).toHaveBeenCalledWith({ page: 1, page_size: 200 })
    expect(wrapper.get('[data-testid="total-account-stat-total"]').text()).toContain('2')

    await wrapper.get('[data-testid="search-input-stub"]').setValue('#777')
    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('limited')
    await selects[1].setValue('assigned')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      search: '#777',
      account_status: 'limited',
      assigned: true,
    })
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-row-777"]').text()).toContain('southwind_ops')
    expect(wrapper.get('[data-testid="total-account-stat-total"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="total-account-stat-available"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="total-account-stat-assigned"]').text()).toContain('1')
  })

  it('shows the total-pool empty state with the existing import and refresh actions constrained', async () => {
    listTotalAccounts.mockResolvedValue({ items: [] })

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.text()).toContain('No accounts')
    expect(wrapper.text()).toContain('Import accounts to get started.')
    expect(wrapper.text()).not.toContain('No results')

    const importButton = getEmptyStateButton(wrapper, 'Import')
    expectConstrainedActionButton(importButton, 'Import')
    expect(importButton.attributes('disabled')).toBeUndefined()

    const refreshButton = getEmptyStateButton(wrapper, 'Refresh')
    expectConstrainedActionButton(refreshButton, 'Refresh')
    await refreshButton.trigger('click')
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({ page: 1, page_size: 200 })
  })

  it('shows the filtered empty state when backend filters return no total-pool rows', async () => {
    const northwind = totalAccountFixture({ id: 501, name: 'northwind_ops' })
    listTotalAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === 'missing-pool-account') {
        return Promise.resolve({ items: [] })
      }
      return Promise.resolve({ items: [northwind] })
    })

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.text()).not.toContain('No results')
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('northwind_ops')

    await wrapper.get('[data-testid="search-input-stub"]').setValue('missing-pool-account')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      search: 'missing-pool-account',
    })
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('No results')
    expect(wrapper.text()).toContain('Adjust filters.')
    expect(wrapper.text()).toContain('Clear filters')
    expect(wrapper.text()).not.toContain('No accounts')
    expect(wrapper.text()).not.toContain('Import accounts to get started.')

    const clearFiltersButton = getEmptyStateButton(wrapper, 'Clear filters')
    expectConstrainedActionButton(clearFiltersButton, 'Clear filters')
    await clearFiltersButton.trigger('click')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({ page: 1, page_size: 200 })
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('northwind_ops')
  })

  it('normalizes total-pool account statuses before display, stats, and local filters use them', async () => {
    const northwind = totalAccountFixture({ id: 501, name: 'northwind_ops', account_status: ' Available ' })
    const southwind = totalAccountFixture({ id: 777, name: 'southwind_ops', account_status: ' LIMITED ' })
    listTotalAccounts.mockResolvedValue({ items: [northwind, southwind] })

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Available')
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).not.toContain('Not stored')
    expect(wrapper.get('[data-testid="total-account-row-777"]').text()).toContain('Limited')
    expect(wrapper.get('[data-testid="total-account-stat-available"]').text()).toContain('1')

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('available')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      account_status: 'available',
    })
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="total-account-row-777"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-stat-total"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="total-account-stat-available"]').text()).toContain('1')
  })

  it('falls back unknown total-pool account statuses without exposing raw backend status text', async () => {
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({ id: 501, name: 'northwind_ops', account_status: ' custom_backend_status ' })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-501"]')
    expect(row.text()).toContain('Not stored')
    expect(row.text()).not.toContain('custom_backend_status')
    expect(wrapper.get('[data-testid="total-account-stat-available"]').text()).toContain('0')
  })

  it('uses the canonical X/Twitter label for known total-pool platform aliases', async () => {
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({ platform: ' Twitter / X ' })],
    })

    const wrapper = mountView()
    await waitForView()

    const row = wrapper.get('[data-testid="total-account-row-501"]')
    expect(row.text()).toContain('X / Twitter')
    expect(row.text()).not.toContain('Twitter / X')

    await row.get('button').trigger('click')
    expect(wrapper.text()).toContain('Account detail')
    expect(wrapper.text()).toContain('X / Twitter')
    expect(wrapper.text()).not.toContain('Twitter / X')
  })

  it('exports selected total-pool rows with the current filter set instead of downloading the unfiltered pool', async () => {
    const blob = new Blob(['platform,name\nx_twitter,southwind_ops\n'], { type: 'text/csv' })
    const createObjectURL = vi.fn(() => 'blob:total-accounts-filtered')
    const revokeObjectURL = vi.fn()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: revokeObjectURL,
    })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture({ id: 777, name: 'southwind_ops', account_status: 'limited', assigned_user_id: 7, assigned_user_email: 'owner@example.com' })] })
    exportTotalAccounts.mockResolvedValue(blob)

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('#777')
    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('limited')
    await selects[1].setValue('assigned')
    await wrapper.get('[data-testid="total-account-row-777"] input[type="checkbox"]').setValue(true)
    await waitForView()

    const exportButton = getButtonByText(wrapper, 'Export')
    expect(exportButton.attributes('aria-label')).toBe('Export selected')
    expect(exportButton.attributes('title')).toBe('Export selected')
    await exportButton.trigger('click')
    await waitForView()

    expect(exportTotalAccounts).toHaveBeenCalledWith({
      search: '#777',
      account_status: 'limited',
      assigned: true,
      account_ids: [777],
    })
    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:total-accounts-filtered')
  })

  it('disables total-pool import while the upload request is in flight', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    let resolveImport!: (value: { total: number; succeeded: number; created: number; skipped: number; failed: number; duplicates: number; errors: string[]; items: Array<{ id?: number; name?: string; status: string; reason?: string; error?: string }> }) => void
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts.mockReturnValue(new Promise((resolve) => {
      resolveImport = resolve
    }))

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await wrapper.vm.$nextTick()

    const pendingImportButton = getButtonByText(wrapper, 'Processing')
    expect(pendingImportButton.attributes('disabled')).toBeDefined()
    expect(pendingImportButton.attributes('aria-label')).toBe('Processing')
    expect(pendingImportButton.attributes('title')).toBe('Processing')
    expect(pendingImportButton.text()).toContain('Processing')

    await triggerImportFile(wrapper, new File(['name\nsouthwind_ops\n'], 'accounts-2.csv', { type: 'text/csv' }))
    expect(importAccounts).toHaveBeenCalledTimes(1)
    expect(importAccounts).toHaveBeenCalledWith(file, 'x_twitter')

    resolveImport({ total: 1, succeeded: 1, created: 1, skipped: 0, failed: 0, duplicates: 0, errors: [], items: [{ id: 901, name: '@northwind_ops', status: 'succeeded' }] })
    await waitForView()

    expect(getButtonByAriaLabel(wrapper, 'Import').attributes('disabled')).toBeUndefined()
    expect(getButtonByAriaLabel(wrapper, 'Import').text()).toContain('Import')
    expect(showSuccess).toHaveBeenCalledWith('Imported 1 accounts.')
  })

  it('shows total-pool import item results without leaking raw reason codes', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\nnorthwind_ops,secret\n,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts.mockResolvedValue({
      total: 3,
      succeeded: 1,
      created: 1,
      skipped: 2,
      failed: 1,
      duplicates: 1,
      errors: ['duplicate account in total pool', 'missing platform or name'],
      items: [
        { id: 901, name: '  @northwind_ops  ', status: 'succeeded' },
        { name: '  @southwind_ops  ', status: 'duplicate', reason: 'duplicate_in_database', error: 'duplicate account in total pool' },
        { name: '', status: 'failed', reason: 'invalid_input', error: 'missing platform or name' },
      ],
    })

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledWith(file, 'x_twitter')
    expect(showWarning).toHaveBeenCalledWith('Import result: total 3, imported 1, skipped 2, failed 1, duplicates 1.')
    expect(showError).not.toHaveBeenCalledWith('Import result: total 3, imported 1, skipped 2, failed 1, duplicates 1.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-import-result"]')
    expect(resultPanel.text()).toContain('Import result: total 3, imported 1, skipped 2, failed 1, duplicates 1.')
    const resultLabels = Array.from(resultPanel.element.querySelectorAll('span.font-medium'))
      .map(node => node.textContent)
    expect(resultLabels).toEqual(['@northwind_ops', '@southwind_ops', '#3'])
    expect(resultPanel.text()).toContain('Already exists in the total account pool')
    expect(resultPanel.text()).toContain('Import data is invalid')
    expect(resultPanel.text()).not.toContain('  @northwind_ops  ')
    expect(resultPanel.text()).not.toContain('  @southwind_ops  ')
    expect(resultPanel.text()).not.toContain('duplicate_in_database')
    expect(resultPanel.text()).not.toContain('invalid_input')
  })

  it('uses the import-specific fallback for unknown total-pool import row failures', async () => {
    const file = new File(['name,password\nbroken_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      created: 0,
      skipped: 0,
      failed: 1,
      duplicates: 0,
      errors: ['database row detail should stay internal'],
      items: [
        { name: '@broken_ops', status: 'failed', reason: 'unexpected_import_state', error: 'database row detail should stay internal' },
      ],
    })

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(showError).toHaveBeenCalledWith('Import result: total 1, imported 0, skipped 0, failed 1, duplicates 0.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-import-result"]')
    expect(resultPanel.text()).toContain('Error')
    expect(resultPanel.text()).toContain('Could not import this account')
    expect(resultPanel.text()).not.toContain('unexpected_import_state')
    expect(resultPanel.text()).not.toContain('database row detail should stay internal')
    expect(resultPanel.text()).not.toContain('common.unknown')
  })

  it('clears the stale import result panel when a later batch assignment completes', async () => {
    const file = new File(['name,password\nsouthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    importAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      created: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [{ id: 901, name: '@southwind_ops', status: 'succeeded' }],
    })
    batchAssign.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 501, status: 'succeeded' }],
    }))

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(wrapper.get('[data-testid="total-accounts-import-result"]').text())
      .toContain('Import result: total 1, imported 1, skipped 0, failed 0, duplicates 0.')

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    expect(wrapper.find('[data-testid="total-accounts-import-result"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-accounts-batch-result"]').text())
      .toContain('Assignment result: total 1, succeeded 1, skipped 0, failed 0. Target user: owner@example.com.')
  })

  it('clears a stale batch result panel when a later import request fails', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    batchAssign.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 501, status: 'succeeded' }],
    }))
    importAccounts.mockRejectedValue(new Error('raw parser failure'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="total-accounts-batch-result"]').text())
      .toContain('Assignment result: total 1, succeeded 1, skipped 0, failed 0. Target user: owner@example.com.')

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Import Failed')
    expect(wrapper.find('[data-testid="total-accounts-batch-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Assignment result: total 1, succeeded 1, skipped 0, failed 0. Target user: owner@example.com.')
  })

  it('restores total-pool import controls and shows a safe error when import fails', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts.mockRejectedValue(new Error('raw parser failure'))

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledTimes(1)
    expect(importAccounts).toHaveBeenCalledWith(file, 'x_twitter')
    expect(getButtonByAriaLabel(wrapper, 'Import').attributes('disabled')).toBeUndefined()
    expect(getButtonByAriaLabel(wrapper, 'Import').text()).toContain('Import')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.import', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Import Failed')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw parser failure'))
  })

  it('rejects an empty total-pool import file before upload with the existing recovery message', async () => {
    const staleFile = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    const file = new File([''], 'empty.csv', { type: 'text/csv' })
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      created: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [{ id: 901, name: '@northwind_ops', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, staleFile)
    await waitForView()
    expect(wrapper.get('[data-testid="total-accounts-import-result"]').text())
      .toContain('Import result: total 1, imported 1, skipped 0, failed 0, duplicates 0.')

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledTimes(1)
    expect(importAccounts).toHaveBeenCalledWith(staleFile, 'x_twitter')
    expect(showError).toHaveBeenCalledWith('Choose a valid import file and try again.')
    expect(wrapper.find('[data-testid="total-accounts-import-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Import result: total 1, imported 1, skipped 0, failed 0, duplicates 0.')
  })

  it('clears the stale import result panel when a retry import request fails', async () => {
    const file = new File(['name,password\nnorthwind_ops,secret\n'], 'accounts.csv', { type: 'text/csv' })
    listTotalAccounts.mockResolvedValue({ items: [] })
    importAccounts
      .mockResolvedValueOnce({
        total: 1,
        succeeded: 1,
        created: 1,
        skipped: 0,
        failed: 0,
        duplicates: 0,
        errors: [],
        items: [{ id: 901, name: '@northwind_ops', status: 'succeeded' }],
      })
      .mockRejectedValueOnce(new Error('raw retry import failure'))

    const wrapper = mountView()
    await waitForView()

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="total-accounts-import-result"]').text())
      .toContain('Import result: total 1, imported 1, skipped 0, failed 0, duplicates 0.')

    await triggerImportFile(wrapper, file)
    await waitForView()

    expect(importAccounts).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Import Failed')
    expect(wrapper.find('[data-testid="total-accounts-import-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Import result: total 1, imported 1, skipped 0, failed 0, duplicates 0.')
  })

  it('exports records through the total account pool API surface', async () => {
    const blob = new Blob(['platform,name\nx_twitter,northwind_ops\n'], { type: 'text/csv' })
    const createObjectURL = vi.fn(() => 'blob:total-accounts')
    const revokeObjectURL = vi.fn()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: revokeObjectURL,
    })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    exportTotalAccounts.mockResolvedValue(blob)

    const wrapper = mountView()
    await waitForView()

    await clickButtonByText(wrapper, 'Export')
    await waitForView()

    expect(exportTotalAccounts).toHaveBeenCalledTimes(1)
    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:total-accounts')
  })

  it('disables total-pool export while the download request is in flight', async () => {
    const blob = new Blob(['platform,name\nx_twitter,northwind_ops\n'], { type: 'text/csv' })
    const createObjectURL = vi.fn(() => 'blob:total-accounts-pending')
    const revokeObjectURL = vi.fn()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    let resolveExport!: (value: Blob) => void
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: revokeObjectURL,
    })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    exportTotalAccounts.mockReturnValue(new Promise((resolve) => {
      resolveExport = resolve
    }))

    const wrapper = mountView()
    await waitForView()

    await getButtonByAriaLabel(wrapper, 'Export').trigger('click')
    await wrapper.vm.$nextTick()

    const pendingExportButton = getButtonByText(wrapper, 'Processing')
    expect(pendingExportButton.attributes('disabled')).toBeDefined()
    expect(pendingExportButton.attributes('aria-label')).toBe('Processing')
    expect(pendingExportButton.attributes('title')).toBe('Processing')
    expect(pendingExportButton.text()).toContain('Processing')

    await pendingExportButton.trigger('click')
    expect(exportTotalAccounts).toHaveBeenCalledTimes(1)
    expect(createObjectURL).not.toHaveBeenCalled()

    resolveExport(blob)
    await waitForView()

    expect(getButtonByAriaLabel(wrapper, 'Export').attributes('disabled')).toBeUndefined()
    expect(getButtonByAriaLabel(wrapper, 'Export').text()).toContain('Export')
    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:total-accounts-pending')
  })

  it('restores total-pool export controls and shows a safe error when export fails', async () => {
    const createObjectURL = vi.fn(() => 'blob:should-not-download')
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    exportTotalAccounts.mockRejectedValue(new Error('raw storage failure'))

    const wrapper = mountView()
    await waitForView()

    await getButtonByAriaLabel(wrapper, 'Export').trigger('click')
    await waitForView()

    expect(exportTotalAccounts).toHaveBeenCalledTimes(1)
    expect(getButtonByAriaLabel(wrapper, 'Export').attributes('disabled')).toBeUndefined()
    expect(getButtonByAriaLabel(wrapper, 'Export').text()).toContain('Export')
    expect(createObjectURL).not.toHaveBeenCalled()
    expect(click).not.toHaveBeenCalled()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.export', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Export Failed')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw storage failure'))
  })

  it('closes the reclaim confirmation when refreshed accounts no longer contain the selection', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7 })] })
      .mockResolvedValueOnce({ items: [] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    expect(wrapper.text()).toContain('Reclaim accounts')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('Reclaim accounts')
    })
    expect(batchReclaim).not.toHaveBeenCalled()
  })

  it('closes the reclaim confirmation when refreshed accounts become unassigned', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7, assigned_user_email: 'owner@example.com' })] })
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: null, assigned_user_email: '' })] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    expect(wrapper.text()).toContain('Reclaim accounts')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Unassigned')
      expect(wrapper.text()).not.toContain('Reclaim accounts')
    })
    expect(batchReclaim).not.toHaveBeenCalled()
  })

  it('closes the delete confirmation when refreshed accounts no longer contain the selection', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [] })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Delete')
    expect(wrapper.text()).toContain('Delete accounts')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('Delete accounts')
    })
    expect(batchDelete).not.toHaveBeenCalled()
  })

  it('shows a warning summary when batch assignment partially skips selected accounts', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7 })] })
    batchAssign.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      items: [
        { id: 501, status: ' SUCCEEDED ' },
        { id: 502, status: ' SKIPPED ', reason: ' already_assigned ' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    expect(showWarning).toHaveBeenCalledWith('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')
    expect(showSuccess).not.toHaveBeenCalledWith('Assigned 1 accounts to owner@example.com.')
    expect(showError).not.toHaveBeenCalled()
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')
    expect(resultPanel.text()).toContain('Succeeded')
    expect(resultPanel.text()).toContain('Skipped')
    expect(resultPanel.text()).toContain('Already assigned')
    expect(resultPanel.text()).not.toContain(' SUCCEEDED ')
    expect(resultPanel.text()).not.toContain(' SKIPPED ')
    expect(resultPanel.text()).not.toContain(' already_assigned ')
    expect(resultPanel.text()).not.toContain('already_assigned')
  })

  it('uses a safe fallback for unknown total-pool batch item failures', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
    batchAssign.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      errors: ['database constraint detail should stay internal'],
      items: [
        {
          id: 501,
          status: 'failed',
          reason: 'unexpected_backend_state',
          error: 'database constraint detail should stay internal',
        },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    expect(showError).toHaveBeenCalledWith('Assignment result: total 1, succeeded 0, skipped 0, failed 1. Target user: owner@example.com.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Failed')
    expect(resultPanel.text()).toContain('Could not process this account')
    expect(resultPanel.text()).not.toContain('unexpected_backend_state')
    expect(resultPanel.text()).not.toContain('database constraint detail should stay internal')
    expect(resultPanel.text()).not.toContain('Unknown')
  })

  it('shows a friendly total-pool error when batch assignment is rejected by code', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    batchAssign.mockRejectedValue({
      reason: 'USER_NOT_FOUND',
      message: 'user not found token=secret',
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.assign', expect.any(Object))
    expect(showError).toHaveBeenCalledWith('The target user no longer exists or is unavailable. Refresh the user list and try again.')
    const dialog = wrapper.get('section[role="dialog"]')
    const alert = dialog.get('[role="alert"]')
    expect(dialog.text()).toContain('Confirm assignment')
    expect(alert.text()).toBe('The target user no longer exists or is unavailable. Refresh the user list and try again.')
    expect(alert.attributes('title')).toBe('The target user no longer exists or is unavailable. Refresh the user list and try again.')
    expect(alert.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(showError).not.toHaveBeenCalledWith('Error')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('user not found'))
    expect(dialog.text()).not.toContain('user not found')
    expect(dialog.text()).not.toContain('token=secret')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('token=secret')
  })

  it('clears the stale batch assignment result panel when a retry request fails', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    batchAssign
      .mockResolvedValueOnce(batchResult({
        total: 1,
        succeeded: 0,
        skipped: 0,
        failed: 1,
        errors: ['account could not be assigned'],
        items: [{ id: 501, status: 'failed', reason: 'assign_failed', error: 'account could not be assigned' }],
      }))
      .mockRejectedValueOnce(new Error('raw assign failure'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="total-accounts-batch-result"]').text())
      .toContain('Assignment result: total 1, succeeded 0, skipped 0, failed 1. Target user: owner@example.com.')

    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Failed to assign accounts')
    expect(showError).not.toHaveBeenCalledWith('Error')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw assign failure'))
    expect(wrapper.find('[data-testid="total-accounts-batch-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Assignment result: total 1, succeeded 0, skipped 0, failed 1. Target user: owner@example.com.')
  })

  it('keeps skipped batch assignment rows selected after syncing succeeded accounts locally', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [
          totalAccountFixture({ id: 501, name: 'northwind_ops' }),
          totalAccountFixture({ id: 502, name: 'southwind_ops' }),
        ],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchAssign.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      items: [
        { id: 501, status: 'succeeded' },
        { id: 502, status: 'skipped', reason: 'already_assigned' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501, 502], 7)
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('owner@example.com')
    expect((wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(showWarning).toHaveBeenCalledWith('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')

    resolveRefresh({
      items: [
        totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
        totalAccountFixture({ id: 502, name: 'southwind_ops' }),
      ],
    })
    await waitForView()
  })

  it('keeps only skipped batch assignment rows selected in the unassigned filter when refresh fails', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    const unassignedAccounts = [
      totalAccountFixture({ id: 501, name: 'northwind_ops' }),
      totalAccountFixture({ id: 502, name: 'southwind_ops' }),
    ]
    let assignmentSaved = false
    listTotalAccounts.mockImplementation((params: Record<string, unknown> = {}) => {
      if (params.unassigned === true && assignmentSaved) {
        return Promise.reject(new Error('follow-up refresh failed'))
      }
      return Promise.resolve({ items: unassignedAccounts })
    })
    batchAssign.mockImplementation(async () => {
      assignmentSaved = true
      return batchResult({
        total: 2,
        succeeded: 1,
        skipped: 1,
        failed: 0,
        items: [
          { id: 501, status: 'succeeded' },
          { id: 502, status: 'skipped', reason: 'already_assigned' },
        ],
      })
    })

    const wrapper = mountView()
    await waitForView()

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[1].setValue('unassigned')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await waitForView()
    expect(wrapper.text()).toContain('2 selected')

    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501, 502], 7)
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('2 selected')
    expect(showWarning).toHaveBeenCalledWith('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Assignment result: total 2, succeeded 1, skipped 1, failed 0. Target user: owner@example.com.')
    expect(resultPanel.text()).toContain('Succeeded')
    expect(resultPanel.text()).toContain('Already assigned')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_accounts', expect.any(Error))

    wrapper.unmount()
  })

  it('normalizes succeeded batch assignment item status before syncing local rows', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [
          totalAccountFixture({ id: 501, name: 'northwind_ops' }),
          totalAccountFixture({ id: 502, name: 'southwind_ops' }),
        ],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchAssign.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      items: [
        { id: 501, status: ' SUCCEEDED ' },
        { id: 502, status: 'skipped', reason: 'already_assigned' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501, 502], 7)
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('owner@example.com')
    expect((wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')

    resolveRefresh({
      items: [
        totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
        totalAccountFixture({ id: 502, name: 'southwind_ops' }),
      ],
    })
    await waitForView()
  })

  it('syncs assigned total-pool accounts locally before the next list refresh finishes', async () => {
    listUsers.mockResolvedValue({ items: [userFixture()] })
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [totalAccountFixture({
          default_proxy_snapshot: '{"id":301,"endpoint":"http://stale-proxy.example:8080"}',
        })],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchAssign.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 501, status: 'succeeded' }],
    }))

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Unassigned')
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Configured')
    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Assign')
    await clickButtonByText(wrapper, 'owner@example.com')
    await clickButtonByText(wrapper, 'Review assignment')
    await clickButtonByText(wrapper, 'Confirm assign')
    await waitForView()

    expect(batchAssign).toHaveBeenCalledWith([501], 7)
    const updatedRow = wrapper.get('[data-testid="total-account-row-501"]')
    expect(updatedRow.text()).toContain('owner@example.com')
    expect(updatedRow.text()).toContain('Not configured')
    expect(wrapper.text()).toContain('0 selected')
    expect(showSuccess).toHaveBeenCalledWith('Assigned 1 accounts to owner@example.com.')

    resolveRefresh({
      items: [totalAccountFixture({
        assigned_user_id: 7,
        assigned_user_email: 'owner@example.com',
        default_proxy_snapshot: '',
      })],
    })
    await waitForView()
  })

  it('removes reclaimed rows from the current assigned filter before the next list refresh finishes', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    const assignedAccount = totalAccountFixture({
      assigned_user_id: 7,
      assigned_user_email: 'owner@example.com',
      default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080"}',
    })
    let keepNextRefreshPending = false
    listTotalAccounts.mockImplementation(() => {
      if (keepNextRefreshPending) {
        return new Promise((resolve) => {
          resolveRefresh = resolve
        })
      }
      return Promise.resolve({ items: [assignedAccount] })
    })
    batchReclaim.mockImplementation(() => {
      keepNextRefreshPending = true
      return Promise.resolve(batchResult({
        total: 1,
        succeeded: 1,
        skipped: 0,
        failed: 0,
        items: [{ id: 501, status: 'succeeded' }],
      }))
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] button').trigger('click')
    await waitForView()
    expect(wrapper.text()).toContain('Account detail')

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[1].setValue('assigned')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await waitForView()
    expect(wrapper.text()).toContain('1 selected')
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501])
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Account detail')
    expect(wrapper.text()).toContain('0 selected')

    resolveRefresh({
      items: [totalAccountFixture({
        assigned_user_id: null,
        assigned_user_email: '',
        default_proxy_snapshot: '',
      })],
    })
    await waitForView()
  })

  it('keeps only failed batch reclaim rows selected in the assigned filter when refresh fails', async () => {
    const assignedAccounts = [
      totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
      totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: 8, assigned_user_email: 'other@example.com' }),
    ]
    let reclaimSaved = false
    listTotalAccounts.mockImplementation((params: Record<string, unknown> = {}) => {
      if (params.assigned === true && reclaimSaved) {
        return Promise.reject(new Error('follow-up refresh failed'))
      }
      return Promise.resolve({ items: assignedAccounts })
    })
    batchReclaim.mockImplementation(async () => {
      reclaimSaved = true
      return batchResult({
        total: 2,
        succeeded: 1,
        skipped: 0,
        failed: 1,
        errors: ['account could not be reclaimed'],
        items: [
          { id: 501, status: 'succeeded' },
          { id: 502, status: 'failed', reason: 'reclaim_failed', error: 'account could not be reclaimed' },
        ],
      })
    })

    const wrapper = mountView()
    await waitForView()

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[1].setValue('assigned')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await waitForView()
    expect(wrapper.text()).toContain('2 selected')

    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501, 502])
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('2 selected')
    expect(showWarning).toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 0, failed 1.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Reclaim result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(resultPanel.text()).toContain('Reclaim failed')
    expect(resultPanel.text()).not.toContain('reclaim_failed')
    expect(resultPanel.text()).not.toContain('account could not be reclaimed')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_accounts', expect.any(Error))

    wrapper.unmount()
  })

  it('syncs edited total-pool delivery fields locally before the next list refresh finishes', async () => {
    let resolveUpdate!: (value: ReturnType<typeof totalAccountFixture>) => void
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    const editedDeliveryFields = {
      password: '  updated-password  ',
      email_password: '  updated-email-password  ',
      two_factor: '  654321  ',
      backup_code: '  backup one  ',
      email_client_id: '  client-id-updated  ',
      email_token: '  email-token-updated  ',
      auth_cookie: '  ct0=updated-cookie; auth_token=updated-token  ',
      execution_auth: '  encrypted-updated-execution-auth  ',
      remark: '  updated delivery note  ',
    }
    const updatedAccount = totalAccountFixture({
      password: editedDeliveryFields.password,
      email: 'ops-updated@example.com',
      email_password: editedDeliveryFields.email_password,
      two_factor: editedDeliveryFields.two_factor,
      backup_code: editedDeliveryFields.backup_code,
      email_client_id: editedDeliveryFields.email_client_id,
      email_token: editedDeliveryFields.email_token,
      registration_ip: '203.0.113.77',
      auth_cookie: editedDeliveryFields.auth_cookie,
      execution_auth: editedDeliveryFields.execution_auth,
      remark: editedDeliveryFields.remark,
    })
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    updateAccount.mockReturnValue(new Promise((resolve) => {
      resolveUpdate = resolve
    }))

    const wrapper = mountView()
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('ops@example.com')
    await wrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    expect(wrapper.text()).toContain('Edit account')

    await wrapper.get('#total-account-edit-password').setValue(editedDeliveryFields.password)
    await wrapper.get('#total-account-edit-email').setValue('ops-updated@example.com')
    await wrapper.get('#total-account-edit-email-password').setValue(editedDeliveryFields.email_password)
    await wrapper.get('#total-account-edit-two-factor').setValue(editedDeliveryFields.two_factor)
    await wrapper.get('#total-account-edit-backup-code').setValue(editedDeliveryFields.backup_code)
    await wrapper.get('#total-account-edit-email-client-id').setValue(editedDeliveryFields.email_client_id)
    await wrapper.get('#total-account-edit-email-token').setValue(editedDeliveryFields.email_token)
    await wrapper.get('#total-account-edit-registration-ip').setValue('203.0.113.77')
    await wrapper.get('#total-account-edit-auth-cookie').setValue(editedDeliveryFields.auth_cookie)
    await wrapper.get('#total-account-edit-execution-auth').setValue(editedDeliveryFields.execution_auth)
    await wrapper.get('#total-account-edit-remark').setValue(editedDeliveryFields.remark)
    await clickButtonByText(wrapper, 'Confirm')
    await wrapper.vm.$nextTick()

    expect(updateAccount).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Processing')
    const savingButton = wrapper.findAll('button').find(item => item.text().includes('Processing'))
    expect(savingButton, 'saving edit button').toBeTruthy()
    expect(savingButton!.attributes('disabled')).toBeDefined()
    await savingButton!.trigger('click')
    await clickButtonByText(wrapper, 'Cancel')
    expect(updateAccount).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Edit account')

    resolveUpdate(updatedAccount)
    await waitForView()

    expect(updateAccount).toHaveBeenCalledWith(501, expect.objectContaining({
      password: editedDeliveryFields.password,
      email: 'ops-updated@example.com',
      email_password: editedDeliveryFields.email_password,
      two_factor: editedDeliveryFields.two_factor,
      backup_code: editedDeliveryFields.backup_code,
      email_client_id: editedDeliveryFields.email_client_id,
      email_token: editedDeliveryFields.email_token,
      registration_ip: '203.0.113.77',
      auth_cookie: editedDeliveryFields.auth_cookie,
      execution_auth: editedDeliveryFields.execution_auth,
      remark: editedDeliveryFields.remark,
    }))
    const updatedRow = wrapper.get('[data-testid="total-account-row-501"]')
    expect(updatedRow.text()).toContain('ops-updated@example.com')
    expect(updatedRow.text()).not.toContain('ops@example.com')
    expect(showSuccess).toHaveBeenCalledWith('Saved')

    resolveRefresh({ items: [updatedAccount] })
    await waitForView()
  })

  it('shows an action-specific safe error when total-pool edit save fails', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    updateAccount.mockRejectedValue(new Error('raw edit storage failure'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await wrapper.get('#total-account-edit-email').setValue('ops-updated@example.com')
    await clickButtonByText(wrapper, 'Confirm')
    await waitForView()

    expect(updateAccount).toHaveBeenCalledTimes(1)
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.edit', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Failed to save account')
    expect(showError).not.toHaveBeenCalledWith('Error')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw edit storage failure'))
    expect(wrapper.text()).toContain('Edit account')
    const inlineError = wrapper.get('section[role="dialog"] [role="alert"]')
    expect(inlineError.text()).toBe('Failed to save account')
    expect(inlineError.attributes('title')).toBe('Failed to save account')
    expect(wrapper.text()).not.toContain('raw edit storage failure')

    await clickLatestDialogClose(wrapper)
    await waitForView()
    expect(wrapper.text()).not.toContain('Failed to save account')

    await wrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await waitForView()
    expect(wrapper.find('section[role="dialog"] [role="alert"]').exists()).toBe(false)
  })

  it('clears an edited row selection when the current filter no longer matches before refresh finishes', async () => {
    let resolveUpdate!: (value: ReturnType<typeof totalAccountFixture>) => void
    const noMatchRefreshResolvers: Array<(value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void> = []
    const updatedAccount = totalAccountFixture({
      email: 'ops-updated@example.com',
      remark: 'updated while filtered out',
    })
    listTotalAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === 'no-matching-pool-account') {
        return new Promise(resolve => {
          noMatchRefreshResolvers.push(resolve)
        })
      }
      return Promise.resolve({ items: [totalAccountFixture()] })
    })
    updateAccount.mockReturnValue(new Promise((resolve) => {
      resolveUpdate = resolve
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    expect(wrapper.text()).toContain('1 selected')
    await wrapper.get('[data-testid="total-account-row-501"] button[title="Edit"]').trigger('click')
    await wrapper.get('#total-account-edit-email').setValue('ops-updated@example.com')
    await clickButtonByText(wrapper, 'Confirm')
    await wrapper.vm.$nextTick()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('no-matching-pool-account')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(listTotalAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      search: 'no-matching-pool-account',
    })
    expect(wrapper.text()).toContain('1 selected')

    resolveUpdate(updatedAccount)
    await waitForView()

    expect(updateAccount).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('0 selected')
    expect(wrapper.text()).not.toContain('Edit account')

    for (const resolve of noMatchRefreshResolvers) {
      resolve({ items: [] })
    }
    await waitForView()
  })

  it('shows a warning summary when batch reclaim partially succeeds with failures', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7 })] })
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7 })] })
    batchReclaim.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 1,
      errors: ['account could not be reclaimed'],
      items: [
        { id: 501, status: 'succeeded' },
        { id: 502, status: 'failed', reason: 'reclaim_failed', error: 'account could not be reclaimed' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501])
    expect(showWarning).toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 1, failed 1.')
    expect(showError).not.toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 1, failed 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Reclaimed 1 accounts and marked them unassigned.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Reclaim result: total 2, succeeded 1, skipped 1, failed 1.')
    expect(resultPanel.text()).toContain('Reclaim failed')
    expect(resultPanel.text()).not.toContain('reclaim_failed')
    expect(resultPanel.text()).not.toContain('account could not be reclaimed')
  })

  it('clears the stale batch reclaim result panel when a retry request fails', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture({ assigned_user_id: 7 })] })
    batchReclaim
      .mockResolvedValueOnce(batchResult({
        total: 1,
        succeeded: 0,
        skipped: 0,
        failed: 1,
        errors: ['account could not be reclaimed'],
        items: [{ id: 501, status: 'failed', reason: 'reclaim_failed', error: 'account could not be reclaimed' }],
      }))
      .mockRejectedValueOnce(new Error('raw reclaim failure'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="total-accounts-batch-result"]').text())
      .toContain('Reclaim result: total 1, succeeded 0, skipped 0, failed 1.')

    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Failed to reclaim accounts')
    const reclaimDialog = wrapper.get('section[role="dialog"]')
    const reclaimAlert = reclaimDialog.get('[role="alert"]')
    expect(reclaimDialog.text()).toContain('Reclaim accounts')
    expect(reclaimAlert.text()).toBe('Failed to reclaim accounts')
    expect(reclaimAlert.attributes('title')).toBe('Failed to reclaim accounts')
    expect(reclaimAlert.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(showError).not.toHaveBeenCalledWith('Error')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw reclaim failure'))
    expect(reclaimDialog.text()).not.toContain('raw reclaim failure')
    expect(wrapper.find('[data-testid="total-accounts-batch-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Reclaim result: total 1, succeeded 0, skipped 0, failed 1.')
  })

  it('clears a stale reclaim dialog error when refreshed accounts become unreclaimable', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: 7, assigned_user_email: 'owner@example.com' })] })
      .mockResolvedValueOnce({ items: [totalAccountFixture({ assigned_user_id: null, assigned_user_email: '' })] })
    batchReclaim.mockRejectedValue(new Error('raw reclaim failure token=secret'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(showError).toHaveBeenCalledWith('Failed to reclaim accounts')
    expect(wrapper.get('section[role="dialog"]').text()).toContain('Failed to reclaim accounts')
    expect(wrapper.text()).not.toContain('token=secret')

    await clickButtonByText(wrapper, 'Refresh')
    await waitForView()

    await waitForCondition(() => {
      expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Unassigned')
      expect(wrapper.text()).not.toContain('Reclaim accounts')
      expect(wrapper.text()).not.toContain('Failed to reclaim accounts')
      expect((wrapper.vm as unknown as { reclaimDialogError: string }).reclaimDialogError).toBe('')
    })
  })

  it('disables batch reclaim when the current selection only contains unassigned accounts', async () => {
    listTotalAccounts.mockResolvedValue({
      items: [totalAccountFixture({ assigned_user_id: null, assigned_user_email: '' })],
    })

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    const reclaimButton = wrapper.findAll('button').find(item => item.text().includes('Reclaim'))
    expect(reclaimButton, 'toolbar reclaim button').toBeTruthy()
    expect(reclaimButton!.attributes('disabled')).toBeDefined()
    expect(reclaimButton!.attributes('title')).toBe('Select at least one assigned account to return to the pool.')

    await reclaimButton!.trigger('click')

    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('Reclaim accounts')
    expect(batchReclaim).not.toHaveBeenCalled()
  })

  it('allows mixed batch reclaim selections so the backend can report skipped unassigned rows', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [
          totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
          totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: null, assigned_user_email: '' }),
        ],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchReclaim.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      items: [
        { id: 501, status: 'succeeded' },
        { id: 502, status: 'skipped', reason: 'not_assigned' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    const reclaimButton = wrapper.findAll('button').find(item => item.text().includes('Reclaim'))
    expect(reclaimButton, 'toolbar reclaim button').toBeTruthy()
    expect(reclaimButton!.attributes('disabled')).toBeUndefined()

    await reclaimButton!.trigger('click')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501, 502])
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Unassigned')
    expect((wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(showWarning).toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 1, failed 0.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Already unassigned')
    expect(resultPanel.text()).not.toContain('not_assigned')

    resolveRefresh({
      items: [
        totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: null, assigned_user_email: '' }),
        totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: null, assigned_user_email: '' }),
      ],
    })
    await waitForView()
  })

  it('keeps failed batch reclaim rows selected after syncing succeeded accounts locally', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [
          totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
          totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: 8, assigned_user_email: 'other@example.com' }),
        ],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchReclaim.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      errors: ['account could not be reclaimed'],
      items: [
        { id: 501, status: 'succeeded' },
        { id: 502, status: 'failed', reason: 'reclaim_failed', error: 'account could not be reclaimed' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501, 502])
    expect(wrapper.get('[data-testid="total-account-row-501"]').text()).toContain('Unassigned')
    expect((wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(showWarning).toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(showError).not.toHaveBeenCalledWith('Reclaim result: total 2, succeeded 1, skipped 0, failed 1.')

    resolveRefresh({
      items: [
        totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: null, assigned_user_email: '' }),
        totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: 8, assigned_user_email: 'other@example.com' }),
      ],
    })
    await waitForView()
  })

  it('syncs reclaimed total-pool accounts locally before the next list refresh finishes', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [totalAccountFixture({
          assigned_user_id: 7,
          assigned_user_email: 'owner@example.com',
          default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080"}',
        })],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchReclaim.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 501, status: 'succeeded' }],
    }))

    const wrapper = mountView()
    await waitForView()

    const initialRow = wrapper.get('[data-testid="total-account-row-501"]')
    expect(initialRow.text()).toContain('owner@example.com')
    expect(initialRow.text()).toContain('Configured')

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Reclaim')
    await clickButtonByText(wrapper, 'Confirm reclaim')
    await waitForView()

    expect(batchReclaim).toHaveBeenCalledWith([501])
    const updatedRow = wrapper.get('[data-testid="total-account-row-501"]')
    expect(updatedRow.text()).toContain('Unassigned')
    expect(updatedRow.text()).not.toContain('owner@example.com')
    expect(updatedRow.text()).toContain('Not configured')
    expect(showSuccess).toHaveBeenCalledWith('Reclaimed 1 accounts and marked them unassigned.')

    resolveRefresh({
      items: [totalAccountFixture({
        assigned_user_id: null,
        assigned_user_email: '',
        default_proxy_snapshot: '',
      })],
    })
    await waitForView()
  })

  it('removes deleted total-pool accounts locally before the next list refresh finishes', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchDelete.mockResolvedValue(batchResult({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 501, status: 'succeeded' }],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledWith([501])
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('0 selected')
    expect(showSuccess).toHaveBeenCalledWith('Deleted 1 accounts.')

    resolveRefresh({ items: [] })
    await waitForView()
  })

  it('keeps failed batch-delete rows selected after removing succeeded accounts locally', async () => {
    let resolveRefresh!: (value: { items: Array<ReturnType<typeof totalAccountFixture>> }) => void
    listTotalAccounts
      .mockResolvedValueOnce({
        items: [
          totalAccountFixture({ id: 501, name: 'northwind_ops' }),
          totalAccountFixture({ id: 502, name: 'southwind_ops' }),
        ],
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchDelete.mockResolvedValue(batchResult({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      errors: ['account could not be deleted'],
      items: [
        { id: 501, status: 'succeeded' },
        { id: 502, status: 'failed', reason: 'delete_failed', error: 'account could not be deleted' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledWith([501, 502])
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(showWarning).toHaveBeenCalledWith('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(showError).not.toHaveBeenCalledWith('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(resultPanel.text()).toContain('Delete failed')
    expect(resultPanel.text()).not.toContain('delete_failed')
    expect(resultPanel.text()).not.toContain('account could not be deleted')

    resolveRefresh({
      items: [
        totalAccountFixture({ id: 502, name: 'southwind_ops' }),
      ],
    })
    await waitForView()
  })

  it('clears the stale batch-delete result panel when a retry request fails', async () => {
    listTotalAccounts.mockResolvedValue({ items: [totalAccountFixture()] })
    batchDelete
      .mockResolvedValueOnce(batchResult({
        total: 1,
        succeeded: 0,
        skipped: 0,
        failed: 1,
        errors: ['account could not be deleted'],
        items: [{ id: 501, status: 'failed', reason: 'delete_failed', error: 'account could not be deleted' }],
      }))
      .mockRejectedValueOnce(new Error('raw delete failure'))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="total-accounts-batch-result"]').text())
      .toContain('Delete result: total 1, succeeded 0, skipped 0, failed 1.')

    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Failed to delete accounts')
    expect(showError).not.toHaveBeenCalledWith('Error')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw delete failure'))
    const deleteError = wrapper.get('[role="alert"]')
    expect(deleteError.text()).toContain('Failed to delete accounts')
    expect(deleteError.attributes('title')).toBe('Failed to delete accounts')
    expect(deleteError.attributes('aria-live')).toBe('assertive')
    expect(deleteError.attributes('aria-atomic')).toBe('true')
    expect(wrapper.find('[data-testid="total-accounts-batch-result"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Delete result: total 1, succeeded 0, skipped 0, failed 1.')

    await clickButtonByText(wrapper, 'Cancel')
    await clickButtonByText(wrapper, 'Delete')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('keeps only failed batch-delete rows selected in the assigned filter when refresh fails', async () => {
    const assignedAccounts = [
      totalAccountFixture({ id: 501, name: 'northwind_ops', assigned_user_id: 7, assigned_user_email: 'owner@example.com' }),
      totalAccountFixture({ id: 502, name: 'southwind_ops', assigned_user_id: 8, assigned_user_email: 'other@example.com' }),
    ]
    let deleteSaved = false
    listTotalAccounts.mockImplementation((params: Record<string, unknown> = {}) => {
      if (params.assigned === true && deleteSaved) {
        return Promise.reject(new Error('follow-up refresh failed'))
      }
      return Promise.resolve({ items: assignedAccounts })
    })
    batchDelete.mockImplementation(async () => {
      deleteSaved = true
      return batchResult({
        total: 2,
        succeeded: 1,
        skipped: 0,
        failed: 1,
        errors: ['account could not be deleted'],
        items: [
          { id: 501, status: 'succeeded' },
          { id: 502, status: 'failed', reason: 'delete_failed', error: 'account could not be deleted' },
        ],
      })
    })

    const wrapper = mountView()
    await waitForView()

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[1].setValue('assigned')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForView()

    expect(wrapper.get('[data-testid="total-account-row-501"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').setValue(true)
    await waitForView()
    expect(wrapper.text()).toContain('2 selected')

    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledWith([501, 502])
    expect(wrapper.find('[data-testid="total-account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="total-account-row-502"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="total-account-row-502"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('2 selected')
    expect(showWarning).toHaveBeenCalledWith('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    const resultPanel = wrapper.get('[data-testid="total-accounts-batch-result"]')
    expect(resultPanel.text()).toContain('Delete result: total 2, succeeded 1, skipped 0, failed 1.')
    expect(resultPanel.text()).toContain('Delete failed')
    expect(resultPanel.text()).not.toContain('delete_failed')
    expect(resultPanel.text()).not.toContain('account could not be deleted')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('admin.total_accounts.load_accounts', expect.any(Error))

    wrapper.unmount()
  })

  it('uses the backend delete summary instead of the selected count fallback', async () => {
    listTotalAccounts
      .mockResolvedValueOnce({ items: [totalAccountFixture()] })
      .mockResolvedValueOnce({ items: [] })
    batchDelete.mockResolvedValue(batchResult({
      total: 3,
      succeeded: 1,
      skipped: 2,
      failed: 0,
      items: [
        { id: 501, status: 'succeeded' },
        { id: 0, status: 'skipped', reason: 'invalid_id' },
        { id: 502, status: 'skipped', reason: 'not_found' },
      ],
    }))

    const wrapper = mountView()
    await waitForView()

    await wrapper.get('[data-testid="total-account-row-501"] input[type="checkbox"]').setValue(true)
    await clickButtonByText(wrapper, 'Delete')
    await clickButtonByText(wrapper, 'Confirm delete')
    await waitForView()

    expect(batchDelete).toHaveBeenCalledWith([501])
    expect(showWarning).toHaveBeenCalledWith('Delete result: total 3, succeeded 1, skipped 2, failed 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Deleted 1 accounts.')
    expect(showError).not.toHaveBeenCalled()
  })
})
