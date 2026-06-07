import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SubscriptionsView.vue')
const source = readFileSync(viewPath, 'utf8')

describe('user SubscriptionsView source contract', () => {
  it('does not show daily or weekly guardrail progress in the main user subscription cards', () => {
    expect(source).not.toContain('<!-- Daily guardrail -->')
    expect(source).not.toContain('<!-- Weekly guardrail -->')
    expect(source).not.toContain("quotaPeriod(subscription) !== 'daily'")
    expect(source).not.toContain("quotaPeriod(subscription) !== 'weekly'")
  })

  it('labels the visible quota with the concrete reset period', () => {
    expect(source).toContain('quotaUsageLabel(subscription)')
    expect(source).not.toContain("{{ t('payment.planCard.periodQuota') }}")
  })

  it('does not render English duration unit fallbacks in localized subscription reset text', () => {
    expect(source).not.toContain('`${parts.days}d ${parts.hours}h`')
    expect(source).not.toContain('`${parts.hours}h ${parts.minutes}m`')
    expect(source).not.toContain('`${parts.minutes}m`')
  })

  it('keeps subscription load failures visible with retry instead of falling through to empty state', () => {
    expect(source).not.toContain('console.error')
    expect(source).toContain('loadError')
    expect(source).toContain('@click="loadSubscriptions"')
    expect(source.indexOf('v-else-if="loadError"')).toBeLessThan(source.indexOf('v-else-if="subscriptions.length === 0"'))
  })

  it('uses branded platform logos instead of text-only platform markers', () => {
    expect(source).toContain("import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'")
    expect(source).toContain('<SubscriptionPlatformLogo')
    expect(source).not.toContain("{{ platformLabel(subscriptionPlatform(subscription)) }}")
  })
})
