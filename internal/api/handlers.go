package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"ai-kms/internal/models"
	"ai-kms/internal/openai"
	"ai-kms/internal/repository"
	"ai-kms/internal/services"
	"ai-kms/internal/services/collaboration"

	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
// Learning: Uses INTERFACES defined in this package (consumer-driven)
type Handler struct {
	docRepo      *repository.DocumentRepositoryImpl  // Concrete type for now
	embRepo      *repository.EmbeddingRepositoryImpl // Concrete type for now
	embService   EmbeddingService                    // Interface defined in this package!
	wsHandler    *collaboration.WebSocketHandler     // WebSocket for real-time collab
	ragService   *services.RAGService                // RAG for AI features
	openaiClient *openai.Client                      // OpenAI client
	linkRepo     *repository.LinkRepositoryImpl      // Knowledge graph links
}

func NewHandler(
	docRepo *repository.DocumentRepositoryImpl,
	embRepo *repository.EmbeddingRepositoryImpl,
	embService EmbeddingService, // Accept interface
	wsHandler *collaboration.WebSocketHandler,
	ragService *services.RAGService,
	openaiClient *openai.Client,
	linkRepo *repository.LinkRepositoryImpl,
) *Handler {
	return &Handler{
		docRepo:      docRepo,
		embRepo:      embRepo,
		embService:   embService,
		wsHandler:    wsHandler,
		ragService:   ragService,
		openaiClient: openaiClient,
		linkRepo:     linkRepo,
	}
}

// Document handlers

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var doc models.DocumentCreate
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default format if not provided
	if doc.Format == "" {
		doc.Format = models.FormatMarkdown
	}

	created, err := h.docRepo.Create(r.Context(), &doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Automatically submit for embedding generation
	// Learning: Submitting to worker pool is non-blocking
	job := services.EmbeddingJob{
		DocumentID: created.ID,
		Content:    created.Content,
	}
	if err := h.embService.SubmitJob(job); err != nil {
		// Log but don't fail the request
		// The embedding can be regenerated later
		http.Error(w, "Document created but embedding generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	documents, err := h.docRepo.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": documents,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	doc, err := h.docRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var update models.DocumentUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := h.docRepo.Update(r.Context(), id, &update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If content was updated, regenerate embeddings
	if update.Content != nil {
		job := services.EmbeddingJob{
			DocumentID: updated.ID,
			Content:    updated.Content,
		}
		_ = h.embService.SubmitJob(job) // Best effort
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check for hard delete flag
	hardDelete := r.URL.Query().Get("hard") == "true"

	var err error
	if hardDelete {
		err = h.docRepo.HardDelete(r.Context(), id)
	} else {
		err = h.docRepo.Delete(r.Context(), id) // Soft delete
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Embedding handlers

func (h *Handler) GenerateEmbeddings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get the document
	doc, err := h.docRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Submit to worker pool
	job := services.EmbeddingJob{
		DocumentID: doc.ID,
		Content:    doc.Content,
	}

	if err := h.embService.SubmitJob(job); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Embedding generation submitted",
		"document_id":  doc.ID,
		"queue_length": h.embService.GetQueueLength(),
	})
}

func (h *Handler) SemanticSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	// Generate embedding for query
	embedding, err := h.openaiClient.CreateEmbeddings([]string{req.Query})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate query embedding: %v", err), http.StatusInternalServerError)
		return
	}

	// Perform semantic search
	results, err := h.embRepo.SemanticSearch(r.Context(), embedding, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}

// RAG handlers

func (h *Handler) QueryWithRAG(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string `json:"query"`
		MaxChunks int    `json:"max_chunks,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.MaxChunks == 0 {
		req.MaxChunks = 5
	}

	answer, sources, err := h.ragService.QueryWithContext(r.Context(), req.Query, req.MaxChunks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   req.Query,
		"answer":  answer,
		"sources": sources,
	})
}

func (h *Handler) SummarizeDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		MaxWords int `json:"max_words,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body, use default
		req.MaxWords = 150
	}

	if req.MaxWords == 0 {
		req.MaxWords = 150
	}

	// Get document
	doc, err := h.docRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Generate summary
	summary, err := h.ragService.SummarizeDocument(r.Context(), doc.Content, req.MaxWords)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"document_id": id,
		"summary":     summary,
		"max_words":   req.MaxWords,
	})
}

func (h *Handler) QueryDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get document
	doc, err := h.docRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Use document as context for the query
	answer, err := h.ragService.SummarizeDocument(r.Context(),
		fmt.Sprintf("Based on this document:\n\n%s\n\nQuestion: %s", doc.Content, req.Query),
		300,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"document_id": id,
		"query":       req.Query,
		"answer":      answer,
	})
}

// Knowledge graph handlers

func (h *Handler) GetKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	links, err := h.linkRepo.GetAllLinks(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := h.linkRepo.GetGraphStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"links": links,
		"stats": stats,
	})
}

func (h *Handler) GenerateKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	// Get all documents
	docs, err := h.docRepo.List(r.Context(), 1000, 0) // Get all docs
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	linksCreated := 0

	// Extract [[links]] from each document
	for _, doc := range docs {
		linkTitles := repository.ExtractLinksFromContent(doc.Content)

		for _, linkTitle := range linkTitles {
			// Find target document by title
			targetDocs, err := h.docRepo.List(r.Context(), 1, 0) // Simple search
			if err != nil {
				continue
			}

			// Find matching document - in production, use proper search
			for _, target := range targetDocs {
				if target.Title == linkTitle {
					// Create link
					_, err := h.linkRepo.CreateLink(r.Context(), doc.ID, target.ID, "reference")
					if err == nil {
						linksCreated++
					}
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Knowledge graph generated",
		"links_created": linksCreated,
	})
}

func (h *Handler) GetGraphNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	node, err := h.linkRepo.GetGraphNode(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get outgoing and incoming links
	outgoing, _ := h.linkRepo.GetOutgoingLinks(r.Context(), id)
	incoming, _ := h.linkRepo.GetIncomingLinks(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node":           node,
		"outgoing_links": outgoing,
		"incoming_links": incoming,
	})
}
