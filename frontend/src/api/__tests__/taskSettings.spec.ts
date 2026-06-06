import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    delete: del,
  },
}))

import taskSettingsAPI from '@/api/taskSettings'

describe('task settings api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    del.mockReset()
  })

  it('lists and saves templates through user task settings routes', async () => {
    const template = {
      id: 'tmpl_1',
      name: 'Follow targets',
      type: 'follow',
      params: { targets: ['@target'] },
      is_default: true,
      created_at: '',
      updated_at: '',
    }
    get.mockResolvedValue({ data: [template] })
    post.mockResolvedValue({ data: template })

    await expect(taskSettingsAPI.listTemplates()).resolves.toEqual([template])
    await expect(taskSettingsAPI.saveTemplate({ name: 'Follow targets', type: 'follow', params: { targets: ['@target'] }, is_default: true })).resolves.toEqual(template)

    expect(get).toHaveBeenCalledWith('/task-settings/templates')
    expect(post).toHaveBeenCalledWith('/task-settings/templates', { name: 'Follow targets', type: 'follow', params: { targets: ['@target'] }, is_default: true })
  })

  it('normalizes a null template list response to an empty list', async () => {
    get.mockResolvedValue({ data: null })

    await expect(taskSettingsAPI.listTemplates()).resolves.toEqual([])

    expect(get).toHaveBeenCalledWith('/task-settings/templates')
  })

  it('validates, copies, defaults, and deletes templates', async () => {
    post.mockResolvedValueOnce({ data: { valid: true, type: 'post', targets: 0, contents: 1, errors: [] } })
    post.mockResolvedValueOnce({ data: { id: 'copy', name: 'copy', type: 'post', params: {}, is_default: false, created_at: '', updated_at: '' } })
    post.mockResolvedValueOnce({ data: { id: 'tmpl', name: 'tmpl', type: 'post', params: {}, is_default: true, created_at: '', updated_at: '' } })
    del.mockResolvedValue({ data: undefined })

    await taskSettingsAPI.validateTemplate({ name: 'Post', type: 'post', params: { contents: ['hello'] } })
    await taskSettingsAPI.copyTemplate('tmpl')
    await taskSettingsAPI.setDefaultTemplate('tmpl')
    await taskSettingsAPI.deleteTemplate('tmpl')

    expect(post).toHaveBeenNthCalledWith(1, '/task-settings/templates/validate', { name: 'Post', type: 'post', params: { contents: ['hello'] } })
    expect(post).toHaveBeenNthCalledWith(2, '/task-settings/templates/tmpl/copy')
    expect(post).toHaveBeenNthCalledWith(3, '/task-settings/templates/tmpl/default')
    expect(del).toHaveBeenCalledWith('/task-settings/templates/tmpl')
  })
})
