export interface WorkbenchSearchAccount {
  id: number
  name: string
  platform: string
  username: string
  platformUserId: string
  password: string
  phone: string
  email: string
  emailPassword: string
  twoFactor: string
  backupCode: string
  emailClientId: string
  emailToken: string
  registrationIp: string
  authCookie: string
  taskMessage: string
  defaultProxySnapshot: string
  remark: string
}

export interface WorkbenchSearchLabels {
  accountStatus: string
  taskStatus: string
}

export function accountMatchesWorkbenchSearch(
  account: WorkbenchSearchAccount,
  keyword: string,
  labels: WorkbenchSearchLabels,
) {
  const normalizedKeyword = keyword.trim().toLowerCase()
  if (!normalizedKeyword) return true

  return accountWorkbenchSearchValues(account, labels).some(value =>
    value.toLowerCase().includes(normalizedKeyword),
  )
}

function accountWorkbenchSearchValues(account: WorkbenchSearchAccount, labels: WorkbenchSearchLabels) {
  return [
    String(account.id),
    `#${account.id}`,
    account.name,
    account.platform,
    account.username,
    account.platformUserId,
    account.password,
    account.phone,
    account.email,
    account.emailPassword,
    account.twoFactor,
    account.backupCode,
    account.emailClientId,
    account.emailToken,
    account.registrationIp,
    account.authCookie,
    labels.accountStatus,
    labels.taskStatus,
    account.taskMessage,
    account.defaultProxySnapshot,
    account.remark,
  ]
}
