package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"ai-kms/internal/models"
	"ai-kms/internal/openai"

	"github.com/pgvector/pgvector-go"
)

/*
LEARNING: EMBEDDING WORKER POOL PATTERN

This service demonstrates a core Go concurrency pattern: the Worker Pool.

Key Concepts:
1. **Goroutines**: Lightweight threads managed by Go runtime
2. **Channels**: Communication pipes between goroutines (thread-safe)
3. **Worker Pool**: Fixed number of workers processing jobs from a queue
4. **Graceful Shutdown**: Using context and WaitGroup for cleanup

Benefits:
- Limits concurrent API calls to OpenAI (avoid rate limits)
- Efficient resource usage (reuse workers vs spawning unlimited goroutines)
- Backpressure handling (bounded queue prevents memory issues)
*/

// EmbeddingJob represents a document that needs embeddings generated
type EmbeddingJob struct {
	DocumentID string
	Content    string
}

// EmbeddingServiceImpl handles document embedding generation with a worker pool
// This is the IMPLEMENTATION - the api package defines what interface it needs
type EmbeddingServiceImpl struct {
	openaiClient *openai.Client
	embRepo      EmbeddingRepository  // Interface from this package (consumer-driven!)
	docRepo      DocumentRepository   // Interface from this package

	// Worker pool components
	jobs    chan EmbeddingJob      // Buffered channel for job queue
	workers int                    // Number of concurrent workers
	wg      sync.WaitGroup         // Tracks active workers for graceful shutdown
	ctx     context.Context        // Context for cancellation
	cancel  context.CancelFunc     // Function to cancel all workers
}

// NewEmbeddingService creates a new embedding service with worker pool
// Learning: This initializes the worker pool but doesn't start it yet
// Returns concrete type - "Accept interfaces, return structs"
func NewEmbeddingService(
	openaiClient *openai.Client,
	embRepo EmbeddingRepository,    // Accept interface (consumer defines this)
	docRepo DocumentRepository,     // Accept interface
	numWorkers int,
	queueSize int,
) *EmbeddingServiceImpl {
	ctx, cancel := context.WithCancel(context.Background())

	return &EmbeddingServiceImpl{
		openaiClient: openaiClient,
		embRepo:      embRepo,
		docRepo:      docRepo,
		jobs:         make(chan EmbeddingJob, queueSize), // Buffered channel
		workers:      numWorkers,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start initializes the worker pool
// Learning: Spawns numWorkers goroutines that wait for jobs
func (s *EmbeddingServiceImpl) Start() {
	log.Printf("🔧 Starting embedding worker pool with %d workers", s.workers)

	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		// Spawn a worker goroutine
		// Learning: Each worker runs concurrently
		go s.worker(i)
	}

	log.Println("✓ Embedding worker pool started")
}

// worker is the goroutine function that processes jobs
// Learning: This runs in a loop, pulling jobs from the channel
func (s *EmbeddingServiceImpl) worker(id int) {
	defer s.wg.Done() // Decrement WaitGroup when worker exits

	log.Printf("  Worker %d started", id)

	for {
		select {
		case <-s.ctx.Done():
			// Context cancelled - shutdown signal
			log.Printf("  Worker %d shutting down", id)
			return

		case job, ok := <-s.jobs:
			if !ok {
				// Channel closed - no more jobs
				log.Printf("  Worker %d: jobs channel closed", id)
				return
			}

			// Process the job
			log.Printf("  Worker %d processing document %s", id, job.DocumentID)
			if err := s.processEmbedding(job); err != nil {
				log.Printf("  Worker %d error: %v", id, err)
			} else {
				log.Printf("  Worker %d completed document %s", id, job.DocumentID)
			}
		}
	}
}

// SubmitJob adds a job to the queue
// Learning: This is non-blocking if queue has space, blocks if full (backpressure)
func (s *EmbeddingServiceImpl) SubmitJob(job EmbeddingJob) error {
	select {
	case s.jobs <- job:
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("service is shutting down")
	}
}

// processEmbedding handles the actual embedding generation
// Learning: This is where the real work happens - called by workers
func (s *EmbeddingServiceImpl) processEmbedding(job EmbeddingJob) error {
	ctx := context.Background()

	// Delete existing embeddings for this document
	if err := s.embRepo.DeleteEmbeddingsByDocumentID(ctx, job.DocumentID); err != nil {
		return fmt.Errorf("failed to delete old embeddings: %w", err)
	}

	// Chunk the document content
	// Learning: Large documents need to be split into chunks for embedding
	chunks := s.chunkText(job.Content, 500) // 500 words per chunk

	// Generate embeddings for each chunk
	for i, chunk := range chunks {
		// Call OpenAI API to generate embedding
		embeddingVector, err := s.openaiClient.CreateEmbeddings([]string{chunk})
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		// Convert to pgvector format
		vec := pgvector.NewVector(embeddingVector)

		// Store in database
		embedding := &models.Embedding{
			DocumentID: job.DocumentID,
			ChunkIndex: i,
			ChunkText:  chunk,
			Embedding:  vec,
		}

		if err := s.embRepo.StoreEmbedding(ctx, embedding); err != nil {
			return fmt.Errorf("failed to store embedding for chunk %d: %w", i, err)
		}
	}

	log.Printf("  Generated %d embeddings for document %s", len(chunks), job.DocumentID)
	return nil
}

// chunkText splits text into chunks of approximately maxWords words
// Learning: Embeddings work best on reasonably-sized chunks
func (s *EmbeddingServiceImpl) chunkText(text string, maxWords int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var chunks []string
	for i := 0; i < len(words); i += maxWords {
		end := i + maxWords
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
	}

	return chunks
}

// Shutdown gracefully stops the worker pool
// Learning: Important for clean shutdown - wait for all workers to finish
func (s *EmbeddingServiceImpl) Shutdown() {
	log.Println("🛑 Shutting down embedding service...")

	// Close the jobs channel - no more jobs accepted
	close(s.jobs)

	// Cancel context to signal workers
	s.cancel()

	// Wait for all workers to finish current jobs
	s.wg.Wait()

	log.Println("✓ Embedding service shutdown complete")
}

// GetQueueLength returns current number of pending jobs
// Learning: Useful for monitoring/debugging
func (s *EmbeddingServiceImpl) GetQueueLength() int {
	return len(s.jobs)
}
