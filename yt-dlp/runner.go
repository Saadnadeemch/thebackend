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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetInfoFromYTDLP(videoURL string) (*models.VideoInfo, error) {
	binary, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp binary not found: %w", err)
	}

	cmd := exec.Command(
		binary,
		"-j",
		"--no-playlist",
		"--cookies-from-browser", "firefox",
		"--no-warnings",
		"--no-check-certificate",
		videoURL,
	)

	fmt.Printf("[yt-dlp CMD] %s\n", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()

	fmt.Printf("[yt-dlp RAW OUTPUT]\n%s\n", string(output))

	if err != nil {
		return nil, fmt.Errorf("yt-dlp exec error: %w | raw: %s", err, string(output))
	}

	var data models.YTDLPINFO
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("yt-dlp parse error: %w | raw: %s", err, string(output))
	}

	// Map to your model
	videoInfo := &models.VideoInfo{
		Title:       data.Title,
		Uploader:    data.Uploader,
		Thumbnail:   data.Thumbnail,
		Description: data.Description,
		UploadDate:  data.UploadDate,
		LikeCount:   data.LikeCount,
		VideoPage:   videoURL,
		Source:      "yt-dlp",
	}

	return videoInfo, nil
}

func GetDirectInfoFromYTDLP(videoURL string) (*models.VideoInfo, error) {

	binary, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp binary not found: %v", err)
	}

	// set output path (downloads folder or wherever you want)
	outputPath := "downloads/%(title)s.%(ext)s"

	cmd := exec.Command(
		binary,
		"--no-playlist",
		"-f", "best",
		"--merge-output-format", "mp4",
		"--cookies-from-browser", "firefox",
		"--print-json",
		"-o", outputPath,

		videoURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp exec error: %v | output: %s", err, string(output))
	}

	fmt.Println(string(output)) // debug JSON

	// parse JSON metadata from yt-dlp
	var data struct {
		Title       string  `json:"title"`
		Uploader    string  `json:"uploader"`
		Thumbnail   string  `json:"thumbnail"`
		ViewCount   int64   `json:"view_count"`
		Description *string `json:"description"`
		UploadDate  *string `json:"upload_date"`
		LikeCount   *int64  `json:"like_count"`
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
		DownloadURL: nil, // we already downloaded, no direct URL needed
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
	fragArg := util.GetFragmentsWithConnection(activeUsers, quality)

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

func DownloadAudioAsMP3(
	ctx context.Context,
	request models.AudioRequest,
	ws *webSocketMain.WSConnection,
) (*models.VideoDownloadResult, error) {

	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "audio_" + request.RequestID[:8]
	}
	safeTitle := util.SanitizedFileName(title)

	outputDir := "downloads"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "Directory error", 100)
		return nil, err
	}

	outputPath := filepath.Join(outputDir, safeTitle+"_prodl.%(ext)s")

	webSocketMain.SendSimpleProgress(ws, request.RequestID, "starting", "Starting Download", 0)

	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--newline",
		"--cookies-from-browser", "firefox",
		"-o", outputPath,
		request.URL,
	)

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "yt-dlp failed to start", 100)
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	progressRegex := regexp.MustCompile(`(\d{1,3}(?:\.\d+)?)%`)
	var last float64

	for scanner.Scan() {
		line := scanner.Text()

		if webSocketMain.IsRequestAborted(request.RequestID) {
			_ = cmd.Process.Kill()
			webSocketMain.SendSimpleProgress(ws, request.RequestID, "aborted", "Download aborted", last)
			return nil, fmt.Errorf("aborted")
		}

		if m := progressRegex.FindStringSubmatch(line); len(m) == 2 {
			if p, _ := strconv.ParseFloat(m[1], 64); p > last {
				last = p
				webSocketMain.SendSimpleProgress(ws, request.RequestID, "downloading", "Downloading Audio", p)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "Audio download failed", 100)
		return nil, err
	}

	var finalFile string
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasPrefix(info.Name(), safeTitle+"_prodl") {
			finalFile = path
		}
		return nil
	})

	if finalFile == "" {
		webSocketMain.SendSimpleProgress(ws, request.RequestID, "error", "File not found", 100)
		return nil, fmt.Errorf("file not found")
	}

	return &models.VideoDownloadResult{
		RequestID:   request.RequestID,
		FilePath:    finalFile,
		FileName:    filepath.Base(finalFile),
		Title:       title,
		DownloadURL: "/downloads/" + filepath.Base(finalFile),
		CleanupAt:   util.EstimateCleanupTime(0),
	}, nil
}

func DownloadStream(req models.StreamVideoDownloadRequest, c *gin.Context) error {
	log.Printf("[Runner] Starting yt-dlp stream | URL=%s", req.URL)

	args := []string{
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--newline",

		"--cookies-from-browser", "firefox",

		"-f", "best", // <-- try this first
		"-o", "-", // stream to stdout
		req.URL,
	}

	cmd := exec.Command("yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			n, e := stderr.Read(buf)
			if n > 0 {
				log.Printf("yt-dlp: %s", string(buf[:n]))
			}
			if e != nil {
				return
			}
		}
	}()

	// headers BEFORE streaming
	c.Header("Content-Type", "video/mp4")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)

	go func() {
		<-c.Request.Context().Done()
		log.Println("client disconnected → killing yt-dlp")
		_ = cmd.Process.Kill()
	}()

	_, copyErr := io.Copy(c.Writer, stdout)

	if flusher != nil {
		flusher.Flush()
	}

	if copyErr != nil {
		log.Printf("stream interrupted: %v", copyErr)
		_ = cmd.Process.Kill()
		return nil
	}

	_ = cmd.Wait()

	log.Println("stream finished")
	return nil
}
