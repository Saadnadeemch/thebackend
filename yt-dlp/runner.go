package ytdlp

import (
	"backend/models"
	"encoding/json"
	"fmt"
	"os/exec"
)

func GetDirectInfoFromYTDLP(videoURL string) (*models.VideoInfo, error) {

	cmd := exec.Command(
		"yt-dlp",
		"-j",
		"--no-playlist",
		"--no-warnings",
		"--no-check-certificate",
		videoURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp exec error: %v | output: %s", err, string(output))
	}

	var data struct {
		Title       string  `json:"title"`
		Uploader    string  `json:"uploader"`
		Thumbnail   string  `json:"thumbnail"`
		ViewCount   int64   `json:"view_count"`
		Description *string `json:"description"`
		UploadDate  *string `json:"upload_date"`
		LikeCount   *int64  `json:"like_count"`
		URL         *string `json:"url"`
	}

	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("yt-dlp parse error: %v | raw: %s", err, string(output))
	}

	return &models.VideoInfo{
		Title:       data.Title,
		Uploader:    data.Uploader,
		Thumbnail:   data.Thumbnail,
		Views:       data.ViewCount,
		Description: data.Description,
		UploadDate:  data.UploadDate,
		LikeCount:   data.LikeCount,
		DownloadURL: data.URL,
		VideoPage:   videoURL,
	}, nil
}
