package ytdlp

import (
	"backend/models"
	util "backend/utils"
	webSocketMain "backend/websocket"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func Downloadbestformat(
	ctx context.Context,
	url, output string,
	request models.DownloadVideoRequest,
	ws *webSocketMain.WSConnection,
) (*models.VideoDownloadResult, error) {
	activeUsers := webSocketMain.GetActiveConnectionsCount()
	quality := request.Quality
	fragArg, dlArg := util.GetFragmentsWithConnection(activeUsers, quality)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "best",
		"-o", output,
		"--cookies-from-browser", "firefox",
		"--no-playlist",
		"--no-warnings",
		"--progress",
		"--newline",
		fragArg,
		"--downloader", "aria2c",
		"--downloader-args", dlArg,
		url,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("yt-dlp start error: %v", err)
	}

	// Capture stderr output in background
	var stderrBuf bytes.Buffer
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	percentRegex := regexp.MustCompile(`(?m)(\d{1,3}(?:\.\d+)?)%`)
	sizeRegex := regexp.MustCompile(`of\s+~?\s*([\d\.]+\s*[KMG]i?B)`)

	var lastSent time.Time = time.Now().Add(-2 * time.Second)
	var lastPercent float64 = 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Handle cancellation
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return nil, fmt.Errorf("download aborted by context cancel")
		default:
		}

		percentMatch := percentRegex.FindStringSubmatch(line)
		if len(percentMatch) == 2 {
			percent, err := strconv.ParseFloat(percentMatch[1], 64)
			if err != nil {
				continue
			}

			// Extract size to ignore fake 100% lines
			sizeMatch := sizeRegex.FindStringSubmatch(line)
			if len(sizeMatch) == 2 {
				size := sizeMatch[1]
				if percent == 100 && (strings.Contains(size, "KiB") || strings.Contains(size, "B")) {
					// Ignore tiny "100%" messages (init fragments, metadata)
					continue
				}
			}

			// Prevent regressions
			if percent < lastPercent {
				continue
			}

			// Send update every ~1s or final 100%
			if time.Since(lastSent) >= time.Second || percent >= 100 {
				webSocketMain.SendSimpleProgress(ws, request.RequestID, "downloading", "Downloading Video", percent)
				lastSent = time.Now()
				lastPercent = percent
			}
		}
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("scanner error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if stderrBuf.Len() > 0 {
			return nil, fmt.Errorf("yt-dlp error: %s", stderrBuf.String())
		}
		return nil, err
	}

	// Guarantee final 100% if not already sent
	if lastPercent < 100 {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "completed", "Download completed (no merge needed)", 100.0)
	}

	fileInfo, err := os.Stat(output)
	if err != nil {
		return nil, fmt.Errorf("failed to stat output: %v", err)
	}

	return &models.VideoDownloadResult{
		RequestID:   request.RequestID,
		FilePath:    output,
		FileName:    filepath.Base(output),
		Title:       request.Title,
		DownloadURL: "/downloads/" + filepath.Base(output),
		CleanupAt:   util.EstimateCleanupTime(fileInfo.Size()),
	}, nil
}

func RunYTDownloadWithProgressContextWithMerge(
	ctx context.Context,
	format, output, url string,
	request models.DownloadVideoRequest,
	ws *webSocketMain.WSConnection,
) error {

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", format,
		"-o", output,
		"--merge-output-format", "mp4",
		"--cookies-from-browser", "firefox",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:142.0) Gecko/20100101 Firefox/142.0",
		"--progress",
		"--newline",
		"--no-playlist",
		url,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe error: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("yt-dlp start failed: %w", err)
	}

	reader := bufio.NewScanner(stdout)
	percentRegex := regexp.MustCompile(`(?m)(\d{1,3}(?:\.\d+)?)%`)
	sizeRegex := regexp.MustCompile(`of\s+~?\s*([\d\.]+\s*[KMG]i?B)`)

	var lastSent time.Time = time.Now().Add(-2 * time.Second)
	var lastPercent float64 = 0

	// Kill process if context is cancelled
	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()

	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}
		fmt.Printf("[yt-dlp VIDEO] %s\n", line)

		percentMatch := percentRegex.FindStringSubmatch(line)
		if len(percentMatch) == 2 {
			percent, err := strconv.ParseFloat(percentMatch[1], 64)
			if err != nil {
				continue
			}

			// Check size (to ignore fake 100% like "1.00KiB")
			sizeMatch := sizeRegex.FindStringSubmatch(line)
			if len(sizeMatch) == 2 {
				size := sizeMatch[1]
				if percent == 100 && (strings.Contains(size, "KiB") || strings.Contains(size, "B")) {
					// skip tiny fake 100% lines
					continue
				}
			}

			// Prevent regressions (never go backward in percent)
			if percent < lastPercent {
				continue
			}

			// Send update if 1s passed OR it's final 100%
			if time.Since(lastSent) >= time.Second || percent >= 100 {
				webSocketMain.SendSimpleProgress(ws, request.RequestID, "video download", "Downloading", percent)
				lastSent = time.Now()
				lastPercent = percent
			}
		}
	}

	// Wait for process
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	// Guarantee final 100%
	if lastPercent < 100 {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "video download", "Completed", 100)
	}

	return nil
}
