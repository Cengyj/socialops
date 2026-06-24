import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'accountWorkbench.credentials.empty') return 'Empty'
      if (key === 'accountWorkbench.credentials.length') return `${params?.count} chars`
      if (key === 'accountWorkbench.credentials.encryptedStored') return 'Encrypted value stored'
      if (key === 'accountWorkbench.credentials.rawCookieDetected') return 'Raw cookie detected'
      if (key === 'accountWorkbench.credentials.loginRefreshRequired') return 'Login refresh required'
      if (key === 'accountWorkbench.credentials.copyRaw') return `Copy ${params?.field}`
      return key
    },
  }),
}))

import { useSocialAccountCredentialPreview } from '../useSocialAccountCredentialPreview'

const deliveryFields = {
  password: 'account-secret',
  phone: '+15550001111',
  email: 'ops@example.test',
  emailPassword: 'mail-secret',
  twoFactor: 'JBSWY3DPEHPK3PXP',
  backupCode: 'backup-code',
  emailClientId: 'mail-client',
  emailToken: '  mail-token  ',
  authCookie: 'ct0=ok; auth_token=ok',
  executionAuth: 'encrypted-execution-auth-ciphertext',
}

describe('useSocialAccountCredentialPreview', () => {
  it('builds the shared delivery credential rows without account-page-specific drift', () => {
    const { buildDeliveryCredentialItems } = useSocialAccountCredentialPreview()

    const rows = buildDeliveryCredentialItems(deliveryFields, {
      emailTokenTestId: 'account-email-token-preview',
      emailTokenCopyTestId: 'account-email-token-copy',
    })

    expect(rows.map(row => row.key)).toEqual([
      'password',
      'phone',
      'email',
      'emailPassword',
      'twoFactor',
      'backupCode',
      'emailClientId',
      'emailToken',
    ])
    expect(rows.at(-1)).toMatchObject({
      key: 'emailToken',
      label: 'admin.socialAccountWorkbench.columns.emailToken',
      value: '10 chars',
      testId: 'account-email-token-preview',
      copyAction: 'emailToken',
      copyable: true,
      copyTitle: 'Copy admin.socialAccountWorkbench.columns.emailToken',
      copyTestId: 'account-email-token-copy',
    })
  })

  it('builds the shared auth cookie and execution auth preview cards', () => {
    const { buildCredentialPreviews, buildDeliveryCredentialItems } = useSocialAccountCredentialPreview()

    expect(buildDeliveryCredentialItems({ ...deliveryFields, emailToken: ' ' }, {
      emailTokenTestId: 'total-account-email-token-preview',
      emailTokenCopyTestId: 'total-account-email-token-copy',
    }).at(-1)?.copyable).toBe(false)

    const previews = buildCredentialPreviews(deliveryFields)

    expect(previews.map(preview => preview.key)).toEqual(['authCookie', 'executionAuth'])
    expect(previews[0].copyable).toBe(true)
    expect(previews[1].meta).toContain('Encrypted value stored')
  })

  it('summarizes execution auth without exposing the stored ciphertext in preview data', () => {
    const { buildCredentialPreviews } = useSocialAccountCredentialPreview()

    const previews = buildCredentialPreviews(deliveryFields)

    expect(previews[1].meta).toEqual([`${deliveryFields.executionAuth.length} chars`, 'Encrypted value stored'])
    expect(JSON.stringify(previews)).not.toContain(deliveryFields.executionAuth)
    expect(previews[1]).not.toHaveProperty('value')
  })

  it('keeps raw auth-cookie and missing execution-auth hints separate without leaking cookie text', () => {
    const { buildCredentialPreviews } = useSocialAccountCredentialPreview()
    const authCookie = 'ct0=raw-cookie; auth_token=raw-token'

    const previews = buildCredentialPreviews({
      ...deliveryFields,
      authCookie,
      executionAuth: '',
    })

    expect(previews[0].meta).toEqual([`${authCookie.length} chars`, 'Raw cookie detected'])
    expect(previews[1].meta).toEqual(['Empty', 'Login refresh required'])
    expect(JSON.stringify(previews)).not.toContain(authCookie)
    expect(JSON.stringify(previews)).not.toContain('raw-token')
  })
})
