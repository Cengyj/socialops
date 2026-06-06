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
                  <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('admin.socialAccountWorkbench.failedToLoad') }}</p>
                  <p class="mt-1 text-sm text-red-600 dark:text-red-300/80">{{ loadError }}</p>
                </div>
              </div>
              <button type="button" class="btn btn-secondary shrink-0" @click="loadAccounts">{{ t('common.retry') }}</button>
            </div>
          </div>

          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <div v-for="stat in stats" :key="stat.label" class="rounded-lg border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="flex items-center justify-between gap-3">
                <div class="truncate text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
                <div class="shrink-0 text-lg font-semibold leading-6 text-gray-900 dark:text-white">{{ stat.value }}</div>
              </div>
            </div>
          </div>

          <div data-testid="total-accounts-toolbar" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
            <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-2 xl:flex-row xl:items-center">
                <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2 xl:flex-nowrap">
                  <SearchInput v-model="searchQuery" :placeholder="t('admin.socialAccountWorkbench.searchPlaceholder')" class="w-full shrink-0 sm:w-[220px] xl:w-[200px] 2xl:w-[300px]" />
                  <Select v-model="accountStatusFilter" :options="accountStatusOptions" class="w-full shrink-0 sm:w-[168px] xl:w-[148px] 2xl:w-[184px]" />
                  <Select v-model="assignmentFilter" :options="assignmentOptions" class="w-full shrink-0 sm:w-[152px] xl:w-[132px] 2xl:w-[168px]" />
                  <Select v-model="importPlatform" :options="importPlatformOptions" class="w-full shrink-0 sm:w-[152px] xl:w-[132px] 2xl:w-[168px]" />
                  <div class="hidden h-6 w-px shrink-0 bg-gray-200 dark:bg-dark-700 xl:block"></div>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('common.refresh')"
                    :title="t('common.refresh')"
                    :disabled="loading"
                    @click="loadAccounts"
                  >
                    <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                    <span class="xl:hidden 2xl:inline">{{ t('common.refresh') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('admin.socialAccountWorkbench.toolbar.importAccounts')"
                    :title="t('admin.socialAccountWorkbench.toolbar.importAccounts')"
                    @click="triggerImport"
                  >
                    <Icon name="upload" size="sm" />
                    <span class="xl:hidden 2xl:inline">{{ t('admin.socialAccountWorkbench.toolbar.importAccounts') }}</span>
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto xl:w-10 xl:min-w-[40px] xl:max-w-[40px] xl:px-0 2xl:w-auto 2xl:min-w-0 2xl:max-w-none 2xl:px-3"
                    :aria-label="t('admin.socialAccountWorkbench.toolbar.exportRecords')"
                    :title="t('admin.socialAccountWorkbench.toolbar.exportRecords')"
                    @click="exportAccounts"
                  >
                    <Icon name="download" size="sm" />
                    <span class="xl:hidden 2xl:inline">{{ t('admin.socialAccountWorkbench.toolbar.exportRecords') }}</span>
                  </button>
                  <input ref="importFileInput" type="file" accept=".csv,.json,.xlsx" class="hidden" @change="handleImportFile" />
                </div>

                <div class="flex w-full shrink-0 flex-col gap-2 sm:flex-row xl:ml-auto xl:w-auto xl:items-center">
                  <div class="flex h-10 w-full shrink-0 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800 sm:w-auto">
                    <div class="flex min-w-[132px] flex-1 items-center justify-center whitespace-nowrap bg-primary-50 px-3 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 sm:flex-none xl:min-w-[112px] xl:px-2 2xl:min-w-[132px] 2xl:px-3">
                      {{ t('admin.socialAccountWorkbench.executionBar.selectedCount', { count: selectedIds.length }) }}
                    </div>
                    <button
                      type="button"
                      class="flex h-full w-10 shrink-0 items-center justify-center border-l border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-gray-100"
                      :aria-label="t('admin.socialAccountWorkbench.executionBar.clear')"
                      :title="t('admin.socialAccountWorkbench.executionBar.clear')"
                      :disabled="!hasSelection"
                      @click="clearSelection"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>
                  <button type="button" class="btn btn-primary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4" :disabled="!canAssignSelected || assigning" @click="openAssignDialog">
                    <Icon name="userPlus" size="sm" />
                    <span>{{ t('admin.socialAccountWorkbench.actions.assign') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4" :disabled="!hasSelection || reclaiming" @click="openReclaimDialog">
                    <Icon name="swap" size="sm" />
                    <span>{{ t('admin.socialAccountWorkbench.actions.reclaim') }}</span>
                  </button>
                  <button type="button" class="btn btn-danger btn-sm h-10 w-full shrink-0 justify-center whitespace-nowrap sm:w-auto sm:px-4 xl:px-3 2xl:px-4" :disabled="!hasSelection || deleting" @click="openDeleteDialog">
                    <Icon name="trash" size="sm" />
                    <span>{{ t('admin.socialAccountWorkbench.actions.delete') }}</span>
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
          default-sort-order="desc"
          :estimate-row-height="72"
          :sticky-first-column="false"
          :sticky-actions-column="true"
        >
          <template #header-select>
            <div class="flex justify-center">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="allVisibleSelected"
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
                class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="isSelected(row.id)"
                @click.stop
                @change="toggleSelection(row.id)"
              />
            </div>
          </template>
          <template #cell-account="{ row }">
            <button class="flex min-w-[220px] max-w-[260px] items-center gap-3 text-left" @click="openDetailDialog(row)">
              <span :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border text-xs font-semibold', platformAvatarClass(row.platform)]">
                {{ platformInitial(row.platform) }}
              </span>
              <span class="min-w-0">
                <span class="block truncate font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">{{ row.account }}</span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">#{{ row.id }} · {{ row.username || '-' }}</span>
              </span>
            </button>
          </template>
          <template #cell-platform="{ value }">
            <span class="inline-block max-w-[110px] truncate rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200" :title="String(value || '')">
              {{ value || '-' }}
            </span>
          </template>
          <template #cell-email="{ value }">
            <span class="block w-[200px] truncate text-sm text-gray-700 dark:text-gray-300" :title="String(value || '')">{{ value || '-' }}</span>
          </template>
          <template #cell-credentials="{ row }">
            <div class="grid w-full min-w-0 max-w-full gap-1.5 text-xs text-gray-700 dark:text-gray-300 sm:w-[300px] sm:max-w-[300px]">
              <div class="flex min-w-0 gap-1.5">
                <span class="shrink-0 font-medium text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.password') }}</span>
                <span class="min-w-0 truncate" :title="row.password">{{ row.password || '-' }}</span>
              </div>
              <div class="flex min-w-0 gap-1.5">
                <span class="shrink-0 font-medium text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.emailPassword') }}</span>
                <span class="min-w-0 truncate" :title="row.emailPassword">{{ row.emailPassword || '-' }}</span>
              </div>
              <div class="flex min-w-0 gap-1.5">
                <span class="shrink-0 font-medium text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.authCookie') }}</span>
                <span class="min-w-0 truncate" :title="row.authCookie">{{ row.authCookie || '-' }}</span>
              </div>
              <div class="flex min-w-0 gap-1.5">
                <span class="shrink-0 font-medium text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.executionAuth') }}</span>
                <span class="min-w-0 truncate" :title="row.executionAuth">{{ row.executionAuth || '-' }}</span>
              </div>
            </div>
          </template>
          <template #cell-defaultProxySnapshot="{ row }">
            <span class="block w-[220px] truncate text-sm text-gray-700 dark:text-gray-300" :title="row.defaultProxySnapshot">{{ row.defaultProxySnapshot || '-' }}</span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', accountStatusBadgeClass(String(value))]">{{ t(`admin.socialAccountWorkbench.accountStatus.${value}`) }}</span>
          </template>
          <template #cell-assignedUser="{ row }">
            <div class="max-w-[220px]">
              <span :class="['badge max-w-full truncate', row.assignedUser ? 'badge-primary' : 'badge-warning']">
                {{ row.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}
              </span>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-start gap-2">
              <button class="btn btn-secondary h-9 w-9 px-0" :aria-label="t('admin.socialAccountWorkbench.rowActions.detail')" :title="t('admin.socialAccountWorkbench.rowActions.detail')" @click="openDetailDialog(row)">
                <Icon name="eye" size="sm" />
              </button>
              <button class="btn btn-secondary h-9 w-9 px-0" :aria-label="t('common.edit')" :title="t('common.edit')" @click="openEditDialog(row)">
                <Icon name="edit" size="sm" />
              </button>
            </div>
          </template>
          <template #empty>
            <div class="flex flex-col items-center py-8 text-center">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ accounts.length === 0 ? t('admin.socialAccountWorkbench.empty.title') : t('admin.socialAccountWorkbench.noResults.title') }}
              </p>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
                {{ accounts.length === 0 ? t('admin.socialAccountWorkbench.empty.description') : t('admin.socialAccountWorkbench.noResults.description') }}
              </p>
              <div v-if="accounts.length === 0" class="mt-4 flex flex-wrap justify-center gap-2">
                <button type="button" class="btn btn-primary btn-sm" @click="triggerImport">
                  <Icon name="upload" size="sm" />
                  <span>{{ t('admin.socialAccountWorkbench.toolbar.importAccounts') }}</span>
                </button>
                <button type="button" class="btn btn-secondary btn-sm" @click="loadAccounts">
                  <Icon name="refresh" size="sm" />
                  <span>{{ t('common.refresh') }}</span>
                </button>
              </div>
              <button v-else type="button" class="btn btn-secondary btn-sm mt-4" @click="clearAccountFilters">
                <Icon name="x" size="sm" />
                <span>{{ t('admin.socialAccountWorkbench.filters.clear') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="detailDialogOpen = false">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.tabs.poolDescription') }}
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
        <div v-if="selectedAccount.taskMessage" :class="['rounded-lg border p-3 text-sm', resultMessagePanelClass(selectedAccount.accountStatus)]">
          {{ selectedAccount.taskMessage }}
        </div>
      </div>
      <template #footer>
        <button class="btn btn-primary" @click="detailDialogOpen = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="editDialogOpen" :title="t('admin.socialAccountWorkbench.editTitle')" width="wide" @close="editDialogOpen = false">
      <div v-if="selectedAccount" class="space-y-4">
        <div data-testid="total-account-edit-identity" class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-700/60">
          <div class="mb-2 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.edit.identityTitle') }}</div>
          <div class="grid gap-2 sm:grid-cols-3">
            <div v-for="item in editIdentityItems" :key="item.label">
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</div>
              <div class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.edit.identityHint') }}</p>
        </div>

        <div data-testid="total-account-edit-form" class="space-y-3">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.credentials') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-password">{{ t('admin.socialAccountWorkbench.form.password') }}</label>
              <input id="total-account-edit-password" v-model="accountForm.password" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-phone">{{ t('admin.socialAccountWorkbench.form.phone') }}</label>
              <input id="total-account-edit-phone" v-model="accountForm.phone" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email">{{ t('admin.socialAccountWorkbench.form.email') }}</label>
              <input id="total-account-edit-email" v-model="accountForm.email" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-password">{{ t('admin.socialAccountWorkbench.form.emailPassword') }}</label>
              <input id="total-account-edit-email-password" v-model="accountForm.emailPassword" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-two-factor">{{ t('admin.socialAccountWorkbench.form.twoFactor') }}</label>
              <input id="total-account-edit-two-factor" v-model="accountForm.twoFactor" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-backup-code">{{ t('admin.socialAccountWorkbench.form.backupCode') }}</label>
              <input id="total-account-edit-backup-code" v-model="accountForm.backupCode" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-client-id">{{ t('admin.socialAccountWorkbench.form.emailClientId') }}</label>
              <input id="total-account-edit-email-client-id" v-model="accountForm.emailClientId" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-email-token">{{ t('admin.socialAccountWorkbench.form.emailToken') }}</label>
              <input id="total-account-edit-email-token" v-model="accountForm.emailToken" type="text" class="input mt-2 bg-white dark:bg-dark-800" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-auth-cookie">{{ t('admin.socialAccountWorkbench.form.authCookie') }}</label>
              <textarea id="total-account-edit-auth-cookie" v-model="accountForm.authCookie" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800"></textarea>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-execution-auth">{{ t('admin.socialAccountWorkbench.form.executionAuth') }}</label>
              <textarea id="total-account-edit-execution-auth" v-model="accountForm.executionAuth" class="input mt-2 min-h-[120px] bg-white dark:bg-dark-800"></textarea>
            </div>
          </div>

          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('accountWorkbench.detailSections.operations') }}</div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
              <label class="mb-2 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.columns.accountStatus') }}</label>
              <Select v-model="accountForm.accountStatus" :options="accountStatusOptionsWithoutAll" />
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700 sm:col-span-2">
              <label class="block text-xs text-gray-500 dark:text-gray-400" for="total-account-edit-remark">{{ t('admin.socialAccountWorkbench.form.remark') }}</label>
              <textarea id="total-account-edit-remark" v-model="accountForm.remark" class="input mt-2 min-h-[88px] bg-white dark:bg-dark-800"></textarea>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="editDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!canSubmitAccount" @click="submitEditDialog">{{ t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="assignDialogOpen" :title="t('admin.socialAccountWorkbench.assignDialog.title')" width="wide" @close="closeAssignDialog">
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
        <button class="btn btn-secondary" @click="closeAssignDialog">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!targetUser || assigning" @click="openAssignConfirmDialog">{{ t('admin.socialAccountWorkbench.assignDialog.reviewButton') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="assignConfirmDialogOpen" :title="t('admin.socialAccountWorkbench.assignDialog.confirmTitle')" width="normal" @close="assignConfirmDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.assignDialog.confirmHint', { count: selectedIds.length, user: selectedTargetUser ? selectedTargetUser.email : '-' }) }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.assignDialog.accountSummary') }}</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedIds.length }}</div>
            <div class="mt-3 flex flex-wrap gap-2">
              <span
                v-for="account in selectedAccountPreview"
                :key="account.id"
                class="max-w-full truncate rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200"
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
              <div class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ selectedTargetUser.email }}</div>
              <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">#{{ selectedTargetUser.id }} · {{ selectedTargetUser.role }}</div>
              <div class="mt-3 rounded-lg bg-gray-50 px-3 py-2 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ t('admin.socialAccountWorkbench.assignDialog.assignedCountLabel', { count: assignedCountForUser(selectedTargetUser.id) }) }}
              </div>
            </div>
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300">
          {{ t('admin.socialAccountWorkbench.assignDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" :disabled="assigning" @click="assignConfirmDialogOpen = false">{{ t('admin.socialAccountWorkbench.assignDialog.backToSelect') }}</button>
        <button class="btn btn-primary" :disabled="!targetUser || assigning" @click="confirmAssignDialog">
          <Icon name="refresh" size="sm" :class="assigning ? 'animate-spin' : 'hidden'" />
          <span>{{ t('admin.socialAccountWorkbench.assignDialog.confirm') }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="reclaimDialogOpen" :title="t('admin.socialAccountWorkbench.reclaimDialog.title')" width="normal" @close="reclaimDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('admin.socialAccountWorkbench.reclaimDialog.hint', { count: selectedIds.length }) }}
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
              <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ account.account }}</span>
              <span class="shrink-0 truncate text-xs text-gray-500 dark:text-gray-400">{{ account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}</span>
            </div>
            <div v-if="remainingSelectedAccountCount > 0" class="rounded-lg bg-primary-50 px-3 py-2 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" :disabled="reclaiming" @click="reclaimDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!hasSelection || reclaiming" @click="reclaimSelectedAccounts">
          <Icon name="refresh" size="sm" :class="reclaiming ? 'animate-spin' : 'hidden'" />
          <span>{{ t('admin.socialAccountWorkbench.reclaimDialog.confirm') }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('admin.socialAccountWorkbench.deleteDialog.title')" width="normal" @close="deleteDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300">
          {{ t('admin.socialAccountWorkbench.deleteDialog.hint', { count: selectedIds.length }) }}
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
              <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ account.account }}</span>
              <span class="shrink-0 truncate text-xs text-gray-500 dark:text-gray-400">{{ account.assignedUser || t('admin.socialAccountWorkbench.assignment.unassigned') }}</span>
            </div>
            <div v-if="remainingSelectedAccountCount > 0" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 dark:bg-red-900/20 dark:text-red-300">
              {{ t('admin.socialAccountWorkbench.assignDialog.accountSummaryMore', { count: remainingSelectedAccountCount }) }}
            </div>
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300">
          {{ t('admin.socialAccountWorkbench.deleteDialog.impactHint') }}
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" :disabled="deleting" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-danger" :disabled="!hasSelection || deleting" @click="confirmDeleteDialog">
          <Icon name="refresh" size="sm" :class="deleting ? 'animate-spin' : 'hidden'" />
          <span>{{ t('admin.socialAccountWorkbench.deleteDialog.confirm') }}</span>
        </button>
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
import type { SocialAccount } from '@/api/admin'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

type AccountStatus = 'pending_check' | 'available' | 'limited' | 'invalid' | 'not_stored'

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

const { t } = useI18n()
const appStore = useAppStore()

const searchQuery = ref('')
const accountStatusFilter = ref('all')
const assignmentFilter = ref('all')
const selectedIds = ref<number[]>([])
const accounts = ref<AccountRow[]>([])
const users = ref<AdminUser[]>([])
const loading = ref(false)
const loadError = ref('')
const selectedAccount = ref<AccountRow | null>(null)
const selectedAccountId = ref<number | null>(null)
const detailDialogOpen = ref(false)
const editDialogOpen = ref(false)
const assignDialogOpen = ref(false)
const assignConfirmDialogOpen = ref(false)
const reclaimDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const assigning = ref(false)
const reclaiming = ref(false)
const deleting = ref(false)
const targetUser = ref('')
const targetUserSearch = ref('')
const importPlatform = ref('x_twitter')
const importFileInput = ref<HTMLInputElement | null>(null)

const accountForm = reactive({
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
  accountStatus: 'pending_check',
  remark: '',
})

onMounted(async () => {
  await loadUsers()
  await loadAccounts()
})

watch(assignDialogOpen, (open) => {
  if (open) {
    targetUser.value = ''
    targetUserSearch.value = ''
  }
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px] text-center' },
  { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), sortable: true, class: 'w-[84px] min-w-[84px]' },
  { key: 'account', label: t('admin.socialAccountWorkbench.columns.account'), sortable: true, class: 'min-w-[240px]' },
  { key: 'platform', label: t('admin.socialAccountWorkbench.columns.platform'), sortable: true, class: 'min-w-[118px]' },
  { key: 'email', label: t('admin.socialAccountWorkbench.columns.email'), sortable: true, class: 'min-w-[210px] max-w-[220px]' },
  { key: 'credentials', label: t('admin.socialAccountWorkbench.columns.credentials'), class: 'min-w-[300px] max-w-[320px]' },
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
  { value: 'x_twitter', label: 'X / Twitter' },
])

const filteredTargetUsers = computed(() => {
  const keyword = targetUserSearch.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter(user => [user.email, user.username, user.role].some(value => value.toLowerCase().includes(keyword)))
})

const filteredAccounts = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return accounts.value.filter(account => {
    const values = [
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
      account.executionAuth,
      account.defaultProxySnapshot,
      account.assignedUser ?? '',
      account.taskMessage,
      account.remark,
    ]
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
const hasSelection = computed(() => selectedIds.value.length > 0)
const selectedAssignedCount = computed(() => selectedAccounts.value.filter(account => account.assignedUserId).length)
const selectedUnassignedCount = computed(() => selectedAccounts.value.length - selectedAssignedCount.value)
const canAssignSelected = computed(() => hasSelection.value && selectedAssignedCount.value === 0)
const selectedAccountPreview = computed(() => selectedAccounts.value.slice(0, 6))
const remainingSelectedAccountCount = computed(() => Math.max(0, selectedAccounts.value.length - selectedAccountPreview.value.length))
const selectedTargetUser = computed(() => users.value.find(user => String(user.id) === targetUser.value) ?? null)
const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)
const canSubmitAccount = computed(() => selectedAccountId.value !== null)
const detailSections = computed(() => {
  if (!selectedAccount.value) return []
  return [
    {
      title: t('accountWorkbench.detailSections.identity'),
      items: [
        { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
        { label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
        { label: t('admin.socialAccountWorkbench.columns.platform'), value: selectedAccount.value.platform },
        { label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
        { label: t('admin.socialAccountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
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
        { label: t('admin.socialAccountWorkbench.columns.defaultProxySnapshot'), value: selectedAccount.value.defaultProxySnapshot },
        { label: t('admin.socialAccountWorkbench.form.remark'), value: selectedAccount.value.remark },
        { label: t('admin.socialAccountWorkbench.columns.assignedUser'), value: selectedAccount.value.assignedUser ?? t('admin.socialAccountWorkbench.assignment.unassigned') },
        { label: t('admin.socialAccountWorkbench.columns.createdAt'), value: selectedAccount.value.createdAt },
      ],
    },
  ]
})

const editIdentityItems = computed(() => {
  if (!selectedAccount.value) return []
  return [
    { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
    { label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
    { label: t('admin.socialAccountWorkbench.columns.platform'), value: selectedAccount.value.platform },
    { label: t('accountWorkbench.columns.username'), value: selectedAccount.value.username },
    { label: t('admin.socialAccountWorkbench.columns.platformUserId'), value: selectedAccount.value.platformUserId },
    { label: t('admin.socialAccountWorkbench.columns.registrationIp'), value: selectedAccount.value.registrationIp },
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
  loading.value = true
  loadError.value = ''
  try {
    const result = await adminAPI.totalAccounts.list({ page: 1, page_size: 200 })
    accounts.value = (result.items ?? []).map(mapApiAccount)
    selectedIds.value = selectedIds.value.filter(id => accounts.value.some(account => account.id === id))
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.load_accounts', error)
    loadError.value = extractSafeApiErrorMessage(error, t('admin.socialAccountWorkbench.failedToLoad'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

function mapApiAccount(account: SocialAccount): AccountRow {
  return {
    id: account.id,
    account: account.name,
    platform: account.platform,
    username: account.username ?? normalizeUsername(account.name),
    platformUserId: account.platform_user_id ?? '',
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
    defaultProxySnapshot: account.default_proxy_snapshot ?? '',
    accountStatus: toAccountStatus(account.account_status),
    taskStatus: account.task_status,
    taskMessage: account.task_message ?? '',
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

function clearAccountFilters() {
  searchQuery.value = ''
  accountStatusFilter.value = 'all'
  assignmentFilter.value = 'all'
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
  selectedAccount.value = row
  accountForm.password = row.password
  accountForm.phone = row.phone
  accountForm.email = row.email
  accountForm.emailPassword = row.emailPassword
  accountForm.twoFactor = row.twoFactor
  accountForm.backupCode = row.backupCode
  accountForm.emailClientId = row.emailClientId
  accountForm.emailToken = row.emailToken
  accountForm.authCookie = row.authCookie
  accountForm.executionAuth = row.executionAuth
  accountForm.accountStatus = row.accountStatus
  accountForm.remark = row.remark
  editDialogOpen.value = true
}

async function submitEditDialog() {
  if (!selectedAccountId.value || !canSubmitAccount.value) return
  try {
    await adminAPI.accountWorkbench.update(selectedAccountId.value, {
      password: accountForm.password,
      phone: accountForm.phone,
      email: accountForm.email,
      email_password: accountForm.emailPassword,
      two_factor: accountForm.twoFactor,
      backup_code: accountForm.backupCode,
      email_client_id: accountForm.emailClientId,
      email_token: accountForm.emailToken,
      auth_cookie: accountForm.authCookie,
      execution_auth: accountForm.executionAuth,
      account_status: accountForm.accountStatus,
      remark: accountForm.remark,
    })
    appStore.showSuccess(t('admin.socialAccountWorkbench.saved'))
    editDialogOpen.value = false
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.edit', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
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
    const result = await adminAPI.accountWorkbench.importAccounts(file, importPlatform.value)
    appStore.showSuccess(t('admin.socialAccountWorkbench.imported', { count: result.created }))
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.import', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  } finally {
    input.value = ''
  }
}

async function exportAccounts() {
  try {
    const blob = await adminAPI.accountWorkbench.exportAccounts()
    downloadBlob(blob, 'social_account_pool.csv')
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.export', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
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

function closeAssignDialog() {
  assignDialogOpen.value = false
  assignConfirmDialogOpen.value = false
}

function openAssignConfirmDialog() {
  const userIdNum = Number(targetUser.value)
  if (!Number.isFinite(userIdNum) || userIdNum <= 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.selectTargetUser'))
    return
  }
  assignConfirmDialogOpen.value = true
}

async function confirmAssignDialog() {
  if (assigning.value) return
  const accountIds = [...selectedIds.value]
  const userIdNum = Number(targetUser.value)
  if (!accountIds.length || !Number.isFinite(userIdNum) || userIdNum <= 0) {
    appStore.showError(t('admin.socialAccountWorkbench.toasts.selectTargetUser'))
    return
  }
  assigning.value = true
  try {
    const result = await adminAPI.totalAccounts.batchAssign(accountIds, userIdNum)
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.assigned', { count: result.succeeded, user: selectedTargetUser.value?.email ?? `#${userIdNum}` }))
    closeAssignDialog()
    clearSelection()
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.assign', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  } finally {
    assigning.value = false
  }
}

function openReclaimDialog() {
  if (!hasSelection.value) return
  reclaimDialogOpen.value = true
}

async function reclaimSelectedAccounts() {
  if (reclaiming.value || !hasSelection.value) return
  const accountIds = [...selectedIds.value]
  reclaiming.value = true
  try {
    const result = await adminAPI.totalAccounts.batchReclaim(accountIds)
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.reclaimed', { count: result.succeeded }))
    reclaimDialogOpen.value = false
    clearSelection()
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.reclaim', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  } finally {
    reclaiming.value = false
  }
}

function openDeleteDialog() {
  if (!hasSelection.value) return
  deleteDialogOpen.value = true
}

async function confirmDeleteDialog() {
  if (deleting.value || !hasSelection.value) return
  const accountIds = [...selectedIds.value]
  const deleteCount = accountIds.length
  deleting.value = true
  try {
    const result = await adminAPI.totalAccounts.batchDelete(accountIds)
    appStore.showSuccess(t('admin.socialAccountWorkbench.toasts.deleted', { count: result.succeeded || deleteCount }))
    deleteDialogOpen.value = false
    clearSelection()
    await loadAccounts()
  } catch (error) {
    recordClientDiagnostic('admin.total_accounts.delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  } finally {
    deleting.value = false
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

function normalizePlatform(value?: string | null): string {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[-/\s]+/g, '_')
  if (['twitter', 'x', 'x_twitter', 'twitter_x'].includes(normalized)) return 'x_twitter'
  return normalized
}

function normalizeUsername(value?: string | null): string {
  return String(value || '').trim().toLowerCase().replace(/^@+/, '').trim()
}

function platformInitial(value?: string | null): string {
  const normalized = normalizePlatform(value)
  if (normalized === 'x_twitter') return 'X'
  return (normalized || '?').slice(0, 2).toUpperCase()
}

function platformAvatarClass(value?: string | null): string {
  const normalized = normalizePlatform(value)
  if (normalized === 'x_twitter') return 'border-gray-900 bg-gray-900 text-white dark:border-gray-100 dark:bg-gray-100 dark:text-gray-950'
  if (normalized === 'instagram') return 'border-pink-200 bg-pink-50 text-pink-700 dark:border-pink-900/50 dark:bg-pink-900/20 dark:text-pink-300'
  if (normalized === 'tiktok') return 'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-900/50 dark:bg-cyan-900/20 dark:text-cyan-300'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200'
}

function toAccountStatus(status: string): AccountStatus {
  if (status === 'pending_check' || status === 'available' || status === 'limited' || status === 'invalid' || status === 'not_stored') {
    return status
  }
  return 'not_stored'
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
