package cliruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RemoteClient 调用已部署 Regulus HTTP API。
type RemoteClient struct {
	baseURL string
	userID  string
	client  *http.Client
}

// NewRemoteClient 创建远程客户端。
func NewRemoteClient(baseURL, userID string) *RemoteClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &RemoteClient{
		baseURL: baseURL,
		userID:  userID,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

// Online 探测 /health。
func (c *RemoteClient) Online(ctx context.Context) bool {
	if c == nil || c.baseURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *RemoteClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-User-Id", c.userID)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errBody)
		if errBody.Error != "" {
			return fmt.Errorf("%s", errBody.Error)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}

type remoteDomainSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (c *RemoteClient) resolveDomainID(ctx context.Context, slug string) (string, error) {
	var out struct {
		Domains []remoteDomainSummary `json:"domains"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/domains", nil, &out); err != nil {
		return "", err
	}
	for _, d := range out.Domains {
		if d.Slug == slug {
			return d.ID, nil
		}
	}
	return "", fmt.Errorf("远程无课程 slug=%s，请先在 Web 建课或 regulus build", slug)
}

// SessionStart 远程开课。
func (c *RemoteClient) SessionStart(ctx context.Context, slug, nodeKey, layer string) (*SessionStartResult, error) {
	domainID, err := c.resolveDomainID(ctx, slug)
	if err != nil {
		return nil, err
	}
	if layer == "" {
		layer = "entry"
	}
	body := map[string]string{
		"domainId": domainID,
		"nodeKey":  nodeKey,
		"layer":    layer,
	}
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/api/session/start", body, &out); err != nil {
		return nil, err
	}
	res := &SessionStartResult{
		SessionID: str(out["sessionId"]),
		DomainID:  str(out["domainId"]),
		Slug:      slug,
		NodeKey:   str(out["nodeKey"]),
		Phase:     str(out["phase"]),
		Content:   str(out["content"]),
		Resumed:   out["resumed"] == true,
	}
	if res.Phase == "" {
		res.Phase = "explain"
	}
	return res, nil
}

// SessionMessage 远程发消息。
func (c *RemoteClient) SessionMessage(ctx context.Context, sessionID, content string) (*SessionMessageResult, error) {
	body := map[string]string{
		"sessionId": sessionID,
		"content":   content,
	}
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/api/session/message", body, &out); err != nil {
		return nil, err
	}
	return &SessionMessageResult{
		SessionID:     str(out["sessionId"]),
		Phase:         str(out["phase"]),
		Content:       str(out["content"]),
		NextSessionID: str(out["nextSessionId"]),
	}, nil
}

// FetchRemoteProgress 拉取远程进度。
func (c *RemoteClient) FetchRemoteProgress(ctx context.Context, domainID string) ([]ProgressRow, error) {
	path := "/api/user/progress"
	if domainID != "" {
		path += "?domainId=" + domainID
	}
	var out struct {
		Progress []struct {
			DomainID  string  `json:"domainId"`
			NodeKey   string  `json:"nodeKey"`
			Layer     string  `json:"layer"`
			Status    string  `json:"status"`
			Mastery   float64 `json:"mastery"`
			UpdatedAt string  `json:"updatedAt"`
		} `json:"progress"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	rows := make([]ProgressRow, 0, len(out.Progress))
	for _, p := range out.Progress {
		rows = append(rows, ProgressRow{
			DomainID:  p.DomainID,
			NodeKey:   p.NodeKey,
			Layer:     p.Layer,
			Status:    p.Status,
			Mastery:   p.Mastery,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return rows, nil
}

// PushProgress 推送进度到远程（按 slug）。
func (c *RemoteClient) PushProgress(ctx context.Context, items []SyncProgressItem) (int, error) {
	var out struct {
		Merged int `json:"merged"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/sync/progress", map[string]any{"items": items}, &out); err != nil {
		return 0, err
	}
	return out.Merged, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
