import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../AdminPaymentPlansView.vue'),
  'utf8'
)

describe('AdminPaymentPlansView production states', () => {
  it('renders load failures as a retryable error state before the table', () => {
    expect(source).toContain('const plansError = ref')
    expect(source).toContain('v-if="plansError"')
    expect(source).toContain("t('payment.admin.failedToLoadPlans')")
    expect(source.indexOf('v-if="plansError"')).toBeLessThan(source.indexOf('<DataTable'))
  })

  it('keeps raw plan API failures in diagnostics instead of rendering backend details', () => {
    expect(source).toContain("import { extractSafeApiErrorMessage } from '@/utils/apiError'")
    expect(source).toContain("import { recordClientDiagnostic } from '@/utils/clientDiagnostics'")
    expect(source).toContain("recordClientDiagnostic('admin.payment_plans.load'")
    expect(source).not.toContain('extractApiErrorMessage')
  })

  it('offers a first-plan creation action in the empty state', () => {
    expect(source).toContain('<template #empty>')
    expect(source).toContain("t('payment.admin.noPlansHint')")
    expect(source).toContain('@click="openPlanEdit(null)"')
  })

  it('uses branded platform logos in the platform column instead of text-only badges', () => {
    expect(source).toContain("import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'")
    expect(source).toContain('<SubscriptionPlatformLogo')
    expect(source).not.toContain("{{ platformLabel(row.platform || row.group_platform || '') }}")
  })
})
