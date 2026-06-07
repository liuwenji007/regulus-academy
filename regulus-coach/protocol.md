# Regulus Academy Coach 协议（总览）

本文件供 Skill 用户阅读；App 运行时从 `prompts/`、`schemas/`、`triggers.yaml` 加载。

## 状态机（Phase）

| Phase | 含义 | 用户典型动作 |
|-------|------|--------------|
| `explain` | 讲解与答疑 | 提问；回复「开始练习」进入练习；可说「已经掌握，下一节」申请完成 |
| `exercise` | 已出题，等待作答 | 提交答案；可说「不懂/回讲解」；可说「换一题」；可说「已经掌握，下一节」 |
| `review` | 首次未通过后补讲 | 提问；回复「开始练习」再练一题；可说「已经掌握，下一节」 |
| `completed` | 本节点通过 | Web 点「继续 · 下一节」或 IM 说「下一节」进入下一节点；也可返回知识树选节点 |

### 阶段转换

```
explain --[开始练习/准备好了/出题/来一题]--> exercise
exercise --[提交答案]--> grade
  grade --[通过]--> completed
  grade --[未通过]--> review（中文反馈；可说「不懂，回讲解」或「开始练习」）
  explain|exercise|review --[已经掌握，下一节]--> mastery_check
    mastery_check --[ready]--> completed
    mastery_check --[未 ready]--> 原 phase（提示薄弱点；再次坚持则 completed 并记易错）
review --[开始练习]--> exercise
exercise --[不懂/回讲解]--> explain
exercise --[换一题]--> exercise（新题）
```

### 触发词

App 运行时从 [`triggers.yaml`](./triggers.yaml) 加载；Skill 手动推进时可参考该文件。

## Prompt 模块

| 文件 | 用途 |
|------|------|
| [`prompts/core.md`](./prompts/core.md) | 角色、学习方式、与 App 的行为边界（传入 LLM system） |
| [`prompts/phase_explain.md`](./prompts/phase_explain.md) | 讲解/答疑 |
| [`prompts/phase_review.md`](./prompts/phase_review.md) | 巩固答疑（review） |
| [`prompts/phase_exercise.md`](./prompts/phase_exercise.md) | 出题 |
| [`prompts/phase_grade.md`](./prompts/phase_grade.md) | 批改 |
| [`prompts/phase_mastery.md`](./prompts/phase_mastery.md) | 掌握度评估（用户说「已经掌握，下一节」） |
| [`prompts/phase_profile_refresh.md`](./prompts/phase_profile_refresh.md) | 节末画像回顾（App 自动，用户不可见） |
| [`prompts/phase_profile_init.md`](./prompts/phase_profile_init.md) | 新用户引导冷启动画像（App 自动） |
| [`prompts/phase_profile_merge.md`](./prompts/phase_profile_merge.md) | 设置页对话补充画像（App 自动） |

## JSON Schema

- 出题：[`schemas/exercise.json`](./schemas/exercise.json)
- 批改：[`schemas/grade.json`](./schemas/grade.json)
- 掌握度评估：[`schemas/mastery_check.json`](./schemas/mastery_check.json)
- 节末画像合并：[`schemas/profile_refresh.json`](./schemas/profile_refresh.json)
- 引导冷启动画像：[`schemas/profile_init.json`](./schemas/profile_init.json)
- 设置页画像补充：[`schemas/profile_merge.json`](./schemas/profile_merge.json)

## 学生画像（App 自动）

### 新用户引导（`profile_init`）

首次进入 Web 时，用户可回答 2～3 个引导问题（可跳过）。App 调用 `profile_init` 将答案压缩为 `profile_summary`（≤500 字），写入 `users.profile_summary` 并标记 `onboarded_at`。建课与个性化裁剪会注入该画像。

### 设置页补充（`profile_merge`）

用户在设置页提交补充说明时，App 调用 `profile_merge`：输入旧 `profile_summary` + 用户补充 → 合并 ≤500 字后写回。

### 节末回顾（`profile_refresh`）

节点点亮（`completed`）后，App **异步**调用 `profile_refresh`：读取本节对话摘录 + 旧 `profile_summary`，合并写入用户画像（≤500 字）。失败不影响点亮；下一节讲解/答疑自动注入新画像。

## App 自动注入的上下文

- 当前节点 YAML：`core_concepts`、`common_mistakes`、`boundaries`、`exercise_ideas`、`grading_hints`（批改时）
- 用户已完成节点摘要、可选巩固概念、本次薄弱点、学生画像
- 本会话已考查 / 待覆盖核心概念（出题、批改、掌握度评估；用于节点内多概念覆盖）
- 对话历史（条数按任务类型调整）

## 节点内多概念覆盖

- 每题 `reinforced_concepts` 记入会话「已考查」；上下文中的【待覆盖】列出尚未练到的 `core_concepts`。
- 核心概念 ≥3 且仍有 ≥2 个未在练习中考到时，批改通过或掌握度 `ready` 不会立即点亮，需再练一题（可通过环境变量 `REGULUS_STRICT_CONCEPT_COVERAGE=0` 关闭）。

## Skill 与 App（运维说明，不传入 LLM）

- `prompts/`、`domains/`、`schemas/`、`triggers.yaml` 为本目录运行时真相源
- App（Go 后端）从同目录加载；Skill 在 IDE 中手动按状态机推进（见上文 Phase）
- App 额外持久化：SQLite 进度、错题、会话 phase；Web/IM 负责 UI 与会话切换
- App 对用户可见回复会做 JSON 剥离（误把批改/出题结构当正文时只保留 `feedback`）；Web 端同样规范化历史消息
- `completed` 且用户说「下一节」时：App 创建下一节点新 session 并返回 `nextSessionId`（Web 切 session；IM 在同一通道续聊）
