package collaboration

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"ai-kms/internal/middleware"
	"ai-kms/internal/models"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
)

/*
LEARNING: WEBSOCKET UPGRADER

The upgrader converts HTTP connections to WebSocket connections.

Key settings:
- ReadBufferSize/WriteBufferSize: Memory for I/O operations
- CheckOrigin: CORS validation for WebSocket connections
*/

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin properly
		return true
	},
}

// WebSocketHandler handles WebSocket connections for document collaboration
type WebSocketHandler struct {
	sessionManager *SessionManager
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(sessionManager *SessionManager) *WebSocketHandler {
	return &WebSocketHandler{
		sessionManager: sessionManager,
	}
}

// HandleDocumentConnection handles WebSocket connection for a specific document
func (h *WebSocketHandler) HandleDocumentConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	documentID := vars["id"]

	// Extract user info from query params (in production, use proper auth)
	userID := r.URL.Query().Get("user_id")
	userName := r.URL.Query().Get("user_name")
	clientIDStr := r.URL.Query().Get("client_id")

	if userID == "" {
		userID = "anonymous"
	}
	if userName == "" {
		userName = "Anonymous"
	}

	clientID, _ := strconv.Atoi(clientIDStr)
	if clientID == 0 {
		clientID = int(time.Now().UnixNano() % 1000000) // Generate if not provided
	}

	// Create span for connection
	ctx, span := middleware.StartSpan(ctx, "WebSocket.Connect",
		attribute.String("document.id", documentID),
		attribute.String("user.id", userID),
		attribute.Int("client.id", clientID),
	)
	defer span.End()

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		middleware.AddSpanError(ctx, err)
		return
	}

	// Create session
	session := &Session{
		Session:  models.NewSession(documentID, userID, userName),
		Conn:     conn,
		Send:     make(chan []byte, 256), // Buffered channel
		Manager:  h.sessionManager,
		ClientID: clientID,
		State:    make(map[string]interface{}),
	}

	// Register session
	h.sessionManager.register <- session

	// Send initial state to client
	h.sendInitialState(session)

	// Start read and write pumps in separate goroutines
	// Learning: Separate goroutines prevent deadlock between reading and writing
	go session.WritePump(ctx)
	go session.ReadPump(ctx)

	log.Printf("✓ WebSocket connection established for document %s (user: %s, client: %d)",
		documentID, userName, clientID)
}

// sendInitialState sends the current document state to a new client
// Learning: For Yjs, we send all historical updates for CRDT reconstruction
func (h *WebSocketHandler) sendInitialState(session *Session) {
	// Get all Yjs updates for this document
	if h.sessionManager.yjsRepo != nil {
		updates, err := h.sessionManager.yjsRepo.GetAllUpdates(context.Background(), session.DocumentID)
		if err != nil {
			log.Printf("Failed to get Yjs updates: %v", err)
			return
		}

		// Send each update to the client
		// Learning: Client's Yjs doc will apply all updates to reconstruct state
		for _, update := range updates {
			select {
			case session.Send <- update.Update:
				// Sent successfully
			default:
				log.Printf("Failed to send initial update to session %s", session.ID)
			}
		}

		log.Printf("Sent %d Yjs updates to session %s for initial sync", len(updates), session.ID)
	}

	// Send awareness state for other users
	if aware := session.Manager.GetAwareness(session.DocumentID); aware != nil {
		awareData, err := json.Marshal(map[string]interface{}{
			"type":      "awareness",
			"awareness": aware,
		})
		if err == nil {
			session.Send <- awareData
		}
	}
}

// HandleUpdatesConnection handles global updates WebSocket
// This can be used for system-wide notifications
func (h *WebSocketHandler) HandleUpdatesConnection(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement global updates channel for notifications
	// For example: "Document X was updated by User Y"
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}
