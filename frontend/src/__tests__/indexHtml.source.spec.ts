import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('index.html SocialOps product metadata', () => {
  it('uses the SocialOps product title before runtime settings are injected', () => {
    const html = readFileSync(resolve(__dirname, '../../index.html'), 'utf8')

    expect(html).toContain('<title>SocialOps - Website Account Pool Social Operations Platform</title>')
    expect(html).not.toContain('Social Account Operations Platform')
    expect(html).not.toContain('AI API Gateway')
  })
})
