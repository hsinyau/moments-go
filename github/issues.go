package github

import (
	"fmt"
	"moments-go/types"
	"strconv"
	"time"
	"moments-go/config"
)

// CreateGitHubIssue 创建 GitHub Issue
func CreateGitHubIssue(content string) (*types.GitHubIssueResponse, error) {
	return CreateGitHubIssueWithLabels(content, []string{"动态"})
}

// CreateGitHubIssueWithLabels 创建带标签的 GitHub Issue
func CreateGitHubIssueWithLabels(content string, labels []string) (*types.GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}

	client := NewGitHubClient()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	
	issueData := map[string]interface{}{
		"title":  timestamp,
		"body":   content,
		"labels": labels,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo)
	resp, err := client.makeRequest("POST", url, issueData)
	if err != nil {
		return nil, err
	}

	var issue types.GitHubIssueResponse
	if err := client.handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

// GetGitHubIssue 获取 GitHub Issue
func GetGitHubIssue(issueNumber int) (*types.GitHubIssueResponse, error) {
	client := NewGitHubClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo, issueNumber)
	
	resp, err := client.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var issue types.GitHubIssueResponse
	if err := client.handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

// UpdateGitHubIssue 更新 GitHub Issue
func UpdateGitHubIssue(issueNumber int, content string, labels []string) (*types.GitHubIssueResponse, error) {
	if len(content) > 5000 {
		return nil, fmt.Errorf("内容长度不能超过5000字符")
	}

	client := NewGitHubClient()
	updateData := map[string]interface{}{
		"body":   content,
		"labels": labels,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo, issueNumber)
	resp, err := client.makeRequest("PATCH", url, updateData)
	if err != nil {
		return nil, err
	}

	var issue types.GitHubIssueResponse
	if err := client.handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

// GetRecentIssues 获取最近的动态列表
func GetRecentIssues(limit int) ([]types.GitHubIssueResponse, error) {
	client := NewGitHubClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open&per_page=%d&sort=created&direction=desc", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo, limit)
	
	resp, err := client.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var issues []types.GitHubIssueResponse
	if err := client.handleResponse(resp, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// DeleteGitHubIssue 删除 GitHub Issue
func DeleteGitHubIssue(issueNumber int) error {
	client := NewGitHubClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo, issueNumber)
	
	resp, err := client.makeRequest("PATCH", url, map[string]string{"state": "closed"})
	if err != nil {
		return err
	}

	return client.handleResponse(resp, nil)
} 
