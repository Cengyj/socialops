import { describe, expect, it } from 'vitest'

import { createTaskSettingsErrorMessages } from '../taskSettingsErrorMessages'

describe('task settings error messages', () => {
  it('maps backend task-template error codes to translated safe messages', () => {
    const messages = createTaskSettingsErrorMessages(key => `translated:${key}`)

    expect(messages).toMatchObject({
      TASK_TEMPLATE_SERVICE_UNAVAILABLE: 'translated:taskSettings.errors.serviceUnavailable',
      TASK_TEMPLATE_INPUT_REQUIRED: 'translated:taskSettings.errors.templateMissing',
      TASK_TEMPLATE_ID_REQUIRED: 'translated:taskSettings.errors.templateMissing',
      TASK_TEMPLATE_NOT_FOUND: 'translated:taskSettings.errors.templateNotFound',
      TASK_TEMPLATE_NAME_REQUIRED: 'translated:taskSettings.validation.nameRequired',
      TASK_TEMPLATE_INVALID: 'translated:taskSettings.errors.templateInvalid',
      TASK_TEMPLATE_STORE_INVALID: 'translated:taskSettings.errors.storeInvalid',
      TASK_TEMPLATE_MEDIA_SERVICE_UNAVAILABLE: 'translated:taskSettings.errors.mediaServiceUnavailable',
      TASK_TEMPLATE_MEDIA_STORAGE_KEY_REQUIRED: 'translated:taskSettings.errors.mediaMissing',
      TASK_TEMPLATE_MEDIA_SOURCE_UNSUPPORTED: 'translated:taskSettings.validation.mediaSourceUnsupported',
      SOCIAL_TASK_UNSUPPORTED_ACTION: 'translated:taskSettings.validation.unsupportedType',
    })
  })
})
