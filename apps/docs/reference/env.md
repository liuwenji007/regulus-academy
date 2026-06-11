# 环境变量

完整模板见仓库根目录 [`.env.example`](https://github.com/liuwenji007/regulus-academy/blob/main/.env.example)。

## 核心

| 变量 | 说明 |
|------|------|
| `LLM_PROVIDER` | deepseek / openai / openrouter / ollama / custom |
| `LLM_API_KEY` | 模型 API Key |
| `DATABASE_PATH` | SQLite 路径 |

## Cloud Demo 专用

| 变量 | 说明 |
|------|------|
| `REGULUS_DEPLOYMENT` | 设为 `cloud` 启用在线版策略 |
| `ADMIN_TOKEN` | 管理接口与 `#/admin` 登录 |
| `REGULUS_CLOUD_ENCRYPTION_KEY` | BYOK 加密密钥 |
| `REGULUS_CLOUD_QUOTA_DAILY_MESSAGES` | 每用户每日免费消息数 |
| `REGULUS_CLOUD_GITHUB_URL` | 页脚 GitHub 链接 |
| `REGULUS_CLOUD_DOCS_URL` | 页脚文档链接 |

Cloud 模板：[`deploy/railway/env.cloud.example`](https://github.com/liuwenji007/regulus-academy/blob/main/deploy/railway/env.cloud.example)
