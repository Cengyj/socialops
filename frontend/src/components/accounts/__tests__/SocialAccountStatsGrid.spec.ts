import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SocialAccountStatsGrid from '../SocialAccountStatsGrid.vue'

describe('SocialAccountStatsGrid', () => {
  it('renders stat cards with stable test ids, values, and meta text', () => {
    const wrapper = mount(SocialAccountStatsGrid, {
      props: {
        gridClass: 'grid gap-2 custom-grid',
        testIdPrefix: 'account-stat',
        stats: [
          { key: 'running', label: 'Running', value: 2, meta: 'Active tasks' },
          { key: 'failed', label: 'Failed', value: '1', meta: 'Needs review' },
        ],
      },
    })

    expect(wrapper.classes()).toContain('custom-grid')
    expect(wrapper.get('[data-testid="account-stat-running"]').text()).toContain('Running')
    expect(wrapper.get('[data-testid="account-stat-running"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="account-stat-running"]').text()).toContain('Active tasks')
    expect(wrapper.get('[data-testid="account-stat-failed"]').text()).toContain('Needs review')
  })

  it('omits the meta row when a stat has no meta text', () => {
    const wrapper = mount(SocialAccountStatsGrid, {
      props: {
        testIdPrefix: 'total-account-stat',
        stats: [
          { key: 'total', label: 'Total', value: 12 },
        ],
      },
    })

    const card = wrapper.get('[data-testid="total-account-stat-total"]')

    expect(card.text()).toContain('Total')
    expect(card.text()).toContain('12')
    expect(card.find('.mt-1').exists()).toBe(false)
  })

  it('stays presentation-only without actions or emitted events', () => {
    const wrapper = mount(SocialAccountStatsGrid, {
      props: {
        testIdPrefix: 'account-stat',
        stats: [
          { key: 'assigned', label: 'Assigned', value: 3 },
        ],
      },
    })

    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.emitted()).toEqual({})
  })
})
