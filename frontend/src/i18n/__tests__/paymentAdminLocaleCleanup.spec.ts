import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

function paymentAdminMessages(locale: typeof en | typeof zh) {
  return locale.payment.admin as Record<string, unknown>
}

describe('payment admin locale cleanup', () => {
  it('uses provider-instance wording for payment routing', () => {
    expect(en.admin.settings.payment.loadBalanceLeastAmount).toBe('Least consumed provider first')
    expect(zh.admin.settings.payment.loadBalanceLeastAmount).toBe('优先低消耗服务商')
  })

  it('does not keep removed payment channel management labels', () => {
    for (const messages of [paymentAdminMessages(en), paymentAdminMessages(zh)]) {
      expect(messages.tabs as Record<string, unknown>).not.toHaveProperty('channels')
      expect(messages).not.toHaveProperty('channelName')
      expect(messages).not.toHaveProperty('channelDescription')
      expect(messages).not.toHaveProperty('createChannel')
      expect(messages).not.toHaveProperty('editChannel')
      expect(messages).not.toHaveProperty('deleteChannel')
      expect(messages).not.toHaveProperty('deleteChannelConfirm')
    }
  })
})
