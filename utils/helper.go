package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// check if the downlaod directory exist
func EnsureRootDirectory() error {
	const folderName = "downloads"

	absPath, err := filepath.Abs(folderName)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(absPath, 0755); mkErr != nil {
			return fmt.Errorf("failed to create directory '%s': %w", absPath, mkErr)
		}
	}
	return nil
}

// GenerateRequestID creates a short 16-char hex ID for websocket
func GenerateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// EstimateCleanupTime calculates when file should be removed after download
const (
	MinDownloadSpeedKBps = 50
	CleanupBufferSeconds = 2 * 60
)

func EstimateCleanupTime(size int64) int64 {
	est := size / (MinDownloadSpeedKBps * 1024)
	return time.Now().Add(time.Second * time.Duration(est+CleanupBufferSeconds)).Unix()
}

// FindDownloadedFile locates downloaded file by requestID and optional tag
func FindDownloadedFile(dir, requestID, tag string) (string, error) {
	matches, _ := filepath.Glob(fmt.Sprintf("%s/%s*%s*", dir, requestID, tag))
	if len(matches) == 0 {
		return "", fmt.Errorf("file not found: %s", tag)
	}
	return matches[0], nil
}

func DeleteFilesOlderThan(dir string, olderThan time.Duration) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > olderThan {
			path := filepath.Join(dir, file.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("[CLEANUP] Failed to delete %s: %v", path, err)
			} else {
				log.Printf("[CLEANUP] Deleted old file: %s", path)
			}
		}
	}

	return nil
}

func SanitizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("URL parsing failed: %s", rawURL)
		return rawURL
	}

	var videoID string
	query := parsed.Query()
	videoID = query.Get("v")

	if videoID == "" && strings.Contains(parsed.Host, "youtu.be") {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) > 0 {
			videoID = parts[len(parts)-1]
		}
	}

	if videoID == "" {
		log.Printf("No video ID found in URL: %s", rawURL)
		return rawURL
	}

	sanitized := "https://www.youtube.com/watch?v=" + videoID
	log.Printf("Sanitized URL: %s", sanitized)
	return sanitized
}

var downloadLimit = make(chan struct{}, 25)

// Block until slot is acquired
func AcquireSlot() {
	downloadLimit <- struct{}{}
}

func ReleaseSlot() {
	select {
	case <-downloadLimit:
	default:
	}
}

// Helper: check if all slots are full (non-blocking)
func SlotsFull() bool {
	return len(downloadLimit) == cap(downloadLimit)
}

func SanitizedFileName(name string) string {
	name = strings.TrimSpace(name)

	if name == "" {
		return "file"
	}

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces with underscore first
	name = strings.ReplaceAll(name, " ", "_")

	// Remove non ASCII characters (Arabic, emojis, etc)
	regNonASCII := regexp.MustCompile(`[^\x00-\x7F]+`)
	name = regNonASCII.ReplaceAllString(name, "")

	// Keep only safe filename chars
	regSafe := regexp.MustCompile(`[^a-z0-9\-_]`)
	name = regSafe.ReplaceAllString(name, "")

	// Collapse multiple underscores
	regMulti := regexp.MustCompile(`_+`)
	name = regMulti.ReplaceAllString(name, "_")

	// Trim underscores
	name = strings.Trim(name, "_")

	if name == "" {
		return "video"
	}

	return name
}
