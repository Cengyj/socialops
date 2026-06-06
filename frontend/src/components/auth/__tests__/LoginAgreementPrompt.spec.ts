import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import loginViewSource from '@/views/auth/LoginView.vue?raw'
import registerViewSource from '@/views/auth/RegisterView.vue?raw'

const localeState = vi.hoisted(() => ({
  current: 'en',
}))

const messages: Record<string, Record<string, string>> = {
  en: {
    'auth.loginAgreement.promptTitle': 'You must accept the latest terms before continuing.',
    'auth.loginAgreement.promptDescription': 'Email/password input and quick sign-in stay disabled until you accept.',
    'auth.loginAgreement.viewTerms': 'View terms',
    'auth.loginAgreement.updatedTitle': 'Terms updated',
    'auth.loginAgreement.updatedDescription': 'Our terms were updated on {date}. Please review and accept the following terms before continuing.',
    'auth.loginAgreement.documentsTitle': 'Related documents',
    'auth.loginAgreement.reject': 'Reject',
    'auth.loginAgreement.accept': 'Agree and continue',
    'auth.loginAgreement.recently': 'recently',
  },
  zh: {
    'auth.loginAgreement.promptTitle': '继续登录前需要先同意最新条款。',
  },
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: localeState.current },
    t: (key: string, params?: Record<string, string>) => {
      let value = messages[localeState.current]?.[key] ?? key
      Object.entries(params ?? {}).forEach(([paramKey, paramValue]) => {
        value = value.replace(`{${paramKey}}`, paramValue)
      })
      return value
    },
  }),
}))

function mountPrompt(locale: 'en' | 'zh' = 'en') {
  localeState.current = locale

  return mount(LoginAgreementPrompt, {
    global: {
      stubs: {
        Icon: true,
        RouterLink: {
          props: ['to'],
          template: '<a><slot /></a>',
        },
        Teleport: true,
      },
    },
    props: {
      accepted: false,
      documents: [{ id: 'terms', title: 'Terms of Service' }],
      mode: 'modal',
      updatedAt: '2026-06-01',
      visible: true,
    },
  })
}

describe('LoginAgreementPrompt i18n', () => {
  it('renders the agreement prompt in the active English locale without Chinese fallback text', () => {
    const wrapper = mountPrompt('en')

    expect(wrapper.text()).toContain('You must accept the latest terms before continuing.')
    expect(wrapper.text()).toContain('View terms')
    expect(wrapper.text()).toContain('Terms updated')
    expect(wrapper.text()).toContain('Reject')
    expect(wrapper.text()).toContain('Agree and continue')
    expect(wrapper.text()).not.toMatch(/[\u4e00-\u9fff]/)
  })

  it('keeps login and register agreement warnings behind i18n keys', () => {
    const combinedSource = `${loginViewSource}\n${registerViewSource}`

    expect(combinedSource).not.toContain('未同意最新条款前')
    expect(combinedSource).not.toContain('请先阅读并同意最新条款')
  })
})
