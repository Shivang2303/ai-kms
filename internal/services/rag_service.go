package services

import (
	"context"
	"fmt"
	"strings"

	"ai-kms/internal/middleware"
	"ai-kms/internal/models"
	"ai-kms/internal/openai"

	"go.opentelemetry.io/otel/attribute"
)

/*
LEARNING: RAG (Retrieval Augmented Generation)

RAG combines two powerful techniques:
1. **Retrieval**: Find relevant context from your knowledge base
2. **Generation**: Use LLM to generate answers using that context

Why RAG?
- Reduces hallucination (LLM has real facts)
- Allows querying your own data
- More accurate than pure LLM
- Can cite sources

Flow:
  User Question
    ↓
  Generate Embedding
    ↓
  Semantic Search (find relevant chunks)
    ↓
  Build Prompt with Context
    ↓
  Send to LLM
    ↓
  Get Grounded Answer
*/

// RAGService handles Retrieval Augmented Generation
type RAGService struct {
	openaiClient *openai.Client
	embRepo      EmbeddingRepository
}

// NewRAGService creates a new RAG service
func NewRAGService(
	openaiClient *openai.Client,
	embRepo EmbeddingRepository,
) *RAGService {
	return &RAGService{
		openaiClient: openaiClient,
		embRepo:      embRepo,
	}
}

// QueryWithContext performs RAG query
// Learning: This is the core RAG implementation
func (s *RAGService) QueryWithContext(ctx context.Context, query string, maxChunks int) (string, []*models.SearchResult, error) {
	ctx, span := middleware.StartSpan(ctx, "RAG.QueryWithContext",
		attribute.String("query", query),
		attribute.Int("max_chunks", maxChunks),
	)
	defer span.End()

	// Step 1: Generate embedding for the query
	// Learning: Convert question to same vector space as documents
	queryEmbedding, err := s.openaiClient.CreateEmbeddings([]string{query})
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return "", nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Step 2: Semantic search to find relevant context
	// Learning: Find most similar chunks using cosine similarity
	results, err := s.embRepo.SemanticSearch(ctx, queryEmbedding, maxChunks)
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return "", nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return "I don't have enough context to answer this question.", nil, nil
	}

	// Step 3: Build context from retrieved chunks
	contextParts := make([]string, 0, len(results))
	for i, result := range results {
		contextParts = append(contextParts, fmt.Sprintf(
			"[Document %d] %s:\n%s\n",
			i+1,
			result.Title,
			result.ChunkText,
		))
	}
	context := strings.Join(contextParts, "\n")

	// Step 4: Build prompt with context
	prompt := buildRAGPrompt(query, context)

	// Step 5: Get answer from LLM
	answer, err := s.openaiClient.ChatCompletion(ctx, []openai.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant that answers questions based on the provided context. Always cite which document you're using."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return "", results, fmt.Errorf("failed to get completion: %w", err)
	}

	middleware.AddSpanEvent(ctx, "rag_completed",
		attribute.Int("context_chunks", len(results)),
		attribute.Int("answer_length", len(answer)),
	)

	return answer, results, nil
}

// SummarizeDocument generates a summary of a document
func (s *RAGService) SummarizeDocument(ctx context.Context, content string, maxLength int) (string, error) {
	ctx, span := middleware.StartSpan(ctx, "RAG.SummarizeDocument",
		attribute.Int("content_length", len(content)),
		attribute.Int("max_length", maxLength),
	)
	defer span.End()

	prompt := fmt.Sprintf(
		"Please provide a concise summary of the following document in no more than %d words:\n\n%s",
		maxLength,
		content,
	)

	summary, err := s.openaiClient.ChatCompletion(ctx, []openai.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant that creates concise, accurate summaries."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return summary, nil
}

// ExtractKeywords extracts key topics and keywords from text
func (s *RAGService) ExtractKeywords(ctx context.Context, content string, count int) ([]string, error) {
	ctx, span := middleware.StartSpan(ctx, "RAG.ExtractKeywords",
		attribute.Int("content_length", len(content)),
		attribute.Int("keyword_count", count),
	)
	defer span.End()

	prompt := fmt.Sprintf(
		"Extract %d key topics or keywords from the following text. Return only the keywords separated by commas:\n\n%s",
		count,
		content,
	)

	response, err := s.openaiClient.ChatCompletion(ctx, []openai.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant that extracts key topics and keywords."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return nil, fmt.Errorf("failed to extract keywords: %w", err)
	}

	// Parse comma-separated keywords
	keywords := strings.Split(response, ",")
	result := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		trimmed := strings.TrimSpace(kw)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}

// RelatedDocuments finds documents related to the given document
func (s *RAGService) RelatedDocuments(ctx context.Context, documentContent string, limit int) ([]*models.SearchResult, error) {
	ctx, span := middleware.StartSpan(ctx, "RAG.RelatedDocuments",
		attribute.Int("limit", limit),
	)
	defer span.End()

	// Generate embedding for the document
	embedding, err := s.openaiClient.CreateEmbeddings([]string{documentContent})
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Find similar documents
	results, err := s.embRepo.SemanticSearch(ctx, embedding, limit)
	if err != nil {
		middleware.AddSpanError(ctx, err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
}

// buildRAGPrompt constructs the prompt for RAG
func buildRAGPrompt(query, context string) string {
	return fmt.Sprintf(`Based on the following context, please answer the question. If the context doesn't contain enough information to answer, say so.

Context:
%s

Question: %s

Answer:`, context, query)
}
