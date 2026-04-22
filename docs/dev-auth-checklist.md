# MoleSociety 开发联调清单（登录/会话）

用于本地开发时快速排查“登录成功但仍未登录”“看不到请求”“后端连不上”等问题。

## 1. 地址必须统一

推荐统一使用 `127.0.0.1`，不要和 `localhost` 混用。

- 前端访问地址：`http://127.0.0.1:4173`
- 后端地址：`http://127.0.0.1:8080`
- 前端环境变量（`frontend/.env.local`）：

```env
VITE_SOCIAL_API_URL=http://127.0.0.1:8080
```

说明：如果前端页面是 `localhost`，API 是 `127.0.0.1`，浏览器 Cookie 可能不会按预期携带，导致 `auth/me` 持续 `401`。

## 2. 后端 Cookie 配置（开发环境）

开发环境必须：

```powershell
$env:COOKIE_SECURE="false"
```

然后重启后端。  
`COOKIE_SECURE=true` 只适用于 HTTPS 场景。

## 3. 前端请求配置

所有鉴权相关请求必须带：

```ts
credentials: 'include'
```

否则浏览器不会携带会话 Cookie。

## 4. 判断是否真正登录成功

登录流程成功标准：

1. `POST /api/v1/auth/password-login` 返回 `200`
2. 响应头存在 `Set-Cookie: molesociety_session=...`
3. 随后 `GET /api/v1/auth/me` 返回 `200`

如果第 1 步成功但第 3 步还是 `401`，优先检查：
- 地址是否混用（`localhost` / `127.0.0.1`）
- `COOKIE_SECURE` 是否为 `false`
- 浏览器是否拦截 Cookie

## 5. 常见状态码解释

- `GET /api/v1/auth/me -> 401`：当前未登录（正常状态探测）
- `POST /api/v1/auth/password-login -> 401`：账号或密码错误，或账号状态不满足登录
- `POST /api/v1/auth/register -> 409`：用户名/邮箱/钱包已被占用
- `favicon.ico -> 404`：仅图标缺失，不影响业务

## 6. 一次性清理后重试

当状态混乱时，按下面顺序重置：

1. 清理浏览器 `127.0.0.1` 的站点 Cookie
2. 确认 `frontend/.env.local` 为 `127.0.0.1:8080`
3. 后端使用 `COOKIE_SECURE=false` 重启
4. 前端重启 dev server
5. 重新登录并观察 `password-login` 与 `auth/me`

## 7. 给前端同学的最小排查日志

建议临时打印以下日志：

- 登录页 `handleSignIn` 是否触发
- `passwordLogin` 函数是否执行
- `request()` 抛错时打印 `status/code/type/error`

这样可以快速区分：
- 没触发点击事件
- 触发了但被前端校验拦截
- 请求已发出但后端返回业务错误
