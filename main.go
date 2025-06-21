package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// ==================== 类型定义 ====================

// GitHubUploadResponse GitHub 文件上传响应
type GitHubUploadResponse struct {
	Content *struct {
		DownloadURL string `json:"download_url"`
	} `json:"content"`
}

// GitHubIssueResponse GitHub Issue 响应
type GitHubIssueResponse struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

// MediaFile 媒体文件结构
type MediaFile struct {
	Name    string
	Content []byte
	Type    string
}

// PendingMedia 待处理的媒体文件
type PendingMedia struct {
	FileID  string
	Type    string
	Caption string
}

// Config 配置结构
type Config struct {
	TelegramBotToken string
	TelegramUserID   int64
	GitHubSecret     string
	GitHubFileRepo   string
}

// ==================== 配置常量 ====================
const (
	WaitTime = 5 * time.Minute // 5分钟等待时间
	MaxFileSize = 50 * 1024 * 1024 // 50MB
)

// ==================== 全局变量 ====================
var (
	config      Config
	pendingMedia = make(map[int64]*PendingMedia)
	mediaMutex   sync.RWMutex
)

// ==================== 工具函数 ====================

// loadConfig 加载配置
func loadConfig() error {
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 无法加载 .env 文件")
	}

	config.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if config.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN 未设置")
	}

	userIDStr := os.Getenv("TELEGRAM_USER_ID")
	if userIDStr == "" {
		return fmt.Errorf("TELEGRAM_USER_ID 未设置")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的 TELEGRAM_USER_ID: %v", err)
	}
	config.TelegramUserID = userID

	config.GitHubSecret = os.Getenv("GITHUB_SECRET")
	if config.GitHubSecret == "" {
		return fmt.Errorf("GITHUB_SECRET 未设置")
	}

	config.GitHubFileRepo = os.Getenv("GITHUB_FILE_REPO")
	if config.GitHubFileRepo == "" {
		config.GitHubFileRepo = "moments-files" // 默认值
	}

	return nil
}

// isAuthorizedUser 验证用户权限
func isAuthorizedUser(chatID int64) bool {
	return chatID == config.TelegramUserID
}

// sendMessage 发送消息并处理错误
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("发送消息失败: %v", err)
		return err
	}
	return nil
}

// downloadFile 从 Telegram 下载文件
func downloadFile(bot *tgbotapi.BotAPI, fileID string) ([]byte, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	if file.FilePath == "" {
		return nil, fmt.Errorf("无法获取文件路径")
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.TelegramBotToken, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %v", err)
	}

	return content, nil
}

// createGitHubIssue 创建 GitHub Issue
func createGitHubIssue(content string) (*GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	issueData := map[string]interface{}{
		"title": timestamp,
		"body":  content,
		"labels": []string{"动态"},
	}

	jsonData, err := json.Marshal(issueData)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.github.com/repos/hsinyau/moments/issues", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var issue GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &issue, nil
}

// uploadFileToGitHub 上传文件到 GitHub
func uploadFileToGitHub(file *MediaFile, timestamp string) (string, error) {
	// 将字节数组转换为 base64
	base64Content := base64.StdEncoding.EncodeToString(file.Content)

	uploadData := map[string]interface{}{
		"message": fmt.Sprintf("Add media file: %s", file.Name),
		"content": base64Content,
	}

	jsonData, err := json.Marshal(uploadData)
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %v", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/%s/contents/moments/%s_%s", 
		config.GitHubFileRepo, timestamp, file.Name)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var uploadResult GitHubUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if uploadResult.Content == nil || uploadResult.Content.DownloadURL == "" {
		return "", fmt.Errorf("文件 %s 上传失败", file.Name)
	}

	return uploadResult.Content.DownloadURL, nil
}

// uploadToGitHub 上传媒体文件到 GitHub 并发布动态
func uploadToGitHub(bot *tgbotapi.BotAPI, content string, mediaFiles []*MediaFile) (*GitHubIssueResponse, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	var mediaUrls []string

	if len(mediaFiles) > 0 {
		if err := sendMessage(bot, config.TelegramUserID, "📤 正在上传媒体文件..."); err != nil {
			return nil, err
		}

		for _, file := range mediaFiles {
			downloadURL, err := uploadFileToGitHub(file, timestamp)
			if err != nil {
				return nil, fmt.Errorf("上传文件 %s 失败: %v", file.Name, err)
			}
			mediaUrls = append(mediaUrls, downloadURL)
		}
	}

	// 构建完整内容
	fullContent := content
	if len(mediaUrls) > 0 {
		for _, url := range mediaUrls {
			fullContent += fmt.Sprintf("\n![%s](%s)", url, url)
		}
	}

	// 创建 GitHub Issue
	return createGitHubIssue(fullContent)
}

// processPendingMedia 处理待发布的媒体文件
func processPendingMedia(bot *tgbotapi.BotAPI, chatID int64, content string) error {
	mediaMutex.Lock()
	pending, exists := pendingMedia[chatID]
	if !exists {
		mediaMutex.Unlock()
		return nil
	}
	delete(pendingMedia, chatID)
	mediaMutex.Unlock()

	if err := sendMessage(bot, chatID, "⏳ 正在处理媒体文件..."); err != nil {
		return err
	}

	// 下载文件
	fileBuffer, err := downloadFile(bot, pending.FileID)
	if err != nil {
		return err
	}

	// 生成文件名
	timestamp := time.Now().Unix()
	var extension string
	var fileType string
	if pending.Type == "photo" {
		extension = "jpg"
		fileType = "image/jpeg"
	} else {
		extension = "mp4"
		fileType = "video/mp4"
	}
	fileName := fmt.Sprintf("%s_%d.%s", pending.Type, timestamp, extension)

	// 确定内容
	finalContent := content
	if finalContent == "" {
		finalContent = pending.Caption
		if finalContent == "" {
			if pending.Type == "photo" {
				finalContent = "📷 分享了一张图片"
			} else {
				finalContent = "🎥 分享了一个视频"
			}
		}
	}

	// 上传到 GitHub 并发布动态
	mediaFiles := []*MediaFile{{
		Name:    fileName,
		Content: fileBuffer,
		Type:    fileType,
	}}

	_, err = uploadToGitHub(bot, finalContent, mediaFiles)
	if err != nil {
		return err
	}

	return sendMessage(bot, chatID, "✅ 动态发布成功！")
}

// scheduleMediaPublish 设置定时器，自动发布媒体文件
func scheduleMediaPublish(bot *tgbotapi.BotAPI, chatID int64) {
	time.AfterFunc(WaitTime, func() {
		mediaMutex.RLock()
		_, exists := pendingMedia[chatID]
		mediaMutex.RUnlock()

		if exists {
			processPendingMedia(bot, chatID, "")
		}
	})
}

// ==================== 命令处理器 ====================

// handleStartCommand 处理 /start 命令
func handleStartCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	message := `你好！欢迎使用机器人。

使用方法：
1. 发送图片/视频，会自动发布动态
2. 直接发送 /say <内容> 发布纯文字动态

💡 提示：发送媒体文件后，可以再发送文字来更新动态！`

	return sendMessage(bot, update.Message.Chat.ID, message)
}

// handleSayCommand 处理 /say 命令
func handleSayCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	text := update.Message.Text
	if text == "" {
		return sendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}

	// 解析命令参数
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return sendMessage(bot, update.Message.Chat.ID, "❌ 格式错误\n正确格式：/say <内容>")
	}

	content := strings.Join(parts[1:], " ")
	if content == "" {
		return sendMessage(bot, update.Message.Chat.ID, "❌ 内容不能为空")
	}

	// 纯文字动态
	if err := sendMessage(bot, update.Message.Chat.ID, "⏳ 正在发布动态..."); err != nil {
		return err
	}

	_, err := createGitHubIssue(content)
	if err != nil {
		log.Printf("发布动态失败: %v", err)
		return sendMessage(bot, update.Message.Chat.ID, "❌ 发布失败，请稍后重试")
	}

	return sendMessage(bot, update.Message.Chat.ID, "✅ 动态发布成功！")
}

// handleUnknownCommand 处理未知命令
func handleUnknownCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	message := `❓ 未知命令

使用方法：
1. 发送图片/视频，会自动发布动态
2. 直接发送 /say <内容> 发布纯文字动态

💡 提示：发送媒体文件后，可以再发送文字来更新动态！`

	return sendMessage(bot, update.Message.Chat.ID, message)
}

// ==================== 媒体文件处理器 ====================

// handlePhotoMessage 处理图片消息
func handlePhotoMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	// 获取最大尺寸的图片
	photos := update.Message.Photo
	if len(photos) == 0 {
		return nil
	}

	photo := photos[len(photos)-1]

	// 将图片存储到待处理队列中
	mediaMutex.Lock()
	pendingMedia[update.Message.Chat.ID] = &PendingMedia{
		FileID:  photo.FileID,
		Type:    "photo",
		Caption: update.Message.Caption,
	}
	mediaMutex.Unlock()

	// 设置定时器，5分钟后自动发布
	scheduleMediaPublish(bot, update.Message.Chat.ID)

	// 通知用户
	message := "📷 图片已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 你可以继续发送文字来更新动态内容，或者等待5分钟后自动发布。"

	return sendMessage(bot, update.Message.Chat.ID, message)
}

// handleVideoMessage 处理视频消息
func handleVideoMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	video := update.Message.Video
	if video == nil {
		return nil
	}

	// 检查视频大小（限制为 50MB）
	if video.FileSize > MaxFileSize {
		return sendMessage(bot, update.Message.Chat.ID, "❌ 视频文件过大，请上传小于 50MB 的视频")
	}

	// 将视频存储到待处理队列中
	mediaMutex.Lock()
	pendingMedia[update.Message.Chat.ID] = &PendingMedia{
		FileID:  video.FileID,
		Type:    "video",
		Caption: update.Message.Caption,
	}
	mediaMutex.Unlock()

	// 设置定时器，5分钟后自动发布
	scheduleMediaPublish(bot, update.Message.Chat.ID)

	// 通知用户
	message := "🎥 视频已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 你可以继续发送文字来更新动态内容，或者等待5分钟后自动发布。"

	return sendMessage(bot, update.Message.Chat.ID, message)
}

// handleTextMessage 处理文本消息
func handleTextMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !isAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	text := update.Message.Text
	if text == "" {
		return nil
	}

	// 检查是否是命令
	if strings.HasPrefix(text, "/") {
		if strings.HasPrefix(text, "/start") {
			return handleStartCommand(bot, update)
		} else if strings.HasPrefix(text, "/say") {
			return handleSayCommand(bot, update)
		} else {
			return handleUnknownCommand(bot, update)
		}
	}

	// 检查是否有待处理的媒体文件
	mediaMutex.RLock()
	_, exists := pendingMedia[update.Message.Chat.ID]
	mediaMutex.RUnlock()

	if exists {
		return processPendingMedia(bot, update.Message.Chat.ID, text)
	}

	// 普通文本消息，创建纯文字动态
	if err := sendMessage(bot, update.Message.Chat.ID, "⏳ 正在发布动态..."); err != nil {
		return err
	}

	_, err := createGitHubIssue(text)
	if err != nil {
		log.Printf("发布动态失败: %v", err)
		return sendMessage(bot, update.Message.Chat.ID, "❌ 发布失败，请稍后重试")
	}

	return sendMessage(bot, update.Message.Chat.ID, "✅ 动态发布成功！")
}

// ==================== 主函数 ====================

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建机器人实例
	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Fatalf("创建机器人失败: %v", err)
	}

	bot.Debug = false
	log.Printf("机器人已启动: %s", bot.Self.UserName)

	// 设置更新配置
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// 获取更新通道
	updates := bot.GetUpdatesChan(updateConfig)

	// 处理更新
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("收到消息: [%d] %s", update.Message.Chat.ID, update.Message.Text)

		var err error

		// 根据消息类型处理
		switch {
		case update.Message.Photo != nil:
			err = handlePhotoMessage(bot, update)
		case update.Message.Video != nil:
			err = handleVideoMessage(bot, update)
		case update.Message.Text != "":
			err = handleTextMessage(bot, update)
		}

		if err != nil {
			log.Printf("处理消息失败: %v", err)
		}
	}
} 
