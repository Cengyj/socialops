<template>
  <div class="contents">
    <AppLayout>
      <TablePageLayout>
        <template #filters>
          <div class="space-y-4">
            <LoadErrorBanner
              v-if="loadError"
              :title="t('proxies.failedToLoad')"
              :message="loadError"
              :retry-label="t('common.retry')"
              @retry="loadProxies"
            />

            <ProxyStatsGrid :stats="stats" />
            <ProxyToolbar
              v-model:search-query="searchQuery"
              v-model:status-filter="statusFilter"
              v-model:type-filter="typeFilter"
              :has-proxies="hasAnyProxy"
              :loading="loading"
              :selected-count="selectedIds.length"
              :status-options="statusOptions"
              :testing="testing"
              :type-options="typeOptions"
              @create="openCreateDialog"
              @refresh="loadProxies"
              @test-all="testAll"
              @test-selected="testSelected"
            />
          </div>
        </template>

        <template #table>
          <ProxyTable
            :all-visible-selected="allVisibleSelected"
            :has-active-proxy-filters="hasActiveProxyFilters"
            :is-selected="isSelected"
            :loading="loading"
            :proxies="proxies"
            :proxy-status-label="proxyStatusLabel"
            :proxy-type-label="proxyTypeLabel"
            :some-visible-selected="someVisibleSelected"
            :status-badge-class="statusBadgeClass"
            :testing="testing"
            @create="openCreateDialog"
            @delete="openDeleteDialog"
            @edit="openEditDialog"
            @test="testProxy"
            @toggle-all-visible="toggleAllVisible"
            @toggle-selection="toggleSelection"
          />
        </template>
      </TablePageLayout>
    </AppLayout>

    <ProxyDialogs
      :show-form="proxyFormDialogOpen"
      :show-delete="proxyDeleteDialogOpen"
      :form-locked="loading || testing"
      :delete-locked="loading || testing"
      :editing-proxy="editingProxy"
      :proxy-to-delete="proxyToDelete"
      @close-form="closeProxyFormDialog"
      @close-delete="closeProxyDeleteDialog"
      @saved="handleProxySaved"
      @deleted="handleProxyDeleted"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import ProxyDialogs from './ProxyDialogs.vue'
import ProxyStatsGrid from './components/ProxyStatsGrid.vue'
import ProxyToolbar from './components/ProxyToolbar.vue'
import ProxyTable from './components/ProxyTable.vue'
import LoadErrorBanner from '@/components/common/LoadErrorBanner.vue'
import { useProxyManagement } from './useProxyManagement'

const { t } = useI18n()

const {
  allVisibleSelected,
  closeProxyDeleteDialog,
  closeProxyFormDialog,
  editingProxy,
  handleProxyDeleted,
  handleProxySaved,
  hasActiveProxyFilters,
  hasAnyProxy,
  isSelected,
  loadError,
  loading,
  loadProxies,
  openCreateDialog,
  openDeleteDialog,
  openEditDialog,
  proxies,
  proxyDeleteDialogOpen,
  proxyFormDialogOpen,
  proxyStatusLabel,
  proxyToDelete,
  proxyTypeLabel,
  searchQuery,
  selectedIds,
  someVisibleSelected,
  stats,
  statusBadgeClass,
  statusFilter,
  statusOptions,
  testAll,
  testProxy,
  testSelected,
  testing,
  toggleAllVisible,
  toggleSelection,
  typeFilter,
  typeOptions,
} = useProxyManagement({ t })

void loadProxies()
</script>
