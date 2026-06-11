# 快速上手

1. 打开在线 Demo 或本地 `http://localhost:8080`
2. 创建学习角色（输入昵称）
3. 在首页输入学习主题，例如「Go 并发」
4. 在知识树中选节点，开始 AI 教练对话
5. 完成练习后节点点亮，可在「知识银河」查看全景

本地开发：

```bash
cp .env.example .env   # 填入 LLM_API_KEY
pnpm install
pnpm dev               # Go API + Vite 前端
```
