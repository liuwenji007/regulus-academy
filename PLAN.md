# Regulus Academy — 实施计划

> 最后更新：2026-06-07

## 里程碑总览

```
Phase 0 ✅      Phase 1 ✅      Phase 1.5 ✅    Phase 2 ✅      Phase 3 ✅      Phase 4 🔄
项目立项        后端+Skill+Web  Channel 接入    首个闭环        开源就绪        持续迭代
```

---

## Phase 0 · 项目立项 ✅

- [x] 竞品分析（DeepTutor 实测、OpenMAIC 实测）
- [x] 设计理念文档（DESIGN.md）
- [x] 贡献手册（CONTRIBUTING.md）
- [x] 项目规划（PLAN.md，本文件）
- [x] 开源许可证（LICENSE · Apache 2.0）
- [x] .gitignore / CODE_OF_CONDUCT.md
- [x] README.md 完善（项目介绍 + 快速开始）

---

## Phase 1 · 后端 + Skill + Web ✅

### 1.1 后端

- [x] Go 项目初始化（`go.mod`、`cmd/server/main.go`）
- [x] SQLite 数据库初始化（`migrations/`）
- [x] HTTP 路由框架（`internal/api/handler.go`）
- [x] OpenAI 兼容 API 调用封装（`internal/llm/`）
- [x] 多 Provider 配置（deepseek / openai / openrouter / ollama / custom）
- [x] LLM Profile 热切换

### 1.2 Web 前端

- [x] Vite + TypeScript PWA（`web/src/`）
- [x] 知识树可视化（vis-network，`#/tree/:id`）
- [x] 知识银河（多领域全景图，`#/graph`）
- [x] 教学对话页（`#/coach/:sessionId`）
- [x] 课程列表、进度可视化
- [x] PDF/URL 导入建课（`#/import`）
- [x] Docker 一键启动 / 安装脚本

### 1.3 Skill 骨架

- [x] `regulus-coach/SKILL.md`
- [x] `regulus-coach/domains/go-concurrency/`（tree.yaml + 节点）
- [x] `protocol.md`、`schemas/`

### 1.4 基础设施

- [x] Docker Compose（本地 build + 预构建镜像两套）
- [x] `.env.example`
- [x] GitHub Actions CI（`go test` + 前端构建）
- [x] 单元测试（`internal/agent/`、`internal/domain/`）

---

## Phase 1.5 · Channel 接入 ✅

- [x] Telegram 机器人（Long Polling）
- [x] 钉钉机器人（Stream 模式）
- [x] 飞书机器人（WebSocket 长连接）
- [x] 企业微信回调（`POST /webhook/wecom`）
- [x] 角色绑定：IM → Web user_id
- [x] 进度 / 会话跨端共用（`channel_bindings` + `sessions`）
- [x] IM 自然语言导航（规则优先 + LLM 兜底）

---

## Phase 2 · 首个闭环 ✅

- [x] `regulus-coach/` 骨架（protocol / SKILL / schemas / go-concurrency 域）
- [x] **教学 Agent**（讲解 / 出题 / 批改 / 状态机）
- [x] **建树 Agent**（任意领域 LLM 生成知识树，带异步 Job）
- [x] PDF/URL 导入 → LLM 蒸馏 → 知识树（`/api/domain/build/from-source`）
- [x] 纵深扩展（`/api/domain/{id}/extend`，完成度 ≥80% 解锁）
- [x] SQLite 进度 / 错题 / 会话 / 用户画像
- [x] 无感错题强化
- [x] 用户画像裁剪（背景 × 学习目标，`profile_summary` ≤500 字）
- [x] 多学习角色（进度与课程列表按角色隔离）
- [x] 重建保留进度（按 `node_key` 迁移）
- [x] 导出 Skill 包（`/api/domain/{id}/export`）

---

## Phase 3 · 开源就绪 ✅

- [x] README 完善（Logo + 截图 + 快速开始 + 设计理念链接）
- [x] CONTRIBUTING / SECURITY / CODE_OF_CONDUCT 文档
- [x] GitHub Actions CI
- [x] 一键安装脚本（`scripts/install.sh`，自动重试、端口冲突处理）
- [x] GHCR 预构建镜像（`docker-compose.image.yml`）
- [x] Langfuse OTLP 可观测性（可选，默认关闭）
- [ ] 提交到 awesome-deepseek-integration
- [ ] 在 V2EX / Twitter / 少数派 分享项目故事
- [ ] 收集第一批社区反馈

---

## Phase 4 · 持续迭代 🔄

> 基于社区反馈动态排序，当前优先级：

### 高优（已在做）

- [x] 知识银河体验打磨（LOD 分级、星座光晕、物理引擎调优）
- [ ] Skill 发布到市场（hermes / 其他 Agent 框架）
- [ ] 更多内置知识域（Agent 原理 / Python / RAG / Nginx / Docker 进阶）

### 中优（计划中）

- [ ] 面试高频标签（节点维度：面试必考 / 生产常见 / 原理深挖）
- [ ] 每日推荐（Agent 根据进度主动推荐 15 分钟微任务）
- [ ] 移动端适配优化（对话页布局、银河触屏手势）

### 待验证（有想法，没开始）

- [ ] 多模态练习（截图找 bug、流程图理解）
- [ ] 团队共享知识树（只读分享链接）
- [ ] Embedding + RAG（用户上传自己的代码 / 笔记 / 文档作为补充材料）
- [ ] 搜索服务（学习内容时效性强时补充最新资料）
- [ ] **LLM Wiki**（远期 · 待 MVP 知识沉淀验证后再投入）：Agent 持续维护 / 重构笔记、跨 domain 自动建链；对 vault 做 RAG 反哺教学上下文（教练引用用户自己的笔记讲解）。注意与「一个 Key 就能用」原则的张力，RAG 需要 Embedding，应做成可选项。

---

## Phase 5 · 知识沉淀（规划中）

> 闭环补上最后一环：讲解 → 练习 → 反馈 → 点亮 → **沉淀**。把锁在 SQLite 里的学习成果变成用户可带走的本地 Markdown 知识库，兼容 Obsidian。
>
> 设计草案见 [`docs/knowledge-vault.md`](docs/knowledge-vault.md)。

### 5.1 MVP：Obsidian Vault 导出

- [ ] 新增 `node_notes(user_id, domain_id, node_key, content_md, updated_at)` 表（新 migration）
- [ ] 新增 `TaskNoteDistill`：节点点亮后异步蒸馏对话 → 300~500 字摘要，写入 `node_notes`（复用 `scheduleProfileRefresh` 的 goroutine + 超时 + Trace 管线，`internal/agent/profile_refresh.go`）
- [ ] 笔记模板：frontmatter（domain / module / layer / mastery / status / tags）+ 摘要 + 核心概念（节点 YAML `core_concepts`）+ 「踩过的坑」（`mistakes` 表）+ 关键问答摘录
- [ ] 链接生成：依据 `tree_json` 前置 / 同模块关系生成 `[[wikilink]]`；每个 domain 一篇 MOC 索引笔记，Obsidian Graph View 即「知识银河」本地镜像
- [ ] 导出 API：`GET /api/domain/{id}/export/vault` 打 zip 下载（对齐现有 `exportDomain`，`internal/api/handler.go`）
- [ ] Web 入口：课程树页与「导出 Skill 包」并列「导出学习笔记（Obsidian）」按钮（`web/src/pages/tree.ts`）

### 5.2 复习增强（MVP 验证后）

- [ ] `mistakes` 蒸馏为 flashcard（兼容 Obsidian Spaced Repetition 插件 `#flashcards` 语法）
- [ ] frontmatter 做 Dataview 友好，支持「mastery < 0.6」复习视图

---

## 关键原则（贯穿所有 Phase）

1. **一个 Key 就能用** — 不要求配置 Embedding 或搜索服务
2. **微闭环** — 一个节点 = 讲解 + 一题 + 反馈（15 分钟是场景叙事，不是 Prompt 计时）
3. **入口极简** — 主路径三步：输入领域 → 选节点 → 对话学习
4. **先跑通，再开源** — 私有仓库开发，公开仓库发布
5. **中文优先** — 所有界面、提示、文档默认中文
6. **知识边界 > 知识库** — 节点定义边界，LLM 在边界内自由生成


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
- [x] README 完善（Logo + 截图 + 快速开始 + 设计理念链接）
- [x] GitHub Actions CI（`go test` + 前端构建）
- [x] CONTRIBUTING / SECURITY 文档对齐
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
