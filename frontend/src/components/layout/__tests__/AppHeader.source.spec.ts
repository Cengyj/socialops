import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppHeader mode navigation contract', () => {
  it('does not expose the subscriptions progress shortcut in simple mode', () => {
    expect(componentSource).toContain('<SubscriptionProgressMini v-if="user && !authStore.isSimpleMode" />')
    expect(componentSource).not.toContain('<SubscriptionProgressMini v-if="user" />')
  })

  it('does not expose balance shortcuts in simple mode', () => {
    expect(componentSource).toContain('v-if="user && !authStore.isSimpleMode"')
    expect(componentSource).not.toContain('v-if="user"\n          class="hidden items-center gap-2')
    expect(componentSource).not.toContain('v-if="user"\n                class="border-b border-gray-100 px-4 py-2')
  })
})
