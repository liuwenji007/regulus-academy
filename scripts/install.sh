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
SKIP_GIT_UPDATE="${REGULUS_SKIP_UPDATE:-0}"

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

is_project_root() {
  local dir="${1:-.}"
  [[ -f "$dir/docker-compose.yml" && -f "$dir/Dockerfile" && -f "$dir/.env.example" ]]
}

# 尝试 fast-forward 更新；任何失败都不阻断安装
try_git_update() {
  local dir="$1"
  [[ -d "$dir/.git" ]] || return 0
  [[ "${REGULUS_SKIP_GIT_UPDATE:-}" == "1" ]] && return 0

  yellow "检查代码更新（失败不影响安装，可设 REGULUS_SKIP_GIT_UPDATE=1 跳过）…"
  export GIT_TERMINAL_PROMPT=0
  if ! git -C "$dir" fetch --depth 1 origin "$BRANCH" 2>/dev/null; then
    yellow "无法连接远程，将使用本地已有版本继续"
    return 0
  fi
  git -C "$dir" checkout "$BRANCH" 2>/dev/null || true
  if git -C "$dir" merge --ff-only "origin/$BRANCH" 2>/dev/null || \
     git -C "$dir" pull --ff-only origin "$BRANCH" 2>/dev/null; then
    green "✓ 已更新到最新代码"
  else
    yellow "本地有改动或与远程不一致，跳过更新（可稍后在该目录手动 git pull）"
  fi
}

# 解析安装目录：优先当前目录 → 已有安装目录 → clone
if is_project_root "."; then
  INSTALL_DIR="$(pwd)"
  yellow "使用当前目录: $INSTALL_DIR"
elif is_project_root "$INSTALL_DIR"; then
  yellow "使用已有安装目录: $INSTALL_DIR"
  cd "$INSTALL_DIR"
  try_git_update "$INSTALL_DIR"
else
  need_cmd git
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    yellow "使用已有目录: $INSTALL_DIR"
    cd "$INSTALL_DIR"
    if ! is_project_root "."; then
      red "目录 $INSTALL_DIR 不是完整的 Regulus Academy 项目（缺少 docker-compose.yml 等）"
      echo "  请删除该目录后重试，或在已 clone 的项目根目录运行: bash scripts/install.sh"
      exit 1
    fi
    try_git_update "$INSTALL_DIR"
  elif [[ -d "$INSTALL_DIR" ]] && [[ -n "$(ls -A "$INSTALL_DIR" 2>/dev/null || true)" ]]; then
    red "目录 $INSTALL_DIR 已存在且不是 Git 仓库"
    echo "  · 若已手动 clone：进入项目根目录执行 bash scripts/install.sh"
    echo "  · 或设置 REGULUS_INSTALL_DIR 指向其他空目录后重试"
    exit 1
  else
    yellow "正在下载到 $INSTALL_DIR …"
    GIT_TERMINAL_PROMPT=0 git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
  fi
fi

mkdir -p data

env_llm_key_value() {
  local line val
  line=$(grep -E '^LLM_API_KEY=' .env 2>/dev/null | tail -1 || true)
  if [[ -n "$line" ]]; then
    val="${line#LLM_API_KEY=}"
    val="${val// /}"
    val="${val//\"/}"
    val="${val//\'/}"
    val="${val//$'\r'/}"
    if [[ -n "$val" ]]; then
      printf '%s' "$val"
      return 0
    fi
  fi
  line=$(grep -E '^DEEPSEEK_API_KEY=' .env 2>/dev/null | tail -1 || true)
  if [[ -n "$line" ]]; then
    val="${line#DEEPSEEK_API_KEY=}"
    val="${val// /}"
    val="${val//\"/}"
    val="${val//\'/}"
    val="${val//$'\r'/}"
    if [[ -n "$val" ]]; then
      printf '%s' "$val"
      return 0
    fi
  fi
  return 1
}

env_has_llm_key() {
  env_llm_key_value >/dev/null 2>&1
}

can_prompt_tty() {
  [[ -r /dev/tty ]]
}

# 从终端读取（curl | bash 时 stdin 被脚本占用，必须读 /dev/tty）
read_tty() {
  local prompt="$1"
  local __var="$2"
  if ! can_prompt_tty; then
    return 1
  fi
  printf '%s' "$prompt" > /dev/tty
  # shellcheck disable=SC2162
  IFS= read -r "$__var" < /dev/tty
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

  if ! can_prompt_tty; then
    yellow "当前环境无法交互输入，已跳过 Key 配置，继续安装…"
    echo "  请安装完成后编辑 ${INSTALL_DIR}/.env 填入 LLM_API_KEY=sk-..."
    return 1
  fi

  local configure_now="" api_key=""
  if ! read_tty $'是否现在配置 LLM_API_KEY？[Y/n] ' configure_now; then
    yellow "无法读取输入，已跳过 Key 配置"
    return 1
  fi
  configure_now=${configure_now:-Y}
  if [[ "$configure_now" =~ ^[Nn]$ ]]; then
    yellow "已跳过 Key 配置，继续安装…"
    return 1
  fi

  printf '\n' > /dev/tty
  if ! read_tty "请粘贴 LLM_API_KEY: " api_key; then
    yellow "无法读取输入，已跳过 Key 配置"
    return 1
  fi
  api_key="${api_key// /}"
  api_key="${api_key//$'\r'/}"
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
