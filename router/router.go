package router

import (
	controllers "backend/controller"
	"backend/models"
	Runner "backend/yt-dlp"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// sanitizeFileName removes illegal filename characters for all platforms
func sanitizeFileName(name string) string {
	if name == "" {
		return "video"
	}
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) {
			return -1
		}
		return r
	}, name)
}

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.POST("/api/getvideo", controllers.VideoHandler)

	r.POST("/api/getaudio", controllers.AudioHandler)

	r.GET("/api/stream", func(c *gin.Context) {
		url := c.Query("url")
		title := sanitizeFileName(c.Query("title"))

		if url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "[Router] missing url parameter"})
			return
		}

		filename := fmt.Sprintf("%s.mp4", title)

		// force browser to download
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		req := models.StreamVideoDownloadRequest{
			URL: url,
		}

		if err := Runner.DownloadStream(req, c); err != nil {
			log.Printf("[Rotuter] Streaming error: %v", err)
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "[Router] failed to stream video"})
			}
		}
	})

	r.GET("/api/proxy-download", func(c *gin.Context) {
		url := c.Query("url")
		title := sanitizeFileName(c.Query("title"))
		platform := c.Query("platform")

		if url == "" || title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "[Router] missing required parameters"})
			return
		}

		filename := fmt.Sprintf("%s-%s.mp4", title, platform)

		// Fetch the actual video
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[Router] Proxy fetch failed: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch video"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": "[Router] remote server error"})
			return
		}

		// Force download headers
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "video/mp4")
		c.Header("Content-Length", fmt.Sprintf("%d", resp.ContentLength))

		// Stream response body directly to client
		if _, err := io.Copy(c.Writer, resp.Body); err != nil {
			log.Printf("[Router]  Proxy stream error: %v", err)
		}
	})

	r.GET("/api/ws/:request_id", controllers.WebSocketHandler)

	r.GET("/api/downloads/:filename", func(c *gin.Context) {
		filename := sanitizeFileName(c.Param("filename"))
		filePath := "./downloads/" + filename

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "application/octet-stream")
		c.File(filePath)
	})

	return r
}
