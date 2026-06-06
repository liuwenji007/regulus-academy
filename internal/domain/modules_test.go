package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestValidateModulesRejectsProgressLabels(t *testing.T) {
	keys := map[string]struct{}{"a": {}, "b": {}}
	_, err := validateModules([]TreeModuleDef{
		{Key: "entry", Label: "入门", Nodes: []string{"a", "b"}},
	}, keys, ScopeNarrow)
	if err == nil {
		t.Fatal("应拒绝进度层名称作为模块 label")
	}
}

func TestValidateModulesRejectsOutOfBoundsCount(t *testing.T) {
	keys := map[string]struct{}{"a": {}, "b": {}}
	_, err := validateModules([]TreeModuleDef{
		{Key: "m1", Label: "基础", Nodes: []string{"a", "b"}},
	}, keys, ScopeNarrow)
	if err == nil {
		t.Fatal("narrow scope 应拒绝仅 1 个模块")
	}

	mods := []TreeModuleDef{
		{Key: "m1", Label: "基础", Nodes: []string{"a"}},
		{Key: "m2", Label: "进阶", Nodes: []string{"b"}},
	}
	if _, err := validateModules(mods, keys, ScopeNarrow); err != nil {
		t.Fatal(err)
	}

	wideKeys := map[string]struct{}{"a": {}, "b": {}, "c": {}, "d": {}}
	_, err = validateModules([]TreeModuleDef{
		{Key: "m1", Label: "基础", Nodes: []string{"a"}},
		{Key: "m2", Label: "类型", Nodes: []string{"b"}},
		{Key: "m3", Label: "并发", Nodes: []string{"c"}},
		{Key: "m4", Label: "网络", Nodes: []string{"d"}},
	}, wideKeys, ScopeNarrow)
	if err == nil {
		t.Fatal("narrow scope 应拒绝超过 3 个模块")
	}
}

func TestValidateModulesRequiresFullCoverage(t *testing.T) {
	keys := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	_, err := validateModules([]TreeModuleDef{
		{Key: "basics", Label: "基础", Nodes: []string{"a"}},
		{Key: "types", Label: "类型系统", Nodes: []string{"b"}},
		{Key: "more", Label: "进阶", Nodes: []string{"c"}},
	}, keys, ScopeModerate)
	if err != nil {
		t.Fatal(err)
	}

	_, err = validateModules([]TreeModuleDef{
		{Key: "basics", Label: "基础", Nodes: []string{"a"}},
		{Key: "types", Label: "类型系统", Nodes: []string{"b"}},
	}, keys, ScopeModerate)
	if err == nil {
		t.Fatal("应拒绝未覆盖全部节点")
	}
}

func TestFilterModulesForTree(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Modules: []storage.TreeModule{
			{Key: "m1", Label: "基础", Nodes: []string{"a", "b", "c"}},
			{Key: "m2", Label: "并发", Nodes: []string{"d"}},
		},
	}
	selected := map[string]struct{}{"a": {}, "d": {}}
	out := filterModulesForTree(tree, selected)
	if len(out) != 2 {
		t.Fatalf("modules=%d", len(out))
	}
	if len(out[0].Nodes) != 1 || out[0].Nodes[0] != "a" {
		t.Fatalf("m1 nodes=%v", out[0].Nodes)
	}
	if len(out[1].Nodes) != 1 || out[1].Nodes[0] != "d" {
		t.Fatalf("m2 nodes=%v", out[1].Nodes)
	}
}

func TestApplySelectionKeepsModules(t *testing.T) {
	public := &storage.KnowledgeTree{
		DomainName: "Go 并发",
		Layers: []storage.TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []storage.TreeNode{{Key: "a", Title: "A"}, {Key: "b", Title: "B"}}},
			{Key: "advanced", Label: "精通", Nodes: []storage.TreeNode{{Key: "c", Title: "C"}}},
		},
		Modules: []storage.TreeModule{
			{Key: "basics", Label: "基础", Nodes: []string{"a", "b"}},
			{Key: "deep", Label: "底层", Nodes: []string{"c"}},
		},
	}
	sel := &PersonalSelection{
		Selected: []string{"a", "c"},
		Order:    []string{"a", "c"},
	}
	personal := ApplySelection(public, sel)
	if len(personal.Modules) != 2 {
		t.Fatalf("modules=%d", len(personal.Modules))
	}
	if len(personal.Modules[0].Nodes) != 1 || personal.Modules[0].Nodes[0] != "a" {
		t.Fatalf("module0=%v", personal.Modules[0].Nodes)
	}
}

func TestLoadTreeWithModules(t *testing.T) {
	r := NewRegistry()
	tree, err := r.LoadTree("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Modules) != 3 {
		t.Fatalf("modules=%d", len(tree.Modules))
	}
}
