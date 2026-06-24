import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageStatsCards from '../UsageStatsCards.vue'

describe('UsageStatsCards', () => {
  it('renders SocialOps usage statistic cards with icons and meta text', () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        items: [
          {
            label: 'Total Operations',
            value: '34',
            meta: 'In selected range',
            icon: 'chart',
            iconWrapClass: 'bg-blue-100',
            iconClass: 'text-blue-600',
            valueClass: 'text-gray-900',
          },
          {
            label: 'Total Charged',
            value: '$8.25',
            meta: 'Charged successful tasks',
            icon: 'dollar',
            iconWrapClass: 'bg-green-100',
            iconClass: 'text-green-600',
            valueClass: 'text-green-600',
            cardClass: 'border-green-100 bg-green-50/40',
          },
        ],
      },
    })

    const cards = wrapper.findAll('[data-testid="usage-stat-card"]')
    expect(cards).toHaveLength(2)
    expect(cards[0].text()).toContain('Total Operations')
    expect(cards[0].text()).toContain('34')
    expect(cards[0].text()).toContain('In selected range')
    expect(cards[1].text()).toContain('Total Charged')
    expect(cards[1].text()).toContain('$8.25')
    expect(cards[1].text()).toContain('Charged successful tasks')
    expect(wrapper.findAll('svg')).toHaveLength(2)
    expect(cards[0].classes()).toContain('card-hover')
    expect(cards[0].find('.h-11.w-11').exists()).toBe(true)
    expect(cards[0].find('.tabular-nums').exists()).toBe(true)
    expect(cards[1].classes()).toEqual(expect.arrayContaining(['border-green-100', 'bg-green-50/40']))
    expect(cards[1].find('.text-green-600').exists()).toBe(true)
  })
})
