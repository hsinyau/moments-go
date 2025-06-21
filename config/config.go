package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"moments-go/types"
)

const (
	WaitTime    = 5 * 60 // 5分钟等待时间（秒）
	MaxFileSize = 50 * 1024 * 1024 // 50MB
	LabelCacheTime = 30 * 60 // 标签缓存时间（30分钟）
)

var (
	Cfg             types.Config
	PendingMedia    = make(map[int64]*types.PendingMedia)
	UserDefaultLabels = make(map[int64]string) // 用户默认标签
	MediaMutex      sync.RWMutex
	
	// 标签缓存
	LabelsCache     []string
	LabelsCacheTime time.Time
	LabelsMutex     sync.RWMutex
)

func LoadConfig() error {
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 无法加载 .env 文件")
	}

	Cfg.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if Cfg.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN 未设置")
	}

	userIDStr := os.Getenv("TELEGRAM_USER_ID")
	if userIDStr == "" {
		return fmt.Errorf("TELEGRAM_USER_ID 未设置")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的 TELEGRAM_USER_ID: %v", err)
	}
	Cfg.TelegramUserID = userID

	Cfg.GitHubSecret = os.Getenv("GITHUB_SECRET")
	if Cfg.GitHubSecret == "" {
		return fmt.Errorf("GITHUB_SECRET 未设置")
	}

	Cfg.GitHubFileRepo = os.Getenv("GITHUB_FILE_REPO")
	if Cfg.GitHubFileRepo == "" {
		Cfg.GitHubFileRepo = "moments-files" // 默认值
	}

	return nil
}

func IsAuthorizedUser(chatID int64) bool {
	return chatID == Cfg.TelegramUserID
}

// GetLabels 获取标签列表（带缓存）
func GetLabels() []string {
	LabelsMutex.RLock()
	if time.Since(LabelsCacheTime) < LabelCacheTime*time.Second && len(LabelsCache) > 0 {
		defer LabelsMutex.RUnlock()
		return LabelsCache
	}
	LabelsMutex.RUnlock()
	
	// 缓存过期，重新获取
	return types.DefaultLabels
}

// SetLabels 设置标签缓存
func SetLabels(labels []string) {
	LabelsMutex.Lock()
	defer LabelsMutex.Unlock()
	LabelsCache = labels
	LabelsCacheTime = time.Now()
} 
