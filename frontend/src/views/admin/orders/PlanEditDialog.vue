<template>
  <BaseDialog :show="show" :title="dialogTitle" width="wide" @close="emit('close')">
    <div class="grid gap-3 sm:grid-cols-2">
      <input v-model="form.name" class="input" :placeholder="t('payment.admin.planName')" />
      <input v-model.number="form.group_id" class="input" type="number" min="1" :placeholder="t('payment.admin.group')" />
      <input v-model.number="form.price" class="input" type="number" min="0" step="0.01" :placeholder="t('payment.admin.price')" />
      <input v-model.number="form.original_price" class="input" type="number" min="0" step="0.01" :placeholder="t('payment.admin.originalPrice')" />
      <input v-model.number="form.validity_days" class="input" type="number" min="1" :placeholder="t('payment.admin.validityDays')" />
      <select v-model="form.validity_unit" class="input">
        <option value="days">{{ t('payment.admin.days') }}</option>
        <option value="months">{{ t('payment.admin.months') }}</option>
        <option value="years">{{ t('payment.admin.years') }}</option>
      </select>
      <input v-model="form.product_name" class="input" :placeholder="t('payment.admin.productName')" />
      <input v-model.number="form.sort_order" class="input" type="number" :placeholder="t('payment.admin.sortOrder')" />
      <textarea v-model="form.description" class="input min-h-[88px] sm:col-span-2" :placeholder="t('payment.admin.description')"></textarea>
      <textarea v-model="form.features" class="input min-h-[120px] sm:col-span-2" :placeholder="t('payment.admin.features')"></textarea>
      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
        <input v-model="form.for_sale" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
        {{ t('payment.admin.forSale') }}
      </label>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button class="btn btn-primary" :disabled="saving || !canSave" @click="save">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { SubscriptionPlan } from '@/types/payment'

const props = defineProps<{
  show: boolean
  plan: SubscriptionPlan | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const saving = ref(false)

const form = reactive({
  group_id: 0,
  name: '',
  description: '',
  price: 0,
  original_price: null as number | null,
  validity_days: 30,
  validity_unit: 'days',
  features: '',
  product_name: '',
  for_sale: true,
  sort_order: 0,
})

const dialogTitle = computed(() => props.plan ? t('payment.admin.editPlan') : t('payment.admin.createPlan'))
const canSave = computed(() => form.group_id > 0 && form.name.trim() !== '' && form.price > 0 && form.validity_days > 0)

watch(
  () => [props.show, props.plan] as const,
  () => {
    if (!props.show) return
    form.group_id = props.plan?.group_id ?? 0
    form.name = props.plan?.name ?? ''
    form.description = props.plan?.description ?? ''
    form.price = props.plan?.price ?? 0
    form.original_price = props.plan?.original_price ?? null
    form.validity_days = props.plan?.validity_days ?? 30
    form.validity_unit = props.plan?.validity_unit ?? 'days'
    form.features = Array.isArray(props.plan?.features) ? props.plan!.features.join('\n') : ''
    form.product_name = props.plan?.name ?? ''
    form.for_sale = props.plan?.for_sale ?? true
    form.sort_order = props.plan?.sort_order ?? 0
  },
  { immediate: true }
)

async function save() {
  if (!canSave.value || saving.value) return
  saving.value = true
  const payload = {
    group_id: form.group_id,
    name: form.name.trim(),
    description: form.description.trim(),
    price: form.price,
    original_price: form.original_price ?? undefined,
    validity_days: form.validity_days,
    validity_unit: form.validity_unit,
    features: form.features,
    product_name: form.product_name.trim() || form.name.trim(),
    for_sale: form.for_sale,
    sort_order: form.sort_order,
  }
  try {
    if (props.plan) {
      await adminPaymentAPI.updatePlan(props.plan.id, payload)
    } else {
      await adminPaymentAPI.createPlan(payload)
    }
    emit('saved')
    emit('close')
  } finally {
    saving.value = false
  }
}
</script>
