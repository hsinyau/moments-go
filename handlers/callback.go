package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"moments-go/config"
	"moments-go/github"
	"moments-go/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleCallbackQuery 处理回调查询（标签选择）
func HandleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	if !config.IsAuthorizedUser(update.CallbackQuery.From.ID) {
		return nil
	}
	
	callback := update.CallbackQuery
	data := callback.Data
	
	if strings.HasPrefix(data, "label:") {
		return handleLabelCallback(bot, callback)
	}
	
	// 处理删除确认回调
	if strings.HasPrefix(data, "delete:") {
		return handleDeleteCallback(bot, callback)
	}
	
	return nil
}

// handleLabelCallback 处理标签选择回调
func handleLabelCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) error {
	label := strings.TrimPrefix(callback.Data, "label:")
	
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

// handleDeleteCallback 处理删除确认回调
func handleDeleteCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) error {
	deleteAction := strings.TrimPrefix(callback.Data, "delete:")
	
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
	
	return nil
} 
