import { describe, expect, it } from 'vitest'

import {
  normalizeSocialPlatform,
  socialPlatformAvatarClass,
  socialPlatformInitial,
  socialPlatformLabel,
} from '../socialPlatformDisplay'

describe('social platform display helpers', () => {
  it('normalizes existing X/Twitter aliases without adding new platform concepts', () => {
    expect(normalizeSocialPlatform(' Twitter / X ')).toBe('x_twitter')
    expect(normalizeSocialPlatform('twitter-x')).toBe('x_twitter')
    expect(normalizeSocialPlatform('x')).toBe('x_twitter')
  })

  it('keeps unknown platform values normalized and empty initials stable', () => {
    expect(normalizeSocialPlatform(' custom platform ')).toBe('custom_platform')
    expect(socialPlatformInitial(' custom platform ')).toBe('CU')
    expect(socialPlatformInitial('  ')).toBe('?')
  })

  it('builds shared labels while preserving page-specific fallback choices', () => {
    expect(socialPlatformLabel(' Twitter / X ')).toBe('X / Twitter')
    expect(socialPlatformLabel(' custom platform ')).toBe('custom platform')
    expect(socialPlatformLabel(' custom platform ', { unknownCase: 'upper' })).toBe('CUSTOM_PLATFORM')
    expect(socialPlatformLabel(' ', { emptyLabel: 'Unknown' })).toBe('Unknown')
    expect(socialPlatformLabel(' ')).toBe('-')
  })

  it('maps avatar classes from the shared platform semantics', () => {
    expect(socialPlatformAvatarClass(' Twitter / X ')).toContain('bg-gray-900')
    expect(socialPlatformAvatarClass('instagram')).toContain('bg-pink-50')
    expect(socialPlatformAvatarClass('tiktok')).toContain('bg-cyan-50')
    expect(socialPlatformAvatarClass('custom platform')).toContain('bg-gray-50')
  })
})
