#!/usr/bin/env bash
# Regulus Academy — 小白一键安装（仅需 Docker）
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
# 或本地：bash scripts/install.sh

set -euo pipefail

REPO_URL="${REGULUS_REPO:-https://github.com/liuwenji007/regulus-academy.git}"
BRANCH="${REGULUS_BRANCH:-main}"
INSTALL_DIR="${REGULUS_INSTALL_DIR:-$HOME/regulus-academy}"
PORT="${REGULUS_PORT:-8080}"

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    red "未找到命令: $1"
    exit 1
  fi
}

need_cmd docker
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  red "需要 Docker Compose（Docker Desktop 已自带）"
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  red "Docker 未运行，请先启动 Docker Desktop"
  exit 1
fi

# 若当前目录已是仓库根目录，直接在此安装
if [[ -f docker-compose.yml && -f Dockerfile && -f .env.example ]]; then
  INSTALL_DIR="$(pwd)"
  yellow "使用当前目录: $INSTALL_DIR"
else
  need_cmd git
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    yellow "更新已有目录: $INSTALL_DIR"
    git -C "$INSTALL_DIR" fetch --depth 1 origin "$BRANCH" 2>/dev/null || true
    git -C "$INSTALL_DIR" checkout "$BRANCH" 2>/dev/null || true
    git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH" 2>/dev/null || \
      git -C "$INSTALL_DIR" reset --hard "origin/$BRANCH" 2>/dev/null || true
  else
    yellow "正在下载到 $INSTALL_DIR …"
    git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
  fi
  cd "$INSTALL_DIR"
fi

mkdir -p data

if [[ ! -f .env ]]; then
  cp .env.example .env
  echo ""
  yellow "请配置 DeepSeek（或其它兼容）API Key。"
  yellow "获取地址: https://platform.deepseek.com/api_keys"
  echo ""
  read -r -p "LLM_API_KEY: " api_key
  if [[ -z "${api_key// }" ]]; then
    red "API Key 不能为空。可稍后编辑 $INSTALL_DIR/.env"
    exit 1
  fi
  # 写入 Key（macOS / Linux 兼容）
  if grep -q '^LLM_API_KEY=' .env; then
    if [[ "$(uname)" == Darwin ]]; then
      sed -i '' "s|^LLM_API_KEY=.*|LLM_API_KEY=$api_key|" .env
    else
      sed -i "s|^LLM_API_KEY=.*|LLM_API_KEY=$api_key|" .env
    fi
  else
    echo "LLM_API_KEY=$api_key" >> .env
  fi
else
  if ! grep -q '^LLM_API_KEY=.\+' .env 2>/dev/null && ! grep -q '^DEEPSEEK_API_KEY=.\+' .env 2>/dev/null; then
    yellow "检测到 .env 存在但未配置 LLM_API_KEY，请编辑: $INSTALL_DIR/.env"
  fi
fi

yellow "正在构建并启动（首次约 3～8 分钟，视网络而定）…"
export PORT
$COMPOSE up --build -d

echo ""
green "✓ Regulus Academy 已启动"
echo ""
echo "  浏览器打开: http://localhost:${PORT}"
echo "  安装目录:   ${INSTALL_DIR}"
echo "  数据目录:   ${INSTALL_DIR}/data"
echo ""
echo "  常用命令:"
echo "    查看日志: cd \"${INSTALL_DIR}\" && $COMPOSE logs -f"
echo "    停止服务: cd \"${INSTALL_DIR}\" && $COMPOSE down"
echo "    修改 Key:  编辑 ${INSTALL_DIR}/.env 后执行 $COMPOSE up -d --build"
echo ""
