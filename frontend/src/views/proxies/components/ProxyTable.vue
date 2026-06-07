<template>
  <DataTable :columns="columns" :data="proxies" :loading="loading" row-key="id" default-sort-key="updatedAt" default-sort-order="desc">
    <template #header-select>
      <input
        type="checkbox"
        class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="allVisibleSelected"
        :indeterminate="someVisibleSelected"
        @click.stop
        @change="emit('toggleAllVisible')"
      />
    </template>
    <template #cell-select="{ row }">
      <input
        type="checkbox"
        class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="isSelected(row.id)"
        @click.stop
        @change="emit('toggleSelection', row.id)"
      />
    </template>
    <template #cell-name="{ row }">
      <button type="button" class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="emit('edit', row)">
        {{ row.name }}
      </button>
    </template>
    <template #cell-type="{ value }">
      <span class="rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
        {{ proxyTypeLabel(String(value)) }}
      </span>
    </template>
    <template #cell-endpoint="{ value }">
      <span class="block max-w-[360px] truncate" :title="String(value || '')">{{ value || '-' }}</span>
    </template>
    <template #cell-status="{ value }">
      <span :class="['badge', statusBadgeClass(String(value))]">{{ proxyStatusLabel(String(value)) }}</span>
    </template>
    <template #cell-latency="{ value }">
      <span>{{ value ? `${value}ms` : '-' }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="flex flex-wrap items-center justify-end gap-2">
        <button type="button" class="btn btn-secondary px-2 py-1 text-xs" :disabled="testing" @click="emit('test', row.id)">{{ t('proxies.test') }}</button>
        <button type="button" class="btn btn-secondary px-2 py-1 text-xs" @click="emit('edit', row)">{{ t('common.edit') }}</button>
        <button type="button" class="btn btn-danger px-2 py-1 text-xs" @click="emit('delete', row)">{{ t('common.delete') }}</button>
      </div>
    </template>
    <template #empty>
      <div class="flex flex-col items-center py-8 text-center">
        <Icon name="server" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
        <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
          {{ hasActiveProxyFilters ? t('proxies.noResults.title') : t('proxies.empty.title') }}
        </p>
        <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
          {{ hasActiveProxyFilters ? t('proxies.noResults.description') : t('proxies.empty.description') }}
        </p>
        <button v-if="!hasActiveProxyFilters" type="button" class="btn btn-primary btn-sm mt-4" @click="emit('create')">
          <Icon name="plus" size="sm" />
          <span>{{ t('proxies.addProxy') }}</span>
        </button>
      </div>
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import type { ProxyRow } from '../useProxyManagement'

defineProps<{
  allVisibleSelected: boolean
  hasActiveProxyFilters: boolean
  isSelected: (id: number) => boolean
  loading: boolean
  proxies: ProxyRow[]
  proxyStatusLabel: (status: string) => string
  proxyTypeLabel: (type: string) => string
  someVisibleSelected: boolean
  statusBadgeClass: (status: string) => string
  testing: boolean
}>()

const emit = defineEmits<{
  create: []
  delete: [row: ProxyRow]
  edit: [row: ProxyRow]
  test: [id: number]
  toggleAllVisible: []
  toggleSelection: [id: number]
}>()

const { t } = useI18n()

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px]' },
  { key: 'name', label: t('proxies.columns.name'), sortable: true, class: 'min-w-[180px]' },
  { key: 'type', label: t('proxies.columns.type'), sortable: true, class: 'min-w-[128px]' },
  { key: 'endpoint', label: t('proxies.columns.endpoint'), class: 'min-w-[260px]' },
  { key: 'status', label: t('proxies.columns.status'), sortable: true, class: 'min-w-[120px]' },
  { key: 'latency', label: t('proxies.columns.latency'), sortable: true, class: 'min-w-[100px]' },
  { key: 'lastCheck', label: t('proxies.columns.lastCheck'), sortable: true, class: 'min-w-[180px]' },
  { key: 'actions', label: t('common.actions'), class: 'w-[180px] min-w-[180px]' },
])
</script>
