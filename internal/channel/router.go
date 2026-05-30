package channel

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type flatNode struct {
	Key   string
	Title string
	Layer string
}

// Router 处理 IM 命令与 Coach 转发
type Router struct {
	store    *storage.Store
	sessions *service.SessionService
	mu       sync.Mutex
	pending  map[string]string // userID -> domainID（学习命令后待选节点）
}

// NewRouter 创建消息路由器
func NewRouter(store *storage.Store, sessions *service.SessionService) *Router {
	return &Router{
		store:    store,
		sessions: sessions,
		pending:  make(map[string]string),
	}
}

// Handle 处理入站消息，返回回复文本列表
func (r *Router) Handle(ctx context.Context, ev MessageEvent) []string {
	text := strings.TrimSpace(ev.Text)
	if text == "" {
		return nil
	}

	if !r.allowed(ev) {
		return []string{"你暂无权限使用此机器人。"}
	}

	binding, _ := r.store.GetChannelBinding(ev.Platform, ev.PlatformUserID)
	cmd, arg := parseCommand(text)

	if cmd == "bind" {
		return r.handleBind(ev, arg)
	}
	if binding == nil {
		return []string{"请先绑定学习角色，发送：绑定 你的角色名\n（角色需在 Web 端先创建）"}
	}

	userID := binding.UserID

	switch cmd {
	case "help":
		return []string{helpText()}
	case "courses":
		return r.handleCourses(userID)
	case "learn":
		return r.handleLearn(userID, arg)
	case "node":
		return r.handleNode(ctx, userID, arg)
	case "continue":
		return r.handleContinue(ctx, userID)
	case "progress":
		return r.handleProgress(userID)
	default:
		return r.handleChat(ctx, userID, text)
	}
}

func (r *Router) allowed(ev MessageEvent) bool {
	// 平台级 allowlist 在 adapter 层也可配置；此处默认允许
	return ev.PlatformUserID != ""
}

func (r *Router) handleBind(ev MessageEvent, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{"请发送：绑定 你的角色名\n例如：绑定 小明"}
	}
	user, err := r.store.FindUserByDisplayName(name)
	if err != nil {
		return []string{err.Error()}
	}
	if err := r.store.UpsertChannelBinding(ev.Platform, ev.PlatformUserID, user.ID, user.DisplayName); err != nil {
		return []string{"绑定失败，请稍后重试。"}
	}
	return []string{fmt.Sprintf("已绑定为「%s」。发送「课程」查看知识库，「帮助」查看命令。", user.DisplayName)}
}

func (r *Router) handleCourses(userID string) []string {
	list, err := r.store.ListDomainSummaries(userID)
	if err != nil {
		return []string{"无法加载课程列表。"}
	}
	if len(list) == 0 {
		return []string{"还没有课程。请先在 Web 端创建知识库。"}
	}
	var b strings.Builder
	b.WriteString("你的课程：\n")
	for i, d := range list {
		b.WriteString(fmt.Sprintf("%d. %s（已完成 %d/%d）\n", i+1, d.Name, d.Completed, d.NodeTotal))
	}
	b.WriteString("\n发送「学习 序号」查看节点，例如：学习 1")
	return []string{b.String()}
}

func (r *Router) handleLearn(userID, arg string) []string {
	list, err := r.store.ListDomainSummaries(userID)
	if err != nil || len(list) == 0 {
		return []string{"还没有课程。请先在 Web 端创建知识库。"}
	}

	var domainID string
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return []string{"请发送：学习 课程序号\n例如：学习 1"}
	}

	if n := parsePositiveInt(arg); n > 0 && n <= len(list) {
		domainID = list[n-1].ID
	} else {
		for _, d := range list {
			if d.Slug == arg || d.ID == arg || d.Name == arg {
				domainID = d.ID
				break
			}
		}
	}
	if domainID == "" {
		return []string{"未找到该课程。发送「课程」查看列表。"}
	}

	tree, err := r.store.GetDomainTree(userID, domainID)
	if err != nil || tree == nil {
		return []string{"无法加载知识树。"}
	}

	r.mu.Lock()
	r.pending[userID] = domainID
	r.mu.Unlock()

	nodes := flattenNodes(tree)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("《%s》节点列表：\n", tree.DomainName))
	for i, n := range nodes {
		b.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, n.Title, n.Key))
	}
	b.WriteString("\n发送「节点 序号」开始学习，例如：节点 1")
	return []string{b.String()}
}

func (r *Router) handleNode(ctx context.Context, userID, arg string) []string {
	r.mu.Lock()
	domainID := r.pending[userID]
	r.mu.Unlock()

	if domainID == "" {
		active, _ := r.store.GetChannelActiveNode(userID)
		if active != nil {
			domainID = active.DomainID
		}
	}
	if domainID == "" {
		return []string{"请先发送「学习 课程序号」选择课程。"}
	}

	tree, err := r.store.GetDomainTree(userID, domainID)
	if err != nil || tree == nil {
		return []string{"无法加载知识树。"}
	}
	nodes := flattenNodes(tree)
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return []string{"请发送：节点 序号\n例如：节点 1"}
	}

	var nodeKey, layer string
	if n := parsePositiveInt(arg); n > 0 && n <= len(nodes) {
		nodeKey = nodes[n-1].Key
		layer = nodes[n-1].Layer
	} else {
		for _, nd := range nodes {
			if nd.Key == arg {
				nodeKey = nd.Key
				layer = nd.Layer
				break
			}
		}
	}
	if nodeKey == "" {
		return []string{"未找到该节点。请先「学习」查看节点列表。"}
	}

	result, err := r.sessions.StartOrResumeSession(ctx, userID, domainID, nodeKey, layer)
	if err != nil {
		return []string{"开始学习失败：" + err.Error()}
	}

	_ = r.store.SetChannelActiveNode(userID, domainID, nodeKey)

	if result.Resumed {
		msgs, _ := r.store.ListMessages(result.Session.ID)
		last := ""
		if len(msgs) > 0 {
			last = msgs[len(msgs)-1].Content
		}
		title := domain.NodeTitle(tree, nodeKey)
		if last != "" {
			return []string{fmt.Sprintf("继续学习「%s」（阶段：%s）。上一条：\n%s\n\n直接回复即可继续。", title, result.Session.Phase, truncate(last, 200))}
		}
		return []string{fmt.Sprintf("继续学习「%s」（阶段：%s）。直接回复即可。", title, result.Session.Phase)}
	}

	title := domain.NodeTitle(tree, nodeKey)
	return []string{fmt.Sprintf("开始学习「%s」\n\n%s", title, result.Content)}
}

func (r *Router) handleContinue(ctx context.Context, userID string) []string {
	sess, err := r.sessions.ActiveSessionForUser(userID)
	if err != nil || sess == nil {
		return []string{"没有进行中的学习。发送「课程」选课，或「学习 1」开始。"}
	}

	tree, _ := r.store.GetDomainTree(userID, sess.DomainID)
	title := sess.NodeKey
	if tree != nil {
		title = domain.NodeTitle(tree, sess.NodeKey)
	}
	msgs, _ := r.store.ListMessages(sess.ID)
	last := ""
	if len(msgs) > 0 {
		last = msgs[len(msgs)-1].Content
	}
	if last != "" {
		return []string{fmt.Sprintf("当前：「%s」（%s）\n上一条：\n%s\n\n直接回复继续。", title, sess.Phase, truncate(last, 300))}
	}
	return []string{fmt.Sprintf("当前：「%s」（%s）。直接回复继续。", title, sess.Phase)}
}

func (r *Router) handleProgress(userID string) []string {
	list, err := r.store.ListDomainSummaries(userID)
	if err != nil || len(list) == 0 {
		return []string{"暂无学习进度。"}
	}
	var b strings.Builder
	b.WriteString("学习进度：\n")
	for _, d := range list {
		pct := 0
		if d.NodeTotal > 0 {
			pct = d.Completed * 100 / d.NodeTotal
		}
		b.WriteString(fmt.Sprintf("· %s：%d/%d（%d%%）\n", d.Name, d.Completed, d.NodeTotal, pct))
	}
	return []string{b.String()}
}

func (r *Router) handleChat(ctx context.Context, userID, text string) []string {
	sess, err := r.sessions.ActiveSessionForUser(userID)
	if err != nil || sess == nil {
		return []string{"请先选课并开始节点：\n1. 课程\n2. 学习 1\n3. 节点 1\n\n或发送「帮助」查看命令。"}
	}

	out, err := r.sessions.SendCoachMessage(ctx, userID, sess.ID, text)
	if err != nil {
		return []string{"处理失败：" + err.Error()}
	}
	return []string{out.Result.Content}
}

func parseCommand(text string) (cmd, arg string) {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	switch {
	case strings.HasPrefix(text, "绑定"):
		return "bind", strings.TrimSpace(strings.TrimPrefix(text, "绑定"))
	case text == "帮助" || lower == "help" || text == "/help":
		return "help", ""
	case text == "课程" || lower == "courses":
		return "courses", ""
	case strings.HasPrefix(text, "学习"):
		return "learn", strings.TrimSpace(strings.TrimPrefix(text, "学习"))
	case strings.HasPrefix(text, "节点"):
		return "node", strings.TrimSpace(strings.TrimPrefix(text, "节点"))
	case text == "继续" || lower == "continue":
		return "continue", ""
	case text == "进度" || lower == "progress":
		return "progress", ""
	default:
		return "", text
	}
}

func helpText() string {
	return `Regulus 学习教练 — 命令

绑定 名字 — 绑定 Web 端学习角色
课程 — 查看知识库
学习 序号 — 查看课程节点
节点 序号 — 开始/继续节点学习
继续 — 查看当前学习状态
进度 — 查看完成进度
帮助 — 显示本说明

绑定后直接发消息即可与教练对话。`
}

func flattenNodes(tree *storage.KnowledgeTree) []flatNode {
	var out []flatNode
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			out = append(out, flatNode{Key: n.Key, Title: n.Title, Layer: layer.Key})
		}
	}
	return out
}

func parsePositiveInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// Dispatch 处理消息并通过 adapter 发送回复
func Dispatch(ctx context.Context, router *Router, adapter Adapter, ev MessageEvent) {
	log.Printf("[gateway/%s] 收到: user=%s text=%q", adapter.Name(), ev.PlatformUserID, truncate(ev.Text, 80))
	replies := router.Handle(ctx, ev)
	if len(replies) == 0 {
		log.Printf("[gateway/%s] 无回复", adapter.Name())
		return
	}
	target := ReplyFromEvent(ev)
	for i, reply := range replies {
		for _, chunk := range SplitMessage(reply, defaultChunkRunes) {
			if err := adapter.SendText(ctx, target, chunk); err != nil {
				log.Printf("[gateway/%s] 发送失败: %v", adapter.Name(), err)
			} else {
				log.Printf("[gateway/%s] 已回复 (%d/%d)", adapter.Name(), i+1, len(replies))
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
