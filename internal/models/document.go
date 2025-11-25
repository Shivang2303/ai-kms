package models

import (
	"time"

	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

type DocumentFormat string

const (
	FormatMarkdown DocumentFormat = "markdown"
	FormatJSON     DocumentFormat = "json"
	FormatText     DocumentFormat = "text"
)

// Document represents a knowledge base document
// Learning: Using KSUID instead of UUID provides:
// - Time-based sorting (first 32 bits are timestamp)
// - Better database index performance (sequential, less B-tree fragmentation)
// - Smaller string representation (27 chars vs 36 for UUID)
// - No collisions across distributed systems
type Document struct {
	ID        string         `json:"id" gorm:"type:char(27);primaryKey"`
	Title     string         `json:"title" gorm:"type:text;not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Format    DocumentFormat `json:"format" gorm:"type:varchar(50);not null;default:'markdown'"`
	Metadata  map[string]any `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"` // Soft delete support
}

// BeforeCreate hook generates KSUID before inserting
func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = ksuid.New().String()
	}
	return nil
}

type DocumentCreate struct {
	Title    string         `json:"title"`
	Content  string         `json:"content"`
	Format   DocumentFormat `json:"format"`
	Metadata map[string]any `json:"metadata"`
}

type DocumentUpdate struct {
	Title    *string         `json:"title,omitempty"`
	Content  *string         `json:"content,omitempty"`
	Format   *DocumentFormat `json:"format,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}
