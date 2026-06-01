package domain

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DomainMeta 可用知识域摘要
type DomainMeta struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PublicDomainEntry 公共知识库目录项（含版本与规模）
type PublicDomainEntry struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	NodeCount   int    `json:"nodeCount"`
}

// ListDomains 扫描 regulus-coach/domains/ 下已有 Skill 包
func (r *Registry) ListDomains() ([]DomainMeta, error) {
	dir := filepath.Join(r.root, "domains")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var list []DomainMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		path := filepath.Join(dir, slug, "tree.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var tf TreeFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}
		name := tf.Domain
		if name == "" {
			name = slug
		}
		desc := tf.Description
		if tf.Slug != "" {
			slug = tf.Slug
		}
		list = append(list, DomainMeta{Slug: slug, Name: name, Description: desc})
	}
	return list, nil
}

// ListPublicDomains 扫描公共 Skill 包并附带版本、节点数
func (r *Registry) ListPublicDomains() ([]PublicDomainEntry, error) {
	metaList, err := r.ListDomains()
	if err != nil {
		return nil, err
	}
	out := make([]PublicDomainEntry, 0, len(metaList))
	for _, m := range metaList {
		tree, err := r.LoadTree(m.Slug)
		if err != nil {
			continue
		}
		count := 0
		for _, layer := range tree.Layers {
			count += len(layer.Nodes)
		}
		out = append(out, PublicDomainEntry{
			Slug:        m.Slug,
			Name:        m.Name,
			Description: m.Description,
			Version:     r.LoadTreeVersion(m.Slug),
			NodeCount:   count,
		})
	}
	return out, nil
}

// FindDomainBySlug 在目录列表中查找 slug
func (r *Registry) FindDomainBySlug(slug string) (DomainMeta, bool) {
	list, err := r.ListDomains()
	if err != nil {
		return DomainMeta{}, false
	}
	for _, d := range list {
		if d.Slug == slug {
			return d, true
		}
	}
	return DomainMeta{}, false
}

// readTreeFileBySlug 按 slug 读取 tree.yaml（目录名与 slug 可不同）
func (r *Registry) readTreeFileBySlug(slug string) (TreeFile, error) {
	dir := filepath.Join(r.root, "domains")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return TreeFile{}, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "tree.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var tf TreeFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}
		dirSlug := e.Name()
		if tf.Slug != "" {
			dirSlug = tf.Slug
		}
		if dirSlug == slug {
			return tf, nil
		}
	}
	// 回退：目录名即 slug
	path := filepath.Join(dir, slug, "tree.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return TreeFile{}, fmt.Errorf("未找到 slug=%s 的知识包", slug)
	}
	var tf TreeFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return TreeFile{}, err
	}
	return tf, nil
}
