package handlers

import (
	"moments-go/config"
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
3. 发送 /tags 查看所有可用标签
4. 发送 /refresh 刷新标签列表
5. 发送 /edit <编号> 编辑指定动态
6. 发送 /delete <编号> 删除指定动态
7. 发送 /cancel 取消编辑

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）
• 发布后可以使用 /edit 命令编辑动态内容
• 可以使用 /delete 命令删除不需要的动态`
	return safeSendMessage(bot, update.Message.Chat.ID, message)
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
4. 发送 /refresh 刷新标签列表
5. 发送 /edit <编号> 编辑指定动态
6. 发送 /delete <编号> 删除指定动态
7. 发送 /cancel 取消编辑

💡 提示：
• 发送媒体文件或文字后，选择标签即可发布动态
• 选择标签后，可以继续发送文字来更新动态内容
• 媒体文件会在5分钟后自动发布（如果未手动发布）
• 发布后可以使用 /edit 命令编辑动态内容
• 可以使用 /delete 命令删除不需要的动态`
	return safeSendMessage(bot, update.Message.Chat.ID, message)
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
