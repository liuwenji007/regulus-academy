#!/usr/bin/env bash
# Regulus Coach Skill — 缺课时自动建课。三档：CLI → 远程 API → 提示 Web。
set -euo pipefail

TOPIC="${1:-}"
if [[ -z "${TOPIC// }" ]]; then
  echo "用法: build-domain.sh \"学习主题\"" >&2
  echo "示例: build-domain.sh \"想学 TypeScript\"" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COACH_ROOT="$SCRIPT_DIR"
DOMAINS_DIR="$COACH_ROOT/domains"
LINK_FILE="$COACH_ROOT/.regulus/link.json"
mkdir -p "$DOMAINS_DIR" "$COACH_ROOT/data"

run_regulus_build() {
  local bin="$1"
  "$bin" build --coach-root "$COACH_ROOT" "$TOPIC"
}

# ── 1. 本地 regulus CLI ─────────────────────────────────────────────
if [[ -x "$COACH_ROOT/bin/regulus" ]]; then
  run_regulus_build "$COACH_ROOT/bin/regulus"
  exit $?
fi

find_repo() {
  if [[ -n "${REGULUS_REPO_ROOT:-}" ]] && [[ -f "${REGULUS_REPO_ROOT}/go.mod" ]] && [[ -d "${REGULUS_REPO_ROOT}/cmd/regulus" ]]; then
    echo "$REGULUS_REPO_ROOT"
    return 0
  fi
  local d="$SCRIPT_DIR"
  while [[ "$d" != "/" ]]; do
    if [[ -f "$d/go.mod" ]] && [[ -d "$d/cmd/regulus" ]]; then
      echo "$d"
      return 0
    fi
    d="$(dirname "$d")"
  done
  return 1
}

if command -v regulus >/dev/null 2>&1; then
  run_regulus_build regulus
  exit $?
fi

if REPO="$(find_repo)"; then
  if [[ -x "$REPO/bin/regulus" ]]; then
    run_regulus_build "$REPO/bin/regulus"
    exit $?
  fi
  if [[ -f "$REPO/cmd/regulus/main.go" ]]; then
    cd "$REPO"
    export REGULUS_COACH_ROOT="$COACH_ROOT"
    go run ./cmd/regulus build --coach-root "$COACH_ROOT" "$TOPIC"
    exit $?
  fi
fi

# ── 2. 远程 API（Linked）────────────────────────────────────────────
run_remote_build() {
  if [[ ! -f "$LINK_FILE" ]] || ! command -v python3 >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
    return 1
  fi
  API_URL="$(python3 -c "import json; print(json.load(open('$LINK_FILE'))['apiUrl'].rstrip('/'))")"
  USER_ID="$(python3 -c "import json; d=json.load(open('$LINK_FILE')); print(d.get('userId') or 'default')")"
  [[ -z "$API_URL" ]] && return 1

  echo "通过远程 API 建课: $API_URL" >&2
  body=$(BUILD_TOPIC="$TOPIC" python3 -c "import json,os; print(json.dumps({'name':os.environ['BUILD_TOPIC'],'force':True}))")
  resp=$(curl -sS -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-User-Id: $USER_ID" \
    -d "$body" \
    "$API_URL/api/domain/build")
  code=$(echo "$resp" | tail -n1)
  json=$(echo "$resp" | sed '$d')
  if [[ "$code" != "202" ]]; then
    echo "远程建课失败 HTTP $code: $json" >&2
    return 1
  fi
  job_id=$(echo "$json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('jobId',''))")
  [[ -z "$job_id" ]] && return 1

  echo "等待建课任务 $job_id …" >&2
  for _ in $(seq 1 120); do
    sleep 2
    job=$(curl -sS -H "X-User-Id: $USER_ID" "$API_URL/api/domain/build/jobs/$job_id")
    status=$(echo "$job" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))")
    if [[ "$status" == "done" ]]; then
      domain_id=$(echo "$job" | python3 -c "
import json,sys
j=json.load(sys.stdin)
r=j.get('result') or {}
t=r.get('tree') or {}
print(t.get('domainId',''))
")
      if [[ -z "$domain_id" ]]; then
        echo "建课完成但无法解析 domainId" >&2
        return 1
      fi
      tmpzip="$(mktemp -t regulus-domain.XXXXXX.zip)"
      curl -sS -H "X-User-Id: $USER_ID" \
        "$API_URL/api/domain/$domain_id/export" -o "$tmpzip"
      tmpdir="$(mktemp -d)"
      unzip -qo "$tmpzip" -d "$tmpdir"
      slug_dir="$(find "$tmpdir" -mindepth 1 -maxdepth 1 -type d | head -1)"
      if [[ -z "$slug_dir" ]]; then
        echo "Domain zip 格式异常" >&2
        rm -rf "$tmpdir" "$tmpzip"
        return 1
      fi
      rm -rf "$DOMAINS_DIR/$(basename "$slug_dir")"
      mv "$slug_dir" "$DOMAINS_DIR/"
      rm -rf "$tmpdir" "$tmpzip"
      echo "已下载 Domain 包到 domains/" >&2
      return 0
    fi
    if [[ "$status" == "failed" ]]; then
      err=$(echo "$job" | python3 -c "import json,sys; print(json.load(sys.stdin).get('error',''))")
      echo "远程建课失败: $err" >&2
      return 1
    fi
  done
  echo "远程建课超时" >&2
  return 1
}

if run_remote_build; then
  exit 0
fi

# ── 3. 提示 Web 手动导出 ────────────────────────────────────────────
cat >&2 <<'EOF'
错误: 无法自动建课。

可选方案：
  1. 安装 regulus CLI（GitHub Releases 或 GET /api/coach/cli?platform=...）并配置 LLM_API_KEY
  2. 配置 .regulus/link.json 后重试（远程建课并拉取 Domain 包）
  3. Web 建课 → 课程详情「导出 Domain 包」→ 解压到 domains/

并在 <coach-root> 或 data/.env 配置 LLM_API_KEY（CLI 模式需要）。
EOF
exit 1
