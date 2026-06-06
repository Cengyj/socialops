import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminOrdersView from '../AdminOrdersView.vue'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { PaymentOrder } from '@/types/payment'

const getOrders = vi.hoisted(() => vi.fn())
const getOrder = vi.hoisted(() => vi.fn())
const cancelOrder = vi.hoisted(() => vi.fn())
const retryRecharge = vi.hoisted(() => vi.fn())
const refundOrder = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getOrders,
    getOrder,
    cancelOrder,
    retryRecharge,
    refundOrder,
  },
  default: {
    getOrders,
    getOrder,
    cancelOrder,
    retryRecharge,
    refundOrder,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

const orderFactory = (overrides: Partial<PaymentOrder> = {}): PaymentOrder => ({
  id: 77,
  user_id: 9,
  user_email: 'buyer@example.com',
  user_name: 'buyer',
  amount: 100,
  pay_amount: 103,
  fee_rate: 3,
  currency: 'HKD',
  payment_type: 'stripe',
  out_trade_no: 'sub2_admin_view_currency',
  status: 'COMPLETED',
  order_type: 'subscription',
  created_at: '2026-06-04T12:00:00Z',
  expires_at: '2026-06-04T12:30:00Z',
  refund_amount: 0,
  ...overrides,
})

describe('AdminOrdersView', () => {
  beforeEach(() => {
    getOrders.mockReset()
    getOrder.mockReset()
    cancelOrder.mockReset()
    retryRecharge.mockReset()
    refundOrder.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('formats order detail gateway amounts with the order currency', async () => {
    const order = orderFactory()
    getOrders.mockResolvedValue({
      data: {
        items: [order],
        total: 1,
      },
    })
    getOrder.mockResolvedValue({
      data: {
        order,
        auditLogs: [],
      },
    })

    const wrapper = mount(AdminOrdersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: '<div />',
          },
          Pagination: true,
          Icon: true,
          OrderStatusBadge: true,
          AdminRefundDialog: true,
          OrderTable: {
            props: ['orders'],
            template: `
              <div>
                <div v-for="row in orders" :key="row.id">
                  <slot name="actions" :row="row" />
                </div>
              </div>
            `,
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
        },
      },
    })

    await flushPromises()
    const viewButton = wrapper.findAll('button').find(button => button.text().includes('common.view'))
    expect(viewButton).toBeTruthy()
    await viewButton!.trigger('click')
    await flushPromises()

    expect(getOrder).toHaveBeenCalledWith(77)
    expect(wrapper.text()).toContain(formatPaymentAmount(103, 'HKD'))
    expect(wrapper.text()).not.toContain('¥103.00')
  })

  it('surfaces order detail load errors instead of silently hiding missing audit logs', async () => {
    const order = orderFactory()
    getOrders.mockResolvedValue({
      data: {
        items: [order],
        total: 1,
      },
    })
    getOrder.mockRejectedValue({ message: 'audit unavailable' })

    const wrapper = mount(AdminOrdersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: '<div />',
          },
          Pagination: true,
          Icon: true,
          OrderStatusBadge: true,
          AdminRefundDialog: true,
          OrderTable: {
            props: ['orders'],
            template: `
              <div>
                <div v-for="row in orders" :key="row.id">
                  <slot name="actions" :row="row" />
                </div>
              </div>
            `,
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
        },
      },
    })

    await flushPromises()
    const viewButton = wrapper.findAll('button').find(button => button.text().includes('common.view'))
    expect(viewButton).toBeTruthy()
    await viewButton!.trigger('click')
    await flushPromises()

    expect(getOrder).toHaveBeenCalledWith(77)
    expect(showError).toHaveBeenCalledWith('audit unavailable')
  })
})
