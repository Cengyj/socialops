<template>
  <div class="flex items-start gap-4">
    <!-- Preview Box -->
    <div class="flex-shrink-0">
      <div
        class="flex items-center justify-center overflow-hidden rounded-xl border-2 border-dashed border-gray-300 bg-gray-50 dark:border-dark-600 dark:bg-dark-800"
        :class="[previewSizeClass, { 'border-solid': hasPreviewValue }]"
      >
        <!-- SVG mode: render inline -->
        <span
          v-if="mode === 'svg' && resolvedPreviewValue"
          class="text-gray-600 dark:text-gray-300 [&>svg]:h-full [&>svg]:w-full"
          :class="innerSizeClass"
          v-html="sanitizedValue"
        ></span>
        <!-- Image mode: show as img -->
        <img
          v-else-if="mode === 'image' && resolvedPreviewValue"
          :src="resolvedPreviewValue"
          alt=""
          class="h-full w-full object-contain"
        />
        <video
          v-else-if="mode === 'media' && previewType === 'video' && resolvedPreviewValue"
          :src="resolvedPreviewValue"
          class="h-full w-full object-cover"
          controls
          playsinline
          muted
        ></video>
        <img
          v-else-if="mode === 'media' && resolvedPreviewValue"
          :src="resolvedPreviewValue"
          alt=""
          class="h-full w-full object-contain"
        />
        <!-- Empty placeholder -->
        <svg
          v-else
          class="text-gray-400 dark:text-dark-500"
          :class="placeholderSizeClass"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.5"
            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
      </div>
    </div>

    <!-- Controls -->
    <div class="min-w-0 flex-1 space-y-2">
      <div class="flex items-center gap-2">
        <label
          class="btn btn-secondary btn-sm"
          :class="disabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'"
          :aria-disabled="disabled ? 'true' : undefined"
        >
          <input
            type="file"
            :accept="acceptTypes"
            class="hidden"
            :disabled="disabled"
            @change="handleUpload"
          />
          <Icon name="upload" size="sm" class="mr-1.5" :stroke-width="2" />
          {{ uploadLabel }}
        </label>
        <button
          v-if="hasPreviewValue"
          type="button"
          class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
          :disabled="disabled"
          @click="$emit('update:modelValue', '')"
        >
          <Icon name="trash" size="sm" class="mr-1.5" :stroke-width="2" />
          {{ removeLabel }}
        </button>
      </div>
      <p v-if="hint" class="min-w-0 break-words text-xs text-gray-500 dark:text-gray-400" :title="hint">{{ hint }}</p>
      <p v-if="error" class="text-xs text-red-500">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeSvg } from '@/utils/sanitize'

const props = withDefaults(defineProps<{
  modelValue: string
  mode?: 'image' | 'svg' | 'media'
  size?: 'sm' | 'md'
  uploadLabel?: string
  removeLabel?: string
  hint?: string
  maxSize?: number // bytes
  previewSrc?: string
  previewContentType?: string
  hasValue?: boolean
  disabled?: boolean
}>(), {
  mode: 'image',
  size: 'md',
  uploadLabel: 'Upload',
  removeLabel: 'Remove',
  hint: '',
  maxSize: 300 * 1024,
  previewSrc: '',
  previewContentType: '',
  hasValue: undefined,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()
const error = ref('')

const acceptTypes = computed(() => {
  if (props.mode === 'svg') return '.svg'
  return 'image/*'
})
const resolvedPreviewValue = computed(() => String(props.previewSrc || props.modelValue || '').trim())
const previewType = computed<'image' | 'video'>(() => {
  const explicitContentType = String(props.previewContentType || '').trim().toLowerCase()
  if (explicitContentType.startsWith('video/')) return 'video'
  return resolvedPreviewValue.value.toLowerCase().startsWith('data:video/')
    ? 'video'
    : 'image'
})
const hasPreviewValue = computed(() => {
  if (typeof props.hasValue === 'boolean') return props.hasValue
  return resolvedPreviewValue.value !== ''
})

const sanitizedValue = computed(() =>
  props.mode === 'svg' ? sanitizeSvg(resolvedPreviewValue.value) : ''
)

const previewSizeClass = computed(() => props.size === 'sm' ? 'h-14 w-14' : 'h-20 w-20')
const innerSizeClass = computed(() => props.size === 'sm' ? 'h-7 w-7' : 'h-12 w-12')
const placeholderSizeClass = computed(() => props.size === 'sm' ? 'h-5 w-5' : 'h-8 w-8')

function handleUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  error.value = ''

  if (props.disabled) {
    input.value = ''
    return
  }

  if (!file) return

  if (props.maxSize && file.size > props.maxSize) {
    error.value = t('common.imageUpload.fileTooLarge', {
      size: formatFileSize(file.size),
      max: formatFileSize(props.maxSize),
    })
    input.value = ''
    return
  }

  const reader = new FileReader()
  if (props.mode === 'svg') {
    reader.onload = (e) => {
      const text = (e.target?.result ?? reader.result) as string
      if (text) emit('update:modelValue', text.trim())
    }
    reader.readAsText(file)
  } else {
    const imageFile = file.type.startsWith('image/')

    if (!imageFile) {
      error.value = t(props.mode === 'media' ? 'common.imageUpload.invalidMediaType' : 'common.imageUpload.invalidImageType')
      input.value = ''
      return
    }
    reader.onload = (e) => {
      emit('update:modelValue', (e.target?.result ?? reader.result) as string)
    }
    reader.readAsDataURL(file)
  }

  reader.onerror = () => {
    error.value = t('common.imageUpload.readFailed')
  }
  input.value = ''
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`
  }
  return `${(bytes / 1024).toFixed(1)} KB`
}
</script>
