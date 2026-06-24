import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CommonStatsGrid from '../CommonStatsGrid.vue'

describe('CommonStatsGrid', () => {
  it('renders inline stat cards with optional meta text and stable ids', () => {
    const wrapper = mount(CommonStatsGrid, {
      props: {
        gridClass: 'grid custom-grid',
        sectionTestId: 'shared-stats',
        stats: [
          { key: 'total', label: 'Total', value: 8, meta: 'All rows', testId: 'stat-total' },
          { key: 'ready', label: 'Ready', value: '3', testId: 'stat-ready' },
        ],
      },
    })

    expect(wrapper.get('[data-testid="shared-stats"]').classes()).toContain('custom-grid')
    expect(wrapper.get('[data-testid="stat-total"]').text()).toContain('Total')
    expect(wrapper.get('[data-testid="stat-total"]').text()).toContain('8')
    expect(wrapper.get('[data-testid="stat-total"]').text()).toContain('All rows')
    expect(wrapper.get('[data-testid="stat-ready"]').text()).toContain('Ready')
    expect(wrapper.get('[data-testid="stat-ready"]').find('.mt-1').exists()).toBe(false)
  })

  it('renders stacked stat cards with caller-owned classes', () => {
    const wrapper = mount(CommonStatsGrid, {
      props: {
        cardClass: 'rounded custom-card',
        layout: 'stacked',
        stackedValueClass: 'value-class',
        stats: [
          { key: 'online', label: 'Online', value: 2 },
        ],
      },
    })

    const card = wrapper.get('.custom-card')

    expect(card.text()).toContain('Online')
    expect(card.get('.value-class').text()).toBe('2')
  })

  it('stays presentation-only without actions or emitted events', () => {
    const wrapper = mount(CommonStatsGrid, {
      props: {
        stats: [
          { key: 'assigned', label: 'Assigned', value: 3 },
        ],
      },
    })

    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.emitted()).toEqual({})
  })
})
