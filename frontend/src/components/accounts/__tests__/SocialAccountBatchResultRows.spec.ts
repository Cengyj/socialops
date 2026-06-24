import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SocialAccountBatchResultRows from '../SocialAccountBatchResultRows.vue'

describe('SocialAccountBatchResultRows', () => {
  it('renders formatted rows and remaining-count text', () => {
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.error || item.reason || '-',
        items: [
          { id: 1, name: '@ok', status: 'succeeded' },
          { id: 2, status: 'skipped', reason: 'already_assigned' },
        ],
        remainingCount: 4,
        rowToneClass: status => `tone-${status}`,
        rowsMoreText: '4 more rows',
        statusLabel: status => String(status || '-').toUpperCase(),
      },
    })

    expect(wrapper.text()).toContain('@ok')
    expect(wrapper.text()).toContain('SUCCEEDED')
    expect(wrapper.text()).toContain('#2')
    expect(wrapper.text()).toContain('already_assigned')
    expect(wrapper.text()).toContain('4 more rows')
    expect(wrapper.find('.tone-skipped').exists()).toBe(true)
    expect(wrapper.classes()).toContain('min-w-0')
    expect(wrapper.attributes('role')).toBe('list')
    expect(wrapper.findAll('[role="listitem"]')).toHaveLength(3)
    const firstLabel = wrapper.find('span.font-medium')
    expect(firstLabel.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    expect(firstLabel.attributes('title')).toBe('@ok')
    const firstStatus = wrapper.findAll('span').find(node => node.text() === 'SUCCEEDED')
    expect(firstStatus?.attributes('title')).toBe('SUCCEEDED')
    expect(firstStatus?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.findAll('span').some(node => node.attributes('title') === 'already_assigned')).toBe(true)
    const remainingText = wrapper.findAll('div').find(node => node.text() === '4 more rows')
    expect(remainingText).toBeTruthy()
    expect(remainingText!.attributes('role')).toBe('listitem')
    expect(remainingText!.attributes('aria-live')).toBe('polite')
    expect(remainingText!.attributes('aria-atomic')).toBe('true')
    expect(remainingText!.attributes('title')).toBe('4 more rows')
    expect(remainingText!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })

  it('does not call formatters or render hidden-row text for an empty item list', () => {
    const itemLabel = vi.fn(() => '#1')
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        itemLabel,
        itemMessage: vi.fn(() => '-'),
        items: [],
        remainingCount: 2,
        rowToneClass: vi.fn(() => 'tone-ok'),
        rowsMoreText: '2 more rows',
        statusLabel: vi.fn(() => 'Succeeded'),
      },
    })

    expect(wrapper.text()).toBe('')
    expect(itemLabel).not.toHaveBeenCalled()
  })

  it('derives each visible row once before rendering repeated title and text fields', () => {
    const itemLabel = vi.fn((item, index) => item.name || `#${item.id || index}`)
    const itemMessage = vi.fn(item => item.error || item.reason || '-')
    const statusLabel = vi.fn(status => String(status || '-').toUpperCase())
    const rowToneClass = vi.fn(status => `tone-${status}`)

    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        itemLabel,
        itemMessage,
        items: [
          { id: 1, name: '@ok', status: 'succeeded' },
          { id: 2, status: 'failed', error: 'Missing name' },
        ],
        remainingCount: 0,
        rowToneClass,
        rowsMoreText: '0 more rows',
        statusLabel,
      },
    })

    expect(wrapper.text()).toContain('@ok')
    expect(wrapper.text()).toContain('FAILED')
    expect(wrapper.text()).toContain('Missing name')
    expect(itemLabel).toHaveBeenCalledTimes(2)
    expect(itemMessage).toHaveBeenCalledTimes(2)
    expect(statusLabel).toHaveBeenCalledTimes(2)
    expect(rowToneClass).toHaveBeenCalledTimes(2)
  })

  it('can preserve the compact status-message text used by account dialogs', () => {
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        combineStatusAndMessage: true,
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.error || '-',
        items: [{ id: 1, name: '@failed', status: 'failed', error: 'Selected proxy unavailable' }],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-failed'),
        rowsMoreText: '0 more rows',
        statusLabel: vi.fn(() => 'Error'),
      },
    })

    expect(wrapper.text()).toContain('Error · Selected proxy unavailable')
    expect(wrapper.text()).not.toContain('0 more rows')
    expect(wrapper.find('span.font-medium').attributes('title')).toBe('@failed')
    expect(wrapper.findAll('span').some(node => node.attributes('title') === 'Error · Selected proxy unavailable')).toBe(true)
  })

  it('omits empty compact message placeholders when a row only has a status', () => {
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        combineStatusAndMessage: true,
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: () => '-',
        items: [{ id: 1, name: '@ok', status: 'succeeded' }],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-ok'),
        rowsMoreText: '0 more rows',
        statusLabel: vi.fn(() => 'Success'),
      },
    })

    expect(wrapper.text()).toContain('Success')
    expect(wrapper.text()).not.toContain('Success · -')
    expect(wrapper.findAll('span').some(node => node.attributes('title') === 'Success')).toBe(true)
  })

  it('falls back when row formatters return blank display text', () => {
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        combineStatusAndMessage: true,
        itemLabel: () => '   ',
        itemMessage: () => ' ',
        items: [
          { id: 7, status: 'failed', error: ' ' },
          { status: null as unknown as string, reason: null },
        ],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-fallback'),
        rowsMoreText: '0 more rows',
        statusLabel: () => '',
      },
    })

    const labels = wrapper.findAll('span.font-medium')
    expect(labels.map(label => label.text())).toEqual(['#7', '#2'])
    expect(labels.map(label => label.attributes('title'))).toEqual(['#7', '#2'])
    expect(wrapper.text()).not.toContain('- · -')
    expect(wrapper.findAll('span').some(node => node.attributes('title') === '-')).toBe(true)
  })

  it('keeps long status labels readable in normal row layout', () => {
    const longStatus = 'Backend fallback status with a very long safe label 0123456789abcdef'
    const wrapper = mount(SocialAccountBatchResultRows, {
      props: {
        itemLabel: item => item.name || `#${item.id}`,
        itemMessage: item => item.reason || '-',
        items: [{ id: 1, name: '@pending', status: 'unknown_long_status', reason: 'pending_review' }],
        remainingCount: 0,
        rowToneClass: vi.fn(() => 'tone-pending'),
        rowsMoreText: '0 more rows',
        statusLabel: vi.fn(() => longStatus),
      },
    })

    const status = wrapper.findAll('span').find(node => node.text() === longStatus)

    expect(status?.attributes('title')).toBe(longStatus)
    expect(status?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })
})
