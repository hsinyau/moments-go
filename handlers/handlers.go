package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"moments-go/config"
	"moments-go/github"
	"moments-go/telegram"
	"moments-go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// cleanUTF8String 清理字符串，确保是有效的UTF-8编码
func cleanUTF8String(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	
	// 如果字符串不是有效的UTF-8，进行清理
	var result strings.Builder
	for _, r := range s {
		if r == utf8.RuneError {
			// 跳过无效的UTF-8字符
			continue
		}
		result.WriteRune(r)
	}
	
	cleaned := result.String()
	if cleaned == "" {
		return "内容已清理"
	}
	return cleaned
}

// safeSendMessage 安全发送消息，确保UTF-8编码
func safeSendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) error {
	cleanedMessage := cleanUTF8String(message)
	return github.SendMessage(bot, chatID, cleanedMessage)
}

// HandleStartCommand 处理 /start 命令
func HandleStartCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	message := `你好！欢迎使用机器人。

使用方法：
1. 发送图片/视频，会自动弹出标签选择按钮
2. 发送文字消息，也会弹出标签选择按钮
3. 发送 /tags 查看所有可用标签
4. 发送 /label <标签名> 设置默认标签
5. 发送 /refresh 刷新标签列表
6. 发送 /edit 查看最近的动态列表
7. 发送 /edit <编号> 编辑指定动态
8. 发送 /delete 查看最近的动态列表
9. 发送 /delete <编号> 删除指定动态
10. 发送 /cancel 取消编辑

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）
• 发布后可以使用 /edit 命令编辑动态内容
• 可以使用 /delete 命令删除不需要的动态`
	return safeSendMessage(bot, update.Message.Chat.ID, message)
}

// HandleTagsCommand 处理 /tags 命令
func HandleTagsCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}

	// 先尝试从 GitHub 获取最新标签
	labels, err := github.GetGitHubLabels()
	if err != nil {
		log.Printf("获取 GitHub 标签失败: %v，使用缓存标签", err)
		// 获取失败时使用缓存标签
		labels = config.GetLabels()
	} else {
		// 获取成功，更新缓存
		config.SetLabels(labels)
	}
	
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
	_, err = bot.Send(msg)
	return err
}

// HandleRefreshCommand 处理 /refresh 命令
func HandleRefreshCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	if err := safeSendMessage(bot, update.Message.Chat.ID, "🔄 正在刷新标签列表..."); err != nil {
		return err
	}
	
	// 强制从 GitHub 获取最新标签
	labels, err := github.GetGitHubLabels()
	if err != nil {
		log.Printf("刷新标签失败: %v", err)
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 刷新标签失败，请稍后重试")
	}
	
	// 更新缓存
	config.SetLabels(labels)
	
	message := "✅ 标签列表已刷新！\n\n📋 可用标签：\n"
	for i, label := range labels {
		message += fmt.Sprintf("%d. %s\n", i+1, label)
	}
	
	return safeSendMessage(bot, update.Message.Chat.ID, message)
}

// HandleLabelCommand 处理 /label 命令
func HandleLabelCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	text := update.Message.Text
	if text == "" {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}
	
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 格式错误\n正确格式：/label <标签名>")
	}
	
	label := strings.Join(parts[1:], " ")
	
	// 先尝试从 GitHub 获取最新标签进行验证
	labels, err := github.GetGitHubLabels()
	if err != nil {
		log.Printf("获取 GitHub 标签失败: %v，使用缓存标签", err)
		// 获取失败时使用缓存标签
		labels = config.GetLabels()
	} else {
		// 获取成功，更新缓存
		config.SetLabels(labels)
	}
	
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
		return safeSendMessage(bot, update.Message.Chat.ID, message)
	}
	
	// 存储用户默认标签
	config.MediaMutex.Lock()
	if config.UserDefaultLabels == nil {
		config.UserDefaultLabels = make(map[int64]string)
	}
	config.UserDefaultLabels[update.Message.Chat.ID] = label
	config.MediaMutex.Unlock()
	
	return safeSendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("✅ 默认标签已设置为：%s", label))
}

// createLabelKeyboard 创建标签选择键盘
func createLabelKeyboard() tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// 先尝试从 GitHub 获取最新标签
	labels, err := github.GetGitHubLabels()
	if err != nil {
		log.Printf("获取 GitHub 标签失败: %v，使用缓存标签", err)
		// 获取失败时使用缓存标签
		labels = config.GetLabels()
	} else {
		// 获取成功，更新缓存
		config.SetLabels(labels)
	}
	
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
		return safeSendMessage(bot, callback.From.ID, fmt.Sprintf("📝 你的默认标签已设置为：%s\n下次发动态会自动带上该标签。", label))
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
		
		// 检查是否在编辑模式
		editState, inEditMode := config.GetEditState(callback.From.ID)
		
		if !exists && !inEditMode {
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 没有待处理的内容")
			bot.Send(msg)
			return nil
		}
		
		// 设置标签
		if inEditMode {
			// 编辑模式：更新编辑状态的标签
			config.EditMutex.Lock()
			editState.SelectedLabels = []string{label}
			config.EditStates[callback.From.ID] = editState
			config.EditMutex.Unlock()
			
			// 更新消息
			message := fmt.Sprintf("✅ 已选择标签：%s\n\n💡 现在可以发送新的内容来更新动态。", label)
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, message)
			bot.Send(msg)
			
			// 发送确认消息
			return safeSendMessage(bot, callback.From.ID, fmt.Sprintf("📝 标签已设置为：%s\n\n现在可以发送新的内容来更新动态！", label))
		} else {
			// 发布模式：设置待发布媒体的标签
			config.MediaMutex.Lock()
			pending.Labels = []string{label}
			config.PendingMedia[callback.From.ID] = pending
			config.MediaMutex.Unlock()
		}
		
		// 如果是文字消息，立即发布
		if pending.Type == "text" {
			// 更新消息
			message := fmt.Sprintf("✅ 已选择标签：%s\n\n⏳ 正在发布文字动态...", label)
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, message)
			bot.Send(msg)
			
			// 立即处理文字消息，不发送重复的进度消息
			return ProcessPendingMediaWithProgress(bot, callback.From.ID, "", false)
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
		return safeSendMessage(bot, callback.From.ID, fmt.Sprintf("📝 标签已设置为：%s\n\n现在可以发送文字来更新动态内容！", label))
	}
	
	// 处理删除确认回调
	if strings.HasPrefix(data, "delete:") {
		deleteAction := strings.TrimPrefix(data, "delete:")
		
		if deleteAction == "cancel" {
			// 取消删除
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 已取消删除")
			bot.Send(msg)
			return nil
		}
		
		if strings.HasPrefix(deleteAction, "confirm:") {
			// 确认删除
			issueNumberStr := strings.TrimPrefix(deleteAction, "confirm:")
			issueNumber, err := strconv.Atoi(issueNumberStr)
			if err != nil {
				msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "❌ 无效的动态编号")
				bot.Send(msg)
				return nil
			}
			
			// 更新消息显示正在删除
			msg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, "⏳ 正在删除动态...")
			bot.Send(msg)
			
			// 删除 GitHub Issue
			err = github.DeleteGitHubIssue(issueNumber)
			if err != nil {
				errorMsg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, fmt.Sprintf("❌ 删除失败：%v", err))
				bot.Send(errorMsg)
				return nil
			}
			
			// 从缓存中删除
			config.PublishedMutex.Lock()
			delete(config.PublishedMoments, issueNumber)
			config.PublishedMutex.Unlock()
			
			// 更新消息显示删除成功
			successMsg := tgbotapi.NewEditMessageText(callback.From.ID, callback.Message.MessageID, fmt.Sprintf("✅ 动态 #%d 已删除", issueNumber))
			bot.Send(successMsg)
			
			return nil
		}
	}
	
	return nil
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
3. 发送 /tags 查看所有可用标签
4. 发送 /label <标签名> 设置默认标签
5. 发送 /refresh 刷新标签列表
6. 发送 /edit 查看最近的动态列表
7. 发送 /edit <编号> 编辑指定动态
8. 发送 /delete 查看最近的动态列表
9. 发送 /delete <编号> 删除指定动态
10. 发送 /cancel 取消编辑

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）
• 发布后可以使用 /edit 命令编辑动态内容
• 可以使用 /delete 命令删除不需要的动态`
	return safeSendMessage(bot, update.Message.Chat.ID, message)
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
		} else if strings.HasPrefix(text, "/label") {
			return HandleLabelCommand(bot, update)
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

// HandleEditCommand 处理 /edit 命令
func HandleEditCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	text := update.Message.Text
	if text == "" {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}
	
	parts := strings.Fields(text)
	if len(parts) < 2 {
		// 显示最近的动态列表供选择
		return showRecentMoments(bot, update.Message.Chat.ID)
	}
	
	// 解析 Issue Number
	issueNumberStr := parts[1]
	issueNumber, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 无效的动态编号\n\n💡 发送 /edit 查看最近的动态列表")
	}
	
	// 获取动态内容
	moment, exists := config.GetPublishedMoment(issueNumber)
	if !exists {
		// 尝试从 GitHub 获取
		issue, err := github.GetGitHubIssue(issueNumber)
		if err != nil {
			return safeSendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("❌ 无法获取动态 #%d\n\n错误：%v", issueNumber, err))
		}
		
		// 创建动态对象并缓存
		moment = &types.PublishedMoment{
			IssueID:     issue.ID,
			IssueNumber: issue.Number,
			Content:     issue.Body,
			Labels:      []string{}, // 这里可以解析标签
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}
		config.AddPublishedMoment(moment)
	}
	
	// 设置编辑状态
	config.SetEditState(update.Message.Chat.ID, issueNumber, moment.Content, moment.Labels)
	
	// 创建标签选择键盘
	keyboard := createLabelKeyboard()
	
	message := fmt.Sprintf("✏️ 正在编辑动态 #%d\n\n", issueNumber)
	message += "📝 当前内容：\n"
	message += fmt.Sprintf("```\n%s\n```\n\n", moment.Content)
	
	// 显示当前标签
	if len(moment.Labels) > 0 {
		message += "🏷️ 当前标签："
		for i, label := range moment.Labels {
			if i > 0 {
				message += ", "
			}
			message += label
		}
		message += "\n\n"
	}
	
	message += "💡 请选择标签，然后发送新的内容来更新动态\n"
	message += "❌ 发送 /cancel 取消编辑"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, cleanUTF8String(message))
	msg.ReplyMarkup = keyboard
	_, sendErr := bot.Send(msg)
	return sendErr
}

// HandleCancelCommand 处理 /cancel 命令
func HandleCancelCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	if !config.IsInEditMode(update.Message.Chat.ID) {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 当前不在编辑模式")
	}
	
	config.ClearEditState(update.Message.Chat.ID)
	return safeSendMessage(bot, update.Message.Chat.ID, "✅ 已取消编辑")
}

// showRecentMoments 显示最近的动态列表
func showRecentMoments(bot *tgbotapi.BotAPI, chatID int64) error {
	issues, err := github.GetRecentIssues(10)
	if err != nil {
		return safeSendMessage(bot, chatID, fmt.Sprintf("❌ 获取动态列表失败：%v", err))
	}
	
	if len(issues) == 0 {
		return safeSendMessage(bot, chatID, "📝 暂无动态")
	}
	
	message := "📋 最近的动态列表：\n\n"
	for i, issue := range issues {
		// 截取内容预览
		preview := issue.Body
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		
		message += fmt.Sprintf("%d. #%d - %s\n", i+1, issue.Number, preview)
	}
	
	message += "\n💡 发送 /edit <编号> 编辑指定动态\n"
	message += "例如：/edit 123"
	
	return safeSendMessage(bot, chatID, message)
}

// HandleEditTextMessage 处理编辑模式下的文字消息
func HandleEditTextMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	// 检查是否在编辑模式
	editState, exists := config.GetEditState(update.Message.Chat.ID)
	if !exists {
		return nil // 不是编辑模式，交给普通文字处理
	}
	
	newContent := update.Message.Text
	if newContent == "" {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 内容不能为空")
	}
	
	// 清理用户输入的文字，确保UTF-8编码
	newContent = cleanUTF8String(newContent)
	
	// 检查内容长度
	if len(newContent) > 5000 {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 内容长度不能超过5000字符")
	}
	
	// 获取原始动态信息
	moment, exists := config.GetPublishedMoment(editState.IssueNumber)
	if !exists {
		// 尝试从 GitHub 获取
		issue, err := github.GetGitHubIssue(editState.IssueNumber)
		if err != nil {
			config.ClearEditState(update.Message.Chat.ID)
			return safeSendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("❌ 无法获取动态 #%d：%v", editState.IssueNumber, err))
		}
		
		moment = &types.PublishedMoment{
			IssueID:     issue.ID,
			IssueNumber: issue.Number,
			Content:     issue.Body,
			Labels:      []string{}, // 这里需要从GitHub API获取标签
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}
	}
	
	// 如果编辑状态中没有标签，使用原始标签
	if len(editState.SelectedLabels) == 0 {
		editState.SelectedLabels = moment.Labels
		if len(editState.SelectedLabels) == 0 {
			editState.SelectedLabels = []string{"动态"} // 默认标签
		}
	}
	
	// 发送更新进度消息
	if err := safeSendMessage(bot, update.Message.Chat.ID, "⏳ 正在更新动态..."); err != nil {
		return err
	}
	
	// 更新 GitHub Issue
	updatedIssue, err := github.UpdateGitHubIssue(editState.IssueNumber, newContent, editState.SelectedLabels)
	if err != nil {
		config.ClearEditState(update.Message.Chat.ID)
		return safeSendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("❌ 更新动态失败：%v", err))
	}
	
	// 更新缓存
	moment.Content = newContent
	moment.Labels = editState.SelectedLabels
	moment.UpdatedAt = time.Now().Unix()
	config.AddPublishedMoment(moment)
	
	// 清除编辑状态
	config.ClearEditState(update.Message.Chat.ID)
	
	successMessage := fmt.Sprintf("✅ 动态 #%d 更新成功！\n\n", editState.IssueNumber)
	successMessage += "📝 新内容：\n"
	successMessage += fmt.Sprintf("```\n%s\n```\n\n", newContent)
	
	// 显示标签
	if len(editState.SelectedLabels) > 0 {
		successMessage += "🏷️ 标签："
		for i, label := range editState.SelectedLabels {
			if i > 0 {
				successMessage += ", "
			}
			successMessage += label
		}
		successMessage += "\n\n"
	}
	
	successMessage += fmt.Sprintf("🔗 查看链接：%s", updatedIssue.HTMLURL)
	
	return safeSendMessage(bot, update.Message.Chat.ID, successMessage)
}

// HandleDeleteCommand 处理 /delete 命令
func HandleDeleteCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.Message.Chat.ID) {
		return nil
	}
	
	text := update.Message.Text
	if text == "" {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 无效的消息格式")
	}
	
	parts := strings.Fields(text)
	if len(parts) < 2 {
		// 显示最近的动态列表供选择
		return showRecentMomentsForDelete(bot, update.Message.Chat.ID)
	}
	
	// 解析 Issue Number
	issueNumberStr := parts[1]
	issueNumber, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return safeSendMessage(bot, update.Message.Chat.ID, "❌ 无效的动态编号\n\n💡 发送 /delete 查看最近的动态列表")
	}
	
	// 获取动态内容
	moment, exists := config.GetPublishedMoment(issueNumber)
	if !exists {
		// 尝试从 GitHub 获取
		issue, err := github.GetGitHubIssue(issueNumber)
		if err != nil {
			return safeSendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("❌ 无法获取动态 #%d\n\n错误：%v", issueNumber, err))
		}
		
		// 创建动态对象并缓存
		moment = &types.PublishedMoment{
			IssueID:     issue.ID,
			IssueNumber: issue.Number,
			Content:     issue.Body,
			Labels:      []string{},
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}
		config.AddPublishedMoment(moment)
	}
	
	// 创建确认删除的键盘
	var buttons [][]tgbotapi.InlineKeyboardButton
	confirmRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("✅ 确认删除", fmt.Sprintf("delete:confirm:%d", issueNumber)),
		tgbotapi.NewInlineKeyboardButtonData("❌ 取消", "delete:cancel"),
	}
	buttons = append(buttons, confirmRow)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	
	message := fmt.Sprintf("🗑️ 确认删除动态 #%d？\n\n", issueNumber)
	message += "📝 动态内容：\n"
	
	// 截取内容预览
	preview := moment.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	message += fmt.Sprintf("```\n%s\n```\n\n", preview)
	
	// 显示标签
	if len(moment.Labels) > 0 {
		message += "🏷️ 标签："
		for i, label := range moment.Labels {
			if i > 0 {
				message += ", "
			}
			message += label
		}
		message += "\n\n"
	}
	
	message += "⚠️ 删除后无法恢复，请确认！"
	
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, cleanUTF8String(message))
	msg.ReplyMarkup = keyboard
	_, sendErr := bot.Send(msg)
	return sendErr
}

// showRecentMomentsForDelete 显示最近的动态列表（用于删除）
func showRecentMomentsForDelete(bot *tgbotapi.BotAPI, chatID int64) error {
	issues, err := github.GetRecentIssues(10)
	if err != nil {
		return safeSendMessage(bot, chatID, fmt.Sprintf("❌ 获取动态列表失败：%v", err))
	}
	
	if len(issues) == 0 {
		return safeSendMessage(bot, chatID, "📝 暂无动态")
	}
	
	message := "🗑️ 选择要删除的动态：\n\n"
	for _, issue := range issues {
		// 截取内容预览
		preview := issue.Body
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		
		message += fmt.Sprintf("#%d - %s\n", issue.Number, preview)
	}
	
	message += "\n💡 发送 /delete <编号> 删除指定动态\n"
	message += "例如：/delete 123"
	
	return safeSendMessage(bot, chatID, message)
} 
