package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

// Embedding represents a document chunk with its vector embedding
// Using KSUID for time-ordered IDs and better database performance
type Embedding struct {
	ID         string          `json:"id" gorm:"type:char(27);primaryKey"`
	DocumentID string          `json:"document_id" gorm:"type:char(27);not null;index"`
	ChunkIndex int             `json:"chunk_index" gorm:"not null"`
	ChunkText  string          `json:"chunk_text" gorm:"type:text;not null"`
	Embedding  pgvector.Vector `json:"embedding" gorm:"type:vector(1536);not null"`
	CreatedAt  time.Time       `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	DeletedAt  gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"` // Soft delete

	// Relationship
	Document Document `json:"document,omitempty" gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE"`
}

// BeforeCreate hook generates KSUID before inserting
func (e *Embedding) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = ksuid.New().String()
	}
	return nil
}

// SearchResult represents a semantic search result
type SearchResult struct {
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	ChunkText  string  `json:"chunk_text"`
	Score      float32 `json:"score"` // Similarity score (0-1)
}
