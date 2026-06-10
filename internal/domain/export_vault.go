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

// BuildVaultZip 将学习进度与笔记组装为 Obsidian 兼容的 vault.zip
// 不调用 LLM；无笔记内容的已完成节点生成「知识点占位」笔记，未开始节点生成只含 frontmatter 的占位文件
func BuildVaultZip(in *VaultInput) ([]byte, error) {
	if in == nil || in.Tree == nil {
		return nil, fmt.Errorf("VaultInput 为空")
	}

	domainName := in.Tree.DomainName
	today := time.Now().UTC().Format("2006-01-02")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	dirName := domainName + "/"

	// 每个节点生成一个 .md
	for _, layer := range in.Tree.Layers {
		for _, n := range layer.Nodes {
			prog := in.Progress[n.Key]
			note := in.Notes[n.Key]
			mistakes := in.Mistakes[n.Key]
			spec := in.Nodes[n.Key]

			md := buildNodeMD(n, layer, domainName, prog, note, mistakes, spec, today)
			if err := addBytes(zw, dirName+n.Key+".md", []byte(md)); err != nil {
				return nil, err
			}
		}
	}

	// 生成 _MOC.md
	moc := buildMOC(in.Tree, in.Progress, today)
	if err := addBytes(zw, dirName+"_MOC.md", []byte(moc)); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 vault zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func buildNodeMD(
	n storage.TreeNode,
	layer storage.TreeLayer,
	domainName string,
	prog storage.UserProgress,
	note string,
	mistakes []string,
	spec *NodeSpec,
	today string,
) string {
	status := prog.Status
	if status == "" {
		status = "pending"
	}
	mastery := prog.Mastery

	var sb strings.Builder

	// frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("domain: %q\n", domainName))
	sb.WriteString(fmt.Sprintf("layer: %q\n", layer.Label))
	sb.WriteString(fmt.Sprintf("node: %q\n", n.Key))
	sb.WriteString(fmt.Sprintf("mastery: %.2f\n", mastery))
	sb.WriteString(fmt.Sprintf("status: %q\n", status))
	sb.WriteString(fmt.Sprintf("updated: %s\n", today))
	sb.WriteString("---\n\n")

	// 标题
	sb.WriteString("# " + n.Title + "\n\n")

	if note != "" {
		// 有蒸馏笔记（链路 A 完成后填入），直接使用
		sb.WriteString(note)
		sb.WriteString("\n")
	} else if status == "completed" || mastery > 0 {
		// 无蒸馏笔记但已练习过：用节点 YAML 的 core_concepts 生成结构化占位内容
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

		// wikilink：来自节点 requires
		if len(n.Requires) > 0 {
			sb.WriteString("## 相关节点\n\n")
			for _, req := range n.Requires {
				sb.WriteString("- [[" + req + "]]\n")
			}
			sb.WriteString("\n")
		}
	} else {
		// 未开始或掌握度为 0：极简占位
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

func buildMOC(tree *storage.KnowledgeTree, progress map[string]storage.UserProgress, today string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("domain: %q\n", tree.DomainName))
	sb.WriteString("type: MOC\n")
	sb.WriteString(fmt.Sprintf("updated: %s\n", today))
	sb.WriteString("---\n\n")
	sb.WriteString("# " + tree.DomainName + " — 学习地图\n\n")

	for _, layer := range tree.Layers {
		sb.WriteString("## " + layer.Label + "\n\n")
		for _, n := range layer.Nodes {
			prog := progress[n.Key]
			icon := masteryIcon(prog.Mastery, prog.Status)
			line := fmt.Sprintf("- [[%s]] %s", n.Key, icon)
			if prog.Mastery > 0 {
				line += fmt.Sprintf(" 掌握度 %.0f%%", prog.Mastery*100)
			} else if prog.Status == "" {
				line += " 未开始"
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("掌握度图例：✅ ≥ 80%　🔄 40~79%　⬜ < 40% 或未开始\n")
	return sb.String()
}
