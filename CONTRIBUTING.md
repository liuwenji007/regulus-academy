# Regulus Academy — 开源贡献手册

感谢你有兴趣参与！这份手册会告诉你从哪里开始、怎么协作、以及我们的工作方式。

---

## 行为准则

这是社区的最低共识，不是公司的员工手册。

- **友善** — 每个人都是从某个节点开始学习的，包括你和我
- **务实** — 先跑通再去讨论"完美的方案"
- **尊重用户** — 任何设计决策，先想"一个通勤 15 分钟的在职开发者会不会想用"
- **中文优先** — 代码注释、commit message、issue 讨论全部用中文

---

## 项目概况

| | |
|---|---|
| 定位 | 面向在职开发者的碎片化学习 AI 私教 |
| 技术栈 | Go (后端) + PWA (前端) + SQLite + DeepSeek API |
| 语言 | 全部中文 |
| 许可证 | TBD |
| 沟通 | GitHub Issues |

---

## 我能贡献什么？

### 你是开发者

| 技能 | 你能做 |
|------|--------|
| Go 后端 | 实现 Agent 逻辑、记忆管理、API 路由 |
| 前端开发 | PWA 页面、知识树可视化、对话 UI |
| AI/LLM | 优化 prompt、设计教学策略、改进错题强化 |
| 测试 | 写单元测试、集成测试、体验找 bug |

### 你不是开发者

| 你能做 | 说明 |
|--------|------|
| 定义知识节点 | 写一个知识点的核心概念、常见误区、边界 |
| 翻译 / 国际化 | 目前只支持中文，未来可能需要英文版 |
| 体验反馈 | 用一用，告诉我们哪里卡住、哪里不快 |
| 文档 | 写教程、改 README、翻译设计文档 |
| 布道 | 在社区分享、写文章 |

---

## 快速开始

```bash
# 1. 克隆仓库（待公开后）
git clone git@github.com:<你的账户>/regulus-academy.git
cd regulus-academy

# 2. 启动后端（需要 DeepSeek API Key）
cp .env.example .env
# 编辑 .env，填入 DEEPSEEK_API_KEY

go run cmd/server/main.go

# 3. 启动前端
cd web
pnpm install
pnpm dev
```

---

## 项目结构（规划）

```
regulus-academy/
├── cmd/
│   └── server/          # 后端入口
├── internal/
│   ├── agent/           # Agent 核心逻辑
│   │   ├── coach.go     # 教练行为：建树、教学、反馈
│   │   ├── memory.go    # 三层记忆管理
│   │   └── prompt.go    # System prompt 构建
│   ├── domain/          # 知识领域定义
│   │   ├── registry.go  # 领域注册（Go并发、Agent原理...）
│   │   ├── go_concurrency/
│   │   │   ├── tree.yaml # 三层知识树
│   │   │   └── nodes/    # 每个节点的边界定义
│   ├── storage/
│   │   └── sqlite.go    # SQLite 操作
│   └── api/
│       └── handler.go   # HTTP 路由
├── web/                 # PWA 前端
│   ├── src/
│   │   ├── pages/
│   │   │   ├── domain/  # 知识树页面
│   │   │   └── coach/   # 对话页面
│   │   ├── components/  # 共享组件
│   │   └── lib/         # 工具函数
├── docs/                # 文档
├── DESIGN.md            # 设计理念
├── PLAN.md              # 项目规划
└── CONTRIBUTING.md      # 本文件
```

---

## 加一个新的知识领域

这是最高价值也最低门槛的贡献方式。你不需要会写代码。

每个知识领域需要两样东西：

### 1. 三层知识树（`tree.yaml`）

```yaml
domain: Go 并发
description: 用 goroutine 和 channel 写出并发安全的 Go 程序

layers:
  entry:
    label: 入门
    time: ~2 小时
    goal: 能看懂并发代码，能创建简单的 goroutine
    nodes:
      - goroutine 是什么
      - 启动第一个 goroutine
      - sync.WaitGroup 等待完成

  intermediate:
    label: 熟悉
    time: ~8 小时
    goal: 能独立写生产级并发代码
    nodes:
      - channel 通信
      - select 多路复用
      - context 超时控制
      - sync.Mutex 互斥锁
      - sync.RWMutex 读写锁
      - 并发模式：生产者-消费者

  advanced:
    label: 精通
    time: ~20 小时
    goal: 理解调度模型，能排查并发 bug
    nodes:
      - GMP 调度模型
      - channel 底层数据结构
      - sync.Pool 对象复用
      - race condition 检测与修复
      - 内存模型与 happens-before
```

### 2. 节点边界定义（`nodes/<节点名>.yaml`）

```yaml
node: channel 通信
layer: 熟悉

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
```

---

## 以 Skill 文件贡献知识领域

除了在仓库代码中定义知识节点，你也可以将完整的知识域打包为 `regulus-coach` Skill 文件。

### Skill 文件结构

```
regulus-coach/
├── SKILL.md              # 教练行为定义
├── domains/              # 知识领域
│   └── your-domain/
│       ├── tree.yaml     # 三层知识树
│       └── nodes/        # 节点边界定义
└── tools/
    └── progress.py       # 进度追踪脚本（可选）
```

### 贡献步骤

1. 创建知识域目录：`domains/<your-domain>/`
2. 编写 `tree.yaml`（参考上节格式）和节点边界定义
3. 确保目录结构与 Skill 规范一致
4. 在 PR 中描述知识域覆盖范围和深度

### 相互反哺

你贡献的知识域会同时出现在两个分发渠道中：

- **Skill 用户**可以直接安装使用
- **App 用户**也能看到这些知识树
- 你的名字会出现在该知识域的贡献者列表中

---

## 开发约定

### Commit Message

```
节点: 添加 Go 并发入门层节点定义

方向: 教练知识边界 · 节点定义

- goroutine 是什么
- 启动第一个 goroutine
- sync.WaitGroup 等待完成
```

格式：`类型: 简短描述`，类型用中文。没有强制格式，但要让人一眼看懂做了什么。

### 代码规范

- 所有注释和错误信息用中文
- 变量名和函数名用英文（Go/JS 惯例）
- 导出函数必须有注释
- 不追求完美，追求可用

### 测试

- 核心 Agent 逻辑必须有测试
- 提交 PR 前跑 `go test ./...`
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

## 提 PR

1. 从 `main` 拉一个新分支
2. 改代码 + 加测试（如果需要）
3. 自己跑一遍看没有挂
4. 提 PR，描述清楚改了什么、为什么改
5. 等人 review（或自己先合并，我们会看）

我们没有严格的 review 流程。小修小改可以直接合，大改动等 1-2 个人看过再合。

---

## 更多问题

- 看 [DESIGN.md](./DESIGN.md) 了解设计理念
- 看 [PLAN.md](./PLAN.md) 了解项目规划
- 在 Issues 里直接问，不用先读什么

---

> 这个项目还在早期。你来的越早，留下的印记越深。
