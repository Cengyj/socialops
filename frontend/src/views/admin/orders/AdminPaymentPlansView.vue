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
        <DataTable :columns="planColumns" :data="filteredPlans" :loading="plansLoading" row-key="id">
          <template #cell-name="{ value, row }">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ value }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.description || '-' }}</div>
            </div>
          </template>
          <template #cell-group_id="{ value, row }">
            <span class="badge badge-primary">{{ row.group_name || `#${value}` }}</span>
          </template>
          <template #cell-price="{ value, row }">
            <div class="text-sm">
              <span class="font-medium text-gray-900 dark:text-white">${{ Number(value ?? 0).toFixed(2) }}</span>
              <span v-if="row.original_price" class="ml-1 text-xs text-gray-400 line-through">${{ row.original_price.toFixed(2) }}</span>
            </div>
          </template>
          <template #cell-validity_days="{ value, row }">
            <span>{{ value }} {{ t(`payment.admin.${row.validity_unit || 'days'}`) }}</span>
          </template>
          <template #cell-features="{ value }">
            <span class="block max-w-sm truncate text-sm text-gray-600 dark:text-gray-300">{{ featureText(value) || '-' }}</span>
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

    <PlanEditDialog :show="showPlanDialog" :plan="editingPlan" @close="showPlanDialog = false" @saved="loadPlans" />
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

const { t } = useI18n()
const appStore = useAppStore()
const plansLoading = ref(false)
const plans = ref<SubscriptionPlan[]>([])
const searchQuery = ref('')
const saleFilter = ref<'all' | 'sale' | 'hidden'>('all')
const showPlanDialog = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)

const planColumns = computed<Column[]>(() => [
  { key: 'id', label: 'ID', sortable: true },
  { key: 'name', label: t('payment.admin.planName'), sortable: true },
  { key: 'group_id', label: t('payment.admin.group') },
  { key: 'price', label: t('payment.admin.price'), sortable: true },
  { key: 'validity_days', label: t('payment.admin.validityDays') },
  { key: 'features', label: t('payment.admin.features') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'sort_order', label: t('payment.admin.sortOrder'), sortable: true },
  { key: 'actions', label: t('common.actions') },
])

const filteredPlans = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return plans.value.filter(plan => {
    const matchesKeyword = !keyword || [plan.name, plan.description, String(plan.group_id), plan.group_name ?? ''].some(value => value.toLowerCase().includes(keyword))
    const matchesSale = saleFilter.value === 'all' || (saleFilter.value === 'sale' ? plan.for_sale : !plan.for_sale)
    return matchesKeyword && matchesSale
  })
})

async function loadPlans() {
  plansLoading.value = true
  try {
    const response = await adminPaymentAPI.getPlans()
    plans.value = (response.data || []).map((plan: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...plan,
      features: Array.isArray(plan.features)
        ? plan.features
        : String(plan.features || '').split('\n').map(item => item.trim()).filter(Boolean),
    }))
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
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
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

async function deletePlan(plan: SubscriptionPlan) {
  if (!window.confirm(t('payment.admin.deletePlanConfirm'))) return
  try {
    await adminPaymentAPI.deletePlan(plan.id)
    appStore.showSuccess(t('common.deleted'))
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  }
}

function featureText(value: unknown): string {
  if (Array.isArray(value)) return value.join(', ')
  return String(value || '')
}

onMounted(loadPlans)
</script>
