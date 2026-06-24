<template>
  <AppLayout>
    <div class="space-y-5">
      <LoadErrorBanner
        v-if="loadError"
        :title="t('taskSettings.failedToLoad')"
        :message="loadError"
        :retry-label="t('common.retry')"
        @retry="loadTemplates"
      />

      <TaskTypeSelector :active-type="activeType" :cards="taskTypeCards" @select="chooseType" />

      <TemplateStatsGrid :stats="templateStats" />

      <div class="grid items-start gap-4 2xl:grid-cols-[320px_minmax(0,1fr)_320px]">
        <SavedTemplateList
          :active-type-label="taskTypeLabel(activeType)"
          :is-template-usable="isTemplateUsable"
          :loading="loading"
          :selected-template-id="selectedTemplateId"
          :task-type-badge-class="taskTypeBadgeClass"
          :task-type-label="taskTypeLabel"
          :template-parameter-state-label="templateParameterStateLabel"
          :templates="orderedTemplates"
          :total-template-count="templates.length"
          @new-template="newTemplate"
          @select="selectTemplate"
        />

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
              <TemplateEditorActions
                :can-save="canSave"
                :has-selected-template="!!selectedTemplateId"
                :is-default="form.isDefault"
                :operation="templateOperation"
                :save-disabled-reason="saveDisabledReason"
                :saving="saving"
                @copy="copyCurrentTemplate"
                @delete="deleteCurrentTemplate"
                @save="saveTemplate"
                @set-default="setDefault"
                @validate="validateCurrent"
              />
            </div>
          </div>

          <div class="space-y-5 p-4 lg:p-5">
            <div
              v-if="editorOperationError"
              class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
              role="alert"
              aria-live="assertive"
              aria-atomic="true"
              :title="editorOperationError"
            >
              {{ editorOperationError }}
            </div>

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
                  <p class="mt-1 min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="activeValueHelp">{{ activeValueHelp }}</p>
                </div>
                <div class="flex min-w-0 flex-wrap gap-2">
                  <input ref="fileInputRef" type="file" class="hidden" accept=".txt,text/plain,.csv,text/csv" :disabled="saving" @change="handleFileImport" />
                  <button type="button" class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center" data-testid="import-button" :aria-label="importButtonTitle" :title="importButtonTitle" :disabled="saving" @click="fileInputRef?.click()">
                    <Icon name="upload" size="sm" />
                    <span class="min-w-0 truncate">{{ t('taskSettings.importFile') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center" data-testid="view-all-button" :aria-label="viewAllButtonTitle" :title="viewAllButtonTitle" :disabled="saving || activeValues.length === 0" @click="viewAllDialogOpen = true">
                    <Icon name="eye" size="sm" />
                    <span class="min-w-0 truncate">{{ t('taskSettings.viewAll') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center" data-testid="dedupe-button" :aria-label="dedupeButtonTitle" :title="dedupeButtonTitle" :disabled="saving || poolAnalysis.duplicateCount === 0" @click="dedupeValues">
                    <Icon name="sparkles" size="sm" />
                    <span class="min-w-0 truncate">{{ t('taskSettings.dedupe') }}</span>
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center" data-testid="clear-pool-button" :aria-label="clearPoolButtonTitle" :title="clearPoolButtonTitle" :disabled="saving || !canClearActiveValues" @click="clearValues">
                    <Icon name="x" size="sm" />
                    <span class="min-w-0 truncate">{{ t('taskSettings.clearValues') }}</span>
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

                <TemplatePoolAnalysisPanel
                  :analysis="poolAnalysis"
                  :capacity-message="poolCapacityMessage"
                  :max-value-length="MAX_TEMPLATE_VALUE_LENGTH"
                />
              </div>
            </div>

            <div v-if="form.type === 'post'" class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="border-b border-gray-100 p-4 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.media.postEnhancementsTitle') }}</h3>
                <p class="mt-1 min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="t('taskSettings.media.postImagesHint')">{{ t('taskSettings.media.postImagesHint') }}</p>
              </div>

              <div class="space-y-5 p-4">
                <div class="space-y-2">
                  <label class="input-label" for="quote-post-url">{{ t('taskSettings.form.quotePostUrl') }}</label>
                  <input
                    id="quote-post-url"
                    v-model="form.quotePostURL"
                    type="url"
                    class="input"
                    data-testid="quote-post-url-input"
                    :placeholder="t('taskSettings.form.quotePostUrlPlaceholder')"
                    @input="resetValidationResult"
                  />
                </div>

                <div class="space-y-3">
                  <div class="flex min-w-0 flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0">
                      <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.media.postImages') }}</h4>
                      <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
                        {{ t('taskSettings.media.postImageCount', { count: postMediaCount, max: MAX_POST_MEDIA_ITEMS }) }}
                      </p>
                    </div>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center"
                      data-testid="add-post-media-button"
                      :aria-label="addPostMediaButtonTitle"
                      :title="addPostMediaButtonTitle"
                      :disabled="!canAddPostMedia"
                      @click="addPostMedia"
                    >
                      <Icon name="plus" size="sm" />
                      <span class="min-w-0 truncate">{{ t('taskSettings.media.addPostImage') }}</span>
                    </button>
                  </div>

                  <div
                    v-if="form.postMedia.length === 0"
                    data-testid="post-media-empty"
                    class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400"
                  >
                    {{ t('taskSettings.media.postImagesEmpty') }}
                  </div>

                  <div v-else class="space-y-3">
                    <div
                      v-for="(media, index) in form.postMedia"
                      :key="`post-media-${index}`"
                      :data-testid="`post-media-item-${index}`"
                      class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/50"
                    >
                      <div class="mb-3 flex min-w-0 items-center justify-between gap-3">
                        <div class="min-w-0 break-words text-sm font-medium text-gray-900 dark:text-white" :title="t('taskSettings.media.postImageItem', { index: index + 1 })">{{ t('taskSettings.media.postImageItem', { index: index + 1 }) }}</div>
                        <button
                          type="button"
                          class="btn btn-secondary btn-sm h-10 min-w-0 max-w-full justify-center text-red-600 hover:border-red-200 hover:bg-red-50 dark:text-red-300 dark:hover:border-red-900/60 dark:hover:bg-red-950/30"
                          :data-testid="`remove-post-media-button-${index}`"
                          :aria-label="removePostMediaButtonTitle"
                          :title="removePostMediaButtonTitle"
                          :disabled="saving"
                          @click="removePostMedia(index)"
                        >
                          <Icon name="trash" size="sm" />
                          <span class="min-w-0 truncate">{{ t('taskSettings.media.removePostImage') }}</span>
                        </button>
                      </div>
                      <ImageUpload
                        mode="media"
                        :model-value="media.url || ''"
                        :preview-src="storedMediaPreviewURLs.post[index] || media.url || ''"
                        :preview-content-type="media.content_type || ''"
                        :has-value="hasMediaRef(media)"
                        :max-size="MAX_TASK_MEDIA_UPLOAD_BYTES"
                        :upload-label="t('taskSettings.media.uploadPostImage')"
                        :remove-label="t('taskSettings.media.removePostImage')"
                        :hint="t('taskSettings.media.postImagesHint')"
                        :disabled="saving"
                        @update:model-value="(value) => updatePostMedia(index, value)"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else-if="form.type === 'update_profile'" class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="border-b border-gray-100 p-4 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.profile.title') }}</h3>
                <p class="mt-1 min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="activeValueHelp">{{ activeValueHelp }}</p>
              </div>

              <div class="grid gap-4 p-4 md:grid-cols-2">
                <div class="space-y-2">
                  <label class="input-label" for="profile-display-name">{{ t('taskSettings.form.profileDisplayName') }}</label>
                  <input
                    id="profile-display-name"
                    v-model="form.profileDisplayName"
                    type="text"
                    class="input"
                    data-testid="profile-display-name-input"
                    :placeholder="t('taskSettings.form.profileDisplayNamePlaceholder')"
                    @input="resetValidationResult"
                  />
                </div>
                <div class="space-y-2">
                  <label class="input-label" for="profile-screen-name">{{ t('taskSettings.form.profileScreenName') }}</label>
                  <input
                    id="profile-screen-name"
                    v-model="form.profileScreenName"
                    type="text"
                    class="input"
                    data-testid="profile-screen-name-input"
                    :placeholder="t('taskSettings.form.profileScreenNamePlaceholder')"
                    @input="resetValidationResult"
                  />
                </div>
                <div class="space-y-2 md:col-span-2">
                  <label class="input-label" for="profile-description">{{ t('taskSettings.form.profileDescription') }}</label>
                  <textarea
                    id="profile-description"
                    v-model="form.profileDescription"
                    class="input min-h-[120px] resize-y leading-6"
                    data-testid="profile-description-textarea"
                    :placeholder="t('taskSettings.form.profileDescriptionPlaceholder')"
                    @input="resetValidationResult"
                  ></textarea>
                </div>
                <div class="space-y-2">
                  <label class="input-label" for="profile-location">{{ t('taskSettings.form.profileLocation') }}</label>
                  <input
                    id="profile-location"
                    v-model="form.profileLocation"
                    type="text"
                    class="input"
                    data-testid="profile-location-input"
                    :placeholder="t('taskSettings.form.profileLocationPlaceholder')"
                    @input="resetValidationResult"
                  />
                </div>
                <div class="space-y-2">
                  <label class="input-label" for="profile-url">{{ t('taskSettings.form.profileUrl') }}</label>
                  <input
                    id="profile-url"
                    v-model="form.profileURL"
                    type="url"
                    class="input"
                    data-testid="profile-url-input"
                    :placeholder="t('taskSettings.form.profileUrlPlaceholder')"
                    @input="resetValidationResult"
                  />
                </div>
              </div>
            </div>

            <div v-else-if="form.type === 'update_avatar'" data-testid="avatar-editor" class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="border-b border-gray-100 p-4 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.media.avatarTitle') }}</h3>
                <p class="mt-1 min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="t('taskSettings.media.avatarHint')">{{ t('taskSettings.media.avatarHint') }}</p>
              </div>

              <div class="p-4">
                <ImageUpload
                  :model-value="form.avatar.url || ''"
                  :preview-src="storedMediaPreviewURLs.avatar || form.avatar.url || ''"
                  :has-value="hasMediaRef(form.avatar)"
                  :max-size="MAX_TASK_MEDIA_UPLOAD_BYTES"
                  :upload-label="t('taskSettings.media.uploadAvatar')"
                  :remove-label="t('taskSettings.media.removeAvatar')"
                  :hint="t('taskSettings.media.avatarHint')"
                  :disabled="saving"
                  @update:model-value="updateAvatarMedia"
                />
              </div>
            </div>

            <div v-else-if="form.type === 'update_banner'" data-testid="banner-editor" class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="border-b border-gray-100 p-4 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('taskSettings.media.bannerTitle') }}</h3>
                <p class="mt-1 min-w-0 break-words text-xs leading-5 text-gray-500 dark:text-dark-400" :title="t('taskSettings.media.bannerHint')">{{ t('taskSettings.media.bannerHint') }}</p>
              </div>

              <div class="p-4">
                <ImageUpload
                  :model-value="form.banner.url || ''"
                  :preview-src="storedMediaPreviewURLs.banner || form.banner.url || ''"
                  :has-value="hasMediaRef(form.banner)"
                  :max-size="MAX_TASK_MEDIA_UPLOAD_BYTES"
                  :upload-label="t('taskSettings.media.uploadBanner')"
                  :remove-label="t('taskSettings.media.removeBanner')"
                  :hint="t('taskSettings.media.bannerHint')"
                  :disabled="saving"
                  @update:model-value="updateBannerMedia"
                />
              </div>
            </div>
          </div>
        </section>

        <TemplateSummaryPanel
          :rows="templateSummaryRows"
          :save-disabled-reason="saveDisabledReason"
          :validation-result="validationResult"
        />
      </div>
    </div>

    <BaseDialog :show="viewAllDialogOpen" :title="activeValueLabel" width="wide" @close="viewAllDialogOpen = false">
      <div v-if="activeValues.length === 0" class="rounded-lg border border-gray-200 bg-gray-50 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-400">
        {{ t('taskSettings.pool.empty') }}
      </div>
      <div v-else class="max-h-[520px] divide-y divide-gray-100 overflow-auto rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
        <div v-for="(value, index) in activeValues" :key="`${index}-${value}`" class="grid min-w-0 grid-cols-[64px_minmax(0,1fr)] gap-3 px-3 py-2 text-sm">
          <span class="text-gray-500 dark:text-dark-400">#{{ index + 1 }}</span>
          <span class="min-w-0 break-all font-medium text-gray-900 dark:text-white" :title="value">{{ value }}</span>
        </div>
      </div>
      <template #footer>
        <button
          type="button"
          class="btn btn-primary min-w-0 max-w-full justify-center"
          :aria-label="t('common.close')"
          :title="t('common.close')"
          @click="viewAllDialogOpen = false"
        >
          <span class="min-w-0 truncate">{{ t('common.close') }}</span>
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteDialogOpen" :title="t('taskSettings.deleteDialog.title')" @close="closeDeleteDialog">
      <div v-if="templateToDelete" class="space-y-3">
        <p class="min-w-0 break-words text-sm text-gray-600 dark:text-gray-300" :title="deleteDialogDescription">{{ deleteDialogDescription }}</p>
        <div
          class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          :title="deleteDialogWarning"
        >
          {{ deleteDialogWarning }}
        </div>
        <div
          v-if="deleteDialogError"
          class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200"
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          :title="deleteDialogError"
        >
          {{ deleteDialogError }}
        </div>
        <dl class="grid gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex min-w-0 justify-between gap-3">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('taskSettings.form.name') }}</dt>
            <dd class="min-w-0 break-all text-right text-gray-900 sm:truncate dark:text-white" :title="templateToDelete.name">{{ templateToDelete.name }}</dd>
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
        <button
          type="button"
          class="btn btn-secondary min-w-0 max-w-full justify-center"
          :aria-label="deleteDialogCancelButtonTitle"
          :title="deleteDialogCancelButtonTitle"
          :disabled="saving"
          @click="closeDeleteDialog"
        >
          <span class="min-w-0 truncate">{{ t('common.cancel') }}</span>
        </button>
        <button
          type="button"
          class="btn btn-danger min-w-0 max-w-full justify-center"
          :aria-label="deleteDialogConfirmButtonTitle"
          :title="deleteDialogConfirmButtonTitle"
          :disabled="saving"
          @click="confirmDeleteTemplate"
        >
          <span class="min-w-0 truncate">{{ deleteDialogConfirmButtonTitle }}</span>
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadErrorBanner from '@/components/common/LoadErrorBanner.vue'
import Icon from '@/components/icons/Icon.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import SavedTemplateList from './components/SavedTemplateList.vue'
import TemplateEditorActions from './components/TemplateEditorActions.vue'
import TemplatePoolAnalysisPanel from './components/TemplatePoolAnalysisPanel.vue'
import TemplateSummaryPanel, { type TemplateSummaryRow } from './components/TemplateSummaryPanel.vue'
import TaskTypeSelector from './components/TaskTypeSelector.vue'
import TemplateStatsGrid from './components/TemplateStatsGrid.vue'
import {
  parameterPoolClearButtonTitle as buildParameterPoolClearButtonTitle,
  parameterPoolDedupeButtonTitle as buildParameterPoolDedupeButtonTitle,
  parameterPoolImportButtonTitle as buildParameterPoolImportButtonTitle,
  parameterPoolViewAllButtonTitle as buildParameterPoolViewAllButtonTitle,
} from './parameterPoolActionTitles'
import {
  templateDeleteCancelButtonTitle as buildTemplateDeleteCancelButtonTitle,
  templateDeleteConfirmButtonTitle as buildTemplateDeleteConfirmButtonTitle,
  templateEditorAddPostMediaButtonTitle as buildAddPostMediaButtonTitle,
  templateEditorRemovePostMediaButtonTitle as buildRemovePostMediaButtonTitle,
} from './templateEditorActionTitles'
import taskSettingsAPI from '@/api/taskSettings'
import type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  TaskTemplate,
  TaskTemplateInput,
  TaskTemplateType,
  TaskTemplateValidationResult,
} from '@/api/taskSettings'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import { createObjectURLSafe, revokeObjectURLSafe } from '@/utils/browser'
import {
  cloneMediaRef,
  cloneMediaRefs,
  hasMediaRef,
  normalizeMediaRef,
  normalizeMediaRefs,
  updateInlineMediaRef,
} from './taskMedia'
import {
  MAX_TEMPLATE_POOL_VALUES,
  MAX_TEMPLATE_VALUE_LENGTH,
  analyzeTemplatePool,
  countIgnoredEmptyValues,
  normalizeTemplatePoolValues,
  splitContentValues,
  splitTargetValues,
  type TemplatePoolAnalysis,
} from './templatePool'
import {
  TASK_SETTINGS_MAX_POST_MEDIA_ITEMS,
  TASK_SETTINGS_PARAMETER_TASK_TYPES,
  TASK_SETTINGS_TARGET_TYPES,
  countProfileFields,
  isTaskTemplateUsable as isTemplateUsable,
  resolveTaskTemplateSaveDisabledReason,
  type ParameterTaskTemplateType,
} from './templateReadiness'
import { createTaskSettingsErrorMessages } from './taskSettingsErrorMessages'

type ParameterTaskTemplate = TaskTemplate & { type: ParameterTaskTemplateType }
type IconName = 'checkCircle' | 'userPlus' | 'sync' | 'chatBubble' | 'edit' | 'userCircle' | 'grid'
type TemplateOperation = 'validate' | 'save' | 'copy' | 'default' | 'delete'
type TaskSettingsOperationErrorScope =
  | 'task_settings.validate'
  | 'task_settings.save'
  | 'task_settings.copy'
  | 'task_settings.default'
  | 'task_settings.delete'

const { t } = useI18n()
const appStore = useAppStore()

const MAX_POST_MEDIA_ITEMS = TASK_SETTINGS_MAX_POST_MEDIA_ITEMS
const MAX_TASK_MEDIA_UPLOAD_BYTES = 2 * 1024 * 1024
const TASK_TYPES = TASK_SETTINGS_PARAMETER_TASK_TYPES
const targetTypes = TASK_SETTINGS_TARGET_TYPES

const templates = ref<ParameterTaskTemplate[]>([])
const selectedTemplateId = ref('')
const activeType = ref<ParameterTaskTemplateType>('follow')
const loading = ref(false)
const templateOperation = ref<TemplateOperation | null>(null)
const saving = computed(() => templateOperation.value !== null)
const deleteDialogCancelButtonTitle = computed(() => buildTemplateDeleteCancelButtonTitle(t, { saving: saving.value }))
const deleteDialogConfirmButtonTitle = computed(() => buildTemplateDeleteConfirmButtonTitle(t, { deleting: templateOperation.value === 'delete' }))
const loadError = ref('')
const validationResult = ref<TaskTemplateValidationResult | null>(null)
const editorOperationError = ref('')
const viewAllDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const templateToDelete = ref<ParameterTaskTemplate | null>(null)
const deleteDialogError = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)
const storedMediaPreviewURLs = reactive({
  avatar: '',
  banner: '',
  post: [] as string[],
})
let latestTemplateLoadRequestID = 0
let storedMediaPreviewRequestID = 0
let latestValidationRequestID = 0
let editorContextRevision = 0

const form = reactive({
  id: '',
  name: '',
  type: 'follow' as ParameterTaskTemplateType,
  targetsText: '',
  contentsText: '',
  quotePostURL: '',
  profileDisplayName: '',
  profileScreenName: '',
  profileDescription: '',
  profileLocation: '',
  profileURL: '',
  postMedia: [] as SocialTaskMediaRef[],
  avatar: {} as SocialTaskMediaRef,
  banner: {} as SocialTaskMediaRef,
  isDefault: false,
})

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

const postMediaCount = computed(() => normalizeMediaRefs(form.postMedia, 'post-image').length)
const profileFieldCount = computed(() => countProfileFields(buildProfileParams()))
const structuredSummaryRows = computed(() => {
  switch (form.type) {
    case 'post':
      return [
        { label: t('taskSettings.summary.contents'), value: contentValues.value.length },
        { label: t('taskSettings.summary.quotePost'), value: form.quotePostURL.trim() ? t('common.yes') : t('common.no') },
        { label: t('taskSettings.summary.media'), value: postMediaCount.value },
      ]
    case 'update_profile':
      return [{ label: t('taskSettings.summary.profileFields'), value: profileFieldCount.value }]
    case 'update_avatar':
      return [{ label: t('taskSettings.summary.media'), value: hasMediaRef(form.avatar) ? 1 : 0 }]
    case 'update_banner':
      return [{ label: t('taskSettings.summary.media'), value: hasMediaRef(form.banner) ? 1 : 0 }]
    default:
      return []
  }
})
const templateSummaryRows = computed<TemplateSummaryRow[]>(() => {
  const rows: TemplateSummaryRow[] = [
    { key: 'type', label: t('taskSettings.summary.type'), value: taskTypeLabel(form.type) },
  ]
  if (activePoolKind.value) {
    rows.push({
      key: activePoolKind.value,
      label: activePoolKind.value === 'targets' ? t('taskSettings.summary.targets') : t('taskSettings.summary.contents'),
      value: activeValues.value.length,
    })
  }
  rows.push(...structuredSummaryRows.value.map((row, index) => ({
    key: `structured-${index}-${row.label}`,
    label: row.label,
    value: row.value,
  })))
  if (!activePoolKind.value && structuredSummaryRows.value.length === 0) {
    rows.push({ key: 'params', label: t('taskSettings.summary.params'), value: t('taskSettings.counts.none') })
  }
  rows.push({
    key: 'default',
    label: t('taskSettings.summary.default'),
    value: form.isDefault ? t('common.yes') : t('common.no'),
  })
  return rows
})

const activeTypeDescription = computed(() => t(`taskSettings.typeDescriptions.${form.type}`))
const activeValueLabel = computed(() => {
  if (form.type === 'post') return t('taskSettings.form.contents')
  if (targetTypes.has(form.type)) return t('taskSettings.form.targets')
  return t('taskSettings.pool.title')
})
const activeValueHelp = computed(() => t(`taskSettings.typeRequirements.${form.type}`))
const activeValuePlaceholder = computed(() => {
  if (form.type === 'post') return t('taskSettings.form.contentsPlaceholder')
  if (form.type === 'follow') return t('taskSettings.form.followTargetsPlaceholder')
  return t('taskSettings.form.postTargetsPlaceholder')
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
const poolAnalysis = computed<TemplatePoolAnalysis>(() => analyzeTemplatePool(activeValues.value, poolEmptyLineCount.value))
const poolCapacityMessage = computed(() => {
  if (poolAnalysis.value.overCapacity) return t('taskSettings.validation.tooManyValues', { max: MAX_TEMPLATE_POOL_VALUES })
  return t('taskSettings.pool.capacityHint', { count: activeValues.value.length, max: MAX_TEMPLATE_POOL_VALUES })
})
const defaultImpactText = computed(() => t('taskSettings.defaultImpact', { type: taskTypeLabel(form.type) }))
const canAddPostMedia = computed(() => {
  const media = normalizeMediaRefs(form.postMedia, 'post-image')
  return !saving.value && media.length < MAX_POST_MEDIA_ITEMS
})
const addPostMediaButtonTitle = computed(() => buildAddPostMediaButtonTitle(t, {
  saving: saving.value,
  mediaCount: postMediaCount.value,
  maxMediaItems: MAX_POST_MEDIA_ITEMS,
}))
const removePostMediaButtonTitle = computed(() => buildRemovePostMediaButtonTitle(t, { saving: saving.value }))
const saveDisabledReason = computed(() => resolveTaskTemplateSaveDisabledReason({
  name: form.name,
  type: form.type,
  targetValues: targetValues.value,
  contentValues: contentValues.value,
  postMedia: form.postMedia,
  profile: buildProfileParams(),
  avatar: form.avatar,
  banner: form.banner,
}, t))
const canSave = computed(() => !saveDisabledReason.value)
const currentTemplateReady = computed(() => !saveDisabledReason.value)
const canClearActiveValues = computed(() => activeValuesText.value.length > 0)
const importButtonTitle = computed(() => buildParameterPoolImportButtonTitle(t, { saving: saving.value }))
const viewAllButtonTitle = computed(() => buildParameterPoolViewAllButtonTitle(t, {
  saving: saving.value,
  valueCount: activeValues.value.length,
}))
const dedupeButtonTitle = computed(() => buildParameterPoolDedupeButtonTitle(t, {
  saving: saving.value,
  duplicateCount: poolAnalysis.value.duplicateCount,
}))
const clearPoolButtonTitle = computed(() => buildParameterPoolClearButtonTitle(t, {
  saving: saving.value,
  canClear: canClearActiveValues.value,
}))
const deleteDialogDescription = computed(() => templateToDelete.value
  ? t('taskSettings.deleteDialog.description', { name: templateToDelete.value.name })
  : ''
)
const deleteDialogWarning = computed(() => t('taskSettings.deleteDialog.warning'))
const taskSettingsErrorMessages = computed(() => createTaskSettingsErrorMessages(t))

void loadTemplates()
onBeforeUnmount(() => clearStoredMediaPreviewURLs())

async function loadTemplates(options: { syncEditor?: boolean } = {}) {
  const syncEditor = options.syncEditor ?? true
  const requestID = ++latestTemplateLoadRequestID
  const loadEditorRevision = editorContextRevision
  loading.value = true
  loadError.value = ''
  try {
    const loadedTemplates = await taskSettingsAPI.listTemplates()
    if (!isLatestTemplateLoadRequest(requestID)) return
    templates.value = Array.isArray(loadedTemplates) ? loadedTemplates.filter(isParameterTaskTemplate).map(normalizeParameterTaskTemplate) : []
    syncDeleteDialogFromTemplates()
    if (syncEditor && editorContextRevision === loadEditorRevision) {
      const selected = templates.value.find(template => template.id === selectedTemplateId.value)
      if (selected) {
        selectTemplate(selected, false)
      } else {
        selectBestTemplateForType(activeType.value)
      }
    }
  } catch (error) {
    if (!isLatestTemplateLoadRequest(requestID)) return
    recordClientDiagnostic('task_settings.load', error)
    loadError.value = extractSafeApiErrorMessage(error, t('taskSettings.failedToLoad'), taskSettingsErrorMessages.value)
    appStore.showError(loadError.value)
  } finally {
    if (isLatestTemplateLoadRequest(requestID)) {
      loading.value = false
    }
  }
}

function isLatestTemplateLoadRequest(requestID: number) {
  return requestID === latestTemplateLoadRequestID
}

function chooseType(type: ParameterTaskTemplateType) {
  activeType.value = type
  resetValidationResult()
  selectBestTemplateForType(type)
}

function selectBestTemplateForType(type: ParameterTaskTemplateType) {
  const candidates = templates.value.filter(template => template.type === type)
  const next = candidates.find(template => template.is_default) ?? candidates[0]
  if (next) {
    selectTemplate(next, false)
  } else {
    resetFormForType(type)
  }
}

function selectTemplate(template: TaskTemplate, trackUserContext = true) {
  if (trackUserContext) markEditorContextChanged()
  if (!isParameterTaskTemplate(template)) {
    selectBestTemplateForType(activeType.value)
    return
  }
  const safeTemplate = normalizeParameterTaskTemplate(template)
  const previewRequestID = clearStoredMediaPreviewURLs()
  selectedTemplateId.value = safeTemplate.id
  activeType.value = safeTemplate.type
  form.id = safeTemplate.id
  form.name = safeTemplate.name
  form.type = safeTemplate.type
  form.targetsText = targetTypes.has(safeTemplate.type) ? normalizeTemplatePoolValues(safeTemplate.params.targets).join('\n') : ''
  form.contentsText = safeTemplate.type === 'post' ? normalizeTemplatePoolValues(safeTemplate.params.contents).join('\n') : ''
  form.quotePostURL = safeTemplate.type === 'post' ? String(safeTemplate.params.quote_post_url ?? '').trim() : ''
  form.postMedia = safeTemplate.type === 'post' ? cloneMediaRefs(safeTemplate.params.media) : []
  form.profileDisplayName = safeTemplate.type === 'update_profile' ? String(safeTemplate.params.profile?.display_name ?? '').trim() : ''
  form.profileScreenName = safeTemplate.type === 'update_profile' ? String(safeTemplate.params.profile?.screen_name ?? '').trim() : ''
  form.profileDescription = safeTemplate.type === 'update_profile' ? String(safeTemplate.params.profile?.description ?? '').trim() : ''
  form.profileLocation = safeTemplate.type === 'update_profile' ? String(safeTemplate.params.profile?.location ?? '').trim() : ''
  form.profileURL = safeTemplate.type === 'update_profile' ? String(safeTemplate.params.profile?.url ?? '').trim() : ''
  form.avatar = safeTemplate.type === 'update_avatar' ? cloneMediaRef(safeTemplate.params.avatar) : {}
  form.banner = safeTemplate.type === 'update_banner' ? cloneMediaRef(safeTemplate.params.banner) : {}
  form.isDefault = safeTemplate.is_default
  clearValidationResult()
  void loadStoredMediaPreviewsForForm(previewRequestID)
}

async function refreshTemplatesAndSelect(template: TaskTemplate, editorRevision: number) {
  const shouldSyncEditor = editorContextRevision === editorRevision
  const localTemplate = upsertTemplateIntoLocalState(template)
  await loadTemplates({ syncEditor: shouldSyncEditor })
  if (editorContextRevision !== editorRevision) return
  const refreshed = templates.value.find(item => item.id === template.id)
  selectTemplate(refreshed ?? localTemplate ?? template, false)
}

function upsertTemplateIntoLocalState(template: TaskTemplate) {
  if (!isParameterTaskTemplate(template)) return null
  const safeTemplate = normalizeParameterTaskTemplate(template)
  templates.value = templates.value.map(item => {
    if (item.id === safeTemplate.id) return safeTemplate
    if (safeTemplate.is_default && item.type === safeTemplate.type) {
      return { ...item, is_default: false }
    }
    return item
  })
  if (!templates.value.some(item => item.id === safeTemplate.id)) {
    templates.value = [safeTemplate, ...templates.value]
  }
  return safeTemplate
}

function resetFormForType(type: ParameterTaskTemplateType) {
  clearStoredMediaPreviewURLs()
  selectedTemplateId.value = ''
  activeType.value = type
  form.id = ''
  form.name = ''
  form.type = type
  form.targetsText = ''
  form.contentsText = ''
  form.quotePostURL = ''
  form.profileDisplayName = ''
  form.profileScreenName = ''
  form.profileDescription = ''
  form.profileLocation = ''
  form.profileURL = ''
  form.postMedia = []
  form.avatar = {}
  form.banner = {}
  form.isDefault = templates.value.filter(template => template.type === type).length === 0
  clearValidationResult()
}

function newTemplate() {
  markEditorContextChanged()
  resetFormForType(activeType.value)
}

async function validateCurrent() {
  if (saving.value) return
  const requestID = ++latestValidationRequestID
  validationResult.value = null
  await runTemplateOperation('validate', async () => {
    try {
      const result = await taskSettingsAPI.validateTemplate(buildInput())
      if (!isLatestValidationRequest(requestID)) return
      validationResult.value = result
      if (result.valid) {
        appStore.showSuccess(t('taskSettings.validation.valid'))
      } else {
        appStore.showWarning(t('taskSettings.validation.invalid'))
      }
    } catch (error) {
      if (!isLatestValidationRequest(requestID)) return
      showTaskSettingsEditorOperationError('task_settings.validate', error, t('taskSettings.validation.failed'))
    }
  })
}

function isLatestValidationRequest(requestID: number) {
  return requestID === latestValidationRequestID
}

function showTaskSettingsOperationError(scope: TaskSettingsOperationErrorScope, error: unknown, fallback: string) {
  recordClientDiagnostic(scope, error)
  const message = extractSafeApiErrorMessage(error, fallback, taskSettingsErrorMessages.value)
  appStore.showError(message)
  return message
}

function showTaskSettingsEditorOperationError(scope: TaskSettingsOperationErrorScope, error: unknown, fallback: string) {
  const message = showTaskSettingsOperationError(scope, error, fallback)
  editorOperationError.value = message
  return message
}

async function runTemplateOperation(operation: TemplateOperation, action: () => Promise<void>) {
  if (saving.value) return
  templateOperation.value = operation
  if (operation !== 'delete') {
    editorOperationError.value = ''
  }
  try {
    await action()
  } finally {
    templateOperation.value = null
  }
}

async function saveTemplate() {
  if (!canSave.value || saving.value) return
  const editorRevision = editorContextRevision
  await runTemplateOperation('save', async () => {
    try {
      const result = await taskSettingsAPI.saveTemplate(buildInput())
      appStore.showSuccess(t('taskSettings.saved'))
      await refreshTemplatesAndSelect(result, editorRevision)
    } catch (error) {
      showTaskSettingsEditorOperationError('task_settings.save', error, t('taskSettings.saveFailed'))
    }
  })
}

async function copyCurrentTemplate() {
  if (!selectedTemplateId.value || saving.value) return
  const templateId = selectedTemplateId.value
  const editorRevision = editorContextRevision
  await runTemplateOperation('copy', async () => {
    try {
      const result = await taskSettingsAPI.copyTemplate(templateId)
      appStore.showSuccess(t('taskSettings.copied'))
      await refreshTemplatesAndSelect(result, editorRevision)
    } catch (error) {
      showTaskSettingsEditorOperationError('task_settings.copy', error, t('taskSettings.copyFailed'))
    }
  })
}

async function setDefault() {
  if (!selectedTemplateId.value || saving.value) return
  const templateId = selectedTemplateId.value
  const editorRevision = editorContextRevision
  await runTemplateOperation('default', async () => {
    try {
      const result = await taskSettingsAPI.setDefaultTemplate(templateId)
      appStore.showSuccess(t('taskSettings.defaultSaved'))
      await refreshTemplatesAndSelect(result, editorRevision)
    } catch (error) {
      showTaskSettingsEditorOperationError('task_settings.default', error, t('taskSettings.defaultFailed'))
    }
  })
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
  deleteDialogError.value = ''
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  if (saving.value) return
  deleteDialogOpen.value = false
  templateToDelete.value = null
  deleteDialogError.value = ''
}

function syncDeleteDialogFromTemplates() {
  if (!deleteDialogOpen.value || !templateToDelete.value || templateOperation.value === 'delete') return
  const updated = templates.value.find(template => template.id === templateToDelete.value?.id)
  if (updated) {
    templateToDelete.value = updated
  } else {
    closeDeleteDialog()
  }
}

async function confirmDeleteTemplate() {
  if (saving.value || !templateToDelete.value) return
  const template = templateToDelete.value
  deleteDialogError.value = ''
  await runTemplateOperation('delete', async () => {
    try {
      await taskSettingsAPI.deleteTemplate(template.id)
      appStore.showSuccess(t('taskSettings.deleted'))
      deleteDialogOpen.value = false
      templateToDelete.value = null
      deleteDialogError.value = ''
      removeDeletedTemplateFromLocalState(template)
      await loadTemplates()
    } catch (error) {
      deleteDialogError.value = showTaskSettingsOperationError('task_settings.delete', error, t('taskSettings.deleteFailed'))
    }
  })
}

function removeDeletedTemplateFromLocalState(template: ParameterTaskTemplate) {
  templates.value = templates.value.filter(item => item.id !== template.id)
  if (selectedTemplateId.value === template.id) {
    selectBestTemplateForType(template.type)
  }
}

async function handleFileImport(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  const importPoolKind = activePoolKind.value
  const importType = form.type
  const importTemplateId = selectedTemplateId.value
  if (saving.value || !file || !importPoolKind) {
    input.value = ''
    return
  }
  editorOperationError.value = ''
  try {
    const text = await file.text()
    if (saving.value || activePoolKind.value !== importPoolKind || form.type !== importType || selectedTemplateId.value !== importTemplateId) return
    const imported = importPoolKind === 'targets' ? splitTargetValues(text) : splitContentValues(text)
    if (imported.length === 0) {
      appStore.showWarning(t('taskSettings.importEmpty'))
      return
    }
    activeValuesText.value = [activeValuesText.value.trim(), imported.join('\n')].filter(Boolean).join('\n')
    resetValidationResult()
    appStore.showSuccess(t('taskSettings.imported', { count: imported.length }))
  } catch (error) {
    recordClientDiagnostic('task_settings.import_file', error)
    editorOperationError.value = t('taskSettings.importFailed')
    appStore.showError(editorOperationError.value)
  } finally {
    input.value = ''
  }
}

function clearValues() {
  if (saving.value) return
  if (!canClearActiveValues.value) return
  activeValuesText.value = ''
  resetValidationResult()
}

function dedupeValues() {
  if (saving.value) return
  const before = activeValues.value.length
  const deduped = Array.from(new Set(activeValues.value))
  if (deduped.length === before) return
  activeValuesText.value = deduped.join('\n')
  resetValidationResult()
  appStore.showSuccess(t('taskSettings.deduped', { count: before - deduped.length }))
}

function addPostMedia() {
  if (!canAddPostMedia.value) return
  invalidateStoredMediaPreviewRequests()
  form.postMedia.push({})
  storedMediaPreviewURLs.post.push('')
  resetValidationResult()
}

function removePostMedia(index: number) {
  if (saving.value) return
  invalidateStoredMediaPreviewRequests()
  revokeStoredMediaPreviewURL('post', index)
  form.postMedia.splice(index, 1)
  storedMediaPreviewURLs.post.splice(index, 1)
  resetValidationResult()
}

function updatePostMedia(index: number, value: string) {
  if (saving.value) return
  invalidateStoredMediaPreviewRequests()
  const next = updateInlineMediaRef(form.postMedia[index], value, `post-image-${index + 1}`)
  revokeStoredMediaPreviewURL('post', index)
  form.postMedia[index] = next
  storedMediaPreviewURLs.post[index] = ''
  resetValidationResult()
}

async function updateAvatarMedia(value: string) {
  if (saving.value) return
  invalidateStoredMediaPreviewRequests()
  revokeStoredMediaPreviewURL('avatar')
  form.avatar = await updateInlineMediaRefWithDimensions(form.avatar, value, 'avatar-image')
  resetValidationResult()
}

async function updateBannerMedia(value: string) {
  if (saving.value) return
  invalidateStoredMediaPreviewRequests()
  revokeStoredMediaPreviewURL('banner')
  form.banner = await updateInlineMediaRefWithDimensions(form.banner, value, 'banner-image')
  resetValidationResult()
}

function resetValidationResult() {
  markEditorContextChanged()
  clearValidationResult()
}

function clearValidationResult() {
  latestValidationRequestID += 1
  validationResult.value = null
  editorOperationError.value = ''
}

function markEditorContextChanged() {
  editorContextRevision += 1
}

function buildTemplateParams(): TaskTemplateInput['params'] {
  if (targetTypes.has(form.type)) {
    return { targets: targetValues.value }
  }
  if (form.type === 'post') {
    const media = normalizeMediaRefs(form.postMedia, 'post-image')
    const params: TaskTemplateInput['params'] = { contents: contentValues.value }
    if (form.quotePostURL.trim()) params.quote_post_url = form.quotePostURL.trim()
    if (media.length > 0) params.media = media
    return params
  }
  if (form.type === 'update_profile') {
    const profile = buildProfileParams()
    return profile ? { profile } : {}
  }
  if (form.type === 'update_avatar') {
    const avatar = normalizeMediaRef(form.avatar, 'avatar-image')
    return avatar ? { avatar } : {}
  }
  if (form.type === 'update_banner') {
    const banner = normalizeMediaRef(form.banner, 'banner-image')
    return banner ? { banner } : {}
  }
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

function buildProfileParams(): SocialProfileUpdateParams | undefined {
  const profile: SocialProfileUpdateParams = {
    display_name: form.profileDisplayName.trim(),
    screen_name: form.profileScreenName.trim(),
    description: form.profileDescription.trim(),
    location: form.profileLocation.trim(),
    url: form.profileURL.trim(),
  }
  return countProfileFields(profile) > 0 ? profile : undefined
}

async function loadStoredMediaPreviewsForForm(requestID = storedMediaPreviewRequestID) {
  if (form.type === 'update_avatar') {
    const url = await resolveStoredMediaPreviewURL(form.avatar)
    if (!isCurrentStoredMediaPreviewRequest(requestID)) {
      revokeObjectURLSafe(url)
      return
    }
    storedMediaPreviewURLs.avatar = url
    return
  }
  if (form.type === 'update_banner') {
    const url = await resolveStoredMediaPreviewURL(form.banner)
    if (!isCurrentStoredMediaPreviewRequest(requestID)) {
      revokeObjectURLSafe(url)
      return
    }
    storedMediaPreviewURLs.banner = url
    return
  }
  if (form.type === 'post') {
    const urls = await Promise.all(form.postMedia.map(item => resolveStoredMediaPreviewURL(item)))
    if (!isCurrentStoredMediaPreviewRequest(requestID)) {
      urls.forEach(url => revokeObjectURLSafe(url))
      return
    }
    storedMediaPreviewURLs.post = urls
  }
}

function isCurrentStoredMediaPreviewRequest(requestID: number) {
  return requestID === storedMediaPreviewRequestID
}

async function resolveStoredMediaPreviewURL(item?: SocialTaskMediaRef | null) {
  const storageKey = String(item?.storage_key || '').trim()
  const source = String(item?.source || '').trim().toLowerCase()
  if (source !== 'library' || !storageKey.toLowerCase().startsWith('social-task/')) return ''
  try {
    const blob = await taskSettingsAPI.previewMedia(storageKey)
    return createObjectURLSafe(blob)
  } catch (error) {
    recordClientDiagnostic('task_settings.preview_media', error)
    return ''
  }
}

function revokeStoredMediaPreviewURL(kind: 'avatar' | 'banner' | 'post', index?: number) {
  if (kind === 'post') {
    const targetIndex = Number(index ?? -1)
    if (targetIndex >= 0 && targetIndex < storedMediaPreviewURLs.post.length) {
      revokeObjectURLSafe(storedMediaPreviewURLs.post[targetIndex] || '')
      storedMediaPreviewURLs.post[targetIndex] = ''
    }
    return
  }
  revokeObjectURLSafe(storedMediaPreviewURLs[kind])
  storedMediaPreviewURLs[kind] = ''
}

function invalidateStoredMediaPreviewRequests() {
  storedMediaPreviewRequestID += 1
  return storedMediaPreviewRequestID
}

function clearStoredMediaPreviewURLs() {
  invalidateStoredMediaPreviewRequests()
  revokeStoredMediaPreviewURL('avatar')
  revokeStoredMediaPreviewURL('banner')
  storedMediaPreviewURLs.post.forEach(url => revokeObjectURLSafe(url))
  storedMediaPreviewURLs.post = []
  return storedMediaPreviewRequestID
}

async function updateInlineMediaRefWithDimensions(current: SocialTaskMediaRef | undefined, value: string, fallbackBaseName: string): Promise<SocialTaskMediaRef> {
  const next = updateInlineMediaRef(current, value, fallbackBaseName)
  if (!hasMediaRef(next)) return next
  const dimensions = await readInlineImageDimensions(String(next.url || ''))
  if (!dimensions) return next
  return {
    ...next,
    width: dimensions.width,
    height: dimensions.height,
  }
}

function readInlineImageDimensions(value: string): Promise<{ width: number; height: number } | null> {
  if (!String(value || '').startsWith('data:image/')) {
    return Promise.resolve(null)
  }
  return new Promise((resolve) => {
    const image = new Image()
    image.onload = () => {
      const width = Math.max(0, Math.round(image.naturalWidth || 0))
      const height = Math.max(0, Math.round(image.naturalHeight || 0))
      if (width <= 0 || height <= 0) {
        resolve(null)
        return
      }
      resolve({ width, height })
    }
    image.onerror = () => resolve(null)
    image.src = value
  })
}

function templateParameterSummary(template: ParameterTaskTemplate): { label: string; count: string | number } {
  if (targetTypes.has(template.type)) {
    return { label: t('taskSettings.summary.targets'), count: (template.params.targets ?? []).length }
  }
  if (template.type === 'post') {
    return { label: t('taskSettings.summary.contents'), count: (template.params.contents ?? []).length }
  }
  if (template.type === 'update_profile') {
    return { label: t('taskSettings.summary.profileFields'), count: countProfileFields(template.params.profile) }
  }
  if (template.type === 'update_avatar' || template.type === 'update_banner') {
    const media = template.type === 'update_avatar' ? template.params.avatar : template.params.banner
    return { label: t('taskSettings.summary.media'), count: hasMediaRef(media) ? 1 : 0 }
  }
  return { label: '', count: '' }
}

function templateParameterStateLabel(template: ParameterTaskTemplate) {
  return isTemplateUsable(template)
    ? t('taskSettings.savedConfigs.readyState')
    : t('taskSettings.savedConfigs.needsInputState')
}

function taskTypeLabel(type: TaskTemplateType | string) {
  const normalized = normalizeTaskTemplateType(type)
  if (normalized) return t(`taskSettings.types.${normalized}`, normalized)
  const fallback = String(type ?? '').trim()
  return fallback || '-'
}

function taskTypeIcon(type: ParameterTaskTemplateType): IconName {
  if (type === 'follow') return 'userPlus'
  if (type === 'retweet') return 'sync'
  if (type === 'post') return 'chatBubble'
  if (type === 'update_profile') return 'edit'
  if (type === 'update_avatar') return 'userCircle'
  if (type === 'update_banner') return 'grid'
  return 'checkCircle'
}

function taskTypeTone(type: ParameterTaskTemplateType) {
  if (type === 'post') return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  if (type === 'retweet') return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300'
  if (type === 'update_profile') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (type === 'update_avatar') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (type === 'update_banner') return 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300'
  return 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
}

function taskTypeBadgeClass(type: ParameterTaskTemplateType) {
  if (type === 'post') return 'badge-primary'
  if (type === 'update_profile') return 'badge-success'
  return 'badge-gray'
}

function normalizeParameterTaskTemplate(template: ParameterTaskTemplate): ParameterTaskTemplate {
  const type = normalizeTaskTemplateType(template.type) ?? template.type
  return {
    ...template,
    name: String(template.name ?? '').trim(),
    type,
    params: isTemplateParamsObject(template.params) ? template.params : {},
  }
}

function isTemplateParamsObject(value: unknown): value is TaskTemplate['params'] {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function isParameterTaskTemplate(template: unknown): template is ParameterTaskTemplate {
  return !!template
    && typeof template === 'object'
    && normalizeTaskTemplateType((template as Partial<TaskTemplate>).type) !== null
}

function normalizeTaskTemplateType(value: unknown): ParameterTaskTemplateType | null {
  const normalized = String(value ?? '').trim().toLowerCase()
  return (TASK_TYPES as string[]).includes(normalized)
    ? normalized as ParameterTaskTemplateType
    : null
}
</script>
