# 多阶段构建 Dockerfile for Geofence-Updater-Lite

# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git make protobuf

# 创建工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY scripts/ ./scripts/
COPY test/ ./test/

# 构建 publisher 工具
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o publisher ./cmd/publisher

# 构建 SDK 示例
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sdk-example ./cmd/sdk-example

# 第二阶段：运行时阶段
FROM alpine:latest

# 安装 ca-certificates（用于 HTTPS）
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 -S 1000 guluser && \
    adduser -u 1000 -G guluser -G 1000 guluser && \
    mkdir -p /home/guluser && \
    chown -R guluser:guluser /home/guluser

# 切换到非 root 用户
USER guluser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/publisher /usr/local/bin/publisher
COPY --from=builder /app/sdk-example /usr/local/bin/sdk-example

# 暴露端口（如果需要）
# EXPOSE 8080

# 默认命令
CMD ["publisher"]
