#!/usr/bin/env bash
# Regulus Coach — Linked 模式：通过 HTTP API 开课与会话（无需 bin/regulus）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COACH_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LINK_FILE="$COACH_ROOT/.regulus/link.json"

usage() {
  cat >&2 <<'EOF'
用法:
  scripts/api-session.sh start --slug <slug> [--node <key>] [--layer entry]
  scripts/api-session.sh message --session <id> "用户原话"

需配置 .regulus/link.json（可复制 .regulus/link.json.example）：
  { "apiUrl": "https://...", "userId": "default" }
EOF
  exit 1
}

load_link() {
  if [[ ! -f "$LINK_FILE" ]]; then
    echo "错误: 未找到 $LINK_FILE，请先关联 Regulus 实例" >&2
    echo "  cp .regulus/link.json.example .regulus/link.json 并编辑 apiUrl" >&2
    exit 1
  fi
  if ! command -v python3 >/dev/null 2>&1; then
    echo "错误: 需要 python3 读取 link.json" >&2
    exit 1
  fi
  API_URL="$(python3 -c "import json; print(json.load(open('$LINK_FILE'))['apiUrl'].rstrip('/'))")"
  USER_ID="$(python3 -c "import json; d=json.load(open('$LINK_FILE')); print(d.get('userId') or 'default')")"
  if [[ -z "$API_URL" ]]; then
    echo "错误: link.json 缺少 apiUrl" >&2
    exit 1
  fi
}

api_json() {
  local method="$1" path="$2" body="${3:-}"
  local tmp
  tmp="$(mktemp)"
  local code
  if [[ -n "$body" ]]; then
    code=$(curl -sS -o "$tmp" -w "%{http_code}" -X "$method" \
      -H "Content-Type: application/json" \
      -H "X-User-Id: $USER_ID" \
      -d "$body" \
      "$API_URL$path")
  else
    code=$(curl -sS -o "$tmp" -w "%{http_code}" -X "$method" \
      -H "X-User-Id: $USER_ID" \
      "$API_URL$path")
  fi
  if [[ "$code" -ge 400 ]]; then
    echo "HTTP $code: $(cat "$tmp")" >&2
    rm -f "$tmp"
    exit 1
  fi
  cat "$tmp"
  rm -f "$tmp"
}

resolve_domain_id() {
  local slug="$1"
  api_json GET "/api/domains" | python3 -c "
import json, sys
slug = sys.argv[1]
data = json.load(sys.stdin)
for d in data.get('domains', []):
    if d.get('slug') == slug:
        print(d['id'])
        break
" "$slug"
}

pick_first_node() {
  local slug="$1" layer="${2:-entry}"
  local tree="$COACH_ROOT/domains/$slug/tree.yaml"
  if [[ ! -f "$tree" ]]; then
    echo "错误: 本地无 domains/$slug/tree.yaml" >&2
    exit 1
  fi
  python3 -c "
import re, sys
text = open(sys.argv[1]).read()
layer = sys.argv[2]
# 匹配 layers: 下 entry: ... nodes: - key: xxx
pat = rf'{re.escape(layer)}:.*?nodes:\s*(?:\n\s*- key:\s*(\S+))'
m = re.search(pat, text, re.S)
if m:
    print(m.group(1))
else:
    # 回退：modules 第一个节点
    m2 = re.search(r'nodes:\s*\n\s*-\s*(\S+)', text)
    print(m2.group(1) if m2 else '')
" "$tree" "$layer"
}

cmd_start() {
  local slug="" node="" layer="entry"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --slug) slug="$2"; shift 2 ;;
      --node) node="$2"; shift 2 ;;
      --layer) layer="$2"; shift 2 ;;
      *) echo "未知参数: $1" >&2; usage ;;
    esac
  done
  [[ -z "$slug" ]] && usage
  load_link
  if [[ -z "$node" ]]; then
    node="$(pick_first_node "$slug" "$layer")"
  fi
  if [[ -z "$node" ]]; then
    echo "错误: 无法解析首个节点，请传 --node" >&2
    exit 1
  fi
  domain_id="$(resolve_domain_id "$slug")"
  if [[ -z "$domain_id" ]]; then
    echo "错误: 远程无课程 slug=$slug，请先在 Web 建课或 build-domain.sh" >&2
    exit 1
  fi
  body=$(python3 -c "import json; print(json.dumps({'domainId':'$domain_id','nodeKey':'$node','layer':'$layer'}))")
  api_json POST "/api/session/start" "$body"
}

cmd_message() {
  local session="" text=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --session) session="$2"; shift 2 ;;
      *) text="$*"; break ;;
    esac
  done
  [[ -z "$session" || -z "$text" ]] && usage
  load_link
  body=$(python3 -c "import json,sys; print(json.dumps({'sessionId':sys.argv[1],'content':sys.argv[2]}))" "$session" "$text")
  api_json POST "/api/session/message" "$body"
}

[[ $# -lt 1 ]] && usage
sub="$1"; shift
case "$sub" in
  start) cmd_start "$@" ;;
  message) cmd_message "$@" ;;
  *) usage ;;
esac
