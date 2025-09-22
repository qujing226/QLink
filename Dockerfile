# 使用官方Go镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o qlink-node ./cmd/qlink-node
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o qlink-cli ./cmd/qlink-cli

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest

# 安装ca证书和时区数据
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 qlink && \
    adduser -D -s /bin/sh -u 1001 -G qlink qlink

# 设置工作目录
WORKDIR /home/qlink

# 从构建阶段复制二进制文件
COPY --from=builder /app/qlink-node .
COPY --from=builder /app/qlink-cli .

# 创建配置和数据目录
RUN mkdir -p config data logs && \
    chown -R qlink:qlink /home/qlink

# 复制配置文件
COPY config.yaml ./config/

# 切换到非root用户
USER qlink

# 暴露端口
EXPOSE 8080 8081 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./qlink-cli health || exit 1

# 启动命令
CMD ["./qlink-node", "--config", "./config/config.yaml"]