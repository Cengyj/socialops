import { describe, expect, it } from 'vitest'

import { formatSafeUserTaskResult } from '../socialTaskResult'
import { formatSocialTaskResultMessage } from '../socialTaskResultMessage'

describe('formatSafeUserTaskResult', () => {
  it('keeps concise business results readable', () => {
    expect(formatSafeUserTaskResult('follow succeeded', 'hidden')).toBe('follow succeeded')
  })

  it('hides internal diagnostic and sensitive response details', () => {
    const raw = 'upstream response body authorization Bearer abc token=secret proxy http://user:pass@127.0.0.1:8080 trace_id=trace-123'

    expect(formatSafeUserTaskResult(raw, 'Task failed; details hidden')).toBe('Task failed; details hidden')
  })

  it('normalizes whitespace and caps long results', () => {
    const raw = `ok ${'x'.repeat(200)}`

    expect(formatSafeUserTaskResult(raw)).toHaveLength(163)
    expect(formatSafeUserTaskResult('  ok \n done  ')).toBe('ok done')
  })

  it('maps exact avatar and banner dimension safe results through localization keys', () => {
    const messages: Record<string, string> = {
      'usage.taskResults.avatarSizeInvalid': 'Avatar image must be 400 × 400; not charged',
      'usage.taskResults.bannerSizeInvalid': 'Banner image must be 1500 × 500; not charged',
      'usage.taskResults.mediaAssetUnavailable': 'Task media asset is unavailable; not charged',
      'usage.taskResults.mediaUploadFailed': 'Platform media upload failed; not charged',
      'usage.taskResults.mediaSourceUnsupported': 'Media references are not available yet; not charged',
      'usage.taskResults.postVideoUnavailable': 'Video post media is not available yet; not charged',
      'usage.taskResults.postMediaTypeUnsupported': 'Post media type is not supported yet; not charged',
      'usage.taskResults.challengeRequired': 'Account requires additional verification; not charged',
    }
    const t = (key: string) => messages[key] ?? key

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '头像图片尺寸必须为 400x400，本次未扣费',
    }, t)).toBe('Avatar image must be 400 × 400; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '背景图图片尺寸必须为 1500x500，本次未扣费',
    }, t)).toBe('Banner image must be 1500 × 500; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '任务媒体资源不可用，本次未扣费',
    }, t)).toBe('Task media asset is unavailable; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '平台媒体上传失败，本次未扣费',
    }, t)).toBe('Platform media upload failed; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '平台媒体上传失败，本次未扣费',
    }, t)).toBe('Platform media upload failed; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '媒体引用暂未开放，本次未扣费',
    }, t)).toBe('Media references are not available yet; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '视频发帖媒体暂未开放，本次未扣费',
    }, t)).toBe('Video post media is not available yet; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '发帖媒体类型暂不支持，本次未扣费',
    }, t)).toBe('Post media type is not supported yet; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '账号需要额外验证，本次未扣费',
    }, t)).toBe('Account requires additional verification; not charged')
  })
})
