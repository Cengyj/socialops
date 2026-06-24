import { describe, expect, it } from 'vitest'

import {
  buildAccountEditPayload,
  buildTotalAccountEditPayload,
  preserveEditableDeliveryField,
  trimEditableField,
} from '../accountWorkbenchEditPayload'

describe('account workbench payload', () => {
  it('preserves account delivery field text while normalizing contact fields', () => {
    const payload = buildAccountEditPayload({
      password: '  account-secret  ',
      phone: '  +15550001111  ',
      email: '  operator@example.com  ',
      emailPassword: '  mail-secret  ',
      twoFactor: '  totp-secret  ',
      backupCode: '  backup-code  ',
      emailClientId: '  mail-client  ',
      emailToken: '  mail-token  ',
      registrationIp: '  203.0.113.10  ',
      authCookie: '  ct0=value; auth_token=value  ',
      executionAuth: '  encrypted-execution-auth-ciphertext  ',
      remark: '  operator note  ',
    })

    expect(payload).toEqual({
      password: '  account-secret  ',
      phone: '+15550001111',
      email: 'operator@example.com',
      email_password: '  mail-secret  ',
      two_factor: '  totp-secret  ',
      backup_code: '  backup-code  ',
      email_client_id: '  mail-client  ',
      email_token: '  mail-token  ',
      registration_ip: '203.0.113.10',
      auth_cookie: '  ct0=value; auth_token=value  ',
      execution_auth: '  encrypted-execution-auth-ciphertext  ',
      remark: '  operator note  ',
    })
  })

  it('submits only mutable account delivery and contact fields', () => {
    const payload = buildAccountEditPayload({
      password: 'secret',
      phone: '',
      email: '',
      registrationIp: '',
    })

    expect(Object.keys(payload).sort()).toEqual([
      'auth_cookie',
      'backup_code',
      'email',
      'email_client_id',
      'email_password',
      'email_token',
      'execution_auth',
      'password',
      'phone',
      'registration_ip',
      'remark',
      'two_factor',
    ])
    expect(JSON.stringify(payload)).not.toContain('name')
    expect(JSON.stringify(payload)).not.toContain('platform')
    expect(JSON.stringify(payload)).not.toContain('platform_user_id')
    expect(JSON.stringify(payload)).not.toContain('account_status')
    expect(JSON.stringify(payload)).not.toContain('task_status')
    expect(JSON.stringify(payload)).not.toContain('default_proxy_snapshot')
  })

  it('keeps blank delivery fields as explicit clear requests', () => {
    expect(preserveEditableDeliveryField('   ')).toBe('   ')
    expect(trimEditableField('   ')).toBe('')
    expect(buildAccountEditPayload({ password: 'secret', twoFactor: '   ' }).two_factor).toBe('   ')
  })

  it('adds the existing total-pool account status to admin edit payloads', () => {
    const payload = buildTotalAccountEditPayload({
      password: '  account-secret  ',
      phone: '  +15550001111  ',
      email: '  pool@example.test  ',
      emailPassword: '  mail-secret  ',
      accountStatus: '  available  ',
      registrationIp: '  203.0.113.10  ',
      authCookie: '  ct0=value; auth_token=value  ',
      executionAuth: '  encrypted-edit-execution-auth  ',
      remark: '  admin note  ',
    })

    expect(payload).toMatchObject({
      password: '  account-secret  ',
      phone: '+15550001111',
      email: 'pool@example.test',
      email_password: '  mail-secret  ',
      account_status: 'available',
      registration_ip: '203.0.113.10',
      auth_cookie: '  ct0=value; auth_token=value  ',
      execution_auth: '  encrypted-edit-execution-auth  ',
      remark: '  admin note  ',
    })
    expect(Object.keys(payload).sort()).toEqual([
      'account_status',
      'auth_cookie',
      'backup_code',
      'email',
      'email_client_id',
      'email_password',
      'email_token',
      'execution_auth',
      'password',
      'phone',
      'registration_ip',
      'remark',
      'two_factor',
    ])
  })
})
