import { describe, expect, it } from 'vitest'

import { adminPaymentAPI } from '@/api/admin/payment'

describe('admin payment api contract', () => {
  it('does not expose unregistered admin channel management endpoints', () => {
    expect(('get' + 'Channels') in adminPaymentAPI).toBe(false)
    expect(('create' + 'Channel') in adminPaymentAPI).toBe(false)
    expect(('update' + 'Channel') in adminPaymentAPI).toBe(false)
    expect(('delete' + 'Channel') in adminPaymentAPI).toBe(false)
  })
})
