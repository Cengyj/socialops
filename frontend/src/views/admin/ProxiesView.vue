<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="grid gap-3 md:grid-cols-4">
            <div v-for="stat in stats" :key="stat.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
            </div>
          </div>

          <div class="flex flex-col gap-3 rounded-lg border border-primary-100 bg-primary-50/60 p-3 dark:border-primary-900/40 dark:bg-primary-900/10 xl:flex-row xl:items-center xl:justify-between">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <SearchInput v-model="searchQuery" :placeholder="t('admin.proxies.searchPlaceholder')" class="w-full sm:w-64" />
              <Select v-model="statusFilter" :options="statusOptions" class="w-full sm:w-40" />
              <Select v-model="typeFilter" :options="typeOptions" class="w-full sm:w-44" />
            </div>
            <div class="flex flex-wrap items-center justify-end gap-3">
              <button class="btn btn-secondary" :disabled="selectedIds.length === 0" @click="testSelected">
                <Icon name="play" size="md" class="mr-2" />
                {{ t('admin.proxies.testSelected') }}
              </button>
              <button class="btn btn-primary" @click="openCreateDialog">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.proxies.addProxy') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredProxies" row-key="id" default-sort-key="id" default-sort-order="desc">
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
          <template #cell-name="{ row }">
            <button class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openEditDialog(row)">{{ row.name }}</button>
          </template>
          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(String(value))]">{{ t(`admin.proxies.status.${value}`) }}</span>
          </template>
          <template #cell-latency="{ value }">
            <span>{{ value ? `${value}ms` : '-' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="testProxy(row.id)">{{ t('admin.proxies.test') }}</button>
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="openEditDialog(row)">{{ t('common.edit') }}</button>
              <button class="btn btn-danger px-2 py-1 text-xs" @click="deleteProxy(row.id)">{{ t('common.delete') }}</button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="dialogOpen" :title="dialogTitle" width="wide" @close="dialogOpen = false">
      <div class="grid gap-3 sm:grid-cols-2">
        <Select v-model="form.userId" :options="userOptions" class="sm:col-span-2" :disabled="editingProxyId !== null" />
        <input v-model="form.name" type="text" class="input" :placeholder="t('admin.proxies.form.namePlaceholder')" />
        <Select v-model="form.ipType" :options="typeOptionsWithoutAll" />
        <input v-model="form.endpoint" type="text" class="input sm:col-span-2" :placeholder="t('admin.proxies.form.endpointPlaceholder')" />
        <textarea v-model="form.remark" class="input min-h-[90px] sm:col-span-2" :placeholder="t('admin.proxies.form.remarkPlaceholder')"></textarea>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="dialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!canSubmitForm" @click="submitForm">{{ t('common.confirm') }}</button>
      </template>
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
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { AdminProxy } from '@/api/admin'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'

interface ProxyRow {
  id: number
  userId: number
  owner: string
  name: string
  type: string
  endpoint: string
  status: string
  latency: number | null
  lastCheck: string
  remark: string
}

const { t } = useI18n()
const appStore = useAppStore()

const searchQuery = ref('')
const statusFilter = ref('all')
const typeFilter = ref('all')
const selectedIds = ref<number[]>([])
const dialogOpen = ref(false)
const editingProxyId = ref<number | null>(null)
const proxies = ref<ProxyRow[]>([])
const users = ref<AdminUser[]>([])
const form = reactive({
  userId: '',
  name: '',
  ipType: 'residential',
  endpoint: '',
  remark: '',
})

function mapProxy(proxy: AdminProxy): ProxyRow {
  const owner = users.value.find(user => user.id === proxy.user_id)
  return {
    id: proxy.id,
    userId: proxy.user_id,
    owner: owner?.email ?? `#${proxy.user_id}`,
    name: proxy.name,
    type: proxy.ip_type,
    endpoint: proxy.endpoint ?? '',
    status: proxy.status,
    latency: proxy.latency_ms ?? null,
    lastCheck: proxy.last_check_at ? new Date(proxy.last_check_at).toLocaleString() : '-',
    remark: proxy.remark ?? '',
  }
}

async function loadUsers() {
  try {
    const result = await adminAPI.users.list(1, 200, { status: 'active' })
    users.value = result.items ?? []
  } catch {
    users.value = []
  }
}

async function loadProxies() {
  try {
    const result = await adminAPI.proxies.list({ page: 1, page_size: 200 })
    proxies.value = (result.items ?? []).map(mapProxy)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.proxies.failedToLoad'))
  }
}

onMounted(async () => {
  await loadUsers()
  await loadProxies()
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', width: '48px' },
  { key: 'id', label: t('admin.proxies.columns.id'), sortable: true },
  { key: 'name', label: t('admin.proxies.columns.name'), sortable: true },
  { key: 'owner', label: t('admin.proxies.columns.owner'), sortable: true },
  { key: 'type', label: t('admin.proxies.columns.type'), sortable: true },
  { key: 'endpoint', label: t('admin.proxies.columns.endpoint') },
  { key: 'status', label: t('admin.proxies.columns.status'), sortable: true },
  { key: 'latency', label: t('admin.proxies.columns.latency'), sortable: true },
  { key: 'lastCheck', label: t('admin.proxies.columns.lastCheck'), sortable: true },
  { key: 'actions', label: t('admin.proxies.columns.actions') },
])

const statusOptions = computed(() => [
  { value: 'all', label: t('admin.proxies.filters.allStatus') },
  { value: 'unknown', label: t('admin.proxies.status.unknown') },
  { value: 'online', label: t('admin.proxies.status.online') },
  { value: 'offline', label: t('admin.proxies.status.offline') },
])

const typeOptionsWithoutAll = computed(() => [
  { value: 'residential', label: t('admin.proxies.types.residential') },
  { value: 'static', label: t('admin.proxies.types.static') },
  { value: 'mobile', label: t('admin.proxies.types.mobile') },
  { value: 'datacenter', label: t('admin.proxies.types.datacenter') },
])

const typeOptions = computed(() => [
  { value: 'all', label: t('admin.proxies.filters.allTypes') },
  ...typeOptionsWithoutAll.value,
])

const userOptions = computed(() => [
  { value: '', label: t('admin.proxies.form.userPlaceholder') },
  ...users.value.map(user => ({ value: String(user.id), label: `${user.email} (#${user.id})` })),
])

const filteredProxies = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return proxies.value.filter(proxy => {
    const matchesKeyword = !keyword || [proxy.name, proxy.owner, proxy.endpoint, proxy.remark].some(value => value.toLowerCase().includes(keyword))
    const matchesStatus = statusFilter.value === 'all' || proxy.status === statusFilter.value
    const matchesType = typeFilter.value === 'all' || proxy.type === typeFilter.value
    return matchesKeyword && matchesStatus && matchesType
  })
})

const stats = computed(() => [
  { label: t('admin.proxies.stats.total'), value: proxies.value.length },
  { label: t('admin.proxies.stats.online'), value: proxies.value.filter(proxy => proxy.status === 'online').length },
  { label: t('admin.proxies.stats.offline'), value: proxies.value.filter(proxy => proxy.status === 'offline').length },
  { label: t('admin.proxies.stats.unknown'), value: proxies.value.filter(proxy => proxy.status === 'unknown').length },
])

const visibleIds = computed(() => filteredProxies.value.map(proxy => proxy.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)
const dialogTitle = computed(() => editingProxyId.value ? t('admin.proxies.editTitle') : t('admin.proxies.addProxy'))
const canSubmitForm = computed(() => Number(form.userId) > 0 && form.name.trim() !== '')

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

function resetForm() {
  editingProxyId.value = null
  form.userId = ''
  form.name = ''
  form.ipType = 'residential'
  form.endpoint = ''
  form.remark = ''
}

function openCreateDialog() {
  resetForm()
  dialogOpen.value = true
}

function openEditDialog(row: ProxyRow) {
  editingProxyId.value = row.id
  form.userId = String(row.userId)
  form.name = row.name
  form.ipType = row.type
  form.endpoint = row.endpoint
  form.remark = row.remark
  dialogOpen.value = true
}

async function submitForm() {
  if (!canSubmitForm.value) return
  try {
    const payload = {
      name: form.name.trim(),
      ip_type: form.ipType,
      endpoint: form.endpoint.trim() || undefined,
      remark: form.remark.trim() || undefined,
    }
    if (editingProxyId.value) {
      await adminAPI.proxies.update(editingProxyId.value, payload)
      appStore.showSuccess(t('admin.proxies.saved'))
    } else {
      await adminAPI.proxies.create({ ...payload, user_id: Number(form.userId) })
      appStore.showSuccess(t('admin.proxies.created'))
    }
    dialogOpen.value = false
    await loadProxies()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

async function testProxy(id: number) {
  try {
    const result = await adminAPI.proxies.test(id)
    appStore.showSuccess(t('admin.proxies.testResult', { status: t(`admin.proxies.status.${result.status}`) }))
    await loadProxies()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

async function testSelected() {
  for (const id of selectedIds.value) {
    await testProxy(id)
  }
}

async function deleteProxy(id: number) {
  try {
    await adminAPI.proxies.delete(id)
    selectedIds.value = selectedIds.value.filter(selectedId => selectedId !== id)
    appStore.showSuccess(t('admin.proxies.deleted'))
    await loadProxies()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function statusBadgeClass(status: string): string {
  if (status === 'online') return 'badge-success'
  if (status === 'offline') return 'badge-danger'
  return 'badge-warning'
}
</script>
