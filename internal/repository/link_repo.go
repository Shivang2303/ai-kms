package repository

import (
	"context"
	"fmt"

	"ai-kms/internal/models"

	"gorm.io/gorm"
)

/*
LEARNING: KNOWLEDGE GRAPH REPOSITORY

This repository manages the graph of links between documents.

Operations:
- CreateLink: Add connection between documents
- DeleteLink: Remove connection
- GetOutgoingLinks: Find documents this one links to
- GetIncomingLinks: Find documents that link to this one
- GetGraph: Get entire graph structure
*/

// LinkRepositoryImpl handles knowledge graph operations
type LinkRepositoryImpl struct {
	db *gorm.DB
}

// NewLinkRepository creates a new link repository
func NewLinkRepository(db *gorm.DB) *LinkRepositoryImpl {
	return &LinkRepositoryImpl{db: db}
}

// CreateLink creates a new link between documents
func (r *LinkRepositoryImpl) CreateLink(ctx context.Context, sourceID, targetID, linkType string) (*models.Link, error) {
	link := &models.Link{
		SourceID: sourceID,
		TargetID: targetID,
		LinkType: linkType,
	}

	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	return link, nil
}

// DeleteLink removes a link between documents
func (r *LinkRepositoryImpl) DeleteLink(ctx context.Context, sourceID, targetID string) error {
	result := r.db.WithContext(ctx).
		Where("source_id = ? AND target_id = ?", sourceID, targetID).
		Delete(&models.Link{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete link: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("link not found")
	}

	return nil
}

// GetOutgoingLinks gets all documents that sourceID links to
func (r *LinkRepositoryImpl) GetOutgoingLinks(ctx context.Context, sourceID string) ([]*models.Link, error) {
	var links []*models.Link

	err := r.db.WithContext(ctx).
		Where("source_id = ?", sourceID).
		Preload("Target").
		Find(&links).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing links: %w", err)
	}

	return links, nil
}

// GetIncomingLinks gets all documents that link to targetID (backlinks)
func (r *LinkRepositoryImpl) GetIncomingLinks(ctx context.Context, targetID string) ([]*models.Link, error) {
	var links []*models.Link

	err := r.db.WithContext(ctx).
		Where("target_id = ?", targetID).
		Preload("Source").
		Find(&links).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get incoming links: %w", err)
	}

	return links, nil
}

// GetAllLinks gets all links in the graph
func (r *LinkRepositoryImpl) GetAllLinks(ctx context.Context) ([]*models.Link, error) {
	var links []*models.Link

	err := r.db.WithContext(ctx).
		Preload("Source").
		Preload("Target").
		Find(&links).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all links: %w", err)
	}

	return links, nil
}

// GetGraphNode gets graph information for a specific document
func (r *LinkRepositoryImpl) GetGraphNode(ctx context.Context, documentID string) (*models.GraphNode, error) {
	var outgoingCount int64
	var incomingCount int64

	// Count outgoing links
	if err := r.db.WithContext(ctx).Model(&models.Link{}).
		Where("source_id = ?", documentID).
		Count(&outgoingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count outgoing links: %w", err)
	}

	// Count incoming links
	if err := r.db.WithContext(ctx).Model(&models.Link{}).
		Where("target_id = ?", documentID).
		Count(&incomingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count incoming links: %w", err)
	}

	// Get connected node IDs
	var connectedIDs []string

	// Outgoing
	var outgoingLinks []*models.Link
	if err := r.db.WithContext(ctx).
		Select("target_id").
		Where("source_id = ?", documentID).
		Find(&outgoingLinks).Error; err != nil {
		return nil, err
	}
	for _, link := range outgoingLinks {
		connectedIDs = append(connectedIDs, link.TargetID)
	}

	// Incoming
	var incomingLinks []*models.Link
	if err := r.db.WithContext(ctx).
		Select("source_id").
		Where("target_id = ?", documentID).
		Find(&incomingLinks).Error; err != nil {
		return nil, err
	}
	for _, link := range incomingLinks {
		connectedIDs = append(connectedIDs, link.SourceID)
	}

	// Get document title
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Select("id, title").
		First(&doc, "id = ?", documentID).Error; err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	return &models.GraphNode{
		ID:             documentID,
		Title:          doc.Title,
		OutgoingLinks:  int(outgoingCount),
		IncomingLinks:  int(incomingCount),
		ConnectedNodes: connectedIDs,
	}, nil
}

// GetGraphStats returns statistics about the knowledge graph
func (r *LinkRepositoryImpl) GetGraphStats(ctx context.Context) (*models.GraphStats, error) {
	var totalDocs int64
	var totalLinks int64

	// Count documents
	if err := r.db.WithContext(ctx).Model(&models.Document{}).Count(&totalDocs).Error; err != nil {
		return nil, err
	}

	// Count links
	if err := r.db.WithContext(ctx).Model(&models.Link{}).Count(&totalLinks).Error; err != nil {
		return nil, err
	}

	avgDegree := float64(0)
	if totalDocs > 0 {
		avgDegree = float64(totalLinks*2) / float64(totalDocs) // Each link connects 2 nodes
	}

	return &models.GraphStats{
		TotalDocuments: int(totalDocs),
		TotalLinks:     int(totalLinks),
		AvgDegree:      avgDegree,
		Clusters:       1, // TODO: Implement cluster detection
	}, nil
}

// ExtractLinksFromContent parses [[wiki-style]] links from content
func ExtractLinksFromContent(content string) []string {
	// Simple regex-like extraction of [[link]] patterns
	links := []string{}
	inLink := false
	linkStart := 0

	for i := 0; i < len(content)-1; i++ {
		if content[i] == '[' && content[i+1] == '[' && !inLink {
			inLink = true
			linkStart = i + 2
			i++ // Skip next '['
		} else if content[i] == ']' && content[i+1] == ']' && inLink {
			linkText := content[linkStart:i]
			if len(linkText) > 0 {
				links = append(links, linkText)
			}
			inLink = false
			i++ // Skip next ']'
		}
	}

	return links
}
