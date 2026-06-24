import type { SocialTaskResultRecord } from './socialTaskResultMessage'

const manualReviewResultMessages = new Set([
  '账号认证信息不可用，本次未扣费',
  '密码错误，本次未扣费',
  '账号需要额外验证，本次未扣费',
  '账号状态或频率受限，本次未扣费',
])

const proxyUnavailableResultMessages = new Set([
  '执行代理不可用，本次未扣费',
])

const failedResultMessages = new Set([
  '任务执行失败，本次未扣费',
  '任务执行超时，本次未扣费',
  '平台网络请求失败，本次未扣费',
  '该动作暂不支持，本次未扣费',
  '该平台动作暂不可用，本次未扣费',
  '执行已完成，但扣费确认异常，请联系管理员处理',
  '登录依赖服务未配置，本次未扣费',
  '任务参数不完整，本次未扣费',
  '头像图片尺寸必须为 400x400，本次未扣费',
  '背景图图片尺寸必须为 1500x500，本次未扣费',
  '任务媒体资源不可用，本次未扣费',
  '平台媒体上传失败，本次未扣费',
  '媒体引用暂未开放，本次未扣费',
  '视频发帖媒体暂未开放，本次未扣费',
  '发帖媒体类型暂不支持，本次未扣费',
  '执行目标不存在，本次未扣费',
  '内容或目标状态不符合平台要求，本次未扣费',
  '平台拒绝执行，本次未扣费',
])

const accountAttentionTaskStatuses = new Set([
  'manual_review',
  'ip_unavailable',
  'register_failed',
  'risk_rejected',
  'duplicate',
])

export function socialAccountTaskStatusFromTaskResult(row: SocialTaskResultRecord, currentTaskStatus?: string | null) {
  const status = String(row.status || '').trim().toLowerCase()
  if (status === 'pending' || status === 'running') return status
  if (status === 'success') return 'stored'
  if (status !== 'failed') return ''

  const message = String(row.result_message || '').trim()
  if (proxyUnavailableResultMessages.has(message)) return 'ip_unavailable'
  if (manualReviewResultMessages.has(message)) return 'manual_review'

  const current = String(currentTaskStatus || '').trim().toLowerCase()
  if (accountAttentionTaskStatuses.has(current)) return current
  if (failedResultMessages.has(message)) return 'failed'
  return 'stored'
}

export function socialAccountTaskStatusFromAccountSnapshot(currentTaskStatus?: string | null, currentTaskMessage?: string | null) {
  const current = String(currentTaskStatus || '').trim().toLowerCase()
  const message = String(currentTaskMessage || '').trim()
  if (current === 'stored' && proxyUnavailableResultMessages.has(message)) return 'ip_unavailable'
  if (current === 'stored' && manualReviewResultMessages.has(message)) return 'manual_review'
  if (current === 'stored' && failedResultMessages.has(message)) return 'failed'
  return current
}
