# MoleSociety 鉴权实现流程

本文档描述 MoleSociety 的推荐登录体系：

- 首次注册：账号信息 + 唯一钱包绑定
- 日常登录：账号密码
- 敏感操作：钱包签名二次确认

目标是同时满足两件事：

- 降低去中心化社交 DApp 的使用门槛
- 保留钱包作为根身份的可信性

## 1. 设计原则

- 钱包是根身份，不是每次登录都必须使用的钱包弹窗入口
- 账号密码是便捷登录方式，不替代钱包所有权证明
- 一个账号只允许绑定一个主钱包
- 一个主钱包只允许绑定一个账号
- 高风险操作必须要求钱包再次签名
- 普通会话使用后端 Session Cookie，不在前端长期保存敏感凭证

## 2. 用户登录模型

### 2.1 首次注册

首次使用时，用户完成以下步骤：

1. 填写用户名、密码、可选邮箱
2. 连接 EVM 钱包
3. 后端生成钱包绑定挑战 `bind challenge`
4. 用户钱包签名绑定消息
5. 后端验签成功后创建账号
6. 后端写入账号与钱包唯一绑定关系
7. 后端创建登录会话并返回 Session Cookie

### 2.2 日常登录

用户后续登录可直接使用：

- 用户名或邮箱
- 密码

后端验证通过后：

- 创建 Session Cookie
- 返回当前用户资料

此时不强制要求连接钱包。

### 2.3 敏感操作

以下操作建议要求再次钱包签名：

- 修改绑定钱包
- 找回账号
- 发起链上交易
- 内容上链存证
- 管理员提权
- 删除账号

## 3. 系统角色划分

### 3.1 钱包负责什么

- 证明用户真正控制某个地址
- 作为 DApp 根身份
- 用于高风险操作确认

### 3.2 账号密码负责什么

- 降低普通登录门槛
- 支撑日常浏览、发帖、聊天的快速登录
- 与钱包绑定后提供 Web2 风格体验

### 3.3 后端 Session 负责什么

- 表示用户当前登录态
- 统一保护发帖、上传、私信等业务接口
- 让前端不需要每次操作都弹钱包

## 4. 数据模型

建议至少新增以下数据结构。

### 4.1 account

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | string | 账号 ID |
| username | string | 用户名，唯一 |
| email | string | 邮箱，可选唯一 |
| password_hash | string | 密码哈希 |
| status | string | 账号状态 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 4.2 wallet_binding

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | string | 绑定记录 ID |
| account_id | string | 账号 ID |
| wallet_address | string | 钱包地址，唯一 |
| chain_id | number | 链 ID |
| is_primary | bool | 是否主钱包 |
| verified_at | string | 验签通过时间 |
| created_at | string | 创建时间 |

### 4.3 auth_session

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | string | Session ID |
| account_id | string | 账号 ID |
| wallet_address | string | 当前绑定钱包 |
| issued_at | string | 签发时间 |
| expires_at | string | 过期时间 |

### 4.4 wallet_challenge

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| nonce | string | 一次性随机串 |
| address | string | 钱包地址 |
| purpose | string | `bind` / `login` / `recover` / `confirm` |
| message | string | 待签名消息 |
| expires_at | string | 过期时间 |

## 5. 接口设计

### 5.1 账号注册并绑定钱包

`POST /api/v1/auth/register`

请求体：

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "plain-password",
  "walletAddress": "0x1234...",
  "chainId": 10143,
  "signature": "0x...",
  "nonce": "bind_nonce"
}
```

后端完成：

- 校验用户名和邮箱唯一性
- 校验 nonce 是否存在、是否过期、是否一次性
- 校验签名是否与 `walletAddress` 匹配
- 创建账号
- 创建唯一钱包绑定
- 创建 Session

### 5.2 获取绑定挑战

`POST /api/v1/auth/bind-challenge`

请求体：

```json
{
  "walletAddress": "0x1234...",
  "chainId": 10143
}
```

返回：

```json
{
  "ok": true,
  "data": {
    "nonce": "abc123",
    "message": "MoleSociety wallet binding message...",
    "expiresAt": "2026-04-10T12:00:00Z"
  }
}
```

### 5.3 账号密码登录

`POST /api/v1/auth/password-login`

请求体：

```json
{
  "identifier": "alice",
  "password": "plain-password"
}
```

后端完成：

- 查找用户名或邮箱
- 校验密码哈希
- 读取主钱包绑定信息
- 创建 Session Cookie
- 返回账号资料和绑定钱包摘要

### 5.4 当前登录用户

`GET /api/v1/auth/me`

用途：

- 页面刷新后恢复登录态
- 路由守卫
- 业务接口的当前用户识别

### 5.5 退出登录

`POST /api/v1/auth/logout`

后端完成：

- 删除 Session
- 清 Cookie

### 5.6 钱包二次确认挑战

`POST /api/v1/auth/action-challenge`

用途：

- 换绑钱包
- 链上操作确认
- 找回账号

## 6. 页面流程

### 6.1 登录页

建议做成双 Tab：

- 快捷登录
- 钱包登录

快捷登录区域：

- 用户名 / 邮箱输入框
- 密码输入框
- 登录按钮
- 注册并绑定钱包入口
- 说明文案：账号已与唯一钱包绑定，敏感操作仍需钱包确认

钱包登录区域：

- 连接钱包并登录按钮
- 首次使用说明
- 找回账号入口

### 6.2 注册页

建议流程：

1. 填写用户名、密码、可选邮箱
2. 点击连接钱包
3. 获取绑定挑战
4. 钱包签名
5. 调用注册接口完成账号创建和钱包绑定

### 6.3 个人设置页

建议提供：

- 当前账号信息
- 已绑定主钱包
- 换绑钱包按钮
- 重置密码入口
- 设备会话管理

## 7. 安全规则

- 密码必须存哈希，建议 `bcrypt` 或 `argon2id`
- nonce 必须一次性使用
- challenge 有效期建议 5 分钟
- Session Cookie 必须 `HttpOnly`
- 生产环境 Cookie 必须 `Secure`
- 建议 `SameSite=Lax`
- 登录接口必须限流
- 敏感操作必须校验 Session + 钱包签名
- 钱包绑定必须保证唯一约束
- 忘记密码流程建议走钱包验证或邮箱 + 钱包双重验证

## 8. 推荐实现阶段

### Phase 1

目标：先完成混合登录闭环

- 保留现有钱包签名登录能力
- 新增账号表和钱包绑定表
- 新增账号密码登录
- 新增注册并绑定钱包
- `/auth/me` 和 Session 统一化

### Phase 2

目标：补齐账号管理能力

- 忘记密码
- 修改密码
- 会话管理
- 绑定状态展示

### Phase 3

目标：补齐高风险操作验证

- 改绑钱包
- 上链前确认签名
- 管理员敏感操作验签

## 9. 后端职责

我主要负责后端部分，范围如下：

- 设计并实现账号、钱包绑定、Session、challenge 的后端数据结构
- 补全注册、密码登录、钱包绑定、`/auth/me`、退出登录接口
- 实现密码哈希校验
- 实现钱包签名验签
- 实现唯一钱包绑定约束
- 把发帖、上传、私信等接口统一接到后端 Session 鉴权
- 为前端提供稳定的错误码和返回结构

## 10. 前后端分工建议

前端负责：

- 登录页双 Tab UI
- 注册页表单与钱包连接交互
- 登录态恢复与路由守卫
- 敏感操作二次签名弹窗

后端负责：

- 所有鉴权接口
- Session 管理
- challenge 管理
- 签名验签
- 密码登录
- 账号与钱包唯一绑定
- 业务接口鉴权落地

## 11. 当前代码状态

当前仓库已具备一部分钱包签名登录基础能力：

- `POST /api/v1/auth/challenge`
- `POST /api/v1/auth/verify`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`

下一步建议直接在此基础上扩展：

1. 新增账号表与钱包绑定表
2. 新增 `bind-challenge`
3. 新增 `register`
4. 新增 `password-login`
5. 将当前 `EnsureWalletUser` 过渡为正式账号绑定逻辑
