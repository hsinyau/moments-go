#!/bin/bash

# 部署脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境变量
check_env() {
    print_message "检查环境变量..."
    
    if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
        print_error "TELEGRAM_BOT_TOKEN 未设置"
        exit 1
    fi
    
    if [ -z "$TELEGRAM_USER_ID" ]; then
        print_error "TELEGRAM_USER_ID 未设置"
        exit 1
    fi
    
    if [ -z "$GITHUB_SECRET" ]; then
        print_error "GITHUB_SECRET 未设置"
        exit 1
    fi
    
    print_message "环境变量检查通过"
}

# 创建日志目录
create_logs_dir() {
    print_message "创建日志目录..."
    mkdir -p logs
}

# 构建镜像
build_image() {
    print_message "构建 Docker 镜像..."
    docker build -t moments-go:latest .
}

# 停止并删除旧容器
cleanup_old_container() {
    print_message "清理旧容器..."
    docker stop moments-go-bot 2>/dev/null || true
    docker rm moments-go-bot 2>/dev/null || true
}

# 启动新容器
start_container() {
    print_message "启动新容器..."
    docker-compose up -d
}

# 检查容器状态
check_container_status() {
    print_message "检查容器状态..."
    sleep 5
    
    if docker ps | grep -q moments-go-bot; then
        print_message "容器启动成功！"
        docker logs moments-go-bot --tail 20
    else
        print_error "容器启动失败！"
        docker logs moments-go-bot --tail 50
        exit 1
    fi
}

# 主函数
main() {
    print_message "开始部署 Moments-Go..."
    
    check_env
    create_logs_dir
    build_image
    cleanup_old_container
    start_container
    check_container_status
    
    print_message "部署完成！"
}

# 运行主函数
main "$@" 
