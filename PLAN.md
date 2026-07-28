# Regulus Academy — 实施计划

> 最后更新：2026-07-27

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

- [x] **课程体检与优化** — 规则 + 可选 LLM 质检报告；勾选建议后批量补全节点 YAML（不改 key、不挡开练）
- [x] **建课后自动规则体检** — LLM 生成课保存后附带 `autoAudit` 摘要；课程树页提示并引导打开完整体检
- [x] **笔记 / 错题产品内兑现** — `GET /api/domain/{id}/notes|mistakes`；课程树行内展开；教练按 `requires` 注入前置笔记（无 Embedding）
- [x] **只读 MCP Server（自托管）** — 同二进制 `server mcp` / `REGULUS_MCP=1`；工具：`get_progress` / `get_next_node` / `get_mistakes` / `get_notes` / `open_session_link`；不启动教学；Cloud 禁用
- [ ] **课程知识点考核可视化** — 掌握点进度可查看；节点维度展示考过/未考、掌握层级与薄弱点分布（笔记/错题行内展开已覆盖一部分）
- [ ] **学习进度按课程隔离** — 节点进度注入已按课隔离 ✅；画像注入（按课摘要 + 剥离旧【进展】）随记忆重构完成
- [x] **行动助手节奏恢复** — 过载分流 + 可钉北星 + 今日/清障可勾选 + 跨会话保留；四象限为展开详情（目标是恢复学习节奏，而非通用待办看板）
- [ ] **课程节点可视化管理** — 图谱 / 树形编辑器；重新生成课程时可附带用户反馈；支持手动拖拽、增删、调序（保留 `node_key` 与进度）
- [ ] 面试高频标签（节点维度：面试必考 / 生产常见 / 原理深挖）
- [ ] Agent 主动日推（Agent 根据进度主动推荐 15 分钟微任务）
- [ ] 移动端适配优化（对话页布局、图谱触屏手势）

### 待验证（有想法，没开始）

- [ ] 多模态练习（截图找 bug、流程图理解）
- [ ] **团队版拆离** — 完善团队系统（成员、权限、共享课程与进度），拆出独立团队版本 / 部署形态（见下方「团队与版本」）
- [ ] **建课资料原件持久化与复用** — 上传 PDF/URL 原件落盘并与课程关联，便于重建/优化时复用（不做「文件库索引」；建课仍走现有 Distill）
- [ ] **Domain Pack 分发** — 内置 / 自建 / 导入知识域包的浏览、版本与启用；依赖社区用户量，非当前护城河
- [ ] **可信信息源 + 联网搜索** — 白名单来源；**默认关闭**，仅建课阶段定向检索，控制脏数据与成本
- [ ] **LLM Wiki（笔记维护）** — Agent 持续更新笔记、跨 domain 建链；教学反哺已用「按 node_key / requires 关联取笔记」替代 Embedding RAG

### 明确不做

> 对抗式审查后的边界（2026-07-27）。避免反复把通用 Agent 能力塞进产品。

- **MCP Client / 接入任意第三方 MCP Server** — 违反「一个 Key 就能用」；工具增多会让教练跑题
- **通用 Skill 安装**（联网搜索、代码执行等能力插件）— 红海，无护城河；本产品的 Skill 指出口教学包，不是通用插件市场
- **Embedding + 向量 RAG** — 需第二个 Key；笔记关联是知识树图遍历问题，不是检索问题；替代方案见「笔记 / 错题产品内兑现」
- **FTS5 全文检索** — 中文分词代价高，且不必要
- **「文件库索引」** — `domain.Distill` 已 map-reduce 全文过材料，建课不需要索引层
- **在 MCP 内跑教学** — 宿主 Agent 驱动讲练批会绕开 `regulus-coach/prompts/phase_*.md` 节拍；MCP 只读 + Web 深链 handoff

---

## Phase 5 · 知识沉淀

> 闭环：讲解 → 练习 → 反馈 → 点亮 → **沉淀**。设计草案见 [`docs/knowledge-vault.md`](docs/knowledge-vault.md)。

### 5.1 MVP：Obsidian Vault 导出 ✅

- [x] `node_notes` 表（`migrations/014_node_notes.sql`）
- [x] `TaskNoteDistill`：节点点亮后异步蒸馏对话 → 写入 `node_notes`（`internal/agent/note_distill.go`）
- [x] 笔记模板：frontmatter + 摘要 + 核心概念 + 踩坑 + wikilink + MOC（`internal/domain/export_vault.go`）
- [x] 导出 API：`GET /api/domain/{id}/export/vault`
- [x] Web 入口：课程树页「导出学习笔记（Obsidian）」；主页「Skill 下载」

### 5.2 笔记反哺教学 ✅（原「LLM Wiki · RAG」的低成本替代）

- [x] 读接口：`GET /api/domain/{id}/notes`、`GET /api/domain/{id}/mistakes`
- [x] 课程树行内展开笔记 / 踩坑（不新增独立页面）
- [x] 教练 `PromptInput.RelatedNotes`：按节点 `requires` 取前置笔记注入上下文（主键查询，无 Embedding）

### 5.3 复习增强（MVP 验证后）

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
7. **MCP 只读** — 宿主可查进度与笔记，教学纪律留在 Regulus 内

---

## 后续规划（2026 Q3 及以后）

> 已验证产品方向，根据用户反馈逐步推进。2026-07-27 对抗式审查后：笔记兑现与只读 MCP 提前；Domain Pack / 原件持久化 / Embedding RAG 降级或移入不做清单。

### 学习进度与考核

| 方向 | 说明 | 状态 |
|------|------|------|
| 考核可视化 | 课程内掌握点进度、考验证记录、薄弱点分布可查看 | Phase 4 中优（笔记/错题行内已部分兑现） |
| 进度按课程隔离 | 节点进度 + 画像注入按当前课隔离；全局仅背景/目标 | Phase 4 中优（画像 ✅） |

### 内容与知识库

| 方向 | 说明 | 状态 |
|------|------|------|
| 建课资料原件持久化 | 原件落盘与复用；不做索引 | 待验证（条件触发） |
| Embedding RAG | **明确不做**；用 requires 关联笔记替代 | 不做清单 |
| 可信信息源 | 白名单联网；默认关，仅建课 | 待验证 |

### 行动助手与课程编辑

| 方向 | 说明 | 状态 |
|------|------|------|
| 节奏恢复 / 聚焦位 | 过载分流 + 可钉北星 + 勾选清障/今日 + 跨会话保留；四象限为展开详情 | Phase 4 中优 ✅ MVP |
| 节点可视化管理 | 可视化编辑节点；重建课程可带反馈；手动调整保留进度 | Phase 4 中优 |

### 团队与 Domain Pack 生态

| 方向 | 说明 | 状态 |
|------|------|------|
| 团队版拆离 | 成员 / 权限 / 共享课程；独立团队版本或部署 | 待验证 |
| Domain Pack 分发 | 内置 / 自建 / 导入知识域包；依赖用户量 | 待验证（原「Skill 库管理」） |
| Skill 市场集成 | 发布到 hermes 等；社区可复用教学包 | Phase 4 高优 |
| 只读 MCP Server | 自托管 stdio；查进度/笔记 + Web 深链 | Phase 4 中优 ✅ |

### 课程深度提升（延续）

- [ ] **建课阶段联网搜索** — 归入「可信信息源」；优先建课场景，不全程开放以控制脏数据
- [x] **笔记按知识树关联注入教练** — 替代「可选 RAG 补充」
- [ ] **Domain Pack 系统** — 与分发、市场发布一并推进（等社区贡献）

### 评测与自动优化

- [ ] **自动 prompt 优化闭环** — 变体评测 + 评分 → LLM 自动改写 prompt → 循环 N 轮 → 选出最佳变体（参考 darwin-skill 的优化策略）
- [ ] **评测结果 → 课程改进** — 低分流程直接转化为提示词改进项
- [ ] **社区 Benchmark** — 公开的 Agent 评测排行榜，吸引社区贡献 skill