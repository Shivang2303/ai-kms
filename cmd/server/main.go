package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-kms/internal/api"
	"ai-kms/internal/config"
	"ai-kms/internal/db"
	"ai-kms/internal/openai"
	"ai-kms/internal/repository"
	"ai-kms/internal/services"
	"ai-kms/internal/services/collaboration"
	"ai-kms/internal/telemetry"
)

/*
LEARNING: GRACEFUL SHUTDOWN PATTERN WITH OBSERVABILITY

This main function demonstrates:
1. Service initialization and dependency injection
2. Concurrent server and worker pool management
3. Distributed tracing with Jaeger
4. Graceful shutdown handling (listening for SIGINT/SIGTERM)
5. Proper resource cleanup order
*/

func main() {
	log.Println("🚀 Starting AI Knowledge Management System...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Initialize Jaeger tracing
	// Learning: Do this FIRST so all operations are traced
	jaegerShutdown, err := telemetry.InitJaeger("ai-kms", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("⚠️  Failed to initialize Jaeger: %v (continuing without tracing)", err)
		jaegerShutdown = func(ctx context.Context) error { return nil }
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := jaegerShutdown(ctx); err != nil {
			log.Printf("⚠️  Failed to shutdown Jaeger: %v", err)
		}
	}()

	// Initialize GORM database
	database, err := db.NewGorm(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize OpenAI client
	openaiClient := openai.NewClient(cfg.OpenAIAPIKey)
	log.Println("✓ OpenAI client initialized")

	// Initialize repositories
	docRepo := repository.NewDocumentRepository(database.DB)
	embRepo := repository.NewEmbeddingRepository(database.DB)

	// Initialize embedding service with worker pool
	// Learning: This creates the worker pool but doesn't start it yet
	embService := services.NewEmbeddingService(
		openaiClient,
		embRepo,
		docRepo,
		cfg.EmbeddingWorkers,
		cfg.EmbeddingQueueSize,
	)

	// Start the worker pool
	// Learning: This spawns goroutines that will process jobs concurrently
	embService.Start()

	// Initialize WebSocket session manager for real-time collaboration
	// Learning: Phase 2 - Real-time collaboration with CRDTs
	sessionManager := collaboration.NewSessionManager()

	// Initialize Yjs repository for CRDT persistence
	yjsRepo := repository.NewYjsRepository(database.DB)
	sessionManager.SetYjsRepository(yjsRepo)

	sessionManager.Start()

	// Initialize WebSocket handler
	wsHandler := collaboration.NewWebSocketHandler(sessionManager)

	// Initialize RAG service for AI features
	// Learning: RAG combines retrieval (semantic search) with generation (LLM)
	ragService := services.NewRAGService(openaiClient, embRepo)

	// Initialize knowledge graph repository
	linkRepo := repository.NewLinkRepository(database.DB)

	// Initialize handlers with dependency injection
	handler := api.NewHandler(docRepo, embRepo, embService, wsHandler, ragService, openaiClient, linkRepo)

	// Setup routes
	router := api.SetupRoutes(handler)

	// Configure HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a goroutine
	// Learning: This allows us to handle shutdown signals concurrently
	go func() {
		log.Printf("🌐 Server listening on http://%s", addr)
		log.Printf("📚 API Endpoints:")
		log.Printf("   POST   /api/documents           - Create document")
		log.Printf("   GET    /api/documents           - List documents")
		log.Printf("   GET    /api/documents/:id       - Get document")
		log.Printf("   PUT    /api/documents/:id       - Update document")
		log.Printf("   DELETE /api/documents/:id       - Delete document (soft)")
		log.Printf("   POST   /api/documents/:id/embed - Generate embeddings")
		log.Println()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	// Learning: This is the graceful shutdown pattern
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")

	// Shutdown HTTP server with timeout
	// Learning: Give the server 30 seconds to finish existing requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	// Shutdown embedding service
	// Learning: This waits for workers to finish their current jobs
	embService.Shutdown()

	// Shutdown WebSocket session manager
	// Learning: This closes all active WebSocket connections gracefully
	sessionManager.Shutdown()

	log.Println("✓ Server shutdown complete")
}
