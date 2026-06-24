import type { SocialTaskMediaRef } from '@/types/socialTask'

export type UnsupportedSocialPostMediaKind = 'video' | 'type' | 'source' | ''

export function socialTaskMediaRefExecutableInline(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  const source = String(item.source || '').trim().toLowerCase()
  const rawURL = String(item.url || '').trim()
  return (source === '' || source === 'inline') && rawURL.toLowerCase().startsWith('data:')
}

export function socialTaskMediaRefExecutableStored(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  const source = String(item.source || '').trim().toLowerCase()
  const storageKey = String(item.storage_key || '').trim().toLowerCase()
  return source === 'library' && storageKey.startsWith('social-task/')
}

export function socialTaskMediaRefExecutable(item?: SocialTaskMediaRef | null) {
  return socialTaskMediaRefExecutableInline(item) || socialTaskMediaRefExecutableStored(item)
}

export function inferSocialTaskMediaContentType(item?: SocialTaskMediaRef | null) {
  if (!item) return ''
  const explicit = String(item.content_type || '').trim().toLowerCase()
  if (explicit) return explicit
  const rawURL = String(item.url || '').trim()
  if (!rawURL.startsWith('data:')) return ''
  const comma = rawURL.indexOf(',')
  if (comma <= 5) return ''
  const meta = rawURL.slice(5, comma).trim()
  if (!meta) return ''
  return meta.replace(/;base64$/i, '').trim().toLowerCase()
}

export function unsupportedSocialPostMediaKind(items?: SocialTaskMediaRef[] | null): UnsupportedSocialPostMediaKind {
  for (const item of items ?? []) {
    if (!socialTaskMediaRefExecutable(item)) return 'source'
    const contentType = inferSocialTaskMediaContentType(item)
    if (contentType.startsWith('video/')) return 'video'
    if (!contentType || !contentType.startsWith('image/')) return 'type'
  }

  return ''
}

export function socialPostMediaRefsSupported(items?: SocialTaskMediaRef[] | null) {
  return unsupportedSocialPostMediaKind(items) === ''
}
