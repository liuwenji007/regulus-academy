# AI 模型配置

入口：设置 → **AI 模型**

Regulus 通过 OpenAI 兼容接口调用大模型完成讲解、出题、批改与建课。配置好 API Key 后，首页与左下角会显示「模型已连接」。

## 三种配置方式

| 方式 | 适合谁 | 说明 |
|------|--------|------|
| **Web 设置页** | 所有用户 | 设置 → AI 模型，添加模型卡片，保存即生效 |
| **`.env` 文件** | 自托管 / Docker | 首次启动前写入 `LLM_API_KEY`；可与 Web 配置并存 |
| **Cloud 自带 Key** | 在线 Demo 额度用尽后 | 在设置页填写自己的 Key，消息走你的账户计费 |

左下角菜单可**快速切换当前使用的模型**（无需每次进设置页）。

## Web 设置页怎么用

1. 打开 **设置 → AI 模型**
2. 填写 **显示名称**、**接口类型**（DeepSeek / OpenAI / Ollama 等）、**模型 ID**
3. 按需填写 **API Key**（可留空以沿用全局 Key）
4. 点 **测试** 探测连接，点 **保存** 写入配置
5. 在左下角将本条设为「使用中」

支持添加多条模型配置（如「DeepSeek 主力」「本地 Ollama」），随时切换。

### 支持的接口类型

| 类型 | 说明 |
|------|------|
| `deepseek` | 默认推荐；Key 见 [DeepSeek 开放平台](https://platform.deepseek.com/api_keys) |
| `openai` | OpenAI 官方 API |
| `openrouter` | 多模型聚合 |
| `ollama` | 本地 Ollama，通常无需 Key |
| `custom` | 任意 OpenAI 兼容接口，须填 **Base URL**（只填域名，不要带 `/v1/chat/completions`） |

保存后写入 `data/llm-profiles.json`；若保存的是当前使用中的模型，会同步更新 `.env` 运行时配置。

### 场景模型绑定（可选）

在「AI 模型」页可为不同场景指定不同模型卡片，例如：

| 场景 | 说明 |
|------|------|
| 教练主线 | 讲解 / 出题 / 批改 |
| **划词助教** | 划词术语卡与短问答（可选用更轻量、更便宜的模型） |
| 建课 / 行动助手等 | 见设置页下拉 |

助教未单独指定时，回退到当前「使用中」的默认模型。说明见 [划词助教](./aside-assistant.md)。

## 自托管：`.env` 快速配置

Docker 或 `go run` 启动前，在仓库根目录：

```bash
cp .env.example .env
```

```bash
LLM_PROVIDER=deepseek
LLM_API_KEY=sk-...
# LLM_MODEL=deepseek-chat    # 可选
# LLM_BASE_URL=              # custom 时必填
```

修改 `.env` 后需**重启后端**（`docker compose restart` 或重新 `go run`）。

## Cloud 在线 Demo

- 无需 Key 即可开始：平台提供每日免费教练消息额度
- 额度用尽后，在设置页填写自己的 Key 继续使用
- 首页会展示今日剩余额度；用尽时页面会提示配置 Key

详见 [在线体验版](./cloud-demo.md)。

## 教练行为调优（可选）

除模型本身外，还可通过环境变量调整「练多少才点亮」等行为，与模型选择无关：

| 变量 | 作用 |
|------|------|
| `REGULUS_STRICT_CONCEPT_COVERAGE` | 多核心概念时是否建议多练几题 |
| `REGULUS_REQUIRE_APPLY_EXERCISE` | 熟悉/精通层是否建议应用级练习 |
| `REGULUS_LLM_COMPLETION_CHECK` | 点亮前是否做 LLM 综合评估 |

完整列表与组合示例见 [环境变量](../reference/env.md#教练点亮教学流程)。

## 常见问题

**首页显示「未配置 API Key」**  
自托管：检查 `.env` 中 `LLM_API_KEY` 并重启后端；或在 Web 设置页保存一条带 Key 的模型并设为使用中。

**建课很慢或超时**  
可调大 `REGULUS_LLM_TIMEOUT_SEC`、`REGULUS_DOMAIN_BUILD_TIMEOUT_SEC`，或换更快的模型。见 [环境变量](../reference/env.md#llm-超时与建课)。

**Ollama 连不上**  
确认本机 `ollama serve` 已启动，且 `LLM_BASE_URL` 指向正确地址（默认 `http://localhost:11434`）。

## 相关

- [快速上手](./quick-start.md)
- [环境变量](../reference/env.md)
- [自托管部署](./self-host.md)
