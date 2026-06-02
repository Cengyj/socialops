<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <nav class="flex flex-wrap gap-2 rounded-xl border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-700 dark:bg-dark-800" :aria-label="t('nav.socialAccounts')">
            <router-link to="/admin/accounts" :class="accountTabInactiveClass">
              <span>{{ t('admin.socialAccountWorkbench.tabs.management') }}</span>
              <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ t('admin.socialAccountWorkbench.tabs.managementCount', { count: managementTabCount }) }}
              </span>
            </router-link>
            <router-link to="/admin/total-accounts" :class="accountTabActiveClass">
              <span>{{ t('admin.socialAccountWorkbench.tabs.pool') }}</span>
              <span class="rounded-full bg-white/80 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-200">
                {{ t('admin.socialAccountWorkbench.tabs.poolCount', { count: poolTabCount }) }}
              </span>
            </router-link>
          </nav>

          <div class="grid gap-3 md:grid-cols-4">
            <div v-for="stat in stats" :key="stat.label" class="rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
            </div>
          </div>

          <div class="flex flex-col gap-3 rounded-xl border border-primary-100 bg-primary-50/60 p-3 dark:border-primary-900/40 dark:bg-primary-900/10 xl:flex-row xl:items-center xl:justify-between">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <SearchInput v-model="searchQuery" :placeholder="t('admin.socialAccountWorkbench.searchPlaceholder')" class="w-full sm:w-72" />
              <Select v-model="accountStatusFilter" :options="accountStatusOptions" class="w-full sm:w-44" />
              <Select v-model="assignmentFilter" :options="assignmentOptions" class="w-full sm:w-40" />
            </div>
            <div class="flex flex-wrap items-center justify-end gap-3">
              <span class="rounded-full bg-white px-3 py-1 text-sm font-medium text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white">
                {{ t('admin.socialAccountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}
              </span>
              <button class="btn btn-secondary" :disabled="selectedIds.length === 0" @click="clearSelection">
                {{ t('admin.socialAccountWorkbench.executionBar.clear') }}
              </button>
              <button class="btn btn-secondary" @click="triggerImport">
                <Icon name="upload" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.toolbar.importAccounts') }}
              </button>
              <button class="btn btn-secondary" @click="exportAccounts">
                <Icon name="download" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.toolbar.exportRecords') }}
              </button>
              <button class="btn btn-primary" :disabled="selectedIds.length === 0" @click="openAssignDialog">
                {{ t('admin.socialAccountWorkbench.actions.assign') }}
              </button>
              <button class="btn btn-secondary" :disabled="selectedIds.length === 0" @click="reclaimSelectedAccounts">
                {{ t('admin.socialAccountWorkbench.actions.reclaim') }}
              </button>
              <button class="btn btn-danger" :disabled="selectedIds.length === 0" @click="deleteDialogOpen = true">
                {{ t('admin.socialAccountWorkbench.actions.delete') }}
              </button>
              <input ref="importFileInput" type="file" accept=".csv,.json" class="hidden" @change="handleImportFile" />
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredAccounts" row-key="id" default-sort-key="id" default-sort-order="desc">
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allVisibleSelected"
              :indeterminate="someVisibleSelected"
              @click.stop
              @change="toggleAllVisible"
            />
          </template>
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isSelected(row.id)"
              @click.stop
              @change="toggleSelection(row.id)"
            />
          </template>
          <template #cell-account="{ row }">
            <button class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openDetailDialog(row)">{{ row.account }}</button>
          </template>
          <template #cell-password="{ value }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
          </template>
          <template #cell-emailPassword="{ value }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
          </template>
          <template #cell-boundIp="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ defaultProxyLabel(row.boundIp) }}</span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', accountStatusBadgeClass(String(value))]">{{ t(`admin.socialAccountWorkbench.accountStatus.${value}`) }}</span>
          </template>
          <template #cell-assignedUser="{ row }">
            <span :class="['badge', row.assignedUser ? 'badge-primary' : 'badge-warning']">
              {{ row.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <button class="btn btn-secondary px-2 py-1 text-xs" @click="openEditDialog(row)">{{ t('common.edit') }}</button>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="detailDialogOpen = false">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.tabs.poolDescription') }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div v-for="item in detailItems" :key="item.label" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <div class="text-gray-500 dark:text-gray-400">{{ item.label }}</div>
            <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
          </div>
        </div>
        <div v-if="selectedAccount.taskMessage" :class="['rounded-lg border p-3 text-sm', resultMessagePanelClass(selectedAccount.accountStatus)]">
          {{ selectedAccount.taskMessage }}
        </div>
      </div>
      <template #footer>
        <button class="btn btn-primary" @click="detailDialogOpen = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="editDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="editDialogOpen = false">
      <div class="grid gap-3 sm:grid-cols-2">
        <input v-model="accountForm.name" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.account')" />
        <input v-model="accountForm.accountId" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.accountId')" />
        <Select v-model="accountForm.accountStatus" :options="accountStatusOptionsWithoutAll" />
        <input v-model="accountForm.password" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.password')" />
        <input v-model="accountForm.phone" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.phone')" />
        <input v-model="accountForm.email" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.email')" />
        <input v-model="accountForm.emailPassword" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.emailPassword')" />
        <Select v-if="selectedAccountOwnerId" v-model="accountForm.defaultProxyId" :options="defaultProxyOptions" />
        <textarea v-model="accountForm.remark" class="input min-h-[88px] sm:col-span-2" :placeholder="t('admin.socialAccountWorkbench.form.remark')"></textarea>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="editDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!canSubmitAccount" @click="submitEditDialog">{{ t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="assignDialogOpen" :title="t('admin.socialAccountWorkbench.assignDialog.title')" width="wide" @close="assignDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.assignDialog.hint', { count: selectedAccounts.length }) }}
        </div>

        <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.socialAccountWorkbench.assignDialog.accountSummary') }}</label>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.assignment.unassigned') }}</span>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="account in selectedAccountPreview"
              :key="account.id"
              class="rounded-full border border-white bg-white px-3 py-1 text-xs font-medium text-gray-700 shadow-sm dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
            >
              {{ account.account }}
            </span>
            <span
              v-if="remainingSelectedAccountCount > 0"
              class="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300"
            >
              {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
            </span>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-[minmax(0,1.15fr)_minmax(260px,0.85fr)]">
          <div class="space-y-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.socialAccountWorkbench.assignDialog.targetUser') }}</label>
            <div class="rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
              <div class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-600 dark:bg-dark-700/60">
                <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
                <input
                  v-model="targetUserSearch"
                  type="text"
                  :placeholder="t('admin.socialAccountWorkbench.assignDialog.searchPlaceholder')"
                  class="flex-1 bg-transparent text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none dark:text-gray-100 dark:placeholder:text-dark-400"
                />
              </div>
              <div class="max-h-72 overflow-y-auto p-2">
                <div class="px-2 pb-2 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.socialAccountWorkbench.assignDialog.userListLabel') }}
                </div>
                <button
                  v-for="user in filteredTargetUsers"
                  :key="user.id"
                  type="button"
                  :class="[
                    'mb-1 w-full rounded-lg border px-3 py-2 text-left transition-colors last:mb-0',
                    targetUser === String(user.id)
                      ? 'border-primary-300 bg-primary-50 shadow-sm dark:border-primary-800/70 dark:bg-primary-900/20'
                      : 'border-transparent hover:bg-gray-50 dark:hover:bg-dark-700'
                  ]"
                  @click="targetUser = String(user.id)"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="font-medium text-gray-900 dark:text-white">{{ user.email }}</div>
                      <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">#{{ user.id }} · {{ user.role }}</div>
                    </div>
                    <span :class="['shrink-0 rounded-full px-2 py-0.5 text-xs font-medium', user.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300']">
                      {{ t(`admin.socialAccountWorkbench.assignDialog.userStatus.${user.status}`) }}
                    </span>
                  </div>
                  <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: assignedCountForUser(user.id) }) }}
                  </div>
                </button>
                <div v-if="filteredTargetUsers.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.socialAccountWorkbench.assignDialog.noUsersFound') }}
                </div>
              </div>
            </div>
          </div>

          <div class="rounded-xl border border-primary-100 bg-primary-50/60 p-4 dark:border-primary-900/40 dark:bg-primary-900/10">
            <div class="text-xs font-medium uppercase text-primary-700 dark:text-primary-300">
              {{ t('admin.socialAccountWorkbench.assignDialog.selectedUserLabel') }}
            </div>
            <div v-if="selectedTargetUser" class="mt-3 space-y-3">
              <div>
                <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ selectedTargetUser.email }}</div>
                <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">#{{ selectedTargetUser.id }} · {{ selectedTargetUser.role }}</div>
              </div>
              <div class="flex flex-wrap gap-2 text-xs">
                <span class="rounded-full bg-white px-2.5 py-1 font-medium text-gray-700 shadow-sm dark:bg-dark-700 dark:text-gray-200">{{ selectedTargetUser.status }}</span>
                <span class="rounded-full bg-white px-2.5 py-1 font-medium text-gray-700 shadow-sm dark:bg-dark-700 dark:text-gray-200">{{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: assignedCountForUser(selectedTargetUser.id) }) }}</span>
              </div>
            </div>
            <div v-else class="mt-3 rounded-lg border border-dashed border-primary-200 bg-white/70 p-4 text-sm text-gray-500 dark:border-primary-800/60 dark:bg-dark-800/60 dark:text-gray-400">
              {{ t('admin.socialAccountWorkbench.assignDialog.noSelectedUserPrompt') }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="assignDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!targetUser" @click="confirmAssignDialog">{{ t('admin.socialAccountWorkbench.assignDialog.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('admin.socialAccountWorkbench.deleteDialog.title')" width="normal" @close="deleteDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300">
          {{ t('admin.socialAccountWorkbench.deleteDialog.hint', { count: selectedIds.length }) }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div v-for="account in selectedAccounts" :key="account.id" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <div class="font-medium text-gray-900 dark:text-white">{{ account.account }}</div>
            <div class="mt-1 text-gray-500 dark:text-gray-400">{{ account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-danger" @click="confirmDeleteDialog">{{ t('admin.socialAccountWorkbench.deleteDialog.confirm') }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { AdminProxy, SocialAccount } from '@/api/admin'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'

type AccountStatus = 'pending_check' | 'available' | 'limited' | 'invalid' | 'not_stored'
type Source = 'registered' | 'manual_import' | 'file_upload'

interface AccountRow {
  id: number
  account: string
  platform: string
  accountId: string
  password: string
  phone: string
  email: string
  emailPassword: string
  boundIp: string
  accountStatus: AccountStatus
  taskStatus: string
  taskMessage: string
  source: Source
  assignedUserId: number | null
  assignedUser: string | null
  remark: string
  createdAt: string
}

const { t } = useI18n()
const appStore = useAppStore()

const searchQuery = ref('')
const accountStatusFilter = ref('all')
const assignmentFilter = ref('all')
const selectedIds = ref<number[]>([])
const accounts = ref<AccountRow[]>([])
const users = ref<AdminUser[]>([])
const proxies = ref<AdminProxy[]>([])
const selectedAccount = ref<AccountRow | null>(null)
const selectedAccountId = ref<number | null>(null)
const selectedAccountOwnerId = ref<number | null>(null)
const initialDefaultProxyId = ref('')
const detailDialogOpen = ref(false)
const editDialogOpen = ref(false)
const assignDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const targetUser = ref('')
const targetUserSearch = ref('')
const importFileInput = ref<HTMLInputElement | null>(null)

const accountForm = reactive({
  name: '',
  accountId: '',
  password: '',
  phone: '',
  email: '',
  emailPassword: '',
  accountStatus: 'pending_check',
  defaultProxyId: '',
  remark: '',
})

onMounted(async () => {
  await loadUsers()
  await Promise.all([loadAccounts(), loadProxies()])
})

watch(assignDialogOpen, (open) => {
  if (open) {
    targetUser.value = ''
    targetUserSearch.value = ''
  }
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', width: '48px' },
  { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), sortable: true },
  { key: 'account', label: t('admin.socialAccountWorkbench.columns.account'), sortable: true },
  { key: 'platform', label: t('admin.socialAccountWorkbench.columns.platform'), sortable: true },
  { key: 'password', label: t('admin.socialAccountWorkbench.columns.password') },
  { key: 'email', label: t('admin.socialAccountWorkbench.columns.email'), sortable: true },
  { key: 'emailPassword', label: t('admin.socialAccountWorkbench.columns.emailPassword') },
  { key: 'boundIp', label: t('admin.socialAccountWorkbench.columns.boundIp'), sortable: true },
  { key: 'accountStatus', label: t('admin.socialAccountWorkbench.columns.accountStatus'), sortable: true },
  { key: 'assignedUser', label: t('admin.socialAccountWorkbench.columns.assignedUser'), sortable: true },
  { key: 'actions', label: t('admin.socialAccountWorkbench.columns.actions') },
])

const accountStatusOptionsWithoutAll = computed(() => [
  { value: 'pending_check', label: t('admin.socialAccountWorkbench.accountStatus.pending_check') },
  { value: 'available', label: t('admin.socialAccountWorkbench.accountStatus.available') },
  { value: 'limited', label: t('admin.socialAccountWorkbench.accountStatus.limited') },
  { value: 'invalid', label: t('admin.socialAccountWorkbench.accountStatus.invalid') },
  { value: 'not_stored', label: t('admin.socialAccountWorkbench.accountStatus.not_stored') },
])

const accountStatusOptions = computed(() => [
  { value: 'all', label: t('admin.socialAccountWorkbench.filters.allAccountStatus') },
  ...accountStatusOptionsWithoutAll.value,
])

const assignmentOptions = computed(() => [
  { value: 'all', label: t('admin.socialAccountWorkbench.assignment.all') },
  { value: 'assigned', label: t('admin.socialAccountWorkbench.assignment.assigned') },
  { value: 'unassigned', label: t('admin.socialAccountWorkbench.assignment.unassigned') },
])

const filteredTargetUsers = computed(() => {
  const keyword = targetUserSearch.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter(user => [user.email, user.username, user.role].some(value => value.toLowerCase().includes(keyword)))
})

const filteredAccounts = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return accounts.value.filter(account => {
    const values = [account.account, account.platform, account.accountId, account.password, account.phone, account.email, account.emailPassword, account.boundIp, account.assignedUser ?? '', account.taskMessage, account.remark]
    const matchesKeyword = !keyword || values.some(value => value.toLowerCase().includes(keyword))
    const matchesStatus = accountStatusFilter.value === 'all' || account.accountStatus === accountStatusFilter.value
    const matchesAssignment = assignmentFilter.value === 'all' || (assignmentFilter.value === 'assigned' ? !!account.assignedUserId : !account.assignedUserId)
    return matchesKeyword && matchesStatus && matchesAssignment
  })
})

const stats = computed(() => [
  { label: t('admin.socialAccountWorkbench.stats.total'), value: accounts.value.length },
  { label: t('admin.socialAccountWorkbench.stats.available'), value: accounts.value.filter(account => account.accountStatus === 'available').length },
  { label: t('admin.socialAccountWorkbench.stats.assigned'), value: accounts.value.filter(account => account.assignedUserId).length },
  { label: t('admin.socialAccountWorkbench.stats.unassigned'), value: accounts.value.filter(account => !account.assignedUserId).length },
])

const selectedAccounts = computed(() => accounts.value.filter(account => selectedIds.value.includes(account.id)))
const selectedAccountPreview = computed(() => selectedAccounts.value.slice(0, 6))
const remainingSelectedAccountCount = computed(() => Math.max(0, selectedAccounts.value.length - selectedAccountPreview.value.length))
const selectedTargetUser = computed(() => users.value.find(user => String(user.id) === targetUser.value) ?? null)
const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)
const canSubmitAccount = computed(() => accountForm.name.trim() !== '')
const managementTabCount = computed(() => accounts.value.length)
const poolTabCount = computed(() => accounts.value.filter(account => !account.assignedUserId).length)
const accountTabActiveClass = 'flex min-h-[40px] items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-semibold text-white shadow-sm'
const accountTabInactiveClass = 'flex min-h-[40px] items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700'
const defaultProxyOptions = computed(() => {
  const ownerID = selectedAccountOwnerId.value
  const options = [{ value: '', label: t('admin.socialAccountWorkbench.defaultProxy.clear') }]
  if (!ownerID) return options
  return [
    ...options,
    ...proxies.value
      .filter(proxy => proxy.user_id === ownerID)
      .map(proxy => ({
        value: String(proxy.id),
        label: proxyLabel(proxy),
        disabled: proxy.status !== 'online',
      })),
  ]
})
const detailItems = computed(() => {
  if (!selectedAccount.value) return []
  return [
    { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
    { label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
    { label: t('admin.socialAccountWorkbench.columns.platform'), value: selectedAccount.value.platform },
    { label: t('admin.socialAccountWorkbench.columns.password'), value: selectedAccount.value.password },
    { label: t('admin.socialAccountWorkbench.columns.phone'), value: selectedAccount.value.phone },
    { label: t('admin.socialAccountWorkbench.columns.email'), value: selectedAccount.value.email },
    { label: t('admin.socialAccountWorkbench.columns.emailPassword'), value: selectedAccount.value.emailPassword },
    { label: t('admin.socialAccountWorkbench.columns.boundIp'), value: defaultProxyLabel(selectedAccount.value.boundIp) },
    { label: t('admin.socialAccountWorkbench.columns.assignedUser'), value: selectedAccount.value.assignedUser ?? t('admin.socialAccountWorkbench.assignment.unassigned') },
    { label: t('admin.socialAccountWorkbench.columns.createdAt'), value: selectedAccount.value.createdAt },
  ]
})

async function loadUsers() {
  try {
    const result = await adminAPI.users.list(1, 200, { status: 'active' })
    users.value = result.items ?? []
  } catch {
    users.value = []
  }
}

async function loadAccounts() {
  try {
    const result = await adminAPI.socialAccounts.list({ page: 1, page_size: 200 })
    accounts.value = (result.items ?? []).map(mapApiAccount)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.socialAccountWorkbench.failedToLoad'))
  }
}

async function loadProxies() {
  try {
    const result = await adminAPI.proxies.list({ page: 1, page_size: 200 })
    proxies.value = result.items ?? []
  } catch {
    proxies.value = []
  }
}

function mapApiAccount(account: SocialAccount): AccountRow {
  return {
    id: account.id,
    account: account.name,
    platform: account.platform,
    accountId: account.account_id ?? '',
    password: account.password ?? '',
    phone: account.phone ?? '',
    email: account.email ?? '',
    emailPassword: account.email_password ?? '',
    boundIp: account.bound_ip ?? '',
    accountStatus: toAccountStatus(account.account_status),
    taskStatus: account.task_status,
    taskMessage: account.task_message ?? '',
    source: toSource(account.source),
    assignedUserId: account.assigned_user_id ?? null,
    assignedUser: ownerLabel(account.assigned_user_id),
    remark: account.remark ?? '',
    createdAt: new Date(account.created_at).toLocaleString(),
  }
}

function ownerLabel(userID?: number | null): string | null {
  if (!userID) return null
  const user = users.value.find(item => item.id === userID)
  return user?.email ?? `#${userID}`
}

function isSelected(id: number): boolean {
  return selectedIds.value.includes(id)
}

function toggleSelection(id: number) {
  selectedIds.value = isSelected(id) ? selectedIds.value.filter(selectedId => selectedId !== id) : [...selectedIds.value, id]
}

function toggleAllVisible() {
  if (allVisibleSelected.value) {
    selectedIds.value = selectedIds.value.filter(id => !visibleIds.value.includes(id))
    return
  }
  selectedIds.value = Array.from(new Set([...selectedIds.value, ...visibleIds.value]))
}

function clearSelection() {
  selectedIds.value = []
}

function assignedCountForUser(userID: number): number {
  return accounts.value.filter(account => account.assignedUserId === userID).length
}

function openDetailDialog(row: AccountRow) {
  selectedAccount.value = row
  detailDialogOpen.value = true
}

function openEditDialog(row: AccountRow) {
  selectedAccountId.value = row.id
  selectedAccountOwnerId.value = row.assignedUserId
  accountForm.name = row.account
  accountForm.accountId = row.accountId
  accountForm.password = row.password
  accountForm.phone = row.phone
  accountForm.email = row.email
  accountForm.emailPassword = row.emailPassword
  accountForm.accountStatus = row.accountStatus
  accountForm.defaultProxyId = defaultProxyIdFromSnapshot(row.boundIp)
  initialDefaultProxyId.value = accountForm.defaultProxyId
  accountForm.remark = row.remark
  editDialogOpen.value = true
}

async function submitEditDialog() {
  if (!selectedAccountId.value || !canSubmitAccount.value) return
  try {
    await adminAPI.socialAccounts.update(selectedAccountId.value, {
      name: accountForm.name.trim(),
      account_id: accountForm.accountId || undefined,
      password: accountForm.password || undefined,
      phone: accountForm.phone || undefined,
      email: accountForm.email || undefined,
      email_password: accountForm.emailPassword || undefined,
      account_status: accountForm.accountStatus,
      remark: accountForm.remark || undefined,
    })
    if (accountForm.defaultProxyId !== initialDefaultProxyId.value) {
      const proxyId = accountForm.defaultProxyId ? Number(accountForm.defaultProxyId) : null
      await adminAPI.socialAccounts.setDefaultProxy(selectedAccountId.value, proxyId)
    }
    appStore.showSuccess(t('admin.socialAccountWorkbench.saved'))
    editDialogOpen.value = false
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function triggerImport() {
  importFileInput.value?.click()
}

async function handleImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const result = await adminAPI.socialAccounts.importAccounts(file)
    appStore.showSuccess(t('admin.socialAccountWorkbench.imported', { count: result.created }))
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    input.value = ''
  }
}

async function exportAccounts() {
  try {
    const blob = await adminAPI.socialAccounts.exportAccounts()
    downloadBlob(blob, 'social_account_pool.csv')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function openAssignDialog() {
  const assignedCount = selectedAccounts.value.filter(account => account.assignedUserId).length
  if (assignedCount > 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.assignRequiresUnassigned', { count: assignedCount }))
    return
  }
  assignDialogOpen.value = true
}

async function confirmAssignDialog() {
  const userIdNum = Number(targetUser.value)
  if (!Number.isFinite(userIdNum) || userIdNum <= 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.selectTargetUser'))
    return
  }
  try {
    for (const id of selectedIds.value) {
      await adminAPI.socialAccounts.assign(id, userIdNum)
    }
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.assigned', { count: selectedIds.value.length, user: selectedTargetUser.value?.email ?? `#${userIdNum}` }))
    assignDialogOpen.value = false
    clearSelection()
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

async function reclaimSelectedAccounts() {
  try {
    for (const id of selectedIds.value) {
      await adminAPI.socialAccounts.reclaim(id)
    }
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.reclaimed', { count: selectedIds.value.length }))
    clearSelection()
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

async function confirmDeleteDialog() {
  const deleteCount = selectedIds.value.length
  try {
    await adminAPI.socialAccounts.batchDelete(selectedIds.value)
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.deleted', { count: deleteCount }))
    deleteDialogOpen.value = false
    clearSelection()
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function accountStatusBadgeClass(status: string): string {
  if (status === 'available') return 'badge-success'
  if (status === 'pending_check') return 'badge-warning'
  if (status === 'limited') return 'badge-primary'
  return 'badge-danger'
}

function resultMessagePanelClass(status: string): string {
  if (status === 'available') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/60 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (status === 'invalid') return 'border-red-200 bg-red-50 text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300'
  return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300'
}

function proxyLabel(proxy: AdminProxy): string {
  const status = t(`admin.proxies.status.${proxy.status}`)
  return `${proxy.name} (#${proxy.id}, ${status})`
}

function defaultProxyIdFromSnapshot(snapshot: string): string {
  const parsed = parseDefaultProxySnapshot(snapshot)
  return parsed?.id ? String(parsed.id) : ''
}

function defaultProxyLabel(snapshot: string): string {
  const parsed = parseDefaultProxySnapshot(snapshot)
  if (!parsed?.id) return '-'
  const proxy = proxies.value.find(item => item.id === parsed.id)
  if (proxy) return proxyLabel(proxy)
  return parsed.name ? `${parsed.name} (#${parsed.id})` : `#${parsed.id}`
}

function parseDefaultProxySnapshot(snapshot: string): { id?: number; name?: string } | null {
  const trimmed = snapshot.trim()
  if (!trimmed || !trimmed.startsWith('{')) return null
  try {
    const parsed = JSON.parse(trimmed) as { id?: unknown; name?: unknown }
    const id = Number(parsed.id)
    return {
      id: Number.isFinite(id) && id > 0 ? id : undefined,
      name: typeof parsed.name === 'string' ? parsed.name : undefined,
    }
  } catch {
    return null
  }
}

function toAccountStatus(status: string): AccountStatus {
  if (status === 'pending_check' || status === 'available' || status === 'limited' || status === 'invalid' || status === 'not_stored') {
    return status
  }
  return 'not_stored'
}

function toSource(source: string): Source {
  if (source === 'registered' || source === 'manual_import' || source === 'file_upload') {
    return source
  }
  return 'manual_import'
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
</script>
