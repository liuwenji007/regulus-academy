package cliruntime

import (
	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Runtime 便携 Skill 运行时（本地 SQLite + Coach）。
type Runtime struct {
	paths    Paths
	config   Config
	link     *LinkConfig
	store    *storage.Store
	registry *domain.Registry
	coach    *agent.Coach
	sessions *service.SessionService
	llm      llm.Provider
	remote   *RemoteClient
}

// Open 打开或初始化本地运行时。
func Open(coachRootFlag string) (*Runtime, error) {
	paths, err := ResolvePaths(coachRootFlag)
	if err != nil {
		return nil, err
	}
	LoadDotEnv(paths.CoachRoot, paths.DataDir)

	cfg := config.Load()
	client := llm.NewFromConfig(cfg.LLM)

	store, err := storage.Open(paths.DBPath)
	if err != nil {
		return nil, err
	}

	rt := &Runtime{
		paths:    paths,
		llm:      client,
		store:    store,
		registry: domain.NewRegistry(),
	}
	if err := rt.loadConfig(); err != nil {
		store.Close()
		return nil, err
	}
	if err := rt.loadLink(); err != nil {
		store.Close()
		return nil, err
	}
	if err := store.EnsureUser(rt.UserID()); err != nil {
		store.Close()
		return nil, err
	}

	coach, err := agent.NewCoach(store, client)
	if err != nil {
		store.Close()
		return nil, err
	}
	rt.coach = coach
	rt.sessions = service.NewSessionService(store, coach, client)
	if rt.Linked() {
		rt.remote = NewRemoteClient(rt.link.APIURL, rt.UserID())
	}
	return rt, nil
}

// Close 关闭数据库。
func (rt *Runtime) Close() error {
	if rt.store == nil {
		return nil
	}
	return rt.store.Close()
}

// Paths 返回路径布局。
func (rt *Runtime) Paths() Paths { return rt.paths }

// LLMConfigured 是否已配置模型 Key。
func (rt *Runtime) LLMConfigured() bool {
	return rt.llm != nil && rt.llm.Configured()
}

// Store 本地库（测试或扩展用）。
func (rt *Runtime) Store() *storage.Store { return rt.store }

// Registry 知识域注册表。
func (rt *Runtime) Registry() *domain.Registry { return rt.registry }

// Sessions 教练会话服务（本地模式）。
func (rt *Runtime) Sessions() *service.SessionService { return rt.sessions }

// Remote 远程 API 客户端（已 link 时）。
func (rt *Runtime) Remote() *RemoteClient { return rt.remote }

// CoachRoot 目录。
func (rt *Runtime) CoachRoot() string { return rt.paths.CoachRoot }
