#!/usr/bin/env bash
# 下载可选 regulus CLI 到 bin/regulus（与 Web 同状态机）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COACH_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LINK_FILE="$COACH_ROOT/.regulus/link.json"
BIN_DIR="$COACH_ROOT/bin"
OUT="$BIN_DIR/regulus"
GITHUB_REPO="liuwenji007/regulus-academy"

usage() {
  cat >&2 <<'EOF'
用法: bash scripts/install-cli.sh [选项]

选项:
  --url <apiUrl>       自托管 Regulus 根地址（默认读 .regulus/link.json）
  --platform <name>    darwin_arm64 | darwin_amd64 | linux_amd64 | linux_arm64
  --github             从 GitHub Releases 下载（无已部署实例时）
  -h, --help           显示帮助

示例:
  bash scripts/install-cli.sh
  bash scripts/install-cli.sh --url https://你的实例
  bash scripts/install-cli.sh --github
EOF
  exit "${1:-0}"
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin)
      case "$arch" in arm64|aarch64) echo darwin_arm64 ;; *) echo darwin_amd64 ;; esac
      ;;
    linux)
      case "$arch" in aarch64|arm64) echo linux_arm64 ;; *) echo linux_amd64 ;; esac
      ;;
    *)
      echo "错误: 暂不支持平台 $os $arch，请从 GitHub Releases 手动下载" >&2
      exit 1
      ;;
  esac
}

read_api_url() {
  if [[ -n "${REGULUS_API_URL:-}" ]]; then
    echo "${REGULUS_API_URL%/}"
    return 0
  fi
  if [[ -f "$LINK_FILE" ]] && command -v python3 >/dev/null 2>&1; then
    python3 -c "import json; u=json.load(open('$LINK_FILE')).get('apiUrl','').rstrip('/'); print(u)" 2>/dev/null || true
  fi
}

API_URL=""
PLATFORM=""
USE_GITHUB=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url)
      API_URL="${2%/}"
      shift 2
      ;;
    --platform)
      PLATFORM="$2"
      shift 2
      ;;
    --github)
      USE_GITHUB=1
      shift
      ;;
    -h|--help)
      usage 0
      ;;
    *)
      echo "未知选项: $1" >&2
      usage 1
      ;;
  esac
done

if [[ -z "$PLATFORM" ]]; then
  PLATFORM="$(detect_platform)"
fi

if [[ -x "$OUT" ]]; then
  echo "已存在可执行文件 $OUT（跳过下载）。运行 ./bin/regulus doctor 自检。"
  exit 0
fi

mkdir -p "$BIN_DIR"

if [[ $USE_GITHUB -eq 0 && -z "$API_URL" ]]; then
  API_URL="$(read_api_url || true)"
fi

tmp="$(mktemp)"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT

if [[ $USE_GITHUB -eq 1 || -z "$API_URL" ]]; then
  asset="regulus-${PLATFORM}"
  url="https://github.com/${GITHUB_REPO}/releases/latest/download/${asset}"
  echo "从 GitHub Releases 下载 ${asset} …"
  if ! curl -fsSL "$url" -o "$tmp"; then
    echo "下载失败。请打开 https://github.com/${GITHUB_REPO}/releases 手动下载 ${asset}" >&2
    exit 1
  fi
else
  url="${API_URL}/api/coach/cli?platform=${PLATFORM}"
  echo "从 ${API_URL} 下载 CLI（${PLATFORM}）…"
  if ! curl -fsSL "$url" -o "$tmp"; then
    echo "实例下载失败，可重试: bash scripts/install-cli.sh --github" >&2
    exit 1
  fi
fi

mv "$tmp" "$OUT"
chmod +x "$OUT"
trap - EXIT

echo "已安装 $OUT"
"$OUT" doctor 2>/dev/null || echo "提示: 配置 .env 或 data/.env 中的 LLM_API_KEY 后使用 regulus session / build"
