# go_eth

Gin + SIWE（EIP-4361）登录 + ERC-20 Transfer 索引入库 + 查询接口。

## 你会得到什么
- **后台索引器**：从以太坊 RPC 拉取指定 ERC-20 合约的 `Transfer` 事件写入 SQLite，支持断点续跑与确认数（防 reorg）。
- **SIWE 登录**：`/auth/siwe/nonce` 生成一次性 nonce；`/auth/siwe/login` 校验 message + signature，成功后返回 JWT。
- **查询接口**：`/api/transfers` 返回“登录地址”的转账记录（from/to 任一匹配），支持游标分页。

## 快速启动（Windows）
如果你本机设置了错误的 `GOROOT`，可按下面方式临时清掉环境变量，让 `go` 使用它自带的 GOROOT：

```bat
cd F:\GoWorkspace\eth_project\go_eth
set GOROOT=
set GOSUMDB=off
set JWT_SECRET=change-me
go run .\cmd\server
```

健康检查：

```bash
curl http://127.0.0.1:8080/healthz
```

## 配置（环境变量）
- `HTTP_ADDR`：HTTP 监听地址（默认 `:8080`）
- `ETH_RPC_URL`：以太坊 RPC（默认 `https://sepolia.drpc.org`）
- `TOKEN_ADDRESS`：Token 合约地址（默认 `0x0b18F517d8e66b3bd6fB799d44A0ebee473Df20C`）
- `TOKEN_ADDRESSES`：Token 合约地址列表（逗号分隔，形如 `0x..,0x..`）。`TOKEN_ADDRESSES` 优先，若为空则回退使用 `TOKEN_ADDRESS`。
- `CHAIN_ID`：链 ID（默认 `11155111`）
- `START_BLOCK`：可选；指定从哪个区块高度开始补历史。不填则从启动时最新区块（减去确认数）开始，只索引未来。
- `CONFIRMATIONS`：确认数（默认 `6`）
- `POLL_INTERVAL`：轮询间隔（默认 `8s`）
- `DB_DSN`：SQLite DSN（默认 `file:go_eth.db?_busy_timeout=5000&_foreign_keys=1&_journal_mode=WAL`）
- `JWT_SECRET`：JWT 密钥（必填）
- `ALLOWED_DOMAIN`：可选；限制 SIWE 的 `domain`

## API（给前端）

### 通用约定
- **Base URL**：`http://127.0.0.1:8080`（本地默认）
- **Content-Type**：除特别说明外，POST 请求使用 `application/json`
- **鉴权方式**：除 `/healthz` 与 `/auth/siwe/*` 外，其余接口都需要
  - `Authorization: Bearer <jwt>`
- **错误返回格式**：发生错误时，返回 JSON：

```json
{"error":"<message>"}
```

- **地址大小写**：服务端会将地址统一转为小写存储/比较；前端展示如需 checksum 可自行转换。

### 鉴权流程（SIWE -> JWT）
1) `POST /auth/siwe/nonce` 获取 nonce  
2) 使用钱包对 **SIWE 原始 message 文本**做 `personal_sign`（ethers.js：`signMessage`）  
3) `POST /auth/siwe/login` 换取 JWT  
4) 调用 `/api/*` 时带上 `Authorization: Bearer <jwt>`

### 1) 获取 nonce
`POST /auth/siwe/nonce`

用途：获取一次性 nonce，用于构造 SIWE message（防重放）。

请求：无请求体。

响应：

```json
{"nonce":"<random>"}
```

状态码：
- `200`：成功
- `500`：生成/落库 nonce 失败

### 2) SIWE 登录
`POST /auth/siwe/login`

请求体：

```json
{
  "message": "<siwe_message_text>",
  "signature": "0x..."
}
```

说明：
- `message` 必须是钱包签名的**原始文本**（EIP-191 personal_sign）。
- 服务端会校验 `Chain ID` 是否等于 `CHAIN_ID`，并校验 nonce 未过期且未使用。
- 若配置了 `ALLOWED_DOMAIN`，会校验 message 的 `domain` 是否匹配。

推荐 message 模板（字段名必须包含：`URI`、`Version`、`Chain ID`、`Nonce`、`Issued At`）：

```text
{domain} wants you to sign in with your Ethereum account:
{address}

{statement}

URI: {uri}
Version: 1
Chain ID: {chainId}
Nonce: {nonce}
Issued At: {issuedAtRFC3339}
```

前端（ethers.js）示例：

```ts
// 1) 取 nonce
const { nonce } = await fetch(`${baseUrl}/auth/siwe/nonce`, { method: "POST" }).then((r) => r.json());

// 2) 拼 message 并 personal_sign（signMessage）
const address = await signer.getAddress();
const issuedAt = new Date().toISOString();
const domain = window.location.host; // 也可用你自己的业务域名字符串
const chainId = (await signer.provider!.getNetwork()).chainId;

const message = `${domain} wants you to sign in with your Ethereum account:
${address}

Sign in to Go-ETH.

URI: ${window.location.origin}
Version: 1
Chain ID: ${chainId}
Nonce: ${nonce}
Issued At: ${issuedAt}`;

const signature = await signer.signMessage(message);

// 3) 换 JWT
const loginResp = await fetch(`${baseUrl}/auth/siwe/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ message, signature }),
}).then((r) => r.json());

const token = loginResp.token;
```

响应：

```json
{"token":"<jwt>","address":"0x..."}
```

状态码：
- `200`：成功
- `400`：message 格式不合法 / chainId 不匹配 / domain 不允许
- `401`：nonce 无效/过期/已使用 / 签名无效 / 签名地址与 message 地址不一致
- `500`：DB 更新 nonce 已用状态失败 / JWT 签名失败

### 3) 查询登录地址转账记录
`GET /api/transfers?limit=50&cursor=...&tokenAddress=...`

Header：
- `Authorization: Bearer <jwt>`

Query 参数：
- `limit`：可选；返回条数，范围 **1-200**，默认 `50`
- `cursor`：可选；分页游标（由接口返回的 `nextCursor` 透传即可；内容对前端应视为不透明）
- `tokenAddress`：可选；筛选指定 token 合约地址的 Transfer（地址会被统一转为 lower-case）；不填则返回所有 token 的记录

分页规则：
- 排序为 **`blockNumber` 倒序**，同区块内按 **`logIndex` 倒序**
- `nextCursor` 表示“下一页从哪条之后开始取”（严格小于上一页最后一条的二元组）

响应：

```json
{
  "items": [
    {
      "tokenAddress": "0x...",
      "txHash": "0x...",
      "logIndex": 12,
      "blockNumber": 123,
      "from": "0x...",
      "to": "0x...",
      "amount": "1000000000000000000"
    }
  ],
  "nextCursor": "..."
}
```

状态码：
- `200`：成功
- `400`：cursor 不合法
- `401`：缺少/非法 JWT
- `500`：DB 错误

curl 示例：

```bash
# 首页
curl -H "Authorization: Bearer <jwt>" "http://127.0.0.1:8080/api/transfers?limit=50"

# 下一页（把 nextCursor 透传回来）
curl -H "Authorization: Bearer <jwt>" "http://127.0.0.1:8080/api/transfers?limit=50&cursor=<nextCursor>"
```

### 4) 健康检查
`GET /healthz`

响应：

```json
{"ok":true}
```
