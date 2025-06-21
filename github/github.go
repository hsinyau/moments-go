package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"moments-go/config"
	"moments-go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GitHubLabel GitHub 标签结构
type GitHubLabel struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// GetGitHubLabels 从 GitHub 获取所有标签
func GetGitHubLabels() ([]string, error) {
	url := "https://api.github.com/repos/hsinyau/moments/labels"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var labels []GitHubLabel
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取标签名称
	var labelNames []string
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}

	// 如果没有标签，返回默认标签
	if len(labelNames) == 0 {
		return types.DefaultLabels, nil
	}

	return labelNames, nil
}

// CreateGitHubIssue 创建 GitHub Issue
func CreateGitHubIssue(content string) (*types.GitHubIssueResponse, error) {
	return CreateGitHubIssueWithLabels(content, []string{"动态"})
}

// CreateGitHubIssueWithLabels 创建带标签的 GitHub Issue
func CreateGitHubIssueWithLabels(content string, labels []string) (*types.GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	issueData := map[string]interface{}{
		"title": timestamp,
		"body":  content,
		"labels": labels,
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

// UploadFileToGitHub 上传文件到 GitHub
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

// GetGitHubIssue 获取 GitHub Issue
func GetGitHubIssue(issueNumber int) (*types.GitHubIssueResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/moments/issues/%d", issueNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var issue types.GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &issue, nil
}

// UpdateGitHubIssue 更新 GitHub Issue
func UpdateGitHubIssue(issueNumber int, content string, labels []string) (*types.GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}

	updateData := map[string]interface{}{
		"body":   content,
		"labels": labels,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/moments/issues/%d", issueNumber)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var issue types.GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &issue, nil
}

// GetRecentIssues 获取最近的动态列表
func GetRecentIssues(limit int) ([]types.GitHubIssueResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/moments/issues?state=open&per_page=%d&sort=created&direction=desc", limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var issues []types.GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return issues, nil
}

// DeleteGitHubIssue 删除 GitHub Issue
func DeleteGitHubIssue(issueNumber int) error {
	url := fmt.Sprintf("https://api.github.com/repos/hsinyau/moments/issues/%d", issueNumber)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer([]byte(`{"state":"closed"}`)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.Cfg.GitHubSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hsinyau-bot/1.0")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return nil
} 
