import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const viewPath = resolve(__dirname, '../TaskSettingsView.vue')
const savedTemplateListPath = resolve(__dirname, '../components/SavedTemplateList.vue')
const templateEditorActionsPath = resolve(__dirname, '../components/TemplateEditorActions.vue')
const templatePoolAnalysisPanelPath = resolve(__dirname, '../components/TemplatePoolAnalysisPanel.vue')
const templateSummaryPanelPath = resolve(__dirname, '../components/TemplateSummaryPanel.vue')
const taskTypeSelectorPath = resolve(__dirname, '../components/TaskTypeSelector.vue')
const templateStatsGridPath = resolve(__dirname, '../components/TemplateStatsGrid.vue')
const templatePoolPath = resolve(__dirname, '../templatePool.ts')
const taskMediaPath = resolve(__dirname, '../taskMedia.ts')
const templateReadinessPath = resolve(__dirname, '../templateReadiness.ts')
const templateEditorActionTitlesPath = resolve(__dirname, '../templateEditorActionTitles.ts')
const parameterPoolActionTitlesPath = resolve(__dirname, '../parameterPoolActionTitles.ts')
const taskSettingsErrorMessagesPath = resolve(__dirname, '../taskSettingsErrorMessages.ts')
const enLocalePath = resolve(__dirname, '../../../i18n/locales/en.ts')
const zhLocalePath = resolve(__dirname, '../../../i18n/locales/zh.ts')

function readViewSource(): string {
  return readFileSync(viewPath, 'utf8')
}

function readSavedTemplateListSource(): string {
  return readFileSync(savedTemplateListPath, 'utf8')
}

function readTemplateEditorActionsSource(): string {
  return readFileSync(templateEditorActionsPath, 'utf8')
}

function readTemplatePoolAnalysisPanelSource(): string {
  return readFileSync(templatePoolAnalysisPanelPath, 'utf8')
}

function readTemplateSummaryPanelSource(): string {
  return readFileSync(templateSummaryPanelPath, 'utf8')
}

function readTemplateEditorActionTitlesSource(): string {
  return readFileSync(templateEditorActionTitlesPath, 'utf8')
}

function readParameterPoolActionTitlesSource(): string {
  return readFileSync(parameterPoolActionTitlesPath, 'utf8')
}

function readTaskTypeSelectorSource(): string {
  return readFileSync(taskTypeSelectorPath, 'utf8')
}

function readTemplateStatsGridSource(): string {
  return readFileSync(templateStatsGridPath, 'utf8')
}

function readTemplatePoolSource(): string {
  return readFileSync(templatePoolPath, 'utf8')
}

function readTaskMediaSource(): string {
  return readFileSync(taskMediaPath, 'utf8')
}

function readTemplateReadinessSource(): string {
  return readFileSync(templateReadinessPath, 'utf8')
}

function readTaskSettingsErrorMessagesSource(): string {
  return readFileSync(taskSettingsErrorMessagesPath, 'utf8')
}

function readEnLocaleSource(): string {
  return readFileSync(enLocalePath, 'utf8')
}

function readZhLocaleSource(): string {
  return readFileSync(zhLocalePath, 'utf8')
}

describe('TaskSettingsView source contract', () => {
  it('keeps the page-level duplicate heading out and keeps editor actions non-duplicative', () => {
    const source = readViewSource()
    const editorActions = readTemplateEditorActionsSource()

    expect(source).not.toContain('<h1')
    expect(source).not.toContain("t('taskSettings.title')")
    expect(source).not.toContain("t('taskSettings.description')")
    expect(source).toContain('<TemplateEditorActions')
    expect(editorActions).toContain('data-testid="editor-template-actions"')
    expect(editorActions).toContain('data-testid="save-template-button"')
    expect(editorActions).toContain('data-testid="validation-button"')
    expect(source).not.toContain('data-testid="save-template-button-secondary"')
  })

  it('keeps the page load error affordance aligned with the other core pages', () => {
    const source = readViewSource()
    const loadErrorSource = source.slice(
      source.indexOf('<LoadErrorBanner'),
      source.indexOf('<TaskTypeSelector'),
    )

    expect(source).toContain("import LoadErrorBanner from '@/components/common/LoadErrorBanner.vue'")
    expect(loadErrorSource).toContain('v-if="loadError"')
    expect(loadErrorSource).toContain(":title=\"t('taskSettings.failedToLoad')\"")
    expect(loadErrorSource).toContain(':message="loadError"')
    expect(loadErrorSource).toContain(":retry-label=\"t('common.retry')\"")
    expect(loadErrorSource).toContain('@retry="loadTemplates"')
    expect(source).not.toContain('<div v-if="loadError" class="rounded-lg border border-red-200')
  })

  it('keeps save readiness aligned with template completeness', () => {
    const source = readViewSource()
    const editorActions = readTemplateEditorActionsSource()
    const editorActionTitles = readTemplateEditorActionTitlesSource()
    const templatePool = readTemplatePoolSource()
    const templateReadiness = readTemplateReadinessSource()

    expect(source).toContain('const saveDisabledReason = computed')
    expect(source).toContain('resolveTaskTemplateSaveDisabledReason({')
    expect(templateReadiness).toContain('function resolveTaskTemplateSaveDisabledReason')
    expect(templatePool).toContain('MAX_TEMPLATE_POOL_VALUES = 500')
    expect(templatePool).toContain('MAX_TEMPLATE_VALUE_LENGTH = 2048')
    expect(templateReadiness).toContain("t('taskSettings.validation.nameRequired')")
    expect(templateReadiness).toContain("t('taskSettings.validation.targetsRequired')")
    expect(templateReadiness).toContain("t('taskSettings.validation.postConfigurationRequired')")
    expect(templateReadiness).toContain("t('taskSettings.validation.postMediaTooMany'")
    expect(templateReadiness).toContain("t('taskSettings.validation.postVideoUnavailable')")
    expect(templateReadiness).toContain("t('taskSettings.validation.postMediaTypeUnsupported')")
    expect(templateReadiness).toContain("t('taskSettings.validation.mediaSourceUnsupported')")
    expect(templateReadiness).toContain("t('taskSettings.validation.avatarDimensions'")
    expect(templateReadiness).toContain("t('taskSettings.validation.bannerDimensions'")
    expect(templateReadiness).toContain("t('taskSettings.validation.tooManyValues'")
    expect(templateReadiness).toContain("t('taskSettings.validation.valueTooLong'")
    expect(templatePool).toContain('Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH')
    expect(source).toContain('const canSave = computed(() => !saveDisabledReason.value)')
    expect(source).toContain(':save-disabled-reason="saveDisabledReason"')
    expect(editorActions).toContain(':aria-label="saveButtonLabel"')
    expect(editorActions).toContain(':title="saveButtonTitle"')
    expect(editorActions).toContain("from '../templateEditorActionTitles'")
    expect(editorActions).toContain('const saveButtonLabel = computed(() => buildSaveButtonLabel')
    expect(editorActions).toContain('const saveButtonTitle = computed(() => buildSaveButtonTitle')
    expect(editorActionTitles).toContain('export function templateEditorSaveButtonLabel')
    expect(editorActionTitles).toContain('export function templateEditorSaveButtonTitle')
    expect(editorActionTitles).toContain('if (state.saveDisabledReason) return state.saveDisabledReason')
    expect(source).toContain('templateDeleteCancelButtonTitle as buildTemplateDeleteCancelButtonTitle')
    expect(source).toContain('templateDeleteConfirmButtonTitle as buildTemplateDeleteConfirmButtonTitle')
    expect(source).toContain("from './templateEditorActionTitles'")
    expect(source).toContain('const deleteDialogCancelButtonTitle = computed(() => buildTemplateDeleteCancelButtonTitle')
    expect(source).toContain('const deleteDialogConfirmButtonTitle = computed(() => buildTemplateDeleteConfirmButtonTitle')
    expect(source).toContain(':aria-label="deleteDialogCancelButtonTitle"')
    expect(source).toContain(':title="deleteDialogCancelButtonTitle"')
    expect(source).toContain(':title="deleteDialogConfirmButtonTitle"')
    expect(source).toContain(':aria-label="deleteDialogConfirmButtonTitle"')
    expect(editorActionTitles).toContain('export function templateDeleteCancelButtonTitle')
    expect(editorActionTitles).toContain('export function templateDeleteConfirmButtonTitle')
    expect(source).not.toContain("const canSave = computed(() => form.name.trim() !== '')")
  })

  it('keeps parameter-pool toolbar disabled titles aligned with the blocking reason', () => {
    const source = readViewSource()
    const parameterPoolActionTitles = readParameterPoolActionTitlesSource()
    const zh = readZhLocaleSource()

    expect(source).toContain("from './parameterPoolActionTitles'")
    expect(source).toContain('const importButtonTitle = computed')
    expect(source).toContain('const viewAllButtonTitle = computed')
    expect(source).toContain('const dedupeButtonTitle = computed')
    expect(source).toContain('const clearPoolButtonTitle = computed')
    expect(source).toContain('buildParameterPoolImportButtonTitle(t, { saving: saving.value })')
    expect(source).toContain('buildParameterPoolViewAllButtonTitle(t, {')
    expect(source).toContain('buildParameterPoolDedupeButtonTitle(t, {')
    expect(source).toContain('buildParameterPoolClearButtonTitle(t, {')
    expect(source).toContain(':aria-label="importButtonTitle"')
    expect(source).toContain(':aria-label="viewAllButtonTitle"')
    expect(source).toContain(':aria-label="dedupeButtonTitle"')
    expect(source).toContain(':aria-label="clearPoolButtonTitle"')
    expect(source).toContain(':title="importButtonTitle"')
    expect(source).toContain(':title="viewAllButtonTitle"')
    expect(source).toContain(':title="dedupeButtonTitle"')
    expect(source).toContain(':title="clearPoolButtonTitle"')
    expect(parameterPoolActionTitles).toContain("const poolEmptyKey = 'taskSettings.pool.empty'")
    expect(parameterPoolActionTitles).toContain("t('taskSettings.pool.noDuplicates')")
    expect(zh).toContain('当前没有可移除的重复参数。')
  })

  it('clears stale validation results when editable fields change', () => {
    const source = readViewSource()
    const taskTypeSelector = readTaskTypeSelectorSource()

    expect(source).toContain('@input="resetValidationResult"')
    expect(source).toContain('@change="resetValidationResult"')
    expect(source).toContain('@select="chooseType"')
    expect(taskTypeSelector).toContain("@click=\"emit('select', card.type)\"")
    expect(source).toContain('function resetValidationResult()')
    expect(source).toContain('function chooseType')
    expect(source).toContain('validationResult.value = null')
  })

  it('ignores stale template list responses after a newer refresh starts', () => {
    const source = readViewSource()

    expect(source).toContain('let latestTemplateLoadRequestID = 0')
    expect(source).toContain('const requestID = ++latestTemplateLoadRequestID')
    expect(source).toContain('if (!isLatestTemplateLoadRequest(requestID)) return')
    expect(source).toContain('function isLatestTemplateLoadRequest')
    expect(source).toContain('if (isLatestTemplateLoadRequest(requestID))')
  })

  it('keeps task-template operation errors centralized and safe to display', () => {
    const source = readViewSource()
    const errorMessages = readTaskSettingsErrorMessagesSource()

    expect(source).toContain('function showTaskSettingsOperationError')
    expect(source).toContain('recordClientDiagnostic(scope, error)')
    expect(source).toContain("import { createTaskSettingsErrorMessages } from './taskSettingsErrorMessages'")
    expect(source).toContain('const taskSettingsErrorMessages = computed(() => createTaskSettingsErrorMessages(t))')
    expect(source).toContain('extractSafeApiErrorMessage(error, fallback, taskSettingsErrorMessages.value)')
    expect(errorMessages).toContain('TASK_TEMPLATE_SERVICE_UNAVAILABLE')
    expect(errorMessages).toContain('TASK_TEMPLATE_MEDIA_SERVICE_UNAVAILABLE')
    expect(source).toContain("const editorOperationError = ref('')")
    expect(source).toContain('function showTaskSettingsEditorOperationError')
    expect(source).toContain('const message = showTaskSettingsOperationError(scope, error, fallback)')
    expect(source).toContain('editorOperationError.value = message')
    expect(source).toContain('v-if="editorOperationError"')
    expect(source).toContain('role="alert"')
    expect(source).toContain('aria-live="assertive"')
    expect(source).toContain("showTaskSettingsEditorOperationError('task_settings.validate', error, t('taskSettings.validation.failed'))")
    expect(source).toContain("showTaskSettingsEditorOperationError('task_settings.save', error, t('taskSettings.saveFailed'))")
    expect(source).toContain("showTaskSettingsEditorOperationError('task_settings.copy', error, t('taskSettings.copyFailed'))")
    expect(source).toContain("showTaskSettingsEditorOperationError('task_settings.default', error, t('taskSettings.defaultFailed'))")
    expect(source).toContain("showTaskSettingsOperationError('task_settings.delete', error, t('taskSettings.deleteFailed'))")
  })

  it('keeps the visible pool and save payload aligned with the selected task type', () => {
    const source = readViewSource()

    expect(source).toContain("const activePoolKind = computed<'targets' | 'contents' | null>")
    expect(source).toContain("if (targetTypes.has(form.type)) return 'targets'")
    expect(source).toContain("if (form.type === 'post') return 'contents'")
    expect(source).toContain('return { targets: targetValues.value }')
    expect(source).toContain("const params: TaskTemplateInput['params'] = { contents: contentValues.value }")
    expect(source).toContain("label: activePoolKind.value === 'targets' ? t('taskSettings.summary.targets') : t('taskSettings.summary.contents')")
    expect(source).toContain('value: activeValues.value.length')
  })

  it('connects structured post, profile, avatar, and banner editors to the SocialOps-native template flow', () => {
    const source = readViewSource()
    const taskMedia = readTaskMediaSource()
    const templateReadiness = readTemplateReadinessSource()
    const zh = readZhLocaleSource()

    expect(source).toContain('data-testid="quote-post-url-input"')
    expect(source).toContain('data-testid="post-media-empty"')
    expect(source).toContain('data-testid="add-post-media-button"')
    expect(source).toContain('data-testid="avatar-editor"')
    expect(source).toContain('data-testid="banner-editor"')
    expect(source).toContain('data-testid="profile-display-name-input"')
    expect(source).toContain('<ImageUpload')
    expect(source).toContain(':max-size="MAX_TASK_MEDIA_UPLOAD_BYTES"')
    expect(source).toContain("} from './templateReadiness'")
    expect(templateReadiness).toContain('socialPostMediaRefsSupported')
    expect(templateReadiness).toContain('socialTaskMediaRefExecutable')
    expect(source).toContain("if (form.type === 'update_profile')")
    expect(source).toContain("if (form.type === 'update_avatar')")
    expect(source).toContain("if (form.type === 'update_banner')")
    expect(source).toContain("} from './taskMedia'")
    expect(taskMedia).toContain('function normalizeMediaRef')
    expect(taskMedia).toContain('function hasMediaRef')
    expect(source).not.toContain('media_data')
    expect(zh).toContain('管理关注、点赞、转发、发帖和资料更新任务复用的执行参数模板')
    expect(zh).toContain('最多附加 4 张图片用于发帖')
  })

  it('keeps the three-column editor layout for extra-wide screens so the editor stays usable on desktop', () => {
    const source = readViewSource()

    expect(source).toContain('class="grid items-start gap-4 2xl:grid-cols-[320px_minmax(0,1fr)_320px]"')
    expect(source).not.toContain('class="grid items-start gap-4 xl:grid-cols-[320px_minmax(0,1fr)_320px]"')
  })

  it('contains the task type carousel inside the page width on narrow screens', () => {
    const taskTypeSelector = readTaskTypeSelectorSource()
    const templateStatsGrid = readTemplateStatsGridSource()

    expect(taskTypeSelector).toContain('max-w-full rounded-lg border border-gray-200')
    expect(taskTypeSelector).toContain('class="grid max-w-full grid-cols-1 gap-2 sm:grid-cols-2 xl:flex xl:overflow-x-auto"')
    expect(taskTypeSelector).toContain('xl:min-w-[170px]')
    expect(templateStatsGrid).toContain("import CommonStatsGrid")
    expect(templateStatsGrid).toContain('grid-class="grid min-w-0 gap-2 sm:grid-cols-3"')
    expect(templateStatsGrid).toContain('section-test-id="template-stats"')
    expect(templateStatsGrid).toContain('card-class="min-w-0 rounded-lg border border-gray-200 bg-white px-3 py-2.5')
  })

  it('keeps the saved template list as a presentational component wired by the page', () => {
    const source = readViewSource()
    const savedTemplateList = readSavedTemplateListSource()

    expect(source).toContain("import SavedTemplateList from './components/SavedTemplateList.vue'")
    expect(source).toContain('<SavedTemplateList')
    expect(source).toContain(':templates="orderedTemplates"')
    expect(source).toContain(':total-template-count="templates.length"')
    expect(source).toContain(':active-type-label="taskTypeLabel(activeType)"')
    expect(source).toContain(':task-type-label="taskTypeLabel"')
    expect(source).toContain(':task-type-badge-class="taskTypeBadgeClass"')
    expect(source).toContain(':is-template-usable="isTemplateUsable"')
    expect(source).toContain(':template-parameter-state-label="templateParameterStateLabel"')
    expect(source).toContain('@select="selectTemplate"')
    expect(source).toContain('@new-template="newTemplate"')
    expect(savedTemplateList).toContain('data-testid="active-type-empty-state"')
    expect(savedTemplateList).toContain('data-template-card="saved"')
    expect(savedTemplateList).toContain(':data-testid="`saved-template-card-${template.id}`"')
    expect(savedTemplateList).toContain('<Icon name="plus"')
    expect(savedTemplateList).toContain('<Icon name="chevronRight"')
    expect(savedTemplateList).toContain("emit('select', template)")
    expect(savedTemplateList).toContain("emit('new-template')")
  })

  it('keeps task type and template stats as presentational components wired by the page', () => {
    const source = readViewSource()
    const editorActions = readTemplateEditorActionsSource()
    const editorActionTitles = readTemplateEditorActionTitlesSource()
    const enLocaleSource = readEnLocaleSource()
    const zhLocaleSource = readZhLocaleSource()

    expect(source).toContain("import TemplateEditorActions from './components/TemplateEditorActions.vue'")
    expect(source).toContain("import TaskTypeSelector from './components/TaskTypeSelector.vue'")
    expect(source).toContain("import TemplateStatsGrid from './components/TemplateStatsGrid.vue'")
    expect(source).toContain('<TemplateEditorActions')
    expect(source).toContain('@save="saveTemplate"')
    expect(source).toContain('@validate="validateCurrent"')
    expect(source).toContain('@copy="copyCurrentTemplate"')
    expect(source).toContain('@set-default="setDefault"')
    expect(source).toContain('@delete="deleteCurrentTemplate"')
    expect(source).toContain('<TaskTypeSelector :active-type="activeType" :cards="taskTypeCards" @select="chooseType" />')
    expect(source).toContain('<TemplateStatsGrid :stats="templateStats" />')
    expect(editorActions).toContain('const validateButtonLabel = computed(() => buildValidateButtonLabel')
    expect(editorActions).toContain('const copyButtonLabel = computed(() => buildCopyButtonLabel')
    expect(editorActions).toContain('const setDefaultButtonLabel = computed(() => buildSetDefaultButtonLabel')
    expect(editorActions).toContain('const copyButtonTitle = computed(() => buildCopyButtonTitle')
    expect(editorActions).toContain('const setDefaultButtonTitle = computed(() => buildSetDefaultButtonTitle')
    expect(editorActions).toContain('const deleteButtonTitle = computed(() => buildDeleteButtonTitle')
    expect(editorActions).toContain(':aria-label="validateButtonLabel"')
    expect(editorActions).toContain(':aria-label="copyButtonLabel"')
    expect(editorActions).toContain(':aria-label="setDefaultButtonLabel"')
    expect(editorActions).toContain(':title="copyButtonTitle"')
    expect(editorActions).toContain(':title="setDefaultButtonTitle"')
    expect(editorActions).toContain(':title="deleteButtonTitle"')
    expect(editorActionTitles).toContain('export function templateEditorValidateButtonLabel')
    expect(editorActionTitles).toContain('export function templateEditorCopyButtonLabel')
    expect(editorActionTitles).toContain('export function templateEditorSetDefaultButtonLabel')
    expect(editorActionTitles).toContain("const selectedTemplateRequiredKey = 'taskSettings.savedConfigs.selectTemplateFirst'")
    expect(editorActionTitles).toContain('if (!state.hasSelectedTemplate) return t(selectedTemplateRequiredKey)')
    expect(editorActionTitles).toContain("if (state.isDefault) return t('taskSettings.alreadyDefault')")
    expect(enLocaleSource).toContain('"selectTemplateFirst": "Select a saved template first."')
    expect(enLocaleSource).toContain('"alreadyDefault": "This template is already the default."')
    expect(zhLocaleSource).toContain('"selectTemplateFirst": "请先选择一个已保存模板。"')
    expect(zhLocaleSource).toContain('"alreadyDefault": "该模板已经是默认模板。"')
  })

  it('keeps the summary and validation panel presentational and fed by page-computed state', () => {
    const source = readViewSource()
    const summaryPanel = readTemplateSummaryPanelSource()

    expect(source).toContain("import TemplateSummaryPanel, { type TemplateSummaryRow } from './components/TemplateSummaryPanel.vue'")
    expect(source).toContain('const templateSummaryRows = computed<TemplateSummaryRow[]>')
    expect(source).toContain('<TemplateSummaryPanel')
    expect(source).toContain(':rows="templateSummaryRows"')
    expect(source).toContain(':save-disabled-reason="saveDisabledReason"')
    expect(source).toContain(':validation-result="validationResult"')
    expect(summaryPanel).toContain('data-testid="template-summary-panel"')
    expect(summaryPanel).toContain('data-testid="template-validation-panel"')
    expect(summaryPanel).toContain('v-for="row in rows"')
    expect(summaryPanel).toContain("t('taskSettings.status.ready')")
    expect(summaryPanel).toContain("t('taskSettings.validation.valid')")
    expect(summaryPanel).toContain("t('taskSettings.validation.invalid')")
    expect(summaryPanel).toContain("t('taskSettings.summary.executionHint')")
    expect(summaryPanel).not.toContain('defineEmits')
    expect(summaryPanel).not.toContain('<button')
  })

  it('keeps pool analysis presentational while pool editing actions stay on the page', () => {
    const source = readViewSource()
    const poolAnalysisPanel = readTemplatePoolAnalysisPanelSource()

    expect(source).toContain("import TemplatePoolAnalysisPanel from './components/TemplatePoolAnalysisPanel.vue'")
    expect(source).toContain("} from './templatePool'")
    expect(source).toContain('const poolAnalysis = computed<TemplatePoolAnalysis>')
    expect(source).toContain('analyzeTemplatePool(activeValues.value, poolEmptyLineCount.value)')
    expect(source).toContain('<TemplatePoolAnalysisPanel')
    expect(source).toContain(':analysis="poolAnalysis"')
    expect(source).toContain(':capacity-message="poolCapacityMessage"')
    expect(source).toContain(':max-value-length="MAX_TEMPLATE_VALUE_LENGTH"')
    expect(source).toContain('data-testid="import-button"')
    expect(source).toContain('data-testid="view-all-button"')
    expect(source).toContain('data-testid="dedupe-button"')
    expect(source).toContain('data-testid="clear-pool-button"')
    expect(source).toContain('const canClearActiveValues = computed(() => activeValuesText.value.length > 0)')
    expect(source).toContain(':disabled="saving || !canClearActiveValues"')
    expect(source).toContain('if (!canClearActiveValues.value) return')
    expect(source).toContain(':data-testid="activePoolKind === \'targets\' ? \'target-pool-textarea\' : \'content-pool-textarea\'"')
    expect(poolAnalysisPanel).toContain('data-testid="template-pool-analysis-panel"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-valid"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-empty-lines"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-duplicates"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-too-long"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-capacity"')
    expect(poolAnalysisPanel).toContain('data-testid="pool-empty-lines-hint"')
    expect(poolAnalysisPanel).toContain("t('taskSettings.pool.duplicateHint'")
    expect(poolAnalysisPanel).toContain("t('taskSettings.pool.tooLongHint'")
    expect(poolAnalysisPanel).not.toContain('defineEmits')
    expect(poolAnalysisPanel).not.toContain('<button')
  })

  it('keeps task-settings limited to parameterized actions', () => {
    const source = readViewSource()
    const templateReadiness = readTemplateReadinessSource()

    expect(source).toContain('v-if="activePoolKind"')
    expect(source).toContain('const TASK_TYPES = TASK_SETTINGS_PARAMETER_TASK_TYPES')
    expect(templateReadiness).toContain("export const TASK_SETTINGS_PARAMETER_TASK_TYPES: ParameterTaskTemplateType[] = [")
    expect(templateReadiness).toContain("'follow'")
    expect(templateReadiness).toContain("'like'")
    expect(templateReadiness).toContain("'retweet'")
    expect(templateReadiness).toContain("'post'")
    expect(templateReadiness).toContain("'update_profile'")
    expect(templateReadiness).toContain("'update_avatar'")
    expect(templateReadiness).toContain("'update_banner'")
    expect(source).toContain('function isParameterTaskTemplate')
    expect(source).toContain('loadedTemplates.filter(isParameterTaskTemplate)')
  })

  it('uses an in-app confirmation dialog for deleting templates', () => {
    const source = readViewSource()

    expect(source).toContain('deleteDialogOpen')
    expect(source).toContain('templateToDelete')
    expect(source).toContain('function confirmDeleteTemplate')
    expect(source).toContain("t('taskSettings.deleteDialog.title')")
    expect(source).toContain("t('taskSettings.deleteDialog.description'")
    expect(source).not.toContain('window.confirm')
    expect(source).not.toContain("t('taskSettings.deleteConfirm'")
  })

  it('uses the same type-aware params in save payloads and delete fallback summaries', () => {
    const source = readViewSource()

    expect(source).toContain('function buildTemplateParams')
    expect(source).toContain('params: buildTemplateParams()')
    expect(source).not.toContain('targets: targetValues.value,\n      contents: contentValues.value')
  })

  it('preserves commas inside post content while allowing comma-separated targets', () => {
    const source = readViewSource()
    const templatePool = readTemplatePoolSource()

    expect(source).toContain('const targetValues = computed(() => splitTargetValues(form.targetsText))')
    expect(source).toContain('const contentValues = computed(() => splitContentValues(form.contentsText))')
    expect(source).toContain("} from './templatePool'")
    expect(templatePool).toContain('function splitTargetValues(value: string): string[]')
    expect(templatePool).toContain('function splitContentValues(value: string): string[]')
    expect(templatePool).toContain('.split(/\\r?\\n|,/)')
    expect(templatePool).toContain('.split(/\\r?\\n/)')
    expect(source).not.toContain('const contentValues = computed(() => splitValues(form.contentsText))')
  })

  it('guards parameter file imports against stale editor context switches and pending operations', () => {
    const source = readViewSource()
    const importSource = source.slice(
      source.indexOf('async function handleFileImport'),
      source.indexOf('function clearValues'),
    )

    expect(importSource).toContain('const importPoolKind = activePoolKind.value')
    expect(importSource).toContain('const importType = form.type')
    expect(importSource).toContain('const importTemplateId = selectedTemplateId.value')
    expect(importSource).toContain('if (saving.value || !file || !importPoolKind)')
    expect(importSource).toContain("editorOperationError.value = ''")
    expect(importSource).toContain('if (saving.value || activePoolKind.value !== importPoolKind || form.type !== importType || selectedTemplateId.value !== importTemplateId) return')
    expect(importSource).toContain("editorOperationError.value = t('taskSettings.importFailed')")
    expect(importSource).toContain('appStore.showError(editorOperationError.value)')
  })

  it('guards media update handlers while a template operation is pending', () => {
    const source = readViewSource()
    const postMediaSource = source.slice(
      source.indexOf('function updatePostMedia'),
      source.indexOf('async function updateAvatarMedia'),
    )
    const avatarMediaSource = source.slice(
      source.indexOf('async function updateAvatarMedia'),
      source.indexOf('async function updateBannerMedia'),
    )
    const bannerMediaSource = source.slice(
      source.indexOf('async function updateBannerMedia'),
      source.indexOf('function resetValidationResult'),
    )

    expect(postMediaSource).toContain('if (saving.value) return')
    expect(avatarMediaSource).toContain('if (saving.value) return')
    expect(bannerMediaSource).toContain('if (saving.value) return')
  })

  it('keeps post media action disabled titles aligned with saving and capacity limits', () => {
    const source = readViewSource()
    const editorActionTitles = readTemplateEditorActionTitlesSource()

    expect(source).toContain(':title="addPostMediaButtonTitle"')
    expect(source).toContain(':aria-label="addPostMediaButtonTitle"')
    expect(source).toContain(':title="removePostMediaButtonTitle"')
    expect(source).toContain(':aria-label="removePostMediaButtonTitle"')
    expect(source).toContain('templateEditorAddPostMediaButtonTitle as buildAddPostMediaButtonTitle')
    expect(source).toContain('templateEditorRemovePostMediaButtonTitle as buildRemovePostMediaButtonTitle')
    expect(source).toContain('const addPostMediaButtonTitle = computed(() => buildAddPostMediaButtonTitle')
    expect(source).toContain('mediaCount: postMediaCount.value')
    expect(source).toContain('maxMediaItems: MAX_POST_MEDIA_ITEMS')
    expect(source).toContain('const removePostMediaButtonTitle = computed(() => buildRemovePostMediaButtonTitle')
    expect(editorActionTitles).toContain('export function templateEditorAddPostMediaButtonTitle')
    expect(editorActionTitles).toContain('export function templateEditorRemovePostMediaButtonTitle')
  })

  it('keeps Chinese task-settings copy natural and free of exposed technical English labels', () => {
    const source = readZhLocaleSource()

    expect(source).toContain('管理关注、点赞、转发、发帖和资料更新任务复用的执行参数模板')
    expect(source).toContain('目标：帖子链接或帖子 ID')
    expect(source).toContain('内容池：每行一条发帖内容')
    expect(source).not.toContain('Targets：')
    expect(source).not.toContain('Targets:')
    expect(source).not.toContain('Contents：')
    expect(source).not.toContain('Contents:')
    expect(source).not.toContain('Tweet URL')
    expect(source).not.toContain('Tweet ID')
    expect(source).not.toContain('tweetTargetsPlaceholder')
  })
})
