import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const accountWorkbenchPath = resolve(__dirname, '../accountWorkbench.ts')
const taskSettingsPath = resolve(__dirname, '../taskSettings.ts')
const socialTaskTypesPath = resolve(__dirname, '../../types/socialTask.ts')

const accountWorkbenchSource = readFileSync(accountWorkbenchPath, 'utf8')
const taskSettingsSource = readFileSync(taskSettingsPath, 'utf8')
const socialTaskTypesSource = readFileSync(socialTaskTypesPath, 'utf8')

describe('SocialOps task action API type contract', () => {
  it('keeps account task submission typed to current executable actions', () => {
    expect(socialTaskTypesSource).toContain("export const DIRECT_SOCIAL_TASK_ACTIONS = ['login', 'login_check'] as const")
    expect(socialTaskTypesSource).toContain('export const PARAMETER_SOCIAL_TASK_ACTIONS = [')
    expect(socialTaskTypesSource).toContain("  'follow',")
    expect(socialTaskTypesSource).toContain("  'post',")
    expect(socialTaskTypesSource).toContain("  'like',")
    expect(socialTaskTypesSource).toContain("  'retweet',")
    expect(socialTaskTypesSource).toContain("  'update_profile',")
    expect(socialTaskTypesSource).toContain("  'update_avatar',")
    expect(socialTaskTypesSource).toContain("  'update_banner',")
    expect(socialTaskTypesSource).toContain('export const EXECUTABLE_SOCIAL_TASK_ACTIONS = [')
    expect(socialTaskTypesSource).toContain('export type DirectSocialTaskAction = typeof DIRECT_SOCIAL_TASK_ACTIONS[number]')
    expect(socialTaskTypesSource).toContain('export type ParameterSocialTaskAction = typeof PARAMETER_SOCIAL_TASK_ACTIONS[number]')
    expect(socialTaskTypesSource).toContain('export type ExecutableSocialTaskAction = typeof EXECUTABLE_SOCIAL_TASK_ACTIONS[number]')

    expect(accountWorkbenchSource).toContain('action: ExecutableSocialTaskAction')
    expect(accountWorkbenchSource).not.toContain('action: string')
  })

  it('keeps task templates limited to parameterized executable actions', () => {
    expect(taskSettingsSource).toContain('export type TaskTemplateType = ParameterSocialTaskAction')
    expect(taskSettingsSource).not.toContain("'login_check'")
    expect(taskSettingsSource).not.toContain("'tweet'")
    expect(taskSettingsSource).not.toContain("'dm'")
    expect(taskSettingsSource).not.toContain("'message'")
  })
})
