type ProxyErrorTranslateFn = (key: string) => string

export function createProxyErrorMessages(t: ProxyErrorTranslateFn): Record<string, string> {
  return {
    SOCIAL_IP_SERVICE_UNAVAILABLE: t('proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE'),
    SOCIAL_IP_INPUT_REQUIRED: t('proxies.errors.SOCIAL_IP_INPUT_REQUIRED'),
    SOCIAL_IP_NAME_REQUIRED: t('proxies.errors.SOCIAL_IP_NAME_REQUIRED'),
    SOCIAL_IP_TYPE_INVALID: t('proxies.errors.SOCIAL_IP_TYPE_INVALID'),
    INVALID_PROXY_ENDPOINT: t('proxies.errors.INVALID_PROXY_ENDPOINT'),
    SOCIAL_IP_NOT_FOUND: t('proxies.errors.SOCIAL_IP_NOT_FOUND'),
    SOCIAL_IP_OWNER_NOT_FOUND: t('proxies.errors.SOCIAL_IP_OWNER_NOT_FOUND'),
    SOCIAL_IP_USER_ID_NOT_ACCEPTED: t('proxies.errors.SOCIAL_IP_USER_ID_NOT_ACCEPTED'),
  }
}
