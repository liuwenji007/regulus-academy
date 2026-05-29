.PHONY: dev backend frontend test build docker

dev:
	@echo "请开两个终端分别运行："
	@echo "  make backend"
	@echo "  make frontend"

backend:
	go run ./cmd/server

frontend:
	cd web && pnpm install && pnpm dev

test:
	GOPROXY=https://goproxy.cn,direct go test ./...

build: frontend-build
	GOPROXY=https://goproxy.cn,direct go build -o bin/server ./cmd/server

frontend-build:
	cd web && pnpm install && pnpm build

docker:
	docker compose up --build
