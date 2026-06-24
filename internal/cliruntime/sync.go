package cliruntime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// SyncProgressItem 按 slug 同步的进度条目。
type SyncProgressItem struct {
	Slug      string  `json:"slug"`
	NodeKey   string  `json:"nodeKey"`
	Layer     string  `json:"layer"`
	Status    string  `json:"status"`
	Mastery   float64 `json:"mastery"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// remoteProgressWins 远程进度是否应覆盖本地（无本地记录，或远程时间戳更新且可解析）。
func remoteProgressWins(local *storage.UserProgress, remoteUpdatedAt string) bool {
	if local == nil {
		return true
	}
	remoteAt, err := time.Parse(time.RFC3339, remoteUpdatedAt)
	if err != nil || remoteAt.IsZero() {
		return false
	}
	return local.UpdatedAt.Before(remoteAt.UTC())
}

// Link 保存远程关联并探测连通性。
func (rt *Runtime) Link(apiURL, userID string) error {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return fmt.Errorf("apiUrl 不能为空")
	}
	if userID == "" {
		userID = rt.UserID()
	}
	link := LinkConfig{APIURL: apiURL, UserID: userID}
	if err := rt.SaveLink(link); err != nil {
		return err
	}
	rt.link = &link
	rt.remote = NewRemoteClient(apiURL, userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !rt.remote.Online(ctx) {
		return fmt.Errorf("已保存 link，但无法连接 %s/health", apiURL)
	}
	return nil
}

// SyncPull 从远程拉取进度合并到本地。
func (rt *Runtime) SyncPull(ctx context.Context) (merged int, err error) {
	if !rt.Linked() {
		return 0, fmt.Errorf("未关联远程，请先 regulus link")
	}
	if !rt.remote.Online(ctx) {
		return 0, fmt.Errorf("远程不可达: %s", rt.link.APIURL)
	}
	var remoteDomains struct {
		Domains []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"domains"`
	}
	if err := rt.remote.doJSON(ctx, "GET", "/api/domains", nil, &remoteDomains); err != nil {
		return 0, err
	}
	slugByRemoteID := map[string]string{}
	for _, d := range remoteDomains.Domains {
		if d.Slug != "" {
			slugByRemoteID[d.ID] = d.Slug
			if _, _, e := rt.EnsureDomainFromSlug(d.Slug); e != nil {
				// 远程有课但本地 domains/ 无文件时跳过注册
				continue
			}
		}
	}
	remoteRows, err := rt.remote.FetchRemoteProgress(ctx, "")
	if err != nil {
		return 0, err
	}
	for _, row := range remoteRows {
		slug := slugByRemoteID[row.DomainID]
		if slug == "" {
			continue
		}
		localDom, _, err := rt.EnsureDomainFromSlug(slug)
		if err != nil {
			continue
		}
		localList, _ := rt.store.ListProgress(rt.UserID(), localDom.ID)
		var local *storage.UserProgress
		for i := range localList {
			if localList[i].NodeKey == row.NodeKey {
				local = &localList[i]
				break
			}
		}
		if !remoteProgressWins(local, row.UpdatedAt) {
			continue
		}
		var updatedAt time.Time
		if t, err := time.Parse(time.RFC3339, row.UpdatedAt); err == nil {
			updatedAt = t.UTC()
		}
		_ = rt.store.UpsertProgress(storage.UserProgress{
			UserID:    rt.UserID(),
			DomainID:  localDom.ID,
			NodeKey:   row.NodeKey,
			Layer:     row.Layer,
			Status:    row.Status,
			Mastery:   row.Mastery,
			UpdatedAt: updatedAt,
		})
		merged++
	}
	return merged, nil
}

// SyncPush 将本地进度推送到远程。
func (rt *Runtime) SyncPush(ctx context.Context) (merged int, err error) {
	if !rt.Linked() {
		return 0, fmt.Errorf("未关联远程，请先 regulus link")
	}
	if !rt.remote.Online(ctx) {
		return 0, fmt.Errorf("远程不可达: %s", rt.link.APIURL)
	}
	localRows, err := rt.ListProgress("")
	if err != nil {
		return 0, err
	}
	items := make([]SyncProgressItem, 0, len(localRows))
	for _, row := range localRows {
		if row.Slug == "" {
			continue
		}
		items = append(items, SyncProgressItem{
			Slug:      row.Slug,
			NodeKey:   row.NodeKey,
			Layer:     row.Layer,
			Status:    row.Status,
			Mastery:   row.Mastery,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return rt.remote.PushProgress(ctx, items)
}

// DoctorReport 环境自检。
type DoctorReport struct {
	CoachRoot     string `json:"coachRoot"`
	DataDir       string `json:"dataDir"`
	LLMConfigured bool   `json:"llmConfigured"`
	Linked        bool   `json:"linked"`
	RemoteOnline  bool   `json:"remoteOnline,omitempty"`
	RemoteURL     string `json:"remoteUrl,omitempty"`
	UserID        string `json:"userId"`
	DomainCount   int    `json:"domainCount"`
}

// Doctor 运行自检。
func (rt *Runtime) Doctor(ctx context.Context) DoctorReport {
	report := DoctorReport{
		CoachRoot:     rt.paths.CoachRoot,
		DataDir:       rt.paths.DataDir,
		LLMConfigured: rt.LLMConfigured(),
		Linked:        rt.Linked(),
		UserID:        rt.UserID(),
	}
	if list, err := rt.registry.ListDomains(); err == nil {
		report.DomainCount = len(list)
	}
	if rt.Linked() {
		report.RemoteURL = rt.link.APIURL
		report.RemoteOnline = rt.remote.Online(ctx)
	}
	return report
}

// BuildDomain 本地 LLM 建课（与 cmd 原逻辑相同）。
func (rt *Runtime) BuildDomain(ctx context.Context, topic string, force bool, profile string) error {
	if !rt.LLMConfigured() {
		return fmt.Errorf("未配置 LLM API Key")
	}
	reg := rt.registry
	intent, err := reg.ParseIntent(ctx, rt.llm, topic)
	if err != nil {
		return err
	}
	outParent := rt.paths.DomainsDir

	if intent.Source == domain.SourceSkillPack && !force {
		existing := fmt.Sprintf("%s/%s", outParent, intent.Slug)
		if st, statErr := os.Stat(existing); statErr == nil && st.IsDir() {
			fmt.Printf("已有内置 Domain: %s\n", existing)
			fmt.Println("无需建课。若要重新生成，请加 --force")
			return nil
		}
		fmt.Printf("主题「%s」匹配内置 Skill（slug=%s），目录: domains/%s/\n", intent.DisplayName, intent.Slug, intent.Slug)
		fmt.Println("若要 LLM 重新生成，请加 --force")
		return nil
	}
	if intent.Source == domain.SourceSkillPack && force {
		intent.Source = domain.SourceGenerated
	}

	builder := domain.NewTreeBuilder(reg)
	tree, nodes, err := builder.Build(ctx, rt.llm, intent, topic, profile)
	if err != nil {
		return err
	}
	files, err := domain.ExportToFiles(tree, intent.Slug, "", "", 1, nodes)
	if err != nil {
		return err
	}
	dest, err := domain.WriteDomainFiles(outParent, intent.Slug, files)
	if err != nil {
		return err
	}
	n := 0
	for _, ly := range tree.Layers {
		n += len(ly.Nodes)
	}
	fmt.Printf("建课完成: %s（slug=%s，%d 个节点）\n", intent.DisplayName, intent.Slug, n)
	fmt.Printf("已写入: %s\n", dest)
	_, _, _ = rt.EnsureDomainFromSlug(intent.Slug)
	return nil
}
