import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ProxyStatsGrid from '../components/ProxyStatsGrid.vue'

describe('ProxyStatsGrid', () => {
  it('renders proxy statistic cards in the existing stacked layout', () => {
    const wrapper = mount(ProxyStatsGrid, {
      props: {
        stats: [
          { label: 'Total', value: 4 },
          { label: 'Online', value: 2 },
        ],
      },
    })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['grid', 'gap-3', 'sm:grid-cols-2', 'xl:grid-cols-4']))
    expect(wrapper.text()).toContain('Total')
    expect(wrapper.text()).toContain('4')
    expect(wrapper.text()).toContain('Online')
    expect(wrapper.text()).toContain('2')
    expect(wrapper.find('.mt-1.text-xl.font-semibold').exists()).toBe(true)
  })

  it('stays presentation-only without actions or emitted events', () => {
    const wrapper = mount(ProxyStatsGrid, {
      props: {
        stats: [
          { label: 'Offline', value: 1 },
        ],
      },
    })

    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.emitted()).toEqual({})
  })
})
