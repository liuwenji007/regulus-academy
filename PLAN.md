# Regulus Academy — 实施计划

## 里程碑总览

```
Phase 0 · 项目立项        Phase 1 · 后端+Skill+Web    Phase 1.5 · Channel       Phase 2 · 首个闭环       Phase 3 · 开源
  今天                      本周                         第 2 周                   下周末                     第 3 周
  ┌──────────────┐        ┌──────────────┐           ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
  │ ✓ 竞品分析    │        │ □ Go 后端     │           │ □ 企微机器人   │        │ □ Go 并发知识树│        │ □ GitHub 公开 │
  │ ✓ 设计理念    │        │ □ Skill 骨架  │           │ □ 飞书/钉钉    │        │ □ 教学 Agent  │        │ □ 提交到精选集│
  │ ✓ 记忆管理    │        │ □ Web 页面    │           │ □ 跨端共享记忆│        │ □ 记忆持久化  │        │ □ 社交媒体分享│
  │ ✓ 贡献手册    │        │ □ 知识域打包  │           │              │        │ □ 错题强化    │        │ □ 收集早期反馈│
  │ ✓ 许可证      │        │ □ 单元测试    │           │              │        │ □ 三层闭环    │        │              │
  └──────────────┘        └──────────────┘           └──────────────┘        └──────────────┘        └──────────────┘
```

---

## Phase 0 · 项目立项（今天）

- [x] 竞品分析（DeepTutor 实测、OpenMAIC 实测）
- [x] 设计理念文档（DESIGN.md）
- [x] 贡献手册（CONTRIBUTING.md）
- [x] 项目规划（PLAN.md，本文件）
- [x] 开源许可证（LICENSE · Apache 2.0）
- [x] .gitignore
- [x] CODE_OF_CONDUCT.md
- [x] README.md 完善（项目介绍 + 快速开始 + Logo）

---

## Phase 1 · 后端 + Skill + Web（本周）

> 目标：Go 后端启动 + regulus-coach Skill 骨架 + 本地 Web 页面。三层分发中最核心的 Local 层。

### 1.1 后端

- [ ] Go 项目初始化
- [ ] SQLite 数据库初始化
- [ ] HTTP 路由框架
- [ ] DeepSeek API 调用封装

### 1.2 Web 前端

- [ ] 纯 HTML + CSS + 原生 JS
- [ ] 知识树可视化页面
- [ ] 教学对话页面
- [ ] 本地 Docker 一键启动

### 1.3 Skill 骨架

- [ ] `regulus-coach/SKILL.md`
- [ ] `regulus-coach/domains/go-concurrency/`（tree.yaml + 节点）
- [ ] Skill 可独立安装使用

### 1.4 基础设施

- [ ] Docker Compose
- [ ] `.env.example`
- [ ] 单元测试框架

---

## Phase 1.5 · Channel 接入（第 2 周）

> 接入 Telegram / 钉钉 / 飞书 / 企业微信机器人，用户直接在 IM 里跟教练对话。

- [x] Telegram 机器人（Long Polling）
- [x] 钉钉机器人（Stream 模式）
- [x] 飞书机器人（WebSocket 长连接）
- [x] 企业微信回调（`POST /webhook/wecom`，需公网 HTTPS）
- [x] 角色绑定：`绑定 角色名` 映射到 Web 端 user_id
- [x] 进度/会话与 Web 共用（`channel_bindings` + `sessions`）
- [x] Gateway 与 Coach 直连（`internal/channel`）

---

## Phase 2 · 首个闭环（下周末前）

> 目标：完成「Go 并发」完整教学闭环，Skill + Local Web + Channel 三种入口都可用。

### 2.0 regulus-coach 骨架（最先做）

- [x] 创建仓库根目录 `regulus-coach/`
- [x] `protocol.md`
- [x] `SKILL.md`
- [x] `schemas/exercise.json`、`schemas/grade.json`
- [x] `domains/go-concurrency/`（`tree.yaml` + 10 个节点）

### 2.1 知识领域

- [x] 完善 `tree.yaml`（三层 10 节点）
- [x] 各节点 `nodes/*.yaml`
- [x] `internal/domain/registry.go` 从 `regulus-coach/` 加载

### 2.2 Agent 核心逻辑

- [x] **教学 Agent**（讲解 / 出题 / 批改 / 状态机）
- [ ] **建树 Agent**（P1，非 Go 并发领域）
- [x] P0 非「Go 并发」输入 → 提示 MVP 范围

### 2.3 记忆管理

- [x] SQLite 进度、错题、会话 phase/context
- [x] `BuildContext` 注入进度与可选巩固
- [x] 错题记录 + 强化计数
- [x] 无感错题强化

### 2.4 教学闭环串联

- [x] 「Go 并发」→ registry 加载树
- [x] 点节点 → LLM 讲解 → 练习 → 批改 → 点亮
- [x] `GET /api/session/{id}` 恢复对话
- [x] Channel 消息路由接入教学流程（`internal/channel`）

### 2.5 品牌与文档对齐

- [x] 文案统一 **Regulus Academy**
- [x] README 状态更新
- [x] Skill 说明与 CONTRIBUTING 一致
- [ ] （Phase 3）Skill / Channel 发布

---

## Phase 3 · 开源（第 3 周）

- [ ] 代码从私有仓库推送到公开仓库
- [ ] README 完善（Logo + 截图 + 快速开始 + 设计理念链接）
- [ ] 提交到 awesome-deepseek-integration
- [ ] 在 DeepSeek 社群分享
- [ ] 在 V2EX / Twitter 分享项目故事
- [ ] 收集第一批用户反馈

---

## Phase 4 · 迭代（第 4 周起）

基于早期用户反馈决定优先级：

| 可能的迭代方向 | 触发条件 |
|---------------|---------|
| 加更多知识领域（Agent 原理、RAG 架构...） | 用户说"Go 学完了，然后呢？" |
| 自定义 API Key 页面 | 用户担心隐私 |
| 新办公 IM 接入（企业微信/飞书/钉钉/Teams） | 用户问"XXX 上能用吗" |
| 面试痛点覆盖（节点标签："面试高频"） | 用户问"这个面试会考吗" |
| Langfuse 监控接入 | 教学效果不好，需要 debug |
| Embedding + RAG | 用户上传自己的笔记/代码 |
| 搜索服务 | 学习内容更新很快（框架新版） |

---

## 关键原则（贯穿所有 Phase）

1. **一个 Key 就能用** — 不要求用户配置 Embedding 或搜索
2. **微闭环** — 一个节点 = 讲解 + 一题 + 反馈（15 分钟是用户场景叙事，不写入 Prompt 计时）
3. **入口极简** — 没有功能介绍页、新手引导、多级菜单
4. **先跑通，再开源** — 私有仓库开发，公开仓库发布
5. **中文优先** — 所有界面、提示、文档用中文
6. **知识边界 > 知识库** — 不预置大量内容，依赖 LLM 在边界内生成
