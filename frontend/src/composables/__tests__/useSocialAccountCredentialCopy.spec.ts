import { describe, expect, it, beforeEach, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  copyTextToClipboard: vi.fn(),
  recordClientDiagnostic: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

vi.mock('@/utils/browser', () => ({
  copyTextToClipboard: mocks.copyTextToClipboard,
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic: mocks.recordClientDiagnostic,
}))

import { useSocialAccountCredentialCopy } from '../useSocialAccountCredentialCopy'

const account = {
  authCookie: '  auth-cookie=value  ',
  executionAuth: 'encrypted-execution-auth-ciphertext',
  emailToken: '  email-token  ',
}

function createCopyHelpers(getAccount = () => account) {
  return useSocialAccountCredentialCopy({
    getAccount,
    credentialDiagnosticContext: 'test.copy_credential',
    emailTokenDiagnosticContext: 'test.copy_email_token',
  })
}

describe('useSocialAccountCredentialCopy', () => {
  beforeEach(() => {
    mocks.copyTextToClipboard.mockReset()
    mocks.recordClientDiagnostic.mockReset()
    mocks.showSuccess.mockReset()
    mocks.showError.mockReset()
  })

  it('copies the selected raw credential without trimming stored delivery data', async () => {
    const { copySelectedCredential } = createCopyHelpers()

    await copySelectedCredential('authCookie')

    expect(mocks.copyTextToClipboard).toHaveBeenCalledWith('  auth-cookie=value  ')
    expect(mocks.showSuccess).toHaveBeenCalledWith('accountWorkbench.credentials.copied')
    expect(mocks.showError).not.toHaveBeenCalled()
  })

  it('copies the selected email token through the same success feedback', async () => {
    const { copySelectedEmailToken } = createCopyHelpers()

    await copySelectedEmailToken()

    expect(mocks.copyTextToClipboard).toHaveBeenCalledWith('  email-token  ')
    expect(mocks.showSuccess).toHaveBeenCalledWith('accountWorkbench.credentials.copied')
  })

  it('ignores missing or blank values without user-facing noise', async () => {
    const { copySelectedCredential, copySelectedEmailToken } = createCopyHelpers(() => ({
      authCookie: ' ',
      executionAuth: '',
      emailToken: '\n',
    }))

    await copySelectedCredential('authCookie')
    await copySelectedCredential('executionAuth')
    await copySelectedEmailToken()

    expect(mocks.copyTextToClipboard).not.toHaveBeenCalled()
    expect(mocks.showSuccess).not.toHaveBeenCalled()
    expect(mocks.showError).not.toHaveBeenCalled()
    expect(mocks.recordClientDiagnostic).not.toHaveBeenCalled()
  })

  it('records the supplied diagnostic context and shows copy failure feedback', async () => {
    const error = new Error('clipboard denied')
    mocks.copyTextToClipboard.mockRejectedValueOnce(error)
    const { copySelectedCredential } = createCopyHelpers()

    await copySelectedCredential('executionAuth')

    expect(mocks.recordClientDiagnostic).toHaveBeenCalledWith('test.copy_credential', error)
    expect(mocks.showError).toHaveBeenCalledWith('accountWorkbench.credentials.copyFailed')
  })
})
