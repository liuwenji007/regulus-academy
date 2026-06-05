---
name: regulus-coach
description: Regulus Academy 碎片化学习 AI 教练。用于学习 Go 并发等知识域：建知识树、节点讲解、微练习出题与批改。用户提到 Regulus Academy、regulus-coach、Go 并发微训练时使用。
---

# Regulus Academy Coach

## 何时使用

- 用户要学习 **Go 并发** 或已安装的本仓库知识域
- 需要 **知识树导航**、**单节点讲解**、**微练习**、**作答批改**
- 在 IDE 里边看代码边学，或终端里碎片化练习

## 怎么做

1. 阅读 [protocol.md](./protocol.md) — 我们的学习方式（只读这一份学法说明）
2. 若学 Go 并发：读 `domains/go-concurrency/tree.yaml` 了解路径，再读 `domains/go-concurrency/nodes/<节点key>.yaml` 获取当前节点边界
3. 按节点推进：**讲解** → 用户回复「开始练习」→ **出一道题**（见 `schemas/exercise.json`）→ **批改**（见 `schemas/grade.json`）

## 与 Regulus Academy App 的关系

- 本目录是 **Skill 与 App 的唯一真相源**；仓库内 Go 后端从同目录加载
- App 负责进度 SQLite、知识树可视化、会话 phase 与切节；Skill 可在任意 Agent 入口使用，进度可由用户口述或自行记录
- 运维与编排细节（JSON 剥离、`nextSessionId` 等）见 [protocol.md](./protocol.md) 末尾「Skill 与 App」；**不**写入 `prompts/core.md`（该文件会传入 LLM）

## 贡献知识域

见仓库 [CONTRIBUTING.md](../CONTRIBUTING.md) — 在 `domains/<your-domain>/` 下添加 `tree.yaml` 与 `nodes/*.yaml`。
