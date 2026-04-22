# Phase 1 验证清单

本文档用于验证 `MoleSociety` 的 Phase 1 是否已经达到“PostgreSQL 主持久化路径 + Redis 辅助回退路径”的目标状态。

## 1. 验证目标

确认以下事项：

- PostgreSQL 配置可被后端读取
- 启动时 migration 会自动执行
- 健康检查能返回数据库状态与 migration 状态
- 核心读写路径在数据库可用时优先走 PostgreSQL
- Redis 在 Phase 1 后仅承担 session / challenge / 回退辅助角色
- 不配置 `DATABASE_URL` 时，系统仍可回退到 memory 实现

---

## 2. 本地环境准备

### 必需

- Node.js / npm
- Java 17+
- Maven 3.9+
- PostgreSQL 14+

### 可选

- Redis
  - 如果不启动 Redis，系统仍可运行
  - 但 session / challenge 会回退到内存模式

---

## 3. 推荐环境变量

在项目根目录新建 `.env`，至少填写：

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

说明：

- `DB_MIGRATIONS_DIR=./migrations` 是相对于 `backend/` 目录生效的
- `start-dev.ps1` / `start-dev.cmd` 启动后端时，会以 `backend/` 作为工作目录运行 `mvn spring-boot:run`

---

## 4. 数据库初始化验证

### Step 1：启动 PostgreSQL

确保数据库存在：

- 数据库名：`molesociety`

### Step 2：启动项目

推荐方式：

```powershell
./start-dev.ps1
```

或：

```bat
start-dev.cmd
```

### Step 3：检查健康接口

访问：

- `http://127.0.0.1:8080/healthz`

期望响应中包含：

- `database.enabled = true`
- `database.mode = connected`
- `databaseMode = postgres(jdbc)+redis`
- `migrations.count >= 1`
- `migrations.applied` 包含 `0001_initial_schema.sql`

如果未配置 `DATABASE_URL`，则期望：

- `database.enabled = false`
- `databaseMode = memory+redis`

---

## 5. 核心表验证

连接 PostgreSQL 后检查以下表是否存在：

- `schema_migrations`
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

并确认：

- `schema_migrations` 中至少有 `0001_initial_schema.sql`

---

## 6. 写路径验证

建议按以下顺序验证。

### 6.1 注册用户 / 账户

调用：

- `POST /api/v1/auth/register`

验证：

- `social_users` 新增记录
- `auth_accounts` 新增记录

### 6.2 发帖

调用：

- `POST /api/v1/social/posts`

验证：

- `social_posts` 新增记录
- 如有媒体，`post_media_links` 新增记录

### 6.3 上传媒体元数据

调用：

- `POST /api/v1/social/media`

验证：

- `media_assets` 新增记录

### 6.4 关注 / 取消关注

调用：

- `POST /api/v1/social/users/{id}/follow`
- `DELETE /api/v1/social/users/{id}/follow`

验证：

- `user_follows` 正确增删

### 6.5 创建会话 / 发送消息

调用：

- `POST /api/v1/social/conversations`
- `POST /api/v1/social/conversations/{id}/messages`

验证：

- `conversations` 新增记录
- `conversation_participants` 新增参与者记录
- `chat_messages` 新增消息记录

---

## 7. 读路径验证

验证以下接口在 PostgreSQL 可用时都能正常返回：

- `GET /api/v1/social/bootstrap`
- `GET /api/v1/social/feed`
- `GET /api/v1/social/users`
- `GET /api/v1/social/users/{id}`
- `GET /api/v1/social/media`
- `GET /api/v1/social/posts/{id}`
- `GET /api/v1/social/posts/{id}/thread`
- `GET /api/v1/social/posts/{id}/replies`
- `GET /api/v1/social/conversations`
- `GET /api/v1/social/conversations/{id}`

检查点：

- 新写入的数据可被读出
- 重启后端后数据仍存在
- 不依赖 Redis 快照也能正常读取数据库中的数据

---

## 8. Redis 角色验证

### 启动 Redis 时

验证：

- 登录后的 session 可正常维持
- challenge / session 相关流程正常

### 不启动 Redis 时

验证：

- 后端仍可启动
- 非登录依赖的数据库读写仍可用
- session / challenge 退回内存模式

结论标准：

- Redis 不再是主数据来源
- PostgreSQL 才是 Phase 1 的主数据路径

---

## 9. 回退模式验证

删除或清空 `DATABASE_URL` 后重新启动，验证：

- 健康检查显示 `database.enabled = false`
- API 仍可工作
- 写入路径回退到 memory store

这一步用于确认当前 Phase 1 仍保留原型兼容性。

---

## 10. Phase 1 通过标准

满足以下条件即可视为 Phase 1 验证通过：

1. PostgreSQL 可连接
2. migration 自动执行成功
3. 健康检查能显示数据库与 migration 状态
4. users/accounts/posts/media/follows/conversations/messages 已写入 PostgreSQL
5. 核心读路径能从 PostgreSQL 返回数据
6. 重启服务后数据库数据不丢失
7. Redis 不可用时系统仍能以降级模式运行
8. `DATABASE_URL` 缺失时能安全回退到 memory store
