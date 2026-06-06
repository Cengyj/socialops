import { getConfiguredTableDefaultPageSize, normalizeTablePageSize } from '@/utils/tablePreferences'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

const STORAGE_KEY = 'table-page-size'
const STORAGE_SOURCE_KEY = 'table-page-size-source'

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  if (typeof window !== 'undefined') {
    try {
      if (window.localStorage.getItem(STORAGE_SOURCE_KEY) === 'user') {
        return normalizeTablePageSize(getConfiguredTableDefaultPageSize() || fallback)
      }
      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored !== null) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      recordClientDiagnostic('tablePageSize.readPersisted', error)
    }
  }
  return normalizeTablePageSize(getConfiguredTableDefaultPageSize() || fallback)
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(size))
    window.localStorage.removeItem(STORAGE_SOURCE_KEY)
  } catch (error) {
    recordClientDiagnostic('tablePageSize.writePersisted', error)
  }
}
