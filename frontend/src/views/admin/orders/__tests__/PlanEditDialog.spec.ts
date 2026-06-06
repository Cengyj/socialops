import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlanEditDialog from '../PlanEditDialog.vue'

const mocks = vi.hoisted(() => ({
  createPlan: vi.fn(),
  updatePlan: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    createPlan: mocks.createPlan,
    updatePlan: mocks.updatePlan,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic: mocks.recordClientDiagnostic,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'common.cancel': 'Cancel',
      'common.save': 'Save',
      'common.saved': 'Saved',
      'common.saving': 'Saving',
      'common.selectOption': 'Select option',
      'payment.admin.advancedSettings': 'Advanced settings',
      'payment.admin.createPlan': 'Create plan',
      'payment.admin.dailyGuardrailExceedsQuota': 'Daily guardrail cannot exceed the package quota',
      'payment.admin.days': 'days',
      'payment.admin.forSale': 'For sale',
      'payment.admin.months': 'months',
      'payment.admin.originalPrice': 'Original price',
      'payment.admin.planName': 'Plan name',
      'payment.admin.platform': 'Platform',
      'payment.admin.price': 'Price',
      'payment.admin.priceRequired': 'Price is required',
      'payment.admin.productName': 'Product name',
      'payment.admin.productNameHint': 'Shared package name',
      'payment.admin.quotaAmount': 'Quota amount',
      'payment.admin.quotaPresetHint': 'Choose a common quota',
      'payment.admin.quotaRequired': 'Quota is required',
      'payment.admin.sortOrder': 'Sort order',
      'payment.admin.validityDaysRequired': 'Validity is required',
      'payment.admin.validityUnit': 'Validity unit',
      'payment.admin.validityValue': 'Validity value',
      'payment.admin.weeklyGuardrail': 'Weekly guardrail',
      'payment.admin.dailyGuardrail': 'Daily guardrail',
      'payment.admin.features': 'Features',
      'payment.admin.featuresHint': 'One feature per line',
      'payment.admin.featuresPlaceholder': 'Follow\nPost',
      'payment.admin.planDescription': 'Description',
      'payment.admin.weeks': 'weeks',
      'payment.admin.years': 'years',
    }[key] || key),
  }),
}))

const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot /><footer><slot name="footer" /></footer></section>',
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: `
    <select
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value); $emit('change', $event.target.value)"
    >
      <option v-for="option in options" :key="String(option.value)" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `,
}

function mountDialog() {
  return mount(PlanEditDialog, {
    props: {
      show: true,
      plan: null,
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: SelectStub,
        Icon: true,
      },
    },
  })
}

async function fillRequiredFields(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.get('[data-testid="plan-name"]').setValue('X Monthly 100')
  await wrapper.get('[data-testid="plan-price"]').setValue('9.9')
  await wrapper.get('[data-testid="plan-validity-days"]').setValue('1')
  await wrapper.get('[data-testid="plan-quota-usd"]').setValue('100')
}

async function openAdvancedSettings(wrapper: ReturnType<typeof mountDialog>) {
  const toggle = wrapper.findAll('button').find((button) => button.text().includes('Advanced settings'))
  expect(toggle).toBeTruthy()
  await toggle!.trigger('click')
}

describe('PlanEditDialog', () => {
  beforeEach(() => {
    mocks.createPlan.mockReset()
    mocks.updatePlan.mockReset()
    mocks.showError.mockReset()
    mocks.showSuccess.mockReset()
    mocks.recordClientDiagnostic.mockReset()
    mocks.createPlan.mockResolvedValue({ data: { id: 1 } })
  })

  it('rejects guardrails that exceed the package quota', async () => {
    const wrapper = mountDialog()

    await fillRequiredFields(wrapper)
    await openAdvancedSettings(wrapper)
    await wrapper.get('[data-testid="plan-daily-guardrail"]').setValue('120')
    await wrapper.find('form').trigger('submit')

    expect(mocks.showError).toHaveBeenCalledWith('Daily guardrail cannot exceed the package quota')
    expect(mocks.createPlan).not.toHaveBeenCalled()
  })

  it('creates a plan with semantic quota and optional guardrails', async () => {
    const wrapper = mountDialog()

    await fillRequiredFields(wrapper)
    await openAdvancedSettings(wrapper)
    await wrapper.get('[data-testid="plan-daily-guardrail"]').setValue('10')
    await wrapper.get('[data-testid="plan-weekly-guardrail"]').setValue('60')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.createPlan).toHaveBeenCalledWith(expect.objectContaining({
      name: 'X Monthly 100',
      platform: 'x_twitter',
      price: 9.9,
      validity_days: 1,
      validity_unit: 'months',
      quota_usd: 100,
      daily_limit_usd: 10,
      weekly_limit_usd: 60,
      product_name: 'X Monthly 100',
    }))
    expect(wrapper.emitted('saved')).toBeTruthy()
  })

  it('shows a safe save failure while retaining diagnostics', async () => {
    mocks.createPlan.mockRejectedValueOnce(new Error('provider stack trace token=secret'))
    const wrapper = mountDialog()

    await fillRequiredFields(wrapper)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.recordClientDiagnostic).toHaveBeenCalledWith(
      'admin.payment_plans.save',
      expect.any(Error),
    )
    expect(mocks.showError).toHaveBeenCalledWith('common.error')
    expect(mocks.showError).not.toHaveBeenCalledWith(expect.stringContaining('token=secret'))
  })
})
