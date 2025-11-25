package models

import (
	"time"

	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

/*
LEARNING: KNOWLEDGE GRAPH

A knowledge graph represents relationships between documents (nodes).
Think of it like a wiki with [[links]] between pages.

Structure:
  Document A [[links to]] Document B
  Document B [[backlinks from]] Document A

This enables:
- Bidirectional navigation
- Graph visualization
- Finding connected knowledge
- Discovering patterns
*/

// Link represents a connection between two documents
type Link struct {
	ID        string         `gorm:"type:varchar(27);primaryKey" json:"id"`
	SourceID  string         `gorm:"type:varchar(27);not null;index" json:"source_id"`
	TargetID  string         `gorm:"type:varchar(27);not null;index" json:"target_id"`
	LinkType  string         `gorm:"type:varchar(50);default:'reference'" json:"link_type"` // reference, related, parent, child
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Source *Document `gorm:"foreignKey:SourceID;references:ID" json:"source,omitempty"`
	Target *Document `gorm:"foreignKey:TargetID;references:ID" json:"target,omitempty"`
}

// BeforeCreate generates KSUID before creating
func (l *Link) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = ksuid.New().String()
	}
	return nil
}

// GraphNode represents a document in the knowledge graph
type GraphNode struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	OutgoingLinks  int      `json:"outgoing_links"`  // How many docs this links to
	IncomingLinks  int      `json:"incoming_links"`  // How many docs link to this
	ConnectedNodes []string `json:"connected_nodes"` // IDs of connected documents
}

// GraphStats represents overall graph statistics
type GraphStats struct {
	TotalDocuments int     `json:"total_documents"`
	TotalLinks     int     `json:"total_links"`
	AvgDegree      float64 `json:"avg_degree"` // Average connections per node
	Clusters       int     `json:"clusters"`   // Number of disconnected subgraphs
}

// TableName override
func (Link) TableName() string {
	return "links"
}
