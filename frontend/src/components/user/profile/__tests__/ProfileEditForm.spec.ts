import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import type { User } from '@/types'

const updateProfileMock = vi.hoisted(() => vi.fn())
const showErrorMock = vi.hoisted(() => vi.fn())
const showSuccessMock = vi.hoisted(() => vi.fn())
const authStore = vi.hoisted(() => ({ user: null as User | null }))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: updateProfileMock
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 1,
    username: 'operator',
    email: 'operator@example.com',
    avatar_url: null,
    role: 'user',
    balance: 0,
    concurrency: 5,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    ...overrides
  }
}

describe('ProfileEditForm', () => {
  it('submits a trimmed username after passing the required check', async () => {
    const updatedUser = createUser({ username: 'operator-updated' })
    updateProfileMock.mockResolvedValueOnce(updatedUser)

    const wrapper = mount(ProfileEditForm, {
      props: {
        initialUsername: 'operator'
      }
    })

    await wrapper.get('#username').setValue('  operator-updated  ')
    await wrapper.get('form').trigger('submit')

    expect(updateProfileMock).toHaveBeenCalledWith({
      username: 'operator-updated'
    })
    expect(authStore.user).toEqual(updatedUser)
    expect(showSuccessMock).toHaveBeenCalledWith('profile.updateSuccess')
  })
})
