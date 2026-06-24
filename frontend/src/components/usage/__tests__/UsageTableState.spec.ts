import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageTableState from '../UsageTableState.vue'

const messages: Record<string, string> = {
  'usage.loading': 'Loading usage records...',
  'usage.failedToLoad': 'Failed to load usage records',
  'usage.empty': 'No SocialOps operations yet',
  'usage.emptyFiltered': 'No usage records match the current filters.',
  'usage.filters.clear': 'Clear filters',
  'common.refresh': 'Refresh',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('UsageTableState', () => {
  it('renders the loading state with a stable test id and icon', () => {
    const wrapper = mount(UsageTableState, {
      props: { state: 'loading' },
    })

    expect(wrapper.get('[data-testid="usage-loading-state"]').text()).toContain('Loading usage records...')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('emits retry from the error state action', async () => {
    const wrapper = mount(UsageTableState, {
      props: { state: 'error' },
    })

    expect(wrapper.get('[data-testid="usage-error-state"]').text()).toContain('Failed to load usage records')
    await wrapper.get('[data-testid="usage-retry-load"]').trigger('click')

    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('distinguishes regular empty and filtered empty states', async () => {
    const empty = mount(UsageTableState, {
      props: { state: 'empty' },
    })

    expect(empty.get('[data-testid="usage-empty-state"]').text()).toContain('No SocialOps operations yet')
    expect(empty.find('[data-testid="usage-empty-clear-filters"]').exists()).toBe(false)

    const filtered = mount(UsageTableState, {
      props: { state: 'empty', filtered: true },
    })

    expect(filtered.get('[data-testid="usage-empty-state"]').text()).toContain('No usage records match the current filters.')
    await filtered.get('[data-testid="usage-empty-clear-filters"]').trigger('click')

    expect(filtered.emitted('clear')).toHaveLength(1)
  })
})
