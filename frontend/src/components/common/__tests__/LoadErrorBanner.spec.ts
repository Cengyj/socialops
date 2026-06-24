import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import LoadErrorBanner from '../LoadErrorBanner.vue'

describe('LoadErrorBanner', () => {
  it('renders a constrained warning banner and emits retry', async () => {
    const wrapper = mount(LoadErrorBanner, {
      props: {
        title: 'Load failed',
        message: 'A very long load error message that should wrap instead of overflowing its container.',
        retryLabel: 'Retry',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Load failed')
    expect(wrapper.text()).toContain('A very long load error message')
    expect(wrapper.classes()).toEqual(expect.arrayContaining(['rounded-lg', 'border', 'border-red-200', 'bg-red-50', 'p-4']))
    expect(wrapper.attributes('role')).toBe('alert')
    expect(wrapper.attributes('aria-live')).toBe('assertive')
    expect(wrapper.attributes('aria-atomic')).toBe('true')

    const message = wrapper.get('p[title="A very long load error message that should wrap instead of overflowing its container."]')
    expect(message.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))

    const icon = wrapper.getComponent({ name: 'Icon' })
    expect(icon.props()).toMatchObject({
      name: 'exclamationTriangle',
      size: 'md',
    })

    const button = wrapper.get('button[aria-label="Retry"]')
    expect(button.attributes('title')).toBe('Retry')
    expect(button.text()).toBe('Retry')

    await button.trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('does not repeat the detail line when the safe fallback matches the title', () => {
    const wrapper = mount(LoadErrorBanner, {
      props: {
        title: 'Failed to load proxies',
        message: '  Failed to load proxies  ',
        retryLabel: 'Retry',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Failed to load proxies')
    expect(wrapper.text().match(/Failed to load proxies/g)).toHaveLength(1)
    expect(wrapper.find('p[title="Failed to load proxies"]').exists()).toBe(false)
    expect(wrapper.get('button[aria-label="Retry"]').exists()).toBe(true)
  })
})
