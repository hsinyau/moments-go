package github

import (
	"fmt"
	"moments-go/config"
	"moments-go/types"
)

// GetGitHubLabels 从 GitHub 获取所有标签
func GetGitHubLabels() ([]string, error) {
	client := NewGitHubClient()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/labels", config.Cfg.GitHubUsername, config.Cfg.GitHubRepo)
	
	resp, err := client.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var labels []GitHubLabel
	if err := client.handleResponse(resp, &labels); err != nil {
		return nil, err
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
