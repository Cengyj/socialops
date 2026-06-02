<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <nav class="flex flex-wrap gap-2 rounded-xl border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-700 dark:bg-dark-800" :aria-label="t('nav.socialAccounts')">
            <router-link to="/admin/accounts" :class="accountTabActiveClass">
              <span>{{ t('admin.socialAccountWorkbench.tabs.management') }}</span>
              <span class="rounded-full bg-white/80 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-200">
                {{ t('admin.socialAccountWorkbench.tabs.managementCount', { count: managementTabCount }) }}
              </span>
            </router-link>
            <router-link to="/admin/total-accounts" :class="accountTabInactiveClass">
              <span>{{ t('admin.socialAccountWorkbench.tabs.pool') }}</span>
              <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ t('admin.socialAccountWorkbench.tabs.poolCount', { count: poolTabCount }) }}
              </span>
            </router-link>
          </nav>

          <div class="grid gap-3 sm:grid-cols-3">
            <div v-for="stat in stats" :key="stat.label" class="rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
            </div>
          </div>

          <div class="flex flex-col gap-3 rounded-xl border border-primary-100 bg-primary-50/60 p-3 dark:border-primary-900/40 dark:bg-primary-900/10 xl:flex-row xl:items-center xl:justify-between">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <SearchInput v-model="searchQuery" :placeholder="t('admin.socialAccountWorkbench.searchPlaceholder')" class="w-full sm:w-72" />
              <Select v-model="accountStatusFilter" :options="accountStatusOptions" class="w-full sm:w-44" />
            </div>
            <div class="flex flex-wrap items-center justify-end gap-3">
              <span class="rounded-full bg-white px-3 py-1 text-sm font-medium text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white">
                {{ t('admin.socialAccountWorkbench.selection.selectedCount', { count: selectedIds.length }) }}
              </span>
              <button class="btn btn-secondary" :disabled="selectedIds.length === 0" @click="clearSelection">
                {{ t('admin.socialAccountWorkbench.executionBar.clear') }}
              </button>
              <button class="btn btn-secondary" @click="openRegisterDialog">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.actions.register') }}
              </button>
              <button class="btn btn-secondary" @click="openCreateDialog">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.actions.manualImport') }}
              </button>
              <button class="btn btn-secondary" @click="openImportDialog">
                <Icon name="upload" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.toolbar.importAccounts') }}
              </button>
              <button class="btn btn-secondary" @click="exportAccounts">
                <Icon name="download" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.toolbar.exportRecords') }}
              </button>
              <Select v-model="executionAction" :options="executionActionOptions" class="w-full sm:w-52" />
              <button class="btn btn-primary" :disabled="!canStartExecution" @click="startExecution">
                <Icon name="play" size="md" class="mr-2" />
                {{ t('admin.socialAccountWorkbench.executionBar.start') }}
              </button>
              <input ref="importFileInput" type="file" accept=".csv,.json" class="hidden" @change="handleImportFile" />
            </div>
          </div>

          <div v-if="executionRuns.length > 0" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div>
                <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.socialAccountWorkbench.executionLog.recentTitle') }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.recentHint') }}</div>
              </div>
              <span class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
                {{ t('admin.socialAccountWorkbench.executionLog.activeCount', { count: activeExecutionCount }) }}
              </span>
            </div>
            <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              <button
                v-for="run in recentExecutionRuns"
                :key="run.id"
                type="button"
                class="group rounded-lg border border-gray-200 bg-gray-50 p-3 text-left transition-all hover:border-primary-300 hover:bg-primary-50/70 hover:shadow-sm dark:border-dark-600 dark:bg-dark-700/70 dark:hover:border-primary-800 dark:hover:bg-primary-900/20"
                @click="openExecutionLog(run.id)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ run.title }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.cardMeta', { count: run.targetCount, time: run.startedAt }) }}</div>
                  </div>
                  <span :class="['shrink-0 rounded-full px-2 py-0.5 text-xs font-medium', executionStatusBadgeClass(run.status)]">
                    {{ t(`admin.socialAccountWorkbench.executionLog.status.${run.status}`) }}
                  </span>
                </div>
                <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div :class="['h-full rounded-full transition-all', executionProgressClass(run.status)]" :style="{ width: `${run.progress}%` }"></div>
                </div>
                <div class="mt-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                  <span>{{ run.progress }}%</span>
                  <span class="font-medium text-primary-600 opacity-0 transition-opacity group-hover:opacity-100 dark:text-primary-300">{{ t('admin.socialAccountWorkbench.executionLog.open') }}</span>
                </div>
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredAccounts" row-key="id" default-sort-key="id" default-sort-order="desc">
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
          <template #cell-account="{ row }">
            <button class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openDetailDialog(row)">{{ row.account }}</button>
          </template>
          <template #cell-password="{ value }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
          </template>
          <template #cell-emailPassword="{ value }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
          </template>
          <template #cell-accountStatus="{ value }">
            <span :class="['badge', accountStatusBadgeClass(String(value))]">{{ accountStatusLabel(String(value)) }}</span>
          </template>
          <template #cell-taskStatus="{ value }">
            <span :class="['badge', taskStatusBadgeClass(String(value))]">{{ taskStatusLabel(String(value)) }}</span>
          </template>
          <template #cell-source="{ value }">
            <span class="badge badge-primary">{{ sourceLabel(String(value)) }}</span>
          </template>
          <template #cell-taskMessage="{ row }">
            <span class="block min-w-[220px] max-w-sm truncate text-sm text-gray-600 dark:text-gray-300" :title="row.taskMessage">
              {{ row.taskMessage || '-' }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <button class="btn btn-secondary px-2 py-1 text-xs" @click="openEditDialog(row)">{{ t('common.edit') }}</button>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="detailDialogOpen" :title="t('admin.socialAccountWorkbench.detailTitle')" width="wide" @close="detailDialogOpen = false">
      <div v-if="selectedAccount" class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.tabs.managementDescription') }}
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div v-for="item in detailItems" :key="item.label" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <div class="text-gray-500 dark:text-gray-400">{{ item.label }}</div>
            <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.value || '-' }}</div>
          </div>
        </div>
        <div v-if="selectedAccount.taskMessage" :class="['rounded-lg border p-3 text-sm', taskMessagePanelClass(selectedAccount.taskStatus)]">
          {{ selectedAccount.taskMessage }}
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="selectedAccount && openEditDialog(selectedAccount)">{{ t('common.edit') }}</button>
        <button class="btn btn-primary" @click="detailDialogOpen = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="accountDialogOpen" :title="accountDialogTitle" width="wide" @close="accountDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.tabs.managementDescription') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <input v-model="accountForm.name" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.account')" />
          <input v-model="accountForm.platform" type="text" class="input" :disabled="selectedAccountId !== null" :placeholder="t('admin.socialAccountWorkbench.form.platform')" />
          <input v-model="accountForm.accountId" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.accountId')" />
          <Select v-if="selectedAccountId" v-model="accountForm.accountStatus" :options="accountStatusOptionsWithoutAll" />
          <input v-model="accountForm.password" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.password')" />
          <input v-model="accountForm.phone" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.phone')" />
          <input v-model="accountForm.email" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.email')" />
          <input v-model="accountForm.emailPassword" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.emailPassword')" />
          <textarea v-model="accountForm.remark" class="input min-h-[88px] sm:col-span-2" :placeholder="t('admin.socialAccountWorkbench.form.remark')"></textarea>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="accountDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!canSubmitAccount" @click="submitAccountDialog">{{ t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="registerDialogOpen" :title="t('admin.socialAccountWorkbench.actions.register')" width="wide" @close="registerDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-300">
          {{ t('admin.socialAccountWorkbench.registerNotConfigured') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <input v-model="registerForm.name" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.account')" />
          <input v-model="registerForm.platform" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.platform')" />
          <input v-model="registerForm.password" type="text" class="input" :placeholder="t('admin.socialAccountWorkbench.form.password')" />
          <textarea v-model="registerForm.remark" class="input min-h-[88px] sm:col-span-2" :placeholder="t('admin.socialAccountWorkbench.form.remark')"></textarea>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="registerDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!registerForm.name.trim() || !registerForm.platform.trim()" @click="submitRegister">{{ t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="importDialogOpen" :title="t('admin.socialAccountWorkbench.actions.fileUpload')" width="wide" @close="importDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.form.fileHint') }}
        </div>

        <button type="button" class="input flex min-h-[44px] items-center justify-between gap-3 text-left" @click="triggerImport">
          <span class="truncate text-gray-700 dark:text-gray-200">
            {{ selectedImportFileName || t('admin.socialAccountWorkbench.form.filePlaceholder') }}
          </span>
          <Icon name="upload" size="sm" class="shrink-0 text-gray-400" />
        </button>

        <div class="grid gap-3 sm:grid-cols-4">
          <div v-for="item in uploadPreview" :key="item.label" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</div>
            <div class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="importDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!pendingImportFile" @click="submitImportFile">{{ t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="executionParamDialogOpen" :title="t('admin.socialAccountWorkbench.executionParams.title')" width="wide" @close="executionParamDialogOpen = false">
      <div class="space-y-4">
        <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-800/60 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('admin.socialAccountWorkbench.executionParams.hint', { action: executionActionLabel(executionAction), count: selectedIds.length }) }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <input
            v-if="executionNeedsTarget"
            v-model="executionTarget"
            type="text"
            class="input"
            :placeholder="t('admin.socialAccountWorkbench.executionParams.target')"
          />
          <textarea
            v-if="executionNeedsContent"
            v-model="executionContent"
            class="input min-h-[110px] sm:col-span-2"
            :placeholder="t('admin.socialAccountWorkbench.executionParams.content')"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="executionParamDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!canSubmitExecutionParams" @click="submitExecutionFromDialog">{{ t('admin.socialAccountWorkbench.executionBar.start') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="executionLogDialogOpen" :title="activeExecutionRun?.title ?? t('admin.socialAccountWorkbench.executionLog.title')" width="wide" @close="executionLogDialogOpen = false">
      <div v-if="activeExecutionRun" class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-3">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.summary.status') }}</div>
            <div :class="['mt-1 text-sm font-semibold', executionStatusTextClass(activeExecutionRun.status)]">{{ t(`admin.socialAccountWorkbench.executionLog.status.${activeExecutionRun.status}`) }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.summary.scope') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.socialAccountWorkbench.executionLog.summary.scopeValue', { count: activeExecutionRun.targetCount }) }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.summary.startedAt') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ activeExecutionRun.startedAt }}</div>
          </div>
        </div>

        <div class="rounded-xl border border-gray-700 bg-gray-900 shadow-inner dark:border-gray-800 dark:bg-black">
          <div class="flex items-center justify-between border-b border-gray-800 px-4 py-3">
            <div class="flex items-center gap-2 text-sm font-medium text-gray-200">
              <Icon name="terminal" size="sm" class="text-primary-300" />
              <span>{{ t('admin.socialAccountWorkbench.executionLog.terminalTitle') }}</span>
              <span v-if="activeExecutionRun.status === 'running'" class="h-2 w-2 animate-pulse rounded-full bg-yellow-400"></span>
            </div>
            <button type="button" class="rounded-lg bg-gray-800/80 p-1.5 text-gray-400 transition-colors hover:bg-gray-700 hover:text-white" :title="t('admin.socialAccountWorkbench.executionLog.copy')" @click="copyExecutionLog(activeExecutionRun)">
              <Icon name="copy" size="sm" />
            </button>
          </div>
          <div ref="executionTerminalRef" class="max-h-[340px] min-h-[240px] overflow-y-auto p-4 font-mono text-sm">
            <div v-for="line in activeExecutionRun.lines" :key="line.id" :class="logLineClass(line.level)">
              <span class="text-gray-500">[{{ line.time }}]</span>
              <span class="ml-2">{{ line.text }}</span>
            </div>
            <div v-if="activeExecutionRun.status === 'running'" class="mt-2 text-yellow-400">
              {{ t('admin.socialAccountWorkbench.executionLog.runningCursor') }}<span class="animate-pulse">_</span>
            </div>
          </div>
        </div>

        <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
          <div :class="['h-full rounded-full transition-all', executionProgressClass(activeExecutionRun.status)]" :style="{ width: `${activeExecutionRun.progress}%` }"></div>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.socialAccountWorkbench.executionLog.closeHint') }}</span>
          <button class="btn btn-primary" @click="executionLogDialogOpen = false">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
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
import type { SocialTaskLog } from '@/api/socialAccounts'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'

type AccountStatus = 'pending_check' | 'available' | 'limited' | 'invalid' | 'not_stored'
type Source = 'registered' | 'manual_import' | 'file_upload'
type ExecutionAction = 'register' | 'login_check' | 'manualImport' | 'fileUpload' | 'follow' | 'message' | 'post' | 'like'
type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed'
type LogLineLevel = 'info' | 'warning' | 'error'

interface AccountRow {
  id: number
  account: string
  platform: string
  accountId: string
  password: string
  phone: string
  email: string
  emailPassword: string
  accountStatus: AccountStatus
  taskStatus: string
  taskMessage: string
  source: Source
  assignedUserId: number | null
  remark: string
  createdAt: string
}

interface ExecutionLogLine {
  id: number
  time: string
  level: LogLineLevel
  text: string
}

interface ExecutionRun {
  id: number
  action: ExecutionAction
  title: string
  targetCount: number
  startedAt: string
  status: ExecutionStatus
  progress: number
  lines: ExecutionLogLine[]
  logStatuses: Record<number, string>
}

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const searchQuery = ref('')
const accountStatusFilter = ref('all')
const executionAction = ref<ExecutionAction>('register')
const selectedIds = ref<number[]>([])
const accounts = ref<AccountRow[]>([])
const selectedAccount = ref<AccountRow | null>(null)
const selectedAccountId = ref<number | null>(null)
const detailDialogOpen = ref(false)
const accountDialogOpen = ref(false)
const registerDialogOpen = ref(false)
const importDialogOpen = ref(false)
const executionParamDialogOpen = ref(false)
const importFileInput = ref<HTMLInputElement | null>(null)
const pendingImportFile = ref<File | null>(null)
const selectedImportFileName = ref('')
const executionTarget = ref('')
const executionContent = ref('')
const executionRuns = ref<ExecutionRun[]>([])
const executionLogDialogOpen = ref(false)
const activeExecutionRunId = ref<number | null>(null)
const executionTerminalRef = ref<HTMLElement | null>(null)
let nextExecutionRunId = 1
let nextExecutionLineId = 1
const executionTimers: number[] = []

const accountForm = reactive({
  name: '',
  platform: 'x_twitter',
  accountId: '',
  password: '',
  phone: '',
  email: '',
  emailPassword: '',
  accountStatus: 'pending_check',
  remark: '',
})

const registerForm = reactive({
  name: '',
  platform: 'x_twitter',
  password: '',
  remark: '',
})

onMounted(loadAccounts)

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', width: '48px' },
  { key: 'id', label: t('admin.socialAccountWorkbench.columns.id'), sortable: true },
  { key: 'account', label: t('admin.socialAccountWorkbench.columns.account'), sortable: true },
  { key: 'platform', label: t('admin.socialAccountWorkbench.columns.platform'), sortable: true },
  { key: 'password', label: t('admin.socialAccountWorkbench.columns.password') },
  { key: 'phone', label: t('admin.socialAccountWorkbench.columns.phone'), sortable: true },
  { key: 'email', label: t('admin.socialAccountWorkbench.columns.email'), sortable: true },
  { key: 'emailPassword', label: t('admin.socialAccountWorkbench.columns.emailPassword') },
  { key: 'accountStatus', label: t('admin.socialAccountWorkbench.columns.accountStatus'), sortable: true },
  { key: 'taskStatus', label: t('admin.socialAccountWorkbench.columns.taskStatus'), sortable: true },
  { key: 'source', label: t('admin.socialAccountWorkbench.columns.source'), sortable: true },
  { key: 'taskMessage', label: t('admin.socialAccountWorkbench.columns.taskMessage') },
  { key: 'actions', label: t('admin.socialAccountWorkbench.columns.actions') },
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

const executionActionOptions = computed(() => [
  { value: 'register', label: t('admin.socialAccountWorkbench.actions.register') },
  { value: 'login_check', label: t('admin.socialAccountWorkbench.executionActions.login_check') },
  { value: 'manualImport', label: t('admin.socialAccountWorkbench.actions.manualImport') },
  { value: 'fileUpload', label: t('admin.socialAccountWorkbench.actions.fileUpload') },
  { value: 'message', label: t('admin.socialAccountWorkbench.executionActions.message') },
  { value: 'follow', label: t('admin.socialAccountWorkbench.executionActions.follow') },
  { value: 'post', label: t('admin.socialAccountWorkbench.executionActions.post') },
  { value: 'like', label: t('admin.socialAccountWorkbench.executionActions.like') },
])

const filteredAccounts = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return accounts.value.filter(account => {
    const values = [String(account.id), account.account, account.platform, account.accountId, account.phone, account.email, account.taskMessage, account.remark]
    const matchesKeyword = !keyword || values.some(value => value.toLowerCase().includes(keyword))
    const matchesStatus = accountStatusFilter.value === 'all' || account.accountStatus === accountStatusFilter.value
    return matchesKeyword && matchesStatus
  })
})

const stats = computed(() => [
  { label: t('admin.socialAccountWorkbench.stats.total'), value: accounts.value.length },
  { label: t('admin.socialAccountWorkbench.stats.stored'), value: accounts.value.filter(account => account.taskStatus === 'stored').length },
  { label: t('admin.socialAccountWorkbench.stats.failed'), value: accounts.value.filter(account => isFailureTask(account.taskStatus)).length },
])

const visibleIds = computed(() => filteredAccounts.value.map(account => account.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)
const accountDialogTitle = computed(() => selectedAccountId.value ? t('admin.socialAccountWorkbench.detailTitle') : t('admin.socialAccountWorkbench.actions.manualImport'))
const canSubmitAccount = computed(() => accountForm.name.trim() !== '' && accountForm.platform.trim() !== '')
const selectionRequiredActions: ExecutionAction[] = ['login_check', 'message', 'follow', 'post', 'like']
const canStartExecution = computed(() => !selectionRequiredActions.includes(executionAction.value) || selectedIds.value.length > 0)
const executionNeedsTarget = computed(() => ['message', 'follow', 'like'].includes(executionAction.value))
const executionNeedsContent = computed(() => ['message', 'post'].includes(executionAction.value))
const canSubmitExecutionParams = computed(() => {
  if (executionNeedsTarget.value && executionTarget.value.trim() === '') return false
  if (executionNeedsContent.value && executionContent.value.trim() === '') return false
  return true
})
const managementTabCount = computed(() => accounts.value.length)
const poolTabCount = computed(() => accounts.value.filter(account => !account.assignedUserId).length)
const activeExecutionRun = computed(() => executionRuns.value.find(run => run.id === activeExecutionRunId.value) ?? null)
const activeExecutionCount = computed(() => executionRuns.value.filter(run => run.status === 'running').length)
const recentExecutionRuns = computed(() => executionRuns.value.slice(0, 6))
const accountTabActiveClass = 'flex min-h-[40px] items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-semibold text-white shadow-sm'
const accountTabInactiveClass = 'flex min-h-[40px] items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700'
const uploadPreview = computed(() => {
  const file = pendingImportFile.value
  return [
    { label: t('admin.socialAccountWorkbench.uploadPreview.file'), value: selectedImportFileName.value || t('admin.socialAccountWorkbench.uploadPreview.noFile') },
    { label: t('admin.socialAccountWorkbench.uploadPreview.size'), value: file ? formatFileSize(file.size) : '-' },
    { label: t('admin.socialAccountWorkbench.uploadPreview.type'), value: file?.type || '-' },
    { label: t('admin.socialAccountWorkbench.uploadPreview.ready'), value: file ? t('common.yes') : t('common.no') },
  ]
})
const detailItems = computed(() => {
  if (!selectedAccount.value) return []
  return [
    { label: t('admin.socialAccountWorkbench.columns.id'), value: selectedAccount.value.id },
    { label: t('admin.socialAccountWorkbench.columns.account'), value: selectedAccount.value.account },
    { label: t('admin.socialAccountWorkbench.columns.platform'), value: selectedAccount.value.platform },
    { label: t('admin.socialAccountWorkbench.columns.password'), value: selectedAccount.value.password },
    { label: t('admin.socialAccountWorkbench.columns.phone'), value: selectedAccount.value.phone },
    { label: t('admin.socialAccountWorkbench.columns.email'), value: selectedAccount.value.email },
    { label: t('admin.socialAccountWorkbench.columns.emailPassword'), value: selectedAccount.value.emailPassword },
    { label: t('admin.socialAccountWorkbench.columns.source'), value: sourceLabel(selectedAccount.value.source) },
    { label: t('admin.socialAccountWorkbench.columns.createdAt'), value: selectedAccount.value.createdAt },
  ]
})

async function loadAccounts() {
  try {
    const result = await adminAPI.socialAccounts.list({ page: 1, page_size: 200 })
    accounts.value = (result.items ?? []).map(mapApiAccount)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.socialAccountWorkbench.failedToLoad'))
  }
}

function mapApiAccount(account: SocialAccount): AccountRow {
  return {
    id: account.id,
    account: account.name,
    platform: account.platform,
    accountId: account.account_id ?? '',
    password: account.password ?? '',
    phone: account.phone ?? '',
    email: account.email ?? '',
    emailPassword: account.email_password ?? '',
    accountStatus: toAccountStatus(account.account_status),
    taskStatus: account.task_status,
    taskMessage: account.task_message ?? '',
    source: toSource(account.source),
    assignedUserId: account.assigned_user_id ?? null,
    remark: account.remark ?? '',
    createdAt: new Date(account.created_at).toLocaleString(),
  }
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

function resetAccountForm() {
  selectedAccountId.value = null
  accountForm.name = ''
  accountForm.platform = 'x_twitter'
  accountForm.accountId = ''
  accountForm.password = ''
  accountForm.phone = ''
  accountForm.email = ''
  accountForm.emailPassword = ''
  accountForm.accountStatus = 'pending_check'
  accountForm.remark = ''
}

function openCreateDialog() {
  resetAccountForm()
  accountDialogOpen.value = true
}

function openDetailDialog(row: AccountRow) {
  selectedAccount.value = row
  detailDialogOpen.value = true
}

function openEditDialog(row: AccountRow) {
  detailDialogOpen.value = false
  selectedAccountId.value = row.id
  accountForm.name = row.account
  accountForm.platform = row.platform
  accountForm.accountId = row.accountId
  accountForm.password = row.password
  accountForm.phone = row.phone
  accountForm.email = row.email
  accountForm.emailPassword = row.emailPassword
  accountForm.accountStatus = row.accountStatus
  accountForm.remark = row.remark
  accountDialogOpen.value = true
}

async function submitAccountDialog() {
  if (!canSubmitAccount.value) return
  try {
    if (selectedAccountId.value) {
      await adminAPI.socialAccounts.update(selectedAccountId.value, {
        name: accountForm.name.trim(),
        account_id: accountForm.accountId,
        password: accountForm.password,
        phone: accountForm.phone,
        email: accountForm.email,
        email_password: accountForm.emailPassword,
        account_status: accountForm.accountStatus,
        remark: accountForm.remark,
      })
      appStore.showSuccess(t('admin.socialAccountWorkbench.saved'))
    } else {
      await adminAPI.socialAccounts.create({
        name: accountForm.name.trim(),
        platform: accountForm.platform.trim(),
        account_id: accountForm.accountId || undefined,
        password: accountForm.password || undefined,
        phone: accountForm.phone || undefined,
        email: accountForm.email || undefined,
        email_password: accountForm.emailPassword || undefined,
        remark: accountForm.remark || undefined,
        source: 'manual_import',
      })
      appStore.showSuccess(t('admin.socialAccountWorkbench.created'))
    }
    accountDialogOpen.value = false
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function openRegisterDialog() {
  registerForm.name = ''
  registerForm.platform = 'x_twitter'
  registerForm.password = ''
  registerForm.remark = ''
  registerDialogOpen.value = true
}

function openImportDialog() {
  pendingImportFile.value = null
  selectedImportFileName.value = ''
  importDialogOpen.value = true
}

async function submitRegister() {
  try {
    await adminAPI.socialAccounts.register({
      name: registerForm.name.trim(),
      platform: registerForm.platform.trim(),
      password: registerForm.password || undefined,
      remark: registerForm.remark || undefined,
      source: 'registered',
    })
    registerDialogOpen.value = false
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.socialAccountWorkbench.registerNotConfigured'))
  }
}

function triggerImport() {
  importFileInput.value?.click()
}

function handleImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  pendingImportFile.value = file
  selectedImportFileName.value = file.name
  input.value = ''
}

async function submitImportFile() {
  if (!pendingImportFile.value) return
  try {
    const result = await adminAPI.socialAccounts.importAccounts(pendingImportFile.value)
    appStore.showSuccess(t('admin.socialAccountWorkbench.imported', { count: result.created }))
    importDialogOpen.value = false
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    pendingImportFile.value = null
    selectedImportFileName.value = ''
  }
}

async function exportAccounts() {
  try {
    const blob = await adminAPI.socialAccounts.exportAccounts()
    downloadBlob(blob, 'social_accounts.csv')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function startExecution() {
  if (!canStartExecution.value) return
  if (executionAction.value === 'register') {
    openRegisterDialog()
    return
  }
  if (executionAction.value === 'manualImport') {
    openCreateDialog()
    return
  }
  if (executionAction.value === 'fileUpload') {
    openImportDialog()
    return
  }
  executionTarget.value = ''
  executionContent.value = ''
  if (executionNeedsTarget.value || executionNeedsContent.value) {
    executionParamDialogOpen.value = true
    return
  }
  void createExecutionRun(executionAction.value, selectedIds.value.length)
}

function submitExecutionFromDialog() {
  if (!canSubmitExecutionParams.value) return
  executionParamDialogOpen.value = false
  void createExecutionRun(executionAction.value, selectedIds.value.length)
}

async function createExecutionRun(action: ExecutionAction, targetCount: number) {
  const run: ExecutionRun = {
    id: nextExecutionRunId++,
    action,
    title: t('admin.socialAccountWorkbench.executionLog.runTitle', { action: executionActionLabel(action) }),
    targetCount,
    startedAt: formatClock(new Date()),
    status: 'pending',
    progress: 0,
    lines: [],
    logStatuses: {},
  }
  executionRuns.value = [run, ...executionRuns.value]
  activeExecutionRunId.value = run.id
  executionLogDialogOpen.value = true
  appStore.showInfo(t('admin.socialAccountWorkbench.executionLog.startedToast', { action: executionActionLabel(action), count: targetCount }))
  appendExecutionLine(run.id, 'info', t('admin.socialAccountWorkbench.executionLog.lines.start', { action: executionActionLabel(action), count: targetCount }))
  try {
    const payload = {
      account_ids: [...selectedIds.value],
      action,
      target: executionTarget.value.trim() || undefined,
      content: executionContent.value.trim() || undefined,
      client_request_id: `admin-${Date.now()}-${run.id}`,
    }
    appendExecutionLine(run.id, 'info', t('admin.socialAccountWorkbench.executionLog.lines.estimate'))
    const estimate = await adminAPI.socialAccounts.estimateTask(payload)
    const estimatedTotal = Object.values(estimate.estimates ?? {}).reduce((sum, item) => sum + item.estimated_total, 0)
    appendExecutionLine(run.id, 'info', t('admin.socialAccountWorkbench.executionLog.lines.estimated', { amount: estimatedTotal.toFixed(2) }))

    run.status = 'running'
    run.progress = 20
    const result = await adminAPI.socialAccounts.submitTask(payload)
    appendExecutionLine(run.id, 'info', t('admin.socialAccountWorkbench.executionLog.lines.submitted', {
      submitted: result.submitted,
      enqueued: result.enqueued,
      failed: result.failed_closed ?? 0,
    }))
    updateRunFromLogs(run, result.logs)
    pollExecutionLogs(run.id, result.logs.map(log => log.id))
  } catch (error: any) {
    run.status = 'failed'
    run.progress = 100
    appendExecutionLine(run.id, 'error', error?.message || t('common.error'))
    appStore.showError(error?.message || t('common.error'))
  }
}

function executionActionLabel(action: ExecutionAction): string {
  if (action === 'register') return t('admin.socialAccountWorkbench.actions.register')
  if (action === 'manualImport') return t('admin.socialAccountWorkbench.actions.manualImport')
  if (action === 'fileUpload') return t('admin.socialAccountWorkbench.actions.fileUpload')
  return t(`admin.socialAccountWorkbench.executionActions.${action}`)
}

function pollExecutionLogs(runId: number, logIds: number[]) {
  if (logIds.length === 0) return
  const timerId = window.setInterval(async () => {
    const run = executionRuns.value.find(item => item.id === runId)
    if (!run || run.status === 'success' || run.status === 'failed') {
      window.clearInterval(timerId)
      return
    }
    try {
      const page = await adminAPI.socialAccounts.listTaskLogs({ page: 1, page_size: 100 })
      const logs = page.items.filter(log => logIds.includes(log.id))
      updateRunFromLogs(run, logs)
    } catch (error: any) {
      appendExecutionLine(runId, 'warning', error?.message || t('common.error'))
    }
  }, 1500)
  executionTimers.push(timerId)
}

function updateRunFromLogs(run: ExecutionRun, logs: SocialTaskLog[]) {
  if (logs.length === 0) return
  const completed = logs.filter(log => log.status === 'success' || log.status === 'failed').length
  const failed = logs.filter(log => log.status === 'failed').length
  const success = logs.filter(log => log.status === 'success').length
  run.progress = Math.max(run.progress, Math.round((completed / logs.length) * 100))
  for (const log of logs) {
    const statusKey = `${log.status}:${log.result_message ?? ''}:${log.charged_amount}`
    if (run.logStatuses[log.id] === statusKey) continue
    run.logStatuses[log.id] = statusKey
    const message = log.result_message || t('admin.socialAccountWorkbench.executionLog.lines.statusLine', {
      id: log.social_account_id,
      status: log.status,
      charged: log.charged_amount.toFixed(2),
    })
    appendExecutionLine(run.id, log.status === 'failed' ? 'warning' : 'info', `#${log.social_account_id} ${log.action} ${log.status}: ${message}`)
  }
  if (completed === logs.length) {
    run.progress = 100
    run.status = failed > 0 ? 'failed' : 'success'
    appendExecutionLine(run.id, failed > 0 ? 'warning' : 'info', t('admin.socialAccountWorkbench.executionLog.lines.finished', { success, failed }))
    return
  }
  run.status = 'running'
}

function appendExecutionLine(runId: number, level: LogLineLevel, text: string) {
  const run = executionRuns.value.find(item => item.id === runId)
  if (!run) return
  run.lines.push({ id: nextExecutionLineId++, time: formatClock(new Date()), level, text })
  if (activeExecutionRunId.value === runId) scrollExecutionLogToBottom()
}

function openExecutionLog(runId: number) {
  activeExecutionRunId.value = runId
  executionLogDialogOpen.value = true
  scrollExecutionLogToBottom()
}

async function scrollExecutionLogToBottom() {
  await nextTick()
  if (executionTerminalRef.value) {
    executionTerminalRef.value.scrollTop = executionTerminalRef.value.scrollHeight
  }
}

function copyExecutionLog(run: ExecutionRun) {
  const content = run.lines.map(line => `[${line.time}] ${line.text}`).join('\n')
  copyToClipboard(content, t('admin.socialAccountWorkbench.executionLog.copied'))
}

function accountStatusLabel(status: string): string {
  return t(`admin.socialAccountWorkbench.accountStatus.${status}`)
}

function taskStatusLabel(status: string): string {
  return t(`admin.socialAccountWorkbench.taskStatus.${status}`)
}

function sourceLabel(source: string): string {
  return t(`admin.socialAccountWorkbench.sources.${source}`)
}

function accountStatusBadgeClass(status: string): string {
  if (status === 'available') return 'badge-success'
  if (status === 'pending_check') return 'badge-warning'
  if (status === 'limited') return 'badge-primary'
  return 'badge-danger'
}

function taskStatusBadgeClass(status: string): string {
  if (status === 'stored') return 'badge-success'
  if (status === 'pending' || status === 'registering' || status === 'importing' || status === 'parsing') return 'badge-warning'
  if (status === 'manual_review') return 'badge-primary'
  return 'badge-danger'
}

function taskMessagePanelClass(status: string): string {
  if (status === 'stored') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/60 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (isFailureTask(status)) return 'border-red-200 bg-red-50 text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300'
  return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300'
}

function executionStatusBadgeClass(status: ExecutionStatus): string {
  if (status === 'success') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  return 'bg-yellow-50 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-300'
}

function executionStatusTextClass(status: ExecutionStatus): string {
  if (status === 'success') return 'text-emerald-600 dark:text-emerald-400'
  if (status === 'failed') return 'text-red-600 dark:text-red-400'
  return 'text-yellow-600 dark:text-yellow-400'
}

function executionProgressClass(status: ExecutionStatus): string {
  if (status === 'success') return 'bg-emerald-500'
  if (status === 'failed') return 'bg-red-500'
  return 'bg-primary-500'
}

function logLineClass(level: LogLineLevel): string {
  if (level === 'warning') return 'text-yellow-400'
  if (level === 'error') return 'text-red-400'
  return 'text-gray-300'
}

function isFailureTask(status: string): boolean {
  return status === 'register_failed' || status === 'risk_rejected' || status === 'duplicate' || status === 'ip_unavailable'
}

function toAccountStatus(status: string): AccountStatus {
  if (status === 'pending_check' || status === 'available' || status === 'limited' || status === 'invalid' || status === 'not_stored') {
    return status
  }
  return 'not_stored'
}

function toSource(source: string): Source {
  if (source === 'registered' || source === 'manual_import' || source === 'file_upload') {
    return source
  }
  return 'manual_import'
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function formatFileSize(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function formatClock(date: Date): string {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
}

onUnmounted(() => {
  for (const timerId of executionTimers) {
    window.clearInterval(timerId)
  }
})
</script>
