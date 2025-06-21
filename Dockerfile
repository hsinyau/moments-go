# 使用官方 Go 镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o moments-go ./cmd

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装 ca-certificates，用于 HTTPS 请求
RUN apk --no-cache add ca-certificates

# 创建非 root 用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/moments-go .

# 复制环境变量示例文件
COPY env.example .env.example

# 更改文件所有者
RUN chown -R appuser:appgroup /app

# 切换到非 root 用户
USER appuser

# 暴露端口（如果需要的话）
EXPOSE 8080

# 运行应用
CMD ["./moments-go"] 
