package models

import (
	"time"

	"github.com/segmentio/ksuid"
)

// Session represents an active WebSocket connection to a document
type Session struct {
	ID           string    `json:"id"`
	DocumentID   string    `json:"document_id"`
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// AwarenessState represents user presence information (cursor, selection, etc.)
// Learning: This is separate from document content - it's ephemeral user state
type AwarenessState struct {
	ClientID int                    `json:"client_id"`
	User     *UserInfo              `json:"user,omitempty"`
	Cursor   *CursorPosition        `json:"cursor,omitempty"`
	State    map[string]interface{} `json:"state,omitempty"`
}

// UserInfo represents information about a connected user
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"` // Hex color for cursor/highlight
}

// CursorPosition represents where a user's cursor is in the document
type CursorPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// YjsMessage represents a message in the Yjs sync protocol
// Learning: Yjs uses a binary protocol for efficient CRDT synchronization
type YjsMessage struct {
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
}

// MessageType defines types of messages in the collaboration protocol
type MessageType int

const (
	// Yjs sync protocol messages
	MessageTypeSync          MessageType = 0 // Sync step 1: Send state vector
	MessageTypeSyncUpdate    MessageType = 1 // Sync step 2: Send missing updates
	MessageTypeAwareness     MessageType = 2 // Awareness (cursors, users)
	MessageTypeQueryAwareness MessageType = 3 // Request awareness state

	// Custom application messages
	MessageTypeJoin  MessageType = 10 // User joined
	MessageTypeLeave MessageType = 11 // User left
	MessageTypeError MessageType = 99 // Error message
)

func NewSession(documentID, userID, userName string) *Session {
	return &Session{
		ID:           ksuid.New().String(),
		DocumentID:   documentID,
		UserID:       userID,
		UserName:     userName,
		ConnectedAt:  time.Now(),
		LastActiveAt: time.Now(),
	}
}
