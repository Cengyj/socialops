import { ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { i18n } from '@/i18n'
import { copyTextToClipboard } from '@/utils/browser'

const { t } = i18n.global

export function useClipboard() {
  const appStore = useAppStore()
  const copied = ref(false)

  const copyToClipboard = async (
    text: string,
    successMessage?: string
  ): Promise<boolean> => {
    if (!text) return false

    let success = false

    try {
      await copyTextToClipboard(text)
      success = true
    } catch {
      success = false
    }

    if (success) {
      copied.value = true
      appStore.showSuccess(successMessage || t('common.copiedToClipboard'))
      setTimeout(() => {
        copied.value = false
      }, 2000)
    } else {
      appStore.showError(t('common.copyFailed'))
    }

    return success
  }

  return { copied, copyToClipboard }
}
