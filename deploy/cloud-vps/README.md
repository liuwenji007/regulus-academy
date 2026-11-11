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

## 4. 可选：打 tag 自动发体验版

流水线：

```text
git push origin vX.Y.Z
  → Docker Publish（推 GHCR :vX.Y.Z + :latest）
  → Deploy Cloud VPS（SSH pull + up + /health）
```

手动 **Docker Publish** 只推镜像、不部署。回滚或重发可在 Actions 里跑 **Deploy Cloud VPS**（指定 `image_tag`）。

1. 生成**专用**部署密钥（勿把个人登录私钥放进 CI）：

```bash
ssh-keygen -t ed25519 -C "regulus-deploy" -f regulus-deploy -N ""
# 仅将 regulus-deploy.pub 追加到目标机 authorized_keys
```

2. 在仓库 **Settings → Secrets and variables → Actions** 配置：

| 名称 | 类型 | 含义 |
|------|------|------|
| `DEPLOY_HOST` | Secret | 主机（IP 或域名） |
| `DEPLOY_USER` | Secret | SSH 用户 |
| `DEPLOY_SSH_KEY` | Secret | 部署私钥全文 |
| `DEPLOY_PATH` | Secret | 可选，默认 `/opt/regulus-academy` |
| `DEPLOY_IMAGE` | Secret | 可选；默认 `ghcr.io/<本仓库>`（不要带 tag） |
| `DEPLOY_SSH_HOST_FINGERPRINT` | Secret | 可选；主机公钥 SHA256 fingerprint（`ssh-keyscan` 后用 `ssh-keygen -lf`）。不配则忽略 host key 校验 |
| `DEPLOY_GHCR_TOKEN` | Secret | 可选；GHCR 只读 PAT。**当前包对匿名 pull 会 401 时必配**，否则远程 `docker compose pull` 失败 |
| `DEPLOY_GHCR_USER` | Secret | 可选；GHCR 登录用户，默认仓库 owner |
| `DEPLOY_CLOUD_VPS` | Variable | 设为 `true` 时：缺 `DEPLOY_*` 会导致 deploy job **失败**（主仓建议开；fork 保持未设以便跳过） |

未配置 `DEPLOY_HOST` 且未开 `DEPLOY_CLOUD_VPS` 时，[deploy-cloud-vps.yml](../../.github/workflows/deploy-cloud-vps.yml) 会跳过，不影响镜像发布。看 Job Summary：应是 `Deploying …` / health ok，而不是 `skipped`。

获取 host fingerprint 示例：

```bash
ssh-keyscan -t ed25519 YOUR_HOST 2>/dev/null | ssh-keygen -lf - -E sha256
# 把输出里 SHA256: 开头的整段写入 DEPLOY_SSH_HOST_FINGERPRINT
```

3. 主机上先按上文放好 `docker-compose.yml`、`.env`、`data/`，确认能手动 `docker compose pull && up -d`（私有 GHCR 须先 `docker login ghcr.io`，或改配 `DEPLOY_GHCR_TOKEN` 让 CI 登录）。

4. **Tag 必须打在已包含本 workflow 的 commit 上**（先合入/推送再 `git tag`）。推送 tag 后到 Actions 看 **Docker Publish** → **deploy-demo**；以 Summary 与日志里的 `health ok` 为准。

## 5. 同机多服务

- 使用不同 `HOST_PORT` / 反代虚拟主机，避免端口冲突  
- Compose 已限制 `mem_limit: 512m`  
- 公网 Demo 保持 `GATEWAY_ENABLED=false`

## 与 Railway

Railway 仍可作为可选托管方式，见 [`../railway/env.cloud.example`](../railway/env.cloud.example)。  
本目录面向自建 Docker 主机；**默认按空库全新部署**即可。
