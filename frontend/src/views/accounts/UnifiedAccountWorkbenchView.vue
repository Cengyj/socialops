<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div v-if="loadError" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-950/30">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-start gap-3">
                <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0 text-red-500" />
                <div>
                  <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('accountWorkbench.failedToLoad') }}</p>
                  <p class="mt-1 text-sm text-red-600 dark:text-red-300/80">{{ loadError }}</p>
                </div>
              </div>
              <button type="button" class="btn btn-secondary shrink-0" @click="loadData">{{ t('common.retry') }}</button>
            </div>
          </div>
          <div v-if="dependencyLoadError && !loadError" class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-start gap-3">
                <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0 text-amber-500" />
                <div>
                  <p class="font-medium">{{ t('accountWorkbench.dependencyLoadWarning') }}</p>
                  <p class="mt-1 text-amber-600 dark:text-amber-300/80">{{ dependencyLoadError }}</p>
                </div>
              </div>
              <button type="button" class="btn btn-secondary shrink-0" @click="loadData">{{ t('common.retry') }}</button>
            </div>
          </div>

          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="stat in statCards"
              :key="stat.label"
              class="rounded-lg border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
                  <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ stat.meta }}</div>
                </div>
                <div class="shrink-0 text-lg font-semibold leading-6 text-gray-900 dark:text-white">{{ stat.value }}</div>
              </div>
            </div>
          </div>

          <div data-testid="accounts-toolbar" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
            <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-2 xl:flex-row xl:items-center">
                <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2 xl:flex-nowrap">
                  <SearchInput v-model="searchQuery" :placeholder="t('accountWorkbench.searchPlaceholder')" class="w-full shrink-0 sm:w-[220px] xl:w-[220px] 2xl:w-[280px]" />
                  <Select v-model="statusFilter" :options="statusFilterOptions" class="w-full shrink-0 sm:w-[144px] xl:w-[136px] 2xl:w-[148px]" />
                  <Select v-model="platformFilter" :options="platformFilterOptions" class="w-full shrink-0 sm:w-[144px] xl:w-[136px] 2xl:w-[148px]" />
                  <div class="hidden h-6 w-px shrink-0 bg-gray-200 dark:bg-dark-700 xl:block"></div>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('common.refresh')"
                    :title="t('common.refresh')"
                    :disabled="loading"
                    @click="loadData"
                  >
                    <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                    <span class="xl:hidden 2xl:inline">{{ t('common.refresh') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('accountWorkbench.import.batchAction')"
                    :title="t('accountWorkbench.import.batchAction')"
                    @click="openBatchImportDialog"
                  >
                    <Icon name="upload" size="sm" />
                    <span class="xl:hidden 2xl:inline">{{ t('accountWorkbench.import.batchAction') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('accountWorkbench.exportAccounts')"
                    :title="t('accountWorkbench.exportAccounts')"
                    @click="exportAccounts"
                  >
                    <Icon name="download" size="sm" />
                    <span class="xl:hidden 2xl:inline">{{ t('accountWorkbench.exportAccounts') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('accountWorkbench.proxy.batchAction')"
                    :title="t('accountWorkbench.proxy.batchAction')"
                    :disabled="selectedIds.length === 0"
                    @click="openBatchProxyDialog"
                  >
                    <Icon name="server" size="sm" />
                    <span class="xl:hidden 2xl:inline">{{ t('accountWorkbench.proxy.batchAction') }}</span>
                  </button>
                  <div class="flex h-10 w-full shrink-0 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800 sm:w-auto xl:ml-1">
                    <div class="flex min-w-[116px] flex-1 items-center justify-center whitespace-nowrap bg-primary-50 px-3 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 sm:flex-none xl:min-w-[88px] xl:px-2 2xl:min-w-[116px] 2xl:px-3">
                      {{ t('accountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}
                    </div>
                    <button
                      type="button"
                      class="flex h-full w-10 shrink-0 items-center justify-center border-l border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-gray-100"
                      :aria-label="t('admin.socialAccountWorkbench.executionBar.clear')"
                      :title="t('admin.socialAccountWorkbench.executionBar.clear')"
                      :disabled="selectedIds.length === 0"
                      @click="clearSelection"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                    <button
                      type="button"
                      class="flex h-full w-10 shrink-0 items-center justify-center border-l border-gray-200 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-red-300 dark:hover:bg-red-950/30 dark:hover:text-red-200"
                      :aria-label="t('accountWorkbench.deleteSelected')"
                      :title="t('accountWorkbench.deleteSelected')"
                      :disabled="selectedIds.length === 0 || deleting"
                      @click="openBatchDeleteDialog"
                    >
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </div>

                <div class="flex w-full shrink-0 flex-col gap-2 sm:flex-row xl:ml-auto xl:w-auto xl:items-center">
                  <Select
                    v-model="selectedTemplateId"
                    :options="executionTemplateOptions"
                    class="w-full shrink-0 sm:flex-1 xl:w-[220px] xl:flex-none 2xl:w-[300px]"
                    :placeholder="t('accountWorkbench.execution.templatePlaceholder')"
                    :empty-text="t('accountWorkbench.execution.noTemplates')"
                  />
                  <button type="button" class="btn btn-primary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4" :disabled="!canStartExecution || submitting" @click="openExecutionConfirmDialog">
                    <Icon name="play" size="sm" />
                    <span>{{ submitting ? t('common.processing') : t('accountWorkbench.execution.start') }}</span>
                  </button>
                </div>
              </div>

              <div
                v-if="selectedIds.length > 0 || selectedTemplate || executionDisabledReason || lastTaskResult || proxyAssignmentResult"
                class="grid gap-2 border-t border-gray-100 pt-3 dark:border-dark-700 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"
              >
                <div v-if="selectedTemplate" class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm dark:border-dark-700 dark:bg-dark-700/60">
                  <div class="font-medium text-gray-900 dark:text-white">{{ selectedTemplate.name }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ templateSummary(selectedTemplate) }}</div>
                  <div v-if="selectedTemplatePreviewCards.length > 0" class="mt-3 flex flex-wrap gap-2">
                    <div
                      v-for="card in selectedTemplatePreviewCards"
                      :key="card.key"
                      class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
                    >
                      <video
                        v-if="card.kind === 'video'"
                        :data-testid="card.summaryTestId"
                        :src="card.src"
                        :aria-label="card.alt"
                        class="h-20 w-20 object-cover"
                        controls
                        playsinline
                        muted
                      />
                      <img
                        v-else
                        :data-testid="card.summaryTestId"
                        :src="card.src"
                        :alt="card.alt"
                        class="h-20 w-20 object-cover"
                      />
                    </div>
                  </div>
                </div>
                <div
                  v-if="selectedIds.length > 0 && executionDisabledReason"
                  class="min-w-0 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
                  :class="selectedTemplate ? '' : 'lg:col-span-2'"
                >
                  {{ executionDisabledReason }}
                </div>
                <div v-if="lastTaskResult" :class="['rounded-lg border px-3 py-2 text-sm', taskResultPanelToneClass]">
                  <div class="font-medium">{{ t('accountWorkbench.execution.resultSummary', { submitted: lastTaskResult.submitted, enqueued: lastTaskResult.enqueued, failed: lastTaskResult.failed_closed || 0 }) }}</div>
                  <div v-if="hasFailedTaskNoChargeSummary" class="mt-1 text-xs font-medium">
                    {{ t('accountWorkbench.execution.failureNoChargeSummary') }}
                  </div>
                  <div v-if="taskResultPreviewLogs.length > 0" class="mt-2 space-y-1.5">
                    <div class="text-xs font-medium uppercase tracking-wide">{{ t('accountWorkbench.execution.resultRows') }}</div>
                    <div
                      v-for="log in taskResultPreviewLogs"
                      :key="log.id"
                      :class="['rounded-md px-2 py-1.5 text-xs', taskResultRowToneClass(log)]"
                    >
                      <div class="flex min-w-0 flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                        <span class="min-w-0 truncate font-medium">{{ log.account_name || `#${log.social_account_id}` }}</span>
                        <span class="shrink-0">{{ taskResultStatusLabel(log.status) }} · {{ taskResultChargeLabel(log.charge_status) }} · {{ formatChargeAmount(log.charged_amount) }}</span>
                      </div>
                      <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
                        <span
                          class="min-w-0 max-w-full truncate rounded bg-white/60 px-1.5 py-0.5 font-medium dark:bg-dark-700/70"
                          :title="taskResultSummary(log)"
                        >
                          {{ taskResultSummaryMeta(log) }}
                        </span>
                      </div>
                      <div v-if="taskResultMessage(log) !== '-'" class="mt-1 line-clamp-2 opacity-80">{{ taskResultMessage(log) }}</div>
                    </div>
                    <div v-if="remainingTaskResultLogCount > 0" class="text-xs font-medium">
                      {{ t('accountWorkbench.execution.resultRowsMore', { count: remainingTaskResultLogCount }) }}
                    </div>
                  </div>
                </div>
                <div v-if="proxyAssignmentResult" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300">
                  {{ t('accountWorkbench.proxy.resultSummary', { total: proxyAssignmentResult.total, succeeded: proxyAssignmentResult.succeeded, failed: proxyAssignmentResult.failed, skipped: proxyAssignmentResult.skipped }) }}
                </div>
              </div>

            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="accountColumns" :data="filteredAccounts" :loading="loading" row-key="id" default-sort-key="updatedAt" default-sort-order="desc">
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
            <button type="button" class="flex min-w-[210px] items-center gap-3 text-left" @click="openDetailDialog(row)">
              <span :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border text-xs font-semibold', platformAvatarClass(row.platform)]">
                {{ platformInitial(row.platform) }}
              </span>
              <span class="min-w-0">
                <span class="block truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">#{{ row.id }} · {{ row.username || '-' }}</span>
              </span>
            </button>
          </template>
          <template #cell-platform="{ value }">
            <span class="rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
              {{ platformLabel(String(value || '')) }}
            </span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', accountStatusBadgeClass(String(value))]">{{ accountStatusLabel(String(value || '')) }}</span>
          </template>
          <template #cell-taskStatus="{ value }">
            <span :class="['badge', taskStatusBadgeClass(String(value))]">{{ taskStatusLabel(String(value || '')) }}</span>
          </template>
          <template #cell-proxy="{ row }">
            <span :class="['badge', row.defaultProxyConfigured ? 'badge-success' : 'badge-warning']">
              {{ row.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-2">
              <button type="button" class="btn btn-secondary px-2 py-1 text-xs" @click="openProxyDialog(row)">
                {{ t('accountWorkbench.proxy.action') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary h-8 w-8 px-0"
                :aria-label="t('common.edit')"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                type="button"
                class="btn btn-secondary px-2 py-1 text-xs text-red-600 hover:border-red-200 hover:bg-red-50 dark:text-red-300 dark:hover:border-red-900/60 dark:hover:bg-red-950/30"
                :title="t('accountWorkbench.deleteOne')"
                :disabled="deleting"
                @click="deleteAccount(row)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
          <template #empty>
            <div class="flex flex-col items-center py-8 text-center">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ accounts.length === 0 ? t('accountWorkbench.empty.title') : t('accountWorkbench.noResults.title') }}
              </p>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
                {{ accounts.length === 0 ? t('accountWorkbench.empty.description') : t('accountWorkbench.noResults.description') }}
              </p>
              <div v-if="accounts.length === 0" class="mt-4 flex flex-wrap justify-center gap-2">
                <button type="button" class="btn btn-primary btn-sm" @click="openBatchImportDialog">
                  <Icon name="upload" size="sm" />
                  <span>{{ t('accountWorkbench.import.batchAction') }}</span>
                </button>
              </div>
              <button v-else type="button" class="btn btn-secondary btn-sm mt-4" @click="clearAccountFilters">
                <Icon name="x" size="sm" />
                <span>{{ t('accountWorkbench.filters.clear') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="detailDialogOpen = false">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('accountWorkbench.sections.managementHint') }}
        </div>
        <div class="space-y-3">
          <div v-for="section in detailSections" :key="section.title" class="rounded-lg border border-gray-200 bg-white p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ section.title }}</div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div v-for="item in section.items" :key="item.label" class="rounded-md bg-gray-50 p-3 dark:bg-dark-700">
                <div class="text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                <div class="mt-1 whitespace-pre-wrap break-all font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="selectedAccount.taskMessage" :class="['rounded-lg border p-3 text-sm', taskMessagePanelClass(selectedAccount.taskStatus)]">
          {{ selectedAccount.taskMessage }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-primary" @click="detailDialogOpen = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="editDialogOpen" :title="t('accountWorkbench.edit.title')" width="wide" @close="closeEditDialog">
      <div v-if="editAccount" class="space-y-4">
        <div data-testid="account-edit-identity" class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-700/60">
          <div class="mb-2 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.edit.identityTitle') }}</div>
          <div class="grid gap-2 sm:grid-cols-3">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.id') }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ editAccount.id }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.columns.platform') }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ platformLabel(editAccount.platform) }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.columns.name') }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ editAccount.name }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.columns.platformUserId') }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ editAccount.platformUserId || '-' }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.registrationIp') }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ editAccount.registrationIp || '-' }}</div>
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.edit.identityHint') }}</p>
        </div>

        <div data-testid="account-edit-form" class="space-y-3">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.credentials') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-password">{{ t('admin.socialAccountWorkbench.form.password') }}</label>
              <input id="account-edit-password" v-model="editAccountForm.password" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-phone">{{ t('admin.socialAccountWorkbench.form.phone') }}</label>
              <input id="account-edit-phone" v-model="editAccountForm.phone" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email">{{ t('admin.socialAccountWorkbench.form.email') }}</label>
              <input id="account-edit-email" v-model="editAccountForm.email" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-password">{{ t('admin.socialAccountWorkbench.form.emailPassword') }}</label>
              <input id="account-edit-email-password" v-model="editAccountForm.emailPassword" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-two-factor">{{ t('admin.socialAccountWorkbench.form.twoFactor') }}</label>
              <input id="account-edit-two-factor" v-model="editAccountForm.twoFactor" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-backup-code">{{ t('admin.socialAccountWorkbench.form.backupCode') }}</label>
              <input id="account-edit-backup-code" v-model="editAccountForm.backupCode" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-client-id">{{ t('admin.socialAccountWorkbench.form.emailClientId') }}</label>
              <input id="account-edit-email-client-id" v-model="editAccountForm.emailClientId" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-token">{{ t('admin.socialAccountWorkbench.form.emailToken') }}</label>
              <input id="account-edit-email-token" v-model="editAccountForm.emailToken" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-auth-cookie">{{ t('admin.socialAccountWorkbench.form.authCookie') }}</label>
              <textarea id="account-edit-auth-cookie" v-model="editAccountForm.authCookie" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800"></textarea>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-execution-auth">{{ t('admin.socialAccountWorkbench.form.executionAuth') }}</label>
              <textarea id="account-edit-execution-auth" v-model="editAccountForm.executionAuth" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800"></textarea>
            </div>
          </div>

          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.operations') }}</div>
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-remark">{{ t('admin.socialAccountWorkbench.form.remark') }}</label>
            <textarea id="account-edit-remark" v-model="editAccountForm.remark" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800"></textarea>
          </div>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="savingAccountEdit" @click="closeEditDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="savingAccountEdit || !canSaveAccountEdit" @click="saveAccountEdit">
          {{ savingAccountEdit ? t('common.saving') : t('common.save') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="batchImportDialogOpen" :title="t('accountWorkbench.import.batchTitle')" width="wide" @close="batchImportDialogOpen = false">
      <div data-testid="accounts-batch-import-dialog" class="space-y-4">
        <p class="rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('accountWorkbench.import.batchHint') }}
        </p>
        <div class="grid gap-2 sm:grid-cols-[180px_minmax(0,1fr)] sm:items-center">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-200" for="accounts-batch-import-platform">{{ t('accountWorkbench.import.defaultPlatform') }}</label>
          <Select id="accounts-batch-import-platform" v-model="importForm.platform" :options="importPlatformOptions" />
        </div>
        <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
          <textarea v-model="batchImportText" class="input min-h-[180px]" :placeholder="t('accountWorkbench.import.batchPlaceholder')" @input="clearBatchImportFileSource"></textarea>
          <div class="space-y-3 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700/60">
            <label class="block text-xs font-medium uppercase text-gray-500 dark:text-gray-400" for="accounts-batch-import-file">{{ t('accountWorkbench.import.fileLabel') }}</label>
            <input
              id="accounts-batch-import-file"
              ref="batchImportFileInput"
              type="file"
              class="block w-full text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-white file:px-3 file:py-2 file:text-sm file:font-medium file:text-gray-700 hover:file:bg-gray-100 dark:text-gray-300 dark:file:bg-dark-800 dark:file:text-gray-100 dark:hover:file:bg-dark-600"
              accept=".txt,.xls,.xlsx,text/plain,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
              :disabled="parsingImportFile || importing"
              @change="handleBatchImportFileChange"
            />
            <div class="min-h-[20px] truncate text-xs text-gray-500 dark:text-gray-400">{{ batchImportFileName || t('accountWorkbench.import.fileEmpty') }}</div>
            <button type="button" class="btn btn-secondary btn-sm w-full justify-center" :disabled="(!batchImportText && !batchImportFileName) || parsingImportFile || importing" @click="clearBatchImportSource">
              <Icon name="x" size="sm" />
              <span>{{ t('accountWorkbench.import.clearSource') }}</span>
            </button>
          </div>
        </div>
        <div v-if="batchImportError" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          {{ batchImportError }}
        </div>
        <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
          {{ t('accountWorkbench.import.previewScopeUser') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-4">
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.import.submitReady') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ canBatchImportAccounts ? t('common.yes') : t('common.no') }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.import.pendingCount') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ batchImportAccountsInput.length }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.columns.platform') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ importForm.platform || 'x_twitter' }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.import.invalidCount') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ batchImportInvalidCount }}</div>
          </div>
        </div>
        <div v-if="activeBatchImportRows.length > 0" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
          <div class="flex items-center justify-between gap-3 border-b border-gray-200 bg-gray-50 px-3 py-2 text-xs font-medium uppercase text-gray-500 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-400">
            <span>{{ t('accountWorkbench.import.previewTitle') }}</span>
            <span>{{ t('accountWorkbench.import.previewMeta', { valid: activeBatchImportValidRows.length, invalid: batchImportInvalidCount }) }}</span>
          </div>
          <div class="max-h-[220px] divide-y divide-gray-100 overflow-auto dark:divide-dark-700">
            <div v-for="row in batchImportPreviewRows" :key="row.rowNumber" class="grid gap-2 px-3 py-2 text-sm md:grid-cols-[64px_minmax(0,1fr)_120px_minmax(0,1.4fr)]">
              <div class="text-xs text-gray-500 dark:text-gray-400">#{{ row.rowNumber }}</div>
              <div class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ row.account.name || '-' }}</div>
              <div :class="['text-xs font-medium', batchImportRowStatusClass(row)]">
                {{ batchImportRowStatusLabel(row) }}
              </div>
              <div class="min-w-0 truncate text-xs text-gray-500 dark:text-gray-400">{{ row.error || credentialSummary(row.account) }}</div>
            </div>
          </div>
        </div>
        <div v-if="batchImportResult" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300">
          <div class="font-medium">{{ t('accountWorkbench.import.resultTitle') }}</div>
          <div class="mt-1">
            {{ t('accountWorkbench.import.resultSummary', {
              total: batchImportResult.total,
              succeeded: batchImportResult.succeeded,
              imported: batchImportResult.imported,
              failed: batchImportResult.failed,
              skipped: batchImportResult.skipped,
              duplicates: batchImportResult.duplicates,
            }) }}
          </div>
          <div v-if="batchImportResultPreviewItems.length > 0" class="mt-2 space-y-1.5">
            <div
              v-for="item in batchImportResultPreviewItems"
              :key="`${item.id || item.name || 'row'}-${item.status}-${item.reason || item.error || ''}`"
              :class="['rounded-md px-2 py-1.5 text-xs', batchImportResultRowToneClass(item.status)]"
            >
              <div class="flex min-w-0 flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                <span class="min-w-0 truncate font-medium">{{ item.name || (item.id ? `#${item.id}` : '-') }}</span>
                <span class="shrink-0">{{ batchImportResultStatusLabel(item.status) }}</span>
              </div>
              <div class="mt-1 line-clamp-2 opacity-80">{{ item.reason || item.error || '-' }}</div>
            </div>
            <div v-if="remainingBatchImportResultItemCount > 0" class="text-xs font-medium">
              {{ t('accountWorkbench.import.resultRowsMore', { count: remainingBatchImportResultItemCount }) }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="batchImportDialogOpen = false">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="!canBatchImportAccounts || importing || parsingImportFile" @click="batchImportAccounts">
          {{ importing || parsingImportFile ? t('common.processing') : t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="proxyDialogOpen" :title="proxyDialogTitle" width="wide" @close="closeProxyDialog">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{
            proxyDialogMode === 'single' && proxyAccount
              ? t('accountWorkbench.proxy.currentAccount', { name: proxyAccount.name })
              : t('accountWorkbench.proxy.batchHint', { count: selectedAccounts.length })
          }}
        </div>
        <div class="grid gap-3 sm:grid-cols-3">
          <button
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'specific' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            @click="proxyAssignmentMode = 'specific'"
          >
            <span class="block font-medium">{{ t('accountWorkbench.proxy.modeSpecific') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.modeSpecificHint') }}</span>
          </button>
          <button
            v-if="proxyDialogMode === 'batch'"
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'random' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            :disabled="usableProxies.length === 0"
            @click="proxyAssignmentMode = 'random'"
          >
            <span class="block font-medium">{{ t('accountWorkbench.proxy.modeRandom') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.modeRandomHint') }}</span>
          </button>
          <button
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'clear' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            @click="proxyAssignmentMode = 'clear'"
          >
            <span class="block font-medium">{{ t('accountWorkbench.proxy.modeClear') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.modeClearHint') }}</span>
          </button>
        </div>
        <div v-if="proxyAssignmentMode === 'specific'" class="space-y-2">
          <Select
            v-model="selectedProxyId"
            :options="usableProxyOptions"
            :placeholder="t('accountWorkbench.proxy.selectPlaceholder')"
            :empty-text="t('accountWorkbench.proxy.noOnlineProxies')"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.hint') }}</p>
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.summaryTotal') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ proxyAssignmentAccountIds.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.summaryMode') }}</div>
            <div class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{{ proxyAssignmentModeLabel }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.summaryOnline') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ usableProxies.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.summarySelected') }}</div>
            <div class="mt-2 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ selectedProxyLabel }}</div>
          </div>
        </div>
        <div v-if="proxyAssignmentDisabledReason" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
          {{ proxyAssignmentDisabledReason }}
        </div>
        <div v-if="proxyAssignmentResult" class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('accountWorkbench.proxy.resultTitle') }}</div>
          <div class="mb-3 rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300">
            {{ t('accountWorkbench.proxy.resultSummary', { total: proxyAssignmentResult.total, succeeded: proxyAssignmentResult.succeeded, failed: proxyAssignmentResult.failed, skipped: proxyAssignmentResult.skipped }) }}
          </div>
          <div class="grid gap-2 text-sm">
            <div
              v-for="(item, index) in proxyAssignmentResultPreviewItems"
              :key="`${item.id || index}-${item.status}-${item.reason || item.error || ''}`"
              :class="['flex min-w-0 flex-col gap-1 rounded-lg px-3 py-2 sm:flex-row sm:items-center sm:justify-between', proxyAssignmentResultRowToneClass(item.status)]"
            >
              <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ item.name || `#${item.id || index + 1}` }}</span>
              <span class="shrink-0 text-xs">{{ proxyAssignmentResultStatusLabel(item.status) }} · {{ item.reason || item.error || '-' }}</span>
            </div>
            <div v-if="remainingProxyAssignmentResultItemCount > 0" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('accountWorkbench.proxy.resultRowsMore', { count: remainingProxyAssignmentResultItemCount }) }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="savingProxy" @click="closeProxyDialog">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="!canConfirmProxyAssignment || savingProxy" @click="confirmProxyAssignment">
          {{ savingProxy ? t('common.saving') : t('accountWorkbench.proxy.apply') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('accountWorkbench.deleteDialog.title')" width="normal" @close="closeDeleteDialog">
      <div class="space-y-4">
        <div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300">
          {{
            deleteDialogMode === 'single' && deleteTargetAccount
              ? t('accountWorkbench.deleteDialog.singleHint', { name: deleteTargetAccount.name })
              : t('accountWorkbench.deleteDialog.batchHint', { count: deleteDialogAccounts.length })
          }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-3">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.selected') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ deleteDialogAccounts.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.executable') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ deleteDialogExecutableCount }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.abnormal') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ deleteDialogAbnormalCount }}</div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('accountWorkbench.deleteDialog.accountSummary') }}</div>
          <div class="grid gap-2 text-sm">
            <div v-for="account in deleteDialogAccountPreview" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ account.name }}</span>
              <span class="shrink-0 truncate text-xs text-gray-500 dark:text-gray-400">{{ platformLabel(account.platform) }} · {{ accountStatusLabel(account.accountStatus) }}</span>
            </div>
            <div v-if="remainingDeleteAccountCount > 0" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 dark:bg-red-900/20 dark:text-red-300">
              {{ t('accountWorkbench.deleteDialog.accountSummaryMore', { count: remainingDeleteAccountCount }) }}
            </div>
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300">
          {{ t('accountWorkbench.deleteDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="deleting" @click="closeDeleteDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-danger" :disabled="deleteDialogAccounts.length === 0 || deleting" @click="confirmDeleteDialog">
          <Icon name="refresh" size="sm" :class="deleting ? 'animate-spin' : 'hidden'" />
          <span>
            {{
              deleteDialogMode === 'single'
                ? t('accountWorkbench.deleteDialog.confirmSingle')
                : t('accountWorkbench.deleteDialog.confirmBatch', { count: deleteDialogAccounts.length })
            }}
          </span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="executionConfirmDialogOpen" :title="t('accountWorkbench.execution.confirmTitle')" width="wide" @close="closeExecutionConfirmDialog">
      <div class="space-y-4">
        <div v-if="selectedTemplate" class="rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('accountWorkbench.execution.confirmHint', { count: selectedIds.length, template: selectedTemplate.name }) }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.selected') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.execution.templateType') }}</div>
            <div class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{{ selectedTemplate ? actionLabel(selectedTemplate.type) : '-' }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800 sm:col-span-2">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.execution.templateDetails') }}</div>
            <div v-if="selectedTemplateMetricRows.length > 0" class="mt-3 grid gap-2 sm:grid-cols-2">
              <div v-for="row in selectedTemplateMetricRows" :key="row.label" class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.label }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ row.value }}</div>
              </div>
            </div>
            <div v-else class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">-</div>
            <div v-if="selectedTemplatePreviewCards.length > 0" class="mt-3 flex flex-wrap gap-2">
              <div
                v-for="card in selectedTemplatePreviewCards"
                :key="`confirm-${card.key}`"
                class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-700"
              >
                <video
                  v-if="card.kind === 'video'"
                  :data-testid="card.confirmTestId"
                  :src="card.src"
                  :aria-label="card.alt"
                  class="h-24 w-24 object-cover"
                  controls
                  playsinline
                  muted
                />
                <img
                  v-else
                  :data-testid="card.confirmTestId"
                  :src="card.src"
                  :alt="card.alt"
                  class="h-24 w-24 object-cover"
                />
              </div>
            </div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('accountWorkbench.execution.accountSummary') }}</div>
          <div class="grid gap-2 text-sm">
            <div v-for="account in selectedAccounts.slice(0, 8)" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ account.name }}</span>
              <span class="shrink-0 truncate text-xs text-gray-500 dark:text-gray-400">{{ platformLabel(account.platform) }} · {{ account.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') }}</span>
            </div>
            <div v-if="selectedIds.length > 8" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('accountWorkbench.deleteDialog.accountSummaryMore', { count: selectedIds.length - 8 }) }}
            </div>
          </div>
        </div>
        <div v-if="executionDisabledReason" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
          {{ executionDisabledReason }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="closeExecutionConfirmDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="!canStartExecution || submitting" @click="submitExecution">
          <Icon name="refresh" size="sm" :class="submitting ? 'animate-spin' : 'hidden'" />
          <span>{{ submitting ? t('common.processing') : t('accountWorkbench.execution.confirmSubmit') }}</span>
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import * as XLSX from 'xlsx'
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import accountWorkbenchAPI from '@/api/accountWorkbench'
import proxiesAPI from '@/api/proxies'
import taskSettingsAPI from '@/api/taskSettings'
import type {
  BatchImportSocialAccountResponse,
  DefaultProxyAssignmentMode,
  ImportSocialAccountRequest,
  SocialAccountBatchResult,
  SocialTaskLog,
  SubmitTaskResponse,
  SubmitTaskRequest,
  UpdateMySocialAccountRequest,
  UserSocialAccount,
} from '@/api/accountWorkbench'
import type { UserProxy } from '@/api/proxies'
import type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  TaskTemplate,
} from '@/api/taskSettings'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import { socialPostMediaRefsSupported, socialTaskMediaRefExecutable, socialTaskMediaRefExecutableStored } from '@/utils/socialTaskMediaValidation'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'
import { formatWorkbenchTaskSummary, formatWorkbenchTaskSummaryMeta } from '@/utils/workbenchTaskSummary'
import { buildAccountImportPreviewRows } from '@/views/accounts/accountImportModel'
import type { Column } from '@/components/common/types'
import type { SelectOption } from '@/types'

interface AccountRow {
  id: number
  name: string
  platform: string
  username: string
  platformUserId: string
  password: string
  phone: string
  email: string
  emailPassword: string
  twoFactor: string
  backupCode: string
  emailClientId: string
  emailToken: string
  registrationIp: string
  authCookie: string
  executionAuth: string
  accountStatus: string
  taskStatus: string
  taskMessage: string
  defaultProxySnapshot: string
  remark: string
  defaultProxyConfigured: boolean
  createdAt: string
  updatedAt: string
}

interface BatchImportRow {
  rowNumber: number
  account: ImportSocialAccountRequest
  valid: boolean
  error: string
  status: 'format_valid' | 'needs_data' | 'batch_duplicate' | 'existing_workbench_duplicate'
}

interface TemplatePreviewCard {
  key: string
  src: string
  alt: string
  kind: 'image' | 'video'
  summaryTestId: string
  confirmTestId: string
}

const { t } = useI18n()
const appStore = useAppStore()

const MAX_TEMPLATE_POOL_VALUES = 500
const MAX_TEMPLATE_VALUE_LENGTH = 2048
const REQUIRED_AVATAR_IMAGE_WIDTH = 400
const REQUIRED_AVATAR_IMAGE_HEIGHT = 400
const REQUIRED_BANNER_IMAGE_WIDTH = 1500
const REQUIRED_BANNER_IMAGE_HEIGHT = 500

const accounts = ref<AccountRow[]>([])
const usableProxies = ref<UserProxy[]>([])
const taskTemplates = ref<TaskTemplate[]>([])
const loading = ref(false)
const submitting = ref(false)
const importing = ref(false)
const deleting = ref(false)
const savingAccountEdit = ref(false)
const savingProxy = ref(false)
const loadError = ref('')
const dependencyLoadError = ref('')
const searchQuery = ref('')
const statusFilter = ref<string | number | boolean | null>('all')
const platformFilter = ref<string | number | boolean | null>('all')
const selectedIds = ref<number[]>([])
const selectedAccount = ref<AccountRow | null>(null)
const detailDialogOpen = ref(false)
const editDialogOpen = ref(false)
const batchImportDialogOpen = ref(false)
const proxyDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const proxyAccount = ref<AccountRow | null>(null)
const editAccount = ref<AccountRow | null>(null)
const deleteTargetAccount = ref<AccountRow | null>(null)
const selectedTemplateId = ref<string | number | boolean | null>(null)
const executionConfirmDialogOpen = ref(false)
const lastTaskResult = ref<SubmitTaskResponse | null>(null)
const proxyAssignmentResult = ref<SocialAccountBatchResult | null>(null)
const batchImportResult = ref<BatchImportSocialAccountResponse | null>(null)
const proxyDialogMode = ref<'single' | 'batch'>('single')
const proxyAssignmentMode = ref<DefaultProxyAssignmentMode>('specific')
const selectedProxyId = ref<string | number | boolean | null>(null)
const batchImportText = ref('')
const batchImportFileRows = ref<BatchImportRow[]>([])
const batchImportFileName = ref('')
const batchImportError = ref('')
const parsingImportFile = ref(false)
const batchImportFileInput = ref<HTMLInputElement | null>(null)
const selectedTemplateMediaPreview = reactive({
  post: [] as string[],
  avatar: '',
  banner: '',
})
let selectedTemplateMediaPreviewToken = 0

const importForm = reactive({ platform: 'x_twitter' })
const editAccountForm = reactive({
  password: '',
  phone: '',
  email: '',
  emailPassword: '',
  twoFactor: '',
  backupCode: '',
  emailClientId: '',
  emailToken: '',
  authCookie: '',
  executionAuth: '',
  remark: '',
})

const accountColumns: Column[] = [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px]' },
  { key: 'name', label: t('accountWorkbench.columns.name'), sortable: true, class: 'min-w-[260px]' },
  { key: 'platform', label: t('accountWorkbench.columns.platform'), sortable: true, class: 'min-w-[128px]' },
  { key: 'accountStatus', label: t('accountWorkbench.columns.accountStatus'), sortable: true, class: 'min-w-[132px]' },
  { key: 'taskStatus', label: t('accountWorkbench.columns.taskStatus'), sortable: true, class: 'min-w-[132px]' },
  { key: 'proxy', label: t('accountWorkbench.columns.proxy'), class: 'min-w-[132px]' },
  { key: 'updatedAt', label: t('accountWorkbench.columns.updatedAt'), sortable: true, class: 'min-w-[184px]', formatter: value => formatDate(String(value || '')) },
  { key: 'actions', label: t('common.actions'), class: 'w-[128px] min-w-[128px]' },
]

const statusFilterOptions = computed<SelectOption[]>(() => [
  { value: 'all', label: t('accountWorkbench.filters.all') },
  { value: 'available', label: t('accountWorkbench.accountStatus.available') },
  { value: 'pending_check', label: t('accountWorkbench.accountStatus.pending_check') },
  { value: 'limited', label: t('accountWorkbench.accountStatus.limited') },
  { value: 'invalid', label: t('accountWorkbench.accountStatus.invalid') },
])

const platformFilterOptions = computed<SelectOption[]>(() => {
  const platforms = Array.from(new Set(accounts.value.map(account => normalizePlatform(account.platform)).filter(Boolean))).sort()
  return [
    { value: 'all', label: t('accountWorkbench.filters.allPlatforms') },
    ...platforms.map(platform => ({ value: platform, label: platformLabel(platform) })),
  ]
})
const importPlatformOptions = computed<SelectOption[]>(() => [
  { value: 'x_twitter', label: 'X / Twitter' },
])

const executionTemplateOptions = computed<SelectOption[]>(() => {
  if (taskTemplates.value.length === 0) {
    return [{ value: null, label: t('accountWorkbench.execution.noTemplates'), disabled: true }]
  }
  const options: SelectOption[] = taskTemplates.value.map(template => ({
    value: template.id,
    label: `${template.name}${template.is_default ? ` · ${t('accountWorkbench.execution.defaultTemplate')}` : ''}`,
    disabled: templateDisabled(template),
    description: templateSummary(template),
  }))
  options.push({
    value: 'message',
    label: `${t('accountWorkbench.actions.message')} · ${t('accountWorkbench.actions.messageUnavailable')}`,
    disabled: true,
  })
  return options
})

const selectedAccounts = computed(() => accounts.value.filter(account => selectedIds.value.includes(account.id)))
const selectedPlatforms = computed(() => Array.from(new Set(selectedAccounts.value.map(account => normalizePlatform(account.platform)))))
const hasMixedPlatforms = computed(() => selectedPlatforms.value.length > 1)
const selectedPlatformUnsupported = computed(() => selectedPlatforms.value.length === 1 && !isTwitterPlatform(selectedPlatforms.value[0]))
const isLoginAction = computed(() => selectedTemplate.value?.type === 'login')
const hasNonExecutableSelection = computed(() => selectedAccounts.value.some(account => isLoginAction.value ? !isLoginableAccount(account) : !isExecutableAccount(account)))
const selectedTemplate = computed(() => taskTemplates.value.find(template => template.id === selectedTemplateId.value) || null)
const selectedTemplateMetricRows = computed(() => selectedTemplate.value ? buildTemplateMetricRows(selectedTemplate.value) : [])
const selectedTemplatePreviewCards = computed<TemplatePreviewCard[]>(() => {
  const template = selectedTemplate.value
  if (!template) return []
  if (template.type === 'post') {
    return selectedTemplateMediaPreview.post
      .map((src, index) => {
        if (!src) return null
        return {
          key: `post-${index}`,
          src,
          alt: `${template.name} media ${index + 1}`,
          kind: inferPreviewCardKind(template.params.media?.[index]),
          summaryTestId: `selected-template-preview-post-${index}`,
          confirmTestId: `execution-confirm-preview-post-${index}`,
        }
      })
      .filter((card): card is TemplatePreviewCard => !!card)
  }
  if (template.type === 'update_avatar' && selectedTemplateMediaPreview.avatar) {
    return [{
      key: 'avatar',
      src: selectedTemplateMediaPreview.avatar,
      alt: `${template.name} avatar`,
      kind: 'image',
      summaryTestId: 'selected-template-preview-avatar',
      confirmTestId: 'execution-confirm-preview-avatar',
    }]
  }
  if (template.type === 'update_banner' && selectedTemplateMediaPreview.banner) {
    return [{
      key: 'banner',
      src: selectedTemplateMediaPreview.banner,
      alt: `${template.name} banner`,
      kind: 'image',
      summaryTestId: 'selected-template-preview-banner',
      confirmTestId: 'execution-confirm-preview-banner',
    }]
  }
  return []
})
const deleteDialogMode = computed<'single' | 'batch'>(() => deleteTargetAccount.value ? 'single' : 'batch')
const deleteDialogAccounts = computed(() => deleteTargetAccount.value ? [deleteTargetAccount.value] : selectedAccounts.value)
const deleteDialogAccountPreview = computed(() => deleteDialogAccounts.value.slice(0, 6))
const remainingDeleteAccountCount = computed(() => Math.max(0, deleteDialogAccounts.value.length - deleteDialogAccountPreview.value.length))
const deleteDialogExecutableCount = computed(() => deleteDialogAccounts.value.filter(isExecutableAccount).length)
const deleteDialogAbnormalCount = computed(() => deleteDialogAccounts.value.length - deleteDialogExecutableCount.value)

const currentActionDisabled = computed(() => !selectedTemplate.value || templateDisabled(selectedTemplate.value))

const batchImportRows = computed(() => parseBatchImportRows(batchImportText.value, importForm.platform.trim()))
const activeBatchImportBaseRows = computed(() => batchImportFileName.value ? batchImportFileRows.value : batchImportRows.value)
const existingWorkbenchImportKeys = computed(() => {
  const keys = new Set<string>()
  accounts.value.forEach((account) => {
    const key = importAccountDedupKey({ platform: account.platform, name: account.name })
    if (key) keys.add(key)
  })
  return keys
})
const activeBatchImportRows = computed(() => markExistingWorkbenchDuplicates(activeBatchImportBaseRows.value))
const activeBatchImportValidRows = computed(() => activeBatchImportRows.value.filter(isBatchImportRowSubmittable))
const batchImportInvalidCount = computed(() => activeBatchImportRows.value.length - activeBatchImportValidRows.value.length)
const batchImportAccountsInput = computed(() => activeBatchImportValidRows.value.map(row => row.account))
const batchImportPreviewRows = computed(() => activeBatchImportRows.value.slice(0, 12))
const canBatchImportAccounts = computed(() => batchImportAccountsInput.value.length > 0 && batchImportInvalidCount.value === 0)
const batchImportResultPreviewItems = computed(() => {
  if (!batchImportResult.value) return []
  return previewBatchImportResultItems(batchImportResult.value)
})
const remainingBatchImportResultItemCount = computed(() => Math.max(0, (batchImportResult.value?.items.length ?? 0) - batchImportResultPreviewItems.value.length))
const proxyAssignmentAccountIds = computed(() => {
  if (proxyDialogMode.value === 'single') return proxyAccount.value ? [proxyAccount.value.id] : []
  return [...selectedIds.value]
})
const proxyAssignmentResultPreviewItems = computed(() => proxyAssignmentResult.value?.items.slice(0, 12) ?? [])
const remainingProxyAssignmentResultItemCount = computed(() => Math.max(0, (proxyAssignmentResult.value?.items.length ?? 0) - proxyAssignmentResultPreviewItems.value.length))
const usableProxyOptions = computed<SelectOption[]>(() => usableProxies.value.map(proxy => ({
  value: proxy.id,
  label: `${proxy.name} · ${proxy.endpoint || '-'}`,
  description: proxy.remark || proxy.ip_type,
})))
const selectedProxy = computed(() => usableProxies.value.find(proxy => proxy.id === Number(selectedProxyId.value)) || null)
const selectedProxyLabel = computed(() => {
  if (proxyAssignmentMode.value === 'clear') return t('accountWorkbench.proxy.modeClear')
  if (proxyAssignmentMode.value === 'random') return t('accountWorkbench.proxy.modeRandom')
  return selectedProxy.value ? selectedProxy.value.name : t('common.none')
})
const proxyAssignmentModeLabel = computed(() => t(`accountWorkbench.proxy.modes.${proxyAssignmentMode.value}`))
const proxyDialogTitle = computed(() => proxyDialogMode.value === 'single' ? t('accountWorkbench.proxy.title') : t('accountWorkbench.proxy.batchTitle'))
const taskResultPreviewLogs = computed(() => lastTaskResult.value?.logs.slice(0, 6) ?? [])
const remainingTaskResultLogCount = computed(() => Math.max(0, (lastTaskResult.value?.logs.length ?? 0) - taskResultPreviewLogs.value.length))
const hasFailedTaskResultLogs = computed(() => !!lastTaskResult.value && ((lastTaskResult.value.failed_closed || 0) > 0 || lastTaskResult.value.logs.some(isFailedTaskResultLog)))
const hasFailedTaskNoChargeSummary = computed(() => !!lastTaskResult.value && (
  (lastTaskResult.value.failed_closed || 0) > 0
  || lastTaskResult.value.logs.some(log => isFailedTaskResultLog(log) && isNoChargeTaskResultLog(log))
))
const taskResultPanelToneClass = computed(() => {
  if (hasFailedTaskResultLogs.value) return 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200'
  return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300'
})
const proxyAssignmentDisabledReason = computed(() => {
  if (proxyAssignmentAccountIds.value.length === 0) return t('accountWorkbench.proxy.selectAccountsFirst')
  if (proxyAssignmentMode.value === 'specific' && !selectedProxy.value) return t('accountWorkbench.proxy.selectOnlineProxyFirst')
  if (proxyAssignmentMode.value === 'random' && usableProxies.value.length === 0) return t('accountWorkbench.proxy.noOnlineProxies')
  return ''
})
const canConfirmProxyAssignment = computed(() => !proxyAssignmentDisabledReason.value && !savingProxy.value)
const canSaveAccountEdit = computed(() => !!editAccount.value)

const canStartExecution = computed(() => {
  if (selectedIds.value.length === 0 || hasNonExecutableSelection.value || hasMixedPlatforms.value || selectedPlatformUnsupported.value || currentActionDisabled.value) return false
  return true
})

const executionDisabledReason = computed(() => {
  if (selectedIds.value.length === 0) return t('accountWorkbench.execution.selectAccountsFirst')
  if (hasNonExecutableSelection.value) return t('accountWorkbench.execution.nonExecutableSelected')
  if (hasMixedPlatforms.value) return t('accountWorkbench.execution.mixedPlatforms')
  if (selectedPlatformUnsupported.value) return t('accountWorkbench.execution.platformUnavailable')
  if (dependencyLoadError.value && taskTemplates.value.length === 0) return t('accountWorkbench.execution.templatesUnavailable')
  if (!selectedTemplate.value) return t('accountWorkbench.execution.templateRequired')
  if (currentActionDisabled.value) return t('accountWorkbench.execution.templateInvalid')
  return ''
})

const filteredAccounts = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  const status = String(statusFilter.value || 'all')
  const platform = String(platformFilter.value || 'all')
  return accounts.value.filter((account) => {
    if (status !== 'all' && presentationAccountStatus(account.accountStatus) !== status) return false
    if (platform !== 'all' && normalizePlatform(account.platform) !== platform) return false
    if (!keyword) return true
    return [
      account.name,
      account.platform,
      account.username,
      account.platformUserId,
      account.password,
      account.phone,
      account.email,
      account.emailPassword,
      account.twoFactor,
      account.backupCode,
      account.emailClientId,
      account.emailToken,
      account.registrationIp,
      account.authCookie,
      account.executionAuth,
      accountStatusLabel(account.accountStatus),
      account.taskStatus,
      account.taskMessage,
      account.defaultProxySnapshot,
      account.remark,
    ].some(value => value.toLowerCase().includes(keyword))
  })
})

const executableAccounts = computed(() => accounts.value.filter(isExecutableAccount))
const abnormalAccounts = computed(() => accounts.value.filter(account => account.accountStatus !== 'available' || !isExecutableAccount(account)))
const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)

const statCards = computed(() => [
  { label: t('accountWorkbench.stats.assigned'), value: String(accounts.value.length), meta: t('accountWorkbench.stats.assignedMeta') },
  { label: t('accountWorkbench.stats.executable'), value: String(executableAccounts.value.length), meta: t('accountWorkbench.stats.executableMeta') },
  { label: t('accountWorkbench.stats.selected'), value: String(selectedIds.value.length), meta: t('accountWorkbench.stats.selectedMeta') },
  { label: t('accountWorkbench.stats.abnormal'), value: String(abnormalAccounts.value.length), meta: t('accountWorkbench.stats.abnormalMeta') },
])

const detailSections = computed(() => {
  if (!selectedAccount.value) return []
  return [
    {
      title: t('accountWorkbench.detailSections.identity'),
      items: [
        { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
        { label: t('accountWorkbench.columns.name'), value: selectedAccount.value.name },
        { label: t('accountWorkbench.columns.platform'), value: platformLabel(selectedAccount.value.platform) },
        { label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
        { label: t('accountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
        { label: t('admin.socialAccountWorkbench.columns.registrationIp'), value: selectedAccount.value.registrationIp },
      ],
    },
    {
      title: t('accountWorkbench.detailSections.credentials'),
      items: [
        { label: t('admin.socialAccountWorkbench.columns.password'), value: selectedAccount.value.password },
        { label: t('admin.socialAccountWorkbench.columns.phone'), value: selectedAccount.value.phone },
        { label: t('admin.socialAccountWorkbench.columns.email'), value: selectedAccount.value.email },
        { label: t('admin.socialAccountWorkbench.columns.emailPassword'), value: selectedAccount.value.emailPassword },
        { label: t('admin.socialAccountWorkbench.columns.twoFactor'), value: selectedAccount.value.twoFactor },
        { label: t('admin.socialAccountWorkbench.columns.backupCode'), value: selectedAccount.value.backupCode },
        { label: t('admin.socialAccountWorkbench.columns.emailClientId'), value: selectedAccount.value.emailClientId },
        { label: t('admin.socialAccountWorkbench.columns.emailToken'), value: selectedAccount.value.emailToken },
        { label: t('admin.socialAccountWorkbench.columns.authCookie'), value: selectedAccount.value.authCookie },
        { label: t('admin.socialAccountWorkbench.columns.executionAuth'), value: selectedAccount.value.executionAuth },
      ],
    },
    {
      title: t('accountWorkbench.detailSections.operations'),
      items: [
        { label: t('accountWorkbench.columns.proxy'), value: selectedAccount.value.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') },
        { label: t('admin.socialAccountWorkbench.columns.defaultProxySnapshot'), value: selectedAccount.value.defaultProxySnapshot },
        { label: t('admin.socialAccountWorkbench.form.remark'), value: selectedAccount.value.remark },
        { label: t('accountWorkbench.columns.accountStatus'), value: accountStatusLabel(selectedAccount.value.accountStatus) },
        { label: t('accountWorkbench.columns.taskStatus'), value: taskStatusLabel(selectedAccount.value.taskStatus) },
        { label: t('accountWorkbench.columns.updatedAt'), value: formatDate(selectedAccount.value.updatedAt) },
      ],
    },
  ]
})

watch(selectedTemplate, async (template) => {
  selectedTemplateMediaPreviewToken += 1
  const previewToken = selectedTemplateMediaPreviewToken
  clearSelectedTemplateMediaPreview()
  if (!template) return
  await loadSelectedTemplateMediaPreview(template, previewToken)
}, { immediate: true })

onBeforeUnmount(() => {
  selectedTemplateMediaPreviewToken += 1
  clearSelectedTemplateMediaPreview()
})

void loadData()

async function loadData() {
  loading.value = true
  loadError.value = ''
  dependencyLoadError.value = ''
  try {
    const accountsResult = await accountWorkbenchAPI.listMyAccounts({ page: 1, page_size: 200 })
    accounts.value = (accountsResult.items ?? []).map(mapAccount)
    selectedIds.value = selectedIds.value.filter(id => accounts.value.some(account => account.id === id))
    await loadOptionalAccountDependencies()
    syncSelectedTemplate()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.load_data', error)
    loadError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.failedToLoad'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function loadOptionalAccountDependencies() {
  const [proxyResult, templateResult] = await Promise.allSettled([
    proxiesAPI.listUsable(),
    taskSettingsAPI.listTemplates(),
  ])
  const failures: string[] = []
  if (proxyResult.status === 'fulfilled') {
    usableProxies.value = proxyResult.value.filter(isUsableProxy)
  } else {
    usableProxies.value = []
    recordClientDiagnostic('account_workbench.unified.load_usable_proxies', proxyResult.reason)
    failures.push(t('accountWorkbench.proxy.dependenciesUnavailable'))
  }
  if (templateResult.status === 'fulfilled') {
    taskTemplates.value = templateResult.value
  } else {
    taskTemplates.value = []
    selectedTemplateId.value = null
    recordClientDiagnostic('account_workbench.unified.load_task_templates', templateResult.reason)
    failures.push(t('accountWorkbench.execution.templatesUnavailable'))
  }
  if (failures.length > 0) {
    dependencyLoadError.value = failures.join(' ')
    appStore.showWarning(t('accountWorkbench.dependencyLoadWarning'))
  }
}

async function batchImportAccounts() {
  if (!canBatchImportAccounts.value || importing.value) return
  importing.value = true
  try {
    const result = await accountWorkbenchAPI.batchImportMyAccounts(batchImportAccountsInput.value)
    batchImportResult.value = result
    appStore.showSuccess(t('accountWorkbench.import.batchSuccess', { count: result.imported, skipped: result.skipped }))
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_import', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.import.batchFailed')))
  } finally {
    importing.value = false
  }
}

function deleteAccount(account: AccountRow) {
  openSingleDeleteDialog(account)
}

function openSingleDeleteDialog(account: AccountRow) {
  if (!account || deleting.value) return
  deleteTargetAccount.value = account
  deleteDialogOpen.value = true
}

function openBatchDeleteDialog() {
  if (selectedIds.value.length === 0 || deleting.value) return
  deleteTargetAccount.value = null
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  if (deleting.value) return
  deleteDialogOpen.value = false
  deleteTargetAccount.value = null
}

async function confirmDeleteDialog() {
  if (deleting.value || deleteDialogAccounts.value.length === 0) return
  if (deleteDialogMode.value === 'single') {
    await deleteSingleAccount()
    return
  }
  await deleteSelectedAccounts()
}

async function deleteSingleAccount() {
  const account = deleteTargetAccount.value
  if (!account || deleting.value) return
  deleting.value = true
  try {
    await accountWorkbenchAPI.deleteMyAccount(account.id)
    appStore.showSuccess(t('accountWorkbench.deleteSuccess', { count: 1 }))
    selectedIds.value = selectedIds.value.filter(id => id !== account.id)
    deleteDialogOpen.value = false
    deleteTargetAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.delete_account', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.deleteFailed')))
  } finally {
    deleting.value = false
  }
}

async function deleteSelectedAccounts() {
  if (selectedIds.value.length === 0 || deleting.value) return
  deleting.value = true
  try {
    const result = await accountWorkbenchAPI.batchDeleteMyAccounts([...selectedIds.value])
    appStore.showSuccess(t('accountWorkbench.batchDeleteSuccess', { count: result.removed, skipped: result.skipped }))
    selectedIds.value = []
    deleteDialogOpen.value = false
    deleteTargetAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.deleteFailed')))
  } finally {
    deleting.value = false
  }
}

async function submitExecution() {
  if (!canStartExecution.value) return
  submitting.value = true
  try {
    const result = await accountWorkbenchAPI.submitTask({
      ...buildTaskPayload(),
      client_request_id: createRequestID(),
    })
    lastTaskResult.value = result
    appStore.showSuccess(t('accountWorkbench.execution.submitted', { count: result.submitted, enqueued: result.enqueued }))
    executionConfirmDialogOpen.value = false
    selectedIds.value = []
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.submit_execution', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.execution.submitFailed')))
  } finally {
    submitting.value = false
  }
}

async function exportAccounts() {
  try {
    const blob = await accountWorkbenchAPI.exportMyAccounts()
    downloadBlob(blob, `socialops-my-accounts-${new Date().toISOString().slice(0, 10)}.csv`)
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.export_accounts', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.exportFailed')))
  }
}

function openBatchImportDialog() {
  batchImportError.value = ''
  batchImportResult.value = null
  batchImportDialogOpen.value = true
}

function openExecutionConfirmDialog() {
  if (!canStartExecution.value) return
  executionConfirmDialogOpen.value = true
}

function closeExecutionConfirmDialog() {
  if (submitting.value) return
  executionConfirmDialogOpen.value = false
}

function openProxyDialog(account: AccountRow) {
  proxyAccount.value = account
  proxyDialogMode.value = 'single'
  proxyAssignmentMode.value = 'specific'
  selectedProxyId.value = null
  proxyAssignmentResult.value = null
  proxyDialogOpen.value = true
}

function openBatchProxyDialog() {
  if (selectedIds.value.length === 0 || savingProxy.value) return
  proxyAccount.value = null
  proxyDialogMode.value = 'batch'
  proxyAssignmentMode.value = usableProxies.value.length > 0 ? 'specific' : 'clear'
  selectedProxyId.value = null
  proxyAssignmentResult.value = null
  proxyDialogOpen.value = true
}

function closeProxyDialog() {
  if (savingProxy.value) return
  proxyDialogOpen.value = false
  proxyAccount.value = null
}

function openEditDialog(account: AccountRow) {
  editAccount.value = account
  resetEditAccountForm(account)
  editDialogOpen.value = true
}

function closeEditDialog() {
  if (savingAccountEdit.value) return
  editDialogOpen.value = false
  editAccount.value = null
}

async function saveAccountEdit() {
  if (!editAccount.value || !canSaveAccountEdit.value || savingAccountEdit.value) return
  savingAccountEdit.value = true
  try {
    const payload: UpdateMySocialAccountRequest = {
      password: trimEditableField(editAccountForm.password),
      phone: trimEditableField(editAccountForm.phone),
      email: trimEditableField(editAccountForm.email),
      email_password: trimEditableField(editAccountForm.emailPassword),
      two_factor: trimEditableField(editAccountForm.twoFactor),
      backup_code: trimEditableField(editAccountForm.backupCode),
      email_client_id: trimEditableField(editAccountForm.emailClientId),
      email_token: trimEditableField(editAccountForm.emailToken),
      auth_cookie: trimEditableField(editAccountForm.authCookie),
      execution_auth: trimEditableField(editAccountForm.executionAuth),
      remark: trimEditableField(editAccountForm.remark),
    }
    await accountWorkbenchAPI.updateMyAccount(editAccount.value.id, payload)
    appStore.showSuccess(t('accountWorkbench.edit.saved'))
    editDialogOpen.value = false
    editAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.update_account', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.edit.failed')))
  } finally {
    savingAccountEdit.value = false
  }
}

async function confirmProxyAssignment() {
  if (!canConfirmProxyAssignment.value) return
  savingProxy.value = true
  try {
    const result = proxyDialogMode.value === 'single' && proxyAccount.value
      ? await assignSingleDefaultProxy(proxyAccount.value)
      : await accountWorkbenchAPI.batchSetDefaultProxy({
        account_ids: proxyAssignmentAccountIds.value,
        mode: proxyAssignmentMode.value,
        proxy_id: proxyAssignmentMode.value === 'specific' ? Number(selectedProxyId.value) : null,
      })
    proxyAssignmentResult.value = result
    appStore.showSuccess(t('accountWorkbench.proxy.savedWithSummary', { succeeded: result.succeeded, failed: result.failed, skipped: result.skipped }))
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.assign_default_proxy', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.proxy.failed')))
  } finally {
    savingProxy.value = false
  }
}

async function assignSingleDefaultProxy(account: AccountRow): Promise<SocialAccountBatchResult> {
  const proxyId = proxyAssignmentMode.value === 'specific' ? Number(selectedProxyId.value) : null
  await accountWorkbenchAPI.setDefaultProxy(account.id, proxyId)
  return {
    total: 1,
    succeeded: 1,
    skipped: 0,
    failed: 0,
    items: [{ id: account.id, name: account.name, status: 'succeeded' }],
  }
}

function openDetailDialog(row: AccountRow) {
  selectedAccount.value = row
  detailDialogOpen.value = true
}

function buildTaskPayload(): SubmitTaskRequest {
  if (!selectedTemplate.value) {
    throw new Error('Task template is required before submitting.')
  }
  return {
    account_ids: [...selectedIds.value],
    template_id: selectedTemplate.value.id,
  }
}

function syncSelectedTemplate() {
  if (selectedTemplate.value && !templateDisabled(selectedTemplate.value)) return
  const firstUsableTemplate = taskTemplates.value.find(template => template.is_default && !templateDisabled(template))
    || taskTemplates.value.find(template => !templateDisabled(template))
  selectedTemplateId.value = firstUsableTemplate?.id ?? null
}

function templateDisabled(template: TaskTemplate) {
  if (!template) return true
  if (selectedPlatformUnsupported.value) return true
  if (template.type === 'follow' || template.type === 'like' || template.type === 'retweet') {
    const targets = normalizedTemplatePoolValues(template.params.targets)
    return !templatePoolValuesValid(targets) || targets.length === 0
  }
  if (template.type === 'post') {
    const contents = normalizedTemplatePoolValues(template.params.contents)
    const mediaCount = countTemplateMediaRefs(template.params.media)
    return !templatePoolValuesValid(contents) || (contents.length === 0 && mediaCount === 0) || !templateMediaRefsValid(template.params.media, 4) || !socialPostMediaRefsSupported(template.params.media)
  }
  if (template.type === 'update_profile') {
    return countTemplateProfileFields(template.params.profile) === 0
  }
  if (template.type === 'update_avatar') {
    return !socialTaskMediaRefExecutable(template.params.avatar)
      || !templateExactImageDimensionsValid(template.params.avatar, REQUIRED_AVATAR_IMAGE_WIDTH, REQUIRED_AVATAR_IMAGE_HEIGHT)
  }
  if (template.type === 'update_banner') {
    return !socialTaskMediaRefExecutable(template.params.banner)
      || !templateExactImageDimensionsValid(template.params.banner, REQUIRED_BANNER_IMAGE_WIDTH, REQUIRED_BANNER_IMAGE_HEIGHT)
  }
  return template.type !== 'login_check' && template.type !== 'login'
}

function templatePoolValuesValid(values?: string[]) {
  const normalized = (values ?? []).map(value => value.trim()).filter(Boolean)
  if (normalized.length > MAX_TEMPLATE_POOL_VALUES) return false
  return normalized.every(value => Array.from(value).length <= MAX_TEMPLATE_VALUE_LENGTH)
}

function templateSummary(template: TaskTemplate) {
  if (template.type === 'login') return t('accountWorkbench.execution.loginSummary')
  if (template.type === 'login_check') return t('accountWorkbench.execution.loginCheckSummary')
  if (template.type === 'post') {
    return t('accountWorkbench.execution.postRichSummary', {
      count: normalizedTemplatePoolValues(template.params.contents).length,
      media: countTemplateMediaRefs(template.params.media),
      quote: String(template.params.quote_post_url || '').trim() ? t('common.yes') : t('common.no'),
    })
  }
  if (template.type === 'update_profile') {
    return t('accountWorkbench.execution.profileSummary', { count: countTemplateProfileFields(template.params.profile) })
  }
  if (template.type === 'update_avatar') return t('accountWorkbench.execution.avatarSummary')
  if (template.type === 'update_banner') return t('accountWorkbench.execution.bannerSummary')
  return t('accountWorkbench.execution.targetPoolSummary', { count: normalizedTemplatePoolValues(template.params.targets).length })
}

function actionLabel(value?: string | null) {
  const normalized = String(value || '').trim()
  return normalized ? t(`accountWorkbench.actions.${normalized}`, normalized) : '-'
}

function buildTemplateMetricRows(template: TaskTemplate) {
  if (template.type === 'follow' || template.type === 'like' || template.type === 'retweet') {
    return [{ label: t('accountWorkbench.execution.targets'), value: normalizedTemplatePoolValues(template.params.targets).length }]
  }
  if (template.type === 'post') {
    return [
      { label: t('accountWorkbench.execution.contents'), value: normalizedTemplatePoolValues(template.params.contents).length },
      { label: t('accountWorkbench.execution.media'), value: countTemplateMediaRefs(template.params.media) },
    ]
  }
  if (template.type === 'update_profile') {
    return [{ label: t('accountWorkbench.execution.profileFields'), value: countTemplateProfileFields(template.params.profile) }]
  }
  if (template.type === 'update_avatar' || template.type === 'update_banner') {
    const media = template.type === 'update_avatar' ? template.params.avatar : template.params.banner
    return [{ label: t('accountWorkbench.execution.media'), value: hasTemplateMediaRef(media) ? 1 : 0 }]
  }
  return []
}

async function loadSelectedTemplateMediaPreview(template: TaskTemplate, previewToken = selectedTemplateMediaPreviewToken) {
  if (template.type === 'post') {
    const previewURLs = await Promise.all((template.params.media ?? []).map(item => resolveSelectedTemplateMediaPreviewURL(item)))
    if (previewToken !== selectedTemplateMediaPreviewToken) {
      previewURLs.forEach(url => revokeObjectURLSafe(url))
      return
    }
    selectedTemplateMediaPreview.post = previewURLs
    return
  }
  if (template.type === 'update_avatar') {
    const url = await resolveSelectedTemplateMediaPreviewURL(template.params.avatar)
    if (previewToken !== selectedTemplateMediaPreviewToken) {
      revokeObjectURLSafe(url)
      return
    }
    selectedTemplateMediaPreview.avatar = url
    return
  }
  if (template.type === 'update_banner') {
    const url = await resolveSelectedTemplateMediaPreviewURL(template.params.banner)
    if (previewToken !== selectedTemplateMediaPreviewToken) {
      revokeObjectURLSafe(url)
      return
    }
    selectedTemplateMediaPreview.banner = url
  }
}

async function resolveSelectedTemplateMediaPreviewURL(item?: SocialTaskMediaRef | null) {
  if (!socialTaskMediaRefExecutableStored(item)) return ''
  const storageKey = String(item?.storage_key || '').trim()
  if (!storageKey) return ''
  try {
    const blob = await taskSettingsAPI.previewMedia(storageKey)
    return createObjectURLSafe(blob)
  } catch {
    return ''
  }
}

function createObjectURLSafe(blob: Blob) {
  const fn = globalThis.URL && typeof globalThis.URL.createObjectURL === 'function'
    ? globalThis.URL.createObjectURL.bind(globalThis.URL)
    : null
  return fn ? fn(blob) : ''
}

function revokeObjectURLSafe(url: string) {
  if (!url) return
  const fn = globalThis.URL && typeof globalThis.URL.revokeObjectURL === 'function'
    ? globalThis.URL.revokeObjectURL.bind(globalThis.URL)
    : null
  if (fn) fn(url)
}

function clearSelectedTemplateMediaPreview() {
  selectedTemplateMediaPreview.post.forEach(url => revokeObjectURLSafe(url))
  revokeObjectURLSafe(selectedTemplateMediaPreview.avatar)
  revokeObjectURLSafe(selectedTemplateMediaPreview.banner)
  selectedTemplateMediaPreview.post = []
  selectedTemplateMediaPreview.avatar = ''
  selectedTemplateMediaPreview.banner = ''
}

function normalizedTemplatePoolValues(values?: string[]) {
  return (values ?? []).map(value => value.trim()).filter(Boolean)
}

function countTemplateProfileFields(profile?: SocialProfileUpdateParams) {
  if (!profile) return 0
  return [
    profile.display_name,
    profile.screen_name,
    profile.description,
    profile.location,
    profile.url,
  ].filter(value => String(value || '').trim() !== '').length
}

function countTemplateMediaRefs(items?: SocialTaskMediaRef[]) {
  return (items ?? []).filter(item => hasTemplateMediaRef(item)).length
}

function templateMediaRefsValid(items?: SocialTaskMediaRef[], maxCount = 4) {
  const validCount = countTemplateMediaRefs(items)
  return validCount === (items ?? []).length && validCount <= maxCount
}

function templateExactImageDimensionsValid(item: SocialTaskMediaRef | undefined, requiredWidth: number, requiredHeight: number) {
  if (!item) return false
  return Number(item.width) === requiredWidth && Number(item.height) === requiredHeight
}

function hasTemplateMediaRef(item?: SocialTaskMediaRef) {
  if (!item) return false
  return [
    item.source,
    item.storage_key,
    item.url,
    item.content_type,
    item.file_name,
    item.sha256,
  ].some(value => String(value || '').trim() !== '')
}

function inferPreviewCardKind(item?: SocialTaskMediaRef | null): 'image' | 'video' {
  return String(item?.content_type || '').trim().toLowerCase() === 'video/mp4' ? 'video' : 'image'
}

function isUsableProxy(proxy: UserProxy) {
  return proxy.status === 'online' && String(proxy.endpoint || '').trim() !== ''
}

function mapAccount(account: UserSocialAccount): AccountRow {
  return {
    id: account.id,
    name: account.name,
    platform: account.platform,
    username: account.username ?? normalizeImportUsername(account.name),
    platformUserId: account.platform_user_id ?? '-',
    password: account.password ?? '',
    phone: account.phone ?? '',
    email: account.email ?? '',
    emailPassword: account.email_password ?? '',
    twoFactor: account.two_factor ?? '',
    backupCode: account.backup_code ?? '',
    emailClientId: account.email_client_id ?? '',
    emailToken: account.email_token ?? '',
    registrationIp: account.registration_ip ?? '',
    authCookie: account.auth_cookie ?? '',
    executionAuth: account.execution_auth ?? '',
    accountStatus: account.account_status,
    taskStatus: account.task_status,
    taskMessage: account.task_message ?? '',
    defaultProxySnapshot: account.default_proxy_snapshot ?? '',
    remark: account.remark ?? '',
    defaultProxyConfigured: account.default_proxy_configured === true,
    createdAt: account.created_at,
    updatedAt: account.updated_at,
  }
}

function toggleSelection(id: number) {
  const account = accounts.value.find(item => item.id === id)
  if (!account) return
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

function clearAccountFilters() {
  searchQuery.value = ''
  statusFilter.value = 'all'
  platformFilter.value = 'all'
}

function isSelected(id: number) {
  return selectedIds.value.includes(id)
}

function isExecutableAccount(account: AccountRow) {
  const taskStatus = account.taskStatus.toLowerCase()
  return account.accountStatus === 'available' && account.defaultProxyConfigured && !['running', 'locked', 'disabled'].includes(taskStatus)
}

function isLoginableAccount(account: AccountRow) {
  // The login action acquires credentials, so it does not require "available".
  // It needs a proxy (login is forced through a proxy) and a password to log in with.
  return account.defaultProxyConfigured && !!account.password
}

function parseBatchImportRows(raw: string, fallbackPlatform: string): BatchImportRow[] {
  return buildAccountImportPreviewRows(raw
    .split(/\r?\n/)
    .map((line, index) => ({ line: line.trim(), rowNumber: index + 1 }))
    .filter(item => item.line)
    .filter((item, index) => index > 0 || !looksLikeImportHeader(splitImportLine(item.line)))
    .map((item) => ({ account: parseImportLine(item.line, fallbackPlatform), rowNumber: item.rowNumber })), {
    duplicateMessage: t('accountWorkbench.import.errors.duplicateAccount'),
    missingAccountMessage: t('accountWorkbench.import.errors.accountRequired'),
    missingPasswordMessage: t('accountWorkbench.import.errors.passwordRequired'),
    missingCredentialMessage: t('accountWorkbench.import.errors.credentialRequired'),
    invalidExecutionAuthMessage: t('accountWorkbench.import.errors.invalidExecutionAuth'),
  }).map((row, index) => ({
    ...row,
    rowNumber: row.rowNumber ?? index + 1,
  }))
}

function markExistingWorkbenchDuplicates(rows: BatchImportRow[]) {
  if (existingWorkbenchImportKeys.value.size === 0) return rows
  return rows.map((row) => {
    if (!row.valid || row.status !== 'format_valid') return row
    const key = importAccountDedupKey(row.account)
    if (!key || !existingWorkbenchImportKeys.value.has(key)) return row
    return {
      ...row,
      valid: false,
      status: 'existing_workbench_duplicate' as const,
      error: t('accountWorkbench.import.errors.duplicateInWorkbench'),
    }
  })
}

function parseImportLine(line: string, fallbackPlatform: string): ImportSocialAccountRequest {
  const columns = splitImportLine(line)
  const [
    name = '',
    password = '',
    twoFactor = '',
    backupCode = '',
    email = '',
    emailPassword = '',
    emailClientID = '',
    emailToken = '',
    registrationIP = '',
    authCookie = '',
    executionAuth = '',
  ] = columns
  return normalizeImportAccount({
    platform: fallbackPlatform || 'x_twitter',
    name,
    password,
    email,
    email_password: emailPassword,
    auth_cookie: authCookie,
    execution_auth: executionAuth,
    two_factor: twoFactor,
    backup_code: backupCode,
    email_client_id: emailClientID,
    email_token: emailToken,
    registration_ip: registrationIP,
  })
}

function splitImportLine(line: string) {
  const delimiter = line.includes('\t') ? /\t/ : /[,，]/
  const parts = line.split(delimiter).map(part => part.trim())
  if (parts.length > 1) return parts
  return line.split(/\s+/).map(part => part.trim()).filter(Boolean)
}

function looksLikeImportHeader(columns: string[]) {
  const normalized = columns.map(normalizeImportHeader)
  return normalized.includes('name') && (normalized.includes('password') || normalized.includes('two_factor') || normalized.includes('auth_cookie') || normalized.includes('execution_auth'))
}

function batchImportRowStatusLabel(row: BatchImportRow) {
  if (row.status === 'batch_duplicate') return t('accountWorkbench.import.status.batchDuplicate')
  if (row.status === 'existing_workbench_duplicate') return t('accountWorkbench.import.status.existingWorkbenchDuplicate')
  if (row.status === 'needs_data') return t('accountWorkbench.import.status.needsData')
  return t('accountWorkbench.accountStatus.not_stored')
}

function batchImportRowStatusClass(row: BatchImportRow) {
  if (row.status === 'batch_duplicate' || row.status === 'existing_workbench_duplicate') return 'text-amber-600 dark:text-amber-300'
  if (row.status === 'needs_data') return 'text-red-600 dark:text-red-300'
  if (!isBatchImportRowSubmittable(row)) return 'text-red-600 dark:text-red-300'
  return 'text-emerald-600 dark:text-emerald-300'
}

function batchImportResultStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'succeeded') return t('common.success')
  if (normalized === 'duplicate') return t('accountWorkbench.import.status.duplicate')
  if (normalized === 'skipped') return t('accountWorkbench.import.status.skipped')
  if (normalized === 'failed') return t('common.error')
  return value || '-'
}

function batchImportResultRowToneClass(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'succeeded') return 'bg-white/75 text-emerald-900 ring-1 ring-emerald-100 dark:bg-dark-800/70 dark:text-emerald-100 dark:ring-emerald-900/40'
  if (normalized === 'duplicate' || normalized === 'skipped') return 'bg-white/75 text-amber-900 ring-1 ring-amber-100 dark:bg-dark-800/70 dark:text-amber-100 dark:ring-amber-900/40'
  if (normalized === 'failed') return 'bg-red-50 text-red-800 ring-1 ring-red-200 dark:bg-red-950/40 dark:text-red-200 dark:ring-red-900/60'
  return 'bg-white/75 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-800/70 dark:text-gray-200 dark:ring-dark-600'
}

function proxyAssignmentResultStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  return t(`accountWorkbench.proxy.resultStatuses.${normalized}`, value || '-')
}

function proxyAssignmentResultRowToneClass(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'succeeded') return 'bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-100 dark:ring-emerald-900/50'
  if (normalized === 'skipped' || normalized === 'duplicate') return 'bg-amber-50 text-amber-800 ring-1 ring-amber-200 dark:bg-amber-950/30 dark:text-amber-100 dark:ring-amber-900/50'
  if (normalized === 'failed') return 'bg-red-50 text-red-800 ring-1 ring-red-200 dark:bg-red-950/40 dark:text-red-200 dark:ring-red-900/60'
  return 'bg-gray-50 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600'
}

function previewBatchImportResultItems(batchImportResult: BatchImportSocialAccountResponse) {
  return batchImportResult.items.slice(0, 12)
}

function isBatchImportRowSubmittable(row: BatchImportRow) {
  if (!row.valid) return false
  return true
}

function importAccountDedupKey(account: ImportSocialAccountRequest) {
  const platform = normalizePlatform(account.platform || 'x_twitter')
  const name = normalizeImportUsername(account.name)
  if (!platform || !name) return ''
  return `${platform}\u0000username\u0000${name}`
}

function normalizeImportUsername(value?: string | null) {
  return trimImportValue(value).toLowerCase().replace(/^@+/, '').trim()
}

function credentialSummary(account: ImportSocialAccountRequest) {
  const parts = []
  if (trimImportValue(account.password)) parts.push(t('accountWorkbench.import.credentials.password'))
  if (trimImportValue(account.two_factor)) parts.push(t('accountWorkbench.import.credentials.twoFactor'))
  if (trimImportValue(account.email)) parts.push(t('accountWorkbench.import.credentials.email'))
  if (trimImportValue(account.auth_cookie)) parts.push(t('accountWorkbench.import.credentials.authCookie'))
  if (trimImportValue(account.execution_auth)) parts.push(t('accountWorkbench.import.credentials.executionAuth'))
  return parts.join(' · ') || '-'
}

function normalizeImportAccount(account: ImportSocialAccountRequest): ImportSocialAccountRequest {
  const normalized: ImportSocialAccountRequest = {
    platform: normalizePlatform(account.platform || 'x_twitter'),
    name: trimImportValue(account.name),
  }
  setImportField(normalized, 'password', trimImportValue(account.password))
  setImportField(normalized, 'phone', trimImportValue(account.phone))
  setImportField(normalized, 'email', trimImportValue(account.email))
  setImportField(normalized, 'email_password', trimImportValue(account.email_password))
  setImportField(normalized, 'auth_cookie', trimImportValue(account.auth_cookie))
  setImportField(normalized, 'execution_auth', trimImportValue(account.execution_auth))
  setImportField(normalized, 'two_factor', trimImportValue(account.two_factor))
  setImportField(normalized, 'backup_code', trimImportValue(account.backup_code))
  setImportField(normalized, 'email_client_id', trimImportValue(account.email_client_id))
  setImportField(normalized, 'email_token', trimImportValue(account.email_token))
  setImportField(normalized, 'registration_ip', trimImportValue(account.registration_ip))
  setImportField(normalized, 'remark', trimImportValue(account.remark))
  return normalized
}

function setImportField<K extends keyof ImportSocialAccountRequest>(account: ImportSocialAccountRequest, key: K, value: string) {
  if (value) {
    account[key] = value as ImportSocialAccountRequest[K]
  }
}

function trimImportValue(value?: string | null) {
  return String(value ?? '').trim()
}

async function handleBatchImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  parsingImportFile.value = true
  batchImportError.value = ''
  try {
    const extension = file.name.split('.').pop()?.toLowerCase() || ''
    let fileText = ''
    if (extension === 'txt') {
      fileText = await file.text()
    } else if (extension === 'xls' || extension === 'xlsx') {
      fileText = await importWorkbookToText(file)
    } else {
      batchImportError.value = t('accountWorkbench.import.errors.unsupportedFile')
      batchImportFileName.value = ''
      batchImportFileRows.value = []
      return
    }
    batchImportText.value = ''
    batchImportFileName.value = file.name
    batchImportFileRows.value = parseBatchImportRows(fileText, importForm.platform.trim())
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_import_file', error)
    batchImportError.value = t('accountWorkbench.import.errors.fileReadFailed')
    batchImportFileName.value = ''
    batchImportFileRows.value = []
  } finally {
    parsingImportFile.value = false
    if (input) input.value = ''
  }
}

async function importWorkbookToText(file: File) {
  const buffer = await file.arrayBuffer()
  const workbook = XLSX.read(buffer, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  if (!sheetName) return ''
  const rows = XLSX.utils.sheet_to_json<Array<string | number | boolean | null>>(workbook.Sheets[sheetName], { header: 1, defval: '' })
  return workbookRowsToImportText(rows)
}

function workbookRowsToImportText(rows: Array<Array<string | number | boolean | null>>) {
  if (rows.length === 0) return ''
  const headers = rows[0].map(value => normalizeImportHeader(String(value || '')))
  const hasHeader = headers.includes('name') || headers.includes('password') || headers.includes('two_factor')
  const dataRows = hasHeader ? rows.slice(1) : rows
  const orderedHeaders = ['name', 'password', 'two_factor', 'backup_code', 'email', 'email_password', 'email_client_id', 'email_token', 'registration_ip', 'auth_cookie', 'execution_auth']
  return dataRows
    .map((row) => {
      if (!row.some(cell => trimImportValue(String(cell || '')))) return ''
      if (hasHeader) {
        const mapped = Object.fromEntries(row.map((cell, index) => [headers[index], trimImportValue(String(cell || ''))]))
        return orderedHeaders.map(header => String(mapped[header] || '')).join('\t')
      }
      return orderedHeaders.map((_, index) => trimImportValue(String(row[index] || ''))).join('\t')
    })
    .filter(Boolean)
    .join('\n')
}

function normalizeImportHeader(header: string) {
  const value = header.trim().toLowerCase().replace(/\s+/g, '_')
  const aliases: Record<string, string> = {
    account: 'name',
    username: 'name',
    user_name: 'name',
    name: 'name',
    '账号': 'name',
    '用户名': 'name',
    password: 'password',
    pass: 'password',
    '密码': 'password',
    two_factor: 'two_factor',
    twofa: 'two_factor',
    '2fa': 'two_factor',
    '二次验证': 'two_factor',
    '两步验证': 'two_factor',
    backup_code: 'backup_code',
    backup: 'backup_code',
    '备份码': 'backup_code',
    email: 'email',
    email_account: 'email',
    mail: 'email',
    '邮箱': 'email',
    '邮箱账号': 'email',
    email_password: 'email_password',
    mail_password: 'email_password',
    '邮箱密码': 'email_password',
    email_client_id: 'email_client_id',
    client_id: 'email_client_id',
    '邮箱客户端id': 'email_client_id',
    '邮箱客户端ID': 'email_client_id',
    email_token: 'email_token',
    mail_token: 'email_token',
    token: 'email_token',
    '邮箱令牌': 'email_token',
    registration_ip: 'registration_ip',
    register_ip: 'registration_ip',
    ip: 'registration_ip',
    '注册ip': 'registration_ip',
    '注册IP': 'registration_ip',
    auth_cookie: 'auth_cookie',
    authcookie: 'auth_cookie',
    cookie: 'auth_cookie',
    cookies: 'auth_cookie',
    '登录cookie': 'auth_cookie',
    '认证cookie': 'auth_cookie',
    '授权cookie': 'auth_cookie',
    execution_auth: 'execution_auth',
    remark: 'remark',
    note: 'remark',
    '备注': 'remark',
  }
  return aliases[value] || value
}

function clearBatchImportSource() {
  batchImportText.value = ''
  batchImportResult.value = null
  clearBatchImportFileSource()
}

function clearBatchImportFileSource() {
  batchImportFileName.value = ''
  batchImportFileRows.value = []
  batchImportError.value = ''
  batchImportResult.value = null
  if (batchImportFileInput.value) {
    batchImportFileInput.value.value = ''
  }
}

function normalizePlatform(value?: string | null) {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[-/\s]+/g, '_')
  if (['twitter', 'x', 'x_twitter', 'twitter_x'].includes(normalized)) return 'x_twitter'
  return normalized
}

function isTwitterPlatform(platform: string) {
  return normalizePlatform(platform) === 'x_twitter'
}

function platformInitial(value?: string | null) {
  const normalized = normalizePlatform(value)
  if (normalized === 'x_twitter') return 'X'
  return (normalized || '?').slice(0, 2).toUpperCase()
}

function platformAvatarClass(value?: string | null) {
  const normalized = normalizePlatform(value)
  if (normalized === 'x_twitter') return 'border-gray-900 bg-gray-900 text-white dark:border-gray-100 dark:bg-gray-100 dark:text-gray-950'
  if (normalized === 'instagram') return 'border-pink-200 bg-pink-50 text-pink-700 dark:border-pink-900/50 dark:bg-pink-900/20 dark:text-pink-300'
  if (normalized === 'tiktok') return 'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-900/50 dark:bg-cyan-900/20 dark:text-cyan-300'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200'
}

function platformLabel(value?: string | null) {
  const normalized = normalizePlatform(value)
  if (!normalized) return t('common.unknown')
  if (normalized === 'x_twitter') return 'X / Twitter'
  return normalized.toUpperCase()
}

function normalizeAccountStatus(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

function presentationAccountStatus(value?: string | null) {
  const normalized = normalizeAccountStatus(value)
  if (normalized === 'not_stored') return 'invalid'
  return normalized
}

function accountStatusLabel(value?: string | null) {
  const normalized = normalizeAccountStatus(value)
  return t(`accountWorkbench.accountStatus.${normalized}`, value || '-')
}

function taskStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  return t(`accountWorkbench.taskStatus.${normalized}`, value || '-')
}

function taskResultStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  return t(`accountWorkbench.execution.taskStatuses.${normalized}`, value || '-')
}

function taskResultChargeLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  return t(`accountWorkbench.execution.chargeStatuses.${normalized}`, value || '-')
}

function taskResultMessage(log: SocialTaskLog) {
  return formatSocialTaskResultMessage(log, t)
}

function taskResultSummary(log: SocialTaskLog) {
  return formatWorkbenchTaskSummary(log, t)
}

function taskResultSummaryMeta(log: SocialTaskLog) {
  return formatWorkbenchTaskSummaryMeta(log, t)
}

function isFailedTaskResultLog(log: SocialTaskLog) {
  const status = String(log.status || '').trim().toLowerCase()
  const chargeStatus = String(log.charge_status || '').trim().toLowerCase()
  return status === 'failed' || chargeStatus === 'charge_failed'
}

function isNoChargeTaskResultLog(log: SocialTaskLog) {
  const chargeStatus = String(log.charge_status || '').trim().toLowerCase()
  return chargeStatus === 'not_charged' || (!log.charged && Number(log.charged_amount || 0) <= 0)
}

function taskResultRowToneClass(log: SocialTaskLog) {
  if (isFailedTaskResultLog(log)) return 'bg-red-50 text-red-800 ring-1 ring-red-200 dark:bg-red-950/40 dark:text-red-200 dark:ring-red-900/60'
  if (log.charged || String(log.charge_status || '').trim().toLowerCase() === 'charged') return 'bg-white/75 text-emerald-900 ring-1 ring-emerald-100 dark:bg-dark-800/70 dark:text-emerald-100 dark:ring-emerald-900/40'
  return 'bg-white/75 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-800/70 dark:text-gray-200 dark:ring-dark-600'
}

function formatChargeAmount(value?: number | null) {
  const amount = Number(value || 0)
  return amount > 0 ? amount.toFixed(2) : '0.00'
}

function accountStatusBadgeClass(value: string) {
  const normalized = presentationAccountStatus(value)
  if (normalized === 'available') return 'badge-success'
  if (['invalid', 'suspended', 'limited'].includes(normalized)) return 'badge-danger'
  return 'badge-warning'
}

function taskStatusBadgeClass(value: string) {
  if (['stored', 'success'].includes(value)) return 'badge-success'
  if (['register_failed', 'risk_rejected', 'ip_unavailable', 'manual_review', 'failed'].includes(value)) return 'badge-danger'
  return 'badge-warning'
}

function taskMessagePanelClass(value?: string | null) {
  const normalized = String(value || '').toLowerCase()
  if (['success', 'stored'].includes(normalized)) return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (['failed', 'register_failed', 'risk_rejected', 'ip_unavailable'].includes(normalized)) return 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300'
  return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300'
}

function resetEditAccountForm(account: AccountRow) {
  editAccountForm.password = account.password
  editAccountForm.phone = account.phone
  editAccountForm.email = account.email
  editAccountForm.emailPassword = account.emailPassword
  editAccountForm.twoFactor = account.twoFactor
  editAccountForm.backupCode = account.backupCode
  editAccountForm.emailClientId = account.emailClientId
  editAccountForm.emailToken = account.emailToken
  editAccountForm.authCookie = account.authCookie
  editAccountForm.executionAuth = account.executionAuth
  editAccountForm.remark = account.remark
}

function trimEditableField(value?: string | null) {
  return String(value ?? '').trim()
}

function formatDate(value?: string) {
  if (!value) return '-'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return '-'
  return time.toLocaleString()
}

function createRequestID() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `social-task-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
</script>
