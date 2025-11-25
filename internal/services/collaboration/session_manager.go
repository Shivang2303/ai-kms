package collaboration

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"ai-kms/internal/middleware"
	"ai-kms/internal/models"
	"ai-kms/internal/repository"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
)

/*
LEARNING: WEBSOCKET SESSION MANAGER

This implements concurrent session management for real-time collaboration.

Key Concepts:
1. **sync.RWMutex**: Read-write lock for concurrent safe map access
2. **Connection Pools**: One pool per document
3. **Broadcast Pattern**: Send message to all connections in a room
4. **Cleanup**: Remove dead connections automatically

The manager uses goroutines for:
*/

// SessionManager manages all active WebSocket sessions
// Learning: Central hub for coordinating real-time collaboration
type SessionManager struct {
	// Session management
	documents  map[string]map[*Session]bool // documentID -> set of sessions
	register   chan *Session
	unregister chan *Session
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex

	// Awareness state (cursor positions, selections)
	awareness map[string]map[int]*models.AwarenessState // documentID -> clientID -> state
	awareMu   sync.RWMutex

	// Yjs repository for persistence
	yjsRepo *repository.YjsRepositoryImpl

	// Control
	done chan struct{}
}

// Session represents an active WebSocket connection
type Session struct {
	*models.Session
	Conn     *websocket.Conn
	Send     chan []byte // Buffered channel for outbound messages
	Manager  *SessionManager
	ClientID int                    // Yjs client ID
	State    map[string]interface{} // Custom session state
}

// BroadcastMessage represents a message to broadcast to a document room
type BroadcastMessage struct {
	DocumentID string
	Message    []byte
	Sender     *Session // Skip this session when broadcasting
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		documents:  make(map[string]map[*Session]bool),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		broadcast:  make(chan *BroadcastMessage, 256),
		awareness:  make(map[string]map[int]*models.AwarenessState),
		done:       make(chan struct{}),
	}
}

// SetYjsRepository sets the Yjs repository for update persistence
func (sm *SessionManager) SetYjsRepository(repo *repository.YjsRepositoryImpl) {
	sm.yjsRepo = repo
}

// Start begins the session manager event loop
// Learning: This goroutine handles all session events concurrently
func (sm *SessionManager) Start() {
	log.Println("🔄 Starting WebSocket session manager...")

	go func() {
		for {
			select {
			case <-sm.done:
				log.Println("Session manager shutting down...")
				return

			case session := <-sm.register:
				sm.handleRegister(session)

			case session := <-sm.unregister:
				sm.handleUnregister(session)

			case msg := <-sm.broadcast:
				sm.handleBroadcast(msg)
			}
		}
	}()

	// Start cleanup goroutine
	go sm.cleanupLoop()

	log.Println("✓ WebSocket session manager started")
}

// handleRegister adds a session to a document room
func (sm *SessionManager) handleRegister(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create document room if doesn't exist
	if sm.documents[session.DocumentID] == nil {
		sm.documents[session.DocumentID] = make(map[*Session]bool)
	}

	sm.documents[session.DocumentID][session] = true

	log.Printf("  Session %s joined document %s (total: %d users)",
		session.ID, session.DocumentID, len(sm.documents[session.DocumentID]))

	// Send join notification to other users
	joinMsg, _ := json.Marshal(map[string]interface{}{
		"type": models.MessageTypeJoin,
		"user": map[string]string{
			"id":   session.UserID,
			"name": session.UserName,
		},
	})

	sm.broadcast <- &BroadcastMessage{
		DocumentID: session.DocumentID,
		Message:    joinMsg,
		Sender:     session, // Don't send to self
	}
}

// handleUnregister removes a session from a document room
func (sm *SessionManager) handleUnregister(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessions, ok := sm.documents[session.DocumentID]; ok {
		if _, ok := sessions[session]; ok {
			delete(sessions, session)
			close(session.Send)

			// Remove empty document rooms
			if len(sessions) == 0 {
				delete(sm.documents, session.DocumentID)
			}

			log.Printf("  Session %s left document %s (remaining: %d users)",
				session.ID, session.DocumentID, len(sessions))

			// Remove from awareness
			sm.awareMu.Lock()
			if aware, exists := sm.awareness[session.DocumentID]; exists {
				delete(aware, session.ClientID)
			}
			sm.awareMu.Unlock()

			// Send leave notification
			leaveMsg, _ := json.Marshal(map[string]interface{}{
				"type": models.MessageTypeLeave,
				"user": map[string]string{
					"id":   session.UserID,
					"name": session.UserName,
				},
			})

			sm.broadcast <- &BroadcastMessage{
				DocumentID: session.DocumentID,
				Message:    leaveMsg,
				Sender:     nil, // Send to everyone
			}
		}
	}
}

// handleBroadcast sends a message to all sessions in a document
func (sm *SessionManager) handleBroadcast(msg *BroadcastMessage) {
	sm.mu.RLock()
	sessions := sm.documents[msg.DocumentID]
	sm.mu.RUnlock()

	for session := range sessions {
		// Skip sender if specified
		if msg.Sender != nil && session == msg.Sender {
			continue
		}

		select {
		case session.Send <- msg.Message:
			// Message queued successfully
		default:
			// Buffer full - connection is slow/dead
			log.Printf("⚠️  Session %s buffer full, closing connection", session.ID)
			sm.unregister <- session
		}
	}
}

// Broadcast sends a message to all users in a document
func (sm *SessionManager) Broadcast(documentID string, message []byte, sender *Session) {
	sm.broadcast <- &BroadcastMessage{
		DocumentID: documentID,
		Message:    message,
		Sender:     sender,
	}
}

// GetSessions returns all active sessions for a document
func (sm *SessionManager) GetSessions(documentID string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := sm.documents[documentID]
	result := make([]*Session, 0, len(sessions))

	for session := range sessions {
		result = append(result, session)
	}

	return result
}

// UpdateAwareness updates user presence state
func (sm *SessionManager) UpdateAwareness(documentID string, clientID int, state *models.AwarenessState) {
	sm.awareMu.Lock()
	defer sm.awareMu.Unlock()

	if sm.awareness[documentID] == nil {
		sm.awareness[documentID] = make(map[int]*models.AwarenessState)
	}

	sm.awareness[documentID][clientID] = state
}

// GetAwareness returns all awareness states for a document
func (sm *SessionManager) GetAwareness(documentID string) map[int]*models.AwarenessState {
	sm.awareMu.RLock()
	defer sm.awareMu.RUnlock()

	return sm.awareness[documentID]
}

// cleanupLoop periodically removes inactive sessions
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.done:
			return
		case <-ticker.C:
			sm.cleanup()
		}
	}
}

// cleanup removes stale sessions
func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	timeout := 5 * time.Minute

	for docID, sessions := range sm.documents {
		for session := range sessions {
			if now.Sub(session.LastActiveAt) > timeout {
				log.Printf("  Cleaning up inactive session %s", session.ID)
				sm.unregister <- session
			}
		}

		// Remove empty document rooms
		if len(sessions) == 0 {
			delete(sm.documents, docID)
		}
	}
}

// Shutdown gracefully closes all connections
func (sm *SessionManager) Shutdown() {
	log.Println("🛑 Shutting down session manager...")

	close(sm.done)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Close all sessions
	for _, sessions := range sm.documents {
		for session := range sessions {
			close(session.Send)
			session.Conn.Close()
		}
	}

	sm.documents = make(map[string]map[*Session]bool)
	log.Println("✓ Session manager shutdown complete")
}

// Session methods

// ReadPump reads messages from the WebSocket connection
// Learning: Each session has its own goroutine reading from the WebSocket
func (s *Session) ReadPump(ctx context.Context) {
	defer func() {
		s.Manager.unregister <- s
		s.Conn.Close()
	}()

	// Set read deadline
	s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.Conn.SetPongHandler(func(string) error {
		s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		s.LastActiveAt = time.Now()
		return nil
	})

	for {
		_, message, err := s.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		s.LastActiveAt = time.Now()

		// Add span for message processing
		msgCtx, span := middleware.StartSpan(ctx, "WebSocket.ProcessMessage",
			attribute.String("session.id", s.ID),
			attribute.String("document.id", s.DocumentID),
			attribute.Int("message.size", len(message)),
		)

		// Store Yjs update if we have repository
		// Learning: Persist CRDT updates for sync and recovery
		if s.Manager.yjsRepo != nil {
			if err := s.Manager.yjsRepo.StoreUpdate(msgCtx, s.DocumentID, message, s.ClientID); err != nil {
				log.Printf("Failed to store Yjs update: %v", err)
				middleware.AddSpanError(msgCtx, err)
			}
		}

		// Broadcast to other clients
		s.Manager.Broadcast(s.DocumentID, message, s)

		span.End()
		_ = msgCtx // Context used for span
	}
}

// WritePump writes messages to the WebSocket connection
// Learning: Separate goroutine for writing prevents blocking on slow clients
func (s *Session) WritePump(ctx context.Context) {
	ticker := time.NewTicker(54 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		s.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.Send:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				s.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.Conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch additional queued messages
			n := len(s.Send)
			for i := 0; i < n; i++ {
				w.Write(<-s.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
