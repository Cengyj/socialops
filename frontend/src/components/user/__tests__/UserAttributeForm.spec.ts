import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UserAttributeForm from '../UserAttributeForm.vue'

const { getUserAttributeValues, listEnabledDefinitions } = vi.hoisted(() => ({
  getUserAttributeValues: vi.fn(),
  listEnabledDefinitions: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    userAttributes: {
      getUserAttributeValues,
      listEnabledDefinitions,
    },
  },
}))

vi.mock('@/components/common/Select.vue', () => ({
  default: {
    name: 'Select',
    props: ['modelValue', 'options'],
    emits: ['update:modelValue', 'change'],
    template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value); $emit(\'change\')"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
  },
}))

describe('UserAttributeForm', () => {
  beforeEach(() => {
    listEnabledDefinitions.mockReset()
    getUserAttributeValues.mockReset()

    listEnabledDefinitions.mockResolvedValue([
      {
        id: 9,
        key: 'tier',
        name: 'Tier',
        description: '',
        type: 'text',
        options: [],
        required: false,
        validation: {},
        placeholder: '',
        display_order: 0,
        enabled: true,
        created_at: '2026-06-01T00:00:00Z',
        updated_at: '2026-06-01T00:00:00Z',
      },
    ])
  })

  it('clears local values when parent model is reset to an empty map', async () => {
    const wrapper = mount(UserAttributeForm, {
      props: {
        modelValue: {},
      },
    })
    await flushPromises()

    await wrapper.setProps({ modelValue: { 9: 'gold' } })
    await flushPromises()

    const input = wrapper.get('input')
    expect((input.element as HTMLInputElement).value).toBe('gold')

    await wrapper.setProps({ modelValue: {} })
    await flushPromises()

    expect((input.element as HTMLInputElement).value).toBe('')
  })
})
