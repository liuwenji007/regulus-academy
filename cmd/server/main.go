package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/api"
	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

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

	var staticHandler http.Handler
	if _, err := os.Stat("web/dist"); err == nil {
		staticHandler = spaHandler(http.Dir("web/dist"))
	}

	server := api.NewServer(handler, staticHandler)
	log.Printf("Regulus Academy 服务启动于 http://localhost%s（LLM: %s / %s）", cfg.Addr(), llmClient.Name(), llmClient.Model())
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
