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
| [`prompts/core.md`](./prompts/core.md) | 角色、学习方式、禁止自行点亮 |
| [`prompts/phase_explain.md`](./prompts/phase_explain.md) | 讲解/答疑 |
| [`prompts/phase_exercise.md`](./prompts/phase_exercise.md) | 出题 |
| [`prompts/phase_grade.md`](./prompts/phase_grade.md) | 批改 |
| [`prompts/phase_mastery.md`](./prompts/phase_mastery.md) | 申请完成评估 |

## JSON Schema

- 出题：[`schemas/exercise.json`](./schemas/exercise.json)
- 批改：[`schemas/grade.json`](./schemas/grade.json)
- 申请完成评估：[`schemas/mastery_check.json`](./schemas/mastery_check.json)

## App 自动注入的上下文

- 当前节点 YAML：`core_concepts`、`common_mistakes`、`boundaries`、`exercise_ideas`、`grading_hints`（批改时）
- 用户已完成节点摘要、可选巩固概念、本次薄弱点、学生画像
- 对话历史（条数按任务类型调整）
