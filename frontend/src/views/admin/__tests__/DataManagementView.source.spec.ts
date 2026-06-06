import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../DataManagementView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('DataManagementView source contract', () => {
  it('loads agent health before requesting agent-gated data', () => {
    expect(viewSource).toContain('dataManagementAPI.getAgentHealth')
    expect(viewSource).toContain('if (!agentHealth.value?.enabled)')
    expect(viewSource).toContain('loadAgentBackedData')
  })

  it('surfaces fail-closed diagnostics without exposing raw developer logs', () => {
    expect(viewSource).toContain("t('admin.dataManagement.agent.disabled')")
    expect(viewSource).toContain("t('admin.dataManagement.agent.reasonLabel')")
    expect(viewSource).toContain("t('admin.dataManagement.actions.disabledHint')")
    expect(viewSource).not.toContain('console.error')
  })
})
