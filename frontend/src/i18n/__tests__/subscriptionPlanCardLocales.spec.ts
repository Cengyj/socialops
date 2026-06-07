import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const quotaPeriodKeys = [
  'todayQuota',
  'thisWeekQuota',
  'thisMonthQuota',
] as const

describe('subscription plan card locale coverage', () => {
  it('localizes concrete quota period labels in zh and en', () => {
    for (const key of quotaPeriodKeys) {
      expect(zh.payment.planCard[key]).toBeTruthy()
      expect(en.payment.planCard[key]).toBeTruthy()
      expect(zh.payment.planCard[key]).not.toBe(`payment.planCard.${key}`)
      expect(en.payment.planCard[key]).not.toBe(`payment.planCard.${key}`)
    }
  })
})
