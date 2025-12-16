package models

// incomming request from the client
type Request struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

type AudioRequest struct {
	URL       string `json:"url"`
	RequestID string `json:"request_id"`
	Title     string
}

type DownloadVideoRequest struct {
	URL       string `json:"url"`
	Quality   string `json:"quality"`
	RequestID string `json:"request_id"`
	FormatID  string `json:"format_id"`
	Title     string `json:"title"`
	Method    string
}

type Format struct {
	FormatID   string `json:"format_id"`
	Extension  string `json:"ext"`
	Resolution string `json:"resolution"`
	Vcodec     string `json:"vcodec"`
	Acodec     string `json:"acodec"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	Protocol   string
}

type VideoInfo struct {
	Title       string  `json:"title"`
	Thumbnail   string  `json:"thumbnail"`
	Uploader    string  `json:"uploader"`
	Views       int64   `json:"views"`
	Source      string  `json:"source"`
	DownloadURL *string `json:"downloadUrl,omitempty"`
	Description *string `json:"description,omitempty"`
	UploadDate  *string `json:"upload_date,omitempty"`
	LikeCount   *int64  `json:"likes,omitempty"`
	VideoPage   string  `json:"url"`
}

// DownloadProgress is sent via WebSocket during download.
type DownloadProgress struct {
	RequestID      string  `json:"request_id"`
	Progress       float64 `json:"progress"`          // 0.0 - 1.0
	Speed          float64 `json:"speed"`             // bytes/sec
	DownloadedSize int64   `json:"downloaded_size"`   // bytes
	TotalSize      int64   `json:"total_size"`        // bytes
	Status         string  `json:"status"`            // e.g. "downloading", "merging", "done"
	Message        string  `json:"message,omitempty"` // extra info or errors
}

// VideoDownloadResult is returned after a successful download.
type VideoDownloadResult struct {
	RequestID   string `json:"request_id"`
	FilePath    string `json:"file_path"`
	Title       string `json:"title"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	CleanupAt   int64  `json:"cleanup_at"`
}

// PlatformInfo describes the platform characteristics.
type PlatformInfo struct {
	Platform       string // e.g. "YouTube"
	IsSupported    bool   // Is it supported?
	Reason         string // If unsupported or unknown, reason why
	DownloadMethod string
}

type StreamVideoDownloadRequest struct {
	URL string `json:"url"`
}

type YTDLPINFO struct {
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	Thumbnail   string  `json:"thumbnail"`
	ViewCount   int64   `json:"view_count"`
	Description *string `json:"description"`
	UploadDate  *string `json:"upload_date"`
	LikeCount   *int64  `json:"likes"`
	URL         *string `json:"url"`
}
