package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/regulus-academy/regulus-academy/internal/api"
	"github.com/regulus-academy/regulus-academy/internal/channel"
	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/service"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func logGatewayDisabledHint() {
	hasCreds := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) != "" ||
		(strings.TrimSpace(os.Getenv("DINGTALK_CLIENT_ID")) != "" && strings.TrimSpace(os.Getenv("DINGTALK_CLIENT_SECRET")) != "") ||
		(strings.TrimSpace(os.Getenv("FEISHU_APP_ID")) != "" && strings.TrimSpace(os.Getenv("FEISHU_APP_SECRET")) != "") ||
		strings.TrimSpace(os.Getenv("WECOM_TOKEN")) != ""
	if !hasCreds {
		return
	}
	log.Println("[gateway] 警告: 已配置 IM 凭证但 GATEWAY_ENABLED=false，机器人不会启动。请在 Web「IM 频道」开启 Gateway 或设置 GATEWAY_ENABLED=true")
}

func main() {
	cfg := config.Load()

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer store.Close()

	llmClient := llm.NewFromConfig(cfg.LLM)
	handler, err := api.NewHandler(store, llmClient)
	if err != nil {
		log.Fatalf("初始化 API 失败: %v", err)
	}

	sessions := service.NewSessionService(store, handler.Coach(), llmClient)
	gw := channel.NewGateway(store, sessions, cfg.Gateway)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Gateway.Enabled {
		go gw.Start(ctx)
	} else {
		logGatewayDisabledHint()
	}

	var staticHandler http.Handler
	if _, err := os.Stat("web/dist"); err == nil {
		staticHandler = spaHandler(http.Dir("web/dist"))
	}

	server := api.NewServer(handler, staticHandler, gw.RegisterWebhooks)
	log.Printf("Regulus Academy 服务启动于 http://localhost%s（LLM: %s / %s）", cfg.Addr(), llmClient.Name(), llmClient.Model())
	if cfg.Gateway.Enabled {
		log.Println("IM Gateway 已启用（Telegram / 钉钉 / 飞书 / 企微 webhook）")
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	if err := http.ListenAndServe(cfg.Addr(), server); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// spaHandler 为 PWA 提供静态文件，未知路径回退到 index.html
func spaHandler(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != "/" {
			if f, err := root.Open(r.URL.Path[1:]); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
