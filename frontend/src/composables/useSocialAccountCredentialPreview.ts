import { useI18n } from 'vue-i18n'
import {
  buildSocialAccountCredentialPreview,
  socialAccountCredentialCharacterCountText,
} from '@/utils/socialAccountCredentials'
import type {
  SocialAccountCredentialPreviewKey,
  SocialAccountCredentialPreviewText,
} from '@/utils/socialAccountCredentials'

export interface SocialAccountDeliveryCredentialFields {
  password: string
  phone: string
  email: string
  emailPassword: string
  twoFactor: string
  backupCode: string
  emailClientId: string
  emailToken: string
  authCookie: string
  executionAuth: string
}

export interface SocialAccountDeliveryCredentialItem {
  key: string
  label: string
  value: string
  testId?: string
  copyAction?: 'emailToken'
  copyable?: boolean
  copyTitle?: string
  copyTestId?: string
}

export interface SocialAccountDeliveryCredentialOptions {
  emailTokenTestId: string
  emailTokenCopyTestId: string
}

export function useSocialAccountCredentialPreview() {
  const { t } = useI18n()

  function previewText(): SocialAccountCredentialPreviewText {
    return {
      empty: t('accountWorkbench.credentials.empty'),
      length: count => t('accountWorkbench.credentials.length', { count }),
      encryptedStored: t('accountWorkbench.credentials.encryptedStored'),
      oauthReady: t('accountWorkbench.credentials.oauthReady'),
      rawCookieDetected: t('accountWorkbench.credentials.rawCookieDetected'),
      oauthPartial: t('accountWorkbench.credentials.oauthPartial'),
      jsonDetected: t('accountWorkbench.credentials.jsonDetected'),
      loginRefreshRequired: t('accountWorkbench.credentials.loginRefreshRequired'),
    }
  }

  function credentialCharacterCountText(rawValue?: string | null) {
    return socialAccountCredentialCharacterCountText(rawValue, previewText())
  }

  function buildCredentialPreview(key: SocialAccountCredentialPreviewKey, raw: string, authCookie = '') {
    const label = key === 'authCookie'
      ? t('admin.socialAccountWorkbench.columns.authCookie')
      : t('admin.socialAccountWorkbench.columns.executionAuth')
    return buildSocialAccountCredentialPreview({
      key,
      raw,
      authCookie,
      label,
      description: t(`accountWorkbench.credentials.${key}Description`),
      copyTitle: t('accountWorkbench.credentials.copyRaw', { field: label }),
      text: previewText(),
    })
  }

  function buildDeliveryCredentialItems(
    account: SocialAccountDeliveryCredentialFields,
    options: SocialAccountDeliveryCredentialOptions,
  ): SocialAccountDeliveryCredentialItem[] {
    const emailToken = account.emailToken
    const emailTokenLabel = t('admin.socialAccountWorkbench.columns.emailToken')
    return [
      { key: 'password', label: t('admin.socialAccountWorkbench.columns.password'), value: account.password },
      { key: 'phone', label: t('admin.socialAccountWorkbench.columns.phone'), value: account.phone },
      { key: 'email', label: t('admin.socialAccountWorkbench.columns.email'), value: account.email },
      { key: 'emailPassword', label: t('admin.socialAccountWorkbench.columns.emailPassword'), value: account.emailPassword },
      { key: 'twoFactor', label: t('admin.socialAccountWorkbench.columns.twoFactor'), value: account.twoFactor },
      { key: 'backupCode', label: t('admin.socialAccountWorkbench.columns.backupCode'), value: account.backupCode },
      { key: 'emailClientId', label: t('admin.socialAccountWorkbench.columns.emailClientId'), value: account.emailClientId },
      {
        key: 'emailToken',
        label: emailTokenLabel,
        value: credentialCharacterCountText(emailToken),
        testId: options.emailTokenTestId,
        copyAction: 'emailToken',
        copyable: emailToken.trim() !== '',
        copyTitle: t('accountWorkbench.credentials.copyRaw', { field: emailTokenLabel }),
        copyTestId: options.emailTokenCopyTestId,
      },
    ]
  }

  function buildCredentialPreviews(account: Pick<SocialAccountDeliveryCredentialFields, 'authCookie' | 'executionAuth'>) {
    return [
      buildCredentialPreview('authCookie', account.authCookie),
      buildCredentialPreview('executionAuth', account.executionAuth, account.authCookie),
    ]
  }

  return {
    buildCredentialPreview,
    buildCredentialPreviews,
    buildDeliveryCredentialItems,
    credentialCharacterCountText,
  }
}
