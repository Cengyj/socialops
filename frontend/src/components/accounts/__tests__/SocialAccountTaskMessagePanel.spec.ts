import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SocialAccountTaskMessagePanel from '../SocialAccountTaskMessagePanel.vue'

describe('SocialAccountTaskMessagePanel', () => {
  it('renders task messages as readable polite status feedback', () => {
    const wrapper = mount(SocialAccountTaskMessagePanel, {
      props: {
        message: 'proxy unavailable during registration',
        status: 'ip_unavailable',
      },
    })

    expect(wrapper.text()).toBe('proxy unavailable during registration')
    expect(wrapper.attributes('title')).toBe('proxy unavailable during registration')
    expect(wrapper.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words', 'border-red-200']))
    expect(wrapper.attributes('role')).toBe('status')
    expect(wrapper.attributes('aria-live')).toBe('polite')
    expect(wrapper.attributes('aria-atomic')).toBe('true')
  })

  it('uses the neutral warning tone for pending or unknown task states', () => {
    const wrapper = mount(SocialAccountTaskMessagePanel, {
      props: {
        message: 'waiting for execution',
        status: 'pending',
      },
    })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['border-amber-200', 'bg-amber-50', 'text-amber-700']))
  })

  it('uses the attention tone for manual-review task messages', () => {
    const wrapper = mount(SocialAccountTaskMessagePanel, {
      props: {
        message: '账号认证信息不可用，本次未扣费',
        status: 'manual_review',
      },
    })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['border-red-200', 'bg-red-50', 'text-red-700']))
    expect(wrapper.classes()).not.toContain('border-amber-200')
  })
})
