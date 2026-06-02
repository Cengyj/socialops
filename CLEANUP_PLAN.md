# SocialOps 长期清理改造方案

> **本文档是 PROJECT_GUIDE.md 的执行细则。**
> PROJECT_GUIDE.md 说明"做什么"，本文档说明"怎么做、按什么顺序、如何验证"。
>
> **核心原则：每一批删除后必须编译通过才能继续下一批。出错立即停止，不要强行推进。**

---

## 总体节奏

> **当前审计状态（2026-06-02）：** AI 网关核心清理、社交账号总账号池、用户导入匹配、代理管理、任务日志、失败关闭和成功扣费框架已基本成型；真实社交执行器、管理员真实注册器、部分通用后台框架、社交化仪表盘/用量展示、容器服务最终更新仍未完成。后续继续按本文件的验收口径核查，不因已有实现而跳过验证。

```
Phase 1 → Phase 2A → 2B → 2C → 2D → 2E → Phase 3A → 3B → 3C
          ↑每步都要按当前代码重新验证↑
```

本文件描述执行顺序、验收口径和当前下一步重点；如果某个阶段在当前代码中已经完成，把对应章节当作审计清单使用。

### 当前下一步执行重点

1. **真实执行器接入**：参考 `/home/ceng/Downloads/FlyingBird` 的登录、关注、点赞、发帖等实现，迁移为当前项目结构下的可插拔 SocialOps 执行器；不要照搬 FlyingBird 的路由、模型或全局状态，必须适配当前账号池、代理、日志、计费和错误返回标准。
2. **社交账号 UI 对接执行功能**：迁移完成后，把 `/admin/accounts` 和 `/admin/total-accounts` 中的执行入口对接到当前任务接口，确保 UI 展示 pending/running/success/failed、扣费状态、失败原因和代理快照。
3. **补齐通用后台框架**：恢复或重写为 SocialOps 口径的订阅分组管理、风控、仪表盘聚合、用量清理和用量展示；不能继续把这些接口保留为 skeleton/disabled。
4. **清理残留 AI 语义**：重点检查设置页、仪表盘、用量页、测试和文案中的 Token/模型/渠道/AI 账号语义；保留通用 API Key、OAuth token、支付 gateway 等非 AI 概念。
5. **验证和容器更新**：完成代码后依次运行后端 unit、前端 typecheck/build、必要 E2E、Docker 构建和 HTTP smoke，再更新容器中的服务。

Phase 2 分 5 个批次，从"最安全、依赖最少"到"需要手术式修改"逐步推进：

| 批次 | 内容 | 文件数 | 风险 |
|------|------|--------|------|
| 2A | 删除 AI 网关路由和 Handler | ~30 | 低 |
| 2B | 删除纯 AI Service（OpenAI/Gemini/Antigravity/Bedrock） | ~150 | 低 |
| 2C | 删除 Ops 监控、Channel、Scheduler、AI 账号 | ~120 | 低 |
| 2D | 删除 AI Ent Schema，重新生成 ORM | ~11 | 中 |
| 2E | 手术式修改混用文件（billing/subscription/usage） | ~8 | 高 |

---

## Phase 2A：删除 AI 网关路由和 Handler

**目标：** 让后端不再注册任何 AI 网关路由，但业务逻辑代码暂时保留。

**为什么先做这步：** Handler 依赖 Service，但 Service 不依赖 Handler。先删 Handler 可以切断路由入口，同时不影响 Service 层的编译。

### 2A-1 删除路由文件

```bash
cd backend
rm internal/server/routes/gateway.go
```

然后检查 `internal/server/routes/` 下是否有其他文件引用了 gateway.go 中的函数：

```bash
grep -rn "RegisterGatewayRoutes\|gateway\." internal/server/ --include="*.go"
```

找到引用后，从调用处删除对应的函数调用行。

### 2A-2 删除 Handler 根目录的 AI 文件

```bash
cd backend/internal/handler
rm gateway_handler.go
rm gateway_handler_chat_completions.go
rm gateway_handler_responses.go
rm gateway_helper.go
rm openai_gateway_handler.go
rm openai_chat_completions.go
rm openai_images.go
rm gemini_v1beta_handler.go
rm available_channel_handler.go
rm channel_monitor_user_handler.go
rm failover_loop.go
rm idempotency_helper.go
rm image_concurrency_limiter.go
rm request_body_limit.go
rm ops_error_logger.go
```

### 2A-3 删除 admin Handler 的 AI 文件

```bash
cd backend/internal/handler/admin
rm account_handler.go account_data.go account_codex_import.go account_today_stats_cache.go
rm channel_handler.go channel_monitor_handler.go channel_monitor_template_handler.go
rm proxy_handler.go proxy_data.go
rm openai_oauth_handler.go gemini_oauth_handler.go antigravity_oauth_handler.go
rm error_passthrough_handler.go tls_fingerprint_profile_handler.go
rm ops_handler.go ops_dashboard_handler.go ops_realtime_handler.go
rm ops_alerts_handler.go ops_settings_handler.go ops_system_log_handler.go
rm ops_ws_handler.go ops_snapshot_v2_handler.go
rm scheduled_test_handler.go
```

### 2A-4 清理 endpoint.go 中的 AI 路由注册

`internal/handler/endpoint.go` 不能整个删除（它还有通用路由逻辑），需要手动删除 AI 相关的路由注册代码。

删除以下路由注册块：
- `/v1/messages`、`/v1/chat/completions`、`/v1/responses`
- `/v1/images/*`、`/v1beta/*`
- `/antigravity/*`、`/sora/*`
- `/api/v1/admin/accounts`（AI 账号）
- `/api/v1/admin/groups`（AI 分组，注意保留订阅分组路由）
- `/api/v1/admin/channels`、`/api/v1/admin/proxies`
- `/api/v1/admin/ops/*`
- `/api/v1/available-channels`

### 2A-5 清理 wire.go 中的 AI Handler 注入

`backend/cmd/server/wire.go` 中删除已删除 Handler 的 Provider 注入。

### 2A-6 验证

```bash
cd backend
go build ./...
```

**预期结果：** 编译通过。如果有 undefined 错误，说明还有引用未清理，逐一修复。

---

## Phase 2B：删除纯 AI Service（约 150 个文件）

**目标：** 删除 OpenAI、Gemini、Antigravity、Bedrock、Claude 相关的所有 Service 文件。

**为什么这批安全：** 这些文件只被已删除的 Handler 引用，Handler 删完后它们就是孤立代码。

### 2B-1 删除 OpenAI Service（约 50 个文件）

```bash
cd backend/internal/service
rm openai_gateway_service.go
rm openai_gateway_chat_completions.go openai_gateway_chat_completions_raw.go
rm openai_gateway_messages.go openai_gateway_responses_chat_fallback.go
rm openai_images.go openai_images_responses.go
rm openai_token_provider.go openai_oauth_service.go openai_privacy_service.go
rm openai_account_scheduler.go openai_account_runtime_block_fastpath.go
rm openai_client_transport.go openai_client_restriction_detector.go
rm openai_compat_model.go openai_compat_prompt_cache_key.go
rm openai_model_mapping.go openai_model_alias.go
rm openai_codex_transform.go openai_codex_instructions_template.go
rm openai_compact_probe.go openai_endpoint_url.go
rm openai_content_session_seed.go openai_previous_response_id.go
rm openai_silent_refusal.go openai_sse_data.go openai_sticky_compat.go
rm openai_tool_continuation.go openai_tool_corrector.go
rm openai_messages_bridge.go openai_messages_continuation.go
rm openai_messages_digest_session.go openai_messages_dispatch.go
rm openai_messages_replay_guard.go openai_messages_todo_guard.go
rm openai_ws_client.go openai_ws_forwarder.go openai_ws_pool.go
rm openai_ws_protocol_resolver.go openai_ws_state_store.go
rm openai_ws_v2_passthrough_adapter.go
rm openai_403_counter.go openai_apikey_responses_probe.go
# 删除 openai_ws_v2 子目录
rm -rf openai_ws_v2/
```

### 2B-2 删除 Anthropic/Claude Service

```bash
rm anthropic_session.go claude_token_provider.go claude_code_validator.go
```

### 2B-3 删除 Gemini Service（约 18 个文件）

```bash
rm gemini_session.go gemini_token_provider.go gemini_token_refresher.go gemini_token_cache.go
rm gemini_oauth_service.go gemini_oauth.go gemini_quota.go
rm gemini_chat_completions_compat_service.go gemini_messages_compat_service.go
rm gemini_native_signature_cleaner.go geminicli_codeassist.go
```

### 2B-4 删除 Antigravity Service（约 10 个文件）

```bash
rm antigravity_gateway_service.go antigravity_oauth_service.go
rm antigravity_privacy_service.go antigravity_quota_fetcher.go antigravity_quota_scope.go
rm antigravity_subscription_service.go antigravity_token_provider.go antigravity_token_refresher.go
rm antigravity_credits_overages.go antigravity_internal500_penalty.go
```

### 2B-5 删除 Bedrock Service

```bash
rm bedrock_request.go bedrock_signer.go bedrock_stream.go
```

### 2B-6 删除 AI 网关核心 Service

```bash
rm gateway_service.go gateway_request.go
rm gateway_billing_block.go gateway_billing_header.go
rm gateway_forward_as_chat_completions.go gateway_forward_as_responses.go
rm gateway_messages_cache.go gateway_tool_rewrite.go gateway_websearch_emulation.go
```

### 2B-7 清理 wire.go

删除 wire.go 中所有已删除 Service 的 Provider 注入。

### 2B-8 验证

```bash
cd backend && go build ./...
```

---

## Phase 2C：删除 Ops、Channel、Scheduler、AI 账号（约 120 个文件）

**目标：** 删除运维监控、渠道管理、AI 账号调度等模块。

### 2C-1 删除 Ops 监控（约 30 个文件）

```bash
cd backend/internal/service
rm ops_service.go ops_aggregation_service.go ops_alert_evaluator_service.go ops_alerts.go
rm ops_cleanup_service.go ops_cleanup_executor.go
rm ops_dashboard.go ops_dashboard_models.go
rm ops_errors.go ops_health_score.go ops_histograms.go
rm ops_log_runtime.go ops_metrics_collector.go ops_models.go
rm ops_openai_token_stats.go ops_openai_token_stats_models.go
rm ops_port.go ops_query_mode.go
rm ops_realtime.go ops_realtime_models.go ops_realtime_traffic.go ops_realtime_traffic_models.go
rm ops_request_details.go ops_scheduled_report_service.go
rm ops_settings.go ops_settings_models.go
rm ops_system_log_service.go ops_system_log_sink.go
rm ops_trend_models.go ops_trends.go ops_upstream_context.go
rm ops_account_availability.go ops_advisory_lock.go ops_concurrency.go
rm ops_window_stats.go
```

### 2C-2 删除 Channel 管理（约 15 个文件）

```bash
rm channel.go channel_service.go channel_available.go
rm channel_monitor_service.go channel_monitor_checker.go channel_monitor_runner.go
rm channel_monitor_aggregator.go channel_monitor_challenge.go channel_monitor_ssrf.go
rm channel_monitor_template_service.go channel_monitor_template_types.go
rm channel_monitor_types.go channel_monitor_validate.go channel_monitor_const.go
```

### 2C-3 删除 AI 代理管理

```bash
rm proxy.go proxy_service.go proxy_latency_cache.go
rm tls_fingerprint_profile_service.go
```

### 2C-4 删除 AI 账号管理（约 10 个文件）

```bash
rm account.go account_service.go account_group.go
rm account_credentials_persistence.go account_credentials_redact.go
rm account_expiry_service.go account_usage_service.go
rm account_test_service.go account_stats_pricing.go
```

### 2C-5 删除 AI 调度器

```bash
rm scheduler_cache.go scheduler_events.go scheduler_outbox.go scheduler_snapshot_service.go
rm timing_wheel_service.go
```

### 2C-6 删除 AI 限速/并发

```bash
rm ratelimit_service.go concurrency_service.go model_rate_limit.go session_limit_cache.go
```

### 2C-7 删除 AI 幂等/错误透传

```bash
rm idempotency.go idempotency_observability.go idempotency_cleanup_service.go
rm error_passthrough_service.go error_passthrough_runtime.go
```

### 2C-8 删除 AI Token 刷新

```bash
rm token_refresher.go token_refresh_service.go token_cache_invalidator.go token_cache_key.go
rm refresh_token_cache.go refresh_policy.go
```

### 2C-9 删除 AI OAuth（账号级）

```bash
rm oauth_service.go oauth_refresh_api.go
```

### 2C-10 删除 AI 用量记录和图像计费

```bash
rm usage_record_worker_pool.go usage_billing.go usage_log_create_result.go
rm image_billing_multiplier.go image_billing_size.go
rm image_generation_intent.go image_output_accounting.go
rm codex_image_generation_bridge.go
```

### 2C-11 删除其他 AI 专用文件

```bash
rm crs_sync_service.go
rm scheduled_test_runner_service.go scheduled_test_service.go scheduled_test_port.go
rm websearch_config.go upstream_models.go upstream_response_limit.go
rm sse_scanner_buffer_pool.go http_upstream_port.go
rm digest_session_store.go internal500_counter.go
rm rpm_cache.go temp_unsched.go
rm vertex_service_account.go billing_cache_port.go
```

### 2C-12 清理 wire.go 和 config.go

- `wire.go`：删除所有已删除 Service 的 Provider 注入
- `internal/config/config.go`：删除 `GatewayConfig`、`OpsConfig`、`TokenRefreshConfig` 和账号级 provider 调度/冷却等 AI-only 配置；保留仍被通用平台使用的 `APIKeyAuthCacheConfig`、`SubscriptionCacheConfig`、`IdempotencyConfig`、`ConcurrencyConfig` 等基础配置

### 2C-13 验证

```bash
cd backend && go build ./...
```

---

## Phase 2D：删除 AI Ent Schema，重新生成 ORM

**目标：** 从数据库层面移除 AI 相关表定义。

> ⚠️ **注意：** 删除 Schema 后必须重新生成 Ent，生成的文件会有大量变化，这是正常的。
> 不要手动编辑生成文件，只编辑 `ent/schema/` 下的源文件。

### 2D-1 删除 AI Schema 文件

```bash
cd backend/ent/schema
rm account.go account_group.go
rm group.go          # AI 账号分组（不是订阅分组 group_service.go）
rm proxy.go
rm channel_monitor.go channel_monitor_history.go
rm channel_monitor_daily_rollup.go channel_monitor_request_template.go
rm error_passthrough_rule.go
rm tls_fingerprint_profile.go
rm idempotency_record.go
```

### 2D-2 重新生成 Ent ORM

```bash
cd backend
go generate ./ent
```

**预期结果：** `ent/` 目录下的生成文件会更新，删除了对应表的 CRUD 代码。

### 2D-3 重新生成 Wire

```bash
go generate ./cmd/server
```

### 2D-4 验证

```bash
go build ./...
```

如果有编译错误，说明还有代码引用了已删除的 Ent 实体，逐一修复。

---

## Phase 2E：手术式修改混用文件

**目标：** 处理同时服务于 AI 网关和通用平台的文件，保留平台部分，删除 AI 部分。

> ⚠️ **这是风险最高的批次，每修改一个文件就编译一次验证。**

### 2E-1 billing_service.go

**问题：** 同时包含 AI Token 计费（按 Token 数计算成本）和订阅套餐定价（按套餐价格）。

**操作：**
1. 读取文件，识别哪些函数是 AI Token 计费（函数名含 token、model、cost_per_token 等）
2. 删除 AI Token 计费相关函数
3. 保留订阅套餐定价相关函数
4. 编译验证

### 2E-2 subscription_service.go

**问题：** 订阅生命周期管理（保留）+ AI 配额检查（删除，如每日/每周 Token 用量检查）。

**操作：**
1. 搜索 `token`、`quota`、`daily_usage`、`weekly_usage` 等关键词
2. 删除 AI 配额检查相关逻辑
3. 保留订阅激活、过期、续费等生命周期逻辑
4. 编译验证

### 2E-3 usage_service.go 和 usage_log.go（service）

**问题：** 当前记录 AI Token 用量，后续改为社交操作统计。

**操作（Phase 2E 阶段）：**
1. 暂时保留文件结构，只删除 AI 专用的字段引用（如 `input_tokens`、`model_name`、`billing_rate`）
2. 将函数签名改为接受通用参数
3. Phase 3 再填充社交操作的具体逻辑

### 2E-4 pricing_service.go

**问题：** 订阅套餐定价（保留）+ AI 模型定价（删除）。

**操作：**
1. 删除模型定价相关函数（`GetModelPrice`、`ResolveModelPricing` 等）
2. 保留套餐定价相关函数
3. 编译验证

### 2E-5 最终验证

```bash
cd backend
go build ./...
go test -tags=unit ./internal/service/... 2>&1 | grep -E "FAIL|ok"
```

---

## Phase 2 完成检查清单

完成 Phase 2 后，验证以下内容：

```bash
# 1. 编译通过
cd backend && go build ./...

# 2. 不再有 AI 路由
curl http://localhost:8080/v1/messages        # 应返回 404
curl http://localhost:8080/v1/chat/completions # 应返回 404
curl http://localhost:8080/health              # 应返回 {"status":"ok"}

# 3. 平台功能正常
curl http://localhost:8080/api/v1/auth/login   # 应返回 401（未登录）
curl http://localhost:8080/api/v1/payment/config # 应返回支付配置

# 4. 重新构建 Docker 镜像
cd deploy && docker compose -f docker-compose.dev.yml build [参数见 PROJECT_GUIDE.md]
docker compose -f docker-compose.dev.yml up -d --no-build
```

---

## Phase 3A：对齐社交账号 Schema 和基础 API

**前提：** Phase 2 全部完成，`go build` 通过。

### 3A-0 先锁定当前业务目标

Phase 3A 开始前先按 `PROJECT_GUIDE.md` 的权威规则对齐，不参考外部项目、不新增平行业务模型：

- 总账号池是唯一处理范围：非池内账号不录入、不绑定、不执行、不计费。
- 用户导入只按平台用户名匹配总账号池（如 X 的 `@xxx`，即账号 `name` 字段）；匹配前 trim，X/Twitter 允许 `xxx` 或 `@xxx` 并统一按 `@xxx` 大小写不敏感匹配；带平台则按 `platform + name` 匹配，未带平台且命中多条则拒绝为歧义匹配；未分配则绑定当前用户，已分配或匹配不到则拒绝/忽略。
- 用户可查看/导出自己名下账号完整数据，包括账号密码、邮箱密码、IP、备注；不做额外脱敏、不做额外应用层加密。
- 如果当前实现存在账号密码/邮箱密码应用层加密、JSON 隐藏、导出空密码、因加密 key 缺失导致账号无法导入/创建等逻辑，均按待对齐项处理，不能反过来修改业务目标。
- IP/代理是用户独有的执行网络出口/代理凭据，但维护入口统一收敛到管理端 `/admin/proxies`；账号可以保存默认执行代理字段，方便执行器稳定调用并减少随机代理差异；当前可短期复用 `bound_ip`，后续推荐收敛为 `default_proxy_id` 或 `default_proxy_snapshot`；同一用户自己的代理可用于自己名下多个账号，不能跨用户使用；账号回收时必须清空默认执行代理；IP 测试免费。
- 真实社交执行动作统一单价 `0.1`；优先扣订阅金额额度，不足扣钱包；成功才最终扣费，非成功不最终扣费。SocialOps 只作为社交执行消费方调用原项目订阅/钱包入口，不修改订阅套餐、余额、配额窗口、重置周期和基础扣费语义。账号导入/导出、分配/回收、IP/代理管理、设置默认执行代理不收费。
- 当前阶段不实现真实社交平台执行器，只保留可替换入口；执行器未接入时必须失败关闭，不能标记成功，不能扣费。

Phase 3A 的最小完成口径：

| 检查项 | 通过标准 |
|---|---|
| 总账号池 | `social_accounts` 表示唯一账号池；非池内账号不会被用户导入创建、绑定、执行或计费。 |
| 用户导入 | 只按规范化后的平台用户名匹配；未分配账号绑定当前用户，已分配/未匹配/歧义匹配不新增账号。 |
| 凭据可见 | 用户和管理员导出/查看目标账号数据时，账号密码、邮箱密码等完整字段不被额外脱敏、清空或应用层加密。 |
| 分配回收 | 分配只能面向未分配账号；回收清空账号归属和原用户默认执行代理/代理快照。 |
| IP/代理 | 代理是用户独有资源，同一用户可多账号使用，不能跨用户；管理入口为 `/admin/proxies`；测试和默认代理配置免费。 |
| 执行入口 | 真实执行器未接入时登录检测/关注/私信/发帖/点赞都失败关闭，不能成功、不能扣费。 |
| 计费接入 | 成功任务才按账号 + 动作计费，单价 `0.1`；订阅/钱包、预扣/确认/退回复用原项目入口，不新增平行账本。 |
| 日志幂等 | 任务日志记录价格、扣费状态/来源、代理快照和幂等键；重复请求不重复创建任务或重复扣费。 |

### 3A-1 新增或对齐 Ent Schema

如果当前仓库尚未存在以下文件，则创建；如果已经存在，则按 PROJECT_GUIDE.md 第四节对齐业务含义，不重复新增平行模型：

```bash
touch backend/ent/schema/social_account.go
touch backend/ent/schema/social_task_log.go
touch backend/ent/schema/social_ip.go
```

对齐重点：

- `social_accounts` 代表网站总账号池，`assigned_user_id` 为空表示未分配；出售/出租不设计独立状态。
- 账号密码和邮箱密码按当前字段直接保存和导出；如果当前实现存在额外应用层加密、JSON 隐藏或导出空密码，需要在 Phase 3A 修正。
- 用户导入匹配和管理员导入去重优先按规范化后的 `platform + name`；同一平台同一用户名不创建第二条。
- `social_task_logs` 至少能记录动作、账号、用户、状态、结果、单价、已扣金额、扣费状态、扣费来源、IP/代理快照和幂等标识；不要新建一套独立复杂账本。
- `social_ips` 只表示用户独有的 IP/代理资源，是当前内部执行代理存储，不是用户端公开 API 目标；如果已有 `bound_social_account_id`，不能作为账号所有权状态。账号 `bound_ip` 或后续 `default_proxy` 字段可以作为该账号的默认执行代理/代理快照，供执行器优先调用；同一用户自己的代理可用于自己名下多个账号，不能跨用户使用；账号回收时必须清空该字段。
- `edit_ip`、`set_default_proxy`、`change_proxy` 等代理配置动作不进入真实社交执行计费；如果当前实现把它们放入任务执行器，应改为免费配置接口或默认执行代理设置接口。

编写 Schema 后重新生成：

```bash
cd backend
go generate ./ent
go generate ./cmd/server
go build ./...
```

### 3A-2 必要时新增数据库迁移

```bash
# 如果现有迁移尚未覆盖需要的 schema 差异，在 backend/migrations/ 下创建新迁移文件
# 命名格式：YYYYMMDD_description.sql
touch backend/migrations/20260601_add_social_tables.sql
```

### 3A-3 新增或对齐 Service 文件

```bash
touch backend/internal/service/social_account_service.go
touch backend/internal/service/social_task_service.go
touch backend/internal/service/social_ip_service.go
```

### 3A-4 新增或对齐 Handler 文件

```bash
touch backend/internal/handler/social_account_handler.go
touch backend/internal/handler/admin/social_account_admin_handler.go
```

如果当前仓库已经存在 `backend/internal/handler/admin/social_account_handler.go`，优先在现有文件内对齐，不要额外新增同职责文件。

### 3A-5 注册路由

在 `internal/handler/endpoint.go` 中添加：

```go
// 管理端社交账号路由
adminGroup.GET("/social-accounts", ...)
adminGroup.POST("/social-accounts/import", ...)
adminGroup.POST("/social-accounts/register", ...)
adminGroup.POST("/social-accounts/assign", ...)
adminGroup.POST("/social-accounts/reclaim", ...)
adminGroup.GET("/proxies", ...)
adminGroup.POST("/proxies", ...)
adminGroup.PUT("/proxies/:id", ...)
adminGroup.DELETE("/proxies/:id", ...)
adminGroup.POST("/proxies/:id/test", ...)

// 用户端社交账号路由
userGroup.GET("/social-accounts", ...)
userGroup.POST("/social-accounts/import", ...)
userGroup.GET("/social-accounts/export", ...)
userGroup.POST("/social-accounts/tasks", ...)
userGroup.PUT("/social-accounts/:id/default-proxy", ...) // 如保留，必须校验账号和代理同属当前用户
```

业务约束按 PROJECT_GUIDE.md 当前目标执行：
- 用户导入只按平台用户名匹配网站总账号池，例如 X 的 `@xxx`，对应账号 `name` 字段；匹配前 trim，X/Twitter 允许 `xxx` 或 `@xxx` 并统一按 `@xxx` 大小写不敏感匹配；带平台则按 `platform + name` 匹配，未带平台且命中多条则拒绝为歧义匹配；不新增外部账号。
- 匹配到未分配账号自动绑定当前用户；匹配到已分配账号拒绝；匹配不到不处理。
- 用户导出返回自己名下账号完整数据，不额外脱敏，不记录用户导出审计日志；默认 CSV，至少包含 platform、name、account_id、password、phone、email、email_password、bound_ip/default_proxy、account_status、task_status、source、remark、created_at、updated_at。
- IP/代理属于用户独有资源，是执行时可选资源，但维护入口统一在管理端 `/admin/proxies`，不再暴露用户端 `/api/v1/social-ips`。管理员可以代用户设置已分配账号的默认执行代理，但代理也必须属于该账号当前所属用户；未分配账号不保留默认执行代理。执行时优先使用请求显式选择的代理，其次使用账号默认执行代理；同一用户自己的代理可用于自己名下多个账号，不能跨用户使用；账号回收时必须清空默认执行代理。IP 测试免费。
- 任务提交必须先校验账号归属、账号状态、参数、IP/代理归属、订阅/钱包额度，再创建任务日志；当前最小状态准入是只有 `available` 允许执行。
- 真实社交执行动作统一单价 0.1，扣费/预扣/退款通过原项目订阅和钱包入口完成，成功才最终扣费，非成功退回/不扣；不修改原项目订阅套餐、钱包余额、配额窗口、重置周期和基础扣费语义；账号导入/导出、分配/回收、管理端 IP/代理管理、设置默认执行代理不收费。
- `edit_ip`、`set_default_proxy`、`change_proxy` 等代理配置动作免费，不作为社交执行动作计费。
- 执行器未接入、账号状态失败、IP 不可用、参数错误、权限失败、执行异常等都属于非成功结果，不最终扣费。
- 批量任务按账号 + 动作计费，成功几个扣几次，失败账号不扣。
- 同一任务/请求需要幂等，避免重复提交或重试造成重复扣费；请求需要支持 `client_request_id` 或等价幂等键，建议按 `user_id + client_request_id + account_id + action` 识别重复任务。
- 管理员注册路由是真实社交平台注册入口；真实注册器未接入时必须返回 not configured/失败关闭，不创建账号，注册失败不入池。

### 3A-6 验证

```bash
cd backend && go build ./...
# 测试新 API
curl -X GET http://localhost:8080/api/v1/social-accounts -H "Authorization: Bearer <token>"
```

---

## Phase 3B：前端接入真实 API

**前提：** Phase 3A 完成，后端 API 可用。

### 3B-1 保留兼容存根，新增专用社交 API

以下文件是保留视图的兼容存根，禁止改为真实社交账号调用：

```
frontend/src/api/admin/accounts.ts
frontend/src/api/admin/groups.ts
```

`frontend/src/api/admin/proxies.ts` 已改为 SocialOps 管理端执行代理 API；不得再当作 AI 网关代理存根，也不得接入 provider/channel/model 语义。

新增 API 文件：

```
frontend/src/api/socialAccounts.ts       — 用户端社交账号 API
frontend/src/api/admin/socialAccounts.ts — 管理端社交账号 API
frontend/src/api/admin/proxies.ts         — 管理端执行代理 API
```

### 3B-2 替换 mock 数据

逐页面替换（每替换一个页面就测试一次）：

1. `admin/AccountOnboardingView.vue` — 作为备份式“账号管理”入口，接入 `GET/POST/PUT/DELETE /api/v1/admin/social-accounts`、import/register/export；页面可保留执行日志内容，但真实执行器未接入时必须失败关闭且不扣费。
2. `admin/TotalAccountsView.vue` — 作为备份式“总账号池”入口，接入 `GET/PUT/DELETE /api/v1/admin/social-accounts`、assign/reclaim/default-proxy/export，承担分配、回收、默认执行代理和批量清理。
3. `admin/ProxiesView.vue` — 接入 `GET/POST/PUT/DELETE /api/v1/admin/proxies` 和免费 `POST /api/v1/admin/proxies/:id/test`
4. 后续若需要用户端查看/导出/执行账号，只能保留一个唯一入口接入 `GET /api/v1/social-accounts`、import/export/tasks；不得恢复 `AccountManagementView.vue` 与 `AccountFunctionsView.vue` 两套任务 UI
5. 不恢复 `user/IpManagementView.vue` 或用户端 `/api/v1/social-ips`

### 3B-3 验证

```bash
cd frontend && pnpm run build
# 启动开发服务器测试各页面
pnpm run dev
```

---

## Phase 3C：调整现有页面

**前提：** Phase 3B 完成。

### 3C-1 admin/DashboardView.vue

替换 AI 统计卡片为社交平台统计：
- 删除：Token 用量、RPM、模型分布图
- 新增：社交账号总数/活跃数、今日任务执行次数、活跃用户数
- 保留：订阅收入、新增用户数

### 3C-2 admin/UsageView.vue

替换 AI 用量字段为社交操作统计：
- 删除列：model_name、input_tokens、output_tokens、billing_rate、rate_multiplier
- 新增列：action（操作类型）、account（社交账号）、status（执行状态）、executed_at

### 3C-3 admin/SettingsView.vue

删除 gateway 标签页（包含 Rectifier、Beta Policy、OpenAI Fast Policy 等 AI 配置）。

### 3C-4 user/UsageView.vue

替换 Token 用量为社交操作次数统计。

### 3C-5 前端遗留清理

```bash
cd frontend/src/views/admin
rm groupsMessagesDispatch.ts groupsSupportedModelScopes.ts
rm __tests__/groupsMessagesDispatch.spec.ts __tests__/groupsSupportedModelScopes.spec.ts
```

### 3C-6 最终验证

```bash
cd frontend && pnpm run build
cd deploy && docker compose -f docker-compose.dev.yml build [参数]
docker compose -f docker-compose.dev.yml up -d --no-build
# 全流程测试：管理员导入/注册账号→分配账号或用户导入匹配→管理员维护用户执行代理→执行任务→成功扣费→查看记录
```

---

## 安全操作规范

### 每次操作前

1. 确认当前 `go build ./...` 通过（后端）
2. 确认当前 `pnpm run build` 通过（前端）
3. 记录当前状态（可以用 `git stash` 或备份目录）

### 每次操作后

1. 立即运行 `go build ./...` 验证
2. 如果编译失败，**不要继续删除**，先修复错误
3. 修复方式：查找 undefined 符号的引用位置，删除引用或补充存根

### 遇到不确定的文件

1. 先用 `grep -rn "文件名中的函数名" backend/` 查找所有引用
2. 如果引用都在已删除的文件中，可以安全删除
3. 如果引用在保留的文件中，需要先处理引用再删除

### 绝对不做的事

- 不要用 `rm -rf internal/service/` 一次性删除整个目录
- 不要在编译失败的状态下继续删除文件
- 不要删除任何 `_test.go` 文件（除非对应的源文件已删除）
- 不要修改 `wire_gen.go`（只修改 `wire.go`，然后 `go generate`）
- 不要修改 `ent/` 下的生成文件（只修改 `ent/schema/`，然后 `go generate ./ent`）

---

## 阶段验收口径

本文档不记录“已完成/待执行”的历史进度。阶段是否完成，以当前代码、构建结果和测试结果为准。

- Phase 2 验收：AI 网关入口、AI 渠道、AI 账号池、AI Token 计费和 AI OAuth 相关代码已移除或隔离；用户、支付、订阅、钱包、兑换码、推广、公告、管理员后台等通用平台能力仍可编译运行。
- Phase 3A 验收：总账号池、用户账号归属、管理端执行代理、任务日志和执行入口的后端框架存在；非池内账号不处理；用户导入只按平台用户名匹配；真实执行器未接入时失败关闭；订阅/钱包扣费只通过原项目入口接入。
- Phase 3B 验收：管理端“社交账号”保留备份式双入口（`/admin/accounts` 账号管理、`/admin/total-accounts` 总账号池）并接入真实 SocialOps API 或明确的失败关闭接口；管理端执行代理入口在 `/admin/proxies`；不保留用户端账号管理/账号功能/IP 管理重复入口；保留页面不再展示 AI Token/模型/渠道语义。
- Phase 3C 验收：用量、后台统计和文案只表达社交执行动作；成功任务按账号 + 动作计费，失败和未执行任务不最终扣费。
