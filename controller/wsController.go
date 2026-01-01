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
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WsController] Upgrade failed: %v", err)
		return
	}

	ws := webSocket.NewWSConnection(conn)

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	webSocket.RegisterWSConnection(requestID, ws)

	pingTicker := time.NewTicker(30 * time.Second)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-pingTicker.C:
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				if err != nil {
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	log.Printf("[WsController] Connected | %s", requestID)

	ws.Listen(requestID)

	close(done)
	pingTicker.Stop()
	util.TriggerCancelFunc(requestID)
	webSocket.UnregisterWSConnection(requestID)
	webSocket.MarkRequestAborted(requestID)

	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(2*time.Second),
	)

	_ = conn.Close()

	log.Printf("[WsController] Download Completed Connection Closed | %s", requestID)
}
