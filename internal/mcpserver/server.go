// Package mcpserver 提供只读 MCP Server（stdio），供 Cursor 等宿主查询学习进度。
// 不暴露教学状态机：宿主只能查数据并拿到 Web 深链，真正讲练批仍在 Regulus Web 内完成。
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Config MCP 运行配置
type Config struct {
	UserID     string // REGULUS_MCP_USER_ID，默认 default
	WebBaseURL string // REGULUS_WEB_BASE_URL，用于深链，默认 http://localhost:8080
}

// Server 只读 MCP 服务
type Server struct {
	store     *storage.Store
	shortcuts *service.ShortcutsService
	cfg       Config
}

// New 创建 MCP Server
func New(store *storage.Store, shortcuts *service.ShortcutsService, cfg Config) *Server {
	if strings.TrimSpace(cfg.UserID) == "" {
		cfg.UserID = storage.DefaultUserID
	}
	if strings.TrimSpace(cfg.WebBaseURL) == "" {
		cfg.WebBaseURL = "http://localhost:8080"
	}
	cfg.WebBaseURL = strings.TrimRight(cfg.WebBaseURL, "/")
	return &Server{store: store, shortcuts: shortcuts, cfg: cfg}
}

// Run 在 stdio 上跑 JSON-RPC MCP，直到 stdin EOF。
func (s *Server) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// MCP 消息可能较大
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = writeJSON(out, errorResponse(nil, -32700, "Parse error"))
			continue
		}
		// 通知无需响应
		if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
			continue
		}
		resp := s.handle(ctx, &req)
		if resp == nil {
			continue
		}
		if err := writeJSON(out, resp); err != nil {
			return err
		}
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

func errorResponse(id json.RawMessage, code int, msg string) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}

func (s *Server) handle(ctx context.Context, req *jsonRPCRequest) *jsonRPCResponse {
	_ = ctx
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "regulus-academy",
					"version": "0.1.0",
				},
				"instructions": "Regulus Academy 只读学习助手：可查进度、错题、笔记与推荐下一节，并返回 Web 深链。真正的讲练批请在 Regulus Web 中完成。",
			},
		}
	case "ping":
		return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolDefs()}}
	case "tools/call":
		return s.callTool(req)
	default:
		if req.ID == nil {
			return nil
		}
		return errorResponse(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func toolDefs() []map[string]any {
	return []map[string]any{
		toolSchema("get_progress", "查询用户在某门课（或全部课）的节点学习进度", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"domainId": map[string]any{"type": "string", "description": "课程 ID；省略则返回全部课程摘要进度"},
			},
		}),
		toolSchema("get_next_node", "返回今日推荐下一节（侧栏快捷入口同源逻辑）", map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}),
		toolSchema("get_mistakes", "查询某门课的踩坑概念列表", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"domainId": map[string]any{"type": "string", "description": "课程 ID"},
				"nodeKey":  map[string]any{"type": "string", "description": "可选，只返回该节点"},
			},
			"required": []string{"domainId"},
		}),
		toolSchema("get_notes", "查询某门课蒸馏后的学习笔记", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"domainId": map[string]any{"type": "string", "description": "课程 ID"},
				"nodeKey":  map[string]any{"type": "string", "description": "可选，只返回该节点"},
			},
			"required": []string{"domainId"},
		}),
		toolSchema("open_session_link", "生成在 Regulus Web 打开某节点学习的深链（不启动教学）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"domainId": map[string]any{"type": "string", "description": "课程 ID"},
				"nodeKey":  map[string]any{"type": "string", "description": "节点 key；省略则打开课程树页"},
			},
			"required": []string{"domainId"},
		}),
	}
}

func toolSchema(name, desc string, inputSchema map[string]any) map[string]any {
	return map[string]any{
		"name":        name,
		"description": desc,
		"inputSchema": inputSchema,
	}
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) callTool(req *jsonRPCRequest) *jsonRPCResponse {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params")
	}
	args := map[string]any{}
	if len(params.Arguments) > 0 && string(params.Arguments) != "null" {
		_ = json.Unmarshal(params.Arguments, &args)
	}

	var (
		payload any
		err     error
	)
	switch params.Name {
	case "get_progress":
		payload, err = s.getProgress(strArg(args, "domainId"))
	case "get_next_node":
		payload, err = s.getNextNode()
	case "get_mistakes":
		payload, err = s.getMistakes(strArg(args, "domainId"), strArg(args, "nodeKey"))
	case "get_notes":
		payload, err = s.getNotes(strArg(args, "domainId"), strArg(args, "nodeKey"))
	case "open_session_link":
		payload, err = s.openSessionLink(strArg(args, "domainId"), strArg(args, "nodeKey"))
	default:
		return toolResult(req.ID, fmt.Sprintf("未知工具: %s", params.Name), true)
	}
	if err != nil {
		return toolResult(req.ID, err.Error(), true)
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return toolResult(req.ID, string(b), false)
}

func toolResult(id json.RawMessage, text string, isError bool) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{{
				"type": "text",
				"text": text,
			}},
			"isError": isError,
		},
	}
}

func strArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

func (s *Server) getProgress(domainID string) (any, error) {
	uid := s.cfg.UserID
	if domainID != "" {
		ok, err := s.store.DomainOwnedByUser(uid, domainID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("领域不存在")
		}
		list, err := s.store.ListProgress(uid, domainID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"domainId": domainID, "progress": list}, nil
	}
	domains, err := s.store.ListDomainSummaries(uid)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(domains))
	for _, d := range domains {
		out = append(out, map[string]any{
			"domainId":   d.ID,
			"domainName": d.Name,
			"completed":  d.Completed,
			"nodeTotal":  d.NodeTotal,
			"source":     d.Source,
		})
	}
	return map[string]any{"domains": out}, nil
}

func (s *Server) getNextNode() (any, error) {
	if s.shortcuts == nil {
		return nil, fmt.Errorf("shortcuts 服务未初始化")
	}
	return s.shortcuts.GetLearningShortcuts(s.cfg.UserID)
}

func (s *Server) getMistakes(domainID, nodeKey string) (any, error) {
	if domainID == "" {
		return nil, fmt.Errorf("缺少 domainId")
	}
	uid := s.cfg.UserID
	ok, err := s.store.DomainOwnedByUser(uid, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("领域不存在")
	}
	mistakes, err := s.store.ListMistakesByNode(uid, domainID)
	if err != nil {
		return nil, err
	}
	if nodeKey != "" {
		concepts := mistakes[nodeKey]
		if concepts == nil {
			concepts = []string{}
		}
		return map[string]any{
			"domainId": domainID,
			"mistakes": []map[string]any{{"nodeKey": nodeKey, "concepts": concepts}},
		}, nil
	}
	items := make([]map[string]any, 0, len(mistakes))
	for k, concepts := range mistakes {
		items = append(items, map[string]any{"nodeKey": k, "concepts": concepts})
	}
	return map[string]any{"domainId": domainID, "mistakes": items}, nil
}

func (s *Server) getNotes(domainID, nodeKey string) (any, error) {
	if domainID == "" {
		return nil, fmt.Errorf("缺少 domainId")
	}
	uid := s.cfg.UserID
	ok, err := s.store.DomainOwnedByUser(uid, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("领域不存在")
	}
	if nodeKey != "" {
		content, err := s.store.GetNodeNote(uid, domainID, nodeKey)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"domainId": domainID,
			"notes":    []map[string]string{{"nodeKey": nodeKey, "contentMd": content}},
		}, nil
	}
	notes, err := s.store.ListNodeNotes(uid, domainID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]string, 0, len(notes))
	for k, v := range notes {
		items = append(items, map[string]string{"nodeKey": k, "contentMd": v})
	}
	return map[string]any{"domainId": domainID, "notes": items}, nil
}

func (s *Server) openSessionLink(domainID, nodeKey string) (any, error) {
	if domainID == "" {
		return nil, fmt.Errorf("缺少 domainId")
	}
	uid := s.cfg.UserID
	ok, err := s.store.DomainOwnedByUser(uid, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("领域不存在")
	}
	hash := "#/tree/" + domainID
	if nodeKey != "" {
		// 课程树页消费 ?node= 高亮并滚到该节点；开练仍由用户点击触发
		hash = "#/tree/" + domainID + "?node=" + url.QueryEscape(nodeKey)
	}
	linkURL := s.cfg.WebBaseURL + "/" + hash
	return map[string]any{
		"url":      linkURL,
		"domainId": domainID,
		"nodeKey":  nodeKey,
		"hint":     "在浏览器打开该链接，于 Regulus Web 中完成讲练批。MCP 不会启动教学会话。",
	}, nil
}

// LoadConfigFromEnv 从环境变量读取 MCP 配置
func LoadConfigFromEnv() Config {
	return Config{
		UserID:     strings.TrimSpace(os.Getenv("REGULUS_MCP_USER_ID")),
		WebBaseURL: strings.TrimSpace(os.Getenv("REGULUS_WEB_BASE_URL")),
	}
}
