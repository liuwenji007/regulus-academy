package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
	"gopkg.in/yaml.v3"
)

// Registry 知识领域注册表
type Registry struct {
	root string
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{root: CoachRoot()}
}

// MatchDomain 匹配用户输入的领域名
func (r *Registry) MatchDomain(input string) (slug string, ok bool) {
	n := strings.ToLower(strings.TrimSpace(input))
	aliases := map[string]string{
		"go 并发":          "go-concurrency",
		"go并发":           "go-concurrency",
		"golang 并发":      "go-concurrency",
		"go concurrency": "go-concurrency",
		"goroutine":      "go-concurrency",
		"并发":             "go-concurrency",
	}
	if slug, ok = aliases[n]; ok {
		return slug, true
	}
	if strings.Contains(n, "go") && strings.Contains(n, "并发") {
		return "go-concurrency", true
	}
	return "", false
}

// LoadTree 加载领域知识树
func (r *Registry) LoadTree(slug string) (*storage.KnowledgeTree, error) {
	path := filepath.Join(r.root, "domains", slug, "tree.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载知识树失败: %w", err)
	}
	var tf TreeFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}
	order := []struct {
		key string
	}{
		{"entry"}, {"intermediate"}, {"advanced"},
	}
	tree := &storage.KnowledgeTree{
		DomainName: tf.Domain,
		Layers:     make([]storage.TreeLayer, 0, 3),
	}
	for _, o := range order {
		layer, ok := tf.Layers[o.key]
		if !ok {
			continue
		}
		nodes := make([]storage.TreeNode, len(layer.Nodes))
		for i, n := range layer.Nodes {
			nodes[i] = storage.TreeNode{Key: n.Key, Title: n.Title}
		}
		tree.Layers = append(tree.Layers, storage.TreeLayer{
			Key:   o.key,
			Label: layer.Label,
			Time:  layer.Time,
			Goal:  layer.Goal,
			Nodes: nodes,
		})
	}
	return tree, nil
}

// LoadNode 加载节点边界
func (r *Registry) LoadNode(slug, nodeKey string) (*NodeSpec, error) {
	path := filepath.Join(r.root, "domains", slug, "nodes", nodeKey+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载节点 %s 失败: %w", nodeKey, err)
	}
	var spec NodeSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// LoadProtocol 加载 Learning Protocol
func LoadProtocol() (string, error) {
	b, err := ReadCoachFile("protocol.md")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// LoadSchema 加载 JSON schema 文本
func LoadSchema(name string) (string, error) {
	b, err := ReadCoachFile(filepath.Join("schemas", name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// NodeTitle 从树中查找节点标题
func NodeTitle(tree *storage.KnowledgeTree, nodeKey string) string {
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			if n.Key == nodeKey {
				return n.Title
			}
		}
	}
	return nodeKey
}

// LayerForNode 返回节点所在层 key
func LayerForNode(tree *storage.KnowledgeTree, nodeKey string) string {
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			if n.Key == nodeKey {
				return layer.Key
			}
		}
	}
	return "entry"
}

// TreeToJSON 序列化知识树
func TreeToJSON(tree *storage.KnowledgeTree) (string, error) {
	b, err := json.Marshal(tree)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
