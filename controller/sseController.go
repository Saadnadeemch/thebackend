package controllers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/sse"
)

func SSEHandler(c *gin.Context) {
	requestID := c.Param("request_id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request_id is required",
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	client := sse.Register(requestID)
	defer sse.Unregister(requestID)

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-client.Channel; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}
