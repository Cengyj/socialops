import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

interface ToastStore {
  showError: (message: string) => void
}

export class ProfileDisplayError extends Error {
  readonly displayMessage: string

  constructor(displayMessage: string) {
    super('profile_display_error')
    this.displayMessage = displayMessage
  }
}

function resolveProfileErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ProfileDisplayError) {
    return error.displayMessage
  }

  return extractSafeApiErrorMessage(error, fallback)
}

export function showSafeProfileError(
  appStore: ToastStore,
  context: string,
  error: unknown,
  fallback: string,
): void {
  recordClientDiagnostic(context, error)
  appStore.showError(resolveProfileErrorMessage(error, fallback))
}
