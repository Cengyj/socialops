<template>
  <DataTable :columns="columns" :data="proxies" :loading="loading" row-key="id" default-sort-key="updatedAt" default-sort-order="desc">
    <template #header-select>
      <input
        type="checkbox"
        class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
        :checked="allVisibleSelected"
        :disabled="loading || testing"
        :indeterminate="someVisibleSelected"
        @click.stop
        @change="emit('toggleAllVisible')"
      />
    </template>
    <template #cell-select="{ row }">
      <input
        type="checkbox"
        class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
        :checked="isSelected(row.id)"
        :disabled="loading || testing"
        @click.stop
        @change="emit('toggleSelection', row.id)"
      />
    </template>
    <template #cell-name="{ row }">
      <button type="button" class="inline-block max-w-full break-all text-right font-medium text-primary-600 hover:text-primary-700 disabled:cursor-not-allowed disabled:opacity-60 sm:break-normal dark:text-primary-400" :title="row.name" :disabled="loading || testing" @click="emit('edit', row)">
        {{ row.name }}
      </button>
    </template>
    <template #cell-type="{ value }">
      <span class="rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
        {{ proxyTypeLabel(String(value)) }}
      </span>
    </template>
    <template #cell-endpoint="{ value }">
      <span class="block min-w-0 break-all text-right sm:max-w-[360px] sm:truncate sm:break-normal" :title="String(value || '')">{{ value || '-' }}</span>
    </template>
    <template #cell-status="{ value }">
      <span :class="['badge', statusBadgeClass(String(value))]">{{ proxyStatusLabel(String(value)) }}</span>
    </template>
    <template #cell-latency="{ value }">
      <span>{{ value ? `${value}ms` : '-' }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="flex flex-wrap items-center justify-end gap-2">
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center px-2 py-1 text-xs"
          :aria-label="rowActionTestButtonTitle"
          :title="rowActionTestButtonTitle"
          :disabled="loading || testing"
          @click="emit('test', row.id)"
        >
          <span class="min-w-0 truncate">{{ t('proxies.test') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center px-2 py-1 text-xs"
          :aria-label="rowActionEditButtonTitle"
          :title="rowActionEditButtonTitle"
          :disabled="loading || testing"
          @click="emit('edit', row)"
        >
          <span class="min-w-0 truncate">{{ t('common.edit') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-danger min-w-0 max-w-full justify-center px-2 py-1 text-xs"
          :aria-label="rowActionDeleteButtonTitle"
          :title="rowActionDeleteButtonTitle"
          :disabled="loading || testing"
          @click="emit('delete', row)"
        >
          <span class="min-w-0 truncate">{{ t('common.delete') }}</span>
        </button>
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
        <button
          v-if="!hasActiveProxyFilters"
          type="button"
          class="btn btn-primary btn-sm mt-4 min-w-0 max-w-full justify-center"
          :aria-label="emptyCreateButtonTitle"
          :title="emptyCreateButtonTitle"
          :disabled="testing"
          @click="emit('create')"
        >
          <Icon name="plus" size="sm" />
          <span class="min-w-0 truncate">{{ t('proxies.addProxy') }}</span>
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
import {
  proxyCreateButtonTitle as buildCreateButtonTitle,
  proxyRowDeleteButtonTitle as buildRowDeleteButtonTitle,
  proxyRowEditButtonTitle as buildRowEditButtonTitle,
  proxyRowTestButtonTitle as buildRowTestButtonTitle,
} from '../proxyActionTitles'

const props = defineProps<{
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

const rowActionTestButtonTitle = computed(() => buildRowTestButtonTitle(t, {
  loading: props.loading,
  testing: props.testing,
}))
const rowActionEditButtonTitle = computed(() => buildRowEditButtonTitle(t, {
  loading: props.loading,
  testing: props.testing,
}))
const rowActionDeleteButtonTitle = computed(() => buildRowDeleteButtonTitle(t, {
  loading: props.loading,
  testing: props.testing,
}))
const emptyCreateButtonTitle = computed(() => buildCreateButtonTitle(t, { testing: props.testing }))

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
