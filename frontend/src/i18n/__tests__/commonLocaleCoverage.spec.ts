import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const commonMessageFiles = import.meta.glob<string>([
  '../../**/*.{ts,vue}',
  '!../../**/__tests__/**',
  '!../../**/*.spec.ts'
], {
  eager: true,
  query: '?raw',
  import: 'default'
})

const COMMON_KEY_PATTERNS = [
  /(?:^|[^A-Za-z0-9_$])t\(\s*['"](common\.[A-Za-z0-9_.]+)['"]/g,
  /\$t\(\s*['"](common\.[A-Za-z0-9_.]+)['"]/g,
  /i18n\.global\.t\(\s*['"](common\.[A-Za-z0-9_.]+)['"]/g,
  /titleKey:\s*['"](common\.[A-Za-z0-9_.]+)['"]/g
]

const ADMIN_SETTINGS_KEY_PATTERNS = [
  /(?:^|[^A-Za-z0-9_$])t\(\s*['"](admin\.settings\.[A-Za-z0-9_.]+)['"]/g,
  /\$t\(\s*['"](admin\.settings\.[A-Za-z0-9_.]+)['"]/g,
  /i18n\.global\.t\(\s*['"](admin\.settings\.[A-Za-z0-9_.]+)['"]/g,
  /titleKey:\s*['"](admin\.settings\.[A-Za-z0-9_.]+)['"]/g,
  /descriptionKey:\s*['"](admin\.settings\.[A-Za-z0-9_.]+)['"]/g
]

const ACCOUNT_WORKBENCH_KEY_PATTERNS = [
  /(?:^|[^A-Za-z0-9_$])t\(\s*['"](accountWorkbench\.[A-Za-z0-9_.]+)['"]/g,
  /\$t\(\s*['"](accountWorkbench\.[A-Za-z0-9_.]+)['"]/g,
  /i18n\.global\.t\(\s*['"](accountWorkbench\.[A-Za-z0-9_.]+)['"]/g,
  /titleKey:\s*['"](accountWorkbench\.[A-Za-z0-9_.]+)['"]/g,
  /descriptionKey:\s*['"](accountWorkbench\.[A-Za-z0-9_.]+)['"]/g
]

function collectStaticKeys(patterns: RegExp[]): string[] {
  const keys = new Set<string>()

  Object.values(commonMessageFiles).forEach((source) => {
    patterns.forEach((pattern) => {
      pattern.lastIndex = 0
      let match: RegExpExecArray | null
      while ((match = pattern.exec(source)) !== null) {
        keys.add(match[1])
      }
    })
  })

  return [...keys].sort()
}

function resolvePath(message: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((current, segment) => {
    if (current && typeof current === 'object' && segment in current) {
      return (current as Record<string, unknown>)[segment]
    }

    return undefined
  }, message)
}

describe('common locale coverage', () => {
  it('keeps static common.* keys available in zh and en', () => {
    const keys = collectStaticKeys(COMMON_KEY_PATTERNS)

    expect(keys).toContain('common.login')
    expect(keys.length).toBeGreaterThan(20)
    expect(keys.filter((key) => resolvePath(zh, key) === undefined)).toEqual([])
    expect(keys.filter((key) => resolvePath(en, key) === undefined)).toEqual([])
  })

  it('keeps static admin.settings.* keys available in zh and en', () => {
    const keys = collectStaticKeys(ADMIN_SETTINGS_KEY_PATTERNS)

    expect(keys).toContain('admin.settings.title')
    expect(keys.length).toBeGreaterThan(100)
    expect(keys.filter((key) => resolvePath(zh, key) === undefined)).toEqual([])
    expect(keys.filter((key) => resolvePath(en, key) === undefined)).toEqual([])
  })

  it('keeps static accountWorkbench.* keys available in zh and en', () => {
    const keys = collectStaticKeys(ACCOUNT_WORKBENCH_KEY_PATTERNS)

    expect(keys).toContain('accountWorkbench.title')
    expect(keys.length).toBeGreaterThan(100)
    expect(keys.filter((key) => resolvePath(zh, key) === undefined)).toEqual([])
    expect(keys.filter((key) => resolvePath(en, key) === undefined)).toEqual([])
  })
})
