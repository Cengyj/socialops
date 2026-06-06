<template>
  <AppLayout>
    <div class="space-y-5">
      <div v-if="loadError" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-950/30">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('taskSettings.failedToLoad') }}</p>
            <p class="mt-1 text-sm text-red-600 dark:text-red-300/80">{{ loadError }}</p>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="loadTemplates">{{ t('common.retry') }}</button>
        </div>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex gap-2 overflow-x-auto">
          <button
            v-for="meta in taskTypeCards"
            :key="meta.type"
            type="button"
            :data-testid="`task-type-${meta.type}`"
            :class="[
              'min-w-[170px] rounded-lg border px-3 py-3 text-left transition-colors',
              activeType === meta.type
                ? 'border-primary-300 bg-primary-50 text-primary-900 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-100'
                : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:hover:border-dark-600 dark:hover:bg-dark-700/60'
            ]"
            @click="chooseType(meta.type)"
          >
            <span class="flex items-center gap-2">
              <span :class="['flex h-8 w-8 items-center justify-center rounded-lg', meta.tone]">
                <Icon :name="meta.icon" size="sm" />
              </span>
              <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ meta.label }}</span>
            </span>
            <span class="mt-2 block text-xs leading-5 text-gray-500 dark:text-dark-400">{{ meta.requirement }}</span>
          </button>
        </div>
      </section>

      <section data-testid="template-stats" class="grid gap-2 sm:grid-cols-3">
        <div
          v-for="stat in templateStats"
          :key="stat.label"
          :data-testid="stat.testId"
          class="rounded-lg border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
              <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ stat.meta }}</div>
            </div>
            <div class="shrink-0 text-lg font-semibold leading-6 text-gray-900 dark:text-white">{{ stat.value }}</div>
          </div>
        </div>
      </section>

      <div class="grid items-start gap-4 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
        <section class="min-w-0 rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 p-4 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.savedConfigs.title') }}</h3>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('taskSettings.savedConfigs.description', { type: taskTypeLabel(activeType) }) }}</p>
          </div>

          <div v-if="loading" class="space-y-2 p-4">
            <div class="skeleton h-14 w-full"></div>
            <div class="skeleton h-20 w-full"></div>
            <div class="skeleton h-20 w-full"></div>
          </div>
          <div v-else-if="templates.length === 0" class="p-4">
            <div class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
              <p class="font-medium text-gray-800 dark:text-white">{{ t('taskSettings.empty.title') }}</p>
              <p class="mt-1 leading-6">{{ t('taskSettings.empty.description') }}</p>
              <button type="button" class="btn btn-primary btn-sm mt-3 w-full justify-center" @click="newTemplate">
                <Icon name="plus" size="sm" />
                <span>{{ t('taskSettings.newTemplate') }}</span>
              </button>
            </div>
          </div>
          <div v-else-if="orderedTemplates.length === 0" class="p-4">
            <div data-testid="active-type-empty-state" class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
              <p class="font-medium text-gray-800 dark:text-white">{{ t('taskSettings.savedConfigs.emptyTitle') }}</p>
              <p class="mt-1 leading-6">{{ t('taskSettings.savedConfigs.emptyDescription', { type: taskTypeLabel(activeType) }) }}</p>
              <button type="button" class="btn btn-secondary btn-sm mt-3 w-full justify-center" @click="newTemplate">
                <Icon name="plus" size="sm" />
                <span>{{ t('taskSettings.savedConfigs.newForType', { type: taskTypeLabel(activeType) }) }}</span>
              </button>
            </div>
          </div>
          <div v-else class="space-y-2 p-3">
            <button
              v-for="template in orderedTemplates"
              :key="template.id"
              type="button"
              data-template-card="saved"
              :data-testid="`saved-template-card-${template.id}`"
              :class="[
                'w-full rounded-lg border p-3 text-left transition-colors',
                selectedTemplateId === template.id
                  ? 'border-primary-300 bg-primary-50 dark:border-primary-800 dark:bg-primary-900/20'
                  : 'border-gray-200 bg-white hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-dark-600 dark:hover:bg-dark-700/60'
              ]"
              @click="selectTemplate(template)"
            >
              <span class="flex min-w-0 items-start justify-between gap-3">
                <span class="min-w-0">
                  <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ template.name }}</span>
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

            <button type="button" class="btn btn-secondary btn-sm mt-3 w-full justify-center" @click="newTemplate">
              <Icon name="plus" size="sm" />
              <span>{{ t('taskSettings.savedConfigs.newForType', { type: taskTypeLabel(activeType) }) }}</span>
            </button>
          </div>
        </section>

        <section class="min-w-0 rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 p-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ taskTypeLabel(form.type) }}</h2>
                  <span :class="['badge', currentTemplateReady ? 'badge-success' : 'badge-warning']">
                    {{ currentTemplateReady ? t('taskSettings.status.ready') : t('taskSettings.status.needsInput') }}
                  </span>
                  <span v-if="form.isDefault" class="badge badge-primary">{{ t('taskSettings.defaultBadge') }}</span>
                </div>
                <p class="mt-1 max-w-2xl text-sm leading-6 text-gray-500 dark:text-dark-400">{{ activeTypeDescription }}</p>
              </div>
              <div data-testid="editor-template-actions" class="flex flex-wrap gap-2">
                <button
                  type="button"
                  class="btn btn-primary btn-sm"
                  data-testid="save-template-button"
                  :disabled="!canSave || saving"
                  :title="saveDisabledReason || undefined"
                  @click="saveTemplate"
                >
                  {{ saving ? t('common.saving') : t('taskSettings.save') }}
                </button>
                <button v-if="activePoolKind" type="button" class="btn btn-secondary btn-sm" data-testid="validation-button" :disabled="saving" @click="validateCurrent">{{ t('taskSettings.validate') }}</button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  data-testid="copy-template-button"
                  :disabled="!selectedTemplateId || saving"
                  @click="copyCurrentTemplate"
                >
                  <Icon name="copy" size="sm" />
                  <span>{{ t('taskSettings.copy') }}</span>
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  data-testid="set-default-button"
                  :disabled="!selectedTemplateId || saving || form.isDefault"
                  @click="setDefault"
                >
                  <Icon name="checkCircle" size="sm" />
                  <span>{{ t('taskSettings.setDefault') }}</span>
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm text-red-600 hover:border-red-200 hover:bg-red-50 dark:text-red-300 dark:hover:border-red-900/60 dark:hover:bg-red-950/30"
                  data-testid="delete-template-button"
                  :disabled="!selectedTemplateId || saving"
                  @click="deleteCurrentTemplate"
                >
                  <Icon name="trash" size="sm" />
                  <span>{{ t('common.delete') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="space-y-5 p-4 lg:p-5">
            <div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_240px]">
              <div>
                <label class="input-label" for="template-name">{{ t('taskSettings.form.name') }}</label>
                <input
                  id="template-name"
                  v-model="form.name"
                  type="text"
                  class="input"
                  data-testid="template-name-input"
                  :placeholder="t('taskSettings.form.namePlaceholder')"
                  @input="resetValidationResult"
                />
              </div>
              <label class="flex items-start gap-3 rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-900/50">
                <input v-model="form.isDefault" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" @change="resetValidationResult" />
                <span>
                  <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ t('taskSettings.defaultToggle') }}</span>
                  <span class="mt-1 block text-xs leading-5 text-gray-500 dark:text-dark-400">{{ defaultImpactText }}</span>
                </span>
              </label>
            </div>

            <div v-if="activePoolKind" data-testid="parameter-pool-manager" class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="flex flex-col gap-3 border-b border-gray-100 p-4 dark:border-dark-700 md:flex-row md:items-start md:justify-between">
                <div>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ activeValueLabel }}</h3>
                  <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ activeValueHelp }}</p>
                  <p
                    v-if="form.type === 'post'"
                    data-testid="post-text-only-hint"
                    class="mt-3 rounded-lg border border-blue-200 bg-blue-50 p-3 text-xs leading-5 text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200"
                  >
                    {{ t('taskSettings.media.postTextOnlyHint') }}
                  </p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <input ref="fileInputRef" type="file" class="hidden" accept=".txt,text/plain,.csv,text/csv" @change="handleFileImport" />
                  <button type="button" class="btn btn-secondary btn-sm" data-testid="import-button" @click="fileInputRef?.click()">
                    <Icon name="upload" size="sm" />
                    <span>{{ t('taskSettings.importFile') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" data-testid="view-all-button" :disabled="activeValues.length === 0" @click="viewAllDialogOpen = true">
                    <Icon name="eye" size="sm" />
                    <span>{{ t('taskSettings.viewAll') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" data-testid="dedupe-button" :disabled="poolAnalysis.duplicateCount === 0" @click="dedupeValues">
                    <Icon name="sparkles" size="sm" />
                    <span>{{ t('taskSettings.dedupe') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" data-testid="clear-pool-button" :disabled="activeValues.length === 0" @click="clearValues">
                    <Icon name="x" size="sm" />
                    <span>{{ t('taskSettings.clearValues') }}</span>
                  </button>
                </div>
              </div>

              <div class="grid gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_260px]">
                <textarea
                  v-model="activeValuesText"
                  :data-testid="activePoolKind === 'targets' ? 'target-pool-textarea' : 'content-pool-textarea'"
                  class="input min-h-[300px] resize-y font-mono leading-6"
                  :placeholder="activeValuePlaceholder"
                  @input="resetValidationResult"
                ></textarea>

                <div class="space-y-3">
                  <div class="grid grid-cols-2 gap-2">
                    <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-valid">
                      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('taskSettings.pool.valid') }}</p>
                      <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ poolAnalysis.validCount }}</p>
                    </div>
                    <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-empty-lines">
                      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('taskSettings.pool.emptyLines') }}</p>
                      <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ poolAnalysis.emptyLineCount }}</p>
                    </div>
                    <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-duplicates">
                      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('taskSettings.pool.duplicates') }}</p>
                      <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ poolAnalysis.duplicateCount }}</p>
                    </div>
                    <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-too-long">
                      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('taskSettings.pool.tooLong') }}</p>
                      <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ poolAnalysis.tooLongCount }}</p>
                    </div>
                    <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('taskSettings.pool.remaining') }}</p>
                      <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ poolAnalysis.remaining }}</p>
                    </div>
                  </div>

                  <div
                    data-testid="pool-capacity"
                    :class="[
                      'rounded-lg border p-3 text-sm',
                      poolAnalysis.overCapacity
                        ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300'
                        : 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-300'
                    ]"
                  >
                    {{ poolCapacityMessage }}
                  </div>

                  <div
                    v-if="poolAnalysis.emptyLineCount > 0"
                    data-testid="pool-empty-lines-hint"
                    class="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200"
                  >
                    {{ t('taskSettings.pool.emptyLinesHint', { count: poolAnalysis.emptyLineCount }) }}
                  </div>
                  <div v-if="poolAnalysis.tooLongCount > 0" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
                    {{ t('taskSettings.pool.tooLongHint', { max: MAX_TEMPLATE_VALUE_LENGTH }) }}
                  </div>
                  <div v-else-if="poolAnalysis.duplicateCount > 0" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
                    {{ t('taskSettings.pool.duplicateHint', { count: poolAnalysis.duplicateCount }) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <aside class="min-w-0 space-y-4">
          <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.title') }}</div>
            <div class="mt-3 space-y-3 text-sm">
              <div class="flex justify-between gap-3">
                <span class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.type') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ taskTypeLabel(form.type) }}</span>
              </div>
              <div v-if="activePoolKind" class="flex justify-between gap-3">
                <span class="text-gray-500 dark:text-dark-400">{{ activePoolKind === 'targets' ? t('taskSettings.summary.targets') : t('taskSettings.summary.contents') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ activeValues.length }}</span>
              </div>
              <div v-else class="flex justify-between gap-3">
                <span class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.params') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ t('taskSettings.counts.none') }}</span>
              </div>
              <div class="flex justify-between gap-3">
                <span class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.default') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ form.isDefault ? t('common.yes') : t('common.no') }}</span>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div v-if="validationResult" :class="['rounded-lg border p-3 text-sm', validationResult.valid ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300' : 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300']">
              <div class="font-medium">{{ validationResult.valid ? t('taskSettings.validation.valid') : t('taskSettings.validation.invalid') }}</div>
              <ul v-if="validationResult.errors.length > 0" class="mt-2 list-disc space-y-1 pl-5">
                <li v-for="error in validationResult.errors" :key="error">{{ error }}</li>
              </ul>
            </div>
            <div v-else-if="saveDisabledReason" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300">
              {{ saveDisabledReason }}
            </div>
            <div v-else class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300">
              {{ t('taskSettings.validation.valid') }}
            </div>

            <div class="mt-3 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm leading-6 text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200">
              {{ t('taskSettings.summary.executionHint') }}
            </div>
          </section>
        </aside>
      </div>
    </div>

    <BaseDialog :show="viewAllDialogOpen" :title="activeValueLabel" width="wide" @close="viewAllDialogOpen = false">
      <div v-if="activeValues.length === 0" class="rounded-lg border border-gray-200 bg-gray-50 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
        {{ t('taskSettings.pool.empty') }}
      </div>
      <div v-else class="max-h-[520px] divide-y divide-gray-100 overflow-auto rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
        <div v-for="(value, index) in activeValues" :key="`${index}-${value}`" class="grid grid-cols-[64px_minmax(0,1fr)] gap-3 px-3 py-2 text-sm">
          <span class="text-gray-500 dark:text-dark-400">#{{ index + 1 }}</span>
          <span class="break-all font-medium text-gray-900 dark:text-white">{{ value }}</span>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-primary" @click="viewAllDialogOpen = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('taskSettings.deleteDialog.title')" @close="closeDeleteDialog">
      <div v-if="templateToDelete" class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('taskSettings.deleteDialog.description', { name: templateToDelete.name }) }}</p>
        <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('taskSettings.deleteDialog.warning') }}
        </div>
        <dl class="grid gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex min-w-0 justify-between gap-3">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.form.name') }}</dt>
            <dd class="min-w-0 truncate text-gray-900 dark:text-white">{{ templateToDelete.name }}</dd>
          </div>
          <div class="flex min-w-0 justify-between gap-3">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.type') }}</dt>
            <dd class="min-w-0 truncate text-gray-900 dark:text-white">{{ taskTypeLabel(templateToDelete.type) }}</dd>
          </div>
          <div v-if="templateParameterSummary(templateToDelete).label" class="flex min-w-0 justify-between gap-3">
            <dt class="text-gray-500 dark:text-dark-400">{{ templateParameterSummary(templateToDelete).label }}</dt>
            <dd class="text-gray-900 dark:text-white">{{ templateParameterSummary(templateToDelete).count }}</dd>
          </div>
        </dl>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="closeDeleteDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-danger" :disabled="saving" @click="confirmDeleteTemplate">
          {{ saving ? t('common.processing') : t('common.delete') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import taskSettingsAPI from '@/api/taskSettings'
import type { TaskTemplate, TaskTemplateInput, TaskTemplateType, TaskTemplateValidationResult } from '@/api/taskSettings'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'

type ParameterTaskTemplateType = Extract<TaskTemplateType, 'follow' | 'like' | 'retweet' | 'post'>
type ParameterTaskTemplate = TaskTemplate & { type: ParameterTaskTemplateType }
type IconName = 'checkCircle' | 'userPlus' | 'sync' | 'chatBubble'

const { t } = useI18n()
const appStore = useAppStore()

const templates = ref<ParameterTaskTemplate[]>([])
const selectedTemplateId = ref('')
const activeType = ref<ParameterTaskTemplateType>('follow')
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const validationResult = ref<TaskTemplateValidationResult | null>(null)
const viewAllDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const templateToDelete = ref<ParameterTaskTemplate | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const form = reactive({
  id: '',
  name: '',
  type: 'follow' as ParameterTaskTemplateType,
  targetsText: '',
  contentsText: '',
  isDefault: false,
})

const MAX_TEMPLATE_POOL_VALUES = 500
const MAX_TEMPLATE_VALUE_LENGTH = 2048
const TASK_TYPES: ParameterTaskTemplateType[] = ['follow', 'like', 'retweet', 'post']
const targetTypes = new Set<ParameterTaskTemplateType>(['follow', 'like', 'retweet'])

const activeTypeTemplates = computed(() => templates.value.filter(template => template.type === activeType.value))
const orderedTemplates = computed(() => [...activeTypeTemplates.value].sort((a, b) => {
  const typeDiff = TASK_TYPES.indexOf(a.type) - TASK_TYPES.indexOf(b.type)
  if (typeDiff !== 0) return typeDiff
  if (a.is_default !== b.is_default) return a.is_default ? -1 : 1
  return a.name.localeCompare(b.name)
}))
const targetValues = computed(() => splitTargetValues(form.targetsText))
const contentValues = computed(() => splitContentValues(form.contentsText))
const activePoolKind = computed<'targets' | 'contents' | null>(() => {
  if (targetTypes.has(form.type)) return 'targets'
  if (form.type === 'post') return 'contents'
  return null
})
const activeValues = computed(() => {
  if (activePoolKind.value === 'targets') return targetValues.value
  if (activePoolKind.value === 'contents') return contentValues.value
  return []
})
const activeValuesText = computed({
  get: () => activePoolKind.value === 'targets' ? form.targetsText : form.contentsText,
  set: (value: string) => {
    if (activePoolKind.value === 'targets') {
      form.targetsText = value
    } else if (activePoolKind.value === 'contents') {
      form.contentsText = value
    }
  },
})
const activeTypeDescription = computed(() => t(`taskSettings.typeDescriptions.${form.type}`))
const activeValueLabel = computed(() => {
  if (form.type === 'post') return t('taskSettings.form.contents')
  if (targetTypes.has(form.type)) return t('taskSettings.form.targets')
  return t('taskSettings.pool.title')
})
const activeValueHelp = computed(() => {
  if (form.type === 'follow') return t('taskSettings.typeRequirements.follow')
  if (form.type === 'like') return t('taskSettings.typeRequirements.like')
  if (form.type === 'retweet') return t('taskSettings.typeRequirements.retweet')
  if (form.type === 'post') return t('taskSettings.typeRequirements.post')
  return ''
})
const activeValuePlaceholder = computed(() => {
  if (form.type === 'post') return t('taskSettings.form.contentsPlaceholder')
  if (form.type === 'follow') return t('taskSettings.form.followTargetsPlaceholder')
  return t('taskSettings.form.tweetTargetsPlaceholder')
})

const taskTypeCards = computed(() => TASK_TYPES.map(type => ({
  type,
  label: taskTypeLabel(type),
  description: t(`taskSettings.typeDescriptions.${type}`),
  requirement: t(`taskSettings.typeRequirements.${type}`),
  icon: taskTypeIcon(type),
  tone: taskTypeTone(type),
})))

const templateStats = computed(() => [
  {
    testId: 'template-stats-total',
    label: t('taskSettings.stats.total'),
    meta: t('taskSettings.stats.totalMeta', { type: taskTypeLabel(activeType.value) }),
    value: activeTypeTemplates.value.length,
  },
  {
    testId: 'template-stats-defaults',
    label: t('taskSettings.stats.defaults'),
    meta: t('taskSettings.stats.defaultsMeta', { type: taskTypeLabel(activeType.value) }),
    value: activeTypeTemplates.value.filter(template => template.is_default).length,
  },
  {
    testId: 'template-stats-unusable',
    label: t('taskSettings.stats.unusable'),
    meta: t('taskSettings.stats.unusableMeta', { type: taskTypeLabel(activeType.value) }),
    value: activeTypeTemplates.value.filter(template => !isTemplateUsable(template)).length,
  },
])

const poolEmptyLineCount = computed(() => countIgnoredEmptyValues(activeValuesText.value, activePoolKind.value))
const poolAnalysis = computed(() => analyzePool(activeValues.value, poolEmptyLineCount.value))
const poolCapacityMessage = computed(() => {
  if (poolAnalysis.value.overCapacity) return t('taskSettings.validation.tooManyValues', { max: MAX_TEMPLATE_POOL_VALUES })
  return t('taskSettings.pool.capacityHint', { count: activeValues.value.length, max: MAX_TEMPLATE_POOL_VALUES })
})
const defaultImpactText = computed(() => t('taskSettings.defaultImpact', { type: taskTypeLabel(form.type) }))
const saveDisabledReason = computed(() => {
  if (form.name.trim() === '') return t('taskSettings.validation.nameRequired')
  if (targetTypes.has(form.type) && targetValues.value.length === 0) return t('taskSettings.validation.targetsRequired')
  if (form.type === 'post' && contentValues.value.length === 0) return t('taskSettings.validation.contentsRequired')
  if (activeValues.value.length > MAX_TEMPLATE_POOL_VALUES) return t('taskSettings.validation.tooManyValues', { max: MAX_TEMPLATE_POOL_VALUES })
  if (activeValues.value.some(value => Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH)) return t('taskSettings.validation.valueTooLong', { max: MAX_TEMPLATE_VALUE_LENGTH })
  return ''
})
const canSave = computed(() => !saveDisabledReason.value)
const currentTemplateReady = computed(() => !saveDisabledReason.value)

void loadTemplates()

async function loadTemplates() {
  loading.value = true
  loadError.value = ''
  try {
    const loadedTemplates = await taskSettingsAPI.listTemplates()
    templates.value = Array.isArray(loadedTemplates) ? loadedTemplates.filter(isParameterTaskTemplate) : []
    const selected = templates.value.find(template => template.id === selectedTemplateId.value)
    if (selected) {
      selectTemplate(selected)
    } else {
      selectBestTemplateForType(activeType.value)
    }
  } catch (error) {
    recordClientDiagnostic('task_settings.load', error)
    loadError.value = extractSafeApiErrorMessage(error, t('taskSettings.failedToLoad'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

function chooseType(type: ParameterTaskTemplateType) {
  activeType.value = type
  selectBestTemplateForType(type)
}

function selectBestTemplateForType(type: ParameterTaskTemplateType) {
  const candidates = templates.value.filter(template => template.type === type)
  const next = candidates.find(template => template.is_default) ?? candidates[0]
  if (next) {
    selectTemplate(next)
  } else {
    resetFormForType(type)
  }
}

function selectTemplate(template: TaskTemplate) {
  if (!isParameterTaskTemplate(template)) {
    selectBestTemplateForType(activeType.value)
    return
  }
  selectedTemplateId.value = template.id
  activeType.value = template.type
  form.id = template.id
  form.name = template.name
  form.type = template.type
  form.targetsText = targetTypes.has(template.type) ? (template.params.targets ?? []).join('\n') : ''
  form.contentsText = template.type === 'post' ? (template.params.contents ?? []).join('\n') : ''
  form.isDefault = template.is_default
  validationResult.value = null
}

function resetFormForType(type: ParameterTaskTemplateType) {
  selectedTemplateId.value = ''
  activeType.value = type
  form.id = ''
  form.name = ''
  form.type = type
  form.targetsText = ''
  form.contentsText = ''
  form.isDefault = activeTypeTemplates.value.length === 0
  validationResult.value = null
}

function newTemplate() {
  resetFormForType(activeType.value)
}

async function validateCurrent() {
  try {
    validationResult.value = await taskSettingsAPI.validateTemplate(buildInput())
    appStore.showSuccess(validationResult.value.valid ? t('taskSettings.validation.valid') : t('taskSettings.validation.invalid'))
  } catch (error) {
    recordClientDiagnostic('task_settings.validate', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('taskSettings.validation.failed')))
  }
}

async function saveTemplate() {
  if (!canSave.value || saving.value) return
  saving.value = true
  try {
    const result = await taskSettingsAPI.saveTemplate(buildInput())
    appStore.showSuccess(t('taskSettings.saved'))
    await loadTemplates()
    selectTemplate(result)
  } catch (error) {
    recordClientDiagnostic('task_settings.save', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('taskSettings.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function copyCurrentTemplate() {
  if (!selectedTemplateId.value || saving.value) return
  saving.value = true
  try {
    const result = await taskSettingsAPI.copyTemplate(selectedTemplateId.value)
    appStore.showSuccess(t('taskSettings.copied'))
    await loadTemplates()
    selectTemplate(result)
  } catch (error) {
    recordClientDiagnostic('task_settings.copy', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('taskSettings.copyFailed')))
  } finally {
    saving.value = false
  }
}

async function setDefault() {
  if (!selectedTemplateId.value || saving.value) return
  saving.value = true
  try {
    const result = await taskSettingsAPI.setDefaultTemplate(selectedTemplateId.value)
    appStore.showSuccess(t('taskSettings.defaultSaved'))
    await loadTemplates()
    selectTemplate(result)
  } catch (error) {
    recordClientDiagnostic('task_settings.default', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('taskSettings.defaultFailed')))
  } finally {
    saving.value = false
  }
}

async function deleteCurrentTemplate() {
  if (!selectedTemplateId.value || saving.value) return
  const template = templates.value.find(item => item.id === selectedTemplateId.value)
  templateToDelete.value = template ?? {
    id: selectedTemplateId.value,
    name: form.name,
    type: form.type,
    params: buildTemplateParams(),
    is_default: form.isDefault,
    created_at: '',
    updated_at: '',
  }
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  if (saving.value) return
  deleteDialogOpen.value = false
  templateToDelete.value = null
}

async function confirmDeleteTemplate() {
  if (saving.value || !templateToDelete.value) return
  const template = templateToDelete.value
  saving.value = true
  try {
    await taskSettingsAPI.deleteTemplate(template.id)
    appStore.showSuccess(t('taskSettings.deleted'))
    deleteDialogOpen.value = false
    templateToDelete.value = null
    await loadTemplates()
  } catch (error) {
    recordClientDiagnostic('task_settings.delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('taskSettings.deleteFailed')))
  } finally {
    saving.value = false
  }
}

async function handleFileImport(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !activePoolKind.value) return
  try {
    const text = await file.text()
    const imported = activePoolKind.value === 'targets' ? splitTargetValues(text) : splitContentValues(text)
    activeValuesText.value = [activeValuesText.value.trim(), imported.join('\n')].filter(Boolean).join('\n')
    resetValidationResult()
    appStore.showSuccess(t('taskSettings.imported', { count: imported.length }))
  } catch (error) {
    recordClientDiagnostic('task_settings.import_file', error)
    appStore.showError(t('taskSettings.importFailed'))
  } finally {
    input.value = ''
  }
}

function clearValues() {
  activeValuesText.value = ''
  resetValidationResult()
}

function dedupeValues() {
  const before = activeValues.value.length
  const deduped = Array.from(new Set(activeValues.value))
  activeValuesText.value = deduped.join('\n')
  resetValidationResult()
  appStore.showSuccess(t('taskSettings.deduped', { count: before - deduped.length }))
}

function resetValidationResult() {
  validationResult.value = null
}

function buildTemplateParams(): TaskTemplateInput['params'] {
  if (targetTypes.has(form.type)) return { targets: targetValues.value }
  if (form.type === 'post') return { contents: contentValues.value }
  return {}
}

function buildInput(): TaskTemplateInput {
  return {
    id: form.id || undefined,
    name: form.name.trim(),
    type: form.type,
    params: buildTemplateParams(),
    is_default: form.isDefault,
  }
}

function splitTargetValues(value: string): string[] {
  return value
    .split(/\r?\n|,/)
    .map(item => item.trim())
    .filter(Boolean)
}

function splitContentValues(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

function countIgnoredEmptyValues(value: string, kind: 'targets' | 'contents' | null) {
  if (!value || !kind) return 0
  const parts = kind === 'targets' ? value.split(/\r?\n|,/) : value.split(/\r?\n/)
  return parts.filter(item => item.trim() === '').length
}

function analyzePool(values: string[], emptyLineCount = 0) {
  const seen = new Set<string>()
  let duplicateCount = 0
  let tooLongCount = 0
  for (const value of values) {
    if (seen.has(value)) duplicateCount += 1
    seen.add(value)
    if (Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH) tooLongCount += 1
  }
  return {
    validCount: values.length - tooLongCount,
    emptyLineCount,
    duplicateCount,
    tooLongCount,
    remaining: Math.max(0, MAX_TEMPLATE_POOL_VALUES - values.length),
    overCapacity: values.length > MAX_TEMPLATE_POOL_VALUES,
  }
}

function templateParameterValues(template: ParameterTaskTemplate): string[] {
  if (targetTypes.has(template.type)) return template.params.targets ?? []
  if (template.type === 'post') return template.params.contents ?? []
  return []
}

function templateParameterSummary(template: ParameterTaskTemplate): { label: string; count: string | number } {
  if (targetTypes.has(template.type)) {
    return { label: t('taskSettings.summary.targets'), count: (template.params.targets ?? []).length }
  }
  if (template.type === 'post') {
    return { label: t('taskSettings.summary.contents'), count: (template.params.contents ?? []).length }
  }
  return { label: '', count: '' }
}

function isTemplateUsable(template: ParameterTaskTemplate) {
  const values = templateParameterValues(template)
  if (values.length === 0 || values.length > MAX_TEMPLATE_POOL_VALUES) return false
  return values.every(value => Array.from(value).length <= MAX_TEMPLATE_VALUE_LENGTH)
}

function templateParameterStateLabel(template: ParameterTaskTemplate) {
  return isTemplateUsable(template)
    ? t('taskSettings.savedConfigs.readyState')
    : t('taskSettings.savedConfigs.needsInputState')
}

function taskTypeLabel(type: TaskTemplateType | string) {
  return t(`taskSettings.types.${type}`, type)
}

function taskTypeIcon(type: ParameterTaskTemplateType): IconName {
  if (type === 'follow') return 'userPlus'
  if (type === 'retweet') return 'sync'
  if (type === 'post') return 'chatBubble'
  return 'checkCircle'
}

function taskTypeTone(type: ParameterTaskTemplateType) {
  if (type === 'post') return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  if (type === 'retweet') return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300'
  return 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
}

function taskTypeBadgeClass(type: ParameterTaskTemplateType) {
  if (type === 'post') return 'badge-primary'
  if (type === 'retweet') return 'badge-gray'
  return 'badge-gray'
}

function isParameterTaskTemplate(template: TaskTemplate): template is ParameterTaskTemplate {
  return (TASK_TYPES as TaskTemplateType[]).includes(template.type)
}
</script>
