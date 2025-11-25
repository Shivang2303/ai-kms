package api

import (
	"context"

	"ai-kms/internal/models"
	"ai-kms/internal/services"
)

/*
LEARNING: CONSUMER-DRIVEN INTERFACES (Go Idiom)

This package (api/handlers) is the CONSUMER of services, so service interfaces live HERE.

The handler doesn't care about service implementation details - it only cares about
the methods it needs to call. This is the "Interface Segregation Principle" from SOLID.

Benefits:
- Handler package defines exactly what it needs
- Service implementations can change without affecting handler
- Easy to create mock services for testing handlers
- No circular dependencies
*/

// EmbeddingService defines what handlers need from the embedding service
// Only methods called by handlers are declared
type EmbeddingService interface {
	Start()
	SubmitJob(job services.EmbeddingJob) error
	Shutdown()
	GetQueueLength() int
}

// Future interfaces for other services used by handlers

// AIService for chat and summarization endpoints
type AIService interface {
	Summarize(ctx context.Context, text string) (string, error)
	Chat(ctx context.Context, messages []services.ChatMessage) (string, error)
}

// SearchService for semantic search endpoints
type SearchService interface {
	Search(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)
}
