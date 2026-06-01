package domain

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
	"gopkg.in/yaml.v3"
)

var layerKeyToLabel = map[string]string{
	"entry":        "入门",
	"intermediate": "熟悉",
	"advanced":     "精通",
}

// ExportPackage 可贡献到 regulus-coach/domains/ 的文件包
type ExportPackage struct {
	Slug        string            `json:"slug"`
	DomainName  string            `json:"domainName"`
	Description string            `json:"description"`
	Version     int               `json:"version"`
	Source      string            `json:"source"`
	Files       map[string]string `json:"files"`
}

// ExportToFiles 将知识树与节点边界还原为 tree.yaml + nodes/*.yaml
func ExportToFiles(
	tree *storage.KnowledgeTree,
	slug, description string,
	version int,
	nodes map[string]NodeSpec,
) (map[string]string, error) {
	if tree == nil {
		return nil, fmt.Errorf("知识树为空")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("缺少 slug，无法导出")
	}
	if version <= 0 {
		version = 1
	}

	tf := TreeFile{
		Domain:      tree.DomainName,
		Slug:        slug,
		Version:     version,
		Description: description,
		Layers:      make(map[string]TreeLayerDef, len(tree.Layers)),
	}
	if len(tree.Modules) > 0 {
		tf.Modules = make([]TreeModuleDef, len(tree.Modules))
		for i, m := range tree.Modules {
			tf.Modules[i] = TreeModuleDef{
				Key: m.Key, Label: m.Label, Goal: m.Goal, Order: m.Order,
				Nodes: append([]string(nil), m.Nodes...),
			}
		}
	}
	for _, layer := range tree.Layers {
		def := TreeLayerDef{
			Label: layer.Label,
			Time:  layer.Time,
			Goal:  layer.Goal,
			Nodes: make([]TreeNodeDef, len(layer.Nodes)),
		}
		for i, n := range layer.Nodes {
			def.Nodes[i] = TreeNodeDef{Key: n.Key, Title: n.Title}
		}
		tf.Layers[layer.Key] = def
	}

	treeYAML, err := yaml.Marshal(&tf)
	if err != nil {
		return nil, fmt.Errorf("序列化 tree.yaml 失败: %w", err)
	}

	files := map[string]string{
		"tree.yaml": string(treeYAML),
	}

	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			spec, ok := nodes[n.Key]
			if !ok {
				return nil, fmt.Errorf("缺少节点边界定义: %s", n.Key)
			}
			out := spec
			if out.Key == "" {
				out.Key = n.Key
			}
			if out.Node == "" {
				out.Node = n.Title
			}
			if out.Layer == "" {
				out.Layer = layerKeyToLabel[layer.Key]
				if out.Layer == "" {
					out.Layer = layer.Label
				}
			}
			nodeYAML, err := yaml.Marshal(&out)
			if err != nil {
				return nil, fmt.Errorf("序列化节点 %s 失败: %w", n.Key, err)
			}
			files["nodes/"+n.Key+".yaml"] = string(nodeYAML)
		}
	}
	return files, nil
}

// CollectNodesForExport 从 Skill 包或数据库 nodes_json 收集导出所需节点
func (r *Registry) CollectNodesForExport(
	store *storage.Store,
	tree *storage.KnowledgeTree,
	slug, domainID string,
) (map[string]NodeSpec, error) {
	nodes := make(map[string]NodeSpec)
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			spec, err := r.GetNode(store, domainID, slug, n.Key)
			if err != nil {
				return nil, fmt.Errorf("加载节点 %s 失败: %w", n.Key, err)
			}
			nodes[n.Key] = *spec
		}
	}
	return nodes, nil
}

// ExportDomain 导出用户课程为 Skill 包文件结构
func (r *Registry) ExportDomain(store *storage.Store, userID, domainID string) (*ExportPackage, error) {
	domain, err := store.GetDomain(userID, domainID)
	if err != nil {
		return nil, err
	}

	tree, err := r.ResolveTree(store, userID, domainID)
	if err != nil {
		return nil, err
	}

	slug := domain.Slug
	if slug == "" {
		slug = slugifyExportName(tree.DomainName)
	}

	nodes, err := r.CollectNodesForExport(store, tree, slug, domainID)
	if err != nil {
		return nil, err
	}

	description := ""
	if meta, ok := r.FindDomainBySlug(slug); ok {
		description = meta.Description
	}

	version := 1
	if domain.Source != storage.DomainSourceGenerated {
		version = r.LoadTreeVersion(slug)
		if version <= 0 {
			version = 1
		}
	}

	files, err := ExportToFiles(tree, slug, description, version, nodes)
	if err != nil {
		return nil, err
	}

	return &ExportPackage{
		Slug:        slug,
		DomainName:  tree.DomainName,
		Description: description,
		Version:     version,
		Source:      domain.Source,
		Files:       files,
	}, nil
}

func slugifyExportName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	repl := strings.NewReplacer(
		" ", "-",
		"　", "-",
		"_", "-",
	)
	s = repl.Replace(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "generated-domain"
	}
	return out
}
