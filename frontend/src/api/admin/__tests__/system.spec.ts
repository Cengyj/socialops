import { beforeEach, describe, expect, it, vi } from 'vitest'

import { performUpdate, restartService, rollback } from '../system'
import { apiClient } from '../../client'

vi.mock('../../client', () => ({
  apiClient: {
    post: vi.fn(),
  },
}))

const postMock = vi.mocked(apiClient.post)

describe('admin system API', () => {
  beforeEach(() => {
    postMock.mockReset()
    postMock.mockResolvedValue({ data: { message: 'ok', need_restart: true } })
  })

  it('sends idempotency keys for high-impact system operations', async () => {
    await performUpdate()
    await rollback()
    await restartService()

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/admin/system/update',
      undefined,
      expect.objectContaining({
        headers: expect.objectContaining({
          'Idempotency-Key': expect.stringMatching(/^system-update-/),
        }),
      }),
    )
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/admin/system/rollback',
      undefined,
      expect.objectContaining({
        headers: expect.objectContaining({
          'Idempotency-Key': expect.stringMatching(/^system-rollback-/),
        }),
      }),
    )
    expect(postMock).toHaveBeenNthCalledWith(
      3,
      '/admin/system/restart',
      undefined,
      expect.objectContaining({
        headers: expect.objectContaining({
          'Idempotency-Key': expect.stringMatching(/^system-restart-/),
        }),
      }),
    )
  })
})
