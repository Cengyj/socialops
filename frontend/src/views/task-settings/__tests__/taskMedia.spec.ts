import { describe, expect, it } from 'vitest'

import type { SocialTaskMediaRef } from '@/api/taskSettings'
import {
  cloneMediaRef,
  cloneMediaRefs,
  fileExtensionForContentType,
  hasMediaRef,
  inferDataURLContentType,
  mediaDimensionsEqual,
  normalizeMediaRef,
  normalizeMediaRefs,
  updateInlineMediaRef,
} from '../taskMedia'

describe('task media helpers', () => {
  it('clones media refs without sharing mutable objects', () => {
    const item: SocialTaskMediaRef = { source: 'inline', url: 'data:image/png;base64,AAA' }
    const cloned = cloneMediaRef(item)
    const clonedList = cloneMediaRefs([item])

    expect(cloned).toEqual(item)
    expect(cloned).not.toBe(item)
    expect(clonedList).toEqual([item])
    expect(clonedList[0]).not.toBe(item)
    expect(cloneMediaRef(null)).toEqual({})
    expect(cloneMediaRefs(null)).toEqual([])
  })

  it('builds inline media refs from uploaded data URLs while preserving existing metadata', () => {
    expect(updateInlineMediaRef(undefined, '  data:image/webp;base64,AAA  ', 'post-image-1')).toEqual({
      source: 'inline',
      url: 'data:image/webp;base64,AAA',
      content_type: 'image/webp',
      file_name: 'post-image-1.webp',
    })

    expect(updateInlineMediaRef({ content_type: 'image/jpeg', file_name: 'avatar.jpg' }, 'https://example.test/avatar', 'avatar-image')).toEqual({
      source: 'inline',
      url: 'https://example.test/avatar',
      content_type: 'image/jpeg',
      file_name: 'avatar.jpg',
    })

    expect(updateInlineMediaRef(undefined, '   ', 'avatar-image')).toEqual({})
  })

  it('normalizes saved media refs and filters blank rows from lists', () => {
    expect(normalizeMediaRef({
      source: ' inline ',
      url: ' data:image/png;base64,AAA ',
      content_type: '',
      file_name: '',
      sha256: ' abc ',
      byte_size: 12,
      width: 400,
      height: 400,
    }, 'avatar-image')).toEqual({
      source: 'inline',
      url: 'data:image/png;base64,AAA',
      content_type: 'image/png',
      file_name: 'avatar-image.png',
      sha256: 'abc',
      byte_size: 12,
      width: 400,
      height: 400,
    })

    expect(normalizeMediaRefs([
      {},
      { source: 'library', storage_key: ' social-task/42/post.jpg ', content_type: 'image/jpeg' },
    ], 'post-image')).toEqual([
      {
        source: 'library',
        storage_key: 'social-task/42/post.jpg',
        content_type: 'image/jpeg',
        file_name: 'post-image-2.jpg',
      },
    ])
  })

  it('detects media presence and exact execution dimensions', () => {
    expect(hasMediaRef({ file_name: 'avatar.png' })).toBe(true)
    expect(hasMediaRef({ width: 400, height: 400 })).toBe(false)
    expect(hasMediaRef(null)).toBe(false)
    expect(mediaDimensionsEqual({ width: 400, height: 400 }, 400, 400)).toBe(true)
    expect(mediaDimensionsEqual({ width: 400, height: 399 }, 400, 400)).toBe(false)
    expect(mediaDimensionsEqual(null, 400, 400)).toBe(false)
  })

  it('maps data URL content types to stable fallback filenames', () => {
    expect(inferDataURLContentType('data:image/jpeg;base64,AAA')).toBe('image/jpeg')
    expect(inferDataURLContentType('https://example.test/image.jpg')).toBe('')
    expect(fileExtensionForContentType('image/jpeg')).toBe('.jpg')
    expect(fileExtensionForContentType('image/gif')).toBe('.gif')
    expect(fileExtensionForContentType('image/webp')).toBe('.webp')
    expect(fileExtensionForContentType('application/octet-stream')).toBe('.png')
  })
})
