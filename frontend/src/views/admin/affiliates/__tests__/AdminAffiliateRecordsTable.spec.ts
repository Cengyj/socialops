import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminAffiliateRecordsTable from '../AdminAffiliateRecordsTable.vue'
import { formatPaymentAmount } from '@/components/payment/currency'

const listInviteRecords = vi.hoisted(() => vi.fn())
const listRebateRecords = vi.hoisted(() => vi.fn())
const listTransferRecords = vi.hoisted(() => vi.fn())
const getUserOverview = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())

const affiliatesAPIMock = vi.hoisted(() => ({
  listInviteRecords,
  listRebateRecords,
  listTransferRecords,
  getUserOverview,
}))

vi.mock('@/api/admin/affiliates', () => ({
  affiliatesAPI: affiliatesAPIMock,
  default: affiliatesAPIMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, _params?: Record<string, unknown>, fallback?: string) => fallback ?? key,
    }),
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <table>
      <tbody>
        <tr v-for="row in data" :key="row.order_id || row.ledger_id || row.invitee_id">
          <td v-for="column in columns" :key="column.key">
            <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  `,
}

describe('AdminAffiliateRecordsTable', () => {
  it('formats rebate gateway pay amount with the order currency', async () => {
    listRebateRecords.mockResolvedValue({
      items: [
        {
          order_id: 9,
          out_trade_no: 'affiliate-hkd-order',
          inviter_id: 1,
          inviter_email: 'inviter@example.com',
          inviter_username: 'inviter',
          invitee_id: 2,
          invitee_email: 'buyer@example.com',
          invitee_username: 'buyer',
          order_amount: 100,
          pay_amount: 103,
          currency: 'HKD',
          rebate_amount: 20,
          payment_type: 'stripe',
          order_status: 'COMPLETED',
          created_at: '2026-06-04T12:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(AdminAffiliateRecordsTable, {
      props: {
        type: 'rebates',
      },
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: '<div><slot /></div>' },
          Icon: true,
          OrderStatusBadge: true,
        },
      },
    })

    await flushPromises()

    expect(listRebateRecords).toHaveBeenCalled()
    expect(wrapper.text()).toContain(formatPaymentAmount(103, 'HKD'))
    expect(wrapper.text()).not.toContain('¥103.00')
  })
})
