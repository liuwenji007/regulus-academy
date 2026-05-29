package domain

import (
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
