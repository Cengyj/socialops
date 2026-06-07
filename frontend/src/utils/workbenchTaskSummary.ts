import type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  SocialTaskPayload,
  SocialTaskTemplateSnapshot,
} from '@/types/socialTask'

export interface WorkbenchTaskSummaryRecord {
  action?: string | null
  target?: string | null
  content?: string | null
  payload?: SocialTaskPayload | null
  template_snapshot?: SocialTaskTemplateSnapshot | null
}

type Translate = (key: string, params?: Record<string, string | number>) => string

interface WorkbenchTaskSummaryLocaleOptions {
  actionKeyPrefix?: string
  summaryKeyPrefix?: string
}

const defaultLocaleOptions: Required<WorkbenchTaskSummaryLocaleOptions> = {
  actionKeyPrefix: 'accountWorkbench.actions',
  summaryKeyPrefix: 'accountWorkbench.execution',
}

export function formatWorkbenchTaskSummary(
  row: WorkbenchTaskSummaryRecord,
  translate: Translate,
  options?: WorkbenchTaskSummaryLocaleOptions,
) {
  const locale = { ...defaultLocaleOptions, ...(options ?? {}) }
  const action = String(row.action || '').trim().toLowerCase()
  const payload = row.payload ?? undefined
  const snapshot = row.template_snapshot ?? undefined
  const target = firstNonEmpty(row.target, payload?.target, snapshot?.params?.targets?.[0])
  const content = firstNonEmpty(row.content, payload?.post?.text, snapshot?.params?.contents?.[0])
  const quote = firstNonEmpty(payload?.post?.quote_post_url, snapshot?.params?.quote_post_url)

  if (action === 'follow' || action === 'like' || action === 'retweet') {
    return target
      ? translate(`${locale.summaryKeyPrefix}.taskSummaryTarget`, { value: target })
      : translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
  }

  if (action === 'post') {
    const parts: string[] = []
    if (content) parts.push(translate(`${locale.summaryKeyPrefix}.taskSummaryContent`, { value: content }))
    if (quote) parts.push(translate(`${locale.summaryKeyPrefix}.taskSummaryQuote`, { value: quote }))
    const mediaCount = countMediaRefs(payload?.post?.media) || countMediaRefs(snapshot?.params?.media)
    if (mediaCount > 0) parts.push(translate(`${locale.summaryKeyPrefix}.taskSummaryMedia`, { count: mediaCount }))
    return parts.join(' · ') || translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
  }

  if (action === 'update_profile') {
    const fieldCount = countProfileFields(payload?.profile) || countProfileFields(snapshot?.params?.profile)
    return fieldCount > 0
      ? translate(`${locale.summaryKeyPrefix}.taskSummaryProfile`, { count: fieldCount })
      : translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
  }

  if (action === 'update_avatar') {
    return hasMediaRef(payload?.avatar) || hasMediaRef(snapshot?.params?.avatar)
      ? translate(`${locale.summaryKeyPrefix}.taskSummaryAvatar`)
      : translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
  }

  if (action === 'update_banner') {
    return hasMediaRef(payload?.banner) || hasMediaRef(snapshot?.params?.banner)
      ? translate(`${locale.summaryKeyPrefix}.taskSummaryBanner`)
      : translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
  }

  if (target) return translate(`${locale.summaryKeyPrefix}.taskSummaryTarget`, { value: target })
  if (content) return translate(`${locale.summaryKeyPrefix}.taskSummaryContent`, { value: content })
  return translate(`${locale.summaryKeyPrefix}.taskSummaryNoDetails`)
}

export function formatWorkbenchTaskSummaryMeta(
  row: WorkbenchTaskSummaryRecord,
  translate: Translate,
  options?: WorkbenchTaskSummaryLocaleOptions,
) {
  const locale = { ...defaultLocaleOptions, ...(options ?? {}) }
  const action = String(row.action || '').trim()
  const actionKey = action ? `${locale.actionKeyPrefix}.${action}` : ''
  const translatedAction = actionKey ? translate(actionKey) : ''
  const actionLabel = translatedAction && translatedAction !== actionKey ? translatedAction : (action || '-')
  return `${actionLabel} · ${formatWorkbenchTaskSummary(row, translate, locale)}`
}

function firstNonEmpty(...values: Array<string | null | undefined>) {
  return values.map(value => String(value || '').trim()).find(Boolean) || ''
}

function countProfileFields(profile?: SocialProfileUpdateParams | null) {
  if (!profile) return 0
  return [
    profile.display_name,
    profile.screen_name,
    profile.description,
    profile.location,
    profile.url,
  ].filter(value => String(value || '').trim() !== '').length
}

function countMediaRefs(items?: SocialTaskMediaRef[] | null) {
  return (items ?? []).filter(item => hasMediaRef(item)).length
}

function hasMediaRef(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  return [
    item.source,
    item.storage_key,
    item.url,
    item.content_type,
    item.file_name,
    item.sha256,
  ].some(value => String(value || '').trim() !== '')
}
