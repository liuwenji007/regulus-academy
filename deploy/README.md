# 部署指南

**用户文档**：https://regulus-academy-docs.vercel.app

## 自托管（默认）

| 方式 | 说明 |
|------|------|
| 一键安装 | `curl -fsSL .../scripts/install.sh \| bash` |
| Docker | `docker compose -f docker-compose.image.yml up -d` |
| 源码 | `go run ./cmd/server` + `cd web && pnpm dev` |

无需设置 `REGULUS_DEPLOYMENT`，行为与开源版一致。

## Cloud Demo（Railway）

1. 在 [Railway](https://railway.com) 新建 Project → Deploy from GitHub
2. Builder 选 **Dockerfile**（根目录）
3. 添加 **Volume**，挂载路径 `/app/data`
4. 从 [`railway/env.cloud.example`](railway/env.cloud.example) 复制变量到 Railway Variables
5. 生成并填入 `ADMIN_TOKEN`、`REGULUS_CLOUD_ENCRYPTION_KEY`（各 `openssl rand -hex 32`）
6. 部署完成后访问 Railway 分配的 HTTPS 域名

健康检查：`GET /health`

### 常见部署错误

| 报错 | 处理 |
|------|------|
| `docker VOLUME ... is not supported` | Dockerfile 勿写 `VOLUME`；在画布 **+ Add → Volume** 挂 `/app/data` |
| `The executable pnpm could not be found` | 根目录须有 `railway.toml`（`startCommand = "/app/server"`）；或在 Settings → Deploy 清空自定义启动命令 |
| 域名一直 pending | 先等部署 Success，再 **Networking → Generate Domain**；Variables 勿手写 `PORT` |

## 使用文档（Vercel）

文档站位于 `apps/docs`（VitePress）。

1. 在 [Vercel](https://vercel.com) 导入同一 GitHub 仓库
2. **Root Directory** 设为 `apps/docs`
3. Framework Preset：VitePress（或留空，使用 `vercel.json`）
4. 将产出 URL 写入 Railway 的 `REGULUS_CLOUD_DOCS_URL`

## Monorepo 脚本（根目录）

```bash
pnpm install          # 安装 web + docs 依赖
pnpm dev              # 并行启动 Go API + Vite 前端
pnpm build            # 构建前端 + Go 二进制
pnpm build:docs       # 构建文档站
make test             # Go + 前端测试
```
