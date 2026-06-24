import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageDetailMediaGrid from '../UsageDetailMediaGrid.vue'

describe('UsageDetailMediaGrid', () => {
  it('renders media metadata cards with safe preview URLs', () => {
    const wrapper = mount(UsageDetailMediaGrid, {
      props: {
        cards: [
          {
            title: 'Media #1',
            previewSrc: 'blob:usage-media-preview',
            previewTestId: 'usage-media-preview-payload-post-0',
            rows: [
              { label: 'File name', value: 'post.png' },
              { label: 'Content type', value: 'image/png' },
            ],
          },
        ],
      },
    })

    const preview = wrapper.find('[data-testid="usage-media-preview-payload-post-0"]')
    expect(wrapper.text()).toContain('Media #1')
    expect(wrapper.text()).toContain('File name:')
    expect(wrapper.text()).toContain('post.png')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('blob:usage-media-preview')
    expect(preview.attributes('alt')).toBe('Media #1')
  })

  it('omits the image element when a preview URL is unavailable', () => {
    const wrapper = mount(UsageDetailMediaGrid, {
      props: {
        cards: [
          {
            title: 'Avatar',
            previewSrc: '',
            previewTestId: 'usage-media-preview-payload-avatar',
            rows: [{ label: 'Source', value: 'inline' }],
          },
        ],
      },
    })

    expect(wrapper.text()).toContain('Avatar')
    expect(wrapper.text()).toContain('Source:')
    expect(wrapper.find('img').exists()).toBe(false)
  })
})
