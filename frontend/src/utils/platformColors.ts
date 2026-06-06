/**
 * Platform colors for social account platforms
 */
export type Platform = 'x_twitter' | 'instagram' | 'tiktok' | 'facebook'

export const platformColors: Record<string, { bg: string; text: string; border: string }> = {
  x_twitter: { bg: 'bg-gray-100 dark:bg-gray-800', text: 'text-gray-900 dark:text-gray-100', border: 'border-gray-200 dark:border-gray-700' },
  instagram: { bg: 'bg-pink-50 dark:bg-pink-900/20', text: 'text-pink-700 dark:text-pink-300', border: 'border-pink-200 dark:border-pink-800' },
  tiktok: { bg: 'bg-gray-100 dark:bg-gray-800', text: 'text-gray-900 dark:text-gray-100', border: 'border-gray-200 dark:border-gray-700' },
  facebook: { bg: 'bg-blue-50 dark:bg-blue-900/20', text: 'text-blue-700 dark:text-blue-300', border: 'border-blue-200 dark:border-blue-800' },
}

export function getPlatformColor(platform: string) {
  return platformColors[platform] || platformColors.x_twitter
}
