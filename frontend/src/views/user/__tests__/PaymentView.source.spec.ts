import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../PaymentView.vue')
const source = readFileSync(viewPath, 'utf8')

describe('user PaymentView subscription quota display contract', () => {
  it('labels the selected plan quota by its concrete reset period', () => {
    expect(source).toContain('selectedPlanQuotaLabel')
    expect(source).toContain('getPlanQuotaPeriod')
    expect(source).not.toContain("{{ t('payment.planCard.periodQuota') }}")
  })
})
