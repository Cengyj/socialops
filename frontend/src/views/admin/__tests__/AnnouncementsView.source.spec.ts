import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewSource = readFileSync(resolve(testDir, '../AnnouncementsView.vue'), 'utf8')
const targetingEditorSource = readFileSync(
  resolve(testDir, '../../../components/admin/announcements/AnnouncementTargetingEditor.vue'),
  'utf8',
)

describe('admin AnnouncementsView source contract', () => {
  it('loads existing subscription groups for announcement targeting', () => {
    const loadGroupsSource = viewSource.slice(
      viewSource.indexOf('async function loadSubscriptionGroups'),
      viewSource.indexOf('function resetForm'),
    )

    expect(loadGroupsSource).toContain('adminAPI.groups.getAll')
    expect(loadGroupsSource).toContain("subscription_type: 'subscription'")
    expect(loadGroupsSource).not.toContain('Promise.resolve([])')
  })

  it('renders a real subscription group selector for subscription targeting conditions', () => {
    const subscriptionBranch = targetingEditorSource.slice(
      targetingEditorSource.indexOf('cond.type === \'subscription\''),
      targetingEditorSource.indexOf('<div v-else'),
    )

    expect(subscriptionBranch).toContain('type="checkbox"')
    expect(subscriptionBranch).toContain('toggleSubscriptionGroup')
    expect(subscriptionBranch).not.toContain('v-model="subscriptionSelections[groupIndex][condIndex]"')
  })

  it('keeps the announcements view covered by TypeScript checks', () => {
    expect(viewSource).not.toContain('@ts-nocheck')
  })

  it('validates targeting before saving with the shared frontend contract', () => {
    const saveSource = viewSource.slice(
      viewSource.indexOf('async function handleSave'),
      viewSource.indexOf('saving.value = true'),
    )

    expect(viewSource).toContain("from '@/utils/announcementTargeting'")
    expect(saveSource).toContain('validateAnnouncementTargeting(form.targeting)')
    expect(saveSource).not.toContain('const anyOf = form.targeting?.any_of ?? []')
  })

  it('keeps the targeting editor on the same validation contract as save', () => {
    expect(targetingEditorSource).toContain("from '@/utils/announcementTargeting'")
    expect(targetingEditorSource).toContain('validateAnnouncementTargeting(props.modelValue)')
  })
})
