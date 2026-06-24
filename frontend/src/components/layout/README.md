# Layout Components

This directory contains the shared shell components for the SocialOps frontend.

## Components

- `AppLayout.vue`: authenticated application shell that combines the sidebar, header, mobile overlay behavior, and page slot.
- `AppSidebar.vue`: role-aware navigation for the current SocialOps SaaS surface, including account workbench, task settings, proxies, usage, subscriptions, payment/order areas, redeem/promo flows, affiliate records, announcements, users, settings, custom menu entries, and simple-mode filtering.
- `AppHeader.vue`: top bar with mobile sidebar toggle, route title, balance display, profile menu, and logout action.
- `AuthLayout.vue`: centered public auth shell used by login, registration, verification, reset, and OAuth callback views.
- `TablePageLayout.vue`: constrained table-page structure used by dense management and workbench screens; import it directly from `@/components/layout/TablePageLayout.vue`.

## Import Pattern

Existing auth views import `AuthLayout` from the layout barrel:

```ts
import { AuthLayout } from '@/components/layout'
```

Most authenticated pages import shell components directly:

```ts
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
```

Keep this boundary explicit unless a real repeated import pattern needs the barrel. Avoid adding example-only exports.

## Route Titles

Page titles and descriptions come from route metadata in `frontend/src/router/index.ts` and are rendered by the shell. Keep route metadata as the source of truth instead of duplicating route maps in component docs.

## Navigation Notes

- Admin navigation is grouped around current SocialOps operations: dashboard, users, account workbench, task settings, proxies, total account pool, subscriptions, payment plans, announcements, redeem/promo codes, affiliate records, orders, and settings.
- User navigation covers dashboard, assigned accounts, task settings, proxies, usage, subscriptions, purchases, orders, redeem, affiliate, profile, and configured custom pages.
- Feature-flagged items use `frontend/src/utils/featureFlags.ts` and public/admin settings already loaded by the app stores.
- Custom menu SVG is sanitized with `sanitizeSvg` before rendering.

Do not document removed admin modules or old gateway concepts here. The sidebar README should describe the current shell, not serve as a historical route catalog.
