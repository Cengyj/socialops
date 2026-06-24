import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post
  }
}))

describe('oauth adoption auth api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
    localStorage.clear()
    document.cookie = 'oauth_bind_access_token=; Max-Age=0; path=/'
  })

  it('posts adoption decisions when exchanging pending oauth completion', async () => {
    const { exchangePendingOAuthCompletion } = await import('@/api/auth')

    await exchangePendingOAuthCompletion({
      adoptDisplayName: false,
      adoptAvatar: true
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/pending/exchange', {
      adopt_display_name: false,
      adopt_avatar: true
    })
  })

  it('posts linuxdo invitation completion with adoption decisions', async () => {
    const { completeLinuxDoOAuthRegistration } = await import('@/api/auth')

    await completeLinuxDoOAuthRegistration('invite-code', {
      adoptDisplayName: true,
      adoptAvatar: false
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/linuxdo/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: true,
      adopt_avatar: false
    })
  })

  it('posts linuxdo create-account completion with adoption decisions', async () => {
    const { createPendingLinuxDoOAuthAccount } = await import('@/api/auth')

    await createPendingLinuxDoOAuthAccount('invite-code', {
      adoptDisplayName: false,
      adoptAvatar: true
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/linuxdo/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: false,
      adopt_avatar: true
    })
  })

  it('posts affiliate code when completing linuxdo oauth registration', async () => {
    const { completeLinuxDoOAuthRegistration } = await import('@/api/auth')

    await completeLinuxDoOAuthRegistration(
      'invite-code',
      {
        adoptDisplayName: true,
        adoptAvatar: false
      },
      ' AFF123 '
    )

    expect(post).toHaveBeenCalledWith('/auth/oauth/linuxdo/complete-registration', {
      invitation_code: 'invite-code',
      aff_code: 'AFF123',
      adopt_display_name: true,
      adopt_avatar: false
    })
  })

  it('posts oidc invitation completion with adoption decisions', async () => {
    const { completeOIDCOAuthRegistration } = await import('@/api/auth')

    await completeOIDCOAuthRegistration('invite-code', {
      adoptDisplayName: false,
      adoptAvatar: true
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/oidc/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: false,
      adopt_avatar: true
    })
  })

  it('posts oidc create-account completion with adoption decisions', async () => {
    const { createPendingOIDCOAuthAccount } = await import('@/api/auth')

    await createPendingOIDCOAuthAccount('invite-code', {
      adoptDisplayName: true,
      adoptAvatar: false
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/oidc/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: true,
      adopt_avatar: false
    })
  })

  it('posts wechat invitation completion with adoption decisions', async () => {
    const { completeWeChatOAuthRegistration } = await import('@/api/auth')

    await completeWeChatOAuthRegistration('invite-code', {
      adoptDisplayName: true,
      adoptAvatar: true
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/wechat/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: true,
      adopt_avatar: true
    })
  })

  it('posts wechat create-account completion with adoption decisions', async () => {
    const { createPendingWeChatOAuthAccount } = await import('@/api/auth')

    await createPendingWeChatOAuthAccount('invite-code', {
      adoptDisplayName: false,
      adoptAvatar: false
    })

    expect(post).toHaveBeenCalledWith('/auth/oauth/wechat/complete-registration', {
      invitation_code: 'invite-code',
      adopt_display_name: false,
      adopt_avatar: false
    })
  })

  it('posts affiliate code when creating pending wechat oauth account', async () => {
    const { createPendingWeChatOAuthAccount } = await import('@/api/auth')

    await createPendingWeChatOAuthAccount(
      'invite-code',
      {
        adoptDisplayName: false,
        adoptAvatar: true
      },
      'WXAFF'
    )

    expect(post).toHaveBeenCalledWith('/auth/oauth/wechat/complete-registration', {
      invitation_code: 'invite-code',
      aff_code: 'WXAFF',
      adopt_display_name: false,
      adopt_avatar: true
    })
  })

  it('classifies oauth completion results as login or bind', async () => {
    const { getOAuthCompletionKind } = await import('@/api/auth')

    expect(getOAuthCompletionKind({ access_token: 'access-token' })).toBe('login')
    expect(getOAuthCompletionKind({ redirect: '/profile' })).toBe('bind')
  })

  it('requests an HttpOnly oauth bind cookie before redirect binding', async () => {
    localStorage.setItem('auth_token', 'access-token-value')
    const { prepareOAuthBindAccessTokenCookie } = await import('@/api/auth')

    await prepareOAuthBindAccessTokenCookie()

    expect(post).toHaveBeenCalledWith('/auth/oauth/bind-token')
  })
})
