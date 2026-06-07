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
  '头像图片尺寸必须为 400x400，本次未扣费': 'usage.taskResults.avatarSizeInvalid',
  '背景图图片尺寸必须为 1500x500，本次未扣费': 'usage.taskResults.bannerSizeInvalid',
  '任务媒体资源不可用，本次未扣费': 'usage.taskResults.mediaAssetUnavailable',
  '平台媒体上传失败，本次未扣费': 'usage.taskResults.mediaUploadFailed',
  '媒体引用暂未开放，本次未扣费': 'usage.taskResults.mediaSourceUnsupported',
  '视频发帖媒体暂未开放，本次未扣费': 'usage.taskResults.postVideoUnavailable',
  '发帖媒体类型暂不支持，本次未扣费': 'usage.taskResults.postMediaTypeUnsupported',
  '执行目标不存在，本次未扣费': 'usage.taskResults.targetMissing',
  '账号需要额外验证，本次未扣费': 'usage.taskResults.challengeRequired',
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
