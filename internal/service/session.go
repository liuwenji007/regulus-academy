package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// SessionService 教学会话服务（Web 与 IM Channel 共用）
type SessionService struct {
	store       *storage.Store
	coach       *agent.Coach
	llm         atomic.Value // llm.Provider
	sessionLock sync.Map     // sessionID -> *sync.Mutex
}

// NewSessionService 创建会话服务
func NewSessionService(store *storage.Store, coach *agent.Coach, llmClient llm.Provider) *SessionService {
	s := &SessionService{store: store, coach: coach}
	s.llm.Store(llmClient)
	return s
}

func (s *SessionService) llmClient() llm.Provider {
	if v := s.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
}

// SetLLM 热更新 LLM 客户端
func (s *SessionService) SetLLM(client llm.Provider) {
	if client != nil {
		s.llm.Store(client)
	}
}

// StartSessionResult 开始或恢复会话的结果
type StartSessionResult struct {
	Session   *storage.Session
	Content   string
	Resumed   bool
	FirstOpen bool
}

// StartOrResumeSession 开始新会话或恢复已有会话；新会话时调用 Coach.Begin
func (s *SessionService) StartOrResumeSession(ctx context.Context, userID, domainID, nodeKey, layer string) (*StartSessionResult, error) {
	if !s.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	if domainID == "" || nodeKey == "" {
		return nil, fmt.Errorf("domainId 和 nodeKey 不能为空")
	}

	ok, err := s.store.DomainOwnedByUser(userID, domainID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("课程不存在")
	}

	if layer == "" {
		layer = "entry"
	}

	if existing, err := s.store.FindLatestSession(userID, domainID, nodeKey); err == nil && existing != nil {
		return &StartSessionResult{Session: existing, Resumed: true}, nil
	}

	slug, _ := s.store.GetDomainSlug(domainID)

	// 节点已点亮但无会话（如重建后仅迁了进度）：进入复习态，不重新讲解、不降级进度。
	if s.nodeProgressCompleted(userID, domainID, nodeKey) {
		sctx := &storage.SessionContext{DomainSlug: slug}
		sess, err := s.store.CreateSession(userID, domainID, slug, nodeKey, "completed", sctx)
		if err != nil {
			return nil, err
		}
		sess.Status = "completed"
		_ = s.store.UpdateSession(sess)
		return &StartSessionResult{Session: sess, Resumed: false}, nil
	}

	sctx := &storage.SessionContext{DomainSlug: slug}
	sess, err := s.store.CreateSession(userID, domainID, slug, nodeKey, "explain", sctx)
	if err != nil {
		return nil, err
	}

	_ = s.store.UpsertProgress(storage.UserProgress{
		UserID:   userID,
		DomainID: domainID,
		NodeKey:  nodeKey,
		Layer:    layer,
		Status:   "in_progress",
		Mastery:  0,
	})

	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	runCtx, endTrace := observability.Trace(runCtx, observability.TraceMeta{
		Name: "coach.begin", UserID: userID, SessionID: sess.ID,
		DomainID: domainID, NodeKey: nodeKey, Phase: "explain",
	})
	defer endTrace()
	content, err := s.coach.Begin(runCtx, sess)
	if err != nil {
		return nil, err
	}
	_, _ = s.store.AddMessage(sess.ID, "assistant", content)

	return &StartSessionResult{
		Session:   sess,
		Content:   content,
		Resumed:   false,
		FirstOpen: true,
	}, nil
}

// StartNextNode 当前节点已完成后进入下一未完成节点；若该节点已有会话则直接恢复，否则生成开场讲解
func (s *SessionService) StartNextNode(ctx context.Context, userID, completedSessionID string) (*StartSessionResult, error) {
	sess, err := s.store.GetSession(completedSessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, fmt.Errorf("无权访问此会话")
	}
	if sess.Phase != "completed" {
		if !s.nodeProgressCompleted(userID, sess.DomainID, sess.NodeKey) {
			return nil, fmt.Errorf("当前节点尚未完成")
		}
		sess.Phase = "completed"
		sess.Status = "completed"
		_ = s.store.UpdateSession(sess)
	}
	tree, err := s.store.GetDomainTree(userID, sess.DomainID)
	if err != nil || tree == nil {
		return nil, fmt.Errorf("无法加载知识树")
	}
	progress, err := s.store.ListProgress(userID, sess.DomainID)
	if err != nil {
		return nil, fmt.Errorf("读取进度失败")
	}
	completed := domain.CompletedKeysFromProgress(progress)
	nextKey, layer, _, ok := domain.NextUncompletedNodeAfter(tree, sess.NodeKey, completed)
	if !ok {
		return nil, fmt.Errorf("本课程节点已全部完成")
	}

	return s.StartOrResumeSession(ctx, userID, sess.DomainID, nextKey, layer)
}

func (s *SessionService) nodeProgressCompleted(userID, domainID, nodeKey string) bool {
	list, err := s.store.ListProgress(userID, domainID)
	if err != nil {
		return false
	}
	for _, p := range list {
		if p.NodeKey == nodeKey && p.Status == "completed" {
			return true
		}
	}
	return false
}

// SendMessageResult 发送消息结果
type SendMessageResult struct {
	Result  *agent.MessageResult
	Session *storage.Session
}

// SendCoachMessage 向会话发送用户消息并获取 Coach 回复
func (s *SessionService) SendCoachMessage(ctx context.Context, userID, sessionID, content string) (*SendMessageResult, error) {
	if !s.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	content = strings.TrimSpace(content)
	if sessionID == "" || content == "" {
		return nil, fmt.Errorf("sessionId 和 content 不能为空")
	}

	mu := s.lockForSession(sessionID)
	if !mu.TryLock() {
		return nil, ErrSessionBusy
	}
	defer mu.Unlock()

	sess, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, fmt.Errorf("无权访问此会话")
	}

	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	runCtx, endTrace := observability.Trace(runCtx, observability.TraceMeta{
		Name: "coach.message", UserID: userID, SessionID: sessionID,
		DomainID: sess.DomainID, NodeKey: sess.NodeKey, Phase: sess.Phase,
		Input: content,
	})
	defer endTrace()
	result, err := s.coach.HandleMessage(runCtx, sess, content)
	if err != nil {
		return nil, err
	}

	targetSessID := sessionID
	if result.NextSessionID != "" {
		targetSessID = result.NextSessionID
	}
	if _, err := s.store.AddMessage(targetSessID, "user", content); err != nil {
		return nil, err
	}
	if _, err := s.store.AddMessage(targetSessID, "assistant", result.Content); err != nil {
		return nil, err
	}

	sess, err = s.store.GetSession(targetSessID)
	if err != nil {
		return nil, err
	}
	return &SendMessageResult{Result: result, Session: sess}, nil
}

// ActiveSessionForUser 获取用户当前活跃教学会话（基于 channel_active_node）
func (s *SessionService) ActiveSessionForUser(userID string) (*storage.Session, error) {
	active, err := s.store.GetChannelActiveNode(userID)
	if err != nil || active == nil {
		return nil, err
	}
	return s.store.FindLatestSession(userID, active.DomainID, active.NodeKey)
}

func (s *SessionService) lockForSession(sessionID string) *sync.Mutex {
	v, _ := s.sessionLock.LoadOrStore(sessionID, &sync.Mutex{})
	return v.(*sync.Mutex)
}
