# MoleSociety 前端对接文档（鉴权 + 聊天）

本文档给 Windsurf 对接使用，基于当前 Spring Boot 后端实现（`backend/src/main/java/com/molesociety/backend/MoleSocietyApplication.java`）。

## 1. 基础约定

- Base URL：`VITE_SOCIAL_API_URL`（示例：`http://127.0.0.1:8080`）
- 鉴权方式：后端通过 `Set-Cookie` 写入会话，前端请求需带 `credentials: 'include'`
- 通用成功结构：

```json
{
  "ok": true,
  "data": {}
}
```

- 鉴权错误结构（auth 相关接口）：

```json
{
  "ok": false,
  "error": "提示文案",
  "code": "AUTH_XXX",
  "type": "validation|wallet|session|account|conflict|unknown"
}
```

## 2. 鉴权接口

### 2.1 获取钱包登录挑战

- `POST /api/v1/auth/challenge`
- Request:

```json
{
  "address": "0x...",
  "chainId": 10143
}
```

- Response `200`：返回 `nonce/message/expiresAt`，前端拉起钱包签名 `message`

### 2.2 钱包签名登录

- `POST /api/v1/auth/verify`
- Request:

```json
{
  "address": "0x...",
  "nonce": "xxxx",
  "signature": "0x..."
}
```

- Response `200`：登录成功，后端写 cookie
- Response `404`：`AUTH_WALLET_NOT_BOUND`（钱包未绑定账号）
- Response `401`：签名无效/挑战过期/地址不匹配

### 2.3 账号密码登录（账号需已绑定钱包）

- `POST /api/v1/auth/password-login`
- Request:

```json
{
  "identifier": "username_or_email",
  "password": "******"
}
```

- Response `200`：登录成功，后端写 cookie
- Response `404`：`AUTH_ACCOUNT_NOT_FOUND`
- Response `401`：密码错误、账号未绑定钱包（`AUTH_WALLET_REQUIRED`）等

### 2.4 注册（注册时必须绑定钱包）

推荐流程：
1. `POST /api/v1/auth/bind-challenge`
2. 钱包签名返回 `nonce + signature`
3. `POST /api/v1/auth/register`

- `POST /api/v1/auth/bind-challenge`

```json
{
  "walletAddress": "0x...",
  "chainId": 10143
}
```

- `POST /api/v1/auth/register`

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "123456",
  "walletAddress": "0x...",
  "chainId": 10143,
  "nonce": "xxxx",
  "signature": "0x..."
}
```

- Response `201`：注册成功并自动登录（写 cookie）
- Response `409`：用户名/邮箱/钱包已占用（`AUTH_USERNAME_TAKEN` / `AUTH_EMAIL_TAKEN` / `AUTH_WALLET_ALREADY_BOUND`）
- Response `400`：参数不合法、签名缺失、挑战不匹配等

### 2.4.1 免钱包插件注册（自动托管钱包）

当不传 `walletAddress/nonce/signature` 时，后端会自动生成唯一钱包地址并绑定账号。  
前端可直接调用同一个接口：

- `POST /api/v1/auth/register`

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "123456",
  "autoWallet": true
}
```

兼容说明：
- `autoWallet` 可省略；只要不传 `walletAddress`，后端默认走自动钱包模式
- 成功后同样自动登录并写 cookie

### 2.5 当前登录态

- `GET /api/v1/auth/me`
- Response `200`：返回当前用户
- Response `401`：未登录或会话失效（建议前端直接回到登录页）

### 2.6 退出登录

- `POST /api/v1/auth/logout`
- Response `200`：`{ "ok": true, "data": { "loggedOut": true } }`

## 3. 社交与聊天室接口

### 3.1 关注 / 取关

- `POST /api/v1/social/users/{id}/follow`
- `DELETE /api/v1/social/users/{id}/follow`
- 需要登录（cookie）

### 3.2 创建会话（当前为双人会话）

- `POST /api/v1/social/conversations`
- 需要登录
- Request:

```json
{
  "title": "聊天标题",
  "participantIds": ["user_xxx"],
  "encrypted": true
}
```

说明：
- 后端会把当前登录用户自动作为 `initiator`
- 最终参与者必须恰好 2 人，否则返回错误

### 3.3 发送消息

- `POST /api/v1/social/conversations/{id}/messages`
- 需要登录
- Request:

```json
{
  "body": "hello"
}
```

后端会自动把 `senderId` 覆盖为当前登录用户。

## 4. 聊天权限规则（核心）

对于双人会话 A 与 B：

1. 如果互相关注（A 关注 B 且 B 关注 A）：
- 双方都可无限发送消息

2. 如果不是互相关注：
- 只有发起会话的一方（initiator）可以发送
- 且只能发送 1 条
- 非发起方发送会收到错误：`the other user has not followed you back yet`
- 发起方超过 1 条会收到错误：`awaiting follow-back: only one message is allowed`

## 5. 前端提示文案建议（避免术语）

下面是推荐直接给用户的中文提示：

- `AUTH_WALLET_NOT_BOUND`：`这个钱包还没有关联账号，请先注册。`
- `AUTH_ACCOUNT_NOT_FOUND`：`没有找到这个账号，请检查后再试。`
- `AUTH_WALLET_ALREADY_BOUND`：`这个钱包已关联其他账号，请更换钱包。`
- `AUTH_WEAK_PASSWORD`：`密码至少需要 6 位。`
- `AUTH_SESSION_REQUIRED`：`登录状态已失效，请重新登录。`
- 聊天未互关且非发起方：`对方还没有关注你，暂时不能回复。`
- 聊天未互关且超出 1 条：`在对方关注你之前，只能先发送一条消息。`

兜底提示建议：`操作没有成功，请稍后再试。`

## 6. 前端最小实现清单（给 Windsurf）

1. 封装统一请求层：默认 `credentials: 'include'`
2. 鉴权页支持三条链路：
- 钱包登录：`challenge -> wallet sign -> verify`
- 账号登录：`password-login`
- 注册：`bind-challenge -> wallet sign -> register`
3. 启动时调用 `GET /api/v1/auth/me` 恢复登录态
4. 聊天发送失败时按 `error/code` 走友好提示，不展示后端原始英文
5. 会话页根据错误提示引导“先互相关注再聊天”
