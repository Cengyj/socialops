export interface SocialTaskResultRecord {
  status?: string | null
  result_message?: string | null
}

type Translate = (key: string) => string

const backendSafeResultKeys: Record<string, string> = {
  '任务已完成，详细结果已隐藏': 'usage.taskResults.completedHidden',
  '任务执行失败，本次未扣费': 'usage.taskResults.failedNoCharge',
  '任务队列繁忙，本次未扣费': 'usage.taskResults.queueBusy',
  '该平台动作暂不可用，本次未扣费': 'usage.taskResults.platformUnavailable',
  '执行已完成，但扣费确认异常，请联系管理员处理': 'usage.taskResults.billingAnomaly',
  '账号认证信息不可用，本次未扣费': 'usage.taskResults.authUnavailable',
  '执行代理不可用，本次未扣费': 'usage.taskResults.proxyUnavailable',
  '该动作暂不支持，本次未扣费': 'usage.taskResults.unsupportedAction',
  '任务参数不完整，本次未扣费': 'usage.taskResults.invalidParams',
  '执行目标不存在，本次未扣费': 'usage.taskResults.targetMissing',
  '账号状态或频率受限，本次未扣费': 'usage.taskResults.accountLimited',
  '内容或目标状态不符合平台要求，本次未扣费': 'usage.taskResults.contentRejected',
  '平台拒绝执行，本次未扣费': 'usage.taskResults.accessDenied',
}

export function formatSocialTaskResultMessage(row: SocialTaskResultRecord, translate: Translate) {
  const message = String(row.result_message || '').trim()
  if (!message) return '-'

  const localizedKey = backendSafeResultKeys[message]
  if (localizedKey) {
    const translated = translate(localizedKey)
    return translated === localizedKey ? message : translated
  }

  if (String(row.status || '').toLowerCase() === 'failed' && hasUnsafeDiagnosticDetail(message)) {
    return translate('usage.safeResult')
  }

  return message
}

function hasUnsafeDiagnosticDetail(message: string) {
  const normalized = message.toLowerCase()
  return [
    'http://',
    'https://',
    'authorization',
    'bearer ',
    'token',
    'secret',
    'cookie',
    'password',
    'proxy',
    'trace',
    'trace_id',
    '127.0.0.1',
  ].some(marker => normalized.includes(marker))
}
