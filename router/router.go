package router

import (
	controllers "backend/controller"
	"strings"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.POST("/video", controllers.VideoHandler)
	r.GET("/stream/:request_id", controllers.SSEHandler)

	r.GET("/downloads/:filename", func(c *gin.Context) {
		filename := sanitizeFileName(c.Param("filename"))
		filePath := "./downloads/" + filename
		c.FileAttachment(filePath, filename)
	})

	return r
}

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
