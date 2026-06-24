import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount, type DOMWrapper } from '@vue/test-utils'
import * as XLSX from 'xlsx'

import UnifiedAccountWorkbenchView from '../UnifiedAccountWorkbenchView.vue'
import SocialAccountBatchResultRows from '@/components/accounts/SocialAccountBatchResultRows.vue'
import type { BatchDeleteSocialAccountResponse } from '@/api/accountWorkbench'

const {
  listMyAccounts,
  batchImportMyAccounts,
  deleteMyAccount,
  batchDeleteMyAccounts,
  updateMyAccount,
  submitTask,
  listTaskLogs,
  setDefaultProxy,
  batchSetDefaultProxy,
  exportMyAccounts,
  storeWorkbenchAccounts,
  listUsable,
  listTemplates,
  previewMedia,
  adminState,
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
  listTaskLogs: vi.fn(),
  setDefaultProxy: vi.fn(),
  batchSetDefaultProxy: vi.fn(),
  exportMyAccounts: vi.fn(),
  storeWorkbenchAccounts: vi.fn(),
  listUsable: vi.fn(),
  listTemplates: vi.fn(),
  previewMedia: vi.fn(),
  adminState: { isAdmin: false },
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

const originalCreateObjectURL = globalThis.URL.createObjectURL
const originalRevokeObjectURL = globalThis.URL.revokeObjectURL
const originalClipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
const writeClipboard = vi.fn()
const mountedWrappers: Array<{ unmount: () => void }> = []

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.cancel': 'Cancel',
  'common.close': 'Close',
  'common.confirm': 'Confirm',
  'common.edit': 'Edit',
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
  'accountWorkbench.exportSelectedAccounts': 'Export selected accounts',
  'accountWorkbench.exportFailed': 'Export failed',
  'accountWorkbench.deleteSelected': 'Delete selected',
  'accountWorkbench.deleteOne': 'Delete account',
  'accountWorkbench.noResults.title': 'No results',
  'accountWorkbench.noResults.description': 'Adjust your filters.',
  'accountWorkbench.empty.title': 'No accounts',
  'accountWorkbench.empty.description': 'Import accounts to get started.',
  'accountWorkbench.selection.selectedCount': '{count} selected',
  'accountWorkbench.storeWorkbench.title': 'Upload to Pool',
  'accountWorkbench.storeWorkbench.hint': 'Upload {count} selected not-stored account(s).',
  'accountWorkbench.storeWorkbench.storeable': 'Storeable',
  'accountWorkbench.storeWorkbench.skippedSelection': 'Not storeable',
  'accountWorkbench.storeWorkbench.accountSummary': 'Accounts to upload',
  'accountWorkbench.storeWorkbench.selectAccountsFirst': 'Select accounts first.',
  'accountWorkbench.storeWorkbench.onlyNotStored': 'Only not-stored accounts can be uploaded.',
  'accountWorkbench.storeWorkbench.confirm': 'Confirm upload',
  'accountWorkbench.storeWorkbench.resultTitle': 'Upload result',
  'accountWorkbench.storeWorkbench.resultSummary': 'Total {total}, succeeded {succeeded}, failed {failed}, skipped {skipped}.',
  'accountWorkbench.storeWorkbench.resultRowsMore': '{count} more',
  'accountWorkbench.storeWorkbench.savedWithSummary': 'Upload saved',
  'accountWorkbench.storeWorkbench.failed': 'Upload failed',
  'accountWorkbench.batchResultReasons.invalidId': 'Invalid account ID',
  'accountWorkbench.batchResultReasons.duplicateInBatch': 'Duplicate in batch',
  'accountWorkbench.batchResultReasons.accountNotFound': 'Account not found',
  'accountWorkbench.batchResultReasons.accountNotAssigned': 'Account not assigned',
  'accountWorkbench.batchResultReasons.proxyNotAvailable': 'Selected proxy unavailable',
  'accountWorkbench.batchResultReasons.assignFailed': 'Assignment failed',
  'accountWorkbench.batchResultReasons.notFound': 'Record not found',
  'accountWorkbench.batchResultReasons.alreadyStored': 'Already stored',
  'accountWorkbench.batchResultReasons.invalidCredentials': 'Required credentials are incomplete',
  'accountWorkbench.batchResultReasons.alreadyAssigned': 'Already assigned',
  'accountWorkbench.batchResultReasons.targetUserNotFound': 'Target user not found',
  'accountWorkbench.batchResultReasons.reclaimFailed': 'Reclaim failed',
  'accountWorkbench.batchResultReasons.deleteFailed': 'Delete failed',
  'accountWorkbench.batchResultReasons.createFailed': 'Create failed',
  'accountWorkbench.batchResultReasons.loadFailed': 'Load failed',
  'accountWorkbench.batchResultReasons.uploadFailed': 'Upload failed',
  'accountWorkbench.batchResultReasons.operationFailed': 'Could not process this account',
  'accountWorkbench.batchResultReasons.stateChanged': 'State changed',
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
  'accountWorkbench.proxy.resultStatuses.succeeded': 'Success',
  'accountWorkbench.proxy.resultStatuses.skipped': 'Skipped',
  'accountWorkbench.proxy.resultStatuses.failed': 'Error',
  'accountWorkbench.proxy.savedWithSummary': 'Proxy saved',
  'accountWorkbench.proxy.selectPlaceholder': 'Select proxy',
  'accountWorkbench.proxy.noOnlineProxies': 'No online proxies',
  'accountWorkbench.proxy.selectAccountsFirst': 'Select accounts first',
  'accountWorkbench.proxy.selectOnlineProxyFirst': 'Select an online proxy first',
  'accountWorkbench.proxy.failed': 'Proxy failed',
  'proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE': 'Proxy service is temporarily unavailable.',
  'accountWorkbench.proxy.errors.SOCIAL_IP_NOT_AVAILABLE': 'Selected proxy is unavailable.',
  'accountWorkbench.proxy.errors.SOCIAL_IP_NOT_FOUND': 'Selected proxy was not found.',
  'accountWorkbench.proxy.errors.SOCIAL_IP_POOL_EMPTY': 'No online proxies available for random assignment.',
  'accountWorkbench.proxy.errors.SOCIAL_IP_REQUIRED': 'Select an online proxy before assigning.',
  'accountWorkbench.proxy.errors.SOCIAL_IP_ASSIGNMENT_MODE_INVALID': 'Proxy assignment mode changed. Reopen the dialog and try again.',
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
  'accountWorkbench.import.fileDropTitle': 'Drop file here',
  'accountWorkbench.import.fileDropHint': 'Supports txt / xls / xlsx. Choose a file to generate the preview automatically.',
  'accountWorkbench.import.chooseFile': 'Choose file',
  'accountWorkbench.import.fileEmpty': 'No file selected',
  'accountWorkbench.import.clearSource': 'Clear source',
  'accountWorkbench.import.previewScopeUser': 'Preview is scoped to your workbench.',
  'accountWorkbench.import.submitReady': 'Ready',
  'accountWorkbench.import.pendingCount': 'Pending',
  'accountWorkbench.import.invalidCount': 'Invalid',
  'accountWorkbench.import.previewTitle': 'Import preview',
  'accountWorkbench.import.previewMeta': 'Valid {valid} / invalid {invalid}',
  'accountWorkbench.import.resultTitle': 'Import result',
  'accountWorkbench.import.resultSummary': 'Total {total}, imported {imported}, failed {failed}, skipped {skipped}, duplicates {duplicates}.',
  'accountWorkbench.import.resultRowsMore': '{count} more',
  'accountWorkbench.import.batchSuccess': 'Imported {count}',
  'accountWorkbench.import.batchFailed': 'Import failed',
  'accountWorkbench.import.errors.accountRequired': 'Account required',
  'accountWorkbench.import.errors.passwordRequired': 'Password required',
  'accountWorkbench.import.errors.credentialRequired': 'Credential required',
  'accountWorkbench.import.errors.duplicateAccount': 'Duplicate account',
  'accountWorkbench.import.errors.duplicateInWorkbench': 'Already exists',
  'accountWorkbench.import.errors.unsupportedFile': 'Unsupported file',
  'accountWorkbench.import.errors.emptyFile': 'No importable account rows were found in this file',
  'accountWorkbench.import.errors.fileReadFailed': 'File read failed',
  'accountWorkbench.import.status.batchDuplicate': 'Batch duplicate',
  'accountWorkbench.import.status.existingWorkbenchDuplicate': 'Existing duplicate',
  'accountWorkbench.import.status.needsData': 'Needs data',
  'accountWorkbench.import.status.pendingBackendMatch': 'Pending backend match',
  'accountWorkbench.import.status.duplicate': 'Duplicate',
  'accountWorkbench.import.status.skipped': 'Skipped',
  'accountWorkbench.import.resultReasons.matchedTotalPool': 'Matched total pool',
  'accountWorkbench.import.resultReasons.stagedNotStored': 'Staged not stored',
  'accountWorkbench.import.resultReasons.duplicateInBatch': 'Duplicate in batch',
  'accountWorkbench.import.resultReasons.duplicateInDatabase': 'Duplicate in total pool',
  'accountWorkbench.import.resultReasons.alreadyInWorkbench': 'Already in workbench',
  'accountWorkbench.import.resultReasons.alreadyAssigned': 'Already assigned',
  'accountWorkbench.import.resultReasons.ambiguousTotalPoolMatch': 'Ambiguous total pool match',
  'accountWorkbench.import.resultReasons.invalidInput': 'Invalid import data',
  'accountWorkbench.import.resultReasons.importFailed': 'Import failed',
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
  'accountWorkbench.taskStatus.pending': 'Pending',
  'accountWorkbench.taskStatus.running': 'Running',
  'accountWorkbench.taskStatus.success': 'Success',
  'accountWorkbench.taskStatus.failed': 'Failed',
  'accountWorkbench.taskStatus.stored': 'Stored',
  'accountWorkbench.taskStatus.manual_review': 'Manual review',
  'accountWorkbench.taskStatus.ip_unavailable': 'Proxy unavailable',
  'accountWorkbench.stats.assigned': 'Assigned',
  'accountWorkbench.stats.assignedMeta': 'Assigned accounts',
  'accountWorkbench.stats.executable': 'Executable',
  'accountWorkbench.stats.executableMeta': 'Ready accounts',
  'accountWorkbench.stats.selected': 'Selected',
  'accountWorkbench.stats.selectedMeta': 'Current selection',
  'accountWorkbench.stats.pending': 'Queued',
  'accountWorkbench.stats.pendingMeta': 'Waiting tasks',
  'accountWorkbench.stats.running': 'Running',
  'accountWorkbench.stats.runningMeta': 'Active tasks',
  'accountWorkbench.stats.success': 'Success',
  'accountWorkbench.stats.successMeta': 'Succeeded tasks',
  'accountWorkbench.stats.failed': 'Failed',
  'accountWorkbench.stats.failedMeta': 'Failed tasks',
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
  'accountWorkbench.actions.login': 'Login',
  'accountWorkbench.actions.login_check': 'Login check',
  'accountWorkbench.actions.follow': 'Follow',
  'accountWorkbench.actions.like': 'Like',
  'accountWorkbench.actions.retweet': 'Retweet',
  'accountWorkbench.actions.post': 'Post',
  'accountWorkbench.actions.update_profile': 'Update profile',
  'accountWorkbench.actions.update_avatar': 'Update avatar',
  'accountWorkbench.actions.update_banner': 'Update banner',
  'accountWorkbench.execution.noActions': 'No actions',
  'accountWorkbench.execution.noTemplates': 'No templates',
  'accountWorkbench.execution.defaultTemplateMissing': 'Default template missing',
  'accountWorkbench.execution.defaultTemplateDescription': 'Default template: {template} - {summary}',
  'accountWorkbench.execution.defaultTemplate': 'Default',
  'accountWorkbench.execution.start': 'Submit task',
  'accountWorkbench.execution.confirmTitle': 'Confirm task execution',
  'accountWorkbench.execution.confirmHint': 'Submit {count} account(s) for {action} using default template {template}.',
  'accountWorkbench.execution.confirmSubmit': 'Submit task',
  'accountWorkbench.execution.actionType': 'Function',
  'accountWorkbench.execution.templateType': 'Template type',
  'accountWorkbench.execution.targets': 'Targets',
  'accountWorkbench.execution.contents': 'Contents',
  'accountWorkbench.execution.profileFields': 'Profile fields',
  'accountWorkbench.execution.media': 'Media',
  'accountWorkbench.execution.templateDetails': 'Template details',
  'accountWorkbench.execution.executionDetails': 'Execution details',
  'accountWorkbench.execution.accountSummary': 'Account summary',
  'accountWorkbench.execution.loginSummary': 'Logs in with the account password and captures execution credentials; no extra parameters.',
  'accountWorkbench.execution.loginCheckSummary': 'No extra parameters.',
  'accountWorkbench.execution.targetPoolSummary': '{count} target(s)',
  'accountWorkbench.execution.contentPoolSummary': '{count} content item(s)',
  'accountWorkbench.execution.postRichSummary': '{count} content item(s), {media} media item(s), quote link {quote}.',
  'accountWorkbench.execution.profileSummary': '{count} profile field(s)',
  'accountWorkbench.execution.avatarSummary': '1 avatar image ready.',
  'accountWorkbench.execution.bannerSummary': '1 banner image ready.',
  'accountWorkbench.execution.resultSummary': 'Submitted {submitted}; queued {enqueued}; failed {failed}.',
  'accountWorkbench.execution.taskSummaryNoDetails': 'No details',
  'accountWorkbench.execution.selectAccountsFirst': 'Select accounts first.',
  'accountWorkbench.execution.nonExecutableSelected': 'Non-executable account selected.',
  'accountWorkbench.execution.loginProxyRequired': 'Login needs a default proxy.',
  'accountWorkbench.execution.loginPasswordRequired': 'Login needs the account password.',
  'accountWorkbench.execution.loginProxyAndPasswordRequired': 'Login needs a default proxy and password.',
  'accountWorkbench.execution.mixedPlatforms': 'Mixed platforms are not allowed.',
  'accountWorkbench.execution.platformUnavailable': 'Platform unavailable.',
  'accountWorkbench.execution.defaultTemplateRequired': 'Default template required.',
  'accountWorkbench.execution.templatesUnavailable': 'Templates unavailable.',
  'accountWorkbench.execution.defaultTemplateInvalid': 'Default template invalid.',
  'accountWorkbench.execution.submitFailed': 'Submit failed',
  'accountWorkbench.execution.errors.SOCIAL_TASK_SERVICE_UNAVAILABLE': 'Task service is temporarily unavailable.',
  'accountWorkbench.execution.errors.TASK_TEMPLATE_SERVICE_UNAVAILABLE': 'Task template service is temporarily unavailable.',
  'accountWorkbench.execution.errors.SOCIAL_IP_SERVICE_UNAVAILABLE': 'Proxy service is temporarily unavailable.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_INSUFFICIENT_FUNDS': 'Insufficient balance: required {required_total}, balance {wallet_balance}, need {wallet_required}.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_INPUT_REQUIRED': 'Task submission details are incomplete. Refresh and submit again.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_ACCOUNTS_REQUIRED': 'Select at least one account before submitting.',
  'accountWorkbench.execution.errors.TASK_DEFAULT_TEMPLATE_REQUIRED': 'Default task template is missing. Set it again before submitting.',
  'accountWorkbench.execution.errors.SOCIAL_IP_NOT_AVAILABLE': 'Default proxy unavailable. Reassign a usable proxy before submitting.',
  'accountWorkbench.execution.errors.GLOBAL_PROXY_NOT_AVAILABLE': 'Global proxy unavailable. Contact an administrator.',
  'accountWorkbench.execution.errors.GLOBAL_PROXY_SERVICE_UNAVAILABLE': 'Global proxy service unavailable.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_PLATFORM_REQUIRED': 'A selected account is missing platform information. Refresh and select again.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_ACCOUNT_ID_INVALID': 'Selected account list is stale. Refresh and select again.',
  'accountWorkbench.execution.errors.SOCIAL_TASK_MIXED_PLATFORMS': 'Selected accounts must belong to the same platform. Refresh and submit one platform at a time.',
  'accountWorkbench.execution.submitted': 'Submitted {count}',
  'accountWorkbench.execution.actionPlaceholder': 'Select an action',
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
  'accountWorkbench.batchDeleteResultSummary': 'Total {total}, removed {removed}, failed {failed}, skipped {skipped}.',
  'accountWorkbench.sections.managementHint': 'Manage SocialOps account credentials here.',
  'accountWorkbench.detailSections.identity': 'Identity',
  'accountWorkbench.detailSections.credentials': 'Credentials',
  'accountWorkbench.detailSections.operations': 'Operations',
  'accountWorkbench.edit.title': 'Edit account',
  'accountWorkbench.edit.identityTitle': 'Identity',
  'accountWorkbench.edit.identityHint': 'Identity fields are read-only.',
  'accountWorkbench.edit.saved': 'Saved',
  'accountWorkbench.edit.failed': 'Save failed',
  'accountWorkbench.edit.noChanges': 'No changes to save.',
  'accountWorkbench.edit.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID': 'Execution auth format is invalid.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE': 'Account service is temporarily unavailable.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED': 'Account details are incomplete. Check the form and try again.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED': 'Enter the account password.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE': 'Account delivery is incomplete. Provide a password and at least one login credential: 2FA, complete email credentials, or auth cookie.',
  'admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND': 'The account no longer exists or was updated. Refresh the list and try again.',
  'accountWorkbench.credentials.title': 'Credential Details',
  'accountWorkbench.credentials.previewHint': 'Long credential values are summarized by character count here.',
  'accountWorkbench.credentials.authCookieDescription': 'Login auth cookie.',
  'accountWorkbench.credentials.executionAuthDescription': 'Encrypted execution auth.',
  'accountWorkbench.credentials.copy': 'Copy',
  'accountWorkbench.credentials.copyRaw': 'Copy raw {field}',
  'accountWorkbench.credentials.copied': 'Credential copied.',
  'accountWorkbench.credentials.copyFailed': 'Copy failed',
  'accountWorkbench.credentials.emptyCopy': 'Empty field',
  'accountWorkbench.credentials.refresh': 'Refresh',
  'accountWorkbench.credentials.refreshTitle': 'Refresh execution auth',
  'accountWorkbench.credentials.refreshSubmitted': 'Submitted {count}, queued {enqueued}.',
  'accountWorkbench.credentials.refreshFailed': 'Refresh failed',
  'accountWorkbench.credentials.executionAuthAlreadyReady': 'Execution auth already ready.',
  'accountWorkbench.credentials.refreshNeedsProxy': 'Refresh needs proxy.',
  'accountWorkbench.credentials.refreshNeedsPassword': 'Refresh needs password.',
  'accountWorkbench.credentials.refreshNeedsProxyAndPassword': 'Refresh needs proxy and password.',
  'accountWorkbench.credentials.empty': 'Not configured',
  'accountWorkbench.credentials.length': '{count} chars',
  'accountWorkbench.credentials.encryptedStored': 'Encrypted value stored',
  'accountWorkbench.credentials.oauthReady': 'OAuth complete',
  'accountWorkbench.credentials.rawCookieDetected': 'Raw cookie',
  'accountWorkbench.credentials.oauthPartial': 'OAuth incomplete',
  'accountWorkbench.credentials.jsonDetected': 'JSON detected',
  'accountWorkbench.credentials.loginRefreshRequired': 'Login refresh required to capture execution auth',
  'admin.socialAccountWorkbench.executionBar.clear': 'Clear selection',
  'admin.socialAccountWorkbench.executionBar.selectionRequired': 'Select at least one account first.',
  'admin.socialAccountWorkbench.actions.fileUpload': 'Upload to Pool',
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
  'admin.socialAccountWorkbench.form.executionAuthHelp': 'Provide encrypted execution auth only.',
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
      listTaskLogs,
      setDefaultProxy,
      batchSetDefaultProxy,
      exportMyAccounts,
    },
    accountWorkbenchAdminAPI: {
      ...actual.accountWorkbenchAdminAPI,
      storeWorkbenchAccounts,
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

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAdmin: adminState.isAdmin,
    user: { id: 42 },
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
      <div data-testid="data-table-header">
        <slot name="header-select" />
      </div>
      <div v-for="row in data" :key="rowId(row)" :data-testid="\`account-row-\${rowId(row)}\`">
        <slot name="cell-select" :row="row" />
        <slot name="cell-id" :row="row" :value="rowValue(row, 'id')" />
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

function buildAccount(id = 101, overrides: Record<string, unknown> = {}) {
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
    ...overrides,
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
  const wrapper = mount(UnifiedAccountWorkbenchView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<section v-if="show" role="dialog"><button type="button" aria-label="Close modal" @click="$emit(\'close\')"></button><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
        },
        DataTable: DataTableStub,
        SearchInput: SearchInputStub,
        Select: SelectStub,
        Icon: true,
      },
    },
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

async function waitForWorkbench() {
  await flushPromises()
  await flushPromises()
}

function findActionSelect(wrapper: ReturnType<typeof mount>) {
  const select = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('Post') || node.text().includes('Update avatar'))
  expect(select).toBeTruthy()
  return select!
}

async function chooseExecutionAction(wrapper: ReturnType<typeof mount>, action: string) {
  await findActionSelect(wrapper).setValue(action)
  await waitForWorkbench()
}

function findExecutionStartButton(wrapper: ReturnType<typeof mount>) {
  const button = wrapper.findAll('button').find(node => node.text().includes('Submit task'))
  expect(button).toBeTruthy()
  return button!
}

function expectConstrainedActionButton(button: DOMWrapper<Element>, label: string) {
  expect(button.attributes('aria-label')).toBe(label)
  expect(button.attributes('title')).toBe(label)
  expect(button.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
  const text = button.findAll('span').find(node => node.text() === label)
  expect(text, `${label} button text`).toBeTruthy()
  expect(text!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
}

function getEmptyStateButton(wrapper: ReturnType<typeof mount>, label: string) {
  const button = wrapper.findAll('.py-8.text-center button')
    .find(node => node.attributes('aria-label') === label)
  expect(button, `empty-state button "${label}"`).toBeTruthy()
  return button!
}

function expectConstrainedIconButton(button: DOMWrapper<Element>, label: string) {
  expect(button.attributes('aria-label')).toBe(label)
  expect(button.attributes('title')).toBe(label)
  expect(button.classes()).toEqual(expect.arrayContaining(['h-8', 'w-8', 'px-0']))
}

function expectConstrainedDialogButton(button: DOMWrapper<Element>, label: string) {
  expectConstrainedActionButton(button, label)
}

function expectPoliteStatus(element: DOMWrapper<Element>) {
  expect(element.attributes('role')).toBe('status')
  expect(element.attributes('aria-live')).toBe('polite')
  expect(element.attributes('aria-atomic')).toBe('true')
}

async function clickLatestDialogClose(wrapper: ReturnType<typeof mount>) {
  const closeButtons = wrapper.findAll('button[aria-label="Close modal"]')
  expect(closeButtons.length, 'dialog close buttons').toBeGreaterThan(0)
  await closeButtons[closeButtons.length - 1].trigger('click')
}

async function openBatchProxyDialogForSelectedAccounts(wrapper: ReturnType<typeof mount>) {
  const button = wrapper.findAll('button').find(node => node.text().includes('Batch proxy'))
  expect(button).toBeTruthy()
  expect(button!.attributes('disabled')).toBeUndefined()
  await button!.trigger('click')
  await waitForWorkbench()
}

describe('accounts UnifiedAccountWorkbenchView', () => {
  beforeEach(() => {
    localStorage.clear()
    listMyAccounts.mockReset()
    batchImportMyAccounts.mockReset()
    deleteMyAccount.mockReset()
    batchDeleteMyAccounts.mockReset()
    updateMyAccount.mockReset()
    submitTask.mockReset()
    listTaskLogs.mockReset()
    setDefaultProxy.mockReset()
    batchSetDefaultProxy.mockReset()
    exportMyAccounts.mockReset()
    storeWorkbenchAccounts.mockReset()
    listUsable.mockReset()
    listTemplates.mockReset()
    previewMedia.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    showSuccess.mockReset()
    recordClientDiagnostic.mockReset()
    writeClipboard.mockReset()
    writeClipboard.mockResolvedValue(undefined)
    adminState.isAdmin = false

    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: writeClipboard },
    })

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
    listTaskLogs.mockResolvedValue({ logs: [] })
  })

  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) {
      wrapper.unmount()
    }
    globalThis.URL.createObjectURL = originalCreateObjectURL
    globalThis.URL.revokeObjectURL = originalRevokeObjectURL
    if (originalClipboardDescriptor) {
      Object.defineProperty(navigator, 'clipboard', originalClipboardDescriptor)
    } else {
      delete (navigator as { clipboard?: Clipboard }).clipboard
    }
  })

  it('keeps the account load error readable in the existing retry panel', async () => {
    listMyAccounts.mockRejectedValue({})
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Failed to load')
    expect(wrapper.text().match(/Failed to load/g)).toHaveLength(1)
    expect(wrapper.find('p[title="Failed to load"]').exists()).toBe(false)
    const retryButton = wrapper.get('button[aria-label="Retry"]')
    expectConstrainedActionButton(retryButton, 'Retry')
  })

  it('maps account service availability load errors to the existing retry panel', async () => {
    const loadErrorMessage = 'Account service is temporarily unavailable.'
    listMyAccounts.mockRejectedValue({ reason: 'SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE', message: 'social account service is unavailable' })
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.load_data', expect.any(Object))
    expect(showError).toHaveBeenCalledWith(loadErrorMessage)
    expect(wrapper.text()).toContain('Failed to load')
    const errorMessage = wrapper.findAll('p').find(node => node.text() === loadErrorMessage)
    expect(errorMessage).toBeTruthy()
    expect(errorMessage!.attributes('title')).toBe(loadErrorMessage)
  })

  it('keeps optional dependency warnings readable in the existing retry panel', async () => {
    listUsable.mockRejectedValue(new Error('proxy dependency request failed'))
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const dependencyMessage = wrapper.find('p[title="Proxy dependency unavailable"]')
    expect(dependencyMessage.exists()).toBe(true)
    expect(dependencyMessage.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    const dependencyStatus = wrapper.get('[role="status"]')
    expect(dependencyStatus.attributes('aria-live')).toBe('polite')
    expect(dependencyStatus.attributes('aria-atomic')).toBe('true')
    expect(dependencyStatus.text()).toContain('Dependencies unavailable')
    expect(dependencyStatus.text()).toContain('Proxy dependency unavailable')
    const retryButton = wrapper.get('button[aria-label="Retry"]')
    expectConstrainedActionButton(retryButton, 'Retry')
  })

  it('hides admin-only upload-to-pool actions from regular users', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Register Account')
    expect(wrapper.text()).not.toContain('Upload to Pool')
  })

  it('defaults the account table to stable ID ascending order instead of mutable update time', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101, { name: 'created-first', updated_at: '2026-06-01T00:00:00Z' }),
        buildAccount(202, { name: 'updated-later', updated_at: '2026-06-24T08:00:00Z' }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const table = wrapper.getComponent(DataTableStub)
    expect(table.props('defaultSortKey')).toBe('id')
    expect(table.props('defaultSortOrder')).toBe('asc')
    expect((table.props('data') as Array<{ id: number }>).map(account => account.id)).toEqual([101, 202])
  })

  it('renders account IDs as their own table column instead of burying them in account names', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, { name: 'northwind_ops', username: 'northwind_ops' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const table = wrapper.getComponent(DataTableStub)
    expect((table.props('columns') as Array<{ key: string }>).map(column => column.key).slice(0, 3)).toEqual(['select', 'id', 'name'])
    const row = wrapper.get('[data-testid="account-row-101"]')
    expect(row.text()).toContain('101')
    expect(row.text()).not.toContain('#101')
    expect(row.text()).toContain('northwind_ops')
  })

  it('keeps account toolbar action labels inspectable and constrained on narrow layouts', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    for (const [label, disabled, title] of [
      ['Refresh', false, 'Refresh'],
      ['Batch import', false, 'Batch import'],
      ['Export accounts', false, 'Export accounts'],
      ['Batch proxy', true, 'Select at least one account first.'],
      ['Submit task', true, 'Select accounts first.'],
    ] as const) {
      const button = wrapper.findAll('button')
        .find(node => node.text().includes(label) && node.attributes('title') === title)
      expect(button, `${label} toolbar button should be present`).toBeTruthy()
      expect(button!.attributes('aria-label')).toBe(title)
      expect(button!.attributes('title')).toBe(title)
      expect(button!.attributes('disabled') !== undefined).toBe(disabled)
      expect(button!.classes()).toEqual(expect.arrayContaining(['h-9', 'min-w-0', 'max-w-full', 'justify-center']))
      expect(button!.get('span.min-w-0.truncate').exists()).toBe(true)
    }

    const blockedSelectionButtons = wrapper.findAll('button[title="Select at least one account first."]')
    expect(blockedSelectionButtons.some(button => button.attributes('aria-label') === 'Select at least one account first.')).toBe(true)
    expect(blockedSelectionButtons.length).toBeGreaterThanOrEqual(3)
  })

  it('keeps row-level account actions named, constrained, and on their existing flows', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = wrapper.get('[data-testid="account-row-101"]')
    const proxyButton = row.get('button[aria-label="Proxy"]')
    expectConstrainedActionButton(proxyButton, 'Proxy')
    expect(proxyButton.attributes('disabled')).toBeUndefined()
    await proxyButton.trigger('click')
    await waitForWorkbench()
    expect(wrapper.get('section[role="dialog"]').text()).toContain('Proxy')

    await wrapper.get('section[role="dialog"] button[aria-label="Cancel"]').trigger('click')
    await waitForWorkbench()

    const editButton = row.get('button[aria-label="Edit"]')
    expectConstrainedIconButton(editButton, 'Edit')
    expect(editButton.attributes('disabled')).toBeUndefined()
    await editButton.trigger('click')
    await waitForWorkbench()
    const editDialog = wrapper.get('section[role="dialog"]')
    expect(editDialog.text()).toContain('Edit account')

    const editCancelButton = editDialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(editCancelButton, 'Cancel')
    expect(editCancelButton.attributes('disabled')).toBeUndefined()

    const editSaveButton = editDialog.get('button[aria-label="Save"]')
    expect(editSaveButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    expect(editSaveButton.findAll('span').find(node => node.text() === 'Save')?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(editSaveButton.attributes('disabled')).toBeDefined()
    expect(editSaveButton.attributes('title')).toBe('No changes to save.')

    await wrapper.get('#account-edit-email').setValue('changed@example.test')
    await waitForWorkbench()
    const changedSaveButton = editDialog.get('button[aria-label="Save"]')
    expect(changedSaveButton.attributes('disabled')).toBeUndefined()
    expect(changedSaveButton.attributes('title')).toBe('Save')

    await editCancelButton.trigger('click')
    await waitForWorkbench()

    const deleteButton = row.get('button[aria-label="Delete account"]')
    expectConstrainedIconButton(deleteButton, 'Delete account')
    expect(deleteButton.attributes('disabled')).toBeUndefined()
    await deleteButton.trigger('click')
    await waitForWorkbench()
    expect(wrapper.get('section[role="dialog"]').text()).toContain('Delete accounts')
  })

  it('reloads user accounts through backend filters when workbench filters change', async () => {
    listTemplates.mockResolvedValue([])
    const northwind = buildAccount(501, { name: 'northwind_ops', username: 'northwind_ops' })
    const filtered = buildAccount(777, {
      name: 'delivery_filter_ops',
      username: 'delivery_filter_ops',
      account_status: 'not_stored',
      task_status: 'pending',
      default_proxy_configured: false,
      password: 'pool-secret',
    })
    listMyAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === '#777' && params.account_status === 'invalid' && params.platform === 'x_twitter') {
        return Promise.resolve({
          items: [filtered],
          total: 1,
          page: 1,
          page_size: 200,
          pages: 1,
        })
      }
      return Promise.resolve({
        items: [northwind, filtered],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenCalledWith({ page: 1, page_size: 200 })
    expect(wrapper.find('[data-testid="account-row-501"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-row-777"]').exists()).toBe(true)

    await wrapper.get('[data-testid="search-input-stub"]').setValue('#777')
    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('invalid')
    await selects[1].setValue('x_twitter')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      search: '#777',
      platform: 'x_twitter',
      account_status: 'invalid',
    })
    expect(wrapper.find('[data-testid="account-row-501"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-777"]').text()).toContain('delivery_filter_ops')
  })

  it('shows the empty workbench state with the existing batch import action constrained', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 0,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.text()).toContain('No accounts')
    expect(wrapper.text()).toContain('Import accounts to get started.')
    expect(wrapper.text()).not.toContain('No results')

    const batchImportButton = getEmptyStateButton(wrapper, 'Batch import')
    expectConstrainedActionButton(batchImportButton, 'Batch import')
    expect(batchImportButton.attributes('disabled')).toBeUndefined()
    await batchImportButton.trigger('click')
    await waitForWorkbench()

    expect(wrapper.get('section[role="dialog"]').text()).toContain('Batch import')
  })

  it('shows the filtered empty state when backend filters return no workbench accounts', async () => {
    listTemplates.mockResolvedValue([])
    const northwind = buildAccount(501, { name: 'northwind_ops', username: 'northwind_ops' })
    listMyAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === 'missing-workbench-account') {
        return Promise.resolve({
          items: [],
          total: 0,
          page: 1,
          page_size: 200,
          pages: 0,
        })
      }
      return Promise.resolve({
        items: [northwind],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-501"]').text()).toContain('northwind_ops')

    await wrapper.get('[data-testid="search-input-stub"]').setValue('missing-workbench-account')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 200,
      search: 'missing-workbench-account',
    })
    expect(wrapper.find('[data-testid="account-row-501"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('No results')
    expect(wrapper.text()).toContain('Adjust your filters.')
    expect(wrapper.text()).toContain('Clear filters')
    expect(wrapper.text()).not.toContain('No accounts')
    expect(wrapper.text()).not.toContain('Import accounts to get started.')

    const clearFiltersButton = getEmptyStateButton(wrapper, 'Clear filters')
    expectConstrainedActionButton(clearFiltersButton, 'Clear filters')
    await clearFiltersButton.trigger('click')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenLastCalledWith({ page: 1, page_size: 200 })
    expect(wrapper.get('[data-testid="account-row-501"]').text()).toContain('northwind_ops')
  })

  it('normalizes account and task statuses before stats and execution gates use them', async () => {
    listTemplates.mockResolvedValue([buildPostTemplate()])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101, {
          account_status: ' Available ',
          task_status: ' Idle ',
          default_proxy_configured: true,
        }),
        buildAccount(202, {
          account_status: ' LIMITED ',
          task_status: ' Failed ',
          default_proxy_configured: false,
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Available')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Idle')
    expect(wrapper.get('[data-testid="account-stat-executable"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="account-stat-abnormal"]').text()).toContain('1')

    await chooseExecutionAction(wrapper, 'post')
    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    expect(findExecutionStartButton(wrapper).attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).not.toContain('Non-executable account selected.')
  })

  it('keeps long default-template names readable in the existing execution confirmation hint', async () => {
    const longTemplateName = 'stage117_post_template_with_really_long_unbroken_identifier_0123456789abcdef'
    const longAccountName = 'stage117_account_ready_for_post_confirmation'
    listTemplates.mockResolvedValue([{ ...buildPostTemplate(), name: longTemplateName }])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        name: longAccountName,
        username: longAccountName,
        account_status: 'available',
        task_status: 'idle',
        default_proxy_configured: true,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await chooseExecutionAction(wrapper, 'post')
    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const expectedHint = `Submit 1 account(s) for Post using default template ${longTemplateName}.`
    const hint = wrapper.find(`[title="${expectedHint}"]`)
    expect(hint.exists()).toBe(true)
    expect(hint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expectPoliteStatus(hint)
    expect(wrapper.text()).toContain('Confirm task execution')
    expect(wrapper.text()).toContain(longTemplateName)
  })

  it('keeps execution confirmation footer actions inspectable and constrained on narrow layouts', async () => {
    listTemplates.mockResolvedValue([buildPostTemplate()])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount()],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await chooseExecutionAction(wrapper, 'post')
    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Confirm task execution')

    const cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')
    expect(cancelButton.attributes('disabled')).toBeUndefined()

    const submitButton = dialog.get('button[aria-label="Submit task"]')
    expectConstrainedDialogButton(submitButton, 'Submit task')
    expect(submitButton.attributes('disabled')).toBeUndefined()
  })

  it('keeps proxy assignment footer actions inspectable and constrained on narrow layouts', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([{
      id: 301,
      user_id: 1,
      name: 'east-proxy',
      ip_type: 'residential',
      endpoint: 'http://proxy.example:8080',
      status: 'online',
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-01T00:00:00Z',
    }])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101), buildAccount(202)],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowProxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(rowProxyButton).toBeTruthy()
    await rowProxyButton!.trigger('click')
    await waitForWorkbench()

    let dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Proxy')
    expect(dialog.text()).toContain('Current account: x-main-101')
    expectPoliteStatus(dialog.get('[title="Current account: x-main-101"]'))

    let proxySelect = dialog.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    let cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')
    expect(cancelButton.attributes('disabled')).toBeUndefined()

    const singleApplyButton = dialog.get('button[aria-label="Apply proxy"]')
    expectConstrainedDialogButton(singleApplyButton, 'Apply proxy')
    expect(singleApplyButton.attributes('disabled')).toBeUndefined()

    await cancelButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Batch proxy')
    expect(dialog.text()).toContain('Apply to 2 accounts.')
    expectPoliteStatus(dialog.get('[title="Apply to 2 accounts."]'))

    proxySelect = dialog.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')

    const batchApplyButton = dialog.get('button[aria-label="Apply proxy"]')
    expectConstrainedDialogButton(batchApplyButton, 'Apply proxy')
    expect(batchApplyButton.attributes('disabled')).toBeUndefined()
  })

  it('keeps batch import footer actions inspectable and constrained on narrow layouts', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Batch import')
    expectPoliteStatus(dialog.get('[title="Paste text or choose a file."]'))
    expectPoliteStatus(dialog.get('[title="Preview is scoped to your workbench."]'))

    const textarea = dialog.get('textarea')
    await textarea.setValue('@stage_import_ops\tstage-pass\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')
    expect(cancelButton.attributes('disabled')).toBeUndefined()

    const confirmButton = dialog.get('button[aria-label="Confirm"]')
    expectConstrainedDialogButton(confirmButton, 'Confirm')
    expect(confirmButton.attributes('disabled')).toBeUndefined()
  })

  it('keeps upload-to-pool footer actions inspectable and constrained on narrow layouts', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    expect(uploadButton!.attributes('disabled')).toBeUndefined()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Upload to Pool')
    expectPoliteStatus(dialog.get('[title="Upload 1 selected not-stored account(s)."]'))
    expect(dialog.text()).toContain('staging-account')

    const cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')
    expect(cancelButton.attributes('disabled')).toBeUndefined()

    const confirmButton = dialog.get('button[aria-label="Confirm upload"]')
    expectConstrainedDialogButton(confirmButton, 'Confirm upload')
    expect(confirmButton.attributes('disabled')).toBeUndefined()
  })

  it('trims backend account names before table, detail, and edit identity surfaces display them', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(909, {
          name: '  padded_account_name  ',
          username: 'padded_account_name',
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = wrapper.get('[data-testid="account-row-909"]')
    expect(row.element.textContent).toContain('padded_account_name')
    expect(row.element.textContent).not.toContain('  padded_account_name  ')

    await row.get('button').trigger('click')
    await waitForWorkbench()
    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('padded_account_name')
    expect(detailDialog.element.textContent).not.toContain('  padded_account_name  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForWorkbench()

    const editButton = wrapper.findAll('[data-testid="account-row-909"] button')
      .find(node => ['Edit', 'common.edit'].includes(node.attributes('aria-label') || ''))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await waitForWorkbench()

    const editIdentity = wrapper.get('[data-testid="account-edit-identity"]')
    expect(editIdentity.text()).toContain('padded_account_name')
    expect(editIdentity.element.textContent).not.toContain('  padded_account_name  ')
    const identityHint = wrapper.get('[title="Identity fields are read-only."]')
    expect(identityHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })

  it('trims backend task messages before account details display them', async () => {
    const taskMessage = 'executor_unavailable_with_really_long_backend_reason_0123456789abcdef'
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(910, {
          task_status: 'manual_review',
          task_message: `  ${taskMessage}  `,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-910"] button').trigger('click')
    await waitForWorkbench()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain(taskMessage)
    expect(detailDialog.element.textContent).not.toContain(`  ${taskMessage}  `)
    const taskMessagePanel = detailDialog.get(`[title="${taskMessage}"]`)
    expect(taskMessagePanel.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(taskMessagePanel.attributes('role')).toBe('status')
    expect(taskMessagePanel.attributes('aria-live')).toBe('polite')
    expect(taskMessagePanel.attributes('aria-atomic')).toBe('true')
  })

  it('trims backend usernames before table and detail identity surfaces display them', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(911, {
          name: 'stage54_account',
          username: '  stage54_user  ',
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = wrapper.get('[data-testid="account-row-911"]')
    expect(row.element.textContent).toContain('stage54_user')
    expect(row.element.textContent).not.toContain('  stage54_user  ')

    await row.get('button').trigger('click')
    await waitForWorkbench()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('stage54_user')
    expect(detailDialog.element.textContent).not.toContain('  stage54_user  ')
  })

  it('trims backend platform user IDs before detail and edit identity surfaces display them', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(912, {
          platform_user_id: '  uid-stage56  ',
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = wrapper.get('[data-testid="account-row-912"]')
    await row.get('button').trigger('click')
    await waitForWorkbench()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('uid-stage56')
    expect(detailDialog.element.textContent).not.toContain('  uid-stage56  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForWorkbench()

    const editButton = wrapper.findAll('[data-testid="account-row-912"] button')
      .find(node => ['Edit', 'common.edit'].includes(node.attributes('aria-label') || ''))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await waitForWorkbench()

    const editIdentity = wrapper.get('[data-testid="account-edit-identity"]')
    expect(editIdentity.text()).toContain('uid-stage56')
    expect(editIdentity.element.textContent).not.toContain('  uid-stage56  ')
  })

  it('trims backend registration IPs before detail and edit identity surfaces display them', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(913, {
          registration_ip: '  203.0.113.58  ',
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = wrapper.get('[data-testid="account-row-913"]')
    await row.get('button').trigger('click')
    await waitForWorkbench()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('203.0.113.58')
    expect(detailDialog.element.textContent).not.toContain('  203.0.113.58  ')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForWorkbench()

    const editButton = wrapper.findAll('[data-testid="account-row-913"] button')
      .find(node => ['Edit', 'common.edit'].includes(node.attributes('aria-label') || ''))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await waitForWorkbench()

    const editIdentity = wrapper.get('[data-testid="account-edit-identity"]')
    expect(editIdentity.text()).toContain('203.0.113.58')
    expect(editIdentity.element.textContent).not.toContain('  203.0.113.58  ')
  })

  it('lets admins upload selected not-stored workbench accounts into the total pool', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101),
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 202, name: 'staging-account', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Register Account')
    expect(wrapper.text()).toContain('Upload to Pool')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(2)
    await checkboxes[2].setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    expect(uploadButton!.attributes('disabled')).toBeUndefined()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Accounts to upload')
    expect(wrapper.text()).toContain('staging-account')
    const uploadHint = wrapper.get('[title="Upload 1 selected not-stored account(s)."]')
    expect(uploadHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202])
    expect(showSuccess).toHaveBeenCalledWith('Upload saved')
  })

  it('keeps long account labels readable in existing confirmation previews', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    const longExecutableName = 'stage109_account_execution_delete_name_with_really_long_unbroken_identifier_0123456789abcdef'
    const longStagingName = 'stage109_account_store_workbench_name_with_really_long_unbroken_identifier_0123456789abcdef'
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101, {
          name: longExecutableName,
          username: longExecutableName,
          default_proxy_configured: true,
          default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
        }),
        buildAccount(202, {
          name: longStagingName,
          username: longStagingName,
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const expectAccountPreview = (name: string) => {
      const preview = wrapper.findAll('[title]').find(node => node.attributes('title') === name && node.classes().includes('break-all'))
      expect(preview, `preview for ${name} should be readable`).toBeTruthy()
      expect(preview!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-all', 'sm:truncate']))
    }
    const expectPreviewMeta = (title: string) => {
      const preview = wrapper.findAll('[title]').find(node => node.attributes('title') === title && node.classes().includes('text-right'))
      expect(preview, `preview metadata ${title} should be readable`).toBeTruthy()
      expect(preview!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words', 'text-right', 'sm:truncate']))
    }
    const expectReadableHint = (title: string) => {
      const hint = wrapper.findAll('[title]').find(node => node.attributes('title') === title && node.classes().includes('break-words'))
      expect(hint, `hint ${title} should keep its full readable text`).toBeTruthy()
      expect(hint!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
      expectPoliteStatus(hint!)
    }

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Accounts to upload')
    expectAccountPreview(longStagingName)
    expectPreviewMeta('X / Twitter · Not stored')

    await wrapper.findAll('button').find(node => node.text() === 'Cancel')!.trigger('click')
    await waitForWorkbench()
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(false)
    await waitForWorkbench()

    const deleteButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Delete accounts')
    expectReadableHint(`Delete ${longExecutableName}.`)
    expectReadableHint('Deletion cannot be undone.')
    expectAccountPreview(longExecutableName)
    expectPreviewMeta('X / Twitter · Available')

    await wrapper.findAll('button').find(node => node.text() === 'Cancel')!.trigger('click')
    await waitForWorkbench()

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    expectReadableHint(`Current account: ${longExecutableName}`)

    await wrapper.findAll('button').find(node => node.text() === 'Cancel')!.trigger('click')
    await waitForWorkbench()
    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')
    expectAccountPreview(longExecutableName)
    expectPreviewMeta('X / Twitter · Configured')
  })

  it('disables upload-to-pool confirmation while the request is in flight', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    let resolveUpload!: (value: {
      total: number
      succeeded: number
      skipped: number
      failed: number
      items: Array<{ id: number; name: string; status: string }>
    }) => void
    storeWorkbenchAccounts.mockReturnValue(new Promise((resolve) => {
      resolveUpload = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    const processingButton = wrapper.findAll('button').find(node => node.text().includes('Processing'))
    expect(processingButton).toBeTruthy()
    expect(processingButton!.attributes('disabled')).toBeDefined()
    const cancelButton = wrapper.findAll('section[role="dialog"] button')
      .find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('aria-label')).toBe('Processing')
    expect(cancelButton!.attributes('title')).toBe('Processing')

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    await processingButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Accounts to upload')
    expect(storeWorkbenchAccounts).toHaveBeenCalledTimes(1)
    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202])

    resolveUpload({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 202, name: 'staging-account', status: 'succeeded' }],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Upload saved')
  })

  it('keeps upload-to-pool request failures visible without exposing raw backend details', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts.mockRejectedValue(new Error('raw upload failure token=secret'))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202])
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.store_workbench_accounts', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Upload failed')
    expect(showError).not.toHaveBeenCalledWith(expect.stringContaining('raw upload failure'))

    const dialog = wrapper.get('section[role="dialog"]')
    const alert = dialog.get('[role="alert"]')
    expect(dialog.text()).toContain('Accounts to upload')
    expect(dialog.text()).toContain('staging-account')
    expect(alert.text()).toBe('Upload failed')
    expect(alert.attributes('title')).toBe('Upload failed')
    expect(dialog.text()).not.toContain('raw upload failure')
    expect(dialog.text()).not.toContain('token=secret')
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()
    expect(wrapper.text()).not.toContain('Upload failed')

    await uploadButton!.trigger('click')
    await waitForWorkbench()
    expect(wrapper.find('section[role="dialog"] [role="alert"]').exists()).toBe(false)
  })

  it('keeps failed upload-to-pool staging accounts selected after a partial result refresh', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(202, {
            name: 'staging-success',
            username: 'staging-success',
            account_status: 'not_stored',
            task_status: 'pending',
            default_proxy_configured: false,
          }),
          buildAccount(303, {
            name: 'staging-failed',
            username: 'staging-failed',
            account_status: 'not_stored',
            task_status: 'pending',
            default_proxy_configured: false,
          }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [
          buildAccount(303, {
            name: 'staging-failed',
            username: 'staging-failed',
            account_status: 'not_stored',
            task_status: 'pending',
            default_proxy_configured: false,
          }),
        ],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { id: 202, name: 'staging-success', status: 'succeeded' },
        { id: 303, name: 'staging-failed', status: 'failed', error: 'invalid credentials' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    expect(wrapper.text()).toContain('2 selected')
    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    expect(uploadButton!.attributes('disabled')).toBeUndefined()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Accounts to upload')
    expect(wrapper.text()).toContain('staging-success')
    expect(wrapper.text()).toContain('staging-failed')

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202, 303])
    expect(showWarning).toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Upload saved')
    expect(showError).not.toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-303"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).toContain('Total 2, succeeded 1, failed 1, skipped 0.')
  })

  it('removes succeeded upload-to-pool accounts locally before the next list refresh finishes', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    const failedAccount = buildAccount(303, {
      name: 'staging-failed',
      username: 'staging-failed',
      account_status: 'not_stored',
      task_status: 'pending',
      default_proxy_configured: false,
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(202, {
            name: 'staging-success',
            username: 'staging-success',
            account_status: 'not_stored',
            task_status: 'pending',
            default_proxy_configured: false,
          }),
          failedAccount,
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    storeWorkbenchAccounts.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { id: 202, name: 'staging-success', status: 'succeeded' },
        { id: 303, name: 'staging-failed', status: 'failed', error: 'invalid credentials' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202, 303])
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-303"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).toContain('Total 2, succeeded 1, failed 1, skipped 0.')

    resolveRefresh({
      items: [failedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('shows an error toast when upload-to-pool returns no successes', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(303, {
          name: 'staging-failed',
          username: 'staging-failed',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [
        { id: 303, name: 'staging-failed', status: 'failed', error: 'invalid credentials' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    expect(uploadButton!.attributes('disabled')).toBeUndefined()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([303])
    expect(showError).toHaveBeenCalledWith('Total 1, succeeded 0, failed 1, skipped 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Upload saved')
    expect(showWarning).not.toHaveBeenCalledWith('Total 1, succeeded 0, failed 1, skipped 0.')
    expect((wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).toContain('Error · Required credentials are incomplete')
    expect(wrapper.text()).not.toContain('Error · invalid credentials')
  })

  it('translates upload-to-pool batch result reasons instead of showing raw enums', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(303, {
          name: 'staging-failed',
          username: 'staging-failed',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [
        { id: 303, name: 'staging-failed', status: 'failed', reason: 'invalid_credentials' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Upload result')
    expect(wrapper.text()).toContain('Error · Required credentials are incomplete')
    expect(wrapper.text()).not.toContain('invalid_credentials')
  })

  it('translates upload-to-pool backend failure reasons before raw errors', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(303, {
          name: 'staging-failed',
          username: 'staging-failed',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [
        { id: 303, name: 'staging-failed', status: 'failed', reason: 'upload_failed', error: 'database timeout from backend' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Upload result')
    expect(wrapper.text()).toContain('Error · Upload failed')
    expect(wrapper.text()).not.toContain('database timeout from backend')
    expect(wrapper.text()).not.toContain('upload_failed')
  })

  it('clears the stale upload-to-pool result when a retry request fails', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(303, {
          name: 'staging-retry',
          username: 'staging-retry',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    storeWorkbenchAccounts
      .mockResolvedValueOnce({
        total: 1,
        succeeded: 0,
        skipped: 0,
        failed: 1,
        items: [
          { id: 303, name: 'staging-retry', status: 'failed', reason: 'invalid_credentials' },
        ],
      })
      .mockRejectedValueOnce(new Error('retry upload failed'))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = () => {
      const button = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
      expect(button).toBeTruthy()
      return button!
    }

    await confirmButton().trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Upload result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    await confirmButton().trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Upload failed')
    expect(wrapper.text()).not.toContain('Upload result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('closes upload-to-pool confirmation when storeable selected accounts disappear after refresh', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(202, {
            name: 'staging-account',
            username: 'staging-account',
            account_status: 'not_stored',
            task_status: 'pending',
            default_proxy_configured: false,
          }),
        ],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    expect(uploadButton!.attributes('disabled')).toBeUndefined()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Accounts to upload')
    expect(wrapper.text()).toContain('staging-account')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Accounts to upload')
    expect(wrapper.text()).not.toContain('Confirm upload')
    expect(storeWorkbenchAccounts).not.toHaveBeenCalled()
  })

  it('submits the default login function directly without template parameters', async () => {
    listTemplates.mockResolvedValue([])
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7000,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Execution details')
    expect(wrapper.text()).toContain('Logs in with the account password')

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      action: 'login',
    })
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0])).not.toContain('template_id')
    expect(typeof submitTask.mock.calls[0]?.[0]?.client_request_id).toBe('string')
    expect(String(submitTask.mock.calls[0]?.[0]?.client_request_id || '')).not.toBe('')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('allows login for a non-available account when password and default proxy are ready', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(202, {
        account_status: 'not_stored',
        task_status: 'pending',
        password: 'secret',
        default_proxy_configured: true,
        default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7001,
        social_account_id: 202,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const startButton = findExecutionStartButton(wrapper)
    expect(startButton.attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).not.toContain('Non-executable account selected.')

    await startButton.trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [202],
      action: 'login',
    })
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0])).not.toContain('template_id')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('closes task confirmation when selected accounts disappear after refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')
    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(submitTask).not.toHaveBeenCalled()
  })

  it('closes task confirmation when refreshed templates remove the selected default action', async () => {
    listTemplates
      .mockResolvedValueOnce([buildPostTemplate()])
      .mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await chooseExecutionAction(wrapper, 'post')
    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')
    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(submitTask).not.toHaveBeenCalled()
  })

  it('prevents duplicate task submission while the confirm request is in flight', async () => {
    listTemplates.mockResolvedValue([])
    let resolveSubmit!: (value: {
      submitted: number
      enqueued: number
      failed_closed: number
      logs: Array<{
        id: number
        social_account_id: number
        action: string
        status: string
        charged: boolean
        charged_amount: number
        charge_status: string
        created_at: string
      }>
    }) => void
    submitTask.mockReturnValue(new Promise((resolve) => {
      resolveSubmit = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    const confirmButton = confirmButtons[confirmButtons.length - 1]
    await confirmButton.trigger('click')
    await confirmButton.trigger('click')
    await waitForWorkbench()

    const processingButton = wrapper.findAll('button').find(node => node.text().includes('Processing'))
    expect(processingButton).toBeTruthy()
    expect(processingButton!.attributes('disabled')).toBeDefined()
    const cancelButton = wrapper.findAll('section[role="dialog"] button')
      .find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('aria-label')).toBe('Processing')
    expect(cancelButton!.attributes('title')).toBe('Processing')
    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()
    expect(wrapper.text()).toContain('Confirm task execution')
    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      action: 'login',
    })

    resolveSubmit({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7000,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalled()
  })

  it('shows a warning and keeps failed-closed accounts selected after a partial task submission', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101), buildAccount(202)],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockResolvedValue({
      submitted: 2,
      enqueued: 1,
      failed_closed: 1,
      logs: [
        {
          id: 7001,
          social_account_id: 101,
          action: 'login',
          status: 'pending',
          charged: false,
          charged_amount: 0,
          charge_status: 'not_charged',
          created_at: new Date().toISOString(),
        },
        {
          id: 7002,
          social_account_id: 202,
          action: 'login',
          status: 'failed',
          result_message: 'executor unavailable',
          charged: false,
          charged_amount: 0,
          charge_status: 'not_charged',
          created_at: new Date().toISOString(),
        },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101, 202],
      action: 'login',
    })
    expect(showWarning).toHaveBeenCalledWith('Submitted 2; queued 1; failed 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Submitted 2')
    expect(showError).not.toHaveBeenCalledWith('Submitted 2; queued 1; failed 1.')
    expect((wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
  })

  it('shows an error and keeps selection when task submission fails closed before enqueue', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount()],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(101, {
          task_status: 'stored',
          task_message: 'executor unavailable',
        })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 0,
      failed_closed: 1,
      logs: [{
        id: 7003,
        social_account_id: 101,
        action: 'login',
        status: 'failed',
        result_message: 'executor unavailable',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Submitted 1; queued 0; failed 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Submitted 1')
    expect(showWarning).not.toHaveBeenCalledWith('Submitted 1; queued 0; failed 1.')
    expect((wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Failed')
    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('executor unavailable')
  })

  it('shows backend stored snapshots with explicit execution failure reasons as failed', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        task_status: 'stored',
        task_message: '任务参数不完整，本次未扣费',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listTaskLogs.mockResolvedValue({ logs: [] })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowText = wrapper.get('[data-testid="account-row-101"]').text()
    expect(rowText).toContain('Failed')
    expect(rowText).not.toContain('Stored')

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('任务参数不完整，本次未扣费')
  })

  it('shows stale task timeout snapshots as failed instead of stored', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        task_status: 'stored',
        task_message: '任务执行超时，本次未扣费',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listTaskLogs.mockResolvedValue({ logs: [] })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowText = wrapper.get('[data-testid="account-row-101"]').text()
    expect(rowText).toContain('Failed')
    expect(rowText).not.toContain('Stored')
  })

  it('shows billing confirmation anomalies as failed instead of stored', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        task_status: 'stored',
        task_message: '执行已完成，但扣费确认异常，请联系管理员处理',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listTaskLogs.mockResolvedValue({ logs: [] })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowText = wrapper.get('[data-testid="account-row-101"]').text()
    expect(rowText).toContain('Failed')
    expect(rowText).not.toContain('Stored')
  })

  it('shows stored snapshots with explicit account attention reasons as needing review', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        task_status: 'stored',
        task_message: '账号认证信息不可用，本次未扣费',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listTaskLogs.mockResolvedValue({ logs: [] })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowText = wrapper.get('[data-testid="account-row-101"]').text()
    expect(rowText).toContain('Manual review')
    expect(rowText).not.toContain('Stored')
  })

  it('shows submitted terminal task log results in the workbench rows', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(101, { task_status: 'pending' }),
          buildAccount(202, { task_status: 'stored' }),
          buildAccount(303, { task_status: 'stored' }),
        ],
        total: 3,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [
          buildAccount(101, { task_status: 'stored', task_message: '任务队列繁忙，本次未扣费' }),
          buildAccount(202, { task_status: 'manual_review', task_message: '账号认证信息不可用，本次未扣费' }),
          buildAccount(303, { task_status: 'ip_unavailable', task_message: '执行代理不可用，本次未扣费' }),
        ],
        total: 3,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    submitTask.mockResolvedValue({
      submitted: 3,
      enqueued: 0,
      failed_closed: 3,
      logs: [
        {
          id: 7011,
          social_account_id: 101,
          action: 'login',
          status: 'failed',
          result_message: '任务队列繁忙，本次未扣费',
          charged: false,
          charged_amount: 0,
          charge_status: 'not_charged',
          created_at: new Date().toISOString(),
        },
        {
          id: 7012,
          social_account_id: 202,
          action: 'login',
          status: 'failed',
          result_message: '账号认证信息不可用，本次未扣费',
          charged: false,
          charged_amount: 0,
          charge_status: 'not_charged',
          created_at: new Date().toISOString(),
        },
        {
          id: 7013,
          social_account_id: 303,
          action: 'login',
          status: 'failed',
          result_message: '执行代理不可用，本次未扣费',
          charged: false,
          charged_amount: 0,
          charge_status: 'not_charged',
          created_at: new Date().toISOString(),
        },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    for (const id of [101, 202, 303]) {
      await wrapper.get(`[data-testid="account-row-${id}"] input[type="checkbox"]`).setValue(true)
    }
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()
    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Failed')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Failed')
    expect(wrapper.get('[data-testid="account-row-303"]').text()).toContain('Failed')
    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('任务队列繁忙，本次未扣费')
    await wrapper.get('[data-testid="account-row-202"] button').trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('账号认证信息不可用，本次未扣费')
    await wrapper.get('[data-testid="account-row-303"] button').trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('执行代理不可用，本次未扣费')
  })

  it('keeps a submitted successful task visible instead of falling back to the stored account snapshot', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101, { task_status: 'stored', task_message: '' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(101, { task_status: 'stored', task_message: 'login succeeded' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7014,
        social_account_id: 101,
        action: 'login',
        status: 'success',
        result_message: 'login succeeded',
        charged: true,
        charged_amount: 0.1,
        charge_status: 'charged',
        executed_at: new Date().toISOString(),
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()
    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    const rowText = wrapper.get('[data-testid="account-row-101"]').text()
    expect(rowText).toContain('Success')
    expect(rowText).not.toContain('Stored')
  })

  it('uses server task logs for top execution stats without rendering a result panel', async () => {
    listTemplates.mockResolvedValue([])
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7100,
        social_account_id: 101,
        account_name: 'x-main-101',
        action: 'login',
        status: 'running',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(listTaskLogs).toHaveBeenCalledWith({
      account_ids: [101],
      statuses: ['pending', 'running', 'success', 'failed'],
      limit: 200,
    })
    expect(wrapper.text()).toContain('Waiting tasks')
    expect(wrapper.text()).toContain('Active tasks')
    expect(wrapper.text()).toContain('Succeeded tasks')
    expect(wrapper.text()).toContain('Failed tasks')
    expect(wrapper.text()).toContain('Queued')
    expect(wrapper.text()).toContain('Running')
    expect(wrapper.get('[data-testid="account-stat-running"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="account-stat-success"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="account-stat-failed"]').text()).toContain('0')
    expect(wrapper.text()).not.toContain('Result rows')
  })

  it('shows recent successful task logs in top execution stats and account rows', async () => {
    listTemplates.mockResolvedValue([])
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7104,
        social_account_id: 101,
        account_name: 'x-main-101',
        action: 'login',
        status: 'success',
        charged: true,
        charged_amount: 0.1,
        charge_status: 'charged',
        executed_at: new Date().toISOString(),
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(listTaskLogs).toHaveBeenCalledWith({
      account_ids: [101],
      statuses: ['pending', 'running', 'success', 'failed'],
      limit: 200,
    })
    expect(wrapper.get('[data-testid="account-stat-success"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="account-stat-running"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Success')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Idle')
  })

  it('shows recent failed task logs in account rows with execution stats', async () => {
    listTemplates.mockResolvedValue([])
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7102,
        social_account_id: 101,
        account_name: 'x-main-101',
        action: 'login',
        status: 'failed',
        result_message: 'old executor failure',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(listTaskLogs).toHaveBeenCalledWith({
      account_ids: [101],
      statuses: ['pending', 'running', 'success', 'failed'],
      limit: 200,
    })
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Failed')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Idle')
    expect(wrapper.get('[data-testid="account-stat-failed"]').text()).toContain('1')
    expect(wrapper.text()).toContain('Failed tasks')
    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('old executor failure')
  })

  it('closes an open task confirmation when refreshed logs make the selection active', async () => {
    listTemplates.mockResolvedValue([])
    let resolveTaskLogs!: (value: {
      logs: Array<{
        id: number
        social_account_id: number
        account_name: string
        action: string
        status: string
        charged: boolean
        charged_amount: number
        charge_status: string
        created_at: string
      }>
    }) => void
    listTaskLogs.mockReturnValue(new Promise((resolve) => {
      resolveTaskLogs = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')

    resolveTaskLogs({
      logs: [{
        id: 7103,
        social_account_id: 101,
        account_name: 'x-main-101',
        action: 'login',
        status: 'running',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Running')
    expect(wrapper.text()).toContain('Non-executable account selected.')
    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(submitTask).not.toHaveBeenCalled()
  })

  it('disables task submission when the selected account already has an active task log', async () => {
    listTemplates.mockResolvedValue([])
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7101,
        social_account_id: 101,
        account_name: 'x-main-101',
        action: 'login',
        status: 'running',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Running')
    expect(wrapper.get('[data-testid="account-row-101"]').find('.animate-spin').exists()).toBe(true)
    expect(wrapper.text()).toContain('Non-executable account selected.')
    const submitButton = findExecutionStartButton(wrapper)
    expect(submitButton.attributes('disabled')).toBeDefined()
    await submitButton.trigger('click')
    await waitForWorkbench()

    expect(submitTask).not.toHaveBeenCalled()
  })

  it('prunes task activity for accounts removed after a list refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7200,
        social_account_id: 202,
        account_name: 'x-main-202',
        action: 'login',
        status: 'running',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    deleteMyAccount.mockResolvedValue(undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Running')
    expect(wrapper.text()).toContain('Active tasks')
    expect(wrapper.text()).toContain('RunningActive tasks1')

    const deleteButton = wrapper.get('[data-testid="account-row-202"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete account'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(deleteMyAccount).toHaveBeenCalledWith(202)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Active tasks')
    expect(wrapper.text()).toContain('RunningActive tasks0')
    expect(wrapper.text()).not.toContain('RunningActive tasks1')
    expect(wrapper.text()).not.toContain('x-main-202')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Running')
  })

  it('locks stale account row and batch actions while a list refresh is pending', async () => {
    listTemplates.mockResolvedValue([])
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.text()).toContain('1 selected')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-testid="data-table-header"] input[type="checkbox"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').attributes('disabled')).toBeDefined()

    const batchProxyButton = wrapper.findAll('button').find(node => node.text().includes('Batch proxy'))
    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    const processingIconButtons = wrapper.findAll('button')
      .filter(node => node.attributes('title') === 'Processing' && node.text().trim() === '')
    const clearSelectionButton = processingIconButtons[0]
    const deleteSelectedButton = processingIconButtons[1]
    const submitButton = findExecutionStartButton(wrapper)
    expect(batchProxyButton?.attributes('disabled')).toBeDefined()
    expect(batchImportButton?.attributes('disabled')).toBeDefined()
    expect(deleteSelectedButton?.attributes('disabled')).toBeDefined()
    expect(clearSelectionButton?.attributes('disabled')).toBeDefined()
    expect(submitButton.attributes('disabled')).toBeDefined()
    expect(batchProxyButton?.attributes('title')).toBe('Processing')
    expect(batchImportButton?.attributes('title')).toBe('Processing')
    expect(batchProxyButton?.attributes('aria-label')).toBe('Processing')
    expect(batchImportButton?.attributes('aria-label')).toBe('Processing')
    expect(deleteSelectedButton?.attributes('aria-label')).toBe('Processing')
    expect(clearSelectionButton?.attributes('aria-label')).toBe('Processing')
    expect(submitButton.attributes('aria-label')).toBe('Processing')
    expect(deleteSelectedButton?.attributes('title')).toBe('Processing')
    expect(clearSelectionButton?.attributes('title')).toBe('Processing')
    expect(submitButton.attributes('title')).toBe('Processing')

    const listCallsWhileRefreshing = listMyAccounts.mock.calls.length
    await batchImportButton!.trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).not.toContain('Paste text or choose a file.')
    expect(listMyAccounts).toHaveBeenCalledTimes(listCallsWhileRefreshing)

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.every(button => button.attributes('disabled') !== undefined)).toBe(true)
    expect(rowButtons[1].attributes('title')).toBe('Processing')
    expect(rowButtons[2].attributes('title')).toBe('Processing')
    expect(rowButtons[3].attributes('title')).toBe('Processing')
    await rowButtons[0].trigger('click')
    await rowButtons[1].trigger('click')
    await rowButtons[2].trigger('click')
    await rowButtons[3].trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Credential Details')
    expect(wrapper.text()).not.toContain('Apply proxy')
    expect(wrapper.text()).not.toContain('Edit account')
    expect(wrapper.text()).not.toContain('Delete account')
    expect(deleteMyAccount).not.toHaveBeenCalled()
    expect(batchDeleteMyAccounts).not.toHaveBeenCalled()
    expect(setDefaultProxy).not.toHaveBeenCalled()
    expect(batchSetDefaultProxy).not.toHaveBeenCalled()
    expect(submitTask).not.toHaveBeenCalled()

    resolveRefresh({
      items: [buildAccount(202)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('1 selected')
  })

  it('ignores stale optional dependency responses after a newer account refresh completes', async () => {
    let resolveInitialProxies!: (value: unknown[]) => void
    let resolveInitialTemplates!: (value: unknown[]) => void
    listUsable
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveInitialProxies = resolve
      }))
      .mockResolvedValueOnce([
        {
          id: 302,
          name: 'Fresh proxy',
          user_id: 42,
          ip_type: 'residential',
          endpoint: 'http://fresh-proxy.example:8080',
          status: 'online',
          latency_ms: 20,
          created_at: '2026-06-01T00:00:00Z',
          updated_at: '2026-06-01T00:00:00Z',
        },
      ])
    listTemplates
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveInitialTemplates = resolve
      }))
      .mockResolvedValueOnce([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('x-main-101')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    expect(wrapper.text()).toContain('Fresh proxy')

    resolveInitialProxies([
      {
        id: 303,
        name: 'Stale proxy',
        user_id: 42,
        ip_type: 'residential',
        endpoint: 'http://stale-proxy.example:8080',
        status: 'online',
        latency_ms: 20,
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    resolveInitialTemplates([])
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Fresh proxy')
    expect(wrapper.text()).not.toContain('Stale proxy')
  })

  it('keeps failed batch-delete accounts selected after a partial result refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(202)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    batchDeleteMyAccounts.mockResolvedValue({
      total: 2,
      succeeded: 1,
      removed: 1,
      skipped: 0,
      failed: 1,
      errors: ['x-main-202 could not be deleted'],
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', error: 'could not be deleted' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    expect(wrapper.text()).toContain('2 selected')
    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Delete 2 accounts')
    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 2 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101, 202])
    expect(showWarning).toHaveBeenCalledWith('Total 2, removed 1, failed 1, skipped 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Deleted 1')
    expect(showError).not.toHaveBeenCalledWith('Total 2, removed 1, failed 1, skipped 0.')
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('Delete 2 accounts')
  })

  it('keeps delete confirmation footer actions inspectable and constrained on narrow layouts', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101), buildAccount(202)],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowDeleteButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(rowDeleteButton).toBeTruthy()
    await rowDeleteButton!.trigger('click')
    await waitForWorkbench()

    let dialog = wrapper.get('section[role="dialog"]')
    expect(dialog.text()).toContain('Delete accounts')

    let cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')
    expect(cancelButton.attributes('disabled')).toBeUndefined()

    const singleConfirmButton = dialog.get('button[aria-label="Delete account"]')
    expectConstrainedDialogButton(singleConfirmButton, 'Delete account')
    expect(singleConfirmButton.attributes('disabled')).toBeUndefined()

    await cancelButton.trigger('click')
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    dialog = wrapper.get('section[role="dialog"]')
    cancelButton = dialog.get('button[aria-label="Cancel"]')
    expectConstrainedDialogButton(cancelButton, 'Cancel')

    const batchConfirmButton = dialog.get('button[aria-label="Delete 2 accounts"]')
    expectConstrainedDialogButton(batchConfirmButton, 'Delete 2 accounts')
    expect(batchConfirmButton.attributes('disabled')).toBeUndefined()
  })

  it('keeps delete confirmation locked with processing titles while delete is pending', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    let resolveDelete!: (value: BatchDeleteSocialAccountResponse) => void
    batchDeleteMyAccounts.mockReturnValue(new Promise((resolve) => {
      resolveDelete = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 1 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    const dialog = wrapper.get('section[role="dialog"]')
    const cancelButton = dialog.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('aria-label')).toBe('Processing')
    expect(cancelButton!.attributes('title')).toBe('Processing')
    const processingButton = dialog.get('button[aria-label="Processing"]')
    expect(processingButton.attributes('disabled')).toBeDefined()
    expect(processingButton.attributes('title')).toBe('Processing')

    await cancelButton.trigger('click')
    await processingButton.trigger('click')
    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Delete accounts')

    resolveDelete({
      total: 1,
      succeeded: 1,
      removed: 1,
      skipped: 0,
      failed: 0,
      errors: [],
      items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Deleted 1')
  })

  it('locks an open delete confirmation while the account list is refreshing', async () => {
    listTemplates.mockResolvedValue([])
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = () => {
      const button = wrapper.findAll('button').find(node => node.text().includes('Delete 1 accounts'))
      expect(button).toBeTruthy()
      return button!
    }
    expect(confirmButton().attributes('disabled')).toBeUndefined()

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(confirmButton().attributes('disabled')).toBeDefined()
    await confirmButton().trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).not.toHaveBeenCalled()
    expect(deleteMyAccount).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Delete 1 accounts')

    resolveRefresh({
      items: [buildAccount(101), buildAccount(202)],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(confirmButton().attributes('disabled')).toBeUndefined()
  })

  it('removes a deleted account locally before the next list refresh finishes', async () => {
    listTemplates.mockResolvedValue([])
    const remainingAccount = buildAccount(101)
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [remainingAccount, buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    listTaskLogs.mockResolvedValue({
      logs: [{
        id: 7200,
        social_account_id: 202,
        account_name: 'x-main-202',
        action: 'login',
        status: 'running',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    deleteMyAccount.mockResolvedValue(undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Running')
    expect(wrapper.text()).toContain('RunningActive tasks1')

    const deleteButton = wrapper.get('[data-testid="account-row-202"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete account'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(deleteMyAccount).toHaveBeenCalledWith(202)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('RunningActive tasks0')
    expect(wrapper.text()).not.toContain('Delete account')

    resolveRefresh({
      items: [remainingAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('closes account detail and clears selection when the open account is deleted', async () => {
    listTemplates.mockResolvedValue([])
    const remainingAccount = buildAccount(101)
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [remainingAccount, buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    deleteMyAccount.mockResolvedValue(undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] button').trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Credential Details')
    expect(wrapper.text()).toContain('1 selected')

    const deleteButton = wrapper.get('[data-testid="account-row-202"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete account'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(deleteMyAccount).toHaveBeenCalledWith(202)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Credential Details')
    expect(wrapper.text()).not.toContain('1 selected')
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)

    resolveRefresh({
      items: [remainingAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('keeps the delete confirmation open with a safe error when single-account delete fails', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(202)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    deleteMyAccount.mockRejectedValue({
      response: {
        data: {
          detail: 'single delete failed token=secret',
        },
      },
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const deleteButton = wrapper.get('[data-testid="account-row-202"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete account'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(deleteMyAccount).toHaveBeenCalledWith(202)
    expect(showError).toHaveBeenCalledWith('Delete failed')
    expect(showError).not.toHaveBeenCalledWith('single delete failed token=secret')
    const dialog = wrapper.get('section[role="dialog"]')
    const deleteError = dialog.get('[role="alert"]')
    expect(deleteError.text()).toContain('Delete failed')
    expect(deleteError.attributes('title')).toBe('Delete failed')
    expect(deleteError.attributes('aria-live')).toBe('assertive')
    expect(deleteError.attributes('aria-atomic')).toBe('true')
    expect(dialog.text()).not.toContain('single delete failed token=secret')

    await dialog.get('button[aria-label="Cancel"]').trigger('click')
    await waitForWorkbench()

    await deleteButton!.trigger('click')
    await waitForWorkbench()
    expect(wrapper.get('section[role="dialog"]').find('[role="alert"]').exists()).toBe(false)
  })

  it('removes only succeeded batch-delete accounts locally before the next list refresh finishes', async () => {
    listTemplates.mockResolvedValue([])
    const failedAccount = buildAccount(202)
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), failedAccount],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchDeleteMyAccounts.mockResolvedValue({
      total: 2,
      succeeded: 1,
      removed: 1,
      skipped: 0,
      failed: 1,
      errors: ['x-main-202 could not be deleted'],
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', error: 'could not be deleted' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 2 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101, 202])
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(showWarning).toHaveBeenCalledWith('Total 2, removed 1, failed 1, skipped 0.')

    resolveRefresh({
      items: [failedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('does not guess local batch-delete removals when partial success rows omit succeeded ids', async () => {
    listTemplates.mockResolvedValue([])
    const failedAccount = buildAccount(202)
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), failedAccount],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchDeleteMyAccounts.mockResolvedValue({
      total: 2,
      succeeded: 1,
      removed: 1,
      skipped: 0,
      failed: 1,
      errors: ['x-main-202 could not be deleted'],
      items: [
        { status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', error: 'could not be deleted' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 2 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101, 202])
    expect(showWarning).toHaveBeenCalledWith('Total 2, removed 1, failed 1, skipped 0.')
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('2 selected')

    resolveRefresh({
      items: [failedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
  })

  it('closes single-delete confirmation when the target account disappears after refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    deleteMyAccount.mockResolvedValue(undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    const deleteButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.attributes('title') === 'Delete account')
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Delete account')
    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Delete account')
    const staleConfirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete account'))
    expect(staleConfirmButton).toBeFalsy()
    expect(deleteMyAccount).not.toHaveBeenCalled()
  })

  it('closes batch-delete confirmation when selected accounts disappear after refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101), buildAccount(202)],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Delete 2 accounts')
    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Delete 2 accounts')
    expect(batchDeleteMyAccounts).not.toHaveBeenCalled()
  })

  it('shows an error toast when batch delete returns no removals', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchDeleteMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      removed: 0,
      skipped: 0,
      failed: 1,
      errors: ['x-main-101 could not be deleted'],
      items: [
        { id: 101, name: 'x-main-101', status: 'failed', error: 'could not be deleted' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 1 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101])
    expect(showError).toHaveBeenCalledWith('Total 1, removed 0, failed 1, skipped 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Deleted 0')
    expect(showWarning).not.toHaveBeenCalledWith('Total 1, removed 0, failed 1, skipped 0.')
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('Delete 1 accounts')
  })

  it('shows an error toast when batch proxy assignment returns no successes', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(showError).toHaveBeenCalledWith('Total 1, succeeded 0, failed 1, skipped 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Proxy saved')
    expect(showWarning).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Error · Selected proxy unavailable')
    expect(wrapper.text()).not.toContain('proxy_not_available')
  })

  it('keeps normalized online proxies available for account proxy assignment', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([{
      id: 301,
      user_id: 1,
      name: 'east-proxy',
      ip_type: 'residential',
      endpoint: 'http://proxy.example:8080',
      status: ' ONLINE ',
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-01T00:00:00Z',
    }])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    expect(applyButton!.attributes('disabled')).toBeUndefined()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'specific',
      proxy_id: 301,
    })
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')
    expect(wrapper.text()).not.toContain('No online proxies')
  })

  it('clears a stale proxy assignment result when a later upload-to-pool result completes', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101),
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    storeWorkbenchAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 202, name: 'staging-account', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202])
    expect(wrapper.text()).toContain('Upload result')
    expect(wrapper.text()).toContain('Total 1, succeeded 1, failed 0, skipped 0.')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('clears a stale proxy assignment result when a later upload-to-pool request fails', async () => {
    adminState.isAdmin = true
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101),
        buildAccount(202, {
          name: 'staging-account',
          username: 'staging-account',
          account_status: 'not_stored',
          task_status: 'pending',
          default_proxy_configured: false,
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    storeWorkbenchAccounts.mockRejectedValue(new Error('upload retry failed'))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const uploadButton = wrapper.findAll('button').find(node => node.text().includes('Upload to Pool'))
    expect(uploadButton).toBeTruthy()
    await uploadButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Confirm upload'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(storeWorkbenchAccounts).toHaveBeenCalledWith([202])
    expect(showError).toHaveBeenCalledWith('Upload failed')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('clears a stale proxy assignment result when a later batch import request fails', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    batchImportMyAccounts.mockRejectedValue(new Error('import retry failed'))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_import\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').filter(node => node.text().includes('Confirm')).at(-1)
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Import failed')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('clears a stale proxy assignment result when a later batch delete completes', async () => {
    listTemplates.mockResolvedValue([])
    const account = buildAccount(101)
    listMyAccounts
      .mockResolvedValueOnce({
        items: [account],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [account],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    batchDeleteMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      removed: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 1 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101])
    expect(showSuccess).toHaveBeenCalledWith('Deleted 1')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('clears a stale proxy assignment result when a later batch delete request fails', async () => {
    listTemplates.mockResolvedValue([])
    const account = buildAccount(101)
    listMyAccounts.mockResolvedValue({
      items: [account],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    batchDeleteMyAccounts.mockRejectedValue({
      response: {
        data: {
          detail: 'batch delete failed token=secret',
        },
      },
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const deleteSelectedButton = wrapper.findAll('button').find(node => node.attributes('title') === 'Delete selected')
    expect(deleteSelectedButton).toBeTruthy()
    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').find(node => node.text().includes('Delete 1 accounts'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    expect(batchDeleteMyAccounts).toHaveBeenCalledWith([101])
    expect(showError).toHaveBeenCalledWith('Delete failed')
    expect(showError).not.toHaveBeenCalledWith('batch delete failed token=secret')
    const dialog = wrapper.get('section[role="dialog"]')
    const deleteError = dialog.get('[role="alert"]')
    expect(deleteError.text()).toContain('Delete failed')
    expect(deleteError.attributes('title')).toBe('Delete failed')
    expect(deleteError.attributes('aria-live')).toBe('assertive')
    expect(deleteError.attributes('aria-atomic')).toBe('true')
    expect(dialog.text()).not.toContain('batch delete failed token=secret')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    await dialog.get('button[aria-label="Cancel"]').trigger('click')
    await waitForWorkbench()

    await deleteSelectedButton!.trigger('click')
    await waitForWorkbench()
    expect(wrapper.get('section[role="dialog"]').find('[role="alert"]').exists()).toBe(false)
  })

  it('clears a stale proxy assignment result when a later account edit completes', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101, {
      password: 'old-password',
      email: 'old@example.com',
      remark: 'old remark',
    })
    const updatedAccount = buildAccount(101, {
      password: 'new-password',
      email: 'new@example.com',
      remark: 'edited after proxy result',
    })
    listMyAccounts.mockResolvedValue({
      items: [updatedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listMyAccounts.mockResolvedValueOnce({
      items: [initialAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    updateMyAccount.mockResolvedValue(updatedAccount)

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('saved-password')
    await wrapper.get('#account-edit-email').setValue('saved@example.com')
    await wrapper.get('#account-edit-remark').setValue('saved after proxy result')
    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: 'saved-password',
      email: 'saved@example.com',
      remark: 'saved after proxy result',
    }))
    expect(showSuccess).toHaveBeenCalledWith('Saved')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('clears a stale proxy assignment result when a later task submission request fails', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    submitTask.mockRejectedValue({
      message: 'task submit failed token=secret',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Submit failed')
    expect(showError).not.toHaveBeenCalledWith('task submit failed token=secret')
    const dialog = wrapper.get('section[role="dialog"]')
    const alert = dialog.get('[role="alert"]')
    expect(dialog.text()).toContain('Confirm task execution')
    expect(alert.text()).toBe('Submit failed')
    expect(alert.attributes('title')).toBe('Submit failed')
    expect(alert.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(dialog.text()).not.toContain('task submit failed token=secret')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(closeButton).toBeTruthy()
    await closeButton!.trigger('click')
    await waitForWorkbench()

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.get('section[role="dialog"]').text()).toContain('Confirm task execution')
    expect(wrapper.find('section[role="dialog"] [role="alert"]').exists()).toBe(false)
  })

  it('only submits filtered visible accounts when selecting all for batch proxy assignment', async () => {
    listTemplates.mockResolvedValue([])
    const visibleAccount = buildAccount(101, {
      name: 'visible-proxy-target',
      username: 'visible-proxy-target',
    })
    const hiddenAccount = buildAccount(202, {
      name: 'hidden-proxy-target',
      username: 'hidden-proxy-target',
    })
    listMyAccounts.mockResolvedValue({
      items: [visibleAccount, hiddenAccount],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 101, name: 'visible-proxy-target', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(true)

    await wrapper.get('[data-testid="search-input-stub"]').setValue('visible-proxy')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)

    await wrapper.get('[data-testid="data-table-header"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.text()).toContain('1 selected')

    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'clear',
      proxy_id: null,
    })
    expect(batchSetDefaultProxy).not.toHaveBeenCalledWith(expect.objectContaining({
      account_ids: expect.arrayContaining([202]),
    }))
  })

  it('shows a localized error when random batch proxy assignment loses the online pool before submit', async () => {
    listTemplates.mockResolvedValue([])
    const staleProxy = {
        id: 301,
        user_id: 10,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://proxy.example:8080',
        status: 'online',
        latency_ms: 40,
        last_check_at: '2026-06-06T01:00:00Z',
        remark: 'east',
        created_at: '2026-06-06T00:00:00Z',
        updated_at: '2026-06-06T01:00:00Z',
      }
    listUsable
      .mockResolvedValueOnce([staleProxy])
      .mockResolvedValueOnce([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockRejectedValue({ code: 'SOCIAL_IP_POOL_EMPTY', message: 'no online proxy is available for assignment' })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const randomModeButton = wrapper.findAll('button').find(node => node.text().includes('Random proxy'))
    expect(randomModeButton).toBeTruthy()
    await randomModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    expect(applyButton!.attributes('disabled')).toBeUndefined()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'random',
      proxy_id: null,
    })
    expect(showError).toHaveBeenCalledWith('No online proxies available for random assignment.')
    expect(showError).not.toHaveBeenCalledWith('Proxy failed')
    expect(showSuccess).not.toHaveBeenCalledWith('Proxy saved')
    expect(listMyAccounts).toHaveBeenCalledTimes(2)
    expect(listUsable).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('No online proxies')
  })

  it('maps invalid batch proxy assignment modes to a safe dialog recovery message', async () => {
    listTemplates.mockResolvedValue([])
    listTaskLogs.mockResolvedValue([])
    listUsable.mockResolvedValue([{
      id: 301,
      user_id: 10,
      name: 'east-proxy',
      ip_type: 'residential',
      endpoint: 'http://proxy.example:8080',
      status: 'online',
      latency_ms: 40,
      last_check_at: '2026-06-06T01:00:00Z',
      remark: 'east',
      created_at: '2026-06-06T00:00:00Z',
      updated_at: '2026-06-06T01:00:00Z',
    }])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockRejectedValue({
      code: 'SOCIAL_IP_ASSIGNMENT_MODE_INVALID',
      message: 'proxy assignment mode is invalid internal=stale-ui',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'specific',
      proxy_id: 301,
    })
    expect(showError).toHaveBeenCalledWith('Proxy assignment mode changed. Reopen the dialog and try again.')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('stale-ui')
  })

  it('clears the previous batch proxy result when the assignment mode changes', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Error · Selected proxy unavailable')

    const specificModeButton = wrapper.findAll('button').find(node => node.text().includes('Specific proxy'))
    expect(specificModeButton).toBeTruthy()
    await specificModeButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Error · Selected proxy unavailable')
  })

  it('clears the previous batch proxy result when the selected proxy changes', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://east-proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
      {
        id: 302,
        user_id: 1,
        name: 'west-proxy',
        ip_type: 'datacenter',
        endpoint: 'http://west-proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('x-main-101Success')
    expect(wrapper.text()).not.toContain('Success · -')

    await proxySelect!.setValue('302')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('x-main-101Success')
  })

  it('closes single-account proxy dialog when the target account disappears after refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    setDefaultProxy.mockResolvedValue(undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Current account: x-main-101')
    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Current account: x-main-101')
    expect(wrapper.text()).not.toContain('Apply proxy')
    expect(setDefaultProxy).not.toHaveBeenCalled()
  })

  it('syncs a single-account proxy clear from the API response before the next list refresh', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
    })
    const clearedAccount = buildAccount(101, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    setDefaultProxy.mockResolvedValue(clearedAccount)

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(setDefaultProxy).toHaveBeenCalledWith(101, null)
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Configured')
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')

    resolveRefresh({
      items: [clearedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('removes a single-account proxy clear from the active proxy search when the follow-up refresh fails', async () => {
    listTemplates.mockResolvedValue([])
    const proxySearch = 'stage86-proxy.example'
    const initialAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: `{"id":301,"endpoint":"http://${proxySearch}:8080","status":"online"}`,
    })
    const clearedAccount = buildAccount(101, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    let proxySaved = false
    listMyAccounts.mockImplementation((params: Record<string, unknown> = {}) => {
      if (params.search === proxySearch && proxySaved) {
        return Promise.reject(new Error('follow-up refresh failed'))
      }
      return Promise.resolve({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    })
    setDefaultProxy.mockImplementation(async () => {
      proxySaved = true
      return clearedAccount
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="search-input-stub"]').setValue(proxySearch)
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(setDefaultProxy).toHaveBeenCalledWith(101, null)
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('No results')
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 1, failed 0, skipped 0.')
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Proxy failed')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.load_data', expect.any(Error))

    wrapper.unmount()
  })

  it('syncs a single-account proxy assignment from the API response before the next list refresh', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    const initialAccount = buildAccount(101, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    const updatedAccount = buildAccount(101, {
      name: 'x-main-101-normalized',
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":301,"name":"east-proxy","ip_type":"residential","endpoint":"http://proxy.example:8080","status":"online"}',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    setDefaultProxy.mockResolvedValue(updatedAccount)

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')

    const detailButton = wrapper.get('[data-testid="account-row-101"]').find('button')
    await detailButton.trigger('click')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('Default proxy snapshot')
    expect(wrapper.text()).not.toContain('http://proxy.example:8080')

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(setDefaultProxy).toHaveBeenCalledWith(101, 301)
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')
    expect(wrapper.text()).toContain('http://proxy.example:8080')
    expect(wrapper.text()).toContain('Current account: x-main-101-normalized')
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')

    resolveRefresh({
      items: [updatedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('clears a selected proxy when refreshing removes it from the usable proxy pool', async () => {
    listTemplates.mockResolvedValue([])
    listUsable
      .mockResolvedValueOnce([
        {
          id: 301,
          user_id: 1,
          name: 'east-proxy',
          ip_type: 'residential',
          endpoint: 'http://proxy.example:8080',
          status: 'online',
          created_at: '2026-06-01T00:00:00Z',
          updated_at: '2026-06-01T00:00:00Z',
        },
      ])
      .mockResolvedValueOnce([])

    const wrapper = mountView()
    await waitForWorkbench()

    const proxyButton = wrapper.get('[data-testid="account-row-101"]').findAll('button').find(node => node.text().includes('Proxy'))
    expect(proxyButton).toBeTruthy()
    await proxyButton!.trigger('click')
    await waitForWorkbench()

    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('east-proxy')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    const refreshedProxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('Select proxy'))
    expect(refreshedProxySelect).toBeTruthy()
    expect((refreshedProxySelect!.element as HTMLSelectElement).value).toBe('__null__')
    expect(wrapper.text()).toContain('Selected proxy')
    expect(wrapper.text()).toContain('None')
    expect(wrapper.text()).toContain('Select an online proxy first')
    const disabledReason = wrapper.findAll('div').find(node => node.text() === 'Select an online proxy first')
    expect(disabledReason).toBeTruthy()
    expect(disabledReason!.attributes('title')).toBe('Select an online proxy first')
    expect(disabledReason!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    expect(applyButton!.attributes('disabled')).toBeDefined()
    await applyButton!.trigger('click')
    expect(setDefaultProxy).not.toHaveBeenCalled()
  })

  it('closes batch proxy dialog when selected accounts disappear after refresh', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    batchSetDefaultProxy.mockResolvedValue({
      total: 0,
      succeeded: 0,
      skipped: 0,
      failed: 0,
      items: [],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    expect(wrapper.text()).toContain('Batch proxy')
    expect(wrapper.text()).toContain('Apply proxy')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Apply proxy')
    expect(batchSetDefaultProxy).not.toHaveBeenCalled()
  })

  it('locks proxy assignment controls while saving the default proxy', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    let resolveProxy!: (value: {
      total: number
      succeeded: number
      skipped: number
      failed: number
      items: Array<{ id: number; name: string; status: string }>
    }) => void
    batchSetDefaultProxy.mockReturnValue(new Promise((resolve) => {
      resolveProxy = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    const savingButton = wrapper.findAll('button').find(node => node.text().includes('Saving'))
    expect(savingButton).toBeTruthy()
    expect(savingButton!.attributes('disabled')).toBeDefined()
    expect(proxySelect!.attributes('disabled')).toBeDefined()
    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('title')).toBe('Saving')

    const specificModeButton = wrapper.findAll('button').find(node => node.text().includes('Specific proxy'))
    const randomModeButton = wrapper.findAll('button').find(node => node.text().includes('Random proxy'))
    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(specificModeButton?.attributes('disabled')).toBeDefined()
    expect(randomModeButton?.attributes('disabled')).toBeDefined()
    expect(clearModeButton?.attributes('disabled')).toBeDefined()

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    await clearModeButton!.trigger('click')
    await savingButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Batch proxy')
    expect(batchSetDefaultProxy).toHaveBeenCalledTimes(1)
    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'specific',
      proxy_id: 301,
    })

    resolveProxy({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')
  })

  it('shows a warning toast when batch proxy assignment partially succeeds', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101), buildAccount(202), buildAccount(303)],
      total: 3,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 3,
      succeeded: 1,
      skipped: 1,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: ' SUCCEEDED ' },
        { id: 202, name: 'x-main-202', status: ' FAILED ', reason: ' proxy_not_available ', error: 'account proxy could not be assigned' },
        { id: 303, name: 'x-main-303', status: ' SKIPPED ', reason: ' custom_backend_reason ' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-303"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101, 202, 303],
      mode: 'clear',
      proxy_id: null,
    })
    expect(showWarning).toHaveBeenCalledWith('Total 3, succeeded 1, failed 1, skipped 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Proxy saved')
    expect(showError).not.toHaveBeenCalledWith('Total 3, succeeded 1, failed 1, skipped 1.')
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 3, succeeded 1, failed 1, skipped 1.')
    expect(wrapper.text()).toContain('x-main-101Success')
    expect(wrapper.text()).not.toContain('Success · -')
    expect(wrapper.text()).toContain('Error · Selected proxy unavailable')
    expect(wrapper.text()).toContain('Skipped · Could not process this account')
    expect(wrapper.text()).not.toContain('custom_backend_reason')
  })

  it('clears the stale proxy assignment result when a retry request fails', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy
      .mockResolvedValueOnce({
        total: 1,
        succeeded: 1,
        skipped: 0,
        failed: 0,
        items: [{ id: 101, name: 'x-main-101', status: 'succeeded' }],
      })
      .mockRejectedValueOnce({
        response: {
          data: {
            detail: 'retry proxy failed token=secret',
          },
        },
      })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = () => {
      const button = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
      expect(button).toBeTruthy()
      return button!
    }

    await applyButton().trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 1, failed 0, skipped 0.')

    await applyButton().trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Proxy failed')
    expect(showError).not.toHaveBeenCalledWith('retry proxy failed token=secret')
    expect(wrapper.text()).toContain('Apply proxy')
    expect(wrapper.text()).toContain('Clear proxy')
    expect(wrapper.text()).toContain('Proxy failed')
    expect(wrapper.text()).not.toContain('retry proxy failed token=secret')
    const proxyErrorAlert = wrapper.find('[role="alert"][title="Proxy failed"]')
    expect(proxyErrorAlert.exists()).toBe(true)
    expect(proxyErrorAlert.classes()).toContain('min-w-0')
    expect(proxyErrorAlert.classes()).toContain('break-words')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 1, failed 0, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy failed')
  })

  it('syncs succeeded batch proxy clears locally before the next list refresh finishes', async () => {
    listTemplates.mockResolvedValue([])
    const clearedAccount = buildAccount(101, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    const failedAccount = buildAccount(202, {
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(101, {
            default_proxy_configured: true,
            default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
          }),
          failedAccount,
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchSetDefaultProxy.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Configured')

    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101, 202],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Configured')
    expect(showWarning).toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')

    resolveRefresh({
      items: [clearedAccount, failedAccount],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('keeps only failed batch proxy clears selected in an active proxy search when refresh fails', async () => {
    listTemplates.mockResolvedValue([])
    const proxySearch = 'stage87-proxy.example'
    const initialAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: `{"id":301,"endpoint":"http://${proxySearch}:8080","status":"online"}`,
    })
    const failedAccount = buildAccount(202, {
      default_proxy_configured: true,
      default_proxy_snapshot: `{"id":302,"endpoint":"http://${proxySearch}:9090","status":"online"}`,
    })
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount, failedAccount],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [initialAccount, failedAccount],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))
    batchSetDefaultProxy.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="search-input-stub"]').setValue(proxySearch)
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.text()).toContain('2 selected')

    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const clearModeButton = wrapper.findAll('button').find(node => node.text().includes('Clear proxy'))
    expect(clearModeButton).toBeTruthy()
    await clearModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101, 202],
      mode: 'clear',
      proxy_id: null,
    })
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-row-202"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Configured')
    expect(wrapper.text()).toContain('1 selected')
    expect(wrapper.text()).not.toContain('2 selected')
    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 2, succeeded 1, failed 1, skipped 0.')
    expect(wrapper.text()).toContain('x-main-101Success')
    expect(wrapper.text()).not.toContain('Success · -')
    expect(wrapper.text()).toContain('Error · Selected proxy unavailable')
    expect(showWarning).toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Proxy failed')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.load_data', expect.any(Error))

    wrapper.unmount()
  })

  it('syncs succeeded batch proxy assignments locally before the next list refresh finishes', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    const assignedAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
    })
    const failedAccount = buildAccount(202, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(101, {
            default_proxy_configured: false,
            default_proxy_snapshot: '',
          }),
          failedAccount,
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchSetDefaultProxy.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Not configured')

    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101, 202],
      mode: 'specific',
      proxy_id: 301,
    })
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Not configured')
    expect(showWarning).toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')

    await wrapper.get('[data-testid="search-input-stub"]').setValue('proxy.example')
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)

    resolveRefresh({
      items: [assignedAccount, failedAccount],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('does not guess local proxy assignment changes when partial success rows omit succeeded ids', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(101, {
            default_proxy_configured: false,
            default_proxy_snapshot: '',
          }),
          buildAccount(202, {
            default_proxy_configured: false,
            default_proxy_snapshot: '',
          }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchSetDefaultProxy.mockResolvedValue({
      total: 2,
      succeeded: 1,
      skipped: 0,
      failed: 1,
      items: [
        { status: 'succeeded' },
        { id: 202, name: 'x-main-202', status: 'failed', reason: 'proxy_not_available', error: 'account proxy could not be assigned' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="account-row-202"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const proxySelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.text().includes('east-proxy'))
    expect(proxySelect).toBeTruthy()
    await proxySelect!.setValue('301')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101, 202],
      mode: 'specific',
      proxy_id: 301,
    })
    expect(showWarning).toHaveBeenCalledWith('Total 2, succeeded 1, failed 1, skipped 0.')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Not configured')

    resolveRefresh({
      items: [
        buildAccount(101, {
          default_proxy_configured: true,
          default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
        }),
        buildAccount(202, {
          default_proxy_configured: false,
          default_proxy_snapshot: '',
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')
    expect(wrapper.get('[data-testid="account-row-202"]').text()).toContain('Not configured')
  })

  it('waits for the list refresh before reflecting random batch proxy assignments', async () => {
    listTemplates.mockResolvedValue([])
    listUsable.mockResolvedValue([
      {
        id: 301,
        user_id: 1,
        name: 'east-proxy',
        ip_type: 'residential',
        endpoint: 'http://east-proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
      {
        id: 302,
        user_id: 1,
        name: 'west-proxy',
        ip_type: 'datacenter',
        endpoint: 'http://west-proxy.example:8080',
        status: 'online',
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
    const assignedAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":302,"endpoint":"http://west-proxy.example:8080","status":"online"}',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [
          buildAccount(101, {
            default_proxy_configured: false,
            default_proxy_snapshot: '',
          }),
        ],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
      items: [
        { id: 101, name: 'x-main-101', status: 'succeeded' },
      ],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')

    await openBatchProxyDialogForSelectedAccounts(wrapper)
    const randomModeButton = wrapper.findAll('button').find(node => node.text().includes('Random proxy'))
    expect(randomModeButton).toBeTruthy()
    await randomModeButton!.trigger('click')
    await waitForWorkbench()

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(batchSetDefaultProxy).toHaveBeenCalledWith({
      account_ids: [101],
      mode: 'random',
      proxy_id: null,
    })
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Not configured')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Configured')
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')

    resolveRefresh({
      items: [assignedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Configured')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).not.toContain('Not configured')
  })

  it('shows the social task affordability metadata when login submission lacks balance', async () => {
    listTemplates.mockResolvedValue([])
    submitTask.mockRejectedValue({
      reason: 'SOCIAL_TASK_INSUFFICIENT_FUNDS',
      message: 'insufficient subscription allowance and wallet balance for social task',
      metadata: {
        required_total: '0.20',
        wallet_balance: '0.00',
        wallet_required: '0.20',
      },
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(showError).toHaveBeenCalledWith('Insufficient balance: required 0.20, balance 0.00, need 0.20.')
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
  })

  it('maps backend mixed-platform task submission rejections to a precise recovery message', async () => {
    listTemplates.mockResolvedValue([])
    submitTask.mockRejectedValue({
      reason: 'SOCIAL_TASK_MIXED_PLATFORMS',
      message: 'selected social accounts must belong to the same platform for one task internal=platform-mismatch',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Selected accounts must belong to the same platform. Refresh and submit one platform at a time.')
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('platform-mismatch')
  })

  it('maps stale account id task submission rejections to a precise recovery message', async () => {
    listTemplates.mockResolvedValue([])
    submitTask.mockRejectedValue({
      reason: 'SOCIAL_TASK_ACCOUNT_ID_INVALID',
      message: 'social account id must be positive internal=stale-selection',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Selected account list is stale. Refresh and select again.')
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
    expect(JSON.stringify(showError.mock.calls)).not.toContain('stale-selection')
    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(listMyAccounts).toHaveBeenCalledTimes(2)
    expect(listTaskLogs).toHaveBeenCalled()
  })

  it.each([
    [
      'SOCIAL_TASK_INPUT_REQUIRED',
      'Task submission details are incomplete. Refresh and submit again.',
      'internal=input-required',
    ],
    [
      'SOCIAL_TASK_ACCOUNTS_REQUIRED',
      'Select at least one account before submitting.',
      'internal=accounts-required',
    ],
    [
      'SOCIAL_TASK_PLATFORM_REQUIRED',
      'A selected account is missing platform information. Refresh and select again.',
      'internal=platform-required',
    ],
  ])('maps %s task submission rejections to a precise recovery message', async (reason, expectedMessage, rawMessage) => {
    listTemplates.mockResolvedValue([])
    submitTask.mockRejectedValue({
      reason,
      message: rawMessage,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expectedMessage)
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
    expect(JSON.stringify(showError.mock.calls)).not.toContain(rawMessage)
  })

  it('shows a precise message when a stale default task template is rejected at submit time', async () => {
    listTemplates
      .mockResolvedValueOnce([buildPostTemplate()])
      .mockResolvedValue([])
    submitTask.mockRejectedValue({
      reason: 'TASK_DEFAULT_TEMPLATE_REQUIRED',
      message: 'default task template is required for this action',
    })

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'post')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    const confirmButtons = wrapper.findAll('button').filter(node => node.text().includes('Submit task'))
    expect(confirmButtons.length).toBeGreaterThan(0)
    await confirmButtons[confirmButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      action: 'post',
    })
    expect(showError).toHaveBeenCalledWith('Default task template is missing. Set it again before submitting.')
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(wrapper.text()).not.toContain('Post template')
    expect(listTemplates).toHaveBeenCalledTimes(2)
    expect(listTaskLogs).toHaveBeenCalled()
  })

  it('refreshes stale default proxy state when task submission rejects an unavailable proxy', async () => {
    listTemplates.mockResolvedValue([])
    const configuredAccount = buildAccount(101, {
      default_proxy_configured: true,
      default_proxy_snapshot: '{"id":301,"endpoint":"http://proxy.example:8080","status":"online"}',
    })
    const refreshedAccount = buildAccount(101, {
      default_proxy_configured: false,
      default_proxy_snapshot: '',
    })
    listMyAccounts
      .mockResolvedValueOnce({
        items: [configuredAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [refreshedAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    submitTask.mockRejectedValue({
      reason: 'SOCIAL_IP_NOT_AVAILABLE',
      message: 'default social IP is required for execution',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const row = () => wrapper.get('[data-testid="account-row-101"]')
    expect(row().text()).toContain('Configured')

    await row().find('input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

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
      action: 'login',
    })
    expect(showError).toHaveBeenCalledWith('Default proxy unavailable. Reassign a usable proxy before submitting.')
    expect(showError).not.toHaveBeenCalledWith('Submit failed')
    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(row().text()).toContain('Not configured')
    expect(row().text()).not.toContain('Configured')
    expect(wrapper.text()).not.toContain('Login needs a default proxy.')
    const executionDisabledReason = wrapper.findAll('[role="status"]')
      .find(node => node.text().includes('Login needs a default proxy.'))
    expect(executionDisabledReason).toBeUndefined()
    expect(findExecutionStartButton(wrapper).attributes('disabled')).toBeUndefined()
    expect(listMyAccounts).toHaveBeenCalledTimes(2)
    expect(listTaskLogs).toHaveBeenCalled()
  })

  it('locks editable delivery fields while account edit is saving', async () => {
    listTemplates.mockResolvedValue([])
    let resolveUpdate!: (value: ReturnType<typeof buildAccount>) => void
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        email: 'old@example.com',
        auth_cookie: 'ct0=old; auth_token=old',
        execution_auth: 'encrypted-old-execution-auth',
        registration_ip: '198.51.100.20',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockReturnValue(new Promise((resolve) => {
      resolveUpdate = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    await wrapper.get('#account-edit-email').setValue('new@example.com')
    await wrapper.get('#account-edit-registration-ip').setValue('203.0.113.10')
    await wrapper.get('#account-edit-auth-cookie').setValue('ct0=new; auth_token=new')
    await wrapper.get('#account-edit-execution-auth').setValue('encrypted-new-execution-auth')
    await wrapper.get('#account-edit-remark').setValue('new remark')

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    const savingButton = wrapper.findAll('button').find(node => node.text().includes('Saving'))
    expect(savingButton).toBeTruthy()
    expect(savingButton!.attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-password').attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-email').attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-registration-ip').attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-auth-cookie').attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-execution-auth').attributes('disabled')).toBeDefined()
    expect(wrapper.get('#account-edit-remark').attributes('disabled')).toBeDefined()

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('title')).toBe('Saving')
    await cancelButton!.trigger('click')
    await savingButton!.trigger('click')
    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledTimes(1)
    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: 'new-password',
      email: 'new@example.com',
      registration_ip: '203.0.113.10',
      auth_cookie: 'ct0=new; auth_token=new',
      execution_auth: 'encrypted-new-execution-auth',
      remark: 'new remark',
    }))
    expect(wrapper.text()).toContain('Edit account')

    resolveUpdate(buildAccount(101, {
      password: 'new-password',
      email: 'new@example.com',
      auth_cookie: 'ct0=new; auth_token=new',
      execution_auth: 'encrypted-new-execution-auth',
      registration_ip: '203.0.113.10',
      remark: 'new remark',
    }))
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Saved')
  })

  it('syncs edited delivery fields from the API response before the next list refresh', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101, {
      password: 'old-password',
      email: 'old@example.com',
      registration_ip: '198.51.100.20',
      remark: 'old remark',
    })
    const updatedAccount = buildAccount(101, {
      password: 'new-password',
      email: 'new@example.com',
      registration_ip: '203.0.113.10',
      remark: 'new local remark',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    updateMyAccount.mockResolvedValue(updatedAccount)

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('x-main-101')

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    await wrapper.get('#account-edit-email').setValue('new@example.com')
    await wrapper.get('#account-edit-registration-ip').setValue('203.0.113.10')
    await wrapper.get('#account-edit-remark').setValue('new local remark')

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: 'new-password',
      email: 'new@example.com',
      registration_ip: '203.0.113.10',
      remark: 'new local remark',
    }))

    await wrapper.get('[data-testid="search-input-stub"]').setValue('new local remark')
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(showSuccess).toHaveBeenCalledWith('Saved')

    resolveRefresh({
      items: [updatedAccount],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('keeps edited account fields locally when the follow-up list refresh fails', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101, {
      password: 'old-password',
      email: 'old@example.com',
      registration_ip: '198.51.100.20',
      remark: 'old remark',
    })
    const updatedAccount = buildAccount(101, {
      password: 'new-password',
      email: 'new@example.com',
      registration_ip: '203.0.113.10',
      remark: 'saved reload fallback remark',
    })
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))
    updateMyAccount.mockResolvedValue(updatedAccount)

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    await wrapper.get('#account-edit-email').setValue('new@example.com')
    await wrapper.get('#account-edit-registration-ip').setValue('203.0.113.10')
    await wrapper.get('#account-edit-remark').setValue('saved reload fallback remark')

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: 'new-password',
      email: 'new@example.com',
      registration_ip: '203.0.113.10',
      remark: 'saved reload fallback remark',
    }))
    expect(showSuccess).toHaveBeenCalledWith('Saved')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(showError).not.toHaveBeenCalledWith('Save failed')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.load_data', expect.any(Error))
    expect(wrapper.text()).not.toContain('Edit account')

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const detailDialog = wrapper.get('[role="dialog"]')
    expect(detailDialog.text()).toContain('203.0.113.10')
    expect(detailDialog.text()).toContain('saved reload fallback remark')
  })

  it('preserves editable delivery field whitespace while normalizing contact fields', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        email_password: 'old-mail-secret',
        auth_cookie: 'ct0=old; auth_token=old',
        execution_auth: 'encrypted-old-execution-auth',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockResolvedValue(buildAccount(101, {
      password: '  new-password  ',
      phone: '+15550001111',
      email: 'trimmed@example.com',
      email_password: '  new-mail-secret  ',
      two_factor: '  totp-secret  ',
      backup_code: '  backup-code  ',
      email_client_id: '  mail-client  ',
      email_token: '  mail-token  ',
      registration_ip: '203.0.113.10',
      auth_cookie: '  ct0=new; auth_token=new  ',
      execution_auth: '  encrypted-execution-auth-ciphertext  ',
      remark: '  operator note  ',
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('  new-password  ')
    await wrapper.get('#account-edit-phone').setValue('  +15550001111  ')
    await wrapper.get('#account-edit-email').setValue('  trimmed@example.com  ')
    await wrapper.get('#account-edit-email-password').setValue('  new-mail-secret  ')
    await wrapper.get('#account-edit-two-factor').setValue('  totp-secret  ')
    await wrapper.get('#account-edit-backup-code').setValue('  backup-code  ')
    await wrapper.get('#account-edit-email-client-id').setValue('  mail-client  ')
    await wrapper.get('#account-edit-email-token').setValue('  mail-token  ')
    await wrapper.get('#account-edit-registration-ip').setValue('  203.0.113.10  ')
    await wrapper.get('#account-edit-auth-cookie').setValue('  ct0=new; auth_token=new  ')
    await wrapper.get('#account-edit-execution-auth').setValue('  encrypted-execution-auth-ciphertext  ')
    await wrapper.get('#account-edit-remark').setValue('  operator note  ')

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: '  new-password  ',
      phone: '+15550001111',
      email: 'trimmed@example.com',
      email_password: '  new-mail-secret  ',
      two_factor: '  totp-secret  ',
      backup_code: '  backup-code  ',
      email_client_id: '  mail-client  ',
      email_token: '  mail-token  ',
      registration_ip: '203.0.113.10',
      auth_cookie: '  ct0=new; auth_token=new  ',
      execution_auth: '  encrypted-execution-auth-ciphertext  ',
      remark: '  operator note  ',
    }))
    const payload = updateMyAccount.mock.calls[0]?.[1] || {}
    expect(JSON.stringify(payload)).not.toContain('"name"')
    expect(JSON.stringify(payload)).not.toContain('"platform"')
    expect(JSON.stringify(payload)).not.toContain('platform_user_id')
    expect(JSON.stringify(payload)).not.toContain('account_status')
    expect(JSON.stringify(payload)).not.toContain('task_status')
    expect(JSON.stringify(payload)).not.toContain('default_proxy_snapshot')
  })

  it('keeps trimmed contact-only account edit changes disabled while preserving delivery field edits', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        phone: '+15550001111',
        email: 'old@example.com',
        registration_ip: '198.51.100.20',
        auth_cookie: 'ct0=old; auth_token=old',
        execution_auth: 'encrypted-old-execution-auth',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockResolvedValue(buildAccount(101, {
      password: '  old-password  ',
      phone: '+15550001111',
      email: 'old@example.com',
      registration_ip: '198.51.100.20',
      auth_cookie: 'ct0=old; auth_token=old',
      execution_auth: 'encrypted-old-execution-auth',
      remark: 'old remark',
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-phone').setValue('  +15550001111  ')
    await wrapper.get('#account-edit-email').setValue('  old@example.com  ')
    await wrapper.get('#account-edit-registration-ip').setValue('  198.51.100.20  ')

    const unchangedSave = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(unchangedSave).toBeTruthy()
    expect(unchangedSave!.attributes('disabled')).toBeDefined()
    expect(unchangedSave!.attributes('title')).toBe('No changes to save.')
    await unchangedSave!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).not.toHaveBeenCalled()

    await wrapper.get('#account-edit-password').setValue('  old-password  ')
    const changedSave = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(changedSave).toBeTruthy()
    expect(changedSave!.attributes('disabled')).toBeUndefined()
    expect(changedSave!.attributes('title')).toBe('Save')
    await changedSave!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledWith(101, expect.objectContaining({
      password: '  old-password  ',
      phone: '+15550001111',
      email: 'old@example.com',
      registration_ip: '198.51.100.20',
      auth_cookie: 'ct0=old; auth_token=old',
      execution_auth: 'encrypted-old-execution-auth',
      remark: 'old remark',
    }))
  })

  it('keeps account edit open and maps invalid execution auth errors safely', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        execution_auth: 'old execution token',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockRejectedValue({
      status: 400,
      reason: 'SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID',
      message: 'account execution auth is invalid',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    await wrapper.get('#account-edit-execution-auth').setValue('invalid-encrypted-execution-auth')
    await wrapper.get('#account-edit-remark').setValue('new remark')
    const listCallsBeforeSave = listMyAccounts.mock.calls.length

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Execution auth format is invalid.')
    expect(showError).not.toHaveBeenCalledWith('Save failed')
    expect(showError).not.toHaveBeenCalledWith('account execution auth is invalid')
    expect(showSuccess).not.toHaveBeenCalledWith('Saved')
    expect(listMyAccounts).toHaveBeenCalledTimes(listCallsBeforeSave)
    expect(wrapper.text()).toContain('Edit account')
    expect((wrapper.get('#account-edit-password').element as HTMLInputElement).value).toBe('new-password')
    expect((wrapper.get('#account-edit-execution-auth').element as HTMLTextAreaElement).value).toBe('invalid-encrypted-execution-auth')
    expect((wrapper.get('#account-edit-remark').element as HTMLTextAreaElement).value).toBe('new remark')
    expect(wrapper.get('#account-edit-password').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('#account-edit-execution-auth').attributes('disabled')).toBeUndefined()
  })

  it('maps structured account input errors on edit without exposing raw backend details', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockRejectedValue({
      status: 400,
      reason: 'SOCIAL_ACCOUNT_INPUT_REQUIRED',
      message: 'unexpected EOF token=secret',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    const listCallsBeforeSave = listMyAccounts.mock.calls.length

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Account details are incomplete. Check the form and try again.')
    expect(showError).not.toHaveBeenCalledWith('Save failed')
    expect(showError).not.toHaveBeenCalledWith('unexpected EOF token=secret')
    expect(listMyAccounts).toHaveBeenCalledTimes(listCallsBeforeSave)
    expect(wrapper.text()).toContain('Edit account')
    const inlineError = wrapper.get('section[role="dialog"] [role="alert"]')
    expect(inlineError.text()).toBe('Account details are incomplete. Check the form and try again.')
    expect(inlineError.attributes('title')).toBe('Account details are incomplete. Check the form and try again.')
    expect(wrapper.text()).not.toContain('unexpected EOF token=secret')
    expect((wrapper.get('#account-edit-password').element as HTMLInputElement).value).toBe('new-password')

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()
    expect(wrapper.text()).not.toContain('Account details are incomplete. Check the form and try again.')

    const reopenedRowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    await reopenedRowButtons[2].trigger('click')
    await waitForWorkbench()
    expect(wrapper.find('section[role="dialog"] [role="alert"]').exists()).toBe(false)
  })

  it.each([
    [
      'SOCIAL_ACCOUNT_PASSWORD_REQUIRED',
      'Enter the account password.',
      'password cannot be empty token=secret',
    ],
    [
      'SOCIAL_ACCOUNT_IMPORT_INCOMPLETE',
      'Account delivery is incomplete. Provide a password and at least one login credential: 2FA, complete email credentials, or auth cookie.',
      'missing 2fa or cookie internal=delivery',
    ],
  ])('maps %s account edit errors without exposing raw backend details', async (reason, expectedMessage, rawMessage) => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        password: 'old-password',
        remark: 'old remark',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    updateMyAccount.mockRejectedValue({
      status: 400,
      reason,
      message: rawMessage,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('new-password')
    const listCallsBeforeSave = listMyAccounts.mock.calls.length

    const saveButton = wrapper.findAll('button').find(node => node.text().includes('Save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await waitForWorkbench()

    expect(updateMyAccount).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expectedMessage)
    expect(showError).not.toHaveBeenCalledWith('Save failed')
    expect(JSON.stringify(showError.mock.calls)).not.toContain(rawMessage)
    expect(listMyAccounts).toHaveBeenCalledTimes(listCallsBeforeSave)
    expect(wrapper.text()).toContain('Edit account')
  })

  it('syncs an open account edit form after account data refreshes', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101, {
          password: 'old-password',
          email: 'old@example.com',
          auth_cookie: 'ct0=old; auth_token=old',
          execution_auth: 'encrypted-old-execution-auth',
          registration_ip: '198.51.100.20',
          remark: 'old remark',
        })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(101, {
          password: 'refreshed-password',
          email: 'refreshed@example.com',
          auth_cookie: 'ct0=refreshed; auth_token=refreshed',
          execution_auth: 'encrypted-refreshed-execution-auth',
          registration_ip: '203.0.113.10',
          remark: 'refreshed remark',
        })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    expect((wrapper.get('#account-edit-password').element as HTMLInputElement).value).toBe('old-password')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect((wrapper.get('#account-edit-password').element as HTMLInputElement).value).toBe('refreshed-password')
    expect((wrapper.get('#account-edit-email').element as HTMLInputElement).value).toBe('refreshed@example.com')
    expect((wrapper.get('#account-edit-registration-ip').element as HTMLInputElement).value).toBe('203.0.113.10')
    expect((wrapper.get('#account-edit-auth-cookie').element as HTMLTextAreaElement).value).toBe('ct0=refreshed; auth_token=refreshed')
    expect((wrapper.get('#account-edit-execution-auth').element as HTMLTextAreaElement).value).toBe('encrypted-refreshed-execution-auth')
    expect((wrapper.get('#account-edit-remark').element as HTMLTextAreaElement).value).toBe('refreshed remark')
  })

  it('preserves unsaved account edit fields when a list refresh settles behind the open dialog', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101, {
          password: 'old-password',
          email: 'old@example.com',
          auth_cookie: 'ct0=old; auth_token=old',
          execution_auth: 'encrypted-old-execution-auth',
          registration_ip: '198.51.100.20',
          remark: 'old remark',
        })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValue({
        items: [buildAccount(101, {
          password: 'refreshed-password',
          email: 'refreshed@example.com',
          auth_cookie: 'ct0=refreshed; auth_token=refreshed',
          execution_auth: 'encrypted-refreshed-execution-auth',
          registration_ip: '203.0.113.10',
          remark: 'refreshed remark',
        })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await waitForWorkbench()

    const rowButtons = wrapper.get('[data-testid="account-row-101"]').findAll('button')
    expect(rowButtons.length).toBeGreaterThan(2)
    await rowButtons[2].trigger('click')
    await waitForWorkbench()

    await wrapper.get('#account-edit-password').setValue('draft-password')
    await wrapper.get('#account-edit-email').setValue('draft@example.com')
    await wrapper.get('#account-edit-auth-cookie').setValue('ct0=draft; auth_token=draft')
    await wrapper.get('#account-edit-execution-auth').setValue('encrypted-draft-execution-auth')
    await wrapper.get('#account-edit-remark').setValue('draft remark')

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect((wrapper.get('#account-edit-password').element as HTMLInputElement).value).toBe('draft-password')
    expect((wrapper.get('#account-edit-email').element as HTMLInputElement).value).toBe('draft@example.com')
    expect((wrapper.get('#account-edit-auth-cookie').element as HTMLTextAreaElement).value).toBe('ct0=draft; auth_token=draft')
    expect((wrapper.get('#account-edit-execution-auth').element as HTMLTextAreaElement).value).toBe('encrypted-draft-execution-auth')
    expect((wrapper.get('#account-edit-remark').element as HTMLTextAreaElement).value).toBe('draft remark')
  })

  it('summarizes long account delivery credential previews while copying full stored values', async () => {
    listTemplates.mockResolvedValue([])
    const emailToken = 'email-token-1234567890-secret'
    const authCookie = 'ct0=raw-cookie; auth_token=raw-token'
    const executionAuth = 'encrypted-execution-auth-ciphertext'
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        email_token: emailToken,
        auth_cookie: authCookie,
        execution_auth: executionAuth,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const credentialPreviewGrid = wrapper.get('[data-testid="account-credential-preview-grid"]')
    expect(credentialPreview.text()).toContain('Credential Details')
    expect(credentialPreviewGrid.classes()).toContain('md:grid-cols-2')
    expect(wrapper.find('[data-testid="account-credential-executionAuth"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-credential-authCookie"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).toContain('Raw cookie')
    expect(credentialPreview.text()).toContain('Encrypted value stored')
    expect(credentialPreview.text()).not.toContain('Executable')
    expect(credentialPreview.text()).not.toContain('screen_name saved')
    expect(credentialPreview.text()).toContain('Copy')
    expect(credentialPreview.text()).toContain('Refresh')
    const emailTokenPreview = wrapper.get('[data-testid="account-email-token-preview"]')
    expect(emailTokenPreview.text()).toContain('Email token')
    expect(emailTokenPreview.text()).toContain(`${emailToken.length} chars`)
    expect(emailTokenPreview.text()).not.toContain(emailToken)
    expect(emailTokenPreview.text()).toContain('Copy')
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).toContain(`${authCookie.length} chars`)
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).not.toContain(authCookie)
    expect(wrapper.get('[data-testid="account-credential-executionAuth"]').text()).toContain(`${executionAuth.length} chars`)
    expect(wrapper.get('[data-testid="account-credential-executionAuth"]').text()).not.toContain(executionAuth)
    expect(wrapper.find('[data-testid="account-credential-authCookie-value"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-credential-executionAuth-value"]').exists()).toBe(false)

    await wrapper.get('[data-testid="account-email-token-copy"]').trigger('click')
    await wrapper.get('[data-testid="account-credential-authCookie-copy"]').trigger('click')
    await wrapper.get('[data-testid="account-credential-executionAuth-copy"]').trigger('click')
    await waitForWorkbench()

    expect(writeClipboard).toHaveBeenCalledWith(emailToken)
    expect(writeClipboard).toHaveBeenCalledWith(authCookie)
    expect(writeClipboard).toHaveBeenCalledWith(executionAuth)
    expect(showSuccess).toHaveBeenCalledWith('Credential copied.')
  })

  it('shows login auth backup and execution auth as separate credential cards', async () => {
    listTemplates.mockResolvedValue([])
    const loginAuthBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    const executionAuth = 'encrypted-execution-auth-ciphertext'
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: loginAuthBackup,
        execution_auth: executionAuth,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    expect(wrapper.find('[data-testid="account-credential-executionAuth"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-credential-authCookie"]').exists()).toBe(true)
    expect(credentialPreview.text()).toContain('OAuth complete')
    expect(wrapper.get('[data-testid="account-credential-executionAuth"]').text()).toContain('Encrypted value stored')
    expect(credentialPreview.text()).toContain('Auth cookie')
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).toContain(`${loginAuthBackup.length} chars`)
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).not.toContain('cookie-access')
    expect(wrapper.get('[data-testid="account-credential-authCookie"]').text()).not.toContain('cookie-secret')
    expect(wrapper.get('[data-testid="account-credential-executionAuth"]').text()).toContain(`${executionAuth.length} chars`)
    expect(wrapper.get('[data-testid="account-credential-executionAuth"]').text()).not.toContain(executionAuth)
    expect(wrapper.find('[data-testid="account-credential-authCookie-value"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-credential-executionAuth-value"]').exists()).toBe(false)
  })

  it('submits a login refresh instead of backfilling execution auth from auth_cookie', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: fullLoginBackup,
        execution_auth: '',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7010,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledWith({
      account_ids: [101],
      action: 'login',
      client_request_id: expect.any(String),
    })
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0] || {})).not.toContain('cookie-access')
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0] || {})).not.toContain('cookie-secret')
    expect(showSuccess).toHaveBeenCalledWith('Submitted 1, queued 1.')
  })

  it('keeps account detail open while execution auth login refresh is in flight', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    let resolveSubmit!: (value: {
      submitted: number
      enqueued: number
      failed_closed: number
      logs: Array<Record<string, unknown>>
    }) => void
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: fullLoginBackup,
        execution_auth: '',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockReturnValue(new Promise((resolve) => {
      resolveSubmit = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    const processingButton = wrapper.findAll('button').find(node => node.text().includes('Processing'))
    expect(processingButton).toBeTruthy()
    expect(processingButton!.attributes('disabled')).toBeDefined()

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    expect(closeButton!.attributes('disabled')).toBeDefined()
    expect(closeButton!.attributes('title')).toBe('Processing')
    await closeButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Credential Details')

    resolveSubmit({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7011,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Submitted 1, queued 1.')
  })

  it('prevents concurrent execution-auth login refreshes across account detail switches', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    let resolveSubmit!: (value: {
      submitted: number
      enqueued: number
      failed_closed: number
      logs: Array<Record<string, unknown>>
    }) => void
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101, {
          name: 'x-main-101',
          auth_cookie: fullLoginBackup,
          execution_auth: '',
        }),
        buildAccount(102, {
          name: 'x-main-102',
          auth_cookie: fullLoginBackup,
          execution_auth: '',
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockReturnValue(new Promise((resolve) => {
      resolveSubmit = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()
    const firstRefreshButton = wrapper.get('[data-testid="account-credential-executionAuth"]').findAll('button').find(node => node.text().includes('Refresh'))
    expect(firstRefreshButton).toBeTruthy()
    await firstRefreshButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-102"] button').trigger('click')
    await waitForWorkbench()

    const secondRefreshButton = wrapper.get('[data-testid="account-credential-executionAuth"]').findAll('button').find(node => node.text().includes('Refresh'))
    expect(secondRefreshButton).toBeTruthy()
    expect(secondRefreshButton!.attributes('disabled')).toBeDefined()
    expect(secondRefreshButton!.attributes('title')).toBe('Processing')
    await secondRefreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask).toHaveBeenCalledWith({
      account_ids: [101],
      action: 'login',
      client_request_id: expect.any(String),
    })

    resolveSubmit({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7012,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    await waitForWorkbench()
  })

  it('handles execution-auth login refresh submit errors safely and keeps detail open', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: fullLoginBackup,
        execution_auth: '',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockRejectedValue({
      status: 400,
      reason: 'SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID',
      message: 'account execution auth is invalid',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Refresh failed')
    expect(showError).not.toHaveBeenCalledWith('account execution auth is invalid')
    expect(wrapper.text()).toContain('Credential Details')
    const closeButton = wrapper.findAll('button').find(node => node.text().includes('Close'))
    expect(closeButton).toBeTruthy()
    expect(closeButton!.attributes('disabled')).toBeUndefined()
  })

  it('clears a stale proxy assignment result when execution-auth login refresh fails', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: fullLoginBackup,
        execution_auth: '',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    batchSetDefaultProxy.mockResolvedValue({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
      items: [{ id: 101, name: 'x-main-101', status: 'failed', reason: 'proxy_not_available' }],
    })
    submitTask.mockRejectedValue({
      status: 400,
      reason: 'SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID',
      message: 'account execution auth is invalid',
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    await openBatchProxyDialogForSelectedAccounts(wrapper)

    const applyButton = wrapper.findAll('button').find(node => node.text().includes('Apply proxy'))
    expect(applyButton).toBeTruthy()
    await applyButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    await cancelButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).toContain('Total 1, succeeded 0, failed 1, skipped 0.')

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Refresh failed')
    expect(showError).not.toHaveBeenCalledWith('account execution auth is invalid')
    expect(wrapper.text()).toContain('Credential Details')
    expect(wrapper.text()).not.toContain('Proxy result')
    expect(wrapper.text()).not.toContain('Total 1, succeeded 0, failed 1, skipped 0.')
  })

  it('does not reinsert refreshed execution credentials into an off-filter account list', async () => {
    listTemplates.mockResolvedValue([])
    const fullLoginBackup = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    const originalAccount = buildAccount(101, {
      name: 'x-refresh-filtered-out',
      username: 'x-refresh-filtered-out',
      auth_cookie: fullLoginBackup,
      execution_auth: '',
    })
    let resolveSubmit!: (value: {
      submitted: number
      enqueued: number
      failed_closed: number
      logs: Array<Record<string, unknown>>
    }) => void
    listMyAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.search === 'no-matching-account') {
        return Promise.resolve({
          items: [],
          total: 0,
          page: 1,
          page_size: 200,
          pages: 1,
        })
      }
      return Promise.resolve({
        items: [originalAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    })
    submitTask.mockReturnValue(new Promise((resolve) => {
      resolveSubmit = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-stat-assigned"]').text()).toContain('1')
    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('no-matching-account')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenCalledWith({
      page: 1,
      page_size: 200,
      search: 'no-matching-account',
    })
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-stat-assigned"]').text()).toContain('0')
    expect(wrapper.text()).not.toContain('Credential Details')

    resolveSubmit({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7012,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('Submitted 1, queued 1.')
    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-stat-assigned"]').text()).toContain('0')
    expect(wrapper.text()).not.toContain('Credential Details')
  })

  it('shows an error when execution-auth refresh login fails closed before enqueue', async () => {
    listTemplates.mockResolvedValue([])
    const account = buildAccount(101, {
      auth_cookie: 'ct0=raw-cookie; auth_token=raw-token',
      execution_auth: '',
      default_proxy_configured: true,
      password: 'secret',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [account],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 0,
      failed_closed: 1,
      logs: [{
        id: 7010,
        social_account_id: 101,
        action: 'login',
        status: 'failed',
        result_message: 'executor unavailable',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    expect(refreshButton!.attributes('disabled')).toBeUndefined()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      action: 'login',
    })
    expect(showError).toHaveBeenCalledWith('Submitted 1; queued 0; failed 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Submitted 1, queued 0.')
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('executor unavailable')

    resolveRefresh({
      items: [buildAccount(101, {
        auth_cookie: 'ct0=raw-cookie; auth_token=raw-token',
        execution_auth: '',
        default_proxy_configured: true,
        password: 'secret',
        task_status: 'failed',
        task_message: 'executor unavailable',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('removes an execution-auth refresh account from selection after it is enqueued', async () => {
    listTemplates.mockResolvedValue([])
    const account = buildAccount(101, {
      auth_cookie: 'ct0=raw-cookie; auth_token=raw-token',
      execution_auth: '',
      default_proxy_configured: true,
      password: 'secret',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [account],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7014,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()
    expect(wrapper.text()).toContain('1 selected')

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledTimes(1)
    expect(submitTask.mock.calls[0]?.[0]).toMatchObject({
      account_ids: [101],
      action: 'login',
    })
    expect(showSuccess).toHaveBeenCalledWith('Submitted 1, queued 1.')
    expect((wrapper.get('[data-testid="account-row-101"] input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    expect(wrapper.text()).not.toContain('1 selected')
    expect(wrapper.get('[data-testid="account-row-101"]').text()).toContain('Pending')

    resolveRefresh({
      items: [buildAccount(101, {
        auth_cookie: 'ct0=raw-cookie; auth_token=raw-token',
        execution_auth: '',
        default_proxy_configured: true,
        password: 'secret',
        task_status: 'pending',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('treats non-json execution auth as an opaque encrypted stored value', async () => {
    listTemplates.mockResolvedValue([])
    const rawExecutionAuth = 'MTIzNDU2Nzg5MGFiY2RlZg=='
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: '{"access_token":"cookie-access","token_secret":"cookie-secret"}',
        execution_auth: rawExecutionAuth,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const executionCard = wrapper.get('[data-testid="account-credential-executionAuth"]')
    expect(executionCard.text()).not.toContain('Raw value saved')
    expect(executionCard.text()).toContain('Encrypted value stored')
    expect(executionCard.text()).not.toContain('OAuth complete')
    expect(executionCard.text()).not.toContain('Executable')
    expect(executionCard.text()).toContain(`${rawExecutionAuth.length} chars`)
    expect(executionCard.text()).not.toContain(rawExecutionAuth)
    expect(wrapper.find('[data-testid="account-credential-executionAuth-value"]').exists()).toBe(false)
  })

  it('treats unsupported execution auth envelopes as opaque encrypted stored values', async () => {
    listTemplates.mockResolvedValue([])
    const wrappedAuth = JSON.stringify({
      kind: 'unsupported_execution_auth_envelope',
      schema_version: 1,
      ['encr' + 'yption']: 'unsupported_envelope',
      access_token: 'wrapped-access',
      token_secret: 'wrapped-secret',
      screen_name: 'wrapped-screen-name',
    })
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: '{"access_token":"cookie-access","token_secret":"cookie-secret"}',
        execution_auth: wrappedAuth,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const executionCard = wrapper.get('[data-testid="account-credential-executionAuth"]')
    expect(executionCard.text()).not.toContain('JSON detected')
    expect(executionCard.text()).not.toContain('Executable')
    expect(executionCard.text()).not.toContain('Incomplete')
    expect(executionCard.text()).toContain('Encrypted value stored')
    expect(executionCard.text()).not.toContain('OAuth complete')
    expect(executionCard.text()).not.toContain('screen_name saved')
    expect(executionCard.text()).toContain(`${wrappedAuth.length} chars`)
    expect(executionCard.text()).not.toContain('wrapped-access')
    expect(executionCard.text()).not.toContain('wrapped-secret')
    expect(executionCard.text()).not.toContain('wrapped-screen-name')
    expect(wrapper.find('[data-testid="account-credential-executionAuth-value"]').exists()).toBe(false)
  })

  it('does not build execution auth plaintext from an OAuth login auth backup in the detail preview', async () => {
    listTemplates.mockResolvedValue([])
    const authCookie = JSON.stringify({
      access_token: 'cookie-access',
      token_secret: 'cookie-secret',
      screen_name: 'northwind_ops',
      guest_token: 'guest-token',
      client_uuid: 'client-uuid',
      user_agent: 'Mozilla/5.0',
    })
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        auth_cookie: authCookie,
        execution_auth: '',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    submitTask.mockResolvedValue({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
      logs: [{
        id: 7013,
        social_account_id: 101,
        action: 'login',
        status: 'pending',
        charged: false,
        charged_amount: 0,
        charge_status: 'not_charged',
        created_at: new Date().toISOString(),
      }],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-101"] button').trigger('click')
    await waitForWorkbench()

    const credentialPreview = wrapper.get('[data-testid="account-credential-preview"]')
    const executionCard = wrapper.get('[data-testid="account-credential-executionAuth"]')
    expect(executionCard.text()).not.toContain('OAuth complete')
    const refreshButton = credentialPreview.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await waitForWorkbench()

    expect(submitTask).toHaveBeenCalledWith({
      account_ids: [101],
      action: 'login',
      client_request_id: expect.any(String),
    })
    expect(updateMyAccount).not.toHaveBeenCalled()
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0] || {})).not.toContain('cookie-access')
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0] || {})).not.toContain('cookie-secret')
    expect(showSuccess).toHaveBeenCalledWith('Submitted 1, queued 1.')
  })

  it('allows login submission without an account default proxy so the backend can use the global proxy pool', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, { default_proxy_configured: false })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Login needs a default proxy.')
    const submitButton = findExecutionStartButton(wrapper)
    expect(submitButton.attributes('disabled')).toBeUndefined()
    await submitButton.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Confirm task execution')
  })

  it.each([
    [{ password: '' }, 'Login needs the account password.'],
    [{ default_proxy_configured: false, password: '' }, 'Login needs the account password.'],
  ])('shows a precise login prerequisite message for %o', async (overrides, message) => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, overrides)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)
    await waitForWorkbench()

    expect(wrapper.text()).toContain(message)
    const submitButton = findExecutionStartButton(wrapper)
    expect(submitButton.attributes('disabled')).toBeDefined()
    await submitButton.trigger('click')
    await waitForWorkbench()

    expect(submitTask).not.toHaveBeenCalled()
  })

  it('marks batch import rows as existing when the current account username matches', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(101, {
        name: 'Northwind Display',
        username: 'northwind_ops',
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@NorthWind_Ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Already exists')
    expect(wrapper.text()).toContain('No')
    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeDefined()
    expect(batchImportMyAccounts).not.toHaveBeenCalled()
  })

  it('checks batch import duplicates against unfiltered workbench accounts after filters narrow the table', async () => {
    listTemplates.mockResolvedValue([])
    const visibleAccount = buildAccount(101, {
      name: 'Visible Ops',
      username: 'visible_ops',
    })
    const hiddenDuplicate = buildAccount(202, {
      name: 'Hidden Duplicate',
      username: 'hidden_ops',
    })
    listMyAccounts.mockImplementation((params: Record<string, unknown>) => {
      if (params.page_size === 1000 && !params.search && !params.platform && !params.account_status) {
        return Promise.resolve({
          items: [visibleAccount, hiddenDuplicate],
          total: 2,
          page: 1,
          page_size: 1000,
          pages: 1,
        })
      }
      if (params.search === 'visible') {
        return Promise.resolve({
          items: [visibleAccount],
          total: 1,
          page: 1,
          page_size: 200,
          pages: 1,
        })
      }
      return Promise.resolve({
        items: [visibleAccount, hiddenDuplicate],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    })

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('visible')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    expect(wrapper.find('[data-testid="account-row-101"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-row-202"]').exists()).toBe(false)

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    expect(listMyAccounts).toHaveBeenCalledWith({ page: 1, page_size: 1000 })

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@Hidden_Ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Already exists')
    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeDefined()
    expect(batchImportMyAccounts).not.toHaveBeenCalled()
  })

  it('locks an open batch import confirmation while the account list is refreshing', async () => {
    listTemplates.mockResolvedValue([])
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [buildAccount(101)],
        total: 1,
        page: 1,
        page_size: 1000,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('textarea').setValue('@fresh_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const confirmButton = () => {
      const buttons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
      expect(buttons.length).toBeGreaterThan(0)
      return buttons[buttons.length - 1]
    }
    expect(confirmButton().attributes('disabled')).toBeUndefined()

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(confirmButton().attributes('disabled')).toBeDefined()
    await confirmButton().trigger('click')
    await waitForWorkbench()
    expect(batchImportMyAccounts).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Batch import')

    resolveRefresh({
      items: [buildAccount(101)],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(confirmButton().attributes('disabled')).toBeUndefined()
  })

  it('shows a warning toast when batch import partially succeeds', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts.mockResolvedValue({
      total: 3,
      succeeded: 1,
      imported: 1,
      skipped: 2,
      failed: 1,
      duplicates: 1,
      errors: [
        'account already exists in the total account pool',
        'multiple total-pool accounts match this username',
      ],
      items: [
        { id: 501, name: '  fresh_ops  ', status: 'succeeded', reason: 'staged_not_stored' },
        { name: '  duplicate_ops  ', status: 'duplicate', reason: 'duplicate_in_database', error: 'account already exists in the total account pool' },
        { name: '  ambiguous_ops  ', status: 'failed', reason: 'ambiguous_total_pool_match', error: 'multiple total-pool accounts match this username' },
      ],
      accounts: [buildAccount(501, { name: 'fresh_ops', username: 'fresh_ops' })],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_ops\tpw\tJBSWY3DPEHPK3PXP\n@duplicate_ops\tpw\tJBSWY3DPEHPK3PXP\n@ambiguous_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeUndefined()
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(batchImportMyAccounts.mock.calls[0]?.[0]).toHaveLength(3)
    expect(showWarning).toHaveBeenCalledWith('Total 3, imported 1, failed 1, skipped 2, duplicates 1.')
    expect(showSuccess).not.toHaveBeenCalledWith('Imported 1')
    expect(showError).not.toHaveBeenCalledWith('Total 3, imported 1, failed 1, skipped 2, duplicates 1.')
    expect(wrapper.text()).toContain('Import result')
    expect(wrapper.text()).toContain('Total 3, imported 1, failed 1, skipped 2, duplicates 1.')
    const resultSummary = wrapper.get('[title="Total 3, imported 1, failed 1, skipped 2, duplicates 1."]')
    expect(resultSummary.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(resultSummary.attributes('role')).toBe('status')
    expect(resultSummary.attributes('aria-live')).toBe('polite')
    expect(resultSummary.attributes('aria-atomic')).toBe('true')
    expect(wrapper.text()).toContain('Duplicate')
    expect(wrapper.text()).toContain('Staged not stored')
    expect(wrapper.text()).toContain('Duplicate in total pool')
    expect(wrapper.text()).toContain('Ambiguous total pool match')
    const resultRows = wrapper.findComponent(SocialAccountBatchResultRows)
    const resultLabels = Array.from(resultRows.element.querySelectorAll('span.font-medium'))
      .map(node => node.textContent)
    expect(resultLabels).toEqual(['fresh_ops', 'duplicate_ops', 'ambiguous_ops'])
    expect(wrapper.text()).not.toContain('staged_not_stored')
    expect(wrapper.text()).not.toContain('duplicate_in_database')
    expect(wrapper.text()).not.toContain('ambiguous_total_pool_match')
    expect(wrapper.text()).not.toContain('account already exists in the total account pool')
    expect(wrapper.text()).not.toContain('multiple total-pool accounts match this username')
  })

  it('syncs imported accounts from the API response before the next list refresh', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101)
    const importedAccount = buildAccount(501, {
      name: '@fresh_local',
      username: 'fresh_local',
      remark: 'instant import remark',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 501, name: '@fresh_local', status: 'succeeded' },
      ],
      accounts: [importedAccount],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_local\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    await wrapper.get('[data-testid="search-input-stub"]').setValue('instant import remark')
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-501"]').exists()).toBe(true)
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')

    resolveRefresh({
      items: [initialAccount, importedAccount],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('keeps imported accounts visible locally when the follow-up list refresh fails', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101)
    const importedAccount = buildAccount(502, {
      name: '@fresh_reload_fallback',
      username: 'fresh_reload_fallback',
      remark: 'reload fallback import remark',
    })
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 1000,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 502, name: '@fresh_reload_fallback', status: 'succeeded' },
      ],
      accounts: [importedAccount],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_reload_fallback\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="account-row-502"]').text()).toContain('fresh_reload_fallback')
    expect(wrapper.text()).toContain('Import result')
    expect(wrapper.text()).toContain('Total 1, imported 1, failed 0, skipped 0, duplicates 0.')
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')
    expect(showError).toHaveBeenCalledWith('Failed to load')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.load_data', expect.any(Error))
  })

  it('syncs matched total-pool delivery fields from batch import before the next list refresh', async () => {
    listTemplates.mockResolvedValue([])
    const initialAccount = buildAccount(101)
    const matchedPoolAccount = buildAccount(601, {
      name: '@pool_match_local',
      username: 'pool_match_local',
      password: 'pool-secret',
      phone: '+15550003333',
      auth_cookie: 'ct0=pool; auth_token=pool',
      remark: 'pool inventory delivery note',
      default_proxy_configured: false,
      default_proxy_snapshot: '',
      task_status: 'stored',
    })
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [initialAccount],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 601, name: '@pool_match_local', status: 'succeeded' },
      ],
      accounts: [matchedPoolAccount],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue([
      '@pool_match_local',
      'typed-secret',
      'TYPED2FA',
      '',
      '',
      '',
      '',
      '',
      '',
      'ct0=typed; auth_token=typed',
      '',
      '',
      'typed import note',
    ].join('\t'))
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    await wrapper.get('[data-testid="search-input-stub"]').setValue('pool inventory delivery note')
    await waitForWorkbench()

    expect(wrapper.get('[data-testid="account-row-601"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('typed import note')

    await wrapper.get('[data-testid="search-input-stub"]').setValue('pool-secret')
    await waitForWorkbench()
    expect(wrapper.get('[data-testid="account-row-601"]').exists()).toBe(true)

    await wrapper.get('[data-testid="search-input-stub"]').setValue('typed-secret')
    await waitForWorkbench()
    expect(wrapper.find('[data-testid="account-row-601"]').exists()).toBe(false)
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')

    resolveRefresh({
      items: [initialAccount, matchedPoolAccount],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()
  })

  it('submits supported delivery fields from extended batch import rows', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 501, name: '@delivery_ops', status: 'succeeded' },
      ],
      accounts: [buildAccount(501, { name: '@delivery_ops', username: 'delivery_ops' })],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const executionAuth = 'encrypted-import-execution-auth'
    const textarea = wrapper.get('textarea')
    await textarea.setValue([
      '@delivery_ops',
      '  account-secret  ',
      'JBSWY3DPEHPK3PXP',
      'backup-1',
      '  delivery@example.com  ',
      '  mail-secret  ',
      'mail-client',
      'mail-token',
      '  203.0.113.8  ',
      '  ct0=ok; auth_token=ok  ',
      `  ${executionAuth}  `,
      '+15550001111',
      '  delivery remark  ',
    ].join('\t'))
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeUndefined()
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(batchImportMyAccounts.mock.calls[0]?.[0]).toEqual([{
      platform: 'x_twitter',
      name: '@delivery_ops',
      password: '  account-secret  ',
      phone: '+15550001111',
      email: 'delivery@example.com',
      email_password: '  mail-secret  ',
      auth_cookie: '  ct0=ok; auth_token=ok  ',
      execution_auth: `  ${executionAuth}  `,
      two_factor: 'JBSWY3DPEHPK3PXP',
      backup_code: 'backup-1',
      email_client_id: 'mail-client',
      email_token: 'mail-token',
      registration_ip: '203.0.113.8',
      remark: '  delivery remark  ',
    }])
    expect(JSON.stringify(batchImportMyAccounts.mock.calls[0]?.[0])).not.toContain('default_proxy_snapshot')
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')
  })

  it('imports XLSX batch rows with the fixed column order', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 502, name: '@xlsx_delivery', status: 'succeeded' },
      ],
      accounts: [buildAccount(502, { name: '@xlsx_delivery', username: 'xlsx_delivery' })],
    })

    const workbook = XLSX.utils.book_new()
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['账号', '密码', '2FA', '手机号', '邮箱账号', '邮箱密码', '邮箱 Client ID', '邮箱 Token'],
      ['@xlsx_delivery', '  account-secret  ', 'JBSWY3DPEHPK3PXP', '  +15550002222  ', '  delivery@example.com  ', '  mail-secret  ', 'mail-client', 'mail-token'],
    ])
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Accounts')
    const fileBuffer = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
    const longFileName = 'accounts_with_extremely_long_delivery_credentials_filename_for_mobile_review.xlsx'
    const file = {
      name: longFileName,
      arrayBuffer: vi.fn(async () => fileBuffer),
    }

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [file],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(wrapper.text()).toContain(longFileName)
    expect(wrapper.text()).toContain('@xlsx_delivery')
    const fileNameLabel = wrapper.findAll('[data-testid="accounts-batch-import-dropzone"] span').find(node => node.text() === longFileName)
    expect(fileNameLabel).toBeTruthy()
    expect(fileNameLabel!.attributes('title')).toBe(longFileName)
    expect(fileNameLabel!.classes()).toContain('break-all')
    expect(fileNameLabel!.classes()).toContain('max-w-full')

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeUndefined()
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(batchImportMyAccounts.mock.calls[0]?.[0]).toEqual([{
      platform: 'x_twitter',
      name: '@xlsx_delivery',
      password: '  account-secret  ',
      phone: '+15550002222',
      email: 'delivery@example.com',
      email_password: '  mail-secret  ',
      two_factor: 'JBSWY3DPEHPK3PXP',
      email_client_id: 'mail-client',
      email_token: 'mail-token',
    }])
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')
  })

  it('accepts dropped txt files in the larger batch import drop zone', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 503, name: '@dropped_delivery', status: 'succeeded' },
      ],
      accounts: [buildAccount(503, { name: '@dropped_delivery', username: 'dropped_delivery' })],
    })
    const file = {
      name: 'dropped-accounts.txt',
      text: vi.fn(async () => '@dropped_delivery\taccount-secret\tJBSWY3DPEHPK3PXP'),
    }

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const dropZone = wrapper.get('[data-testid="accounts-batch-import-dropzone"]')
    expect(dropZone.text()).toContain('Drop file here')
    expect(dropZone.text()).toContain('Choose file')
    await dropZone.trigger('dragenter', { dataTransfer: { files: [file] } })
    expect(dropZone.classes()).toContain('border-primary-400')
    await dropZone.trigger('drop', { dataTransfer: { files: [file] } })
    await waitForWorkbench()

    expect(wrapper.text()).toContain('dropped-accounts.txt')
    expect(wrapper.text()).toContain('@dropped_delivery')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeUndefined()
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(batchImportMyAccounts.mock.calls[0]?.[0]).toEqual([{
      platform: 'x_twitter',
      name: '@dropped_delivery',
      password: 'account-secret',
      two_factor: 'JBSWY3DPEHPK3PXP',
    }])
    expect(showSuccess).toHaveBeenCalledWith('Imported 1')
  })

  it('clears stale pasted rows when a selected batch import file is unsupported', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@stale_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const readySubmitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(readySubmitButtons.length).toBeGreaterThan(0)
    expect(readySubmitButtons[readySubmitButtons.length - 1].attributes('disabled')).toBeUndefined()

    const unsupportedFile = {
      name: 'accounts.csv',
      text: vi.fn(async () => '@fresh_ops\tpw\tJBSWY3DPEHPK3PXP'),
    }
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [unsupportedFile],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Unsupported file')
    const importError = wrapper.findAll('div').find(node => node.text() === 'Unsupported file')
    expect(importError).toBeTruthy()
    expect(importError!.attributes('title')).toBe('Unsupported file')
    expect(importError!.attributes('role')).toBe('alert')
    expect(importError!.attributes('aria-live')).toBe('assertive')
    expect(importError!.attributes('aria-atomic')).toBe('true')
    expect(importError!.classes()).toContain('break-words')
    expect(importError!.classes()).toContain('min-w-0')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).not.toContain('@stale_ops')
    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeDefined()

    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).not.toHaveBeenCalled()
  })

  it('shows a clear batch import error when a selected file has no importable rows', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('textarea').setValue('@stale_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const readySubmitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(readySubmitButtons.length).toBeGreaterThan(0)
    expect(readySubmitButtons[readySubmitButtons.length - 1].attributes('disabled')).toBeUndefined()

    const emptyFile = {
      name: 'empty-accounts.txt',
      text: vi.fn(async () => '   \n\n  '),
    }
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [emptyFile],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('empty-accounts.txt')
    expect(wrapper.text()).toContain('No importable account rows were found in this file')
    const importError = wrapper.findAll('div').find(node => node.text() === 'No importable account rows were found in this file')
    expect(importError).toBeTruthy()
    expect(importError!.attributes('title')).toBe('No importable account rows were found in this file')
    expect(importError!.attributes('role')).toBe('alert')
    expect(importError!.attributes('aria-live')).toBe('assertive')
    expect(importError!.attributes('aria-atomic')).toBe('true')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).not.toContain('@stale_ops')

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeDefined()

    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).not.toHaveBeenCalled()
  })

  it('keeps batch import file read failures safe and clears stale pasted rows', async () => {
    listTemplates.mockResolvedValue([])
    const rawError = new Error('local file read failed token=secret')

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('textarea').setValue('@stale_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const readySubmitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(readySubmitButtons.length).toBeGreaterThan(0)
    expect(readySubmitButtons[readySubmitButtons.length - 1].attributes('disabled')).toBeUndefined()

    const unreadableFile = {
      name: 'unreadable-accounts.txt',
      text: vi.fn(async () => {
        throw rawError
      }),
    }
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [unreadableFile],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('File read failed')
    expect(wrapper.text()).not.toContain('token=secret')
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.batch_import_file', rawError)
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).not.toContain('@stale_ops')
    expect(wrapper.text()).not.toContain('unreadable-accounts.txt')

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeDefined()

    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).not.toHaveBeenCalled()
  })

  it('ignores stale batch import file changes while another file is parsing', async () => {
    listTemplates.mockResolvedValue([])
    let resolveFirstFile: (value: string) => void = () => {}
    const firstFile = {
      name: 'first-accounts.txt',
      text: vi.fn(() => new Promise<string>(resolve => {
        resolveFirstFile = resolve
      })),
    }
    const secondFile = {
      name: 'second-accounts.txt',
      text: vi.fn(async () => '@second_ops\tpw\tJBSWY3DPEHPK3PXP'),
    }

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('textarea').setValue('@stale_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [firstFile],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(firstFile.text).toHaveBeenCalledTimes(1)
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).not.toContain('@stale_ops')
    const parsingCancelButton = wrapper.findAll('section[role="dialog"] button')
      .find(node => node.text().includes('Cancel'))
    expect(parsingCancelButton).toBeTruthy()
    expect(parsingCancelButton!.attributes('disabled')).toBeDefined()
    expect(parsingCancelButton!.attributes('aria-label')).toBe('Processing')
    expect(parsingCancelButton!.attributes('title')).toBe('Processing')

    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [secondFile],
    })
    await input.trigger('change')
    await waitForWorkbench()

    expect(secondFile.text).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('second-accounts.txt')

    resolveFirstFile('@first_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('first-accounts.txt')
    expect(wrapper.text()).toContain('@first_ops')
    expect(wrapper.text()).not.toContain('@second_ops')
  })

  it('clears hidden batch import drafts when the dialog closes', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    await wrapper.get('textarea').setValue('@draft_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()
    expect(wrapper.text()).toContain('@draft_ops')

    const readySubmitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(readySubmitButtons.length).toBeGreaterThan(0)
    expect(readySubmitButtons[readySubmitButtons.length - 1].attributes('disabled')).toBeUndefined()

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    expect(wrapper.find('textarea').exists()).toBe(false)

    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).not.toContain('@draft_ops')
    const reopenedSubmitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(reopenedSubmitButtons.length).toBeGreaterThan(0)
    expect(reopenedSubmitButtons[reopenedSubmitButtons.length - 1].attributes('disabled')).toBeDefined()
  })

  it('keeps batch import dialog locked while the import request is in flight', async () => {
    listTemplates.mockResolvedValue([])
    let resolveImport!: (value: {
      total: number
      succeeded: number
      imported: number
      skipped: number
      failed: number
      duplicates: number
      errors: string[]
      items: Array<{ id?: number; name: string; status: string }>
      accounts: Array<ReturnType<typeof buildAccount>>
    }) => void
    batchImportMyAccounts.mockReturnValue(new Promise((resolve) => {
      resolveImport = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const confirmButton = wrapper.findAll('button').filter(node => node.text().includes('Confirm')).at(-1)
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await waitForWorkbench()

    const processingButton = wrapper.findAll('button').find(node => node.text().includes('Processing'))
    expect(processingButton).toBeTruthy()
    expect(processingButton!.attributes('disabled')).toBeDefined()

    await clickLatestDialogClose(wrapper)
    await waitForWorkbench()

    expect(wrapper.get('textarea').attributes('disabled')).toBeDefined()
    const platformSelect = wrapper.findAll('[data-testid="select-stub"]').find(node => node.attributes('id') === 'accounts-batch-import-platform')
    expect(platformSelect).toBeTruthy()
    expect(platformSelect!.attributes('disabled')).toBeDefined()

    const cancelButton = wrapper.findAll('button').find(node => node.text().includes('Cancel'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeDefined()
    expect(cancelButton!.attributes('title')).toBe('Processing')
    await cancelButton!.trigger('click')
    await processingButton!.trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Batch import')
    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(batchImportMyAccounts.mock.calls[0]?.[0]).toHaveLength(1)

    resolveImport({
      total: 1,
      succeeded: 1,
      imported: 1,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [{ id: 501, name: 'fresh_ops', status: 'succeeded' }],
      accounts: [buildAccount(501, { name: 'fresh_ops', username: 'fresh_ops' })],
    })
    await waitForWorkbench()

    expect(showSuccess).toHaveBeenCalledWith('Imported 1')
    expect(wrapper.text()).toContain('Import result')
    expect(wrapper.text()).toContain('Total 1, imported 1, failed 0, skipped 0, duplicates 0.')
  })

  it('shows an error toast when batch import returns no successes', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts.mockResolvedValue({
      total: 1,
      succeeded: 0,
      imported: 0,
      skipped: 1,
      failed: 1,
      duplicates: 0,
      errors: ['account could not be imported'],
      items: [
        { status: 'failed', error: 'account could not be imported' },
      ],
      accounts: [],
    })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@broken_ops\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const submitButtons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
    expect(submitButtons.length).toBeGreaterThan(0)
    expect(submitButtons[submitButtons.length - 1].attributes('disabled')).toBeUndefined()
    await submitButtons[submitButtons.length - 1].trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Total 1, imported 0, failed 1, skipped 1, duplicates 0.')
    expect(showSuccess).not.toHaveBeenCalledWith('Imported 0')
    expect(showWarning).not.toHaveBeenCalledWith('Total 1, imported 0, failed 1, skipped 1, duplicates 0.')
    expect(wrapper.text()).toContain('Import result')
    expect(wrapper.text()).toContain('Total 1, imported 0, failed 1, skipped 1, duplicates 0.')
    expect(wrapper.text()).toContain('Error')
    expect(wrapper.text()).toContain('Import failed')
    const resultRows = wrapper.findComponent(SocialAccountBatchResultRows)
    const resultLabels = Array.from(resultRows.element.querySelectorAll('span.font-medium'))
      .map(node => node.textContent)
    expect(resultLabels).toEqual(['#1'])
    expect(wrapper.text()).not.toContain('account could not be imported')
    expect(wrapper.text()).not.toContain('import_failed')
  })

  it('clears the stale batch import result when a retry request fails', async () => {
    listTemplates.mockResolvedValue([])
    batchImportMyAccounts
      .mockResolvedValueOnce({
        total: 1,
        succeeded: 0,
        imported: 0,
        skipped: 1,
        failed: 1,
        duplicates: 0,
        errors: ['account could not be imported'],
        items: [
          { name: 'fresh_retry', status: 'failed', reason: 'import_failed' },
        ],
        accounts: [],
      })
      .mockRejectedValueOnce({
        response: {
          data: {
            detail: 'retry import failed token=secret',
          },
        },
      })

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@fresh_retry\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const confirmButton = () => {
      const buttons = wrapper.findAll('button').filter(node => node.text().includes('Confirm'))
      expect(buttons.length).toBeGreaterThan(0)
      return buttons[buttons.length - 1]
    }

    await confirmButton().trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Import result')
    expect(wrapper.text()).toContain('Total 1, imported 0, failed 1, skipped 1, duplicates 0.')

    await confirmButton().trigger('click')
    await waitForWorkbench()

    expect(batchImportMyAccounts).toHaveBeenCalledTimes(2)
    expect(showError).toHaveBeenCalledWith('Import failed')
    expect(showError).not.toHaveBeenCalledWith('retry import failed token=secret')
    expect(wrapper.text()).not.toContain('Import result')
    expect(wrapper.text()).not.toContain('Total 1, imported 0, failed 1, skipped 1, duplicates 0.')
  })

  it('disables account export while the CSV download request is in flight', async () => {
    listTemplates.mockResolvedValue([])
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    let resolveExport: (blob: Blob) => void = () => {}
    exportMyAccounts.mockReturnValue(new Promise<Blob>((resolve) => {
      resolveExport = resolve
    }))

    const wrapper = mountView()
    await waitForWorkbench()

    const exportButton = () => {
      const button = wrapper.findAll('button').find(node => node.text().includes('Export accounts') || node.text().includes('Processing'))
      expect(button).toBeTruthy()
      return button!
    }

    await exportButton().trigger('click')
    await waitForWorkbench()

    expect(exportMyAccounts).toHaveBeenCalledTimes(1)
    expect(exportButton().attributes('disabled')).toBeDefined()
    expect(exportButton().text()).toContain('Processing')

    await exportButton().trigger('click')
    await waitForWorkbench()
    expect(exportMyAccounts).toHaveBeenCalledTimes(1)

    resolveExport(new Blob(['platform,username\nx_twitter,northwind_ops\n'], { type: 'text/csv' }))
    await waitForWorkbench()

    expect(globalThis.URL.createObjectURL).toHaveBeenCalled()
    expect(anchorClick).toHaveBeenCalled()
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalled()
    expect(exportButton().attributes('disabled')).toBeUndefined()
    expect(exportButton().text()).toContain('Export accounts')
    anchorClick.mockRestore()
  })

  it('locks account export while the account list is refreshing', async () => {
    listTemplates.mockResolvedValue([])
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    let resolveRefresh!: (value: {
      items: Array<ReturnType<typeof buildAccount>>
      total: number
      page: number
      page_size: number
      pages: number
    }) => void
    listMyAccounts
      .mockResolvedValueOnce({
        items: [buildAccount()],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockReturnValue(new Promise((resolve) => {
        resolveRefresh = resolve
      }))

    const wrapper = mountView()
    await waitForWorkbench()

    const refreshButton = wrapper.findAll('button').find(node => node.text().includes('Refresh'))
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await wrapper.vm.$nextTick()

    const exportButton = wrapper.findAll('button').find(node => node.text().includes('Export accounts'))
    expect(exportButton).toBeTruthy()
    expect(exportButton!.attributes('disabled')).toBeDefined()

    await exportButton!.trigger('click')
    await waitForWorkbench()

    expect(exportMyAccounts).not.toHaveBeenCalled()
    expect(anchorClick).not.toHaveBeenCalled()

    resolveRefresh({
      items: [buildAccount()],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await waitForWorkbench()

    expect(exportButton!.attributes('disabled')).toBeUndefined()
    anchorClick.mockRestore()
  })

  it('exports selected accounts with the same backend filters used by the current list', async () => {
    listTemplates.mockResolvedValue([])
    listMyAccounts.mockResolvedValue({
      items: [buildAccount(777, {
        name: 'delivery_filter_ops',
        username: 'delivery_filter_ops',
        account_status: 'not_stored',
        default_proxy_configured: false,
      })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    exportMyAccounts.mockResolvedValue(new Blob(['platform,name\nx_twitter,@delivery_filter_ops\n'], { type: 'text/csv' }))
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)

    const wrapper = mountView()
    await waitForWorkbench()

    await wrapper.get('[data-testid="search-input-stub"]').setValue('#777')
    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('invalid')
    await selects[1].setValue('x_twitter')
    await new Promise(resolve => setTimeout(resolve, 300))
    await waitForWorkbench()

    await wrapper.get('[data-testid="account-row-777"] input[type="checkbox"]').setValue(true)
    await waitForWorkbench()

    const exportButton = wrapper.findAll('button').find(node => node.text().includes('Export accounts'))
    expect(exportButton).toBeTruthy()
    expect(exportButton!.attributes('aria-label')).toBe('Export selected accounts')
    expect(exportButton!.attributes('title')).toBe('Export selected accounts')
    await exportButton!.trigger('click')
    await waitForWorkbench()

    expect(exportMyAccounts).toHaveBeenCalledWith({
      search: '#777',
      platform: 'x_twitter',
      account_status: 'invalid',
      account_ids: [777],
    })
    expect(anchorClick).toHaveBeenCalled()
    anchorClick.mockRestore()
  })

  it('restores account export controls and does not download when the CSV request fails', async () => {
    listTemplates.mockResolvedValue([])
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const exportError = {
      response: {
        data: {
          detail: 'export service unavailable token=secret',
        },
      },
    }
    exportMyAccounts.mockRejectedValue(exportError)

    const wrapper = mountView()
    await waitForWorkbench()

    const exportButton = () => {
      const button = wrapper.findAll('button').find(node => node.text().includes('Export accounts') || node.text().includes('Processing'))
      expect(button).toBeTruthy()
      return button!
    }

    await exportButton().trigger('click')
    await waitForWorkbench()

    expect(exportMyAccounts).toHaveBeenCalledTimes(1)
    expect(recordClientDiagnostic).toHaveBeenCalledWith('account_workbench.unified.export_accounts', exportError)
    expect(showError).toHaveBeenCalledWith('Export failed')
    expect(showError).not.toHaveBeenCalledWith('export service unavailable token=secret')
    expect(globalThis.URL.createObjectURL).not.toHaveBeenCalled()
    expect(anchorClick).not.toHaveBeenCalled()
    expect(globalThis.URL.revokeObjectURL).not.toHaveBeenCalled()
    expect(exportButton().attributes('disabled')).toBeUndefined()
    expect(exportButton().text()).toContain('Export accounts')
    anchorClick.mockRestore()
  })

  it('previews stored post media in the selected-template summary and confirm dialog', async () => {
    listTemplates.mockResolvedValue([buildPostTemplate()])

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'post')

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
    await chooseExecutionAction(wrapper, 'post')

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
      action: 'post',
    })
    expect(JSON.stringify(submitTask.mock.calls[0]?.[0])).not.toContain('template_id')
    expect(typeof submitTask.mock.calls[0]?.[0]?.client_request_id).toBe('string')
    expect(String(submitTask.mock.calls[0]?.[0]?.client_request_id || '')).not.toBe('')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('keeps stored video post templates unavailable for execution', async () => {
    listTemplates.mockResolvedValue([buildVideoPostTemplate()])

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'post')

    expect(previewMedia).not.toHaveBeenCalledWith('social-task/42/post-video.mp4')
    expect(wrapper.find('[data-testid="selected-template-preview-post-0"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('1 media item(s)')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    expect(findExecutionStartButton(wrapper).attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Default template invalid.')

    await findExecutionStartButton(wrapper).trigger('click')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Confirm task execution')
    expect(submitTask).not.toHaveBeenCalled()
  })

  it('previews stored avatar and banner defaults, ignores stale non-default refs, and revokes blob urls on action switch', async () => {
    listTemplates.mockResolvedValue([
      {
        ...buildAvatarTemplate(),
        is_default: true,
      },
      {
        ...buildBannerTemplate(),
        is_default: true,
      },
      buildAvatarTemplate('media/avatar.png', 'stale-avatar-template'),
    ])

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'update_avatar')

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(wrapper.get('[data-testid="selected-template-preview-avatar"]').attributes('src')).toBe('blob:workbench-preview-1')

    const actionSelect = findActionSelect(wrapper)
    await actionSelect.setValue('update_banner')
    await waitForWorkbench()

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/banner.png')
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalledWith('blob:workbench-preview-1')
    expect(wrapper.get('[data-testid="selected-template-preview-banner"]').attributes('src')).toBe('blob:workbench-preview-2')

    await actionSelect.setValue('update_avatar')
    await waitForWorkbench()

    expect(previewMedia).not.toHaveBeenCalledWith('media/avatar.png')
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalledWith('blob:workbench-preview-2')
    expect(wrapper.get('[data-testid="selected-template-preview-avatar"]').attributes('src')).toBe('blob:workbench-preview-3')
    expect(wrapper.find('[data-testid="selected-template-preview-banner"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('media/avatar.png')
  })

  it('clears selected template media previews when the selected platform makes the template unavailable', async () => {
    listMyAccounts.mockResolvedValue({
      items: [
        buildAccount(101),
        buildAccount(202, {
          name: 'unsupported-main',
          username: 'unsupported-main',
          platform: 'instagram',
        }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    listTemplates.mockResolvedValue([{
      ...buildAvatarTemplate(),
      is_default: true,
    }])

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'update_avatar')

    expect(previewMedia).toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(wrapper.get('[data-testid="selected-template-preview-avatar"]').attributes('src')).toBe('blob:workbench-preview-1')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(2)
    await checkboxes[2].setValue(true)
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Platform unavailable.')
    expect(findExecutionStartButton(wrapper).attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="selected-template-preview-avatar"]').exists()).toBe(false)
    expect(globalThis.URL.revokeObjectURL).toHaveBeenCalledWith('blob:workbench-preview-1')
  })

  it('disables avatar and banner actions when default template image dimensions are invalid', async () => {
    listTemplates.mockResolvedValue([
      {
        ...buildAvatarTemplate('social-task/42/avatar.png', 'avatar-needs-normalization', 300, 300),
        is_default: true,
      },
      {
        ...buildBannerTemplate('social-task/42/banner.png', 'banner-needs-normalization', 1200, 500),
        is_default: true,
      },
    ])

    const wrapper = mountView()
    await waitForWorkbench()
    await chooseExecutionAction(wrapper, 'update_avatar')

    expect(previewMedia).not.toHaveBeenCalledWith('social-task/42/avatar.png')
    expect(previewMedia).not.toHaveBeenCalledWith('social-task/42/banner.png')
    expect(wrapper.find('[data-testid="selected-template-preview-avatar"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="selected-template-preview-banner"]').exists()).toBe(false)

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThan(1)
    await checkboxes[1].setValue(true)

    expect(findExecutionStartButton(wrapper).attributes('disabled')).toBeDefined()
    await waitForWorkbench()

    expect(wrapper.text()).toContain('Default template invalid.')
  })

  it('accepts opaque execution_auth values in the batch import preview', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('@encrypted_auth\tpw\ttotp\t\t\t\t\t\t\tct0=ok\tencrypted-execution-auth-ciphertext')
    await waitForWorkbench()

    expect(wrapper.text()).not.toContain('Execution auth format is invalid.')
    expect(wrapper.text()).toContain('Valid 1 / invalid 0')
    expect(wrapper.text()).toContain('Pending backend match')
    expect(wrapper.text()).not.toContain('Not stored')
  })

  it('labels valid batch import preview rows as pending backend match', async () => {
    listTemplates.mockResolvedValue([])

    const wrapper = mountView()
    await waitForWorkbench()

    const batchImportButton = wrapper.findAll('button').find(node => node.text().includes('Batch import'))
    expect(batchImportButton).toBeTruthy()
    await batchImportButton!.trigger('click')
    await waitForWorkbench()

    const batchHint = wrapper.find('[title="Paste text or choose a file."]')
    expect(batchHint.exists()).toBe(true)
    expect(batchHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    const previewScope = wrapper.find('[title="Preview is scoped to your workbench."]')
    expect(previewScope.exists()).toBe(true)
    expect(previewScope.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))

    await wrapper.get('textarea').setValue('@pending_match\tpw\tJBSWY3DPEHPK3PXP')
    await waitForWorkbench()

    const previewTitle = wrapper.find('[title="Import preview"]')
    expect(previewTitle.exists()).toBe(true)
    expect(previewTitle.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    const previewMeta = wrapper.find('[title="Valid 1 / invalid 0"]')
    expect(previewMeta.exists()).toBe(true)
    expect(previewMeta.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words', 'text-right']))
    expect(wrapper.text()).toContain('Pending backend match')
    const status = wrapper.findAll('div').find(node => node.text() === 'Pending backend match')
    expect(status).toBeTruthy()
    expect(status!.attributes('title')).toBe('Pending backend match')
    expect(status!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.text()).not.toContain('Not stored')
  })
})
