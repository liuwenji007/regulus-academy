---
name: regulus-coach
description: Regulus Academy 碎片化学习 AI 教练。知识树导航、节点讲解、微练习出题与批改；配合 regulus build 或 Domain 包建课。用户提到 Regulus Academy、regulus-coach 时使用。
---

# Regulus Academy Coach

## 何时使用

- 用户要学习某知识域（内置或已安装的 `domains/`）
- 需要 **知识树导航**、**单节点讲解**、**微练习**、**作答批改**
- 在 IDE 里边看代码边学，或终端里碎片化练习

## 建课

任选其一，让 `domains/<slug>/` 存在后再教学：

1. **CLI（推荐离线）**：在仓库根目录配置 `.env` 中的 LLM Key 后执行  
   `regulus build "想学 Rust"`  
   或 `go run ./cmd/regulus build "想学 Rust"`  
   输出写入 `regulus-coach/domains/<slug>/`
2. **Web**：在 Regulus 主页输入主题建课，或在课程详情页 **导出 Domain 包**，解压到本目录的 `domains/`
3. **内置**：`domains/go-concurrency/` 等社区路径可直接开练

## 教学流程

1. 阅读 [protocol.md](./protocol.md) — 学习方式（只读这一份）
2. 读 `domains/<slug>/tree.yaml` 了解路径，再读 `domains/<slug>/nodes/<节点key>.yaml` 获取当前节点边界
3. 按节点推进：**讲解** → 用户回复「开始练习」→ **出一道题**（见 `schemas/exercise.json`）→ **批改**（见 `schemas/grade.json`）

## 与 Regulus Academy App 的关系

- 本目录是 **Skill 与 App 的唯一真相源**；Go 后端从同目录加载
- App 负责进度 SQLite、知识树可视化、会话 phase 与切节；Skill 可在任意 Agent 入口使用，进度可由用户口述或自行记录
- 运维与编排细节（JSON 剥离、`nextSessionId` 等）见 [protocol.md](./protocol.md) 末尾「Skill 与 App」

## 贡献知识域

见仓库 [CONTRIBUTING.md](../CONTRIBUTING.md) — 在 `domains/<your-domain>/` 下添加 `tree.yaml` 与 `nodes/*.yaml`。
