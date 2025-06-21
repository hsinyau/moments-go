package telegram

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"moments-go/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func DownloadFile(bot *tgbotapi.BotAPI, fileID string) ([]byte, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}
	if file.FilePath == "" {
		return nil, fmt.Errorf("无法获取文件路径")
	}
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.Cfg.TelegramBotToken, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %v", err)
	}
	return content, nil
}

func ScheduleMediaPublish(bot *tgbotapi.BotAPI, chatID int64, callback func()) {
	time.AfterFunc(time.Duration(config.WaitTime)*time.Second, callback)
} 
