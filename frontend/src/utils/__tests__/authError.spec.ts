import { describe, expect, it } from 'vitest'
import { buildAuthErrorMessage, buildSafeAuthErrorMessage } from '@/utils/authError'

describe('buildAuthErrorMessage', () => {
  it('prefers response detail message when available', () => {
    const message = buildAuthErrorMessage(
      {
        response: {
          data: {
            detail: 'detailed message',
            message: 'plain message'
          }
        },
      },
      { fallback: 'fallback' }
    )
    expect(message).toBe('detailed message')
  })

  it('falls back to response message when detail is unavailable', () => {
    const message = buildAuthErrorMessage(
      {
        response: {
          data: {
            message: 'plain message'
          }
        },
      },
      { fallback: 'fallback' }
    )
    expect(message).toBe('plain message')
  })

  it('falls back to error.message when response payload is unavailable', () => {
    const message = buildAuthErrorMessage(
      {
        message: 'error message'
      },
      { fallback: 'fallback' }
    )
    expect(message).toBe('error message')
  })

  it('uses fallback when no message can be extracted', () => {
    expect(buildAuthErrorMessage({}, { fallback: 'fallback' })).toBe('fallback')
  })
})

describe('buildSafeAuthErrorMessage', () => {
  it('maps known auth error codes to local user-facing messages', () => {
    const message = buildSafeAuthErrorMessage(
      {
        response: {
          data: {
            code: 'INVALID_RESET_TOKEN',
            detail: 'stack trace: token secret expired'
          }
        }
      },
      {
        fallback: 'Reset failed',
        messages: {
          INVALID_RESET_TOKEN: 'Invalid or expired token'
        }
      }
    )

    expect(message).toBe('Invalid or expired token')
  })

  it('uses fallback instead of exposing raw backend details for unknown errors', () => {
    const message = buildSafeAuthErrorMessage(
      {
        response: {
          data: {
            detail: 'upstream smtp token=secret failed'
          }
        },
        message: 'authorization: Bearer secret'
      },
      { fallback: 'Please try again later' }
    )

    expect(message).toBe('Please try again later')
  })

  it('maps response reason and error fields without reading raw messages', () => {
    expect(
      buildSafeAuthErrorMessage(
        { response: { data: { reason: 'REGISTRATION_DISABLED', message: 'raw backend message' } } },
        {
          fallback: 'fallback',
          messages: {
            REGISTRATION_DISABLED: 'Registration is closed'
          }
        }
      )
    ).toBe('Registration is closed')

    expect(
      buildSafeAuthErrorMessage(
        { response: { data: { error: 'invalid_state', detail: 'state=secret' } } },
        {
          fallback: 'fallback',
          messages: {
            invalid_state: 'Invalid state'
          }
        }
      )
    ).toBe('Invalid state')
  })
})
