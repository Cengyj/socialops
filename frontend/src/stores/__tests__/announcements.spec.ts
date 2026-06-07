import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAnnouncementStore } from '@/stores/announcements'
import { announcementsAPI } from '@/api'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: vi.fn(),
    markRead: vi.fn(),
  },
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic: vi.fn(),
}))

describe('useAnnouncementStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('propagates mark-as-read failures to explicit callers', async () => {
    const err = new Error('mark read failed')
    vi.mocked(announcementsAPI.markRead).mockRejectedValue(err)
    const store = useAnnouncementStore()
    store.announcements = [
      {
        id: 1,
        title: 'Maintenance',
        content: 'Window',
        notify_mode: 'silent',
        created_at: '2026-06-04T00:00:00Z',
        updated_at: '2026-06-04T00:00:00Z',
      },
    ]

    await expect(store.markAsRead(1)).rejects.toThrow('mark read failed')

    expect(recordClientDiagnostic).toHaveBeenCalledWith('announcements.markRead', err)
    expect(store.announcements[0].read_at).toBeUndefined()
  })
})
