// Package coachstatic 将 regulus-coach 嵌入二进制；由 scripts/sync-coach-embed.sh 同步目录。
package coachstatic

import "embed"

// FS 与仓库根目录 regulus-coach/ 内容一致（构建前需 sync）。
//
//go:embed regulus-coach
var FS embed.FS

const RootDir = "regulus-coach"
