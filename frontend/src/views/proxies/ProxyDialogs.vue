<template>
  <div class="contents">
    <BaseDialog :show="showForm" :title="dialogTitle" width="wide" @close="closeDialog">
      <div id="proxy-create-dialog" class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-name">{{ t('proxies.form.name') }}</label>
            <input id="proxy-name" v-model="form.name" type="text" class="input" :disabled="formInputsDisabled" :placeholder="t('proxies.form.namePlaceholder')" @input="clearFormError" />
          </div>
          <div>
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-type">{{ t('proxies.form.type') }}</label>
            <select id="proxy-type" v-model="form.ipType" class="input" :disabled="formInputsDisabled" @change="clearFormError">
              <option v-for="option in typeOptionsWithoutAll" :key="String(option.value)" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
          <div class="sm:col-span-2">
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-endpoint">{{ t('proxies.form.endpoint') }}</label>
            <input id="proxy-endpoint" v-model="form.endpoint" type="text" class="input" :disabled="formInputsDisabled" :placeholder="t('proxies.form.endpointPlaceholder')" @input="clearFormError" />
            <p class="mt-2 min-w-0 break-words text-xs text-gray-500 dark:text-gray-400" :title="t('proxies.form.endpointHint')">{{ t('proxies.form.endpointHint') }}</p>
          </div>
          <div class="sm:col-span-2">
            <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="proxy-remark">{{ t('proxies.form.remark') }}</label>
            <textarea id="proxy-remark" v-model="form.remark" class="input min-h-[90px]" :disabled="formInputsDisabled" :placeholder="t('proxies.form.remarkPlaceholder')" @input="clearFormError"></textarea>
          </div>
        </div>
        <div v-if="formError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="formError">
          {{ formError }}
        </div>
      </div>

      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="formCancelButtonTitle"
          :title="formCancelButtonTitle"
          :disabled="saving"
          @click="closeDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="formSubmitButtonLabel"
          :title="formSubmitButtonTitle"
          :disabled="!canSubmitForm || saving"
          @click="submitForm"
        >
          <span class="min-w-0 truncate">{{ formSubmitButtonLabel }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showDelete" :title="t('proxies.deleteDialog.title')" width="normal" @close="closeDeleteDialog">
      <div id="proxy-delete-dialog">
        <div v-if="proxyToDelete" class="space-y-3">
          <p class="min-w-0 break-words text-sm text-gray-600 dark:text-gray-300" :title="t('proxies.deleteDialog.description', { name: proxyToDelete.name })">{{ t('proxies.deleteDialog.description', { name: proxyToDelete.name }) }}</p>
          <div class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200" role="status" aria-live="polite" aria-atomic="true" :title="t('proxies.deleteDialog.snapshotWarning')">
            {{ t('proxies.deleteDialog.snapshotWarning') }}
          </div>
          <div v-if="deleteError" class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300" role="alert" aria-live="assertive" aria-atomic="true" :title="deleteError">
            {{ deleteError }}
          </div>
          <dl class="grid gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="flex min-w-0 justify-between gap-3">
              <dt class="text-gray-500 dark:text-gray-400">{{ t('proxies.columns.name') }}</dt>
              <dd class="min-w-0 break-all text-gray-900 sm:truncate dark:text-white" :title="proxyToDelete.name">{{ proxyToDelete.name }}</dd>
            </div>
            <div class="flex min-w-0 justify-between gap-3">
              <dt class="text-gray-500 dark:text-gray-400">{{ t('proxies.columns.endpoint') }}</dt>
              <dd class="min-w-0 break-all text-gray-900 sm:truncate dark:text-white" :title="proxyToDelete.endpoint || '-'">{{ proxyToDelete.endpoint || '-' }}</dd>
            </div>
          </dl>
        </div>
      </div>

      <template #footer>
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="deleteCancelButtonTitle"
          :title="deleteCancelButtonTitle"
          :disabled="deleting"
          @click="closeDeleteDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-danger min-w-0 max-w-full justify-center"
          :aria-label="deleteConfirmButtonTitle"
          :title="deleteConfirmButtonTitle"
          :disabled="deleting || deleteLocked || !proxyToDelete"
          @click="confirmDeleteProxy"
        >
          <span class="min-w-0 truncate">{{ deleting ? t('common.processing') : t('common.delete') }}</span>
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
import type { ProxyPayload, ProxyType, UserProxy } from '@/api/proxies'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import {
  proxyDeleteCancelButtonTitle as buildDeleteCancelButtonTitle,
  proxyDeleteConfirmButtonTitle as buildDeleteConfirmButtonTitle,
  proxyFormCancelButtonTitle as buildFormCancelButtonTitle,
  proxyFormSubmitButtonLabel as buildFormSubmitButtonLabel,
  proxyFormSubmitButtonTitle as buildFormSubmitButtonTitle,
} from './proxyActionTitles'
import { createProxyErrorMessages } from './proxyErrorMessages'

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
  formLocked?: boolean
  deleteLocked?: boolean
  editingProxy: ProxyDialogRow | null
  proxyToDelete: ProxyDialogRow | null
}>()

const emit = defineEmits<{
  closeForm: []
  closeDelete: []
  saved: [proxy: UserProxy]
  deleted: [id: number]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const saving = ref(false)
const deleting = ref(false)
const formError = ref('')
const deleteError = ref('')

const form = reactive({
  name: '',
  ipType: 'residential' as ProxyType,
  endpoint: '',
  remark: '',
})
let formBaselineProxy: ProxyDialogRow | null = null

const typeOptionsWithoutAll = computed(() => [
  { value: 'residential', label: t('proxies.types.residential') },
  { value: 'static', label: t('proxies.types.static') },
  { value: 'dynamic', label: t('proxies.types.dynamic') },
  { value: 'mobile', label: t('proxies.types.mobile') },
  { value: 'datacenter', label: t('proxies.types.datacenter') },
])
const proxyErrorMessages = computed(() => createProxyErrorMessages(t))
const dialogTitle = computed(() => props.editingProxy ? t('proxies.editTitle') : t('proxies.addProxy'))
const editFormLocked = computed(() => Boolean(props.editingProxy && props.formLocked))
const formInputsDisabled = computed(() => saving.value || editFormLocked.value)
const formSubmitDisabledReason = computed(() => {
  if (editFormLocked.value) return t('common.processing')
  if (form.name.trim() === '') return proxyErrorMessages.value.SOCIAL_IP_NAME_REQUIRED
  if (form.ipType === null) return proxyErrorMessages.value.SOCIAL_IP_TYPE_INVALID
  if (props.editingProxy && formMatchesProxy(formBaselineProxy)) return t('proxies.noChanges')
  return ''
})
const canSubmitForm = computed(() => !formSubmitDisabledReason.value)
const formCancelButtonTitle = computed(() => buildFormCancelButtonTitle(t, { saving: saving.value }))
const formSubmitButtonTitle = computed(() => buildFormSubmitButtonTitle(t, {
  saving: saving.value,
  disabledReason: formSubmitDisabledReason.value,
}))
const formSubmitButtonLabel = computed(() => buildFormSubmitButtonLabel(t, {
  saving: saving.value,
}))
const deleteCancelButtonTitle = computed(() => buildDeleteCancelButtonTitle(t, { deleting: deleting.value }))
const deleteConfirmButtonTitle = computed(() => buildDeleteConfirmButtonTitle(t, {
  deleting: deleting.value,
  locked: Boolean(props.deleteLocked),
}))

watch(
  () => [props.showForm, props.editingProxy] as const,
  ([showForm, editingProxy], previous) => {
    const [wasShowing, previousEditingProxy] = previous ?? [false, null]
    if (!showForm) {
      formBaselineProxy = null
      return
    }
    const opened = !wasShowing
    const targetChanged = (editingProxy?.id ?? null) !== (previousEditingProxy?.id ?? null)
    if (!opened && !targetChanged && !formMatchesProxy(formBaselineProxy)) return
    if (editingProxy) {
      resetFormFromProxy(editingProxy)
      return
    }
    resetForm()
  },
  { immediate: true }
)

watch(
  () => [props.showDelete, props.proxyToDelete?.id ?? null] as const,
  ([showDelete, proxyId], previous) => {
    const [wasShowing, previousProxyId] = previous ?? [false, null]
    if (!showDelete) {
      deleteError.value = ''
      return
    }
    if (!wasShowing || proxyId !== previousProxyId) {
      deleteError.value = ''
    }
  },
  { immediate: true }
)

function closeDialog() {
  if (saving.value) return
  emit('closeForm')
}

function closeDeleteDialog() {
  if (deleting.value) return
  deleteError.value = ''
  emit('closeDelete')
}

function resetForm() {
  form.name = ''
  form.ipType = 'residential'
  form.endpoint = ''
  form.remark = ''
  formError.value = ''
  formBaselineProxy = null
}

function resetFormFromProxy(proxy: ProxyDialogRow) {
  form.name = proxy.name
  form.ipType = proxy.type
  form.endpoint = proxy.endpoint
  form.remark = proxy.remark
  formError.value = ''
  formBaselineProxy = { ...proxy }
}

function formMatchesProxy(proxy: ProxyDialogRow | null) {
  return !!proxy
    && normalizeProxyFormText(form.name) === normalizeProxyFormText(proxy.name)
    && form.ipType === proxy.type
    && normalizeProxyFormText(form.endpoint) === normalizeProxyFormText(proxy.endpoint)
    && normalizeProxyFormText(form.remark) === normalizeProxyFormText(proxy.remark)
}

function normalizeProxyFormText(value: string) {
  return value.trim()
}

function clearFormError() {
  if (!saving.value) {
    formError.value = ''
  }
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
      const proxy = await proxiesAPI.update(props.editingProxy.id, payload)
      appStore.showSuccess(t('proxies.saved'))
      resetForm()
      emit('saved', proxy)
    } else {
      const proxy = await proxiesAPI.create(payload)
      appStore.showSuccess(t('proxies.created'))
      resetForm()
      emit('saved', proxy)
    }
  } catch (error) {
    recordClientDiagnostic('proxies.save', error)
    formError.value = extractSafeApiErrorMessage(error, t('proxies.saveFailed'), proxyErrorMessages.value)
    appStore.showError(formError.value)
  } finally {
    saving.value = false
  }
}

async function confirmDeleteProxy() {
  if (deleting.value || props.deleteLocked || !props.proxyToDelete) return
  const row = props.proxyToDelete
  deleting.value = true
  deleteError.value = ''
  try {
    await proxiesAPI.delete(row.id)
    appStore.showSuccess(t('proxies.deleted'))
    emit('deleted', row.id)
  } catch (error) {
    recordClientDiagnostic('proxies.delete', error)
    deleteError.value = extractSafeApiErrorMessage(error, t('proxies.deleteFailed'), proxyErrorMessages.value)
    appStore.showError(deleteError.value)
  } finally {
    deleting.value = false
  }
}
</script>
