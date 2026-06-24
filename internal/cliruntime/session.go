package cliruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
)

// SessionStartResult 开课结果（JSON 输出）。
type SessionStartResult struct {
	SessionID string `json:"sessionId"`
	DomainID  string `json:"domainId"`
	Slug      string `json:"slug"`
	NodeKey   string `json:"nodeKey"`
	NodeTitle string `json:"nodeTitle,omitempty"`
	Phase     string `json:"phase"`
	Content   string `json:"content,omitempty"`
	Resumed   bool   `json:"resumed,omitempty"`
}

// SessionMessageResult 消息结果。
type SessionMessageResult struct {
	SessionID     string `json:"sessionId"`
	Phase         string `json:"phase"`
	Content       string `json:"content"`
	NextSessionID string `json:"nextSessionId,omitempty"`
}

// ProgressRow 进度行。
type ProgressRow struct {
	Slug      string  `json:"slug"`
	DomainID  string  `json:"domainId"`
	NodeKey   string  `json:"nodeKey"`
	Layer     string  `json:"layer"`
	Status    string  `json:"status"`
	Mastery   float64 `json:"mastery"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// SessionStart 开始或恢复某节点学习。
func (rt *Runtime) SessionStart(ctx context.Context, slug, nodeKey, layer string) (*SessionStartResult, error) {
	slug = domain.Slugify(slug)
	if slug == "" {
		return nil, fmt.Errorf("slug 不能为空")
	}

	if rt.Linked() && rt.remote.Online(ctx) {
		if nodeKey == "" {
			_, tree, err := rt.EnsureDomainFromSlug(slug)
			if err != nil {
				return nil, err
			}
			prog, _ := rt.store.ListProgress(rt.UserID(), tree.DomainID)
			nodeKey, layer, err = PickStartNode(tree, prog)
			if err != nil {
				return nil, err
			}
		}
		if layer == "" {
			layer = "entry"
		}
		return rt.remote.SessionStart(ctx, slug, nodeKey, layer)
	}

	if !rt.LLMConfigured() {
		return nil, fmt.Errorf("未配置 LLM，请在 %s 或 data/.env 设置 LLM_API_KEY", rt.paths.CoachRoot)
	}

	dom, tree, err := rt.EnsureDomainFromSlug(slug)
	if err != nil {
		return nil, err
	}
	if nodeKey == "" {
		prog, _ := rt.store.ListProgress(rt.UserID(), dom.ID)
		nodeKey, layer, err = PickStartNode(tree, prog)
		if err != nil {
			return nil, err
		}
	}
	if layer == "" {
		layer = "entry"
	}

	runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	result, err := rt.sessions.StartOrResumeSession(runCtx, rt.UserID(), dom.ID, nodeKey, layer)
	if err != nil {
		return nil, err
	}
	out := &SessionStartResult{
		SessionID: result.Session.ID,
		DomainID:  dom.ID,
		Slug:      slug,
		NodeKey:   result.Session.NodeKey,
		NodeTitle: domain.NodeTitle(tree, result.Session.NodeKey),
		Phase:     result.Session.Phase,
		Resumed:   result.Resumed,
	}
	if !result.Resumed && result.Content != "" {
		out.Content = result.Content
		out.Phase = "explain"
	}
	return out, nil
}

// SessionMessage 发送用户消息。
func (rt *Runtime) SessionMessage(ctx context.Context, sessionID, content string) (*SessionMessageResult, error) {
	content = strings.TrimSpace(content)
	if sessionID == "" || content == "" {
		return nil, fmt.Errorf("sessionId 和 content 不能为空")
	}

	if rt.Linked() && rt.remote.Online(ctx) {
		return rt.remote.SessionMessage(ctx, sessionID, content)
	}

	if !rt.LLMConfigured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}

	runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	sent, err := rt.sessions.SendCoachMessage(runCtx, rt.UserID(), sessionID, content)
	if err != nil {
		return nil, err
	}
	out := &SessionMessageResult{
		SessionID: sent.Session.ID,
		Phase:     sent.Session.Phase,
		Content:   sent.Result.Content,
	}
	if sent.Result.NextSessionID != "" {
		out.NextSessionID = sent.Result.NextSessionID
		out.SessionID = sent.Result.NextSessionID
	}
	return out, nil
}

// ListProgress 列出本地进度。
func (rt *Runtime) ListProgress(slug string) ([]ProgressRow, error) {
	var domainID string
	if slug != "" {
		slug = domain.Slugify(slug)
		dom, _, err := rt.EnsureDomainFromSlug(slug)
		if err != nil {
			return nil, err
		}
		domainID = dom.ID
	}
	list, err := rt.store.ListProgress(rt.UserID(), domainID)
	if err != nil {
		return nil, err
	}
	rows := make([]ProgressRow, 0, len(list))
	for _, p := range list {
		slugVal, _ := rt.store.GetDomainSlug(p.DomainID)
		rows = append(rows, ProgressRow{
			Slug:      slugVal,
			DomainID:  p.DomainID,
			NodeKey:   p.NodeKey,
			Layer:     p.Layer,
			Status:    p.Status,
			Mastery:   p.Mastery,
			UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return rows, nil
}

// PrintJSON 向 stdout 输出 JSON。
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
