import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SocialAccountBatchResultPanel from '../SocialAccountBatchResultPanel.vue'

describe('SocialAccountBatchResultPanel', () => {
  it('renders formatted batch rows, remaining count text, and dismiss action', async () => {
    const wrapper = mount(SocialAccountBatchResultPanel, {
      props: {
        dismissLabel: 'Clear result',
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.error || item.reason || '-',
        items: [
          { id: 101, name: '@northwind', status: 'succeeded' },
          { id: 102, status: 'failed', reason: 'invalid_input', error: 'Missing name' },
        ],
        remainingCount: 3,
        rowToneClass: status => status === 'failed' ? 'tone-failed' : 'tone-ok',
        rowsMoreText: '3 more rows',
        statusLabel: status => String(status || '-').toUpperCase(),
        summary: 'Result summary',
        testId: 'batch-result-panel',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.get('[data-testid="batch-result-panel"]').text()).toContain('Result summary')
    const summary = wrapper.get('div[title="Result summary"]')
    expect(summary.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(summary.attributes('role')).toBe('status')
    expect(summary.attributes('aria-live')).toBe('polite')
    expect(summary.attributes('aria-atomic')).toBe('true')
    expect(wrapper.text()).toContain('@northwind')
    expect(wrapper.text()).toContain('SUCCEEDED')
    expect(wrapper.text()).toContain('FAILED')
    expect(wrapper.text()).toContain('Missing name')
    expect(wrapper.text()).toContain('3 more rows')
    expect(wrapper.find('.tone-failed').exists()).toBe(true)

    const dismissButton = wrapper.get('button[aria-label="Clear result"]')
    expect(dismissButton.attributes('title')).toBe('Clear result')
    expect(dismissButton.classes()).toEqual(expect.arrayContaining(['h-8', 'w-8', 'min-w-[2rem]', 'max-w-[2rem]', 'justify-center', 'px-0']))

    await dismissButton.trigger('click')
    expect(wrapper.emitted('dismiss')).toHaveLength(1)
  })

  it('keeps the hidden-row affordance absent when all rows are visible', () => {
    const wrapper = mount(SocialAccountBatchResultPanel, {
      props: {
        dismissLabel: 'Clear result',
        itemLabel: vi.fn(() => '#1'),
        itemMessage: vi.fn(() => '-'),
        items: [{ id: 1, status: 'succeeded' }],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-ok'),
        rowsMoreText: '0 more rows',
        statusLabel: vi.fn(() => 'Succeeded'),
        summary: 'All visible',
        testId: 'batch-result-panel',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('All visible')
    expect(wrapper.text()).not.toContain('0 more rows')
  })
})
