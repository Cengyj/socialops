import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = readFileSync(resolve(__dirname, '../AuthLayout.vue'), 'utf8')

describe('AuthLayout source', () => {
  it('keeps auth pages constrained on mobile with long site subtitles', () => {
    expect(source).toContain('auth-shell')
    expect(source).toContain('max-width: min(28rem, calc(100vw - 2rem));')
    expect(source).toContain('overflow-x: hidden;')
    expect(source).toContain('overflow-wrap: anywhere;')
    expect(source).toContain('p-6 shadow-glass sm:p-8')
  })
})
