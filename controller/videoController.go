package controllers

import (
	"log"
	"time"

	"backend/models"
	"backend/services"
	util "backend/utils"
	ws "backend/websocket"

	"github.com/gin-gonic/gin"
)

func waitForWebSocket(wsID string, timeout time.Duration) *ws.WSConnection {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			log.Printf("[VideoController: %s] ❌ WebSocket connection timeout", wsID)
			return nil
		default:
			if conn := ws.GetWSConnection(wsID); conn != nil {
				log.Printf("[VideoController: %s] WebSocket connected", wsID)
				return conn
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func VideoController(c *gin.Context, platform, method, url, wsID, title, quality string) {
	go func() {
		wsConn := waitForWebSocket(wsID, 60*time.Second)
		if wsConn == nil {
			log.Printf("[VideoController: %s] ❌ No WebSocket connection, aborting", wsID)
			return
		}
		log.Printf("[VideoController: %s]  Starting Separate-AV Download | Platform=%s", wsID, platform)
		handleSeparateAVDownload(platform, url, wsID, wsConn, title, quality)
	}()
}

func handleSeparateAVDownload(platform, url, wsID string, wsConn *ws.WSConnection, title, quality string) {
	formatID, status := util.CheckAndPickFormat(quality)
	log.Printf("[VideoController: %s] Format detection: %s | formatID=%s", wsID, status, formatID)

	if util.SlotsFull() {
		ws.SendSimpleProgress(wsConn, wsID, "queued", "Too many downloads right now, waiting ...", 0)
	}

	util.AcquireSlot()
	ws.SendSimpleProgress(wsConn, wsID, "start", "Slot acquired", 0)
	defer util.ReleaseSlot()

	downloadReq := models.DownloadVideoRequest{
		URL:       url,
		Quality:   quality,
		RequestID: wsID,
		FormatID:  formatID,
		Title:     title,
	}

	result, err := services.DownloadVideo(downloadReq, wsConn)
	if err != nil {
		log.Printf("[VideoController: %s] ❌ Download failed: %v", wsID, err)
		ws.SendSimpleProgress(wsConn, wsID, "error", "Download failed", 0)
		return
	}

	ws.SendSimpleProgress(wsConn, wsID, "completed", "Download complete", 100)
	_ = wsConn.SendJSON(gin.H{"event": "download_result", "payload": result})

	wsConn.GracefulClose()

}
