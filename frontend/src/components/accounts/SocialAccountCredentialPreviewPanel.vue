<template>
  <div :data-testid="`${testIdPrefix}-credential-preview`" class="rounded-lg border border-gray-200 bg-white p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ title }}</div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ hint }}</div>
      </div>
    </div>
    <div :data-testid="`${testIdPrefix}-credential-preview-grid`" class="grid gap-3 md:grid-cols-2">
      <div
        v-for="credential in credentials"
        :key="credential.key"
        :data-testid="`${testIdPrefix}-credential-${credential.key}`"
        class="relative min-w-0 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700/60"
      >
        <div>
          <div class="min-w-0 pr-40">
            <div class="font-medium text-gray-900 dark:text-white">{{ credential.label }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ credential.description }}</div>
          </div>
          <div class="absolute right-3 top-3 flex shrink-0 flex-wrap justify-end gap-2">
            <button
              type="button"
              :data-testid="`${testIdPrefix}-credential-${credential.key}-copy`"
              class="btn btn-secondary btn-sm"
              :disabled="!credential.copyable"
              :title="credential.copyable ? credential.copyTitle : emptyCopyLabel"
              @click="emit('copy', credential.key)"
            >
              <Icon name="copy" size="sm" />
              <span>{{ copyLabel }}</span>
            </button>
            <button
              v-if="showExecutionAuthRefresh && credential.key === 'executionAuth'"
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="refreshing || !!refreshDisabledReason"
              :title="refreshDisabledReason || refreshTitle"
              @click="emit('refreshExecutionAuth')"
            >
              <Icon name="refresh" size="sm" :class="refreshing ? 'animate-spin' : ''" />
              <span>{{ refreshing ? refreshProcessingLabel : refreshLabel }}</span>
            </button>
          </div>
        </div>
        <div class="mt-3 flex min-h-[28px] flex-wrap gap-1.5">
          <span
            v-for="meta in credential.meta"
            :key="meta"
            class="rounded-md bg-white px-2 py-1 text-xs font-medium text-gray-600 ring-1 ring-gray-200 dark:bg-dark-800 dark:text-gray-200 dark:ring-dark-600"
          >
            {{ meta }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type {
  SocialAccountCredentialPreview,
  SocialAccountCredentialPreviewKey,
} from '@/utils/socialAccountCredentials'

withDefaults(defineProps<{
  credentials: SocialAccountCredentialPreview[]
  testIdPrefix: string
  title: string
  hint: string
  copyLabel: string
  emptyCopyLabel: string
  showExecutionAuthRefresh?: boolean
  refreshing?: boolean
  refreshDisabledReason?: string
  refreshTitle?: string
  refreshLabel?: string
  refreshProcessingLabel?: string
}>(), {
  showExecutionAuthRefresh: false,
  refreshing: false,
  refreshDisabledReason: '',
  refreshTitle: '',
  refreshLabel: '',
  refreshProcessingLabel: '',
})

const emit = defineEmits<{
  copy: [key: SocialAccountCredentialPreviewKey]
  refreshExecutionAuth: []
}>()
</script>
