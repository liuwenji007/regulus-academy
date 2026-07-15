package agent

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const (
	profileRefreshTimeout     = 60 * time.Second
	maxProfileTranscriptRunes = 8000
	minProfileUserMessages    = 1
	maxDomainSummaryRunes     = 200
)

// ProfileRefreshOutput 节末按课摘要（仅 domain_summary 落库）
type ProfileRefreshOutput struct {
	DomainSummary string `json:"domain_summary"`
}

// ProfileGlobalOutput 全局画像结构化输出（init/merge）
type ProfileGlobalOutput struct {
	Background  string `json:"background"`
	Goal        string `json:"goal"`
	Preference  string `json:"preference,omitempty"`
	Summary     string `json:"summary,omitempty"` // legacy，忽略
}

func (c *Coach) scheduleProfileRefresh(sess *storage.Session, sctx *storage.SessionContext) {
	if sess == nil || c == nil {
		return
	}
	sessionID := sess.ID
	userID := sess.UserID
	var ctxCopy storage.SessionContext
	if sctx != nil {
		ctxCopy = *sctx
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), profileRefreshTimeout)
		defer cancel()
		ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
			Name:      "coach.profile_refresh",
			UserID:    userID,
			SessionID: sessionID,
		})
		defer endTrace()
		current, err := c.store.GetSession(sessionID)
		if err != nil || current == nil {
			return
		}
		if current.UserID != userID {
			return
		}
		_ = c.RefreshUserProfileAfterNode(ctx, current, &ctxCopy)
	}()
}

// RefreshUserProfileAfterNode 节点点亮后更新本课摘要；不写全局 profile_summary。
func (c *Coach) RefreshUserProfileAfterNode(ctx context.Context, sess *storage.Session, sctx *storage.SessionContext) error {
	if c == nil || sess == nil || !c.llmClient(ctx).Configured() {
		return nil
	}
	msgs, err := c.store.ListMessages(sess.ID)
	if err != nil {
		return err
	}
	userTurns := 0
	for _, m := range msgs {
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			userTurns++
		}
	}
	if userTurns < minProfileUserMessages {
		return nil
	}
	transcript := formatTranscriptForProfile(msgs)
	if strings.TrimSpace(transcript) == "" {
		return nil
	}

	in, err := c.buildProfileRefreshInput(sess, sctx, transcript)
	if err != nil {
		return err
	}
	schema, _ := domain.LoadSchema("profile_refresh.json")
	msgsLLM := c.prompter.BuildMessages(in, TaskProfileRefresh, schema)
	ctx = observability.WithGeneration(ctx, TaskProfileRefresh.GenerationName())

	var out ProfileRefreshOutput
	if err := c.llmClient(ctx).ChatJSON(ctx, msgsLLM, 0.2, &out); err != nil {
		return err
	}
	summary := strings.TrimSpace(out.DomainSummary)
	if summary == "" {
		return nil
	}
	if utf8.RuneCountInString(summary) > maxDomainSummaryRunes {
		summary = truncateRunes(summary, maxDomainSummaryRunes)
	}
	completedAt := time.Now().UTC()
	return c.store.UpsertDomainProfile(sess.UserID, sess.DomainID, summary, completedAt)
}

func (c *Coach) buildProfileRefreshInput(
	sess *storage.Session,
	sctx *storage.SessionContext,
	transcript string,
) (PromptInput, error) {
	slug := sess.DomainSlug
	node, err := c.registry.GetNode(c.store, sess.DomainID, slug, sess.NodeKey)
	if err != nil {
		return PromptInput{}, err
	}
	tree, _ := c.store.GetDomainTree(sess.UserID, sess.DomainID)
	domainName := "课程"
	if tree != nil {
		domainName = tree.DomainName
	}
	existing := ""
	if dp, err := c.store.GetDomainProfile(sess.UserID, sess.DomainID); err == nil && dp != nil {
		existing = strings.TrimSpace(dp.Summary)
	}
	global, _ := c.store.ComposeForBuild(sess.UserID)
	recent := []string(nil)
	if sctx != nil {
		recent = sctx.RecentMistakes
	}
	task := "请根据【本节对话摘录】与【本课已有摘要】输出本课 domain_summary；勿写全局背景或他课内容。"
	var userMsg strings.Builder
	if existing != "" {
		userMsg.WriteString("【本课已有摘要】\n")
		userMsg.WriteString(existing)
		userMsg.WriteString("\n\n")
	}
	userMsg.WriteString("【本节对话摘录】\n")
	userMsg.WriteString(transcript)
	return PromptInput{
		DomainName:      domainName,
		Node:            node,
		NodeKey:         sess.NodeKey,
		Layer:           node.Layer,
		Phase:           "completed",
		TaskInstruction: task,
		UserMessage:     userMsg.String(),
		RecentMistakes:  recent,
		UserProfile:     global,
	}, nil
}

func formatTranscriptForProfile(msgs []storage.SessionMessage) string {
	var b strings.Builder
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		line := strings.TrimSpace(m.Content)
		if line == "" {
			continue
		}
		role := "用户"
		if m.Role == "assistant" {
			role = "教练"
		}
		chunk := role + "：" + line + "\n"
		if utf8.RuneCountInString(b.String())+utf8.RuneCountInString(chunk) > maxProfileTranscriptRunes {
			break
		}
		b.WriteString(chunk)
	}
	return strings.TrimSpace(b.String())
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
