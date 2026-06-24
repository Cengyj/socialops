# SocialOps 项目指南

最后复核时间：2026-06-09

本文档是当前 SocialOps 项目后续优化、审查、重构时的唯一项目级参考。当前项目路径为 `D:\Downloads\socialops-0.2.1`，项目来源为旧 `sub2api` 代码基线。

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

- 除 `execution_auth` 外，本项目不隐藏、不脱敏、不加密租售账号字段。
- 账号名称、平台 ID、密码、手机号、邮箱、邮箱密码、认证 cookie、默认代理快照、备注等字段都属于租售账号交付数据，不应跟随 `execution_auth` 的密文规则扩大加密范围。
- `execution_auth` 是账号字段里的唯一加密例外：该参数存储、接口返回、详情预览、导入导出和前端状态识别全程保持密文，不存在明文交付形态。
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
- `/usage`
- `/announcements`
- `/redeem`
- `/subscriptions`
- `/accounts`
- `/accounts/batch-import`
- `/accounts/batch-delete`
- `/accounts/export`
- `/accounts/:id/default-proxy`
- `/accounts/default-proxy`
- `/accounts/tasks`
- `/proxies/*`
- `/task-settings/templates/*`

管理员路由：

- `/admin/dashboard/*`
- `/admin/users/*`
- `/admin/announcements/*`
- `/admin/redeem-codes/*`
- `/admin/promo-codes/*`
- `/admin/settings/*`
- `/admin/backups/*`
- `/admin/system/*`
- `/admin/subscriptions/*`
- `/admin/groups/*`
- `/admin/user-attributes/*`
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
- `APIKey`（历史 schema 残留；当前用户/管理端 API Key HTTP 面与旧 API Key 鉴权中间件未注册到运行时启动图）
- `UsageLog`
- `UsageCleanupTask`（历史 schema 与迁移事实保留；当前无运行时 service、repository、handler 或配置入口）
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

- `/accounts` 与总账号池删除账号都是物理删除；删除前会清理关联任务日志并解除代理绑定。
- 总池唯一性通过 `platform_key + name_key` 归一化字段控制。
- 默认执行代理以快照形式保存在 `social_accounts.default_proxy_snapshot`。
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

管理员可以创建、导入、导出、更新、删除、分配和回收社交账号。总账号池是账号库存的唯一来源。

用户批量导入账号时，会先按平台和用户名匹配已有总池账号；匹配到唯一未分配账号时绑定到当前用户。没有匹配项时，系统会创建仅在当前用户工作台可见的 `not_stored` 暂存账号，管理员可通过“上传入库”把暂存账号转入总账号池。账号删除是物理删除，不保留工作台隐藏态或恢复态。

### 用户账号工作台

用户可以：

- 查看已分配账号。
- 批量导入账号，绑定总池账号或创建工作台暂存账号。
- 将未入库暂存账号上传入总账号池。
- 从自己的工作台彻底删除账号。
- 导出租售账号完整字段。
- 配置账号默认执行代理。
- 提交社交执行任务。

除 `execution_auth` 外，租售账号字段必须完整交付。账号名称、平台 ID、密码、手机号、邮箱、邮箱密码、认证 cookie、默认代理快照、备注等字段都属于租售账号业务数据，应按接口权限完整返回、导出和展示。权限边界应通过登录态、角色、账号归属、分配关系和管理权限控制，而不是通过字段省略、星号掩码、哈希、加密或只返回部分字段来控制。`execution_auth` 是唯一字段级加密例外，接口和导出只能交付密文。

### 代理管理

`social_ips` 是用户拥有的执行代理。代理必须通过连通性测试，并且状态为 `online` 后，才能用于任务执行或设置为账号默认代理。

删除代理时，需要清理引用该代理的账号默认代理快照。

### 任务执行

当前规范化动作：

- `login`
- `login_check`
- `follow`
- `post`
- `like`
- `retweet`
- `update_profile`
- `update_avatar`
- `update_banner`

说明：

- `login` / `login_check` 是账号工作台直提动作，不属于任务模板类型。
- 旧动作 `tweet` / `dm` 不再被规范化接受，不要恢复旧别名兼容。
- 当前没有 `message` 规范化动作或可执行路径，除非产品重新定义现有范围内的真实入口与执行契约。

媒体执行边界：

- 任务媒体的权限边界来自账号归属、任务权限、代理与执行校验，不要把社交账号交付字段当系统密钥遮蔽。
- 已存储媒体只有 `source = "library"` 且 `storage_key` 位于 `social-task/` 前缀下，才属于当前社交任务上传域内的可执行引用。
- 旧的泛化 `media/...` 存储键不属于当前执行域，应继续 fail-closed，而不是放宽生产校验。
- 视频发帖媒体当前不在 SocialOps 执行边界内，应明确失败且不扣费，不要使用带有临时待办感的英文占位文案。

任务流程：

1. `AccountWorkbenchService` 标准化并校验请求。
2. 账号必须已分配；在用户模式下必须属于当前用户，并且 `account_status = available`。
3. 用户模式下一批任务只能包含同一平台账号。
4. 每个待执行任务都会创建 `pending` 状态的 `social_task_logs`。
5. 任务日志可带代理快照与幂等键。
6. executor queue 接收 task log ID。
7. executor 不存在或队列已满时，任务必须被标记为失败且不扣费。
8. worker 将 `pending` 任务 claim 为 `running`。
9. 平台执行器执行真实动作。
10. 执行失败时，任务失败且不扣费。
11. 执行成功后调用计费 finalizer，在事务内完成订阅额度、钱包余额、任务状态和 usage ledger 更新。

### 计费

当前社交任务仅登录动作单价为 `0.1`，其他动作价格为 `0`。

计费原则是 success-only billing：

- 实际扣费只发生在真实执行成功之后。
- 结算在事务内更新 `user_subscriptions`、用户余额、`social_task_logs` 和 `usage_logs`。
- 执行失败、平台不可用、代理不可用、认证失败、队列溢出、executor 缺失时都不能扣费。
- 计费失败不能静默吞掉，也不能伪装成成功。

订阅额度与平台相关，依赖 group 或 subscription plan 的 platform 字段。

### Twitter / X 执行器

当前真实执行器位于 `backend/internal/service/twitter_executor.go`。

该执行器会：

- 从 `social_accounts.execution_auth` 在后端内部构造平台认证头；`auth_cookie` 作为完整登录备份/交付字段保留，不作为页面或导出的明文执行凭证来源。
- 要求 OAuth token 和 token secret。
- 要求任务携带代理快照。
- 使用代理构建 HTTP client。
- 调用 Twitter / X 的接口执行登录检查、关注、点赞、发帖、转发。

优化该区域时必须保留：

- fail-closed 行为。
- 除 `execution_auth` 外，租售账号字段完整传递，不做脱敏或加密；`execution_auth` 全程保持密文。
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
- 备份配置、备份记录与恢复操作。
- 看板与使用量投影。
- Redis 缓存与限流。
- 前端嵌入式服务。

这些基础模块后续应保守优化。除非 SocialOps 业务明确要求，否则不要重写其业务含义。

## 7. 已移除或废弃的 AI 网关方向

原项目存在多类 AI 网关概念，例如上游接入、模型配置、请求监控、生成任务和映射规则等。

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
- 旧用户/管理端 API Key HTTP 面、API Key 鉴权中间件、以及应用启动图中的 APIKeyService/Repository/Cache 初始化已移除；后续不要为了兼容旧 AI 网关重新挂回。
- 旧用户 API Key 鉴权缓存配置、认证快照类型、L1/L2 缓存实现、Redis Pub/Sub 失效通道和 `ristretto` 依赖已移除；历史 `api_keys` schema 仍保留给迁移与账务投影边界，但旧 APIKeyRepository 运行时包装已移除。
- 旧用户 API Key 创建尝试 Redis 限流缓存与日用量计数缓存已移除；不要重新为已删除的用户 API Key HTTP 面恢复这些缓存入口。
- 旧用户 API Key 请求费率限制 Redis 计费缓存（`apikey:rate:*`）已移除；当前 BillingCache 只保留余额与订阅用量缓存。
- 旧用户 APIKeyService 运行时服务与 `default.api_key_prefix` 生成配置已移除；不要再为已删除的用户 API Key 生命周期恢复服务层包装。
- 旧用户 API Key IP ACL 的“信任转发 IP”配置、设置接口字段和管理端开关已移除；mock API 默认设置中也不再暴露对应旧字段，并由 mock settings 契约测试阻止回流；系统级 Admin API Key 设置仍保留并按系统密钥处理。
- 旧用户 API Key IP 白/黑名单匹配工具、API Key group context key、无运行时入口的 trusted-client-IP helper，以及没有写入方的 platform/account 日志 context key 已移除；当前 IP 与请求上下文工具仅保留登录、Turnstile、日志等仍在使用的能力。
- 后台用户管理搜索已移除旧 API Key 匹配维度；当前只按邮箱、用户名和备注搜索用户。
- 后台仪表盘 handler 已改为显式 SocialOps 响应 DTO 投影，不再向管理端暴露旧 API Key、Token、TPM 和标准 AI 计费字段；后端 `DashboardStats`、趋势和排行投影也已移除这些旧占位字段。
- 后台仪表盘 mock API 与前端 API wrapper 已收敛到当前 SocialOps DTO：统计、趋势、用户趋势和用户消费排行只使用任务数与实际扣费字段，不再兼容旧 `{ trend: [...] }` 包装、纯数组排行、API Key、Token、TPM 或标准 AI cost 字段。
- 后台仪表盘 HTTP、前端类型、页面和 mock API 的任务统计契约已收敛为 `operations` / `charged` / `recent_operations_per_minute`；不要再恢复旧 `requests`、`actual_cost`、`total_actual_cost`、`total_requests` 或 `rpm` 字段。
- 后台用户 usage 统计接口 `/admin/users/:id/usage` 已收敛为 SocialOps 任务统计 DTO，仅返回 `total_operations` 与 `total_charged`；前端 wrapper、mock API 和测试不再保留旧 `period`、`total_requests`、`total_cost` 或 `total_tokens` 契约。
- 后台仪表盘旧空壳兼容接口已移除，包括 `snapshot-v2`、`realtime`、`groups`、`users-usage`、`user-breakdown` 和 `aggregation/backfill`；当前仅保留已有页面实际使用的统计、趋势、用户趋势和消费排行接口。
- 后端 `usagestats` 包中的旧模型、分组、端点、账号用量响应和批量 API Key/User 用量 DTO 空壳已移除；当前只保留消费记录、仪表盘、趋势和用户消费排行等 SocialOps 真实入口仍在使用的统计投影。
- 用户侧 `/usage/stats` 的后端 `UsageStats` 投影已收敛为 SocialOps 任务统计，仅保留 `total_operations`、`success_count`、`failed_count` 和 `total_charged`；不再保留旧 token、标准 AI cost、账号成本、平均耗时或 endpoint 统计占位字段。
- 用户侧使用记录统计卡和语言包已同步收敛到当前 SocialOps 任务统计，只保留操作总数、成功数、失败数、成功率和总扣费；未引用的旧数量合计/总成本统计文案已移除。
- 用户侧 usage 查询后端过滤 DTO 已收敛为当前任务历史页面真实使用的 SocialOps 条件，仅保留用户、动作、平台、账号名、状态和时间范围；不再保留旧 API Key、group、model、request_type、stream、billing_type 或 billing_mode 兼容过滤分支。
- 前端用户 usage API surface 已移除无调用方的 `usageAPI.query` 与 `usageAPI.getStatsByDateRange` 别名；当前任务历史列表统一通过 `usageAPI.list` 调用 `/usage`，统计统一通过 `usageAPI.getStats` 传入筛选条件，避免继续维护重复包装。
- 前端认证 API surface 已移除无运行时调用方的 pending OAuth test-helper 导出和重复包装，包括 `getPendingOAuthBindLoginKind`、`isPendingOAuthCreateAccountRequired`、`hasPendingOAuthSuggestedProfile`、`completePendingOAuthBindLogin` 与未接入页面的 `createPendingDingTalkOAuthAccount`；当前 OAuth 回调页统一保留 `exchangePendingOAuthCompletion` 与真实 provider 注册完成封装。
- 前端共享类型 barrel 已移除无引用的泛用 API error、validation、table filter/sort/pagination、批量兑换码请求壳和旧代理订阅转换契约；用户余额/并发注释与分组类型标题已收敛到 SocialOps 任务执行计费和订阅访问语义。
- 前端 `useRoutePrefetch` 已移除仅为旧测试保留的 `_getPrefetchConfig`、`_isAdminRoute`、`_adminPrefetchMap` 和 `_userPrefetchMap` 导出；路由预加载测试改为验证公开的 `triggerPrefetch` 行为，避免把内部邻接表继续暴露成维护面。
- 前端 `useNavigationLoading` 已移除仅为测试保留的 `_resetNavigationLoadingInstance` 下划线导出；导航测试改用公开状态对象的 `resetState()` 复位，避免把测试辅助继续暴露成 composable API。
- 根目录 `assets/partners/logos` 下未被源码、文档或构建入口引用的旧 AI/API 合作方 logo 素材已移除；当前 SocialOps 不保留无入口的旧网关生态品牌墙资产。
- 前端语言包中未被调用的旧 `modelDistribution` / `viewModelDistribution`、旧 token/action/cache/group/model dashboard 文案 key、首页旧对比/平台区语言包残留和未引用的套餐卡 `models` 文案已移除；首页标签内部 key 已从旧 API/会话/实时计费语义收敛为订阅访问、账号分配和任务记录语义；当前仪表盘真实入口使用 SocialOps 任务、平台、账号状态和消费排行文案。
- 后端 service 层中真实运行时仍在使用的订阅、通知邮件、幂等、后台管理和账单缓存契约已从 `legacy_stubs` / `restored_defs` / `*_skeleton` 文件名与注释收敛为当前 SocialOps 语义；孤立无引用的旧 API Key IP ACL 客户端访问标记已移除。
- 后端幂等启动图已移除无实际清理行为的 `IdempotencyCleanupService` 空服务、未使用的过期记录清理接口与清理配置项；当前只保留真实写接口和系统操作锁正在使用的幂等协调器。
- 后台仪表盘已移除未接入的缓存/预聚合占位，包括 `dashboard_cache`、`dashboard_aggregation` 配置、`DashboardStatsCache` provider、未注册的 dashboard cache repository、未挂载的 aggregation service 和测试壳；当前仪表盘统计直接走 SocialOps usage 投影，历史聚合表迁移仅作为数据库历史事实保留。
- 部署样例中的旧后台仪表盘预聚合环境变量和标题残留已移除，部署契约测试会阻止 `DASHBOARD_AGGREGATION_*`、`Dashboard Aggregation` 或 `dashboard_aggregation` 重新出现在示例配置中。
- 后台分组 handler 已移除未使用的 dashboard service、admin service 和 group capacity 空依赖；当前只依赖真实使用的 group repository。
- 用户侧 `PaymentHandler` 构造函数已移除只为旧 channel service 调用保留的可变参数兼容尾巴；前端后台仪表盘 API 测试中的已删除 wrapper/array fallback 夹具命名也已从 `legacyShape` 收敛为 removed-shape 语义。
- 用户侧 `/payment/channels` 空数组接口、`paymentAPI.getChannels()` wrapper 和无引用的 `PaymentChannel` 类型已移除；当前购买页支付方式和套餐统一走已有 `/payment/checkout-info`、`/payment/plans` 与 provider 实例配置。
- 后台订阅分配兼容入口 `/admin/subscriptions/assign`、`/admin/subscriptions/bulk-assign`、前端 `subscriptionsAPI.assign` / `bulkAssign` wrapper 和 group-or-plan 请求类型已移除；当前后台订阅创建统一走套餐驱动的 `/admin/subscriptions` 与 `/admin/subscriptions/bulk`，服务层分配原语仅保留给支付履约、兑换码、默认订阅等内部业务链路。
- 无入口的旧定时模型测试前端类型已移除；无 handler/route 且未启动的 usage cleanup service、repository、配置项、测试壳和运行时挂载已移除，历史 Ent schema 与迁移事实暂保留。
- 旧用户 plan 兼容入口 `/plans`、`/my-plan` 以及 `PlanHandler`/`PlanService` 运行时包装已移除；当前套餐目录走 `/payment/plans`，用户订阅状态走 `/subscriptions/*`。
- 旧 `data-management` 代理、数据源 profile、S3 profile 和备份 job 兼容入口已移除；当前备份能力保留在后台设置页备份标签与 `/admin/backups/*` 接口，前端不再保留独立 `/admin/backups` 跳转路由。
- 后台账号工作台注册占位入口已移除，包括 `POST /admin/accounts/register`、前端 `accountWorkbenchAdminAPI.register`、注册占位弹窗按钮和 mock 503 分支；不要为了“占位”恢复无真实业务入口的注册 UI/HTTP 包装。后端服务层真实登录/凭证注册器仅保留在已接入的执行链路中。
- 用户侧账号工作台详情页已移除旧摘要预览假设，邮箱令牌和认证 Cookie 按社交账号交付字段展示；只有 `execution_auth` 是加密参数，它是执行器内部凭证字段，存储、接口交付、详情、导出和前端状态识别全程保持密文，不应还原为 `access_token` / `token_secret` 明文。
- 未接入真实运行时校验的全局 `security.url_allowlist` 配置面已移除，包括 `upstream_hosts` / `crs_hosts` 字段、部署环境变量和启动时“SSRF checks disabled”警告；当前仍保留真实使用的 URL 格式校验、HTTP 客户端 DNS Rebinding 校验和更新服务的 GitHub 来源限制。
- 启动阶段 simple mode 注释已从旧分组初始化阶段说法收敛为当前管理员并发初始化语义；前端 onboarding admin steps 也移除了未使用的 simple mode 参数，避免继续保留无效导览分支暗示。
- 部署示例中的旧用户 API Key 鉴权缓存配置段已移除，部署契约测试会阻止对应 snake_case / 环境变量写法重新出现在示例配置中；系统级 Admin API Key 设置仍按系统密钥边界保留。
- 前端认证视图目录已移除无运行时调用方的 `@/views/auth` barrel，以及脱离当前邮箱登录、OAuth、TOTP、Turnstile 和登录协议真实流程的大体量示例/视觉说明文档；旧登录表单遗留的 `auth.rememberMe` 语言包 key 已移除；layout 组件文档也已收敛为当前 SocialOps 导航与 shell 事实，不再保留已移除入口的示例路线图。
- 英文 README 已移除不存在的 `PROJECT_STATUS.md` / G001-G008 证据链断链和无法由当前仓库验证的迁移完成叙述；当前公开文档统一以 `PROJECT_GUIDE.md` 作为项目边界和迁移债务的权威入口，中文开发者提示也不再暗示需要新增文件或能力。
- `PROJECT_GUIDE.md` 验证命令与后续 Codex 提示词中的旧工作区路径已收敛为当前项目路径 `D:\Downloads\socialops-0.2.1`，避免后续维护任务从错误目录启动。
- `DEV_GUIDE.md` 已移除旧 AI 模型映射故障经验段，仓库说明、Go/golangci-lint 版本和项目结构也已收敛到当前 SocialOps/Codex 维护语义。
- mock API 公开设置已移除无真实前端/后端设置契约的旧 `risk_control_enabled` 开关，并由 mock settings 契约测试阻止该旧后台风控入口配置回流；当前真实 Twitter 风控相关注释只属于社交平台执行稳定性，不代表恢复旧 risk-control 管理模块。
- 后台邮件模板编辑器和 mock API 已收敛到当前 `NotificationEmailService` 真实支持的通知事件；旧 `account.quota_alert`、`content_moderation.*` 与 `ops.*` 模板元数据、旧占位符提示和 mock 可访问入口已移除，不再暗示存在风控/运维邮件模板能力。
- 三语 README 已从“真实执行器后续接入/当前不实现执行器”的旧阶段描述，修正为当前 Twitter/X 执行器与凭证注册器已接入、其他未支持平台或动作保持 fail-closed 的事实；部署契约测试会阻止 README 重新回到旧空壳执行器口径。
- 根目录默认 npm scaffold `package.json` 已删除；当前 Vue 应用依赖入口是 `frontend/package.json`，部署契约测试会阻止 `socialops-main`、固定失败的 `npm test` 或 `ISC` 默认许可证元数据原样回流。
- 前端支付语言包已移除无页面引用的旧渠道管理文案（`payment.admin.tabs.channels`、`channelName`、`createChannel` 等），支付路由策略文案收敛为 provider/服务商实例语义；第三方支付配置中真实需要的易支付渠道 ID 字段仍保留。
- 本地 deploy 运行日志和开发数据目录已明确纳入 `.gitignore` 与 ripgrep `.ignore`（`deploy/dev-*.log`、`deploy/dev-data/`、`deploy/dev-postgres_data/`）；未被进程占用的旧 dev 日志已清理，仍被本地运行进程占用的日志不应当作源码事实。
- 使用记录、账号工作台、任务设置和执行器相关测试中的旧 AI 品牌社交账号/帖子示例已统一替换为中性 `northwind` 夹具；设置页中的旧 AI 品牌词仍只作为“不得出现 AI 网关语义”的负向断言保留。
- 前端订阅套餐 catalog 已移除完成迁移后无调用方的 `SubscriptionPlanFamily` / `familyName` / `buildSubscriptionPlanFamilies` / `subscriptionPlanFamilyKey` 兼容层，当前只保留按订阅额度套餐表达的 `SubscriptionQuotaPackage`、`title` 和 `subscriptionQuotaPackageKey` 契约。
- 服务端社交任务动作归一化已不再接受旧 `tweet` 别名作为 `post` 输入，前端 usage action 语言包也不再保留 `actions.tweet`；任务设置页目标池 key 与中文动作文案已收敛为 post/帖子/发帖/转发语义。当前发帖动作统一使用 `post`，历史或平台级 Twitter/X 描述中的 tweet ID 文案不代表恢复旧动作名。
- 使用记录详情中的纯 URL 代理快照处理已从 `legacy plain proxy snapshot` 命名收敛为 `plain endpoint proxy snapshot`；当前保留的是任务历史代理摘要的安全展示/净化，不代表继续维护旧 AI 网关兼容入口。
- 设置仓储集成测试中的旧 `Sub2api` / `Subscription to API` 站点夹具已替换为 SocialOps 站点语义；`UsageLog` 类型注释也改为正向描述 SocialOps 任务活动与扣费记录，不再通过 “not AI token billing” 反向定义。
- 前端 usage API 测试中用于验证旧统计字段被丢弃的 `model` 夹具值已改为中性的 `stale-bookkeeping-field`；负向断言仍保留，以防旧模型/Token 账务字段重新进入 SocialOps 使用记录契约。
- 任务设置前后端测试中用于验证“不支持任务类型会被过滤/拒绝”的夹具已从泛化 `legacy_*` 命名收敛为 `unsupported_*`；`tweet` 旧别名拒绝测试保留为明确的 removed action guard。
- 前端 WeChat/OAuth 设置辅助中的 `legacyEnabled` / `legacyMode` / `legacyBindingNoteKeys` 等泛化旧语义命名已收敛为 aggregate/stored/raw-message 语义；当前只是保留既有聚合设置读取与 raw error extractor 边界，不代表恢复旧业务兼容层。
- 前端 profile 与 WeChat callback 测试中的合成 OAuth 邮箱和聚合 WeChat 设置夹具已从泛化 legacy 命名收敛为 synthetic/unbound/aggregate-only 语义；这些测试仍只覆盖当前邮箱绑定展示和回调恢复边界。
- 后端邮箱绑定和 pending OAuth 资料采纳测试中的合成邮箱用户/已有用户名夹具已从泛化 legacy 命名收敛为 synthetic/existing 语义；当前只是覆盖第三方登录合成邮箱绑定、资料采纳和会话完成边界。
- 后端登录时为已有邮箱用户补齐 email identity 的测试，以及 pending OAuth browser session 消费、handler exchange/normalize 时清理存储 completion token 的测试，已从泛化 legacy 口径收敛为 existing/stored 语义；provider callback 中为 pending 注册会话补齐 choice 状态的运行时 helper 也已收敛为 registration-choice 语义。LinuxDo/OIDC handler 中已有身份、`compat_email` 匹配和 no-adoption 注册夹具也已收敛为 existing/stored/no-adoption 语义。这些仍是当前认证与 OAuth session 安全边界。
- 后端用户仓储邮箱查找/存在性测试中的空格与大小写归一化夹具已从 legacy 口径收敛为 stored 语义；当前只是验证已存储邮箱格式不规整时仍按规范化邮箱定位用户。
- 前端 LinuxDo/OIDC/DingTalk/WeChat callback 中的 URL fragment token 恢复解析已抽到 `utils/oauthCallbackFragment.ts`，页面状态命名收敛为 fragment fallback 语义；后端字段 `pending_oauth_token` 仍是当前 OAuth 回调恢复契约，不代表恢复旧 AI 网关能力。
- 订阅默认配置、套餐创建和用户订阅平台过滤测试中的 group-only / Twitter-X 平台别名夹具已从泛化 `legacy` 命名收敛为 group-only/existing/alias 语义；当前仍保留已有订阅兑换、默认订阅和平台别名兼容逻辑。
- 后台设置、后台仪表盘、嵌入前端 bypass 和后台用户统计 API 的旧网关负向 guard 已从 `legacy` 口径收敛为 removed/stale 语义；测试仍继续阻止旧模型、Token、API Key、上游网关路由和旧设置字段回流。
- 用户/后台路由移除 guard、group-only 兑换码校验和 profile 绑定契约测试已从 `legacy` 测试名与夹具收敛为 removed/group-only/auth-binding contract 语义；当前仍保留真实前端依赖的 profile 绑定状态与来源字段。
- 前端共享类型 barrel 与后台设置页测试中已删除的代理订阅转换类型、可见支付方式控件 guard 以及支付负载策略下划线存储别名测试已从泛化 `legacy` 口径收敛为 removed/stored-alias 语义；当前只保留现有支付设置归一化覆盖，不新增支付能力。
- 用户支付结果页与支付 API 测试中的公开 `out_trade_no` 回跳恢复路径已从泛化 `legacy` 命名收敛为 provider-return/public-recovery 语义；当前只是保留既有支付回跳恢复，不新增支付流程。
- 前端支付恢复 snapshot 测试与订阅套餐类型注释已从泛化兼容口径收敛为 stored snapshot、内部执行分组绑定和当前兑换绑定语义；`monthly_limit_usd`、`group_id` 等字段仍按当前接口契约保留，不新增套餐能力。
- 后端部署契约测试中用于阻止旧产品定位、缺失模块、旧 Docker/环境变量文案回流的局部命名与失败信息已从泛化 `legacy` 口径收敛为 removed/stale 语义；管理员社交账号导入对旧 `.xls` 文件格式的拒绝提示也改为 old-format 语义，导入能力范围不变。
- Mock API 管理员账号导入 `.xls` 拒绝提示已与后端 old-format 口径对齐；微信支付回调与用户支付恢复测试中的 openid query 恢复命名也收敛为 public/provider-return 语义，当前只是保留既有回跳恢复输入，不新增支付流程。
- 后端支付 public `out_trade_no` 校验路由注释与 handler 测试已从泛化 `legacy` 口径收敛为 provider-return recovery 语义；当前只是继续覆盖既有支付服务商回跳恢复路径，不新增匿名支付能力。
- 后端公开设置注入漂移测试已移除失效的旧网关字段例外，当前要求公开 DTO 与 SSR 注入字段严格一致；后台设置负向 guard 中的身份补丁提示与已删除监控字段说明已从泛化旧口径收敛为 removed-gateway 语义，测试仍继续阻止已删除网关设置字段进入 SocialOps 设置契约。
- 后端公开设置 SSR 注入边界已收紧：`SettingService.GetPublicSettingsForInjection` 基于 `PublicSettingsInjectionPayload` 生成已序列化 JSON，`internal/web` 的 provider 接口只接受 `json.RawMessage` 并在注入前校验有效 JSON；web 层不应重新接收 `any` 或承担公开设置字段拼装职责。
- 后台设置 handler 负向测试名和断言失败信息已从直写旧网关类型/字段口径收敛为 removed-gateway 语义；当前只是守住已删除设置字段不会通过后台设置读写契约回流。
- 嵌入前端 bypass 列表的旧网关路由负向 guard 已从直写旧网关前缀/上游路径口径收敛为 removed-gateway 语义；仍继续阻止旧网关 API、模型列表和生成任务路径回到 SocialOps 后端前缀。
- 前端 LinuxDo callback 的 pending bind-login 响应测试已从泛化旧绑定口径收敛为当前 pending bind 语义；`bind_login_required` 和 `pending_oauth_token` 仍是现有 OAuth 绑定流程契约字段。
- 后端 WeChat OAuth handler 中 provider key fallback 与 OpenID-only 身份修复的内部命名已从泛化 legacy 语义收敛为 historical provider key / stored OpenID 语义；数据库中既有 `"wechat"` provider key 值仍按当前身份修复契约保留。
- 后端用户 profile identity 仓储测试中的 WeChat alias 夹具已从泛化 legacy 口径收敛为 stored alias 语义；当前仍只验证既有 `"wechat"` provider key 记录被归一到 `"wechat-main"`，不新增身份绑定流程。
- 后端用户 profile 身份摘要中的邮箱展示 fallback 和解绑保护测试已从泛化兼容邮箱口径收敛为当前 account email / profile email backfill 语义；OAuth 返回字段 `compat_email` 仍是现有绑定流程契约字段。
- LinuxDo/OIDC/DingTalk/WeChat OAuth handler 内部的第三方返回邮箱匹配 helper、参数、变量和测试夹具已收敛为 provider-email 语义；响应字段 `compat_email` 与 `compat_email_match` 仍是当前前端绑定流程契约，不应在未同步前端的情况下改名。
- 后端用户仓储 email identity 集成测试中的 LinuxDo reserved 邮箱夹具已从误导性用户夹具名收敛为 synthetic user 语义；当前仍只验证 `linuxdo-connect.invalid` 合成邮箱不会创建 email auth identity。
- 后端验证码与密码重置邮件在模板服务失败后的日志已从旧模板口径收敛为 built-in body；当前只是使用 EmailService 现有内置正文兜底发送，不代表保留旧模板系统。
- 通知邮件退订偏好与投递去重的旧长 key 读取 helper 已从泛化旧命名收敛为 historical key 语义；旧长 key 内部填充值仍按已存储形状保留，只服务既有退订/去重状态读取，不新增通知能力。
- 后端 payment resume 签名 key 迁移兜底和可见支付方式 source 缺失回退测试已从泛化旧口径收敛为 historical verification key / empty-source 语义；当前仍只保留既有支付回跳 token 验签和空 source 跨 provider 路由行为，不新增支付流程。
- 支付服务商配置的 AES-GCM 历史密文读取兜底在注释、测试名和夹具变量上已收敛为 historical ciphertext 语义；该路径属于系统级密钥迁移保护，读取行为仍保留，后续只应在确认线上配置均已重存为明文 JSON 后删除。
- 支付金额单位和可见支付方式别名测试已从泛化旧口径收敛：ISK/UGX 的 Stripe 特殊金额单位按 special-case 命名，支付宝/微信 direct 可见方式回退按 historical alias 命名；这些都只是现有支付规则和存量配置读取，不新增支付能力。
- 前端订阅套餐表单测试中的 `monthly_limit_usd` 读取已从泛化兼容口径收敛为 stored monthly limit fallback；当前仍只保留既有套餐额度字段归一逻辑，不新增订阅套餐能力。
- 后端退款路径中缺少 `provider_instance_id` 的历史订单处理已从泛化旧口径收敛为 historical/no-instance 语义；普通 provider 回查仍只在唯一可识别时解析，用户/管理员退款继续拒绝猜测 provider，权限边界不放宽。
- 后端 webhook/provider 回查测试已从泛化旧 provider/order/fallback 口径收敛为 historical 语义；当前仍只允许历史订单在唯一可识别时解析 provider，缺失快照实例时不回退到猜测 provider。
- 前端 mock subscription API 和后台设置页测试中的泛化兼容标题已收敛为 removed assign endpoints / optional OIDC security flags 语义；当前仍只验证已删除订阅分配入口不会回流，以及 OIDC PKCE/ID token 校验开关按现有设置保存。
- 分组 handler 注释、分组结构负向测试、公开设置注入漂移测试以及前端设置源码 guard 已从直写旧网关品牌口径继续收敛为 removed-gateway 语义；具体旧字段/旧品牌负向断言保留，用于防止旧网关业务字段重新进入当前 SocialOps 契约。
- Mock API 的任务模板契约已与真实前后端收敛：`login_check` 不再是任务模板类型，登录/登录检测仍按当前账号工作台行为无模板直提；需要参数的 follow/post/like/retweet/资料/头像/横幅动作继续通过模板执行，mock 日志保留当前 `payload` 与 `template_snapshot` 结构。
- 社交任务媒体校验与执行器错误文案已收敛到当前 SocialOps 执行边界：已存储媒体必须来自 `social-task/` 上传域，泛化 `media/...` 库引用与视频媒体继续 fail-closed 且不扣费，不再使用带有临时待办感的英文占位口径。
- 已删除失效的 `concurrency_cache_integration_test.go`：当前运行时 `ProvideConcurrencyService` 不再装配 Redis 并发缓存实现，旧测试引用的 `NewConcurrencyCache` 与 Redis key prefix 已不存在，保留该 integration 测试只会制造不可编译的空壳覆盖；当前并发服务行为由 `service` 层单元测试覆盖。
- repository integration 夹具中无调用且引用已删除 ent/service 模型的旧 `Proxy` / `Account` / `AccountGroup` helper 已移除；用户最近使用时间测试的 `usage_logs` fixture 已改为直接写入当前仍需要的 SocialOps 用量投影字段，订阅列表和分组级联测试也同步到当前仓储签名/构造方式，`go test -tags integration ./internal/repository -run "^$"` 已可通过编译检查。
- 后端 `internal/testutil` 中无调用方的 `StubConcurrencyCache` 共享空桩已删除；并发服务测试继续使用本地可观测 stub，避免全局默认空实现继续误导 Redis 并发缓存仍是运行时依赖。
- 钉钉 OAuth start 的 skipped sentinel 已改为真实 disabled 回调测试；DingTalk 配置 disabled 测试名也改为 bypass validation 语义，避免后续扫描把正常用例误判成跳过测试。
- pending auth adoption 中历史 PostgreSQL `pending_auth_session_id IS NULL` 清理路径已移到 repository integration 中用事务内 DDL/fixture 覆盖；service 层 SQLite 下永远跳过的同名单测已删除，当前只剩 Docker/testcontainers 环境门控类 `t.Skip`。
- Billing cache 与 payment refund 测试 double 中的 `not implemented` 文案已改为 `unexpected ... call`，明确这些方法不是半成品能力，而是测试路径不应触达的 guard。
- 用户订阅仓储中备注字段写入注释已收敛为当前 note contract 语义；`notes` 仍按现有字符串字段在创建和更新路径一致保存，不代表保留旧系统兼容层。
- 用户订阅调整、撤销和活跃订阅读取的 service 入口已统一收紧为 `context.Context` 参数，不再通过 `ctx any` 做动态兜底；调用方均应传入真实请求/事务 context，避免订阅计费路径重新扩散模糊运行时契约。
- 社交账号 `execution_auth` 已收敛为执行器内部密文字段契约：这是账号字段中唯一需要加密的参数。Twitter 登录注册器返回的 OAuth 三字段 JSON（`access_token`、`token_secret`、`screen_name`）只允许作为后端写入前的瞬时输入形状，账号服务创建/更新/导入和登录写回必须使用当前 SocialOps 执行凭证加密链路加密后存储；数据库、接口、详情预览、导出和前端状态识别全程都是密文，任务执行器仅在内部构造平台请求时使用该凭证，不提供明文字段交付。`execution_auth` 不使用 TOTP/备份路径的 AES-GCM 加密器，也不保留 AES-GCM 读写兼容。
- Ent schema 中的 `api_keys` 已明确标为历史表行保留，仅服务迁移和账本一致性；用户/分组 edge 也按 historical 口径注释。`usage_logs` 中残留的 `api_key_id`、`model` 和 token/timing 列也标为历史表形状字段，当前 SocialOps 路由和前端仍不恢复用户 API Key、模型或 AI 请求日志能力。
- 支付服务商配置弹窗的前端校验错误已收敛为静态 `useAppStore().showError` 路径，并补充组件测试防止校验失败时误触发保存；账号工作台依赖加载降级提示也改成明确的“控件加载失败”语义，不再使用带临时半成品感的可用性文案。
- 新生成的支付外部订单号前缀已从旧项目名收敛为 `socialops_`；历史 `sub2_数字ID` webhook DB-ID 兜底仍作为旧订单读路径保留，且只在当前 `out_trade_no` 查不到订单时触发。
- printf 风格日志桥接 helper 已从旧兼容命名收敛为 `ComponentPrintf`，结构化日志标记也改为 `stdlog_bridge` / `component_printf_bridge`；这只是当前 logger 边界命名整理，不改变日志级别推断或消息内容。
- 认证身份外部身份迁移集成测试的文件名、测试名、helper、变量和纯夹具展示值已收敛为 historical/stored 语义；迁移 SQL 文件名、报告类型/键和原始元数据保留字段仍作为数据库迁移契约保留，不应为追求表面清零而改动。
- 认证身份 109 回填迁移集成测试已从迁移文件名口径收敛为 stored-email backfill 语义；`109_auth_identity_compat_backfill.sql` 文件名和 metadata 值仍是数据库迁移契约。Atlas baseline 对齐中的 `schema_migrations` 检测也按 historical schema table 语义命名，表名本身保持不变。
- 后端 WeChat Connect 启动配置中的 `WECHAT_OAUTH_*` 读取已按 historical env fallback 语义命名并补充 env 优先级测试；当前 `WECHAT_CONNECT_*` 环境变量和 `wechat_connect.*` 配置仍优先，不新增配置入口。
- 后端 OAuth 注册来源推断 helper 已从泛化旧口径收敛为 `inferSignupSourceFromEmail`，仅在调用方没有显式 signup source 时按第三方登录合成邮箱域名推断当前来源。
- WeChat pending OAuth 身份修复中的 OpenID-only 身份变量已从泛化旧口径收敛为 stored OpenID 语义；该路径只用于把已存储 OpenID subject 修正为当前 canonical subject，不新增身份绑定流程。
- 发帖媒体模板边界已在前后端对齐为当前仅支持图片媒体：任务设置页、上传组件和账号工作台不再把单个 MP4 视频显示为可保存/可执行；后端模板校验与 Twitter executor 中死掉的 MP4 特例分支也已收敛为统一的视频 fail-closed，不新增视频发帖能力。任务媒体资产命名也不再为 `video/mp4` 生成 `.mp4` 扩展名，账号工作台不可达的视频模板预览分支已移除；公共上传组件仅保留存量视频预览显示，不提供视频上传入口。

## 8. 当前已知问题与风险

这些是本次重新阅读项目后确认到的事实，后续审查不能跳过：

- 任务历史页面当前统一通过 `/usage` 与 `usage_log_repo` 从 `social_task_logs` 投影；账号工作台仍保留 `POST /accounts/tasks` 提交入口和 `GET /accounts/tasks` 当前/近期任务活动轮询入口。
- 前端 `DashboardView.vue` 和 `UsageView.vue` 应继续使用 `usageAPI` 读取任务历史；`accountWorkbenchAPI.listTaskLogs` 只用于账号工作台任务活动流，不要恢复旧 `listMyTaskLogs` 命名或把 `/accounts/tasks` 扩展成通用历史页接口。
- 用户侧任务估算入口已从账号工作台移除，执行参数来自 `/task-settings` 保存模板；后续若恢复估算，必须先重新定义模板化估算契约。
- 空壳 `risk-control` 后台入口与接口已移除；旧 `data-management` 后台入口、接口、mock 和前端 API 也已移除。
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
- 除 `execution_auth` 外，租售账号字段完整展示，访问范围由权限控制；`execution_auth` 页面只可接收/显示密文形态。

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
- 除 `execution_auth` 外，租售账号字段必须完整存储、返回、导出和展示；不要把账号密码、邮箱密码、认证 cookie、代理快照等字段当作需要遮蔽的“敏感字段”处理。`execution_auth` 是唯一字段级加密例外，不能在 API、CSV 或页面中明文交付。
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
   - 成功执行后的扣费结算。
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
cd D:\Downloads\socialops-0.2.1\backend
go test ./...
go test ./internal/service ./internal/handler ./internal/repository
go test ./internal/service -run Social
go test ./internal/handler -run AccountWorkbench
go test ./internal/repository -run UsageLog
```

前端：

```powershell
cd D:\Downloads\socialops-0.2.1\frontend
pnpm run typecheck
pnpm run lint:check
pnpm exec vitest run
pnpm exec vitest run src/api src/router src/stores
```

完整构建：

```powershell
cd D:\Downloads\socialops-0.2.1
make build
```

安全扫描：

```powershell
cd D:\Downloads\socialops-0.2.1
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
你现在在 D:\Downloads\socialops-0.2.1 工作。请先完整阅读 PROJECT_GUIDE.md，并按其中的项目事实和质量要求执行。

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
- 除 `execution_auth` 外，不隐藏、不脱敏、不加密租售账号字段；字段访问控制依靠登录态、角色、归属关系、分配关系和管理权限。`execution_auth` 必须全程保持密文，不作为明文交付字段。
- 不在前端页面添加解释实现细节的可见说明文字。
- 运营功能不要做成营销落地页，要直接做可用控制台页面。
- 没有明确问题、重复或边界收益时，不做大范围重构。
