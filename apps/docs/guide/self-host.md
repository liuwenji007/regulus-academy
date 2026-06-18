# 自托管部署

自托管**不要**设置 `REGULUS_DEPLOYMENT=cloud`，行为与开源版一致，数据留在本机 SQLite。

## 一键安装（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
```

## Docker 镜像

```bash
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy
cp .env.example .env
docker compose -f docker-compose.image.yml up -d
```

访问 `http://localhost:8080`（默认端口，见 `.env` 中 `HOST_PORT`）。

## 更新 / 重新部署

再次执行一键安装脚本即可拉取最新镜像并**自动重建、启动**容器，同时清理无用旧镜像：

```bash
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
```

或在安装目录手动执行：

```bash
cd ~/regulus-academy   # 或你的 clone 目录
git pull
docker compose -f docker-compose.image.yml up -d --pull always --force-recreate
docker image prune -f
```

保留旧镜像可加：`REGULUS_SKIP_IMAGE_PRUNE=1 bash scripts/install.sh`

## 主要页面

| 路由 | 用途 |
|------|------|
| `#/` | 开始学习（输入领域、建课） |
| `#/import` | 从 PDF 或网页 URL 导入材料并蒸馏建课 |
| `#/graph` | 知识图谱（图谱/目录双视图，宣纸/星空双主题） |
| `#/courses` | 我的课程 |
| `#/tree/:id` | 课程详情（纵深扩展、Domain 包 / 学习笔记导出） |
| `#/coach/:sessionId` | AI 教练对话 |
| `#/settings` | 设置 |
| `#/settings/profile` | 学习画像 |
| `#/settings/channels` | IM 频道（仅自托管） |

## IM 频道

在 Telegram、钉钉、飞书等与教练对话，进度与 Web 同步。配置步骤、绑定方式与自然语言导航见 **[IM 频道](./im.md)**。

在线 Demo 未开放 IM；环境变量见 [环境变量 · IM Gateway](../reference/env.md#im-gateway)。

## Agent 离线练习导出

主页 **「Agent 离线练习」** 下载 lite 版 `regulus-coach.zip`（协议、schemas、内置 domains、API 脚本）。可选通过 `GET /api/coach/cli?platform=...` 或 [GitHub Releases](https://github.com/liuwenji007/regulus-academy/releases) 安装 `regulus` CLI。

详见 **[Agent 离线练习](./agent-offline.md)** 与包内 `USAGE.md`。

## 模型配置

在 `.env` 中配置 `LLM_API_KEY`，或在 Web **设置 → AI 模型** 管理。详见 **[AI 模型配置](./model-config.md)**。环境变量完整列表见 [环境变量](../reference/env.md)。参与改代码见 [本地开发](./development.md)。
