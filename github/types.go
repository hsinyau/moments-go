package github

// GitHubLabel GitHub 标签结构
type GitHubLabel struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
} 
