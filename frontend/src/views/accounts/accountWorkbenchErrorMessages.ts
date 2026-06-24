type AccountWorkbenchErrorTranslateFn = (key: string) => string

export function createAccountWorkbenchErrorMessages(t: AccountWorkbenchErrorTranslateFn): Record<string, string> {
  return {
    SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_SERVICE_UNAVAILABLE'),
    SOCIAL_ACCOUNT_INPUT_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_INPUT_REQUIRED'),
    SOCIAL_ACCOUNT_PASSWORD_REQUIRED: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_PASSWORD_REQUIRED'),
    SOCIAL_ACCOUNT_IMPORT_INCOMPLETE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_IMPORT_INCOMPLETE'),
    SOCIAL_ACCOUNT_DUPLICATE: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_DUPLICATE'),
    SOCIAL_ACCOUNT_NOT_FOUND: t('admin.socialAccountWorkbench.errors.SOCIAL_ACCOUNT_NOT_FOUND'),
    SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID: t('accountWorkbench.edit.errors.SOCIAL_ACCOUNT_EXECUTION_AUTH_INVALID'),
    SOCIAL_IP_SERVICE_UNAVAILABLE: t('proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE'),
    SOCIAL_IP_NOT_AVAILABLE: t('accountWorkbench.proxy.errors.SOCIAL_IP_NOT_AVAILABLE'),
    GLOBAL_PROXY_NOT_AVAILABLE: t('accountWorkbench.execution.errors.GLOBAL_PROXY_NOT_AVAILABLE'),
    GLOBAL_PROXY_SERVICE_UNAVAILABLE: t('accountWorkbench.execution.errors.GLOBAL_PROXY_SERVICE_UNAVAILABLE'),
    SOCIAL_IP_NOT_FOUND: t('accountWorkbench.proxy.errors.SOCIAL_IP_NOT_FOUND'),
    SOCIAL_IP_POOL_EMPTY: t('accountWorkbench.proxy.errors.SOCIAL_IP_POOL_EMPTY'),
    SOCIAL_IP_REQUIRED: t('accountWorkbench.proxy.errors.SOCIAL_IP_REQUIRED'),
    SOCIAL_IP_ASSIGNMENT_MODE_INVALID: t('accountWorkbench.proxy.errors.SOCIAL_IP_ASSIGNMENT_MODE_INVALID'),
  }
}
