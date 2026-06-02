package channel

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// Gateway IM 消息网关
type Gateway struct {
	cfg      config.GatewayConfig
	router   *Router
	adapters []Adapter
	wecom    *WeComWebhook
	feishu   *FeishuWebhook
}

// NewGateway 创建 Gateway
func NewGateway(store *storage.Store, sessions *service.SessionService, cfg config.GatewayConfig, llmClient llm.Provider) *Gateway {
	router := NewRouter(store, sessions, llmClient)
	g := &Gateway{cfg: cfg, router: router}

	if cfg.Telegram.Enabled && cfg.Telegram.BotToken != "" {
		g.adapters = append(g.adapters, NewTelegramAdapter(cfg.Telegram))
	}
	if cfg.DingTalk.Enabled && cfg.DingTalk.ClientID != "" && cfg.DingTalk.ClientSecret != "" {
		g.adapters = append(g.adapters, NewDingTalkAdapter(cfg.DingTalk))
	}
	if cfg.Feishu.Enabled && cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		switch cfg.Feishu.Mode {
		case "webhook":
			g.feishu = NewFeishuWebhook(cfg.Feishu, router)
		default:
			g.adapters = append(g.adapters, NewFeishuAdapter(cfg.Feishu))
		}
	}
	if cfg.WeCom.Enabled && cfg.WeCom.Token != "" && cfg.WeCom.EncodingAESKey != "" {
		g.wecom = NewWeComWebhook(cfg.WeCom, router)
	}

	return g
}

// RegisterWebhooks 注册 HTTP webhook 路由（企微、飞书）
func (g *Gateway) RegisterWebhooks(mux *http.ServeMux) {
	if g.wecom != nil {
		mux.HandleFunc("GET /webhook/wecom", g.wecom.Verify)
		mux.HandleFunc("POST /webhook/wecom", g.wecom.HandleMessage)
	}
	if g.feishu != nil {
		mux.HandleFunc("POST /webhook/feishu", g.feishu.Handle)
	}
}

// Start 启动全部 adapter（阻塞直到 ctx 取消）
func (g *Gateway) Start(ctx context.Context) {
	if !g.cfg.Enabled {
		return
	}
	if len(g.adapters) == 0 && g.wecom == nil && g.feishu == nil {
		log.Println("[gateway] 已启用但未配置任何平台凭证")
		return
	}

	var wg sync.WaitGroup
	onMessage := func(adapter Adapter) func(MessageEvent) {
		return func(ev MessageEvent) {
			runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			Dispatch(runCtx, g.router, adapter, ev)
		}
	}

	for _, a := range g.adapters {
		wg.Add(1)
		go func(ad Adapter) {
			defer wg.Done()
			handler := onMessage(ad)
			for ctx.Err() == nil {
				log.Printf("[gateway] 启动 %s adapter", ad.Name())
				if err := ad.Start(ctx, handler); err != nil && ctx.Err() == nil {
					log.Printf("[gateway] %s 退出: %v，5 秒后重试", ad.Name(), err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Second):
					}
					continue
				}
				return
			}
		}(a)
	}

	if g.wecom != nil {
		log.Println("[gateway] 企业微信 webhook 已注册（需公网 HTTPS → /webhook/wecom）")
	}
	if g.feishu != nil {
		log.Println("[gateway] 飞书 webhook 已注册（FEISHU_MODE=webhook → /webhook/feishu）")
	}

	wg.Wait()
}

// Router 返回路由器（测试用）
func (g *Gateway) Router() *Router {
	return g.router
}
