FROM node:22-alpine AS web
WORKDIR /app/web
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY web/package.json ./
RUN pnpm install
COPY web/ ./
RUN pnpm build

FROM golang:1.23-alpine AS api
WORKDIR /app
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
