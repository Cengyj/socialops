import { describe, expect, it } from 'vitest'
import {
  extractApiErrorMessage,
  extractI18nErrorMessage,
  extractSafeApiErrorMessage,
  extractSafeI18nErrorMessage,
} from '@/utils/apiError'

function makeT(existingKeys: Record<string, string> = {}) {
  const t = ((key: string, params?: Record<string, unknown>) => {
    const template = existingKeys[key]
    if (!template) return key
    return Object.entries(params ?? {}).reduce(
      (message, [paramKey, value]) => message.replace(`{${paramKey}}`, String(value)),
      template,
    )
  }) as ((key: string, params?: Record<string, unknown>) => string) & { te: (key: string) => boolean }

  t.te = (key: string) => Object.prototype.hasOwnProperty.call(existingKeys, key)
  return t
}

describe('extractI18nErrorMessage', () => {
  it('keeps the legacy raw fallback for admin and diagnostic surfaces', () => {
    const message = extractI18nErrorMessage(
      {
        response: {
          data: {
            detail: 'upstream payment token=secret failed',
          },
        },
      },
      makeT(),
      'payment.errors',
      'Payment failed',
    )

    expect(message).toBe('upstream payment token=secret failed')
  })
})

describe('extractSafeI18nErrorMessage', () => {
  it('uses localized code mappings with metadata interpolation', () => {
    const message = extractSafeI18nErrorMessage(
      {
        reason: 'TOO_MANY_PENDING',
        metadata: { max: 3 },
        message: 'raw queue detail',
      },
      makeT({
        'payment.errors.TOO_MANY_PENDING': 'You have {max} pending orders',
      }),
      'payment.errors',
      'Payment failed',
    )

    expect(message).toBe('You have 3 pending orders')
  })

  it('uses the fallback instead of exposing unknown backend or SDK messages', () => {
    const message = extractSafeI18nErrorMessage(
      {
        response: {
          data: {
            detail: 'provider returned token=secret',
            message: 'internal gateway traceback',
          },
        },
        message: 'Stripe card_declined raw message',
      },
      makeT(),
      'payment.errors',
      'Payment failed',
    )

    expect(message).toBe('Payment failed')
  })
})

describe('extractSafeApiErrorMessage', () => {
  it('maps known codes without reading raw messages', () => {
    expect(
      extractSafeApiErrorMessage(
        {
          response: {
            data: {
              code: 'INVALID_REDEEM_CODE',
              detail: 'redeem code database shard failed',
            },
          },
        },
        'Redeem failed',
        { INVALID_REDEEM_CODE: 'Invalid redeem code' },
      ),
    ).toBe('Invalid redeem code')
  })

  it('falls back for unknown errors while the legacy helper can still extract raw detail', () => {
    const error = {
      response: {
        data: {
          detail: 'wallet ledger stack trace token=secret',
        },
      },
    }

    expect(extractSafeApiErrorMessage(error, 'Operation failed')).toBe('Operation failed')
    expect(extractApiErrorMessage(error, 'Operation failed')).toBe('wallet ledger stack trace token=secret')
  })
})
