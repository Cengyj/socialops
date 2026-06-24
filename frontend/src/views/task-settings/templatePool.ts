export const MAX_TEMPLATE_POOL_VALUES = 500
export const MAX_TEMPLATE_VALUE_LENGTH = 2048

export type TemplatePoolKind = 'targets' | 'contents' | null

export interface TemplatePoolAnalysis {
  validCount: number
  emptyLineCount: number
  duplicateCount: number
  tooLongCount: number
  remaining: number
  overCapacity: boolean
}

export function splitTargetValues(value: string): string[] {
  return value
    .split(/\r?\n|,/)
    .map(item => item.trim())
    .filter(Boolean)
}

export function splitContentValues(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

export function countIgnoredEmptyValues(value: string, kind: TemplatePoolKind) {
  if (!value || !kind) return 0
  const parts = kind === 'targets' ? value.split(/\r?\n|,/) : value.split(/\r?\n/)
  return parts.filter(item => item.trim() === '').length
}

export function analyzeTemplatePool(values: readonly string[], emptyLineCount = 0): TemplatePoolAnalysis {
  const seen = new Set<string>()
  let duplicateCount = 0
  let tooLongCount = 0
  for (const value of values) {
    if (seen.has(value)) duplicateCount += 1
    seen.add(value)
    if (Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH) tooLongCount += 1
  }
  return {
    validCount: values.length - tooLongCount,
    emptyLineCount,
    duplicateCount,
    tooLongCount,
    remaining: Math.max(0, MAX_TEMPLATE_POOL_VALUES - values.length),
    overCapacity: values.length > MAX_TEMPLATE_POOL_VALUES,
  }
}

export function normalizeTemplatePoolValues(values?: readonly unknown[] | null) {
  if (!Array.isArray(values)) return []
  return values.map(value => String(value ?? '').trim()).filter(Boolean)
}

export function templatePoolValuesValid(values?: readonly unknown[] | null) {
  const normalized = normalizeTemplatePoolValues(values)
  if (normalized.length > MAX_TEMPLATE_POOL_VALUES) return false
  return normalized.every(value => Array.from(value).length <= MAX_TEMPLATE_VALUE_LENGTH)
}
