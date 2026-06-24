import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import TemplatePoolAnalysisPanel from '../TemplatePoolAnalysisPanel.vue'
import type { TemplatePoolAnalysis } from '../../templatePool'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, number>) => ({
      'taskSettings.pool.duplicateHint': `${params?.count ?? 0} duplicate value(s) detected. You can deduplicate before saving.`,
      'taskSettings.pool.duplicates': 'Duplicates',
      'taskSettings.pool.emptyLines': 'Empty lines',
      'taskSettings.pool.emptyLinesHint': `${params?.count ?? 0} empty line(s) will be ignored.`,
      'taskSettings.pool.remaining': 'Remaining',
      'taskSettings.pool.tooLong': 'Too long',
      'taskSettings.pool.tooLongHint': `One or more values exceed ${params?.max ?? 0} characters.`,
      'taskSettings.pool.valid': 'Valid',
    }[key] ?? key),
  }),
}))

const baseAnalysis: TemplatePoolAnalysis = {
  validCount: 2,
  emptyLineCount: 0,
  duplicateCount: 0,
  tooLongCount: 0,
  remaining: 498,
  overCapacity: false,
}

function mountPanel(analysis: Partial<TemplatePoolAnalysis> = {}, capacityMessage = '2 / 500 values used in this pool.') {
  return mount(TemplatePoolAnalysisPanel, {
    props: {
      analysis: { ...baseAnalysis, ...analysis },
      capacityMessage,
      maxValueLength: 2048,
    },
  })
}

describe('TemplatePoolAnalysisPanel', () => {
  it('renders existing pool counts and capacity without adding actions', () => {
    const wrapper = mountPanel()

    expect(wrapper.get('[data-testid="template-pool-analysis-panel"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="template-pool-analysis-panel"]').classes()).toContain('min-w-0')
    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('Valid')
    expect(wrapper.get('[data-testid="pool-valid"]').classes()).toContain('min-w-0')
    expect(wrapper.get('[data-testid="pool-valid"] p').attributes('title')).toBe('Valid')
    expect(wrapper.get('[data-testid="pool-valid"]').text()).toContain('2')
    expect(wrapper.get('[data-testid="pool-empty-lines"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="pool-duplicates"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="pool-too-long"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="pool-capacity"]').text()).toContain('2 / 500 values used')
    expect(wrapper.get('[data-testid="pool-capacity"]').attributes('title')).toBe('2 / 500 values used in this pool.')
    expect(wrapper.get('[data-testid="pool-capacity"]').classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.get('[data-testid="pool-capacity"]').attributes('role')).toBeUndefined()
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('renders over-capacity styling from the existing capacity state', () => {
    const wrapper = mountPanel({ overCapacity: true }, 'Too many values')

    const capacity = wrapper.get('[data-testid="pool-capacity"]')
    expect(capacity.text()).toContain('Too many values')
    expect(capacity.classes()).toContain('border-red-200')
    expect(capacity.attributes('role')).toBe('status')
    expect(capacity.attributes('aria-live')).toBe('polite')
    expect(capacity.attributes('aria-atomic')).toBe('true')
  })

  it('renders empty-line and duplicate hints from existing pool analysis', () => {
    const wrapper = mountPanel({ emptyLineCount: 2, duplicateCount: 1 })

    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').text()).toContain('2 empty line(s) will be ignored')
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').attributes('title')).toBe('2 empty line(s) will be ignored.')
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').attributes('role')).toBe('status')
    expect(wrapper.get('[data-testid="pool-empty-lines-hint"]').attributes('aria-live')).toBe('polite')
    const duplicateHint = wrapper.get('[data-testid="pool-duplicate-hint"]')
    expect(duplicateHint.attributes('title')).toBe('1 duplicate value(s) detected. You can deduplicate before saving.')
    expect(duplicateHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(duplicateHint.attributes('role')).toBe('status')
    expect(duplicateHint.attributes('aria-live')).toBe('polite')
    expect(duplicateHint.attributes('aria-atomic')).toBe('true')
    expect(wrapper.text()).toContain('1 duplicate value(s) detected')
  })

  it('prioritizes too-long warnings over duplicate hints like the original panel', () => {
    const wrapper = mountPanel({ duplicateCount: 1, tooLongCount: 1 })

    expect(wrapper.text()).toContain('One or more values exceed 2048 characters')
    const tooLongHint = wrapper.get('[data-testid="pool-too-long-hint"]')
    expect(tooLongHint.attributes('title')).toBe('One or more values exceed 2048 characters.')
    expect(tooLongHint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(tooLongHint.attributes('role')).toBe('status')
    expect(tooLongHint.attributes('aria-live')).toBe('polite')
    expect(tooLongHint.attributes('aria-atomic')).toBe('true')
    expect(wrapper.text()).not.toContain('duplicate value(s) detected')
  })
})
