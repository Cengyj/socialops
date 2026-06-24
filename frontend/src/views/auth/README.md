# Authentication Views

This directory contains the route-level authentication views used by the SocialOps frontend.

## Runtime Entry

Authentication views are loaded directly from `frontend/src/router/index.ts` with lazy imports. Keep new imports explicit, for example:

```ts
component: () => import('@/views/auth/LoginView.vue')
```

Do not reintroduce a directory barrel for these views unless there is a real runtime caller. The previous `@/views/auth` barrel only exported a partial set of views and made stale examples look like supported API.

## Current View Set

- `LoginView.vue`: email/password login, Turnstile gate, login agreement, OAuth entry points, password reset link, and TOTP follow-up modal.
- `RegisterView.vue`: email/password registration with optional invitation code, promo code, Turnstile, login agreement, email verification handoff, affiliate code capture, and OAuth entry points.
- `EmailVerifyView.vue`: email verification completion and resend flow.
- `ForgotPasswordView.vue` and `ResetPasswordView.vue`: password reset request and completion.
- `OAuthCallbackView.vue`: generic email OAuth callback.
- `LinuxDoCallbackView.vue`, `DingTalkCallbackView.vue`, `OidcCallbackView.vue`, and `WechatCallbackView.vue`: provider-specific OAuth adoption, binding, and registration completion flows.
- `DingTalkEmailCompletionView.vue`: DingTalk account completion path.
- `WechatPaymentCallbackView.vue`: payment-related WeChat OAuth callback.

## Maintenance Rules

- Keep auth views as route-level screens; shared UI belongs under `frontend/src/components/auth`.
- Preserve existing SaaS auth behavior when cleaning code: refresh tokens, pending OAuth sessions, account adoption, email verification, login agreements, and TOTP are current product flows.
- Avoid restoring stale username-only or remember-me examples. The current primary credential is email plus password.
- Keep route metadata and feature gates in `frontend/src/router/index.ts`; do not mirror router truth in long example documents.
