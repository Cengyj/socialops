import { describe, expect, it } from 'vitest'

import {
  socialAccountTaskStatusFromAccountSnapshot,
  socialAccountTaskStatusFromTaskResult,
} from '../socialAccountTaskStatus'

describe('socialAccountTaskStatusFromTaskResult', () => {
  it('keeps active task states visible while execution is pending or running', () => {
    expect(socialAccountTaskStatusFromTaskResult({
      status: 'pending',
      result_message: '',
    }, 'stored')).toBe('pending')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'running',
      result_message: '',
    }, 'stored')).toBe('running')
  })

  it('maps terminal task results back to account task state semantics', () => {
    expect(socialAccountTaskStatusFromTaskResult({
      status: 'success',
      result_message: 'follow succeeded',
    }, 'running')).toBe('stored')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务队列繁忙，本次未扣费',
    }, 'pending')).toBe('stored')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '账号认证信息不可用，本次未扣费',
    }, 'stored')).toBe('manual_review')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '密码错误，本次未扣费',
    }, 'stored')).toBe('manual_review')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '执行代理不可用，本次未扣费',
    }, 'stored')).toBe('ip_unavailable')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '该平台动作暂不可用，本次未扣费',
    }, 'stored')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '平台网络请求失败，本次未扣费',
    }, 'stored')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '登录依赖服务未配置，本次未扣费',
    }, 'stored')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务参数不完整，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '头像图片尺寸必须为 400x400，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '背景图图片尺寸必须为 1500x500，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '执行目标不存在，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '内容或目标状态不符合平台要求，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '平台拒绝执行，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务媒体资源不可用，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '账号状态或频率受限，本次未扣费',
    }, 'running')).toBe('manual_review')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务执行失败，本次未扣费',
    }, 'manual_review')).toBe('manual_review')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务执行失败，本次未扣费',
    }, 'running')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '执行已完成，但扣费确认异常，请联系管理员处理',
    }, 'stored')).toBe('failed')

    expect(socialAccountTaskStatusFromTaskResult({
      status: 'failed',
      result_message: '任务执行超时，本次未扣费',
    }, 'running')).toBe('failed')
  })

  it('shows stored account snapshots with known execution failure messages as failed', () => {
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '该动作暂不支持，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '该平台动作暂不可用，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '任务执行超时，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '执行已完成，但扣费确认异常，请联系管理员处理')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '平台网络请求失败，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '登录依赖服务未配置，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '任务参数不完整，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '头像图片尺寸必须为 400x400，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '背景图图片尺寸必须为 1500x500，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '执行目标不存在，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '内容或目标状态不符合平台要求，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '平台拒绝执行，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '任务媒体资源不可用，本次未扣费')).toBe('failed')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '账号认证信息不可用，本次未扣费')).toBe('manual_review')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '密码错误，本次未扣费')).toBe('manual_review')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '账号需要额外验证，本次未扣费')).toBe('manual_review')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '账号状态或频率受限，本次未扣费')).toBe('manual_review')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '执行代理不可用，本次未扣费')).toBe('ip_unavailable')
    expect(socialAccountTaskStatusFromAccountSnapshot('stored', '任务队列繁忙，本次未扣费')).toBe('stored')
    expect(socialAccountTaskStatusFromAccountSnapshot('manual_review', '该平台动作暂不可用，本次未扣费')).toBe('manual_review')
  })
})
