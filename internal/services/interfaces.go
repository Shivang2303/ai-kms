package services

import (
	"context"

	"ai-kms/internal/models"
)

/*
LEARNING: GO INTERFACE BEST PRACTICE

"Accept interfaces, return structs" - Rob Pike

Key Principle: Interfaces should be defined where they are USED, not where implemented.

Why?
1. Consumer-driven design: The user of the dependency defines what it needs
2. Smaller, focused interfaces: Only declare methods you actually use
3. No circular dependencies: Implementation doesn't know about interface
4. Better testability: Easy to mock exactly what you need

Example:
  // ❌ BAD: Interface in repository package
  package repository
  type DocumentRepository interface { ... }

  // ✅ GOOD: Interface in services package (consumer)
  package services
  type DocumentRepository interface { ... }

This package (services) is the CONSUMER of repositories, so interfaces go here!
*/

// DocumentRepository defines what the service needs from document storage
// Only methods actually used by services are declared here
type DocumentRepository interface {
	Create(ctx context.Context, doc *models.DocumentCreate) (*models.Document, error)
	GetByID(ctx context.Context, id string) (*models.Document, error)
	List(ctx context.Context, limit, offset int) ([]*models.Document, error)
	Update(ctx context.Context, id string, update *models.DocumentUpdate) (*models.Document, error)
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
}

// EmbeddingRepository defines what the service needs from embedding storage
type EmbeddingRepository interface {
	StoreEmbedding(ctx context.Context, embedding *models.Embedding) error
	GetEmbeddingsByDocumentID(ctx context.Context, docID string) ([]*models.Embedding, error)
	SemanticSearch(ctx context.Context, queryEmbedding []float32, limit int) ([]*models.SearchResult, error)
	DeleteEmbeddingsByDocumentID(ctx context.Context, docID string) error
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// SearchFilters for advanced search
type SearchFilters struct {
	Format     *models.DocumentFormat
	MinScore   float32
	DateFrom   *string
	DateTo     *string
	ExcludeIDs []string
}

// Future service interfaces (when we implement them)

// AIService will handle OpenAI chat completions, summarization
type AIService interface {
	Summarize(ctx context.Context, text string) (string, error)
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
	ExtractKeywords(ctx context.Context, text string) ([]string, error)
}

// SearchService will handle semantic search operations
type SearchService interface {
	Search(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)
	SearchWithFilters(ctx context.Context, query string, filters SearchFilters, limit int) ([]*models.SearchResult, error)
}
