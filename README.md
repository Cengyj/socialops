# SocialOps

> Developer note: this repository is being migrated from an AI gateway codebase into a website account-pool based social operations platform. Read [PROJECT_GUIDE.md](PROJECT_GUIDE.md) before changing code.

SocialOps is a SaaS platform for operating social accounts from a website-owned account pool. The system only handles accounts already present in the total account pool. Administrators import or truly register social accounts, assign or reclaim unassigned accounts, and users may import platform usernames such as X `@xxx` only to match existing unassigned pool accounts. Real social execution actions such as follow, message, post, like, and login checks keep a submit, validate, queue, log, and billing framework in this phase; real social executors are attached later, and only successful executions are finally charged.

[中文](README_CN.md) | [日本語](README_JA.md)

## Current Direction

- **Total social account pool**: only pool accounts are processed; external unmatched accounts are not stored, assigned, executed, or billed.
- **Account onboarding**: admin import with existing de-duplication rules, true social account registration, manual assignment, reclaim to unassigned, and user import-by-platform-username matching; ambiguous username matches are rejected.
- **Task execution**: user-facing social operation tasks with future backend execution workers; unavailable executors fail closed.
- **IP and proxy management**: user-owned execution resources managed from `/proxies`; every role, including administrators, can only manage their own proxies. Accounts may keep a tested online default execution proxy, account reclaim or proxy deletion clears the default proxy, and IP/proxy tests are free.
- **Execution billing**: real social execution actions use the built-in unit price `0.1`; account/IP management actions are free, the existing subscription/wallet framework is reused without redesign, subscription monetary allowance is consumed first, wallet balance second, and only successful executions are finally charged while non-success results are refunded/released or left uncharged.
- **SaaS foundation**: authentication, JWT refresh, OAuth login, profile management, 2FA, admin users, settings, announcements, subscriptions, payments, balance recharge, redeem codes, promo codes, and affiliate rebates.
- **Embedded frontend**: Vue/Vite build output is embedded into the Go binary with `-tags embed`.

## Implementation Status

The current migration has completed the audit, AI gateway cleanup, SaaS foundation recovery, SocialOps backend and frontend framework checkpoints, full generate/test/build, Docker Compose health smoke, hostile G008 validation, ai-slop-cleaner no-op review, independent code review, and UltraQA-equivalent adversarial QA. `PROJECT_STATUS.md` tracks the live G001-G008 evidence.

Real social-platform registrars and executors remain integration points. Until they are connected, admin registration and user execution actions fail closed and must not create fake successful accounts or final charges.

## Explicit Non-Goals

The AI gateway business is not part of SocialOps. Do not restore AI provider routing, model channels, upstream account credential pools, model pricing, token billing, gateway monitoring, or provider-specific OAuth/token refresh flows.

The current SocialOps phase also does not implement account purchase flow first, real social-platform executors, separate sold/rented account states, automatic rental expiry, a new wallet/subscription billing system, or extra masking/application-layer encryption for account data that belongs to the current user.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.26.3, Gin, Ent |
| Frontend | Vue 3, Vite, TypeScript, TailwindCSS |
| Database | PostgreSQL |
| Cache/Queue | Redis |
| Deployment | Docker Compose, systemd, embedded frontend binary |

## Repository Layout

```text
socialops/
├── backend/        # Go module; run Go commands here
├── frontend/       # Vue 3 app; pnpm only
├── deploy/         # Docker/systemd/install scripts and local runtime data
├── docs/           # Payment and administration docs
├── tools/          # Audit and secret-scan helpers
├── Makefile        # Root orchestration for build/test
└── Dockerfile      # Frontend build -> Go embed build -> final image
```

## Local Build

```bash
# Frontend
cd frontend
pnpm install
pnpm run build

# Backend with embedded UI
cd ../backend
go build -tags embed -o socialops ./cmd/server
```

## Validation

```bash
make build
make test
make test-frontend
make secret-scan

cd backend && go test -tags=unit ./...
cd frontend && pnpm run typecheck && pnpm run build && pnpm run lint:check
```

## Deployment

Use the files under `deploy/`. For development images:

```bash
cd deploy
docker compose -f docker-compose.dev.yml build
docker compose -f docker-compose.dev.yml up -d --no-build
```

Do not clear runtime database or Redis volumes unless you explicitly intend to destroy local data.

## License

This project is licensed under the [GNU Lesser General Public License v3.0](LICENSE) or later.
