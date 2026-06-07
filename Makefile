.PHONY: build build-backend build-frontend test test-backend test-frontend test-frontend-critical secret-scan dev-services-up dev-services-down dev-backend dev-frontend

DEV_ENV_FILE ?= deploy/.env.host-dev

FRONTEND_CRITICAL_VITEST := \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-frontend-critical:
	@pnpm --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)

secret-scan:
	@python3 tools/secret_scan.py

dev-services-up:
	@docker compose --env-file $(DEV_ENV_FILE) -f deploy/docker-compose.host-dev.yml up -d

dev-services-down:
	@docker compose --env-file $(DEV_ENV_FILE) -f deploy/docker-compose.host-dev.yml down

dev-backend:
	@./tools/dev/run-backend-watch.sh $(DEV_ENV_FILE)

dev-frontend:
	@cd frontend && VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080 pnpm run dev
