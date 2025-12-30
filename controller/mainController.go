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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "[MAINCONTROLLER.GO] Invalid JSON payload"})
		return
	}

	if req.URL == "" || req.Quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "[MAINCONTROLLER.GO] URL and Quality are required"})
		return
	}

	sanitizedURL := util.SanitizeURL(req.URL)

	platformInfo := util.DetectPlatform(req.URL)
	log.Printf("[DetectPlatform] Detected Platform: %s | Supported: %v | Method: %v",
		platformInfo.Platform, platformInfo.IsSupported, platformInfo.DownloadMethod)

	if !platformInfo.IsSupported {
		log.Printf("[DetectPlatform] Unsupported platform | Reason: %s", platformInfo.Reason)
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
	log.Printf("[MAINCONTROLLER.GO] Request received | Platform: %s | Method: %s | URL: %s", platform, method, url)

	videoInfo, err := services.GetVideoInfoService(url, platform, method)
	if err != nil {
		log.Printf("InfoService %s] Error fetching video info: %v", websocketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "[INFOSERVICE] Failed to fetch video info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"websocket_id": websocketID,
		"type":         method,
		"video_info":   videoInfo,
	})

	//  Trigger background download (only if needed)
	if method == "separate-av" || method == "default" {
		go func(title string) {
			log.Printf("[MAINCONTROLLER.GO] Triggering VideoDownloadHandler | WS_ID: %s | Platform: %s | URL: %s | Title: %s",
				websocketID, platform, url, title)

			VideoController(c.Copy(), platform, method, url, websocketID, title, quality)
		}(videoInfo.Title)
	}

}
