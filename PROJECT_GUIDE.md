# SocialOps 项目指南

最后复核时间：2026-06-04

本文档是当前 SocialOps 项目后续优化、审查、重构时的唯一项目级参考。当前项目路径为 `D:\Downloads\socialops-main`，用户提供的原项目路径为 `D:\Downloads\sub2api-0.1.133`。

## 1. 项目定位

当前项目是从 `sub2api-0.1.133` 魔改而来。原项目的 AI 网关方向已经被移除或绕开，当前保留的是一套 SaaS 管理平台底座，包括用户、认证、设置、支付、订阅、兑换码、优惠码、公告、分销、公开配置注入、管理后台、部署脚本等基础能力。

当前 SocialOps 的产品方向是“网站总账号池 + 社交账号执行计费平台”：

- 管理员维护社交账号总池。
- 用户从总池导入或绑定账号到自己的工作台。
- 租售账号字段是业务交付内容，应按权限完整存储、返回、导出和展示。
- 用户配置执行代理，提交社交任务。
- 任务异步执行。
- 任务执行成功才结算扣费，失败不扣费。
- 当前真实接入的执行平台是 Twitter / X。
- 通用使用量看板仍然保留，但 SocialOps 的核心执行记录是 `social_task_logs`。

账号字段交付原则：

- 本项目不隐藏、不脱敏、不加密租售账号字段。
- 账号名称、平台 ID、密码、手机号、邮箱、邮箱密码、认证 cookie、默认代理快照、备注等字段都属于租售账号交付数据。
- 后端接口、前端页面、导入导出、任务执行器和管理后台都应保持这些字段完整传递。
- 权限边界通过登录态、角色、账号归属、分配关系和管理权限控制，而不是通过字段省略、星号掩码、哈希、加密或只返回部分字段控制。
- 不要因为字段名包含 `password`、`cookie`、`proxy`、`credential` 等词，就按通用密钥处理逻辑自动遮蔽租售账号交付字段。
- 系统级密钥仍然与租售账号字段分开处理，例如 JWT secret、支付密钥、OAuth client secret、数据库密码、第三方平台配置密钥等。

后续优化时，不能把项目改回 AI 网关方向。仍然残留的网关、模型、通道、上游等命名，应按兼容债务处理，除非当前真实代码路径仍然依赖这些概念。

## 2. 当前架构

### 后端

后端技术栈：

- Go 模块：`github.com/Wei-Shaw/socialops`
- HTTP 框架：Gin
- ORM：Ent
- 数据库：PostgreSQL
- 缓存与会话辅助：Redis
- 依赖注入：Google Wire
- 服务入口：`backend/cmd/server/main.go`

后端关键目录：

- `backend/cmd/server`：服务入口、版本参数、setup 模式、Wire 初始化。
- `backend/internal/config`：配置结构、默认值、校验、OAuth 与安全配置。
- `backend/internal/server`：Gin Router、HTTP Server、中间件注册。
- `backend/internal/server/routes`：`/api/v1` 下的 auth、user、admin、payment 路由注册。
- `backend/internal/handler`：HTTP Handler、请求解析、响应映射。
- `backend/internal/service`：业务逻辑，包括认证、支付、订阅、账号池、代理、任务执行、计费。
- `backend/internal/repository`：Ent、SQL、Redis、仓储与基础设施。
- `backend/ent/schema`：Ent 实体 schema。
- `backend/migrations`：嵌入式 SQL 迁移，是数据库结构的权威来源。
- `backend/internal/web`：前端静态资源嵌入、公开设置注入。

运行模式：

- `standard`：完整 SaaS 模式，启用支付、订阅、余额、计费等能力。
- `simple`：简易模式，关闭或跳过部分 SaaS 计费/订阅能力，前后端都有访问限制。

启动流程：

1. `main.go` 初始化 bootstrap 日志。
2. `--setup` 参数进入 CLI setup。
3. 首次运行需要配置时，启动 setup wizard。
4. 正常模式下通过 `config.LoadForBootstrap` 加载配置。
5. Wire 初始化 repository、service、handler、router、HTTP server。
6. Ent 连接 PostgreSQL，执行嵌入式迁移，补齐系统密钥，校验完整配置。
7. 启动 HTTP 服务并等待退出信号。

### 前端

前端技术栈：

- Vue 3
- TypeScript
- Vite
- Pinia
- Vue Router
- Tailwind CSS
- Vitest

前端关键目录：

- `frontend/src/main.ts`：前端启动入口。
- `frontend/src/router/index.ts`：路由表与路由守卫。
- `frontend/src/api`：基于 `apiClient` 的类型化 API 封装。
- `frontend/src/stores`：认证、应用设置、订阅、支付/管理设置等状态。
- `frontend/src/components/common`：表格、弹窗、输入、分页、Toast 等通用组件。
- `frontend/src/components/layout`：应用布局、顶部栏、侧边栏。
- `frontend/src/views/accounts`：统一账号工作台。
- `frontend/src/views/admin`：管理后台页面。
- `frontend/src/views/user`：用户仪表盘、使用记录、资料、订阅、购买、订单等页面。
- `frontend/src/i18n`：中英文语言包。

构建行为：

- `frontend/vite.config.ts` 将生产构建输出到 `backend/internal/web/dist`。
- 开发模式将 `/api`、`/v1`、`/setup` 代理到 `VITE_DEV_PROXY_TARGET` 或 `http://localhost:8080`。
- 开发和生产都会尽量向 `window.__APP_CONFIG__` 注入公开配置，减少页面闪烁。

## 3. 后端路由地图

主 API 均位于 `/api/v1` 下。

公开认证与设置：

- `/auth/register`
- `/auth/login`
- `/auth/login/2fa`
- `/auth/refresh`
- `/auth/logout`
- `/auth/send-verify-code`
- 找回密码、重置密码、优惠码校验、邀请码校验
- LinuxDo、WeChat、OIDC、DingTalk、GitHub、Google OAuth 相关路由
- `/settings/public`

已登录用户路由：

- `/user/profile`
- `/user`
- `/user/password`
- `/user/account-bindings/*`
- `/user/auth-identities/bind/start`
- `/user/notify-email/*`
- `/user/totp/*`
- `/keys`
- `/usage`
- `/announcements`
- `/redeem`
- `/subscriptions`
- `/accounts`
- `/accounts/import`
- `/accounts/batch-import`
- `/accounts/batch-delete`
- `/accounts/export`
- `/accounts/:id/default-proxy`
- `/accounts/default-proxy`
- `/accounts/tasks/estimate`
- `/accounts/tasks`
- `/proxies/*`
- `/task-settings/templates/*`
- `/plans`
- `/my-plan`

管理员路由：

- `/admin/dashboard/*`
- `/admin/users/*`
- `/admin/announcements/*`
- `/admin/redeem-codes/*`
- `/admin/promo-codes/*`
- `/admin/settings/*`
- `/admin/data-management/*`
- `/admin/backups/*`
- `/admin/system/*`
- `/admin/subscriptions/*`
- `/admin/groups/*`
- `/admin/user-attributes/*`
- `/admin/api-keys/*`
- `/admin/affiliates/*`
- `/admin/accounts/*`
- `/admin/total-accounts/:id/assign`
- `/admin/total-accounts/:id/reclaim`

支付路由在 `backend/internal/server/routes/payment.go` 单独注册。

## 4. 核心数据模型

当前 Ent schema 包括：

- `User`
- `AuthIdentity`
- `AuthIdentityChannel`
- `PendingAuthSession`
- `Group`
- `UserAllowedGroup`
- `UserSubscription`
- `SubscriptionPlan`
- `APIKey`
- `UsageLog`
- `UsageCleanupTask`
- `PaymentOrder`
- `PaymentProviderInstance`
- `PaymentAuditLog`
- `RedeemCode`
- `PromoCode`
- `PromoCodeUsage`
- `Announcement`
- `AnnouncementRead`
- `Setting`
- `SecuritySecret`
- `UserAttributeDefinition`
- `UserAttributeValue`
- `SocialAccount`
- `SocialIP`
- `SocialTaskLog`

SocialOps 新增或核心表：

- `social_accounts`：总账号池与用户分配账号记录。
- `social_ips`：用户拥有的执行代理。
- `social_task_logs`：社交任务执行记录，是当前业务的核心流水。

重要数据事实：

- `social_accounts` 使用软删除。
- 总池唯一性通过 `platform_key + name_key` 归一化字段控制，并限制在 `deleted_at IS NULL`。
- 用户从工作台移出账号，不等于从总池删除账号，而是写入 `user_workbench_deleted_at`。
- 默认执行代理以快照形式保存在 `social_accounts.bound_ip`。
- `social_task_logs` 保存执行状态、扣费状态、代理快照、幂等键、结果与执行时间。
- `usage_logs` 仍保留，用于看板和计费投影；成功结算的社交任务可能写入 usage ledger。

迁移事实：

- SQL 迁移通过 `backend/migrations/migrations.go` 嵌入到二进制。
- 自定义 migration runner 会校验 SHA256 checksum。
- 已经应用过的迁移不能修改。
- 普通 `*.sql` 在事务内执行。
- `*_notx.sql` 专用于并发索引等不能放入事务的场景。

## 5. SocialOps 核心业务流程

### 总账号池

管理员可以创建、注册、导入、导出、更新、删除、分配和回收社交账号。总账号池是账号库存的唯一来源。

用户导入账号时，不会创建新账号。用户只能按平台和用户名，从已有且未分配的总池账号中绑定账号。如果这个账号之前被同一用户移出工作台，再次导入会恢复显示。

### 用户账号工作台

用户可以：

- 查看已分配账号。
- 从总池导入单个账号。
- 批量导入账号。
- 从自己的工作台移出账号。
- 导出租售账号完整字段。
- 配置账号默认执行代理。
- 估算任务费用。
- 提交社交执行任务。

租售账号字段必须完整交付。账号名称、平台 ID、密码、手机号、邮箱、邮箱密码、认证 cookie、默认代理快照、备注等字段都属于租售账号业务数据，应按接口权限完整返回、导出和展示。权限边界应通过登录态、角色、账号归属、分配关系和管理权限控制，而不是通过字段省略、星号掩码、哈希、加密或只返回部分字段来控制。

### 代理管理

`social_ips` 是用户拥有的执行代理。代理必须通过连通性测试，并且状态为 `online` 后，才能用于任务执行或设置为账号默认代理。

删除代理时，需要清理引用该代理的账号默认代理快照。

### 任务执行

当前规范化动作：

- `login_check`
- `follow`
- `post`
- `like`
- `retweet`
- `message`

说明：

- `message` 当前存在于规范化动作中，但实际执行路径是不可用或 fail-closed 状态。
- 旧动作 `tweet` 会映射为 `post`。
- 旧动作 `dm` 会映射为 `message`。

任务流程：

1. `AccountWorkbenchService` 标准化并校验请求。
2. 账号必须已分配、在用户模式下必须属于当前用户且未被用户移出工作台，并且 `account_status = available`。
3. 用户模式下一批任务只能包含同一平台账号。
4. 创建任务日志前先进行费用估算。
5. 每个待执行任务都会创建 `pending` 状态的 `social_task_logs`。
6. 任务日志可带代理快照与幂等键。
7. executor queue 接收 task log ID。
8. executor 不存在或队列已满时，任务必须被标记为失败且不扣费。
9. worker 将 `pending` 任务 claim 为 `running`。
10. 平台执行器执行真实动作。
11. 执行失败时，任务失败且不扣费。
12. 执行成功后调用计费 finalizer，在事务内完成订阅额度、钱包余额、任务状态和 usage ledger 更新。

### 计费

当前社交任务单价为 `0.1`。

计费原则是 success-only billing：

- 估算时先使用订阅额度，再计算需要钱包补足的金额。
- 实际扣费只发生在真实执行成功之后。
- 结算在事务内更新 `user_subscriptions`、用户余额、`social_task_logs` 和 `usage_logs`。
- 执行失败、平台不可用、代理不可用、认证失败、队列溢出、executor 缺失时都不能扣费。
- 计费失败不能静默吞掉，也不能伪装成成功。

订阅额度与平台相关，依赖 group 或 subscription plan 的 platform 字段。

### Twitter / X 执行器

当前真实执行器位于 `backend/internal/service/twitter_executor.go`。

该执行器会：

- 从 `social_accounts.auth_cookie` 读取平台认证头。
- 要求 OAuth token 和 token secret。
- 要求任务携带代理快照。
- 使用代理构建 HTTP client。
- 调用 Twitter / X 的接口执行登录检查、关注、点赞、发帖、转发。

优化该区域时必须保留：

- fail-closed 行为。
- 租售账号字段完整传递，不做脱敏或加密。
- 强制代理执行。
- 成功才扣费。
- 平台执行错误可以转换为业务状态，但不能删除、掩码或篡改账号交付字段。

## 6. 从原项目保留的基础能力

当前项目保留了原 sub2api 的大量 SaaS 底座能力：

- 首次运行 setup wizard。
- JWT 登录与 refresh token。
- 会话撤销。
- 邮箱/密码注册登录。
- TOTP 双因素认证。
- OAuth 登录与身份绑定。
- 管理员/普通用户权限。
- 用户管理。
- 用户组、订阅、兑换码、优惠码。
- 多支付提供商与订单。
- 公开设置、站点配置、自定义菜单、自定义页面。
- 首页、法律文档、登录协议。
- 公告与已读状态。
- 分销记录。
- 备份和数据管理相关配置或兼容接口。
- 看板与使用量投影。
- Redis 缓存与限流。
- 前端嵌入式服务。

这些基础模块后续应保守优化。除非 SocialOps 业务明确要求，否则不要重写其业务含义。

## 7. 已移除或废弃的 AI 网关方向

原项目存在 AI 网关概念，例如 channel、upstream account、model、monitoring template、Sora、image generation、model mapping 等。

当前 SocialOps 不应重新引入这些作为产品核心概念。

仍可能残留的旧名词：

- `api_key`
- `usage_logs`
- `model`
- `channel`
- `upstream`
- `subscription`
- `balance`

处理规则：

- 如果它们属于当前仍在使用的 SaaS 认证、计费、看板、兼容基础设施，则可以保留并谨慎优化。
- 如果它们表示 AI 请求转发、模型路由、上游通道、AI 监控，则通常是迁移债务。
- 如果 UI 文案仍用“模型”表示社交动作，应在前后端契约、语言包和测试同步后再改。

## 8. 当前已知问题与风险

这些是本次重新阅读项目后确认到的事实，后续审查不能跳过：

- 任务历史当前统一通过 `/usage` 与 `usage_log_repo` 从 `social_task_logs` 投影，`/accounts/tasks` 只保留提交入口。
- 前端 `DashboardView.vue` 和 `UsageView.vue` 应继续使用 `usageAPI` 读取任务历史，不要恢复 `accountWorkbenchAPI.listMyTaskLogs` 或 `GET /accounts/tasks` 死入口。
- 用户侧任务估算入口已从账号工作台移除，执行参数来自 `/task-settings` 保存模板；后续若恢复估算，必须先重新定义模板化估算契约。
- 前端部分类型仍保留旧的代理订阅转换概念，可能不再符合 SocialOps 产品方向。
- 部分注释仍引用旧阶段或已经删除的实体，例如 simple mode group seeding 注释。
- `risk-control`、部分 data-management、backup 路由存在 stub 或兼容面。
- 根目录和 frontend 目录中存在日志、截图、QA artifact，不应当作源码事实。
- 当前目录不是 git 仓库，无法用 `git diff` 审查变更，只能通过文件系统和命令检查。

这些问题应在对应模块优化时修复，或明确记录为何延后。

## 9. 前端行为与契约

### API Client

`frontend/src/api/client.ts` 使用 Axios，核心行为：

- Base URL 来自 `VITE_API_BASE_URL`，默认 `/api/v1`。
- 启用 `withCredentials`。
- 从 `localStorage.auth_token` 注入 Bearer token。
- 根据 i18n 注入 `Accept-Language`。
- GET 请求自动携带用户时区参数。
- 自动解包 `{ code, message, data }` 响应。
- 401 时尝试 refresh token 并重放请求。

新增 API wrapper 时，应返回类型化的已解包数据，不要重复手动解析响应 envelope。

### 路由守卫

`frontend/src/router/index.ts` 负责：

- setup 页面可访问性。
- 登录态校验。
- 管理员权限校验。
- 支付功能开关校验。
- simple mode 页面限制。
- backend mode 页面限制。
- OAuth 与支付回调公开访问。
- 动态 chunk 加载失败后的刷新处理。

新增或优化页面时，必须同步 route meta、sidebar、i18n、API wrapper 和测试。

### 侧边栏与功能开关

`AppSidebar.vue` 根据以下条件构建导航：

- 用户角色。
- simple mode。
- backend mode。
- 公开功能开关，例如 payment、affiliate、risk control。
- 自定义菜单项。

当前管理员账号中心包含：

- `/accounts`：统一账号工作台。
- `/admin/total-accounts`：总账号池分配/回收界面。

### UI 方向

这是一个运营型 SaaS 控制台，不是营销站点。优化前端时应优先保证：

- 信息密度适中。
- 表格和筛选易扫描。
- 状态清晰。
- 移动端可用。
- 错误、加载、空状态完整。
- 租售账号字段完整展示，访问范围由权限控制。

优先使用已有组件：

- `AppLayout`
- `TablePageLayout`
- `DataTable`
- `Pagination`
- `BaseDialog`
- `ConfirmDialog`
- `SearchInput`
- `Select`
- `Input`
- `TextArea`
- `Toggle`
- `StatusBadge`
- `EmptyState`
- `LoadingSpinner`
- `Toast`

不要轻易引入新的设计系统。

## 10. 优化质量标准

用户目标是对每一个基础功能的前后端进行全面优化和审查，提升代码质量、设计逻辑、契约一致性、安全性和用户体验，同时尽量保留原项目中仍有效的基础功能。

决策规则：

- 当前稳定且符合 SocialOps 方向的行为，应优先保留。
- 优化重点是正确性、安全性、可维护性、清晰度和一致性。
- 只有在确实减少重复、修复边界问题或降低复杂度时才重构。
- 不做猜测式抽象。
- 保持模块边界清楚。
- 修改行为时增加或更新聚焦测试。
- 优先小步、可验证的优化，不做大范围无依据重写。

后端质量清单：

- Handler、Service、Repository、Schema 职责分离。
- Handler 负责请求解析、认证主体、分页、响应映射和错误响应。
- Service 负责业务不变量和事务。
- Repository 负责持久化细节。
- migration 只新增，不修改已应用文件。
- 使用 `infraerrors` 和 `response.ErrorFrom` 返回结构化错误。
- 认证、计费、代理、任务执行必须 fail-closed。
- 租售账号字段必须完整存储、返回、导出和展示；不要把账号密码、邮箱密码、认证 cookie、代理快照等字段当作需要遮蔽的“敏感字段”处理。
- 系统级密钥、JWT secret、支付密钥、OAuth client secret、数据库密码等平台配置不属于租售账号交付字段，仍应只在有权限的配置入口中处理。
- 修改计费时必须覆盖订阅额度、钱包补足、竞争保护、失败不扣费。
- 修改异步任务时必须覆盖队列满、executor 缺失、幂等、失败状态。

前端质量清单：

- API wrapper 类型必须贴合后端契约。
- 页面组件保持聚焦，只有真实复用或复杂度需要时才抽组件/组合函数。
- route meta、sidebar、i18n、API wrapper、页面调用必须同步。
- 认证、应用设置、订阅等状态优先使用已有 store。
- 保留 token refresh 和 pending OAuth session 行为。
- 避免无意义的 `any`。
- 加载、空状态、错误状态、禁用状态、移动端状态必须明确。
- 用户可见账号字段必须保持完整；任务结果展示应准确反映执行与扣费状态。
- API 契约、路由守卫、store、复杂页面逻辑要有 Vitest 覆盖。

## 11. 推荐审查顺序

建议按以下顺序推进“基础功能全面优化”：

1. API 契约一致性
   - 核对前端 API wrapper 与后端路由。
   - 修复缺失或过时方法，例如任务日志列表。
   - 校验响应 envelope 与分页结构。

2. SocialOps 账号工作台
   - 用户账号导入、列表、删除、导出。
   - 管理员账号创建、导入、导出、更新、删除。
   - 分配与回收。
   - 租售账号字段完整返回与权限边界。

3. 代理管理
   - 代理 CRUD。
   - 连通性测试。
   - 所有权校验。
   - 默认代理快照。
   - 删除清理。

4. 任务执行
   - 动作归一化与校验。
   - 批量提交。
   - 幂等。
   - executor queue fail-closed。
   - Twitter / X 执行器。

5. 计费与订阅
   - 费用估算。
   - 订阅额度。
   - 钱包补足。
   - usage ledger 投影。
   - 缓存失效。

6. 认证与身份
   - 登录、注册、刷新、会话撤销。
   - TOTP。
   - OAuth callback、创建账号、绑定账号。
   - backend mode 与 simple mode。

7. 支付
   - 套餐、订单、支付配置、webhook、resume flow。
   - 用户购买页与管理端支付页。

8. 管理后台基础功能
   - 用户、用户组、订阅、兑换码、优惠码、公告、设置、分销。

9. 前端 UX 一致性
   - 表格页面。
   - 表单与弹窗。
   - 空状态、错误状态、加载状态。
   - 移动端。
   - i18n 覆盖。

10. 部署与运维
    - 配置样例。
    - Docker 和 deploy 脚本。
    - 日志。
    - 安全响应头、CORS、trusted proxies。
    - secret scan。

## 12. 验证命令

后端：

```powershell
cd D:\Downloads\socialops-main\backend
go test ./...
go test ./internal/service ./internal/handler ./internal/repository
go test ./internal/service -run Social
go test ./internal/handler -run AccountWorkbench
go test ./internal/repository -run UsageLog
```

前端：

```powershell
cd D:\Downloads\socialops-main\frontend
pnpm run typecheck
pnpm run lint:check
pnpm exec vitest run
pnpm exec vitest run src/api src/router src/stores
```

完整构建：

```powershell
cd D:\Downloads\socialops-main
make build
```

安全扫描：

```powershell
cd D:\Downloads\socialops-main
python tools/secret_scan.py
```

注意：`backend/Makefile` 的 `make test` 会运行 `golangci-lint`，需要本机已安装该工具。

## 13. Codex 目标功能使用建议

如果使用 Codex 的“目标/Goal”功能，建议目标不要写得过宽。不要写“优化整个项目”这类不可收敛目标，而要写成可完成、可验证的模块目标。

推荐目标格式：

```text
目标：完成 [模块名] 的前后端契约审查与质量优化，修复发现的真实问题，并通过相关验证命令。
范围：[后端文件/前端文件/测试范围]
完成标准：
1. 已阅读 PROJECT_GUIDE.md 和模块相关代码。
2. 已列出并修复契约不一致、权限、错误处理、租售账号字段完整交付、状态流或 UI 状态问题。
3. 已补充或更新必要测试。
4. 已运行相关验证命令，并说明结果。
```

不建议一次性把“所有基础功能”放进一个 Goal。更适合按本文第 11 节的审查顺序逐个模块推进。

## 14. 后续 Codex 优化提示词

后续可以直接使用下面这段中文提示词：

```text
你现在在 D:\Downloads\socialops-main 工作。请先完整阅读 PROJECT_GUIDE.md，并按其中的项目事实和质量要求执行。

目标：对当前项目的一个基础功能模块进行前后端全面优化和审查，提高代码质量、契约一致性、可维护性、安全性和用户体验。当前项目是从 D:\Downloads\sub2api-0.1.133 魔改而来，已移除 AI 网关方向，保留大部分 SaaS 基础能力；不要把功能改回 AI 网关。

本轮模块：[填写模块，例如“账号工作台任务日志列表契约”或“代理管理”]

要求：
1. 禁止跳过项目理解。先阅读相关后端路由、handler、service、repository/schema/migration，再阅读前端 route、api、store、view/component、i18n 和测试。
2. 先找前后端契约不一致、错误处理、权限边界、租售账号字段未完整返回/导出/展示、数据竞争、计费/状态语义、UI 状态缺失和测试缺口。
3. 在不破坏现有基础功能的前提下进行小步重构或修复。保留原项目中仍有效的认证、订阅、支付、设置等底座逻辑。
4. 新增或修改 API 时，同步后端路由/handler/service、前端 API 类型、页面调用、i18n 和测试。
5. 对 SocialOps 任务执行和计费保持 fail-closed 与 success-only billing：失败不扣费，队列/代理/认证/平台执行异常不应破坏账号字段完整性或造成错误扣费。
6. 使用现有代码风格和共享组件，不引入不必要的新抽象。
7. 完成后运行与变更范围匹配的 Go/Vitest/typecheck/lint 命令；无法运行时说明原因。
8. 最终输出：修改文件、关键修复点、验证结果、剩余风险。
```

## 15. 本项目编辑规则

- 不修改已经应用过的 migration 文件，需要变更时新增 migration。
- 不删除 `README.md`、`README_CN.md`、`README_JA.md`、`DEV_GUIDE.md`、`CLA.md`、`LICENSE`、`docs/`、`deploy/` 文档，除非用户明确要求。
- 不把截图、日志、QA artifacts 当作源码事实。
- 不隐藏、不脱敏、不加密租售账号字段；字段访问控制依靠登录态、角色、归属关系、分配关系和管理权限。
- 不在前端页面添加解释实现细节的可见说明文字。
- 运营功能不要做成营销落地页，要直接做可用控制台页面。
- 没有明确问题、重复或边界收益时，不做大范围重构。
