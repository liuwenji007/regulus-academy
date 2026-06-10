# Regulus Academy — 开源贡献手册

感谢你有兴趣参与！这份手册会告诉你从哪里开始、怎么协作、以及我们的工作方式。

---

## 行为准则

这是社区的最低共识，不是公司的员工手册。

- **友善** — 每个人都是从某个节点开始学习的，包括你和我
- **务实** — 先跑通再去讨论"完美的方案"
- **尊重用户** — 任何设计决策，先想"一个通勤 15 分钟的在职开发者会不会想用"
- **中文优先** — 默认用中文协作；见下文「语言与协作」，用户可见内容须为中文

---

## 项目概况

| | |
|---|---|
| 定位 | 面向在职开发者的碎片化学习 AI 私教 |
| 技术栈 | Go (后端) + PWA (前端) + SQLite + DeepSeek API |
| 语言 | 中文优先（产品面向中文用户；协作细则见下） |
| 许可证 | Apache 2.0 |
| 沟通 | GitHub Issues |

---

## 我能贡献什么？

### 你是开发者

| 技能 | 你能做 | 对应模块 |
|------|--------|---------|
| Go 后端 | Agent 逻辑优化、记忆管理、API 路由 | `internal/agent/`、`internal/service/`、`internal/api/` |
| 前端 TypeScript | 知识银河体验、对话 UI、课程详情页 | `web/src/pages/`、`web/src/lib/knowledge-graph.ts` |
| AI/LLM Prompt | 优化教学 Prompt、设计练习生成策略、改进批改准确率 | `internal/agent/prompt.go`、`internal/domain/builder.go` |
| 测试 | Agent 状态机边界、Domain 建树校验、组件测试 | `internal/agent/*_test.go`、`internal/domain/*_test.go` |
| IM Channel | 新接入平台、导航规则优化 | `internal/channel/` |
| DevOps | CI 优化、Docker 镜像、安装脚本 | `.github/workflows/`、`scripts/` |

**当前最欢迎的 PR：**

- 知识银河：更好的节点布局算法、LOD 切换体验、拖动后节点重新收敛
- 教练质量：更准确的掌握度判断（`internal/agent/mastery_skip.go`）、练习题多样性
- 新知识域：Agent 原理 / Python 基础 / Docker 实战 / Nginx / RAG 等（见下方「加一个新的知识领域」）

### 你不是开发者

| 你能做 | 说明 |
|--------|------|
| **定义知识节点** | 最高价值的贡献：写一个知识点的核心概念、常见误区、边界（见下方格式） |
| 体验反馈 | 用一用，告诉我们哪里卡住、哪里不顺手，开 `[体验]` Issue |
| 文档 | 写教程、改 README、补充示例、翻译设计文档 |
| 布道 | 在社区分享、写文章、推荐给可能有需求的朋友 |

---

## 快速开始

```bash
# 1. 克隆仓库
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy

# 2. 配置并启动后端
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY

go run ./cmd/server

# 3. 启动前端（新终端，开发模式）
cd web
pnpm install
pnpm dev
```

浏览器打开 http://localhost:5173 。主路径：**输入领域 → 选节点 → 对话学习**。

| 路由 | 用途 |
|------|------|
| `#/` | 开始学习（建课） |
| `#/graph` | 知识图谱 |
| `#/courses` | 我的课程 |
| `#/tree/:id` | 课程详情 |
| `#/coach/:sessionId` | AI 教练对话 |
| `#/settings` | 设置 |
| `#/settings/channels` | IM 频道（Telegram / 钉钉 / 飞书 / 企微） |

---

## 项目结构

```
regulus-academy/
├── cmd/
│   └── server/              # 后端入口（main.go）
├── internal/
│   ├── agent/               # Coach 教学状态机
│   │   ├── coach.go         # 讲解 / 出题 / 批改 FSM
│   │   ├── coach_next.go    # 完成态「下一节」、下一节点提示
│   │   ├── mastery_skip.go  # 「已经掌握」评估与强制完成
│   │   ├── exercise_format.go / exercise_adopt.go  # 练习作答方式与误输出 JSON 采纳
│   │   ├── assistant_content.go  # 批改/掌握度 JSON 剥离为纯文本
│   │   ├── intent.go        # 开始练习、实际案例、申请完成等触发词
│   │   ├── prompt.go        # System prompt 构建（注入节点边界、进度、用户画像）
│   │   └── memory.go        # 错题强化概念选取
│   ├── channel/             # IM Gateway（Telegram / 钉钉 / 飞书 / 企微）
│   │   ├── gateway.go       # 适配器注册与启动
│   │   ├── router.go        # IM 路由（命令 + 规则/LLM 导航 + Coach 转发）
│   │   ├── nav_rules.go     # 自然语言导航规则（零 token）
│   │   ├── nav_intent.go    # LLM 导航意图兜底
│   │   └── delivery.go      # 统一出站（分片、重试）
│   ├── domain/              # 知识领域（加载 / 建树 / 个性化 / modules）
│   │   ├── registry.go      # 从 regulus-coach/ 加载 YAML
│   │   ├── builder.go       # LLM 动态建树
│   │   ├── modules.go       # 主题模块校验
│   │   └── personalizer.go  # 用户画像裁剪
│   ├── storage/             # SQLite 持久化
│   │   └── sqlite.go
│   ├── service/             # 会话服务（Web 与 IM 共用）
│   │   └── session.go       # StartOrResume、发消息、下一节 session 切换
│   ├── config/              # 配置读取与 Gateway 设置
│   └── api/                 # HTTP 路由
│       └── handler.go
├── web/                     # Vite + TypeScript PWA 前端
│   ├── src/
│   │   ├── pages/           # home / graph / courses / tree / coach / settings / channels
│   │   ├── components/      # layout / sidebar（课程快捷、切换角色）
│   │   └── lib/             # API、coach-exercise、start-node-session、profile
│   └── dist/                # 构建产物（go embed 打包进二进制）
├── .github/workflows/       # CI（go test + 前端构建）
├── regulus-coach/           # Skill 定义（知识边界 + 教练协议）
│   ├── SKILL.md
│   ├── protocol.md
│   └── domains/go-concurrency/
├── docs/                    # Banner / 界面截图等静态资源
├── DESIGN.md                # 设计理念
├── SECURITY.md              # 安全报告方式
└── CONTRIBUTING.md          # 本文件
```

---

## 加一个新的知识领域

这是最高价值也最低门槛的贡献方式。你不需要会写代码。

每个知识领域需要两样东西：

### 1. 知识树（`tree.yaml`）

知识树有两个**正交**维度：

- **`modules`** — 主题分簇（如「Goroutine 基础」「Channel 与通信」），供图谱展示
- **`layers`** — 掌握深度（入门 / 熟悉 / 精通），供课程列表与学习路径

```yaml
domain: Go 并发
slug: go-concurrency
version: 1
description: 用 goroutine 和 channel 写出并发安全的 Go 程序

modules:
  - key: goroutine_foundation
    label: Goroutine 基础
    goal: 理解轻量级线程与等待模型
    nodes:
      - goroutine_basics
      - first_goroutine
      - waitgroup

layers:
  entry:
    label: 入门
    time: "约 2.5～3 小时"
    goal: 能看懂并发代码，能创建简单的 goroutine
    nodes:
      - key: goroutine_basics
        title: goroutine 是什么
      - key: first_goroutine
        title: 启动第一个 goroutine
      - key: waitgroup
        title: sync.WaitGroup 等待完成

  intermediate:
    label: 熟悉
    time: "约 10～14 小时"
    goal: 能独立写生产级并发代码
    nodes:
      - key: channel
        title: channel 通信
      # …

  advanced:
    label: 精通
    time: "约 20～30 小时"
    goal: 理解调度模型，能排查并发 bug
    nodes:
      - key: gmp
        title: GMP 调度模型
      # …
```

完整示例见 [`regulus-coach/domains/go-concurrency/tree.yaml`](./regulus-coach/domains/go-concurrency/tree.yaml)。

### 2. 节点边界定义（`nodes/<节点名>.yaml`）

```yaml
node: channel 通信
key: channel
layer: 熟悉
requires:          # 可选：建议先完成的节点 key（不阻止学习，用于图谱与课程页提示）
  - waitgroup

core_concepts:
  - 无缓冲 channel 的同步特性
  - 带缓冲 channel 的容量与阻塞条件
  - 方向 channel（只读/只写）
  - for range 遍历 channel

common_mistakes:
  - 往已关闭的 channel 发送数据导致 panic
  - 忘记 close channel 导致 goroutine 泄漏
  - 无缓冲 channel 双向阻塞导致死锁

boundaries:
  - 不讲 select 语句（那是下一个节点）
  - 不讲 channel 底层实现（那是精通层）
  - 不讲 context 取消（那是另一个节点）

exercise_ideas:
  - "用 channel 实现两个 goroutine 轮流打印数字"
  - "以下代码有什么问题？为什么 deadlock？"

grading_hints:
  - "可选：与 core_concepts 对齐的评分要点，供 Coach 批改时对照"
  - "例如：无缓冲 channel 发送会阻塞直到有人接收"

# 可选：递进式教学节拍（教考对齐）
first_exercise_level: recognition   # recognition | recall | apply
domain_kind: applied                # applied | academic | mixed

teaching_beats:
  - concept: 无缓冲 channel 的同步特性   # 与 core_concepts 条目一一对应
    must_teach:
      - 发送与接收同步握手
      - 常用于 goroutine 间同步
    context_type: workplace           # workplace | intuition | exam_pattern | prerequisite_link
```

`context_type` 说明第二拍「锚点」类型：工程类用 `workplace`；学术类（如高数）用 `intuition` 或 `exam_pattern`，不必硬套工作场景。
```

---

### 从 App 导出并提交 PR

如果你在 App 里用 LLM 生成了知识树，觉得质量不错，可以贡献回社区：

1. 打开该课程的 **课程详情**（`#/tree/:id`），点击 **「导出 Skill 包」**
2. 会下载 `{slug}-skill.zip`，解压后得到 `regulus-coach-{slug}/` 目录
3. 把其中的 `domains/{slug}/` 复制到仓库的 `regulus-coach/domains/{slug}/`
4. 检查 `tree.yaml` 顶部的 `version: 1`，补充 `description`（LLM 已尝试自动填充，请人工核对）
5. 提 PR，说明覆盖范围、目标用户、与现有公共库的差异

> **直接安装到 Agent 练习**：解压后将整个 `regulus-coach-{slug}/` 目录放入你的 Agent skills 目录（如 Cursor 的 `.cursor/skills/`），无需提 PR 即可立刻练习。

导出 API：`GET /api/domain/{id}/export`（响应 `application/zip`，需属于当前用户）。

公共知识库目录 API：`GET /api/domains/public`（无需 LLM，浏览 `regulus-coach/domains/` 下已有 Skill 包）。

---

## 以 Skill 文件贡献知识领域

除了在仓库代码中定义知识节点，你也可以将完整的知识域打包为 `regulus-coach` Skill 文件。

### Skill 文件结构

```
regulus-coach/
├── SKILL.md              # 教练行为定义
├── domains/              # 知识领域
│   └── your-domain/
│       ├── tree.yaml     # modules + layers
│       └── nodes/        # 节点边界定义
```

### 贡献步骤

1. 创建知识域目录：`domains/<your-domain>/`
2. 编写 `tree.yaml`（参考上节格式）和节点边界定义
3. 确保目录结构与 Skill 规范一致
4. 在 PR 中描述知识域覆盖范围和深度

### 相互反哺

你贡献的知识域会同时出现在两个分发渠道中：

- **Skill 用户**可直接使用仓库内 `regulus-coach/`（或待发布的 Skill 市场安装）
- **App 用户**也能看到这些知识树
- 你的名字会出现在该知识域的贡献者列表中

---

## 开发约定

### 语言与协作

**原则：中文优先，不排斥英文。**

| 范围 | 要求 |
|------|------|
| 用户可见文案 | UI、错误提示、节点说明、面向读者的文档 — **须为中文**（或通过 i18n 键管理；当前仅 `zh`） |
| Issue / PR 描述 | **建议中文**，便于维护者快速理解；可用英文撰写，请附一两句中文摘要 |
| Commit message | **建议中文**（如 `修复:`、`节点:`）；小修、依赖升级等可用英文，保持标题清晰即可 |
| 代码注释 | **建议中文**；涉及标准库/API/协议名时保留英文术语无妨 |
| 标识符与代码 | 变量名、函数名、包名 — **英文**（Go/JS 惯例） |

英文 PR / Issue 不会被因语言拒绝；若描述较长，维护者可能请你补一句中文摘要，方便归档与检索。

### Commit Message

```
节点: 添加 Go 并发入门层节点定义

方向: 教练知识边界 · 节点定义

- goroutine 是什么
- 启动第一个 goroutine
- sync.WaitGroup 等待完成
```

格式：`类型: 简短描述`，类型**建议**用中文。没有强制格式，但要让人一眼看懂做了什么。

### 代码规范

- 面向用户的错误信息与 UI 文案用中文
- 代码注释建议中文；导出函数须有注释（中或英均可，以说清意图为准）
- 变量名和函数名用英文（Go/JS 惯例）
- 不追求完美，追求可用

### 测试

- 核心 Agent 逻辑必须有测试
- 提交 PR 前跑 `make test`（或 `go test ./...` + `cd web && pnpm exec tsc --noEmit && pnpm build`）
- CI 会在 PR 上自动执行相同检查
- 不要求 100% 覆盖率，但关键路径必须有

---

## 提 Issue

issue 不分类，用前缀区分：

- `[Bug]` — 描述现象 + 复现步骤
- `[需求]` — 你想要什么功能，为什么需要
- `[节点]` — 想加什么知识领域/节点
- `[讨论]` — 不确定的方案，想听听想法
- `[体验]` — 用着不舒服的地方

---

## 分支与工作流

`main` 是**唯一长期分支**，始终代表可部署的最新代码。不要直接向 `main` push（维护者同样遵守）。

### 分支命名

| 前缀 | 用途 | 示例 |
|------|------|------|
| `feat/` | 新功能 | `feat/node-requires` |
| `fix/` | Bug 修复 | `fix/coach-prereq-prompt` |
| `docs/` | 仅文档 | `docs/contributing-workflow` |
| `chore/` | CI、依赖、脚本 | `chore/docker-publish-main` |

### 贡献者流程

1. Fork 仓库（或直接 clone 后加 upstream）
2. 从最新 `main` 拉分支：`git checkout -b feat/your-topic main`
3. 改代码；需要时加测试
4. 本地验证：`go test ./...`，UI 变更再跑 `cd web && pnpm exec tsc --noEmit && pnpm build`
5. Push 分支，向 `main` 提 **Pull Request**
6. 等 **CI 全绿** 后再合并；维护者 review 后 Squash merge

### 维护者流程

与贡献者相同：**一律通过 PR 合并到 `main`**，不直接 push `main`（含文档小改）。好处是留审查记录、触发 CI、Docker Publish 只在 merge 后跑。

| 场景 | 做法 |
|------|------|
| 日常功能 / 修复 | 分支 → PR → CI 绿 → Squash merge |
| 紧急热修 | `fix/hotfix-xxx` 分支，仍走 PR，可 self-merge |
| Merge 之后 | 看 Actions（CI + Docker Publish）；确认 GHCR `latest` 可 pull |
| 版本发布 | 打 tag `v*`（触发镜像多 tag）；写 GitHub Release 说明 |

GitHub 仓库设置（分支保护、Packages 公开、Secrets 等）见 **[docs/github-maintenance.md](./docs/github-maintenance.md)**。

---

## 提 PR

1. 目标分支固定为 **`main`**
2. 填写 PR 模板：变更说明、测试勾选、UI 变更附截图
3. 关联 Issue：`Fixes #123`（如有）
4. 确保 CI 通过后再请求 review 或合并

小改动也欢迎 PR；维护者通常会在 1–2 个工作日内看。大改动请先在 Issue 里 `[讨论]` 对齐方案。

---

## 更多问题

- 看 [DESIGN.md](./DESIGN.md) 了解设计理念
- 维护者与 GitHub 配置见 [docs/github-maintenance.md](./docs/github-maintenance.md)
- 安全漏洞报告见 [SECURITY.md](./SECURITY.md)
- 在 Issues 里直接问，不用先读什么

---

> 这个项目还在早期。你来得越早，留下的印记越深。
> 不管是一个知识节点、一次 Issue、还是一条分享，都算参与了这件事。
