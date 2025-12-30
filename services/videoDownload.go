package services

import (
	"backend/models"
	util "backend/utils"
	webSocketMain "backend/websocket"
	runner "backend/yt-dlp"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
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

func DownloadStream(req models.StreamVideoDownloadRequest, c *gin.Context) error {
	log.Printf("🚀 Starting yt-dlp stream |  URL=%s", req.URL)

	args := []string{
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--newline",
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4", // force mp4 container
		"-o", "-", // write to stdout
		req.URL,
	}

	cmd := exec.Command("yt-dlp", args...)

	// Capture stdout (video stream) and stderr (logs)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe error: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe error: %w", err)
	}

	// Start yt-dlp process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Log stderr in background
	go func() {
		buf := make([]byte, 2048)
		for {
			n, e := stderr.Read(buf)
			if n > 0 {
				log.Printf("yt-dlp: %s", string(buf[:n]))
			}
			if e != nil {
				break
			}
		}
	}()

	// --- Delay sending headers until yt-dlp actually outputs data ---
	firstChunk := make([]byte, 32*1024) // buffer for first bytes
	n, readErr := stdout.Read(firstChunk)
	if readErr != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("yt-dlp produced no output: %w", readErr)
	}

	// Now we know yt-dlp is outputting data → send headers
	c.Header("Content-Type", "video/mp4")
	c.Status(http.StatusOK)
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	// Write the first chunk to response
	if _, err := c.Writer.Write(firstChunk[:n]); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write first chunk: %w", err)
	}

	// --- Continue piping remaining stdout to HTTP response ---
	_, copyErr := io.Copy(c.Writer, stdout)

	// Handle client disconnects / write failures
	if copyErr != nil {
		log.Printf("⚠️ Client disconnected or write failed: %v", copyErr)
		_ = cmd.Process.Kill()
	}

	// Ensure yt-dlp finishes
	waitErr := cmd.Wait()
	if waitErr != nil && copyErr == nil {
		return fmt.Errorf("yt-dlp process error: %w", waitErr)
	}

	log.Println("✅ Streaming completed successfully")
	return nil
}
