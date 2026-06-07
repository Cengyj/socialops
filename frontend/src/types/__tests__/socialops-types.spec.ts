import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const typesPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const source = readFileSync(typesPath, 'utf8')

describe('SocialOps core frontend types', () => {
  it('does not export legacy proxy subscription conversion contracts from the shared type barrel', () => {
    expect(source).not.toContain('export interface Subscription {')
    expect(source).not.toContain('export interface ProxyNode')
    expect(source).not.toContain('export interface ConversionRequest')
    expect(source).not.toContain('export interface ConversionResult')
    expect(source).not.toContain("target_type: 'clash'")
    expect(source).not.toContain('total_conversions')
  })
})
