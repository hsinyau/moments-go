package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"moments-go/config"
)

// GitHubClient GitHub API 客户端
type GitHubClient struct {
	client *http.Client
	token  string
}

// NewGitHubClient 创建新的 GitHub 客户端
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: &http.Client{},
		token:  config.Cfg.GitHubSecret,
	}
}

// makeRequest 发送 HTTP 请求到 GitHub API
func (c *GitHubClient) makeRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化数据失败: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", config.Cfg.GitHubUserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}

	return resp, nil
}

// handleResponse 处理 HTTP 响应
func (c *GitHubClient) handleResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("解析响应失败: %v", err)
		}
	}

	return nil
} 
