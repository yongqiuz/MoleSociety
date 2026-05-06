# MoleSociety 生产部署说明

本文档对应当前仓库中的后端 Docker 化部署文件：

- `docker-compose.prod.yml`
- `docker-compose.dev.yml`
- `backend/Dockerfile`
- `.env.prod.example`
- `.env.dev.example`

默认部署形态（Docker 仅负责后端）：

- `backend`：Spring Boot
- `postgres`：PostgreSQL 16
- `redis`：Redis 7

前端需独立部署，且通过 `frontend/src/env.js` 控制请求开发或生产后端。

## 1. 服务器准备

建议最低配置：

- 2 vCPU
- 4 GB RAM
- 40 GB SSD
- Ubuntu 22.04 / 24.04 LTS

需要放通端口：

- `22`：SSH
- `80`：HTTP
- `443`：HTTPS（启用证书后）

## 2. 上传项目

```bash
git clone <你的仓库地址> /opt/molesociety
cd /opt/molesociety
```

或直接把本地项目打包上传到服务器后解压到 `/opt/molesociety`。

## 3. 准备生产环境变量

```bash
cp .env.prod.example .env.prod
vim .env.prod
```

至少修改：

- `POSTGRES_PASSWORD`
- `PUBLIC_BASE_URL`

如果你当前还没有给站点配 HTTPS，务必额外设置：

- `COOKIE_SECURE=false`

示例：

```env
POSTGRES_PASSWORD=replace-with-a-strong-password
PUBLIC_BASE_URL=https://molesociety.longyinstudio.cn
COOKIE_SECURE=false
```

如果域名和 HTTPS 已就绪：

```env
PUBLIC_BASE_URL=https://molesociety.longyinstudio.cn
COOKIE_SECURE=true
```

## 4. 启动

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build
```

查看状态：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml ps
```

查看日志：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f backend
```

## 4.1 清理旧 Docker 配置

如果你之前把前端、后端、数据库都混在一套旧容器里，先清理旧资源，再按新的 prod/dev 结构重建。

只清理当前项目相关的容器、网络、旧镜像，保留数据库数据卷：

```bash
cd /opt/molesociety
bash scripts/docker-cleanup-molesociety.sh
```

如果你连旧的 PostgreSQL / Redis 数据也要一起重置：

```bash
cd /opt/molesociety
bash scripts/docker-cleanup-molesociety.sh --drop-data
```

说明：

- 该脚本会移除旧的 `molesociety-frontend`，因为前端不再由 Docker 部署。
- `prod` 与 `dev` 的数据库卷分离，不会再共用一份库。
- `--drop-data` 是彻底重置，只在你确认旧库不要保留时使用。

## 5. 验证

后端健康检查：

```bash
curl http://127.0.0.1:8080/healthz
```

## 5.1 开发环境后端容器

```bash
cp .env.dev.example .env.dev
docker compose --env-file .env.dev -f docker-compose.dev.yml up -d --build
curl http://127.0.0.1:8081/healthz
```

开发环境首次启动后，会自动使用和生产同结构的表；联邦实例首项会显示为“摩尔开发1号”，用于区分环境。

## 6. 更新发布

```bash
cd /opt/molesociety
git pull
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build
```

## 7. 数据持久化

以下数据会持久化：

- PostgreSQL：`postgres_data`
- Redis：`redis_data`

查看卷：

```bash
docker volume ls
```

## 8. HTTPS

当前 compose 只开放了 `80`，适合先用 IP 或 HTTP 验证。

如果需要正式域名访问，建议在服务器上再加一层反向代理（如 Nginx Proxy Manager / Caddy / 宿主机 Nginx）处理：

- `80 -> 443` 跳转
- Let's Encrypt 证书
- `molesociety.longyinstudio.cn` 指向该服务器公网 IP

## 9. 关闭

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml down
```

保留数据库卷：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml down
```

连数据卷一起删除：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml down -v
```
