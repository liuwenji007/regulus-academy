package cliruntime

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
)

// Paths Skill 运行时目录布局。
type Paths struct {
	CoachRoot  string
	DataDir    string
	DBPath     string
	RegulusDir string
	ConfigPath string
	LinkPath   string
	DomainsDir string
	BinPath    string
}

// ResolvePaths 解析 coach 根目录与数据路径。
func ResolvePaths(coachRootFlag string) (Paths, error) {
	root := strings.TrimSpace(coachRootFlag)
	if root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return Paths{}, err
		}
		root = abs
		_ = os.Setenv("REGULUS_COACH_ROOT", root)
	} else {
		root = domain.CoachRoot()
	}
	dataDir := os.Getenv("REGULUS_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(root, "data")
	}
	regulusDir := filepath.Join(root, ".regulus")
	return Paths{
		CoachRoot:  root,
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "regulus.db"),
		RegulusDir: regulusDir,
		ConfigPath: filepath.Join(regulusDir, "config.json"),
		LinkPath:   filepath.Join(regulusDir, "link.json"),
		DomainsDir: filepath.Join(root, "domains"),
		BinPath:    filepath.Join(root, "bin", "regulus"),
	}, nil
}
