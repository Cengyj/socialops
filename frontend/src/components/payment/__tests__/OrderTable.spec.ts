import { describe, expect, it, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import OrderTable from '../OrderTable.vue'
import { formatPaymentAmount } from '../currency'
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
  id: 42,
  amount: 103,
  pay_amount: 103,
  fee_rate: 0,
  payment_type: 'stripe',
  out_trade_no: 'socialops_order_table_currency',
  status: 'COMPLETED',
  order_type: 'subscription',
  created_at: '2026-06-04T12:00:00Z',
  expires_at: '2026-06-04T12:30:00Z',
  refund_amount: 0,
  ...overrides,
})

describe('OrderTable', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('formats gateway payment amounts with the order currency', () => {
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    }))

    const wrapper = mount(OrderTable, {
      props: {
        orders: [orderFactory({ currency: 'HKD' })],
        loading: false,
      },
      global: {
        stubs: {
          OrderStatusBadge: true,
        },
      },
    })

    expect(wrapper.text()).toContain(formatPaymentAmount(103, 'HKD'))
    expect(wrapper.text()).not.toContain('¥103.00')
  })
})
