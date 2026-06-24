package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeCLIPlatform 校验并规范化平台标识（如 darwin_arm64）。
func NormalizeCLIPlatform(platform string) (string, error) {
	p := strings.TrimSpace(strings.ToLower(platform))
	if p == "" {
		p = runtime.GOOS + "_" + runtime.GOARCH
	}
	switch p {
	case "darwin_arm64", "darwin_amd64", "linux_amd64", "linux_arm64", "windows_amd64":
		return p, nil
	default:
		return "", fmt.Errorf("不支持的平台: %s（可选 darwin_arm64 / darwin_amd64 / linux_amd64 / linux_arm64）", platform)
	}
}

// ResolveCLIBinary 查找 regulus CLI 二进制；优先平台命名文件，其次本机已构建的 bin/regulus。
func ResolveCLIBinary(platform string) (path string, content []byte, err error) {
	p, err := NormalizeCLIPlatform(platform)
	if err != nil {
		return "", nil, err
	}
	root := CoachRoot()
	repoRoot := filepath.Dir(root)
	candidates := []string{
		filepath.Join(root, "bin", "regulus-"+p),
		filepath.Join(repoRoot, "bin", "regulus-"+p),
		filepath.Join(root, "dist", "regulus-"+p),
		filepath.Join(repoRoot, "dist", "regulus-"+p),
	}
	host := runtime.GOOS + "_" + runtime.GOARCH
	if p == host {
		candidates = append(candidates,
			filepath.Join(root, "bin", "regulus"),
			filepath.Join(repoRoot, "bin", "regulus"),
		)
	}
	for _, c := range candidates {
		info, statErr := os.Stat(c)
		if statErr != nil || info.IsDir() {
			continue
		}
		b, readErr := os.ReadFile(c)
		if readErr != nil {
			return "", nil, readErr
		}
		return c, b, nil
	}
	return "", nil, fmt.Errorf("未找到平台 %s 的 regulus CLI（请从 GitHub Releases 下载或本地 make cli）", p)
}
