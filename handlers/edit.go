package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"moments-go/config"
	"moments-go/github"
	"moments-go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
