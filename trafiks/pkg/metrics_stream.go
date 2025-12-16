package pkg

import (
	"encoding/json"
	"sync"
	"time"
)

// MetricsEvent represents a metrics update event
type MetricsEvent struct {
	Type      string                 `json:"type"`
	ProjectID uint                   `json:"project_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type MetricsStreamHubClient interface {
	RegisterClient(projectID uint) chan []byte
	UnregisterClient(projectID uint, clientChan chan []byte)
	Broadcast(projectID uint, event MetricsEvent)
	GetClientCount(projectID uint) int
}

type MetricsStreamHub struct {
	clients map[uint]map[chan []byte]bool
	mu      sync.RWMutex
}

func NewMetricsStreamHub() MetricsStreamHubClient {
	return &MetricsStreamHub{
		clients: make(map[uint]map[chan []byte]bool),
	}
}

func (h *MetricsStreamHub) RegisterClient(projectID uint) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	clientChan := make(chan []byte, 10)

	if h.clients[projectID] == nil {
		h.clients[projectID] = make(map[chan []byte]bool)
	}
	h.clients[projectID][clientChan] = true

	return clientChan
}

func (h *MetricsStreamHub) UnregisterClient(projectID uint, clientChan chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[projectID]; ok {
		delete(clients, clientChan)
		close(clientChan)
		if len(clients) == 0 {
			delete(h.clients, projectID)
		}
	}
}

func (h *MetricsStreamHub) Broadcast(projectID uint, event MetricsEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	if clients, ok := h.clients[projectID]; ok {
		for clientChan := range clients {
			select {
			case clientChan <- eventJSON:
			default:
			}
		}
	}
}

func (h *MetricsStreamHub) GetClientCount(projectID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[projectID]; ok {
		return len(clients)
	}
	return 0
}
