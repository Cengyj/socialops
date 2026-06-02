# SocialOps 改造状态

**最后更新：2026-06-02**

## Ultragoal 进度

| 目标 | 状态 | 证据 |
|---|---|---|
| G001 审计矩阵 | 完成 | `.omx/audit/socialops-g001-audit-matrix.md` |
| G002 清 AI 网关残留 | 完成 | `.omx/ultragoal/goals.json` G002 checkpoint |
| G003 恢复基础框架 | 完成 | Ent/Wire 生成、后端测试、前端 typecheck checkpoint |
| G004 后端框架 | 完成 | SocialOps 后端账户池/IP/任务/计费框架与测试 checkpoint |
| G005 前端框架 | 完成 | SocialOps 前端 API/页面/i18n/typecheck checkpoint |
| G006 文档同步 | 完成 | `PROJECT_GUIDE.md`、`PROJECT_GOAL.md`、`CLEANUP_PLAN.md`、`PROJECT_STATUS.md`、README/docs/deploy 文案同步与残留扫描 |
| G007 验证并修复失败 | 完成 | 后端 generate/test、前端 install/typecheck/build、根 build、Compose build/up/restart/health、HTTP smoke |
| G008 敌意验证 | 完成 | 社交导入/执行/扣费/IP/权限/旧 AI 路由敌意场景测试与 smoke 通过 |

Codex Goal 本轮完成条件已满足：重复用户端入口、重复管理端账号池和错位 IP/代理入口已清理，验证、容器更新、HTTP smoke 和浏览器 QA 已通过；同轮最终报告将标记 Goal complete。

## 已完成能力

- 社交账号总池 schema、迁移、服务、用户/管理员 handler 和路由已接入。
- 用户导入只匹配总池账号；用户导出返回账号完整字段。
- 管理员可导入、创建、分配、回收账号；注册入口在未接入真实注册器时失败关闭。
- 管理端 `/admin/proxies` 已接入用户独有执行代理 CRUD 和免费连通性测试；用户端 `/api/v1/social-ips` 不再作为公开 API。
- 社交任务支持估价、提交、幂等键、失败关闭日志、代理快照和成功后扣费挂钩。
- 计费框架按统一单价 `0.1` 估算，订阅额度优先、不足钱包；执行器未接入时不扣费。
- 前端已清理用户端账号管理、账号功能、IP 管理三套重复入口；管理端“社交账号”按备份版 UI 恢复为 `/admin/accounts` 账号管理和 `/admin/total-accounts` 总账号池两个入口，执行代理入口收敛到 `/admin/proxies`。

## 最终质量门

- ai-slop-cleaner no-op 报告：`.omx/reports/socialops-g008-ai-slop-cleaner.md`。
- UltraQA/等价敌意 QA 报告：`.omx/reports/socialops-g008-ultraqa.md`。
- 最终质量门 JSON：`.omx/ultragoal/quality-gate-g008.json`。
- 独立 code-review 结论：code-reviewer `APPROVE`；architect 初始 `WATCH`（`bound_social_account_id` 兼容字段可能误导），已通过 schema 注释、service/API DTO 收敛和同代理多账号回归测试缓解。

## 本轮补充证据

- 已补清后端 AI provider 配置残留：`token_refresh`、`RateLimitConfig`、`rate_limit.overload_cooldown_minutes`、`oauth_401_cooldown_minutes` 和 deploy 示例中的旧 529/定价代理文案已移除。
- 残留扫描：`rg "RateLimit\\.|RateLimitConfig|rate_limit\\.|overload_cooldown|oauth_401|TokenRefresh|token_refresh|pricing data|529过载|upstream returns 529" backend/internal backend/cmd deploy/config.example.yaml docs` 无命中。
- 针对性验证：`GOMODCACHE=/tmp/socialops-gomodcache GOCACHE=/tmp/socialops-gocache GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn go test ./internal/config ./internal/pkg/logger ./internal/service` 通过。
- G008 新增回归测试：`backend/internal/handler/social_account_handler_test.go` 覆盖执行器未接入失败关闭不扣费、异常账号拒绝且不写日志、余额不足拒绝且不写日志、IP 测试免费、用户只能看到自己的账号和代理；`backend/internal/server/middleware/admin_auth_test.go` 覆盖普通用户 JWT 访问管理端被 403 拒绝。
- G008 既有服务测试：`backend/internal/service/socialops_fail_closed_test.go` 覆盖非总池导入拒绝、已绑定导入拒绝、未分配账号绑定、歧义匹配拒绝、分配防覆盖、回收清空默认代理、订阅优先与钱包兜底、余额不足拒绝、执行器未接入不标成功、代理安全校验。
- 验证命令：`go test -tags=unit ./internal/handler -run 'TestSocialAccountHandler'`、`go test -tags=unit ./internal/server/middleware -run 'TestAdminAuthJWTValidatesTokenVersion'`、`go test -tags=unit ./internal/service -run 'TestSocial(Account|Task|Billing|IP|AdminServiceSkeleton|SubscriptionService)'`、`go test -tags=unit ./...`、`go test ./...` 均通过。
- 构建命令：`make generate`、`pnpm run typecheck`、`pnpm run build`、`GOMODCACHE=$PWD/backend/.cache/gomod GOCACHE=$PWD/backend/.cache/gocache GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn make build` 通过；默认 Go module cache 无写权限时，根构建曾失败于 `/home/ceng/go/pkg/mod/cache` permission denied，使用项目本地 cache 后通过。
- 容器与 HTTP smoke：`socialops-dev`、`socialops-postgres-dev`、`socialops-redis-dev` 均 healthy；`GET /health` 200，`GET /api/v1/settings/public` 200，`POST /v1/chat/completions` 404，`GET /api/v1/admin/channels` 404，未认证 `GET /api/v1/social-accounts` 401。
- 最终补充验证：`go vet ./internal/service ./internal/web ./internal/handler/admin ./internal/handler` 通过；fresh compose rebuild/up 后 `/health`、`/`、`/login`、用户/管理 API 未认证拒绝、旧 AI 路由 404 且不返回 SPA HTML 均通过。
- 重复入口收尾验证：`rg "account-management|account-functions|ip-management|AccountManagement|AccountFunctions|IpManagement|listMyIPs|createIP|updateIP|deleteIP|testIP|testAllIPs|userIpManagement|accountFunctions" frontend/src frontend/AGENTS.md` 无用户端重复入口命中；`/admin/total-accounts` 与 `TotalAccountsView.vue` 为当前保留的备份式总账号池入口。
- 最新 UI 修正：`/admin/accounts` 已恢复备份式账号管理内容，包括执行动作栏、最近执行日志、账号详情弹窗、上传入库弹窗和终端日志；上传弹窗显示文件名/大小/类型/就绪状态，确认后调用当前 SocialOps 导入接口；未接入真实执行器的动作在前端日志中失败关闭，不标记成功、不扣费。
- `/social-ips` 收尾扫描：`rg "/social-ips|ListMyIPs|CreateIP|UpdateIP|DeleteIP|TestAllIPs" backend/internal backend/cmd/server frontend/src AGENTS.md PROJECT_GUIDE.md PROJECT_GOAL.md CLEANUP_PLAN.md README.md PROJECT_STATUS.md docs` 仅剩文档说明“用户端 API 已移除/不恢复”。
- AI 关键入口扫描：关键 router/sidebar/admin proxies 文件中 `channels|models|provider|OpenAI|Anthropic|Gemini|Ollama|DeepSeek|virtual key|virtualKey|pricing|token` 无命中。
- 最新代码生成与验证：`cd backend && make generate` 通过；`go test -tags=unit ./internal/service ./internal/handler/admin ./ent/schema` 通过；`PATH=$PWD/backend/.cache/bin:$PATH GOMODCACHE=$PWD/backend/.cache/gomod GOCACHE=$PWD/backend/.cache/gocache GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn make test` 通过；`GOMODCACHE=$PWD/backend/.cache/gomod GOCACHE=$PWD/backend/.cache/gocache GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn make build` 通过。
- 最新容器更新与 smoke：`docker compose -p socialops-dev -f deploy/docker-compose.dev.yml build ...` 与 `up -d --no-build` 通过；`socialops-dev` healthy；`GET /health`、`/`、`/login`、`/api/v1/settings/public` 返回 200；未认证 `/api/v1/social-accounts`、`/api/v1/admin/proxies` 返回 401；`/api/v1/social-ips`、`/api/v1/admin/channels`、`POST /v1/chat/completions` 返回 404。
- 最新浏览器 QA：Playwright Firefox headless 使用本地临时浏览器 cache 通过；桌面和移动 `/login` 均 200、title 为 `Login - SocialOps`、无 console/network 错误；未登录访问 `/admin/accounts` 重定向到 `/login?redirect=/admin/accounts`；截图和 JSON 证据在 `.omx/reports/browser-qa/`。

## 待接入点

- 真实社交平台注册器：接入前管理员注册必须失败关闭且不入库。
- 真实社交任务执行器：接入前所有真实动作不得标记 success，不得最终扣费。
- 更细的执行器动作参数、平台差异和结果码映射。
- 可选：将当前 `bound_ip` 代理快照字段后续收敛为显式 `default_proxy_id/default_proxy_snapshot`；`bound_social_account_id` 仅为兼容字段，不参与默认代理授权和 API DTO。

## 剩余风险

- 真实社交平台注册器和执行器仍未接入；当前实现按目标失败关闭，不伪造成功，不最终扣费。
- Playwright MCP 默认浏览器路径仍不可用，但已通过本地 Playwright Firefox headless 脚本完成浏览器 QA；后续若要用 MCP 面板交互，可继续修复全局 npm cache/浏览器路径。
- 部署环境存在本地运行数据和密钥文件，验证时不得破坏 `deploy/data`、`deploy/postgres_data`、`deploy/redis_data`。
