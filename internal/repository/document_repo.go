package repository

import (
	"context"
	"fmt"

	"ai-kms/internal/models"

	"gorm.io/gorm"
)

// DocumentRepositoryImpl handles all database operations for documents using GORM
// Learning: This is the IMPLEMENTATION. It doesn't know about any interface.
// The services package will declare the interface it needs.
type DocumentRepositoryImpl struct {
	db *gorm.DB
}

// NewDocumentRepository creates a new document repository
// Returns concrete type - "Accept interfaces, return structs"
func NewDocumentRepository(db *gorm.DB) *DocumentRepositoryImpl {
	return &DocumentRepositoryImpl{db: db}
}

// Create inserts a new document into the database
// The KSUID is auto-generated in the BeforeCreate hook
func (r *DocumentRepositoryImpl) Create(ctx context.Context, doc *models.DocumentCreate) (*models.Document, error) {
	document := &models.Document{
		Title:    doc.Title,
		Content:  doc.Content,
		Format:   doc.Format,
		Metadata: doc.Metadata,
	}

	// GORM automatically:
	// 1. Generates KSUID via BeforeCreate hook
	// 2. Sets timestamps (CreatedAt, UpdatedAt)
	if err := r.db.WithContext(ctx).Create(document).Error; err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return document, nil
}

// GetByID retrieves a document by its KSUID
// Soft-deleted documents are automatically excluded
func (r *DocumentRepositoryImpl) GetByID(ctx context.Context, id string) (*models.Document, error) {
	var doc models.Document

	err := r.db.WithContext(ctx).First(&doc, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// List returns all documents with pagination
// Learning: KSUID allows natural time-based ordering without created_at index
func (r *DocumentRepositoryImpl) List(ctx context.Context, limit, offset int) ([]*models.Document, error) {
	var documents []*models.Document

	err := r.db.WithContext(ctx).
		Order("id DESC"). // KSUID is time-ordered, so sorting by ID = sorting by creation time
		Limit(limit).
		Offset(offset).
		Find(&documents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return documents, nil
}

// Update modifies an existing document
// Learning: GORM's Updates() only updates non-zero fields
func (r *DocumentRepositoryImpl) Update(ctx context.Context, id string, update *models.DocumentUpdate) (*models.Document, error) {
	var doc models.Document

	// First, find the document
	if err := r.db.WithContext(ctx).First(&doc, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// Build update map to handle nil pointers correctly
	updates := make(map[string]interface{})
	if update.Title != nil {
		updates["title"] = *update.Title
	}
	if update.Content != nil {
		updates["content"] = *update.Content
	}
	if update.Format != nil {
		updates["format"] = *update.Format
	}
	if update.Metadata != nil {
		updates["metadata"] = update.Metadata
	}

	// Perform update (UpdatedAt is automatically set by GORM)
	if err := r.db.WithContext(ctx).Model(&doc).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &doc, nil
}

// Delete performs a soft delete on the document
// Learning: GORM automatically sets DeletedAt timestamp instead of removing the row
// This allows data recovery and audit trails
func (r *DocumentRepositoryImpl) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Document{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}

// HardDelete permanently removes a document (bypasses soft delete)
// Use with caution - this is irreversible
func (r *DocumentRepositoryImpl) HardDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&models.Document{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to hard delete document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}
