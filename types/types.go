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
}

type Config struct {
	TelegramBotToken string
	TelegramUserID   int64
	GitHubSecret     string
	GitHubFileRepo   string
} 
