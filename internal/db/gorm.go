package db

import (
	"fmt"
	"log"

	"ai-kms/internal/config"
	"ai-kms/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps the GORM database instance
type GormDB struct {
	*gorm.DB
}

// NewGorm initializes a new GORM database connection
// Learning: GORM provides a higher-level abstraction over raw SQL
func NewGorm(cfg *config.Config) (*GormDB, error) {
	dsn := cfg.DatabaseURL()

	// Configure GORM with custom logger
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Shows SQL queries for learning
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return nil, fmt.Errorf("failed to enable pgvector extension: %w", err)
	}

	// Auto-migrate schema
	// Learning: GORM automatically creates/updates tables based on struct definitions
	if err := db.AutoMigrate(
		&models.Document{},
		&models.Embedding{},
		&models.Link{},      // Knowledge graph links
		&models.YjsUpdate{}, // CRDT updates
	); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create vector index for embeddings
	// Note: This is done manually since GORM doesn't have built-in vector index support
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_embeddings_vector 
		ON embeddings USING ivfflat (embedding vector_cosine_ops)
	`).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create vector index: %w", err)
	}

	log.Println("✓ Database connected and migrated successfully")

	return &GormDB{db}, nil
}

// Close closes the database connection
func (db *GormDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
