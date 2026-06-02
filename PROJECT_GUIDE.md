# SocialOps 项目指南

> **本文档是项目的权威说明。所有 AI 助手在修改本项目前必须先读完本文档。**
> **未在"允许修改"列表中的文件，禁止删除或重构。**
> **当当前代码、测试、旧注释或历史文档与本文档目标冲突时，以本文档目标为准；冲突实现属于待改造对象。**
>
> 配套文档：**[CLEANUP_PLAN.md](CLEANUP_PLAN.md)** — 详细的分批执行步骤（怎么做、按什么顺序、如何验证）

---

## 一、项目定位

### 当前定位

本项目正在从 **AI API 网关平台** 改造为 **基于网站总账号池的社交账号运营平台**。

改造按以下范围推进。本文档记录长期目标、业务边界和当前未完成任务快照；每次开工前仍必须以当前代码和验证结果为准。

| 阶段 | 范围 |
|------|------|
| Phase 1 | 前端清理：删除 AI 相关页面，新增社交账号 UI |
| Phase 2 | 后端清理：删除 AI 网关代码，保留通用平台代码 |
| Phase 3 | 社交功能实现：账号池、用户账号、管理端执行代理、执行计费、任务日志 |

### 最终目标

**核心边界：** SocialOps 只处理网站总账号池中已有的社交账号。任何不在总账号池里的账号，一律不录入、不绑定、不执行、不计费、不进入任务系统。

**用户侧：** 用户查看和导出自己名下账号完整数据；用户导入账号时只按平台用户名匹配总账号池已有账号（例如 X 的 `@xxx`，即账号 `name` 字段，不按邮箱、手机号或平台账号 ID 匹配），匹配到未分配账号则自动绑定给当前用户，匹配到已绑定账号或匹配不到则拒绝/忽略；用户不再拥有独立的账号管理、账号功能执行、IP/代理管理三套前端入口，也不暴露用户端 `/api/v1/social-ips`；真实社交执行动作成功后统一按次计费。

**管理侧：** 管理员维护网站总账号池，可导入账号、真实注册社交平台账号、手动分配未分配账号、回收账号。注册成功后按当前账号字段标准入池，注册失败不入池。账号回收后回到未分配状态，并清空该账号上属于原用户的默认执行代理/代理快照；不需要区分出售/出租状态。

**执行侧：** 后端真实执行器由后续人工接入；当前阶段只需要保留稳定的提交、校验、排队、状态流转、失败关闭、计费挂钩和日志框架。真实执行未接入、账号状态不允许、参数错误、IP 不可用、权限失败、执行异常等都属于非成功结果，不最终扣费。

**计费侧：** 真实社交执行动作单价先内置为 `0.1`。订阅套餐额度沿用原项目的金额额度逻辑，SocialOps 只接入原有订阅/钱包框架做额度检查、扣费确认和失败退回，不修改订阅套餐、钱包余额、配额窗口、重置周期等基础逻辑：优先扣订阅套餐额度，不足再扣钱包余额；成功才最终扣费，非成功不最终扣费，若已预扣则按原逻辑退回。账号导入/导出、分配/回收、IP/代理新增/修改/删除/测试、设置账号默认执行代理都不收费。任务日志需要能追踪本次执行的价格、扣费状态和扣费来源，但不要重建一套独立复杂账本。

**平台基础：** 复用原有的用户系统、支付、订阅、钱包余额、兑换码、推广返利、公告、管理后台框架。不要为了 SocialOps 重新设计这些基础能力。

### 当前业务规则（权威）

1. **总账号池优先**：网站总账号池是唯一处理范围。非总账号池账号不录入、不绑定、不执行、不计费。
2. **用户导入只匹配**：用户导入账号只按平台用户名匹配总账号池已有账号，例如 X 的 `@xxx`，对应账号 `name` 字段；不按邮箱、手机号或平台账号 ID 匹配。匹配前先做最小规范化：去除首尾空格；X/Twitter 用户名允许用户输入 `xxx` 或 `@xxx`，统一按 `@xxx` 匹配；X/Twitter 用户名匹配大小写不敏感。若导入数据带平台，则按 `platform + name` 匹配；若未带平台且命中多条，则拒绝为歧义匹配。未分配则自动绑定给当前用户；已绑定其他用户则拒绝；匹配不到则不处理。导入去重沿用当前项目已有逻辑。
3. **管理员入池**：管理员导入账号直接进入总账号池并按已有导入去重逻辑处理；去重优先按规范化后的 `platform + name` 判断，同一平台同一用户名不创建第二条。管理员注册代表真实注册社交平台账号，成功后有什么字段就按当前账号字段标准保存什么字段，失败不入池。
4. **分配与回收**：管理员可以手动把未分配账号分配给用户，也可以回收账号；回收后账号回到未分配状态，并清空该账号上属于原用户的默认执行代理/代理快照。出售/出租只是业务表达，不设计独立账号状态。
5. **账号数据可见性**：普通用户可以查看并导出自己名下账号完整数据，包括账号密码、邮箱密码、IP、备注等；不做额外脱敏、不做额外应用层加密、不记录用户导出审计日志。
6. **账号状态来源**：账号状态直接参考当前前端和当前后端已有状态，不另起一套复杂状态体系。真实社交执行准入先按最小规则执行：只有 `available` 允许执行；`pending_check`、`limited`、`invalid`、`not_stored` 等非可用状态都失败关闭且不扣费。若后续某些动作需要例外（例如仅登录检测允许 limited），再由真实执行器接入时单独补规则。
7. **IP/代理定位**：IP/代理是用户独有的执行网络出口/代理凭据，只能由所属用户使用，但维护入口统一放在管理端 `/admin/proxies`。账号可以保存一个默认执行代理字段，方便后续执行器按账号稳定调用，并避免每次随机分配导致差异过大；该字段不是账号所有权状态，账号回收时必须清空。当前实现可短期复用 `bound_ip` 表达默认执行代理/代理快照，后续更推荐收敛为 `default_proxy_id` 或 `default_proxy_snapshot`。用户自己的同一 IP/代理可用于自己名下多个账号，但不能跨用户使用。IP 测试免费，不计费。
   管理员可以代用户维护代理并给已分配账号设置默认执行代理，但代理仍必须属于该账号当前所属用户；未分配账号不保留默认执行代理。若后续恢复普通用户默认代理设置接口，仍必须只允许自己账号和自己代理，且不得恢复独立 `/ip-management` 大页面。
8. **执行准入校验**：用户提交执行任务时，必须先校验账号存在于总账号池、账号已分配给当前用户、账号状态允许执行、本次显式选择的 IP/代理或账号默认执行代理属于当前用户且可用、参数合法、订阅/钱包满足原项目扣费框架要求。
9. **用户执行计费**：只有真实社交执行动作成功后才计费，例如登录检测、关注、私信、发帖、点赞等需要调用后续真实执行器的动作。账号导入/导出、分配/回收、IP/代理新增/修改/删除/测试、设置账号默认执行代理都不收费。`edit_ip`、`set_default_proxy`、`change_proxy` 等代理配置动作属于免费账号/IP 配置，不应作为收费执行动作进入真实社交执行计费。
10. **统一单价**：当前所有真实社交执行动作内置单价统一为 `0.1`，暂不做后台可配置单价表。
11. **扣费复用原逻辑**：订阅套餐额度按原项目金额额度处理，优先扣订阅，不足扣钱包。SocialOps 只能作为社交执行消费方接入原项目已有订阅/钱包入口，不直接改余额字段、套餐定义、配额窗口、重置周期或基础扣费语义；不新增平行账本、不重写余额规则。原项目若采用预扣费，则沿用预扣费；原项目若已有订阅不足转钱包、拆分扣费或退款路径，则按原路径执行并在任务日志记录结果；执行成功确认扣费，非成功退回/不扣。
12. **成功才消费**：只有真实社交执行任务最终状态为 `success` 时才形成有效消费。失败、异常、权限失败、参数错误、账号状态失败、IP 不可用、执行器未接入等结果都不最终扣费。
13. **任务日志是执行凭证**：任务日志记录执行动作、账号、用户、目标、内容、执行状态、结果信息、单价、已扣金额、扣费状态、扣费来源和执行时使用的 IP/代理快照。`social_task_logs` 是社交执行主日志；原 usage/统计体系只作为展示、统计或订阅用量投影，优先参考当前项目已有日志/用量统计风格，避免新建平行复杂日志体系。
14. **执行器后续接入**：当前阶段不实现真实社交平台登录、关注、私信、发帖、点赞等执行逻辑；只保留可替换的执行入口。未接入执行器时必须失败关闭，不能误标成功，不能扣费。
15. **幂等与重复提交**：同一个任务或请求不能重复扣费。任务提交需要支持 `client_request_id` 或等价幂等键；建议按 `user_id + client_request_id + account_id + action` 识别重复提交。重复提交返回已有任务/日志结果，不重复创建任务、不重复预扣或扣费。优先复用原项目已有请求幂等、用量去重、钱包和订阅扣费机制；需要新增字段时只补足社交任务所需的最小标识。
16. **日志少改**：任务和用量日志优先参考当前项目/原项目已有逻辑，做字段含义和展示文案的细微调整即可，不重建复杂日志体系。
17. **暂不实现购买账号流程**：当前阶段不需要先实现账号购买流程，先完成管理员导入/注册/分配、用户导入匹配、账号归属、管理端代理、执行计费和日志闭环。
18. **批量任务计费粒度**：真实社交执行动作按“账号 + 动作”计费。一批 10 个账号执行同一动作，成功几个扣几次；失败、未执行、参数错误、账号状态失败、IP 不可用、执行器未接入的账号都不扣费。
19. **用户导出字段**：用户导出自己名下账号时，默认 CSV 即可；字段至少包括 `platform`、`name`、`account_id`、`password`、`phone`、`email`、`email_password`、`bound_ip/default_proxy`、`account_status`、`task_status`、`source`、`remark`、`created_at`、`updated_at`。这些是用户名下账号完整数据，不额外脱敏，不额外应用层加密，不记录导出审计日志。

### 目标判定速查

后续实现遇到具体功能时，先按下表判断是否属于 SocialOps 当前目标：

| 场景/操作 | 是否处理 | 是否收费 | 目标口径 |
|---|---|---|---|
| 管理员导入账号 | 处理 | 免费 | 进入网站总账号池；按规范化后的 `platform + name` 去重，同平台同用户名不重复建号。 |
| 管理员注册账号 | 保留入口 | 免费 | 代表真实社交平台注册；真实注册器未接入时失败关闭且不入池，接入后成功才按已有账号字段标准入池。 |
| 管理员分配账号 | 处理 | 免费 | 只能分配未分配账号；已绑定其他用户时拒绝，避免覆盖归属。 |
| 管理员回收账号 | 处理 | 免费 | 清空 `assigned_user_id` 并清空属于原用户的默认执行代理/代理快照，账号回到未分配状态。 |
| 用户导入账号 | 只匹配 | 免费 | 只按平台用户名匹配总账号池；匹配到未分配账号才自动绑定；匹配不到、歧义匹配、已绑定其他用户都不新增账号。 |
| 用户查看/导出账号 | 处理 | 免费 | 返回自己名下账号完整数据；不额外脱敏、不额外应用层加密、不记录用户导出审计日志。 |
| 管理端 IP/代理 CRUD/测试 | 处理 | 免费 | IP/代理是用户独有执行资源，但统一由 `/admin/proxies` 管理；同一用户可给多个账号使用，不能跨用户使用；测试不进入执行计费。 |
| 设置账号默认执行代理 | 处理 | 免费 | 只是执行配置，不是账号归属；普通用户只能设置自己账号和自己代理，管理员代设置也必须保持同一用户归属。 |
| 登录检测/关注/私信/发帖/点赞 | 处理执行框架 | 成功才收费 | 真实执行动作统一单价 `0.1`；按账号 + 动作计费；优先扣订阅额度，不足扣钱包，复用原项目预扣/确认/退回逻辑。 |
| 非总账号池账号 | 不处理 | 不收费 | 不录入、不绑定、不执行、不进入任务系统。 |
| 执行器未接入、账号状态失败、IP 不可用、参数错误、权限失败、执行异常 | 失败关闭 | 不收费 | 不能标记 `success`；如果已有预扣，按原项目逻辑退回/释放。 |
| 购买账号流程、出售/出租状态、后台单价表、新钱包/新账本、重复用户端执行页面 | 当前不做 | 不适用 | 先完成账号池、归属、管理端代理、执行框架、成功计费和日志闭环。 |

### Phase 3 最小闭环验收

Phase 3 不是把所有真实社交平台能力一次做完，而是先把平台闭环对齐到可接执行器：

1. 管理员可以导入账号入总账号池；管理员注册入口存在但在真实注册器未接入前失败关闭且不创建账号。
2. 管理员可以把未分配账号分配给用户，也可以回收账号；回收后账号未分配，且原用户默认执行代理/代理快照被清空。
3. 用户导入只按平台用户名匹配总账号池，匹配成功绑定，其他情况不新增外部账号。
4. 用户可以查看和导出自己名下账号完整数据，账号密码和邮箱密码不被额外隐藏、清空或应用层加密。
5. 管理员可以在 `/admin/proxies` 维护用户独有的 IP/代理资源；执行任务只能使用账号所属用户自己的代理，账号默认执行代理也必须属于当前账号所属用户。
6. 执行任务提交前完成账号池、账号归属、账号状态、参数、IP/代理归属和订阅/钱包可用性校验。
7. 真实执行器未接入时所有真实社交动作失败关闭；只有最终 `success` 的账号 + 动作才按 `0.1` 形成有效消费。
8. 扣费接入原项目订阅/钱包框架；不新增平行账本，不重写套餐额度、钱包余额、重置周期或基础扣费语义。
9. `social_task_logs` 能记录动作、账号、用户、状态、结果、单价、扣费状态/来源、代理快照和幂等键；重复提交不重复扣费。

### 当前完成度快照（2026-06-02）

本节是当前仓库审计快照，不是历史进度流水。后续继续修改前，必须重新读取当前代码、运行必要验证，再按本节口径更新判断。

| 模块 | 当前状态 | 说明 |
|---|---|---|
| AI 网关清理 | 基本完成 | 未发现 AI 网关路由、OpenAI/Claude/Gemini/Channel/Model 等核心文件继续作为业务入口；剩余 gateway 字样主要来自支付网关、测试说明或兼容注释。 |
| 前端重复入口清理 | 基本完成 | 不应恢复用户端 `/account-management`、`/account-functions`、`/ip-management`。当前社交账号入口应收敛为管理端 `/admin/accounts`、`/admin/total-accounts` 和 `/admin/proxies`，后续若做用户端账号页也只能保留一个统一入口。 |
| 社交账号总账号池 | 基本完成 | 管理端账号 CRUD、导入、导出、分配、回收、默认执行代理和任务提交框架已经存在；后续重点是修正边界缺口和真实执行器接入，不要重复新增一套账号模型。 |
| 用户导入和导出 | 基本完成 | 用户导入只匹配总账号池已有未分配账号；用户导出应返回完整账号数据，包括密码、邮箱密码、代理快照等，不做额外脱敏或应用层加密。 |
| IP/代理 | 基本完成 | 代理是用户独有执行资源，统一由 `/admin/proxies` 管理；账号默认执行代理只是执行配置，不是账号归属，回收账号和删除代理时必须清理旧代理快照。 |
| 任务日志和幂等 | 基本完成 | `social_task_logs` 应保留动作、账号、用户、状态、结果、单价、扣费状态/来源、代理快照、幂等键等字段；重复提交不得重复扣费。 |
| 计费框架 | 框架完成，仍需真实成功流验证 | 当前目标是成功任务才结算，优先扣订阅额度，不足扣钱包；失败、未执行、执行器未接入、账号状态失败、IP 不可用、参数错误均不最终扣费。真实执行器接入后必须用端到端测试证明成功扣费与失败不扣费。 |
| 真实社交执行器 | 未完成 | 登录检测、关注、私信、发帖、点赞等动作当前只能失败关闭。后续可参考 `/home/ceng/Downloads/FlyingBird` 的 Twitter 登录和动作实现，但必须适配当前项目结构、接口返回、代理、日志和计费标准，不能照搬造成第二套架构。 |
| 管理员真实注册账号 | 未完成真实注册 | 注册入口可保留失败关闭；真实注册器接入后，成功才按当前社交账号字段标准入池，失败不得创建账号。 |
| 订阅管理 | 部分完成 | 订阅列表、分配、批量分配、延期、重置、撤销等基础能力应保留并复用原项目逻辑；不要为 SocialOps 重写套餐、钱包或配额窗口。仍需修复与分组、前端返回类型和用量展示相关的残缺。 |
| 通用后台框架 | 部分完成 | 用户、认证、支付、订阅、钱包、兑换码、推广、公告等通用能力必须保留；分组管理、风控、部分仪表盘聚合、用量清理仍有 skeleton/disabled 状态，需要恢复到 SocialOps 可用口径。 |
| 容器服务更新 | 未完成本轮验证 | 代码和文档更新后，最终交付前仍需要重新构建前端、后端 `-tags embed`、Docker 镜像，并做 HTTP smoke/E2E 验证。 |

### 下一步未完成任务

后续实施按以下优先级推进，每完成一项都必须补充或更新对应测试，不能只改 UI 或只改文案：

1. **接入真实社交执行器**：在当前 `SocialTaskExecutor` 或等价可插拔执行接口下实现登录检测、关注、私信、发帖、点赞；参考 FlyingBird 时只迁移必要协议、请求、代理、锁和错误处理思路，输出必须符合当前 SocialOps 的任务日志和计费接口。
2. **补齐执行成功闭环测试**：覆盖用户/管理员提交任务、账号状态准入、代理归属、执行器未接入失败关闭、真实执行成功扣费、失败不扣费、订阅不足转钱包、钱包不足拒绝、幂等重复提交不重复扣费。
3. **完善社交账号管理和总账号池 UI**：确保 `/admin/accounts` 负责账号导入/注册入口/编辑/详情/执行日志，`/admin/total-accounts` 负责分配状态、分配、回收、默认代理、导出和批量清理；执行功能必须从社交账号 UI 调用当前任务接口。
4. **恢复通用后台基础能力**：优先修复订阅分组管理、风险控制、仪表盘聚合、用量清理和用量展示中的空壳接口；保留原项目用户、支付、订阅、钱包、兑换码、推广、公告等框架，不要因清理 AI 网关而继续删除基础能力。
5. **清理残留 AI 语义**：继续检查设置页、用量页、仪表盘、测试、文案和兼容存根，删除或改写 Token/模型/渠道/AI 账号语义；但 `api_key`、OAuth token、支付 gateway 等通用或非 AI 概念不得误删。
6. **端到端和容器验证**：按管理员导入账号、分配账号、维护用户代理、用户查看/导出、提交执行任务、失败不扣费、成功扣费、查看日志的顺序做 E2E；最后更新容器服务并做 smoke test。

### 当前实现待对齐清单

以下条目是当前代码中可能存在的过渡实现或历史残留。它们不是新的业务目标，后续实现时按本文档目标修正：

1. **账号凭据不要被现有加密/隐藏逻辑带偏**：如果当前实现存在账号密码或邮箱密码应用层加密、`json:"-"` 隐藏、导出空密码、因为加密 key 未配置导致账号无法导入/创建等逻辑，均属于待对齐项。目标是用户可查看并导出自己名下账号完整数据，管理员可按账号池管理需要查看/导出账号池完整数据。
2. **用户导入和用户导出必须补齐**：若当前用户端只有账号列表或任务提交接口，仍需补 `POST /api/v1/social-accounts/import` 和 `GET /api/v1/social-accounts/export`。用户导入只匹配总账号池，不新增外部账号；用户导出完整数据默认 CSV。
3. **管理员注册入口失败关闭**：若当前管理端缺少注册路由，需补 `POST /api/v1/admin/social-accounts/register`。真实注册器未接入前只能返回 not configured/失败关闭，不创建账号；注册失败不入池。
4. **执行动作必须先失败关闭**：真实执行器未接入前，登录检测、关注、私信、发帖、点赞等动作不能返回 `success`，不能扣费，只能记录未配置/失败关闭结果。
5. **计费挂钩只补最小闭环**：若当前 `social_task_logs` 缺少单价、扣费状态、扣费来源、幂等键或代理快照字段，应在 Phase 3 补足。不要改订阅/钱包基础逻辑，不要新增平行账本。
6. **代理配置不按执行动作收费**：如果当前代码把 `edit_ip` 或类似代理修改放在任务执行器里，应改为免费配置接口或设置默认执行代理接口；它不属于真实社交执行动作。
7. **分配/回收要防覆盖和清理代理**：分配只能面向未分配账号；已绑定其他用户应拒绝。回收账号必须清空属于原用户的默认执行代理/代理快照，避免下一位用户继续使用原用户代理。

---

## 二、绝对禁止操作

以下操作在任何情况下都不允许，除非用户明确说"我知道风险，请执行"：

### 禁止删除的文件（通用平台核心）

**后端 Service（`backend/internal/service/`）：**
- `user_service.go` — 用户管理
- `auth_service.go` — 认证（含 OAuth 登录）
- `payment_service.go` 及所有 `payment_*.go` — 支付系统
- `subscription_service.go`、`subscription_expiry_service.go`、`subscription_maintenance_queue.go` — 订阅管理
- `redeem_service.go`、`redeem_code.go` — 兑换码
- `promo_service.go`、`promo_code.go` — 优惠码
- `affiliate_service.go` — 推广返利
- `announcement_service.go`、`announcement*.go` — 公告
- `setting_service.go`、`setting.go`、`settings_view.go` — 系统设置
- `admin_service.go` — 管理员操作
- `api_key_service.go`、`api_key.go`、`api_key_auth_cache.go`、`api_key_auth_cache_impl.go`、`api_key_auth_cache_invalidate.go` — **用户认证 Key（不是 AI 调用 Key，必须保留）**
- `balance_notify_service.go`、`balance_notify_check_test.go` — 余额通知
- `email_service.go`、`email_queue_service.go` — 邮件服务
- `totp_service.go` — 2FA
- `turnstile_service.go` — 人机验证
- `update_service.go` — 版本更新检测
- `backup_service.go` — 数据备份
- `data_management_service.go`、`data_management_grpc.go` — 数据管理
- `identity_service.go` — 身份绑定
- `notification_email_service.go`、`notify_email_entry.go` — 通知邮件
- `registration_email_policy.go` — 注册邮件策略
- `usage_service.go`、`usage_cleanup_service.go` — 用量统计（后续改为社交操作统计）
- `auth_email_binding.go`、`auth_email_oauth.go`、`auth_oauth_email_flow.go`、`auth_oauth_first_bind.go`、`auth_pending_identity_service.go` — OAuth 绑定流程
- `deferred_service.go` — 延迟任务
- `system_operation_lock_service.go` — 系统操作锁
- `group_capacity_service.go`、`group.go`、`group_service.go` — **订阅分组（不是 AI 账号分组，必须保留）**
- `user_group_rate.go`、`user_group_rate_resolver.go` — 用户分组费率
- `user_subscription.go`、`user_subscription_port.go` — 用户订阅
- `user_attribute.go`、`user_attribute_service.go` — 用户属性
- `user_msg_queue_service.go`、`user_msg_queue_helper.go` — 用户消息队列
- `user_rpm_cache.go` — 用户 RPM 缓存
- `user.go` — 用户模型
- `pricing_service.go`、`model_pricing_resolver.go` — **订阅套餐定价（不是 AI 模型定价，需要保留定价框架）**
- `payment_amounts.go`、`payment_config_limits.go`、`payment_config_plans.go`、`payment_config_providers.go`、`payment_config_service.go`、`payment_currency.go`、`payment_fulfillment.go`、`payment_order.go`、`payment_order_expiry_service.go`、`payment_order_lifecycle.go`、`payment_order_provider_snapshot.go`、`payment_refund.go`、`payment_resume_lookup.go`、`payment_resume_service.go`、`payment_stats.go`、`payment_visible_method_instances.go`、`payment_webhook_provider.go` — 支付全套
- `promo_code.go`、`promo_code_repository.go`、`promo_service.go` — 优惠码
- `parse_integral_number_unit.go`、`slice_helpers.go`、`sql_errors.go`、`domain_constants.go`、`header_util.go`、`metadata_userid.go`、`request_metadata.go` — 通用工具

**后端 Ent Schema（`backend/ent/schema/`）：**
- `user.go` — 用户表
- `auth_identity.go`、`auth_identity_channel.go` — OAuth 身份
- `api_key.go` — 用户认证 Key
- `payment_order.go`、`payment_audit_log.go`、`payment_provider_instance.go` — 支付
- `subscription_plan.go`、`user_subscription.go` — 订阅
- `redeem_code.go`、`promo_code.go`、`promo_code_usage.go` — 兑换/优惠
- `announcement.go`、`announcement_read.go` — 公告
- `setting.go`、`security_secret.go` — 设置/密钥
- `usage_log.go` — 用量日志（后续改为社交操作日志）
- `user_allowed_group.go` — 用户可用订阅分组
- `user_attribute_definition.go`、`user_attribute_value.go` — 用户属性
- `pending_auth_session.go`、`identity_adoption_decision.go` — OAuth 会话
- `usage_cleanup_task.go` — 清理任务

**前端页面（`frontend/src/views/`）：**
- `user/DashboardView.vue`、`user/ProfileView.vue`、`user/UsageView.vue`
- `user/SubscriptionsView.vue`、`user/PaymentView.vue`、`user/UserOrdersView.vue`
- `user/RedeemView.vue`、`user/AffiliateView.vue`
- `user/PaymentQRCodeView.vue`、`user/PaymentResultView.vue`、`user/StripePaymentView.vue`、`user/AirwallexPaymentView.vue`
- `admin/DashboardView.vue`、`admin/UsersView.vue`、`admin/SubscriptionsView.vue`
- `admin/AccountOnboardingView.vue`、`admin/ProxiesView.vue`
- `admin/AnnouncementsView.vue`、`admin/RedeemView.vue`、`admin/PromoCodesView.vue`
- `admin/SettingsView.vue`、`admin/RiskControlView.vue`、`admin/BackupView.vue`
- `admin/UsageView.vue`
- `admin/affiliates/`、`admin/orders/`、`admin/settings/` 目录下所有文件
- 所有 `auth/` 下的认证页面
- 所有 `setup/` 下的设置向导页面

**前端兼容存根 API（禁止删除，也禁止添加真实调用）：**
- `frontend/src/api/admin/groups.ts` — 返回空数组的存根，供 UsersView 等编译通过
- `frontend/src/api/admin/accounts.ts` — 返回空数组的存根

**前端 SocialOps 管理端 API：**
- `frontend/src/api/admin/proxies.ts` — 管理端执行代理 API，调用 `/api/v1/admin/proxies`；不得恢复 AI 网关代理/provider/channel/model 语义

---

## 三、Phase 2 待删除内容（后端清理）

> **执行前提：必须先确保 `go build` 通过，再逐步删除。每删一批就重新编译验证。**

### 3.1 可以安全删除的后端 Service 文件

以下文件是纯 AI 网关功能，与通用平台无任何依赖关系，可以直接删除：

**AI 网关核心：**
- `gateway_service.go`、`gateway_request.go`、`gateway_billing_block.go`、`gateway_billing_header.go`
- `gateway_forward_as_chat_completions.go`、`gateway_forward_as_responses.go`
- `gateway_messages_cache.go`、`gateway_tool_rewrite.go`、`gateway_websearch_emulation.go`

**OpenAI 全套（约 50 个文件）：**
- 所有 `openai_*.go` 文件（openai_gateway_service、openai_ws_*、openai_messages_*、openai_token_provider 等）

**Anthropic/Claude 全套：**
- `anthropic_session.go`、`claude_token_provider.go`、`claude_code_validator.go`

**Gemini 全套：**
- 所有 `gemini_*.go` 文件（gemini_session、gemini_token_*、gemini_oauth*、gemini_quota 等）
- `geminicli_codeassist.go`

**Antigravity 全套：**
- 所有 `antigravity_*.go` 文件

**Bedrock 全套：**
- `bedrock_request.go`、`bedrock_signer.go`、`bedrock_stream.go`

**渠道管理全套：**
- `channel.go`、`channel_service.go`、`channel_available.go`
- 所有 `channel_monitor_*.go` 文件

**AI 代理管理：**
- `proxy.go`、`proxy_service.go`、`proxy_latency_cache.go`
- `tls_fingerprint_profile_service.go`

**AI 账号管理（注意：不是社交账号）：**
- `account.go`、`account_service.go`、`account_group.go`
- `account_credentials_persistence.go`、`account_credentials_redact.go`
- `account_expiry_service.go`、`account_usage_service.go`
- `account_test_service.go`、`account_stats_pricing.go`

**AI 调度器：**
- `scheduler_cache.go`、`scheduler_events.go`、`scheduler_outbox.go`、`scheduler_snapshot_service.go`
- `timing_wheel_service.go`

**Ops 监控全套（约 30 个文件）：**
- 所有 `ops_*.go` 文件

**AI 限速/并发：**
- `ratelimit_service.go`、`concurrency_service.go`、`model_rate_limit.go`、`session_limit_cache.go`

**AI 幂等：**
- `idempotency.go`、`idempotency_observability.go`、`idempotency_cleanup_service.go`

**AI 错误透传：**
- `error_passthrough_service.go`、`error_passthrough_runtime.go`

**AI Token 刷新：**
- `token_refresher.go`、`token_refresh_service.go`、`token_cache_invalidator.go`、`token_cache_key.go`
- `refresh_token_cache.go`、`refresh_policy.go`

**AI OAuth（账号级，不是用户登录 OAuth）：**
- `oauth_service.go`、`oauth_refresh_api.go`

**AI 用量记录：**
- `usage_record_worker_pool.go`、`usage_billing.go`、`usage_log_create_result.go`

**AI 图像计费：**
- `image_billing_multiplier.go`、`image_billing_size.go`
- `image_generation_intent.go`、`image_output_accounting.go`
- `codex_image_generation_bridge.go`

**其他 AI 专用：**
- `crs_sync_service.go`、`scheduled_test_runner_service.go`、`scheduled_test_service.go`、`scheduled_test_port.go`
- `websearch_config.go`、`upstream_models.go`、`upstream_response_limit.go`
- `sse_scanner_buffer_pool.go`、`http_upstream_port.go`
- `digest_session_store.go`、`internal500_counter.go`
- `openai_403_counter.go`、`rpm_cache.go`
- `temp_unsched.go`、`overload_cooldown_test.go`
- `vertex_service_account.go`
- `bedrock_signer.go`、`billing_cache_port.go`

### 3.2 需要谨慎处理的文件（不能直接删除）

| 文件 | 问题 | 处理方式 |
|------|------|----------|
| `billing_service.go` | 同时服务于支付定价和 AI Token 计费 | 保留支付相关函数，删除 Token 计费函数 |
| `billing_cache_service.go` | 同上 | 评估是否还被支付系统引用，若无则删除 |
| `subscription_service.go` | 含 AI 配额检查逻辑（每日/每周 Token 用量） | 删除 AI 配额检查部分，保留订阅生命周期管理 |
| `usage_service.go` | 当前记录 AI Token 用量，后续改为社交操作统计 | 保留框架，替换字段含义 |
| `usage_log.go`（service） | 同上 | 保留，后续重构 |
| `group_service.go` | 订阅分组管理，与 AI 账号分组同名但不同用途 | **必须保留** |
| `pricing_service.go` | 订阅套餐定价，与 AI 模型定价混用 | 保留套餐定价部分 |
| `api_key_service.go` | 用户认证 Key，不是 AI 调用 Key | **必须完整保留** |

### 3.3 可以安全删除的 Ent Schema

- `account.go` — AI 账号凭据
- `account_group.go` — AI 账号分组关联
- `group.go` — AI 账号分组（**注意：`group_service.go` 对应的是订阅分组，schema 里的 `group.go` 是 AI 账号分组，两者不同**）
- `proxy.go` — AI HTTP 代理
- `channel_monitor.go`、`channel_monitor_history.go`、`channel_monitor_daily_rollup.go`、`channel_monitor_request_template.go`
- `error_passthrough_rule.go`
- `tls_fingerprint_profile.go`
- `idempotency_record.go`

### 3.4 可以安全删除的 Handler 文件

**`internal/handler/` 根目录：**
- `gateway_handler.go`、`gateway_handler_chat_completions.go`、`gateway_handler_responses.go`
- `gateway_helper.go`
- `openai_gateway_handler.go`、`openai_chat_completions.go`、`openai_images.go`
- `gemini_v1beta_handler.go`
- `available_channel_handler.go`、`channel_monitor_user_handler.go`
- `failover_loop.go`、`idempotency_helper.go`、`image_concurrency_limiter.go`
- `request_body_limit.go`、`ops_error_logger.go`

**`internal/handler/admin/` 目录：**
- `account_handler.go`、`account_data.go`、`account_codex_import.go`、`account_today_stats_cache.go`
- `channel_handler.go`、`channel_monitor_handler.go`、`channel_monitor_template_handler.go`
- `proxy_handler.go`、`proxy_data.go`
- `openai_oauth_handler.go`、`gemini_oauth_handler.go`、`antigravity_oauth_handler.go`
- `error_passthrough_handler.go`、`tls_fingerprint_profile_handler.go`
- `ops_handler.go`、`ops_dashboard_handler.go`、`ops_realtime_handler.go`
- `ops_alerts_handler.go`、`ops_settings_handler.go`、`ops_system_log_handler.go`
- `ops_ws_handler.go`、`ops_snapshot_v2_handler.go`
- `scheduled_test_handler.go`

### 3.5 需要修改的路由文件

**`internal/server/routes/gateway.go`** — 整个文件删除（AI 网关路由）

**`internal/handler/endpoint.go`** — 删除以下路由注册：
- `/v1/messages`、`/v1/chat/completions`、`/v1/responses`
- `/v1/images/generations`、`/v1/images/edits`
- `/v1beta/models`、`/v1beta/*`
- `/antigravity/*`、`/sora/*`
- `/api/v1/admin/accounts`（AI 账号）、`/api/v1/admin/groups`（AI 分组）
- `/api/v1/admin/channels`、`/api/v1/admin/proxies`
- `/api/v1/admin/ops/*`
- `/api/v1/available-channels`

### 3.6 删除后必须执行的命令

```bash
cd backend
go generate ./ent          # 重新生成 Ent ORM
go generate ./cmd/server   # 重新生成 Wire DI
go build ./...             # 验证编译通过
```

---

## 四、Phase 3 待新增/对齐内容（社交功能实现）

> **Phase 2 完成并编译通过后再开始 Phase 3。**
>
> 若当前仓库已经存在部分 SocialOps Schema/API/页面，Phase 3 的目标是**对齐和修复现有实现**，不是重复新增一套平行模型。

### 4.1 需要新增或对齐的 Ent Schema

```
social_account.go       — 社交账号表
  字段：id, name, platform(x/instagram/tiktok), account_id, password,
        phone, email, email_password, bound_ip/default_proxy, status, assigned_user_id,
        source(registered/manual/file), task_status, task_message,
        remark, created_at, updated_at, deleted_at
  规则：该表代表网站总账号池；assigned_user_id 为空表示未分配，非空表示已分配给用户。
        出售/出租不需要独立状态；账号密码/邮箱密码不额外应用层加密或脱敏。
        若当前实现已有凭据加密、json 隐藏或导出空密码等逻辑，Phase 3 必须对齐为用户可查看/导出完整数据。
        name 是平台用户名匹配字段，例如 X/Twitter 的 @xxx；不要用邮箱、手机号或 account_id 做用户导入匹配。
        用户名匹配前做最小规范化：trim；X/Twitter 接受 xxx 或 @xxx，统一按 @xxx 比较，且大小写不敏感。
        管理员导入去重优先按规范化后的 platform + name 判断，同一平台同一用户名不创建第二条。
        bound_ip/default_proxy 表示该账号的默认执行代理/代理快照，供后续执行器优先调用；
        当前可短期复用 bound_ip，后续推荐收敛为 default_proxy_id 或 default_proxy_snapshot；
        它引用用户独有代理资源，不能跨用户使用；账号回收时必须清空该字段。

social_task_log.go      — 任务执行日志
  字段：id, account_id, user_id, action(follow/dm/tweet/like/login_check),
        target, content, status(pending/running/success/failed),
        result_message, price, charged_amount, charge_status, charge_source,
        proxy_id/proxy_snapshot, billing_request_id/idempotency_key,
        executed_at, created_at
  规则：任务日志沿用当前/原项目日志风格；所有真实社交执行动作内置单价 0.1，
        成功才最终扣费，非成功不最终扣费。charge_status 至少能区分未扣费、
        已预扣、已确认、已退回/已释放、失败不扣。不要新建独立复杂账本，
        钱包和订阅扣费仍复用原项目逻辑。批量任务按账号 + 动作计费，
        成功几个账号扣几次；失败账号不扣费。billing_request_id/idempotency_key
        用于避免重复提交和重复扣费。

social_ip.go            — 用户 IP/代理资源表
  字段：id, user_id, name, type(residential/static), endpoint,
        status, latency_ms, last_check_at, remark, created_at
  规则：IP/代理属于用户独有资源，是执行任务时的网络出口/代理凭据；只能由所属用户使用。
        同一用户自己的一个 IP/代理可用于自己名下多个账号，不能跨用户使用。
        若当前实现已有 bound_account_id/bound_social_account_id 字段，后续不应把它作为账号所有权状态。
        账号表可以保存默认执行代理字段，用于减少随机代理差异并方便执行器调用；
        任务执行时优先使用请求显式选择的代理，其次使用账号默认执行代理，再按后续实现决定是否自动选择。
        账号回收时必须清空账号上的默认执行代理/代理快照。
        用户删除或更新代理时，后续实现按最小安全规则处理：已保存到账号上的默认代理若失效，
        执行任务应失败关闭且不扣费；是否自动改用其他代理留给真实执行器后续实现。
        IP 测试免费，不进入执行计费。
```

### 4.2 需要新增或对齐的后端 API

```
# 管理端
POST   /api/v1/admin/social-accounts/import     — 批量导入社交账号
POST   /api/v1/admin/social-accounts/register   — 批量注册社交账号；真实注册器未接入时返回 not configured，不入池
GET    /api/v1/admin/social-accounts            — 账号工作台列表
GET    /api/v1/admin/social-accounts/:id        — 账号详情
PUT    /api/v1/admin/social-accounts/:id        — 更新账号信息
DELETE /api/v1/admin/social-accounts/:id        — 删除单个账号
POST   /api/v1/admin/social-accounts/batch-delete — 批量删除账号
POST   /api/v1/admin/social-accounts/:id/assign — 分配账号给用户
POST   /api/v1/admin/social-accounts/:id/reclaim — 回收账号
PUT    /api/v1/admin/social-accounts/:id/default-proxy — 设置或清空默认执行代理
GET    /api/v1/admin/social-accounts/export     — 导出账号池完整数据
GET    /api/v1/admin/proxies                    — 执行代理列表，可按用户/状态/类型搜索
POST   /api/v1/admin/proxies                    — 新增用户独有执行代理
PUT    /api/v1/admin/proxies/:id                — 更新执行代理
DELETE /api/v1/admin/proxies/:id                — 删除执行代理
POST   /api/v1/admin/proxies/:id/test           — 测试 IP/代理连通性（免费）

# 用户端
GET    /api/v1/social-accounts                  — 我的社交账号列表
POST   /api/v1/social-accounts/import           — 按平台用户名（如 X 的 @xxx）匹配总账号池并自动绑定未分配账号；多命中时拒绝为歧义匹配
GET    /api/v1/social-accounts/export           — 导出我的账号完整数据，默认 CSV
POST   /api/v1/social-accounts/tasks            — 提交批量任务
GET    /api/v1/social-accounts/tasks            — 任务执行记录
PUT    /api/v1/social-accounts/:id/default-proxy — 设置账号默认执行代理（若保留用户端接口，必须校验账号和代理同属当前用户）
```

#### 4.2.1 API 业务约束

- 用户提交执行任务时，账号必须存在于 `social_accounts` 且 `assigned_user_id` 为当前用户。
- 用户提交执行任务时，账号状态必须允许执行；当前最小准入规则是只有 `available` 允许执行，其他状态失败关闭且不扣款。
- 用户提交执行任务时，若显式选择 IP/代理，该 IP/代理必须属于当前用户；若未显式选择，则可以使用账号字段中的默认执行代理。默认执行代理用于稳定调用，不是账号所有权状态；账号回收时必须清空，避免下一位用户继续使用原用户代理。
- 管理员通过 `/api/v1/admin/proxies` 维护用户独有代理；管理员代用户设置默认执行代理时，代理也必须属于该账号当前所属用户；未分配账号不能保留默认执行代理。用户端不暴露 `/api/v1/social-ips`。
- 用户导入账号时，只按平台用户名匹配总账号池，例如 X 的 `@xxx`，对应账号 `name` 字段；不按邮箱、手机号或平台账号 ID 匹配。匹配前 trim；X/Twitter 允许 `xxx` 或 `@xxx`，统一按 `@xxx` 且大小写不敏感匹配。若导入数据带平台，则按 `platform + name` 匹配；若未带平台且命中多条，则拒绝为歧义匹配。匹配不到不新增账号；匹配到已分配账号则拒绝；匹配到未分配账号则自动分配给当前用户。
- 管理员导入账号时进入总账号池，重复账号按当前已有导入去重逻辑处理；社交账号去重优先按规范化后的 `platform + name` 判断，同一平台同一用户名不创建第二条。
- 管理员注册账号是真实社交平台注册流程的入口；真实注册执行器可后续接入。当前真实注册器未接入时可以保留路由，但必须返回 not configured/失败关闭，不创建账号；注册失败不入池。
- 真实社交执行动作价格统一为 `0.1`，扣费/预扣/退款复用原项目订阅与钱包逻辑；不修改原项目订阅套餐、钱包余额、配额窗口、重置周期和基础扣费语义。账号导入/导出、分配/回收、管理端 IP/代理新增/修改/删除/测试、设置账号默认执行代理不收费。
- `edit_ip`、`set_default_proxy`、`change_proxy` 等代理配置动作不属于真实社交执行动作，不进入执行计费；应作为免费配置接口或默认执行代理设置接口处理。
- 执行提交流程建议为：准入校验 → 生成任务日志 → 按原项目逻辑预扣或记录待扣 → 排队/执行 → 成功确认扣费并写 usage/任务日志 → 非成功按原项目逻辑退回/释放预扣并写失败日志。
- 非成功任务不最终扣费；若沿用预扣费，失败、异常、权限失败、参数错误、账号状态失败、IP 不可用、真实执行器未接入等结果必须退回预扣或保持未扣费。
- 批量任务按账号 + 动作计费；一批多个账号执行同一动作，成功几个扣几次，失败账号不扣。
- 同一任务/请求需要幂等，避免重复提交或重试造成重复扣费。请求需要支持 `client_request_id` 或等价幂等键；建议按 `user_id + client_request_id + account_id + action` 识别重复任务，重复提交返回已有任务/日志结果。
- 用户导出完整数据默认 CSV，至少包含 platform、name、account_id、password、phone、email、email_password、bound_ip/default_proxy、account_status、task_status、source、remark、created_at、updated_at。
- 当前阶段不实现真实执行器逻辑，只保留可替换的执行入口；未接入执行器时必须失败关闭。

### 4.3 前端需要接入真实 API 的页面

当前阶段管理端“社交账号”按备份版 UI 恢复为两个清晰入口：`/admin/accounts` 负责账号管理/导入/注册入口/详情整理，`/admin/total-accounts` 负责总账号池分配状态、分配、回收、默认执行代理和批量清理。两者都必须使用当前 SocialOps 的 `/api/v1/admin/social-accounts` 后端能力，不得恢复原 AI 网关 `/admin/accounts` 语义。不要恢复用户端 `/account-management`、`/account-functions`、`/ip-management` 三套重复页面；代理维护入口仍统一在管理端 `/admin/proxies`。

| 页面 | 当前状态 | 需要对接的 API |
|------|----------|----------------|
| `admin/AccountOnboardingView.vue` | 账号管理：导入、注册入口、编辑、详情和执行失败关闭日志 | `GET/POST/PUT/DELETE /api/v1/admin/social-accounts`、import/register/export；执行器未接入时失败关闭且不扣费 |
| `admin/TotalAccountsView.vue` | 总账号池：查看分配状态、分配、回收、默认执行代理、导出和批量清理 | `GET/PUT/DELETE /api/v1/admin/social-accounts`、assign/reclaim/default-proxy/export |
| `admin/ProxiesView.vue` | 管理端执行代理入口 | `GET/POST/PUT/DELETE /api/v1/admin/proxies`、`POST /api/v1/admin/proxies/:id/test` |
| 后续唯一用户端账号入口 | 待产品收敛 | `GET /api/v1/social-accounts`、import/export/tasks；不得同时恢复账号管理和账号功能两套页面 |

> 注意：`frontend/src/api/admin/groups.ts`、`frontend/src/api/admin/accounts.ts` 是保留视图的兼容存根，禁止在这些文件里接真实社交账号 API。`frontend/src/api/admin/proxies.ts` 是 SocialOps 管理端执行代理 API，不得恢复 AI 网关代理语义。

### 4.4 需要调整的现有页面

| 页面 | 调整内容 |
|------|----------|
| `admin/DashboardView.vue` | 替换 Token/RPM 统计为：社交账号总数、今日任务执行次数、活跃用户数、订阅收入 |
| `admin/UsageView.vue` | 替换 Token/模型/计费倍率字段为：操作类型、账号、执行状态、执行时间 |
| `admin/SettingsView.vue` | 删除 gateway 标签页（Rectifier、Beta Policy、OpenAI Fast Policy 等 AI 配置） |
| `user/UsageView.vue` | 替换 Token 用量为社交操作次数统计 |

---

## 五、前端遗留清理（可随时执行）

以下文件已无任何引用，可以直接删除：

```
frontend/src/views/admin/groupsMessagesDispatch.ts
frontend/src/views/admin/groupsSupportedModelScopes.ts
frontend/src/views/admin/__tests__/groupsMessagesDispatch.spec.ts
frontend/src/views/admin/__tests__/groupsSupportedModelScopes.spec.ts
```

---

## 六、重要概念区分（防止混淆）

以下概念在代码中同名但含义不同，**绝对不能混淆**：

| 概念 | AI 网关版本（删除） | 社交平台版本（保留/新增） |
|------|---------------------|--------------------------|
| Account | AI 平台账号凭据（Claude/OpenAI/Gemini 的 OAuth token 或 API Key） | 社交媒体账号（X/Instagram 的用户名密码） |
| Group | AI 账号分组（用于负载均衡和模型路由） | 订阅分组（用于区分不同套餐的用户权限） |
| API Key | 用户调用 AI 的密钥（`/v1/messages` 鉴权用） | 用户调用本平台 API 的密钥（`/api/v1` 鉴权用，**必须保留**） |
| Proxy | AI 账号的 HTTP 代理（用于绕过 IP 限制访问 OpenAI 等） | 用户独有的 IP/代理执行资源，由管理端 `/admin/proxies` 管理；执行时选择/使用，可保存为账号默认执行代理，账号回收时清空 |
| Usage | AI Token 用量（input_tokens、output_tokens、cost） | 社交操作次数（follow、dm、tweet、like 的执行记录） |
| Billing | AI Token 计费（按 Token 数 × 单价 × 倍率） | 真实社交执行动作按次计费，统一单价 0.1，优先扣订阅金额额度，不足扣钱包 |
| Subscription | AI 调用配额/Token 窗口 | 真实社交执行动作可消耗的金额额度，沿用原项目订阅/钱包扣费逻辑 |

---

## 七、技术栈和开发规范

### 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26.3, Gin, Ent ORM, Wire DI |
| 前端 | Vue 3.4+, Vite 5, TypeScript, TailwindCSS, Pinia |
| 数据库 | PostgreSQL 15+ |
| 缓存 | Redis 7+ |
| 容器 | Docker + Docker Compose |

### 开发规范

- 后端 Go 命令在 `backend/` 目录运行
- 前端只用 pnpm，不用 npm/yarn
- 前端构建输出到 `backend/internal/web/dist`（不是本地 dist）
- 生产后端必须用 `-tags embed` 构建才能服务前端
- 修改 Ent schema 后必须运行 `go generate ./ent`
- 修改 Wire providers 后必须运行 `go generate ./cmd/server`
- 不要手动编辑 `wire_gen.go` 和 `ent/` 下的生成文件

### Docker 本地构建（网络受限环境）

```bash
cd deploy
docker compose -f docker-compose.dev.yml build \
  --build-arg NODE_IMAGE=node:22-alpine \
  --build-arg GOLANG_IMAGE=docker.m.daocloud.io/library/golang:1.26.3-alpine \
  --build-arg ALPINE_IMAGE=alpine:latest \
  --build-arg POSTGRES_IMAGE=postgres:18-alpine \
  --build-arg GOPROXY=https://goproxy.cn,direct \
  --build-arg GOSUMDB=sum.golang.google.cn
docker compose -f docker-compose.dev.yml up -d --no-build
```

---

## 八、当前已知问题

1. **真实注册器未接入**：管理员注册入口必须继续失败关闭，真实平台注册成功前不得创建总池账号。

2. **真实执行器未接入**：登录检测、关注、私信、发帖、点赞等真实社交动作当前只保留提交、校验、日志和计费挂钩；未接入执行器时必须失败关闭且不最终扣费。

3. **前端存根 API 必须保留**：`api/admin/groups.ts`、`accounts.ts` 返回空数组，供保留的用户/订阅/兑换等通用后台视图编译通过。不要在这两个文件中添加真实社交账号调用。`api/admin/proxies.ts` 是 SocialOps 管理端执行代理真实 API，不是存根。

4. **默认执行代理字段仍是过渡表达**：当前可短期复用 `bound_ip` 保存默认执行代理快照；后续如需更强约束，可迁移到显式 `default_proxy_id/default_proxy_snapshot`，迁移必须保持回收账号清空代理字段。

5. **最终验收已完成**：G001-G008、全量构建/容器/smoke、ai-slop-cleaner no-op 检查、独立 code-review 和 UltraQA/等价敌意验证已通过；后续改动仍必须重新按本指南和验收清单验证。
