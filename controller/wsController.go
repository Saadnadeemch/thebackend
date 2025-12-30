package controllers

import (
	"log"
	"net/http"
	"time"

	util "backend/utils"
	webSocket "backend/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebSocketHandler(c *gin.Context) {
	requestID := c.Param("request_id")
	if requestID == "" {
		log.Println("Error: request_id parameter is missing.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed for request ID %s: %v", requestID, err)
		return
	}

	ws := webSocket.NewWSConnection(conn)

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // refresh deadline on pong
		return nil
	})

	webSocket.RegisterWSConnection(requestID, ws)

	// Start ping ticker (server pings client every 30 seconds)
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-pingTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("[ERROR] Failed to send ping | RequestID: %s | Error: %v", requestID, err)
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	defer func() {
		util.TriggerCancelFunc(requestID)
		webSocket.UnregisterWSConnection(requestID)
		webSocket.MarkRequestAborted(requestID)
		log.Printf("[INFO] WebSocket closed by client | RequestID: %s", requestID)
	}()

	log.Printf("[INFO] WebSocket connection established | RequestID: %s", requestID)

	ws.Listen(requestID)

	close(done) // stop ping goroutine
	log.Printf("[INFO] WebSocket listener stopped | RequestID: %s", requestID)
}
