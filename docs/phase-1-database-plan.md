# Phase 1 数据库迁移说明

Phase 1 已完成，当前系统已从“仅数据库骨架”推进到“PostgreSQL 优先的 Phase 1 持久化架构”。

## 当前行为

- 如果未设置 `DATABASE_URL`：
  - 后端继续使用当前的内存 + Redis 回退模式
  - 所有 store 自动回退到 memory 实现
- 如果设置了 `DATABASE_URL` 且数据库可连接：
  - 后端会建立 PostgreSQL 连接
  - 启动时自动执行 `backend/migrations/*.sql`
  - 健康检查会暴露数据库连接状态、migration 文件和已执行版本
  - `users / accounts / posts / media / follows / conversations / messages` 已优先走 PostgreSQL store
  - `bootstrap / feed / post / replies / media / user / conversations` 等核心读路径已优先走 PostgreSQL store

## 新增环境变量

- `DATABASE_URL`
  - PostgreSQL 连接串，例如：
  - `postgres://postgres:postgres@localhost:5432/molesociety?sslmode=disable`
- `POSTGRES_MAX_OPEN_CONNS`
  - 默认 `10`
- `POSTGRES_MAX_IDLE_CONNS`
  - 默认 `5`
- `POSTGRES_CONN_MAX_LIFETIME`
  - 默认 `30m`
- `DB_MIGRATIONS_DIR`
  - 默认 `./migrations`

## Schema 文件

当前首版 schema 位于：

- `backend/migrations/0001_initial_schema.sql`

该 schema 已定义以下核心表：

- `social_users`
- `auth_accounts`
- `media_assets`
- `social_posts`
- `post_media_links`
- `conversations`
- `conversation_participants`
- `chat_messages`
- `federation_instances`
- `user_follows`
- `schema_migrations`（运行时自动确保存在）

## Phase 1 已完成内容

- PostgreSQL 配置接入
- 连接初始化
- 启动时自动执行 migration
- 健康检查暴露数据库状态与 migration 状态
- `Store / UserStore / SocialStore` 分层入口建立
- `users / accounts` 持久化接入 PostgreSQL
- `posts / media / follows` 写路径接入 PostgreSQL
- `conversations / messages` 写路径接入 PostgreSQL
- 核心读路径优先走 PostgreSQL：
  - 用户列表 / 用户详情
  - 媒体列表
  - 时间线 / 我的时间线
  - 单帖 / 线程 / 回复
  - 会话列表 / 会话详情
  - bootstrap

## Redis 在 Phase 1 结束时的角色

Phase 1 结束后，Redis 不再是架构目标中的主持久化方案，而是过渡性辅助组件：

- Session / challenge 存储
- 无数据库时的原型回退
- 兼容旧逻辑的临时快照能力

也就是说：

- **PostgreSQL = 主持久化路径**
- **Redis = 会话与回退辅助路径**

## 当前阶段仍未覆盖的内容

以下不属于 Phase 1，留待后续阶段：

- 实时消息推送
- 搜索与索引系统
- 对象存储 / 去中心化存储真实上传
- Attestation 状态机
- 联邦协议实现
- Redis 职责彻底收缩后的历史清理与迁移工具

## Phase 1 完成结论

截至当前，Phase 1 的目标已经完成：

- 系统具备 PostgreSQL 主持久化骨架
- migration 机制已可运行
- 核心实体已接入数据库 store
- 核心读写路径已具备数据库优先能力
- Redis 已被降级为辅助与回退角色
