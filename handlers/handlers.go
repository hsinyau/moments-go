package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"moments-go/config"
	"moments-go/github"
	"moments-go/telegram"
	"moments-go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleStartCommand 处理 /start 命令
func HandleStartCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	message := `你好！欢迎使用机器人。

使用方法：
1. 发送图片/视频，会自动发布动态
2. 直接发送 /say <内容> 发布纯文字动态

💡 提示：发送媒体文件后，可以再发送文字来更新动态！`
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandleSayCommand 处理 /say 命令
func HandleSayCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	text := update.Message.Text
	if text == "" {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 格式错误\n正确格式：/say <内容>")
	}
	content := strings.Join(parts[1:], " ")
	if content == "" {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 内容不能为空")
	}
	if err := github.SendMessage(bot, update.Message.Chat.ID, "⏳ 正在发布动态..."); err != nil {
		return err
	}
	_, err := github.CreateGitHubIssue(content)
	if err != nil {
		log.Printf("发布动态失败: %v", err)
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 发布失败，请稍后重试")
	}
	return github.SendMessage(bot, update.Message.Chat.ID, "✅ 动态发布成功！")
}

// HandleUnknownCommand 处理未知命令
func HandleUnknownCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	message := `❓ 未知命令

使用方法：
1. 发送图片/视频，会自动发布动态
2. 直接发送 /say <内容> 发布纯文字动态

💡 提示：发送媒体文件后，可以再发送文字来更新动态！`
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandlePhotoMessage 处理图片消息
func HandlePhotoMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	photos := update.Message.Photo
	if len(photos) == 0 {
		return nil
	}
	photo := photos[len(photos)-1]
	config.MediaMutex.Lock()
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  photo.FileID,
		Type:    "photo",
		Caption: update.Message.Caption,
	}
	config.MediaMutex.Unlock()

	// 设置定时器，5分钟后自动发布
	telegram.ScheduleMediaPublish(bot, update.Message.Chat.ID, func() {
		config.MediaMutex.RLock()
		_, exists := config.PendingMedia[update.Message.Chat.ID]
		config.MediaMutex.RUnlock()
		if exists {
			ProcessPendingMedia(bot, update.Message.Chat.ID, "")
		}
	})

	message := "📷 图片已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 你可以继续发送文字来更新动态内容，或者等待5分钟后自动发布。"
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandleVideoMessage 处理视频消息
func HandleVideoMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	video := update.Message.Video
	if video == nil {
		return nil
	}
	if video.FileSize > config.MaxFileSize {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 视频文件过大，请上传小于 50MB 的视频")
	}
	config.MediaMutex.Lock()
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  video.FileID,
		Type:    "video",
		Caption: update.Message.Caption,
	}
	config.MediaMutex.Unlock()
	telegram.ScheduleMediaPublish(bot, update.Message.Chat.ID, func() {
		config.MediaMutex.RLock()
		_, exists := config.PendingMedia[update.Message.Chat.ID]
		config.MediaMutex.RUnlock()
		if exists {
			ProcessPendingMedia(bot, update.Message.Chat.ID, "")
		}
	})
	message := "🎥 视频已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 你可以继续发送文字来更新动态内容，或者等待5分钟后自动发布。"
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandleTextMessage 处理文本消息
func HandleTextMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	text := update.Message.Text
	if text == "" {
		return nil
	}
	if strings.HasPrefix(text, "/") {
		if strings.HasPrefix(text, "/start") {
			return HandleStartCommand(bot, update)
		} else if strings.HasPrefix(text, "/say") {
			return HandleSayCommand(bot, update)
		} else {
			return HandleUnknownCommand(bot, update)
		}
	}
	config.MediaMutex.RLock()
	_, exists := config.PendingMedia[update.Message.Chat.ID]
	config.MediaMutex.RUnlock()
	if exists {
		return ProcessPendingMedia(bot, update.Message.Chat.ID, text)
	}
	if err := github.SendMessage(bot, update.Message.Chat.ID, "⏳ 正在发布动态..."); err != nil {
		return err
	}
	_, err := github.CreateGitHubIssue(text)
	if err != nil {
		log.Printf("发布动态失败: %v", err)
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 发布失败，请稍后重试")
	}
	return github.SendMessage(bot, update.Message.Chat.ID, "✅ 动态发布成功！")
}

// ProcessPendingMedia 处理待发布的媒体文件
func ProcessPendingMedia(bot *tgbotapi.BotAPI, chatID int64, content string) error {
	config.MediaMutex.Lock()
	pending, exists := config.PendingMedia[chatID]
	if !exists {
		config.MediaMutex.Unlock()
		return nil
	}
	delete(config.PendingMedia, chatID)
	config.MediaMutex.Unlock()
	if err := github.SendMessage(bot, chatID, "⏳ 正在处理媒体文件..."); err != nil {
		return err
	}
	fileBuffer, err := telegram.DownloadFile(bot, pending.FileID)
	if err != nil {
		return err
	}
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
	mediaFiles := []*types.MediaFile{{
		Name:    fileName,
		Content: fileBuffer,
		Type:    fileType,
	}}
	_, err = github.UploadToGitHub(bot, finalContent, mediaFiles)
	if err != nil {
		return err
	}
	return github.SendMessage(bot, chatID, "✅ 动态发布成功！")
} 
