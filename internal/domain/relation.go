package domain

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const (
	RelationExistingSubtopic = "existing_subtopic" // 已有窄主题，用户想建宽主题
	RelationNewIsSubtopic    = "new_is_subtopic"   // 新主题是已有宽主题的子话题
)

// DomainRelation 与已有课程的主题层级关系
type DomainRelation struct {
	Kind            string                `json:"kind"`
	Message         string                `json:"message"`
	ExistingDomain  storage.DomainSummary `json:"existingDomain"`
	ExistingTree    *storage.KnowledgeTree `json:"-"`
}

// ParentSlug 读取 Skill 包的 parent_slug
func (r *Registry) ParentSlug(slug string) string {
	tf, err := r.readTreeFileBySlug(slug)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(tf.ParentSlug)
}

// TopicRoot 归一化到主题根（如 go-language → go）
func TopicRoot(slug string) string {
	s := strings.ToLower(strings.TrimSpace(slug))
	switch {
	case s == "":
		return ""
	case s == "go" || s == "golang" || s == "go-language" || strings.HasPrefix(s, "go-"):
		return "go"
	default:
		return s
	}
}

// IsSubtopicOf 判断 narrow 是否是 broad 的子话题（narrow 更具体）
func (r *Registry) IsSubtopicOf(narrow, broad string) bool {
	narrow = strings.ToLower(strings.TrimSpace(narrow))
	broad = strings.ToLower(strings.TrimSpace(broad))
	if narrow == "" || broad == "" || narrow == broad {
		return false
	}

	broadRoot := TopicRoot(broad)
	narrowRoot := TopicRoot(narrow)
	if broadRoot != "" && narrowRoot != "" && broadRoot != narrowRoot {
		return false
	}

	for cur := narrow; cur != ""; {
		parent := strings.ToLower(strings.TrimSpace(r.ParentSlug(cur)))
		if parent == broad || parent == broadRoot {
			return true
		}
		if parent == "" || parent == cur {
			break
		}
		cur = parent
	}

	// 同一主题族：带 parent 的子包 vs 宽泛生成课（如 go-concurrency vs go-language）
	if broadRoot != "" && narrowRoot == broadRoot && r.ParentSlug(narrow) != "" {
		if broad == broadRoot || strings.Contains(broad, broadRoot) {
			return true
		}
	}
	return false
}

// FindRelatedDomain 检查新课程是否与已有课程存在父子/同族关系
func (r *Registry) FindRelatedDomain(
	existing []storage.DomainSummary,
	newSlug, newName string,
) (*DomainRelation, error) {
	newSlug = strings.ToLower(strings.TrimSpace(newSlug))
	if newSlug == "" {
		return nil, nil
	}

	for _, d := range existing {
		existSlug := strings.ToLower(strings.TrimSpace(d.Slug))
		if existSlug == "" || existSlug == newSlug {
			continue
		}

		if r.IsSubtopicOf(existSlug, newSlug) {
			return &DomainRelation{
				Kind:           RelationExistingSubtopic,
				Message:        fmt.Sprintf("你已在学习「%s」，它属于「%s」的一部分。建议继续现有课程，或确认后再新建完整路径。", d.Name, newName),
				ExistingDomain: d,
			}, nil
		}

		if r.IsSubtopicOf(newSlug, existSlug) {
			return &DomainRelation{
				Kind:           RelationNewIsSubtopic,
				Message:        fmt.Sprintf("「%s」已涵盖在「%s」中，已为你打开现有课程。", newName, d.Name),
				ExistingDomain: d,
			}, nil
		}
	}
	return nil, nil
}
