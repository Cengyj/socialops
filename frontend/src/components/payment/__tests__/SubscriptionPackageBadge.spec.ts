import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SubscriptionPackageBadge from '../SubscriptionPackageBadge.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        'payment.admin.hidden': '下架状态',
        'payment.admin.packageLabel': '订阅套餐',
        'payment.platformFallback': '通用平台',
      }
      return messages[key] || key
    },
  }),
}))

describe('SubscriptionPackageBadge', () => {
  it('uses a platform logo instead of a text placeholder', () => {
    const wrapper = mount(SubscriptionPackageBadge, {
      props: {
        name: 'X 执行套餐',
        platform: 'x_twitter',
      },
    })

    expect(wrapper.find('[data-platform-logo]').exists()).toBe(true)
    expect(wrapper.find('[data-platform-logo]').text()).toBe('')
    expect(wrapper.text()).toContain('X / Twitter')
  })

  it('localizes empty package and hidden fallbacks', () => {
    const wrapper = mount(SubscriptionPackageBadge, {
      props: {
        hidden: true,
      },
    })

    expect(wrapper.text()).toContain('订阅套餐')
    expect(wrapper.text()).toContain('下架状态')
    expect(wrapper.text()).toContain('通用平台')
    expect(wrapper.text()).not.toContain('Subscription Package')
    expect(wrapper.text()).not.toContain('Hidden')
    expect(wrapper.text()).not.toContain('Social')
  })
})
