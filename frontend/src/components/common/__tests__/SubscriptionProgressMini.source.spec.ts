import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SubscriptionProgressMini.vue')
const source = readFileSync(componentPath, 'utf8')

describe('SubscriptionProgressMini quota display contract', () => {
  it('shows only the active reset-period quota instead of mixing guardrail rows', () => {
    expect(source).toContain('getSubscriptionQuotaUsage')
    expect(source).not.toContain("quotaPeriod(subscription) !== 'daily'")
    expect(source).not.toContain("quotaPeriod(subscription) !== 'weekly'")
    expect(source).not.toContain('subscription.daily_usage_usd, dailyLimit(subscription)')
    expect(source).not.toContain('subscription.weekly_usage_usd, weeklyLimit(subscription)')
  })
})
