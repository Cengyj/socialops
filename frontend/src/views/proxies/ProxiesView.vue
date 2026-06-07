<template>
  <div class="contents">
    <AppLayout>
      <TablePageLayout>
        <template #filters>
          <div class="space-y-4">
            <div v-if="loadError" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-950/30">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('proxies.failedToLoad') }}</p>
                  <p class="mt-1 text-sm text-red-600 dark:text-red-300/80">{{ loadError }}</p>
                </div>
                <button type="button" class="btn btn-secondary shrink-0" @click="loadProxies">{{ t('common.retry') }}</button>
              </div>
            </div>

            <ProxyStatsGrid :stats="stats" />
            <ProxyToolbar
              v-model:search-query="searchQuery"
              v-model:status-filter="statusFilter"
              v-model:type-filter="typeFilter"
              :has-proxies="proxies.length > 0"
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
            <ProxyTestResultsPanel
              :preview-rows="testResultPreviewRows"
              :proxy-name-by-id="proxyNameById"
              :proxy-status-label="proxyStatusLabel"
              :results="lastTestResults"
              :row-tone-class="proxyTestRowToneClass"
              :summary="testResultSummary"
              @clear="clearTestResults"
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
import ProxyTestResultsPanel from './components/ProxyTestResultsPanel.vue'
import ProxyTable from './components/ProxyTable.vue'
import { useProxyManagement } from './useProxyManagement'

const { t } = useI18n()

const {
  allVisibleSelected,
  closeProxyDeleteDialog,
  closeProxyFormDialog,
  clearTestResults,
  editingProxy,
  handleProxyDeleted,
  handleProxySaved,
  hasActiveProxyFilters,
  isSelected,
  lastTestResults,
  loadError,
  loading,
  loadProxies,
  openCreateDialog,
  openDeleteDialog,
  openEditDialog,
  proxies,
  proxyDeleteDialogOpen,
  proxyFormDialogOpen,
  proxyNameById,
  proxyStatusLabel,
  proxyTestRowToneClass,
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
  testResultPreviewRows,
  testResultSummary,
  testSelected,
  testing,
  toggleAllVisible,
  toggleSelection,
  typeFilter,
  typeOptions,
} = useProxyManagement({ t })

void loadProxies()
</script>
