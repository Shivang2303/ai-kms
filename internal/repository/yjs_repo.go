package repository

import (
	"context"
	"fmt"

	"ai-kms/internal/models"

	"gorm.io/gorm"
)

/*
LEARNING: YJS UPDATE PERSISTENCE

Storing Yjs updates allows:
1. New clients to get full document state
2. Server restart without data loss
3. Conflict-free merging of offline changes

Query patterns:
- GetAllUpdates: Initial sync (get everything)
- GetUpdatesSince: Incremental sync (get new updates)
- StoreUpdate: Persist changes
*/

// YjsRepositoryImpl handles Yjs update storage
type YjsRepositoryImpl struct {
	db *gorm.DB
}

// NewYjsRepository creates a new Yjs repository
func NewYjsRepository(db *gorm.DB) *YjsRepositoryImpl {
	return &YjsRepositoryImpl{db: db}
}

// StoreUpdate stores a Yjs update
func (r *YjsRepositoryImpl) StoreUpdate(ctx context.Context, documentID string, update []byte, clientID int) error {
	yjsUpdate := &models.YjsUpdate{
		DocumentID: documentID,
		Update:     update,
		ClientID:   clientID,
	}

	if err := r.db.WithContext(ctx).Create(yjsUpdate).Error; err != nil {
		return fmt.Errorf("failed to store yjs update: %w", err)
	}

	return nil
}

// GetAllUpdates retrieves all updates for a document
// Used for initial sync
func (r *YjsRepositoryImpl) GetAllUpdates(ctx context.Context, documentID string) ([]*models.YjsUpdate, error) {
	var updates []*models.YjsUpdate

	err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("created_at ASC").
		Find(&updates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get yjs updates: %w", err)
	}

	return updates, nil
}

// GetUpdatesAfter retrieves updates after a specific update ID
// Used for incremental sync
func (r *YjsRepositoryImpl) GetUpdatesAfter(ctx context.Context, documentID string, afterID string) ([]*models.YjsUpdate, error) {
	var updates []*models.YjsUpdate

	// Get the timestamp of the 'after' update
	var afterUpdate models.YjsUpdate
	if err := r.db.WithContext(ctx).First(&afterUpdate, "id = ?", afterID).Error; err != nil {
		return nil, fmt.Errorf("failed to find reference update: %w", err)
	}

	// Get all updates after that timestamp
	err := r.db.WithContext(ctx).
		Where("document_id = ? AND created_at > ?", documentID, afterUpdate.CreatedAt).
		Order("created_at ASC").
		Find(&updates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get yjs updates: %w", err)
	}

	return updates, nil
}

// GetLatestUpdate gets the most recent update for a document
func (r *YjsRepositoryImpl) GetLatestUpdate(ctx context.Context, documentID string) (*models.YjsUpdate, error) {
	var update models.YjsUpdate

	err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("created_at DESC").
		First(&update).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No updates yet
		}
		return nil, fmt.Errorf("failed to get latest update: %w", err)
	}

	return &update, nil
}

// DeleteOldUpdates removes updates older than a certain threshold
// Call periodically to prevent unbounded growth
func (r *YjsRepositoryImpl) DeleteOldUpdates(ctx context.Context, documentID string, keepCount int) error {
	// Get total count
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.YjsUpdate{}).
		Where("document_id = ?", documentID).
		Count(&count).Error; err != nil {
		return err
	}

	if count <= int64(keepCount) {
		return nil // Nothing to delete
	}

	// Get the cutoff update
	var cutoffUpdate models.YjsUpdate
	offset := count - int64(keepCount)
	if err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("created_at ASC").
		Offset(int(offset)).
		First(&cutoffUpdate).Error; err != nil {
		return err
	}

	// Delete updates before cutoff
	result := r.db.WithContext(ctx).
		Where("document_id = ? AND created_at < ?", documentID, cutoffUpdate.CreatedAt).
		Delete(&models.YjsUpdate{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete old updates: %w", result.Error)
	}

	return nil
}
