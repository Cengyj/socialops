# SocialOps

> Developer note: this repository is being migrated from an AI gateway codebase into a website account-pool based social operations platform. Read [PROJECT_GUIDE.md](PROJECT_GUIDE.md) before changing code.

SocialOps is a SaaS platform for operating social accounts from a website-owned account pool. Administrators import or truly register social accounts, assign or reclaim unassigned accounts, and user imports first match platform usernames such as X `@xxx` against existing unassigned pool accounts. Unmatched imports are staged only as not-stored user workbench accounts; they do not enter the total pool, execute tasks, or bill until an administrator uploads them into the pool. Twitter/X execution currently uses the existing submit, validate, queue, executor, log, and billing flow for supported actions such as login checks, follow, post, like, and retweet; unsupported actions or unavailable executors fail closed, and only successful executions are finally charged.

[中文](README_CN.md) | [日本語](README_JA.md)

## Current Direction

- **Total social account pool**: the pool is the only inventory for assignable and executable accounts; unmatched imports are staged in the user workbench and are not pool accounts, executable, or billable until uploaded into the pool.
- **Account onboarding**: admin import with existing de-duplication rules, true social account registration, manual assignment, reclaim to unassigned, user import-by-platform-username matching, and not-stored workbench staging for unmatched imports; ambiguous username matches are rejected.
- **Task execution**: user-facing Twitter/X social operation tasks with the current backend executor; unsupported platforms or unavailable actions fail closed.
- **IP and proxy management**: user-owned execution resources managed from `/proxies`; every role, including administrators, can only manage their own proxies. Accounts may keep a tested online default execution proxy, account reclaim or proxy deletion clears the default proxy, and IP/proxy tests are free.
- **Execution billing**: real social execution actions use the built-in unit price `0.1`; account/IP management actions are free, the existing subscription/wallet framework is reused without redesign, subscription monetary allowance is consumed first, wallet balance second, and only successful executions are finally charged while non-success results are refunded/released or left uncharged.
- **SaaS foundation**: authentication, JWT refresh, OAuth login, profile management, 2FA, admin users, settings, announcements, subscriptions, payments, balance recharge, redeem codes, promo codes, and affiliate rebates.
- **Embedded frontend**: Vue/Vite build output is embedded into the Go binary with `-tags embed`.

## Implementation Notes

This repository still carries some historical migration context from its AI gateway origin. `PROJECT_GUIDE.md` is the authoritative project-level guide for current SocialOps boundaries, cleanup rules, and known migration debt.

Social-platform registrars and executors remain bounded integration points. The current Twitter/X registrar and executor must fail closed when credentials, proxies, media, or platform responses are unavailable, and they must not create fake successful accounts or final charges.

## Explicit Non-Goals

The AI gateway business is not part of SocialOps. Do not restore AI provider routing, model channels, upstream account credential pools, model pricing, token billing, gateway monitoring, or provider-specific OAuth/token refresh flows.

The current SocialOps phase also does not implement account purchase flow first, additional social-platform executors beyond the current Twitter/X integration, separate sold/rented account states, automatic rental expiry, a new wallet/subscription billing system, or extra masking/application-layer encryption for account data that belongs to the current user.

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
