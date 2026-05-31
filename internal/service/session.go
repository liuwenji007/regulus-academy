package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// SessionService 教学会话服务（Web 与 IM Channel 共用）
type SessionService struct {
	store       *storage.Store
	coach       *agent.Coach
	llm         llm.Provider
	sessionLock sync.Map // sessionID -> *sync.Mutex
}

// NewSessionService 创建会话服务
func NewSessionService(store *storage.Store, coach *agent.Coach, llmClient llm.Provider) *SessionService {
	return &SessionService{store: store, coach: coach, llm: llmClient}
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
	if !s.llm.Configured() {
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

// SendMessageResult 发送消息结果
type SendMessageResult struct {
	Result  *agent.MessageResult
	Session *storage.Session
}

// SendCoachMessage 向会话发送用户消息并获取 Coach 回复
func (s *SessionService) SendCoachMessage(ctx context.Context, userID, sessionID, content string) (*SendMessageResult, error) {
	if !s.llm.Configured() {
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

	userRecord, err := s.store.AddMessage(sessionID, "user", content)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := s.coach.HandleMessage(runCtx, sess, content)
	if err != nil {
		_ = s.store.DeleteMessage(userRecord.ID)
		return nil, err
	}

	if _, err := s.store.AddMessage(sessionID, "assistant", result.Content); err != nil {
		return nil, err
	}

	// 重新加载 session（phase 可能已更新）
	sess, _ = s.store.GetSession(sessionID)
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
