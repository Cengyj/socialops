import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { copyTextToClipboard } from '@/utils/browser'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import type { SocialAccountCredentialPreviewKey } from '@/utils/socialAccountCredentials'

export interface SocialAccountCredentialCopyFields {
  authCookie: string
  executionAuth: string
  emailToken: string
}

export interface SocialAccountCredentialCopyOptions {
  getAccount: () => SocialAccountCredentialCopyFields | null | undefined
  credentialDiagnosticContext: string
  emailTokenDiagnosticContext: string
}

export function useSocialAccountCredentialCopy(options: SocialAccountCredentialCopyOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()

  async function copyRawValue(value: string, diagnosticContext: string) {
    if (!value.trim()) return
    try {
      await copyTextToClipboard(value)
      appStore.showSuccess(t('accountWorkbench.credentials.copied'))
    } catch (error) {
      recordClientDiagnostic(diagnosticContext, error)
      appStore.showError(t('accountWorkbench.credentials.copyFailed'))
    }
  }

  async function copySelectedCredential(key: SocialAccountCredentialPreviewKey) {
    const account = options.getAccount()
    if (!account) return
    await copyRawValue(
      key === 'authCookie' ? account.authCookie : account.executionAuth,
      options.credentialDiagnosticContext,
    )
  }

  async function copySelectedEmailToken() {
    await copyRawValue(options.getAccount()?.emailToken ?? '', options.emailTokenDiagnosticContext)
  }

  return {
    copySelectedCredential,
    copySelectedEmailToken,
  }
}
