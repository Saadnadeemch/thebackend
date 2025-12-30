package ytdlp

import (
	"backend/models"
	util "backend/utils"
	webSocketMain "backend/websocket"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetDirectInfoFromYTDLP(videoURL string) (*models.VideoInfo, error) {
	binary, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp binary path not found: %v", err)
	}

	cmd := exec.Command(
		binary,
		"-f", "best",
		"-j",
		"--no-playlist",
		"--cookies-from-browser", "firefox",
		"--no-warnings",
		"--no-check-certificate",
		videoURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp exec error: %v | output: %s", err, string(output))
	}

	fmt.Println(output)

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

func RunYTDownloadWithProgressContextWithMerge(
	ctx context.Context,
	format, output, url string,
	request models.DownloadVideoRequest,
	ws *webSocketMain.WSConnection,
) error {

	activeUsers := webSocketMain.GetActiveConnectionsCount()
	quality := request.Quality
	fragArg, dlArg := util.GetFragmentsWithConnection(activeUsers, quality)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", format,
		"-o", output,
		"--merge-output-format", "mp4",
		"--cookies-from-browser", "firefox",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:142.0) Gecko/20100101 Firefox/142.0",
		"--progress",
		"--newline",
		"--no-playlist",
		fragArg,
		"--downloader", "aria2c",
		"--downloader-args", dlArg,
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

func DownloadStream(req models.StreamVideoDownloadRequest, c *gin.Context) error {
	log.Printf("[Runner] Starting yt-dlp stream |  URL=%s", req.URL)

	args := []string{
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--newline",
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"-o", "-",
		req.URL,
	}

	cmd := exec.Command("yt-dlp", args...)

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
	firstChunk := make([]byte, 32*1024)
	n, readErr := stdout.Read(firstChunk)
	if readErr != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("yt-dlp produced no output: %w", readErr)
	}

	c.Header("Content-Type", "video/mp4")
	c.Status(http.StatusOK)
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	if _, err := c.Writer.Write(firstChunk[:n]); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write first chunk: %w", err)
	}

	_, copyErr := io.Copy(c.Writer, stdout)

	if copyErr != nil {
		log.Printf("⚠️ Client disconnected or write failed: %v", copyErr)
		_ = cmd.Process.Kill()
	}

	waitErr := cmd.Wait()
	if waitErr != nil && copyErr == nil {
		return fmt.Errorf("yt-dlp process error: %w", waitErr)
	}

	log.Println("✅ Streaming completed successfully")
	return nil
}
