<template>
  <section class="min-w-0 rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="border-b border-gray-100 p-4 dark:border-dark-700">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.savedConfigs.title') }}</h3>
      <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('taskSettings.savedConfigs.description', { type: activeTypeLabel }) }}</p>
    </div>

    <div v-if="loading" class="space-y-2 p-4">
      <div class="skeleton h-14 w-full"></div>
      <div class="skeleton h-20 w-full"></div>
      <div class="skeleton h-20 w-full"></div>
    </div>
    <div v-else-if="totalTemplateCount === 0" class="p-4">
      <div class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
        <p class="font-medium text-gray-800 dark:text-white">{{ t('taskSettings.empty.title') }}</p>
        <p class="mt-1 leading-6">{{ t('taskSettings.empty.description') }}</p>
        <button
          type="button"
          class="btn btn-primary btn-sm mt-3 w-full min-w-0 max-w-full justify-center"
          :aria-label="t('taskSettings.newTemplate')"
          :title="t('taskSettings.newTemplate')"
          @click="emit('new-template')"
        >
          <Icon name="plus" size="sm" />
          <span class="min-w-0 truncate">{{ t('taskSettings.newTemplate') }}</span>
        </button>
      </div>
    </div>
    <div v-else-if="templates.length === 0" class="p-4">
      <div data-testid="active-type-empty-state" class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
        <p class="font-medium text-gray-800 dark:text-white">{{ t('taskSettings.savedConfigs.emptyTitle') }}</p>
        <p class="mt-1 leading-6">{{ t('taskSettings.savedConfigs.emptyDescription', { type: activeTypeLabel }) }}</p>
        <button
          type="button"
          class="btn btn-secondary btn-sm mt-3 w-full min-w-0 max-w-full justify-center"
          :aria-label="t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel })"
          :title="t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel })"
          @click="emit('new-template')"
        >
          <Icon name="plus" size="sm" />
          <span class="min-w-0 truncate">{{ t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel }) }}</span>
        </button>
      </div>
    </div>
    <div v-else class="space-y-2 p-3">
      <button
        v-for="template in templates"
        :key="template.id"
        type="button"
        data-template-card="saved"
        :data-testid="`saved-template-card-${template.id}`"
        :aria-current="selectedTemplateId === template.id ? 'true' : undefined"
        :class="[
          'w-full rounded-lg border p-3 text-left transition-colors',
          selectedTemplateId === template.id
            ? 'border-primary-300 bg-primary-50 dark:border-primary-800 dark:bg-primary-900/20'
            : 'border-gray-200 bg-white hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-dark-600 dark:hover:bg-dark-700/60'
        ]"
        @click="emit('select', template)"
      >
        <span class="flex min-w-0 items-start justify-between gap-3">
          <span class="min-w-0">
            <span class="block break-all text-sm font-semibold text-gray-900 sm:truncate dark:text-white" :title="template.name">{{ template.name }}</span>
            <span class="mt-1 flex flex-wrap items-center gap-1.5">
              <span :class="['badge', taskTypeBadgeClass(template.type)]">{{ taskTypeLabel(template.type) }}</span>
              <span :class="['badge', isTemplateUsable(template) ? 'badge-success' : 'badge-warning']">
                {{ templateParameterStateLabel(template) }}
              </span>
              <span v-if="template.is_default" class="badge badge-primary">{{ t('taskSettings.defaultBadge') }}</span>
            </span>
          </span>
          <Icon name="chevronRight" size="sm" class="mt-1 shrink-0 text-gray-400" />
        </span>
      </button>

      <button
        type="button"
        class="btn btn-secondary btn-sm mt-3 w-full min-w-0 max-w-full justify-center"
        :aria-label="t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel })"
        :title="t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel })"
        @click="emit('new-template')"
      >
        <Icon name="plus" size="sm" />
        <span class="min-w-0 truncate">{{ t('taskSettings.savedConfigs.newForType', { type: activeTypeLabel }) }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { TaskTemplate } from '@/api/taskSettings'

defineProps<{
  activeTypeLabel: string
  loading: boolean
  selectedTemplateId: string
  templates: TaskTemplate[]
  totalTemplateCount: number
  isTemplateUsable: (template: TaskTemplate) => boolean
  taskTypeBadgeClass: (type: TaskTemplate['type']) => string
  taskTypeLabel: (type: TaskTemplate['type']) => string
  templateParameterStateLabel: (template: TaskTemplate) => string
}>()

const emit = defineEmits<{
  select: [template: TaskTemplate]
  'new-template': []
}>()

const { t } = useI18n()
</script>
