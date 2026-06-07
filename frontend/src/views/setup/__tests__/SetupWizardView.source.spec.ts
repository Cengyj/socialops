import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewPath = resolve(testDir, '../SetupWizardView.vue')
const source = readFileSync(viewPath, 'utf8')

describe('SetupWizardView source contract', () => {
  it('keeps Redis TLS configuration in the Redis step only', () => {
    const databaseStep = source.slice(
      source.indexOf('<!-- Step 1: Database -->'),
      source.indexOf('<!-- Step 2: Redis -->'),
    )
    const redisStep = source.slice(
      source.indexOf('<!-- Step 2: Redis -->'),
      source.indexOf('<!-- Step 3: Admin -->'),
    )

    expect(databaseStep).not.toContain('formData.redis.enable_tls')
    expect(databaseStep).not.toContain('setup.redis.enableTls')
    expect(redisStep).toContain('formData.redis.enable_tls')
    expect(redisStep).toContain('setup.redis.enableTls')
  })
})
