import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  copyTextToClipboard,
  createObjectURLSafe,
  downloadBlob,
  revokeObjectURLSafe,
} from '../browser'

const originalClipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
const originalSecureContextDescriptor = Object.getOwnPropertyDescriptor(window, 'isSecureContext')
const originalCreateObjectURL = globalThis.URL.createObjectURL
const originalRevokeObjectURL = globalThis.URL.revokeObjectURL

describe('browser utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'isSecureContext', {
      configurable: true,
      value: true,
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  afterEach(() => {
    if (originalClipboardDescriptor) {
      Object.defineProperty(navigator, 'clipboard', originalClipboardDescriptor)
    } else {
      delete (navigator as Navigator & { clipboard?: Clipboard }).clipboard
    }

    if (originalSecureContextDescriptor) {
      Object.defineProperty(window, 'isSecureContext', originalSecureContextDescriptor)
    } else {
      delete (window as Window & { isSecureContext?: boolean }).isSecureContext
    }

    if ('execCommand' in document) {
      delete (document as Document & { execCommand?: typeof document.execCommand }).execCommand
    }

    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: originalCreateObjectURL,
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: originalRevokeObjectURL,
    })
  })

  it('uses the Clipboard API in a secure context', async () => {
    await copyTextToClipboard('northwind')

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('northwind')
  })

  it('falls back to execCommand outside a secure context', async () => {
    Object.defineProperty(window, 'isSecureContext', {
      configurable: true,
      value: false,
    })
    document.execCommand = vi.fn().mockReturnValue(true)

    await copyTextToClipboard('fallback')

    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
    expect(document.execCommand).toHaveBeenCalledWith('copy')
  })

  it('throws when clipboard and fallback copy both fail', async () => {
    Object.defineProperty(window, 'isSecureContext', {
      configurable: true,
      value: false,
    })
    document.execCommand = vi.fn().mockReturnValue(false)

    await expect(copyTextToClipboard('blocked')).rejects.toThrow('copy command failed')
  })

  it('creates and revokes object URLs only when the platform supports them', () => {
    const blob = new Blob(['preview'])
    const createObjectURL = vi.fn(() => 'blob:preview')
    const revokeObjectURL = vi.fn()
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      value: revokeObjectURL,
    })

    expect(createObjectURLSafe(blob)).toBe('blob:preview')
    revokeObjectURLSafe('blob:preview')
    revokeObjectURLSafe('')

    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(revokeObjectURL).toHaveBeenCalledTimes(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:preview')
  })

  it('downloads blobs through an anchor and revokes the temporary URL', () => {
    const blob = new Blob(['platform,name'])
    const createObjectURL = vi.fn(() => 'blob:download')
    const revokeObjectURL = vi.fn()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      value: createObjectURL,
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      value: revokeObjectURL,
    })

    downloadBlob(blob, 'accounts.csv')

    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:download')
    click.mockRestore()
  })
})
