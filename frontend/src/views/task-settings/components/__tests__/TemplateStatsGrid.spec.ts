import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import TemplateStatsGrid, { type TemplateStatCard } from '../TemplateStatsGrid.vue'

const stats: TemplateStatCard[] = [
  {
    testId: 'template-stats-total',
    label: 'Total templates',
    meta: 'Saved Follow templates in this workspace',
    value: 2,
  },
  {
    testId: 'template-stats-defaults',
    label: 'Default templates',
    meta: 'At most one default for the current Follow type',
    value: 1,
  },
]

describe('TemplateStatsGrid', () => {
  it('renders the existing task template statistic cards', () => {
    const wrapper = mount(TemplateStatsGrid, { props: { stats } })

    expect(wrapper.get('[data-testid="template-stats"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('Total templates')
    expect(wrapper.get('[data-testid="template-stats-total"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="template-stats-defaults"]').text()).toContain('Default templates')
  })

  it('keeps narrow-screen width constraints on the stats grid and cards', () => {
    const wrapper = mount(TemplateStatsGrid, { props: { stats } })

    expect(wrapper.get('[data-testid="template-stats"]').classes()).toContain('min-w-0')
    expect(wrapper.get('[data-testid="template-stats-total"]').classes()).toContain('min-w-0')
  })
})
