package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Event is a task change notification broadcast to relevant clients.
type Event struct {
	Type   string      `json:"type"` // task.created, task.updated, task.deleted
	Task   interface{} `json:"task,omitempty"`
	TaskID string      `json:"task_id,omitempty"`
	UserID string      `json:"-"` // owner of the task, used for routing
}

type client struct {
	userID string
	role   string
	ch     chan Event
}

// Hub fans out task events to connected SSE clients, scoped by ownership/role.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// Broadcast sends the event to the owning user and any admin clients.
func (h *Hub) Broadcast(ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if c.userID == ev.UserID || c.role == "admin" {
			select {
			case c.ch <- ev:
			default:
			}
		}
	}
}

// ServeHTTP handles an SSE subscription for the authenticated user.
func (h *Hub) ServeHTTP(userID, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		c := &client{userID: userID, role: role, ch: make(chan Event, 16)}
		h.mu.Lock()
		h.clients[c] = struct{}{}
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.clients, c)
			h.mu.Unlock()
			close(c.ch)
		}()

		fmt.Fprintf(w, ": connected\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-c.ch:
				data, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
				flusher.Flush()
			}
		}
	}
}
