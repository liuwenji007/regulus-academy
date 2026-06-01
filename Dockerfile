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
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=api /server /app/server
ENV PORT=8080
ENV DATABASE_PATH=/app/data/regulus.db
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["/app/server"]
