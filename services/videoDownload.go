package services

import (
	"backend/models"
	util "backend/utils"
	webSocketMain "backend/websocket"
	runner "backend/yt-dlp"
	"context"
	"fmt"
	"log"
	"os"

	"path/filepath"
	"strings"
)

func DownloadVideo(req models.DownloadVideoRequest, conn *webSocketMain.WSConnection) (*models.VideoDownloadResult, error) {

	if err := util.EnsureRootDirectory(); err != nil {
		return nil, fmt.Errorf("failed to ensure download directory: %w", err)
	}

	if webSocketMain.IsRequestAborted(req.RequestID) {
		webSocketMain.SendSimpleProgress(conn, req.RequestID, "aborted", "Download aborted before download started", 0)
		return nil, fmt.Errorf("request aborted before starting: %s", req.RequestID)
	}

	//Starting Downlaod with Logs
	// log.Printf("[WS_ID: %s]  Starting Separate AV download | URL=%s | Quality=%s",
	// 	req.RequestID, req.URL, req.Quality)

	result, err := DownloadAndMergeYTAV(req, conn)
	if err != nil {
		return nil, fmt.Errorf("DownloadAndMergeYTAV failed: %w", err)
	}

	//Check final path exists or not
	if _, err := os.Stat(result.FilePath); err != nil {
		return nil, fmt.Errorf("merged file not found: %w", err)
	}

	return result, nil
}

func DownloadAndMergeYTAV(
	request models.DownloadVideoRequest,
	ws *webSocketMain.WSConnection,
) (*models.VideoDownloadResult, error) {
	url := request.URL
	format := request.FormatID
	method := request.Method

	log.Printf("[DL] starting DownloadAndMergeYTAV request=%s url=%s format=%s", request.RequestID, url, format)

	// Ensure title is safe
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "prodownloader_" + request.RequestID[:8]
	}
	safeTitle := util.SanitizedFileName(title)

	// Ensure output dir exists
	outputDir := "downloads"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Printf("[DL] mkdir error: %v", err)
	}

	// Final merged file path (force .mp4)
	mergedOut := filepath.Join(outputDir, fmt.Sprintf("%s_prodownloader.mp4", safeTitle))

	webSocketMain.SendSimpleProgress(ws, request.RequestID, "Starting Download", "Getting things ready", 0)

	// Context + cancel handling
	ctx, cancel := context.WithCancel(context.Background())
	util.RegisterCancelFunc(request.RequestID, cancel)
	defer func() {
		cancel()
		util.CleanupCancelFunc(request.RequestID)
	}()

	// If format = best → direct yt-dlp download
	if format == "best" || method == "default" {
		log.Printf("[DL] format == best, doing simple download to %s", mergedOut)
		return runner.Downloadbestformat(ctx, url, mergedOut, request, ws)
	}

	// Otherwise, combine requested video format with bestaudio
	fullFormat := fmt.Sprintf("%s+bestaudio", format)
	log.Printf("[DL] downloading combined format=%s", fullFormat)

	// Run yt-dlp with forced mp4 merge
	err := runner.RunYTDownloadWithProgressContextWithMerge(ctx, fullFormat, mergedOut, url, request, ws)
	if err != nil {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "Download failed", 100)
		return nil, fmt.Errorf("DownloadAndMergeYTAV failed: %w", err)
	}

	webSocketMain.SendSimpleProgress(ws, request.RequestID, "Download Completed", "Download completed successfully", 100.0)
	log.Printf("[DL] merged file ready: %s", mergedOut)

	// File info
	fileInfo, err := os.Stat(mergedOut)
	if err != nil {
		return nil, fmt.Errorf("failed to stat merged file: %w", err)
	}

	return &models.VideoDownloadResult{
		RequestID:   request.RequestID,
		FilePath:    mergedOut,
		FileName:    filepath.Base(mergedOut),
		Title:       title,
		DownloadURL: "/downloads/" + filepath.Base(mergedOut),
		CleanupAt:   util.EstimateCleanupTime(fileInfo.Size()),
	}, nil
}
