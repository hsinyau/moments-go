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
		Labels:  []string{},
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

	// 创建标签选择键盘
	keyboard := createLabelKeyboard()
	message := "📷 图片已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 请选择标签，然后可以发送文字来更新动态内容！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, cleanUTF8String(message))
	msg.ReplyMarkup = keyboard
	_, err := bot.Send(msg)
	return err
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
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 视频文件过大，请上传小于 50MB 的视频")
	}
	
	config.MediaMutex.Lock()
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  video.FileID,
		Type:    "video",
		Caption: update.Message.Caption,
		Labels:  []string{},
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
	
	// 创建标签选择键盘
	keyboard := createLabelKeyboard()
	message := "🎥 视频已接收！"
	if update.Message.Caption != "" {
		message += fmt.Sprintf("\n\n当前文字：%s", update.Message.Caption)
	}
	message += "\n\n💡 请选择标签，然后可以发送文字来更新动态内容！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, cleanUTF8String(message))
	msg.ReplyMarkup = keyboard
	_, err := bot.Send(msg)
	return err
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
	
	// 清理用户输入的文字，确保UTF-8编码
	text = cleanUTF8String(text)
	
	if strings.HasPrefix(text, "/") {
		if strings.HasPrefix(text, "/start") {
			return HandleStartCommand(bot, update)
		} else if strings.HasPrefix(text, "/tags") {
			return HandleTagsCommand(bot, update)
		} else if strings.HasPrefix(text, "/refresh") {
			return HandleRefreshCommand(bot, update)
		} else if strings.HasPrefix(text, "/edit") {
			return HandleEditCommand(bot, update)
		} else if strings.HasPrefix(text, "/delete") {
			return HandleDeleteCommand(bot, update)
		} else if strings.HasPrefix(text, "/cancel") {
			return HandleCancelCommand(bot, update)
		} else {
			return HandleUnknownCommand(bot, update)
		}
	}
	
	// 检查是否在编辑模式
	if config.IsInEditMode(update.Message.Chat.ID) {
		return HandleEditTextMessage(bot, update)
	}
	
	// 检查是否有待处理的媒体文件
	config.MediaMutex.RLock()
	_, exists := config.PendingMedia[update.Message.Chat.ID]
	config.MediaMutex.RUnlock()
	if exists {
		return ProcessPendingMedia(bot, update.Message.Chat.ID, text)
	}
	
	// 处理纯文字消息 - 弹出标签选择按钮
	config.MediaMutex.Lock()
	// 将文字消息存储为待发布内容
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  "",
		Type:    "text",
		Caption: text,
		Labels:  []string{},
	}
	config.MediaMutex.Unlock()
	
	// 创建标签选择键盘
	keyboard := createLabelKeyboard()
	message := "📝 文字已接收！"
	message += fmt.Sprintf("\n\n当前文字：%s", text)
	message += "\n\n💡 请选择标签来发布动态！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, cleanUTF8String(message))
	msg.ReplyMarkup = keyboard
	_, sendErr := bot.Send(msg)
	return sendErr
}

// ProcessPendingMedia 处理待发布的媒体文件
func ProcessPendingMedia(bot *tgbotapi.BotAPI, chatID int64, content string) error {
	return ProcessPendingMediaWithProgress(bot, chatID, content, true)
}

// ProcessPendingMediaWithProgress 处理待发布的媒体文件，可选择是否发送进度消息
func ProcessPendingMediaWithProgress(bot *tgbotapi.BotAPI, chatID int64, content string, showProgress bool) error {
	config.MediaMutex.Lock()
	pending, exists := config.PendingMedia[chatID]
	if !exists {
		config.MediaMutex.Unlock()
		return nil
	}
	delete(config.PendingMedia, chatID)
	config.MediaMutex.Unlock()
	
	// 处理纯文字消息
	if pending.Type == "text" {
		if showProgress {
			if err := safeSendMessage(bot, chatID, "⏳ 正在发布文字动态..."); err != nil {
				return err
			}
		}
		
		finalContent := content
		if finalContent == "" {
			finalContent = pending.Caption
		}
		
		// 使用标签
		labels := pending.Labels
		if len(labels) == 0 {
			labels = []string{"动态"}
		}
		
		issue, err := github.CreateGitHubIssueWithLabels(finalContent, labels)
		if err != nil {
			log.Printf("发布文字动态失败: %v", err)
			return safeSendMessage(bot, chatID, "❌ 发布失败，请稍后重试")
		}
		
		// 缓存已发布的动态信息
		moment := &types.PublishedMoment{
			IssueID:     issue.ID,
			IssueNumber: issue.Number,
			Content:     finalContent,
			Labels:      labels,
			MediaURLs:   []string{},
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}
		config.AddPublishedMoment(moment)
		
		successMessage := fmt.Sprintf("✅ 文字动态发布成功！\n\n🔗 查看链接：%s", issue.HTMLURL)
		return safeSendMessage(bot, chatID, successMessage)
	}
	
	// 处理媒体文件
	if showProgress {
		if err := safeSendMessage(bot, chatID, "⏳ 正在处理媒体文件..."); err != nil {
			return err
		}
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
	
	// 使用标签
	labels := pending.Labels
	if len(labels) == 0 {
		labels = []string{"动态"}
	}
	
	_, err = github.UploadToGitHubWithLabels(bot, finalContent, mediaFiles, labels)
	if err != nil {
		return err
	}
	
	// 缓存已发布的动态信息（这里需要从响应中获取，暂时跳过）
	// TODO: 修改 UploadToGitHubWithLabels 返回 Issue 信息
	
	return safeSendMessage(bot, chatID, "✅ 动态发布成功！")
} 
