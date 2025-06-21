package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"moments-go/config"
	"moments-go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CreateGitHubIssue(content string) (*types.GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	issueData := map[string]interface{}{
		"title": timestamp,
		"body":  content,
		"labels": []string{"动态"},
	}
	jsonData, err := json.Marshal(issueData)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}
	req, err := http.NewRequest("POST", "https://api.github.com/repos/hsinyau/moments/issues", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	var issue types.GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	return &issue, nil
}

func UploadFileToGitHub(file *types.MediaFile, timestamp string) (string, error) {
	base64Content := base64.StdEncoding.EncodeToString(file.Content)
	uploadData := map[string]interface{}{
		"message": fmt.Sprintf("Add media file: %s", file.Name),
		"content": base64Content,
	}
	jsonData, err := json.Marshal(uploadData)
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %v", err)
	}
	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/%s/contents/moments/%s_%s", config.Cfg.GitHubFileRepo, timestamp, file.Name)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	var uploadResult types.GitHubUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}
	if uploadResult.Content == nil || uploadResult.Content.DownloadURL == "" {
		return "", fmt.Errorf("文件 %s 上传失败", file.Name)
	}
	return uploadResult.Content.DownloadURL, nil
}

func UploadToGitHub(bot *tgbotapi.BotAPI, content string, mediaFiles []*types.MediaFile) (*types.GitHubIssueResponse, error) {
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
	return CreateGitHubIssue(fullContent)
}

func SendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	return err
} 
