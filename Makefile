.PHONY: dev backend frontend test build docker cli

dev:
	@echo "请开两个终端分别运行："
	@echo "  make backend"
	@echo "  make frontend"

backend:
	go run ./cmd/server

frontend:
	cd web && pnpm install && pnpm dev

coach-embed:
	bash scripts/sync-coach-embed.sh

test: coach-embed cli
	GOPROXY=https://goproxy.cn,direct go test ./...

build: coach-embed frontend-build
	GOPROXY=https://goproxy.cn,direct go build -o bin/server ./cmd/server

cli:
	GOPROXY=https://goproxy.cn,direct go build -o bin/regulus ./cmd/regulus
	mkdir -p regulus-coach/bin
	GOPROXY=https://goproxy.cn,direct go build -o regulus-coach/bin/regulus ./cmd/regulus

frontend-build:
	cd web && pnpm install && pnpm build

docker:
	docker compose up --build
