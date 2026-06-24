import { describe, expect, it } from 'vitest'

import {
  formatWorkbenchTaskSummary,
  formatWorkbenchTaskSummaryMeta,
} from '../workbenchTaskSummary'

const messages: Record<string, string> = {
  'accountWorkbench.actions.follow': 'Follow',
  'accountWorkbench.actions.like': 'Like',
  'accountWorkbench.actions.post': 'Post',
  'accountWorkbench.actions.retweet': 'Retweet',
  'accountWorkbench.actions.update_profile': 'Update profile',
  'accountWorkbench.actions.update_avatar': 'Update avatar',
  'accountWorkbench.actions.update_banner': 'Update banner',
  'accountWorkbench.execution.taskSummaryTarget': 'Target: {value}',
  'accountWorkbench.execution.taskSummaryContent': 'Text: {value}',
  'accountWorkbench.execution.taskSummaryQuote': 'Quote: {value}',
  'accountWorkbench.execution.taskSummaryMedia': '{count} media item(s)',
  'accountWorkbench.execution.taskSummaryProfile': '{count} profile field(s)',
  'accountWorkbench.execution.taskSummaryAvatar': 'Avatar image ready',
  'accountWorkbench.execution.taskSummaryBanner': 'Banner image ready',
  'accountWorkbench.execution.taskSummaryNoDetails': 'No structured details',
}

function t(key: string, params?: Record<string, string | number>) {
  let value = messages[key] ?? key
  if (!params) return value
  Object.entries(params).forEach(([name, replacement]) => {
    value = value.replace(`{${name}}`, String(replacement))
  })
  return value
}

describe('formatWorkbenchTaskSummary', () => {
  it('summarizes structured post payloads with text, quote, and media count', () => {
    const summary = formatWorkbenchTaskSummary({
      action: 'post',
      content: 'hello world',
      payload: {
        post: {
          text: 'hello world',
          quote_post_url: 'https://x.com/northwind/status/2',
          media: [
            { source: 'inline', file_name: 'post-image-1.png', content_type: 'image/png' },
            { source: 'inline', file_name: 'post-image-2.png', content_type: 'image/png' },
          ],
        },
      },
    }, t)

    expect(summary).toBe('Text: hello world · Quote: https://x.com/northwind/status/2 · 2 media item(s)')
  })

  it('summarizes structured profile, avatar, and banner payloads', () => {
    expect(formatWorkbenchTaskSummary({
      action: 'update_profile',
      payload: {
        profile: {
          display_name: 'Northwind Ops',
          description: 'Operator account',
        },
      },
    }, t)).toBe('2 profile field(s)')

    expect(formatWorkbenchTaskSummary({
      action: 'update_avatar',
      payload: {
        avatar: { source: 'inline', file_name: 'avatar.png', content_type: 'image/png' },
      },
    }, t)).toBe('Avatar image ready')

    expect(formatWorkbenchTaskSummary({
      action: 'update_banner',
      payload: {
        banner: { source: 'inline', file_name: 'banner.png', content_type: 'image/png' },
      },
    }, t)).toBe('Banner image ready')
  })

  it('falls back to target-based summaries for follow-like actions', () => {
    const summary = formatWorkbenchTaskSummary({
      action: 'follow',
      target: '@northwind',
    }, t)

    expect(summary).toBe('Target: @northwind')
  })

  it('falls back to template snapshot details when payload details are absent', () => {
    const summary = formatWorkbenchTaskSummary({
      action: 'post',
      template_snapshot: {
        template_type: 'post',
        params: {
          contents: ['queued content'],
          quote_post_url: 'https://x.com/northwind/status/9',
          media: [{ source: 'inline', file_name: 'queued.png', content_type: 'image/png' }],
        },
      },
    }, t)

    expect(summary).toBe('Text: queued content · Quote: https://x.com/northwind/status/9 · 1 media item(s)')
  })
})

describe('formatWorkbenchTaskSummaryMeta', () => {
  it('prefixes summaries with the localized action label', () => {
    const summary = formatWorkbenchTaskSummaryMeta({
      action: 'update_profile',
      payload: {
        profile: {
          display_name: 'Northwind Ops',
        },
      },
    }, t)

    expect(summary).toBe('Update profile · 1 profile field(s)')
  })

  it('returns a localized fallback when there are no structured details', () => {
    const summary = formatWorkbenchTaskSummaryMeta({
      action: 'retweet',
    }, t)

    expect(summary).toBe('Retweet · No structured details')
  })

  it('supports alternate action and summary locale namespaces', () => {
    const usageMessages: Record<string, string> = {
      'usage.actions.post': 'Post',
      'usage.taskSummaryContent': 'Text: {value}',
      'usage.taskSummaryQuote': 'Quote: {value}',
      'usage.taskSummaryMedia': '{count} media item(s)',
      'usage.taskSummaryNoDetails': 'No structured details',
    }
    const translate = (key: string, params?: Record<string, string | number>) => {
      let value = usageMessages[key] ?? key
      if (!params) return value
      Object.entries(params).forEach(([name, replacement]) => {
        value = value.replace(`{${name}}`, String(replacement))
      })
      return value
    }

    const summary = formatWorkbenchTaskSummaryMeta({
      action: 'post',
      payload: {
        post: {
          text: 'hello world',
          quote_post_url: 'https://x.com/northwind/status/2',
          media: [{ source: 'inline', file_name: 'queued.png', content_type: 'image/png' }],
        },
      },
    }, translate, {
      actionKeyPrefix: 'usage.actions',
      summaryKeyPrefix: 'usage',
    })

    expect(summary).toBe('Post · Text: hello world · Quote: https://x.com/northwind/status/2 · 1 media item(s)')
  })
})
