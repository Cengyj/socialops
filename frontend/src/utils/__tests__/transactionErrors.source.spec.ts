import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

const root = resolve(__dirname, '../..')

const checkedEntries = [
  resolve(root, 'views/user/PaymentView.vue'),
  resolve(root, 'views/user/paymentUx.ts'),
  resolve(root, 'views/user/PaymentQRCodeView.vue'),
  resolve(root, 'views/user/StripePaymentView.vue'),
  resolve(root, 'views/user/StripePopupView.vue'),
  resolve(root, 'views/user/AirwallexPaymentView.vue'),
  resolve(root, 'views/user/UserOrdersView.vue'),
  resolve(root, 'views/user/RedeemView.vue'),
  resolve(root, 'views/user/AffiliateView.vue'),
  resolve(root, 'views/user/SubscriptionsView.vue'),
  resolve(root, 'components/payment'),
]

function collectSourceFiles(path: string): string[] {
  if (statSync(path).isFile()) return [path]
  return readdirSync(path)
    .flatMap((entry: string) => {
      const fullPath = join(path, entry)
      if (entry === '__tests__') return []
      if (statSync(fullPath).isDirectory()) return collectSourceFiles(fullPath)
      return /\.(vue|ts)$/.test(entry) ? [fullPath] : []
    })
}

describe('transaction-facing user errors', () => {
  it('does not show raw backend or SDK messages on payment, wallet, order, redeem, or affiliate surfaces', () => {
    const forbiddenPatterns = [
      /response\?\.data\?\.detail/,
      /response\?\.data\?\.message/,
      /\berr\.message\b/,
      /\berror\.message\b/,
      /extractApiErrorMessage\(/,
      /extractI18nErrorMessage\(/,
    ]

    const offenders = checkedEntries
      .flatMap(collectSourceFiles)
      .filter((file) => {
        const source = readFileSync(file, 'utf8')
        return forbiddenPatterns.some((pattern) => pattern.test(source))
      })

    expect(offenders.map((file) => relative(root, file))).toEqual([])
  })
})
