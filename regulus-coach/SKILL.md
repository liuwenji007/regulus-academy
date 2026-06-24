---
name: regulus-coach
description: Regulus Academy 碎片化学习 AI 教练。用户说想学某主题、要练某门课、或提到 regulus-coach 时使用。优先 Linked API 或 Agent-lite 协议；可选 regulus CLI 高保真离线。
---

# Regulus Academy Coach

安装与双模式说明见 **[USAGE.md](./USAGE.md)**。Agent 必读本文与 **[protocol-lite.md](./protocol-lite.md)**。

## 首次使用引导（Agent 必做）

在**第一次**帮用户练课之前，检查 `data/onboarding.json` 的 `completed` 是否为 `true`。若为 `false` 或文件不存在：

1. **用简短话术说明三种模式**（勿一次堆砌过长）：
   - **Linked**（已部署 Regulus 时推荐）：配置 `link.json`，走 `api-session.sh`，与 Web **共用进度**，无需本地 LLM Key。
   - **Agent-lite**（默认）：纯离线，Agent 按 `protocol-lite.md` + schemas，进度在 `data/progress.json`。
   - **CLI**（可选）：安装 `bin/regulus`，与 Web **同状态机**，适合要强离线、有 `LLM_API_KEY` 的用户。
2. **询问用户**：是否已部署 Regulus？是否需要安装 CLI？
3. **按回答行动**：
   - 有 Regulus → 引导 `cp .regulus/link.json.example .regulus/link.json` 并编辑 `apiUrl`。
   - 要 CLI → 在 Skill 根目录执行 `bash scripts/install-cli.sh`（有 link 时自动从实例下载，否则可加 `--github`）。
   - 纯离线、无 CLI → 说明 Agent-lite 即可，无需额外安装。
4. 将 `data/onboarding.json` 更新为 `completed: true`，可记录 `preferredMode`。

**不要在未完成引导前直接即兴讲解/出题**（Linked/CLI 模式尤其禁止）。

## 何时使用

- 用户要学习某知识域，或要继续已有课程练习
- 需要 **知识树导航**、**单节点讲解**、**微练习**、**作答批改**

## 模式选择（按序判断）

```
1. 存在 .regulus/link.json 且 apiUrl 可达？
   → Linked 模式（推荐）：scripts/api-session.sh + 远程 API
2. 存在可执行的 bin/regulus？
   → CLI 模式：regulus session（与 Web 同状态机）
3. 否则
   → Agent-lite：protocol-lite + agent-prompts + schemas + data/progress.json
```

**Linked 与 CLI 模式下禁止即兴扮演讲解/出题/批改。** Agent-lite 须遵循 `protocol-lite.md` 与 `schemas/`，并更新 `data/progress.json`。

## 入口流程

用户提出学习主题（如「我想学 TypeScript」）时：

1. **检查本地课程**：列出 `domains/` 子目录；读 `tree.yaml` 的 `domain` / `slug` 是否匹配。
2. **若无匹配课程**：
   - 在本 Skill 根目录执行 `bash build-domain.sh "用户原话"`
   - 禁止无 YAML 时编造知识树；禁止用其他 Skill 代替
   - 失败时说明原因，并引导 Web 建课 → 导出 Domain 包 → 解压到 `domains/`
3. **若有匹配课程**：按上方模式选择进入教练流程。

## Linked 模式（已部署 Regulus，推荐）

配置 `.regulus/link.json`（见 `.regulus/link.json.example`）：

```json
{ "apiUrl": "https://你的实例", "userId": "default" }
```

```bash
bash scripts/api-session.sh start --slug go-concurrency
bash scripts/api-session.sh message --session <id> "用户原话"
```

进度以服务端为准。可选 `POST /api/sync/progress` 合并离线进度（见 USAGE.md）。

## CLI 模式（可选高保真离线）

首次需要时，在 Skill 根目录运行（勿在 Web 主页下载）：

```bash
bash scripts/install-cli.sh          # 有 link.json 时从实例下载
bash scripts/install-cli.sh --github # 或从 GitHub Releases
./bin/regulus doctor
```

也可从 [GitHub Releases](https://github.com/liuwenji007/regulus-academy/releases) 手动放入 `bin/`。需 `.env` 中 `LLM_API_KEY`。

```bash
./bin/regulus session start --slug go-concurrency
./bin/regulus session message --session <id> "用户原话"
```

`regulus link` 可与 Linked 模式共用同一实例。

## Agent-lite 模式（默认离线）

无 link、无 CLI 时：

1. 读 `domains/<slug>/nodes/<key>.yaml` 与 [agent-prompts.md](./agent-prompts.md)
2. 按 [protocol-lite.md](./protocol-lite.md) 推进 explain → exercise → grade → completed
3. 出题/批改 JSON 符合 `schemas/exercise.json`、`schemas/grade.json`
4. 节点完成后更新 `data/progress.json`（格式见 `schemas/progress.schema.json`）

## 建课

`build-domain.sh` 按优先级：本地 CLI → 远程 API 建课并拉取 Domain 包 → 提示 Web 手动导出。

## 叠加新课程

Web 课程详情导出 `{slug}-domain.zip`，解压后将 `{slug}/` 复制到 `domains/`。

## 贡献

见仓库 [CONTRIBUTING.md](../CONTRIBUTING.md)。
