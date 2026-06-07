<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('dashboard.eyebrow') }}</p>
          <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ t('dashboard.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('dashboard.description') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-secondary" :disabled="refreshing" @click="loadData">
            <Icon name="refresh" size="sm" :class="refreshing ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
          <router-link class="btn btn-primary" to="/usage">
            <Icon name="chart" size="sm" />
            <span>{{ t('dashboard.viewLogs') }}</span>
          </router-link>
        </div>
      </div>

      <div
        v-if="loading"
        class="flex min-h-[260px] items-center justify-center rounded-lg border border-dashed border-gray-200 bg-white text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ t('dashboard.loadingMine') }}
      </div>

      <template v-else>
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-6">
          <article v-for="card in metricCards" :key="card.label" class="card p-4">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</p>
                <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ card.value }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ card.meta }}</p>
              </div>
              <div :class="['flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg', card.iconBg]">
                <Icon :name="card.icon" size="sm" :class="card.iconClass" :stroke-width="2" />
              </div>
            </div>
          </article>
        </div>

        <div class="grid gap-6 xl:grid-cols-3">
          <section class="card p-5 xl:col-span-2">
            <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.trendTitle') }}</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.trendDescription') }}</p>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.granularityDay') }}</span>
            </div>

            <div class="mt-5 h-48">
              <div v-if="trendDots.length === 0" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('dashboard.trendEmpty') }}
              </div>
              <svg v-else viewBox="0 0 360 130" preserveAspectRatio="none" class="h-full w-full overflow-visible">
                <line x1="12" y1="118" x2="348" y2="118" class="stroke-gray-200 dark:stroke-dark-700" />
                <polyline
                  :points="trendPolyline"
                  fill="none"
                  class="stroke-primary-600 dark:stroke-primary-400"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <circle
                  v-for="point in trendDots"
                  :key="point.date"
                  :cx="point.x"
                  :cy="point.y"
                  r="3.5"
                  class="fill-white stroke-primary-600 dark:fill-dark-800 dark:stroke-primary-400"
                  stroke-width="2"
                />
              </svg>
            </div>

            <div class="mt-4 grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
              <div
                v-for="point in recentTrendRows"
                :key="point.date"
                class="rounded-lg border border-gray-100 px-3 py-2 text-sm dark:border-dark-700"
              >
                <div class="flex items-center justify-between gap-3">
                  <span class="text-gray-500 dark:text-gray-400">{{ formatTrendLabel(point.date) }}</span>
                  <span class="font-semibold text-gray-900 dark:text-white">{{ t('dashboard.trendCount', { count: formatNumber(point.requests || 0) }) }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('dashboard.chargedAmount', { amount: formatCurrency(point.actual_cost || point.cost || 0) }) }}
                </div>
              </div>
            </div>
          </section>

          <section class="card min-w-0 p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.platformDistribution') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.platformDistributionDescription') }}</p>
            <DistributionList class="mt-5" :rows="platformDistribution" :empty-text="t('dashboard.platformDistributionEmpty')" />
          </section>
        </div>

        <div class="grid gap-6 xl:grid-cols-3">
          <section class="card min-w-0 overflow-hidden xl:col-span-2">
            <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.recentUsage') }}</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.recentUsageDescription') }}</p>
              </div>
              <router-link class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" to="/usage">
                {{ t('dashboard.allRecords') }}
              </router-link>
            </div>
            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.operation') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.platform') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.account') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.status') }}</th>
                    <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.quantity') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.chargeStatus') }}</th>
                    <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.cost') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.result') }}</th>
                    <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                  <tr v-for="row in recentTasks" :key="row.id">
                    <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ actionLabel(row.operation) }}</td>
                    <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ platformLabel(row.platform) }}</td>
                    <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ row.account_name || '-' }}</td>
                    <td class="px-5 py-3 text-sm">
                      <span :class="['badge', statusClass(row.status)]">{{ statusLabel(row.status) }}</span>
                    </td>
                    <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatNumber(row.quantity) }}</td>
                    <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ chargeStatusLabel(row.charge_status) }}</td>
                    <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatCurrency(row.cost) }}</td>
                    <td class="max-w-xs px-5 py-3 text-sm text-gray-600 dark:text-gray-300">
                      <div v-if="resultSummary(row)" class="line-clamp-2 font-medium text-gray-900 dark:text-white">{{ resultSummary(row) }}</div>
                      <div class="line-clamp-2" :class="resultSummary(row) ? 'mt-1' : ''">{{ resultMessage(row) }}</div>
                    </td>
                    <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.completed_at || row.created_at) }}</td>
                  </tr>
                  <tr v-if="recentTasks.length === 0">
                    <td class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400" colspan="9">
                      {{ t('dashboard.noUsageRecords') }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="card min-w-0 p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickEntries') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.quickEntriesDescription') }}</p>
            <div class="mt-5 space-y-3">
              <router-link
                v-for="link in quickLinks"
                :key="link.to"
                :to="link.to"
                class="flex items-center gap-3 rounded-lg border border-gray-200 px-3 py-3 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:hover:border-primary-800 dark:hover:bg-primary-900/20"
              >
                <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  <Icon :name="link.icon" size="sm" :stroke-width="2" />
                </div>
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ link.label }}</p>
                  <p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ link.description }}</p>
                </div>
              </router-link>
            </div>
          </section>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import accountWorkbenchAPI from '@/api/accountWorkbench'
import type { UserSocialAccount } from '@/api/accountWorkbench'
import { usageAPI } from '@/api/usage'
import type { DashboardTrendPoint, PlatformDashboardStats, UsageLog, UserDashboardStats } from '@/api/usage'
import { useAppStore } from '@/stores/app'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'
import { formatWorkbenchTaskSummaryMeta } from '@/utils/workbenchTaskSummary'

type IconName = 'database' | 'checkCircle' | 'clock' | 'chart' | 'dollar' | 'shield' | 'creditCard' | 'user'

interface DistributionRow {
  label: string
  value: number
  percent: number
}

const DistributionList = defineComponent({
  name: 'DistributionList',
  props: {
    rows: {
      type: Array as () => DistributionRow[],
      required: true,
    },
    emptyText: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    return () => {
      if (props.rows.length === 0) {
        return h('p', { class: 'py-8 text-center text-sm text-gray-500 dark:text-gray-400' }, props.emptyText)
      }

      return h('div', { class: 'space-y-3' }, props.rows.map(row => h('div', { key: row.label }, [
        h('div', { class: 'mb-1 flex items-center justify-between gap-3 text-sm' }, [
          h('span', { class: 'truncate font-medium text-gray-700 dark:text-gray-200' }, row.label),
          h('span', { class: 'text-gray-500 dark:text-gray-400' }, `${row.value} (${row.percent}%)`),
        ]),
        h('div', { class: 'h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700' }, [
          h('div', {
            class: 'h-full rounded-full bg-primary-500',
            style: { width: `${row.percent}%` },
          }),
        ]),
      ])))
    }
  },
})

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const refreshing = ref(false)
const stats = ref<UserDashboardStats>({})
const accountTotal = ref(0)
const accounts = ref<UserSocialAccount[]>([])
const recentTasks = ref<UsageLog[]>([])
const trend = ref<DashboardTrendPoint[]>([])

const executableAccounts = computed(() => accounts.value.filter(account => {
  const accountStatus = String(account.account_status || '').toLowerCase()
  const taskStatus = String(account.task_status || '').toLowerCase()
  return accountStatus === 'available' && !['running', 'locked', 'disabled'].includes(taskStatus)
}).length)

const healthStats = computed(() => {
  const total = recentTasks.value.length
  const success = recentTasks.value.filter(row => row.status === 'success').length
  const failed = recentTasks.value.filter(row => row.status === 'failed').length
  return {
    total,
    success,
    failed,
    pending: Math.max(total - success - failed, 0),
  }
})

const successRate = computed(() => {
  if (!healthStats.value.total) return 0
  return Math.round((healthStats.value.success / healthStats.value.total) * 100)
})

const metricCards = computed<Array<{
  label: string
  value: string
  meta: string
  icon: IconName
  iconBg: string
  iconClass: string
}>>(() => [
  {
    label: t('dashboard.boundAccounts'),
    value: formatNumber(accountTotal.value),
    meta: accounts.value.length < accountTotal.value
      ? t('dashboard.sampledAccounts', { count: formatNumber(accounts.value.length) })
      : t('dashboard.coveredPlatforms', { count: formatNumber(platformDistribution.value.length) }),
    icon: 'database',
    iconBg: 'bg-blue-50 dark:bg-blue-900/20',
    iconClass: 'text-blue-600 dark:text-blue-300',
  },
  {
    label: t('dashboard.executableAccounts'),
    value: formatNumber(executableAccounts.value),
    meta: t('dashboard.totalPoolAccounts'),
    icon: 'checkCircle',
    iconBg: 'bg-green-50 dark:bg-green-900/20',
    iconClass: 'text-green-600 dark:text-green-300',
  },
  {
    label: t('dashboard.todayRequests'),
    value: formatNumber(stats.value.today_requests || 0),
    meta: t('dashboard.chargedAmount', { amount: formatCurrency(stats.value.today_actual_cost || 0) }),
    icon: 'clock',
    iconBg: 'bg-amber-50 dark:bg-amber-900/20',
    iconClass: 'text-amber-600 dark:text-amber-300',
  },
  {
    label: t('dashboard.totalRequests'),
    value: formatNumber(stats.value.total_requests || 0),
    meta: t('dashboard.recentRpm', { count: formatNumber(stats.value.rpm || 0) }),
    icon: 'chart',
    iconBg: 'bg-indigo-50 dark:bg-indigo-900/20',
    iconClass: 'text-indigo-600 dark:text-indigo-300',
  },
  {
    label: t('dashboard.totalCharged'),
    value: formatCurrency(stats.value.total_actual_cost || 0),
    meta: t('dashboard.successOnlyBilling'),
    icon: 'dollar',
    iconBg: 'bg-emerald-50 dark:bg-emerald-900/20',
    iconClass: 'text-emerald-600 dark:text-emerald-300',
  },
  {
    label: t('dashboard.recentSuccessRate'),
    value: `${successRate.value}%`,
    meta: t('dashboard.successFailureMeta', {
      success: formatNumber(healthStats.value.success),
      failed: formatNumber(healthStats.value.failed),
    }),
    icon: 'shield',
    iconBg: 'bg-rose-50 dark:bg-rose-900/20',
    iconClass: 'text-rose-600 dark:text-rose-300',
  },
])

const trendDots = computed(() => {
  const width = 360
  const height = 130
  const padding = 12
  const points = trend.value
  const maxValue = Math.max(1, ...points.map(point => point.requests || 0))

  return points.map((point, index) => {
    const x = points.length > 1
      ? padding + ((width - padding * 2) / (points.length - 1)) * index
      : width / 2
    const y = height - padding - ((point.requests || 0) / maxValue) * (height - padding * 2)
    return { ...point, x, y }
  })
})

const trendPolyline = computed(() => trendDots.value.map(point => `${point.x},${point.y}`).join(' '))
const recentTrendRows = computed(() => trend.value.slice(-6).reverse())

const platformDistribution = computed(() => {
  const rowsFromStats = buildPlatformRows(stats.value.by_platform ?? [])
  if (rowsFromStats.length > 0) return rowsFromStats

  const byPlatform = new Map<string, number>()
  accounts.value.forEach(account => {
    const key = platformLabel(account.platform)
    byPlatform.set(key, (byPlatform.get(key) || 0) + 1)
  })
  return mapDistribution(byPlatform)
})

const quickLinks = computed<Array<{ to: string; label: string; description: string; icon: IconName }>>(() => [
  { to: '/usage', label: t('dashboard.quickLinks.taskLogs.label'), description: t('dashboard.quickLinks.taskLogs.description'), icon: 'chart' },
  { to: '/purchase', label: t('dashboard.quickLinks.purchase.label'), description: t('dashboard.quickLinks.purchase.description'), icon: 'creditCard' },
  { to: '/subscriptions', label: t('dashboard.quickLinks.subscriptions.label'), description: t('dashboard.quickLinks.subscriptions.description'), icon: 'shield' },
  { to: '/profile', label: t('dashboard.quickLinks.profile.label'), description: t('dashboard.quickLinks.profile.description'), icon: 'user' },
])

async function loadData() {
  refreshing.value = true
  if (recentTasks.value.length === 0 && trend.value.length === 0) {
    loading.value = true
  }

  const [statsResult, trendResult, accountsResult, tasksResult] = await Promise.allSettled([
    usageAPI.getDashboardStats(),
    usageAPI.getDashboardTrend({ granularity: 'day' }),
    accountWorkbenchAPI.listMyAccounts({ page: 1, page_size: 100 }),
    usageAPI.list({ page: 1, page_size: 8 }),
  ])

  if (statsResult.status === 'fulfilled') stats.value = statsResult.value || {}
  if (trendResult.status === 'fulfilled') trend.value = trendResult.value
  if (accountsResult.status === 'fulfilled') {
    accounts.value = accountsResult.value.items ?? []
    accountTotal.value = accountsResult.value.total ?? accounts.value.length
  }
  if (tasksResult.status === 'fulfilled') recentTasks.value = tasksResult.value.items ?? []

  const failedCount = [statsResult, trendResult, accountsResult, tasksResult].filter(result => result.status === 'rejected').length
  if (failedCount >= 3) {
    appStore.showError(t('dashboard.loadIncomplete'))
  }

  loading.value = false
  refreshing.value = false
}

function buildPlatformRows(items: PlatformDashboardStats[]): DistributionRow[] {
  const source = new Map<string, number>()
  items.forEach(item => {
    const value = item.total_requests || 0
    if (value > 0) {
      source.set(platformLabel(item.platform), value)
    }
  })
  return mapDistribution(source)
}

function mapDistribution(source: Map<string, number>): DistributionRow[] {
  const total = Array.from(source.values()).reduce((sum, value) => sum + value, 0)
  if (!total) return []

  return Array.from(source.entries())
    .map(([label, value]) => ({
      label,
      value,
      percent: Math.round((value / total) * 100),
    }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 6)
}

function platformLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return t('dashboard.unknownPlatform')
  const key = `usage.platforms.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized.toUpperCase()
}

function actionLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return t('common.unknown')
  const key = `usage.actions.${normalized}`
  const translated = t(key)
  return translated === key ? value || t('common.unknown') : translated
}

function statusLabel(value?: string | null) {
  const normalized = String(value || '').toLowerCase()
  if (!normalized) return '-'
  const key = `usage.statuses.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function statusClass(status?: string | null) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-error'
  return 'badge-warning'
}

function chargeStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.chargeStatuses.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function resultMessage(row: UsageLog) {
  return formatSocialTaskResultMessage(row, t)
}

function resultSummary(row: UsageLog) {
  const summary = formatWorkbenchTaskSummaryMeta({
    action: row.operation,
    target: row.target,
    content: row.content,
    payload: row.payload,
    template_snapshot: row.template_snapshot,
  }, t, {
    actionKeyPrefix: 'usage.actions',
    summaryKeyPrefix: 'usage',
  })

  return summary.endsWith(`· ${t('usage.taskSummaryNoDetails')}`) ? '' : summary
}

function formatNumber(value?: number) {
  return Number(value || 0).toLocaleString()
}

function formatCurrency(value?: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function formatTrendLabel(value: string) {
  if (!value) return '-'
  return value.replace('T', ' ').slice(0, 16)
}

onMounted(() => {
  void loadData()
})
</script>
