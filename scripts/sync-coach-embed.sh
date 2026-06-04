#!/usr/bin/env bash
# 将 regulus-coach 同步到 internal/coachstatic/，供 go:embed 打入二进制（Docker 旧镜像回退）。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
src="$ROOT/regulus-coach"
dest="$ROOT/internal/coachstatic/regulus-coach"
rm -rf "$dest"
cp -a "$src" "$dest"
echo "synced: $dest"
