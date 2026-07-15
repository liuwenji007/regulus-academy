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
  grade --[通过]--> completion_readiness（掌握度 JSON；默认开启 REGULUS_LLM_COMPLETION_CHECK）
    completion_readiness --[ready]--> completed
    completion_readiness --[not ready]--> exercise（自动连下一题；或 review 提示）
  grade --[第 1 次未通过]--> exercise（只点错因，不泄题，同一题再答）
  grade --[第 2 次未通过]--> exercise（简短讲解后自动换相似题）
  explain|exercise|review --[已经掌握，下一节]--> completion_readiness
    completion_readiness --[ready 且规则满足]--> completed
    completion_readiness --[ready 但规则建议再练]--> exercise（REGULUS_LLM_COMPLETION_CHECK=0 时硬挡）
    completion_readiness --[not ready]--> 原 phase（提示薄弱点；再次坚持则 completed 并记易错）
review --[开始练习]--> exercise
exercise --[不懂/回讲解]--> explain（针对当前题局部讲解，仍可续答）
exercise --[换一题]--> exercise（新题）
explain|review --[同一概念追问达阈值]--> 递进深讲（仍留在原 phase）
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
| [`prompts/phase_mastery.md`](./prompts/phase_mastery.md) | 掌握度 / 完成评估（申请掌握、练习答对后总评） |
| [`prompts/phase_deepen.md`](./prompts/phase_deepen.md) | 追问递进深讲 |
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

## 学生画像（记忆分层 · App 自动）

教学记忆分五层：**全局背景/目标**（`users.profile_background` / `profile_goal`）、**按课摘要**（`user_domain_profiles`）、**事实层**（`user_progress` / `mistakes`）、**工作记忆**（会话 `SessionContext`）、**派生兼容**（`profile_summary` ≤500 字，由 ProfileStore 重算）。

### 新用户引导（`profile_init`）

首次进入 Web 时，用户可回答 2～3 个引导问题（可跳过）。App 调用 `profile_init` 写入结构化 `background` / `goal`（可选 `preference`），派生 `profile_summary` 并标记 `onboarded_at`。**不写**按课进展散文。

### 设置页补充（`profile_merge`）

用户在设置页提交补充说明时，App 调用 `profile_merge`：输入当前全局画像 + 用户补充 → 更新 `background` / `goal` / `preference`，派生写回 `profile_summary`。**禁止**合并按课课堂细节。

### 节末回顾（`profile_refresh`）

节点点亮（`completed`）后，App **异步**调用 `profile_refresh`：读取本节对话摘录 + 【本课已有摘要】→ 仅 upsert `user_domain_profiles.summary`（≤200 字）。**不写**全局 `profile_summary`。失败不影响点亮。

### 注入策略

| 消费方 | 注入内容 |
|--------|----------|
| Coach 讲解/出题 | `ComposeForCoach`：全局背景/目标 + 本课摘要 + 本课 `user_progress` / `mistakes` |
| 建课/裁剪/Planner | `ComposeForBuild`：仅全局背景/目标 |

设置页「整理画像」可触发保守迁移：仅有 `user_progress` 的域才尝试从旧【进展】归因拆分。

## App 自动注入的上下文

- 当前节点 YAML：`core_concepts`、`common_mistakes`、`boundaries`、`exercise_ideas`、`grading_hints`（批改时）
- 用户已完成节点摘要、可选巩固概念、本次薄弱点、学生画像
- 本会话已考查 / 待覆盖核心概念（出题、批改、掌握度评估；用于节点内多概念覆盖）
- 对话历史（条数按任务类型调整）

## 节点完成门槛

每题 `reinforced_concepts` 记入会话「已考查」；上下文【待考查】列出尚未练到的 `core_concepts`。应用级练习（`code_fill` / `bug_find` 等）通过后记入 `ApplyExercisePassed`。

### 规则建议（`EvaluateDeferComplete`）

| 规则 | 条件 | 环境变量（默认开） |
|------|------|-------------------|
| 概念覆盖 | 核心概念 ≥3 且仍有 ≥2 个未在练习中考到 | `REGULUS_STRICT_CONCEPT_COVERAGE` |
| 应用练习 | 熟悉/精通层且尚未通过应用级练习 | `REGULUS_REQUIRE_APPLY_EXERCISE`（**入门层豁免**） |

### LLM 综合评估（`REGULUS_LLM_COMPLETION_CHECK`，默认开）

- 练习答对或用户申请掌握时，App 调用 `mastery_check` schema（见 `phase_mastery.md`）。
- 规则建议写入 prompt 为【系统规则建议】；模型可结合答疑/深讲**软豁免**后仍 `ready=true`。
- `ready=false` 时自动连下一题（优先 gap / apply）。
- 设为 `0` 时：练习答对走规则**硬挡**；申请掌握 `not ready` 时不连题，再次申请可强制完成。

### 追问深讲

`explain` / `review` 阶段，用户对同一核心概念连续追问达到阈值时，App 触发递进深讲（`phase_deepen`），每个概念每节仅一次。

## Skill 缺课自动建课

用户要学 `domains/` 中**不存在**的主题时，Agent **必须先建课再教学**：

1. 执行 `bash build-domain.sh "用户原话"`（CLI → 远程 API → Web 导出三档）
2. 成功后读取新生成的 `domains/<slug>/`
3. **不得**改用第三方技能市场或即兴生成无 YAML 边界的课程

## Skill 运行模式（Agent 教学）

lite zip 默认不含 `bin/regulus`。按优先级：

1. **Linked**：`.regulus/link.json` + `scripts/api-session.sh`（与 Web 同 API）
2. **CLI**（可选）：`./bin/regulus session`（完整状态机）
3. **Agent-lite**：`protocol-lite.md` + `agent-prompts.md` + `schemas/` + `data/progress.json`

Linked / CLI 模式下 Agent **不得**自行扮演讲解 / 出题 / 批改。详见 [SKILL.md](./SKILL.md)、[USAGE.md](./USAGE.md)。

## Skill 与 App（运维说明，不传入 LLM）

- `prompts/`、`domains/`、`schemas/`、`triggers.yaml` 为本目录运行时真相源
- App（Go 后端）从同目录加载；Linked/CLI 与 Web 共用 Coach 状态机
- App 额外持久化：SQLite 进度、错题、会话 phase；Web/IM 负责 UI 与会话切换
- App 对用户可见回复会做 JSON 剥离（误把批改/出题结构当正文时只保留 `feedback`）；Web 端同样规范化历史消息
- `completed` 且用户说「下一节」时：App 创建下一节点新 session 并返回 `nextSessionId`（Web 切 session；IM 在同一通道续聊）
