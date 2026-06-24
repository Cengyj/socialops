import { describe, expect, it } from 'vitest'

import {
  accountImportCredentialSummary,
  buildAccountImportPreviewRows,
  buildAccountImportPreview,
  normalizeAccountImportRequest,
  parseSocialAccountImportTextRows,
  socialAccountImportDedupKey,
  socialAccountImportWorkbookRowsToText,
} from '../accountImportModel'

const messages = {
  duplicateMessage: 'duplicate',
  missingAccountMessage: 'missing account',
  missingPasswordMessage: 'missing password',
  missingCredentialMessage: 'missing credential',
}

describe('account import model', () => {
  it('summarizes visible credential types for the batch import preview', () => {
    const labels = {
      password: 'Password',
      twoFactor: '2FA',
      email: 'Email',
      authCookie: 'Auth cookie',
      executionAuth: 'Execution auth',
    }

    expect(accountImportCredentialSummary({
      platform: 'x_twitter',
      name: '@summary',
      password: '  account-secret  ',
      two_factor: 'totp',
      email: 'mail@example.test',
      auth_cookie: 'ct0=ok',
      execution_auth: 'encrypted-execution-auth-ciphertext',
    }, labels)).toBe('Password · 2FA · Email · Auth cookie · Execution auth')

    expect(accountImportCredentialSummary({
      platform: 'x_twitter',
      name: '@summary',
      password: '   ',
    }, { ...labels, fallback: 'None' })).toBe('None')
  })

  it('deduplicates Twitter/X imports only by normalized screen name', () => {
    expect(socialAccountImportDedupKey({ platform: 'twitter', name: '@jack', platform_user_id: '44196397' }))
      .toBe('x_twitter\x00username\x00jack')
    expect(socialAccountImportDedupKey({ platform: 'x', name: ' JACK ', platform_user_id: '999' }))
      .toBe('x_twitter\x00username\x00jack')
    expect(socialAccountImportDedupKey({ platform: 'X/Twitter', name: '  @CaseMix  ' }))
      .toBe('x_twitter\x00username\x00casemix')
    expect(socialAccountImportDedupKey({ platform: 'x_twitter', name: 'casemix' }))
      .toBe('x_twitter\x00username\x00casemix')
  })

  it('keeps auth cookie out of the duplicate identity key', () => {
    expect(socialAccountImportDedupKey({ platform: 'twitter', name: '@cookie', platform_user_id: '44196397' }))
      .toBe(socialAccountImportDedupKey({ platform: 'x', name: ' COOKIE ', platform_user_id: '99999999' }))
  })

  it('drops rest_id metadata from normalized import payloads', () => {
    const normalized = normalizeAccountImportRequest({
      platform: 'x_twitter',
      name: ' @Jack ',
      platform_user_id: '44196397',
      password: 'pw',
      auth_cookie: 'ct0=ok',
      default_proxy_snapshot: '{"id":999,"endpoint":"http://proxy.example:8080"}',
    })

    expect(normalized).toMatchObject({
      platform: 'x_twitter',
      name: '@Jack',
      password: 'pw',
      auth_cookie: 'ct0=ok',
    })
    expect(normalized.platform_user_id).toBeUndefined()
    expect('default_proxy_snapshot' in normalized).toBe(false)
  })

  it('preserves delivery field whitespace while trimming identity and contact fields', () => {
    const normalized = normalizeAccountImportRequest({
      platform: ' X/Twitter ',
      name: ' @ImportUser ',
      password: '  account-secret  ',
      email: '  mail@example.com  ',
      email_password: '  mail-secret  ',
      two_factor: '  totp-secret  ',
      backup_code: '  backup-code  ',
      email_client_id: '  mail-client  ',
      email_token: '  mail-token  ',
      auth_cookie: '  ct0=ok; auth_token=ok  ',
      execution_auth: '  encrypted-execution-auth-ciphertext  ',
      registration_ip: '  203.0.113.20  ',
      remark: '  operator note  ',
    })

    expect(normalized).toMatchObject({
      platform: 'x_twitter',
      name: '@ImportUser',
      password: '  account-secret  ',
      email: 'mail@example.com',
      email_password: '  mail-secret  ',
      two_factor: '  totp-secret  ',
      backup_code: '  backup-code  ',
      email_client_id: '  mail-client  ',
      email_token: '  mail-token  ',
      auth_cookie: '  ct0=ok; auth_token=ok  ',
      execution_auth: '  encrypted-execution-auth-ciphertext  ',
      registration_ip: '203.0.113.20',
      remark: '  operator note  ',
    })
  })

  it('accepts auth cookie as an import credential and still marks duplicate identities', () => {
    const preview = buildAccountImportPreview(
      [
        { platform: 'twitter', name: '@cookie', password: 'pw', auth_cookie: 'ct0=one; auth_token=one' },
        { platform: 'x', name: '@cookie', password: 'pw', auth_cookie: 'ct0=two; auth_token=two' },
      ],
      messages,
    )

    expect(preview.validRows.map(row => row.account.auth_cookie)).toEqual(['ct0=one; auth_token=one'])
    expect(preview.invalidRows.map(row => row.error)).toEqual(['duplicate'])
    expect(preview.duplicateCount).toBe(1)
  })

  it('marks local batch duplicates and rejects execution-auth-only credentials in preview', () => {
    const preview = buildAccountImportPreview(
      [
        { platform: 'twitter', name: '@one', password: 'pw', two_factor: 'totp' },
        { platform: 'x', name: '@one', password: 'pw', two_factor: 'totp' },
        { platform: 'x_twitter', name: '@two', password: 'pw', execution_auth: 'encrypted-only-auth' },
        { platform: 'x_twitter', name: '@three', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: 'encrypted-execution-auth-ciphertext' },
      ],
      messages,
    )

    expect(preview.validRows.map(row => row.account.name)).toEqual(['@one', '@three'])
    expect(preview.invalidRows.map(row => row.error)).toEqual(['duplicate', 'missing credential'])
    expect(preview.duplicateCount).toBe(1)
  })

  it('builds preview rows with preserved row numbers and workbench-ready statuses', () => {
    const rows = buildAccountImportPreviewRows(
      [
        { rowNumber: 4, account: { platform: 'twitter', name: '@one', password: 'pw', two_factor: 'totp' } },
        { rowNumber: 8, account: { platform: 'x', name: '@one', password: 'pw', two_factor: 'totp' } },
        { rowNumber: 11, account: { platform: 'x_twitter', name: '@bad', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: 'encrypted-partial-shaped-ciphertext' } },
      ],
      messages,
    )

    expect(rows).toEqual([
      {
        rowNumber: 4,
        account: { platform: 'x_twitter', name: '@one', password: 'pw', two_factor: 'totp' },
        valid: true,
        status: 'format_valid',
        error: '',
      },
      {
        rowNumber: 8,
        account: { platform: 'x_twitter', name: '@one', password: 'pw', two_factor: 'totp' },
        valid: false,
        status: 'batch_duplicate',
        error: 'duplicate',
      },
      {
        rowNumber: 11,
        account: { platform: 'x_twitter', name: '@bad', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: 'encrypted-partial-shaped-ciphertext' },
        valid: true,
        status: 'format_valid',
        error: '',
      },
    ])
  })

  it('parses text import rows with the workbench delivery-field order intact', () => {
    const executionAuth = 'encrypted-import-execution-auth'
    const rows = parseSocialAccountImportTextRows([
      'name\tpassword\ttwo_factor',
      [
        '@delivery_ops',
        '  account-secret  ',
        'JBSWY3DPEHPK3PXP',
        'backup-1',
        '  delivery@example.com  ',
        '  mail-secret  ',
        'mail-client',
        'mail-token',
        '  203.0.113.8  ',
        '  ct0=ok; auth_token=ok  ',
        `  ${executionAuth}  `,
        '+15550001111',
        '  delivery remark  ',
      ].join('\t'),
    ].join('\n'), 'twitter', messages)

    expect(rows).toEqual([{
      rowNumber: 2,
      account: {
        platform: 'x_twitter',
        name: '@delivery_ops',
        password: '  account-secret  ',
        phone: '+15550001111',
        email: 'delivery@example.com',
        email_password: '  mail-secret  ',
        auth_cookie: '  ct0=ok; auth_token=ok  ',
        execution_auth: `  ${executionAuth}  `,
        two_factor: 'JBSWY3DPEHPK3PXP',
        backup_code: 'backup-1',
        email_client_id: 'mail-client',
        email_token: 'mail-token',
        registration_ip: '203.0.113.8',
        remark: '  delivery remark  ',
      },
      valid: true,
      status: 'format_valid',
      error: '',
    }])
  })

  it('converts workbook rows with the fixed XLSX column order', () => {
    const text = socialAccountImportWorkbookRowsToText([
      ['密码', '账号', '2FA', '手机号', '邮箱账号', '邮箱密码', '邮箱 Client ID', '邮箱 Token'],
      ['@xlsx_delivery', '  account-secret  ', 'JBSWY3DPEHPK3PXP', '  +15550002222  ', '  delivery@example.com  ', '  mail-secret  ', 'mail-client', 'mail-token'],
    ])

    expect(text).toBe([
      '@xlsx_delivery',
      '  account-secret  ',
      'JBSWY3DPEHPK3PXP',
      '',
      '  delivery@example.com  ',
      '  mail-secret  ',
      'mail-client',
      'mail-token',
      '',
      '',
      '',
      '  +15550002222  ',
      '',
    ].join('\t'))

    expect(parseSocialAccountImportTextRows(text, 'x_twitter', messages)[0]?.account).toMatchObject({
      platform: 'x_twitter',
      name: '@xlsx_delivery',
      password: '  account-secret  ',
      phone: '+15550002222',
      email: 'delivery@example.com',
      email_password: '  mail-secret  ',
      two_factor: 'JBSWY3DPEHPK3PXP',
      email_client_id: 'mail-client',
      email_token: 'mail-token',
    })
  })
})
