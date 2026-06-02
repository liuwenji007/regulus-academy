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

env_has_llm_key() {
  grep -qE '^LLM_API_KEY=.+$' .env 2>/dev/null || grep -qE '^DEEPSEEK_API_KEY=.+$' .env 2>/dev/null
}

write_llm_key() {
  local api_key="$1"
  if grep -q '^LLM_API_KEY=' .env; then
    if [[ "$(uname)" == Darwin ]]; then
      sed -i '' "s|^LLM_API_KEY=.*|LLM_API_KEY=$api_key|" .env
    else
      sed -i "s|^LLM_API_KEY=.*|LLM_API_KEY=$api_key|" .env
    fi
  else
    echo "LLM_API_KEY=$api_key" >> .env
  fi
}

# 引导配置 LLM Key；跳过返回 1，已写入返回 0
prompt_configure_llm_key() {
  echo ""
  yellow "【步骤 2/3】配置模型 API Key"
  echo ""
  echo "  AI 教练（讲解、出题、批改）需要 LLM API Key，推荐 DeepSeek："
  echo "    ① 打开 https://platform.deepseek.com/api_keys"
  echo "    ② 注册 / 登录 → 创建 API Key（一般以 sk- 开头）"
  echo "    ③ 复制 Key，粘贴到下面"
  echo ""
  echo "  也可先跳过：服务会正常启动，但对话与建课暂不可用。"
  echo "  稍后在 ${INSTALL_DIR}/.env 填入 LLM_API_KEY=sk-... 并重启即可。"
  echo ""
  read -r -p "是否现在配置 LLM_API_KEY？[Y/n] " configure_now
  configure_now=${configure_now:-Y}
  if [[ "$configure_now" =~ ^[Nn]$ ]]; then
    yellow "已跳过 Key 配置，继续安装…"
    return 1
  fi
  echo ""
  read -r -p "请粘贴 LLM_API_KEY: " api_key
  api_key="${api_key// /}"
  if [[ -z "$api_key" ]]; then
    yellow "未输入 Key，已跳过。可稍后编辑 ${INSTALL_DIR}/.env"
    return 1
  fi
  write_llm_key "$api_key"
  green "✓ API Key 已写入 .env"
  return 0
}

SKIPPED_LLM_KEY=0

if [[ ! -f .env ]]; then
  cp .env.example .env
  yellow "【步骤 1/3】已创建配置文件 .env"
  if ! prompt_configure_llm_key; then
    SKIPPED_LLM_KEY=1
  fi
elif ! env_has_llm_key; then
  yellow "检测到 .env 存在但未配置 LLM_API_KEY"
  if ! prompt_configure_llm_key; then
    SKIPPED_LLM_KEY=1
  fi
fi

yellow "【步骤 3/3】正在构建并启动（首次约 3～8 分钟，视网络而定）…"
export PORT
$COMPOSE up --build -d

echo ""
green "✓ Regulus Academy 已启动"
echo ""
echo "  浏览器打开: http://localhost:${PORT}"
echo "  安装目录:   ${INSTALL_DIR}"
echo "  数据目录:   ${INSTALL_DIR}/data"
echo ""
if [[ "$SKIPPED_LLM_KEY" -eq 1 ]] || ! env_has_llm_key; then
  yellow "  ⚠ 尚未配置 LLM API Key，AI 教练暂不可用。"
  echo "  后续操作:"
  echo "    1. 编辑 ${INSTALL_DIR}/.env"
  echo "    2. 填入 LLM_API_KEY=sk-...（获取: https://platform.deepseek.com/api_keys）"
  echo "    3. 重启: cd \"${INSTALL_DIR}\" && $COMPOSE up -d --build"
  echo "  打开 Web 后，侧栏会显示「LLM 未配置」直到配置完成。"
  echo ""
fi
echo "  常用命令:"
echo "    查看日志: cd \"${INSTALL_DIR}\" && $COMPOSE logs -f"
echo "    停止服务: cd \"${INSTALL_DIR}\" && $COMPOSE down"
echo "    修改 Key:  编辑 ${INSTALL_DIR}/.env 后执行 $COMPOSE up -d --build"
echo ""
