# Windsurf 前端改动 TODO（登录/注册/会话）

目标：修复“登录后仍 401”“注册强依赖钱包插件”“错误提示不明确”。

## 1. 环境与启动

文件：`E:/MoleSociety/frontend/.env.local`

```env
VITE_SOCIAL_API_URL=http://127.0.0.1:8080
```

要求：
- 前端访问统一使用 `http://127.0.0.1:4173`
- 不要用 `localhost`

## 2. `frontend/src/api/authApi.ts`

### 2.1 注册参数支持自动钱包

把 `registerAccount` 入参改为可选钱包字段：

```ts
export async function registerAccount(payload: {
  username: string;
  email?: string;
  password: string;
  autoWallet?: boolean;
  walletAddress?: string;
  chainId?: number;
  signature?: string;
  nonce?: string;
}) { ... }
```

### 2.2 保持会话请求带 Cookie

确认 `request()` 中始终有：

```ts
credentials: 'include'
```

### 2.3 错误日志临时保留（联调阶段）

`request()` 抛错前打印：
- `status`
- `payload.error`
- `payload.code`
- `payload.type`

## 3. `frontend/src/pages/LoginPage.vue`

### 3.1 注册改为“无钱包插件也可注册”

当前注册不要先调 `connectWalletForRegistration()`。  
改为直接：

```ts
await registerAccount({
  username,
  email: email || undefined,
  password,
  autoWallet: true,
});
```

### 3.2 注册成功后直接进应用

注册成功后不要再二次调用登录接口，直接：

```ts
redirectAfterAuth();
```

原因：后端 `register` 已自动写会话 Cookie。

### 3.3 登录失败提示文案

按 `ApiError.code` 映射用户友好提示：
- `AUTH_INVALID_PASSWORD`：密码不对，请再试一次
- `AUTH_ACCOUNT_NOT_FOUND`：没有找到这个账号
- `AUTH_WALLET_REQUIRED`：这个账号需要先完成钱包关联
- 兜底：登录没有成功，请稍后再试

## 4. `frontend/src/composables/useAuth.ts`

### 4.1 `loadSession` 行为

- `fetchCurrentSession()` 返回 `401` 时，设置 `session = null` 即可
- 不弹错误 toast

这是正常“未登录探测”。

## 5. `frontend/src/pages/MainApp.vue`

### 5.1 未登录时不要进离线主页

`loadBootstrap()` 里：
- 无 `authSession` -> 直接跳 `/login`
- `ApiError.status === 401` -> 跳 `/login`
- 仅在“已登录但服务异常”时走 `applyFallback`

## 6. 路由守卫核对

文件：`frontend/src/router/index.ts`

确认：
- 进入 `meta.requiresAuth` 页面前必须 `loadSession()`
- `!isAuthenticated` 跳 `/login?redirect=...`
- `guestOnly` 页面在已登录时跳 `/app`

## 7. 联调验收（必须通过）

1. 打开 `http://127.0.0.1:4173/login`
2. 新用户注册（不装钱包插件）成功并跳 `/app`
3. 刷新页面后仍保持登录（`/api/v1/auth/me` 返回 `200`）
4. 退出登录后，`/api/v1/auth/me` 返回 `401`
5. 再次登录成功并进入 `/app`

## 8. 可选优化（收尾）

- 给 `409` 注册冲突增加“去登录”按钮
- 增加 `frontend/public/favicon.ico`，消除 404 噪音
