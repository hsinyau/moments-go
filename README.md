# Moments-Go

一个基于 Go 语言的 Telegram 机器人，用于将消息、图片和视频发布到 GitHub Issues。

## 功能特性

- 📝 发送文字消息，弹出标签选择按钮
- 📷 发送图片，自动弹出标签选择按钮
- 🎥 发送视频，自动弹出标签选择按钮
- 🏷️ 动态标签管理，从 GitHub 仓库获取
- ⏰ 媒体文件延迟发布（5分钟）
- 🔄 标签缓存和刷新机制
- 🚀 Docker 部署支持

## 快速开始

### 1. 环境要求

- Go 1.19+
- Docker (可选)
- Telegram Bot Token
- GitHub Personal Access Token

### 2. 配置

复制环境变量文件并填写配置：

```bash
cp env.example .env
```

编辑 `.env` 文件：

```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
GITHUB_TOKEN=your_github_token
GITHUB_REPO=your_username/your_repo
AUTHORIZED_USERS=123456789,987654321
WAIT_TIME=300
```

### 3. 运行

#### 本地运行

```bash
go run cmd/main.go
```

#### Docker 运行

```bash
docker-compose up -d
```

## 使用方法

1. **发送文字消息** - 弹出标签选择按钮，选择后立即发布
2. **发送图片/视频** - 弹出标签选择按钮，选择后可继续发送文字更新内容
3. **命令列表**：
   - `/start` - 显示帮助信息
   - `/tags` - 查看所有可用标签
   - `/refresh` - 刷新标签列表

## 网络问题排查

如果遇到 `tls: bad record MAC` 或其他网络连接错误，请按以下步骤排查：

### 1. 运行网络诊断

```bash
./scripts/test_connection.sh
```

### 2. 常见解决方案

#### 网络连接问题
- 检查网络连接是否稳定
- 尝试重启网络连接
- 检查防火墙设置

#### 代理设置
如果使用代理，请设置环境变量：

```bash
export HTTP_PROXY=http://proxy:port
export HTTPS_PROXY=http://proxy:port
```

#### DNS 问题
尝试使用公共 DNS：

```bash
# 临时设置 DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf
```

#### 系统时间问题
确保系统时间正确：

```bash
sudo ntpdate -s time.nist.gov
```

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t moments-go .

# 运行容器
docker run -d --name moments-go --env-file .env moments-go
```

### Docker Compose

```bash
docker-compose up -d
```

### 生产环境部署

使用提供的部署脚本：

```bash
./scripts/deploy.sh
```

## 开发

### 项目结构

```
Moments-Go/
├── cmd/           # 主程序入口
├── config/        # 配置管理
├── github/        # GitHub API 集成
├── handlers/      # 消息处理器
├── telegram/      # Telegram API 集成
├── types/         # 数据类型定义
├── scripts/       # 部署和工具脚本
└── docker-compose.yml
```

### 构建

```bash
go build -o moments-go cmd/main.go
```

### 测试

```bash
go test ./...
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！ 
