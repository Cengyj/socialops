<template>
  <section class="max-w-full rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="grid max-w-full grid-cols-1 gap-2 sm:grid-cols-2 xl:flex xl:overflow-x-auto">
      <button
        v-for="card in cards"
        :key="card.type"
        type="button"
        :data-testid="`task-type-${card.type}`"
        :class="[
          'min-w-0 rounded-lg border px-3 py-3 text-left transition-colors xl:min-w-[170px]',
          activeType === card.type
            ? 'border-primary-300 bg-primary-50 text-primary-900 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-100'
            : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:hover:border-dark-600 dark:hover:bg-dark-700/60'
        ]"
        @click="emit('select', card.type)"
      >
        <span class="flex min-w-0 items-center gap-2">
          <span :class="['flex h-8 w-8 items-center justify-center rounded-lg', card.tone]">
            <Icon :name="card.icon" size="sm" />
          </span>
          <span class="min-w-0 break-words text-sm font-semibold text-gray-900 dark:text-white" :title="card.label">{{ card.label }}</span>
        </span>
        <span class="mt-2 block min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="card.requirement">{{ card.requirement }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

export type TaskSettingsTaskType = 'follow' | 'like' | 'retweet' | 'post' | 'update_profile' | 'update_avatar' | 'update_banner'
export type TaskSettingsTaskTypeIcon = 'checkCircle' | 'userPlus' | 'sync' | 'chatBubble' | 'edit' | 'userCircle' | 'grid'

export interface TaskTypeSelectorCard {
  type: TaskSettingsTaskType
  label: string
  requirement: string
  icon: TaskSettingsTaskTypeIcon
  tone: string
}

defineProps<{
  activeType: TaskSettingsTaskType
  cards: TaskTypeSelectorCard[]
}>()

const emit = defineEmits<{
  select: [type: TaskSettingsTaskType]
}>()
</script>
