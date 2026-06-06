<template>
  <BaseDialog :show="show" :title="dialogTitle" width="wide" @close="emit('close')">
    <form id="plan-form" class="space-y-4" @submit.prevent="handleSavePlan">
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.planName') }} <span class="text-red-500">*</span></label>
          <input v-model="planForm.name" data-testid="plan-name" type="text" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.platform') }} <span class="text-red-500">*</span></label>
          <Select
            v-model="planForm.platform"
            :options="platformOptions"
            :placeholder="t('common.selectOption')"
            class="w-full"
          />
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.price') }} <span class="text-red-500">*</span></label>
          <input v-model.number="planForm.price" data-testid="plan-price" type="number" step="0.01" min="0.01" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.originalPrice') }}</label>
          <input v-model.number="planForm.original_price" type="number" step="0.01" min="0" class="input" />
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-3">
        <div>
          <label class="input-label">{{ t('payment.admin.validityValue') }} <span class="text-red-500">*</span></label>
          <input v-model.number="planForm.validity_days" data-testid="plan-validity-days" type="number" min="1" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.validityUnit') }} <span class="text-red-500">*</span></label>
          <Select v-model="planForm.validity_unit" :options="validityUnitOptions" />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.quotaAmount') }} <span class="text-red-500">*</span></label>
          <input v-model.number="planForm.quota_usd" data-testid="plan-quota-usd" type="number" step="0.01" min="0.01" class="input" required />
          <div class="mt-2 flex flex-wrap gap-2">
            <button
              v-for="amount in quotaPresetOptions"
              :key="amount"
              type="button"
              :class="[
                'rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
                isQuotaPresetSelected(amount)
                  ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/30 dark:text-primary-200'
                  : 'border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-700'
              ]"
              @click="setQuotaPreset(amount)"
            >
              ${{ amount }}
            </button>
          </div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('payment.admin.quotaPresetHint') }}
          </p>
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.sortOrder') }}</label>
          <input v-model.number="planForm.sort_order" type="number" min="0" class="input" />
        </div>
        <label class="flex items-center gap-3 self-end text-sm text-gray-700 dark:text-gray-300">
          <button
            type="button"
            :class="[
              'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              planForm.for_sale ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
            ]"
            @click="planForm.for_sale = !planForm.for_sale"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transition',
                planForm.for_sale ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
          {{ t('payment.admin.forSale') }}
        </label>
      </div>

      <div>
        <button
          type="button"
          class="flex w-full items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-left text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:text-gray-200 dark:hover:bg-dark-700"
          @click="showAdvanced = !showAdvanced"
        >
          <span>{{ t('payment.admin.advancedSettings') }}</span>
          <Icon :name="showAdvanced ? 'chevronUp' : 'chevronDown'" size="sm" />
        </button>
      </div>

      <div v-if="showAdvanced" class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700">
        <div>
          <label class="input-label">{{ t('payment.admin.productName') }}</label>
          <input v-model="planForm.product_name" type="text" class="input" />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('payment.admin.productNameHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">{{ t('payment.admin.planDescription') }}</label>
          <textarea v-model="planForm.description" rows="2" class="input"></textarea>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.admin.dailyGuardrail') }}</label>
            <input v-model.number="planForm.daily_limit_usd" data-testid="plan-daily-guardrail" type="number" step="0.01" min="0" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.weeklyGuardrail') }}</label>
            <input v-model.number="planForm.weekly_limit_usd" data-testid="plan-weekly-guardrail" type="number" step="0.01" min="0" class="input" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('payment.admin.features') }}</label>
          <textarea
            v-model="planFeaturesText"
            rows="3"
            class="input"
            :placeholder="t('payment.admin.featuresPlaceholder')"
          ></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.featuresHint') }}</p>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="plan-form" class="btn btn-primary" :disabled="saving">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI, type SubscriptionPlanPayload } from '@/api/admin/payment'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import type { SubscriptionPlan } from '@/types/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  SUBSCRIPTION_QUOTA_PRESETS,
  getSubscriptionPlanPlatformOptions,
} from '@/utils/subscriptionPlanDisplay'
import {
  buildSubscriptionPlanPayload,
  createSubscriptionPlanFormDefaults,
  shouldShowSubscriptionPlanAdvancedSettings,
  subscriptionPlanToFormState,
  validateSubscriptionPlanForm,
} from '@/utils/subscriptionPlanForm'

const props = defineProps<{
  show: boolean
  plan: SubscriptionPlan | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const showAdvanced = ref(false)

const planForm = reactive(createSubscriptionPlanFormDefaults())
const planFeaturesText = ref('')

const dialogTitle = computed(() => (props.plan ? t('payment.admin.editPlan') : t('payment.admin.createPlan')))

const validityUnitOptions = computed(() => [
  { value: 'days', label: t('payment.admin.days') },
  { value: 'weeks', label: t('payment.admin.weeks') },
  { value: 'months', label: t('payment.admin.months') },
  { value: 'years', label: t('payment.admin.years') }
])

const platformOptions = computed(() => getSubscriptionPlanPlatformOptions())
const quotaPresetOptions = SUBSCRIPTION_QUOTA_PRESETS

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    Object.assign(planForm, subscriptionPlanToFormState(props.plan))
    planFeaturesText.value = (props.plan?.features || []).join('\n')
    showAdvanced.value = shouldShowSubscriptionPlanAdvancedSettings(planForm, planFeaturesText.value)
  }
)

function buildPlanPayload(): SubscriptionPlanPayload {
  return buildSubscriptionPlanPayload(planForm, planFeaturesText.value)
}

async function handleSavePlan() {
  if (!planForm.platform) {
    appStore.showError(t('payment.admin.platform'))
    return
  }
  if (!planForm.price || planForm.price <= 0) {
    appStore.showError(t('payment.admin.priceRequired'))
    return
  }
  if (!planForm.validity_days || planForm.validity_days < 1) {
    appStore.showError(t('payment.admin.validityDaysRequired'))
    return
  }
  if (!planForm.quota_usd || planForm.quota_usd <= 0) {
    appStore.showError(t('payment.admin.quotaRequired'))
    return
  }
  const validation = validateSubscriptionPlanForm(planForm)
  if (!validation.valid) {
    appStore.showError(t(validation.messageKey || 'payment.admin.quotaRequired'))
    return
  }

  saving.value = true
  try {
    const payload = buildPlanPayload()
    if (props.plan) {
      await adminPaymentAPI.updatePlan(props.plan.id, payload)
    } else {
      await adminPaymentAPI.createPlan(payload)
    }
    appStore.showSuccess(t('common.saved'))
    emit('close')
    emit('saved')
  } catch (error: unknown) {
    recordClientDiagnostic('admin.payment_plans.save', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('common.error')))
  } finally {
    saving.value = false
  }
}

function setQuotaPreset(amount: number) {
  planForm.quota_usd = amount
}

function isQuotaPresetSelected(amount: number): boolean {
  return Number(planForm.quota_usd) === amount
}

</script>
