import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AnnouncementBell from '../AnnouncementBell.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import { i18n } from '@/i18n'

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: vi.fn(),
    markRead: vi.fn(),
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

describe('AnnouncementBell', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
    vi.clearAllMocks()
  })

  it('does not show success or close detail when mark-as-read fails', async () => {
    const announcementStore = useAnnouncementStore()
    const appStore = useAppStore()
    announcementStore.announcements = [
      {
        id: 1,
        title: 'Maintenance',
        content: 'Window',
        notify_mode: 'silent',
        created_at: '2026-06-04T00:00:00Z',
        updated_at: '2026-06-04T00:00:00Z',
      },
    ]
    vi.spyOn(announcementStore, 'markAsRead').mockRejectedValue(new Error('mark read failed'))
    const showSuccess = vi.spyOn(appStore, 'showSuccess')
    const showError = vi.spyOn(appStore, 'showError')

    const wrapper = mount(AnnouncementBell, {
      global: {
        stubs: {
          Icon: true,
          Teleport: true,
          Transition: false,
        },
      },
    })

    await wrapper.get('button[aria-label="announcements.title"]').trigger('click')
    await wrapper.get('.group').trigger('click')
    await flushPromises()

    const markButtons = wrapper
      .findAll('button')
      .filter((button) => button.text().includes('announcements.markRead'))
    expect(markButtons.length).toBeGreaterThan(0)

    await markButtons[0].trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('mark read failed')
    expect(showSuccess).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Maintenance')
  })
})
