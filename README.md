# Moments-Go

一个用 Golang 实现的 Telegram 机器人，用于自动发布动态到 GitHub Issues。

## 功能特性

- 📱 支持发送图片和视频，自动发布到 GitHub Issues
- 💬 支持纯文字动态发布
- 🔐 用户权限验证，只允许授权用户使用
- 📤 自动上传媒体文件到 GitHub 仓库
- ⏰ 支持延迟发布功能
- 🏷️ 自动添加标签分类

## 环境要求

- Go 1.21 或更高版本
- Telegram Bot Token
- GitHub Personal Access Token

## 安装和配置

### 1. 克隆项目

```bash
git clone <repository-url>
cd moments-go
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 配置环境变量

复制 `env.example` 文件为 `.env`：

```bash
cp env.example .env
```

编辑 `.env` 文件，填入你的配置信息：

```env
# Telegram 机器人配置
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_USER_ID=your_telegram_user_id_here

# GitHub 配置
GITHUB_SECRET=your_github_personal_access_token_here
GITHUB_FILE_REPO=moments-files
```

### 4. 获取配置信息

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

## 使用方法

### 启动机器人

```bash
go run main.go
```

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
├── main.go          # 主程序文件
├── go.mod           # Go 模块文件
├── go.sum           # 依赖校验文件
├── env.example      # 环境变量示例
└── README.md        # 项目说明
```

## 技术栈

- **语言**：Golang 1.21+
- **Telegram API**：go-telegram-bot-api/v5
- **配置管理**：godotenv
- **HTTP 客户端**：标准库 net/http
- **JSON 处理**：标准库 encoding/json

## 注意事项

1. 确保 GitHub 仓库存在且有写入权限
2. 视频文件大小限制为 50MB
3. 动态内容长度限制为 5000 字符
4. 只有授权的用户 ID 才能使用机器人

## 许可证

MIT License 
