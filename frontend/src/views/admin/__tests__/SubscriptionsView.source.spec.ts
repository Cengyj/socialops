import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SubscriptionsView.vue')
const source = readFileSync(viewPath, 'utf8')

describe('admin SubscriptionsView quota display contract', () => {
  it('renders one canonical quota period instead of daily, weekly, and monthly rows together', () => {
    expect(source).toContain('getSubscriptionQuotaUsage')
    expect(source).toContain('subscriptionQuotaUsage(row)')
    expect(source).not.toContain('v-if="subscriptionDailyLimit(row) !== null"')
    expect(source).not.toContain('v-if="subscriptionWeeklyLimit(row) !== null"')
    expect(source).not.toContain('v-if="subscriptionMonthlyLimit(row) !== null"')
  })

  it('uses the same active quota amount in the package badge as the usage cell', () => {
    expect(source).toContain('subscriptionQuotaLimit(row)')
    expect(source).not.toContain(':quota-display="formatLimitValue(subscriptionMonthlyLimit(row))"')
  })

  it('labels create-form package quota by the selected package reset period', () => {
    expect(source).toContain('selectedCreatePlanQuotaLabel')
    expect(source).toContain('getPlanQuotaPeriod')
    expect(source).not.toContain("{{ t('payment.planCard.periodQuota') }}")
  })

  it('uses branded platform logos in subscription package selectors instead of text-only chips', () => {
    expect(source).toContain("import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'")
    expect(source).toContain('<SubscriptionPlatformLogo')
    expect(source).not.toContain("{{ platformLabel(String(option.platform || 'social')) }}")
  })
})
