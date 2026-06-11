FROM node:22-alpine AS web
WORKDIR /app/web
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# go.mod 要求 1.26.3；基础镜像用 1.23 + GOTOOLCHAIN=auto 自动拉取匹配工具链
FROM golang:1.23-alpine AS api
WORKDIR /app
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
RUN cp -a regulus-coach internal/coachstatic/regulus-coach
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=api /server /app/server
# 运行时从磁盘加载：prompts、schemas、domains、triggers（非编译进二进制）
COPY --from=api /app/regulus-coach /app/regulus-coach
COPY --from=api /app/web/dist /app/web/dist
ENV PORT=8080
ENV DATABASE_PATH=/app/data/regulus.db
ENV REGULUS_COACH_ROOT=/app/regulus-coach
EXPOSE 8080
# 数据目录：本地用 docker-compose 挂载 ./data；Railway 在控制台添加 Volume 挂 /app/data（勿写 VOLUME 指令）
CMD ["/app/server"]
