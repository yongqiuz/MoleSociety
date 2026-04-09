# Whale Vault Social API 文档

本文档描述当前 Go 社交后端的接口设计，重点覆盖统一 `posts` 模型、帖子线程 `thread`、回复楼层 `replies`、媒体、会话和实例相关接口。

## 1. 基础信息

- 基础地址：`http://localhost:8080`
- 内容类型：`application/json`
- 接口风格：REST
- 当前后端技术：Go + Gorilla Mux + Redis 快照持久化

所有接口统一返回以下包裹结构：

```json
{
  "ok": true,
  "data": {}
}
```

失败时返回：

```json
{
  "ok": false,
  "error": "具体错误信息"
}
```

## 2. 统一 Posts 设计

当前系统采用“类 Twitter”的统一内容模型，不再单独设计评论表。

也就是说：

- 普通动态是 `posts`
- 评论 / 回复也是 `posts`
- 线程关系通过帖子自身的父子字段表达

### 2.1 字段语义

`SocialPost` 关键字段如下：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 帖子 ID |
| `authorId` | string | 作者 ID |
| `authorHandle` | string | 作者 handle |
| `authorName` | string | 作者显示名 |
| `instance` | string | 作者所属实例 |
| `kind` | string | 内容类型，当前取值为 `post` 或 `reply` |
| `content` | string | 正文 |
| `visibility` | string | 可见性，当前原型主要为 `public` |
| `storageUri` | string | 内容存储 URI |
| `attestationUri` | string | 存证 URI |
| `tags` | string[] | 标签数组 |
| `media` | PostMedia[] \| null | 附件媒体 |
| `parentPostId` | string | 直接父帖 ID；顶层帖子为空 |
| `rootPostId` | string | 所在线程的根帖 ID |
| `replyDepth` | number | 回复深度，根帖为 `0` |
| `replies` | number | 直接回复数 |
| `boosts` | number | 转发数 |
| `likes` | number | 喜欢数 |
| `createdAt` | string | 创建时间 |

### 2.2 关系规则

- 顶层帖子：
  - `kind = "post"`
  - `parentPostId = ""`
  - `rootPostId = ""`
  - `replyDepth = 0`
- 回复帖子：
  - `kind = "reply"`
  - `parentPostId` 指向直接父帖
  - `rootPostId` 指向整条线程的根帖
  - `replyDepth >= 1`

### 2.3 时间线规则

为了和主流社交产品体验一致：

- `GET /api/v1/social/feed` 只返回顶层帖子
- `GET /api/v1/social/bootstrap` 中的 `feed` 也只返回顶层帖子
- 楼层评论通过 `thread / replies` 获取
- `replies` 返回的是当前帖子下面的回复子树，而不只是第一层

这意味着首页时间线不会把评论直接混入主流中。

## 3. 通用实体

### 3.1 用户 `SocialUser`

```json
{
  "id": "user_archive",
  "handle": "@archive",
  "displayName": "Whale Archive",
  "bio": "为创作者提供永久内容归档与链上身份锚定。",
  "instance": "vault.social",
  "wallet": "0xa18f...3c92",
  "avatarUrl": "https://example.com/avatar.png",
  "followers": 1284,
  "following": 312,
  "createdAt": "2026-04-05T10:22:41Z"
}
```

### 3.2 媒体 `MediaAsset`

```json
{
  "id": "media_manifesto",
  "ownerId": "user_archive",
  "name": "genesis-manifesto.png",
  "kind": "image",
  "url": "https://example.com/manifesto.png",
  "storageUri": "ar://7xv91manifesto",
  "cid": "bafybeih7f4manifesto",
  "sizeBytes": 2400000,
  "status": "mirrored",
  "createdAt": "2026-04-06T14:22:41Z"
}
```

### 3.3 会话 `Conversation`

```json
{
  "id": "conv_curator",
  "title": "Archive Curator",
  "participantIds": ["user_archive", "user_librarian"],
  "encrypted": true,
  "messages": [],
  "updatedAt": "2026-04-07T10:07:41Z"
}
```

## 4. 接口目录

### 4.1 健康检查

#### `GET /healthz`

用于检查后端、Redis、Relay 是否可用。

示例响应：

```json
{
  "ok": true,
  "redis": true,
  "relayReady": true,
  "service": "whale-vault-social-backend",
  "socialStats": {
    "users": 3,
    "posts": 7,
    "mediaAssets": 2,
    "conversations": 1
  }
}
```

### 4.2 Bootstrap

#### `GET /api/v1/social/bootstrap?limit=20`

前端初始化聚合接口，适合首页首次加载。

#### 查询参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `limit` | int | feed / media / conversations 等集合的裁剪上限 |

#### 响应数据

```json
{
  "ok": true,
  "data": {
    "currentUser": {},
    "stats": {},
    "feed": [],
    "users": [],
    "media": [],
    "conversations": [],
    "instances": []
  }
}
```

#### 说明

- `feed` 只包含顶层帖子
- 评论不会出现在这里
- 当前前端首页主要依赖此接口初始化

### 4.3 用户

#### `GET /api/v1/social/users`

获取用户列表。

#### `POST /api/v1/social/users`

创建用户。

请求示例：

```json
{
  "handle": "@newcomer",
  "displayName": "New Comer",
  "bio": "联邦社交体验者",
  "instance": "vault.social",
  "wallet": "0x1234",
  "avatarUrl": "https://example.com/avatar.png"
}
```

#### `GET /api/v1/social/users/{id}`

获取指定用户详情。

### 4.4 时间线与帖子

#### `GET /api/v1/social/feed?limit=20`

获取公共时间线。

#### 说明

- 只返回 `kind = "post"` 的顶层帖子
- 不返回回复

#### `GET /api/v1/social/posts/{id}`

获取单条帖子详情。

这个接口只返回单条帖子本身，不带线程上下文。

#### `POST /api/v1/social/posts`

创建帖子或回复。

##### 创建顶层帖子示例

```json
{
  "authorId": "user_archive",
  "kind": "post",
  "content": "把内容、关系和媒体一起做成可迁移的数字资产。",
  "visibility": "public",
  "storageUri": "ar://post-001",
  "attestationUri": "attestation://post/001",
  "tags": ["数字主权", "内容归档"],
  "mediaIds": []
}
```

##### 创建回复示例

```json
{
  "authorId": "user_librarian",
  "kind": "reply",
  "content": "同意，而且回复本身也应该是可检索、可迁移的内容对象。",
  "visibility": "public",
  "storageUri": "ar://reply-001",
  "attestationUri": "attestation://reply/001",
  "tags": ["线程回复"],
  "mediaIds": [],
  "parentPostId": "post_archive",
  "rootPostId": "post_archive"
}
```

##### 行为说明

- 如果传入 `parentPostId`，后端会将该内容视为回复
- `rootPostId` 为空时，后端会根据父帖自动补全
- `replyDepth` 由后端自动计算
- 父帖不存在时返回 400

### 4.5 线程与回复

#### `GET /api/v1/social/posts/{id}/thread?limit=20`

获取帖子线程。

#### 响应结构

```json
{
  "ok": true,
  "data": {
    "post": {},
    "ancestors": [],
    "replies": []
  }
}
```

#### 字段说明

- `post`：当前聚焦的帖子
- `ancestors`：从根帖到当前帖父节点的祖先链
- `replies`：当前帖子下面的回复子树，按线程顺序展开

#### 适用场景

- 帖子详情页
- 线程上下文查看
- 楼层页渲染

#### `GET /api/v1/social/posts/{id}/replies?limit=20`

获取某条帖子下面的回复子树。

#### 说明

- 适合前端只刷新楼层列表时单独调用
- 可用于根帖详情页渲染楼中楼
- 当前前端帖子详情页就是 `thread + replies` 组合驱动

### 4.6 媒体

#### `GET /api/v1/social/media?limit=20`

获取媒体资产列表。

#### `POST /api/v1/social/media`

创建媒体元数据记录。

请求示例：

```json
{
  "ownerId": "user_archive",
  "name": "manifesto.png",
  "kind": "image",
  "url": "data:image/png;base64,...",
  "storageUri": "preview://1712500000",
  "cid": "draft-k3x9w",
  "sizeBytes": 2400000,
  "status": "uploaded"
}
```

#### 说明

当前阶段这里记录的是媒体元数据，不是完整对象存储上传流水线。

### 4.7 会话与消息

#### `GET /api/v1/social/conversations?limit=20`

获取会话列表。

#### `POST /api/v1/social/conversations`

创建会话。

请求示例：

```json
{
  "title": "Archive Curator",
  "participantIds": ["user_archive", "user_librarian"],
  "encrypted": true
}
```

#### `GET /api/v1/social/conversations/{id}`

获取单个会话详情。

#### `POST /api/v1/social/conversations/{id}/messages`

向会话追加消息。

请求示例：

```json
{
  "senderId": "user_archive",
  "body": "我们下一步把线程页和私信实时化。"
}
```

### 4.8 联邦实例

#### `GET /api/v1/social/instances`

获取联邦实例列表。

示例响应：

```json
{
  "ok": true,
  "data": [
    {
      "name": "vault.social",
      "focus": "创作者主权与链上身份",
      "members": "12.4k",
      "latency": "43 ms",
      "status": "healthy"
    }
  ]
}
```

## 5. 错误处理约定

### 5.1 常见状态码

| 状态码 | 场景 |
| --- | --- |
| `200` | 查询成功 |
| `201` | 创建成功 |
| `400` | 请求参数错误，例如父帖不存在、JSON 非法 |
| `404` | 资源不存在 |

### 5.2 典型错误

#### 父帖不存在

```json
{
  "ok": false,
  "error": "parent post not found: post_xxx"
}
```

#### 帖子不存在

```json
{
  "ok": false,
  "error": "post not found: post_xxx"
}
```

## 6. 前端对接建议

当前 Vue 前端建议按以下方式使用接口：

1. 首页初始化：`GET /api/v1/social/bootstrap`
2. 发图文：
   - 先 `POST /api/v1/social/media`
   - 再 `POST /api/v1/social/posts`
3. 帖子详情页：
   - `GET /api/v1/social/posts/{id}/thread`
   - `GET /api/v1/social/posts/{id}/replies`
4. 私信发送：`POST /api/v1/social/conversations/{id}/messages`

## 7. 后续演进建议

当系统进入生产级阶段，建议继续补充以下接口能力：

- 关注 / 取关
- 点赞 / 取消点赞
- 转发 / 引用转发
- 帖子删除与软删除
- 通知中心
- WebSocket 实时消息
- ActivityPub 出站与入站收件箱
- 媒体真实对象存储上传
- 搜索与索引能力
