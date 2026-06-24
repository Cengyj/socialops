import { describe, expect, it } from 'vitest'

import { formatSafeUserTaskResult } from '../socialTaskResult'
import { formatSocialTaskResultMessage } from '../socialTaskResultMessage'

describe('formatSafeUserTaskResult', () => {
  it('keeps concise business results readable', () => {
    expect(formatSafeUserTaskResult('follow succeeded', 'hidden')).toBe('follow succeeded')
  })

  it('keeps raw diagnostic response details readable', () => {
    const raw = 'upstream response body authorization Bearer abc token=secret proxy http://user:pass@127.0.0.1:8080 trace_id=trace-123'

    expect(formatSafeUserTaskResult(raw, 'Task failed; details hidden')).toBe(raw)
  })

  it('keeps credential-shaped words readable when they are part of the raw error', () => {
    expect(formatSafeUserTaskResult('execution_auth refresh failed', 'hidden')).toBe('execution_auth refresh failed')
    expect(formatSafeUserTaskResult('auth_cookie backup unavailable', 'hidden')).toBe('auth_cookie backup unavailable')
    expect(formatSafeUserTaskResult('access_token exchange failed', 'hidden')).toBe('access_token exchange failed')
    expect(formatSafeUserTaskResult('token_secret missing', 'hidden')).toBe('token_secret missing')
    expect(formatSafeUserTaskResult('api_key rejected', 'hidden')).toBe('api_key rejected')
  })

  it('normalizes whitespace and caps long results', () => {
    const raw = `ok ${'x'.repeat(200)}`

    expect(formatSafeUserTaskResult(raw)).toHaveLength(163)
    expect(formatSafeUserTaskResult('  ok \n done  ')).toBe('ok done')
  })

  it('keeps raw platform business errors readable in task result messages', () => {
    const t = (key: string) => key

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: 'twitter error 399: Sorry, we could not find your account.',
    }, t)).toBe('twitter error 399: Sorry, we could not find your account.')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: 'twitter login error: The password you entered is incorrect.',
    }, t)).toBe('twitter login error: The password you entered is incorrect.')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: 'twitter error 89: token=secret-token-value',
    }, t)).toBe('twitter error 89: token=secret-token-value')
  })

  it('maps exact avatar and banner dimension safe results through localization keys', () => {
    const messages: Record<string, string> = {
      'usage.taskResults.avatarSizeInvalid': 'Avatar image must be 400 × 400; not charged',
      'usage.taskResults.bannerSizeInvalid': 'Banner image must be 1500 × 500; not charged',
      'usage.taskResults.mediaAssetUnavailable': 'Task media asset is unavailable; not charged',
      'usage.taskResults.mediaUploadFailed': 'Platform media upload failed; not charged',
      'usage.taskResults.mediaSourceUnsupported': 'Media references are not available for execution; not charged',
      'usage.taskResults.postVideoUnavailable': 'Video post media is not supported for execution; not charged',
      'usage.taskResults.postMediaTypeUnsupported': 'Post media type is not supported for execution; not charged',
      'usage.taskResults.executionTimeout': 'Task timed out; not charged',
      'usage.taskResults.platformNetworkFailed': 'Platform network request failed; not charged',
      'usage.taskResults.passwordInvalid': 'Password is incorrect; not charged',
      'usage.taskResults.loginDependencyNotConfigured': 'Login dependency is not configured; not charged',
      'usage.taskResults.accountNotFound': 'Account does not exist; not charged',
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
    }, t)).toBe('Media references are not available for execution; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '视频发帖媒体暂未开放，本次未扣费',
    }, t)).toBe('Video post media is not supported for execution; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '发帖媒体类型暂不支持，本次未扣费',
    }, t)).toBe('Post media type is not supported for execution; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '任务执行超时，本次未扣费',
    }, t)).toBe('Task timed out; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '平台网络请求失败，本次未扣费',
    }, t)).toBe('Platform network request failed; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '密码错误，本次未扣费',
    }, t)).toBe('Password is incorrect; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '登录依赖服务未配置，本次未扣费',
    }, t)).toBe('Login dependency is not configured; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '账号不存在，本次未扣费',
    }, t)).toBe('Account does not exist; not charged')

    expect(formatSocialTaskResultMessage({
      status: 'failed',
      result_message: '账号需要额外验证，本次未扣费',
    }, t)).toBe('Account requires additional verification; not charged')
  })

})
