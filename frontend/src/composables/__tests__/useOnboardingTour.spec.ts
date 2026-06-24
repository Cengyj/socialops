import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'

const driverMock = vi.hoisted(() => {
  let activeIndex = 0
  let active = false
  let config: any = null

  const instance = {
    destroy: vi.fn(() => {
      active = false
    }),
    drive: vi.fn((index = 0) => {
      active = true
      activeIndex = index
    }),
    getActiveIndex: vi.fn(() => activeIndex),
    isActive: vi.fn(() => active),
    moveNext: vi.fn(() => {
      activeIndex += 1
    }),
    movePrevious: vi.fn(() => {
      activeIndex = Math.max(0, activeIndex - 1)
    })
  }

  return {
    driver: vi.fn((nextConfig: any) => {
      config = nextConfig
      return instance
    }),
    getConfig: () => config,
    instance,
    reset: () => {
      activeIndex = 0
      active = false
      config = null
      Object.values(instance).forEach((method) => {
        if (typeof method === 'function' && 'mockClear' in method) {
          method.mockClear()
        }
      })
    },
    setActiveIndex: (index: number) => {
      activeIndex = index
    }
  }
})

vi.mock('driver.js', () => ({
  driver: driverMock.driver
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, fallback?: string) => fallback ?? key
  })
}))

vi.mock('@/api', () => ({
  authAPI: {},
  isTotp2FARequired: vi.fn(() => false)
}))

const Harness = defineComponent({
  setup() {
    const tour = useOnboardingTour({ storageKey: 'admin_guide', autoStart: false })

    return { tour }
  },
  template: '<div />'
})

function setAdminUser() {
  const authStore = useAuthStore()
  authStore.user = {
    id: 42,
    email: 'admin@example.com',
    name: 'Admin',
    role: 'admin'
  } as any
}

async function mountHarness() {
  const wrapper = mount(Harness)
  await nextTick()
  return wrapper
}

describe('useOnboardingTour first-step skip behavior', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    driverMock.reset()
    setAdminUser()
  })

  it('lets the welcome-step previous button act as skip and remember the tour as seen', async () => {
    const wrapper = await mountHarness()

    await (wrapper.vm as any).tour.startTour()
    driverMock.getConfig().onPrevClick()

    expect(localStorage.getItem('admin_guide_42_admin_v4_interactive')).toBe('true')
    expect(driverMock.instance.destroy).toHaveBeenCalledTimes(1)
    expect(driverMock.instance.movePrevious).not.toHaveBeenCalled()
    expect(useOnboardingStore().getDriverInstance()).toBeNull()
  })

  it('keeps previous navigation unchanged after the welcome step', async () => {
    const wrapper = await mountHarness()

    await (wrapper.vm as any).tour.startTour()
    driverMock.setActiveIndex(1)
    driverMock.getConfig().onPrevClick()

    expect(localStorage.getItem('admin_guide_42_admin_v4_interactive')).toBeNull()
    expect(driverMock.instance.destroy).not.toHaveBeenCalled()
    expect(driverMock.instance.movePrevious).toHaveBeenCalledTimes(1)
  })

  it('removes the driver default disabled state from the welcome-step skip button', async () => {
    const wrapper = await mountHarness()

    await (wrapper.vm as any).tour.startTour()
    const footer = document.createElement('footer')
    const previousButton = document.createElement('button')
    previousButton.classList.add('driver-popover-btn-disabled')
    previousButton.disabled = true
    previousButton.setAttribute('disabled', '')
    previousButton.setAttribute('aria-disabled', 'true')
    const nextButton = document.createElement('button')
    footer.append(previousButton, nextButton)

    driverMock.getConfig().onPopoverRender(
      {
        description: document.createElement('div'),
        footer,
        nextButton,
        previousButton,
        title: document.createElement('h2')
      },
      {
        config: { steps: [{ popover: {} }, { popover: {} }] },
        state: { activeIndex: 0 }
      }
    )

    expect(previousButton.disabled).toBe(false)
    expect(previousButton.hasAttribute('disabled')).toBe(false)
    expect(previousButton.hasAttribute('aria-disabled')).toBe(false)
    expect(previousButton.classList.contains('driver-popover-btn-disabled')).toBe(false)

    previousButton.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(localStorage.getItem('admin_guide_42_admin_v4_interactive')).toBe('true')
    expect(driverMock.instance.destroy).toHaveBeenCalledTimes(1)
    expect(useOnboardingStore().getDriverInstance()).toBeNull()
  })
})
