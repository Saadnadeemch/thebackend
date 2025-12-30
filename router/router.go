package router

import (
	controllers "backend/controller"
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

	// 4. WebSocket for progress updates
	r.GET("/api/ws/:request_id", controllers.WebSocketHandler)

	// 5. Serve downloaded files
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
