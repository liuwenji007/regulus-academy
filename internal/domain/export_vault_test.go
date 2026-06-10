package domain

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func sampleVaultInput() *VaultInput {
	tree := &storage.KnowledgeTree{
		DomainName: "Go 并发",
		Layers: []storage.TreeLayer{
			{
				Key: "entry", Label: "入门",
				Nodes: []storage.TreeNode{
					{Key: "goroutine_basics", Title: "goroutine 基础", Requires: []string{}},
					{Key: "channel_basics", Title: "channel 基础", Requires: []string{"goroutine_basics"}},
				},
			},
		},
	}
	progress := map[string]storage.UserProgress{
		"goroutine_basics": {NodeKey: "goroutine_basics", Status: "completed", Mastery: 0.85},
		"channel_basics":   {NodeKey: "channel_basics", Status: "completed", Mastery: 0.50},
	}
	nodes := map[string]*NodeSpec{
		"goroutine_basics": {
			Key:          "goroutine_basics",
			CoreConcepts: []string{"go 关键字", "M:N 调度", "goroutine 泄漏"},
		},
	}
	return &VaultInput{
		UserID:   "u1",
		DomainID: "d1",
		Tree:     tree,
		Progress: progress,
		Notes:    map[string]string{},
		Mistakes: map[string][]string{"goroutine_basics": {"用 time.Sleep 做同步"}},
		Nodes:    nodes,
	}
}

func TestBuildVaultZip(t *testing.T) {
	in := sampleVaultInput()
	zipBytes, err := BuildVaultZip(in)
	if err != nil {
		t.Fatalf("BuildVaultZip: %v", err)
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
			t.Fatalf("打开条目 %s 失败: %v", f.Name, err)
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rc)
		_ = rc.Close()
		fileSet[f.Name] = buf.String()
	}

	dir := "Go 并发/"
	required := []string{
		dir + "_MOC.md",
		dir + "goroutine_basics.md",
		dir + "channel_basics.md",
	}
	for _, req := range required {
		if _, ok := fileSet[req]; !ok {
			t.Errorf("vault zip 缺少: %s（已有: %v）", req, zipKeys(fileSet))
		}
	}

	// MOC 应含掌握度标记
	moc := fileSet[dir+"_MOC.md"]
	if !strings.Contains(moc, "goroutine_basics") {
		t.Errorf("_MOC.md 缺少节点链接，内容:\n%s", moc)
	}
	if !strings.Contains(moc, "✅") {
		t.Errorf("_MOC.md 缺少掌握度图例 ✅，内容:\n%s", moc)
	}

	// 已完成节点应含 core_concepts
	gMD := fileSet[dir+"goroutine_basics.md"]
	if !strings.Contains(gMD, "goroutine 泄漏") {
		t.Errorf("goroutine_basics.md 应含 core_concepts，内容:\n%s", gMD)
	}
	if !strings.Contains(gMD, "time.Sleep") {
		t.Errorf("goroutine_basics.md 应含 mistakes，内容:\n%s", gMD)
	}

	// channel_basics 有 requires → wikilink
	cMD := fileSet[dir+"channel_basics.md"]
	if !strings.Contains(cMD, "[[goroutine_basics]]") {
		t.Errorf("channel_basics.md 应含 wikilink，内容:\n%s", cMD)
	}
}

func TestBuildVaultZipNilInput(t *testing.T) {
	_, err := BuildVaultZip(nil)
	if err == nil {
		t.Fatal("nil 输入应返回错误")
	}
}

func TestBuildVaultZipEmptyProgress(t *testing.T) {
	in := sampleVaultInput()
	in.Progress = map[string]storage.UserProgress{}
	zipBytes, err := BuildVaultZip(in)
	if err != nil {
		t.Fatalf("空进度时仍应能导出: %v", err)
	}
	if len(zipBytes) == 0 {
		t.Fatal("zip 为空")
	}
}
