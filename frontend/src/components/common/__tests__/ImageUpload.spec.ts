import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ImageUpload from '../ImageUpload.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.imageUpload.fileTooLarge': '文件过大（当前 {size}，上限 {max}）',
    'common.imageUpload.invalidImageType': '请选择图片文件',
    'common.imageUpload.readFailed': '读取所选文件失败',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) =>
        (messages[key] ?? key).replace(/\{(\w+)\}/g, (_, token) => params?.[token] ?? `{${token}}`),
    }),
  }
})

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />',
  },
}))

describe('ImageUpload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('localizes invalid image type errors', async () => {
    const wrapper = mount(ImageUpload, {
      props: {
        modelValue: '',
        mode: 'image',
      },
    })
    const input = wrapper.find('input[type="file"]').element as HTMLInputElement
    const file = new File(['plain text'], 'note.txt', { type: 'text/plain' })

    Object.defineProperty(input, 'files', {
      configurable: true,
      value: [file],
    })
    await wrapper.find('input[type="file"]').trigger('change')
    await nextTick()

    expect(wrapper.text()).toContain('请选择图片文件')
    expect(wrapper.text()).not.toContain('Please select an image file')
  })

  it('localizes file size errors with current and maximum sizes', async () => {
    const wrapper = mount(ImageUpload, {
      props: {
        modelValue: '',
        mode: 'image',
        maxSize: 2,
      },
    })
    const input = wrapper.find('input[type="file"]').element as HTMLInputElement
    const file = new File(['123456'], 'tiny.png', { type: 'image/png' })

    Object.defineProperty(input, 'files', {
      configurable: true,
      value: [file],
    })
    await wrapper.find('input[type="file"]').trigger('change')
    await nextTick()

    expect(wrapper.text()).toContain('文件过大')
    expect(wrapper.text()).toContain('当前')
    expect(wrapper.text()).toContain('上限')
    expect(wrapper.text()).not.toContain('File too large')
  })
})
