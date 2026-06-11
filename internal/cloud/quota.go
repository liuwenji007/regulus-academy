package cloud

import (
	"errors"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// ErrQuotaExceeded 平台日配额用尽且未配置 BYOK
var ErrQuotaExceeded = errors.New("quota_exceeded")

// QuotaStatus 用户当日 LLM 配额状态
type QuotaStatus struct {
	Used           int  `json:"used"`
	Limit          int  `json:"limit"`
	Remaining      int  `json:"remaining"`
	HasBYOK        bool `json:"hasByok"`
	PromptTokens   int  `json:"promptTokensToday"`
	CompletionTokens int `json:"completionTokensToday"`
}

// Service Cloud 运行时服务
type Service struct {
	cfg            Config
	store          *storage.Store
	buildLimiter   *BuildLimiter
	rateLimiter    *RateLimiter
	lastSeen       *LastSeenThrottler
	platformLLM    llm.Provider
}

func NewService(cfg Config, store *storage.Store, platformLLM llm.Provider) *Service {
	s := &Service{cfg: cfg, store: store, platformLLM: platformLLM}
	if cfg.Enabled() {
		s.buildLimiter = NewBuildLimiter(cfg.MaxBuildJobsGlobal)
		s.rateLimiter = NewRateLimiter(cfg.RateLimitPerIP)
		s.lastSeen = NewLastSeenThrottler()
	}
	return s
}

func (s *Service) TouchLastSeen(userID string) {
	if !s.cfg.Enabled() || s.lastSeen == nil {
		return
	}
	if !s.lastSeen.ShouldTouch(userID) {
		return
	}
	_ = s.store.TouchUserLastSeen(userID)
}

func (s *Service) Config() Config { return s.cfg }

func (s *Service) BuildLimiter() *BuildLimiter { return s.buildLimiter }

func (s *Service) RateLimiter() *RateLimiter { return s.rateLimiter }

func (s *Service) ValidateUserID(userID string) error {
	if !s.cfg.Enabled() {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == storage.DefaultUserID {
		return fmt.Errorf("需要有效的学习角色，请先创建或选择角色")
	}
	return nil
}

func (s *Service) QuotaStatus(userID string) (QuotaStatus, error) {
	if !s.cfg.Enabled() {
		return QuotaStatus{}, nil
	}
	hasBYOK, _ := s.store.HasUserLLMCredentials(userID)
	usage, err := s.store.GetLLMUsageDaily(userID, storage.TodayUTC())
	if err != nil {
		return QuotaStatus{}, err
	}
	limit := s.cfg.QuotaDailyMessages
	remaining := limit - usage.MessageCount
	if remaining < 0 {
		remaining = 0
	}
	return QuotaStatus{
		Used:             usage.MessageCount,
		Limit:            limit,
		Remaining:        remaining,
		HasBYOK:          hasBYOK,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
	}, nil
}

func (s *Service) CheckMessageQuota(userID string) error {
	if !s.cfg.Enabled() {
		return nil
	}
	hasBYOK, err := s.store.HasUserLLMCredentials(userID)
	if err != nil {
		return err
	}
	if hasBYOK {
		return nil
	}
	usage, err := s.store.GetLLMUsageDaily(userID, storage.TodayUTC())
	if err != nil {
		return err
	}
	if usage.MessageCount >= s.cfg.QuotaDailyMessages {
		return ErrQuotaExceeded
	}
	return nil
}

func (s *Service) RecordMessageUsage(userID string) error {
	if !s.cfg.Enabled() {
		return nil
	}
	return s.store.IncrementLLMMessageCount(userID, storage.TodayUTC())
}

func (s *Service) RecordTokenUsage(userID, callKind, billedTo string, prompt, completion, total int) error {
	if !s.cfg.Enabled() {
		return nil
	}
	date := storage.TodayUTC()
	if err := s.store.AddLLMTokenUsage(userID, date, callKind, billedTo, prompt, completion, total); err != nil {
		return err
	}
	return s.store.AddLLMUsageDailyTokens(userID, date, prompt, completion)
}
