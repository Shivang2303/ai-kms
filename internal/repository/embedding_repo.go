package repository

import (
	"context"
	"fmt"

	"ai-kms/internal/models"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// EmbeddingRepositoryImpl handles vector operations using pgvector
// This is the IMPLEMENTATION - doesn't know about interfaces
type EmbeddingRepositoryImpl struct {
	db *gorm.DB
}

// NewEmbeddingRepository creates a new embedding repository
// Returns concrete type - consumer will use interface
func NewEmbeddingRepository(db *gorm.DB) *EmbeddingRepositoryImpl {
	return &EmbeddingRepositoryImpl{db: db}
}

// StoreEmbedding saves a document chunk embedding to the database
// KSUID is auto-generated via BeforeCreate hook
func (r *EmbeddingRepositoryImpl) StoreEmbedding(ctx context.Context, embedding *models.Embedding) error {
	if err := r.db.WithContext(ctx).Create(embedding).Error; err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}
	return nil
}

// GetEmbeddingsByDocumentID retrieves all embeddings for a document
func (r *EmbeddingRepositoryImpl) GetEmbeddingsByDocumentID(ctx context.Context, docID string) ([]*models.Embedding, error) {
	var embeddings []*models.Embedding

	err := r.db.WithContext(ctx).
		Where("document_id = ?", docID).
		Order("chunk_index").
		Find(&embeddings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings: %w", err)
	}

	return embeddings, nil
}

// SemanticSearch performs vector similarity search using cosine distance
// Learning: The <=> operator from pgvector calculates cosine distance
// Lower distance = more similar documents
func (r *EmbeddingRepositoryImpl) SemanticSearch(ctx context.Context, queryEmbedding []float32, limit int) ([]*models.SearchResult, error) {
	vec := pgvector.NewVector(queryEmbedding)

	var results []*models.SearchResult

	// Using raw SQL for vector operations since GORM doesn't have native support
	// The <=> operator is from pgvector and calculates cosine distance
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			e.document_id,
			d.title,
			e.chunk_text,
			1 - (e.embedding <=> ?) as score
		FROM embeddings e
		JOIN documents d ON d.id = e.document_id
		WHERE e.deleted_at IS NULL AND d.deleted_at IS NULL
		ORDER BY e.embedding <=> ?
		LIMIT ?
	`, vec, vec, limit).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to perform semantic search: %w", err)
	}

	return results, nil
}

// DeleteEmbeddingsByDocumentID performs soft delete on all embeddings for a document
func (r *EmbeddingRepositoryImpl) DeleteEmbeddingsByDocumentID(ctx context.Context, docID string) error {
	if err := r.db.WithContext(ctx).Where("document_id = ?", docID).Delete(&models.Embedding{}).Error; err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}
	return nil
}
