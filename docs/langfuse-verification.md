# Langfuse 开发期验收清单

在 `.env` 中设置 `LANGFUSE_ENABLED=true` 并配置自建实例后，重启 `go run ./cmd/server`。

## Tracing 过滤

- Langfuse UI → **Tracing** → `environment = development`

## 必现 trace（v1）

| Trace | 触发方式 |
|-------|----------|
| `coach.message` | Web/IM 教练页发送一条消息 |
| `coach.begin` | 新开节点或「下一节」 |
| `coach.profile_refresh` | 节点点亮后（异步，稍晚出现） |
| `domain.build` | 首页/树页建课 |
| `domain.intent` | 建课流程中的意图分析（`domain.build` 子 span） |
| `domain.personalize` | Skill 包 + 目标/画像个性化建课 |
| `channel.nav` | IM 发送模糊导航句（规则未命中时走 LLM） |

## Generation 检查

- 子 span 名如 `coach.explain_qa`、`coach.grade`、`domain.build_tree`
- `gen_ai.request.model` 与耗时存在
- `LANGFUSE_LOG_CONTENT=true` 时可见 prompt/completion
- `ChatJSON` 解析失败重试：同一 trace 下 **两条** 同名或连续 generation

## 安全

- span 中无 `LLM_API_KEY`
- `llm.Ping` 不产生 generation

## 关闭

- `LANGFUSE_ENABLED=false` 重启后无 OTLP 流量，现有单元测试不依赖网络
