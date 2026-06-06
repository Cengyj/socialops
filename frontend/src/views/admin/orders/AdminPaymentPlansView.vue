<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('payment.admin.searchPlans')" class="w-full sm:w-72" />
            <select v-model="saleFilter" class="input w-full sm:w-40">
              <option value="all">{{ t('common.all') }}</option>
              <option value="sale">{{ t('payment.admin.forSale') }}</option>
              <option value="hidden">{{ t('payment.admin.hidden') }}</option>
            </select>
          </div>
          <div class="flex items-center gap-2">
            <button class="btn btn-secondary" :disabled="plansLoading" @click="loadPlans">
              <Icon name="refresh" size="md" :class="plansLoading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-primary" @click="openPlanEdit(null)">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('payment.admin.createPlan') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div
          v-if="plansError"
          class="rounded-lg border border-red-200 bg-red-50 px-6 py-8 text-center dark:border-red-900/60 dark:bg-red-950/30"
        >
          <Icon name="exclamationTriangle" size="xl" class="mx-auto text-red-500 dark:text-red-400" />
          <h3 class="mt-3 text-base font-semibold text-red-900 dark:text-red-100">
            {{ t('payment.admin.failedToLoadPlans') }}
          </h3>
          <p class="mx-auto mt-2 max-w-xl text-sm text-red-700 dark:text-red-200">
            {{ plansError }}
          </p>
          <button class="btn btn-secondary mt-4" :disabled="plansLoading" @click="loadPlans">
            <Icon name="refresh" size="sm" :class="plansLoading ? 'animate-spin' : ''" class="mr-2" />
            {{ t('common.retry') }}
          </button>
        </div>

        <DataTable v-else :columns="planColumns" :data="filteredPlans" :loading="plansLoading" row-key="id">
          <template #empty>
            <div class="flex flex-col items-center px-4 py-3">
              <Icon name="gift" size="xl" class="mb-4 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {{ t('payment.admin.noPlans') }}
              </p>
              <p class="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
                {{ plans.length === 0 ? t('payment.admin.noPlansHint') : t('payment.admin.noPlansSearchHint') }}
              </p>
              <button
                v-if="plans.length === 0"
                type="button"
                class="btn btn-primary mt-4"
                @click="openPlanEdit(null)"
              >
                <Icon name="plus" size="sm" class="mr-2" />
                {{ t('payment.admin.createPlan') }}
              </button>
            </div>
          </template>
          <template #cell-name="{ value, row }">
            <SubscriptionPackageBadge
              :name="String(value || row.name)"
              :platform="row.platform || row.group_platform || 'social'"
              :description="row.description || ''"
              compact
            />
          </template>
          <template #cell-platform="{ row }">
            <span
              :class="[
                'inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium',
                platformBadgeClass(planPlatform(row))
              ]"
            >
              <SubscriptionPlatformLogo :platform="planPlatform(row)" compact />
              <span>{{ planPlatformLabel(row) }}</span>
            </span>
          </template>
          <template #cell-price="{ value, row }">
            <div class="text-sm">
              <span class="font-medium text-gray-900 dark:text-white">${{ Number(value ?? 0).toFixed(2) }}</span>
              <span v-if="row.original_price" class="ml-1 text-xs text-gray-400 line-through">${{ row.original_price.toFixed(2) }}</span>
            </div>
          </template>
          <template #cell-quota="{ row }">
            <div class="space-y-1">
              <div class="font-medium text-gray-900 dark:text-white">{{ formatQuotaAmount(getPlanQuotaAmount(row)) }}</div>
              <div v-if="formatPlanGuardrails(row)" class="text-xs text-gray-500 dark:text-gray-400">
                {{ formatPlanGuardrails(row) }}
              </div>
            </div>
          </template>
          <template #cell-validity="{ row }">
            <span>{{ formatPlanValidity(row) }}</span>
          </template>
          <template #cell-for_sale="{ row }">
            <button type="button" :class="['badge', row.for_sale ? 'badge-success' : 'badge-secondary']" @click="toggleForSale(row)">
              {{ row.for_sale ? t('payment.admin.forSale') : t('payment.admin.hidden') }}
            </button>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="openPlanEdit(row)">{{ t('common.edit') }}</button>
              <button class="btn btn-danger px-2 py-1 text-xs" @click="deletePlan(row)">{{ t('common.delete') }}</button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <PlanEditDialog
      :show="showPlanDialog"
      :plan="editingPlan"
      @close="showPlanDialog = false"
      @saved="loadPlans"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { SubscriptionPlan } from '@/types/payment'
import PlanEditDialog from './PlanEditDialog.vue'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import { getPlanQuotaAmount } from '@/utils/subscriptionQuotaPlans'
import SubscriptionPackageBadge from '@/components/payment/SubscriptionPackageBadge.vue'
import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'
import { getPlatformColor } from '@/utils/platformColors'
import {
  formatSubscriptionPlanGuardrails,
  formatSubscriptionPlanValidity,
  formatSubscriptionQuotaAmount,
  getSubscriptionPlatformLabel,
} from '@/utils/subscriptionPlanDisplay'

const { t } = useI18n()
const appStore = useAppStore()
const plansLoading = ref(false)
const plansError = ref('')
const plans = ref<SubscriptionPlan[]>([])
const searchQuery = ref('')
const saleFilter = ref<'all' | 'sale' | 'hidden'>('all')
const showPlanDialog = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)

const planColumns = computed<Column[]>(() => [
  { key: 'name', label: t('payment.admin.planName'), sortable: true },
  { key: 'quota', label: t('payment.admin.quotaAmount') },
  { key: 'validity', label: t('payment.admin.validity') },
  { key: 'price', label: t('payment.admin.price'), sortable: true },
  { key: 'platform', label: t('payment.admin.platform') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'actions', label: t('common.actions') },
])

const filteredPlans = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return plans.value.filter(plan => {
    const matchesKeyword = !keyword || [
      plan.name,
      plan.description,
      plan.platform ?? '',
      plan.group_platform ?? '',
    ].some(value => value.toLowerCase().includes(keyword))
    const matchesSale = saleFilter.value === 'all' || (saleFilter.value === 'sale' ? plan.for_sale : !plan.for_sale)
    return matchesKeyword && matchesSale
  })
})

async function loadPlans() {
  plansLoading.value = true
  plansError.value = ''
  try {
    const response = await adminPaymentAPI.getPlans()
    plans.value = (response.data || []).map((plan: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...plan,
      features: Array.isArray(plan.features)
        ? plan.features
        : String(plan.features || '').split('\n').map(item => item.trim()).filter(Boolean),
    }))
  } catch (error: unknown) {
    recordClientDiagnostic('admin.payment_plans.load', error)
    const message = extractSafeApiErrorMessage(error, t('payment.admin.failedToLoadPlans'))
    plansError.value = message
    appStore.showError(message)
  } finally {
    plansLoading.value = false
  }
}

function openPlanEdit(plan: SubscriptionPlan | null) {
  editingPlan.value = plan
  showPlanDialog.value = true
}

async function toggleForSale(plan: SubscriptionPlan) {
  try {
    await adminPaymentAPI.updatePlan(plan.id, { for_sale: !plan.for_sale })
    plan.for_sale = !plan.for_sale
  } catch (error: unknown) {
    recordClientDiagnostic('admin.payment_plans.toggle_sale', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  }
}

async function deletePlan(plan: SubscriptionPlan) {
  if (!window.confirm(t('payment.admin.deletePlanConfirm'))) return
  try {
    await adminPaymentAPI.deletePlan(plan.id)
    appStore.showSuccess(t('common.deleted'))
    await loadPlans()
  } catch (error: unknown) {
    recordClientDiagnostic('admin.payment_plans.delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  }
}

const formatQuotaAmount = (value: unknown) => formatSubscriptionQuotaAmount(value, t('payment.admin.unlimited'))

function formatPlanGuardrails(plan: SubscriptionPlan): string {
  return formatSubscriptionPlanGuardrails(plan, t)
}

function formatPlanValidity(plan: SubscriptionPlan): string {
  return formatSubscriptionPlanValidity(plan, t, navigator.language)
}

function platformLabel(platform: string): string {
  return getSubscriptionPlatformLabel(platform, t('payment.platformFallback'))
}

function planPlatform(plan: SubscriptionPlan): string {
  return String(plan.platform || plan.group_platform || 'social')
}

function planPlatformLabel(plan: SubscriptionPlan): string {
  return platformLabel(planPlatform(plan))
}

function platformBadgeClass(platform: string): string {
  const colors = getPlatformColor(platform)
  return `${colors.bg} ${colors.text} ${colors.border}`
}

onMounted(() => {
  void loadPlans()
})
</script>
