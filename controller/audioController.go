package controllers

import (
	"backend/models"
	"backend/services"
	util "backend/utils"
	ws "backend/websocket"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AudioHandler(c *gin.Context) {
	var req models.AudioRequest

	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" || req.RequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL and RequestID are required"})
		return
	}

	wsID := util.GenerateRequestID()
	sanitizedURL := util.SanitizeURL(req.URL)
	log.Printf("[WS_ID: %s] Sanitized Audio URL: %s", wsID, sanitizedURL)

	audioInfo, err := services.GetAudioInfo(sanitizedURL)
	if err != nil {
		log.Printf("[WS_ID: %s] Error fetching audio info: %v", wsID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audio info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"websocket_id": wsID,
		"audio_info":   audioInfo,
	})

	go func() {
		var wsConn *ws.WSConnection
		wsTimeout := time.After(20 * time.Second)

	WAIT_LOOP:
		for {
			select {
			case <-wsTimeout:
				log.Printf("[WS_ID: %s] WebSocket connection timeout", wsID)
				return
			default:
				if conn := ws.GetWSConnection(wsID); conn != nil {
					wsConn = conn
					log.Printf("[WS_ID: %s] WebSocket connected", wsID)
					break WAIT_LOOP
				}
				time.Sleep(200 * time.Millisecond)
			}
		}

		util.AcquireSlot()
		defer util.ReleaseSlot()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		downloadReq := models.AudioRequest{
			URL:       sanitizedURL,
			Title:     audioInfo.Title,
			RequestID: wsID,
		}

		result, err := services.DownloadAudio(ctx, downloadReq, wsConn)
		if err != nil {
			log.Printf("[WS_ID: %s] Audio download failed: %v", wsID, err)
			ws.SendSimpleProgress(wsConn, wsID, "error", "Audio download failed", 0)
			return
		}

		ws.SendSimpleProgress(wsConn, wsID, "completed", "Audio download complete", 100)

		if err := wsConn.SendJSON(gin.H{
			"event":   "download_result",
			"payload": result,
		}); err != nil {
			log.Printf("[WS_ID: %s] Failed to send result: %v", wsID, err)
		}
		wsConn.GracefulClose()
	}()
}
