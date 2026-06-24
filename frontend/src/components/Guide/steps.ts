import { DriveStep } from 'driver.js'

/**
 * SocialOps onboarding focuses on SaaS administration and social account workflows.
 */
export const getAdminSteps = (t: (key: string) => string): DriveStep[] => [
  {
    popover: {
      title: t('onboarding.admin.welcome.title'),
      description: t('onboarding.admin.welcome.description'),
      align: 'center',
      nextBtnText: t('onboarding.admin.welcome.nextBtn'),
      prevBtnText: t('onboarding.admin.welcome.prevBtn')
    }
  },
  {
    element: '#sidebar-social-account-manage',
    popover: {
      title: t('onboarding.admin.accountManage.title'),
      description: t('onboarding.admin.accountManage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: 'a[href="/admin/subscriptions"]',
    popover: {
      title: t('onboarding.admin.subscriptions.title'),
      description: t('onboarding.admin.subscriptions.description'),
      side: 'right',
      align: 'center',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: 'a[href="/admin/settings"]',
    popover: {
      title: t('onboarding.admin.settings.title'),
      description: t('onboarding.admin.settings.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close', 'previous']
    }
  }
]

export const getUserSteps = (t: (key: string) => string): DriveStep[] => [
  {
    popover: {
      title: t('onboarding.user.welcome.title'),
      description: t('onboarding.user.welcome.description'),
      align: 'center',
      nextBtnText: t('onboarding.user.welcome.nextBtn'),
      prevBtnText: t('onboarding.user.welcome.prevBtn')
    }
  },
  {
    element: 'a[href="/dashboard"]',
    popover: {
      title: t('onboarding.user.dashboard.title'),
      description: t('onboarding.user.dashboard.description'),
      side: 'right',
      align: 'center',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: 'a[href="/subscriptions"]',
    popover: {
      title: t('onboarding.user.subscriptions.title'),
      description: t('onboarding.user.subscriptions.description'),
      side: 'right',
      align: 'center',
      showButtons: ['next', 'previous']
    }
  },
  {
    element: 'a[href="/usage"]',
    popover: {
      title: t('onboarding.user.usage.title'),
      description: t('onboarding.user.usage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close', 'previous']
    }
  }
]
