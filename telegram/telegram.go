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
	// 添加重试机制
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("获取文件信息失败: %v", err)
			}
			fmt.Printf("获取文件信息失败，第%d次尝试: %v，等待重试...\n", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		if file.FilePath == "" {
			return nil, fmt.Errorf("无法获取文件路径")
		}
		
		fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.Cfg.TelegramBotToken, file.FilePath)
		
		// 创建带超时的HTTP客户端
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		
		resp, err := client.Get(fileURL)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("下载文件失败: %v", err)
			}
			fmt.Printf("下载文件失败，第%d次尝试: %v，等待重试...\n", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			if attempt == maxRetries {
				return nil, fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
			}
			fmt.Printf("下载文件失败，状态码: %d，第%d次尝试，等待重试...\n", resp.StatusCode, attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("读取文件内容失败: %v", err)
			}
			fmt.Printf("读取文件内容失败，第%d次尝试: %v，等待重试...\n", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		return content, nil
	}
	
	return nil, fmt.Errorf("下载文件失败，已重试%d次", maxRetries)
}

func ScheduleMediaPublish(bot *tgbotapi.BotAPI, chatID int64, callback func()) {
	time.AfterFunc(time.Duration(config.WaitTime)*time.Second, callback)
} 
