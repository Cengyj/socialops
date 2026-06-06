import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), '../DashboardView.vue')
const source = readFileSync(sourcePath, 'utf8')

describe('admin DashboardView account workbench navigation contract', () => {
  it('routes task submission and account management shortcuts without exposing a dashboard proxy entry', () => {
    expect(source).toContain('to="/accounts"')
    expect(source).toContain("{ to: '/accounts'")
    expect(source).toContain("{ to: '/admin/total-accounts'")
    expect(source).not.toContain('to="/proxies"')
    expect(source).not.toContain("{ to: '/proxies'")
    expect(source).not.toContain('admin.dashboard.quickLinks.proxies')
    expect(source).not.toContain('personalProxyEntry')
    expect(source).not.toContain('to="/admin/accounts"')
    expect(source).not.toContain("{ to: '/admin/accounts'")
    expect(source).not.toContain("'/admin" + "/proxies'")
    expect(source).not.toContain('@/api/admin' + '/proxies')
    expect(source).not.toContain('adminAPI.proxies')
  })
})
