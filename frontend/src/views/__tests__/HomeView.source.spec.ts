import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../HomeView.vue')
const source = readFileSync(viewPath, 'utf8')

describe('HomeView source', () => {
  it('keeps the default public home page constrained on mobile', () => {
    expect(source).toContain('w-full max-w-full flex-col overflow-hidden')
    expect(source).toContain('mx-auto w-full max-w-6xl')
    expect(source).toContain('w-full min-w-0 flex-1 text-center')
    expect(source).toContain('[overflow-wrap:anywhere]')
    expect(source).toContain('w-full max-w-[22rem] min-w-0 items-center justify-center')
    expect(source).toContain('terminal-container')
    expect(source).toContain('display: block;')
    expect(source).toContain('max-width: min(420px, calc(100vw - 2rem));')
    expect(source).toContain('width: 100%;')
    expect(source).toContain('@media (max-width: 640px)')
  })
})
