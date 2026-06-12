# 环境变量

完整模板见仓库根目录 [`.env.example`](https://github.com/liuwenji007/regulus-academy/blob/main/.env.example)。修改后需重启后端（`go run ./cmd/server` 或 `docker compose restart`）。

Cloud 部署模板：[`deploy/railway/env.cloud.example`](https://github.com/liuwenji007/regulus-academy/blob/main/deploy/railway/env.cloud.example)

## LLM 与基础服务

| 变量 | 默认 | 说明 |
|------|------|------|
| `LLM_PROVIDER` | `deepseek` | 提供商：`deepseek` / `openai` / `openrouter` / `ollama` / `custom` |
| `LLM_API_KEY` | — | 模型 API Key（推荐统一使用此变量） |
| `LLM_BASE_URL` | 按提供商 | 只填域名前缀，**不要**带 `/v1/chat/completions` |
| `LLM_MODEL` | 按提供商 | 可选，覆盖预设模型名 |
| `DEEPSEEK_API_KEY` | — | 兼容旧变量；未设 `LLM_API_KEY` 时生效 |
| `DEEPSEEK_BASE_URL` | `https://api.deepseek.com` | DeepSeek API 根地址 |
| `PORT` | `8080` | HTTP 服务端口 |
| `HOST_PORT` | `8080` | Docker 宿主机映射端口；`install.sh` 在 8080 占用时会自动改写 |
| `DATABASE_PATH` | `./data/regulus.db` | SQLite 数据库路径 |
| `REGULUS_COACH_ROOT` | 镜像内 `/app/regulus-coach` | Coach 资源目录；本地 `go run` 一般无需设置 |

## 教练点亮（教学流程）

与 [教练流程](../guide/coach-flow.md) 直接相关。

| 变量 | 默认 | 说明 |
|------|------|------|
| `REGULUS_STRICT_CONCEPT_COVERAGE` | `1`（开） | 设为 `0` 关闭「核心概念 ≥3 且未考 ≥2 时建议再练」 |
| `REGULUS_REQUIRE_APPLY_EXERCISE` | `1`（开） | 设为 `0` 关闭熟悉/精通层的应用级练习建议（入门层始终不要求 apply） |
| `REGULUS_LLM_COMPLETION_CHECK` | `1`（开） | 设为 `0` 关闭点亮前 LLM 综合评估，回退纯规则硬挡 + 传统掌握度评估 |

### 组合示例

**默认（推荐）** — 三项均为 `1`：

- 规则给出覆盖 / apply **建议**
- LLM 结合对话做最终 `ready` 裁决，可软豁免
- 答对但 `not ready` 时自动连题

**严格规则** — `REGULUS_LLM_COMPLETION_CHECK=0`：

- 练习答对：规则满足则必须再练，否则直接点亮
- 申请掌握：`not ready` 时不自动连题，再次申请可强制完成
- 适合弱模型或需要确定性行为的自托管

**宽松练习** — `REGULUS_STRICT_CONCEPT_COVERAGE=0` 且 `REGULUS_REQUIRE_APPLY_EXERCISE=0`：

- 仅保留 LLM 综合评估（若仍开启 `REGULUS_LLM_COMPLETION_CHECK`）
- 单题答对更容易点亮；仍会因明显缺口被模型拒绝

## LLM 超时与建课

| 变量 | 默认 | 说明 |
|------|------|------|
| `REGULUS_LLM_TIMEOUT_SEC` | `240` | 单次 LLM HTTP 超时（秒）；慢速 API 可调到 300～600 |
| `REGULUS_DOMAIN_BUILD_TIMEOUT_SEC` | `360` | `/domain/build`、regenerate 整请求超时（秒） |
| `REGULUS_TREE_CRITIQUE` | `1`（开） | 设为 `0` 关闭建树后 LLM 质检（critique 重生成） |

## 材料导入（C1）

| 变量 | 默认 | 说明 |
|------|------|------|
| `REGULUS_INGEST_MAX_PDF_BYTES` | `20971520` | PDF 最大体积（字节，约 20MB） |
| `REGULUS_INGEST_MAX_PDF_PAGES` | `200` | PDF 最大页数 |
| `REGULUS_INGEST_MAX_PDF_CHARS` | `600000` | PDF 提取正文上限（字符） |
| `REGULUS_INGEST_MAX_URL_CHARS` | `80000` | 网页正文上限（字符） |
| `REGULUS_INGEST_FETCH_TIMEOUT_SEC` | `15` | 抓取 URL 超时（秒） |

## 纵深扩展（C2）

| 变量 | 默认 | 说明 |
|------|------|------|
| `REGULUS_EXTEND_MIN_RATIO` | `0.8` | 解锁「纵深扩展」所需的课程完成度（80%） |

## Cloud Demo 专用

本地自托管**勿设置** `REGULUS_DEPLOYMENT=cloud`。

| 变量 | 说明 |
|------|------|
| `REGULUS_DEPLOYMENT` | 设为 `cloud` 启用在线版策略 |
| `ADMIN_TOKEN` | 管理接口与 `#/admin` 登录（cloud 模式必填） |
| `REGULUS_CLOUD_ENCRYPTION_KEY` | BYOK 加密密钥（`openssl rand -hex 32`） |
| `REGULUS_CLOUD_QUOTA_DAILY_MESSAGES` | 每用户每日免费教练消息数（默认 20） |
| `REGULUS_CLOUD_GITHUB_URL` | 页脚 GitHub 链接 |
| `REGULUS_CLOUD_DOCS_URL` | 页脚文档链接 |
| `REGULUS_CLOUD_DEMO_URL` | 页脚在线 Demo 链接 |
| `REGULUS_CLOUD_MAX_BUILD_JOBS_GLOBAL` | 全局建课并发上限 |
| `REGULUS_CLOUD_RATE_LIMIT_PER_IP` | 每 IP 每分钟请求上限 |

自托管若需保护管理接口，可单独设置 `ADMIN_TOKEN`（`Authorization: Bearer <token>`）。

## IM Gateway

| 变量 | 默认 | 说明 |
|------|------|------|
| `GATEWAY_ENABLED` | `false` | 是否启用 IM Gateway |
| `GATEWAY_PUBLIC_URL` | — | 公网访问地址（展示 webhook 回调 URL） |

### Telegram

| 变量 | 默认 | 说明 |
|------|------|------|
| `TELEGRAM_ENABLED` | `true` | Gateway 开启后是否启用 Telegram |
| `TELEGRAM_BOT_TOKEN` | — | @BotFather 创建的 Bot Token |
| `TELEGRAM_ALLOWED_USERS` | — | 可选，逗号分隔 user id 白名单 |

### 钉钉

| 变量 | 说明 |
|------|------|
| `DINGTALK_ENABLED` | 是否启用（默认 `true`，需 Gateway 开） |
| `DINGTALK_CLIENT_ID` | 开放平台 AppKey |
| `DINGTALK_CLIENT_SECRET` | AppSecret |

### 飞书

| 变量 | 默认 | 说明 |
|------|------|------|
| `FEISHU_ENABLED` | `true` | 是否启用 |
| `FEISHU_MODE` | `websocket` | `websocket`（内网可用）或 `webhook`（需公网 HTTPS） |
| `FEISHU_APP_ID` | — | 应用 App ID |
| `FEISHU_APP_SECRET` | — | 应用 App Secret |
| `FEISHU_VERIFY_TOKEN` | — | webhook 模式推荐 |
| `FEISHU_ALLOWED_USERS` | — | 可选，open_id 白名单 |

### 企业微信

| 变量 | 说明 |
|------|------|
| `WECOM_ENABLED` | 默认 `false`；需公网 HTTPS 回调 |
| `WECOM_CORP_ID` / `WECOM_AGENT_ID` / `WECOM_SECRET` | 应用凭证 |
| `WECOM_TOKEN` / `WECOM_ENCODING_AES_KEY` | 回调加解密 |
| `WECOM_ALLOWED_USERS` | 可选用户白名单 |

IM 配置步骤见 [自托管部署](../guide/self-host.md)。

## Langfuse（开发期追踪）

默认关闭；Docker / 生产建议保持 `false`。

| 变量 | 默认 | 说明 |
|------|------|------|
| `LANGFUSE_ENABLED` | `false` | 开启后向 Langfuse 导出 OTLP trace |
| `LANGFUSE_PUBLIC_KEY` | — | Langfuse Public Key |
| `LANGFUSE_SECRET_KEY` | — | Langfuse Secret Key |
| `LANGFUSE_BASE_URL` | `http://localhost:3000` | Langfuse 根地址 |
| `LANGFUSE_ENVIRONMENT` | `development` | trace 环境标签 |
| `LANGFUSE_LOG_CONTENT` | `true` | `false` 时不记录 prompt/completion 正文 |

## 相关文档

- [教学模式](../guide/teaching-model.md)
- [教练流程](../guide/coach-flow.md)
- [自托管部署](../guide/self-host.md)
