# MoleSociety 生产部署说明

本文档对应当前仓库中的 Docker 化部署文件：

- `docker-compose.prod.yml`
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `frontend/nginx.conf`
- `.env.prod.example`

默认部署形态：

- `frontend`：Nginx 托管前端静态资源，并反向代理 `/api/*` 和 `/healthz`
- `backend`：Spring Boot
- `postgres`：PostgreSQL 16
- `redis`：Redis 7

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

示例：

```env
POSTGRES_PASSWORD=replace-with-a-strong-password
PUBLIC_BASE_URL=http://81.70.208.113
```

如果域名和 HTTPS 已就绪：

```env
PUBLIC_BASE_URL=https://molesociety.club
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
docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f frontend
```

## 5. 验证

后端健康检查：

```bash
curl http://127.0.0.1/healthz
```

前端首页：

```bash
curl -I http://127.0.0.1/
```

如果部署正常，浏览器访问：

- `http://81.70.208.113`
- 或 `https://molesociety.club`

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
- `molesociety.club` 指向该服务器公网 IP

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
