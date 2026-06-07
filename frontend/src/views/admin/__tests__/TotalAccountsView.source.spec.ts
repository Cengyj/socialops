import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const sourcePath = resolve(__dirname, '../TotalAccountsView.vue')
const source = readFileSync(sourcePath, 'utf8')

describe('admin TotalAccountsView total account pool contract', () => {
  it('does not duplicate the AppHeader page title inside the content area', () => {
    expect(source).not.toContain("<h1 class=\"text-2xl font-bold text-gray-900 dark:text-white\">{{ t('admin.totalAccounts.title') }}</h1>")
    expect(source).not.toContain("{{ t('admin.totalAccounts.description') }}")
  })

  it('keeps total account pool governance actions visible in a unified toolbar', () => {
    expect(source).toContain('data-testid="total-accounts-toolbar"')
    expect(source).toContain("t('admin.socialAccountWorkbench.actions.assign')")
    expect(source).toContain("t('admin.socialAccountWorkbench.actions.reclaim')")
    expect(source).toContain("t('admin.socialAccountWorkbench.actions.delete')")
    expect(source).toContain(':disabled="!canAssignSelected || assigning"')
    expect(source).toContain(':disabled="!hasSelection || reclaiming"')
    expect(source).toContain(':disabled="!hasSelection || deleting"')
    expect(source).toContain('@click="openAssignDialog"')
    expect(source).toContain('@click="openReclaimDialog"')
    expect(source).toContain('@click="openDeleteDialog"')
    expect(source).toContain('const assignConfirmDialogOpen = ref(false)')
    expect(source).toContain('const reclaimDialogOpen = ref(false)')
    expect(source).not.toContain('v-if="selectedIds.length > 0" class="flex flex-col gap-3 rounded-lg')
  })

  it('keeps the total account table visually continuous without folded sticky edge columns', () => {
    expect(source).toContain(':sticky-first-column="false"')
    expect(source).toContain(':sticky-actions-column="true"')
    expect(source).toContain('class="total-accounts-table"')
    expect(source).toContain('.total-accounts-table :deep(.sticky-col-right::before)')
    expect(source).toContain("class: 'w-[56px] min-w-[56px] text-center'")
    expect(source).toContain("class: 'w-[128px] min-w-[128px]'")
  })

  it('keeps assign, reclaim, and delete behind final confirmation dialogs', () => {
    const openAssignConfirmSource = source.slice(
      source.indexOf('function openAssignConfirmDialog'),
      source.indexOf('async function confirmAssignDialog'),
    )
    const openReclaimSource = source.slice(
      source.indexOf('function openReclaimDialog'),
      source.indexOf('async function reclaimSelectedAccounts'),
    )
    const openDeleteSource = source.slice(
      source.indexOf('function openDeleteDialog'),
      source.indexOf('async function confirmDeleteDialog'),
    )

    expect(source).toContain('@click="openAssignConfirmDialog"')
    expect(source).toContain('@click="confirmAssignDialog"')
    expect(source).toContain('@click="reclaimSelectedAccounts"')
    expect(source).toContain('@click="confirmDeleteDialog"')
    expect(source).toContain("t('admin.socialAccountWorkbench.assignDialog.confirmTitle')")
    expect(source).toContain("t('admin.socialAccountWorkbench.reclaimDialog.title')")
    expect(source).toContain("t('admin.socialAccountWorkbench.deleteDialog.impactHint')")
    expect(openAssignConfirmSource).not.toContain('adminAPI.totalAccounts.assign')
    expect(openReclaimSource).not.toContain('adminAPI.totalAccounts.reclaim')
    expect(openDeleteSource).not.toContain('adminAPI.accountWorkbench.batchDelete')
  })

  it('preserves existing total account pool API paths and action restrictions', () => {
    expect(source).toContain('adminAPI.totalAccounts.list')
    expect(source).toContain('adminAPI.totalAccounts.batchAssign')
    expect(source).toContain('adminAPI.totalAccounts.batchReclaim')
    expect(source).toContain('adminAPI.totalAccounts.batchDelete')
    expect(source).toContain('adminAPI.accountWorkbench.importAccounts')
    expect(source).toContain('adminAPI.accountWorkbench.exportAccounts')
    expect(source).toContain('adminAPI.accountWorkbench.update')
    expect(source).toContain('assignRequiresUnassigned')
    expect(source).not.toContain('adminAPI.accountWorkbench.list')
  })

  it('does not expose account intake source in the total account pool', () => {
    expect(source).not.toContain('type Source')
    expect(source).not.toContain('source:')
    expect(source).not.toContain('toSource')
    expect(source).not.toContain('account.source')
    expect(source).not.toContain('columns.source')
    expect(source).not.toContain('sources.')
  })

  it('renders account delivery fields without masking in table cells and account details', () => {
    expect(source).toContain('{{ row.password || \'-\' }}')
    expect(source).toContain('{{ row.emailPassword || \'-\' }}')
    expect(source).toContain('{{ row.authCookie || \'-\' }}')
    expect(source).toContain('{{ row.executionAuth || \'-\' }}')
    expect(source).toContain('value: selectedAccount.value.password')
    expect(source).toContain('value: selectedAccount.value.emailPassword')
    expect(source).toContain('value: selectedAccount.value.authCookie')
    expect(source).toContain('value: selectedAccount.value.executionAuth')
    expect(source).toContain('value: selectedAccount.value.defaultProxySnapshot')
    expect(source).toContain('value: selectedAccount.value.remark')
    expect(source).toContain("item.value || '-'")
    expect(source).not.toContain('maskedCredentialLabel(row.password)')
    expect(source).not.toContain('maskedCredentialLabel(row.emailPassword)')
    expect(source).not.toContain('maskedCredentialLabel(selectedAccount.value.password)')
    expect(source).not.toContain('maskedCredentialLabel(selectedAccount.value.emailPassword)')
    expect(source).not.toContain('defaultProxyLabel(row.defaultProxySnapshot)')
    expect(source).not.toContain('defaultProxyLabel(selectedAccount.value.defaultProxySnapshot)')
  })

  it('keeps raw account delivery fields editable and searchable from the list', () => {
    const filteredAccountsSource = source.slice(
      source.indexOf('const filteredAccounts = computed'),
      source.indexOf('const stats = computed'),
    )

    expect(source).toContain('accountForm.password = row.password')
    expect(source).toContain('accountForm.emailPassword = row.emailPassword')
    expect(source).toContain('accountForm.authCookie = row.authCookie')
    expect(source).toContain('accountForm.executionAuth = row.executionAuth')
    expect(filteredAccountsSource).toContain('account.password')
    expect(filteredAccountsSource).toContain('account.emailPassword')
    expect(filteredAccountsSource).toContain('account.authCookie')
    expect(filteredAccountsSource).toContain('account.executionAuth')
    expect(source).toContain('authCookie: account.auth_cookie ??')
  })

  it('submits empty delivery field edits instead of omitting them', () => {
    const submitSource = source.slice(
      source.indexOf('async function submitEditDialog'),
      source.indexOf('function triggerImport'),
    )

    expect(submitSource).toContain('password: accountForm.password')
    expect(submitSource).toContain('phone: accountForm.phone')
    expect(submitSource).toContain('email: accountForm.email')
    expect(submitSource).toContain('email_password: accountForm.emailPassword')
    expect(submitSource).toContain('auth_cookie: accountForm.authCookie')
    expect(submitSource).toContain('execution_auth: accountForm.executionAuth')
    expect(submitSource).toContain('remark: accountForm.remark')
    expect(submitSource).not.toContain('name:')
    expect(submitSource).not.toContain('platform_user_id:')
    expect(submitSource).not.toContain('registration_ip:')
    expect(submitSource).not.toContain('accountForm.platformUserId || undefined')
    expect(submitSource).not.toContain('accountForm.password || undefined')
    expect(submitSource).not.toContain('accountForm.phone || undefined')
    expect(submitSource).not.toContain('accountForm.email || undefined')
    expect(submitSource).not.toContain('accountForm.emailPassword || undefined')
    expect(submitSource).not.toContain('accountForm.authCookie || undefined')
    expect(submitSource).not.toContain('accountForm.executionAuth || undefined')
    expect(submitSource).not.toContain('accountForm.remark || undefined')
  })

  it('keeps identity and registration IP read-only in total account edits', () => {
    const editDialogSource = source.slice(
      source.indexOf('<BaseDialog :show="editDialogOpen"'),
      source.indexOf('<BaseDialog :show="assignDialogOpen"'),
    )

    expect(editDialogSource).toContain('data-testid="total-account-edit-identity"')
    expect(editDialogSource).toContain('data-testid="total-account-edit-form"')
    expect(editDialogSource).toContain('editIdentityItems')
    expect(editDialogSource).toContain("t('accountWorkbench.edit.identityHint')")
    expect(editDialogSource).toContain("t('accountWorkbench.detailSections.credentials')")
    expect(editDialogSource).toContain("t('accountWorkbench.detailSections.operations')")
    expect(editDialogSource).toContain('for="total-account-edit-password"')
    expect(editDialogSource).toContain('for="total-account-edit-auth-cookie"')
    expect(editDialogSource).toContain('for="total-account-edit-remark"')
    expect(editDialogSource).not.toContain('v-model="accountForm.name"')
    expect(editDialogSource).not.toContain('v-model="accountForm.platformUserId"')
    expect(editDialogSource).not.toContain('v-model="accountForm.registrationIp"')
  })
})
