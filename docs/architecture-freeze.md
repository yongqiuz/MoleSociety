# 当前架构冻结说明（Phase 0）

本文档用于冻结 `MoleSociety` 当前原型阶段的接口、数据模型与字段语义，作为后续 Phase 1~10 演进的统一基线。

## 1. 当前阶段定位

当前项目状态定义为：

- **真实可运行的社交原型**
- **内存状态 + Redis 快照持久化**
- **存储与存证语义已进入主模型，但尚未完成真实外部闭环**
- **旧 Relay / Mint 业务仍作为兼容层保留**

当前系统已经真实落地：

- 用户模型
- 认证与 Session 原型
- 帖子 / 回复线程
- 媒体资产元数据
- 会话 / 消息
- 联邦实例概念

当前尚未完整落地：

- PostgreSQL 主数据库
- 真实对象存储上传
- 真实去中心化存储 pin / upload
- 真实链上 attestation 流程
- WebSocket / SSE 实时消息
- ActivityPub 联邦互通

---

## 2. 架构基线

### 2.1 前端

- Vue 3
- Vite
- TypeScript
- Tailwind CSS
- `fetch` API 请求
- `ethers` 作为链上扩展能力预留

主要职责：

- 登录态维护
- 页面展示
- 调用社交 / 认证 API
- 草稿、本地回退与交互控制

### 2.2 后端

- Java 17+
- Spring Boot 3.5.x
- Maven
- PostgreSQL JDBC 主存储
- Redis session / challenge / snapshot fallback
- 内存 fallback
- Relay / RPC 兼容接口

主要职责：

- 提供 HTTP API
- 管理社交领域状态
- 管理认证挑战、会话、账户
- 保留旧 Relay 接口

### 2.3 当前持久化方式

当前不是数据库驱动，而是：

- 内存结构体数组 / map 为主
- Redis 作为快照持久化补充
- Redis 不可用时退化为纯内存模式

这一定义在 Phase 0 固定，不在当前阶段将 Redis 误描述为主数据库。

---

## 3. API 冻结清单

## 3.1 稳定接口（允许内部重构，不允许随意改语义）

### 健康检查

- `GET /healthz`

### 社交接口

- `GET /api/v1/social/bootstrap`
- `GET /api/v1/social/instances`
- `GET /api/v1/social/users`
- `POST /api/v1/social/users`
- `GET /api/v1/social/users/{id}`
- `PATCH /api/v1/social/users/{id}`
- `POST /api/v1/social/users/{id}/follow`
- `DELETE /api/v1/social/users/{id}/follow`
- `GET /api/v1/social/feed`
- `POST /api/v1/social/posts`
- `POST /api/v1/social/posts/{id}/poll/vote`
- `GET /api/v1/social/posts/{id}`
- `GET /api/v1/social/posts/{id}/thread`
- `GET /api/v1/social/posts/{id}/replies`
- `GET /api/v1/social/media`
- `POST /api/v1/social/media`
- `GET /api/v1/social/conversations`
- `POST /api/v1/social/conversations`
- `GET /api/v1/social/conversations/{id}`
- `POST /api/v1/social/conversations/{id}/messages`

### 认证接口

- `POST /api/v1/auth/challenge`
- `POST /api/v1/auth/verify`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/password-login`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/bind-challenge`

## 3.2 兼容接口（保留，但视为 legacy）

- `GET /secret/get-binding`
- `GET /secret/verify`
- `POST /relay/mint`
- `POST /relay/save-code`
- `POST /relay/reward`
- `GET /relay/stats`
- `GET /api/admin/check-access`
- `POST /api/admin/social/reset`
- `GET /api/v1/analytics/distribution`

## 3.3 实验性约定

以下内容虽然已存在于模型或界面中，但仍视为实验性语义，不可对外宣称已完全实现：

- 联邦实例互通
- 链上 attestation 自动生成
- 去中心化存储真实写入
- 端到端加密聊天
- 实时消息推送

---

## 4. 数据模型冻结

## 4.1 用户 `SocialUser`

用途：社交身份实体。

核心字段：

- `id`：系统内用户主键
- `handle`：平台 handle，当前表现为 `@name`
- `displayName`：展示名称
- `bio`：简介
- `instance`：所属实例名
- `wallet`：绑定钱包地址或占位值
- `avatarUrl`：头像地址
- `fields`：扩展资料字段
- `featuredTags`：特色标签
- `isBot`：机器人标记
- `followers` / `following`：聚合计数
- `createdAt`：创建时间

冻结规则：

- `SocialUser` 是社交展示模型，不等于认证账户模型
- 后续数据库化时，允许账户表与社交用户表分离

## 4.2 帖子 `SocialPost`

用途：统一内容模型，帖子和回复共用一个实体。

核心字段：

- `id`
- `authorId`
- `authorHandle`
- `authorName`
- `instance`
- `kind`
- `content`
- `visibility`
- `storageUri`
- `attestationUri`
- `tags`
- `media`
- `parentPostId`
- `rootPostID`
- `replyDepth`
- `replies`
- `boosts`
- `likes`
- `type`
- `interaction`
- `poll`
- `createdAt`

冻结规则：

- 顶层内容与回复统一用 `posts` 模型表达
- `kind=post` 表示顶层帖子
- `kind=reply` 表示回复
- 首页时间线只返回顶层帖子，不混入回复

## 4.3 媒体 `MediaAsset`

用途：媒体资源元数据实体。

核心字段：

- `id`
- `ownerId`
- `name`
- `kind`
- `url`
- `storageUri`
- `cid`
- `sizeBytes`
- `status`
- `createdAt`

冻结规则：

- 当前 `MediaAsset` 只代表媒体元数据，不代表对象存储或去中心化存储已真实完成
- 后续允许新增上传任务、转码任务、派生资源等表，不直接破坏该模型对前端的输出结构

## 4.4 会话 `Conversation` / 消息 `ChatMessage`

用途：私信与会话模型。

冻结规则：

- `encrypted` 当前只作为状态位/能力预留，不视为已完成端到端加密实现
- 当前消息链路为 HTTP 写入 + 轮询/刷新，不视为实时 IM

## 4.5 联邦实例 `FederationInstance`

用途：实例概念建模与前端展示。

冻结规则：

- 当前仅为本地实例目录模型
- 不视为已实现 ActivityPub actor / inbox / outbox 互通

---

## 5. 字段语义冻结

## 5.1 `url`

定义：

- 可直接访问的资源地址，偏向 HTTP/HTTPS 访问路径
- 用于前端预览、展示或下载

Phase 0 语义：

- 可以是外链 URL
- 可以是演示资源 URL
- 不能默认等价于去中心化存储地址

## 5.2 `cid`

定义：

- 内容寻址标识
- 当前主要用于表达 IPFS 类内容地址或兼容性引用

Phase 0 语义：

- 当前可能是真实 CID，也可能是演示占位值
- 不要求系统已经能通过它稳定回源

## 5.3 `storageUri`

定义：

- 内容或媒体的“存储层标识”
- 用于描述资源在某个存储系统中的稳定引用

Phase 0 语义冻结为：

1. 它是**存储标识字段**，不是必须可直接浏览器访问的 URL
2. 它可以是：
   - `ar://...`
   - `ipfs://...`
   - `sha256://...`
   - 其他未来定义的协议型 URI
3. 当前如果帖子未显式提供 `storageUri`，后端会生成 `sha256://...` 本地摘要 URI
4. 当前 `storageUri` 的存在 **不等于** 已完成真实去中心化上传

## 5.4 `attestationUri`

定义：

- 内容或声明对应的“证明 / 存证引用”字段

Phase 0 语义冻结为：

1. 它是证明引用字段，不要求当前一定来源于真实链上回执
2. 当前可以是：
   - 手动传入的 URI
   - 演示数据中的占位 URI
   - 后续真实证明系统返回的引用地址
3. 当前系统中 `attestationUri` **不应被描述为默认真实上链证明**

---

## 6. 旧 Relay 接口保留策略

当前兼容层保留，不删除：

- 作为旧图书确权 / mint 业务残留能力
- 作为未来“内容身份 / 创作者身份 / 资产确权”链上入口候选
- 与新社交主接口并行存在

冻结策略：

- Phase 1 不删除 legacy 接口
- Phase 1~3 不主动重构其业务语义
- 未来若拆分服务，应将其作为独立 legacy / relay 模块迁出

---

## 7. 环境变量分层

## 7.1 当前已使用环境变量

### 服务运行

- `BACKEND_ADDR`
- `APP_ENV`

### Redis

- `REDIS_ADDR`

### RPC / 链上 Relay

- `RPC_URL`
- `CHAIN_ID`
- `RELAYER_COUNT`
- `PRIVATE_KEY_0...N`

### 社交数据

- `SOCIAL_SEED`

### 管理能力

- `ADMIN_WALLETS`

## 7.2 Phase 1 预留环境变量

用于数据库化升级：

- `DATABASE_URL`
- `POSTGRES_MAX_OPEN_CONNS`
- `POSTGRES_MAX_IDLE_CONNS`
- `POSTGRES_CONN_MAX_LIFETIME`
- `DB_MIGRATIONS_DIR`

冻结规则：

- Redis 在 Phase 1 之后转为缓存 / session / 队列辅助角色
- PostgreSQL 成为主持久化数据源

---

## 8. Phase 1 目标边界

Phase 1 的目标不是一次性把全部查询都迁移到数据库，而是完成以下基础设施落地：

1. 引入 PostgreSQL 连接初始化
2. 引入 migration 目录与首版 schema
3. 提供数据库健康检查能力
4. 为 repository / store 层预留清晰入口
5. 在不破坏现有 API 的前提下推进持久化演进

Phase 1 不承诺一次性交付：

- 全量 SQL repository 替换
- 全量业务数据迁移工具
- 高级查询优化
- 实时消息和联邦协议

---

## 9. 后续阶段依赖原则

- 所有后续阶段默认以本文档为语义基线
- 若后续字段语义要变更，必须先更新该冻结文档
- 若某接口要从“稳定”降级或废弃，必须补充迁移说明
