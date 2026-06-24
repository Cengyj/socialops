<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <LoadErrorBanner
            v-if="loadError"
            :title="t('admin.socialAccountWorkbench.failedToLoad')"
            :message="loadError"
            :retry-label="t('common.retry')"
            @retry="loadAccounts"
          />

          <SocialAccountStatsGrid
            :stats="stats"
            test-id-prefix="total-account-stat"
            grid-class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4"
          />

          <div data-testid="total-accounts-toolbar" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
            <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-2 2xl:flex-row 2xl:items-center">
                <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2 xl:flex-nowrap">
                  <SearchInput v-model="searchQuery" :placeholder="t('admin.socialAccountWorkbench.searchPlaceholder')" class="w-full shrink-0 sm:w-[220px] xl:w-[200px] 2xl:w-[300px]" />
                  <Select v-model="accountStatusFilter" :options="accountStatusOptions" class="w-full shrink-0 sm:w-[168px] xl:w-[148px] 2xl:w-[184px]" />
                  <Select v-model="assignmentFilter" :options="assignmentOptions" class="w-full shrink-0 sm:w-[152px] xl:w-[132px] 2xl:w-[168px]" />
                  <Select v-model="importPlatform" :options="importPlatformOptions" class="w-full shrink-0 sm:w-[152px] xl:w-[132px] 2xl:w-[168px]" />
                  <div class="hidden h-6 w-px shrink-0 bg-gray-200 dark:bg-dark-700 xl:block"></div>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="refreshAccountsButtonTitle"
                    :title="refreshAccountsButtonTitle"
                    :disabled="loading"
                    @click="loadAccounts"
                  >
                    <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                    <span class="min-w-0 truncate xl:hidden 2xl:inline">{{ t('common.refresh') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="importAccountsButtonTitle"
                    :title="importAccountsButtonTitle"
                    :disabled="loading || importingAccounts"
                    @click="triggerImport"
                  >
                    <Icon name="upload" size="sm" :class="importingAccounts ? 'animate-spin' : ''" />
                    <span class="min-w-0 truncate xl:hidden 2xl:inline">{{ importingAccounts ? t('common.processing') : t('admin.socialAccountWorkbench.toolbar.importAccounts') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="exportAccountsButtonTitle"
                    :title="exportAccountsButtonTitle"
                    :disabled="loading || exportingAccounts"
                    @click="exportAccounts"
                  >
                    <Icon name="download" size="sm" :class="exportingAccounts ? 'animate-spin' : ''" />
                    <span class="min-w-0 truncate xl:hidden 2xl:inline">{{ exportingAccounts ? t('common.processing') : t('admin.socialAccountWorkbench.toolbar.exportRecords') }}</span>
                  </button>
                  <input ref="importFileInput" type="file" accept=".csv,.json,.xlsx" class="hidden" @change="handleImportFile" />
                </div>

                <div class="flex w-full shrink-0 flex-col gap-2 sm:flex-row xl:ml-auto xl:w-auto xl:items-center 2xl:ml-2">
                  <div class="flex h-10 w-full shrink-0 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800 sm:w-auto">
                    <div class="flex min-w-[132px] flex-1 items-center justify-center whitespace-nowrap bg-primary-50 px-3 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 sm:flex-none xl:min-w-[112px] xl:px-2 2xl:min-w-[132px] 2xl:px-3">
                      {{ t('admin.socialAccountWorkbench.executionBar.selectedCount', { count: selectedIds.length }) }}
                    </div>
                    <button
                      type="button"
                      class="flex h-full w-10 shrink-0 items-center justify-center border-l border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-gray-100"
                      :aria-label="clearSelectionButtonTitle"
                      :title="clearSelectionButtonTitle"
                      :disabled="loading || !hasSelection"
                      @click="clearSelection"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>
                  <button
                    type="button"
                    class="btn btn-primary btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4"
                    :aria-label="assignSelectedButtonTitle"
                    :title="assignSelectedButtonTitle"
                    :disabled="loading || !canAssignSelected || assigning"
                    @click="openAssignDialog"
                  >
                    <Icon name="userPlus" size="sm" />
                    <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.actions.assign') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4"
                    :aria-label="reclaimSelectedButtonTitle"
                    :title="reclaimSelectedButtonTitle"
                    :disabled="loading || !canReclaimSelected || reclaiming"
                    @click="openReclaimDialog"
                  >
                    <Icon name="swap" size="sm" />
                    <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.actions.reclaim') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-danger btn-sm h-10 w-full min-w-0 max-w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4"
                    :aria-label="deleteSelectedButtonTitle"
                    :title="deleteSelectedButtonTitle"
                    :disabled="loading || !hasSelection || deleting"
                    @click="openDeleteDialog"
                  >
                    <Icon name="trash" size="sm" />
                    <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.actions.delete') }}</span>
                  </button>
                </div>
              </div>

              <div v-if="hasSelection" class="flex flex-col gap-2 border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex min-w-0 flex-wrap items-center gap-2">
                  <span class="rounded-full bg-gray-100 px-2.5 py-1 font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                    {{ t('admin.socialAccountWorkbench.stats.unassigned') }} {{ selectedUnassignedCount }}
                  </span>
                  <span class="rounded-full bg-gray-100 px-2.5 py-1 font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                    {{ t('admin.socialAccountWorkbench.stats.assigned') }} {{ selectedAssignedCount }}
                  </span>
                </div>
                <span v-if="selectedAssignedCount > 0" class="min-w-0 text-amber-600 dark:text-amber-300">
                  {{ t('admin.socialAccountWorkbench.toasts.assignRequiresUnassigned', { count: selectedAssignedCount }) }}
                </span>
              </div>
              <SocialAccountBatchResultPanel
                v-if="batchOperationResult"
                test-id="total-accounts-batch-result"
                :summary="batchOperationSummary(batchOperationResult)"
                :items="batchOperationResultItems"
                :remaining-count="remainingBatchOperationResultItemCount"
                :dismiss-label="t('admin.socialAccountWorkbench.batchResult.dismiss')"
                :rows-more-text="t('admin.socialAccountWorkbench.batchResult.rowsMore', { count: remainingBatchOperationResultItemCount })"
                :item-label="batchOperationItemLabel"
                :status-label="batchOperationStatusLabel"
                :item-message="batchOperationItemMessage"
                :row-tone-class="batchOperationResultRowToneClass"
                @dismiss="batchOperationResult = null"
              />
              <SocialAccountBatchResultPanel
                v-if="importOperationResult"
                test-id="total-accounts-import-result"
                :summary="importOperationSummary(importOperationResult)"
                :items="importOperationResultItems"
                :remaining-count="remainingImportOperationResultItemCount"
                :dismiss-label="t('admin.socialAccountWorkbench.importResult.dismiss')"
                :rows-more-text="t('admin.socialAccountWorkbench.batchResult.rowsMore', { count: remainingImportOperationResultItemCount })"
                :item-label="batchOperationItemLabel"
                :status-label="importOperationStatusLabel"
                :item-message="importOperationItemMessage"
                :row-tone-class="batchOperationResultRowToneClass"
                @dismiss="importOperationResult = null"
              />
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          class="total-accounts-table"
          :columns="columns"
          :data="filteredAccounts"
          :loading="loading"
          row-key="id"
          default-sort-key="id"
          default-sort-order="asc"
          :estimate-row-height="72"
          :sticky-first-column="false"
          :sticky-actions-column="true"
        >
          <template #header-select>
            <div class="flex justify-center">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
                :checked="allVisibleSelected"
                :disabled="loading"
                :indeterminate="someVisibleSelected"
                @click.stop
                @change="toggleAllVisible"
              />
            </div>
          </template>
          <template #cell-select="{ row }">
            <div class="flex justify-center">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
                :checked="isSelected(row.id)"
                :disabled="loading"
                @click.stop
                @change="toggleSelection(row.id)"
              />
            </div>
          </template>
          <template #cell-account="{ row }">
            <button class="flex w-full min-w-0 max-w-full items-center gap-3 text-left disabled:cursor-not-allowed disabled:opacity-60 md:min-w-[220px] md:max-w-[260px]" :disabled="loading" @click="openDetailDialog(row)">
              <span :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border text-xs font-semibold', platformAvatarClass(row.platform)]">
                {{ platformInitial(row.platform) }}
              </span>
              <span class="min-w-0">
                <span class="block truncate font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">{{ row.account }}</span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">{{ row.username || '-' }}</span>
              </span>
            </button>
          </template>
          <template #cell-platform="{ value }">
            <span class="inline-block max-w-[110px] truncate rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200" :title="platformLabel(String(value || ''))">
              {{ platformLabel(String(value || '')) }}
            </span>
          </template>
          <template #cell-email="{ value }">
            <span class="block w-full max-w-full truncate text-sm text-gray-700 dark:text-gray-300 md:w-[200px]" :title="String(value || '')">{{ value || '-' }}</span>
          </template>
          <template #cell-defaultProxySnapshot="{ row }">
            <span class="block w-full max-w-full truncate text-sm text-gray-700 dark:text-gray-300 md:w-[220px]" :title="row.defaultProxySnapshot">
              {{ defaultProxySnapshotListLabel(row.defaultProxySnapshot) }}
            </span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', totalPoolAccountStatusBadgeClass(String(value))]">{{ t(`admin.socialAccountWorkbench.accountStatus.${value}`) }}</span>
          </template>
          <template #cell-assignedUser="{ row }">
            <div class="min-w-0 max-w-[220px]">
              <span :class="['badge min-w-0 max-w-full truncate', row.assignedUser ? 'badge-primary' : 'badge-warning']" :title="row.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned')">
                {{ row.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}
              </span>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-start gap-2">
              <button class="btn btn-secondary h-9 w-9 px-0" :aria-label="rowDetailButtonTitle" :title="rowDetailButtonTitle" :disabled="loading" @click="openDetailDialog(row)">
                <Icon name="eye" size="sm" />
              </button>
              <button class="btn btn-secondary h-9 w-9 px-0" :aria-label="rowEditButtonTitle" :title="rowEditButtonTitle" :disabled="loading" @click="openEditDialog(row)">
                <Icon name="edit" size="sm" />
              </button>
            </div>
          </template>
          <template #empty>
            <div class="flex flex-col items-center py-8 text-center">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ isTotalAccountPoolEmpty ? t('admin.socialAccountWorkbench.empty.title') : t('admin.socialAccountWorkbench.noResults.title') }}
              </p>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
                {{ isTotalAccountPoolEmpty ? t('admin.socialAccountWorkbench.empty.description') : t('admin.socialAccountWorkbench.noResults.description') }}
              </p>
              <div v-if="isTotalAccountPoolEmpty" class="mt-4 flex flex-wrap justify-center gap-2">
                <button
                  type="button"
                  class="btn btn-primary btn-sm min-w-0 max-w-full justify-center"
                  :aria-label="importAccountsButtonTitle"
                  :title="importAccountsButtonTitle"
                  :disabled="loading || importingAccounts"
                  @click="triggerImport"
                >
                  <Icon name="upload" size="sm" :class="importingAccounts ? 'animate-spin' : ''" />
                  <span class="min-w-0 truncate">{{ importingAccounts ? t('common.processing') : t('admin.socialAccountWorkbench.toolbar.importAccounts') }}</span>
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm min-w-0 max-w-full justify-center"
                  :aria-label="refreshAccountsButtonTitle"
                  :title="refreshAccountsButtonTitle"
                  @click="loadAccounts"
                >
                  <Icon name="refresh" size="sm" />
                  <span class="min-w-0 truncate">{{ t('common.refresh') }}</span>
                </button>
              </div>
              <button
                v-else
                type="button"
                class="btn btn-secondary btn-sm mt-4 min-w-0 max-w-full justify-center"
                :aria-label="t('admin.socialAccountWorkbench.filters.clear')"
                :title="t('admin.socialAccountWorkbench.filters.clear')"
                @click="clearAccountFilters"
              >
                <Icon name="x" size="sm" />
                <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.filters.clear') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="closeDetailDialog">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.tabs.poolDescription') }}
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
          test-id-prefix="total-account"
          :title="t('accountWorkbench.credentials.title')"
          :hint="t('accountWorkbench.credentials.previewHint')"
          :copy-label="t('accountWorkbench.credentials.copy')"
          :empty-copy-label="t('accountWorkbench.credentials.emptyCopy')"
          @copy="copySelectedCredential"
        />
        <SocialAccountTaskMessagePanel
          v-if="selectedAccount.taskMessage"
          :message="selectedAccount.taskMessage"
          :status="selectedAccount.taskStatus"
        />
      </div>
      <template #footer>
        <button class="btn btn-primary" @click="closeDetailDialog">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="editDialogOpen" :title="t('admin.socialAccountWorkbench.editTitle')" width="wide" @close="closeEditDialog">
      <div v-if="selectedAccount" class="space-y-4">
        <div data-testid="total-account-edit-identity" class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-700/60">
          <div class="mb-2 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.edit.identityTitle') }}</div>
          <div class="grid gap-2 sm:grid-cols-3">
            <div v-for="item in editIdentityItems" :key="item.label">
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
            </div>
          </div>
          <p class="mt-3 min-w-0 break-words text-xs text-gray-500 dark:text-gray-400" :title="t('accountWorkbench.edit.identityHint')">{{ t('accountWorkbench.edit.identityHint') }}</p>
        </div>
        <div v-if="accountFormError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="accountFormError">
          {{ accountFormError }}
        </div>

        <div data-testid="total-account-edit-form" class="space-y-3">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.credentials') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-password">{{ t('admin.socialAccountWorkbench.form.password') }}</label>
              <input id="total-account-edit-password" v-model="accountForm.password" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-phone">{{ t('admin.socialAccountWorkbench.form.phone') }}</label>
              <input id="total-account-edit-phone" v-model="accountForm.phone" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email">{{ t('admin.socialAccountWorkbench.form.email') }}</label>
              <input id="total-account-edit-email" v-model="accountForm.email" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-password">{{ t('admin.socialAccountWorkbench.form.emailPassword') }}</label>
              <input id="total-account-edit-email-password" v-model="accountForm.emailPassword" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-two-factor">{{ t('admin.socialAccountWorkbench.form.twoFactor') }}</label>
              <input id="total-account-edit-two-factor" v-model="accountForm.twoFactor" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-backup-code">{{ t('admin.socialAccountWorkbench.form.backupCode') }}</label>
              <input id="total-account-edit-backup-code" v-model="accountForm.backupCode" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-client-id">{{ t('admin.socialAccountWorkbench.form.emailClientId') }}</label>
              <input id="total-account-edit-email-client-id" v-model="accountForm.emailClientId" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-token">{{ t('admin.socialAccountWorkbench.form.emailToken') }}</label>
              <input id="total-account-edit-email-token" v-model="accountForm.emailToken" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-registration-ip">{{ t('admin.socialAccountWorkbench.form.registrationIp') }}</label>
              <input id="total-account-edit-registration-ip" v-model="accountForm.registrationIp" type="text" class="input mt-2 bg-white dark:bg-dark-800" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-auth-cookie">{{ t('admin.socialAccountWorkbench.form.authCookie') }}</label>
              <textarea id="total-account-edit-auth-cookie" v-model="accountForm.authCookie" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800" :disabled="savingAccount"></textarea>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-execution-auth">{{ t('admin.socialAccountWorkbench.form.executionAuth') }}</label>
              <textarea id="total-account-edit-execution-auth" v-model="accountForm.executionAuth" class="input mt-2 min-h-[120px] bg-white dark:bg-dark-800" :disabled="savingAccount"></textarea>
            </div>
          </div>

          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.operations') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="mb-2 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.accountStatus') }}</label>
              <Select v-model="accountForm.accountStatus" :options="accountStatusOptionsWithoutAll" :disabled="savingAccount" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-remark">{{ t('admin.socialAccountWorkbench.form.remark') }}</label>
              <textarea id="total-account-edit-remark" v-model="accountForm.remark" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800" :disabled="savingAccount"></textarea>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="accountEditCancelButtonTitle"
          :title="accountEditCancelButtonTitle"
          :disabled="savingAccount"
          @click="closeEditDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="accountEditSubmitButtonLabel"
          :title="accountEditSubmitButtonTitle"
          :disabled="!canSubmitAccount || savingAccount"
          @click="submitEditDialog"
        >
          <Icon name="refresh" size="sm" :class="savingAccount ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ accountEditSubmitButtonLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="assignDialogOpen" :title="t('admin.socialAccountWorkbench.assignDialog.title')" width="wide" @close="closeAssignDialog">
      <div class="space-y-4">
        <div
          class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="assignDialogHint"
        >
          {{ assignDialogHint }}
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
              class="max-w-full break-all rounded-full border border-white bg-white px-3 py-1 text-xs font-medium text-gray-700 shadow-sm sm:truncate dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
              :title="account.account"
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
                <div
                  v-if="targetUserLoadError"
                  class="mb-2 min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
                  role="alert"
                  aria-live="assertive"
                  aria-atomic="true"
                  :title="targetUserLoadError"
                >
                  {{ targetUserLoadError }}
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
                      <div class="break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="user.email">{{ user.email }}</div>
                      <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">#{{ user.id }} · {{ user.role }}</div>
                    </div>
                    <span :class="['shrink-0 rounded-full px-2 py-0.5 text-xs font-medium', user.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300']">
                      {{ t(`admin.socialAccountWorkbench.assignDialog.userStatus.${user.status}`) }}
                    </span>
                  </div>
                  <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: visibleAssignedCountForUser(user.id) }) }}
                  </div>
                </button>
                <div v-if="!targetUserLoadError && filteredTargetUsers.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
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
                <div class="break-all text-lg font-semibold text-gray-900 sm:truncate dark:text-white" :title="selectedTargetUser.email">{{ selectedTargetUser.email }}</div>
                <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">#{{ selectedTargetUser.id }} · {{ selectedTargetUser.role }}</div>
              </div>
              <div class="flex flex-wrap gap-2 text-xs">
                <span class="rounded-full bg-white px-2.5 py-1 font-medium text-gray-700 shadow-sm dark:bg-dark-700 dark:text-gray-200">{{ selectedTargetUser.status }}</span>
                <span class="rounded-full bg-white px-2.5 py-1 font-medium text-gray-700 shadow-sm dark:bg-dark-700 dark:text-gray-200">{{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: visibleAssignedCountForUser(selectedTargetUser.id) }) }}</span>
              </div>
            </div>
            <div v-else class="mt-3 rounded-lg border border-dashed border-primary-200 bg-white/70 p-4 text-sm text-gray-500 dark:border-primary-800/60 dark:bg-dark-800/60 dark:text-gray-400">
              {{ t('admin.socialAccountWorkbench.assignDialog.noSelectedUserPrompt') }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="assignDialogCancelButtonTitle"
          :title="assignDialogCancelButtonTitle"
          @click="closeAssignDialog()"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="assignReviewButtonTitle"
          :title="assignReviewButtonTitle"
          :disabled="loading || !targetUser || !canAssignSelected || assigning"
          @click="openAssignConfirmDialog"
        >
          <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.assignDialog.reviewButton') }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="assignConfirmDialogOpen" :title="t('admin.socialAccountWorkbench.assignDialog.confirmTitle')" width="normal" @close="closeAssignConfirmDialog">
      <div class="space-y-4">
        <div
          class="min-w-0 break-words rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="assignConfirmHint"
        >
          {{ assignConfirmHint }}
        </div>
        <div
          v-if="assignDialogError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="assignDialogError"
        >
          {{ assignDialogError }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.assignDialog.accountSummary') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
            <div class="mt-3 flex flex-wrap gap-2">
              <span
                v-for="account in selectedAccountPreview"
                :key="account.id"
                class="max-w-full break-all rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 sm:truncate dark:bg-dark-700 dark:text-gray-200"
                :title="account.account"
              >
                {{ account.account }}
              </span>
              <span v-if="remainingSelectedAccountCount > 0" class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
                {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
              </span>
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.assignDialog.targetUser') }}</div>
            <div v-if="selectedTargetUser" class="mt-2 min-w-0">
              <div class="break-all text-base font-semibold text-gray-900 sm:truncate dark:text-white" :title="selectedTargetUser.email">{{ selectedTargetUser.email }}</div>
              <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">#{{ selectedTargetUser.id }} · {{ selectedTargetUser.role }}</div>
              <div class="mt-3 rounded-lg bg-gray-50 px-3 py-2 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: visibleAssignedCountForUser(selectedTargetUser.id) }) }}
              </div>
            </div>
          </div>
        </div>
        <div
          class="min-w-0 break-words rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="t('admin.socialAccountWorkbench.assignDialog.impactHint')"
        >
          {{ t('admin.socialAccountWorkbench.assignDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="assignBackButtonTitle"
          :title="assignBackButtonTitle"
          :disabled="assigning"
          @click="closeAssignConfirmDialog()"
        >
          <span class="min-w-0 truncate">{{ t('admin.socialAccountWorkbench.assignDialog.backToSelect') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="assignConfirmButtonTitle"
          :title="assignConfirmButtonTitle"
          :disabled="loading || !targetUser || !canAssignSelected || assigning"
          @click="confirmAssignDialog"
        >
          <Icon name="refresh" size="sm" :class="assigning ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ assignConfirmButtonTitle }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="reclaimDialogOpen" :title="t('admin.socialAccountWorkbench.reclaimDialog.title')" width="normal" @close="closeReclaimDialog">
      <div class="space-y-4">
        <div
          class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="reclaimDialogHint"
        >
          {{ reclaimDialogHint }}
        </div>
        <div
          v-if="reclaimDialogError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="reclaimDialogError"
        >
          {{ reclaimDialogError }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.stats.assigned') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedAssignedCount }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.reclaimDialog.assignedImpact') }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.stats.unassigned') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedUnassignedCount }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.reclaimDialog.unassignedImpact') }}</div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 flex items-center justify-between gap-2">
            <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.socialAccountWorkbench.reclaimDialog.accountSummary') }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}</div>
          </div>
          <div class="grid gap-2 text-sm">
            <div v-for="account in selectedAccountPreview" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 max-w-[52%] break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="account.account">{{ account.account }}</span>
              <span class="min-w-0 max-w-[48%] break-all text-right text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned')">{{ account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}</span>
            </div>
            <div v-if="remainingSelectedAccountCount > 0" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="reclaimDialogCancelButtonTitle"
          :title="reclaimDialogCancelButtonTitle"
          :disabled="reclaiming"
          @click="closeReclaimDialog()"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="reclaimConfirmButtonTitle"
          :title="reclaimConfirmButtonTitle"
          :disabled="loading || !canReclaimSelected || reclaiming"
          @click="reclaimSelectedAccounts"
        >
          <Icon name="refresh" size="sm" :class="reclaiming ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ reclaimConfirmButtonTitle }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('admin.socialAccountWorkbench.deleteDialog.title')" width="normal" @close="closeDeleteDialog">
      <div class="space-y-4">
        <div
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="deleteDialogHint"
        >
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
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.stats.assigned') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedAssignedCount }}</div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.stats.unassigned') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedUnassignedCount }}</div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.socialAccountWorkbench.deleteDialog.accountSummary') }}</div>
          <div class="grid gap-2 text-sm">
            <div v-for="account in selectedAccountPreview" :key="account.id" class="flex min-w-0 items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700">
              <span class="min-w-0 max-w-[52%] break-all font-medium text-gray-900 sm:truncate dark:text-white" :title="account.account">{{ account.account }}</span>
              <span class="min-w-0 max-w-[48%] break-all text-right text-xs text-gray-500 sm:truncate dark:text-gray-400" :title="account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned')">{{ account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}</span>
            </div>
            <div v-if="remainingSelectedAccountCount > 0" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 dark:bg-red-900/20 dark:text-red-300">
              {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
            </div>
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300" role="status" aria-live="polite" aria-atomic="true" :title="t('admin.socialAccountWorkbench.deleteDialog.impactHint')">
          {{ t('admin.socialAccountWorkbench.deleteDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="deleteDialogCancelButtonTitle"
          :title="deleteDialogCancelButtonTitle"
          :disabled="deleting"
          @click="closeDeleteDialog()"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-danger min-w-0 max-w-full justify-center"
          :aria-label="deleteConfirmButtonTitle"
          :title="deleteConfirmButtonTitle"
          :disabled="loading || !hasSelection || deleting"
          @click="confirmDeleteDialog"
        >
          <Icon name="refresh" size="sm" :class="deleting ? 'animate-spin' : 'hidden'" />
          <span class="min-w-0 truncate">{{ deleteConfirmButtonTitle }}</span>
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
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
import SocialAccountBatchResultPanel from '@/components/accounts/SocialAccountBatchResultPanel.vue'
import SocialAccountStatsGrid from '@/components/accounts/SocialAccountStatsGrid.vue'
import SocialAccountTaskMessagePanel from '@/components/accounts/SocialAccountTaskMessagePanel.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { SocialAccount } from '@/api/admin'
import type { TotalAccountExportParams, TotalAccountImportResult } from '@/api/admin/totalAccounts'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import {
  accountWorkbenchImportResultItemMessage,
  accountWorkbenchImportResultStatusLabel,
  accountWorkbenchImportResultSummaryParams,
} from '@/utils/accountWorkbenchImportResult'
import {
  collectSucceededBatchItemIds,
  formatSocialAccountBatchResultName,
  socialAccountBatchResultItemMessage,
  socialAccountBatchResultRowToneClass,
  socialAccountBatchResultStatusLabel,
  socialAccountBatchResultToastParams,
  showSocialAccountBatchResultToast,
} from '@/utils/accountWorkbenchBatchResult'
import { downloadBlob } from '@/utils/browser'
import { createListPreview } from '@/utils/listPreview'
import {
  removeSelectedIds,
  retainExistingSelectedIds,
  selectedRowsById,
  toggleSelectedId,
  toggleVisibleSelectedIds,
  visibleSelectionState,
} from '@/utils/selection'
import { useSocialAccountCredentialPreview } from '@/composables/useSocialAccountCredentialPreview'
import { useSocialAccountCredentialCopy } from '@/composables/useSocialAccountCredentialCopy'
import { buildTotalAccountEditPayload, trimEditableField } from '@/utils/accountWorkbenchEditPayload'
import { formatAccountWorkbenchDate } from '@/utils/accountWorkbenchDate'
import {
  socialPlatformAvatarClass as platformAvatarClass,
  socialPlatformInitial as platformInitial,
  socialPlatformLabel as platformLabel,
} from '@/utils/socialPlatformDisplay'
import {
  normalizeKnownWorkbenchAccountStatus,
  normalizeWorkbenchTaskStatus,
  totalPoolAccountStatusBadgeClass,
  type WorkbenchAccountStatus,
} from '@/utils/accountWorkbenchStatus'
import { socialAccountTaskStatusFromAccountSnapshot } from '@/utils/socialAccountTaskStatus'
import {
  totalAccountBatchOperationSummaryParams,
  useTotalAccountOperationResultPreview,
} from './totalAccountOperationResults'
import { createTotalAccountErrorMessages } from './totalAccountErrorMessages'
import {
  totalAccountAssignBackButtonTitle as buildAssignBackButtonTitle,
  totalAccountAssignConfirmButtonTitle as buildAssignConfirmButtonTitle,
  totalAccountAssignReviewButtonTitle as buildAssignReviewButtonTitle,
  totalAccountAssignSelectedButtonTitle as buildAssignSelectedButtonTitle,
  totalAccountClearSelectionButtonTitle as buildClearSelectionButtonTitle,
  totalAccountDeleteConfirmButtonTitle as buildDeleteConfirmButtonTitle,
  totalAccountDeleteSelectedButtonTitle as buildDeleteSelectedButtonTitle,
  totalAccountDialogCancelButtonTitle as buildDialogCancelButtonTitle,
  totalAccountEditSubmitButtonTitle as buildAccountEditSubmitButtonTitle,
  totalAccountExportButtonTitle as buildExportAccountsButtonTitle,
  totalAccountImportButtonTitle as buildImportAccountsButtonTitle,
  totalAccountReclaimConfirmButtonTitle as buildReclaimConfirmButtonTitle,
  totalAccountReclaimSelectedButtonTitle as buildReclaimSelectedButtonTitle,
  totalAccountRefreshButtonTitle as buildRefreshAccountsButtonTitle,
  totalAccountRowDetailButtonTitle as buildRowDetailButtonTitle,
  totalAccountRowEditButtonTitle as buildRowEditButtonTitle,
  totalAccountSubmitButtonLabel as buildSubmitButtonLabel,
} from './totalAccountActionTitles'
import type { SocialAccountBatchItemResult, SocialAccountBatchResult } from '@/api/accountWorkbench'
import type { SocialAccountCredentialPreview, SocialAccountCredentialPreviewKey } from '@/utils/socialAccountCredentials'

type AccountStatus = WorkbenchAccountStatus
type BatchOperationAction = 'assigned' | 'reclaimed' | 'deleted'

interface AccountRow {
  id: number
  account: string
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
  defaultProxySnapshot: string
  accountStatus: AccountStatus
  taskStatus: string
  taskMessage: string
  assignedUserId: number | null
  assignedUser: string | null
  remark: string
  createdAt: string
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

interface BatchOperationResultState {
  action: BatchOperationAction
  result: SocialAccountBatchResult
  targetUser?: string
}

const { t } = useI18n()
const appStore = useAppStore()
const { buildCredentialPreviews, buildDeliveryCredentialItems } = useSocialAccountCredentialPreview()

const totalAccountErrorMessages = computed(() => createTotalAccountErrorMessages(t))

const searchQuery = ref('')
const accountStatusFilter = ref('all')
const assignmentFilter = ref('all')
const selectedIds = ref<number[]>([])
const accounts = ref<AccountRow[]>([])
const users = ref<AdminUser[]>([])
const loading = ref(false)
const importingAccounts = ref(false)
const exportingAccounts = ref(false)
const loadError = ref('')
const targetUserLoadError = ref('')
const selectedAccount = ref<AccountRow | null>(null)
const {
  copySelectedCredential: copySelectedAccountCredential,
  copySelectedEmailToken: copySelectedAccountEmailToken,
} = useSocialAccountCredentialCopy({
  getAccount: () => selectedAccount.value,
  credentialDiagnosticContext: 'admin.total_accounts.copy_credential',
  emailTokenDiagnosticContext: 'admin.total_accounts.copy_email_token',
})
const selectedAccountId = ref<number | null>(null)
const detailDialogOpen = ref(false)
const editDialogOpen = ref(false)
const accountFormError = ref('')
const assignDialogOpen = ref(false)
const assignConfirmDialogOpen = ref(false)
const assignDialogError = ref('')
const reclaimDialogOpen = ref(false)
const reclaimDialogError = ref('')
const deleteDialogOpen = ref(false)
const deleteDialogError = ref('')
const savingAccount = ref(false)
const assigning = ref(false)
const reclaiming = ref(false)
const deleting = ref(false)
const targetUser = ref('')
const targetUserSearch = ref('')
const importPlatform = ref('x_twitter')
const importFileInput = ref<HTMLInputElement | null>(null)
const batchOperationResult = ref<BatchOperationResultState | null>(null)
const importOperationResult = ref<TotalAccountImportResult | null>(null)
let latestLoadRequestID = 0
let filterReloadTimer: ReturnType<typeof setTimeout> | undefined

const accountForm = reactive({
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
  accountStatus: 'pending_check',
  remark: '',
})

onMounted(async () => {
  await loadUsers()
  await loadAccounts()
})

onUnmounted(() => {
  if (filterReloadTimer) clearTimeout(filterReloadTimer)
})

watch([searchQuery, accountStatusFilter, assignmentFilter], () => {
  if (filterReloadTimer) clearTimeout(filterReloadTimer)
  filterReloadTimer = setTimeout(() => {
    void loadAccounts()
  }, 250)
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px] text-center' },
  { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), sortable: true, class: 'w-[84px] min-w-[84px]' },
  { key: 'account', label: t('admin.socialAccountWorkbench.columns.account'), sortable: true, class: 'min-w-[240px]' },
  { key: 'platform', label: t('admin.socialAccountWorkbench.columns.platform'), sortable: true, class: 'min-w-[118px]' },
  { key: 'email', label: t('admin.socialAccountWorkbench.columns.email'), sortable: true, class: 'min-w-[210px] max-w-[220px]' },
  { key: 'defaultProxySnapshot', label: t('admin.socialAccountWorkbench.columns.defaultProxySnapshot'), sortable: true, class: 'min-w-[220px] max-w-[240px]' },
  { key: 'accountStatus', label: t('admin.socialAccountWorkbench.columns.accountStatus'), sortable: true, class: 'min-w-[128px]' },
  { key: 'assignedUser', label: t('admin.socialAccountWorkbench.columns.assignedUser'), sortable: true, class: 'min-w-[190px] max-w-[230px]' },
  { key: 'actions', label: t('admin.socialAccountWorkbench.columns.actions'), class: 'w-[128px] min-w-[128px]' },
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
const importPlatformOptions = computed(() => [
  { value: 'x_twitter', label: platformLabel('x_twitter') },
])

const filteredTargetUsers = computed(() => {
  const keyword = targetUserSearch.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter(user => [user.email, user.username, user.role].some(value => String(value ?? '').toLowerCase().includes(keyword)))
})

const filteredAccounts = computed(() => accounts.value.filter(accountMatchesCurrentFilters))

function accountMatchesCurrentFilters(account: AccountRow) {
  const keyword = searchQuery.value.trim().toLowerCase()
  const values = [
    String(account.id),
    `#${account.id}`,
    account.account,
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
    account.defaultProxySnapshot,
    account.assignedUser ?? '',
    account.taskMessage,
    account.remark,
  ]
  const matchesKeyword = !keyword || values.some(value => value.toLowerCase().includes(keyword))
  const matchesStatus = accountStatusFilter.value === 'all' || account.accountStatus === accountStatusFilter.value
  const matchesAssignment = assignmentFilter.value === 'all' || (assignmentFilter.value === 'assigned' ? !!account.assignedUserId : !account.assignedUserId)
  return matchesKeyword && matchesStatus && matchesAssignment
}

const accountFilterParams = computed<TotalAccountExportParams>(() => {
  const params: TotalAccountExportParams = {}
  const search = searchQuery.value.trim()
  if (search) params.search = search
  if (accountStatusFilter.value !== 'all') params.account_status = accountStatusFilter.value
  if (assignmentFilter.value === 'assigned') params.assigned = true
  if (assignmentFilter.value === 'unassigned') params.unassigned = true
  return params
})
const exportAccountParams = computed<TotalAccountExportParams>(() => ({
  ...accountFilterParams.value,
  ...(selectedIds.value.length > 0 ? { account_ids: [...selectedIds.value] } : {}),
}))

const accountListParams = computed(() => ({
  page: 1,
  page_size: 200,
  ...accountFilterParams.value,
}))

const hasActiveAccountFilters = computed(() => Object.keys(accountFilterParams.value).length > 0)
const isTotalAccountPoolEmpty = computed(() => accounts.value.length === 0 && !hasActiveAccountFilters.value)

const stats = computed(() => [
  { key: 'total', label: t('admin.socialAccountWorkbench.stats.total'), value: filteredAccounts.value.length },
  { key: 'available', label: t('admin.socialAccountWorkbench.stats.available'), value: filteredAccounts.value.filter(account => account.accountStatus === 'available').length },
  { key: 'assigned', label: t('admin.socialAccountWorkbench.stats.assigned'), value: filteredAccounts.value.filter(account => account.assignedUserId).length },
  { key: 'unassigned', label: t('admin.socialAccountWorkbench.stats.unassigned'), value: filteredAccounts.value.filter(account => !account.assignedUserId).length },
])

const selectedAccounts = computed(() => selectedRowsById(accounts.value, selectedIds.value))
const hasSelection = computed(() => selectedIds.value.length > 0)
const selectedAssignedCount = computed(() => selectedAccounts.value.filter(account => account.assignedUserId).length)
const selectedUnassignedCount = computed(() => selectedAccounts.value.length - selectedAssignedCount.value)
const canAssignSelected = computed(() => hasSelection.value && selectedAssignedCount.value === 0)
const canReclaimSelected = computed(() => selectedAssignedCount.value > 0)
const refreshAccountsButtonTitle = computed(() => buildRefreshAccountsButtonTitle(t, { loading: loading.value }))
const importAccountsButtonTitle = computed(() => buildImportAccountsButtonTitle(t, {
  loading: loading.value,
  importing: importingAccounts.value,
}))
const exportAccountsButtonTitle = computed(() => buildExportAccountsButtonTitle(t, {
  loading: loading.value,
  exporting: exportingAccounts.value,
  selectedCount: selectedIds.value.length,
}))
const rowDetailButtonTitle = computed(() => buildRowDetailButtonTitle(t, { loading: loading.value }))
const rowEditButtonTitle = computed(() => buildRowEditButtonTitle(t, { loading: loading.value }))
const clearSelectionButtonTitle = computed(() => buildClearSelectionButtonTitle(t, {
  loading: loading.value,
  hasSelection: hasSelection.value,
}))
const assignSelectedButtonTitle = computed(() => buildAssignSelectedButtonTitle(t, {
  loading: loading.value,
  assigning: assigning.value,
  hasSelection: hasSelection.value,
  canAssign: canAssignSelected.value,
  selectedAssignedCount: selectedAssignedCount.value,
}))
const reclaimSelectedButtonTitle = computed(() => buildReclaimSelectedButtonTitle(t, {
  loading: loading.value,
  reclaiming: reclaiming.value,
  hasSelection: hasSelection.value,
  canReclaim: canReclaimSelected.value,
}))
const deleteSelectedButtonTitle = computed(() => buildDeleteSelectedButtonTitle(t, {
  loading: loading.value,
  deleting: deleting.value,
  hasSelection: hasSelection.value,
}))
const selectedAccountPreviewState = computed(() => createListPreview(selectedAccounts.value, 6))
const selectedAccountPreview = computed(() => selectedAccountPreviewState.value.items)
const remainingSelectedAccountCount = computed(() => selectedAccountPreviewState.value.remainingCount)
const selectedTargetUser = computed(() => users.value.find(user => String(user.id) === targetUser.value) ?? null)
const assignDialogHint = computed(() => t('admin.socialAccountWorkbench.assignDialog.hint', { count: selectedAccounts.value.length }))
const assignConfirmHint = computed(() => t('admin.socialAccountWorkbench.assignDialog.confirmHint', {
  count: selectedIds.value.length,
  user: selectedTargetUser.value ? selectedTargetUser.value.email : '-',
}))
const reclaimDialogHint = computed(() => t('admin.socialAccountWorkbench.reclaimDialog.hint', { count: selectedIds.value.length }))
const deleteDialogHint = computed(() => t('admin.socialAccountWorkbench.deleteDialog.hint', { count: selectedIds.value.length }))
const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const currentVisibleSelectionState = computed(() => visibleSelectionState(selectedIds.value, visibleIds.value))
const allVisibleSelected = computed(() => currentVisibleSelectionState.value.allSelected)
const someVisibleSelected = computed(() => currentVisibleSelectionState.value.someSelected)
const canSubmitAccount = computed(() => {
  return selectedAccount.value !== null && !totalAccountFormMatchesAccount(selectedAccount.value)
})
const accountEditSubmitDisabledReason = computed(() => {
  if (selectedAccount.value && totalAccountFormMatchesAccount(selectedAccount.value)) return t('admin.socialAccountWorkbench.noChanges')
  return ''
})
const accountEditCancelButtonTitle = computed(() => buildDialogCancelButtonTitle(t, { processing: savingAccount.value }))
const accountEditSubmitButtonTitle = computed(() => buildAccountEditSubmitButtonTitle(t, {
  saving: savingAccount.value,
  disabledReason: accountEditSubmitDisabledReason.value,
}))
const accountEditSubmitButtonLabel = computed(() => buildSubmitButtonLabel(t, { processing: savingAccount.value }))
const assignDialogCancelButtonTitle = computed(() => buildDialogCancelButtonTitle(t, { processing: assigning.value }))
const assignBackButtonTitle = computed(() => buildAssignBackButtonTitle(t, { assigning: assigning.value }))
const assignConfirmButtonTitle = computed(() => buildAssignConfirmButtonTitle(t, { assigning: assigning.value }))
const assignReviewButtonTitle = computed(() => buildAssignReviewButtonTitle(t, {
  loading: loading.value,
  assigning: assigning.value,
  hasSelection: hasSelection.value,
  hasTargetUser: Boolean(targetUser.value),
  canAssign: canAssignSelected.value,
  selectedAssignedCount: selectedAssignedCount.value,
}))
const reclaimDialogCancelButtonTitle = computed(() => buildDialogCancelButtonTitle(t, { processing: reclaiming.value }))
const reclaimConfirmButtonTitle = computed(() => buildReclaimConfirmButtonTitle(t, { reclaiming: reclaiming.value }))
const deleteDialogCancelButtonTitle = computed(() => buildDialogCancelButtonTitle(t, { processing: deleting.value }))
const deleteConfirmButtonTitle = computed(() => buildDeleteConfirmButtonTitle(t, { deleting: deleting.value }))
const batchOperationResultPreview = useTotalAccountOperationResultPreview<BatchOperationResultState, SocialAccountBatchItemResult>({
  result: batchOperationResult,
  items: state => state.result.items,
})
const batchOperationResultItems = batchOperationResultPreview.items
const remainingBatchOperationResultItemCount = batchOperationResultPreview.remainingCount
const importOperationResultPreview = useTotalAccountOperationResultPreview<TotalAccountImportResult, SocialAccountBatchItemResult>({
  result: importOperationResult,
  items: result => result.items,
})
const importOperationResultItems = importOperationResultPreview.items
const remainingImportOperationResultItemCount = importOperationResultPreview.remainingCount
const detailSections = computed<DetailSection[]>(() => {
  if (!selectedAccount.value) return []
  return [
    {
      title: t('accountWorkbench.detailSections.identity'),
      items: [
        { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
        { key: 'account', label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
        { key: 'platform', label: t('admin.socialAccountWorkbench.columns.platform'), value: platformLabel(selectedAccount.value.platform) },
        { key: 'username', label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
        { key: 'platformUserId', label: t('admin.socialAccountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
        { key: 'registrationIp', label: t('admin.socialAccountWorkbench.columns.registrationIp'), value: selectedAccount.value.registrationIp },
      ],
    },
    {
      title: t('accountWorkbench.detailSections.credentials'),
      items: buildDeliveryCredentialItems(selectedAccount.value, {
        emailTokenTestId: 'total-account-email-token-preview',
        emailTokenCopyTestId: 'total-account-email-token-copy',
      }),
    },
    {
      title: t('accountWorkbench.detailSections.operations'),
      items: [
        { key: 'defaultProxySnapshot', label: t('admin.socialAccountWorkbench.columns.defaultProxySnapshot'), value: selectedAccount.value.defaultProxySnapshot },
        { key: 'remark', label: t('admin.socialAccountWorkbench.form.remark'), value: selectedAccount.value.remark },
        { key: 'assignedUser', label: t('admin.socialAccountWorkbench.columns.assignedUser'), value: selectedAccount.value.assignedUser ?? t('admin.socialAccountWorkbench.assignment.unassigned') },
        { key: 'createdAt', label: t('admin.socialAccountWorkbench.columns.createdAt'), value: selectedAccount.value.createdAt },
      ],
    },
  ]
})

const selectedCredentialPreview = computed<SocialAccountCredentialPreview[] | null>(() => {
  if (!selectedAccount.value) return null
  return buildCredentialPreviews(selectedAccount.value)
})

const editIdentityItems = computed(() => {
  if (!selectedAccount.value) return []
  return [
    { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
    { label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
    { label: t('admin.socialAccountWorkbench.columns.platform'), value: platformLabel(selectedAccount.value.platform) },
    { label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
    { label: t('admin.socialAccountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
    { label: t('admin.socialAccountWorkbench.columns.registrationIp'), value: selectedAccount.value.registrationIp },
  ]
})

async function loadUsers() {
  targetUserLoadError.value = ''
  try {
    const result = await adminAPI.users.list(1, 200, { status: 'active' })
    users.value = result.items ?? []
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.load_users', error)
    users.value = []
    targetUserLoadError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.assignDialog.userLoadFailed'), totalAccountErrorMessages.value)
  }
}

async function loadAccounts() {
  const requestID = ++latestLoadRequestID
  loading.value = true
  loadError.value = ''
  try {
    const result = await adminAPI.totalAccounts.list(accountListParams.value)
    if (requestID !== latestLoadRequestID) return
    accounts.value = (result.items ?? []).map(mapApiAccount)
    selectedIds.value = retainExistingSelectedIds(selectedIds.value, accounts.value)
    syncSelectedAccountFromList()
    syncBulkDialogsFromSelection()
  } catch (error) {
    if (requestID !== latestLoadRequestID) return
    recordClientDiagnostic('admin.total_accounts.load_accounts', error)
    loadError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.failedToLoad'), totalAccountErrorMessages.value)
    appStore.showError(loadError.value)
  } finally {
    if (requestID === latestLoadRequestID) {
      loading.value = false
    }
  }
}

function mapApiAccount(account: SocialAccount): AccountRow {
  const taskMessage = String(account.task_message ?? '').trim()
  return {
    id: account.id,
    account: String(account.name ?? '').trim(),
    platform: account.platform,
    username: String(account.username ?? '').trim() || normalizeUsername(account.name),
    platformUserId: String(account.platform_user_id ?? '').trim(),
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
    defaultProxySnapshot: account.default_proxy_snapshot ?? '',
    accountStatus: normalizeKnownWorkbenchAccountStatus(account.account_status),
    taskStatus: socialAccountTaskStatusFromAccountSnapshot(normalizeWorkbenchTaskStatus(account.task_status), taskMessage),
    taskMessage,
    assignedUserId: account.assigned_user_id ?? null,
    assignedUser: ownerLabel(account.assigned_user_id, account.assigned_user_email),
    remark: account.remark ?? '',
    createdAt: formatAccountWorkbenchDate(account.created_at),
  }
}

function updateAccountFromResponse(account: SocialAccount) {
  const updated = mapApiAccount(account)
  syncAccountRowWithCurrentFilters(updated)
  if (selectedAccount.value?.id === updated.id || selectedAccountId.value === updated.id) {
    selectedAccount.value = updated
  }
}

function syncAccountRowWithCurrentFilters(updated: AccountRow) {
  const index = accounts.value.findIndex(item => item.id === updated.id)
  const matchesCurrentFilters = accountMatchesCurrentFilters(updated)
  if (matchesCurrentFilters && index >= 0) {
    accounts.value.splice(index, 1, updated)
  } else if (matchesCurrentFilters) {
    accounts.value = [updated, ...accounts.value]
  } else {
    accounts.value = accounts.value.filter(item => item.id !== updated.id)
  }
  selectedIds.value = retainExistingSelectedIds(selectedIds.value, accounts.value)
  syncSelectedAccountFromList()
  syncBulkDialogsFromSelection()
}

function ownerLabel(userID?: number | null, email?: string | null): string | null {
  if (!userID) return null
  const apiEmail = email?.trim()
  if (apiEmail) return apiEmail
  const user = users.value.find(item => item.id === userID)
  return user?.email ?? `#${userID}`
}

function isSelected(id: number): boolean {
  return selectedIds.value.includes(id)
}

function toggleSelection(id: number) {
  if (loading.value) return
  selectedIds.value = toggleSelectedId(selectedIds.value, id)
}

function toggleAllVisible() {
  if (loading.value) return
  selectedIds.value = toggleVisibleSelectedIds(selectedIds.value, visibleIds.value, allVisibleSelected.value)
}

function clearSelection() {
  if (loading.value) return
  selectedIds.value = []
  syncBulkDialogsFromSelection()
}

function syncBulkDialogsFromSelection() {
  if (assigning.value || reclaiming.value || deleting.value) return

  if (!canAssignSelected.value) {
    closeAssignDialog()
  }

  if (!canReclaimSelected.value) {
    closeReclaimDialog(true)
  }

  if (!hasSelection.value) {
    closeDeleteDialog(true)
  }
}

function clearAccountFilters() {
  searchQuery.value = ''
  accountStatusFilter.value = 'all'
  assignmentFilter.value = 'all'
}

function visibleAssignedCountForUser(userID: number): number {
  return accounts.value.filter(account => account.assignedUserId === userID).length
}

function defaultProxySnapshotListLabel(snapshot: string): string {
  return snapshot.trim()
    ? t('accountWorkbench.proxy.configured')
    : t('accountWorkbench.proxy.notConfigured')
}

function openDetailDialog(row: AccountRow) {
  if (loading.value) return
  selectedAccount.value = row
  detailDialogOpen.value = true
}

function closeDetailDialog() {
  detailDialogOpen.value = false
  if (!editDialogOpen.value) {
    selectedAccount.value = null
  }
}

async function copySelectedCredential(key: SocialAccountCredentialPreviewKey) {
  await copySelectedAccountCredential(key)
}

async function copySelectedEmailToken() {
  await copySelectedAccountEmailToken()
}

function openEditDialog(row: AccountRow) {
  if (loading.value || savingAccount.value) return
  selectedAccountId.value = row.id
  selectedAccount.value = row
  accountFormError.value = ''
  accountForm.password = row.password
  accountForm.phone = row.phone
  accountForm.email = row.email
  accountForm.emailPassword = row.emailPassword
  accountForm.twoFactor = row.twoFactor
  accountForm.backupCode = row.backupCode
  accountForm.emailClientId = row.emailClientId
  accountForm.emailToken = row.emailToken
  accountForm.registrationIp = row.registrationIp
  accountForm.authCookie = row.authCookie
  accountForm.executionAuth = row.executionAuth
  accountForm.accountStatus = row.accountStatus
  accountForm.remark = row.remark
  editDialogOpen.value = true
}

function totalAccountFormMatchesAccount(account: AccountRow) {
  return editableAccountFormString(accountForm.password) === editableAccountFormString(account.password) &&
    trimEditableField(accountForm.phone) === trimEditableField(account.phone) &&
    trimEditableField(accountForm.email) === trimEditableField(account.email) &&
    editableAccountFormString(accountForm.emailPassword) === editableAccountFormString(account.emailPassword) &&
    editableAccountFormString(accountForm.twoFactor) === editableAccountFormString(account.twoFactor) &&
    editableAccountFormString(accountForm.backupCode) === editableAccountFormString(account.backupCode) &&
    editableAccountFormString(accountForm.emailClientId) === editableAccountFormString(account.emailClientId) &&
    editableAccountFormString(accountForm.emailToken) === editableAccountFormString(account.emailToken) &&
    trimEditableField(accountForm.registrationIp) === trimEditableField(account.registrationIp) &&
    editableAccountFormString(accountForm.authCookie) === editableAccountFormString(account.authCookie) &&
    editableAccountFormString(accountForm.executionAuth) === editableAccountFormString(account.executionAuth) &&
    trimEditableField(accountForm.accountStatus) === trimEditableField(account.accountStatus) &&
    editableAccountFormString(accountForm.remark) === editableAccountFormString(account.remark)
}

function editableAccountFormString(value?: string | number | null) {
  return String(value ?? '')
}

function closeEditDialog() {
  if (savingAccount.value) return
  editDialogOpen.value = false
  selectedAccount.value = null
  selectedAccountId.value = null
  accountFormError.value = ''
}

async function submitEditDialog() {
  if (savingAccount.value || !selectedAccountId.value || !canSubmitAccount.value) return
  savingAccount.value = true
  accountFormError.value = ''
  clearTotalAccountOperationResults()
  try {
    const updated = await adminAPI.totalAccounts.update(selectedAccountId.value, buildTotalAccountEditPayload(accountForm))
    updateAccountFromResponse(updated)
    appStore.showSuccess(t('admin.socialAccountWorkbench.saved'))
    editDialogOpen.value = false
    selectedAccount.value = null
    selectedAccountId.value = null
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.edit', error)
    accountFormError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.saveFailed'), totalAccountErrorMessages.value)
    appStore.showError(accountFormError.value)
  } finally {
    savingAccount.value = false
  }
}

function triggerImport() {
  if (loading.value || importingAccounts.value) return
  importFileInput.value?.click()
}

async function handleImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (loading.value || importingAccounts.value) {
    input.value = ''
    return
  }
  if (file.size === 0) {
    clearTotalAccountOperationResults()
    appStore.showError(totalAccountErrorMessages.value.SOCIAL_ACCOUNT_IMPORT_REQUIRED)
    input.value = ''
    return
  }
  importingAccounts.value = true
  clearTotalAccountOperationResults()
  try {
    const result = await adminAPI.totalAccounts.importAccounts(file, importPlatform.value)
    importOperationResult.value = result
    showImportOperationToast(result)
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.import', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.importFailed'), totalAccountErrorMessages.value))
  } finally {
    importingAccounts.value = false
    input.value = ''
  }
}

function showImportOperationToast(result: TotalAccountImportResult) {
  showSocialAccountBatchResultToast({
    succeeded: result.succeeded,
    failed: result.failed,
    skipped: result.skipped,
    summary: importOperationSummary(result),
    successMessage: t('admin.socialAccountWorkbench.imported', { count: result.created }),
    preferWarning: result.duplicates > 0,
  }, {
    showError: message => appStore.showError(message),
    showSuccess: message => appStore.showSuccess(message),
    showWarning: message => appStore.showWarning(message),
  })
}

function clearTotalAccountOperationResults() {
  importOperationResult.value = null
  batchOperationResult.value = null
}

async function exportAccounts() {
  if (loading.value || exportingAccounts.value) return
  exportingAccounts.value = true
  try {
    const blob = await adminAPI.totalAccounts.exportAccounts(exportAccountParams.value)
    const scope = selectedIds.value.length > 0 ? 'selected' : 'pool'
    downloadBlob(blob, `social_account_${scope}.csv`)
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.export', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.exportFailed'), totalAccountErrorMessages.value))
  } finally {
    exportingAccounts.value = false
  }
}

function openAssignDialog() {
  if (loading.value || assigning.value || !hasSelection.value) return
  const assignedCount = selectedAccounts.value.filter(account => account.assignedUserId).length
  if (assignedCount > 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.assignRequiresUnassigned', { count: assignedCount }))
    return
  }
  targetUser.value = ''
  targetUserSearch.value = ''
  assignDialogError.value = ''
  assignConfirmDialogOpen.value = false
  assignDialogOpen.value = true
}

function closeAssignDialog(force = false) {
  if (!force && assigning.value) return
  assignDialogOpen.value = false
  assignConfirmDialogOpen.value = false
  assignDialogError.value = ''
}

function closeAssignConfirmDialog(force = false) {
  if (!force && assigning.value) return
  assignConfirmDialogOpen.value = false
  assignDialogError.value = ''
  if (!force && canAssignSelected.value) {
    assignDialogOpen.value = true
  }
}

function openAssignConfirmDialog() {
  if (loading.value || assigning.value || !canAssignSelected.value) return
  const userIdNum = Number(targetUser.value)
  if (!Number.isFinite(userIdNum) || userIdNum <= 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.selectTargetUser'))
    return
  }
  assignDialogError.value = ''
  assignDialogOpen.value = false
  assignConfirmDialogOpen.value = true
}

async function confirmAssignDialog() {
  if (loading.value || assigning.value || !canAssignSelected.value) return
  const accountIds = [...selectedIds.value]
  const userIdNum = Number(targetUser.value)
  if (!accountIds.length || !Number.isFinite(userIdNum) || userIdNum <= 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.selectTargetUser'))
    return
  }
  assigning.value = true
  assignDialogError.value = ''
  clearTotalAccountOperationResults()
  try {
    const result = await adminAPI.totalAccounts.batchAssign(accountIds, userIdNum)
    const targetUserLabel = selectedTargetUser.value?.email ?? `#${userIdNum}`
    batchOperationResult.value = { action: 'assigned', result, targetUser: targetUserLabel }
    showBatchOperationToast('assigned', result, { user: targetUserLabel })
    applyBatchAssignResult(result, accountIds, userIdNum)
    removeSucceededBatchItemsFromSelection(result, accountIds)
    closeAssignDialog(true)
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.assign', error)
    assignDialogError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.assignFailed'), totalAccountErrorMessages.value)
    appStore.showError(assignDialogError.value)
  } finally {
    assigning.value = false
  }
}

function openReclaimDialog() {
  if (loading.value || reclaiming.value) return
  if (!canReclaimSelected.value) return
  reclaimDialogError.value = ''
  reclaimDialogOpen.value = true
}

function closeReclaimDialog(force = false) {
  if (!force && reclaiming.value) return
  reclaimDialogOpen.value = false
  reclaimDialogError.value = ''
}

async function reclaimSelectedAccounts() {
  if (loading.value || reclaiming.value || !canReclaimSelected.value) return
  const accountIds = [...selectedIds.value]
  reclaiming.value = true
  reclaimDialogError.value = ''
  clearTotalAccountOperationResults()
  try {
    const result = await adminAPI.totalAccounts.batchReclaim(accountIds)
    batchOperationResult.value = { action: 'reclaimed', result }
    showBatchOperationToast('reclaimed', result)
    applyBatchReclaimResult(result, accountIds)
    removeSucceededBatchItemsFromSelection(result, accountIds)
    closeReclaimDialog(true)
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.reclaim', error)
    reclaimDialogError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.reclaimFailed'), totalAccountErrorMessages.value)
    appStore.showError(reclaimDialogError.value)
  } finally {
    reclaiming.value = false
  }
}

function openDeleteDialog() {
  if (loading.value || deleting.value) return
  if (!hasSelection.value) return
  deleteDialogError.value = ''
  deleteDialogOpen.value = true
}

function closeDeleteDialog(force = false) {
  if (!force && deleting.value) return
  deleteDialogOpen.value = false
  deleteDialogError.value = ''
}

async function confirmDeleteDialog() {
  if (loading.value || deleting.value || !hasSelection.value) return
  const accountIds = [...selectedIds.value]
  deleting.value = true
  deleteDialogError.value = ''
  clearTotalAccountOperationResults()
  try {
    const result = await adminAPI.totalAccounts.batchDelete(accountIds)
    batchOperationResult.value = { action: 'deleted', result }
    showBatchOperationToast('deleted', result)
    removeSucceededBatchItemsFromLocalState(result, accountIds)
    closeDeleteDialog(true)
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.delete', error)
    deleteDialogError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.deleteFailed'), totalAccountErrorMessages.value)
    appStore.showError(deleteDialogError.value)
  } finally {
    deleting.value = false
  }
}

function showBatchOperationToast(action: 'assigned' | 'reclaimed' | 'deleted', result: SocialAccountBatchResult, extraParams: Record<string, string | number> = {}) {
  const params = socialAccountBatchResultToastParams(result, extraParams)
  showSocialAccountBatchResultToast({
    succeeded: result.succeeded,
    failed: result.failed,
    skipped: result.skipped,
    summary: t(`admin.socialAccountWorkbench.toasts.${action}Result`, params),
    successMessage: t(`admin.socialAccountWorkbench.toasts.${action}`, params),
  }, {
    showError: message => appStore.showError(message),
    showSuccess: message => appStore.showSuccess(message),
    showWarning: message => appStore.showWarning(message),
  })
}

function batchOperationSummary(state: BatchOperationResultState) {
  return t(`admin.socialAccountWorkbench.toasts.${state.action}Result`, totalAccountBatchOperationSummaryParams(state))
}

function importOperationSummary(result: TotalAccountImportResult) {
  return t('admin.socialAccountWorkbench.importResult.summary', accountWorkbenchImportResultSummaryParams(result))
}

function importOperationStatusLabel(value?: string | null) {
  return accountWorkbenchImportResultStatusLabel(value, t)
}

function batchOperationStatusLabel(value?: string | null) {
  return socialAccountBatchResultStatusLabel(value, 'admin.socialAccountWorkbench.batchResult.statuses', t)
}

function batchOperationResultRowToneClass(value?: string | null) {
  return socialAccountBatchResultRowToneClass(value)
}

function batchOperationItemLabel(item: SocialAccountBatchItemResult, index: number) {
  const fallback = `#${index + 1}`
  if (typeof item.id === 'number' && item.id > 0) {
    return formatSocialAccountBatchResultName(item.name, accounts.value.find(account => account.id === item.id)?.account || `#${item.id}`)
  }
  return formatSocialAccountBatchResultName(item.name, fallback)
}

function batchOperationItemMessage(item: SocialAccountBatchItemResult) {
  return socialAccountBatchResultItemMessage(item, t)
}

function importOperationItemMessage(item: SocialAccountBatchItemResult) {
  return accountWorkbenchImportResultItemMessage(item, t)
}

function applyBatchAssignResult(result: SocialAccountBatchResult, requestedIds: number[], userID: number) {
  const assignedIds = collectSucceededBatchItemIds(result, requestedIds)
  if (assignedIds.size === 0) return
  const user = users.value.find(item => item.id === userID)
  const assignedUser = user?.email ?? `#${userID}`
  updateAccountsById(assignedIds, account => ({
    ...account,
    assignedUserId: userID,
    assignedUser,
    defaultProxySnapshot: '',
  }))
}

function applyBatchReclaimResult(result: SocialAccountBatchResult, requestedIds: number[]) {
  const reclaimedIds = collectSucceededBatchItemIds(result, requestedIds)
  if (reclaimedIds.size === 0) return
  updateAccountsById(reclaimedIds, account => ({
    ...account,
    assignedUserId: null,
    assignedUser: null,
    defaultProxySnapshot: '',
  }))
}

function removeSucceededBatchItemsFromLocalState(result: SocialAccountBatchResult, requestedIds: number[]) {
  const removedIds = collectSucceededBatchItemIds(result, requestedIds)
  if (removedIds.size === 0) return
  accounts.value = accounts.value.filter(account => !removedIds.has(account.id))
  selectedIds.value = removeSelectedIds(selectedIds.value, removedIds)
  syncSelectedAccountFromList()
  syncBulkDialogsFromSelection()
}

function removeSucceededBatchItemsFromSelection(result: SocialAccountBatchResult, requestedIds: number[]) {
  const succeededIds = collectSucceededBatchItemIds(result, requestedIds)
  if (succeededIds.size === 0) return
  selectedIds.value = removeSelectedIds(selectedIds.value, succeededIds)
  syncBulkDialogsFromSelection()
}

function updateAccountsById(accountIds: Set<number>, update: (account: AccountRow) => AccountRow) {
  accounts.value = accounts.value.flatMap((account) => {
    if (!accountIds.has(account.id)) return [account]
    const updated = update(account)
    return accountMatchesCurrentFilters(updated) ? [updated] : []
  })
  selectedIds.value = retainExistingSelectedIds(selectedIds.value, accounts.value)
  syncSelectedAccountFromList()
  syncBulkDialogsFromSelection()
}

function syncSelectedAccountFromList() {
  if (!selectedAccount.value) return
  const updated = accounts.value.find(account => account.id === selectedAccount.value?.id)
  if (updated) {
    selectedAccount.value = updated
    return
  }
  detailDialogOpen.value = false
  editDialogOpen.value = false
  selectedAccount.value = null
  selectedAccountId.value = null
}

function normalizeUsername(value?: string | null): string {
  return String(value || '').trim().toLowerCase().replace(/^@+/, '').trim()
}

</script>

<style scoped>
.total-accounts-table :deep(.sticky-col-right::before) {
  content: none !important;
  width: 0 !important;
  background: none !important;
  transform: none !important;
}

.total-accounts-table :deep(.sticky-col-right) {
  box-shadow: -1px 0 0 rgb(229 231 235);
}

.dark .total-accounts-table :deep(.sticky-col-right) {
  box-shadow: -1px 0 0 rgb(55 65 81);
}
</style>
