# Regulus Coach 精简协议（Agent-lite）

供 **未关联 Regulus Web** 且 **未安装 regulus CLI** 时使用。完整状态机见仓库内 [`protocol.md`](./protocol.md)（**不含**在用户下载的 lite zip 中）。

## 阶段（Phase）

| Phase | 含义 | 用户动作 |
|-------|------|----------|
| `explain` | 讲解与答疑 | 提问；说「开始练习」进入出题 |
| `exercise` | 已出题，等待作答 | 提交答案；可说「不懂，回讲解」 |
| `review` | 未通过后补讲 | 提问；说「开始练习」再练 |
| `completed` | 本节点通过 | 进入下一节点（读 `tree.yaml` 顺序） |

## 转换（简化）

```
explain --[开始练习/准备好了]--> exercise
exercise --[提交答案]--> grade
grade --[未通过]--> review
grade --[通过]--> completed
review --[开始练习]--> exercise
exercise --[不懂/回讲解]--> explain
completed --> 下一节点 explain
```

**不包含** Web 专属的 `completion_readiness` 硬挡、画像回顾、`mastery_check` 多轮评估。Agent 批改时参考 [agent-prompts.md](./agent-prompts.md) 与 `schemas/grade.json`。

## Agent 必读材料

| 文件 | 用途 |
|------|------|
| `domains/<slug>/tree.yaml` | 课程结构与节点顺序 |
| `domains/<slug>/nodes/<key>.yaml` | 节点核心概念、练习提示 |
| [agent-prompts.md](./agent-prompts.md) | 讲解 / 出题 / 批改要点 |
| `schemas/exercise.json` | 出题 JSON 结构 |
| `schemas/grade.json` | 批改 JSON 结构 |
| `data/progress.json` | 本地进度（见 `schemas/progress.schema.json`） |

## 进度

每完成一个节点，更新 `data/progress.json` 中对应 `slug` + `nodeKey` 的 `status`（`in_progress` / `completed`）与 `mastery`（0～1）。格式见 `schemas/progress.schema.json`。

## 与 Linked 模式

若存在 `.regulus/link.json`，**优先**用 [scripts/api-session.sh](./scripts/api-session.sh) 调远程 API，勿自行扮演教练。进度以服务端为准。

## 与 CLI 模式

若已安装 `bin/regulus`，可改用 `regulus session`，与 Web 共用完整状态机（见 [USAGE.md](./USAGE.md)）。
