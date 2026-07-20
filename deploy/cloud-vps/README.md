# Cloud Demo：Docker VPS 部署

在任意已安装 Docker / Compose 的 Linux 主机上运行公开 Cloud Demo（预构建镜像，无需在服务器上 `docker build`）。

| 文件 | 用途 |
|------|------|
| [`env.cloud.example`](env.cloud.example) | 环境变量模板 |
| [`docker-compose.yml`](docker-compose.yml) | 拉镜像运行 |
| [`Caddyfile.example`](Caddyfile.example) | HTTPS 反代示例（端口须与 `HOST_PORT` 一致） |

> **注意（开源仓库）**  
> 不要把真实域名、公网 IP、SSH 账号、密钥或 `.env` 写进本仓库。  
> 自动部署凭据只放在 GitHub Actions **Secrets**（或自建 CI）中。

## 1. 准备 `.env`

```bash
sudo mkdir -p /opt/regulus-academy/data
cd /opt/regulus-academy
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/deploy/cloud-vps/docker-compose.yml -o docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/deploy/cloud-vps/env.cloud.example -o .env
```

编辑 `.env`：

| 变量 | 说明 |
|------|------|
| `LLM_API_KEY` / `LLM_PROVIDER` | 平台免费额度所用模型 Key |
| `ADMIN_TOKEN` | `openssl rand -hex 32` |
| `REGULUS_CLOUD_ENCRYPTION_KEY` | 另一次 `openssl rand -hex 32`（BYOK 加密） |
| `REGULUS_DEPLOYMENT=cloud` | 必须保留 |
| `REGULUS_CLOUD_DEMO_URL` | 上线后改为你的 `https://…` 公网地址 |
| `HOST_PORT` | 宿主机端口；与机器上其他服务冲突时再改 |

密钥与 `.env` **不要**提交到 Git。

## 2. 启动

```bash
cd /opt/regulus-academy
docker compose pull
docker compose up -d
curl -fsS "http://127.0.0.1:${HOST_PORT:-8080}/health"
```

GHCR 包为 private 时需先 `docker login ghcr.io`。公开包一般可直接 pull。

## 3. 域名与 HTTPS

1. DNS：A / AAAA 记录指向主机  
2. 放行 **80 / 443**  
3. Caddy 或 Nginx 反代到 `HOST_PORT`（见 [`Caddyfile.example`](Caddyfile.example)）  
4. 更新 `.env` 中 `REGULUS_CLOUD_DEMO_URL` 后执行 `docker compose up -d`

## 4. 可选：GitHub Actions 自动发布

仓库已有「push → 构建并推 GHCR」。若要在镜像发布后自动 SSH 更新主机：

1. 生成**专用**部署密钥（勿把个人登录私钥放进 CI）：

```bash
ssh-keygen -t ed25519 -C "regulus-deploy" -f regulus-deploy -N ""
# 仅将 regulus-deploy.pub 追加到目标机 authorized_keys
```

2. 在仓库 **Settings → Secrets and variables → Actions** 配置（名称固定，值为你的私有信息）：

| Secret | 含义 |
|--------|------|
| `DEPLOY_HOST` | 主机（IP 或域名） |
| `DEPLOY_USER` | SSH 用户 |
| `DEPLOY_SSH_KEY` | 部署私钥全文 |
| `DEPLOY_PATH` | 可选，默认 `/opt/regulus-academy` |
| `DEPLOY_IMAGE` | 可选；默认 `ghcr.io/<本仓库>` |

未配置 `DEPLOY_HOST` 时，[deploy-cloud-vps.yml](../../.github/workflows/deploy-cloud-vps.yml) 会跳过，不影响其他 CI。

3. `Docker Publish` 成功后会部署对应短 SHA 镜像，并对 `/health` 重试约一分钟。

## 5. 同机多服务

- 使用不同 `HOST_PORT` / 反代虚拟主机，避免端口冲突  
- Compose 已限制 `mem_limit: 512m`  
- 公网 Demo 保持 `GATEWAY_ENABLED=false`

## 与 Railway

Railway 仍可作为可选托管方式，见 [`../railway/env.cloud.example`](../railway/env.cloud.example)。  
本目录面向自建 Docker 主机；**默认按空库全新部署**即可。
