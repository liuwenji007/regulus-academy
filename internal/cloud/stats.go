package cloud

import (
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// PublicStats 公开共学统计
type PublicStats struct {
	TotalLearners      int    `json:"totalLearners"`
	ActiveLast7Days    int    `json:"activeLast7Days"`
	PlatformTokensToday int   `json:"platformTokensToday"`
	AsOf               string `json:"asOf"`
}

// AdminStats 管理员总览
type AdminStats struct {
	PublicStats
	NewUsersToday      int `json:"newUsersToday"`
	PlatformTokensTotal int `json:"platformTokensTotal"`
	RunningBuildJobs   int `json:"runningBuildJobs"`
}

func (s *Service) PublicStats() (PublicStats, error) {
	total, active, err := s.store.CountLearners(storage.DaysAgoUTC(7))
	if err != nil {
		return PublicStats{}, err
	}
	tokensToday, err := s.store.SumPlatformTokens(storage.TodayUTC())
	if err != nil {
		return PublicStats{}, err
	}
	return PublicStats{
		TotalLearners:       total,
		ActiveLast7Days:     active,
		PlatformTokensToday: tokensToday,
		AsOf:                time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) AdminStats() (AdminStats, error) {
	pub, err := s.PublicStats()
	if err != nil {
		return AdminStats{}, err
	}
	newToday, err := s.store.CountUsersCreatedOn(storage.TodayUTC())
	if err != nil {
		return AdminStats{}, err
	}
	totalTokens, err := s.store.SumPlatformTokensTotal()
	if err != nil {
		return AdminStats{}, err
	}
	running, err := s.store.CountRunningBuildJobs()
	if err != nil {
		return AdminStats{}, err
	}
	return AdminStats{
		PublicStats:         pub,
		NewUsersToday:       newToday,
		PlatformTokensTotal: totalTokens,
		RunningBuildJobs:    running,
	}, nil
}
