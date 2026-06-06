import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const viewPath = resolve(__dirname, '../TaskSettingsView.vue')
const zhLocalePath = resolve(__dirname, '../../../i18n/locales/zh.ts')

function readViewSource(): string {
  return readFileSync(viewPath, 'utf8')
}

function readZhLocaleSource(): string {
  return readFileSync(zhLocalePath, 'utf8')
}

describe('TaskSettingsView source contract', () => {
  it('keeps the page-level duplicate heading out and keeps editor actions non-duplicative', () => {
    const source = readViewSource()

    expect(source).not.toContain('<h1')
    expect(source).not.toContain("t('taskSettings.title')")
    expect(source).not.toContain("t('taskSettings.description')")
    expect(source).toContain('data-testid="editor-template-actions"')
    expect(source).toContain('data-testid="save-template-button"')
    expect(source).toContain('data-testid="validation-button"')
    expect(source).not.toContain('data-testid="save-template-button-secondary"')
  })

  it('keeps save readiness aligned with template completeness', () => {
    const source = readViewSource()

    expect(source).toContain('const saveDisabledReason = computed')
    expect(source).toContain('MAX_TEMPLATE_POOL_VALUES = 500')
    expect(source).toContain('MAX_TEMPLATE_VALUE_LENGTH = 2048')
    expect(source).toContain("t('taskSettings.validation.nameRequired')")
    expect(source).toContain("t('taskSettings.validation.targetsRequired')")
    expect(source).toContain("t('taskSettings.validation.contentsRequired')")
    expect(source).toContain("t('taskSettings.validation.tooManyValues'")
    expect(source).toContain("t('taskSettings.validation.valueTooLong'")
    expect(source).toContain('Array.from(value).length > MAX_TEMPLATE_VALUE_LENGTH')
    expect(source).toContain('const canSave = computed(() => !saveDisabledReason.value)')
    expect(source).toContain(':title="saveDisabledReason || undefined"')
    expect(source).not.toContain("const canSave = computed(() => form.name.trim() !== '')")
  })

  it('clears stale validation results when editable fields change', () => {
    const source = readViewSource()

    expect(source).toContain('@input="resetValidationResult"')
    expect(source).toContain('@change="resetValidationResult"')
    expect(source).toContain('@click="chooseType(meta.type)"')
    expect(source).toContain('function resetValidationResult()')
    expect(source).toContain('function chooseType')
    expect(source).toContain('validationResult.value = null')
  })

  it('keeps the visible pool and save payload aligned with the selected task type', () => {
    const source = readViewSource()

    expect(source).toContain("const activePoolKind = computed<'targets' | 'contents' | null>")
    expect(source).toContain("if (targetTypes.has(form.type)) return 'targets'")
    expect(source).toContain("if (form.type === 'post') return 'contents'")
    expect(source).toContain('if (targetTypes.has(form.type)) return { targets: targetValues.value }')
    expect(source).toContain("if (form.type === 'post') return { contents: contentValues.value }")
    expect(source).toContain("activePoolKind === 'targets' ? t('taskSettings.summary.targets') : t('taskSettings.summary.contents')")
    expect(source).toContain('{{ activeValues.length }}')
  })

  it('keeps post templates text-only until media execution is actually connected', () => {
    const source = readViewSource()
    const zh = readZhLocaleSource()

    expect(source).toContain("t('taskSettings.media.postTextOnlyHint')")
    expect(source).not.toContain('accept="image')
    expect(source).not.toContain('accept="video')
    expect(source).not.toContain('media_data')
    expect(zh).toContain('当前发帖模板仅配置文本内容池')
    expect(zh).toContain('图片或视频附件暂未接入此模板')
  })

  it('keeps login checks out of the task-settings page', () => {
    const source = readViewSource()

    expect(source).toContain('v-if="activePoolKind"')
    expect(source).toContain("const TASK_TYPES: ParameterTaskTemplateType[] = ['follow', 'like', 'retweet', 'post']")
    expect(source).toContain('function isParameterTaskTemplate')
    expect(source).toContain('loadedTemplates.filter(isParameterTaskTemplate)')
    expect(source).not.toContain('data-testid="login-check-zero-params"')
    expect(source).not.toContain('task-type-login_check')
    expect(source).not.toContain("form.type === 'login_check'")
    expect(source).not.toContain("t('taskSettings.loginCheckReady')")
    expect(source).not.toContain("t('taskSettings.typeRequirements.login_check')")
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

    expect(source).toContain('const targetValues = computed(() => splitTargetValues(form.targetsText))')
    expect(source).toContain('const contentValues = computed(() => splitContentValues(form.contentsText))')
    expect(source).toContain('function splitTargetValues(value: string): string[]')
    expect(source).toContain('function splitContentValues(value: string): string[]')
    expect(source).toContain('.split(/\\r?\\n|,/)')
    expect(source).toContain('.split(/\\r?\\n/)')
    expect(source).not.toContain('const contentValues = computed(() => splitValues(form.contentsText))')
  })

  it('keeps Chinese task-settings copy natural and free of exposed technical English labels', () => {
    const source = readZhLocaleSource()

    expect(source).toContain('管理关注、点赞、转发和发帖任务复用的执行参数模板')
    expect(source).toContain('目标：推文链接或推文 ID')
    expect(source).toContain('内容池：每行一条发帖内容')
    expect(source).not.toContain('Targets：')
    expect(source).not.toContain('Targets:')
    expect(source).not.toContain('Contents：')
    expect(source).not.toContain('Contents:')
    expect(source).not.toContain('Tweet URL')
    expect(source).not.toContain('Tweet ID')
  })
})
