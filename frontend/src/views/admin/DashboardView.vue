<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('admin.dashboard.opsEyebrow') }}</p>
          <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.dashboard.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.dashboard.opsDescription') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-secondary" :disabled="refreshing" @click="loadData">
            <Icon name="refresh" size="sm" :class="refreshing ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
          <router-link class="btn btn-primary" to="/accounts">
            <Icon name="play" size="sm" />
            <span>{{ t('admin.dashboard.submitTask') }}</span>
          </router-link>
        </div>
      </div>

      <div
        v-if="loading"
        class="flex min-h-[280px] items-center justify-center rounded-lg border border-dashed border-gray-200 bg-white text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ t('admin.dashboard.loading') }}
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
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.taskTrendTitle') }}</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.taskTrendDescription') }}</p>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.granularityDay') }}</span>
            </div>

            <div class="mt-5 h-48">
              <div v-if="trendDots.length === 0" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.noTrendData') }}
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
                  <span class="font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.taskCount', { count: formatNumber(point.operations || 0) }) }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.successfulCharges', { amount: formatCurrency(point.charged || 0) }) }}
                </div>
              </div>
            </div>
          </section>

          <section class="card p-5">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.resourceHealth') }}</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.resourceHealthDescription') }}</p>
              </div>
              <span :class="['rounded-full px-2.5 py-1 text-xs font-medium', successRateClass]">
                {{ accountAvailabilityRate }}%
              </span>
            </div>

            <div class="mt-5 space-y-4">
              <div>
                <div class="mb-2 flex items-center justify-between text-sm">
                  <span class="text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.accountAvailability') }}</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ availableAccounts }} / {{ poolTotal }}</span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full bg-green-500" :style="{ width: `${accountAvailabilityRate}%` }" />
                </div>
              </div>

              <div class="grid grid-cols-3 gap-3">
                <div class="rounded-lg bg-green-50 p-3 dark:bg-green-900/20">
                  <p class="text-xs text-green-700 dark:text-green-300">{{ t('admin.dashboard.availableAccounts') }}</p>
                  <p class="mt-1 text-lg font-semibold text-green-700 dark:text-green-300">{{ availableAccounts }}</p>
                </div>
                <div class="rounded-lg bg-red-50 p-3 dark:bg-red-900/20">
                  <p class="text-xs text-red-700 dark:text-red-300">{{ t('admin.dashboard.issueAccounts') }}</p>
                  <p class="mt-1 text-lg font-semibold text-red-700 dark:text-red-300">{{ accountIssueCount }}</p>
                </div>
                <div class="rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
                  <p class="text-xs text-amber-700 dark:text-amber-300">{{ t('admin.dashboard.readyAccounts') }}</p>
                  <p class="mt-1 text-lg font-semibold text-amber-700 dark:text-amber-300">{{ storedAccounts }}</p>
                </div>
              </div>
            </div>
          </section>
        </div>

        <div class="grid gap-6 xl:grid-cols-3">
          <section class="card p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.platformDistribution') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.platformDistributionDescription') }}</p>
            <DistributionList class="mt-5" :rows="platformDistribution" :empty-text="t('admin.dashboard.platformDistributionEmpty')" />
          </section>

          <section class="card p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.accountStatusDistribution') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.accountStatusDistributionDescription') }}</p>
            <DistributionList class="mt-5" :rows="accountStatusDistribution" :empty-text="t('admin.dashboard.accountStatusDistributionEmpty')" />
          </section>

          <section class="card p-5">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.userSpendingRanking') }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.userSpendingRankingDescription') }}</p>
              </div>
            </div>

            <div class="mt-5 space-y-3">
              <div v-for="item in rankingRows" :key="item.user_id" class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ userIdentity(item) }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.taskCountShort', { count: formatNumber(item.operations || 0) }) }}</p>
                </div>
                <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatCurrency(item.charged || 0) }}</span>
              </div>
              <p v-if="rankingRows.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.noSpendingRanking') }}
              </p>
            </div>
          </section>
        </div>

        <section class="card overflow-hidden">
          <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.recentExecutionSummary') }}</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.recentExecutionSummaryDescription') }}</p>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.date') }}</th>
                  <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.taskCountColumn') }}</th>
                  <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.successfulChargeColumn') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-for="row in recentActivityRows" :key="row.date">
                  <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ formatTrendLabel(row.date) }}</td>
                  <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatNumber(row.operations || 0) }}</td>
                  <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatCurrency(row.charged || 0) }}</td>
                </tr>
                <tr v-if="recentActivityRows.length === 0">
                  <td class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400" colspan="3">
                    {{ t('admin.dashboard.noExecutionSummary') }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section>
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.quickEntries') }}</h2>
          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <router-link
              v-for="link in quickLinks"
              :key="link.to"
              :to="link.to"
              class="flex items-center gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-800 dark:hover:bg-primary-900/20"
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
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type {
  AdminDashboardStats,
  AdminDashboardTrendPoint,
  AdminUserSpendingRankingItem
} from '@/api/admin/dashboard'
import type { SocialAccount, SocialAccountStats } from '@/api/accountWorkbench'
import { useAppStore } from '@/stores/app'

type IconName =
  | 'database'
  | 'checkCircle'
  | 'clock'
  | 'chart'
  | 'dollar'
  | 'users'
  | 'globe'
  | 'play'
  | 'server'
  | 'shield'

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
const stats = ref<AdminDashboardStats>({})
const accountStats = ref<SocialAccountStats>({ total: 0, stored: 0, available: 0 })
const accounts = ref<SocialAccount[]>([])
const trend = ref<AdminDashboardTrendPoint[]>([])
const rankingRows = ref<AdminUserSpendingRankingItem[]>([])

const poolTotal = computed(() => accountStats.value.total || stats.value.total_accounts || 0)
const availableAccounts = computed(() => accountStats.value.available || stats.value.normal_accounts || 0)
const storedAccounts = computed(() => accountStats.value.stored || 0)
const assignedAccounts = computed(() => Math.max(poolTotal.value - availableAccounts.value, 0))
const accountIssueCount = computed(() => {
  const explicit = Number(stats.value.error_accounts || 0) + Number(stats.value.ratelimit_accounts || 0) + Number(stats.value.overload_accounts || 0)
  return explicit > 0 ? explicit : Math.max(poolTotal.value - availableAccounts.value, 0)
})
const accountAvailabilityRate = computed(() => {
  if (!poolTotal.value) return 0
  return Math.round((availableAccounts.value / poolTotal.value) * 100)
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
    label: t('admin.dashboard.totalAccountPool'),
    value: formatNumber(poolTotal.value),
    meta: t('admin.dashboard.assignedAccountsMeta', { count: formatNumber(assignedAccounts.value) }),
    icon: 'database',
    iconBg: 'bg-blue-50 dark:bg-blue-900/20',
    iconClass: 'text-blue-600 dark:text-blue-300',
  },
  {
    label: t('admin.dashboard.allocatableAccounts'),
    value: formatNumber(availableAccounts.value),
    meta: t('admin.dashboard.storedAccountsMeta', { count: formatNumber(accountStats.value.stored || 0) }),
    icon: 'checkCircle',
    iconBg: 'bg-green-50 dark:bg-green-900/20',
    iconClass: 'text-green-600 dark:text-green-300',
  },
  {
    label: t('admin.dashboard.todayTasks'),
    value: formatNumber(stats.value.today_operations || 0),
    meta: t('admin.dashboard.recentOperationsPerMinuteMeta', { count: formatNumber(stats.value.recent_operations_per_minute || 0) }),
    icon: 'clock',
    iconBg: 'bg-amber-50 dark:bg-amber-900/20',
    iconClass: 'text-amber-600 dark:text-amber-300',
  },
  {
    label: t('admin.dashboard.totalTasks'),
    value: formatNumber(stats.value.total_operations || 0),
    meta: t('admin.dashboard.executionRecordsMeta'),
    icon: 'chart',
    iconBg: 'bg-indigo-50 dark:bg-indigo-900/20',
    iconClass: 'text-indigo-600 dark:text-indigo-300',
  },
  {
    label: t('admin.dashboard.todaySuccessfulCharges'),
    value: formatCurrency(stats.value.today_charged || 0),
    meta: t('admin.dashboard.cumulativeChargesMeta', { amount: formatCurrency(stats.value.total_charged || 0) }),
    icon: 'dollar',
    iconBg: 'bg-emerald-50 dark:bg-emerald-900/20',
    iconClass: 'text-emerald-600 dark:text-emerald-300',
  },
  {
    label: t('admin.dashboard.activeUsers'),
    value: formatNumber(stats.value.active_users || 0),
    meta: t('admin.dashboard.userGrowthMeta', {
      total: formatNumber(stats.value.total_users || 0),
      today: formatNumber(stats.value.today_new_users || 0),
    }),
    icon: 'users',
    iconBg: 'bg-rose-50 dark:bg-rose-900/20',
    iconClass: 'text-rose-600 dark:text-rose-300',
  },
])

const successRateClass = computed(() => {
  if (accountAvailabilityRate.value >= 80) return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (accountAvailabilityRate.value >= 50) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
})

const trendDots = computed(() => {
  const width = 360
  const height = 130
  const padding = 12
  const points = trend.value
  const maxValue = Math.max(1, ...points.map(point => point.operations || 0))

  return points.map((point, index) => {
    const x = points.length > 1
      ? padding + ((width - padding * 2) / (points.length - 1)) * index
      : width / 2
    const y = height - padding - ((point.operations || 0) / maxValue) * (height - padding * 2)
    return { ...point, x, y }
  })
})

const trendPolyline = computed(() => trendDots.value.map(point => `${point.x},${point.y}`).join(' '))
const recentTrendRows = computed(() => trend.value.slice(-6).reverse())
const recentActivityRows = computed(() => trend.value.slice(-8).reverse())

const platformDistribution = computed(() => {
  const byPlatform = new Map<string, number>()

  accounts.value.forEach(account => {
    const key = platformLabel(account.platform)
    byPlatform.set(key, (byPlatform.get(key) || 0) + 1)
  })

  return mapDistribution(byPlatform)
})

const accountStatusDistribution = computed(() => {
  const byStatus = new Map<string, number>()
  accounts.value.forEach(account => {
    const key = accountStatusLabel(account.account_status)
    byStatus.set(key, (byStatus.get(key) || 0) + 1)
  })
  return mapDistribution(byStatus)
})

const quickLinks = computed<Array<{ to: string; label: string; description: string; icon: IconName }>>(() => [
  { to: '/accounts', label: t('admin.dashboard.quickLinks.accounts.label'), description: t('admin.dashboard.quickLinks.accounts.description'), icon: 'play' },
  { to: '/admin/total-accounts', label: t('admin.dashboard.quickLinks.totalAccounts.label'), description: t('admin.dashboard.quickLinks.totalAccounts.description'), icon: 'database' },
])

async function loadData() {
  refreshing.value = true
  if (trend.value.length === 0) {
    loading.value = true
  }

  const [
    dashboardResult,
    socialAccountResult,
    trendResult,
    rankingResult,
    accountListResult,
  ] = await Promise.allSettled([
    adminAPI.dashboard.getStats(),
    adminAPI.accountWorkbench.getStats(),
    adminAPI.dashboard.getUsageTrend({ granularity: 'day' }),
    adminAPI.dashboard.getUserSpendingRanking({ limit: 5 }),
    adminAPI.accountWorkbench.list({ page: 1, page_size: 100 }),
  ])

  if (dashboardResult.status === 'fulfilled') stats.value = dashboardResult.value || {}
  if (socialAccountResult.status === 'fulfilled') accountStats.value = socialAccountResult.value
  if (trendResult.status === 'fulfilled') trend.value = trendResult.value
  if (rankingResult.status === 'fulfilled') rankingRows.value = rankingResult.value.ranking ?? []
  if (accountListResult.status === 'fulfilled') accounts.value = accountListResult.value.items ?? []

  const failedCount = [
    dashboardResult,
    socialAccountResult,
    trendResult,
    rankingResult,
    accountListResult,
  ].filter(result => result.status === 'rejected').length
  if (failedCount >= 4) {
    appStore.showError(t('admin.dashboard.loadIncomplete'))
  }

  loading.value = false
  refreshing.value = false
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
  if (!normalized) return t('admin.dashboard.unknownPlatform')
  if (normalized === 'twitter' || normalized === 'x' || normalized === 'x_twitter') return 'Twitter / X'
  return normalized.toUpperCase()
}

function accountStatusLabel(value?: string | null) {
  const normalized = String(value || '').toLowerCase()
  if (!normalized) return value || '-'
  const key = `admin.dashboard.accountStatuses.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function userIdentity(item: AdminUserSpendingRankingItem) {
  return item.email || t('admin.dashboard.userFallback', { id: item.user_id })
}

function formatNumber(value?: number) {
  return Number(value || 0).toLocaleString()
}

function formatCurrency(value?: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function formatTrendLabel(value: string) {
  if (!value) return '-'
  return value.replace('T', ' ').slice(0, 16)
}

onMounted(() => {
  void loadData()
})
</script>
