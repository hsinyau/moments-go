# Moments-Go

一个用 Golang 实现的 Telegram 机器人，用于自动发布动态到 GitHub Issues。

## 功能特性

- 📱 支持发送图片和视频，自动发布到 GitHub Issues
- 💬 支持纯文字动态发布
- 🔐 用户权限验证，只允许授权用户使用
- 📤 自动上传媒体文件到 GitHub 仓库
- ⏰ 支持延迟发布功能
- 🏷️ 自动添加标签分类
- 🐳 Docker 支持
- 🔄 GitHub Actions 自动构建

## 环境要求

- Go 1.21 或更高版本
- Docker (可选)
- Telegram Bot Token
- GitHub Personal Access Token

## 快速开始

### 方法一：Docker 部署（推荐）

```bash
# 拉取最新镜像
docker pull your-username/moments-go:latest

# 运行容器
docker run -d \
  --name moments-go-bot \
  --restart unless-stopped \
  -e TELEGRAM_BOT_TOKEN="your_token" \
  -e TELEGRAM_USER_ID="your_user_id" \
  -e GITHUB_SECRET="your_secret" \
  your-username/moments-go:latest
```

### 方法二：Docker Compose

```bash
# 创建 .env 文件
cp env.example .env
# 编辑 .env 文件，填入你的配置

# 启动服务
docker-compose up -d
```

### 方法三：本地运行

```bash
# 克隆项目
git clone <repository-url>
cd moments-go

# 安装依赖
go mod tidy

# 配置环境变量
cp env.example .env
# 编辑 .env 文件

# 运行
go run ./cmd
```

## 安装和配置

### 1. 获取配置信息

#### Telegram Bot Token
1. 在 Telegram 中找到 @BotFather
2. 发送 `/newbot` 创建新机器人
3. 获取 Bot Token

#### Telegram User ID
1. 在 Telegram 中找到 @userinfobot
2. 发送任意消息获取你的 User ID

#### GitHub Personal Access Token
1. 访问 GitHub Settings > Developer settings > Personal access tokens
2. 生成新的 token，需要以下权限：
   - `repo` - 完整的仓库访问权限
   - `issues` - Issues 访问权限

### 2. 环境变量配置

```env
# Telegram 机器人配置
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_USER_ID=your_telegram_user_id_here

# GitHub 配置
GITHUB_SECRET=your_github_personal_access_token_here
GITHUB_FILE_REPO=moments-files
```

## 使用方法

### 机器人命令

- `/start` - 显示帮助信息
- `/say <内容>` - 发布纯文字动态

### 使用示例

1. **发送图片**：直接发送图片，机器人会自动发布动态
2. **发送视频**：直接发送视频，机器人会自动发布动态
3. **发送文字**：直接发送文字，机器人会发布纯文字动态
4. **组合使用**：先发送媒体文件，再发送文字来更新动态内容

## 项目结构

```
moments-go/
├── cmd/
│   └── main.go          # 程序入口
├── config/
│   └── config.go        # 配置管理
├── github/
│   └── github.go        # GitHub API 相关
├── telegram/
│   └── telegram.go      # Telegram API 相关
├── handlers/
│   └── handlers.go      # 消息处理器
├── types/
│   └── types.go         # 类型定义
├── scripts/
│   └── deploy.sh        # 部署脚本
├── .github/workflows/   # GitHub Actions
├── Dockerfile           # Docker 构建文件
├── docker-compose.yml   # Docker Compose 配置
└── README.md            # 项目说明
```

## Docker 部署

### 构建镜像

```bash
docker build -t moments-go .
```

### 使用 Docker Compose

```bash
# 开发环境
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.prod.yml up -d
```

### 自动部署脚本

```bash
# 设置环境变量
export TELEGRAM_BOT_TOKEN="your_token"
export TELEGRAM_USER_ID="your_user_id"
export GITHUB_SECRET="your_secret"

# 运行部署脚本
./scripts/deploy.sh
```

## GitHub Actions

项目配置了以下 GitHub Actions 工作流：

- **Test**: 在 PR 和 push 时运行测试
- **Build and Push**: 自动构建并推送到 Docker Hub
- **Release**: 创建 tag 时自动发布

### 设置 GitHub Secrets

在 GitHub 仓库设置中添加以下 secrets：

- `DOCKERHUB_USERNAME`: Docker Hub 用户名
- `DOCKERHUB_TOKEN`: Docker Hub Access Token

### 发布新版本

```bash
# 创建并推送 tag
git tag v1.0.0
git push origin v1.0.0
```

## 技术栈

- **语言**：Golang 1.21+
- **Telegram API**：go-telegram-bot-api/v5
- **配置管理**：godotenv
- **HTTP 客户端**：标准库 net/http
- **JSON 处理**：标准库 encoding/json
- **容器化**：Docker
- **CI/CD**：GitHub Actions

## 注意事项

1. 确保 GitHub 仓库存在且有写入权限
2. 视频文件大小限制为 50MB
3. 动态内容长度限制为 5000 字符
4. 只有授权的用户 ID 才能使用机器人
5. 生产环境建议使用 Docker 部署

## 许可证

MIT License 
