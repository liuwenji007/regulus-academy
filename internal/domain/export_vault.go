package domain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// VaultInput 组装 vault 所需的所有数据
type VaultInput struct {
	UserID   string
	DomainID string
	Tree     *storage.KnowledgeTree
	Progress map[string]storage.UserProgress // node_key → progress
	Notes    map[string]string               // node_key → content_md（来自 node_notes 表）
	Mistakes map[string][]string             // node_key → []concept
	Nodes    map[string]*NodeSpec            // node_key → NodeSpec（含 core_concepts）
}

// vaultNodeMeta Obsidian 笔记展示元数据（中文标题 + 文件名）
type vaultNodeMeta struct {
	key      string
	title    string
	fileBase string
	module   string
}

// BuildVaultZip 将学习进度与笔记组装为 Obsidian 兼容的 vault.zip
// 不调用 LLM；无笔记内容的已完成节点生成「知识点占位」笔记，未开始节点生成只含 frontmatter 的占位文件
func BuildVaultZip(in *VaultInput) ([]byte, error) {
	if in == nil || in.Tree == nil {
		return nil, fmt.Errorf("VaultInput 为空")
	}

	domainName := in.Tree.DomainName
	today := time.Now().UTC().Format("2006-01-02")
	index := buildVaultNodeIndex(in)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	dirName := domainName + "/"

	for _, layer := range in.Tree.Layers {
		for _, n := range layer.Nodes {
			meta, ok := index[n.Key]
			if !ok {
				continue
			}
			prog := in.Progress[n.Key]
			note := in.Notes[n.Key]
			mistakes := in.Mistakes[n.Key]
			spec := in.Nodes[n.Key]

			md := buildNodeMD(n, layer, domainName, prog, note, mistakes, spec, meta, index, today)
			if err := addBytes(zw, dirName+meta.fileBase+".md", []byte(md)); err != nil {
				return nil, err
			}
		}
	}

	moc := buildMOC(in.Tree, in.Progress, index, today)
	if err := addBytes(zw, dirName+"_MOC.md", []byte(moc)); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 vault zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func resolveVaultNodeTitle(n storage.TreeNode, spec *NodeSpec) string {
	if spec != nil {
		if t := strings.TrimSpace(spec.Node); t != "" {
			return t
		}
	}
	if t := strings.TrimSpace(n.Title); t != "" {
		return t
	}
	return n.Key
}

func sanitizeVaultFileBase(title string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", "\"", "-", "<", "-", ">", "-", "|", "-",
	)
	s := strings.TrimSpace(replacer.Replace(title))
	if s == "" {
		return "note"
	}
	return s
}

func moduleLabelForNode(tree *storage.KnowledgeTree, nodeKey string) string {
	for _, mod := range resolveVaultModules(tree) {
		for _, k := range mod.Nodes {
			if k == nodeKey {
				return mod.Label
			}
		}
	}
	return ""
}

func buildVaultNodeIndex(in *VaultInput) map[string]vaultNodeMeta {
	index := make(map[string]vaultNodeMeta)
	usedFiles := map[string]string{}

	for _, layer := range in.Tree.Layers {
		for _, n := range layer.Nodes {
			spec := in.Nodes[n.Key]
			title := resolveVaultNodeTitle(n, spec)
			fileBase := sanitizeVaultFileBase(title)
			if prevKey, dup := usedFiles[fileBase]; dup && prevKey != n.Key {
				fileBase = sanitizeVaultFileBase(title + " (" + n.Key + ")")
			}
			usedFiles[fileBase] = n.Key
			index[n.Key] = vaultNodeMeta{
				key:      n.Key,
				title:    title,
				fileBase: fileBase,
				module:   moduleLabelForNode(in.Tree, n.Key),
			}
		}
	}
	return index
}

// resolveVaultModules 优先用 tree.modules；无模块时按进度层降级（与前端图谱一致）
func resolveVaultModules(tree *storage.KnowledgeTree) []storage.TreeModule {
	if tree == nil {
		return nil
	}
	if len(tree.Modules) > 0 {
		return tree.Modules
	}
	var derived []storage.TreeModule
	for i, layer := range tree.Layers {
		if len(layer.Nodes) == 0 {
			continue
		}
		nodes := make([]string, len(layer.Nodes))
		for j, n := range layer.Nodes {
			nodes[j] = n.Key
		}
		key := layer.Key
		if key == "" {
			key = fmt.Sprintf("layer-%d", i)
		}
		label := layer.Label
		if label == "" {
			label = key
		}
		derived = append(derived, storage.TreeModule{
			Key: key, Label: label, Goal: layer.Goal, Order: i + 1, Nodes: nodes,
		})
	}
	return derived
}

func vaultWikilink(index map[string]vaultNodeMeta, nodeKey string) string {
	if meta, ok := index[nodeKey]; ok {
		return "[[" + meta.fileBase + "]]"
	}
	return "[[" + nodeKey + "]]"
}

func buildNodeMD(
	n storage.TreeNode,
	layer storage.TreeLayer,
	domainName string,
	prog storage.UserProgress,
	note string,
	mistakes []string,
	spec *NodeSpec,
	meta vaultNodeMeta,
	index map[string]vaultNodeMeta,
	today string,
) string {
	status := prog.Status
	if status == "" {
		status = "pending"
	}
	mastery := prog.Mastery

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("domain: %q\n", domainName))
	if meta.module != "" {
		sb.WriteString(fmt.Sprintf("module: %q\n", meta.module))
	}
	sb.WriteString(fmt.Sprintf("layer: %q\n", layer.Label))
	sb.WriteString(fmt.Sprintf("node: %q\n", n.Key))
	sb.WriteString(fmt.Sprintf("mastery: %.2f\n", mastery))
	sb.WriteString(fmt.Sprintf("status: %q\n", status))
	sb.WriteString(fmt.Sprintf("updated: %s\n", today))
	if meta.fileBase != n.Key {
		sb.WriteString("aliases:\n")
		sb.WriteString(fmt.Sprintf("  - %q\n", n.Key))
	}
	sb.WriteString("---\n\n")

	sb.WriteString("# " + meta.title + "\n\n")

	if note != "" {
		sb.WriteString(note)
		sb.WriteString("\n")
	} else if status == "completed" || mastery > 0 {
		sb.WriteString("## 关键概念\n\n")
		if spec != nil && len(spec.CoreConcepts) > 0 {
			for _, c := range spec.CoreConcepts {
				sb.WriteString("- " + c + "\n")
			}
		} else {
			sb.WriteString("_（笔记待生成）_\n")
		}
		sb.WriteString("\n")

		if len(mistakes) > 0 {
			sb.WriteString("## 踩过的坑\n\n")
			for _, m := range mistakes {
				sb.WriteString("- " + m + "\n")
			}
			sb.WriteString("\n")
		}

		if len(n.Requires) > 0 {
			sb.WriteString("## 相关节点\n\n")
			for _, req := range n.Requires {
				sb.WriteString("- " + vaultWikilink(index, req) + "\n")
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("_尚未学习_\n")
	}

	return sb.String()
}

func masteryIcon(mastery float64, status string) string {
	if status == "completed" && mastery >= 0.8 {
		return "✅"
	}
	if mastery >= 0.4 || status == "completed" {
		return "🔄"
	}
	return "⬜"
}

func buildMOC(tree *storage.KnowledgeTree, progress map[string]storage.UserProgress, index map[string]vaultNodeMeta, today string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("domain: %q\n", tree.DomainName))
	sb.WriteString("type: MOC\n")
	sb.WriteString(fmt.Sprintf("updated: %s\n", today))
	sb.WriteString("---\n\n")
	sb.WriteString("# " + tree.DomainName + " — 学习地图\n\n")

	modules := resolveVaultModules(tree)
	useModuleLayout := len(tree.Modules) > 0

	for _, mod := range modules {
		sb.WriteString("## " + mod.Label + "\n\n")
		if strings.TrimSpace(mod.Goal) != "" {
			sb.WriteString("_" + mod.Goal + "_\n\n")
		}
		for _, nodeKey := range mod.Nodes {
			writeMOCLine(&sb, nodeKey, progress, index)
		}
		sb.WriteString("\n")
	}

	if useModuleLayout {
		sb.WriteString("## 掌握进度\n\n")
		for _, layer := range tree.Layers {
			sb.WriteString("### " + layer.Label + "\n\n")
			for _, n := range layer.Nodes {
				writeMOCLine(&sb, n.Key, progress, index)
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("掌握度图例：✅ ≥ 80%　🔄 40~79%　⬜ < 40% 或未开始\n")
	return sb.String()
}

func writeMOCLine(sb *strings.Builder, nodeKey string, progress map[string]storage.UserProgress, index map[string]vaultNodeMeta) {
	if _, ok := index[nodeKey]; !ok {
		return
	}
	prog := progress[nodeKey]
	icon := masteryIcon(prog.Mastery, prog.Status)
	line := fmt.Sprintf("- %s %s", vaultWikilink(index, nodeKey), icon)
	if prog.Mastery > 0 {
		line += fmt.Sprintf(" 掌握度 %.0f%%", prog.Mastery*100)
	} else if prog.Status == "" {
		line += " 未开始"
	}
	sb.WriteString(line + "\n")
}
