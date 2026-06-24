import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AdminRefundDialog from '../AdminRefundDialog.vue'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { PaymentOrder } from '@/types/payment'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const orderFactory = (overrides: Partial<PaymentOrder> = {}): PaymentOrder => ({
  id: 77,
  amount: 100,
  pay_amount: 103,
  fee_rate: 3,
  currency: 'HKD',
  payment_type: 'stripe',
  out_trade_no: 'socialops_refund_dialog_currency',
  status: 'COMPLETED',
  order_type: 'subscription',
  created_at: '2026-06-04T12:00:00Z',
  expires_at: '2026-06-04T12:30:00Z',
  refund_amount: 0,
  ...overrides,
})

describe('AdminRefundDialog', () => {
  it('formats gateway payment amounts with the order currency', () => {
    const wrapper = mount(AdminRefundDialog, {
      props: {
        show: true,
        order: orderFactory(),
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain(formatPaymentAmount(103, 'HKD'))
    expect(wrapper.text()).not.toContain('¥103.00')
  })
})
