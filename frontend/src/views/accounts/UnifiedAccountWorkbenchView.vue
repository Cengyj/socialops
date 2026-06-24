<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <LoadErrorBanner
            v-if="loadError"
            :title="t('accountWorkbench.failedToLoad')"
            :message="loadError"
            :retry-label="t('common.retry')"
            @retry="loadData()"
          />
          <div
            v-if="dependencyLoadError && !loadError"
            role="status"
            aria-live="polite"
            aria-atomic="true"
            class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-start gap-3">
                <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0 text-amber-500" />
                <div class="min-w-0">
                  <p class="font-medium">{{ t('accountWorkbench.dependencyLoadWarning') }}</p>
                  <p class="mt-1 min-w-0 break-words text-amber-600 dark:text-amber-300/80" :title="dependencyLoadError">{{ dependencyLoadError }}</p>
                </div>
              </div>
              <button
                type="button"
                class="btn btn-secondary min-w-0 max-w-full shrink-0 justify-center"
                :aria-label="t('common.retry')"
                :title="t('common.retry')"
                @click="loadData()"
              >
                <span class="min-w-0 truncate">{{ t('common.retry') }}</span>
              </button>
            </div>
          </div>

          <SocialAccountStatsGrid
            :stats="statCards"
            test-id-prefix="account-stat"
            grid-class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4 2xl:grid-cols-7"
          />

          <div data-testid="accounts-toolbar" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
            <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-3">
                <div class="flex flex-wrap items-center gap-2">
                  <SearchInput v-model="searchQuery" :placeholder="t('accountWorkbench.searchPlaceholder')" class="w-full shrink-0 sm:w-[250px] xl:min-w-[280px] xl:flex-1 xl:shrink 2xl:min-w-[320px]" />
                  <Select v-model="statusFilter" :options="statusFilterOptions" class="w-full shrink-0 sm:w-[132px] xl:w-[124px] 2xl:w-[132px]" />
                  <Select v-model="platformFilter" :options="platformFilterOptions" class="w-full shrink-0 sm:w-[132px] xl:w-[124px] 2xl:w-[132px]" />
                  <div class="hidden h-6 w-px shrink-0 bg-gray-200 dark:bg-dark-700 xl:block"></div>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto"
                    :aria-label="refreshAccountsButtonTitle"
                    :title="refreshAccountsButtonTitle"
                    :disabled="loading"
                    @click="loadData()"
                  >
                    <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                    <span class="min-w-0 truncate">{{ t('common.refresh') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto"
                    :aria-label="batchImportButtonTitle"
                    :title="batchImportButtonTitle"
                    :disabled="accountActionsLocked"
                    @click="openBatchImportDialog"
                  >
                    <Icon name="upload" size="sm" />
                    <span class="min-w-0 truncate">{{ t('accountWorkbench.import.batchAction') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto"
                    :aria-label="exportAccountsButtonTitle"
                    :title="exportAccountsButtonTitle"
                    :disabled="loading || exportingAccounts"
                    @click="exportAccounts"
                  >
                    <Icon name="download" size="sm" :class="exportingAccounts ? 'animate-spin' : ''" />
                    <span class="min-w-0 truncate">{{ exportingAccounts ? t('common.processing') : t('accountWorkbench.exportAccounts') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto"
                    :aria-label="batchProxyButtonTitle"
                    :title="batchProxyButtonTitle"
                    :disabled="selectedIds.length === 0 || loading || savingProxy"
                    @click="openBatchProxyDialog"
                  >
                    <Icon name="server" size="sm" />
                    <span class="min-w-0 truncate">{{ t('accountWorkbench.proxy.batchAction') }}</span>
                  </button>
                  <div class="hidden h-6 w-px shrink-0 bg-gray-200 dark:bg-dark-700 xl:block"></div>
                  <div v-if="canUseAdminAccountTools" class="flex w-full shrink-0 flex-col gap-2 sm:w-auto sm:flex-row sm:items-center xl:w-[202px]">
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto"
                      :aria-label="storeWorkbenchDisabledReason || t('admin.socialAccountWorkbench.actions.fileUpload')"
                      :title="storeWorkbenchDisabledReason || t('admin.socialAccountWorkbench.actions.fileUpload')"
                      :disabled="!canStoreWorkbenchSelection"
                      @click="openStoreWorkbenchDialog"
                    >
                      <Icon name="database" size="sm" />
                      <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.actions.fileUpload') }}</span>
                    </button>
                  </div>
                  <div v-else class="hidden shrink-0 xl:block xl:w-[202px]" aria-hidden="true"></div>
                  <div class="flex h-9 w-full shrink-0 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800 sm:w-auto">
                    <div class="flex min-w-[104px] flex-1 items-center justify-center whitespace-nowrap bg-primary-50 px-3 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 sm:flex-none">
                      {{ t('accountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}
                    </div>
                    <button
                      type="button"
                      class="flex h-full w-9 shrink-0 items-center justify-center border-l border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-gray-100"
                      :aria-label="clearSelectionButtonTitle"
                      :title="clearSelectionButtonTitle"
                      :disabled="selectedIds.length === 0 || loading"
                      @click="clearSelection"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                    <button
                      type="button"
                      class="flex h-full w-9 shrink-0 items-center justify-center border-l border-gray-200 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-red-300 dark:hover:bg-red-950/30 dark:hover:text-red-200"
                      :aria-label="deleteSelectedButtonTitle"
                      :title="deleteSelectedButtonTitle"
                      :disabled="selectedIds.length === 0 || deleting || loading"
                      @click="openBatchDeleteDialog"
                    >
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                  <Select
                    v-model="selectedAction"
                    :options="executionActionOptions"
                    class="w-full shrink-0 sm:w-[140px] xl:w-[132px] 2xl:w-[148px]"
                    :placeholder="t('accountWorkbench.execution.actionPlaceholder')"
                    :empty-text="t('accountWorkbench.execution.noActions')"
                  />
                  <button
                    type="button"
                    class="btn btn-primary btn-sm h-9 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-3 2xl:px-4"
                    :aria-label="executionStartButtonTitle"
                    :title="executionStartButtonTitle"
                    :disabled="!canStartExecution || submitting"
                    @click="openExecutionConfirmDialog"
                  >
                    <Icon name="play" size="sm" />
                    <span class="min-w-0 truncate">{{ submitting ? t('common.processing') : t('accountWorkbench.execution.start') }}</span>
                  </button>
                </div>
              </div>

              <div
                v-if="selectedIds.length > 0 || selectedTemplate || executionDisabledReason || proxyAssignmentResult || storeWorkbenchResult"
                class="grid gap-2 border-t border-gray-100 pt-3 dark:border-dark-700 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"
              >
                <div v-if="selectedTemplate" class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm dark:border-dark-700 dark:bg-dark-700/60">
                  <div class="break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="selectedTemplate.name">{{ selectedTemplate.name }}</div>
                  <div class="mt-1 break-words text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="templateSummary(selectedTemplate)">{{ templateSummary(selectedTemplate) }}</div>
                  <div v-if="selectedActionTemplatePreviewCards.length > 0" class="mt-3 flex flex-wrap gap-2">
                    <div
                      v-for="card in selectedActionTemplatePreviewCards"
                      :key="card.key"
                      class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
                    >
                      <img
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
                  class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
                  role="status"
                  aria-live="polite"
                  aria-atomic="true"
                  :class="selectedTemplate ? '' : 'lg:col-span-2'"
                  :title="executionDisabledReason"
                >
                  {{ executionDisabledReason }}
                </div>
                <div v-if="proxyAssignmentResult" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true">
                  {{ proxyAssignmentResultSummary }}
                </div>
                <div v-if="storeWorkbenchResult" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true">
                  {{ storeWorkbenchResultSummary }}
                </div>
              </div>

            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="accountColumns" :data="filteredAccounts" :loading="loading" row-key="id" default-sort-key="id" default-sort-order="asc">
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allVisibleSelected"
              :indeterminate="someVisibleSelected"
              :disabled="accountActionsLocked"
              @click.stop
              @change="toggleAllVisible"
            />
          </template>
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isSelected(row.id)"
              :disabled="accountActionsLocked"
              @click.stop
              @change="toggleSelection(row.id)"
            />
          </template>
          <template #cell-id="{ row }">
            <span class="font-mono text-sm font-medium text-gray-700 dark:text-gray-200">{{ row.id }}</span>
          </template>
          <template #cell-name="{ row }">
            <button type="button" class="flex min-w-0 max-w-full items-center gap-3 text-left disabled:cursor-not-allowed disabled:opacity-60 md:min-w-[210px]" :disabled="accountActionsLocked" @click="openDetailDialog(row)">
              <span :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border text-xs font-semibold', platformAvatarClass(row.platform)]">
                {{ platformInitial(row.platform) }}
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">{{ row.username || '-' }}</span>
              </span>
            </button>
          </template>
          <template #cell-platform="{ value }">
            <span class="rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
              {{ platformLabel(String(value || '')) }}
            </span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', workbenchAccountStatusBadgeClass(String(value))]">{{ accountStatusLabel(String(value || '')) }}</span>
          </template>
          <template #cell-taskStatus="{ row, value }">
            <span :class="['badge inline-flex items-center gap-1', workbenchTaskStatusBadgeClass(displayTaskStatus(row, String(value || '')))]">
              <Icon v-if="displayTaskStatusIsActive(row)" name="refresh" size="xs" class="animate-spin" />
              <span>{{ taskStatusLabel(displayTaskStatus(row, String(value || ''))) }}</span>
            </span>
          </template>
          <template #cell-proxy="{ row }">
            <span :class="['badge', row.defaultProxyConfigured ? 'badge-success' : 'badge-warning']">
              {{ row.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-2">
              <button
                type="button"
                class="btn btn-secondary min-w-0 max-w-full justify-center px-2 py-1 text-xs"
                :aria-label="accountRowProxyButtonTitle"
                :title="accountRowProxyButtonTitle"
                :disabled="accountActionsLocked || savingProxy"
                @click="openProxyDialog(row)"
              >
                <span class="min-w-0 truncate">{{ t('accountWorkbench.proxy.action') }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary h-8 w-8 px-0"
                :aria-label="accountRowEditButtonTitle"
                :title="accountRowEditButtonTitle"
                :disabled="accountActionsLocked || savingAccountEdit"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                type="button"
                class="btn btn-secondary h-8 w-8 px-0 text-red-600 hover:border-red-200 hover:bg-red-50 dark:text-red-300 dark:hover:border-red-900/60 dark:hover:bg-red-950/30"
                :aria-label="accountRowDeleteButtonTitle"
                :title="accountRowDeleteButtonTitle"
                :disabled="deleting || accountActionsLocked"
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
                {{ isAccountWorkbenchEmpty ? t('accountWorkbench.empty.title') : t('accountWorkbench.noResults.title') }}
              </p>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
                {{ isAccountWorkbenchEmpty ? t('accountWorkbench.empty.description') : t('accountWorkbench.noResults.description') }}
              </p>
              <div v-if="isAccountWorkbenchEmpty" class="mt-4 flex flex-wrap justify-center gap-2">
                <button
                  type="button"
                  class="btn btn-primary btn-sm min-w-0 max-w-full justify-center"
                  :aria-label="batchImportButtonTitle"
                  :title="batchImportButtonTitle"
                  :disabled="accountActionsLocked"
                  @click="openBatchImportDialog"
                >
                  <Icon name="upload" size="sm" />
                  <span class="min-w-0 truncate">{{ t('accountWorkbench.import.batchAction') }}</span>
                </button>
              </div>
              <button
                v-else
                type="button"
                class="btn btn-secondary btn-sm mt-4 min-w-0 max-w-full justify-center"
                :aria-label="t('accountWorkbench.filters.clear')"
                :title="t('accountWorkbench.filters.clear')"
                @click="clearAccountFilters"
              >
                <Icon name="x" size="sm" />
                <span class="min-w-0 truncate">{{ t('accountWorkbench.filters.clear') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="closeDetailDialog">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('accountWorkbench.sections.managementHint') }}
        </div>
        <div class="space-y-3">
          <div v-for="section in detailSections" :key="section.title" class="rounded-lg border border-gray-200 bg-white p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ section.title }}</div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div v-for="item in section.items" :key="item.key" :data-testid="item.testId" class="rounded-md bg-gray-50 p-3 dark:bg-dark-700">
                <div class="flex min-w-0 items-start justify-between gap-2">
                  <div class="min-w-0 text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <button
                    v-if="item.copyAction === 'emailToken'"
                    type="button"
                    :data-testid="item.copyTestId"
                    class="btn btn-secondary h-7 shrink-0 px-2 text-xs"
                    :disabled="!item.copyable"
                    :title="item.copyable ? item.copyTitle : t('accountWorkbench.credentials.emptyCopy')"
                    @click="copySelectedEmailToken"
                  >
                    <Icon name="copy" size="sm" />
                    <span>{{ t('accountWorkbench.credentials.copy') }}</span>
                  </button>
                </div>
                <div class="mt-1 whitespace-pre-wrap break-all font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
              </div>
            </div>
          </div>
        </div>
        <SocialAccountCredentialPreviewPanel
          v-if="selectedCredentialPreview"
          :credentials="selectedCredentialPreview"
          test-id-prefix="account"
          :title="t('accountWorkbench.credentials.title')"
          :hint="t('accountWorkbench.credentials.previewHint')"
          :copy-label="t('accountWorkbench.credentials.copy')"
          :empty-copy-label="t('accountWorkbench.credentials.emptyCopy')"
          :show-execution-auth-refresh="true"
          :refreshing="selectedAccountRefreshingExecutionAuth"
          :refresh-disabled-reason="selectedExecutionAuthRefreshDisabledReason"
          :refresh-title="t('accountWorkbench.credentials.refreshTitle')"
          :refresh-label="t('accountWorkbench.credentials.refresh')"
          :refresh-processing-label="t('common.processing')"
          @copy="copySelectedCredential"
          @refresh-execution-auth="refreshSelectedExecutionAuth"
        />
        <SocialAccountTaskMessagePanel
          v-if="selectedAccount.taskMessage"
          :message="selectedAccount.taskMessage"
          :status="displayTaskStatus(selectedAccount, selectedAccount.taskStatus)"
        />
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-primary"
          :title="accountDetailCloseButtonTitle"
          :disabled="selectedAccountRefreshingExecutionAuth"
          @click="closeDetailDialog"
        >
          {{ t('common.close') }}
        </button>
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
          <p class="mt-3 min-w-0 break-words text-xs text-gray-500 dark:text-gray-400" :title="t('accountWorkbench.edit.identityHint')">{{ t('accountWorkbench.edit.identityHint') }}</p>
        </div>
        <div v-if="editAccountError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="editAccountError">
          {{ editAccountError }}
        </div>

        <div data-testid="account-edit-form" class="space-y-3">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.credentials') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-password">{{ t('admin.socialAccountWorkbench.form.password') }}</label>
              <input id="account-edit-password" v-model="editAccountForm.password" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-phone">{{ t('admin.socialAccountWorkbench.form.phone') }}</label>
              <input id="account-edit-phone" v-model="editAccountForm.phone" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email">{{ t('admin.socialAccountWorkbench.form.email') }}</label>
              <input id="account-edit-email" v-model="editAccountForm.email" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-password">{{ t('admin.socialAccountWorkbench.form.emailPassword') }}</label>
              <input id="account-edit-email-password" v-model="editAccountForm.emailPassword" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-two-factor">{{ t('admin.socialAccountWorkbench.form.twoFactor') }}</label>
              <input id="account-edit-two-factor" v-model="editAccountForm.twoFactor" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-backup-code">{{ t('admin.socialAccountWorkbench.form.backupCode') }}</label>
              <input id="account-edit-backup-code" v-model="editAccountForm.backupCode" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-client-id">{{ t('admin.socialAccountWorkbench.form.emailClientId') }}</label>
              <input id="account-edit-email-client-id" v-model="editAccountForm.emailClientId" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-email-token">{{ t('admin.socialAccountWorkbench.form.emailToken') }}</label>
              <input id="account-edit-email-token" v-model="editAccountForm.emailToken" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-registration-ip">{{ t('admin.socialAccountWorkbench.form.registrationIp') }}</label>
              <input id="account-edit-registration-ip" v-model="editAccountForm.registrationIp" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccountEdit" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-auth-cookie">{{ t('admin.socialAccountWorkbench.form.authCookie') }}</label>
              <textarea id="account-edit-auth-cookie" v-model="editAccountForm.authCookie" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800" :disabled="savingAccountEdit"></textarea>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-execution-auth">{{ t('admin.socialAccountWorkbench.form.executionAuth') }}</label>
              <textarea id="account-edit-execution-auth" v-model="editAccountForm.executionAuth" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800" :disabled="savingAccountEdit"></textarea>
              <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.form.executionAuthHelp') }}</p>
            </div>
          </div>

          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.operations') }}</div>
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <label class="block text-xs text-gray-500 dark:text-gray-400" for="account-edit-remark">{{ t('admin.socialAccountWorkbench.form.remark') }}</label>
            <textarea id="account-edit-remark" v-model="editAccountForm.remark" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800" :disabled="savingAccountEdit"></textarea>
          </div>
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="accountEditCancelButtonTitle"
          :title="accountEditCancelButtonTitle"
          :disabled="savingAccountEdit"
          @click="closeEditDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="accountEditSaveButtonLabel"
          :title="accountEditSaveButtonTitle"
          :disabled="savingAccountEdit || !canSaveAccountEdit"
          @click="saveAccountEdit"
        >
          <span class="min-w-0 truncate">{{ accountEditSaveButtonLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="batchImportDialogOpen" :title="t('accountWorkbench.import.batchTitle')" width="wide" @close="closeBatchImportDialog">
      <div data-testid="accounts-batch-import-dialog" class="space-y-4">
        <p class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true" :title="t('accountWorkbench.import.batchHint')">
          {{ t('accountWorkbench.import.batchHint') }}
        </p>
        <div class="grid gap-2 sm:grid-cols-[180px_minmax(0,1fr)] sm:items-center">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-200" for="accounts-batch-import-platform">{{ t('accountWorkbench.import.defaultPlatform') }}</label>
          <Select id="accounts-batch-import-platform" v-model="importForm.platform" :options="importPlatformOptions" :disabled="importing || parsingImportFile" />
        </div>
        <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.48fr)]">
          <textarea v-model="batchImportText" class="input min-h-[132px] sm:min-h-[176px] lg:min-h-[220px]" :placeholder="t('accountWorkbench.import.batchPlaceholder')" :disabled="importing || parsingImportFile" @input="clearBatchImportFileSource"></textarea>
          <div class="space-y-3">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-200" for="accounts-batch-import-file">{{ t('accountWorkbench.import.fileLabel') }}</label>
            <div
              data-testid="accounts-batch-import-dropzone"
              :class="[
                'flex min-h-[148px] flex-col items-center justify-center rounded-lg border border-dashed p-3 text-center transition-colors sm:min-h-[176px] sm:p-4 lg:min-h-[220px] lg:p-5',
                batchImportDragActive
                  ? 'border-primary-400 bg-primary-50 text-primary-700 dark:border-primary-700 dark:bg-primary-900/20 dark:text-primary-200'
                  : 'border-gray-300 bg-gray-50 text-gray-600 hover:border-primary-300 hover:bg-primary-50/50 dark:border-dark-600 dark:bg-dark-700/60 dark:text-gray-300 dark:hover:border-primary-800/70 dark:hover:bg-primary-900/10',
                parsingImportFile || importing ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'
              ]"
              role="button"
              tabindex="0"
              :aria-disabled="parsingImportFile || importing"
              :title="batchImportFileName || t('accountWorkbench.import.fileDropTitle')"
              @click="openBatchImportFilePicker"
              @keydown.enter.prevent="openBatchImportFilePicker"
              @keydown.space.prevent="openBatchImportFilePicker"
              @dragenter.prevent="handleBatchImportDragEnter"
              @dragover.prevent="handleBatchImportDragEnter"
              @dragleave.prevent="handleBatchImportDragLeave"
              @drop.prevent="handleBatchImportFileDrop"
            >
              <input
                id="accounts-batch-import-file"
                ref="batchImportFileInput"
                type="file"
                class="hidden"
                accept=".txt,.xls,.xlsx,text/plain,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                :disabled="parsingImportFile || importing"
                @change="handleBatchImportFileChange"
              />
              <span class="flex h-12 w-12 items-center justify-center rounded-lg border border-primary-100 bg-white text-primary-600 shadow-sm dark:border-primary-900/50 dark:bg-dark-800 dark:text-primary-300">
                <Icon name="upload" size="md" :class="parsingImportFile ? 'animate-pulse' : ''" />
              </span>
              <span class="mt-3 block text-base font-semibold text-gray-900 dark:text-white">
                {{ parsingImportFile ? t('common.processing') : t('accountWorkbench.import.fileDropTitle') }}
              </span>
              <span class="mt-1 hidden max-w-sm text-sm leading-6 text-gray-500 dark:text-gray-400 sm:block">
                {{ t('accountWorkbench.import.fileDropHint') }}
              </span>
              <button type="button" class="btn btn-secondary btn-sm mt-3 justify-center" :disabled="parsingImportFile || importing" @click.stop="openBatchImportFilePicker">
                <Icon name="upload" size="sm" />
                <span>{{ t('accountWorkbench.import.chooseFile') }}</span>
              </button>
              <span class="mt-2 block min-h-[20px] max-w-full break-all text-xs font-medium text-gray-500 dark:text-gray-400" :title="batchImportFileName || t('accountWorkbench.import.fileEmpty')">
                {{ batchImportFileName || t('accountWorkbench.import.fileEmpty') }}
              </span>
            </div>
            <button type="button" class="btn btn-secondary btn-sm w-full justify-center" :disabled="(!batchImportText && !batchImportFileName) || parsingImportFile || importing" @click="clearBatchImportSource">
              <Icon name="x" size="sm" />
              <span>{{ t('accountWorkbench.import.clearSource') }}</span>
            </button>
          </div>
        </div>
        <div v-if="batchImportError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="batchImportError">
          {{ batchImportError }}
        </div>
        <div class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true" :title="t('accountWorkbench.import.previewScopeUser')">
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
          <div class="flex min-w-0 flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 text-xs font-medium uppercase text-gray-500 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-400 sm:gap-3">
            <span class="min-w-0 break-words" :title="t('accountWorkbench.import.previewTitle')">{{ t('accountWorkbench.import.previewTitle') }}</span>
            <span class="min-w-0 break-words text-right" :title="t('accountWorkbench.import.previewMeta', { valid: activeBatchImportValidRows.length, invalid: batchImportInvalidCount })">{{ t('accountWorkbench.import.previewMeta', { valid: activeBatchImportValidRows.length, invalid: batchImportInvalidCount }) }}</span>
          </div>
          <div class="max-h-[220px] divide-y divide-gray-100 overflow-auto dark:divide-dark-700">
            <div v-for="row in batchImportPreviewRows" :key="row.rowNumber" class="grid gap-2 px-3 py-2 text-sm md:grid-cols-[64px_minmax(0,1fr)_120px_minmax(0,1.4fr)]">
              <div class="text-xs text-gray-500 dark:text-gray-400">#{{ row.rowNumber }}</div>
              <div class="min-w-0 break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="row.account.name || '-'">{{ row.account.name || '-' }}</div>
              <div class="min-w-0 break-words text-xs font-medium" :class="batchImportRowStatusClass(row)" :title="batchImportRowStatusLabel(row)">
                {{ batchImportRowStatusLabel(row) }}
              </div>
              <div class="min-w-0 break-words text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="row.error || credentialSummary(row.account)">{{ row.error || credentialSummary(row.account) }}</div>
            </div>
          </div>
        </div>
        <div v-if="batchImportResult" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300">
          <div class="font-medium">{{ t('accountWorkbench.import.resultTitle') }}</div>
          <div
            class="mt-1 min-w-0 break-words"
            role="status"
            aria-live="polite"
            aria-atomic="true"
            :title="batchImportResultSummary"
          >
            {{ batchImportResultSummary }}
          </div>
          <SocialAccountBatchResultRows
            class="mt-2"
            combine-status-and-message
            :items="batchImportResultPreviewItems"
            :remaining-count="remainingBatchImportResultItemCount"
            :rows-more-text="t('accountWorkbench.import.resultRowsMore', { count: remainingBatchImportResultItemCount })"
            :item-label="batchImportResultItemLabel"
            :status-label="batchImportResultStatusLabel"
            :item-message="batchImportResultMessage"
            :row-tone-class="socialAccountBatchResultRowToneClass"
          />
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="batchImportCancelButtonTitle"
          :title="batchImportCancelButtonTitle"
          :disabled="importing || parsingImportFile"
          @click="closeBatchImportDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="batchImportConfirmLabel"
          :title="batchImportConfirmLabel"
          :disabled="!canBatchImportAccounts || importing || parsingImportFile"
          @click="batchImportAccounts"
        >
          <span class="min-w-0 truncate">{{ batchImportConfirmLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="storeWorkbenchDialogOpen" :title="t('accountWorkbench.storeWorkbench.title')" width="wide" @close="closeStoreWorkbenchDialog">
      <div class="space-y-4">
        <div class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true" :title="t('accountWorkbench.storeWorkbench.hint', { count: storeableWorkbenchAccounts.length })">
          {{ t('accountWorkbench.storeWorkbench.hint', { count: storeableWorkbenchAccounts.length }) }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-3">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.selected') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.storeWorkbench.storeable') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ storeableWorkbenchAccounts.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.storeWorkbench.skippedSelection') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedNonStoreableWorkbenchCount }}</div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('accountWorkbench.storeWorkbench.accountSummary') }}</div>
          <div class="grid gap-2 text-sm">
            <div v-for="account in storeWorkbenchPreviewAccounts" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 max-w-[58%] break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="account.name">{{ account.name }}</span>
              <span class="min-w-0 max-w-[42%] break-words text-right text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="`${platformLabel(account.platform)} · ${accountStatusLabel(account.accountStatus)}`">{{ platformLabel(account.platform) }} · {{ accountStatusLabel(account.accountStatus) }}</span>
            </div>
            <div v-if="remainingStoreWorkbenchAccountCount > 0" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('accountWorkbench.deleteDialog.accountSummaryMore', { count: remainingStoreWorkbenchAccountCount }) }}
            </div>
            <div v-if="storeWorkbenchPreviewAccounts.length === 0" class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true">
              {{ storeWorkbenchDisabledReason }}
            </div>
          </div>
        </div>
        <div v-if="storeWorkbenchDisabledReason" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true" :title="storeWorkbenchDisabledReason">
          {{ storeWorkbenchDisabledReason }}
        </div>
        <div v-if="storeWorkbenchError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="storeWorkbenchError">
          {{ storeWorkbenchError }}
        </div>
        <SocialAccountBatchResultCard
          v-if="storeWorkbenchResult"
          :title="t('accountWorkbench.storeWorkbench.resultTitle')"
          :summary="storeWorkbenchResultSummary"
          :items="storeWorkbenchResultPreviewItems"
          :remaining-count="remainingStoreWorkbenchResultItemCount"
          :rows-more-text="t('accountWorkbench.storeWorkbench.resultRowsMore', { count: remainingStoreWorkbenchResultItemCount })"
          :item-label="socialAccountBatchResultItemLabel"
          :status-label="proxyAssignmentResultStatusLabel"
          :item-message="socialAccountBatchResultMessage"
          :row-tone-class="socialAccountBatchResultRowToneClass"
        />
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="storeWorkbenchCancelButtonTitle"
          :title="storeWorkbenchCancelButtonTitle"
          :disabled="storingWorkbenchAccounts"
          @click="closeStoreWorkbenchDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="storeWorkbenchConfirmLabel"
          :title="storeWorkbenchConfirmLabel"
          :disabled="!canConfirmStoreWorkbenchSelection"
          @click="confirmStoreWorkbenchSelection"
        >
          <Icon name="refresh" size="sm" :class="storingWorkbenchAccounts ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ storeWorkbenchConfirmLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="proxyDialogOpen" :title="proxyDialogTitle" width="wide" @close="closeProxyDialog">
      <div class="space-y-4">
        <div class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true" :title="proxyDialogHint">
          {{ proxyDialogHint }}
        </div>
        <div
          v-if="proxyAssignmentError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="proxyAssignmentError"
        >
          {{ proxyAssignmentError }}
        </div>
        <div class="grid gap-3 sm:grid-cols-3">
          <button
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'specific' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            :disabled="savingProxy"
            @click="selectProxyAssignmentMode('specific')"
          >
            <span class="block font-medium">{{ t('accountWorkbench.proxy.modeSpecific') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.modeSpecificHint') }}</span>
          </button>
          <button
            v-if="proxyDialogMode === 'batch'"
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'random' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            :disabled="usableProxies.length === 0 || savingProxy"
            @click="selectProxyAssignmentMode('random')"
          >
            <span class="block font-medium">{{ t('accountWorkbench.proxy.modeRandom') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.proxy.modeRandomHint') }}</span>
          </button>
          <button
            type="button"
            :class="['rounded-lg border px-3 py-2 text-left text-sm transition', proxyAssignmentMode === 'clear' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700']"
            :disabled="savingProxy"
            @click="selectProxyAssignmentMode('clear')"
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
            :disabled="savingProxy"
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
            <div class="mt-2 break-all text-sm font-semibold text-gray-900 sm:truncate dark:text-white" :title="selectedProxyLabel">{{ selectedProxyLabel }}</div>
          </div>
        </div>
        <div v-if="proxyAssignmentDisabledReason" class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true" :title="proxyAssignmentDisabledReason">
          {{ proxyAssignmentDisabledReason }}
        </div>
        <SocialAccountBatchResultCard
          v-if="proxyAssignmentResult"
          :title="t('accountWorkbench.proxy.resultTitle')"
          :summary="proxyAssignmentResultSummary"
          :items="proxyAssignmentResultPreviewItems"
          :remaining-count="remainingProxyAssignmentResultItemCount"
          :rows-more-text="t('accountWorkbench.proxy.resultRowsMore', { count: remainingProxyAssignmentResultItemCount })"
          :item-label="socialAccountBatchResultItemLabel"
          :status-label="proxyAssignmentResultStatusLabel"
          :item-message="socialAccountBatchResultMessage"
          :row-tone-class="socialAccountBatchResultRowToneClass"
        />
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="proxyAssignmentCancelButtonTitle"
          :title="proxyAssignmentCancelButtonTitle"
          :disabled="savingProxy"
          @click="closeProxyDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="proxyAssignmentConfirmLabel"
          :title="proxyAssignmentConfirmLabel"
          :disabled="!canConfirmProxyAssignment || savingProxy"
          @click="confirmProxyAssignment"
        >
          <span class="min-w-0 truncate">{{ proxyAssignmentConfirmLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('accountWorkbench.deleteDialog.title')" width="normal" @close="closeDeleteDialog">
      <div class="space-y-4">
        <div class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300" role="status" aria-live="polite" aria-atomic="true" :title="deleteDialogHint">
          {{ deleteDialogHint }}
        </div>
        <div
          v-if="deleteDialogError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="deleteDialogError"
        >
          {{ deleteDialogError }}
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
              <span class="min-w-0 max-w-[58%] break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="account.name">{{ account.name }}</span>
              <span class="min-w-0 max-w-[42%] break-words text-right text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="`${platformLabel(account.platform)} · ${accountStatusLabel(account.accountStatus)}`">{{ platformLabel(account.platform) }} · {{ accountStatusLabel(account.accountStatus) }}</span>
            </div>
            <div v-if="remainingDeleteAccountCount > 0" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 dark:bg-red-900/20 dark:text-red-300">
              {{ t('accountWorkbench.deleteDialog.accountSummaryMore', { count: remainingDeleteAccountCount }) }}
            </div>
          </div>
        </div>
        <div class="min-w-0 break-words rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300" role="status" aria-live="polite" aria-atomic="true" :title="t('accountWorkbench.deleteDialog.impactHint')">
          {{ t('accountWorkbench.deleteDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="deleteDialogCancelButtonTitle"
          :title="deleteDialogCancelButtonTitle"
          :disabled="deleting"
          @click="closeDeleteDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-danger min-w-0 max-w-full justify-center"
          :aria-label="deleteDialogConfirmLabel"
          :title="deleteDialogConfirmLabel"
          :disabled="deleteDialogAccounts.length === 0 || deleting || accountActionsLocked"
          @click="confirmDeleteDialog"
        >
          <Icon name="refresh" size="sm" :class="deleting ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ deleteDialogConfirmLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="executionConfirmDialogOpen" :title="t('accountWorkbench.execution.confirmTitle')" width="wide" @close="closeExecutionConfirmDialog">
      <div class="space-y-4">
        <div v-if="selectedTemplate" class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true" :title="executionConfirmHint">
          {{ executionConfirmHint }}
        </div>
        <div
          v-if="executionSubmitError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="executionSubmitError"
        >
          {{ executionSubmitError }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.stats.selected') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.execution.actionType') }}</div>
            <div class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{{ actionLabel(selectedAction) }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800 sm:col-span-2">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ currentActionRequiresDefaultTemplate ? t('accountWorkbench.execution.templateDetails') : t('accountWorkbench.execution.executionDetails') }}
            </div>
            <div v-if="selectedActionTemplateMetricRows.length > 0" class="mt-3 grid gap-2 sm:grid-cols-2">
              <div v-for="row in selectedActionTemplateMetricRows" :key="row.label" class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.label }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ row.value }}</div>
              </div>
            </div>
            <div v-else class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{{ directActionDescription(normalizedSelectedAction) || '-' }}</div>
            <div v-if="selectedActionTemplatePreviewCards.length > 0" class="mt-3 flex flex-wrap gap-2">
              <div
                v-for="card in selectedActionTemplatePreviewCards"
                :key="`confirm-${card.key}`"
                class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-700"
              >
                <img
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
            <div v-for="account in executionConfirmAccounts" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 max-w-[58%] break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="account.name">{{ account.name }}</span>
              <span class="min-w-0 max-w-[42%] break-words text-right text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="`${platformLabel(account.platform)} · ${account.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured')}`">{{ platformLabel(account.platform) }} · {{ account.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') }}</span>
            </div>
            <div v-if="remainingExecutionConfirmAccountCount > 0" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('accountWorkbench.deleteDialog.accountSummaryMore', { count: remainingExecutionConfirmAccountCount }) }}
            </div>
          </div>
        </div>
        <div v-if="executionDisabledReason" class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true" :title="executionDisabledReason">
          {{ executionDisabledReason }}
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="executionCancelButtonTitle"
          :title="executionCancelButtonTitle"
          :disabled="submitting"
          @click="closeExecutionConfirmDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="executionConfirmButtonTitle"
          :title="executionConfirmButtonTitle"
          :disabled="!canStartExecution || submitting"
          @click="submitExecution"
        >
          <Icon name="refresh" size="sm" :class="submitting ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ executionConfirmButtonTitle }}</span>
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
import LoadErrorBanner from '@/components/common/LoadErrorBanner.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import SocialAccountCredentialPreviewPanel from '@/components/accounts/SocialAccountCredentialPreviewPanel.vue'
import SocialAccountBatchResultCard from '@/components/accounts/SocialAccountBatchResultCard.vue'
import SocialAccountBatchResultRows from '@/components/accounts/SocialAccountBatchResultRows.vue'
import SocialAccountStatsGrid from '@/components/accounts/SocialAccountStatsGrid.vue'
import SocialAccountTaskMessagePanel from '@/components/accounts/SocialAccountTaskMessagePanel.vue'
import accountWorkbenchAPI, { accountWorkbenchAdminAPI } from '@/api/accountWorkbench'
import proxiesAPI from '@/api/proxies'
import taskSettingsAPI from '@/api/taskSettings'
import {
  socialAccountTaskStatusFromAccountSnapshot,
  socialAccountTaskStatusFromTaskResult,
} from '@/utils/socialAccountTaskStatus'
import type {
  BatchDeleteSocialAccountResponse,
  BatchImportSocialAccountResponse,
  DefaultProxyAssignmentMode,
  ImportSocialAccountRequest,
  MyAccountExportParams,
  SocialAccountBatchResult,
  SocialTaskLog,
  SubmitTaskResponse,
  SubmitTaskRequest,
  UpdateMySocialAccountRequest,
  UserSocialAccount,
} from '@/api/accountWorkbench'
import type { UserProxy } from '@/api/proxies'
import type {
  SocialTaskMediaRef,
  TaskTemplate,
} from '@/api/taskSettings'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorCode, extractSafeApiErrorMessage, extractSafeI18nErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import {
  collectSucceededBatchItemIds,
  formatSocialAccountBatchResultItemLabel,
  socialAccountBatchDeleteResultSummaryParams,
  socialAccountBatchResultItemMessage,
  socialAccountBatchResultRowToneClass,
  socialAccountBatchResultStatusLabel,
  socialAccountBatchResultSummaryParams,
  showSocialAccountBatchResultToast,
} from '@/utils/accountWorkbenchBatchResult'
import {
  accountWorkbenchImportResultItemMessage,
  accountWorkbenchImportResultSummaryParams,
  accountWorkbenchImportResultStatusLabel,
} from '@/utils/accountWorkbenchImportResult'
import { buildAccountEditPayload, trimEditableField } from '@/utils/accountWorkbenchEditPayload'
import {
  createObjectURLSafe,
  downloadBlob,
  revokeObjectURLSafe,
} from '@/utils/browser'
import {
  removeSelectedIds,
  retainExistingSelectedIds,
  selectedRowsById,
  toggleSelectedId,
  toggleVisibleSelectedIds,
  visibleSelectionState,
} from '@/utils/selection'
import { accountMatchesWorkbenchSearch } from '@/utils/accountWorkbenchSearch'
import {
  isActiveWorkbenchTaskLog,
  normalizeWorkbenchAccountStatus,
  normalizeWorkbenchTaskLogStatus,
  normalizeWorkbenchTaskStatus,
  presentationWorkbenchAccountStatus,
  workbenchAccountStatusBadgeClass,
  workbenchStatusFallbackText,
  workbenchTaskStatusBadgeClass,
} from '@/utils/accountWorkbenchStatus'
import {
  normalizeSocialPlatform as normalizePlatform,
  socialPlatformAvatarClass as platformAvatarClass,
  socialPlatformInitial as platformInitial,
  socialPlatformLabel,
} from '@/utils/socialPlatformDisplay'
import {
  actionRequiresDefaultTaskTemplate,
  countTemplateMediaRefs,
  countTemplateProfileFields,
  hasTemplateMediaRef,
  normalizedTemplatePoolValues,
  workbenchTaskTemplateDisabled,
} from '@/utils/accountWorkbenchTaskTemplate'
import type { SocialAccountCredentialPreview, SocialAccountCredentialPreviewKey } from '@/utils/socialAccountCredentials'
import { socialTaskMediaRefExecutableStored } from '@/utils/socialTaskMediaValidation'
import { createListPreview } from '@/utils/listPreview'
import { normalizeProxyStatus } from '@/utils/proxyStatus'
import { useSocialAccountCredentialPreview } from '@/composables/useSocialAccountCredentialPreview'
import { useSocialAccountCredentialCopy } from '@/composables/useSocialAccountCredentialCopy'
import { useAccountOperationResults } from './useAccountOperationResults'
import { createAccountWorkbenchErrorMessages } from './accountWorkbenchErrorMessages'
import {
  accountBatchImportButtonTitle as buildBatchImportButtonTitle,
  accountBatchImportCancelButtonTitle as buildBatchImportCancelButtonTitle,
  accountBatchProxyButtonTitle as buildBatchProxyButtonTitle,
  accountClearSelectionButtonTitle as buildClearSelectionButtonTitle,
  accountDeleteCancelButtonTitle as buildDeleteDialogCancelButtonTitle,
  accountDeleteSelectedButtonTitle as buildDeleteSelectedButtonTitle,
  accountDetailCloseButtonTitle as buildAccountDetailCloseButtonTitle,
  accountEditCancelButtonTitle as buildAccountEditCancelButtonTitle,
  accountEditSaveButtonLabel as buildAccountEditSaveButtonLabel,
  accountEditSaveButtonTitle as buildAccountEditSaveButtonTitle,
  accountExecutionCancelButtonTitle as buildExecutionCancelButtonTitle,
  accountExecutionConfirmButtonTitle as buildExecutionConfirmButtonTitle,
  accountExecutionStartButtonTitle as buildExecutionStartButtonTitle,
  accountExportButtonTitle as buildExportButtonTitle,
  accountProxyCancelButtonTitle as buildProxyAssignmentCancelButtonTitle,
  accountRefreshButtonTitle as buildRefreshButtonTitle,
  accountRowDeleteButtonTitle as buildRowDeleteButtonTitle,
  accountRowEditButtonTitle as buildRowEditButtonTitle,
  accountRowProxyButtonTitle as buildRowProxyButtonTitle,
  accountStoreWorkbenchCancelButtonTitle as buildStoreWorkbenchCancelButtonTitle,
} from './accountWorkbenchActionTitles'
import { formatAccountWorkbenchDate as formatDate } from '@/utils/accountWorkbenchDate'
import {
  accountImportCredentialSummary,
  normalizeSocialImportUsername,
  parseSocialAccountImportTextRows,
  socialAccountImportDedupKey,
  socialAccountImportWorkbookRowsToText,
} from '@/views/accounts/accountImportModel'
import type { SocialAccountImportWorkbookCell } from '@/views/accounts/accountImportModel'
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
  summaryTestId: string
  confirmTestId: string
}

interface DetailSectionItem {
  key: string
  label: string
  value: string | number
  testId?: string
  copyAction?: 'emailToken'
  copyable?: boolean
  copyTitle?: string
  copyTestId?: string
}

interface DetailSection {
  title: string
  items: DetailSectionItem[]
}

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { buildCredentialPreviews, buildDeliveryCredentialItems } = useSocialAccountCredentialPreview()

const EXECUTION_ACTIONS = ['login', 'login_check', 'follow', 'post', 'like', 'retweet', 'update_profile', 'update_avatar', 'update_banner'] as const
type ExecutionAction = typeof EXECUTION_ACTIONS[number]
const runtimeExecutionActions: readonly ExecutionAction[] = EXECUTION_ACTIONS
const TASK_ACTIVITY_VISIBLE_POLL_MS = 2000
const TASK_ACTIVITY_HIDDEN_POLL_MS = 8000
const RECOVERABLE_EXECUTION_SUBMIT_ERROR_CODES = new Set([
  'TASK_DEFAULT_TEMPLATE_REQUIRED',
  'TASK_TEMPLATE_INVALID',
  'SOCIAL_IP_NOT_AVAILABLE',
  'SOCIAL_ACCOUNT_NOT_AVAILABLE',
  'SOCIAL_TASK_LOGIN_PASSWORD_REQUIRED',
  'SOCIAL_TASK_ACCOUNT_ID_INVALID',
  'SOCIAL_TASK_ACCOUNT_BUSY',
  'SOCIAL_TASK_TARGET_REQUIRED',
  'SOCIAL_TASK_POST_CONFIGURATION_REQUIRED',
  'SOCIAL_TASK_PAYLOAD_REQUIRED',
  'SOCIAL_TASK_MEDIA_UNSUPPORTED',
])
const RECOVERABLE_PROXY_ASSIGNMENT_ERROR_CODES = new Set([
  'SOCIAL_IP_NOT_AVAILABLE',
  'SOCIAL_IP_NOT_FOUND',
  'SOCIAL_IP_POOL_EMPTY',
  'PROXY_UNAVAILABLE',
])

const accounts = ref<AccountRow[]>([])
const usableProxies = ref<UserProxy[]>([])
const taskTemplates = ref<TaskTemplate[]>([])
const loading = ref(false)
const submitting = ref(false)
const importing = ref(false)
const deleting = ref(false)
const exportingAccounts = ref(false)
const savingAccountEdit = ref(false)
const savingProxy = ref(false)
const refreshingExecutionAuthAccountId = ref<number | null>(null)
const storingWorkbenchAccounts = ref(false)
const loadError = ref('')
const dependencyLoadError = ref('')
const searchQuery = ref('')
const statusFilter = ref<string | number | boolean | null>('all')
const platformFilter = ref<string | number | boolean | null>('all')
const selectedIds = ref<number[]>([])
const selectedAccount = ref<AccountRow | null>(null)
const {
  copySelectedCredential: copySelectedAccountCredential,
  copySelectedEmailToken: copySelectedAccountEmailToken,
} = useSocialAccountCredentialCopy({
  getAccount: () => selectedAccount.value,
  credentialDiagnosticContext: 'account_workbench.unified.copy_credential',
  emailTokenDiagnosticContext: 'account_workbench.unified.copy_email_token',
})
const detailDialogOpen = ref(false)
const editDialogOpen = ref(false)
const batchImportDialogOpen = ref(false)
const proxyDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const deleteDialogError = ref('')
const storeWorkbenchDialogOpen = ref(false)
const storeWorkbenchError = ref('')
const proxyAccount = ref<AccountRow | null>(null)
const editAccount = ref<AccountRow | null>(null)
const editAccountError = ref('')
const deleteTargetAccount = ref<AccountRow | null>(null)
const selectedAction = ref<string | number | boolean | null>('login')
const executionConfirmDialogOpen = ref(false)
const executionSubmitError = ref('')
const taskLogsById = ref<Map<number, SocialTaskLog>>(new Map())
const {
  batchImportResult,
  batchImportResultPreviewItems,
  batchImportResultSummary,
  clearAccountOperationResults,
  proxyAssignmentResult,
  proxyAssignmentResultPreviewItems,
  proxyAssignmentResultSummary,
  remainingBatchImportResultItemCount,
  remainingProxyAssignmentResultItemCount,
  remainingStoreWorkbenchResultItemCount,
  storeWorkbenchResult,
  storeWorkbenchResultPreviewItems,
  storeWorkbenchResultSummary,
} = useAccountOperationResults({
  batchImportResultSummaryText,
  storeWorkbenchResultSummaryText,
  proxyAssignmentResultSummaryText,
})
const proxyDialogMode = ref<'single' | 'batch'>('single')
const proxyAssignmentMode = ref<DefaultProxyAssignmentMode>('specific')
const proxyAssignmentError = ref('')
const selectedProxyId = ref<string | number | boolean | null>(null)
const batchImportText = ref('')
const batchImportFileRows = ref<BatchImportRow[]>([])
const batchImportFileName = ref('')
const batchImportError = ref('')
const batchImportDedupAccounts = ref<AccountRow[]>([])
const parsingImportFile = ref(false)
const batchImportDragActive = ref(false)
const batchImportFileInput = ref<HTMLInputElement | null>(null)
const selectedActionTemplateMediaPreview = reactive({
  post: [] as string[],
  avatar: '',
  banner: '',
})
let selectedActionTemplateMediaPreviewToken = 0
let taskActivityPollTimer: ReturnType<typeof setTimeout> | null = null
let taskActivityPollInFlight = false
let latestLoadRequestID = 0
let latestBatchImportDedupRequestID = 0
let filterReloadTimer: ReturnType<typeof setTimeout> | undefined

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
  registrationIp: '',
  authCookie: '',
  executionAuth: '',
  remark: '',
})

const accountWorkbenchErrorMessages = computed(() => createAccountWorkbenchErrorMessages(t))
const accountEditErrorMessages = computed(() => accountWorkbenchErrorMessages.value)

const proxyAssignmentErrorMessages = computed(() => ({
  ...accountWorkbenchErrorMessages.value,
  PROXY_UNAVAILABLE: t('accountWorkbench.proxy.errors.SOCIAL_IP_NOT_AVAILABLE'),
}))

const accountColumns: Column[] = [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px]' },
  { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), sortable: true, class: 'w-[84px] min-w-[84px]' },
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
  const selectedPlatform = normalizePlatform(String(platformFilter.value || ''))
  const platformSet = new Set(accounts.value.map(account => normalizePlatform(account.platform)).filter(Boolean))
  if (selectedPlatform && selectedPlatform !== 'all') platformSet.add(selectedPlatform)
  const platforms = Array.from(platformSet).sort()
  return [
    { value: 'all', label: t('accountWorkbench.filters.allPlatforms') },
    ...platforms.map(platform => ({ value: platform, label: platformLabel(platform) })),
  ]
})
const importPlatformOptions = computed<SelectOption[]>(() => [
  { value: 'x_twitter', label: platformLabel('x_twitter') },
])

const executionActionOptions = computed<SelectOption[]>(() => {
  if (runtimeExecutionActions.length === 0) {
    return [{ value: null, label: t('accountWorkbench.execution.noActions'), disabled: true }]
  }
  return runtimeExecutionActions.map(action => {
    const template = defaultTemplateByAction.value.get(action)
    if (!actionRequiresDefaultTemplate(action)) {
      return {
        value: action,
        label: actionLabel(action),
        disabled: actionDisabled(action),
        description: directActionDescription(action),
      }
    }
    return {
      value: action,
      label: actionLabel(action),
      disabled: actionDisabled(action),
      description: template ? defaultTemplateOptionDescription(template) : t('accountWorkbench.execution.defaultTemplateMissing'),
    }
  })
})

const selectedAccounts = computed(() => selectedRowsById(accounts.value, selectedIds.value))
const executionConfirmAccountPreview = computed(() => createListPreview(selectedAccounts.value, 8))
const executionConfirmAccounts = computed(() => executionConfirmAccountPreview.value.items)
const remainingExecutionConfirmAccountCount = computed(() => executionConfirmAccountPreview.value.remainingCount)
const accountActionsLocked = computed(() => loading.value)
const refreshAccountsButtonTitle = computed(() => buildRefreshButtonTitle(t, { loading: loading.value }))
const batchImportButtonTitle = computed(() => buildBatchImportButtonTitle(t, { locked: accountActionsLocked.value }))
const exportAccountsButtonTitle = computed(() => buildExportButtonTitle(t, {
  loading: loading.value,
  exporting: exportingAccounts.value,
  selectedCount: selectedIds.value.length,
}))
const batchProxyButtonTitle = computed(() => buildBatchProxyButtonTitle(t, {
  loading: loading.value,
  savingProxy: savingProxy.value,
  selectedCount: selectedIds.value.length,
}))
const clearSelectionButtonTitle = computed(() => buildClearSelectionButtonTitle(t, {
  loading: loading.value,
  selectedCount: selectedIds.value.length,
}))
const deleteSelectedButtonTitle = computed(() => buildDeleteSelectedButtonTitle(t, {
  loading: loading.value,
  deleting: deleting.value,
  selectedCount: selectedIds.value.length,
}))
const accountRowProxyButtonTitle = computed(() => buildRowProxyButtonTitle(t, {
  locked: accountActionsLocked.value,
  savingProxy: savingProxy.value,
}))
const accountRowEditButtonTitle = computed(() => buildRowEditButtonTitle(t, {
  locked: accountActionsLocked.value,
  saving: savingAccountEdit.value,
}))
const accountRowDeleteButtonTitle = computed(() => buildRowDeleteButtonTitle(t, {
  locked: accountActionsLocked.value,
  deleting: deleting.value,
}))
const canUseAdminAccountTools = computed(() => authStore.isAdmin)
const storeableWorkbenchAccounts = computed(() => selectedAccounts.value.filter(isStoreableWorkbenchAccount))
const selectedNonStoreableWorkbenchCount = computed(() => Math.max(0, selectedAccounts.value.length - storeableWorkbenchAccounts.value.length))
const storeWorkbenchAccountPreview = computed(() => createListPreview(storeableWorkbenchAccounts.value, 8))
const storeWorkbenchPreviewAccounts = computed(() => storeWorkbenchAccountPreview.value.items)
const remainingStoreWorkbenchAccountCount = computed(() => storeWorkbenchAccountPreview.value.remainingCount)
const storeWorkbenchDisabledReason = computed(() => {
  if (!canUseAdminAccountTools.value) return ''
  if (selectedIds.value.length === 0) return t('accountWorkbench.storeWorkbench.selectAccountsFirst')
  if (storeableWorkbenchAccounts.value.length === 0) return t('accountWorkbench.storeWorkbench.onlyNotStored')
  return ''
})
const canStoreWorkbenchSelection = computed(() => canUseAdminAccountTools.value && storeableWorkbenchAccounts.value.length > 0 && !storingWorkbenchAccounts.value && !accountActionsLocked.value)
const canConfirmStoreWorkbenchSelection = computed(() => canStoreWorkbenchSelection.value && storeWorkbenchDialogOpen.value && !storingWorkbenchAccounts.value)
const storeWorkbenchConfirmLabel = computed(() => storingWorkbenchAccounts.value ? t('common.processing') : t('accountWorkbench.storeWorkbench.confirm'))
const storeWorkbenchCancelButtonTitle = computed(() => buildStoreWorkbenchCancelButtonTitle(t, { storing: storingWorkbenchAccounts.value }))
const selectedPlatforms = computed(() => Array.from(new Set(selectedAccounts.value.map(account => normalizePlatform(account.platform)))))
const hasMixedPlatforms = computed(() => selectedPlatforms.value.length > 1)
const selectedPlatformUnsupported = computed(() => selectedPlatforms.value.length === 1 && !isTwitterPlatform(selectedPlatforms.value[0]))
const normalizedSelectedAction = computed<ExecutionAction | ''>(() => normalizeExecutionAction(selectedAction.value))
const isLoginAction = computed(() => normalizedSelectedAction.value === 'login')
const hasNonExecutableSelection = computed(() => selectedAccounts.value.some(account => isLoginAction.value ? !isLoginableAccount(account) : !isExecutableAccount(account)))
const loginMissingPasswordCount = computed(() => isLoginAction.value ? selectedAccounts.value.filter(account => !hasAccountPassword(account)).length : 0)
const defaultTemplateByAction = computed(() => {
  const map = new Map<ExecutionAction, TaskTemplate>()
  for (const action of EXECUTION_ACTIONS) {
    const template = taskTemplates.value.find(item => item.type === action && item.is_default)
    if (template) map.set(action, template)
  }
  return map
})
const selectedTemplate = computed(() => {
  const action = normalizedSelectedAction.value
  if (!action || !actionRequiresDefaultTemplate(action)) return null
  return action ? defaultTemplateByAction.value.get(action) ?? null : null
})
const executionConfirmHint = computed(() => {
  if (!selectedTemplate.value) return ''
  return t('accountWorkbench.execution.confirmHint', {
    count: selectedIds.value.length,
    action: actionLabel(selectedAction.value),
    template: selectedTemplate.value.name,
  })
})
const selectedActionTemplateMetricRows = computed(() => selectedTemplate.value ? buildTemplateMetricRows(selectedTemplate.value) : [])
const selectedActionTemplatePreviewCards = computed<TemplatePreviewCard[]>(() => {
  const template = selectedTemplate.value
  if (!template) return []
  if (template.type === 'post') {
    return selectedActionTemplateMediaPreview.post
      .map((src, index) => {
        if (!src) return null
        return {
          key: `post-${index}`,
          src,
          alt: `${template.name} media ${index + 1}`,
          summaryTestId: `selected-template-preview-post-${index}`,
          confirmTestId: `execution-confirm-preview-post-${index}`,
        }
      })
      .filter((card): card is TemplatePreviewCard => !!card)
  }
  if (template.type === 'update_avatar' && selectedActionTemplateMediaPreview.avatar) {
    return [{
      key: 'avatar',
      src: selectedActionTemplateMediaPreview.avatar,
      alt: `${template.name} avatar`,
      summaryTestId: 'selected-template-preview-avatar',
      confirmTestId: 'execution-confirm-preview-avatar',
    }]
  }
  if (template.type === 'update_banner' && selectedActionTemplateMediaPreview.banner) {
    return [{
      key: 'banner',
      src: selectedActionTemplateMediaPreview.banner,
      alt: `${template.name} banner`,
      summaryTestId: 'selected-template-preview-banner',
      confirmTestId: 'execution-confirm-preview-banner',
    }]
  }
  return []
})
const deleteDialogMode = computed<'single' | 'batch'>(() => deleteTargetAccount.value ? 'single' : 'batch')
const deleteDialogAccounts = computed(() => deleteTargetAccount.value ? [deleteTargetAccount.value] : selectedAccounts.value)
const deleteDialogAccountPreviewState = computed(() => createListPreview(deleteDialogAccounts.value, 6))
const deleteDialogAccountPreview = computed(() => deleteDialogAccountPreviewState.value.items)
const remainingDeleteAccountCount = computed(() => deleteDialogAccountPreviewState.value.remainingCount)
const deleteDialogExecutableCount = computed(() => deleteDialogAccounts.value.filter(isExecutableAccount).length)
const deleteDialogAbnormalCount = computed(() => deleteDialogAccounts.value.length - deleteDialogExecutableCount.value)
const deleteDialogHint = computed(() => {
  if (deleteDialogMode.value === 'single' && deleteTargetAccount.value) {
    return t('accountWorkbench.deleteDialog.singleHint', { name: deleteTargetAccount.value.name })
  }
  return t('accountWorkbench.deleteDialog.batchHint', { count: deleteDialogAccounts.value.length })
})
const deleteDialogConfirmLabel = computed(() => {
  if (deleting.value) return t('common.processing')
  return deleteDialogMode.value === 'single'
    ? t('accountWorkbench.deleteDialog.confirmSingle')
    : t('accountWorkbench.deleteDialog.confirmBatch', { count: deleteDialogAccounts.value.length })
})
const deleteDialogCancelButtonTitle = computed(() => buildDeleteDialogCancelButtonTitle(t, { deleting: deleting.value }))

const currentActionRequiresDefaultTemplate = computed(() => actionRequiresDefaultTemplate(normalizedSelectedAction.value))
const currentActionDisabled = computed(() => actionDisabled(normalizedSelectedAction.value))

const batchImportRows = computed(() => parseBatchImportRows(batchImportText.value, importForm.platform.trim()))
const activeBatchImportBaseRows = computed(() => batchImportFileName.value ? batchImportFileRows.value : batchImportRows.value)
const existingWorkbenchImportKeys = computed(() => {
  const keys = new Set<string>()
  batchImportDedupAccounts.value.forEach((account) => {
    const key = socialAccountImportDedupKey({ platform: account.platform, name: account.username || account.name })
    if (key) keys.add(key)
  })
  return keys
})
const activeBatchImportRows = computed(() => markExistingWorkbenchDuplicates(activeBatchImportBaseRows.value))
const activeBatchImportValidRows = computed(() => activeBatchImportRows.value.filter(isBatchImportRowSubmittable))
const batchImportInvalidCount = computed(() => activeBatchImportRows.value.length - activeBatchImportValidRows.value.length)
const batchImportAccountsInput = computed(() => activeBatchImportValidRows.value.map(row => row.account))
const batchImportPreview = computed(() => createListPreview(activeBatchImportRows.value, 12))
const batchImportPreviewRows = computed(() => batchImportPreview.value.items)
const canBatchImportAccounts = computed(() => !accountActionsLocked.value && batchImportAccountsInput.value.length > 0 && batchImportInvalidCount.value === 0)
const batchImportCancelButtonTitle = computed(() => buildBatchImportCancelButtonTitle(t, { processing: importing.value || parsingImportFile.value }))
const batchImportConfirmLabel = computed(() => importing.value || parsingImportFile.value ? t('common.processing') : t('common.confirm'))
const proxyAssignmentAccountIds = computed(() => {
  if (proxyDialogMode.value === 'single') return proxyAccount.value ? [proxyAccount.value.id] : []
  return [...selectedIds.value]
})
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
const proxyAssignmentConfirmLabel = computed(() => savingProxy.value ? t('common.saving') : t('accountWorkbench.proxy.apply'))
const proxyAssignmentCancelButtonTitle = computed(() => buildProxyAssignmentCancelButtonTitle(t, {
  savingProxy: savingProxy.value,
}))
const proxyDialogTitle = computed(() => proxyDialogMode.value === 'single' ? t('accountWorkbench.proxy.title') : t('accountWorkbench.proxy.batchTitle'))
const proxyDialogHint = computed(() => {
  if (proxyDialogMode.value === 'single' && proxyAccount.value) {
    return t('accountWorkbench.proxy.currentAccount', { name: proxyAccount.value.name })
  }
  return t('accountWorkbench.proxy.batchHint', { count: selectedAccounts.value.length })
})
const taskActivityLogs = computed(() => Array.from(taskLogsById.value.values()).sort(compareTaskLogsDesc))
const hasActiveTaskActivity = computed(() => taskActivityLogs.value.some(isActiveTaskLog))
const taskActivityStatusCounts = computed(() => {
  const counts = { pending: 0, running: 0, success: 0, failed: 0 }
  for (const log of taskActivityLogs.value) {
    const status = normalizeWorkbenchTaskLogStatus(log)
    if (status in counts) counts[status as keyof typeof counts] += 1
  }
  return counts
})
const taskLogsByAccountId = computed(() => {
  const map = new Map<number, SocialTaskLog>()
  for (const log of taskActivityLogs.value) {
    if (!map.has(log.social_account_id)) {
      map.set(log.social_account_id, log)
    }
  }
  return map
})
const proxyAssignmentDisabledReason = computed(() => {
  if (proxyAssignmentAccountIds.value.length === 0) return t('accountWorkbench.proxy.selectAccountsFirst')
  if (proxyAssignmentMode.value === 'specific' && !selectedProxy.value) return t('accountWorkbench.proxy.selectOnlineProxyFirst')
  if (proxyAssignmentMode.value === 'random' && usableProxies.value.length === 0) return t('accountWorkbench.proxy.noOnlineProxies')
  return ''
})
const canConfirmProxyAssignment = computed(() => !proxyAssignmentDisabledReason.value && !savingProxy.value && !accountActionsLocked.value)
const canSaveAccountEdit = computed(() => {
  return !!editAccount.value && !accountActionsLocked.value && !editAccountFormMatchesAccount(editAccount.value)
})
const accountEditSaveDisabledReason = computed(() => {
  if (savingAccountEdit.value) return t('common.saving')
  if (accountActionsLocked.value) return t('common.processing')
  if (editAccount.value && editAccountFormMatchesAccount(editAccount.value)) return t('accountWorkbench.edit.noChanges')
  return ''
})
const accountEditSaveButtonTitle = computed(() => buildAccountEditSaveButtonTitle(t, {
  saving: savingAccountEdit.value,
  locked: accountActionsLocked.value,
  disabledReason: accountEditSaveDisabledReason.value,
}))
const accountEditSaveButtonLabel = computed(() => buildAccountEditSaveButtonLabel(t, {
  saving: savingAccountEdit.value,
}))
const accountEditCancelButtonTitle = computed(() => buildAccountEditCancelButtonTitle(t, {
  saving: savingAccountEdit.value,
}))

const canStartExecution = computed(() => {
  if (accountActionsLocked.value || selectedIds.value.length === 0 || hasNonExecutableSelection.value || hasMixedPlatforms.value || selectedPlatformUnsupported.value || currentActionDisabled.value) return false
  return true
})

const executionDisabledReason = computed(() => {
  if (accountActionsLocked.value) return ''
  if (selectedIds.value.length === 0) return t('accountWorkbench.execution.selectAccountsFirst')
  if (isLoginAction.value && loginMissingPasswordCount.value > 0) return t('accountWorkbench.execution.loginPasswordRequired')
  if (hasNonExecutableSelection.value) return t('accountWorkbench.execution.nonExecutableSelected')
  if (hasMixedPlatforms.value) return t('accountWorkbench.execution.mixedPlatforms')
  if (selectedPlatformUnsupported.value) return t('accountWorkbench.execution.platformUnavailable')
  if (dependencyLoadError.value && taskTemplates.value.length === 0 && currentActionRequiresDefaultTemplate.value) return t('accountWorkbench.execution.templatesUnavailable')
  if (currentActionRequiresDefaultTemplate.value && !selectedTemplate.value) return t('accountWorkbench.execution.defaultTemplateRequired')
  if (currentActionDisabled.value) return t('accountWorkbench.execution.defaultTemplateInvalid')
  return ''
})
const executionStartButtonTitle = computed(() => buildExecutionStartButtonTitle(t, {
  submitting: submitting.value,
  locked: accountActionsLocked.value,
  disabledReason: executionDisabledReason.value,
}))
const executionCancelButtonTitle = computed(() => buildExecutionCancelButtonTitle(t, {
  submitting: submitting.value,
}))
const executionConfirmButtonTitle = computed(() => buildExecutionConfirmButtonTitle(t, {
  submitting: submitting.value,
}))

const filteredAccounts = computed(() => accounts.value.filter(accountMatchesCurrentFilters))

const accountFilterParams = computed<MyAccountExportParams>(() => {
  const params: MyAccountExportParams = {}
  const search = searchQuery.value.trim()
  const platform = normalizePlatform(String(platformFilter.value || ''))
  const status = String(statusFilter.value || 'all')
  if (search) params.search = search
  if (platform && platform !== 'all') params.platform = platform
  if (status !== 'all') params.account_status = status
  return params
})
const exportAccountParams = computed<MyAccountExportParams>(() => ({
  ...accountFilterParams.value,
  ...(selectedIds.value.length > 0 ? { account_ids: [...selectedIds.value] } : {}),
}))

const accountListParams = computed(() => ({
  page: 1,
  page_size: 200,
  ...accountFilterParams.value,
}))

const hasActiveAccountFilters = computed(() => Object.keys(accountFilterParams.value).length > 0)
const isAccountWorkbenchEmpty = computed(() => accounts.value.length === 0 && !hasActiveAccountFilters.value)

const executableAccounts = computed(() => accounts.value.filter(isExecutableAccount))
const abnormalAccounts = computed(() => accounts.value.filter(account => normalizeWorkbenchAccountStatus(account.accountStatus) !== 'available' || !isExecutableAccount(account)))
const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const currentVisibleSelectionState = computed(() => visibleSelectionState(selectedIds.value, visibleIds.value))
const allVisibleSelected = computed(() => currentVisibleSelectionState.value.allSelected)
const someVisibleSelected = computed(() => currentVisibleSelectionState.value.someSelected)

const statCards = computed(() => [
  { key: 'assigned', label: t('accountWorkbench.stats.assigned'), value: String(accounts.value.length), meta: t('accountWorkbench.stats.assignedMeta') },
  { key: 'executable', label: t('accountWorkbench.stats.executable'), value: String(executableAccounts.value.length), meta: t('accountWorkbench.stats.executableMeta') },
  { key: 'abnormal', label: t('accountWorkbench.stats.abnormal'), value: String(abnormalAccounts.value.length), meta: t('accountWorkbench.stats.abnormalMeta') },
  { key: 'pending', label: t('accountWorkbench.stats.pending'), value: String(taskActivityStatusCounts.value.pending), meta: t('accountWorkbench.stats.pendingMeta') },
  { key: 'running', label: t('accountWorkbench.stats.running'), value: String(taskActivityStatusCounts.value.running), meta: t('accountWorkbench.stats.runningMeta') },
  { key: 'success', label: t('accountWorkbench.stats.success'), value: String(taskActivityStatusCounts.value.success), meta: t('accountWorkbench.stats.successMeta') },
  { key: 'failed', label: t('accountWorkbench.stats.failed'), value: String(taskActivityStatusCounts.value.failed), meta: t('accountWorkbench.stats.failedMeta') },
])

const detailSections = computed<DetailSection[]>(() => {
  if (!selectedAccount.value) return []
  return [
    {
      title: t('accountWorkbench.detailSections.identity'),
      items: [
        { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
        { key: 'name', label: t('accountWorkbench.columns.name'), value: selectedAccount.value.name },
        { key: 'platform', label: t('accountWorkbench.columns.platform'), value: platformLabel(selectedAccount.value.platform) },
        { key: 'username', label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
        { key: 'platformUserId', label: t('accountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
        { key: 'registrationIp', label: t('admin.socialAccountWorkbench.columns.registrationIp'), value: selectedAccount.value.registrationIp },
      ],
    },
    {
      title: t('accountWorkbench.detailSections.credentials'),
      items: buildDeliveryCredentialItems(selectedAccount.value, {
        emailTokenTestId: 'account-email-token-preview',
        emailTokenCopyTestId: 'account-email-token-copy',
      }),
    },
    {
      title: t('accountWorkbench.detailSections.operations'),
      items: [
        { key: 'proxy', label: t('accountWorkbench.columns.proxy'), value: selectedAccount.value.defaultProxyConfigured ? t('accountWorkbench.proxy.configured') : t('accountWorkbench.proxy.notConfigured') },
        { key: 'defaultProxySnapshot', label: t('admin.socialAccountWorkbench.columns.defaultProxySnapshot'), value: selectedAccount.value.defaultProxySnapshot },
        { key: 'remark', label: t('admin.socialAccountWorkbench.form.remark'), value: selectedAccount.value.remark },
        { key: 'accountStatus', label: t('accountWorkbench.columns.accountStatus'), value: accountStatusLabel(selectedAccount.value.accountStatus) },
        { key: 'taskStatus', label: t('accountWorkbench.columns.taskStatus'), value: taskStatusLabel(displayTaskStatus(selectedAccount.value, selectedAccount.value.taskStatus)) },
        { key: 'updatedAt', label: t('accountWorkbench.columns.updatedAt'), value: formatDate(selectedAccount.value.updatedAt) },
      ],
    },
  ]
})

const selectedCredentialPreview = computed<SocialAccountCredentialPreview[] | null>(() => {
  if (!selectedAccount.value) return null
  return buildCredentialPreviews(selectedAccount.value)
})
const executionAuthRefreshInProgress = computed(() => refreshingExecutionAuthAccountId.value !== null)
const selectedAccountRefreshingExecutionAuth = computed(() => !!selectedAccount.value && refreshingExecutionAuthAccountId.value === selectedAccount.value.id)
const accountDetailCloseButtonTitle = computed(() => buildAccountDetailCloseButtonTitle(t, {
  refreshingExecutionAuth: selectedAccountRefreshingExecutionAuth.value,
}))

const selectedExecutionAuthRefreshDisabledReason = computed(() => {
  const account = selectedAccount.value
  if (!account) return ''
  if (executionAuthRefreshInProgress.value && !selectedAccountRefreshingExecutionAuth.value) return t('common.processing')
  if (!isTwitterPlatform(account.platform)) return t('accountWorkbench.execution.platformUnavailable')
  const hasStoredExecutionAuth = account.executionAuth.trim() !== ''
  if (hasStoredExecutionAuth) return ''
  if (!account.defaultProxyConfigured && !hasAccountPassword(account)) return t('accountWorkbench.credentials.refreshNeedsProxyAndPassword')
  if (!account.defaultProxyConfigured) return t('accountWorkbench.credentials.refreshNeedsProxy')
  if (!hasAccountPassword(account)) return t('accountWorkbench.credentials.refreshNeedsPassword')
  return ''
})

watch([selectedTemplate, selectedPlatformUnsupported], async ([template]) => {
  const previewToken = invalidateSelectedActionTemplateMediaPreview()
  clearSelectedActionTemplateMediaPreview()
  if (!template || templateDisabled(template)) return
  await loadSelectedActionTemplateMediaPreview(template, previewToken)
}, { immediate: true })

watch(selectedProxyId, () => {
  if (!proxyDialogOpen.value || savingProxy.value) return
  proxyAssignmentResult.value = null
})

watch([searchQuery, statusFilter, platformFilter], () => {
  if (filterReloadTimer) clearTimeout(filterReloadTimer)
  filterReloadTimer = setTimeout(() => {
    void loadData()
  }, 250)
})

onBeforeUnmount(() => {
  if (filterReloadTimer) clearTimeout(filterReloadTimer)
  invalidateSelectedActionTemplateMediaPreview()
  clearSelectedActionTemplateMediaPreview()
  stopTaskActivityPolling()
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', handleTaskActivityVisibilityChange)
  }
})

void initializeAccountsWorkbench()

async function initializeAccountsWorkbench() {
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleTaskActivityVisibilityChange)
  }
  await loadData()
  await refreshTaskLogs({ refreshAccountsOnSettled: false })
}

async function loadData(options?: { silent?: boolean }) {
  const requestID = ++latestLoadRequestID
  if (!options?.silent) loading.value = true
  loadError.value = ''
  dependencyLoadError.value = ''
  try {
    const accountsResult = await accountWorkbenchAPI.listMyAccounts(accountListParams.value)
    if (requestID !== latestLoadRequestID) return
    accounts.value = (accountsResult.items ?? []).map(mapAccount)
    pruneTaskLogsForCurrentAccounts()
    selectedIds.value = retainExistingSelectedIds(selectedIds.value, accounts.value)
    syncSelectedAccountFromList()
    syncEditAccountFromList()
    syncDeleteTargetFromList()
    syncProxyTargetFromList()
    syncStoreWorkbenchDialogFromSelection()
    syncExecutionConfirmFromSelection()
    await loadOptionalAccountDependencies(requestID)
    if (requestID !== latestLoadRequestID) return
    syncExecutionConfirmFromSelection()
    syncSelectedAction()
  } catch (error) {
    if (requestID !== latestLoadRequestID) return
    recordClientDiagnostic('account_workbench.unified.load_data', error)
    loadError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.failedToLoad'), accountWorkbenchErrorMessages.value)
    appStore.showError(loadError.value)
  } finally {
    if (requestID === latestLoadRequestID && !options?.silent) loading.value = false
  }
}

async function refreshBatchImportDedupAccounts() {
  const requestID = ++latestBatchImportDedupRequestID
  batchImportDedupAccounts.value = [...accounts.value]
  try {
    const result = await accountWorkbenchAPI.listMyAccounts({ page: 1, page_size: 1000 })
    if (requestID !== latestBatchImportDedupRequestID) return
    batchImportDedupAccounts.value = (result.items ?? []).map(mapAccount)
  } catch (error) {
    if (requestID !== latestBatchImportDedupRequestID) return
    recordClientDiagnostic('account_workbench.unified.load_batch_import_dedup_accounts', error)
  }
}

async function loadOptionalAccountDependencies(requestID: number) {
  const [proxyResult, templateResult] = await Promise.allSettled([
    proxiesAPI.listUsable(),
    taskSettingsAPI.listTemplates(),
  ])
  if (requestID !== latestLoadRequestID) return
  const failures: string[] = []
  if (proxyResult.status === 'fulfilled') {
    usableProxies.value = proxyResult.value.filter(isUsableProxy)
    syncSelectedProxyFromUsableProxies()
  } else {
    usableProxies.value = []
    syncSelectedProxyFromUsableProxies()
    recordClientDiagnostic('account_workbench.unified.load_usable_proxies', proxyResult.reason)
    failures.push(t('accountWorkbench.proxy.dependenciesUnavailable'))
  }
  if (templateResult.status === 'fulfilled') {
    taskTemplates.value = templateResult.value
  } else {
    taskTemplates.value = []
    selectedAction.value = null
    recordClientDiagnostic('account_workbench.unified.load_task_templates', templateResult.reason)
    failures.push(t('accountWorkbench.execution.templatesUnavailable'))
  }
  if (failures.length > 0) {
    dependencyLoadError.value = failures.join(' ')
    appStore.showWarning(t('accountWorkbench.dependencyLoadWarning'))
  }
}

function syncSelectedProxyFromUsableProxies() {
  if (selectedProxyId.value == null) return
  const selectedID = Number(selectedProxyId.value)
  if (usableProxies.value.some(proxy => proxy.id === selectedID)) return
  selectedProxyId.value = null
}

function trackTaskLogs(logs: SocialTaskLog[]) {
  if (logs.length === 0) return
  mergeTaskLogs(logs)
  if (logs.some(isActiveTaskLog)) {
    scheduleTaskActivityPoll(0)
  }
}

function mergeTaskLogs(logs: SocialTaskLog[]) {
  if (logs.length === 0) return
  const next = new Map(taskLogsById.value)
  for (const log of logs) {
    if (!log || !log.id) continue
    if (!isTaskLogForCurrentAccount(log)) continue
    next.set(log.id, log)
  }
  taskLogsById.value = next
}

function replaceTaskLogs(logs: SocialTaskLog[]) {
  const next = new Map<number, SocialTaskLog>()
  for (const log of logs) {
    if (!log || !log.id) continue
    if (!isTaskLogForCurrentAccount(log)) continue
    next.set(log.id, log)
  }
  taskLogsById.value = next
}

function isTaskLogForCurrentAccount(log: SocialTaskLog) {
  return accounts.value.some(account => account.id === log.social_account_id)
}

function pruneTaskLogsForCurrentAccounts() {
  const currentAccountIds = new Set(accounts.value.map(account => account.id))
  if (currentAccountIds.size === 0) {
    replaceTaskLogs([])
    return
  }
  const next = new Map<number, SocialTaskLog>()
  for (const [id, log] of taskLogsById.value.entries()) {
    if (currentAccountIds.has(log.social_account_id)) {
      next.set(id, log)
    }
  }
  taskLogsById.value = next
}

function stopTaskActivityPolling() {
  if (taskActivityPollTimer) {
    clearTimeout(taskActivityPollTimer)
    taskActivityPollTimer = null
  }
}

function scheduleTaskActivityPoll(delay?: number) {
  stopTaskActivityPolling()
  if (!hasActiveTaskActivity.value) return
  const interval = typeof document !== 'undefined' && document.visibilityState === 'hidden'
    ? TASK_ACTIVITY_HIDDEN_POLL_MS
    : TASK_ACTIVITY_VISIBLE_POLL_MS
  taskActivityPollTimer = setTimeout(() => {
    void refreshTaskLogs({ refreshAccountsOnSettled: true })
  }, delay ?? interval)
}

function handleTaskActivityVisibilityChange() {
  if (typeof document === 'undefined') return
  if (hasActiveTaskActivity.value) {
    scheduleTaskActivityPoll(document.visibilityState === 'hidden' ? undefined : 0)
  }
}

async function refreshTaskLogs(options?: { refreshAccountsOnSettled?: boolean }) {
  if (taskActivityPollInFlight) return
  const accountIds = accounts.value.map(account => account.id)
  if (accountIds.length === 0) {
    replaceTaskLogs([])
    return
  }
  taskActivityPollInFlight = true
  try {
    const result = await accountWorkbenchAPI.listTaskLogs({
      account_ids: accountIds,
      statuses: ['pending', 'running', 'success', 'failed'],
      limit: 200,
    })
    const logs = result.logs ?? []
    replaceTaskLogs(logs)
    updateAccountsTaskStateFromLogs(logs)
    if (hasActiveTaskActivity.value) {
      scheduleTaskActivityPoll()
    } else {
      stopTaskActivityPolling()
      if (options?.refreshAccountsOnSettled) {
        await loadData({ silent: true })
      }
    }
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.refresh_task_logs', error)
    if (hasActiveTaskActivity.value) {
      scheduleTaskActivityPoll()
    }
  } finally {
    taskActivityPollInFlight = false
  }
}

async function batchImportAccounts() {
  if (!canBatchImportAccounts.value || importing.value || parsingImportFile.value) return
  importing.value = true
  clearAccountOperationResults()
  try {
    const result = await accountWorkbenchAPI.batchImportMyAccounts(batchImportAccountsInput.value)
    clearAccountOperationResults({ keep: ['batchImport'] })
    batchImportResult.value = result
    updateAccountsFromResponse(result.accounts ?? [])
    showBatchImportResultToast(result)
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_import', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.import.batchFailed'), accountWorkbenchErrorMessages.value))
  } finally {
    importing.value = false
  }
}

function showBatchImportResultToast(result: BatchImportSocialAccountResponse) {
  showResultSeverityToast({
    succeeded: result.succeeded,
    failed: result.failed,
    skipped: result.skipped,
    summary: batchImportResultSummaryText(result),
    successMessage: t('accountWorkbench.import.batchSuccess', { count: result.imported, skipped: result.skipped }),
  })
}

function batchImportResultSummaryText(result: BatchImportSocialAccountResponse | null | undefined) {
  if (!result) return ''
  return t('accountWorkbench.import.resultSummary', accountWorkbenchImportResultSummaryParams(result))
}

const accountResultSummaryKeys = {
  storeWorkbench: 'accountWorkbench.storeWorkbench.resultSummary',
  proxyAssignment: 'accountWorkbench.proxy.resultSummary',
} as const

type AccountResultSummaryKey = typeof accountResultSummaryKeys[keyof typeof accountResultSummaryKeys]

function socialAccountResultSummary(
  key: AccountResultSummaryKey,
  result: Pick<SocialAccountBatchResult, 'total' | 'succeeded' | 'failed' | 'skipped'> | null | undefined,
) {
  if (!result) return ''
  return t(key, socialAccountBatchResultSummaryParams(result))
}

function storeWorkbenchResultSummaryText(result: Pick<SocialAccountBatchResult, 'total' | 'succeeded' | 'failed' | 'skipped'> | null | undefined) {
  return socialAccountResultSummary(accountResultSummaryKeys.storeWorkbench, result)
}

function proxyAssignmentResultSummaryText(result: Pick<SocialAccountBatchResult, 'total' | 'succeeded' | 'failed' | 'skipped'> | null | undefined) {
  return socialAccountResultSummary(accountResultSummaryKeys.proxyAssignment, result)
}

function deleteAccount(account: AccountRow) {
  if (accountActionsLocked.value) return
  openSingleDeleteDialog(account)
}

function openSingleDeleteDialog(account: AccountRow) {
  if (!account || deleting.value || accountActionsLocked.value) return
  deleteTargetAccount.value = account
  deleteDialogError.value = ''
  deleteDialogOpen.value = true
}

function openBatchDeleteDialog() {
  if (selectedIds.value.length === 0 || deleting.value || accountActionsLocked.value) return
  deleteTargetAccount.value = null
  deleteDialogError.value = ''
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  if (deleting.value) return
  deleteDialogOpen.value = false
  deleteTargetAccount.value = null
  deleteDialogError.value = ''
}

async function confirmDeleteDialog() {
  if (deleting.value || accountActionsLocked.value || deleteDialogAccounts.value.length === 0) return
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
  deleteDialogError.value = ''
  clearAccountOperationResults()
  try {
    await accountWorkbenchAPI.deleteMyAccount(account.id)
    appStore.showSuccess(t('accountWorkbench.deleteSuccess', { count: 1 }))
    removeAccountsFromLocalState([account.id])
    deleteDialogOpen.value = false
    deleteTargetAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.delete_account', error)
    deleteDialogError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.deleteFailed'), accountWorkbenchErrorMessages.value)
    appStore.showError(deleteDialogError.value)
  } finally {
    deleting.value = false
  }
}

async function deleteSelectedAccounts() {
  if (selectedIds.value.length === 0 || deleting.value) return
  const requestedIds = [...selectedIds.value]
  deleting.value = true
  deleteDialogError.value = ''
  clearAccountOperationResults()
  try {
    const result = await accountWorkbenchAPI.batchDeleteMyAccounts(requestedIds)
    showBatchDeleteResultToast(result)
    removeSucceededBatchItemsFromLocalState(result, requestedIds)
    deleteDialogOpen.value = false
    deleteTargetAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_delete', error)
    deleteDialogError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.deleteFailed'), accountWorkbenchErrorMessages.value)
    appStore.showError(deleteDialogError.value)
  } finally {
    deleting.value = false
  }
}

function showBatchDeleteResultToast(result: BatchDeleteSocialAccountResponse) {
  showResultSeverityToast({
    succeeded: result.succeeded,
    failed: result.failed,
    skipped: result.skipped,
    summary: t('accountWorkbench.batchDeleteResultSummary', socialAccountBatchDeleteResultSummaryParams(result)),
    successMessage: t('accountWorkbench.batchDeleteSuccess', { count: result.removed, skipped: result.skipped }),
  })
}

function removeSucceededBatchItemsFromLocalState(result: Pick<SocialAccountBatchResult, 'succeeded' | 'skipped' | 'failed' | 'items'>, requestedIds: number[]) {
  const removedIds = collectSucceededBatchItemIds(result, requestedIds)
  removeAccountsFromLocalState([...removedIds])
}

function removeAccountsFromLocalState(accountIds: number[]) {
  const removedIds = new Set(accountIds.filter(id => typeof id === 'number'))
  if (removedIds.size === 0) return
  accounts.value = accounts.value.filter(account => !removedIds.has(account.id))
  selectedIds.value = removeSelectedIds(selectedIds.value, removedIds)
  pruneTaskLogsForCurrentAccounts()
  syncSelectedAccountFromList()
  syncEditAccountFromList()
  syncDeleteTargetFromList()
  syncProxyTargetFromList()
  syncStoreWorkbenchDialogFromSelection()
  syncExecutionConfirmFromSelection()
}

function updateAccountsProxyStateFromBatchResult(result: Pick<SocialAccountBatchResult, 'succeeded' | 'skipped' | 'failed' | 'items'>, requestedIds: number[], mode: DefaultProxyAssignmentMode, proxySnapshot = '') {
  if (mode === 'random') return
  const succeededIds = collectSucceededBatchItemIds(result, requestedIds)
  if (succeededIds.size === 0) return
  for (const accountId of succeededIds) {
    updateAccountProxyState(accountId, mode, proxySnapshot)
  }
}

function updateAccountProxyState(accountId: number, mode: DefaultProxyAssignmentMode, proxySnapshot = '') {
  const index = accounts.value.findIndex(item => item.id === accountId)
  if (index < 0) return
  const current = accounts.value[index]
  const nextSnapshot = mode === 'clear'
    ? ''
    : (mode === 'specific' && proxySnapshot ? proxySnapshot : current.defaultProxySnapshot)
  const updated = {
    ...current,
    defaultProxyConfigured: mode !== 'clear',
    defaultProxySnapshot: nextSnapshot,
  }
  syncAccountRowWithCurrentFilters(updated)
}

function defaultProxySnapshotFromProxy(proxy: UserProxy | null) {
  if (!proxy) return ''
  return JSON.stringify({
    id: proxy.id,
    name: proxy.name,
    ip_type: proxy.ip_type,
    endpoint: proxy.endpoint || '',
    status: proxy.status,
  })
}

async function submitExecution() {
  if (!canStartExecution.value || submitting.value) return
  submitting.value = true
  executionSubmitError.value = ''
  clearAccountOperationResults()
  try {
    const result = await accountWorkbenchAPI.submitTask({
      ...buildTaskPayload(),
      client_request_id: createRequestID(),
    })
    trackTaskLogs(result.logs ?? [])
    updateAccountsTaskStateFromLogs(result.logs ?? [])
    showTaskSubmitResultToast(result, t('accountWorkbench.execution.submitted', { count: result.submitted, enqueued: result.enqueued }))
    executionConfirmDialogOpen.value = false
    executionSubmitError.value = ''
    removeEnqueuedTaskAccountsFromSelection(result.logs ?? [])
    await loadData({ silent: true })
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.submit_execution', error)
    executionSubmitError.value = extractSafeI18nErrorMessage(error, t, 'accountWorkbench.execution.errors', t('accountWorkbench.execution.submitFailed'))
    appStore.showError(executionSubmitError.value)
    await recoverExecutionSubmitState(error)
  } finally {
    submitting.value = false
  }
}

async function recoverExecutionSubmitState(error: unknown) {
  if (!shouldRecoverExecutionSubmitState(error)) return
  executionConfirmDialogOpen.value = false
  executionSubmitError.value = ''
  try {
    await loadData({ silent: true })
    await refreshTaskLogs({ refreshAccountsOnSettled: false })
  } catch (refreshError) {
    recordClientDiagnostic('account_workbench.unified.submit_execution_recover', refreshError)
  }
}

function shouldRecoverExecutionSubmitState(error: unknown) {
  const code = extractApiErrorCode(error)
  return !!code && RECOVERABLE_EXECUTION_SUBMIT_ERROR_CODES.has(code)
}

function showTaskSubmitResultToast(result: SubmitTaskResponse, successMessage: string) {
  const failedClosed = Number(result.failed_closed || 0)
  showResultSeverityToast({
    succeeded: result.enqueued,
    failed: failedClosed,
    skipped: 0,
    summary: t('accountWorkbench.execution.resultSummary', {
      submitted: result.submitted,
      enqueued: result.enqueued,
      failed: failedClosed,
    }),
    successMessage,
  })
}

function showBatchResultSeverityToast(result: Pick<SocialAccountBatchResult, 'succeeded' | 'failed' | 'skipped'>, summary: string, successMessage: string) {
  showResultSeverityToast({
    succeeded: result.succeeded,
    failed: result.failed,
    skipped: result.skipped,
    summary,
    successMessage,
  })
}

function showResultSeverityToast(result: { succeeded: number; failed: number; skipped: number; summary: string; successMessage: string }) {
  showSocialAccountBatchResultToast(result, {
    showError: message => appStore.showError(message),
    showSuccess: message => appStore.showSuccess(message),
    showWarning: message => appStore.showWarning(message),
  })
}

function removeEnqueuedTaskAccountsFromSelection(logs: SocialTaskLog[]) {
  const enqueuedAccountIds = new Set(
    logs
      .filter(log => isActiveTaskLog(log))
      .map(log => log.social_account_id),
  )
  if (enqueuedAccountIds.size === 0) return
  selectedIds.value = removeSelectedIds(selectedIds.value, enqueuedAccountIds)
}

async function exportAccounts() {
  if (loading.value || exportingAccounts.value) return
  exportingAccounts.value = true
  try {
    const blob = await accountWorkbenchAPI.exportMyAccounts(exportAccountParams.value)
    const scope = selectedIds.value.length > 0 ? 'selected' : 'my'
    downloadBlob(blob, `socialops-${scope}-accounts-${new Date().toISOString().slice(0, 10)}.csv`)
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.export_accounts', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('accountWorkbench.exportFailed'), accountWorkbenchErrorMessages.value))
  } finally {
    exportingAccounts.value = false
  }
}

function openBatchImportDialog() {
  if (accountActionsLocked.value) return
  batchImportError.value = ''
  batchImportResult.value = null
  batchImportDedupAccounts.value = [...accounts.value]
  batchImportDialogOpen.value = true
  void refreshBatchImportDedupAccounts()
}

function closeBatchImportDialog() {
  if (importing.value || parsingImportFile.value) return
  batchImportDialogOpen.value = false
  clearBatchImportSource()
}

function openStoreWorkbenchDialog() {
  if (!canStoreWorkbenchSelection.value) return
  storeWorkbenchResult.value = null
  storeWorkbenchError.value = ''
  storeWorkbenchDialogOpen.value = true
}

function closeStoreWorkbenchDialog() {
  if (storingWorkbenchAccounts.value) return
  storeWorkbenchDialogOpen.value = false
  storeWorkbenchError.value = ''
}

async function confirmStoreWorkbenchSelection() {
  if (!canConfirmStoreWorkbenchSelection.value || storingWorkbenchAccounts.value) return
  const accountIds = storeableWorkbenchAccounts.value.map(account => account.id)
  if (accountIds.length === 0) return
  storingWorkbenchAccounts.value = true
  storeWorkbenchError.value = ''
  clearAccountOperationResults()
  try {
    const result = await accountWorkbenchAdminAPI.storeWorkbenchAccounts(accountIds)
    clearAccountOperationResults({ keep: ['storeWorkbench'] })
    storeWorkbenchResult.value = result
    showStoreWorkbenchResultToast(result)
    removeSucceededBatchItemsFromLocalState(result, accountIds)
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.store_workbench_accounts', error)
    storeWorkbenchError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.storeWorkbench.failed'), accountWorkbenchErrorMessages.value)
    appStore.showError(storeWorkbenchError.value)
  } finally {
    storingWorkbenchAccounts.value = false
  }
}

function showStoreWorkbenchResultToast(result: SocialAccountBatchResult) {
  showBatchResultSeverityToast(
    result,
    storeWorkbenchResultSummaryText(result),
    t('accountWorkbench.storeWorkbench.savedWithSummary', { succeeded: result.succeeded, failed: result.failed, skipped: result.skipped }),
  )
}

function openExecutionConfirmDialog() {
  if (!canStartExecution.value) return
  executionSubmitError.value = ''
  executionConfirmDialogOpen.value = true
}

function closeExecutionConfirmDialog() {
  if (submitting.value) return
  executionConfirmDialogOpen.value = false
  executionSubmitError.value = ''
}

function openProxyDialog(account: AccountRow) {
  if (accountActionsLocked.value || savingProxy.value) return
  proxyAccount.value = account
  proxyDialogMode.value = 'single'
  proxyAssignmentMode.value = 'specific'
  proxyAssignmentError.value = ''
  selectedProxyId.value = null
  proxyAssignmentResult.value = null
  proxyDialogOpen.value = true
}

function openBatchProxyDialog() {
  if (selectedIds.value.length === 0 || savingProxy.value || accountActionsLocked.value) return
  proxyAccount.value = null
  proxyDialogMode.value = 'batch'
  proxyAssignmentMode.value = usableProxies.value.length > 0 ? 'specific' : 'clear'
  proxyAssignmentError.value = ''
  selectedProxyId.value = null
  proxyAssignmentResult.value = null
  proxyDialogOpen.value = true
}

function closeProxyDialog() {
  if (savingProxy.value) return
  proxyDialogOpen.value = false
  proxyAccount.value = null
  proxyAssignmentError.value = ''
}

function selectProxyAssignmentMode(mode: DefaultProxyAssignmentMode) {
  if (savingProxy.value) return
  if (proxyAssignmentMode.value === mode) return
  proxyAssignmentMode.value = mode
  proxyAssignmentError.value = ''
  proxyAssignmentResult.value = null
}

function openEditDialog(account: AccountRow) {
  if (accountActionsLocked.value || savingAccountEdit.value) return
  editAccount.value = account
  editAccountError.value = ''
  resetEditAccountForm(account)
  editDialogOpen.value = true
}

function closeEditDialog() {
  if (savingAccountEdit.value) return
  editDialogOpen.value = false
  editAccount.value = null
  editAccountError.value = ''
}

async function saveAccountEdit() {
  if (!editAccount.value || !canSaveAccountEdit.value || savingAccountEdit.value) return
  savingAccountEdit.value = true
  editAccountError.value = ''
  try {
    const payload: UpdateMySocialAccountRequest = buildAccountEditPayload(editAccountForm)
    const updated = await accountWorkbenchAPI.updateMyAccount(editAccount.value.id, payload)
    if (updated) {
      updateAccountFromResponse(updated)
    }
    clearAccountOperationResults()
    appStore.showSuccess(t('accountWorkbench.edit.saved'))
    editDialogOpen.value = false
    editAccount.value = null
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.update_account', error)
    editAccountError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.edit.failed'), accountEditErrorMessages.value)
    appStore.showError(editAccountError.value)
  } finally {
    savingAccountEdit.value = false
  }
}

async function confirmProxyAssignment() {
  if (!canConfirmProxyAssignment.value) return
  savingProxy.value = true
  proxyAssignmentError.value = ''
  clearAccountOperationResults()
  try {
    const result = proxyDialogMode.value === 'single' && proxyAccount.value
      ? await assignSingleDefaultProxy(proxyAccount.value)
      : await accountWorkbenchAPI.batchSetDefaultProxy({
        account_ids: proxyAssignmentAccountIds.value,
        mode: proxyAssignmentMode.value,
        proxy_id: proxyAssignmentMode.value === 'specific' ? Number(selectedProxyId.value) : null,
    })
    clearAccountOperationResults({ keep: ['proxyAssignment'] })
    proxyAssignmentResult.value = result
    proxyAssignmentError.value = ''
    if (proxyDialogMode.value === 'batch') {
      updateAccountsProxyStateFromBatchResult(result, proxyAssignmentAccountIds.value, proxyAssignmentMode.value, defaultProxySnapshotFromProxy(selectedProxy.value))
    }
    showProxyAssignmentResultToast(result)
    await loadData()
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.assign_default_proxy', error)
    proxyAssignmentError.value = extractSafeApiErrorMessage(error, t('accountWorkbench.proxy.failed'), proxyAssignmentErrorMessages.value)
    appStore.showError(proxyAssignmentError.value)
    if (isRecoverableProxyAssignmentError(error)) {
      await loadData({ silent: true })
    }
  } finally {
    savingProxy.value = false
  }
}

function isRecoverableProxyAssignmentError(error: unknown) {
  const code = extractApiErrorCode(error)
  return !!code && RECOVERABLE_PROXY_ASSIGNMENT_ERROR_CODES.has(code)
}

function showProxyAssignmentResultToast(result: SocialAccountBatchResult) {
  showBatchResultSeverityToast(
    result,
    proxyAssignmentResultSummaryText(result),
    t('accountWorkbench.proxy.savedWithSummary', { succeeded: result.succeeded, failed: result.failed, skipped: result.skipped }),
  )
}

async function assignSingleDefaultProxy(account: AccountRow): Promise<SocialAccountBatchResult> {
  const proxyId = proxyAssignmentMode.value === 'specific' ? Number(selectedProxyId.value) : null
  const updated = await accountWorkbenchAPI.setDefaultProxy(account.id, proxyId)
  if (updated) {
    updateAccountFromResponse(updated)
  }
  return {
    total: 1,
    succeeded: 1,
    skipped: 0,
    failed: 0,
    items: [{ id: account.id, name: account.name, status: 'succeeded' }],
  }
}

function openDetailDialog(row: AccountRow) {
  if (accountActionsLocked.value) return
  selectedAccount.value = row
  detailDialogOpen.value = true
}

function closeDetailDialog() {
  if (selectedAccountRefreshingExecutionAuth.value) return
  detailDialogOpen.value = false
  selectedAccount.value = null
}

async function copySelectedCredential(key: SocialAccountCredentialPreviewKey) {
  await copySelectedAccountCredential(key)
}

async function copySelectedEmailToken() {
  await copySelectedAccountEmailToken()
}

async function refreshSelectedExecutionAuth() {
  const account = selectedAccount.value
  if (!account || executionAuthRefreshInProgress.value) return
  if (hasActiveTaskForAccount(account.id)) {
    appStore.showWarning(t('accountWorkbench.execution.nonExecutableSelected'))
    return
  }
  const hasStoredExecutionAuth = account.executionAuth.trim() !== ''
  if (hasStoredExecutionAuth) {
    appStore.showSuccess(t('accountWorkbench.credentials.executionAuthAlreadyReady'))
    return
  }
  const disabledReason = selectedExecutionAuthRefreshDisabledReason.value
  if (disabledReason) {
    appStore.showWarning(disabledReason)
    return
  }
  refreshingExecutionAuthAccountId.value = account.id
  clearAccountOperationResults()
  try {
    const result = await accountWorkbenchAPI.submitTask({
      account_ids: [account.id],
      action: 'login',
      client_request_id: createRequestID(),
    })
    trackTaskLogs(result.logs ?? [])
    updateAccountsTaskStateFromLogs(result.logs ?? [])
    showTaskSubmitResultToast(result, t('accountWorkbench.credentials.refreshSubmitted', { count: result.submitted, enqueued: result.enqueued }))
    removeEnqueuedTaskAccountsFromSelection(result.logs ?? [])
    await loadData({ silent: true })
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.refresh_execution_auth', error)
    appStore.showError(extractSafeI18nErrorMessage(error, t, 'accountWorkbench.execution.errors', t('accountWorkbench.credentials.refreshFailed')))
  } finally {
    refreshingExecutionAuthAccountId.value = null
  }
}

function updateAccountFromResponse(account: UserSocialAccount) {
  const updated = mapAccount(account)
  syncAccountRowWithCurrentFilters(updated)
  upsertAccountRow(batchImportDedupAccounts, updated)
  syncOpenAccountReferences(updated, { resetEditForm: true })
}

function syncAccountRowWithCurrentFilters(updated: AccountRow) {
  if (accountMatchesCurrentFilters(updated)) {
    upsertAccountRow(accounts, updated)
  } else {
    accounts.value = accounts.value.filter(item => item.id !== updated.id)
  }
  selectedIds.value = retainExistingSelectedIds(selectedIds.value, accounts.value)
  pruneTaskLogsForCurrentAccounts()
  syncSelectedAccountFromList()
  syncEditAccountFromList()
  syncDeleteTargetFromList()
  syncProxyTargetFromList()
  syncStoreWorkbenchDialogFromSelection()
  syncExecutionConfirmFromSelection()
}

function accountMatchesCurrentFilters(account: AccountRow) {
  const keyword = searchQuery.value.trim().toLowerCase()
  const status = String(statusFilter.value || 'all')
  const platform = String(platformFilter.value || 'all')
  if (status !== 'all' && presentationWorkbenchAccountStatus(account.accountStatus) !== status) return false
  if (platform !== 'all' && normalizePlatform(account.platform) !== platform) return false
  return accountMatchesWorkbenchSearch(account, keyword, {
    accountStatus: accountStatusLabel(account.accountStatus),
    taskStatus: displayTaskStatus(account, account.taskStatus),
  })
}

function upsertAccountRow(target: { value: AccountRow[] }, updated: AccountRow) {
  const index = target.value.findIndex(item => item.id === updated.id)
  if (index >= 0) {
    target.value.splice(index, 1, updated)
    return
  }
  target.value = [updated, ...target.value]
}

function syncOpenAccountReferences(updated: AccountRow, options?: { resetEditForm?: boolean }) {
  if (selectedAccount.value?.id === updated.id) {
    selectedAccount.value = updated
  }
  if (editAccount.value?.id === updated.id) {
    editAccount.value = updated
    if (options?.resetEditForm) resetEditAccountForm(updated)
  }
  if (deleteTargetAccount.value?.id === updated.id) {
    deleteTargetAccount.value = updated
  }
  if (proxyAccount.value?.id === updated.id) {
    proxyAccount.value = updated
  }
}

function updateAccountsFromResponse(importedAccounts: UserSocialAccount[]) {
  for (const account of importedAccounts) {
    updateAccountFromResponse(account)
  }
}

function updateAccountsTaskStateFromLogs(logs: SocialTaskLog[]) {
  const latestByAccountId = new Map<number, SocialTaskLog>()
  for (const log of logs) {
    if (!log || typeof log.social_account_id !== 'number') continue
    const existing = latestByAccountId.get(log.social_account_id)
    if (!existing || taskLogIsNewer(log, existing)) {
      latestByAccountId.set(log.social_account_id, log)
    }
  }
  for (const [accountId, log] of latestByAccountId.entries()) {
    updateAccountTaskStateFromLog(accountId, log)
  }
  syncExecutionConfirmFromSelection()
}

function taskLogIsNewer(candidate: SocialTaskLog, current: SocialTaskLog) {
  const candidateTime = taskLogTime(candidate)
  const currentTime = taskLogTime(current)
  if (candidateTime !== currentTime) return candidateTime > currentTime
  return Number(candidate.id || 0) > Number(current.id || 0)
}

function updateAccountTaskStateFromLog(accountId: number, log: SocialTaskLog) {
  const index = accounts.value.findIndex(item => item.id === accountId)
  if (index < 0) return
  const current = accounts.value[index]
  const status = accountTaskStateFromLog(log, current)
  if (!status) return
  const updated = {
    ...current,
    taskStatus: status,
    taskMessage: String(log.result_message || '').trim(),
  }
  accounts.value.splice(index, 1, updated)
  syncOpenAccountReferences(updated)
}

function syncSelectedAccountFromList() {
  if (!selectedAccount.value) return
  const updated = accounts.value.find(account => account.id === selectedAccount.value?.id)
  if (updated) {
    selectedAccount.value = updated
  } else {
    detailDialogOpen.value = false
    selectedAccount.value = null
  }
}

function syncEditAccountFromList() {
  if (!editAccount.value || savingAccountEdit.value) return
  const updated = accounts.value.find(account => account.id === editAccount.value?.id)
  if (updated) {
    const hasUnsavedEdits = !editAccountFormMatchesAccount(editAccount.value)
    editAccount.value = updated
    if (!hasUnsavedEdits) resetEditAccountForm(updated)
  } else {
    editDialogOpen.value = false
    editAccount.value = null
  }
}

function syncDeleteTargetFromList() {
  if (deleting.value) return
  if (!deleteTargetAccount.value) {
    if (deleteDialogOpen.value && deleteDialogMode.value === 'batch' && selectedIds.value.length === 0) {
      closeDeleteDialog()
    }
    return
  }
  const updated = accounts.value.find(account => account.id === deleteTargetAccount.value?.id)
  if (updated) {
    deleteTargetAccount.value = updated
  } else {
    closeDeleteDialog()
  }
}

function syncProxyTargetFromList() {
  if (!proxyDialogOpen.value || savingProxy.value) return
  if (proxyDialogMode.value === 'batch') {
    if (selectedIds.value.length === 0) {
      proxyDialogOpen.value = false
      proxyAssignmentError.value = ''
    }
    return
  }
  if (!proxyAccount.value) return
  const updated = accounts.value.find(account => account.id === proxyAccount.value?.id)
  if (updated) {
    proxyAccount.value = updated
  } else {
    proxyDialogOpen.value = false
    proxyAccount.value = null
    proxyAssignmentError.value = ''
  }
}

function syncStoreWorkbenchDialogFromSelection() {
  if (!storeWorkbenchDialogOpen.value || storingWorkbenchAccounts.value) return
  if (!canStoreWorkbenchSelection.value) {
    storeWorkbenchDialogOpen.value = false
  }
}

function syncExecutionConfirmFromSelection() {
  if (!executionConfirmDialogOpen.value || submitting.value) return
  if (!canStartExecution.value) {
    executionConfirmDialogOpen.value = false
    executionSubmitError.value = ''
  }
}

function buildTaskPayload(): SubmitTaskRequest {
  const action = normalizedSelectedAction.value
  if (!action) {
    throw new Error('Task action is required before submitting.')
  }
  return {
    account_ids: [...selectedIds.value],
    action,
  }
}

function syncSelectedAction() {
  const currentAction = normalizedSelectedAction.value
  if (currentAction && !actionDisabled(currentAction)) return
  const firstUsableAction = runtimeExecutionActions.find(action => !actionDisabled(action))
  selectedAction.value = firstUsableAction ?? 'login'
}

function actionRequiresDefaultTemplate(action?: ExecutionAction | '') {
  return actionRequiresDefaultTaskTemplate(action)
}

function actionDisabled(action?: ExecutionAction | '') {
  if (!action) return true
  if (!actionRequiresDefaultTemplate(action)) return selectedPlatformUnsupported.value
  const template = defaultTemplateByAction.value.get(action)
  return !template || templateDisabled(template)
}

function templateDisabled(template: TaskTemplate) {
  return workbenchTaskTemplateDisabled(template, {
    platformUnsupported: selectedPlatformUnsupported.value,
  })
}

function templateSummary(template: TaskTemplate) {
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

function actionLabel(value?: string | number | boolean | null) {
  const normalized = String(value || '').trim()
  return normalized ? t(`accountWorkbench.actions.${normalized}`, normalized) : '-'
}

function defaultTemplateOptionDescription(template: TaskTemplate) {
  return t('accountWorkbench.execution.defaultTemplateDescription', {
    template: template.name,
    summary: templateSummary(template),
  })
}

function directActionDescription(action?: ExecutionAction | '') {
  if (action === 'login') return t('accountWorkbench.execution.loginSummary')
  if (action === 'login_check') return t('accountWorkbench.execution.loginCheckSummary')
  return ''
}

function normalizeExecutionAction(value?: string | number | boolean | null): ExecutionAction | '' {
  const normalized = String(value || '').trim()
  return (runtimeExecutionActions as readonly string[]).includes(normalized) ? normalized as ExecutionAction : ''
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

async function loadSelectedActionTemplateMediaPreview(template: TaskTemplate, previewToken = selectedActionTemplateMediaPreviewToken) {
  if (template.type === 'post') {
    const previewURLs = await Promise.all((template.params.media ?? []).map(item => resolveSelectedTemplateMediaPreviewURL(item)))
    if (previewToken !== selectedActionTemplateMediaPreviewToken) {
      previewURLs.forEach(url => revokeObjectURLSafe(url))
      return
    }
    selectedActionTemplateMediaPreview.post = previewURLs
    return
  }
  if (template.type === 'update_avatar') {
    const url = await resolveSelectedTemplateMediaPreviewURL(template.params.avatar)
    if (previewToken !== selectedActionTemplateMediaPreviewToken) {
      revokeObjectURLSafe(url)
      return
    }
    selectedActionTemplateMediaPreview.avatar = url
    return
  }
  if (template.type === 'update_banner') {
    const url = await resolveSelectedTemplateMediaPreviewURL(template.params.banner)
    if (previewToken !== selectedActionTemplateMediaPreviewToken) {
      revokeObjectURLSafe(url)
      return
    }
    selectedActionTemplateMediaPreview.banner = url
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

function clearSelectedActionTemplateMediaPreview() {
  selectedActionTemplateMediaPreview.post.forEach(url => revokeObjectURLSafe(url))
  revokeObjectURLSafe(selectedActionTemplateMediaPreview.avatar)
  revokeObjectURLSafe(selectedActionTemplateMediaPreview.banner)
  selectedActionTemplateMediaPreview.post = []
  selectedActionTemplateMediaPreview.avatar = ''
  selectedActionTemplateMediaPreview.banner = ''
}

function invalidateSelectedActionTemplateMediaPreview() {
  selectedActionTemplateMediaPreviewToken += 1
  return selectedActionTemplateMediaPreviewToken
}

function isUsableProxy(proxy: UserProxy) {
  return normalizeProxyStatus(proxy.status) === 'online' && String(proxy.endpoint || '').trim() !== ''
}

function mapAccount(account: UserSocialAccount): AccountRow {
  const accountStatus = normalizeWorkbenchAccountStatus(account.account_status)
  const taskMessage = String(account.task_message ?? '').trim()
  const taskStatus = socialAccountTaskStatusFromAccountSnapshot(
    normalizeWorkbenchTaskStatus(account.task_status),
    taskMessage,
  )
  return {
    id: account.id,
    name: String(account.name ?? '').trim(),
    platform: account.platform,
    username: String(account.username ?? '').trim() || normalizeSocialImportUsername(account.name),
    platformUserId: String(account.platform_user_id ?? '').trim() || '-',
    password: account.password ?? '',
    phone: account.phone ?? '',
    email: account.email ?? '',
    emailPassword: account.email_password ?? '',
    twoFactor: account.two_factor ?? '',
    backupCode: account.backup_code ?? '',
    emailClientId: account.email_client_id ?? '',
    emailToken: account.email_token ?? '',
    registrationIp: String(account.registration_ip ?? '').trim(),
    authCookie: account.auth_cookie ?? '',
    executionAuth: account.execution_auth ?? '',
    accountStatus,
    taskStatus,
    taskMessage,
    defaultProxySnapshot: account.default_proxy_snapshot ?? '',
    remark: account.remark ?? '',
    defaultProxyConfigured: account.default_proxy_configured === true,
    createdAt: account.created_at,
    updatedAt: account.updated_at,
  }
}

function toggleSelection(id: number) {
  if (accountActionsLocked.value) return
  const account = accounts.value.find(item => item.id === id)
  if (!account) return
  selectedIds.value = toggleSelectedId(selectedIds.value, id)
}

function toggleAllVisible() {
  if (accountActionsLocked.value) return
  selectedIds.value = toggleVisibleSelectedIds(selectedIds.value, visibleIds.value, allVisibleSelected.value)
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
  const accountStatus = normalizeWorkbenchAccountStatus(account.accountStatus)
  const taskStatus = normalizeWorkbenchTaskStatus(account.taskStatus)
  return accountStatus === 'available' && account.defaultProxyConfigured && !hasActiveTaskForAccount(account.id) && !['running', 'locked', 'disabled'].includes(taskStatus)
}

function isLoginableAccount(account: AccountRow) {
  // The login action acquires credentials, so it does not require "available".
  // If no user default proxy exists, the backend can fall back to the global proxy pool for login tasks.
  return hasAccountPassword(account) && !hasActiveTaskForAccount(account.id)
}

function hasActiveTaskForAccount(accountId: number) {
  const log = taskLogsByAccountId.value.get(accountId)
  return !!log && isActiveTaskLog(log)
}

function hasAccountPassword(account: AccountRow) {
  return String(account.password || '').trim() !== ''
}

function isStoreableWorkbenchAccount(account: AccountRow) {
  return normalizeWorkbenchAccountStatus(account.accountStatus) === 'not_stored' && normalizeWorkbenchTaskStatus(account.taskStatus) === 'pending'
}

function parseBatchImportRows(raw: string, fallbackPlatform: string): BatchImportRow[] {
  return parseSocialAccountImportTextRows(raw, fallbackPlatform, {
    duplicateMessage: t('accountWorkbench.import.errors.duplicateAccount'),
    missingAccountMessage: t('accountWorkbench.import.errors.accountRequired'),
    missingPasswordMessage: t('accountWorkbench.import.errors.passwordRequired'),
    missingCredentialMessage: t('accountWorkbench.import.errors.credentialRequired'),
  })
}

function markExistingWorkbenchDuplicates(rows: BatchImportRow[]) {
  if (existingWorkbenchImportKeys.value.size === 0) return rows
  return rows.map((row) => {
    if (!row.valid || row.status !== 'format_valid') return row
    const key = socialAccountImportDedupKey(row.account)
    if (!key || !existingWorkbenchImportKeys.value.has(key)) return row
    return {
      ...row,
      valid: false,
      status: 'existing_workbench_duplicate' as const,
      error: t('accountWorkbench.import.errors.duplicateInWorkbench'),
    }
  })
}

function batchImportRowStatusLabel(row: BatchImportRow) {
  if (row.status === 'batch_duplicate') return t('accountWorkbench.import.status.batchDuplicate')
  if (row.status === 'existing_workbench_duplicate') return t('accountWorkbench.import.status.existingWorkbenchDuplicate')
  if (row.status === 'needs_data') return t('accountWorkbench.import.status.needsData')
  return t('accountWorkbench.import.status.pendingBackendMatch')
}

function batchImportRowStatusClass(row: BatchImportRow) {
  if (row.status === 'batch_duplicate' || row.status === 'existing_workbench_duplicate') return 'text-amber-600 dark:text-amber-300'
  if (row.status === 'needs_data') return 'text-red-600 dark:text-red-300'
  if (!isBatchImportRowSubmittable(row)) return 'text-red-600 dark:text-red-300'
  return 'text-emerald-600 dark:text-emerald-300'
}

function batchImportResultStatusLabel(value?: string | null) {
  return accountWorkbenchImportResultStatusLabel(value, t)
}

function batchImportResultMessage(item: { reason?: string | null; error?: string | null }) {
  return accountWorkbenchImportResultItemMessage(item, t)
}

function batchImportResultItemLabel(item: BatchImportSocialAccountResponse['items'][number], index: number) {
  return formatSocialAccountBatchResultItemLabel(item, `#${index + 1}`)
}

function proxyAssignmentResultStatusLabel(value?: string | null) {
  return socialAccountBatchResultStatusLabel(value, 'accountWorkbench.proxy.resultStatuses', t)
}

function socialAccountBatchResultMessage(item: SocialAccountBatchResult['items'][number]) {
  return socialAccountBatchResultItemMessage(item, t)
}

function socialAccountBatchResultItemLabel(item: SocialAccountBatchResult['items'][number], index: number) {
  return formatSocialAccountBatchResultItemLabel(item, `#${index + 1}`)
}

function isBatchImportRowSubmittable(row: BatchImportRow) {
  if (!row.valid) return false
  return true
}

function credentialSummary(account: ImportSocialAccountRequest) {
  return accountImportCredentialSummary(account, {
    password: t('accountWorkbench.import.credentials.password'),
    twoFactor: t('accountWorkbench.import.credentials.twoFactor'),
    email: t('accountWorkbench.import.credentials.email'),
    authCookie: t('accountWorkbench.import.credentials.authCookie'),
    executionAuth: t('accountWorkbench.import.credentials.executionAuth'),
  })
}

async function handleBatchImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  await processBatchImportFile(file)
  input.value = ''
}

function openBatchImportFilePicker() {
  if (parsingImportFile.value || importing.value) return
  batchImportFileInput.value?.click()
}

function handleBatchImportDragEnter() {
  if (parsingImportFile.value || importing.value) return
  batchImportDragActive.value = true
}

function handleBatchImportDragLeave() {
  batchImportDragActive.value = false
}

async function handleBatchImportFileDrop(event: DragEvent) {
  batchImportDragActive.value = false
  const file = event.dataTransfer?.files?.[0]
  if (!file) return
  await processBatchImportFile(file)
}

async function processBatchImportFile(file: File) {
  if (parsingImportFile.value || importing.value) {
    return
  }
  parsingImportFile.value = true
  batchImportError.value = ''
  batchImportText.value = ''
  batchImportResult.value = null
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
    batchImportFileName.value = file.name
    batchImportFileRows.value = parseBatchImportRows(fileText, importForm.platform.trim())
    if (batchImportFileRows.value.length === 0) {
      batchImportError.value = t('accountWorkbench.import.errors.emptyFile')
    }
  } catch (error) {
    recordClientDiagnostic('account_workbench.unified.batch_import_file', error)
    batchImportError.value = t('accountWorkbench.import.errors.fileReadFailed')
    batchImportFileName.value = ''
    batchImportFileRows.value = []
  } finally {
    parsingImportFile.value = false
    batchImportDragActive.value = false
  }
}

async function importWorkbookToText(file: File) {
  const buffer = await file.arrayBuffer()
  const workbook = XLSX.read(buffer, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  if (!sheetName) return ''
  const rows = XLSX.utils.sheet_to_json<SocialAccountImportWorkbookCell[]>(workbook.Sheets[sheetName], { header: 1, defval: '' })
  return socialAccountImportWorkbookRowsToText(rows)
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

function isTwitterPlatform(platform: string) {
  return normalizePlatform(platform) === 'x_twitter'
}

function platformLabel(value?: string | null) {
  return socialPlatformLabel(value, { emptyLabel: t('common.unknown'), unknownCase: 'upper' })
}

function displayTaskStatus(account: AccountRow, fallback: string) {
  const log = taskLogsByAccountId.value.get(account.id)
  const status = normalizeWorkbenchTaskLogStatus(log)
  if (isDisplayableTaskLogStatus(status)) return status
  return fallback
}

function displayTaskStatusIsActive(account: AccountRow) {
  return isActiveTaskLog(taskLogsByAccountId.value.get(account.id))
}

function accountTaskStateFromLog(log: SocialTaskLog, account: AccountRow) {
  return socialAccountTaskStatusFromTaskResult(log, account.taskStatus)
}

function isActiveTaskLog(log?: SocialTaskLog | null) {
  return isActiveWorkbenchTaskLog(log)
}

function isDisplayableTaskLogStatus(status?: string | null) {
  const normalized = normalizeWorkbenchTaskStatus(status)
  return normalized === 'pending' || normalized === 'running' || normalized === 'success' || normalized === 'failed'
}

function compareTaskLogsDesc(left: SocialTaskLog, right: SocialTaskLog) {
  const timeDiff = taskLogTime(right) - taskLogTime(left)
  if (timeDiff !== 0) return timeDiff
  return Number(right.id || 0) - Number(left.id || 0)
}

function taskLogTime(log: SocialTaskLog) {
  const executed = log.executed_at ? Date.parse(log.executed_at) : 0
  if (Number.isFinite(executed) && executed > 0) return executed
  const created = log.created_at ? Date.parse(log.created_at) : 0
  return Number.isFinite(created) ? created : 0
}

function accountStatusLabel(value?: string | null) {
  const normalized = normalizeWorkbenchAccountStatus(value)
  return t(`accountWorkbench.accountStatus.${normalized}`, workbenchStatusFallbackText(value))
}

function taskStatusLabel(value?: string | null) {
  const normalized = normalizeWorkbenchTaskStatus(value)
  return t(`accountWorkbench.taskStatus.${normalized}`, workbenchStatusFallbackText(value))
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
  editAccountForm.registrationIp = account.registrationIp
  editAccountForm.authCookie = account.authCookie
  editAccountForm.executionAuth = account.executionAuth
  editAccountForm.remark = account.remark
}

function editAccountFormMatchesAccount(account: AccountRow) {
  return editableAccountFormString(editAccountForm.password) === editableAccountFormString(account.password) &&
    trimEditableField(editAccountForm.phone) === trimEditableField(account.phone) &&
    trimEditableField(editAccountForm.email) === trimEditableField(account.email) &&
    editableAccountFormString(editAccountForm.emailPassword) === editableAccountFormString(account.emailPassword) &&
    editableAccountFormString(editAccountForm.twoFactor) === editableAccountFormString(account.twoFactor) &&
    editableAccountFormString(editAccountForm.backupCode) === editableAccountFormString(account.backupCode) &&
    editableAccountFormString(editAccountForm.emailClientId) === editableAccountFormString(account.emailClientId) &&
    editableAccountFormString(editAccountForm.emailToken) === editableAccountFormString(account.emailToken) &&
    trimEditableField(editAccountForm.registrationIp) === trimEditableField(account.registrationIp) &&
    editableAccountFormString(editAccountForm.authCookie) === editableAccountFormString(account.authCookie) &&
    editableAccountFormString(editAccountForm.executionAuth) === editableAccountFormString(account.executionAuth) &&
    editableAccountFormString(editAccountForm.remark) === editableAccountFormString(account.remark)
}

function editableAccountFormString(value?: string | null) {
  return String(value ?? '')
}

function createRequestID() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `social-task-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

</script>
