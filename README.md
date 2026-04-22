# MoleSociety

Whale Vault Social 是在原有“实体书确权 + NFT 领取 + 永续存储”项目基础上，重构得到的 **去中心化社交平台原型**。  
当前版本已经从单一的“扫码 Mint”业务，升级为面向创作者社区、阅读社群、媒体归档和联邦协作的社交系统，并保留了区块链中继、永久存储、链上存证等核心叙事。

它的目标不是做一个中心化内容平台的克隆，而是构建一个具备以下特征的社交底座：

- 内容主权：帖子、媒体、会话元数据不依赖单一中心平台
- 身份主权：用户身份可绑定钱包、实例、链上凭证
- 存储主权：媒体和内容可以映射到 Arweave / IPFS / 其他去中心化存储层
- 社交主权：实例之间可演进为联邦互通，而不是只依赖一个站点
- 协议可扩展：当前是 Spring Boot + Vue 原型，后续可平滑演进到 ActivityPub、WebSocket、Matrix、libp2p 等体系

本仓库包含：

- 前端：Vue 3 + Vite + TypeScript + Tailwind CSS
- 后端：Spring Boot + Maven + PostgreSQL JDBC 主存储 + Redis session / snapshot fallback
- 区块链中继：Spring Boot 内置轻量 Relay 兼容接口（当前模拟交易哈希）
- 合约子项目：Foundry + Solidity（保留为链上身份 / 媒体哈希锚定扩展）
- 一键启动脚本：Windows `start-dev.cmd` / `start-dev.ps1`

---

## 4. Phase 1 本地验证与环境说明

### 4.1 推荐本地环境

- PostgreSQL 14+
- Redis（可选，但建议保留用于 session/challenge）
- Node.js / npm
- Java 17+
- Maven 3.9+

### 4.2 最小环境变量

建议在项目根目录准备 `.env`，至少包含：

```env
BACKEND_ADDR=0.0.0.0:8080
REDIS_ADDR=127.0.0.1:6379
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/molesociety?sslmode=disable
POSTGRES_MAX_OPEN_CONNS=10
POSTGRES_MAX_IDLE_CONNS=5
POSTGRES_CONN_MAX_LIFETIME=30m
DB_MIGRATIONS_DIR=./migrations
SOCIAL_SEED=1
```

### 4.3 启动方式

推荐：

```powershell
./start-dev.ps1
```

或：

```bat
start-dev.cmd
```

### 4.4 验证入口

启动后检查：

- 前端：`http://localhost:4173`
- 后端健康检查：`http://127.0.0.1:8080/healthz`

如果 PostgreSQL 可用，健康检查中应看到：

- `database.enabled = true`
- `databaseMode = postgres(jdbc)+redis`
- `migrations.applied` 包含 `0001_initial_schema.sql`

更完整的检查步骤见：

- `docs/phase-1-verification-checklist.md`

## 1. 项目定位

### 1.1 从图书确权到去中心化社交

项目最初围绕“实体书二维码 + 唯一兑换码 + NFT Mint + Arweave 正文解锁”展开，强调：

- 每本书对应唯一数字身份
- 通过 Solidity / EVM 中继保障领取过程不可篡改
- 通过去中心化存储对抗内容被平台删除

在本次改造后，核心理念被进一步抽象为：

- “书”的唯一身份，升级为“用户 / 内容 / 媒体 / 会话”的可验证身份
- “扫码领取”升级为“发帖、上传、会话、联邦分发”的社交流程
- “内容解锁”升级为“内容发布、引用、存证、归档、跨实例传播”

换句话说，Whale Vault Social 不是简单地把原项目 UI 改成社交样式，而是把原来的确权思想推广到了完整的社交系统中。

### 1.2 目标用户

当前系统主要服务四类参与者：

- 普通用户
  - 注册身份
  - 浏览联邦时间线
  - 发布帖子
  - 上传媒体
  - 发起与参与私信会话

- 创作者 / 作者 / 社群运营者
  - 维护个人主页和实例身份
  - 发布图文 / 视频内容
  - 将作品映射到永久存储
  - 通过链上存证强化原创声明

- 实例运营者
  - 维护联邦实例
  - 管理内容索引和服务运行
  - 连接 Redis、对象存储和链上中继服务

- 协议 / 平台方
  - 运行 Spring Boot 中间层
  - 提供历史兼容接口
  - 演进至 ActivityPub、实时消息、内容存储调度等协议层能力

### 1.3 当前版本状态

当前仓库是一个 **可运行的原型系统**，已经具备：

- Vue 前端界面
- Spring Boot 社交 API
- PostgreSQL 主持久化路径（Phase 1）
- Redis 会话 / 回退辅助能力
- 媒体、帖子、用户、会话、消息的数据模型
- 一键启动和本地联调能力

但需要明确：

- 现在的聊天仍是 HTTP 驱动的消息写入，不是 WebSocket 实时推送
- 现在的媒体上传是前端预览数据 + 后端记录元数据，不是真实对象存储流水线
- 现在的联邦实例是平台内建模型，不是完整 ActivityPub 联邦网络
- 现在已完成 Phase 1 的数据库化，但还不是生产级横向扩展架构

---

## 2. 功能总览

### 2.1 社交时间线

前端首页已经改造成去中心化社交平台的时间线视图，支持：

- 浏览联邦时间线
- 展示作者名、Handle、实例、发布时间
- 展示帖子正文、标签、存证引用
- 展示帖子对应的媒体资源
- 展示点赞、转发、回复等基础计数

当前时间线通过以下后端接口驱动：

- `GET /api/v1/social/bootstrap`
- `GET /api/v1/social/feed`

### 2.2 媒体上传与永续资源视图

系统支持前端上传媒体资源，并将其注册为媒体资产实体。当前版本能力包括：

- 前端读取用户选择的图片 / 视频文件
- 生成预览
- 调用后端 `POST /api/v1/social/media`
- 将媒体与帖子实体建立关联
- 在“永续存储”视图中展示存储 URI、CID、状态、大小等信息

当前是“媒体元数据驱动”的实现，便于后续接入：

- Arweave
- IPFS
- Pinata
- Crust
- 其他对象存储或内容寻址网络

### 2.3 会话聊天

系统已经支持基础会话模型：

- 会话列表
- 会话标题
- 参与者集合
- 端到端标识位
- 消息列表
- 消息追加写入

当前相关接口：

- `GET /api/v1/social/conversations`
- `POST /api/v1/social/conversations`
- `GET /api/v1/social/conversations/{id}`
- `POST /api/v1/social/conversations/{id}/messages`

这为后续升级到 WebSocket / Matrix / libp2p 奠定了清晰的数据边界。

### 2.4 联邦实例

系统已内建联邦实例概念：

- 实例名
- 实例关注方向
- 成员规模
- 延迟
- 健康状态

这部分当前用于：

- 构建 Mastodon 风格的“多实例”叙事
- 为后续的联邦发现、实例路由、实例级治理做准备

当前接口：

- `GET /api/v1/social/instances`

### 2.5 区块链兼容与旧业务兼容

虽然项目已转向去中心化社交平台，但后端仍保留了原有图书确权项目的一组兼容接口：

- `GET /secret/get-binding`
- `GET /secret/verify`
- `POST /relay/mint`
- `POST /relay/save-code`
- `POST /relay/reward`
- `GET /relay/stats`
- `GET /api/admin/check-access`
- `GET /api/v1/analytics/distribution`

这意味着：

- 原有“扫码 Mint”模式没有被粗暴删除
- 老业务可以作为未来的“链上身份入口”继续存在
- 社交平台与链上确权逻辑可以长期共存

---

## 3. 技术栈与工程结构

### 3.1 前端

当前前端技术栈：

- Vue 3
- Vite
- TypeScript
- Tailwind CSS
- 原生 `fetch` API
- `ethers`（保留链上调用扩展能力）

关键文件：

- `frontend/src/App.vue`
  - 主界面
  - 时间线、发现、私信、永续存储、联邦实例等多个 section
  - 启动时调用 bootstrap 接口

- `frontend/src/api/socialApi.ts`
  - 前端与 Spring Boot 后端的 API 适配层
  - 负责请求封装、类型定义和统一错误处理

- `frontend/src/main.ts`
  - Vue 入口

### 3.2 后端

当前后端技术栈：

- Java 17+
- Spring Boot 3.5.x
- Maven
- PostgreSQL JDBC 持久化
- Redis session / challenge / snapshot fallback
- Cookie Session

关键文件：

- `backend/src/main/java/com/molesociety/backend/MoleSocietyApplication.java`
  - 应用启动入口
  - 应用启动入口
  - REST Controller
  - CORS / Cookie 配置
  - 认证、社交、兼容接口装配

- `backend/pom.xml`
  - Spring Boot / Maven 依赖与构建配置

### 3.3 合约与链上扩展

合约子项目位于：

- `monad-nft/`

当前保留的意义：

- 未来可把用户身份、帖子哈希、媒体哈希、创作者声明锚定到链上
- 未来可把“内容发布”与“内容所有权证明”绑定
- 未来可把实体书 / 数字内容 / 社交身份三者统一到同一套链上资产体系

### 3.4 启动脚本

当前推荐启动方式：

- `start-dev.ps1`
- `start-dev.cmd`

其特点：

- 同时启动前后端
- 关闭脚本窗口即可终止前后端
- 会检查 4173 和 8080 端口占用
- 后端启动时会自动读取 `.env` 并尝试连接 PostgreSQL / Redis
- 如果设置了 `DATABASE_URL`，后端会自动执行 `backend/migrations/*.sql`

---

## 4. 系统架构

### 4.1 逻辑分层

整个系统可分为五层：

1. 表现层
   - Vue 页面
   - 时间线、会话、媒体、实例视图

2. 接口层
   - `frontend/src/api/socialApi.ts`
   - 把前端行为映射为 REST 请求

3. 应用服务层
   - `backend/src/main/java/com/molesociety/backend/MoleSocietyApplication.java`
   - 路由、环境、兼容接口、跨模块装配

4. 领域层
   - `backend/src/main/java/com/molesociety/backend/MoleSocietyApplication.java`
   - 用户、帖子、媒体、会话、消息、实例等核心模型及操作

5. 存储 / 协议层
   - PostgreSQL 主存储
   - Redis 会话、挑战、快照回退
   - 内存 fallback
   - Relay 兼容接口
   - 后续可扩展至 Arweave / IPFS / ActivityPub / WebSocket

### 4.2 核心请求流

#### 4.2.1 页面初始化

1. 前端加载 `App.vue`
2. 调用 `GET /api/v1/social/bootstrap`
3. 后端读取 SocialService 当前状态
4. 返回：
   - currentUser
   - stats
   - feed
   - users
   - media
   - conversations
   - instances
5. 前端将返回数据映射为页面卡片与列表

#### 4.2.2 发布帖子

1. 用户输入帖子内容
2. 若选择媒体，前端先调用 `POST /api/v1/social/media`
3. 后端创建 MediaAsset
4. 前端再调用 `POST /api/v1/social/posts`
5. 后端根据 `authorId` 和 `mediaIds` 生成 SocialPost
6. 前端把新帖子插入时间线

#### 4.2.3 发送消息

1. 用户选中会话
2. 输入消息内容
3. 前端调用 `POST /api/v1/social/conversations/{id}/messages`
4. 后端查找会话和发送者
5. 生成 ChatMessage
6. 更新会话 `updatedAt`
7. 返回完整会话对象
8. 前端刷新当前会话列表

---

## 5. 数据设计

### 5.1 核心实体

当前系统的核心实体包括：

- `SocialUser`
- `SocialPost`
- `MediaAsset`
- `Conversation`
- `ChatMessage`
- `FederationInstance`

它们不是孤立存在的，而是构成了一套有明确聚合关系的社交领域模型。

### 5.2 实体说明

#### 5.2.1 用户 `SocialUser`

用户是整套系统的身份根节点，关键字段包括：

- `id`
- `handle`
- `displayName`
- `bio`
- `instance`
- `wallet`
- `avatarUrl`
- `followers`
- `following`
- `createdAt`

职责：

- 表示账号身份
- 绑定所属实例
- 绑定钱包或链上身份
- 成为帖子、媒体、消息的所有者或发送者

#### 5.2.2 帖子 `SocialPost`

帖子是社交平台的内容聚合根，关键字段包括：

- `id`
- `authorId`
- `authorHandle`
- `authorName`
- `instance`
- `content`
- `visibility`
- `storageUri`
- `attestationUri`
- `tags`
- `media`
- `replies`
- `boosts`
- `likes`
- `createdAt`

职责：

- 表示时间线中的一个可传播内容对象
- 可携带链上存证 URI
- 可携带去中心化存储 URI
- 可绑定多个媒体附件

#### 5.2.3 媒体 `MediaAsset`

媒体是可以独立存在并被多个页面引用的资源实体，字段包括：

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

职责：

- 表示媒体元数据
- 表示资源所有权归属
- 表示资源在外部存储系统中的寻址信息

#### 5.2.4 会话 `Conversation`

会话是消息聚合根，字段包括：

- `id`
- `title`
- `participantIds`
- `encrypted`
- `messages`
- `updatedAt`

职责：

- 表示一个私信空间
- 管理参与者集合
- 管理消息的顺序与归属

#### 5.2.5 消息 `ChatMessage`

消息属于会话内部实体，字段包括：

- `id`
- `conversationId`
- `senderId`
- `senderHandle`
- `body`
- `createdAt`

职责：

- 表示单条消息内容
- 依附于 Conversation
- 由 Conversation 聚合统一维护

#### 5.2.6 联邦实例 `FederationInstance`

实例表示社交网络中的一个站点节点，字段包括：

- `name`
- `focus`
- `members`
- `latency`
- `status`

职责：

- 表示站点层级的拓扑单元
- 为未来的实例间路由、联邦同步、治理规则提供模型基础

### 5.3 实体关系

当前实体关系可以概括为：

- 一个 `FederationInstance` 可以拥有多个 `SocialUser`
- 一个 `SocialUser` 可以发布多个 `SocialPost`
- 一个 `SocialUser` 可以上传多个 `MediaAsset`
- 一个 `SocialPost` 可以引用多个 `MediaAsset`
- 一个 `Conversation` 可以包含多个 `ChatMessage`
- 一个 `Conversation` 可以关联多个 `SocialUser`
- 一个 `ChatMessage` 只能属于一个 `Conversation`
- 一个 `ChatMessage` 只能由一个 `SocialUser` 发送

如果用关系化方式表示，可理解为：

```text
FederationInstance 1 --- N SocialUser
SocialUser         1 --- N SocialPost
SocialUser         1 --- N MediaAsset
SocialPost         N --- N MediaAsset
Conversation       N --- N SocialUser
Conversation       1 --- N ChatMessage
SocialUser         1 --- N ChatMessage
```

### 5.4 当前实现方式

当前 Spring Boot 后端实现使用轻量 JDBC 持久化，而不是传统 ORM 模式：

- PostgreSQL 可用时，用户、账号、帖子、媒体、关注、会话、消息写入关系表
- Redis 可用时，保存 session / challenge TTL，并保留社交快照作为回退
- PostgreSQL 不可用时，系统自动回退到 Redis snapshot；Redis 也不可用时回退内存状态
- 请求读写仍通过 `SocialService` 聚合操作完成

即：

- `users []SocialUser`
- `posts []SocialPost`
- `media []MediaAsset`
- `conversations []Conversation`
- `instances []FederationInstance`

这种设计的优点：

- 原型开发速度快
- 结构清晰
- 便于前后端快速联调
- 容易演示领域关系
- 数据库可用时具备跨进程持久化能力

这种设计的限制：

- 并发写场景下不够强
- 当前未引入 ORM / Repository 分层，复杂查询仍由 JDBC SQL 维护
- 无法天然支持复杂检索与分页
- fallback 快照粒度较粗

---

## 6. 数据库物理模型设计

### 6.1 当前版本的物理模型

当前版本的主数据库物理模型是 **PostgreSQL 表模型**，Redis 只作为 session / challenge 和降级快照。

当 Redis 可用时，后端仍会将整个领域集合序列化为 JSON，并写入以下回退 Key：

- `social:snapshot:users`
- `social:snapshot:posts`
- `social:snapshot:media`
- `social:snapshot:conversations`
- `social:snapshot:instances`
- `social:snapshot:follows`

这意味着：

- 每个 Key 存的是该实体集合的完整快照
- PostgreSQL 不可用时，服务启动会尝试从 Redis 读取
- 若 PostgreSQL / Redis 中均无数据，则加载内置种子数据
- 每次写操作后同步 JDBC 主存储，并刷新 Redis 快照

### 6.2 旧业务兼容键

为了兼容图书确权时代的逻辑，后端仍使用以下 Redis 结构：

- `vault:roles:publishers_codes`
  - Publisher 角色码集合

- `vault:roles:publishers`
  - Publisher 钱包地址集合

- `vault:codes:valid`
  - 有效兑换码集合

- `vault:bind:{codeHash}`
  - 兑换码绑定信息 Hash

- `vault:referrers`
  - 推荐人排行榜 ZSet

- `vault:rewards:{address}`
  - 奖励统计 Hash

### 6.3 当前持久化模型的评价

JDBC + Redis fallback 适合：

- Demo
- 原型验证
- 单节点开发环境
- 小规模演示数据

仍需后续强化：

- 事务边界拆分
- 数据库连接池配置
- 全文搜索
- 时间线扇出
- 多实例分布式写入

### 6.4 建议的生产级物理模型

如果将本项目继续向生产环境推进，建议改造成以下组合：

#### 关系型数据库

建议使用：

- PostgreSQL

建议拆分的物理表：

- `users`
- `instances`
- `posts`
- `media_assets`
- `post_media_rel`
- `conversations`
- `conversation_participants`
- `messages`
- `follows`
- `notifications`
- `attestations`

#### Redis

Redis 在生产环境中建议承担：

- 时间线缓存
- 热门内容缓存
- 会话最近消息缓存
- 限流
- 分布式锁
- 推荐与排行榜

#### 对象存储 / 内容寻址

建议使用：

- S3 / MinIO 存临时或回源文件
- IPFS / Pinata 存内容寻址副本
- Arweave 存永久归档副本

#### 搜索引擎

建议使用：

- OpenSearch / Elasticsearch

用于：

- 帖子搜索
- 用户搜索
- 标签搜索
- 媒体搜索

### 6.5 推荐的关系表设计

#### 用户表 `users`

```sql
id                varchar primary key
handle            varchar unique not null
display_name      varchar not null
bio               text
instance_id       varchar not null
wallet            varchar
avatar_url        text
followers_count   int default 0
following_count   int default 0
created_at        timestamptz not null
updated_at        timestamptz not null
```

#### 帖子表 `posts`

```sql
id                varchar primary key
author_id         varchar not null
instance_id       varchar not null
content           text not null
visibility        varchar not null
storage_uri       text
attestation_uri   text
replies_count     int default 0
boosts_count      int default 0
likes_count       int default 0
created_at        timestamptz not null
updated_at        timestamptz not null
```

#### 媒体表 `media_assets`

```sql
id                varchar primary key
owner_id          varchar not null
name              varchar not null
kind              varchar not null
url               text
storage_uri       text
cid               text
size_bytes        bigint default 0
status            varchar not null
created_at        timestamptz not null
updated_at        timestamptz not null
```

#### 帖子-媒体关系表 `post_media_rel`

```sql
post_id           varchar not null
media_id          varchar not null
sort_order        int default 0
primary key (post_id, media_id)
```

#### 会话表 `conversations`

```sql
id                varchar primary key
title             varchar
encrypted         boolean default false
updated_at        timestamptz not null
created_at        timestamptz not null
```

#### 会话参与者表 `conversation_participants`

```sql
conversation_id   varchar not null
user_id           varchar not null
joined_at         timestamptz not null
primary key (conversation_id, user_id)
```

#### 消息表 `messages`

```sql
id                varchar primary key
conversation_id   varchar not null
sender_id         varchar not null
body              text not null
created_at        timestamptz not null
```

---

## 7. 用户设计

### 7.1 身份模型

当前用户身份由以下几个维度共同构成：

- 平台内 ID
- Handle
- Display Name
- Instance
- Wallet

这意味着一个用户同时具备：

- 社交平台身份
- 联邦实例身份
- 链上身份扩展位

### 7.2 权限模型

当前版本还没有完整 RBAC，但已经具备雏形：

- 普通用户：可发帖、上传、发消息
- 实例用户：属于某个 instance
- Admin 兼容入口：`/api/admin/check-access`
- Publisher / Reader 旧角色：保留在 legacy relay 逻辑中

未来可扩展为：

- User
- Moderator
- Instance Admin
- Relay Admin
- Publisher / Creator

### 7.3 用户旅程

一个典型用户在当前系统中的行为路径是：

1. 打开前端页面
2. 前端从 bootstrap 接口获取当前用户与时间线
3. 用户浏览帖子
4. 用户上传媒体并发帖
5. 用户打开私信会话
6. 用户继续在实例之间探索内容

未来可以继续扩展：

7. 用户关注其他实例用户
8. 用户加入群组或频道
9. 用户将帖子哈希写入链上
10. 用户把长文、书稿或数字刊物永久归档到 Arweave / IPFS

---

## 8. 功能架构设计

### 8.1 前台功能域

前台目前已经形成五个功能域：

- 时间线域
  - 展示帖子流
  - 发帖
  - 媒体附件展示

- 发现域
  - 热门标签
  - 平台演进方向展示

- 私信域
  - 会话列表
  - 会话详情
  - 发送消息

- 存储域
  - 媒体资产视图
  - CID / StorageURI 展示

- 联邦域
  - 实例列表
  - 实例属性展示

### 8.2 后台 / 服务功能域

后端目前可拆成三个功能域：

- 社交域
  - 用户
  - 帖子
  - 媒体
  - 会话
  - 消息
  - 实例

- 中继兼容域
  - Mint Relay
  - 兑换码校验
  - 推荐统计
  - 奖励累积

- 平台运行域
  - 健康检查
  - 启动装配
  - Redis 初始化
  - RPC 初始化

### 8.3 边界划分

当前架构非常适合继续演进为以下边界：

- `identity-service`
- `social-service`
- `media-service`
- `federation-service`
- `relay-service`
- `notification-service`

即使现在仍然在一个 Spring Boot 服务中，这种边界也已经在代码结构和实体模型中初步形成。

---

## 9. API 设计

### 9.1 社交 API

#### Bootstrap

- `GET /api/v1/social/bootstrap`

返回前端启动所需的聚合数据：

- currentUser
- stats
- feed
- users
- media
- conversations
- instances

#### 用户

- `GET /api/v1/social/users`
- `POST /api/v1/social/users`
- `GET /api/v1/social/users/{id}`

#### 帖子

- `GET /api/v1/social/feed`
- `POST /api/v1/social/posts`
- `GET /api/v1/social/posts/{id}`

#### 媒体

- `GET /api/v1/social/media`
- `POST /api/v1/social/media`

#### 会话

- `GET /api/v1/social/conversations`
- `POST /api/v1/social/conversations`
- `GET /api/v1/social/conversations/{id}`
- `POST /api/v1/social/conversations/{id}/messages`

#### 联邦实例

- `GET /api/v1/social/instances`

### 9.2 兼容接口

为保留原图书确权业务，还支持：

- `GET /secret/get-binding`
- `GET /secret/verify`
- `POST /relay/mint`
- `POST /relay/save-code`
- `POST /relay/reward`
- `GET /relay/stats`
- `GET /api/admin/check-access`
- `GET /api/v1/analytics/distribution`

### 9.3 健康检查

- `GET /healthz`

返回：

- 服务状态
- Redis 状态
- Relay 可用性
- SocialStats

---

## 10. 安装与运行

### 10.1 前置环境

- Node.js 18+
- npm
- Java 17+
- Maven 3.9+
- Redis（推荐，本地开发可选）

### 10.2 安装依赖

前端：

```bash
npm install
```

后端：

```bash
cd backend
mvn dependency:resolve
```

### 10.3 启动方式

#### 方式一：一键启动

```bat
start-dev.cmd
```

特点：

- 同时启动前后端
- 关闭脚本窗口即可终止前后端

#### 方式二：分别启动

前端：

```bash
cd E:\MoleSociety\frontend
npm run dev:frontend
```

后端：

```bash
cd E:\MoleSociety\backend
mvn spring-boot:run
```

### 10.4 访问地址

- 前端：`http://localhost:4173`
- 后端：`http://127.0.0.1:8080`
- 健康检查：`http://127.0.0.1:8080/healthz`

### 10.5 构建

前端构建：

```bash
cd frontend
npm run build
```

后端编译：

```bash
cd backend
mvn package
```

---

## 11. 当前实现与未来路线

### 11.1 当前已实现

- Vue 社交前端
- Spring Boot 社交 API
- PostgreSQL / Redis 持久化社交数据服务
- 时间线
- 媒体元数据
- 会话与消息
- 联邦实例视图
- 一键启动脚本
- 旧 Relay 兼容逻辑

### 11.2 下一阶段推荐

1. 媒体上传改为真实对象存储写入
2. 会话改为 WebSocket 实时同步
3. 引入关注、点赞、转发等关系模型
4. 增加通知系统
5. 将当前 JDBC Store 进一步拆分为 Spring Data JDBC / Repository 层
6. 接入 ActivityPub 联邦协议
7. 将帖子哈希、媒体哈希、创作者声明写入链上
8. 将原“图书确权”能力升级为链上身份与出版资产系统

### 11.3 长期方向

Whale Vault Social 的长期目标不是只成为一个社交网站，而是成为：

- 去中心化内容发布层
- 创作者身份与资产层
- 永续存储入口层
- 联邦社交协议层
- 链上确权与社群协作层

也就是说，它可以同时服务于：

- 创作者内容社区
- 数字出版平台
- 阅读社群
- NFT / 资产证明体系
- 抗审查内容镜像网络

---

## 12. 仓库关键文件索引

- `frontend/src/App.vue`
  - 前端主界面

- `frontend/src/api/socialApi.ts`
  - 前端 API 适配层

- `backend/src/main/java/com/molesociety/backend/MoleSocietyApplication.java`
  - 服务启动与路由入口

- `backend/pom.xml`
  - Spring Boot 依赖与构建配置

- `start-dev.cmd`
  - Windows 一键启动脚本

- `monad-nft/`
  - 合约子项目

---

## 13. 许可证与说明

本项目延续开源协作思路，当前重点在于：

- 架构演进
- 社交产品原型
- 去中心化存储与链上扩展能力

若要走向生产环境，建议在以下方面继续增强：

- 安全审计
- 存储层升级
- 身份与权限体系
- 审计日志
- 备份恢复
- 内容审核与实例治理

Whale Vault Social 当前最重要的价值，在于它已经完成了从“单一链上领取应用”向“去中心化社交平台底座”的结构性跨越。

---

## 14. 接口文档

- 详细接口文档见：`docs/social-api.md`
