<template>
  <div class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-950/30" role="alert" aria-live="assertive" aria-atomic="true">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-start gap-3">
        <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0 text-red-500" />
        <div class="min-w-0">
          <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ title }}</p>
          <p v-if="visibleMessage" class="mt-1 min-w-0 break-words text-sm text-red-600 dark:text-red-300/80" :title="visibleMessage">{{ visibleMessage }}</p>
        </div>
      </div>
      <button
        type="button"
        class="btn btn-secondary min-w-0 max-w-full shrink-0 justify-center"
        :aria-label="retryLabel"
        :title="retryLabel"
        @click="emit('retry')"
      >
        <span class="min-w-0 truncate">{{ retryLabel }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  message: string
  retryLabel: string
  title: string
}>()

const visibleMessage = computed(() => {
  const title = props.title.trim()
  const message = props.message.trim()
  if (!message || message === title) return ''
  return message
})

const emit = defineEmits<{
  retry: []
}>()
</script>
