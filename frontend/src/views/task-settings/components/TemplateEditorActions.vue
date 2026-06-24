<template>
  <div data-testid="editor-template-actions" class="flex flex-wrap gap-2">
    <button
      type="button"
      class="btn btn-primary btn-sm h-10 min-w-0 max-w-full justify-center"
      data-testid="save-template-button"
      :disabled="!canSave || saving"
      :aria-label="saveButtonLabel"
      :title="saveButtonTitle"
      @click="emit('save')"
    >
      <Icon name="check" size="sm" />
      <span class="min-w-0 truncate">{{ saveButtonLabel }}</span>
    </button>
    <button
      type="button"
      class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center"
      data-testid="validation-button"
      :disabled="saving"
      :aria-label="validateButtonLabel"
      :title="validateButtonTitle"
      @click="emit('validate')"
    >
      <Icon name="shield" size="sm" />
      <span class="min-w-0 truncate">{{ validateButtonLabel }}</span>
    </button>
    <button
      type="button"
      class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center"
      data-testid="copy-template-button"
      :disabled="!hasSelectedTemplate || saving"
      :aria-label="copyButtonLabel"
      :title="copyButtonTitle"
      @click="emit('copy')"
    >
      <Icon name="copy" size="sm" />
      <span class="min-w-0 truncate">{{ copyButtonLabel }}</span>
    </button>
    <button
      type="button"
      class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center"
      data-testid="set-default-button"
      :disabled="!hasSelectedTemplate || saving || isDefault"
      :aria-label="setDefaultButtonLabel"
      :title="setDefaultButtonTitle"
      @click="emit('setDefault')"
    >
      <Icon name="checkCircle" size="sm" />
      <span class="min-w-0 truncate">{{ setDefaultButtonLabel }}</span>
    </button>
    <button
      type="button"
      class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center text-red-600 hover:border-red-200 hover:bg-red-50 dark:text-red-300 dark:hover:border-red-900/60 dark:hover:bg-red-950/30"
      data-testid="delete-template-button"
      :disabled="!hasSelectedTemplate || saving"
      :aria-label="deleteButtonTitle"
      :title="deleteButtonTitle"
      @click="emit('delete')"
    >
      <Icon name="trash" size="sm" />
      <span class="min-w-0 truncate">{{ t('common.delete') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  templateEditorCopyButtonLabel as buildCopyButtonLabel,
  templateEditorCopyButtonTitle as buildCopyButtonTitle,
  templateEditorDeleteButtonTitle as buildDeleteButtonTitle,
  templateEditorSaveButtonLabel as buildSaveButtonLabel,
  templateEditorSaveButtonTitle as buildSaveButtonTitle,
  templateEditorSetDefaultButtonLabel as buildSetDefaultButtonLabel,
  templateEditorSetDefaultButtonTitle as buildSetDefaultButtonTitle,
  templateEditorValidateButtonLabel as buildValidateButtonLabel,
  templateEditorValidateButtonTitle as buildValidateButtonTitle,
  type TemplateEditorTitleOperation,
} from '../templateEditorActionTitles'

export type TemplateEditorOperation = TemplateEditorTitleOperation

const props = defineProps<{
  canSave: boolean
  hasSelectedTemplate: boolean
  isDefault: boolean
  operation: TemplateEditorOperation | null
  saveDisabledReason: string
  saving: boolean
}>()

const emit = defineEmits<{
  save: []
  validate: []
  copy: []
  setDefault: []
  delete: []
}>()

const { t } = useI18n()

const saveButtonTitle = computed(() => buildSaveButtonTitle(t, {
  operation: props.operation,
  saveDisabledReason: props.saveDisabledReason,
  saving: props.saving,
}))
const saveButtonLabel = computed(() => buildSaveButtonLabel(t, {
  operation: props.operation,
}))
const validateButtonTitle = computed(() => buildValidateButtonTitle(t, { saving: props.saving }))
const validateButtonLabel = computed(() => buildValidateButtonLabel(t, {
  operation: props.operation,
}))
const copyButtonTitle = computed(() => buildCopyButtonTitle(t, {
  saving: props.saving,
  hasSelectedTemplate: props.hasSelectedTemplate,
}))
const copyButtonLabel = computed(() => buildCopyButtonLabel(t, {
  operation: props.operation,
}))
const setDefaultButtonTitle = computed(() => buildSetDefaultButtonTitle(t, {
  saving: props.saving,
  hasSelectedTemplate: props.hasSelectedTemplate,
  isDefault: props.isDefault,
}))
const setDefaultButtonLabel = computed(() => buildSetDefaultButtonLabel(t, {
  operation: props.operation,
}))
const deleteButtonTitle = computed(() => buildDeleteButtonTitle(t, {
  saving: props.saving,
  hasSelectedTemplate: props.hasSelectedTemplate,
}))
</script>
