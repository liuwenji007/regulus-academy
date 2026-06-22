# Regulus Academy — 实施计划

> 最后更新：2026-06-18

## 里程碑总览

```
Phase 0 ✅   Phase 1 ✅   Phase 1.5 ✅   Phase 2 ✅   Phase 3 ✅   Phase 4 🔄   Phase 5 MVP ✅
项目立项     后端+Web     Channel       首个闭环     开源就绪     持续迭代     知识沉淀 MVP
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
- [x] 知识图谱（多领域全景图，`#/graph`）
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
- [x] 导出 Domain 包（`/api/domain/{id}/export`）与 Coach Skill（`/api/coach/export`）
- [x] CLI 建课（`regulus build`，`cmd/regulus`）

---

## Phase 3 · 开源就绪 ✅

- [x] README 完善（Logo + 截图 + 快速开始 + 设计理念链接）
- [x] CONTRIBUTING / SECURITY / CODE_OF_CONDUCT 文档
- [x] GitHub Actions CI + GHCR 镜像 + Release workflow
- [x] 一键安装脚本（`scripts/install.sh`，自动重试、端口冲突处理）
- [x] Cloud Demo（Railway）+ 文档站（Vercel）
- [x] Langfuse OTLP 可观测性（可选，默认关闭）
- [x] 体验反馈 Issue 模板（`.github/ISSUE_TEMPLATE/experience_feedback.yml`）
- [x] 首个版本标签 `v0.1.0`（见 [CHANGELOG.md](./CHANGELOG.md)）

### 推广待办（运营，非代码）

- [ ] 提交到 awesome-deepseek-integration
- [ ] 在 V2EX / Twitter / 少数派 分享项目故事
- [ ] 收集第一批社区反馈（引导用户开 `[体验]` Issue）

---

## Phase 4 · 持续迭代 🔄

> 基于社区反馈动态排序，当前优先级：

### 高优（已在做）

- [x] 知识图谱体验打磨（LOD 分级、宣纸/星空双主题、星座聚类、物理引擎调优）
- [x] 首批扩展知识域：`python-quickstart`、`rust-quickstart`、`prompt-design`（各 11～13 节点）
- [ ] Skill 发布到市场（hermes / 其他 Agent 框架）；CLI `regulus build` 已支持离线建课
- [ ] 更多内置知识域（Agent 原理 / RAG / Nginx / Docker 进阶）
- [x] `prompt-design` 域节点 `teaching_beats` 教考对齐（Go 并发域为标杆）

### 中优（计划中）

- [ ] 面试高频标签（节点维度：面试必考 / 生产常见 / 原理深挖）
- [ ] 每日推荐（Agent 根据进度主动推荐 15 分钟微任务）
- [ ] 移动端适配优化（对话页布局、图谱触屏手势）

### 待验证（有想法，没开始）

- [ ] 多模态练习（截图找 bug、流程图理解）
- [ ] 团队共享知识树（只读分享链接）
- [ ] Embedding + RAG（用户上传自己的代码 / 笔记 / 文档作为补充材料）
- [ ] 搜索服务（学习内容时效性强时补充最新资料）
- [ ] **LLM Wiki**（远期 · 待 MVP 知识沉淀验证后再投入）：Agent 持续维护 / 重构笔记、跨 domain 自动建链；对 vault 做 RAG 反哺教学上下文（教练引用用户自己的笔记讲解）。注意与「一个 Key 就能用」原则的张力，RAG 需要 Embedding，应做成可选项。

---

## Phase 5 · 知识沉淀

> 闭环：讲解 → 练习 → 反馈 → 点亮 → **沉淀**。设计草案见 [`docs/knowledge-vault.md`](docs/knowledge-vault.md)。

### 5.1 MVP：Obsidian Vault 导出 ✅

- [x] `node_notes` 表（`migrations/014_node_notes.sql`）
- [x] `TaskNoteDistill`：节点点亮后异步蒸馏对话 → 写入 `node_notes`（`internal/agent/note_distill.go`）
- [x] 笔记模板：frontmatter + 摘要 + 核心概念 + 踩坑 + wikilink + MOC（`internal/domain/export_vault.go`）
- [x] 导出 API：`GET /api/domain/{id}/export/vault`
- [x] Web 入口：课程树页「导出学习笔记（Obsidian）」；主页「Skill 下载」

### 5.2 复习增强（MVP 验证后）

- [ ] `mistakes` 蒸馏为 flashcard（兼容 Obsidian Spaced Repetition 插件 `#flashcards` 语法）
- [ ] frontmatter 做 Dataview 友好，支持「mastery < 0.6」复习视图
- [ ] 笔记模板与蒸馏 Prompt 社区共建（见 CONTRIBUTING · 学习笔记导出）

---

## 关键原则（贯穿所有 Phase）

1. **一个 Key 就能用** — 不要求配置 Embedding 或搜索服务
2. **微闭环** — 一个节点 = 讲解 + 一题 + 反馈（15 分钟是场景叙事，不是 Prompt 计时）
3. **入口极简** — 主路径三步：输入领域 → 选节点 → 对话学习
4. **先跑通，再开源** — 私有仓库开发，公开仓库发布
5. **中文优先** — 所有界面、提示、文档默认中文
6. **知识边界 > 知识库** — 节点定义边界，LLM 在边界内自由生成

---

## 后续规划（2026 Q3 及以后）

> 已验证产品方向，根据用户反馈逐步推进。

### 课程深度提升

- [ ] **建课阶段联网搜索** — 仅在课程生成时搜索可信来源（技术博客白名单 / arXiv），提升知识树深度，不全程开放搜索以控制脏数据风险
- [ ] **可选的 RAG 补充** — 用户上传代码 / 笔记 / 文档作为补充学习材料，Embedding 做成可选项，不违反「一个 Key 就能用」原则
- [ ] **Skill 系统集成** — 将评测出的优质 prompt 沉淀为社区可复用的 skill 包

### 评测与自动优化

- [ ] **自动 prompt 优化闭环** — 变体评测 + 评分 → LLM 自动改写 prompt → 循环 N 轮 → 选出最佳变体（参考 darwin-skill 的优化策略）
- [ ] **评测结果 → 课程改进** — 低分流程直接转化为提示词改进项
- [ ] **社区 Benchmark** — 公开的 Agent 评测排行榜，吸引社区贡献 skill
