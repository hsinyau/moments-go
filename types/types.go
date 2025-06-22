package types

type GitHubUploadResponse struct {
	Content *struct {
		DownloadURL string `json:"download_url"`
	} `json:"content"`
}

type GitHubIssueResponse struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

type MediaFile struct {
	Name    string
	Content []byte
	Type    string
}

type PendingMedia struct {
	FileID  string
	Type    string
	Caption string
	Labels  []string
}

// PublishedMoment 已发布的动态
type PublishedMoment struct {
	IssueID   int      `json:"issue_id"`
	IssueNumber int    `json:"issue_number"`
	Content   string   `json:"content"`
	Labels    []string `json:"labels"`
	MediaURLs []string `json:"media_urls"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

type Config struct {
	TelegramBotToken string
	TelegramUserID   int64
	GitHubSecret     string
	GitHubFileRepo   string
	GitHubUsername   string
	GitHubRepo       string
	GitHubUserAgent  string
}

var DefaultLabels = []string{
	"动态",
	"日常",
	"其他",
} 
