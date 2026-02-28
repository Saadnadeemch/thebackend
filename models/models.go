package models

type VideoType string

const (
	VideoTypeReel  VideoType = "reel"
	VideoTypeVideo VideoType = "video"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type Request struct {
	URL         string `json:"url"`
	Quality     string `json:"quality"`
	AudioOnly   bool   `json:"audio_only,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	CloudUpload bool   `json:"cloud_upload,omitempty"`
}

type DownloadVideoRequest struct {
	OriginalReq  Request `json:"original_req"`
	URL          string  `json:"url"`
	RequestID    string  `json:"request_id"`
	VideoQuality string  `json:"video_quality"`
	Title        string  `json:"title"`
	Platform     string  `json:"platform"`
	VideoType    string  `json:"video_type"`
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
	Description *string `json:"description,omitempty"`
	UploadDate  *string `json:"upload_date,omitempty"`
	LikeCount   *int64  `json:"likes,omitempty"`
	VideoPage   string  `json:"url"`
}

type DownloadProgress struct {
	RequestID      string  `json:"request_id"`
	Progress       float64 `json:"progress"`
	Speed          float64 `json:"speed"`
	DownloadedSize int64   `json:"downloaded_size"`
	TotalSize      int64   `json:"total_size"`
	Status         string  `json:"status"`
	Message        string  `json:"message,omitempty"`
}

type VideoDownloadResult struct {
	RequestID   string `json:"request_id"`
	FilePath    string `json:"file_path"`
	Title       string `json:"title"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	CleanupAt   int64  `json:"cleanup_at"`
}

type PlatformInfo struct {
	Platform   string
	VideoType  VideoType
	Confidence Confidence
}

type StreamVideoDownloadRequest struct {
	URL string `json:"url"`
}

type YtdlpInfo struct {
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	Thumbnail   string  `json:"thumbnail"`
	ViewCount   int64   `json:"view_count"`
	Description *string `json:"description"`
	UploadDate  *string `json:"upload_date"`
	LikeCount   *int64  `json:"likes"`
	URL         *string `json:"url"`
}
