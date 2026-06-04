package channel

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
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
	llm      llm.Provider
	mu       sync.Mutex
	pending  map[string]string // userID -> domainID（学习命令后待选节点）
}

// NewRouter 创建消息路由器
func NewRouter(store *storage.Store, sessions *service.SessionService, llmClient llm.Provider) *Router {
	return &Router{
		store:    store,
		sessions: sessions,
		llm:      llmClient,
		pending:  make(map[string]string),
	}
}

// HandleResult IM 路由处理结果
type HandleResult struct {
	InstantReplies []string
	Replies        []string
}

// Handle 处理入站消息
func (r *Router) Handle(ctx context.Context, ev MessageEvent) HandleResult {
	text := strings.TrimSpace(ev.Text)
	if text == "" {
		return HandleResult{}
	}

	if !PlatformUserAllowed(ev.Platform, ev.PlatformUserID) {
		return HandleResult{Replies: []string{"你暂无权限使用此机器人。请联系管理员将你加入允许列表。"}}
	}

	binding, _ := r.store.GetChannelBinding(ev.Platform, ev.PlatformUserID)
	cmd, arg := parseCommand(text)

	if cmd == "bind" {
		return HandleResult{Replies: r.handleBind(ev, arg)}
	}
	if binding == nil {
		return HandleResult{Replies: []string{"请先绑定学习角色，发送：绑定 你的角色名\n或：绑定 Web 端生成的 6 位绑定码\n（角色需在 Web 端先创建）"}}
	}

	userID := binding.UserID

	switch cmd {
	case "help":
		return HandleResult{Replies: []string{helpText()}}
	case "courses":
		return HandleResult{Replies: r.handleCourses(userID)}
	case "learn":
		return HandleResult{Replies: r.handleLearn(userID, arg)}
	case "node":
		return HandleResult{Replies: r.handleNode(ctx, userID, arg)}
	case "continue":
		return HandleResult{Replies: r.handleContinue(ctx, userID)}
	case "progress":
		return HandleResult{Replies: r.handleProgress(userID)}
	case "next":
		return r.handleNextSection(ctx, userID)
	default:
		return r.handleFreeText(ctx, userID, ev.Platform, text)
	}
}

func (r *Router) handleFreeText(ctx context.Context, userID, platform, text string) HandleResult {
	navCtx := r.buildNavContext(userID, platform)
	// 已在节点内学习时，默认交给 Coach；仅响应明确的导航意图（看课表/进度/帮助等）
	if navCtx.HasActiveSession {
		if sess, _ := r.sessions.ActiveSessionForUser(userID); sess != nil && matchesNextSection(text) {
			if sess.Phase == "completed" {
				return r.handleNextSection(ctx, userID)
			}
			return HandleResult{Replies: []string{"当前节点尚未完成。完成练习后，或在对话中说明「已经掌握，下一节」申请完成。"}}
		}
		if intent, ok := matchNavigationRulesWhileLearning(text, navCtx); ok {
			return r.executeNavigation(ctx, userID, intent, text)
		}
		return r.handleChat(ctx, userID, text)
	}
	if intent, ok := matchNavigationRules(text, navCtx); ok {
		return r.executeNavigation(ctx, userID, intent, text)
	}
	if r.llm != nil && r.llm.Configured() {
		intent, err := ParseNavIntent(ctx, r.llm, navCtx, text)
		if err == nil {
			return r.executeNavigation(ctx, userID, intent, text)
		}
	}
	return r.handleChat(ctx, userID, text)
}

func (r *Router) handleBind(ev MessageEvent, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{"请发送：绑定 你的角色名\n或：绑定 AB12CD（Web 端生成的绑定码）"}
	}
	if isBindCode(name) {
		userID, err := r.store.RedeemBindCode(strings.ToUpper(name))
		if err != nil {
			return []string{err.Error()}
		}
		user, err := r.store.GetUser(userID)
		if err != nil {
			return []string{"绑定失败，请稍后重试。"}
		}
		if err := r.store.UpsertChannelBinding(ev.Platform, ev.PlatformUserID, user.ID, user.DisplayName); err != nil {
			return []string{"绑定失败，请稍后重试。"}
		}
		return []string{bindWelcomeMessage(user.DisplayName)}
	}
	user, err := r.store.FindUserByDisplayName(name)
	if err != nil {
		return []string{err.Error()}
	}
	if err := r.store.UpsertChannelBinding(ev.Platform, ev.PlatformUserID, user.ID, user.DisplayName); err != nil {
		return []string{"绑定失败，请稍后重试。"}
	}
	return []string{bindWelcomeMessage(user.DisplayName)}
}

func bindWelcomeMessage(displayName string) string {
	return fmt.Sprintf(`已绑定为「%s」。

你可以直接说：
·「我的课程」查看课表
·「接着学」续上次进度
·「学 Go 并发」或「学习 1」选一门课

发「帮助」可看更多说明。`, displayName)
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
	b.WriteString("\n发送「学习 序号」或「学 课程名」查看节点")
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
	} else if id, ok := resolveCourseRef(list, arg); ok {
		domainID = id
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
	b.WriteString("\n发送「节点 序号」或节点名开始学习，例如：节点 1")
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
	} else if key, ly, ok := resolveNodeRef(nodes, arg); ok {
		nodeKey, layer = key, ly
	}
	if nodeKey == "" {
		return []string{"未找到该节点。请先「学习」查看节点列表。"}
	}

	return r.handleNodeByKey(ctx, userID, domainID, nodeKey, layer, tree)
}

func (r *Router) handleContinue(ctx context.Context, userID string) []string {
	active, _ := r.store.GetChannelActiveNode(userID)
	sess, err := r.sessions.ActiveSessionForUser(userID)
	if err != nil || sess == nil {
		if active != nil {
			tree, _ := r.store.GetDomainTree(userID, active.DomainID)
			name := active.NodeKey
			if tree != nil {
				name = domain.NodeTitle(tree, active.NodeKey)
			}
			return []string{fmt.Sprintf("当前节点「%s」尚未开始会话。发送「节点 序号」或重新选择节点开始学习。", name)}
		}
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

func (r *Router) handleChat(ctx context.Context, userID, text string) HandleResult {
	sess, err := r.sessions.ActiveSessionForUser(userID)
	if err != nil || sess == nil {
		return HandleResult{Replies: []string{"还没有进行中的学习。你可以说「我的课程」选课，或「接着学」续上次进度。"}}
	}

	log.Printf("[gateway] Coach 处理中 user=%s session=%s（LLM 可能需要 10–60 秒）", userID, sess.ID)
	sessID := sess.ID
	out, err := r.sessions.SendCoachMessage(ctx, userID, sessID, text)
	if err != nil {
		if err == service.ErrSessionBusy {
			return HandleResult{Replies: []string{"正在回复上一条消息，请稍候…"}}
		}
		return HandleResult{Replies: []string{"处理失败：" + err.Error()}}
	}
	if out.Result.NextSessionID != "" && out.Session != nil {
		_ = r.store.SetChannelActiveNode(userID, out.Session.DomainID, out.Session.NodeKey)
	}
	return HandleResult{
		Replies: []string{out.Result.Content},
	}
}

func (r *Router) handleNextSection(ctx context.Context, userID string) HandleResult {
	sess, err := r.sessions.ActiveSessionForUser(userID)
	if err != nil || sess == nil {
		return HandleResult{Replies: []string{"还没有学习记录。你可以说「我的课程」选课开始。"}}
	}
	if sess.Phase != "completed" {
		return HandleResult{Replies: []string{"当前节点尚未完成。完成练习后，或在对话中说明「已经掌握，下一节」申请完成。"}}
	}

	log.Printf("[gateway] 进入下一节 user=%s node=%s", userID, sess.NodeKey)
	result, err := r.sessions.StartNextNode(ctx, userID, sess.ID)
	if err != nil {
		return HandleResult{Replies: []string{err.Error()}}
	}
	_ = r.store.SetChannelActiveNode(userID, result.Session.DomainID, result.Session.NodeKey)

	tree, _ := r.store.GetDomainTree(userID, result.Session.DomainID)
	title := result.Session.NodeKey
	if tree != nil {
		title = domain.NodeTitle(tree, result.Session.NodeKey)
	}
	if result.Resumed {
		return HandleResult{Replies: []string{fmt.Sprintf("继续学习「%s」（阶段：%s）。直接回复即可。", title, result.Session.Phase)}}
	}
	return HandleResult{Replies: []string{fmt.Sprintf("已进入下一节「%s」。\n\n%s", title, result.Content)}}
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
	case text == "下一节" || text == "下一章" || lower == "next":
		return "next", ""
	default:
		return "", text
	}
}

func helpText() string {
	return `Regulus 学习教练

可直接用自然语言，例如：
· 我的课程 — 查看课表
· 学 Go 并发 / 学习 1 — 查看节点
· 节点 1 / 第一个节点 — 开始学习
· 接着学 — 续上次进度
· 进度 — 查看完成情况
· 下一节 — 当前节点完成后进入下一节点

也支持精确命令：课程、学习、节点、继续、进度、下一节、帮助。
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

// IsCoachMessage 是否为进行中的 Coach 自由对话（需即时反馈）
func (r *Router) IsCoachMessage(ev MessageEvent) bool {
	text := strings.TrimSpace(ev.Text)
	if text == "" || !PlatformUserAllowed(ev.Platform, ev.PlatformUserID) {
		return false
	}
	binding, _ := r.store.GetChannelBinding(ev.Platform, ev.PlatformUserID)
	if binding == nil {
		return false
	}
	cmd, _ := parseCommand(text)
	if cmd != "" {
		return false
	}
	navCtx := r.buildNavContext(binding.UserID, ev.Platform)
	if navCtx.HasActiveSession {
		if intent, ok := matchExplicitNavigation(text); ok {
			if intent.Action == NavContinue && r.shouldContinueToCoach(binding.UserID) {
				return true
			}
			return false
		}
	} else if intent, ok := matchNavigationRules(text, navCtx); ok {
		if intent.Action == NavContinue && r.shouldContinueToCoach(binding.UserID) {
			return true
		}
		return false
	}
	sess, err := r.sessions.ActiveSessionForUser(binding.UserID)
	return err == nil && sess != nil
}

// Dispatch 处理消息并通过 adapter 发送回复
func Dispatch(ctx context.Context, router *Router, adapter Adapter, ev MessageEvent) {
	log.Printf("[gateway/%s] 收到: user=%s text=%q", adapter.Name(), ev.PlatformUserID, truncate(ev.Text, 80))
	target := ReplyFromEvent(ev)

	isCoach := router.IsCoachMessage(ev)
	if isCoach {
		Deliver(ctx, adapter, target, []string{"思考中…"})
	}

	result := router.Handle(ctx, ev)
	all := append(result.InstantReplies, result.Replies...)
	if len(all) == 0 {
		if !isCoach {
			log.Printf("[gateway/%s] 无回复", adapter.Name())
		}
		return
	}
	Deliver(ctx, adapter, target, all)
}

func isBindCode(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}
