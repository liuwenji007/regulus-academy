package domain

import (
	"sort"
	"strings"
	"unicode"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// DerivationResolver 按子课 domain ID 解析落库的衍生锚点（生成课 derivation_json）
type DerivationResolver func(domainID string) *DerivationDef

// CourseLinkParent 父课程链接
type CourseLinkParent struct {
	DomainID string `json:"domainId"`
	Name     string `json:"name"`
	Slug     string `json:"slug,omitempty"`
}

// CourseDerivation 根课上的衍生子课跳转
type CourseDerivation struct {
	ChildDomainID string `json:"childDomainId"`
	ChildName     string `json:"childName"`
	ChildSlug     string `json:"childSlug,omitempty"`
	AfterNodeKey  string `json:"afterNodeKey"`
	LayerKey      string `json:"layerKey"`
	Label         string `json:"label"`
}

// CourseLinks 课程关联链接
type CourseLinks struct {
	Parent      *CourseLinkParent  `json:"parent,omitempty"`
	Derivations []CourseDerivation `json:"derivations,omitempty"`
}

// SlugMatchesTopicFamily 判断 childParentSlug 是否属于 parentSlug 主题族
func SlugMatchesTopicFamily(childParentSlug, parentSlug string) bool {
	childParentSlug = strings.ToLower(strings.TrimSpace(childParentSlug))
	parentSlug = strings.ToLower(strings.TrimSpace(parentSlug))
	if childParentSlug == "" || parentSlug == "" {
		return false
	}
	if childParentSlug == parentSlug {
		return true
	}
	cr := TopicRoot(childParentSlug)
	pr := TopicRoot(parentSlug)
	return cr != "" && cr == pr
}

// FindParentDomainSummary 在用户课程中查找子课的父课
func FindParentDomainSummary(all []storage.DomainSummary, childParentSlug string) *storage.DomainSummary {
	want := strings.ToLower(strings.TrimSpace(childParentSlug))
	if want == "" {
		return nil
	}
	wantRoot := TopicRoot(want)
	var exact, family *storage.DomainSummary
	for i := range all {
		d := &all[i]
		s := strings.ToLower(strings.TrimSpace(d.Slug))
		if s == "" {
			continue
		}
		if s == want {
			exact = d
			break
		}
		if family == nil && (s == wantRoot || TopicRoot(s) == wantRoot) {
			family = d
		}
	}
	if exact != nil {
		return exact
	}
	return family
}

// FindChildDomainSummaries 查找当前父课下的子课（与 FindParentDomainSummary 互逆）
func FindChildDomainSummaries(r *Registry, all []storage.DomainSummary, parent storage.DomainSummary) []storage.DomainSummary {
	parentID := strings.TrimSpace(parent.ID)
	if parentID == "" {
		return nil
	}
	var out []storage.DomainSummary
	for _, d := range all {
		if d.ID == parentID {
			continue
		}
		ps := strings.TrimSpace(d.ParentSlug)
		if ps == "" && d.Slug != "" {
			ps = strings.TrimSpace(r.ParentSlug(d.Slug))
		}
		if ps == "" {
			continue
		}
		if p := FindParentDomainSummary(all, ps); p == nil || p.ID != parentID {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// ResolveCourseLinks 解析当前课程的父链与衍生跳转
func (r *Registry) ResolveCourseLinks(
	all []storage.DomainSummary,
	current storage.DomainSummary,
	parentTree *storage.KnowledgeTree,
	deriv DerivationResolver,
) CourseLinks {
	var links CourseLinks

	parentSlug := strings.TrimSpace(current.ParentSlug)
	if parentSlug == "" && current.Slug != "" {
		parentSlug = strings.TrimSpace(r.ParentSlug(current.Slug))
	}
	if parentSlug != "" {
		if p := FindParentDomainSummary(all, parentSlug); p != nil {
			links.Parent = &CourseLinkParent{
				DomainID: p.ID,
				Name:     p.Name,
				Slug:     p.Slug,
			}
		}
	}

	if parentTree == nil {
		return links
	}
	children := FindChildDomainSummaries(r, all, current)
	for _, child := range children {
		afterKey, layerKey, label := r.resolveDerivationAnchor(parentTree, child.ID, child.Slug, child.Name, deriv)
		if afterKey == "" {
			continue
		}
		links.Derivations = append(links.Derivations, CourseDerivation{
			ChildDomainID: child.ID,
			ChildName:     child.Name,
			ChildSlug:     child.Slug,
			AfterNodeKey:  afterKey,
			LayerKey:      layerKey,
			Label:         label,
		})
	}
	return links
}

func (r *Registry) resolveDerivationAnchor(parentTree *storage.KnowledgeTree, childID, childSlug, childName string, deriv DerivationResolver) (afterNodeKey, layerKey, label string) {
	label = strings.TrimSpace(childName)
	keywords := r.derivationKeywords(childID, childSlug, childName, deriv)
	if custom := r.derivationJumpLabel(childID, childSlug, deriv); custom != "" {
		label = custom
	}
	if label == "" {
		label = "专题衍生课程"
	}

	var lastKey, lastLayer string
	for _, layer := range parentTree.Layers {
		for _, node := range layer.Nodes {
			lastKey = node.Key
			lastLayer = layer.Key
			if nodeTitleMatchesKeywords(node.Title, keywords) {
				afterNodeKey = node.Key
				layerKey = layer.Key
			}
		}
	}
	_ = lastKey
	_ = lastLayer
	return afterNodeKey, layerKey, label
}

func (r *Registry) derivationKeywords(childID, childSlug, childName string, deriv DerivationResolver) []string {
	if deriv != nil {
		if d := deriv(childID); d != nil && len(d.ParentAnchorKeywords) > 0 {
			return d.ParentAnchorKeywords
		}
	}
	tf, err := r.readTreeFileBySlug(childSlug)
	if err == nil && tf.Derivation != nil && len(tf.Derivation.ParentAnchorKeywords) > 0 {
		return tf.Derivation.ParentAnchorKeywords
	}
	var kw []string
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		low := strings.ToLower(s)
		if _, ok := seen[low]; ok {
			return
		}
		seen[low] = struct{}{}
		kw = append(kw, s)
	}
	for _, part := range splitTopicTokens(childName) {
		add(part)
	}
	if tree, err := r.LoadTree(childSlug); err == nil && len(tree.Modules) > 0 {
		add(tree.Modules[0].Label)
	}
	return kw
}

func (r *Registry) derivationJumpLabel(childID, childSlug string, deriv DerivationResolver) string {
	if deriv != nil {
		if d := deriv(childID); d != nil {
			if label := strings.TrimSpace(d.JumpLabel); label != "" {
				return label
			}
		}
	}
	tf, err := r.readTreeFileBySlug(childSlug)
	if err != nil || tf.Derivation == nil {
		return ""
	}
	return strings.TrimSpace(tf.Derivation.JumpLabel)
}

func nodeTitleMatchesKeywords(title string, keywords []string) bool {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" || len(keywords) == 0 {
		return false
	}
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" && strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

func splitTopicTokens(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}
