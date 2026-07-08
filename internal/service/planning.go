package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// PlanningService 行动助手规划服务
type PlanningService struct {
	store   *storage.Store
	planner *agent.Planner
	llm     atomic.Value
	lock    sync.Map // sessionID -> *sync.Mutex
}

// NewPlanningService 创建规划服务
func NewPlanningService(store *storage.Store, planner *agent.Planner, llmClient llm.Provider) *PlanningService {
	s := &PlanningService{store: store, planner: planner}
	s.llm.Store(llmClient)
	return s
}

func (s *PlanningService) llmClient() llm.Provider {
	if v := s.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
}

// SetLLM 热更新 LLM
func (s *PlanningService) SetLLM(client llm.Provider) {
	if client != nil {
		s.llm.Store(client)
	}
}

// StartPlanningResult 开始或恢复规划会话
type StartPlanningResult struct {
	Session  *storage.PlanningSession
	Content  string
	Resumed  bool
	Plan     *agent.PlanningResult
	Messages []storage.PlanningMessage
}

// StartOrResume 开始新规划或恢复 active 会话
func (s *PlanningService) StartOrResume(ctx context.Context, userID string, forceNew bool) (*StartPlanningResult, error) {
	if !s.llmClient().Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	if userID == "" {
		return nil, fmt.Errorf("请先选择学习角色")
	}

	if !forceNew {
		if existing, err := s.store.FindActivePlanningSession(userID); err != nil {
			return nil, err
		} else if existing != nil {
			msgs, err := s.store.ListPlanningMessages(existing.ID)
			if err != nil {
				return nil, err
			}
			plan, _ := agent.ParsePlanningResult(existing.PlanJSON)
			return &StartPlanningResult{
				Session: existing, Resumed: true, Plan: plan, Messages: msgs,
			}, nil
		}
	}

	sess, err := s.store.CreatePlanningSession(userID, "intake")
	if err != nil {
		return nil, err
	}
	_ = s.store.ArchiveOtherPlanningSessions(userID, sess.ID)

	opening := storage.PlanningOpenMessage
	if _, err := s.store.AddPlanningMessage(sess.ID, "assistant", opening); err != nil {
		return nil, err
	}
	msgs, err := s.store.ListPlanningMessages(sess.ID)
	if err != nil {
		return nil, err
	}
	return &StartPlanningResult{
		Session: sess, Content: opening, Resumed: false, Messages: msgs,
	}, nil
}

// PlanningMessageResult 发送消息结果
type PlanningMessageResult struct {
	Session  *storage.PlanningSession
	Reply    string
	Phase    string
	Plan     *agent.PlanningResult
	Synthesized bool
}

// SendPlanningMessage 处理用户消息
func (s *PlanningService) SendPlanningMessage(ctx context.Context, userID, sessionID, content string) (*PlanningMessageResult, error) {
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

	sess, err := s.store.GetPlanningSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, fmt.Errorf("无权访问此会话")
	}
	if sess.Status != "active" {
		return nil, fmt.Errorf("规划会话已结束")
	}

	msgs, err := s.store.ListPlanningMessages(sessionID)
	if err != nil {
		return nil, err
	}
	history := agent.HistoryFromPlanningMessages(msgs)

	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	runCtx, endTrace := observability.Trace(runCtx, observability.TraceMeta{
		Name: "planning.message", UserID: userID, SessionID: sessionID, Phase: sess.Phase,
		Input: content,
	})
	defer endTrace()

	existingPlan, _ := agent.ParsePlanningResult(sess.PlanJSON)
	shouldSynth := agent.WantsExplicitPlan(content) ||
		(sess.Phase == "plan_ready" && agent.WantsPlanUpdate(content))

	var reply string
	var plan *agent.PlanningResult
	synthesized := false

	if shouldSynth {
		hist := append(history, llm.Message{Role: "user", Content: content})
		refine := ""
		if sess.Phase == "plan_ready" {
			refine = content
		}
		plan, reply, err = s.planner.Synthesize(runCtx, userID, hist, existingPlan, refine)
		if err != nil {
			return nil, err
		}
		synthesized = true
		sess.Phase = "plan_ready"
		planJSON, err := agent.MarshalPlanningResult(plan)
		if err != nil {
			return nil, err
		}
		sess.PlanJSON = planJSON
	} else if sess.Phase == "plan_ready" {
		reply, err = s.planner.PlanReadyChat(runCtx, userID, history, content, existingPlan)
		if err != nil {
			return nil, err
		}
		plan = existingPlan
	} else {
		turn, err := s.planner.IntakeTurn(runCtx, userID, history, content)
		if err != nil {
			return nil, err
		}
		reply = turn.Reply
		if turn.ReadyToPlan {
			plan, synthReply, err := s.planner.Synthesize(runCtx, userID, append(history, llm.Message{Role: "user", Content: content}), nil, "")
			if err != nil {
				return nil, err
			}
			synthesized = true
			sess.Phase = "plan_ready"
			planJSON, err := agent.MarshalPlanningResult(plan)
			if err != nil {
				return nil, err
			}
			sess.PlanJSON = planJSON
			reply = synthReply
		}
	}

	if _, err := s.store.AddPlanningMessage(sessionID, "user", content); err != nil {
		return nil, err
	}
	if _, err := s.store.AddPlanningMessage(sessionID, "assistant", reply); err != nil {
		return nil, err
	}
	if err := s.store.UpdatePlanningSession(sess); err != nil {
		return nil, err
	}

	return &PlanningMessageResult{
		Session: sess, Reply: reply, Phase: sess.Phase, Plan: plan, Synthesized: synthesized,
	}, nil
}

func (s *PlanningService) lockForSession(sessionID string) *sync.Mutex {
	v, _ := s.lock.LoadOrStore(sessionID, &sync.Mutex{})
	return v.(*sync.Mutex)
}
