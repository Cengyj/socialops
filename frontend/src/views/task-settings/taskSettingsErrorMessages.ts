type TaskSettingsErrorTranslateFn = (key: string) => string

export function createTaskSettingsErrorMessages(t: TaskSettingsErrorTranslateFn): Record<string, string> {
  return {
    TASK_TEMPLATE_SERVICE_UNAVAILABLE: t('taskSettings.errors.serviceUnavailable'),
    TASK_TEMPLATE_INPUT_REQUIRED: t('taskSettings.errors.templateMissing'),
    TASK_TEMPLATE_ID_REQUIRED: t('taskSettings.errors.templateMissing'),
    TASK_TEMPLATE_NOT_FOUND: t('taskSettings.errors.templateNotFound'),
    TASK_TEMPLATE_NAME_REQUIRED: t('taskSettings.validation.nameRequired'),
    TASK_TEMPLATE_INVALID: t('taskSettings.errors.templateInvalid'),
    TASK_TEMPLATE_STORE_INVALID: t('taskSettings.errors.storeInvalid'),
    TASK_TEMPLATE_MEDIA_SERVICE_UNAVAILABLE: t('taskSettings.errors.mediaServiceUnavailable'),
    TASK_TEMPLATE_MEDIA_STORAGE_KEY_REQUIRED: t('taskSettings.errors.mediaMissing'),
    TASK_TEMPLATE_MEDIA_SOURCE_UNSUPPORTED: t('taskSettings.validation.mediaSourceUnsupported'),
    SOCIAL_TASK_UNSUPPORTED_ACTION: t('taskSettings.validation.unsupportedType'),
  }
}
