import type { SocialTaskMediaRef } from '@/api/taskSettings'

export function cloneMediaRef(item?: SocialTaskMediaRef | null): SocialTaskMediaRef {
  return item ? { ...item } : {}
}

export function cloneMediaRefs(items?: SocialTaskMediaRef[] | null): SocialTaskMediaRef[] {
  return Array.isArray(items) ? items.map(item => cloneMediaRef(item)) : []
}

export function updateInlineMediaRef(current: SocialTaskMediaRef | undefined, value: string, fallbackBaseName: string): SocialTaskMediaRef {
  const trimmed = String(value || '').trim()
  if (trimmed === '') return {}
  const contentType = inferDataURLContentType(trimmed) || String(current?.content_type || '').trim()
  const fileName = String(current?.file_name || '').trim() || `${fallbackBaseName}${fileExtensionForContentType(contentType)}`
  return {
    source: 'inline',
    url: trimmed,
    content_type: contentType,
    file_name: fileName,
  }
}

export function normalizeMediaRefs(items: SocialTaskMediaRef[], fallbackBaseName: string) {
  return items
    .map((item, index) => normalizeMediaRef(item, `${fallbackBaseName}-${index + 1}`))
    .filter((item): item is SocialTaskMediaRef => !!item)
}

export function normalizeMediaRef(item: SocialTaskMediaRef | undefined, fallbackBaseName: string): SocialTaskMediaRef | undefined {
  if (!item) return undefined
  let source = String(item.source || '').trim()
  const storageKey = String(item.storage_key || '').trim()
  const url = String(item.url || '').trim()
  let contentType = String(item.content_type || '').trim()
  const sha256 = String(item.sha256 || '').trim()
  const byteSize = Number(item.byte_size || 0) || undefined
  const width = Number(item.width || 0) || undefined
  const height = Number(item.height || 0) || undefined

  if (!contentType && url.startsWith('data:')) {
    contentType = inferDataURLContentType(url)
  }
  if (!source && url.startsWith('data:')) {
    source = 'inline'
  }
  const fileName = String(item.file_name || '').trim() || (contentType ? `${fallbackBaseName}${fileExtensionForContentType(contentType)}` : '')

  const normalized: SocialTaskMediaRef = {}
  if (source) normalized.source = source
  if (storageKey) normalized.storage_key = storageKey
  if (url) normalized.url = url
  if (contentType) normalized.content_type = contentType
  if (fileName) normalized.file_name = fileName
  if (sha256) normalized.sha256 = sha256
  if (byteSize) normalized.byte_size = byteSize
  if (width) normalized.width = width
  if (height) normalized.height = height

  if (!normalized.file_name && normalized.content_type) {
    normalized.file_name = `${fallbackBaseName}${fileExtensionForContentType(normalized.content_type)}`
  }
  if (!hasMediaRef(normalized)) return undefined
  return normalized
}

export function hasMediaRef(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  return [
    item.source,
    item.storage_key,
    item.url,
    item.content_type,
    item.file_name,
    item.sha256,
  ].some(value => String(value || '').trim() !== '')
}

export function mediaDimensionsEqual(item: SocialTaskMediaRef | undefined | null, width: number, height: number) {
  if (!item) return false
  return Number(item.width || 0) === width && Number(item.height || 0) === height
}

export function inferDataURLContentType(value: string) {
  if (!value.startsWith('data:')) return ''
  const meta = value.slice(5, value.indexOf(',') > 0 ? value.indexOf(',') : undefined)
  return meta.replace(/;base64$/i, '').trim()
}

export function fileExtensionForContentType(contentType: string) {
  switch (String(contentType || '').toLowerCase()) {
    case 'image/jpeg':
    case 'image/jpg':
      return '.jpg'
    case 'image/gif':
      return '.gif'
    case 'image/webp':
      return '.webp'
    case 'image/png':
      return '.png'
    default:
      return '.png'
  }
}
