package github

import (
	"strings"
	"unicode/utf8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendMessage 发送消息
func SendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) error {
	// 清理消息，确保UTF-8编码
	cleanedMessage := cleanUTF8String(message)
	msg := tgbotapi.NewMessage(chatID, cleanedMessage)
	_, err := bot.Send(msg)
	return err
}

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
