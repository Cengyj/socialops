import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import TaskTypeSelector, { type TaskTypeSelectorCard } from '../TaskTypeSelector.vue'

const cards: TaskTypeSelectorCard[] = [
  {
    type: 'follow',
    label: 'Follow',
    requirement: 'Targets: user ID, username, or profile URL',
    icon: 'userPlus',
    tone: 'bg-primary-100 text-primary-700',
  },
  {
    type: 'post',
    label: 'Post',
    requirement: 'Contents: one post text per line',
    icon: 'chatBubble',
    tone: 'bg-blue-100 text-blue-700',
  },
]

function mountSelector(activeType: TaskTypeSelectorCard['type'] = 'follow') {
  return mount(TaskTypeSelector, {
    props: { activeType, cards },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<span data-testid="icon-stub" :data-icon="name" />' },
      },
    },
  })
}

describe('TaskTypeSelector', () => {
  it('renders existing task type cards without adding actions', () => {
    const wrapper = mountSelector()

    expect(wrapper.get('[data-testid="task-type-follow"]').text()).toContain('Follow')
    expect(wrapper.get('[data-testid="task-type-follow"]').text()).toContain('Targets: user ID')
    expect(wrapper.get('[data-testid="task-type-post"]').text()).toContain('Post')
    expect(wrapper.findAll('button')).toHaveLength(cards.length)
  })

  it('keeps long task type labels and requirements readable without adding actions', () => {
    const longLabel = 'stage118_task_type_label_with_really_long_unbroken_identifier_0123456789abcdef'
    const longRequirement = 'stage118_task_type_requirement_with_really_long_unbroken_identifier_0123456789abcdef'
    const wrapper = mount(TaskTypeSelector, {
      props: {
        activeType: 'follow',
        cards: [{ ...cards[0], label: longLabel, requirement: longRequirement }],
      },
      global: {
        stubs: {
          Icon: { props: ['name'], template: '<span data-testid="icon-stub" :data-icon="name" />' },
        },
      },
    })

    const label = wrapper.get(`[title="${longLabel}"]`)
    const requirement = wrapper.get(`[title="${longRequirement}"]`)
    expect(label.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(requirement.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(wrapper.findAll('button')).toHaveLength(1)
  })

  it('emits the selected task type from existing card clicks', async () => {
    const wrapper = mountSelector()

    await wrapper.get('[data-testid="task-type-post"]').trigger('click')

    expect(wrapper.emitted('select')).toEqual([['post']])
  })
})
