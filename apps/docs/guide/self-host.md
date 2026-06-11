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

访问 `http://localhost:8080`（默认端口，见 `.env` 中 `PORT`）。

## 主要页面

| 路由 | 用途 |
|------|------|
| `#/` | 开始学习（输入领域、建课） |
| `#/import` | 从 PDF 或网页 URL 导入材料并蒸馏建课 |
| `#/graph` | 知识银河（多领域全景） |
| `#/courses` | 我的课程 |
| `#/tree/:id` | 课程详情（纵深扩展、Skill/笔记导出） |
| `#/coach/:sessionId` | AI 教练对话 |
| `#/settings` | 设置 |
| `#/settings/profile` | 学习画像 |
| `#/settings/channels` | IM 频道（仅自托管） |

## IM 频道

在 Telegram、钉钉、飞书等中与教练对话，进度与 Web 同步。

1. 打开 **设置 → IM 频道**（`#/settings/channels`）
2. 开启总开关并填写平台凭证，**保存后重启服务**
3. 在 IM 单聊中发送「绑定 角色名」或 6 位绑定码
4. 用自然语言或命令导航；进入节点后直接发消息与教练对话

常用说法：`课程`、`学习 1`、`节点 1`、`继续`、`下一节`、`进度`、`帮助`。

在线 Demo 未开放 IM，需在自托管环境配置。部署细节见仓库 [`deploy/README.md`](https://github.com/liuwenji007/regulus-academy/blob/main/deploy/README.md)。

## 模型配置

在 `.env` 中配置 `LLM_API_KEY` 与 `LLM_PROVIDER`（deepseek / openai / ollama 等）。首页显示「模型已连接」即表示就绪。

环境变量完整列表见 [参考 · 环境变量](../reference/env.md)。
