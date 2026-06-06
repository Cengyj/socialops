import { describe, expect, it } from 'vitest'

import {
  validateAnnouncementTargeting,
  type AnnouncementTargetingValidationError
} from '../announcementTargeting'
import type { AnnouncementTargeting } from '@/types'

describe('validateAnnouncementTargeting', () => {
  it('accepts empty targeting as all users', () => {
    expect(validateAnnouncementTargeting({ any_of: [] })).toBeNull()
    expect(validateAnnouncementTargeting(undefined)).toBeNull()
  })

  it.each<[string, AnnouncementTargeting, AnnouncementTargetingValidationError]>([
    [
      'empty OR group',
      { any_of: [{ all_of: [] }] },
      'empty_condition_group'
    ],
    [
      'subscription condition without package groups',
      { any_of: [{ all_of: [{ type: 'subscription', operator: 'in', group_ids: [] }] }] },
      'subscription_group_required'
    ],
    [
      'subscription condition with invalid operator',
      { any_of: [{ all_of: [{ type: 'subscription', operator: 'eq', group_ids: [1] }] }] },
      'invalid_subscription_operator'
    ],
    [
      'balance condition with invalid operator',
      { any_of: [{ all_of: [{ type: 'balance', operator: 'in', value: 10 }] }] },
      'invalid_balance_operator'
    ],
  ])('rejects %s', (_name, targeting, expected) => {
    expect(validateAnnouncementTargeting(targeting)).toBe(expected)
  })

  it('accepts valid subscription and balance conditions', () => {
    expect(validateAnnouncementTargeting({
      any_of: [
        {
          all_of: [
            { type: 'subscription', operator: 'in', group_ids: [1, 2] },
            { type: 'balance', operator: 'gte', value: 10 },
          ],
        },
      ],
    })).toBeNull()
  })
})
