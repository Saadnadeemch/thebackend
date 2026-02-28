package ytdlp

import (
	"backend/models"
	sse "backend/sse"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func GetVideoInfoFromYTDLP(videoURL string) (*models.VideoInfo, error) {
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
		"--quiet",
		videoURL,
	)

	fmt.Printf("[yt-dlp CMD] %s\n", strings.Join(cmd.Args, " "))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp exec error: %w | stderr: %s", err, stderr.String())
	}

	var data models.YtdlpInfo
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("yt-dlp parse error: %w | raw: %s", err, stdout.String())
	}

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
func RunYTDownloadWithProgress(ctx context.Context, args []string, requestID string) error {

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("yt-dlp start failed: %w", err)
	}

	reader := bufio.NewReader(stdout)

	percentRegex := regexp.MustCompile(`(\d{1,3}(?:\.\d+)?)%`)
	sizeRegex := regexp.MustCompile(`of\s+~?\s*([\d\.]+\s*[KMG]i?B)`)

	var lastSent time.Time = time.Now().Add(-time.Second)
	var lastPercent float64 = 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Printf("[yt-dlp] %s\n", line)

		percentMatch := percentRegex.FindStringSubmatch(line)
		if len(percentMatch) != 2 {
			continue
		}

		percent, err := strconv.ParseFloat(percentMatch[1], 64)
		if err != nil {
			continue
		}

		sizeMatch := sizeRegex.FindStringSubmatch(line)
		if len(sizeMatch) == 2 {
			size := sizeMatch[1]
			if percent == 100 && (strings.Contains(size, "KiB") || strings.Contains(size, "B")) {
				continue
			}
		}

		if percent < lastPercent {
			continue
		}

		if time.Since(lastSent) >= time.Second || percent >= 100 {
			sse.Send(requestID, map[string]interface{}{
				"status":  "downloading",
				"message": "Downloading",
				"percent": percent,
			})
			lastSent = time.Now()
			lastPercent = percent
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	if lastPercent < 100 {
		sse.Send(requestID, map[string]interface{}{
			"status":  "completed",
			"message": "Download completed",
			"percent": 100,
		})
	}

	return nil
}
