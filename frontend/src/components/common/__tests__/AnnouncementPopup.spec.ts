import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

import AnnouncementPopup from '../AnnouncementPopup.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { i18n } from '@/i18n'

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: vi.fn(),
    markRead: vi.fn().mockResolvedValue({ message: 'ok' }),
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AnnouncementPopup', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    i18n.global.setLocaleMessage('en', {
      common: {
        time: {
          never: () => 'Never',
          justNow: () => 'Just now',
          minutesAgo: ({ named }: { named: (key: string) => unknown }) => `${named('n')}m ago`,
          hoursAgo: ({ named }: { named: (key: string) => unknown }) => `${named('n')}h ago`,
          daysAgo: ({ named }: { named: (key: string) => unknown }) => `${named('n')}d ago`,
        },
      },
    })
    document.body.style.overflow = ''
  })

  it('restores body scrolling after dismissing the popup', async () => {
    const announcementStore = useAnnouncementStore()
    announcementStore.announcements = [
      {
        id: 1,
        title: 'Maintenance',
        content: 'Window',
        notify_mode: 'popup',
        created_at: '2026-06-04T00:00:00Z',
        updated_at: '2026-06-04T00:00:00Z',
      },
    ]
    const wrapper = mount(AnnouncementPopup, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })

    announcementStore.currentPopup = announcementStore.announcements[0]
    await nextTick()

    expect(document.body.style.overflow).toBe('hidden')

    await wrapper.get('button').trigger('click')

    expect(announcementStore.currentPopup).toBeNull()
    expect(document.body.style.overflow).toBe('')
  })
})
