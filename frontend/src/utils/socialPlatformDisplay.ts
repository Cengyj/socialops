export function normalizeSocialPlatform(value?: string | null) {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[-/\s]+/g, '_')
  if (['twitter', 'x', 'x_twitter', 'twitter_x'].includes(normalized)) return 'x_twitter'
  return normalized
}

export interface SocialPlatformLabelOptions {
  emptyLabel?: string
  unknownCase?: 'raw' | 'upper'
}

export function socialPlatformLabel(value?: string | null, options: SocialPlatformLabelOptions = {}) {
  const normalized = normalizeSocialPlatform(value)
  if (!normalized) return options.emptyLabel ?? '-'
  if (normalized === 'x_twitter') return 'X / Twitter'
  if (options.unknownCase === 'upper') return normalized.toUpperCase()
  return String(value || '').trim() || normalized
}

export function socialPlatformInitial(value?: string | null) {
  const normalized = normalizeSocialPlatform(value)
  if (normalized === 'x_twitter') return 'X'
  return (normalized || '?').slice(0, 2).toUpperCase()
}

export function socialPlatformAvatarClass(value?: string | null) {
  const normalized = normalizeSocialPlatform(value)
  if (normalized === 'x_twitter') return 'border-gray-900 bg-gray-900 text-white dark:border-gray-100 dark:bg-gray-100 dark:text-gray-950'
  if (normalized === 'instagram') return 'border-pink-200 bg-pink-50 text-pink-700 dark:border-pink-900/50 dark:bg-pink-900/20 dark:text-pink-300'
  if (normalized === 'tiktok') return 'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-900/50 dark:bg-cyan-900/20 dark:text-cyan-300'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200'
}
