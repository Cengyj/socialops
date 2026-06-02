<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput v-model="userSearch" :placeholder="t('admin.subscriptions.searchUsers')" class="w-full sm:w-72" @input="searchUsers" />
            <select v-model="filters.status" class="input w-full sm:w-40" @change="loadSubscriptions">
              <option value="">{{ t('admin.subscriptions.allStatus') }}</option>
              <option value="active">{{ t('admin.subscriptions.status.active') }}</option>
              <option value="expired">{{ t('admin.subscriptions.status.expired') }}</option>
              <option value="revoked">{{ t('admin.subscriptions.status.revoked') }}</option>
            </select>
            <input v-model.number="filters.group_id" class="input w-full sm:w-40" type="number" min="1" :placeholder="t('admin.subscriptions.form.group')" @change="loadSubscriptions" />
          </div>
          <div class="flex items-center gap-2">
            <button class="btn btn-secondary" :disabled="loading" @click="loadSubscriptions">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-secondary" @click="openBulkAssign">
              {{ t('admin.subscriptions.bulkAssign') }}
            </button>
            <button class="btn btn-primary" @click="openAssign">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.subscriptions.assignSubscription') }}
            </button>
          </div>
        </div>
        <div v-if="userResults.length > 0" class="mt-3 flex flex-wrap gap-2">
          <button v-for="user in userResults" :key="user.id" class="badge badge-primary" @click="selectFilterUser(user)">
            {{ user.email || user.username }} #{{ user.id }}
          </button>
          <button v-if="filters.user_id" class="badge badge-secondary" @click="clearFilterUser">{{ t('common.clear') }}</button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="subscriptions" :loading="loading" row-key="id">
          <template #cell-user="{ row }">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ row.user?.email || `#${row.user_id}` }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.user?.username || '-' }}</div>
            </div>
          </template>
          <template #cell-group="{ row }">
            <span class="badge badge-primary">{{ row.group?.name || `#${row.group_id}` }}</span>
          </template>
          <template #cell-usage="{ row }">
            <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
              <div>{{ t('admin.subscriptions.daily') }}: ${{ row.daily_usage_usd.toFixed(2) }} / {{ limitText(row.group?.daily_limit_usd) }}</div>
              <div>{{ t('admin.subscriptions.weekly') }}: ${{ row.weekly_usage_usd.toFixed(2) }} / {{ limitText(row.group?.weekly_limit_usd) }}</div>
              <div>{{ t('admin.subscriptions.monthly') }}: ${{ row.monthly_usage_usd.toFixed(2) }} / {{ limitText(row.group?.monthly_limit_usd) }}</div>
            </div>
          </template>
          <template #cell-expires_at="{ value }">
            <span>{{ value ? formatDate(String(value)) : t('admin.subscriptions.noExpiration') }}</span>
          </template>
          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(String(value))]">{{ t(`admin.subscriptions.status.${value}`) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="openProgress(row)">{{ t('admin.subscriptions.progress') }}</button>
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="openExtend(row)">{{ t('admin.subscriptions.adjust') }}</button>
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="resetQuota(row)">{{ t('admin.subscriptions.resetQuota') }}</button>
              <button class="btn btn-danger px-2 py-1 text-xs" @click="revoke(row)">{{ t('admin.subscriptions.revoke') }}</button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="assignDialogOpen" :title="bulkMode ? t('admin.subscriptions.bulkAssign') : t('admin.subscriptions.assignSubscription')" width="wide" @close="assignDialogOpen = false">
      <div class="grid gap-3 sm:grid-cols-2">
        <textarea v-if="bulkMode" v-model="assignForm.userIds" class="input min-h-[100px] sm:col-span-2" :placeholder="t('admin.subscriptions.bulkUserIds')"></textarea>
        <input v-else v-model.number="assignForm.userId" class="input" type="number" min="1" :placeholder="t('admin.subscriptions.form.user')" />
        <input v-model.number="assignForm.groupId" class="input" type="number" min="1" :placeholder="t('admin.subscriptions.form.group')" />
        <input v-model.number="assignForm.validityDays" class="input" type="number" min="1" :placeholder="t('admin.subscriptions.form.validityDays')" />
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="assignDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="assigning" @click="submitAssign">{{ assigning ? t('admin.subscriptions.assigning') : t('admin.subscriptions.assign') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="extendDialogOpen" :title="t('admin.subscriptions.adjustSubscription')" @close="extendDialogOpen = false">
      <div class="space-y-3">
        <div class="text-sm text-gray-600 dark:text-gray-300">{{ selectedSubscription?.user?.email || `#${selectedSubscription?.user_id}` }}</div>
        <input v-model.number="extendDays" class="input" type="number" :placeholder="t('admin.subscriptions.form.adjustDays')" />
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="extendDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" @click="submitExtend">{{ t('admin.subscriptions.adjust') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="progressDialogOpen" :title="t('admin.subscriptions.progress')" @close="progressDialogOpen = false">
      <div v-if="progress" class="space-y-3 text-sm">
        <div v-for="key in ['daily', 'weekly', 'monthly']" :key="key" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="font-medium text-gray-900 dark:text-white">{{ t(`admin.subscriptions.${key}`) }}</div>
          <div class="mt-1 text-gray-600 dark:text-gray-300">{{ progressLine(progress[key as 'daily' | 'weekly' | 'monthly']) }}</div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { AdminUser, SubscriptionProgress, UserSubscription } from '@/types'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const assigning = ref(false)
const subscriptions = ref<UserSubscription[]>([])
const userSearch = ref('')
const userResults = ref<AdminUser[]>([])
const assignDialogOpen = ref(false)
const extendDialogOpen = ref(false)
const progressDialogOpen = ref(false)
const bulkMode = ref(false)
const selectedSubscription = ref<UserSubscription | null>(null)
const extendDays = ref(30)
const progress = ref<SubscriptionProgress | null>(null)

const filters = reactive({
  status: '' as '' | 'active' | 'expired' | 'revoked',
  user_id: undefined as number | undefined,
  group_id: undefined as number | undefined,
})

const assignForm = reactive({
  userId: undefined as number | undefined,
  userIds: '',
  groupId: undefined as number | undefined,
  validityDays: 30,
})

const columns = computed<Column[]>(() => [
  { key: 'id', label: 'ID', sortable: true },
  { key: 'user', label: t('admin.subscriptions.columns.user') },
  { key: 'group', label: t('admin.subscriptions.columns.group') },
  { key: 'usage', label: t('admin.subscriptions.columns.usage') },
  { key: 'expires_at', label: t('admin.subscriptions.columns.expires'), sortable: true },
  { key: 'status', label: t('admin.subscriptions.columns.status') },
  { key: 'actions', label: t('common.actions') },
])

async function loadSubscriptions() {
  loading.value = true
  try {
    const result = await adminAPI.subscriptions.list(1, 100, {
      status: filters.status || undefined,
      user_id: filters.user_id,
      group_id: filters.group_id,
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    subscriptions.value = result.items
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function searchUsers() {
  const keyword = userSearch.value.trim()
  if (keyword.length < 2) {
    userResults.value = []
    return
  }
  const result = await adminAPI.users.list(1, 8, { search: keyword })
  userResults.value = result.items
}

function selectFilterUser(user: AdminUser) {
  filters.user_id = user.id
  userSearch.value = `${user.email || user.username} #${user.id}`
  userResults.value = []
  void loadSubscriptions()
}

function clearFilterUser() {
  filters.user_id = undefined
  userSearch.value = ''
  void loadSubscriptions()
}

function openAssign() {
  bulkMode.value = false
  assignDialogOpen.value = true
}

function openBulkAssign() {
  bulkMode.value = true
  assignDialogOpen.value = true
}

async function submitAssign() {
  assigning.value = true
  try {
    if (bulkMode.value) {
      const ids = assignForm.userIds.split(/[,\s]+/).map(Number).filter(id => id > 0)
      const result = await adminAPI.subscriptions.bulkAssign({ user_ids: ids, group_id: Number(assignForm.groupId), validity_days: assignForm.validityDays })
      appStore.showSuccess(t('admin.subscriptions.bulkAssigned', { count: result.success_count }))
    } else {
      await adminAPI.subscriptions.assign({ user_id: Number(assignForm.userId), group_id: Number(assignForm.groupId), validity_days: assignForm.validityDays })
      appStore.showSuccess(t('admin.subscriptions.subscriptionAssigned'))
    }
    assignDialogOpen.value = false
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToAssign'))
  } finally {
    assigning.value = false
  }
}

function openExtend(row: UserSubscription) {
  selectedSubscription.value = row
  extendDays.value = 30
  extendDialogOpen.value = true
}

async function submitExtend() {
  if (!selectedSubscription.value) return
  try {
    await adminAPI.subscriptions.extend(selectedSubscription.value.id, { days: extendDays.value })
    appStore.showSuccess(t('admin.subscriptions.subscriptionAdjusted'))
    extendDialogOpen.value = false
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToAdjust'))
  }
}

async function resetQuota(row: UserSubscription) {
  try {
    await adminAPI.subscriptions.resetQuota(row.id, { daily: true, weekly: true, monthly: true })
    appStore.showSuccess(t('admin.subscriptions.quotaResetSuccess'))
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToResetQuota'))
  }
}

async function revoke(row: UserSubscription) {
  if (!window.confirm(t('admin.subscriptions.revokeConfirm', { user: row.user?.email || `#${row.user_id}` }))) return
  try {
    await adminAPI.subscriptions.revoke(row.id)
    appStore.showSuccess(t('admin.subscriptions.subscriptionRevoked'))
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToRevoke'))
  }
}

async function openProgress(row: UserSubscription) {
  selectedSubscription.value = row
  progress.value = await adminAPI.subscriptions.getProgress(row.id)
  progressDialogOpen.value = true
}

function statusBadgeClass(status: string): string {
  if (status === 'active') return 'badge-success'
  if (status === 'expired') return 'badge-warning'
  return 'badge-danger'
}

function limitText(value?: number | null): string {
  return value == null ? t('admin.subscriptions.unlimited') : `$${value.toFixed(2)}`
}

function progressLine(item: SubscriptionProgress['daily']): string {
  if (!item) return t('admin.subscriptions.windowNotActive')
  const limit = item.limit == null ? t('admin.subscriptions.unlimited') : `$${item.limit.toFixed(2)}`
  return `$${item.used.toFixed(2)} / ${limit} (${item.percentage.toFixed(1)}%)`
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString()
}

onMounted(loadSubscriptions)
</script>
