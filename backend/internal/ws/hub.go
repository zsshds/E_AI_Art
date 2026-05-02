package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type TaskUpdate struct {
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	ResultImageURL string `json:"result_image_url,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	Progress       int    `json:"progress,omitempty"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool // taskID -> connections
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, taskID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	h.mu.Lock()
	if h.clients[taskID] == nil {
		h.clients[taskID] = make(map[*websocket.Conn]bool)
	}
	h.clients[taskID][conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients[taskID], conn)
		if len(h.clients[taskID]) == 0 {
			delete(h.clients, taskID)
		}
		h.mu.Unlock()
		conn.Close()
	}()

	// Keep connection alive, read messages (handle ping/pong)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) PushUpdate(taskID string, update TaskUpdate) {
	h.mu.RLock()
	conns := h.clients[taskID]
	h.mu.RUnlock()

	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("ws marshal error: %v", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws write error: %v", err)
			conn.Close()
			h.mu.Lock()
			delete(h.clients[taskID], conn)
			h.mu.Unlock()
		}
	}
}
