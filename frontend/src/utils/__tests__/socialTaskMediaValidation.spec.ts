import { describe, expect, it } from 'vitest'

import {
  socialPostMediaRefsSupported,
  socialTaskMediaRefExecutable,
  socialTaskMediaRefExecutableStored,
  unsupportedSocialPostMediaKind,
} from '@/utils/socialTaskMediaValidation'

describe('socialTaskMediaValidation', () => {
  it('treats internal social-task library refs as executable stored media', () => {
    expect(socialTaskMediaRefExecutableStored({
      source: 'library',
      storage_key: 'social-task/42/avatar.png',
      content_type: 'image/png',
      width: 400,
      height: 400,
    })).toBe(true)

    expect(socialTaskMediaRefExecutable({
      source: 'library',
      storage_key: 'social-task/42/post-image.jpg',
      content_type: 'image/jpeg',
    })).toBe(true)
  })

  it('keeps stale external library refs fail-closed', () => {
    expect(socialTaskMediaRefExecutableStored({
      source: 'library',
      storage_key: 'media/post.jpg',
      content_type: 'image/jpeg',
    })).toBe(false)

    expect(socialTaskMediaRefExecutable({
      source: 'library',
      storage_key: 'media/avatar.jpg',
      content_type: 'image/jpeg',
      width: 400,
      height: 400,
    })).toBe(false)
  })

  it('allows post media attached from internal task assets while still blocking unsupported refs', () => {
    expect(unsupportedSocialPostMediaKind([
      {
        source: 'library',
        storage_key: 'social-task/42/post-image.jpg',
        content_type: 'image/jpeg',
      },
    ])).toBe('')

    expect(socialPostMediaRefsSupported([
      {
        source: 'library',
        storage_key: 'social-task/42/post-image.jpg',
        content_type: 'image/jpeg',
      },
    ])).toBe(true)

    expect(unsupportedSocialPostMediaKind([
      {
        source: 'library',
        storage_key: 'media/post.jpg',
        content_type: 'image/jpeg',
      },
    ])).toBe('source')
  })

  it('keeps video post media fail-closed even when the ref is executable', () => {
    expect(unsupportedSocialPostMediaKind([
      {
        source: 'library',
        storage_key: 'social-task/42/post-video.mp4',
        content_type: 'video/mp4',
      },
    ])).toBe('video')

    expect(socialPostMediaRefsSupported([
      {
        source: 'inline',
        url: 'data:video/mp4;base64,QUJD',
        content_type: 'video/mp4',
      },
    ])).toBe(false)
  })

  it('keeps mixed, multiple, or unsupported video post media fail-closed', () => {
    expect(unsupportedSocialPostMediaKind([
      {
        source: 'library',
        storage_key: 'social-task/42/post-video.mp4',
        content_type: 'video/mp4',
      },
      {
        source: 'library',
        storage_key: 'social-task/42/post-image.jpg',
        content_type: 'image/jpeg',
      },
    ])).toBe('video')

    expect(unsupportedSocialPostMediaKind([
      {
        source: 'library',
        storage_key: 'social-task/42/post-video-1.mp4',
        content_type: 'video/mp4',
      },
      {
        source: 'library',
        storage_key: 'social-task/42/post-video-2.mp4',
        content_type: 'video/mp4',
      },
    ])).toBe('video')

    expect(unsupportedSocialPostMediaKind([
      {
        source: 'inline',
        url: 'data:video/quicktime;base64,QUJD',
        content_type: 'video/quicktime',
      },
    ])).toBe('video')
  })
})
