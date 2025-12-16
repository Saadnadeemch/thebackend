package utils

import (
	"log"
	"net/http"
	"sync"
	"time"

	"backend/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSConnection struct {
	Conn *websocket.Conn
	Lock sync.Mutex
}

func NewWSConnection(conn *websocket.Conn) *WSConnection {
	return &WSConnection{Conn: conn}
}

var (
	connections     = make(map[string]*WSConnection)
	connMutex       sync.RWMutex
	abortedRequests = make(map[string]bool)
	abortMutex      sync.RWMutex
)

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

func RegisterWSConnection(requestID string, wsConn *WSConnection) {
	connMutex.Lock()
	defer connMutex.Unlock()
	connections[requestID] = wsConn
	log.Printf("Registered WebSocket connection for request ID: %s", requestID)
}

func GetWSConnection(requestID string) *WSConnection {
	connMutex.RLock()
	defer connMutex.RUnlock()
	conn, ok := connections[requestID]
	if !ok {
		log.Printf("[DEBUG] No WebSocket connection found for RequestID: %s", requestID)
	}
	return conn
}

func UnregisterWSConnection(requestID string) {
	connMutex.Lock()
	defer connMutex.Unlock()
	if conn, exists := connections[requestID]; exists {
		conn.Close()
		delete(connections, requestID)
		log.Printf("Unregistered and closed WebSocket connection for request ID: %s", requestID)
	}
}

func MarkRequestAborted(requestID string) {
	abortMutex.Lock()
	defer abortMutex.Unlock()
	abortedRequests[requestID] = true
	log.Printf("Marked request as aborted | RequestID: %s", requestID)
}

func IsRequestAborted(requestID string) bool {
	abortMutex.RLock()
	defer abortMutex.RUnlock()
	return abortedRequests[requestID]
}

func (ws *WSConnection) Close() {
	ws.Lock.Lock()
	defer ws.Lock.Unlock()
	if ws.Conn != nil {
		_ = ws.Conn.Close()
		log.Println("WebSocket connection closed")
	}
}

func SendJSON(ws *WSConnection, data interface{}) error {
	ws.Lock.Lock()
	defer ws.Lock.Unlock()
	ws.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := ws.Conn.WriteJSON(data)
	if err != nil {
		log.Printf("Error sending JSON over WebSocket: %v", err)
	}
	return err
}

// Method on *WSConnection
func (ws *WSConnection) SendJSON(data interface{}) error {
	ws.Lock.Lock()
	defer ws.Lock.Unlock()

	ws.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := ws.Conn.WriteJSON(data)
	if err != nil {
		log.Printf("Error sending JSON over WebSocket: %v", err)
	}
	return err
}

// Depends on ws.SendJSON
func SendProgress(ws *WSConnection, progress models.DownloadProgress) {
	if ws == nil {
		log.Println("Cannot send progress: WSConnection is nil")
		return
	}
	if err := ws.SendJSON(progress); err != nil {
		log.Printf("Failed to send progress update: %v", err)
	} else {
		log.Printf("Sent progress for %s: Status=%s, Progress=%.2f",
			progress.RequestID, progress.Status, progress.Progress)
	}
}

// Depends on SendProgress
func SendSimpleProgress(ws *WSConnection, requestID, status, message string, percent float64) {
	progress := models.DownloadProgress{
		RequestID: requestID,
		Progress:  percent,
		Status:    status,
		Message:   message,
	}
	SendProgress(ws, progress)
}

func (ws *WSConnection) Listen(requestID string) {
	defer func() {
		log.Printf("WebSocket listener stopped | RequestID: %s", requestID)
	}()

	log.Printf("WebSocket listening started | RequestID: %s", requestID)

	for {
		_, p, err := ws.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("WebSocket closed by client | RequestID: %s", requestID)
			} else {
				log.Printf("WebSocket read error | RequestID: %s | Error: %v", requestID, err)
			}
			MarkRequestAborted(requestID)
			break
		}

		log.Printf("Message from client [%s]: %s", requestID, string(p))
	}
}

// -------------------
// Added Utility Methods
// -------------------

// GetActiveConnectionsCount returns the number of currently active WebSocket connections.
func GetActiveConnectionsCount() int {
	connMutex.RLock()
	defer connMutex.RUnlock()
	return len(connections)
}

// GetActiveRequestIDs returns a slice of all active request IDs.
func GetActiveRequestIDs() []string {
	connMutex.RLock()
	defer connMutex.RUnlock()
	ids := make([]string, 0, len(connections))
	for id := range connections {
		ids = append(ids, id)
	}
	return ids
}
