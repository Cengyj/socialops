<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-64" data-filter-user-search>
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="filterUserKeyword"
                type="text"
                :placeholder="t('admin.users.searchUsers')"
                class="input pl-10 pr-8"
                @input="debounceSearchFilterUsers"
                @focus="showFilterUserDropdown = true"
              />
              <button
                v-if="selectedFilterUser"
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                :title="t('common.clear')"
                @click="clearFilterUser"
              >
                <Icon name="x" size="sm" :stroke-width="2" />
              </button>

              <div
                v-if="showFilterUserDropdown && (filterUserResults.length > 0 || filterUserKeyword)"
                class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
              >
                <div v-if="filterUserLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('common.loading') }}
                </div>
                <div
                  v-else-if="filterUserResults.length === 0 && filterUserKeyword"
                  class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
                >
                  {{ t('common.noOptionsFound') }}
                </div>
                <button
                  v-for="user in filterUserResults"
                  :key="user.id"
                  type="button"
                  class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
                  @click="selectFilterUser(user)"
                >
                  <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
                  <span class="ml-2 text-gray-500 dark:text-gray-400">#{{ user.id }}</span>
                </button>
              </div>
            </div>

            <div class="w-full sm:w-40">
              <Select
                v-model="filters.status"
                :options="statusOptions"
                :placeholder="t('admin.subscriptions.allStatus')"
                @change="applyFilters"
              />
            </div>
            <div class="w-full sm:w-48">
              <Select
                v-model="filters.plan_id"
                :options="planFilterOptions"
                :placeholder="t('admin.subscriptions.allPackages')"
                searchable
                @change="applyFilters"
              />
            </div>
            <div class="w-full sm:w-44">
              <Select
                v-model="filters.platform"
                :options="platformFilterOptions"
                :placeholder="t('admin.subscriptions.allPlatforms')"
                @change="applyFilters"
              />
            </div>
          </div>

          <div class="ml-auto flex flex-wrap items-center justify-end gap-3">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadSubscriptions"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>

            <div ref="columnDropdownRef" class="relative">
              <button
                type="button"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('admin.users.columnSettings')"
                @click="showColumnDropdown = !showColumnDropdown"
              >
                <Icon name="cog" size="sm" class="md:mr-1.5" />
                <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
              </button>
              <div
                v-if="showColumnDropdown"
                class="absolute right-0 z-50 mt-2 w-52 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
              >
                <div class="p-2">
                  <div class="mb-2 border-b border-gray-200 pb-2 dark:border-gray-700">
                    <div class="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                      {{ t('admin.subscriptions.columns.user') }}
                    </div>
                    <button
                      type="button"
                      class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                      @click="setUserColumnMode('email')"
                    >
                      <span>{{ t('admin.users.columns.email') }}</span>
                      <Icon v-if="userColumnMode === 'email'" name="check" size="sm" class="text-primary-500" />
                    </button>
                    <button
                      type="button"
                      class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                      @click="setUserColumnMode('username')"
                    >
                      <span>{{ t('admin.users.columns.username') }}</span>
                      <Icon v-if="userColumnMode === 'username'" name="check" size="sm" class="text-primary-500" />
                    </button>
                  </div>
                  <button
                    v-for="col in toggleableColumns"
                    :key="col.key"
                    type="button"
                    class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                    @click="toggleColumn(col.key)"
                  >
                    <span>{{ col.label }}</span>
                    <Icon v-if="isColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
                  </button>
                </div>
              </div>
            </div>

            <button
              type="button"
              class="btn btn-secondary"
              :title="t('admin.subscriptions.guide.showGuide')"
              @click="showGuideModal = true"
            >
              <Icon name="questionCircle" size="md" />
            </button>
            <button
              type="button"
              class="btn btn-secondary"
              :title="t('admin.subscriptions.managePlansHint')"
              @click="goToPlanManagement"
            >
              <Icon name="externalLink" size="md" class="mr-2" />
              {{ t('admin.subscriptions.managePlans') }}
            </button>
            <button type="button" class="btn btn-secondary" @click="openBulkCreate">
              {{ t('admin.subscriptions.bulkCreate') }}
            </button>
            <button type="button" class="btn btn-primary" @click="openCreate">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.subscriptions.createSubscription') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="subscriptions"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          row-key="id"
          :estimate-row-height="112"
          @sort="handleSort"
        >
          <template #cell-user="{ row }">
            <div class="flex items-center gap-2">
              <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
                <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
                  {{ userInitial(row) }}
                </span>
              </div>
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">{{ userDisplay(row) }}</div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ secondaryUserDisplay(row) }}
                </div>
              </div>
            </div>
          </template>

          <template #cell-group="{ row }">
            <SubscriptionPackageBadge
              v-if="subscriptionDisplayName(row)"
              :name="subscriptionDisplayName(row) || `#${row.group_id}`"
              :platform="subscriptionDisplayPlatform(row)"
              :quota-display="formatLimitValue(subscriptionQuotaLimit(row))"
              compact
            />
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-usage="{ row }">
            <div class="min-w-[280px] space-y-2">
              <UsageProgressRow
                v-if="subscriptionQuotaUsage(row)"
                :label="subscriptionQuotaLabel(row)"
                :used="subscriptionQuotaUsage(row)?.used || 0"
                :limit="subscriptionQuotaUsage(row)?.amount || 0"
                :reset-text="subscriptionQuotaResetText(row)"
              />
              <div
                v-if="!subscriptionHasUsageLimits(row)"
                class="flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-2 dark:bg-emerald-900/20"
              >
                <Icon name="sparkles" size="sm" class="text-emerald-600 dark:text-emerald-400" />
                <span class="text-xs font-medium text-emerald-700 dark:text-emerald-300">
                  {{ t('admin.subscriptions.unlimited') }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-expires_at="{ value }">
            <div v-if="value">
              <span
                class="text-sm"
                :class="
                  isExpiringSoon(String(value))
                    ? 'text-orange-600 dark:text-orange-400'
                    : 'text-gray-700 dark:text-gray-300'
                "
              >
                {{ formatDateOnly(String(value)) }}
              </span>
              <div v-if="getDaysRemaining(String(value)) !== null" class="text-xs text-gray-500">
                {{ getDaysRemaining(String(value)) }} {{ t('admin.subscriptions.daysRemaining') }}
              </div>
            </div>
            <span v-else class="text-sm text-gray-500">{{ t('admin.subscriptions.noExpiration') }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(String(value))]">
              {{ t(`admin.subscriptions.status.${value}`) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                v-if="row.status === 'active' || row.status === 'expired'"
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                @click="handleExtend(row)"
              >
                <Icon name="calendar" size="sm" />
                <span class="text-xs">{{ t('admin.subscriptions.adjust') }}</span>
              </button>
              <button
                v-if="row.status === 'active'"
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-orange-50 hover:text-orange-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-orange-900/20 dark:hover:text-orange-400"
                :disabled="resettingQuota && resettingSubscription?.id === row.id"
                @click="handleResetQuota(row)"
              >
                <Icon name="refresh" size="sm" />
                <span class="text-xs">{{ t('admin.subscriptions.resetQuota') }}</span>
              </button>
              <button
                v-if="row.status === 'active'"
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                @click="handleRevoke(row)"
              >
                <Icon name="ban" size="sm" />
                <span class="text-xs">{{ t('admin.subscriptions.revoke') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.subscriptions.noSubscriptionsYet')"
              :description="t('admin.subscriptions.createFirstSubscription')"
            >
              <template #action>
                <div class="flex flex-col items-center justify-center gap-3 sm:flex-row">
                  <button type="button" class="btn btn-primary" @click="openCreate">
                    <Icon name="plus" size="md" class="mr-2" />
                    {{ t('admin.subscriptions.createSubscription') }}
                  </button>
                  <button type="button" class="btn btn-secondary" @click="goToPlanManagement">
                    <Icon name="externalLink" size="md" class="mr-2" />
                    {{ t('admin.subscriptions.managePlans') }}
                  </button>
                </div>
              </template>
            </EmptyState>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showCreateModal"
      :title="bulkMode ? t('admin.subscriptions.bulkCreate') : t('admin.subscriptions.createSubscription')"
      width="normal"
      @close="closeCreateModal"
    >
      <form id="create-subscription-form" class="space-y-5" @submit.prevent="handleCreateSubscription">
        <div v-if="bulkMode">
          <label class="input-label">{{ t('admin.subscriptions.form.user') }}</label>
          <textarea
            v-model="bulkUserIds"
            class="input min-h-[120px]"
            :placeholder="t('admin.subscriptions.bulkUserIds')"
          ></textarea>
        </div>

        <div v-else>
          <label class="input-label">{{ t('admin.subscriptions.form.user') }}</label>
          <div class="relative" data-create-user-search>
            <input
              v-model="userSearchKeyword"
              type="text"
              class="input pr-8"
              :placeholder="t('admin.subscriptions.selectUser')"
              @input="debounceSearchUsers"
              @focus="showUserDropdown = true"
            />
            <button
              v-if="selectedUser"
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              :title="t('common.clear')"
              @click="clearUserSelection"
            >
              <Icon name="x" size="sm" :stroke-width="2" />
            </button>
            <div
              v-if="showUserDropdown && (userSearchResults.length > 0 || userSearchKeyword)"
              class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
            >
              <div v-if="userSearchLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                {{ t('common.loading') }}
              </div>
              <div
                v-else-if="userSearchResults.length === 0 && userSearchKeyword"
                class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
              >
                {{ t('common.noOptionsFound') }}
              </div>
              <button
                v-for="user in userSearchResults"
                :key="user.id"
                type="button"
                class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
                @click="selectUser(user)"
              >
                <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
                <span class="ml-2 text-gray-500 dark:text-gray-400">#{{ user.id }}</span>
              </button>
            </div>
          </div>
        </div>

        <div class="space-y-2">
          <label class="input-label">{{ t('admin.subscriptions.form.package') }}</label>
          <div
            v-if="subscriptionQuotaPackages.length === 0"
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            <div>{{ plansLoading ? t('common.loading') : t('admin.subscriptions.noAvailablePackages') }}</div>
            <button
              v-if="!plansLoading"
              type="button"
              class="btn btn-secondary btn-sm mt-3"
              @click="goToPlanManagement"
            >
              <Icon name="externalLink" size="sm" class="mr-1.5" />
              {{ t('admin.subscriptions.noAvailablePackagesAction') }}
            </button>
          </div>
          <Select
            v-else
            v-model="selectedCreatePackageKey"
            :options="subscriptionPackageOptions"
            searchable
            :placeholder="t('admin.subscriptions.form.package')"
            :empty-text="t('admin.subscriptions.noAvailablePackages')"
          >
            <template #selected="{ option }">
              <div v-if="option" class="flex min-w-0 items-center gap-2">
                <span :class="['inline-flex shrink-0 items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium', platformBadgeClass(createPackagePlatform(option))]">
                  <SubscriptionPlatformLogo :platform="createPackagePlatform(option)" compact />
                  <span>{{ createPackagePlatformLabel(option) }}</span>
                </span>
                <span class="truncate">{{ option.label }}</span>
                <span class="hidden shrink-0 text-xs text-gray-400 sm:inline">{{ option.quotaDisplay }}</span>
              </div>
              <span v-else>{{ t('admin.subscriptions.form.package') }}</span>
            </template>
            <template #option="{ option, selected }">
              <div class="flex min-w-0 flex-1 items-start gap-3">
                <span :class="['mt-0.5 inline-flex shrink-0 items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium', platformBadgeClass(createPackagePlatform(option))]">
                  <SubscriptionPlatformLogo :platform="createPackagePlatform(option)" compact />
                  <span>{{ createPackagePlatformLabel(option) }}</span>
                </span>
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ option.label }}</div>
                  <div class="mt-0.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                    <span>{{ option.validityLabel }}</span>
                    <span>{{ option.quotaDisplay }}</span>
                    <span>{{ t('payment.subscriptionPicker.chooseQuota') }}: {{ option.quotaCount }}</span>
                  </div>
                  <p v-if="option.description" class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ option.description }}
                  </p>
                </div>
              </div>
              <Icon v-if="selected" name="check" size="sm" class="mt-1 shrink-0 text-primary-500" />
            </template>
          </Select>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.packageHint') }}
          </p>
        </div>

        <div v-if="selectedCreatePackage" class="space-y-2">
          <label class="input-label">{{ t('payment.subscriptionPicker.chooseQuota') }}</label>
          <SubscriptionQuotaChoiceList
            :quota-package="selectedCreatePackage"
            :selected-plan-id="createForm.plan_id"
            :format-amount="formatCurrency"
            :unlimited-label="t('admin.subscriptions.unlimited')"
            min-width-class="min-w-[120px]"
            @select="selectCreatePlan"
          />
        </div>

        <div
          v-if="selectedCreatePlanOption"
          class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ selectedCreatePlanOption.label }}
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ selectedCreatePlanOption.description || selectedCreatePlanOption.summary }}
              </div>
            </div>
            <span class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
              {{ selectedCreatePlanOption.validityLabel }}
            </span>
          </div>

          <div class="mt-4 grid gap-3 sm:grid-cols-3">
            <div>
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400">{{ t('payment.admin.platform') }}</div>
              <div class="mt-1 inline-flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span :class="['flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border', platformBadgeClass(selectedCreatePlanOption.platform)]">
                  <SubscriptionPlatformLogo :platform="selectedCreatePlanOption.platform" compact />
                </span>
                {{ platformLabel(selectedCreatePlanOption.platform) }}
              </div>
            </div>
            <div>
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400">{{ selectedCreatePlanQuotaLabel }}</div>
              <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
                {{ formatLimitValue(selectedCreatePlanOption.quota_usd) }}
              </div>
            </div>
            <div>
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400">{{ t('admin.subscriptions.defaultValidity') }}</div>
              <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
                {{ selectedCreatePlanOption.validityLabel }}
              </div>
            </div>
          </div>
          <div v-if="selectedCreatePlanOption.guardrailSummary" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
            {{ selectedCreatePlanOption.guardrailSummary }}
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.validityDays') }}</label>
          <input v-model.number="createForm.validity_days" type="number" min="1" class="input" />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.validityHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.notes') }}</label>
          <textarea
            v-model="createForm.notes"
            rows="3"
            class="input"
            :placeholder="t('admin.subscriptions.notesPlaceholder')"
          ></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.notesHint') }}
          </p>
        </div>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeCreateModal">
          {{ t('common.cancel') }}
        </button>
        <button
          form="create-subscription-form"
          type="submit"
          class="btn btn-primary"
          :disabled="submitting"
        >
          {{ submitting ? t('admin.subscriptions.creating') : t('admin.subscriptions.create') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showExtendModal"
      :title="t('admin.subscriptions.adjustSubscription')"
      width="normal"
      @close="closeExtendModal"
    >
      <form id="extend-subscription-form" class="space-y-4" @submit.prevent="handleExtendSubscription">
        <div v-if="extendingSubscription" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-700/50">
          <div class="text-gray-500 dark:text-gray-400">{{ t('admin.subscriptions.adjustingFor') }}</div>
          <div class="mt-1 font-medium text-gray-900 dark:text-white">
            {{ extendingSubscription.user?.email || `#${extendingSubscription.user_id}` }}
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.currentExpiration') }}:
            {{ extendingSubscription.expires_at ? formatDateOnly(extendingSubscription.expires_at) : t('admin.subscriptions.noExpiration') }}
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.adjustDays') }}</label>
          <input
            v-model.number="extendForm.days"
            type="number"
            class="input"
            min="-36500"
            max="36500"
            :placeholder="t('admin.subscriptions.adjustDaysPlaceholder')"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.subscriptions.adjustHint') }}
          </p>
        </div>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeExtendModal">
          {{ t('common.cancel') }}
        </button>
        <button
          form="extend-subscription-form"
          type="submit"
          class="btn btn-primary"
          :disabled="submitting"
        >
          {{ submitting ? t('admin.subscriptions.adjusting') : t('admin.subscriptions.adjust') }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showRevokeDialog"
      :title="t('admin.subscriptions.revokeSubscription')"
      :message="t('admin.subscriptions.revokeConfirm', { user: subscriptionUserLabel(revokingSubscription) })"
      :confirm-text="t('admin.subscriptions.revoke')"
      danger
      @confirm="confirmRevoke"
      @cancel="showRevokeDialog = false"
    />

    <ConfirmDialog
      :show="showResetQuotaConfirm"
      :title="t('admin.subscriptions.resetQuotaTitle')"
      :message="t('admin.subscriptions.resetQuotaConfirm', { user: subscriptionUserLabel(resettingSubscription) })"
      :confirm-text="t('admin.subscriptions.resetQuota')"
      @confirm="confirmResetQuota"
      @cancel="showResetQuotaConfirm = false"
    />

    <BaseDialog
      :show="showGuideModal"
      :title="t('admin.subscriptions.guide.title')"
      width="wide"
      @close="showGuideModal = false"
    >
      <div class="space-y-5 text-sm text-gray-600 dark:text-gray-300">
        <p>{{ t('admin.subscriptions.guide.subtitle') }}</p>
        <div class="grid gap-4 md:grid-cols-3">
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ t('admin.subscriptions.guide.step1.title') }}
            </div>
            <ul class="mt-3 space-y-2">
              <li>{{ t('admin.subscriptions.guide.step1.line1') }}</li>
              <li>{{ t('admin.subscriptions.guide.step1.line2') }}</li>
              <li>{{ t('admin.subscriptions.guide.step1.line3') }}</li>
            </ul>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ t('admin.subscriptions.guide.step2.title') }}
            </div>
            <ul class="mt-3 space-y-2">
              <li>{{ t('admin.subscriptions.guide.step2.line1') }}</li>
              <li>{{ t('admin.subscriptions.guide.step2.line2') }}</li>
              <li>{{ t('admin.subscriptions.guide.step2.line3') }}</li>
            </ul>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ t('admin.subscriptions.guide.step3.title') }}
            </div>
            <div class="mt-3 space-y-3">
              <div v-for="row in guideActionRows" :key="row.action">
                <div class="font-medium text-gray-800 dark:text-gray-100">{{ row.action }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.desc }}</div>
              </div>
            </div>
          </div>
        </div>
        <p class="rounded-lg bg-primary-50 px-4 py-3 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.subscriptions.guide.tip') }}
        </p>
        <button type="button" class="btn btn-secondary" @click="goToPlanManagement">
          <Icon name="externalLink" size="md" class="mr-2" />
          {{ t('admin.subscriptions.managePlans') }}
        </button>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import SubscriptionPackageBadge from '@/components/payment/SubscriptionPackageBadge.vue'
import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'
import SubscriptionQuotaChoiceList from '@/components/payment/SubscriptionQuotaChoiceList.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { UserSubscription } from '@/types'
import type { SubscriptionPlan } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatDateOnly } from '@/utils/format'
import { getRemainingDurationParts, isOneTimeDailyQuota, type RemainingDurationParts } from '@/utils/subscriptionQuota'
import {
  getSubscriptionPlatform,
  getSubscriptionTitle,
  hasSubscriptionLimits,
} from '@/utils/subscriptionPackages'
import { getPlanQuotaAmount, getPlanQuotaPeriod, getSubscriptionQuotaUsage, type SubscriptionQuotaUsage } from '@/utils/subscriptionQuotaPlans'
import { getPlatformColor } from '@/utils/platformColors'
import {
  buildSubscriptionQuotaPackages,
  type SubscriptionQuotaPackage,
} from '@/utils/subscriptionPlanCatalog'
import {
  formatSubscriptionPlanValidity,
  formatSubscriptionQuotaAmount,
  getSubscriptionPlatformLabel,
  toSubscriptionPlanOption,
  type SubscriptionPlanOption,
} from '@/utils/subscriptionPlanDisplay'

interface SimpleUser {
  id: number
  email: string
  username?: string
}

interface CreatePackageOption extends Record<string, unknown> {
  value: string
  label: string
  description: string
  platform: string
  validityLabel: string
  quotaCount: number
  quotaDisplay: string
}

const UsageProgressRow = defineComponent({
  name: 'UsageProgressRow',
  props: {
    label: { type: String, required: true },
    used: { type: Number, default: 0 },
    limit: { type: Number, required: true },
    resetText: { type: String, default: '' }
  },
  setup(props) {
    const width = computed(() => {
      if (!props.limit) return '0%'
      return `${Math.min(((props.used || 0) / props.limit) * 100, 100)}%`
    })
    const progressClass = computed(() => {
      if (!props.limit) return 'bg-gray-400'
      const percentage = ((props.used || 0) / props.limit) * 100
      if (percentage >= 90) return 'bg-red-500'
      if (percentage >= 70) return 'bg-orange-500'
      return 'bg-green-500'
    })
    return () =>
      h('div', { class: 'usage-row' }, [
        h('div', { class: 'flex items-center gap-2' }, [
          h('span', { class: 'usage-label' }, props.label),
          h('div', { class: 'h-1.5 flex-1 rounded-full bg-gray-200 dark:bg-dark-600' }, [
            h('div', {
              class: ['h-1.5 rounded-full transition-all', progressClass.value],
              style: { width: width.value }
            })
          ]),
          h('span', { class: 'usage-amount' }, `$${(props.used || 0).toFixed(2)} / $${props.limit.toFixed(2)}`)
        ]),
        props.resetText
          ? h('div', { class: 'reset-info' }, [
              h(Icon, { name: 'clock', size: 'xs' }),
              h('span', props.resetText)
            ])
          : null
      ])
  }
})

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const showGuideModal = ref(false)
const guideActionRows = computed(() => [
  { action: t('admin.subscriptions.guide.actions.adjust'), desc: t('admin.subscriptions.guide.actions.adjustDesc') },
  { action: t('admin.subscriptions.guide.actions.resetQuota'), desc: t('admin.subscriptions.guide.actions.resetQuotaDesc') },
  { action: t('admin.subscriptions.guide.actions.revoke'), desc: t('admin.subscriptions.guide.actions.revokeDesc') }
])

const userColumnMode = ref<'email' | 'username'>('email')
const USER_COLUMN_MODE_KEY = 'subscription-user-column-mode'
const HIDDEN_COLUMNS_KEY = 'subscription-hidden-columns'
const hiddenColumns = reactive<Set<string>>(new Set())
const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)

const allColumns = computed<Column[]>(() => [
  {
    key: 'user',
    label: userColumnMode.value === 'email'
      ? t('admin.subscriptions.columns.user')
      : t('admin.users.columns.username'),
    sortable: false
  },
  { key: 'group', label: t('admin.subscriptions.columns.package'), sortable: false },
  { key: 'usage', label: t('admin.subscriptions.columns.usage'), sortable: false },
  { key: 'expires_at', label: t('admin.subscriptions.columns.expires'), sortable: true },
  { key: 'status', label: t('admin.subscriptions.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.subscriptions.columns.actions'), sortable: false }
])

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => col.key !== 'user' && col.key !== 'actions')
)
const columns = computed<Column[]>(() =>
  allColumns.value.filter((col) => col.key === 'user' || col.key === 'actions' || !hiddenColumns.has(col.key))
)

const statusOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allStatus') },
  { value: 'active', label: t('admin.subscriptions.status.active') },
  { value: 'expired', label: t('admin.subscriptions.status.expired') },
  { value: 'revoked', label: t('admin.subscriptions.status.revoked') }
])

const platformFilterOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allPlatforms') },
  { value: 'x_twitter', label: platformLabel('x_twitter') },
  { value: 'instagram', label: platformLabel('instagram') },
  { value: 'tiktok', label: platformLabel('tiktok') },
  { value: 'facebook', label: platformLabel('facebook') }
])

const subscriptions = ref<UserSubscription[]>([])
const plans = ref<SubscriptionPlan[]>([])
const loading = ref(false)
const plansLoading = ref(false)
let abortController: AbortController | null = null

const filterUserKeyword = ref('')
const filterUserResults = ref<SimpleUser[]>([])
const filterUserLoading = ref(false)
const showFilterUserDropdown = ref(false)
const selectedFilterUser = ref<SimpleUser | null>(null)
let filterUserSearchTimeout: ReturnType<typeof setTimeout> | null = null

const userSearchKeyword = ref('')
const userSearchResults = ref<SimpleUser[]>([])
const userSearchLoading = ref(false)
const showUserDropdown = ref(false)
const selectedUser = ref<SimpleUser | null>(null)
let userSearchTimeout: ReturnType<typeof setTimeout> | null = null

const filters = reactive({
  status: 'active',
  plan_id: '',
  platform: '',
  user_id: null as number | null
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const showCreateModal = ref(false)
const bulkMode = ref(false)
const bulkUserIds = ref('')
const showExtendModal = ref(false)
const showRevokeDialog = ref(false)
const showResetQuotaConfirm = ref(false)
const submitting = ref(false)
const resettingSubscription = ref<UserSubscription | null>(null)
const resettingQuota = ref(false)
const extendingSubscription = ref<UserSubscription | null>(null)
const revokingSubscription = ref<UserSubscription | null>(null)

const createForm = reactive({
  user_id: null as number | null,
  plan_id: null as number | null,
  validity_days: 30,
  notes: ''
})

const extendForm = reactive({
  days: 30
})

const planFilterOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allPackages') },
  ...plans.value.map((plan) => ({
    value: String(plan.id),
    label: `${plan.name} - ${formatLimitValue(getPlanQuotaAmount(plan))}`
  }))
])

const createEligiblePlans = computed(() =>
  plans.value.filter((plan) => plan.subscription_type === 'subscription' && plan.group_status === 'active')
)

const subscriptionQuotaPackages = computed(() =>
  buildSubscriptionQuotaPackages(createEligiblePlans.value)
)

const subscriptionPackageOptions = computed<CreatePackageOption[]>(() =>
  subscriptionQuotaPackages.value.map((quotaPackage) => ({
    value: quotaPackage.key,
    label: createPackageTitle(quotaPackage),
    description: quotaPackage.description,
    platform: quotaPackage.platform,
    validityLabel: createPackageValidityLabel(quotaPackage),
    quotaCount: quotaPackage.plans.length,
    quotaDisplay: createPackageQuotaRangeLabel(quotaPackage),
  }))
)

const subscriptionPlanOptions = computed<SubscriptionPlanOption[]>(() =>
  createEligiblePlans.value.map((plan) => toSubscriptionPlanOption(plan, t, navigator.language))
)

const selectedCreatePlanOption = computed(() =>
  subscriptionPlanOptions.value.find((plan) => plan.value === createForm.plan_id) || null
)

const selectedCreatePlan = computed(() =>
  createEligiblePlans.value.find((plan) => plan.id === createForm.plan_id) || null
)

const selectedCreatePlanQuotaLabel = computed(() => {
  const period = selectedCreatePlan.value ? getPlanQuotaPeriod(selectedCreatePlan.value) : null
  if (period === 'daily') return t('payment.planCard.todayQuota')
  if (period === 'weekly') return t('payment.planCard.thisWeekQuota')
  if (period === 'monthly') return t('payment.planCard.thisMonthQuota')
  return t('payment.planCard.periodQuota')
})

const selectedCreatePackage = computed(() =>
  subscriptionQuotaPackages.value.find((quotaPackage) =>
    quotaPackage.plans.some((plan) => plan.id === createForm.plan_id)
  ) || null
)

const selectedCreatePackageKey = computed({
  get: () => selectedCreatePackage.value?.key || '',
  set: (key: string | number | boolean | null) => {
    const quotaPackage = subscriptionQuotaPackages.value.find((item) => item.key === String(key || ''))
    if (quotaPackage) selectCreatePackage(quotaPackage)
  },
})

watch(
  () => createForm.plan_id,
  (planID) => {
    const selectedPlan = subscriptionPlanOptions.value.find((plan) => plan.value === planID)
    if (!selectedPlan) return
    createForm.validity_days = selectedPlan.defaultValidityDays
  }
)

watch(
  subscriptionQuotaPackages,
  () => {
    if (showCreateModal.value) ensureDefaultCreatePlanSelection()
  },
)

function ensureDefaultCreatePlanSelection() {
  if (createForm.plan_id || subscriptionQuotaPackages.value.length === 0) return
  selectCreatePackage(subscriptionQuotaPackages.value[0])
}

function selectCreatePackage(quotaPackage: SubscriptionQuotaPackage) {
  const currentPlan = quotaPackage.plans.find((plan) => plan.id === createForm.plan_id)
  selectCreatePlan(currentPlan || quotaPackage.defaultPlan)
}

function selectCreatePlan(plan: SubscriptionPlan) {
  createForm.plan_id = plan.id
}

function createPackageTitle(quotaPackage: SubscriptionQuotaPackage): string {
  return quotaPackage.title || t('payment.subscriptionPicker.packageFallback', {
    platform: platformLabel(quotaPackage.platform),
    validity: createPackageValidityLabel(quotaPackage),
  })
}

function createPackageValidityLabel(quotaPackage: SubscriptionQuotaPackage): string {
  return formatSubscriptionPlanValidity(quotaPackage.defaultPlan, t, navigator.language)
}

function createPackageQuotaRangeLabel(quotaPackage: SubscriptionQuotaPackage): string {
  const quotaValues = quotaPackage.plans
    .map((plan) => getPlanQuotaAmount(plan))
    .filter((value): value is number => value !== null)

  if (quotaValues.length === 0) return t('admin.subscriptions.unlimited')

  const min = Math.min(...quotaValues)
  const max = Math.max(...quotaValues)
  if (min === max) return formatLimitValue(min)
  return `${formatLimitValue(min)} - ${formatLimitValue(max)}`
}

function loadUserColumnMode() {
  try {
    const saved = localStorage.getItem(USER_COLUMN_MODE_KEY)
    if (saved === 'email' || saved === 'username') userColumnMode.value = saved
  } catch (error) {
    console.error('Failed to load user column mode:', error)
  }
}

function setUserColumnMode(mode: 'email' | 'username') {
  userColumnMode.value = mode
  try {
    localStorage.setItem(USER_COLUMN_MODE_KEY, mode)
  } catch (error) {
    console.error('Failed to save user column mode:', error)
  }
}

function loadSavedColumns() {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (!saved) return
    const parsed = JSON.parse(saved) as string[]
    parsed.forEach((key) => hiddenColumns.add(key))
  } catch (error) {
    console.error('Failed to load subscription columns:', error)
  }
}

function saveColumnsToStorage() {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (error) {
    console.error('Failed to save subscription columns:', error)
  }
}

function toggleColumn(key: string) {
  if (hiddenColumns.has(key)) hiddenColumns.delete(key)
  else hiddenColumns.add(key)
  saveColumnsToStorage()
}

function isColumnVisible(key: string) {
  return !hiddenColumns.has(key)
}

function applyFilters() {
  pagination.page = 1
  void loadSubscriptions()
}

async function loadSubscriptions() {
  abortController?.abort()
  const requestController = new AbortController()
  abortController = requestController
  const { signal } = requestController

  loading.value = true
  try {
    const response = await adminAPI.subscriptions.list(
      pagination.page,
      pagination.page_size,
      {
        status: filters.status ? filters.status as 'active' | 'expired' | 'revoked' : undefined,
        plan_id: filters.plan_id ? Number(filters.plan_id) : undefined,
        platform: filters.platform || undefined,
        user_id: filters.user_id || undefined,
        sort_by: sortState.sort_by,
        sort_order: sortState.sort_order
      },
      { signal }
    )
    if (signal.aborted || abortController !== requestController) return
    subscriptions.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
  } catch (error: any) {
    if (signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') return
    appStore.showError(error?.message || t('admin.subscriptions.failedToLoad'))
    console.error('Error loading subscriptions:', error)
  } finally {
    if (abortController === requestController) {
      loading.value = false
      abortController = null
    }
  }
}

async function loadPlans() {
  plansLoading.value = true
  try {
    const response = await adminPaymentAPI.getPlans()
    plans.value = (response.data || []).map((plan: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...plan,
      features: Array.isArray(plan.features)
        ? plan.features
        : String(plan.features || '').split('\n').map((item) => item.trim()).filter(Boolean),
    }))
  } catch (error) {
    console.error('Error loading plans:', error)
    appStore.showError(t('admin.subscriptions.failedToLoadPlans'))
  } finally {
    plansLoading.value = false
  }
}

function debounceSearchFilterUsers() {
  if (filterUserSearchTimeout) clearTimeout(filterUserSearchTimeout)
  filterUserSearchTimeout = setTimeout(searchFilterUsers, 300)
}

async function searchAdminUsers(keyword: string): Promise<SimpleUser[]> {
  const response = await adminAPI.users.list(1, 10, { search: keyword })
  return (response.items || []).map((user) => ({
    id: user.id,
    email: user.email,
    username: user.username,
  }))
}

async function searchFilterUsers() {
  const keyword = filterUserKeyword.value.trim()
  if (selectedFilterUser.value && keyword !== selectedFilterUser.value.email) {
    selectedFilterUser.value = null
    filters.user_id = null
    applyFilters()
  }
  if (!keyword) {
    filterUserResults.value = []
    return
  }
  filterUserLoading.value = true
  try {
    filterUserResults.value = await searchAdminUsers(keyword)
  } catch (error) {
    console.error('Failed to search users:', error)
    filterUserResults.value = []
  } finally {
    filterUserLoading.value = false
  }
}

function selectFilterUser(user: SimpleUser) {
  selectedFilterUser.value = user
  filterUserKeyword.value = user.email
  showFilterUserDropdown.value = false
  filters.user_id = user.id
  applyFilters()
}

function clearFilterUser() {
  selectedFilterUser.value = null
  filterUserKeyword.value = ''
  filterUserResults.value = []
  showFilterUserDropdown.value = false
  filters.user_id = null
  applyFilters()
}

function debounceSearchUsers() {
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
  userSearchTimeout = setTimeout(searchUsers, 300)
}

async function searchUsers() {
  const keyword = userSearchKeyword.value.trim()
  if (selectedUser.value && keyword !== selectedUser.value.email) {
    selectedUser.value = null
    createForm.user_id = null
  }
  if (!keyword) {
    userSearchResults.value = []
    return
  }
  userSearchLoading.value = true
  try {
    userSearchResults.value = await searchAdminUsers(keyword)
  } catch (error) {
    console.error('Failed to search users:', error)
    userSearchResults.value = []
  } finally {
    userSearchLoading.value = false
  }
}

function selectUser(user: SimpleUser) {
  selectedUser.value = user
  userSearchKeyword.value = user.email
  showUserDropdown.value = false
  createForm.user_id = user.id
}

function clearUserSelection() {
  selectedUser.value = null
  userSearchKeyword.value = ''
  userSearchResults.value = []
  createForm.user_id = null
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadSubscriptions()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadSubscriptions()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadSubscriptions()
}

function openCreate() {
  bulkMode.value = false
  showCreateModal.value = true
  ensureDefaultCreatePlanSelection()
}

function openBulkCreate() {
  bulkMode.value = true
  showCreateModal.value = true
  ensureDefaultCreatePlanSelection()
}

function goToPlanManagement() {
  void router.push('/admin/orders/plans')
}

function closeCreateModal() {
  showCreateModal.value = false
  bulkMode.value = false
  bulkUserIds.value = ''
  createForm.user_id = null
  createForm.plan_id = null
  createForm.validity_days = 30
  createForm.notes = ''
  clearUserSelection()
}

async function handleCreateSubscription() {
  if (!bulkMode.value && !createForm.user_id) {
    appStore.showError(t('admin.subscriptions.pleaseSelectUser'))
    return
  }
  if (!createForm.plan_id) {
    appStore.showError(t('admin.subscriptions.pleaseSelectPlan'))
    return
  }
  if (!createForm.validity_days || createForm.validity_days < 1) {
    appStore.showError(t('admin.subscriptions.validityDaysRequired'))
    return
  }
  const selectedPlan = subscriptionPlanOptions.value.find((plan) => plan.value === createForm.plan_id)
  if (!selectedPlan) {
    appStore.showError(t('admin.subscriptions.pleaseSelectPlan'))
    return
  }

  const ids = bulkMode.value
    ? bulkUserIds.value.split(/[\s,]+/).map((id) => Number(id)).filter((id) => id > 0)
    : []
  if (bulkMode.value && ids.length === 0) {
    appStore.showError(t('admin.subscriptions.pleaseSelectUser'))
    return
  }

  submitting.value = true
  try {
    if (bulkMode.value) {
      const result = await adminAPI.subscriptions.bulkCreate({
        user_ids: Array.from(new Set(ids)),
        plan_id: selectedPlan.plan_id,
        validity_days: createForm.validity_days,
        notes: createForm.notes.trim() || undefined
      })
      appStore.showSuccess(t('admin.subscriptions.bulkCreated', { count: result.success_count }))
    } else {
      await adminAPI.subscriptions.create({
        user_id: createForm.user_id as number,
        plan_id: selectedPlan.plan_id,
        validity_days: createForm.validity_days,
        notes: createForm.notes.trim() || undefined
      })
      appStore.showSuccess(t('admin.subscriptions.subscriptionCreated'))
    }
    closeCreateModal()
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToCreate'))
    console.error('Error creating subscription:', error)
  } finally {
    submitting.value = false
  }
}

function handleExtend(subscription: UserSubscription) {
  extendingSubscription.value = subscription
  extendForm.days = 30
  showExtendModal.value = true
}

function closeExtendModal() {
  showExtendModal.value = false
  extendingSubscription.value = null
}

async function handleExtendSubscription() {
  if (!extendingSubscription.value) return
  if (extendForm.days < -36500 || extendForm.days > 36500) {
    appStore.showError(t('admin.subscriptions.adjustOutOfRange'))
    return
  }
  if (extendingSubscription.value.expires_at) {
    const expiresAt = new Date(extendingSubscription.value.expires_at)
    const newExpiresAt = new Date(expiresAt.getTime() + extendForm.days * 24 * 60 * 60 * 1000)
    if (newExpiresAt <= new Date()) {
      appStore.showError(t('admin.subscriptions.adjustWouldExpire'))
      return
    }
  }

  submitting.value = true
  try {
    await adminAPI.subscriptions.extend(extendingSubscription.value.id, { days: extendForm.days })
    appStore.showSuccess(t('admin.subscriptions.subscriptionAdjusted'))
    closeExtendModal()
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToAdjust'))
    console.error('Error adjusting subscription:', error)
  } finally {
    submitting.value = false
  }
}

function handleRevoke(subscription: UserSubscription) {
  revokingSubscription.value = subscription
  showRevokeDialog.value = true
}

async function confirmRevoke() {
  if (!revokingSubscription.value) return
  try {
    await adminAPI.subscriptions.revoke(revokingSubscription.value.id)
    appStore.showSuccess(t('admin.subscriptions.subscriptionRevoked'))
    showRevokeDialog.value = false
    revokingSubscription.value = null
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToRevoke'))
    console.error('Error revoking subscription:', error)
  }
}

function handleResetQuota(subscription: UserSubscription) {
  resettingSubscription.value = subscription
  showResetQuotaConfirm.value = true
}

async function confirmResetQuota() {
  if (!resettingSubscription.value || resettingQuota.value) return
  resettingQuota.value = true
  try {
    await adminAPI.subscriptions.resetQuota(resettingSubscription.value.id, {
      daily: true,
      weekly: true,
      monthly: true
    })
    appStore.showSuccess(t('admin.subscriptions.quotaResetSuccess'))
    showResetQuotaConfirm.value = false
    resettingSubscription.value = null
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.subscriptions.failedToResetQuota'))
    console.error('Error resetting quota:', error)
  } finally {
    resettingQuota.value = false
  }
}

function getDaysRemaining(expiresAt: string): number | null {
  const diff = new Date(expiresAt).getTime() - Date.now()
  if (diff < 0) return null
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

function isExpiringSoon(expiresAt: string): boolean {
  const days = getDaysRemaining(expiresAt)
  return days !== null && days <= 7
}

function formatResetDuration(parts: RemainingDurationParts): string {
  if (parts.days > 0) {
    return t('admin.subscriptions.resetInDaysHours', { days: parts.days, hours: parts.hours })
  }
  if (parts.hours > 0) {
    return t('admin.subscriptions.resetInHoursMinutes', { hours: parts.hours, minutes: parts.minutes })
  }
  return t('admin.subscriptions.resetInMinutes', { minutes: parts.minutes })
}

function formatQuotaEndDuration(parts: RemainingDurationParts): string {
  if (parts.days > 0) {
    return t('admin.subscriptions.quotaEndsInDaysHours', { days: parts.days, hours: parts.hours })
  }
  if (parts.hours > 0) {
    return t('admin.subscriptions.quotaEndsInHoursMinutes', { hours: parts.hours, minutes: parts.minutes })
  }
  return t('admin.subscriptions.quotaEndsInMinutes', { minutes: parts.minutes })
}

function formatDailyUsageWindow(subscription: UserSubscription): string {
  if (isOneTimeDailyQuota(subscription) && subscription.expires_at) {
    const parts = getRemainingDurationParts(subscription.expires_at)
    return parts ? formatQuotaEndDuration(parts) : t('admin.subscriptions.windowNotActive')
  }
  return formatResetTime(subscription.daily_window_start, 'daily')
}

function formatResetTime(windowStart: string | null, period: 'daily' | 'weekly' | 'monthly'): string {
  if (!windowStart) return t('admin.subscriptions.windowNotActive')
  const start = new Date(windowStart)
  let resetTime: Date
  switch (period) {
    case 'daily':
      resetTime = new Date(start.getTime() + 24 * 60 * 60 * 1000)
      break
    case 'weekly':
      resetTime = new Date(start.getTime() + 7 * 24 * 60 * 60 * 1000)
      break
    case 'monthly':
      resetTime = new Date(start.getTime() + 30 * 24 * 60 * 60 * 1000)
      break
  }
  const parts = getRemainingDurationParts(resetTime)
  return parts ? formatResetDuration(parts) : t('admin.subscriptions.windowNotActive')
}

function statusBadgeClass(status: string): string {
  if (status === 'active') return 'badge-success'
  if (status === 'expired') return 'badge-warning'
  return 'badge-danger'
}

function subscriptionUserLabel(subscription: UserSubscription | null): string {
  if (!subscription) return ''
  return subscription.user?.email || `#${subscription.user_id}`
}

function subscriptionDisplayName(subscription: UserSubscription): string | null {
  return getSubscriptionTitle(subscription)
}

function subscriptionDisplayPlatform(subscription: UserSubscription): string {
  return getSubscriptionPlatform(subscription)
}

function subscriptionQuotaUsage(subscription: UserSubscription): SubscriptionQuotaUsage | null {
  return getSubscriptionQuotaUsage(subscription)
}

function subscriptionQuotaLimit(subscription: UserSubscription): number | null {
  return subscriptionQuotaUsage(subscription)?.amount ?? null
}

function subscriptionQuotaLabel(subscription: UserSubscription): string {
  const period = subscriptionQuotaUsage(subscription)?.period
  if (period === 'daily') return t('admin.subscriptions.todayQuota')
  if (period === 'weekly') return t('admin.subscriptions.thisWeekQuota')
  if (period === 'monthly') return t('admin.subscriptions.thisMonthQuota')
  return t('admin.subscriptions.noLimits')
}

function subscriptionQuotaResetText(subscription: UserSubscription): string {
  const usage = subscriptionQuotaUsage(subscription)
  if (!usage) return ''
  if (usage.period === 'daily') return formatDailyUsageWindow(subscription)
  return formatResetTime(usage.windowStart, usage.period)
}

function subscriptionHasUsageLimits(subscription: UserSubscription): boolean {
  return hasSubscriptionLimits(subscription)
}

function userDisplay(row: UserSubscription): string {
  if (userColumnMode.value === 'username') return row.user?.username || '-'
  return row.user?.email || `#${row.user_id}`
}

function secondaryUserDisplay(row: UserSubscription): string {
  if (userColumnMode.value === 'username') return row.user?.email || `#${row.user_id}`
  return row.user?.username || `#${row.user_id}`
}

function userInitial(row: UserSubscription): string {
  const value = userDisplay(row)
  return value ? value.charAt(0).toUpperCase() : '?'
}

function formatLimitValue(value: number | null | undefined): string {
  return formatSubscriptionQuotaAmount(value, t('admin.subscriptions.unlimited'))
}

function formatCurrency(value: number | null | undefined): string {
  const amount = Number(value || 0)
  return `$${amount.toFixed(2)}`
}

function platformLabel(platform: string | null | undefined): string {
  return getSubscriptionPlatformLabel(platform, t('payment.platformFallback'))
}

function createPackagePlatform(option: Record<string, unknown> | null | undefined): string {
  return String(option?.platform || 'social')
}

function createPackagePlatformLabel(option: Record<string, unknown> | null | undefined): string {
  return platformLabel(createPackagePlatform(option))
}

function platformBadgeClass(platform: string | null | undefined): string {
  const colors = getPlatformColor(platform || 'social')
  return `${colors.bg} ${colors.text} ${colors.border}`
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('[data-create-user-search]')) showUserDropdown.value = false
  if (!target.closest('[data-filter-user-search]')) showFilterUserDropdown.value = false
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) showColumnDropdown.value = false
}

onMounted(() => {
  loadUserColumnMode()
  loadSavedColumns()
  void loadSubscriptions()
  void loadPlans()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  abortController?.abort()
  if (filterUserSearchTimeout) clearTimeout(filterUserSearchTimeout)
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
})
</script>

<style scoped>
.usage-row {
  @apply space-y-1;
}

.usage-label {
  @apply w-10 flex-shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400;
}

.usage-amount {
  @apply whitespace-nowrap text-xs tabular-nums text-gray-600 dark:text-gray-300;
}

.reset-info {
  @apply flex items-center gap-1 pl-12 text-[10px] text-blue-600 dark:text-blue-400;
}
</style>
