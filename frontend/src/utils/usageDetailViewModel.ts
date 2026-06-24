import type {
  SocialProfileUpdateParams,
  UsageLog,
  UsageTaskMediaRef,
  UsageTaskTemplateSnapshot,
} from '@/api/usage'
import {
  buildDetailRows,
  formatByteSize,
  formatCurrency,
  formatDate,
  formatMediaDimensions,
  formatNumber,
  hasMediaMetadata,
  mediaPreviewKey,
  mediaPreviewTestID,
  normalizeStringList,
  parseProxySnapshotValue,
} from '@/utils/usageRecords'

export interface UsageDetailFieldRow {
  label: string
  value: string
  span?: 'auto' | 'full'
  valueTone?: 'normal' | 'money' | 'success' | 'danger' | 'muted' | 'technical'
  badgeClass?: string
}

export interface UsageDetailMediaCard {
  title: string
  rows: UsageDetailFieldRow[]
  previewSrc: string
  previewTestId: string
}

export interface UsageDetailPoolCard {
  title: string
  values: string[]
}

export interface UsageDetailViewModel {
  overviewRows: UsageDetailFieldRow[]
  resultRows: UsageDetailFieldRow[]
  proxyRows: UsageDetailFieldRow[]
  payloadRows: UsageDetailFieldRow[]
  payloadProfileRows: UsageDetailFieldRow[]
  payloadMediaCards: UsageDetailMediaCard[]
  templateSummaryRows: UsageDetailFieldRow[]
  templatePoolCards: UsageDetailPoolCard[]
  templateProfileRows: UsageDetailFieldRow[]
  templateMediaCards: UsageDetailMediaCard[]
  technicalRows: UsageDetailFieldRow[]
}

export type UsageDetailTranslate = (key: string, params?: Record<string, string | number>) => string

export interface UsageDetailViewModelFormatters {
  t: UsageDetailTranslate
  actionLabel: (value?: string | null) => string
  platformLabel: (value?: string | null) => string
  statusLabel: (value?: string | null) => string
  chargeStatusLabel: (value?: string | null) => string
  chargeSourceLabel: (value?: string | null) => string
  proxyStatusLabel: (value?: string | null) => string
  resultMessage: (row: UsageLog) => string
}

export function buildUsageDetailViewModel(
  detail: UsageLog | null | undefined,
  previewURLs: Record<string, string>,
  formatters: UsageDetailViewModelFormatters,
): UsageDetailViewModel {
  return {
    overviewRows: buildOverviewRows(detail, formatters),
    resultRows: buildResultRows(detail, formatters),
    proxyRows: buildProxySnapshotRows(detail?.proxy_snapshot, formatters),
    payloadRows: buildPayloadRows(detail, formatters.t),
    payloadProfileRows: buildProfileRows(detail?.payload?.profile, formatters.t),
    payloadMediaCards: buildPayloadMediaCards(detail, previewURLs, formatters.t),
    templateSummaryRows: buildTemplateSummaryRows(detail?.template_snapshot, formatters),
    templatePoolCards: buildTemplatePoolCards(detail?.template_snapshot, formatters.t),
    templateProfileRows: buildProfileRows(detail?.template_snapshot?.params?.profile, formatters.t),
    templateMediaCards: buildTemplateMediaCards(detail?.template_snapshot, previewURLs, formatters.t),
    technicalRows: buildTechnicalRows(detail, formatters.t),
  }
}

function buildOverviewRows(
  detail: UsageLog | null | undefined,
  formatters: UsageDetailViewModelFormatters,
) {
  if (!detail) return []
  const { t } = formatters
  return buildDetailRows([
    [t('usage.detailLabels.operation'), formatters.actionLabel(detail.operation), { valueTone: 'normal' }],
    [t('usage.detailLabels.platform'), formatters.platformLabel(detail.platform), { valueTone: 'muted' }],
    [t('usage.detailLabels.account'), detail.account_name, { valueTone: 'normal' }],
    [t('usage.detailLabels.status'), formatters.statusLabel(detail.status), { badgeClass: statusBadgeClass(detail.status) }],
    [t('usage.detailLabels.cost'), formatCurrency(detail.cost), { valueTone: 'money' }],
    [t('usage.detailLabels.chargeStatus'), formatters.chargeStatusLabel(detail.charge_status), { badgeClass: chargeStatusBadgeClass(detail.charge_status) }],
    [t('usage.detailLabels.completedAt'), formatDate(detail.completed_at || detail.created_at), { valueTone: 'muted' }],
  ])
}

function buildResultRows(
  detail: UsageLog | null | undefined,
  formatters: UsageDetailViewModelFormatters,
) {
  if (!detail) return []
  const { t } = formatters
  return buildDetailRows([
    [t('usage.detailLabels.result'), formatters.resultMessage(detail), { span: 'full', valueTone: detail.status === 'failed' ? 'danger' : 'normal' }],
    [t('usage.detailLabels.chargeSource'), formatters.chargeSourceLabel(detail.charge_source), { valueTone: 'muted' }],
    [t('usage.detailLabels.quantity'), formatNumber(detail.quantity), { valueTone: 'normal' }],
    [t('usage.detailLabels.createdAt'), formatDate(detail.created_at), { valueTone: 'muted' }],
  ])
}

function buildPayloadRows(detail: UsageLog | null | undefined, t: UsageDetailTranslate) {
  const payload = detail?.payload
  if (!payload) return []
  return buildDetailRows([
    [t('usage.detailLabels.target'), payload.target],
    [t('usage.detailLabels.content'), payload.post?.text, { span: 'full' }],
    [t('usage.detailLabels.quotePostUrl'), payload.post?.quote_post_url, { span: 'full', valueTone: 'muted' }],
  ])
}

function buildTechnicalRows(detail: UsageLog | null | undefined, t: UsageDetailTranslate) {
  if (!detail) return []
  return buildDetailRows([
    [t('usage.detailLabels.billingRequestId'), detail.billing_request_id, { valueTone: 'technical' }],
    [t('usage.detailLabels.idempotencyKey'), detail.idempotency_key, { valueTone: 'technical' }],
  ])
}

function buildProxySnapshotRows(
  value: string | null | undefined,
  formatters: UsageDetailViewModelFormatters,
) {
  const raw = String(value || '').trim()
  if (!raw) return []

  const { t } = formatters
  const parsed = parseProxySnapshotValue(raw)
  if (!parsed) {
    return buildDetailRows([[t('usage.detailLabels.proxySnapshot'), raw]])
  }

  if (parsed.kind === 'endpoint') {
    return buildDetailRows([[t('usage.detailLabels.proxyEndpoint'), parsed.endpoint]])
  }

  return buildDetailRows([
    [t('usage.detailLabels.proxyName'), parsed.name],
    [t('usage.detailLabels.proxyEndpoint'), parsed.endpoint],
    [t('usage.detailLabels.proxyStatus'), formatters.proxyStatusLabel(parsed.status), { badgeClass: proxyStatusBadgeClass(parsed.status) }],
  ])
}

function buildProfileRows(profile: SocialProfileUpdateParams | null | undefined, t: UsageDetailTranslate) {
  if (!profile) return []
  return buildDetailRows([
    [t('usage.detailLabels.displayName'), profile.display_name],
    [t('usage.detailLabels.screenName'), profile.screen_name],
    [t('usage.detailLabels.description'), profile.description],
    [t('usage.detailLabels.location'), profile.location],
    [t('usage.detailLabels.url'), profile.url],
  ])
}

function buildPayloadMediaCards(
  detail: UsageLog | null | undefined,
  previewURLs: Record<string, string>,
  t: UsageDetailTranslate,
) {
  if (!detail) return []
  const cards = buildMediaCards(detail.payload?.post?.media, 'payload', 'post', previewURLs, t)
  const avatarCard = buildNamedMediaCard(t('usage.detailLabels.avatar'), detail.payload?.avatar, mediaPreviewKey('payload', 'avatar'), previewURLs, t)
  const bannerCard = buildNamedMediaCard(t('usage.detailLabels.banner'), detail.payload?.banner, mediaPreviewKey('payload', 'banner'), previewURLs, t)
  if (avatarCard) cards.push(avatarCard)
  if (bannerCard) cards.push(bannerCard)
  return cards
}

function buildTemplateSummaryRows(
  snapshot: UsageTaskTemplateSnapshot | null | undefined,
  formatters: UsageDetailViewModelFormatters,
) {
  if (!snapshot) return []
  const { t } = formatters
  return buildDetailRows([
    [t('usage.detailLabels.templateName'), snapshot.template_name],
    [t('usage.detailLabels.templateType'), formatters.actionLabel(snapshot.template_type)],
    [t('usage.detailLabels.quotePostUrl'), snapshot.params?.quote_post_url],
  ])
}

function buildTemplatePoolCards(
  snapshot: UsageTaskTemplateSnapshot | null | undefined,
  t: UsageDetailTranslate,
) {
  const params = snapshot?.params
  if (!params) return []
  const cards: UsageDetailPoolCard[] = []
  const targets = normalizeStringList(params.targets)
  const contents = normalizeStringList(params.contents)
  if (targets.length > 0) cards.push({ title: t('usage.detailSections.targets'), values: targets })
  if (contents.length > 0) cards.push({ title: t('usage.detailSections.contents'), values: contents })
  return cards
}

function buildTemplateMediaCards(
  snapshot: UsageTaskTemplateSnapshot | null | undefined,
  previewURLs: Record<string, string>,
  t: UsageDetailTranslate,
) {
  if (!snapshot?.params) return []
  const cards = buildMediaCards(snapshot.params.media, 'template', 'post', previewURLs, t)
  const avatarCard = buildNamedMediaCard(t('usage.detailLabels.avatar'), snapshot.params.avatar, mediaPreviewKey('template', 'avatar'), previewURLs, t)
  const bannerCard = buildNamedMediaCard(t('usage.detailLabels.banner'), snapshot.params.banner, mediaPreviewKey('template', 'banner'), previewURLs, t)
  if (avatarCard) cards.push(avatarCard)
  if (bannerCard) cards.push(bannerCard)
  return cards
}

function buildMediaCards(
  items: UsageTaskMediaRef[] | null | undefined,
  scope: 'payload' | 'template',
  section: 'post' | 'avatar' | 'banner',
  previewURLs: Record<string, string>,
  t: UsageDetailTranslate,
) {
  return (items ?? [])
    .map((item, index) => buildNamedMediaCard(
      t('usage.detailLabels.mediaItem', { index: index + 1 }),
      item,
      mediaPreviewKey(scope, section, index),
      previewURLs,
      t,
    ))
    .filter((item): item is UsageDetailMediaCard => !!item)
}

function buildNamedMediaCard(
  title: string,
  item: UsageTaskMediaRef | null | undefined,
  previewKey: string,
  previewURLs: Record<string, string>,
  t: UsageDetailTranslate,
) {
  if (!item || !hasMediaMetadata(item)) return null
  const rows = buildDetailRows([
    [t('usage.detailLabels.fileName'), item.file_name],
    [t('usage.detailLabels.contentType'), item.content_type],
    [t('usage.detailLabels.dimensions'), formatMediaDimensions(item)],
    [t('usage.detailLabels.byteSize'), formatByteSize(item.byte_size)],
    [t('usage.detailLabels.source'), item.source],
  ])
  if (rows.length === 0) return null
  return {
    title,
    rows,
    previewSrc: previewURLs[previewKey] || '',
    previewTestId: mediaPreviewTestID(previewKey),
  }
}

function statusBadgeClass(status?: string | null) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-danger'
  return 'badge-warning'
}

function chargeStatusBadgeClass(status?: string | null) {
  if (status === 'charged') return 'badge-success'
  if (status === 'failed') return 'badge-danger'
  if (status === 'pending') return 'badge-warning'
  return 'badge-gray'
}

function proxyStatusBadgeClass(status?: string | null) {
  if (status === 'online') return 'badge-success'
  if (status === 'offline') return 'badge-danger'
  return 'badge-gray'
}
