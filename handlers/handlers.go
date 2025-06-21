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
1. 发送图片/视频，会自动弹出标签选择按钮
2. 发送文字消息，也会弹出标签选择按钮
3. 直接发送 /say <内容> 发布纯文字动态
4. 发送 /tags 查看所有可用标签
5. 发送 /label <标签名> 设置默认标签
6. 发送 /refresh 刷新标签列表

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）`
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandleTagsCommand 处理 /tags 命令
func HandleTagsCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	labels := config.GetLabels()
	
	// 构建内联键盘
	var buttons [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(labels); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(labels); j++ {
			label := labels[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, "setdefault:"+label))
		}
		buttons = append(buttons, row)
	}
	// 刷新按钮
	refreshRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", "label:refresh"),
	}
	buttons = append(buttons, refreshRow)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "📋 请选择一个标签作为默认标签：")
	msg.ReplyMarkup = keyboard
	_, err := bot.Send(msg)
	return err
}

// HandleRefreshCommand 处理 /refresh 命令
func HandleRefreshCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	if err := github.SendMessage(bot, update.Message.Chat.ID, "🔄 正在刷新标签列表..."); err != nil {
		return err
	}
	
	// 强制从 GitHub 获取最新标签
	labels, err := github.GetGitHubLabels()
	if err != nil {
		log.Printf("刷新标签失败: %v", err)
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 刷新标签失败，请稍后重试")
	}
	
	// 更新缓存
	config.SetLabels(labels)
	
	message := "✅ 标签列表已刷新！\n\n📋 可用标签：\n"
	for i, label := range labels {
		message += fmt.Sprintf("%d. %s\n", i+1, label)
	}
	
	return github.SendMessage(bot, update.Message.Chat.ID, message)
}

// HandleLabelCommand 处理 /label 命令
func HandleLabelCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	text := update.Message.Text
	if text == "" {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}
	
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 格式错误\n正确格式：/label <标签名>")
	}
	
	label := strings.Join(parts[1:], " ")
	
	// 获取当前可用标签
	labels := config.GetLabels()
	
	// 检查标签是否有效
	valid := false
	for _, availableLabel := range labels {
		if availableLabel == label {
			valid = true
			break
		}
	}
	
	if !valid {
		message := "❌ 无效的标签\n\n可用标签：\n"
		for _, l := range labels {
			message += fmt.Sprintf("• %s\n", l)
		}
		message += "\n💡 发送 /refresh 刷新标签列表"
		return github.SendMessage(bot, update.Message.Chat.ID, message)
	}
	
	// 存储用户默认标签
	config.MediaMutex.Lock()
	if config.UserDefaultLabels == nil {
		config.UserDefaultLabels = make(map[int64]string)
	}
	config.UserDefaultLabels[update.Message.Chat.ID] = label
	config.MediaMutex.Unlock()
	
	return github.SendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("✅ 默认标签已设置为：%s", label))
}

// createLabelKeyboard 创建标签选择键盘
func createLabelKeyboard() tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// 获取当前可用标签
	labels := config.GetLabels()
	
	// 每行3个按钮
	for i := 0; i < len(labels); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(labels); j++ {
			label := labels[i+j]
			button := tgbotapi.NewInlineKeyboardButtonData(label, "label:"+label)
			row = append(row, button)
		}
		buttons = append(buttons, row)
	}
	
	// 添加刷新和取消按钮
	actionRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", "label:refresh"),
		tgbotapi.NewInlineKeyboardButtonData("❌ 取消", "label:cancel"),
	}
	buttons = append(buttons, actionRow)
	
	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// HandleCallbackQuery 处理回调查询（标签选择）
func HandleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.CallbackQuery.From.ID) {
		return nil
	}
	
	callback := update.CallbackQuery
	data := callback.Data
	
	if strings.HasPrefix(data, "setdefault:") {
		label := strings.TrimPrefix(data, "setdefault:")
		// 设置为默认标签
		config.MediaMutex.Lock()
		if config.UserDefaultLabels == nil {
			config.UserDefaultLabels = make(map[int64]string)
		}
		config.UserDefaultLabels[callback.From.ID] = label
		config.MediaMutex.Unlock()
		msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, fmt.Sprintf("✅ 默认标签已设置为：%s", label))
		bot.Send(msg)
		return github.SendMessage(bot, callback.From.ID, fmt.Sprintf("📝 你的默认标签已设置为：%s\n下次发动态会自动带上该标签。", label))
	}
	
	if strings.HasPrefix(data, "label:") {
		label := strings.TrimPrefix(data, "label:")
		
		if label == "cancel" {
			// 取消标签选择
			config.MediaMutex.Lock()
			delete(config.PendingMedia, callback.From.ID)
			config.MediaMutex.Unlock()
			
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 已取消标签选择")
			bot.Send(msg)
			return nil
		}
		
		if label == "refresh" {
			// 刷新标签
			labels, err := github.GetGitHubLabels()
			if err != nil {
				log.Printf("刷新标签失败: %v", err)
				msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 刷新标签失败")
				bot.Send(msg)
				return nil
			}
			
			// 更新缓存
			config.SetLabels(labels)
			
			// 重新创建键盘
			newKeyboard := createLabelKeyboard()
			message := "🔄 标签已刷新！\n\n💡 请选择标签，然后可以发送文字来更新动态内容！"
			
			msg := tgbotapi.NewEditMessageTextAndMarkup(callback.From.ID, callback.Message.MessageID, message, newKeyboard)
			bot.Send(msg)
			return nil
		}
		
		// 检查是否有待处理的媒体文件
		config.MediaMutex.Lock()
		pending, exists := config.PendingMedia[callback.From.ID]
		config.MediaMutex.Unlock()
		
		if !exists {
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 没有待处理的媒体文件")
			bot.Send(msg)
			return nil
		}
		
		// 设置标签
		config.MediaMutex.Lock()
		pending.Labels = []string{label}
		config.PendingMedia[callback.From.ID] = pending
		config.MediaMutex.Unlock()
		
		// 如果是文字消息，立即发布
		if pending.Type == "text" {
			// 更新消息
			message := fmt.Sprintf("✅ 已选择标签：%s\n\n⏳ 正在发布文字动态...", label)
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, message)
			bot.Send(msg)
			
			// 立即处理文字消息
			return ProcessPendingMedia(bot, callback.From.ID, "")
		}
		
		// 为媒体文件设置定时器，5分钟后自动发布
		telegram.ScheduleMediaPublish(bot, callback.From.ID, func() {
			config.MediaMutex.RLock()
			_, exists := config.PendingMedia[callback.From.ID]
			config.MediaMutex.RUnlock()
			if exists {
				ProcessPendingMedia(bot, callback.From.ID, "")
			}
		})
		
		// 更新消息
		message := fmt.Sprintf("✅ 已选择标签：%s\n\n💡 你可以继续发送文字来更新动态内容，或者等待5分钟后自动发布。", label)
		msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, message)
		bot.Send(msg)
		
		// 发送确认消息
		return github.SendMessage(bot, callback.From.ID, fmt.Sprintf("📝 标签已设置为：%s\n\n现在可以发送文字来更新动态内容！", label))
	}
	
	return nil
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
	
	// 获取用户默认标签
	config.MediaMutex.RLock()
	defaultLabel := config.UserDefaultLabels[update.Message.Chat.ID]
	config.MediaMutex.RUnlock()
	
	var labels []string
	if defaultLabel != "" {
		labels = []string{defaultLabel}
	} else {
		labels = []string{"动态"}
	}
	
	_, err := github.CreateGitHubIssueWithLabels(content, labels)
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
1. 发送图片/视频，会自动弹出标签选择按钮
2. 发送文字消息，也会弹出标签选择按钮
3. 直接发送 /say <内容> 发布纯文字动态
4. 发送 /tags 查看所有可用标签
5. 发送 /label <标签名> 设置默认标签
6. 发送 /refresh 刷新标签列表

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）`
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
	
	// 获取用户默认标签
	config.MediaMutex.RLock()
	defaultLabel := config.UserDefaultLabels[update.Message.Chat.ID]
	config.MediaMutex.RUnlock()
	
	var labels []string
	if defaultLabel != "" {
		labels = []string{defaultLabel}
	}
	
	config.MediaMutex.Lock()
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  photo.FileID,
		Type:    "photo",
		Caption: update.Message.Caption,
		Labels:  labels,
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
	if len(labels) > 0 {
		message += fmt.Sprintf("\n\n🏷️ 默认标签：%s", labels[0])
	}
	message += "\n\n💡 请选择标签，然后可以发送文字来更新动态内容！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
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
		return github.SendMessage(bot, update.Message.Chat.ID, "❌ 视频文件过大，请上传小于 50MB 的视频")
	}
	
	// 获取用户默认标签
	config.MediaMutex.RLock()
	defaultLabel := config.UserDefaultLabels[update.Message.Chat.ID]
	config.MediaMutex.RUnlock()
	
	var labels []string
	if defaultLabel != "" {
		labels = []string{defaultLabel}
	}
	
	config.MediaMutex.Lock()
	config.PendingMedia[update.Message.Chat.ID] = &types.PendingMedia{
		FileID:  video.FileID,
		Type:    "video",
		Caption: update.Message.Caption,
		Labels:  labels,
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
	if len(labels) > 0 {
		message += fmt.Sprintf("\n\n🏷️ 默认标签：%s", labels[0])
	}
	message += "\n\n💡 请选择标签，然后可以发送文字来更新动态内容！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
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
	if strings.HasPrefix(text, "/") {
		if strings.HasPrefix(text, "/start") {
			return HandleStartCommand(bot, update)
		} else if strings.HasPrefix(text, "/say") {
			return HandleSayCommand(bot, update)
		} else if strings.HasPrefix(text, "/tags") {
			return HandleTagsCommand(bot, update)
		} else if strings.HasPrefix(text, "/label") {
			return HandleLabelCommand(bot, update)
		} else if strings.HasPrefix(text, "/refresh") {
			return HandleRefreshCommand(bot, update)
		} else {
			return HandleUnknownCommand(bot, update)
		}
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
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyMarkup = keyboard
	_, err := bot.Send(msg)
	return err
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
	
	// 处理纯文字消息
	if pending.Type == "text" {
		if err := github.SendMessage(bot, chatID, "⏳ 正在发布文字动态..."); err != nil {
			return err
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
		
		_, err := github.CreateGitHubIssueWithLabels(finalContent, labels)
		if err != nil {
			log.Printf("发布文字动态失败: %v", err)
			return github.SendMessage(bot, chatID, "❌ 发布失败，请稍后重试")
		}
		return github.SendMessage(bot, chatID, "✅ 文字动态发布成功！")
	}
	
	// 处理媒体文件
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
	
	// 使用标签
	labels := pending.Labels
	if len(labels) == 0 {
		labels = []string{"动态"}
	}
	
	_, err = github.UploadToGitHubWithLabels(bot, finalContent, mediaFiles, labels)
	if err != nil {
		return err
	}
	return github.SendMessage(bot, chatID, "✅ 动态发布成功！")
} 
