# Regulus Academy — 碎片化学习 AI 私教

![Banner](./docs/banner.png)

> 在碎片时间完成：讲解 → 练习 → 反馈 → 点亮 → 沉淀为可带走的知识。
>
> **有边界的知识地图 · 会追着你练完的教练 · 越学越懂你**

**状态：** 公开试用 · 知识图谱 · 教练闭环 · 画像裁剪 · Obsidian 导出 MVP · 行动助手 · 课程体检 ✅　|　复习闪卡 · Agent 维护笔记 · RAG · **Agent 主动日推** — 规划中

### 在线体验

| | 链接 |
|---|------|
| **在线 Demo** | https://demo.awoshuile.cn |
| **使用文档** | https://regulus-academy-docs.vercel.app |
| **GitHub** | https://github.com/liuwenji007/regulus-academy |

- **Demo**：无需先备 Key（平台日额度；用尽后可填写自己的 Key 继续）
- **自托管 / 长期使用**：一个 OpenAI 兼容 Key 即可；数据在本机。见 [自托管部署](https://regulus-academy-docs.vercel.app/guide/self-host)

---

## 差在哪

相对「用 ChatGPT / Claude 聊天学」：

1. **有边界的知识地图，不是无限聊天** — 每个节点钉死讲什么 / 易错什么 / 不讲什么
2. **会追着你练完并点亮的教练，不是讲完就走的老师** — 讲解 → 练习 → 批改 → 点亮
3. **越学越懂你——讲解按你的背景裁剪** — 不重复你会的，只练你弱的

Cloud 先免费试用；自托管则进度与笔记留在本机。

曾认真用过 [OpenMAIC](https://github.com/THU-MAIC/OpenMAIC)（学到结构化呈现与多 Agent）与 [DeepTutor](https://github.com/HKUDS/DeepTutor)（学到练习生成与 RAG 思路）；它们优秀，但课堂节奏或配置门槛不适合碎片场景——Regulus 用预定义边界代替 RAG，单次闭环即可完成一个可测量进步。

---

## 你会得到什么

| 结果 | 说明 |
|------|------|
| **节点可点亮** | 讲练批闭环；熟悉/精通层含应用题与掌握度评估 |
| **地图有边界** | 入门 / 熟悉 / 精通分层；多领域知识图谱可视进度 |
| **越学越懂你** | 全局画像 + 按课摘要，建课与讲解按缺口裁剪 |
| **学完可带走** | 导出 Domain 包、Coach Skill、Obsidian 学习笔记 |

另外：不知从哪开课可用 [内置课目录](https://regulus-academy-docs.vercel.app/guide/features#建课与导入)；节奏乱可用 [行动助手](https://regulus-academy-docs.vercel.app/guide/action-assistant)；课质量可 [体检与优化](https://regulus-academy-docs.vercel.app/guide/course-audit)。

---

## 立刻开始

### A. 在线 Demo（推荐，零 Key）

打开 [demo.awoshuile.cn](https://demo.awoshuile.cn) → 创建角色 → 输入主题或从「课程目录」开课 → 选节点练到点亮。

限额与限制见 [在线体验版](https://regulus-academy-docs.vercel.app/guide/cloud-demo)。

### B. 本机 Docker（数据留本机，需 Key）

先装 [Docker Desktop](https://www.docker.com/products/docker-desktop/)。

**一键脚本**（自动 clone、拉镜像、启动）：

```bash
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
```

**或手动三步**（想自己控制目录）：

```bash
# 1. 拉代码
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy

# 2. 配 Key（编辑 .env，填入 LLM_API_KEY；可用 DeepSeek / OpenAI / 任意兼容接口）
cp .env.example .env

# 3. 起服务（拉预构建镜像，约 30 秒～2 分钟）
docker compose -f docker-compose.image.yml up -d
```

打开 **http://localhost:8080**，输入「Go 并发」即可开练（8080 被占用时在 `.env` 设 `HOST_PORT`）。更新只需重跑一键脚本。

### C. 源码运行（不装 Docker）

> 后端是 Go，前端是 Vite/Node，需要 Go + Node(pnpm) 两个运行时，开两个进程。

```bash
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy
cp .env.example .env            # 填入 LLM_API_KEY

# 终端 1：Go 后端
go run ./cmd/server

# 终端 2：前端（Vite dev）
cd web && pnpm install && pnpm dev
```

打开 **http://localhost:5173** 开练（后端默认 8080，前端 dev 已配代理）。

IM、环境变量与教练调优见 [自托管部署](https://regulus-academy-docs.vercel.app/guide/self-host) · [本地开发](https://regulus-academy-docs.vercel.app/guide/development) · [环境变量](https://regulus-academy-docs.vercel.app/reference/env)。

---

## 长什么样

完整图集见 [界面预览](https://regulus-academy-docs.vercel.app/guide/screenshots)。

### 入口与学习路径

| 开始学习 | 课程详情 | 我的课程 |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/home.png" width="260" alt="开始学习页" /> | <img src="./docs/screenshots/tree.png" width="260" alt="课程详情" /> | <img src="./docs/screenshots/courses.png" width="260" alt="我的课程" /> |

### 教练闭环与建课

| AI 教练 · 练习反馈 | PDF / URL 导入建课 |
|:---:|:---:|
| <img src="./docs/screenshots/coach-exercise.png" width="260" alt="教练练习与批改" /> | <img src="./docs/screenshots/import.png" width="260" alt="导入建课" /> |

### 知识图谱

| 图谱 · 宣纸（默认） | 图谱 · 星空 | 目录 |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/graph-paper.png" width="260" alt="知识图谱·宣纸" /> | <img src="./docs/screenshots/graph-sky.png" width="260" alt="知识图谱·星空" /> | <img src="./docs/screenshots/graph-outline.png" width="260" alt="知识图谱·目录" /> |

### 在线体验版（Cloud）

| Cloud 首页 | 角色创建 | 设置 |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/cloud-home.png" width="260" alt="Cloud 首页" /> | <img src="./docs/screenshots/cloud-profile.png" width="260" alt="角色选择" /> | <img src="./docs/screenshots/cloud-settings.png" width="260" alt="Cloud 设置" /> |

---

## 为什么做这个

我是一个人到中年的在职工程师。

不是刚毕业那种焦虑，是另一种——技术在快速更新，我上班消耗掉大部分精力，下班只剩碎片时间。我买过视频课，面对 48 节通关的课程，只看了 3 节；我读过技术书，翻了目录就搁置；我也用 AI 聊天学习，对话很热闹，但第二天什么都没留下。

更麻烦的是，拿起一个新技术栈，完全不知道要学到什么程度才算「会了」。看完文档？能写 demo？还是能上生产？这种模糊让我一直拖着不敢开始。

2025 年底，我决定自己做一个工具来解决这个问题。不是「AI 家教」，不是「智能课堂」，而是一个教练——知道我在哪、记得我已会什么、只纠正最该纠正的那一个动作，在约 15 分钟里完成一次可测量的进步。

**这个项目是我给自己的礼物，也是给所有和我一样、想在碎片时间里保持成长的人的礼物。**

设计原则见 [DESIGN.md](./DESIGN.md)；站内短述见 [为什么是 Regulus](https://regulus-academy-docs.vercel.app/guide/why-regulus)。

---

## 怎么跑起来（一瞥）

| 入口 | 给谁 | 说明 |
|------|------|------|
| **Web（自托管）** | 默认路径 | Docker 一键；SQLite 在本机；单二进制嵌入前端 |
| **Cloud Demo** | 先试用 | 公网实例；有日配额；无 IM |
| **Coach Skill** | Agent / IDE 用户 | 主页下载 lite zip；Linked / Agent-lite / 可选 CLI |
| **IM** | 自托管 | Telegram / 钉钉 / 飞书 / 企微；部分平台需公网 HTTPS，见 [IM 频道](https://regulus-academy-docs.vercel.app/guide/im) |

技术选型：Go + SQLite + OpenAI 兼容 API。不必先配 Embedding / RAG。

---

## 加入我们

- 改进教练点亮或节点教考质量 → [教学质量](https://regulus-academy-docs.vercel.app/guide/contributing-teaching)
- 贡献知识域（YAML 定义节点边界即可）→ [CONTRIBUTING.md](./CONTRIBUTING.md)
- 试用反馈 → [`[体验]` Issue](https://github.com/liuwenji007/regulus-academy/issues/new?template=experience_feedback.yml)

**Star 是最简单的支持，Issues 是最好的对话。**

---

## 许可证

Apache 2.0。

自托管：学习数据在本机 SQLite。Cloud Demo：共享试用环境，请勿放入敏感内容；额度用尽后可填写自己的 Key 继续。

安全漏洞报告见 [SECURITY.md](./SECURITY.md)。
