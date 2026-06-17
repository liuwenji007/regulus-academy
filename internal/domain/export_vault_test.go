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
		Modules: []storage.TreeModule{
			{
				Key: "goroutine_foundation", Label: "Goroutine 基础", Goal: "理解轻量级线程",
				Nodes: []string{"goroutine_basics", "channel_basics"},
			},
		},
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
			Node:         "goroutine 是什么",
			CoreConcepts: []string{"go 关键字", "M:N 调度", "goroutine 泄漏"},
		},
		"channel_basics": {
			Key:  "channel_basics",
			Node: "channel 通信",
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
		dir + "goroutine 是什么.md",
		dir + "channel 通信.md",
	}
	for _, req := range required {
		if _, ok := fileSet[req]; !ok {
			t.Errorf("vault zip 缺少: %s（已有: %v）", req, zipKeys(fileSet))
		}
	}

	moc := fileSet[dir+"_MOC.md"]
	if !strings.Contains(moc, "Goroutine 基础") {
		t.Errorf("_MOC.md 应按模块划分，内容:\n%s", moc)
	}
	if !strings.Contains(moc, "[[goroutine 是什么]]") {
		t.Errorf("_MOC.md 应使用中文标题 wikilink，内容:\n%s", moc)
	}
	if !strings.Contains(moc, "✅") {
		t.Errorf("_MOC.md 缺少掌握度图例 ✅，内容:\n%s", moc)
	}

	gMD := fileSet[dir+"goroutine 是什么.md"]
	if !strings.Contains(gMD, "# goroutine 是什么") {
		t.Errorf("笔记标题应使用节点中文名，内容:\n%s", gMD)
	}
	if !strings.Contains(gMD, `module: "Goroutine 基础"`) {
		t.Errorf("frontmatter 应含 module，内容:\n%s", gMD)
	}
	if !strings.Contains(gMD, "goroutine 泄漏") {
		t.Errorf("goroutine 是什么.md 应含 core_concepts，内容:\n%s", gMD)
	}
	if !strings.Contains(gMD, "time.Sleep") {
		t.Errorf("goroutine 是什么.md 应含 mistakes，内容:\n%s", gMD)
	}

	cMD := fileSet[dir+"channel 通信.md"]
	if !strings.Contains(cMD, "[[goroutine 是什么]]") {
		t.Errorf("channel 通信.md 应含中文标题 wikilink，内容:\n%s", cMD)
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

func TestResolveVaultNodeTitlePrefersNodeSpec(t *testing.T) {
	title := resolveVaultNodeTitle(
		storage.TreeNode{Key: "k", Title: "树标题"},
		&NodeSpec{Node: "节点中文名"},
	)
	if title != "节点中文名" {
		t.Fatalf("title=%q", title)
	}
}
