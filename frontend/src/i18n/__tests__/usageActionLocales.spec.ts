import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const stableUsageActions: Record<string, string> = {
  login_check: 'Login Check',
  login: 'Login',
  follow: 'Follow',
  like: 'Like',
  post: 'Post',
  retweet: 'Retweet',
  repost: 'Retweet',
  reply: 'Reply',
  quote: 'Quote',
  update_profile: 'Update Profile',
  update_avatar: 'Update Avatar',
  update_banner: 'Update Banner',
}

describe('Usage action locale labels', () => {
  it('keeps SocialOps Usage action names stable English in every locale', () => {
    for (const [key, label] of Object.entries(stableUsageActions)) {
      expect(en.usage.actions[key as keyof typeof en.usage.actions]).toBe(label)
      expect(zh.usage.actions[key as keyof typeof zh.usage.actions]).toBe(label)
    }
    expect(en.usage.actions).not.toHaveProperty('tweet')
    expect(zh.usage.actions).not.toHaveProperty('tweet')
  })
})
