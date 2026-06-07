<template>
  <div class="contents">
    <BaseDialog :show="showForm" :title="dialogTitle" width="wide" @close="closeDialog">
      <div id="proxy-create-dialog" class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-name">{{ t('proxies.form.name') }}</label>
            <input id="proxy-name" v-model="form.name" type="text" class="input" :placeholder="t('proxies.form.namePlaceholder')" />
          </div>
          <div>
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-type">{{ t('proxies.form.type') }}</label>
            <select id="proxy-type" v-model="form.ipType" class="input">
              <option v-for="option in typeOptionsWithoutAll" :key="String(option.value)" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
          <div class="sm:col-span-2">
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-endpoint">{{ t('proxies.form.endpoint') }}</label>
            <input id="proxy-endpoint" v-model="form.endpoint" type="text" class="input" :placeholder="t('proxies.form.endpointPlaceholder')" />
            <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('proxies.form.endpointHint') }}</p>
          </div>
          <div class="sm:col-span-2">
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-remark">{{ t('proxies.form.remark') }}</label>
            <textarea id="proxy-remark" v-model="form.remark" class="input min-h-[90px]" :placeholder="t('proxies.form.remarkPlaceholder')"></textarea>
          </div>
        </div>
        <div v-if="formError" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          {{ formError }}
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="closeDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="!canSubmitForm || saving" @click="submitForm">
          {{ saving ? t('common.saving') : t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showDelete" :title="t('proxies.deleteDialog.title')" width="normal" @close="closeDeleteDialog">
      <div id="proxy-delete-dialog">
        <div v-if="proxyToDelete" class="space-y-3">
          <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('proxies.deleteDialog.description', { name: proxyToDelete.name }) }}</p>
          <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200">
            {{ t('proxies.deleteDialog.snapshotWarning') }}
          </div>
          <dl class="grid gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="flex min-w-0 justify-between gap-3">
              <dt class="text-gray-500 dark:text-gray-400">{{ t('proxies.columns.name') }}</dt>
              <dd class="min-w-0 truncate text-gray-900 dark:text-white">{{ proxyToDelete.name }}</dd>
            </div>
            <div class="flex min-w-0 justify-between gap-3">
              <dt class="text-gray-500 dark:text-gray-400">{{ t('proxies.columns.endpoint') }}</dt>
              <dd class="min-w-0 truncate text-gray-900 dark:text-white">{{ proxyToDelete.endpoint || '-' }}</dd>
            </div>
          </dl>
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="deleting" @click="closeDeleteDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-danger" :disabled="deleting" @click="confirmDeleteProxy">
          {{ deleting ? t('common.processing') : t('common.delete') }}
        </button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import proxiesAPI from '@/api/proxies'
import type { ProxyPayload, ProxyType } from '@/api/proxies'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

interface ProxyDialogRow {
  id: number
  name: string
  type: ProxyType
  endpoint: string
  remark: string
}

const props = defineProps<{
  showForm: boolean
  showDelete: boolean
  editingProxy: ProxyDialogRow | null
  proxyToDelete: ProxyDialogRow | null
}>()

const emit = defineEmits<{
  closeForm: []
  closeDelete: []
  saved: []
  deleted: [id: number]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const saving = ref(false)
const deleting = ref(false)
const formError = ref('')

const form = reactive({
  name: '',
  ipType: 'residential' as ProxyType,
  endpoint: '',
  remark: '',
})

const typeOptionsWithoutAll = computed(() => [
  { value: 'residential', label: t('proxies.types.residential') },
  { value: 'static', label: t('proxies.types.static') },
  { value: 'mobile', label: t('proxies.types.mobile') },
  { value: 'datacenter', label: t('proxies.types.datacenter') },
])
const dialogTitle = computed(() => props.editingProxy ? t('proxies.editTitle') : t('proxies.addProxy'))
const canSubmitForm = computed(() => form.name.trim() !== '' && form.ipType !== null)

watch(
  () => [props.showForm, props.editingProxy] as const,
  ([showForm, editingProxy]) => {
    if (!showForm) return
    if (editingProxy) {
      form.name = editingProxy.name
      form.ipType = editingProxy.type
      form.endpoint = editingProxy.endpoint
      form.remark = editingProxy.remark
      formError.value = ''
      return
    }
    resetForm()
  },
  { immediate: true }
)

function closeDialog() {
  if (saving.value) return
  emit('closeForm')
}

function closeDeleteDialog() {
  if (deleting.value) return
  emit('closeDelete')
}

function resetForm() {
  form.name = ''
  form.ipType = 'residential'
  form.endpoint = ''
  form.remark = ''
  formError.value = ''
}

async function submitForm() {
  if (!canSubmitForm.value || saving.value) return
  saving.value = true
  formError.value = ''
  try {
    const payload: ProxyPayload & { name: string; ip_type: ProxyType } = {
      name: form.name.trim(),
      ip_type: form.ipType,
      endpoint: form.endpoint.trim(),
      remark: form.remark.trim(),
    }
    if (props.editingProxy) {
      await proxiesAPI.update(props.editingProxy.id, payload)
      appStore.showSuccess(t('proxies.saved'))
    } else {
      await proxiesAPI.create(payload)
      appStore.showSuccess(t('proxies.created'))
    }
    resetForm()
    emit('saved')
  } catch (error) {
    recordClientDiagnostic('proxies.save', error)
    formError.value = extractSafeApiErrorMessage(error, t('common.error'))
    appStore.showError(formError.value)
  } finally {
    saving.value = false
  }
}

async function confirmDeleteProxy() {
  if (deleting.value || !props.proxyToDelete) return
  const row = props.proxyToDelete
  deleting.value = true
  try {
    await proxiesAPI.delete(row.id)
    appStore.showSuccess(t('proxies.deleted'))
    emit('deleted', row.id)
  } catch (error) {
    recordClientDiagnostic('proxies.delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('proxies.deleteFailed')))
  } finally {
    deleting.value = false
  }
}
</script>
