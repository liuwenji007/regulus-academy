# Regulus Coach Skill — 安装与教学使用

本文面向 **Agent 编排者** 与 **终端用户**。Agent 细则见 [SKILL.md](./SKILL.md)；精简协议见 [protocol-lite.md](./protocol-lite.md)。

## 1. 获取与安装

### 从 Web 下载（推荐）

1. 打开 Regulus 主页 `#/`
2. 点击右上角 **「Skill 下载」** 获取 `regulus-coach.zip`（**lite 包**，约几 MB，不含 CLI 二进制）
3. 解压到 Agent skills 目录，例如：
   - Cursor：`.cursor/skills/regulus-coach/`
   - OpenClaw：`~/.openclaw/skills/regulus-coach/`

### 目录结构

```
regulus-coach/
├── SKILL.md              # Agent 入口（模式选择、流程）
├── USAGE.md              # 本文件
├── protocol-lite.md      # Agent-lite 精简协议（zip 内含）
├── agent-prompts.md      # Agent-lite 讲解/出题/批改要点
├── build-domain.sh       # 缺课建课（三档降级）
├── scripts/
│   ├── api-session.sh    # Linked 模式 HTTP 会话
│   └── install-cli.sh    # 可选 CLI 一键下载
├── schemas/              # lite：exercise / grade / progress.schema.json
├── data/
│   ├── progress.json     # Agent-lite 本地进度
│   └── onboarding.json   # 首次引导完成标记（Agent 读写）
├── .regulus/
│   └── link.json.example # 关联已部署实例
└── domains/              # 内置课程（可叠加 Domain 包）
```

> **不在 lite zip 中**：`protocol.md`、`prompts/`、`triggers.yaml` 及 Web 专属 schemas（画像、mastery 等）——仅供服务端 / CLI / 开发者仓库使用。

### 可选：安装 regulus CLI（高保真离线）

默认 zip **不含** `bin/regulus`。首次使用时 Agent 会说明 Linked / Agent-lite / CLI 区别并询问是否需要 CLI。

需要安装时，在 Skill 根目录执行：

```bash
bash scripts/install-cli.sh           # 已配置 link.json 时从实例下载
bash scripts/install-cli.sh --github  # 或从 GitHub Releases
./bin/regulus doctor
```

也可手动从 [GitHub Releases](https://github.com/liuwenji007/regulus-academy/releases) 或 `GET /api/coach/cli?platform=...` 获取（见 `install-cli.sh` 源码）。

## 2. 三种运行模式

| 模式 | 条件 | 教练逻辑 | 进度 |
|------|------|----------|------|
| **Linked**（推荐） | `.regulus/link.json` | `scripts/api-session.sh` → HTTP API | 服务端 |
| **CLI**（可选） | `bin/regulus` + `LLM_API_KEY` | `regulus session` | `data/regulus.db`，可 sync |
| **Agent-lite**（默认离线） | 无 link、无 CLI | Agent 按 protocol-lite + schemas | `data/progress.json` |

### Linked 模式

```bash
cp .regulus/link.json.example .regulus/link.json
# 编辑 apiUrl、userId

bash scripts/api-session.sh start --slug go-concurrency
bash scripts/api-session.sh message --session <id> "开始练习"
```

关联同一实例时也可用 CLI：`./bin/regulus link --url <apiUrl>`。

### CLI 模式

在 Skill 根目录或 `data/.env` 配置 `LLM_API_KEY`，然后：

```bash
./bin/regulus session start --slug go-concurrency
./bin/regulus session message --session <id> "用户原话"
```

```bash
./bin/regulus link --url https://你的实例 --user-id default
./bin/regulus sync pull
./bin/regulus sync push
```

### Agent-lite 模式

1. 读 `domains/<slug>/tree.yaml` 确定节点顺序
2. 读 `nodes/<key>.yaml` 与 [agent-prompts.md](./agent-prompts.md)
3. 按 [protocol-lite.md](./protocol-lite.md) 讲解 → 出题 → 批改
4. 节点通过后更新 `data/progress.json`

## 3. 教学全流程

用户说「我想学 XXX」时：

### A. 确认课程

```bash
ls domains/
```

### B. 缺课则建课

```bash
bash build-domain.sh "用户原话"
```

脚本顺序：本地 `regulus build` → 远程 API 建课并下载 Domain 包 → 提示 Web 手动导出。

### C. 开课（按模式）

- Linked：`bash scripts/api-session.sh start --slug <slug>`
- CLI：`./bin/regulus session start --slug <slug>`
- Agent-lite：从 `tree.yaml` 取首个未完成节点，进入 explain

### D. 用户每次回复

- Linked：`bash scripts/api-session.sh message --session <id> "..."`
- CLI：`./bin/regulus session message --session <id> "..."`
- Agent-lite：按 phase 处理（见 protocol-lite）

### E. 叠加 Domain 包

```bash
unzip go-concurrency-domain.zip
cp -r go-concurrency domains/
```

## 4. 故障排查

| 现象 | 处理 |
|------|------|
| Linked 401/403 | 检查 `link.json` 的 `userId` 与 Web 一致 |
| 无 CLI | 使用 Agent-lite 或下载平台二进制 |
| 建课失败 | 配置 `LLM_API_KEY`；或 Web 建课后导出 Domain 包 |
| Agent 即兴编讲解 | Linked/CLI 违反协议；lite 模式须跟 schemas |

## 5. 与 Web 版差异

| | Web | Agent + 本 Skill |
|---|-----|------------------|
| 界面 | 知识树、图谱 | 对话由 Agent 呈现 |
| 教练逻辑 | 完整状态机 | Linked/CLI 同 Web；lite 为精简子集 |
| 进度 | 服务端 | Linked 服务端；CLI SQLite；lite progress.json |
| 画像 | 自动注入 | Linked/CLI 部分支持；lite 无画像 |

完整体验见 [自托管](https://regulus-academy-docs.vercel.app/guide/self-host) 或 [在线 Demo](https://demo.awoshuile.cn)。
