import { describe, expect, it } from 'vitest'

import {
  MAX_TEMPLATE_POOL_VALUES,
  MAX_TEMPLATE_VALUE_LENGTH,
  analyzeTemplatePool,
  countIgnoredEmptyValues,
  normalizeTemplatePoolValues,
  splitContentValues,
  splitTargetValues,
  templatePoolValuesValid,
} from '../templatePool'

describe('template pool helpers', () => {
  it('splits target pools by newline or comma while trimming empty values', () => {
    expect(splitTargetValues(' @northwind, @socialops\n\n@third ')).toEqual([
      '@northwind',
      '@socialops',
      '@third',
    ])
  })

  it('splits content pools by newline without splitting commas inside post text', () => {
    expect(splitContentValues('hello, world\n\n second post ')).toEqual([
      'hello, world',
      'second post',
    ])
  })

  it('counts ignored empty values using the active pool delimiter rules', () => {
    expect(countIgnoredEmptyValues('@one,,\n  \n@two', 'targets')).toBe(3)
    expect(countIgnoredEmptyValues('hello\n\n  \nworld, still one post', 'contents')).toBe(2)
    expect(countIgnoredEmptyValues('hello\n\nworld', null)).toBe(0)
  })

  it('analyzes duplicate, too-long, remaining, and over-capacity state', () => {
    const tooLongValue = 'x'.repeat(MAX_TEMPLATE_VALUE_LENGTH + 1)
    expect(analyzeTemplatePool(['@one', '@one', tooLongValue], 2)).toEqual({
      validCount: 2,
      emptyLineCount: 2,
      duplicateCount: 1,
      tooLongCount: 1,
      remaining: MAX_TEMPLATE_POOL_VALUES - 3,
      overCapacity: false,
    })

    expect(analyzeTemplatePool(Array.from({ length: MAX_TEMPLATE_POOL_VALUES + 1 }, (_, index) => `@${index}`)))
      .toEqual(expect.objectContaining({ remaining: 0, overCapacity: true }))
  })

  it('normalizes saved template pool values before readiness checks', () => {
    expect(normalizeTemplatePoolValues([' @one ', '', null, '@two'])).toEqual(['@one', '@two'])
    expect(normalizeTemplatePoolValues(null)).toEqual([])
  })

  it('validates normalized pool size and item length limits', () => {
    expect(templatePoolValuesValid([' @one ', '@two'])).toBe(true)
    expect(templatePoolValuesValid(['x'.repeat(MAX_TEMPLATE_VALUE_LENGTH + 1)])).toBe(false)
    expect(templatePoolValuesValid(Array.from({ length: MAX_TEMPLATE_POOL_VALUES + 1 }, (_, index) => `@${index}`)))
      .toBe(false)
  })
})
