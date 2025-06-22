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
	PublishedMomentCacheTime = 24 * 60 * 60 // 已发布动态缓存时间（24小时）
)

var (
	Cfg             types.Config
	PendingMedia    = make(map[int64]*types.PendingMedia)
	MediaMutex      sync.RWMutex
	
	// 标签缓存
	LabelsCache     []string
	LabelsCacheTime time.Time
	LabelsMutex     sync.RWMutex
	
	// 已发布动态缓存
	PublishedMoments = make(map[int]*types.PublishedMoment) // IssueNumber -> PublishedMoment
	PublishedMutex   sync.RWMutex
	
	// 编辑状态管理
	EditStates = make(map[int64]*EditState) // ChatID -> EditState
	EditMutex  sync.RWMutex
)

// EditState 编辑状态
type EditState struct {
	IssueNumber int      `json:"issue_number"`
	OriginalContent string `json:"original_content"`
	OriginalLabels []string `json:"original_labels"`
	SelectedLabels []string `json:"selected_labels"`
	StartTime   int64    `json:"start_time"`
}

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

	// 新增 GitHub 配置
	Cfg.GitHubUsername = os.Getenv("GITHUB_USERNAME")
	if Cfg.GitHubUsername == "" {
		return fmt.Errorf("GITHUB_USERNAME 未设置")
	}

	Cfg.GitHubRepo = os.Getenv("GITHUB_REPO")
	if Cfg.GitHubRepo == "" {
		Cfg.GitHubRepo = "moments" // 默认值
	}

	Cfg.GitHubUserAgent = os.Getenv("GITHUB_USER_AGENT")
	if Cfg.GitHubUserAgent == "" {
		Cfg.GitHubUserAgent = "moments-bot/1.0" // 默认值
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

// AddPublishedMoment 添加已发布的动态到缓存
func AddPublishedMoment(moment *types.PublishedMoment) {
	PublishedMutex.Lock()
	defer PublishedMutex.Unlock()
	PublishedMoments[moment.IssueNumber] = moment
}

// GetPublishedMoment 获取已发布的动态
func GetPublishedMoment(issueNumber int) (*types.PublishedMoment, bool) {
	PublishedMutex.RLock()
	defer PublishedMutex.RUnlock()
	moment, exists := PublishedMoments[issueNumber]
	return moment, exists
}

// SetEditState 设置编辑状态
func SetEditState(chatID int64, issueNumber int, originalContent string, originalLabels []string) {
	EditMutex.Lock()
	defer EditMutex.Unlock()
	EditStates[chatID] = &EditState{
		IssueNumber: issueNumber,
		OriginalContent: originalContent,
		OriginalLabels: originalLabels,
		SelectedLabels: originalLabels, // 默认使用原标签
		StartTime: time.Now().Unix(),
	}
}

// GetEditState 获取编辑状态
func GetEditState(chatID int64) (*EditState, bool) {
	EditMutex.RLock()
	defer EditMutex.RUnlock()
	state, exists := EditStates[chatID]
	return state, exists
}

// ClearEditState 清除编辑状态
func ClearEditState(chatID int64) {
	EditMutex.Lock()
	defer EditMutex.Unlock()
	delete(EditStates, chatID)
}

// IsInEditMode 检查是否处于编辑模式
func IsInEditMode(chatID int64) bool {
	EditMutex.RLock()
	defer EditMutex.RUnlock()
	_, exists := EditStates[chatID]
	return exists
} 
