package controllers

import (
	"backend/models"
	"backend/services"
	"backend/sse"
	util "backend/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func VideoHandler(c *gin.Context) {
	var req models.Request
	requestID := util.GenerateRequestID()

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON payload",
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	sanitizedURL := util.SanitizeURL(req.URL)
	platformInfo := util.DetectPlatform(sanitizedURL)

	log.Printf(
		"[VIDEO] RequestID=%s | Platform=%s | Type=%s | URL=%s",
		requestID,
		platformInfo.Platform,
		platformInfo.VideoType,
		sanitizedURL,
	)

	videoInfo, err := services.GetVideoInfoService(
		sanitizedURL,
		string(platformInfo.VideoType),
	)
	if err != nil {
		log.Printf("[VIDEO] Metadata failed | RequestID=%s | Error=%v",
			requestID, err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch video info",
		})
		return
	}

	go startDownload(
		requestID,
		sanitizedURL,
		videoInfo.Title,
		req.Quality,
		platformInfo,
	)

	c.JSON(http.StatusOK, gin.H{
		"request_id": requestID,
		"video_info": videoInfo,
	})
}

func startDownload(
	requestID string,
	url string,
	title string,
	quality string,
	platformInfo models.PlatformInfo,
) {

	log.Printf("[DOWNLOAD] Starting | RequestID=%s", requestID)

	videoQuality, status := util.CheckAndPickFormat(
		quality,
		string(platformInfo.VideoType),
	)

	log.Printf(
		"[DOWNLOAD] RequestID=%s | Platform=%s | Format=%s | Status=%s",
		requestID,
		platformInfo.Platform,
		videoQuality,
		status,
	)

	if util.SlotsFull() {
		sse.Send(requestID, gin.H{
			"status":  "queued",
			"message": "Too many downloads. Waiting for slot...",
			"percent": 0,
		})
	}

	util.AcquireSlot()
	defer util.ReleaseSlot()

	sse.Send(requestID, gin.H{
		"status":  "start",
		"message": "Download started",
		"percent": 0,
	})

	downloadReq := models.DownloadVideoRequest{
		URL:          url,
		RequestID:    requestID,
		VideoQuality: videoQuality,
		Title:        title,
		Platform:     string(platformInfo.Platform),
		VideoType:    string(platformInfo.VideoType),
	}

	result, err := services.DownloadService(downloadReq)
	if err != nil {
		log.Printf("[DOWNLOAD] Failed | RequestID=%s | Error=%v",
			requestID, err)

		sse.Send(requestID, gin.H{
			"status":  "error",
			"message": "Download failed",
			"percent": 0,
		})
		return
	}

	sse.Send(requestID, gin.H{
		"status":  "completed",
		"message": "Download complete",
		"percent": 100,
		"result":  result,
	})

	log.Printf("[DOWNLOAD] Completed | RequestID=%s", requestID)
}
