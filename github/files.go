package github

import (
	"encoding/base64"
	"fmt"
	"moments-go/config"
	"moments-go/types"
	"strconv"
	"time"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UploadFileToGitHub 上传文件到 GitHub
func UploadFileToGitHub(file *types.MediaFile, timestamp string) (string, error) {
	client := NewGitHubClient()
	base64Content := base64.StdEncoding.EncodeToString(file.Content)
	
	uploadData := map[string]interface{}{
		"message": fmt.Sprintf("Add media file: %s", file.Name),
		"content": base64Content,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/moments/%s_%s", 
		config.Cfg.GitHubUsername, config.Cfg.GitHubFileRepo, timestamp, file.Name)
	
	resp, err := client.makeRequest("PUT", url, uploadData)
	if err != nil {
		return "", err
	}

	var uploadResult types.GitHubUploadResponse
	if err := client.handleResponse(resp, &uploadResult); err != nil {
		return "", err
	}

	if uploadResult.Content == nil || uploadResult.Content.DownloadURL == "" {
		return "", fmt.Errorf("文件 %s 上传失败", file.Name)
	}

	return uploadResult.Content.DownloadURL, nil
}

// UploadToGitHub 上传媒体文件到 GitHub 并发布动态
func UploadToGitHub(bot *tgbotapi.BotAPI, content string, mediaFiles []*types.MediaFile) (*types.GitHubIssueResponse, error) {
	return UploadToGitHubWithLabels(bot, content, mediaFiles, []string{"动态"})
}

// UploadToGitHubWithLabels 上传媒体文件到 GitHub 并发布带标签的动态
func UploadToGitHubWithLabels(bot *tgbotapi.BotAPI, content string, mediaFiles []*types.MediaFile, labels []string) (*types.GitHubIssueResponse, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	var mediaUrls []string

	if len(mediaFiles) > 0 {
		if err := SendMessage(bot, config.Cfg.TelegramUserID, "📤 正在上传媒体文件..."); err != nil {
			return nil, err
		}

		for _, file := range mediaFiles {
			downloadURL, err := UploadFileToGitHub(file, timestamp)
			if err != nil {
				return nil, fmt.Errorf("上传文件 %s 失败: %v", file.Name, err)
			}
			mediaUrls = append(mediaUrls, downloadURL)
		}
	}

	fullContent := content
	if len(mediaUrls) > 0 {
		for _, url := range mediaUrls {
			fullContent += fmt.Sprintf("\n![%s](%s)", url, url)
		}
	}

	return CreateGitHubIssueWithLabels(fullContent, labels)
} 
