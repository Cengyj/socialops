function canUseClipboardAPI() {
  return typeof navigator !== 'undefined'
    && !!navigator.clipboard?.writeText
    && (typeof window === 'undefined' || window.isSecureContext !== false)
}

function fallbackCopyText(text: string) {
  if (typeof document === 'undefined') return false

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'readonly')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '-9999px'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()

  try {
    return document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
  }
}

export async function copyTextToClipboard(text: string) {
  if (canUseClipboardAPI()) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // Fall through to the textarea fallback below.
    }
  }

  if (!fallbackCopyText(text)) {
    throw new Error('copy command failed')
  }
}

export function createObjectURLSafe(blob: Blob) {
  const fn = globalThis.URL && typeof globalThis.URL.createObjectURL === 'function'
    ? globalThis.URL.createObjectURL.bind(globalThis.URL)
    : null
  return fn ? fn(blob) : ''
}

export function revokeObjectURLSafe(url: string) {
  if (!url) return
  const fn = globalThis.URL && typeof globalThis.URL.revokeObjectURL === 'function'
    ? globalThis.URL.revokeObjectURL.bind(globalThis.URL)
    : null
  if (fn) fn(url)
}

export function downloadBlob(blob: Blob, filename: string) {
  if (typeof document === 'undefined') {
    throw new Error('download unavailable')
  }

  const url = createObjectURLSafe(blob)
  if (!url) {
    throw new Error('download unavailable')
  }

  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.style.display = 'none'
  document.body.appendChild(link)

  try {
    link.click()
  } finally {
    document.body.removeChild(link)
    revokeObjectURLSafe(url)
  }
}
