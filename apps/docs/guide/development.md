# 本地开发

面向参与改代码的开发者。若只想在本机**使用** Regulus，见 [自托管部署](./self-host.md)；若只想快速试用，见 [快速上手](./quick-start.md)。

## 环境要求

| 工具 | 版本 |
|------|------|
| Go | 1.22+（见仓库 `go.mod`） |
| Node.js | 18+ |
| pnpm | 用于前端与文档站 |

## 启动

```bash
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy

cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY

# 终端 1：后端
go run ./cmd/server

# 终端 2：前端（Vite dev）
cd web && pnpm install && pnpm dev
```

浏览器打开 http://localhost:5173 。主路径：**输入领域 → 选节点 → 对话学习**。

或使用仓库根目录一条命令（同时起 Go API + Vite）：

```bash
pnpm install
pnpm dev
```

## 文档站预览

```bash
pnpm dev:docs
```

构建前会自动同步 `docs/screenshots/` 到文档站 `public/`。

## 测试

```bash
make test
```

## 仓库结构（简表）

| 路径 | 说明 |
|------|------|
| `cmd/server/` | Go HTTP 入口 |
| `internal/agent/` | 教练状态机、点亮逻辑 |
| `internal/domain/` | 建课、知识树、课程关联 |
| `web/src/` | 前端页面与知识图谱 |
| `regulus-coach/` | Skill、节点 YAML、Prompt |
| `apps/docs/` | 本文档站（VitePress） |

PR 流程见 [参与贡献](./contributing.md)，节点 YAML 与教练逻辑要求见 [教学质量](./contributing-teaching.md)。

## 相关

- [环境变量](../reference/env.md) — LLM、教练门槛、IM、Cloud
- [参与贡献](./contributing.md) — 如何提 Issue / PR
- [教学质量](./contributing-teaching.md) — 改节点 YAML 或教练点亮逻辑
