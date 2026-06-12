# 快速上手

## 方式 A：在线 Demo（零配置）

1. 打开 [在线 Demo](https://regulus-academy-web-production.up.railway.app)
2. 创建学习角色（输入昵称）

![角色创建](/screenshots/cloud-profile.png)

3. 在首页输入学习主题，例如「Go 并发」
4. 在知识树中选节点，开始 AI 教练对话
5. 完成练习后节点点亮，可在「知识银河」查看全景

额度用尽时按页面提示填写自己的 LLM Key（BYOK）。详见 [在线体验版](./cloud-demo.md)。

## 方式 B：本地 Docker（完整功能）

```bash
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
```

或手动：

```bash
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy
cp .env.example .env   # 填入 LLM_API_KEY
docker compose -f docker-compose.image.yml up -d
```

访问 `http://localhost:8080`，流程与在线 Demo 相同，另可配置 [IM 频道](./self-host.md#im-频道)。

## 方式 C：源码开发

```bash
cp .env.example .env   # 填入 LLM_API_KEY
pnpm install
pnpm dev               # Go API + Vite 前端
```

文档站本地预览：`pnpm dev:docs`（目录 `apps/docs`）。

## 学习路径提示

| 步骤 | 页面 | 说明 |
|------|------|------|
| 建课 | `#/` 或 `#/import` | 输入领域或导入 PDF/URL |
| 选课 | `#/tree/:id` | 查看节点列表与进度 |
| 学习 | `#/coach/:sessionId` | 讲解、练习、批改 |
| 了解教练 | [教练流程](./coach-flow.md) | 阶段、话术、点亮规则 |
| 全景 | `#/graph` | 多领域知识银河 |
| 导出 | `#/tree/:id` | Skill 包或 Obsidian 笔记 |

更多能力见 [功能一览](./features.md) 与 [界面预览](./screenshots.md)。
