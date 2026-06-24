import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SocialAccountBatchResultCard from '../SocialAccountBatchResultCard.vue'

describe('SocialAccountBatchResultCard', () => {
  it('renders the account-dialog batch result card with compact row messages by default', () => {
    const wrapper = mount(SocialAccountBatchResultCard, {
      props: {
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.error || item.reason || '-',
        items: [{ id: 7, name: '@failed', status: 'failed', error: 'Selected proxy unavailable' }],
        remainingCount: 2,
        rowToneClass: vi.fn(() => 'tone-failed'),
        rowsMoreText: '2 more rows',
        statusLabel: vi.fn(() => 'Failed'),
        summary: 'Total 3; failed 1.',
        testId: 'account-dialog-batch-result',
        title: 'Assignment result',
      },
    })

    const panel = wrapper.get('[data-testid="account-dialog-batch-result"]')
    expect(panel.classes()).toEqual(expect.arrayContaining(['rounded-xl', 'border', 'bg-white', 'p-3']))
    expect(panel.text()).toContain('Assignment result')
    expect(panel.text()).toContain('Total 3; failed 1.')
    const summary = wrapper.get('div[title="Total 3; failed 1."]')
    expect(summary.attributes('role')).toBe('status')
    expect(summary.attributes('aria-live')).toBe('polite')
    expect(summary.attributes('aria-atomic')).toBe('true')
    expect(panel.text()).toContain('@failed')
    expect(panel.text()).toContain('Failed · Selected proxy unavailable')
    expect(panel.text()).toContain('2 more rows')
  })

  it('can render the full status and message columns when a caller needs the shared row layout', () => {
    const wrapper = mount(SocialAccountBatchResultCard, {
      props: {
        combineStatusAndMessage: false,
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.reason || '-',
        items: [{ id: 9, name: '@skipped', status: 'skipped', reason: 'already assigned' }],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-skipped'),
        rowsMoreText: '0 more rows',
        statusLabel: vi.fn(() => 'Skipped'),
        summary: 'All visible.',
        title: 'Store result',
      },
    })

    expect(wrapper.text()).toContain('Store result')
    expect(wrapper.text()).toContain('Skipped')
    expect(wrapper.text()).toContain('already assigned')
    expect(wrapper.text()).not.toContain('Skipped · already assigned')
    expect(wrapper.text()).not.toContain('0 more rows')
  })
})
