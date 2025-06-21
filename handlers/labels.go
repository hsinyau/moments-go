package handlers

import (
	"fmt"
	"log"
	"moments-go/config"
	"moments-go/github"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	
	message := "📋 可用标签：\n"
	for i, label := range labels {
		message += fmt.Sprintf("%d. %s\n", i+1, label)
	}
	
	return safeSendMessage(bot, update.Message.Chat.ID, message)
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
