package api

import (
	"net/http"
)

// WebSocket endpoints

// HandleDocumentWebSocket handles WebSocket connections for document collaboration
func (h *Handler) HandleDocumentWebSocket(w http.ResponseWriter, r *http.Request) {
	h.wsHandler.HandleDocumentConnection(w, r)
}

// HandleUpdatesWebSocket handles global update notifications
func (h *Handler) HandleUpdatesWebSocket(w http.ResponseWriter, r *http.Request) {
	h.wsHandler.HandleUpdatesConnection(w, r)
}
