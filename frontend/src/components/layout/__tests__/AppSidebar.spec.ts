import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

function sourceBetween(start: string, end: string): string {
  const startIndex = componentSource.indexOf(start)
  const endIndex = componentSource.indexOf(end)

  expect(startIndex).toBeGreaterThanOrEqual(0)
  expect(endIndex).toBeGreaterThan(startIndex)

  return componentSource.slice(startIndex, endIndex)
}

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('does not force active child groups to remain expanded after manual collapse', () => {
    expect(componentSource).toContain('const collapsedActiveGroups = ref<Set<string>>(new Set())')
    expect(componentSource).toContain('isGroupActive(item) && !collapsedActiveGroups.value.has(item.path)')
    expect(componentSource).not.toContain('return expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar admin payment maintenance navigation', () => {
  it('does not hide admin order and plan maintenance behind the user payment switch', () => {
    expect(componentSource).not.toContain('const flagAdminPayment = () => adminSettingsStore.paymentEnabled')
    expect(componentSource).not.toContain('featureFlag: flagAdminPayment')
  })

  it('exposes subscription plan management once under the subscription center', () => {
    const matches = componentSource.match(/path: '\/admin\/orders\/plans'/g) || []

    expect(matches).toHaveLength(1)
    expect(componentSource).toContain("path: '/admin/subscription-center'")
    expect(componentSource).toContain("{ path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: GiftIcon }")
  })
})

describe('AppSidebar admin operations navigation', () => {
  it('does not expose data operations, data management, or backups as standalone admin navigation', () => {
    expect(componentSource).not.toContain("path: '/admin/data-operations'")
    expect(componentSource).not.toContain("label: t('nav.dataOperations')")
    expect(componentSource).not.toContain("path: '/admin/data-management'")
    expect(componentSource).not.toContain("label: t('nav.dataManagement')")
    expect(componentSource).not.toContain("path: '/admin/backups'")
    expect(componentSource).not.toContain("label: t('nav.backup')")
    expect(componentSource).toContain("path: '/admin/settings'")
  })
})

describe('AppSidebar SocialOps account navigation boundaries', () => {
  it('uses separate navigation declarations for users, admin operations, and admin personal links', () => {
    expect(componentSource).toContain('function buildUserNavItems(): NavItem[]')
    expect(componentSource).toContain('function buildAdminNavItems(): NavItem[]')
    expect(componentSource).toContain('function buildAdminPersonalNavItems(): NavItem[]')
    expect(componentSource).not.toContain('buildSelfNavItems')
  })

  it('keeps the user account workbench discoverable while removing it from admin personal navigation', () => {
    const userNavSource = sourceBetween('function buildUserNavItems', 'function buildAdminPersonalNavItems')
    const adminPersonalSource = sourceBetween('function buildAdminPersonalNavItems', 'function buildAdminNavItems')

    expect(userNavSource).toContain("path: '/accounts'")
    expect(userNavSource.indexOf("path: '/accounts'")).toBeLessThan(userNavSource.indexOf("path: '/usage'"))
    expect(userNavSource.indexOf("path: '/usage'")).toBeLessThan(userNavSource.indexOf("path: '/subscriptions'"))
    expect(adminPersonalSource).not.toContain("path: '/accounts'")
    expect(componentSource).not.toContain("path: '/social-accounts'")
    expect(adminPersonalSource).toContain("path: '/profile'")
  })

  it('keeps admin account workbench and total pool entries in the admin business navigation', () => {
    const adminNavSource = sourceBetween('function buildAdminNavItems', 'function finalizeNav')

    expect(adminNavSource).toContain("path: '/admin/account-center'")
    expect(adminNavSource.indexOf("path: '/admin/account-center'")).toBeLessThan(adminNavSource.indexOf("path: '/admin/subscription-center'"))
    expect(adminNavSource).toContain("path: '/accounts'")
    expect(adminNavSource).toContain("path: '/task-settings'")
    expect(adminNavSource).toContain("path: '/proxies'")
    expect(adminNavSource).toContain("path: '/admin/total-accounts'")
    expect(adminNavSource).not.toContain("path: '/admin/accounts'")
    expect(componentSource).not.toContain("path: '/admin" + "/proxies'")
  })

  it('keeps proxy and task settings as user-scoped navigation for every role', () => {
    const userNavSource = sourceBetween('function buildUserNavItems', 'function buildAdminPersonalNavItems')
    const adminNavSource = sourceBetween('function buildAdminNavItems', 'function finalizeNav')

    expect(userNavSource).toContain("path: '/task-settings'")
    expect(userNavSource).toContain("path: '/proxies'")
    expect(adminNavSource).toContain("path: '/task-settings'")
    expect(adminNavSource).toContain("path: '/proxies'")
    expect(componentSource).not.toContain("href=\"/admin" + "/proxies\"")
  })
})
