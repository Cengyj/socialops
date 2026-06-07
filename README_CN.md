# SocialOps

> ⚠️ **开发者/AI 助手必读：** 本项目正在从 AI 网关平台改造为基于网站总账号池的社交账号运营平台。修改代码前请先阅读 **[PROJECT_GUIDE.md](PROJECT_GUIDE.md)**，其中详细说明了哪些文件必须保留、哪些待删除、哪些待新增，以及当前业务目标。

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**网站总账号池 + 社交账号执行计费平台**

[English](README.md) | 中文

</div>

---

## 项目概述

SocialOps 是一个基于网站总账号池的社交账号运营平台。系统只处理总账号池已有账号：管理员导入或真实注册账号进入账号池，管理员可手动分配/回收账号；用户导入账号时只按平台用户名匹配池内未分配账号（如 X 的 `@xxx`），匹配成功后自动绑定。用户可使用名下账号提交关注、私信、发帖、点赞、登录检测等真实社交执行任务；当前阶段保留提交、校验、排队、日志和计费框架，真实执行器后续接入，只有执行成功才最终扣费。

## 核心功能

- **网站总账号池** — 只处理池内已有账号；非池内账号不录入、不绑定、不执行、不计费
- **社交账号管理** — 管理员维护账号池，支持真实注册、手动导入、文件批量入库和导入去重
- **账号分配** — 管理员可将未分配账号分配给用户，支持回收后回到未分配状态
- **用户导入匹配** — 用户导入账号只按平台用户名匹配池内未分配账号，例如 X 的 `@xxx`；同一用户名多命中时拒绝为歧义匹配，成功后自动绑定
- **任务执行** — 用户可提交关注、私信、发帖、点赞、登录检测等任务，真实执行器后续接入，未接入时失败关闭
- **IP/代理管理** — 用户独有的 IP/代理执行资源统一由 `/proxies` 维护；包括管理员在内的所有角色都只能管理自己的代理。账号可设置已测试在线的默认执行代理，账号回收或代理删除会清空默认代理，IP 测试免费
- **执行计费** — 真实社交执行动作统一单价 `0.1`，优先扣订阅金额额度，不足再扣钱包，成功才最终扣费，非成功退回/不扣；订阅/钱包基础逻辑沿用原项目，不重做；账号/IP 管理动作不收费
- **内置支付系统** — 支持 EasyPay 易支付、支付宝官方、微信官方、Stripe，用户自助充值
- **用户管理** — 完整的注册/登录/2FA/OAuth 登录体系
- **管理后台** — Web 界面进行监控和管理

## 明确不做

- 不恢复 AI 网关能力，包括模型渠道、AI 账号池、Token 计费、AI OAuth 和网关监控。
- 当前阶段不先实现账号购买流程；管理员分配和用户导入匹配是账号归属的主要入口。
- 当前阶段不实现真实社交平台执行器，只保留可替换的执行入口和失败关闭框架。
- 不区分出售/出租账号状态，出售/出租只是业务表达。
- 不重新设计订阅、钱包、导入去重、日志体系，优先复用原项目逻辑。
- 不对用户自己名下账号的密码、邮箱密码等完整数据做额外脱敏或应用层加密。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.26.3, Gin, Ent |
| 前端 | Vue 3.4+, Vite 5+, TailwindCSS |
| 数据库 | PostgreSQL 15+ |
| 缓存/队列 | Redis 7+ |

---

## 部署方式

### 方式一：Docker Compose（推荐）

```bash
# 创建部署目录
mkdir -p socialops-deploy && cd socialops-deploy

# 下载并运行部署准备脚本
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/socialops/main/deploy/docker-deploy.sh | bash

# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f socialops
```

访问 `http://你的服务器IP:8080`

### 方式二：脚本安装

```bash
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/socialops/main/deploy/install.sh | sudo bash
```

安装后访问 `http://你的服务器IP:8080` 完成设置向导。

### 方式三：源码编译

```bash
# 前置条件：Go 1.26.3, Node.js 20+, pnpm 9, PostgreSQL 15+, Redis 7+

# 编译前端
cd frontend && pnpm install && pnpm run build

# 编译后端（嵌入前端）
cd ../backend && go build -tags embed -o socialops ./cmd/server

# 运行
./socialops
```

---

## 配置说明

复制 `deploy/config.example.yaml` 为 `config.yaml`，关键配置项：

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "socialops"

redis:
  host: "localhost"
  port: 6379

jwt:
  secret: "change-this-to-a-secure-random-string"
  expire_hour: 24

default:
  user_balance: 0
```

---

## 项目结构

```
socialops/
├── backend/                  # Go 后端服务
│   ├── cmd/server/           # 应用入口
│   ├── internal/             # 内部模块
│   │   ├── config/           # 配置管理
│   │   ├── service/          # 业务逻辑
│   │   ├── handler/          # HTTP 处理器
│   │   └── repository/       # 数据访问层
│   └── ent/schema/           # 数据库 Schema
│
├── frontend/                 # Vue 3 前端
│   └── src/
│       ├── api/              # API 调用
│       ├── stores/           # 状态管理
│       ├── views/            # 页面组件
│       └── components/       # 通用组件
│
└── deploy/                   # 部署文件
    ├── docker-compose.yml
    ├── config.example.yaml
    └── install.sh
```

---

## 许可证

本项目基于 [GNU 宽通用公共许可证 v3.0](LICENSE)（或更高版本）授权。

Copyright (c) 2026 Wesley Liddick
