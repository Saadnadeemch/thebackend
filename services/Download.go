package services

import (
	"backend/models"
	"backend/sse"
	util "backend/utils"
	runner "backend/yt-dlp"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func DownloadService(req models.DownloadVideoRequest) (*models.VideoDownloadResult, error) {

	result, err := downloadWithDynamicCommand(req)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(result.FilePath); err != nil {
		return nil, fmt.Errorf("final file not found: %w", err)
	}

	return result, nil
}

func downloadWithDynamicCommand(
	request models.DownloadVideoRequest,
) (*models.VideoDownloadResult, error) {

	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "prodl_" + request.RequestID[:8]
	}

	safeTitle := util.SanitizedFileName(title)

	if err := util.EnsureRootDirectory(); err != nil {
		return nil, fmt.Errorf("failed to ensure directory: %w", err)
	}

	ext := "mp3"
	if !request.OriginalReq.AudioOnly {
		ext = "mp4"
	}

	outputPath := filepath.Join("downloads", fmt.Sprintf("%s.%s", safeTitle, ext))

	sse.Send(request.RequestID, map[string]interface{}{
		"status":  "initializing",
		"message": "Preparing download",
		"percent": 0,
	})

	ctx := context.Background()

	args := buildYTArgs(request, outputPath)

	log.Printf("[DownloadService] YT-DLP ARGS:\n__\n%s\n__\n", strings.Join(args, " "))

	err := runner.RunYTDownloadWithProgress(ctx, args, request.RequestID)
	if err != nil {
		sse.Send(request.RequestID, map[string]interface{}{
			"status":  "error",
			"message": "Download failed",
			"percent": 0,
		})
		return nil, fmt.Errorf("yt-dlp execution failed: %w", err)
	}

	sse.Send(request.RequestID, map[string]interface{}{
		"status":  "completed",
		"message": "Download completed",
		"percent": 100,
	})

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("file not found after download: %w", err)
	}

	return &models.VideoDownloadResult{
		RequestID:   request.RequestID,
		FilePath:    outputPath,
		FileName:    filepath.Base(outputPath),
		Title:       title,
		DownloadURL: "/downloads/" + filepath.Base(outputPath),
		CleanupAt:   util.EstimateCleanupTime(fileInfo.Size()),
	}, nil
}

func buildYTArgs(
	request models.DownloadVideoRequest,
	outputPath string,
) []string {

	var args []string

	args = append(args,
		"--no-playlist",
		"--cookies-from-browser", "firefox",
		"--newline",
		"-o", outputPath,
	)

	if request.OriginalReq.AudioOnly {

		args = append(args,
			"-f", "bestaudio",
			"--extract-audio",
			"--audio-format", "mp3",
			"--concurrent-fragments", "4",
		)

	} else {

		fragments := util.GetFragmentsByQuality(request.VideoQuality)

		args = append(args,
			"-f", request.VideoQuality,
			"--concurrent-fragments", fragments,
			"--merge-output-format", "mp4",
		)
	}

	args = append(args, request.URL)

	return args
}
