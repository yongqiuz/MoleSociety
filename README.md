# MoleSociety

MoleSociety 是一个包含前端、Spring Boot 后端和 Monad NFT 合约工程的全栈项目。

当前后端已经从 Go 迁移到 Spring Boot 3.5.9，前端已经统一移动到 `frontend/`，后端位于 `backend/`。旧 Go 后端、历史脚本、临时文件、日志、构建产物和过期文档已经清理。

## 项目结构

```text
MoleSociety/
  backend/       Spring Boot 后端
  frontend/      Vue 3 + Vite 前端
  monad-nft/     Foundry 智能合约工程
  start-dev.ps1  Windows PowerShell 一键启动脚本
  start-dev.cmd  Windows CMD 一键启动入口
```

## 技术栈

- 前端：Vue 3、Vite、TypeScript、Vue Router
- 后端：Spring Boot 3.5.9、Java 17、Maven
- 数据库：PostgreSQL，启动时执行 `backend/migrations` 中的 SQL 迁移
- 缓存：Redis，用于登录挑战、会话和部分快照缓存
- 合约：Foundry，位于 `monad-nft/`

## 环境要求

- JDK 17+
- Maven 3.9+
- Node.js 18+ 和 npm
- PostgreSQL 14+，本机示例安装目录为 `E:\pgsql`
- Redis，默认地址 `127.0.0.1:6379`

## 后端环境变量

后端读取 `backend/.env`。该文件只用于本机开发，已被 `.gitignore` 忽略。

示例：

```env
APP_ENV=development
BACKEND_ADDR=0.0.0.0:8080
SERVER_PORT=8080
DATABASE_URL=postgres://postgres:123456@127.0.0.1:5432/molesociety?sslmode=disable
REDIS_ADDR=127.0.0.1:6379
DB_MIGRATIONS_DIR=./migrations
SOCIAL_SEED=1
COOKIE_SECURE=false
```

当 PostgreSQL 可连接时，后端健康检查会显示 `databaseMode: "postgres(jdbc)+redis"` 和 `database.enabled: true`。如果 PostgreSQL 不可用，后端会降级到内存模式，但这只适合临时开发调试。

## 启动 PostgreSQL

如果 PostgreSQL 安装在 `E:\pgsql`，可以使用：

```powershell
E:\pgsql\bin\pg_ctl.exe start -D E:\pgsql\data -l E:\pgsql\postgres.log
E:\pgsql\bin\pg_ctl.exe status -D E:\pgsql\data
```

停止 PostgreSQL：

```powershell
E:\pgsql\bin\pg_ctl.exe stop -D E:\pgsql\data
```

## 启动后端

```powershell
cd E:\MoleSociety\backend
mvn spring-boot:run
```

后端默认监听：

```text
http://127.0.0.1:8080
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
```

## 启动前端

```powershell
cd E:\MoleSociety\frontend
npm install
npm run dev:frontend
```

前端默认监听：

```text
http://127.0.0.1:4173
```

## 一键启动

在项目根目录执行：

```powershell
.\start-dev.ps1
```

或：

```cmd
start-dev.cmd
```

## 构建和测试

后端测试：

```powershell
cd E:\MoleSociety\backend
mvn test
```

前端构建：

```powershell
cd E:\MoleSociety
npm --prefix frontend run build
```

## 后端目录说明

`backend/migrations/`

保存数据库 schema 迁移 SQL。Spring Boot 后端启动时会检查并执行未应用的迁移，目前包含初始表结构 `0001_initial_schema.sql`。

`backend/src/main/java/com/molesociety/backend/`

后端主代码目录。当前主要逻辑集中在 `MoleSocietyApplication.java`，包含 HTTP 接口、认证、社交数据、PostgreSQL JDBC 持久化、Redis 连接、内存降级实现和启动时 schema 兼容处理。

`backend/src/main/resources/`

Spring Boot 配置目录。`application.properties` 定义应用名、默认端口、JSON 输出和错误信息配置。

`backend/src/test/java/com/molesociety/backend/`

后端测试目录，用于 Spring Boot 启动和基础行为验证。

`backend/target/`

Maven 构建输出目录。该目录可随时删除，重新运行 `mvn test`、`mvn package` 或 `mvn spring-boot:run` 会自动生成。

`backend/.env`

本机后端配置文件，包含数据库、Redis、端口等环境变量。该文件不会提交到 Git。

## 主要接口

- `GET /healthz`：后端健康检查，返回 PostgreSQL/Redis 状态
- `GET /api/v1/social/feed`：社交动态列表
- `POST /api/v1/social/posts`：发布动态
- `POST /api/v1/auth/challenge`：生成钱包登录挑战
- `POST /api/v1/auth/verify`：验证签名并创建登录态
- `POST /api/v1/auth/logout`：退出登录

## 合约工程

`monad-nft/` 是独立的 Foundry 工程，包含合约源码、脚本和测试。它没有被合并进前后端目录，避免应用代码和合约代码互相污染。

常用命令：

```powershell
cd E:\MoleSociety\monad-nft
forge build
forge test
```

## 开发约定

- 前端代码只放在 `frontend/`
- 后端代码只放在 `backend/`
- 不提交 `.env`、日志、构建产物、虚拟环境、`node_modules`
- 后端持久化以 PostgreSQL 为主，Redis 为辅助缓存；内存模式只作为本机服务不可用时的降级方案
