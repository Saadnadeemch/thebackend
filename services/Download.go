package services

import (
	"backend/models"
	util "backend/utils"
	utils "backend/websocket"
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

func DownloadAudio(ctx context.Context, request models.AudioRequest, ws *utils.WSConnection) (*models.VideoDownloadResult, error) {
	if utils.IsRequestAborted(request.RequestID) {
		utils.SendSimpleProgress(ws, request.RequestID, "aborted", "Client disconnected before download started", 0)
		return nil, fmt.Errorf("download aborted: client disconnected")
	}

	util.AcquireSlot()
	defer util.ReleaseSlot()

	if err := util.EnsureRootDirectory(); err != nil {
		utils.SendSimpleProgress(ws, request.RequestID, "error", "Failed to prepare download directory", 0)
		return nil, fmt.Errorf("failed to ensure download directory: %w", err)
	}

	result, err := runner.DownloadAudioAsMP3(ctx, request, ws)
	if err != nil {
		utils.SendSimpleProgress(ws, request.RequestID, "error", "Audio download failed", 0)
		return nil, err
	}

	if _, err := os.Stat(result.FilePath); err != nil {
		utils.SendSimpleProgress(ws, request.RequestID, "error", "Downloaded file not found", 0)
		return nil, fmt.Errorf("failed to stat MP3 file: %w", err)
	}

	return result, nil
}

func DownloadAndMergeYTAV(
	request models.DownloadVideoRequest,
	ws *webSocketMain.WSConnection,
) (*models.VideoDownloadResult, error) {
	url := request.URL
	format := request.FormatID

	log.Printf("[VideoService] starting DownloadAndMergeYTAV request=%s url=%s format=%s", request.RequestID, url, format)

	// Ensure safe title
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "prodl" + request.RequestID[:8]
	}
	safeTitle := util.SanitizedFileName(title)

	outputDir := "downloads"
	if err := util.EnsureRootDirectory(); err != nil {
		return nil, fmt.Errorf("[VideoService] failed to ensure download directory: %w", err)
	}

	mergedOut := filepath.Join(outputDir, fmt.Sprintf("%s_prodl.mp4", safeTitle))

	webSocketMain.SendSimpleProgress(ws, request.RequestID, "Starting Download", "Getting things ready", 0)

	// Context + cancel handling
	ctx, cancel := context.WithCancel(context.Background())
	util.RegisterCancelFunc(request.RequestID, cancel)
	defer func() {
		cancel()
		util.CleanupCancelFunc(request.RequestID)
	}()

	// Calling the Runner
	err := runner.RunYTDownloadWithProgressContextWithMerge(ctx, format, mergedOut, url, request, ws)
	if err != nil {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "Download failed", 100)
		return nil, fmt.Errorf("[VideoService] DownloadAndMergeYTAV failed: %w", err)
	}

	webSocketMain.SendSimpleProgress(ws, request.RequestID, "Download Completed", "Download completed successfully", 100.0)
	log.Printf("[VideoService] merged file ready: %s", mergedOut)

	fileInfo, err := os.Stat(mergedOut)
	if err != nil {
		return nil, fmt.Errorf("failed to merged file: %w", err)
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
