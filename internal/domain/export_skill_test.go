package domain

import (
	"archive/zip"
	"bytes"
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
		"regulus-coach/protocol.md",
		"regulus-coach/schemas/exercise.json",
		"regulus-coach/domains/go-concurrency/tree.yaml",
	}
	for _, req := range required {
		if _, ok := fileSet[req]; !ok {
			t.Errorf("Coach zip 缺少: %s", req)
		}
	}
	for name := range fileSet {
		if strings.Contains(name, "/prompts/") {
			t.Errorf("Coach zip 不应包含 prompts: %s", name)
		}
	}
}

func zipKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
