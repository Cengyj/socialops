import { describe, expect, it } from 'vitest'

import {
  buildAccountImportPreviewRows,
  buildAccountImportPreview,
  normalizeAccountImportRequest,
  socialAccountImportDedupKey,
  validateTwitterExecutionAuth,
} from '../accountImportModel'

describe('account import model', () => {
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

  it('accepts auth cookie as an import credential and still marks duplicate identities', () => {
    const preview = buildAccountImportPreview(
      [
        { platform: 'twitter', name: '@cookie', password: 'pw', auth_cookie: 'ct0=one; auth_token=one' },
        { platform: 'x', name: '@cookie', password: 'pw', auth_cookie: 'ct0=two; auth_token=two' },
      ],
      {
        duplicateMessage: 'duplicate',
        missingAccountMessage: 'missing account',
        missingPasswordMessage: 'missing password',
        missingCredentialMessage: 'missing credential',
        invalidExecutionAuthMessage: 'invalid execution auth',
      },
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
        { platform: 'x_twitter', name: '@two', password: 'pw', execution_auth: '{"access_token":"a"}' },
        { platform: 'x_twitter', name: '@three', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: '{"access_token":"a","token_secret":"b"}' },
      ],
      {
        duplicateMessage: 'duplicate',
        missingAccountMessage: 'missing account',
        missingPasswordMessage: 'missing password',
        missingCredentialMessage: 'missing credential',
        invalidExecutionAuthMessage: 'invalid execution auth',
      },
    )

    expect(preview.validRows.map(row => row.account.name)).toEqual(['@one', '@three'])
    expect(preview.invalidRows.map(row => row.error)).toEqual(['duplicate', 'missing credential'])
    expect(preview.duplicateCount).toBe(1)
  })

  it('accepts JSON and base64 JSON execution credentials with access token and token secret', () => {
    const raw = '{"access_token":"a","token_secret":"b"}'
    const base64 = btoa(raw)

    expect(validateTwitterExecutionAuth(raw)).toBe(true)
    expect(validateTwitterExecutionAuth(base64)).toBe(true)
    expect(validateTwitterExecutionAuth('{"access_token":"a"}')).toBe(false)
    expect(validateTwitterExecutionAuth('not-json')).toBe(false)
  })

  it('builds preview rows with preserved row numbers and workbench-ready statuses', () => {
    const rows = buildAccountImportPreviewRows(
      [
        { rowNumber: 4, account: { platform: 'twitter', name: '@one', password: 'pw', two_factor: 'totp' } },
        { rowNumber: 8, account: { platform: 'x', name: '@one', password: 'pw', two_factor: 'totp' } },
        { rowNumber: 11, account: { platform: 'x_twitter', name: '@bad', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: '{"access_token":"a"}' } },
      ],
      {
        duplicateMessage: 'duplicate',
        missingAccountMessage: 'missing account',
        missingPasswordMessage: 'missing password',
        missingCredentialMessage: 'missing credential',
        invalidExecutionAuthMessage: 'invalid execution auth',
      },
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
        account: { platform: 'x_twitter', name: '@bad', password: 'pw', auth_cookie: 'ct0=ok', execution_auth: '{"access_token":"a"}' },
        valid: false,
        status: 'needs_data',
        error: 'invalid execution auth',
      },
    ])
  })
})
