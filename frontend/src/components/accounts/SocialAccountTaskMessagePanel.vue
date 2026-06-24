<template>
  <div
    :class="['min-w-0 break-words rounded-lg border p-3 text-sm', workbenchTaskMessagePanelClass(status)]"
    role="status"
    aria-live="polite"
    aria-atomic="true"
    :title="message"
  >
    {{ message }}
  </div>
</template>

<script setup lang="ts">
import { workbenchTaskMessagePanelClass } from '@/utils/accountWorkbenchStatus'

defineProps<{
  message: string
  status?: string | null
}>()
</script>
