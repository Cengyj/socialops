import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageDetailFieldGrid from '../UsageDetailFieldGrid.vue'

describe('UsageDetailFieldGrid', () => {
  it('renders usage detail field labels and values in a reusable grid', () => {
    const wrapper = mount(UsageDetailFieldGrid, {
      props: {
        rows: [
          { label: 'Operation', value: 'Follow' },
          { label: 'Result', value: 'Success' },
        ],
      },
    })

    expect(wrapper.text()).toContain('Operation')
    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Result')
    expect(wrapper.text()).toContain('Success')
    expect(wrapper.findAll('.rounded-xl')).toHaveLength(2)
  })

  it('supports muted technical fields for weak internal identifiers', () => {
    const wrapper = mount(UsageDetailFieldGrid, {
      props: {
        rows: [{ label: 'Idempotency key', value: 'usage-task-123' }],
        tone: 'muted',
        valueStyle: 'technical',
      },
    })

    const value = wrapper.find('.font-mono')
    expect(value.exists()).toBe(true)
    expect(value.text()).toBe('usage-task-123')
    expect(wrapper.find('.bg-gray-50').exists()).toBe(true)
  })
})
