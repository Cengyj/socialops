import { describe, expect, it } from 'vitest'

import { accountMatchesWorkbenchSearch, type WorkbenchSearchAccount } from '../accountWorkbenchSearch'

const baseAccount: WorkbenchSearchAccount = {
  id: 42,
  name: '@northwind_ops',
  platform: 'x_twitter',
  username: 'northwind_ops',
  platformUserId: 'rest-id-42',
  password: 'account-password',
  phone: '+15550001111',
  email: 'mail@example.com',
  emailPassword: 'mailbox-password',
  twoFactor: 'totp-secret',
  backupCode: 'backup-code',
  emailClientId: 'mail-client-id',
  emailToken: 'mail-token',
  registrationIp: '203.0.113.10',
  authCookie: 'ct0=cookie-value; auth_token=auth-cookie-token',
  executionAuth: 'encrypted-execution-auth-ciphertext',
  taskMessage: 'ready for execution',
  defaultProxySnapshot: '{"endpoint":"http://proxy.example:8080"}',
  remark: 'operator note',
}

describe('accountMatchesWorkbenchSearch', () => {
  it('matches account identity, status labels, and delivery fields', () => {
    const labels = {
      accountStatus: 'Available',
      taskStatus: 'Stored',
    }

    for (const keyword of [
      'northwind',
      '#42',
      'X_TWITTER',
      'rest-id-42',
      'account-password',
      '+15550001111',
      'mailbox-password',
      'totp-secret',
      'backup-code',
      'mail-client-id',
      'mail-token',
      '203.0.113.10',
      'auth-cookie-token',
      'Available',
      'Stored',
      'ready for execution',
      'proxy.example',
      'operator note',
    ]) {
      expect(accountMatchesWorkbenchSearch(baseAccount, keyword, labels)).toBe(true)
    }
  })

  it('treats blank search as a match and rejects unrelated keywords', () => {
    expect(accountMatchesWorkbenchSearch(baseAccount, '   ', { accountStatus: '', taskStatus: '' })).toBe(true)
    expect(accountMatchesWorkbenchSearch(baseAccount, 'encrypted-execution-auth-ciphertext', { accountStatus: 'Available', taskStatus: 'Stored' })).toBe(false)
    expect(accountMatchesWorkbenchSearch(baseAccount, 'not-present', { accountStatus: 'Available', taskStatus: 'Stored' })).toBe(false)
  })
})
