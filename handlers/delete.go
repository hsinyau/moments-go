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
