import type { UpdateMySocialAccountRequest, UpdateSocialAccountRequest } from '@/api/accountWorkbench'

export interface AccountWorkbenchEditFormLike {
  password?: string | null
  phone?: string | null
  email?: string | null
  emailPassword?: string | null
  twoFactor?: string | null
  backupCode?: string | null
  emailClientId?: string | null
  emailToken?: string | null
  registrationIp?: string | null
  authCookie?: string | null
  executionAuth?: string | null
  remark?: string | null
}

export interface TotalAccountEditFormLike extends AccountWorkbenchEditFormLike {
  accountStatus?: string | null
}

export function buildAccountEditPayload(form: AccountWorkbenchEditFormLike): UpdateMySocialAccountRequest {
  return {
    password: preserveEditableDeliveryField(form.password),
    phone: trimEditableField(form.phone),
    email: trimEditableField(form.email),
    email_password: preserveEditableDeliveryField(form.emailPassword),
    two_factor: preserveEditableDeliveryField(form.twoFactor),
    backup_code: preserveEditableDeliveryField(form.backupCode),
    email_client_id: preserveEditableDeliveryField(form.emailClientId),
    email_token: preserveEditableDeliveryField(form.emailToken),
    registration_ip: trimEditableField(form.registrationIp),
    auth_cookie: preserveEditableDeliveryField(form.authCookie),
    execution_auth: preserveEditableDeliveryField(form.executionAuth),
    remark: preserveEditableDeliveryField(form.remark),
  }
}

export function buildTotalAccountEditPayload(form: TotalAccountEditFormLike): UpdateSocialAccountRequest {
  return {
    ...buildAccountEditPayload(form),
    account_status: trimEditableField(form.accountStatus),
  }
}

export function trimEditableField(value?: string | null) {
  return String(value ?? '').trim()
}

export function preserveEditableDeliveryField(value?: string | null) {
  return String(value ?? '')
}
