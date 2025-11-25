package models

import (
	"time"

	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

/*
LEARNING: YJS CRDT UPDATES

Yjs uses a binary update format to sync CRDT changes.
Each update is a delta that can be applied to reconstruct the document state.

Why persist updates?
- Allows new clients to sync full document history
- Enables offline/online reconciliation
- Provides audit trail of changes

Flow:
  Client A makes change → Generates Yjs update (binary)
  → Send to server → Save to DB → Broadcast to clients
  → Clients apply update → Documents converge
*/

// YjsUpdate stores a single Yjs CRDT update
type YjsUpdate struct {
	ID         string    `gorm:"type:varchar(27);primaryKey" json:"id"`
	DocumentID string    `gorm:"type:varchar(27);not null;index:idx_doc_time" json:"document_id"`
	Update     []byte    `gorm:"type:bytea;not null" json:"-"` // Binary Yjs update
	ClientID   int       `gorm:"not null" json:"client_id"`
	Vector     []byte    `gorm:"type:bytea" json:"-"` // State vector for faster sync
	CreatedAt  time.Time `gorm:"index:idx_doc_time" json:"created_at"`

	// Relationship
	Document *Document `gorm:"foreignKey:DocumentID;references:ID" json:"document,omitempty"`
}

// BeforeCreate generates KSUID
func (y *YjsUpdate) BeforeCreate(tx *gorm.DB) error {
	if y.ID == "" {
		y.ID = ksuid.New().String()
	}
	return nil
}

// TableName override
func (YjsUpdate) TableName() string {
	return "yjs_updates"
}
