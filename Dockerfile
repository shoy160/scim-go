# 阶段1：构建Go可执行文件
FROM registry.cn-hangzhou.aliyuncs.com/shay/golang:alpine AS builder
WORKDIR /app
# 复制依赖文件并下载依赖（利用 Docker 缓存）
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
# 复制源代码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o scim-server .

# 阶段2：生产镜像（使用 Alpine 更轻量且包含必要工具）
FROM alpine:3.19 AS production
# 安装必要的工具
RUN apk --no-cache add bash curl wget && rm -rf /var/cache/apk/*

    # 设置工作目录
WORKDIR /app
# 创建非 root 用户
RUN addgroup -g 1000 nonroot && \
    adduser -u 1000 -G nonroot -s /bin/bash -D nonroot
# 复制可执行文件
COPY --from=builder /app/scim-server /app/scim-server
# 复制配置文件
COPY --from=builder /app/config.yaml /app/config.yaml

# 暴露端口
EXPOSE 8080
# 使用非 root 用户运行
USER nonroot:nonroot
# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["wget", "--spider", "-q", "http://localhost:8080/health"] || exit 1
# 启动命令
ENTRYPOINT ["/app/scim-server"]