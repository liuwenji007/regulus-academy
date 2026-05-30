# Regulus Academy — 碎片化学习 AI 私教

> 面向在职开发者的 Agent 与计算机知识碎片化学习 AI 教练。
> 不模拟课堂，而是像一个耐心的教练，在你每一个 15 分钟的碎片时间里，带你完成一次最小单位的训练。

**状态：Phase 2 完成 · Phase 2.5 可用性增强**

---

## 快速开始

```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY（见下方模型配置说明）

# 2. 启动后端
go run ./cmd/server
# 启动日志会显示：LLM: DeepSeek / deepseek-chat

# 3. 启动前端（新终端，开发模式）
cd web && pnpm install && pnpm dev
```

浏览器打开 http://localhost:5173 ，输入「Go 并发」，即可加载内置知识树并进入 AI 教练对话。

**Docker 一键启动（含前端构建）：**

```bash
cp .env.example .env
docker compose up --build
# 访问 http://localhost:8080
```

### 模型配置（`.env`）

```bash
# 推荐：统一变量
LLM_PROVIDER=deepseek    # deepseek | openai | openrouter | ollama | custom
LLM_API_KEY=sk-...
# LLM_BASE_URL=          # 只填域名，不要带 /v1/chat/completions
# LLM_MODEL=             # 可选，覆盖预设模型

# 兼容旧变量
DEEPSEEK_API_KEY=
DEEPSEEK_BASE_URL=https://api.deepseek.com
```

首页会显示「模型已连接」；未配置时提示修改 `.env` 并重启后端。

**运行测试：**

```bash
make test
```

更多说明见 [CONTRIBUTING.md](./CONTRIBUTING.md) 与 [PLAN.md](./PLAN.md)。

---

## 一、核心定位

一个面向在职开发者的、碎片化学习 Agent 与计算机知识的 AI 私教。它不模拟课堂，而是像一个耐心的教练，在你每一个 15 分钟的碎片时间里，带你完成一次最小单位的训练。

**产品价值主张：以「实战技能提升」为主，「面试准备」为辅。**

用户学完后有两个获得感：
1. **今天真的会写 goroutine 了**（实战能力，主线）
2. **面试问到也不怕**（面试覆盖，副线）

## 二、目标用户画像

| 特征 | 描述 |
|------|------|
| 身份 | 想转型 AI/后端的前端开发、想系统学习 Agent 的在职工程师 |
| 痛点 | 时间碎片化、精力被工作耗尽、面对庞大知识体系感到迷茫 |
| 需求 | 一个能告诉他"下一步学什么"、给他微小任务、并在他完成时给予反馈的陪伴者 |
| 场景 | 通勤地铁、午休间隙、哄睡孩子后的深夜一小时 |

## 三、竞品格局

| 竞品 | 定位 | 可借鉴之处 | 需要避开的坑 |
|------|------|------------|-------------|
| OpenMAIC (清华) | 模拟真实课堂的多角色 Agent | 多 Agent 协作的交互设计、知识点的结构化呈现 | 过于沉浸式，不适合碎片化场景。不要做"课堂"，要做"教练"。 |
| DeepTutor (港大) | 通用个人学习助手 | RAG 检索增强、自动生成练习的机制 | 功能太丰富，界面容易让用户感到认知过载。保持极简，一个时间只做一件事。 |
| **本项目** | **碎片化学习的 AI 私教** | — | — |

## 四、MVP 核心功能（最小闭环）

| 功能模块 | 具体描述 | 优先级 |
|----------|----------|--------|
| 1. 知识树入口 | 用户选择学习大方向（如"Go 语言后端"、"Agent 原理"），展示结构化的学习路径。 | P0 必须 |
| 2. 每日推荐 | Agent 根据用户进度，推荐一个 15 分钟内可完成的微任务（如"理解 goroutine 并完成一个小练习"）。 | P0 必须 |
| 3. 讲解与练习 | Agent 用通俗语言讲解知识点，并给出一个具体的、微小的编码或问答题。 | P0 必须 |
| 4. 反馈与点亮 | 用户完成后，Agent 给予鼓励性反馈，并在知识树上"点亮"该节点。 | P0 必须 |
| 5. 进度面板 | 展示用户已学/未学节点，让成长可视化。 | P1 重要 |
| 6. 自定义 API Key | 允许用户填入自己的 DeepSeek API Key，保护隐私、降低成本。 | P1 重要 |

## 五、技术架构

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 分发入口 | Skill + Local Web + IM Channel | Skill 装到 Agent/IDE；Docker 本地跑 Web；企微/飞书/钉钉机器人做通信管道 |
| 后端 | Go (net/http) | 处理用户请求、管理学习状态、调用 DeepSeek API、Channel 消息路由 |
| 前端 | 纯 HTML + CSS + 原生 JS | 极简 Web 页面：知识树可视化 + 对话界面。无框架，零构建 |
| 数据库 | SQLite | 存储用户学习进度、知识树结构、历史对话。跨端共享通过后端统一 |
| 模型 | DeepSeek API | 用户自填 Key。核心能力：学习计划生成、知识点讲解、练习批改 |
| 监控 | Langfuse | 追踪 Agent 调用，调试和优化教学策略 |

## 七、开源与社区策略

| 行动 | 目标 |
|------|------|
| 源码可见 (Source Available) | 展示核心 Agent 逻辑和架构，保护产品创意。 |
| 提交到 awesome-deepseek-integration | 获得官方生态认证，进入开发者视野。 |
| 在 README 里讲你的故事 | 让贡献者被你的初心打动，而不是仅仅被代码吸引。 |
| 在 DeepSeek 社群分享 | 告诉群友："我用 DeepSeek 做了一个能教人学 Agent 的 Agent。" |

---

## 六、分发策略：Skill × Local × Channel 三层

Regulus 有三层分发方式，从零门槛到团队部署，用户按需选择：

### 第一层：Skill（零门槛，装到自己的 Agent/IDE 里）

教练能力抽象为 Agent Skill，安装到 Hermes、Claude Code 或支持 Skill 的 IDE 中：

```bash
hermes skills install regulus-coach
```

装好后 Agent 或 IDE 立即具备教练能力——建知识树、15 分钟教学、无感错题强化。

### 第二层：Local（本地运行，有 Web 页面）

```bash
git clone git@github.com:<你的账户>/regulus-academy.git
cd regulus-academy
cp .env.example .env   # 填入 DEEPSEEK_API_KEY
docker compose up
# 浏览器打开 http://localhost:8080
```

提供可视化知识树、对话界面、进度管理。不依赖任何 Agent 框架，只要有 Docker 就能跑。

### 第三层：Channel（IM 机器人）

Regulus 在本地运行，通过机器人接入 **Telegram、钉钉、飞书、企业微信**。教学逻辑与进度在本地 SQLite，IM 只是通信管道。

```bash
# .env 中开启 Gateway 并填入对应平台凭证
GATEWAY_ENABLED=true
TELEGRAM_BOT_TOKEN=...        # @BotFather 创建
DINGTALK_CLIENT_ID=...         # 钉钉开放平台 → Stream 机器人
FEISHU_APP_ID=...             # 飞书开放平台
FEISHU_MODE=websocket         # websocket（默认）| webhook（需公网 /webhook/feishu）
# 企微需公网 HTTPS：WECOM_ENABLED=true + 回调 URL /webhook/wecom

go run ./cmd/server
```

**首次使用**：在 IM 里发送 `绑定 你的角色名`（角色需先在 Web 端创建），然后：

| 命令 | 说明 |
|------|------|
| `课程` | 查看知识库 |
| `学习 1` | 查看课程节点 |
| `节点 1` | 开始/继续学习 |
| `继续` | 查看当前学习状态 |
| `帮助` | 命令列表 |

- Telegram / 钉钉 / 飞书（`FEISHU_MODE=websocket`）：内网可跑
- 飞书（`FEISHU_MODE=webhook`）/ 企微：需公网 HTTPS
- Web 与 IM **共用同一角色**的学习进度与聊天记录

### Shared Memory（共享记忆）

三层入口共享同一份记忆——终端上学到一半，换 Web 继续，手机企微里复习。知识树进度、错题记录、教学偏好跨端同步。它们只是容器，教练是同一个。

### Skill 文件结构

```
regulus-coach/
├── SKILL.md              # 教练行为定义
│   ├── 触发条件
│   ├── 建树流程
│   ├── 教学流程
│   └── 记忆管理策略
├── domains/              # 知识领域
│   ├── go-concurrency/
│   │   ├── tree.yaml     # 三层知识树
│   │   └── nodes/        # 节点边界定义
│   ├── agent-principles/
│   └── rag-architecture/
└── tools/
    └── progress.py       # 进度追踪脚本（可选）
```

---

## 八、关键注意点

1. **绝对避免"好学生综合症"** — 不要试图构建完整的知识体系再上线。先只做"Go 并发"这一个知识分支，跑通整个闭环，再扩展其他分支。
2. **坚守"教练"而非"老师"的定位** — 所有话术设计，都要避免"你应该学"。而是"我陪你练"、"这个知识点，我们一起拆解一下"。
3. **知识树必须源于你的实战经验** — 不要照搬教科书目录。把你自学 Go 时遇到的困惑、踩过的坑，都变成知识树上的节点。这才是项目不可替代的价值。
4. **练习必须"微小"** — 一个练习，用户必须在 15 分钟内能完成。否则他就会感到挫败，然后去打游戏。
5. **先跑通，再开源** — 用"私有开发，公开镜像"策略。在私有仓库里把核心闭环跑通，再推送到公开仓库。
6. **数据隐私是信任基石** — 从一开始就支持用户自定义 API Key，并在 README 里明确说明：你的数据，只有你和模型知道。

## 九、下一步行动

- [x] **今天**：体验 OpenMAIC 和 DeepTutor，带着竞品侦察表，记录发现。
- [x] **本周**：在私有仓库里，搭建最简脚手架（一个 Go 后端 + 一个能发消息的前端页面）。
- [ ] **下周末前**：完成"Go 并发"这一个知识分支的最小闭环（用户选择 → Agent 讲解 → 用户练习 → Agent 反馈 → 点亮节点）。

---

## 十、竞品体验记录（DeepTutor v1.4.1 实测）

> 2026-05-27 实际安装使用后的发现

### 配置门槛高
- 安装后需要依次配置：LLM Provider → Embedding → Web Search，三步都填 API Key
- 大部分普通用户只有一个大模型的 API Key，Embedding 和搜索服务的配置劝退
- 我们的应对：**只填一个 Key 就能用**，其他能力降级或内建

### 界面全英文
- 设置页、交互流程全是英文，对中文用户不友好
- 我们的应对：**中文优先**，UI 和 Agent 话术都用中文

### 调用链路长、慢
- 频繁调用模型接口，把返回结果拼凑成问答
- 体验上感觉是在"组装内容"而非"教学对话"
- 最终卡住不动（可能是 API 超时或依赖链断裂）
- 我们的应对：**一次调用 = 一次教学**，不搞多轮拼凑。15 分钟内必须完成一个闭环

### 核心启发
- DeepTutor 像是一个"万能学习平台"，功能多但重
- 我们要做的是"碎片化教练"，功能少但轻
- **少即是多**：一个 Key、一个方向、一个微任务、一次反馈

### OpenMAIC 实测（2026-05-27）
- 以讲课为核心模式，画面和交互设计优秀
- 有"同学"角色陪伴，营造课堂氛围
- 问题：**节奏太慢**，沉浸式课堂需要整块时间投入
- 在职开发者没有那么多时间在里面"玩"
- 验证了之前的判断：**不要做课堂，要做教练**。15 分钟能完成，而不是 45 分钟一节课

### 技术决策：MVP 纯 LLM，不加 Embedding 和搜索
- Embedding（向量化）= RAG 检索用户文档，MVP 阶段不需要，知识树是预定义的，大模型已掌握
- 搜索服务 = 补充实时信息，MVP 阶段不需要，Go/Agent 等基础知识稳定，不依赖时效
- DeepTutor 需要这些是因为要做"万能平台"支持任意学科；我们只做开发者技术学习，模型训练数据已足够
- **过早引入只会增加配置门槛，跟 DeepTutor 一样劝退用户**
- 什么时候加：用户上传自己的笔记/代码让 AI 出题时 → 加 Embedding；学习内容更新很快（如框架每周发新版）→ 加搜索；支持冷门技术栈 → 加搜索

### 内容策略：不求知识库，求知识边界

Duolingo 靠人工制作内容，题库固定、练习重复。我们不需要知识库，依靠 LLM 动态生成内容。

**不需要大量预置内容，只需要定义每个节点的边界：**

```yaml
节点: channel 通信
层级: 熟悉
核心概念: [无缓冲 channel, 带缓冲 channel, 方向 channel]
常见误区: [忘了 close, 往已关闭的写, 死锁]
边界: 不讲 select，不讲 channel 底层实现
```

LLM 在这个框内自由发挥——生成讲解、出练习、批改答案。**内容是活的，边界是死的。**

| | Duolingo | 你的 Agent |
|---|----------|-----------|
| 内容来源 | 人工制作，固定 | LLM 动态生成 |
| 练习题 | 题库轮换，永远那几道 | 每次随机出，不重样 |
| 错题应对 | "再练一次" | 换个角度讲，换种方式考 |
| 新知识点 | 等团队更新 | 用户提到就能教 |

---

## 迭代记录

### 2026-05-27 · 首次规划

- 完成竞品分析：DeepTutor 配置太重、OpenMAIC 节奏太慢
- 确定定位：实战技能为主（2）、面试准备为辅（1）
- 技术决策：MVP 纯 LLM，不加 Embedding 和搜索
- 知识树设计：入门 → 熟悉 → 精通 三层，每层标明投入时间
- Agent 记忆管理方案：三层记忆 + 无感错题强化

---

### 记忆管理设计

#### 第一层：知识掌握度（核心，SQLite 持久化）

```yaml
user_progress:
  goroutine_basics:
    level: 入门
    status: completed
    mastery: 0.8
    last_practice: 2026-05-27
    attempts: 3
    mistakes:
      - concept: goroutine 不是线程
        wrong_count: 2
        last_wrong: 2026-05-20
        reinforcement_count: 1    # 已在练习中强化 1 次
      - concept: WaitGroup.Add 必须在 goroutine 之前
        wrong_count: 1
        last_wrong: 2026-05-25
        reinforcement_count: 0    # 下次练习塞进去
```

#### 第二层：教学偏好（辅助，SQLite 持久化）

```yaml
user_profile:
  preferred_depth: 熟悉
  pace: moderate
  weak_areas: [并发锁, 内存模型]
  skip_pattern: []
```

#### 第三层：会话上下文（短期，system prompt）

```yaml
session_context:
  current_node: channel_select
  last_message: 上次讲到 select 能在多个 channel 上等待
  pending_exercise: true
  recent_mistakes: [把 select 当 switch 用]
```

#### 无感错题强化

Agent 工作流程：
1. 翻档案 → 找到当前节点哪些 mistake 的 reinforcement_count < 2
2. 正常推进新内容
3. 在练习里**悄悄塞入**旧错误，用户不感知
4. 对了 → reinforcement_count +1，mastery +0.1；错了 → wrong_count +1，降低难度
5. 错误按遗忘曲线出现（第 1/3/7/15 天）
6. 强化 2 次且不再错 → 自动清除；3 次以上 → Agent 换个角度讲，不硬塞

