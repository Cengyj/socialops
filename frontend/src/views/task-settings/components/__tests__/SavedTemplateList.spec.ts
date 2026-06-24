import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SavedTemplateList from '../SavedTemplateList.vue'
import type { TaskTemplate } from '@/api/taskSettings'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => ({
      'taskSettings.defaultBadge': 'Default',
      'taskSettings.empty.description': 'Create a reusable task config.',
      'taskSettings.empty.title': 'No templates yet',
      'taskSettings.newTemplate': 'New template',
      'taskSettings.savedConfigs.description': `Saved configs for ${params?.type ?? ''}`,
      'taskSettings.savedConfigs.emptyDescription': `No saved config for ${params?.type ?? ''}`,
      'taskSettings.savedConfigs.emptyTitle': 'No saved config for this type',
      'taskSettings.savedConfigs.newForType': `New ${params?.type ?? ''} template`,
      'taskSettings.savedConfigs.title': 'Saved configs',
    }[key] ?? key),
  }),
}))

type SavedTemplateListProps = {
  activeTypeLabel: string
  loading: boolean
  selectedTemplateId: string
  templates: TaskTemplate[]
  totalTemplateCount: number
  isTemplateUsable: (template: TaskTemplate) => boolean
  taskTypeBadgeClass: (type: TaskTemplate['type']) => string
  taskTypeLabel: (type: TaskTemplate['type']) => string
  templateParameterStateLabel: (template: TaskTemplate) => string
}

const templates: TaskTemplate[] = [
  {
    id: 'tmpl-follow',
    name: 'Follow operators',
    type: 'follow',
    params: { targets: ['@socialops'] },
    is_default: true,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  },
  {
    id: 'tmpl-post-empty',
    name: 'Post draft',
    type: 'post',
    params: { contents: [] },
    is_default: false,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  },
]

function mountList(props: Partial<SavedTemplateListProps> = {}) {
  return mount(SavedTemplateList, {
    props: {
      activeTypeLabel: 'Follow',
      loading: false,
      selectedTemplateId: '',
      templates,
      totalTemplateCount: templates.length,
      isTemplateUsable: (template: TaskTemplate) => template.id !== 'tmpl-post-empty',
      taskTypeBadgeClass: (type: TaskTemplate['type']) => `badge-${type}`,
      taskTypeLabel: (type: TaskTemplate['type']) => type === 'follow' ? 'Follow' : 'Post',
      templateParameterStateLabel: (template: TaskTemplate) => template.id === 'tmpl-post-empty' ? 'Needs input' : 'Ready',
      ...props,
    },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span data-testid="icon-stub" :data-icon="name" />' },
      },
    },
  })
}

describe('SavedTemplateList', () => {
  function expectConstrainedActionButton(wrapper: ReturnType<typeof mountList>, label: string) {
    const button = wrapper.get(`button[aria-label="${label}"]`)
    expect(button.attributes('title')).toBe(label)
    expect(button.classes()).toEqual(expect.arrayContaining(['w-full', 'min-w-0', 'max-w-full', 'justify-center']))
    const text = button.findAll('span').find(node => node.text() === label)
    expect(text, `button text for ${label}`).toBeTruthy()
    expect(text!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    return button
  }

  it('renders the existing loading skeleton without adding actions', () => {
    const wrapper = mountList({ loading: true })

    expect(wrapper.findAll('.skeleton')).toHaveLength(3)
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('renders the all-templates empty state with the existing new-template action', async () => {
    const wrapper = mountList({ templates: [], totalTemplateCount: 0 })

    expect(wrapper.text()).toContain('No templates yet')
    expect(wrapper.text()).toContain('Create a reusable task config.')
    expect(wrapper.findAll('button')).toHaveLength(1)

    const button = expectConstrainedActionButton(wrapper, 'New template')
    await button.trigger('click')

    expect(wrapper.emitted('new-template')).toHaveLength(1)
  })

  it('renders the active-type empty state with the existing scoped new-template action', async () => {
    const wrapper = mountList({
      activeTypeLabel: 'Post',
      templates: [],
      totalTemplateCount: 2,
    })

    expect(wrapper.get('[data-testid="active-type-empty-state"]').text()).toContain('No saved config for this type')
    expect(wrapper.get('[data-testid="active-type-empty-state"]').text()).toContain('No saved config for Post')
    expect(wrapper.findAll('button')).toHaveLength(1)

    const button = expectConstrainedActionButton(wrapper, 'New Post template')
    await button.trigger('click')

    expect(wrapper.emitted('new-template')).toHaveLength(1)
  })

  it('renders saved cards and emits the selected template from existing card clicks', async () => {
    const wrapper = mountList({ selectedTemplateId: 'tmpl-follow' })

    expect(wrapper.findAll('[data-template-card="saved"]')).toHaveLength(2)
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Follow operators')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Follow')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Ready')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').text()).toContain('Default')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').classes()).toContain('border-primary-300')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-follow"]').attributes('aria-current')).toBe('true')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-empty"]').text()).toContain('Needs input')
    expect(wrapper.get('[data-testid="saved-template-card-tmpl-post-empty"]').attributes('aria-current')).toBeUndefined()
    expect(wrapper.findAll('button')).toHaveLength(3)
    expectConstrainedActionButton(wrapper, 'New Follow template')

    await wrapper.get('[data-testid="saved-template-card-tmpl-post-empty"]').trigger('click')

    expect(wrapper.emitted('select')).toEqual([[templates[1]]])
  })

  it('keeps long saved template names readable on mobile and inspectable on desktop', () => {
    const longName = 'stage105-mobile-task-template-name-with-a-very-long-unbroken-identifier-0123456789abcdef'
    const wrapper = mountList({
      templates: [{ ...templates[0], name: longName }],
      totalTemplateCount: 1,
    })

    const name = wrapper.get('[data-testid="saved-template-card-tmpl-follow"] span.font-semibold')
    expect(name.classes()).toEqual(expect.arrayContaining(['break-all', 'sm:truncate']))
    expect(name.attributes('title')).toBe(longName)
  })
})
