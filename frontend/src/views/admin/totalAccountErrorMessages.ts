type TotalAccountErrorTranslateFn = (key: string) => string

export function createTotalAccountErrorMessages(t: TotalAccountErrorTranslateFn): Record<string, string> {
  return {
    SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE'),
    SOCIAL_ACCOUNT_INPUT_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED'),
    SOCIAL_ACCOUNT_NAME_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NAME_REQUIRED'),
    SOCIAL_ACCOUNT_PLATFORM_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PLATFORM_REQUIRED'),
    SOCIAL_ACCOUNT_IDENTITY_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IDENTITY_REQUIRED'),
    SOCIAL_ACCOUNT_PASSWORD_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED'),
    SOCIAL_ACCOUNT_IMPORT_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_REQUIRED'),
    SOCIAL_ACCOUNT_IMPORT_INCOMPLETE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE'),
    SOCIAL_ACCOUNT_DUPLICATE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_DUPLICATE'),
    SOCIAL_ACCOUNT_NOT_FOUND: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND'),
    SOCIAL_ACCOUNT_ALREADY_ASSIGNED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ALREADY_ASSIGNED'),
    SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_ASSIGNMENT_CHANGED'),
    SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID'),
    USER_NOT_FOUND: t('admin.socialAccountWorkbench.errors.USER_NOT_FOUND'),
  }
}
