# 自托管部署

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

访问 `http://localhost:8080`（默认端口，见 `.env` 中 `PORT`）

自托管**不要**设置 `REGULUS_DEPLOYMENT=cloud`，行为与开源版一致。
