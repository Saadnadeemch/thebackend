package controllers

import (
	"backend/models"
	"backend/services"
	util "backend/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handles video metadata request
func VideoHandler(c *gin.Context) {
	var req models.Request
	wsID := util.GenerateRequestID()

	// Parse request JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Validate required fields
	if req.URL == "" || req.Quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL and Quality are required"})
		return
	}

	sanitizedURL := util.SanitizeURL(req.URL)

	// Detect platform (YouTube, TikTok, etc.)
	platformInfo := util.DetectPlatform(req.URL)
	log.Printf("[INFO] Detected Platform: %s | Supported: %v | Method: %v",
		platformInfo.Platform, platformInfo.IsSupported, platformInfo.DownloadMethod)

	// If unsupported → return error
	if !platformInfo.IsSupported {
		log.Printf("[ERROR] Unsupported platform | Reason: %s", platformInfo.Reason)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "Unsupported or invalid platform",
			"reason":   platformInfo.Reason,
			"platform": platformInfo.Platform,
			"method":   platformInfo.DownloadMethod,
		})
		return
	}

	GetVideoInfo(
		c,
		platformInfo.Platform,
		platformInfo.DownloadMethod,
		sanitizedURL,
		wsID,
		req.Quality,
	)
}

func GetVideoInfo(c *gin.Context, platform string, method string, url string, websocketID string, quality string) {
	log.Printf("[INFO] Request received | Platform: %s | Method: %s | URL: %s", platform, method, url)

	// ✅ Fetch video info
	videoInfo, err := services.GetVideoInfoService(url, platform, method)
	if err != nil {
		log.Printf("[WS_ID: %s] Error fetching video info: %v", websocketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch video info"})
		return
	}

	// ✅ Respond early with metadata + WebSocket ID
	c.JSON(http.StatusOK, gin.H{
		"websocket_id": websocketID,
		"type":         method,    // "separate-av" | "default" | others
		"video_info":   videoInfo, // full object instead of flattening
	})

}
