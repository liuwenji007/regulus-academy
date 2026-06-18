package domain

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestBuildDomainZip(t *testing.T) {
	tree := &storage.KnowledgeTree{
		DomainName: "Go 并发",
		Layers: []storage.TreeLayer{
			{
				Key: "entry", Label: "入门",
				Nodes: []storage.TreeNode{
					{Key: "goroutine_basics", Title: "goroutine 基础"},
				},
			},
		},
	}
	nodes := map[string]NodeSpec{
		"goroutine_basics": {
			Key:          "goroutine_basics",
			Node:         "goroutine 基础",
			CoreConcepts: []string{"go 关键字", "M:N 调度"},
		},
	}
	files, err := ExportToFiles(tree, "go-concurrency", "测试描述", "go", 1, nodes)
	if err != nil {
		t.Fatalf("ExportToFiles: %v", err)
	}

	pkg := &ExportPackage{
		Slug:        "go-concurrency",
		DomainName:  "Go 并发",
		Description: "测试描述",
		ParentSlug:  "go",
		Version:     1,
		Source:      "generated",
		Files:       files,
	}

	zipBytes, err := BuildDomainZip(pkg)
	if err != nil {
		t.Fatalf("BuildDomainZip: %v", err)
	}
	if len(zipBytes) == 0 {
		t.Fatal("zip 内容为空")
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}

	fileSet := make(map[string]string)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("打开 zip 条目 %s 失败: %v", f.Name, err)
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rc)
		_ = rc.Close()
		fileSet[f.Name] = buf.String()
	}

	root := "go-concurrency/"
	requiredFiles := []string{
		root + "README.md",
		root + "tree.yaml",
		root + "nodes/goroutine_basics.yaml",
	}
	for _, req := range requiredFiles {
		if _, ok := fileSet[req]; !ok {
			t.Errorf("zip 缺少条目: %s（已有: %v）", req, zipKeys(fileSet))
		}
	}

	readme := fileSet[root+"README.md"]
	if !strings.Contains(readme, "Go 并发") {
		t.Errorf("README 缺少领域名，内容:\n%s", readme)
	}

	treeYAML := fileSet[root+"tree.yaml"]
	if !strings.Contains(treeYAML, "parent_slug: go") {
		t.Errorf("tree.yaml 缺少 parent_slug，内容:\n%s", treeYAML)
	}
}

func TestBuildDomainZipNilPkg(t *testing.T) {
	_, err := BuildDomainZip(nil)
	if err == nil {
		t.Fatal("nil pkg 应返回错误")
	}
}

func TestBuildCoachSkillZip(t *testing.T) {
	chdirCoachRoot(t)
	zipBytes, err := BuildCoachSkillZip()
	if err != nil {
		t.Fatalf("BuildCoachSkillZip: %v", err)
	}
	if len(zipBytes) == 0 {
		t.Fatal("zip 内容为空")
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}

	fileSet := make(map[string]struct{})
	for _, f := range zr.File {
		fileSet[f.Name] = struct{}{}
	}

	required := []string{
		"regulus-coach/SKILL.md",
		"regulus-coach/protocol-lite.md",
		"regulus-coach/agent-prompts.md",
		"regulus-coach/USAGE.md",
		"regulus-coach/build-domain.sh",
		"regulus-coach/scripts/api-session.sh",
		"regulus-coach/schemas/exercise.json",
		"regulus-coach/schemas/progress.schema.json",
		"regulus-coach/data/progress.json",
		"regulus-coach/.regulus/link.json.example",
		"regulus-coach/domains/go-concurrency/tree.yaml",
	}
	for _, req := range required {
		if _, ok := fileSet[req]; !ok {
			t.Errorf("Coach zip 缺少: %s", req)
		}
	}
	if _, hasBin := fileSet["regulus-coach/bin/regulus"]; hasBin {
		t.Error("默认 Coach zip 不应包含 bin/regulus")
	}
	for name := range fileSet {
		if strings.Contains(name, "/prompts/") {
			t.Errorf("Coach zip 不应包含 prompts: %s", name)
		}
	}
}

func TestBuildCoachSkillZipWithBinaryMissingBinary(t *testing.T) {
	chdirCoachRoot(t)
	root := CoachRoot()
	type hiddenBin struct{ from, to string }
	var hidden []hiddenBin
	for _, p := range []string{
		filepath.Join(root, "bin", "regulus"),
		filepath.Join(filepath.Dir(root), "bin", "regulus"),
	} {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		stash := p + ".test-hidden"
		if err := os.Rename(p, stash); err != nil {
			t.Fatal(err)
		}
		hidden = append(hidden, hiddenBin{from: stash, to: p})
	}
	t.Cleanup(func() {
		for _, h := range hidden {
			_ = os.Rename(h.from, h.to)
		}
	})
	if len(hidden) == 0 {
		t.Skip("无 bin/regulus 可隐藏")
	}

	_, err := BuildCoachSkillZipWithBinary()
	if err == nil {
		t.Fatal("缺少 bin/regulus 时应返回错误")
	}
	if !strings.Contains(err.Error(), "bin/regulus") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildCoachSkillZipWithBinary(t *testing.T) {
	chdirCoachRoot(t)
	root := CoachRoot()
	binPath := filepath.Join(root, "bin", "regulus")
	if _, err := os.Stat(binPath); err != nil {
		repoBin := filepath.Join(filepath.Dir(root), "bin", "regulus")
		if _, err2 := os.Stat(repoBin); err2 != nil {
			t.Skip("跳过：未找到 bin/regulus，请先 make cli")
		}
	}
	zipBytes, err := BuildCoachSkillZipWithBinary()
	if err != nil {
		t.Fatalf("BuildCoachSkillZipWithBinary: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "regulus-coach/bin/regulus" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("WithBinary zip 应包含 bin/regulus")
	}
}

func zipKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
