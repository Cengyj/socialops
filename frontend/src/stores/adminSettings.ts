import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminAPI } from '@/api'
import type { CustomMenuItem } from '@/types'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

export const useAdminSettingsStore = defineStore('adminSettings', () => {
  const loaded = ref(false)
  const loading = ref(false)
  let fetchPromise: Promise<void> | null = null

  const readCachedBool = (key: string, defaultValue: boolean): boolean => {
    try {
      const raw = localStorage.getItem(key)
      if (raw === 'true') return true
      if (raw === 'false') return false
    } catch {
      // ignore localStorage failures
    }
    return defaultValue
  }

  const writeCachedBool = (key: string, value: boolean) => {
    try {
      localStorage.setItem(key, value ? 'true' : 'false')
    } catch {
      // ignore localStorage failures
    }
  }

  const paymentEnabled = ref(readCachedBool('payment_enabled_cached', false))
  const customMenuItems = ref<CustomMenuItem[]>([])

  async function fetch(force = false): Promise<void> {
    if (loaded.value && !force) return
    if (fetchPromise) return fetchPromise

    fetchPromise = (async () => {
      loading.value = true
      try {
        const [settings, paymentConfigResp] = await Promise.all([
          adminAPI.settings.getSettings(),
          adminAPI.payment.getConfig()
        ])
        customMenuItems.value = Array.isArray(settings.custom_menu_items) ? settings.custom_menu_items : []

        paymentEnabled.value = paymentConfigResp.data?.enabled ?? false
        writeCachedBool('payment_enabled_cached', paymentEnabled.value)

        loaded.value = true
      } catch (err) {
        // Keep cached/default value: do not "flip" the UI based on a transient fetch failure.
        loaded.value = true
        recordClientDiagnostic('adminSettings.fetch', err)
      } finally {
        loading.value = false
        fetchPromise = null
      }
    })()

    return fetchPromise
  }

  function setPaymentEnabledLocal(value: boolean) {
    paymentEnabled.value = value
    writeCachedBool('payment_enabled_cached', value)
    loaded.value = true
  }

  return {
    loaded,
    loading,
    paymentEnabled,
    customMenuItems,
    fetch,
    setPaymentEnabledLocal
  }
})
