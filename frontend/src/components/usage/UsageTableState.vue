<template>
  <div class="flex flex-col items-center gap-3" :data-testid="testId">
    <div :class="['rounded-full p-3', iconWrapClass]">
      <Icon :name="iconName" size="lg" :class="iconClass" />
    </div>
    <div class="font-medium text-gray-700 dark:text-gray-200">{{ message }}</div>
    <button
      v-if="actionLabel"
      type="button"
      class="btn btn-secondary btn-sm"
      :data-testid="actionTestId"
      @click="handleAction"
    >
      {{ actionLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  state: 'loading' | 'error' | 'empty'
  filtered?: boolean
}>()

const emit = defineEmits<{
  retry: []
  clear: []
}>()

const { t } = useI18n()

const testId = computed(() => {
  if (props.state === 'loading') return 'usage-loading-state'
  if (props.state === 'error') return 'usage-error-state'
  return 'usage-empty-state'
})

const iconName = computed(() => {
  if (props.state === 'loading') return 'refresh'
  if (props.state === 'error') return 'exclamationCircle'
  return props.filtered ? 'filter' : 'inbox'
})

const iconWrapClass = computed(() => {
  if (props.state === 'loading') return 'bg-blue-100 dark:bg-blue-900/30'
  if (props.state === 'error') return 'bg-rose-100 dark:bg-rose-900/30'
  return 'bg-gray-100 dark:bg-dark-800'
})

const iconClass = computed(() => {
  if (props.state === 'loading') return 'animate-spin text-blue-600 dark:text-blue-400'
  if (props.state === 'error') return 'text-rose-600 dark:text-rose-400'
  return 'text-gray-400 dark:text-dark-400'
})

const message = computed(() => {
  if (props.state === 'loading') return t('usage.loading')
  if (props.state === 'error') return t('usage.failedToLoad')
  return props.filtered ? t('usage.emptyFiltered') : t('usage.empty')
})

const actionLabel = computed(() => {
  if (props.state === 'error') return t('common.refresh')
  if (props.state === 'empty' && props.filtered) return t('usage.filters.clear')
  return ''
})

const actionTestId = computed(() => {
  if (props.state === 'error') return 'usage-retry-load'
  if (props.state === 'empty' && props.filtered) return 'usage-empty-clear-filters'
  return undefined
})

function handleAction() {
  if (props.state === 'error') {
    emit('retry')
    return
  }
  emit('clear')
}
</script>
