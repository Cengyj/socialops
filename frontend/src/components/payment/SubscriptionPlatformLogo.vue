<template>
  <svg
    data-platform-logo
    :class="sizeClass"
    viewBox="0 0 24 24"
    fill="none"
    aria-hidden="true"
  >
    <template v-if="normalizedPlatform === 'x_twitter'">
      <path
        d="M4 4.5h4.7l11.1 15h-4.7L4 4.5Z"
        fill="currentColor"
      />
      <path
        d="M18.9 4.5 5.2 19.5"
        stroke="currentColor"
        stroke-width="2.2"
        stroke-linecap="round"
      />
    </template>

    <template v-else-if="normalizedPlatform === 'instagram'">
      <rect x="5" y="5" width="14" height="14" rx="4.2" stroke="currentColor" stroke-width="2" />
      <circle cx="12" cy="12" r="3.2" stroke="currentColor" stroke-width="2" />
      <circle cx="16.6" cy="7.6" r="1.1" fill="currentColor" />
    </template>

    <template v-else-if="normalizedPlatform === 'tiktok'">
      <path
        d="M13.8 4.5v8.2a4.1 4.1 0 1 1-3.4-4v3.1a1.4 1.4 0 1 0 1.1 1.4V4.5h2.3Z"
        fill="currentColor"
      />
      <path
        d="M13.8 4.5c.8 2.4 2.5 3.8 5 4.1v2.8c-2-.1-3.7-.8-5-2v-4.9Z"
        fill="currentColor"
      />
    </template>

    <template v-else-if="normalizedPlatform === 'facebook'">
      <path
        d="M14.3 8H17V4.5h-3.3c-3.1 0-5 1.9-5 5v2H6v3.7h2.7v4.3h4.1v-4.3h3.1l.6-3.7h-3.7V9.9c0-1.1.4-1.9 1.5-1.9Z"
        fill="currentColor"
      />
    </template>

    <template v-else>
      <circle cx="12" cy="12" r="3" fill="currentColor" />
      <circle cx="5.5" cy="12" r="2" fill="currentColor" opacity="0.72" />
      <circle cx="18.5" cy="12" r="2" fill="currentColor" opacity="0.72" />
      <path d="M7.5 12h9" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" opacity="0.72" />
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { normalizePlanPlatformLabelKey } from '@/utils/subscriptionPlanDisplay'

const props = withDefaults(defineProps<{
  platform?: string | null
  compact?: boolean
}>(), {
  platform: 'social',
  compact: false,
})

const normalizedPlatform = computed(() => normalizePlanPlatformLabelKey(props.platform))
const sizeClass = computed(() => props.compact ? 'h-4 w-4' : 'h-5 w-5')
</script>
