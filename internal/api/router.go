package api

import (
	"net/http"

	"ai-kms/internal/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes(h *Handler) *mux.Router {
	r := mux.NewRouter()

	// Apply global middleware
	// Learning: Middleware runs in order - tracing first, then recovery, then CORS
	r.Use(middleware.TracingMiddleware)       // Add tracing spans to all requests
	r.Use(middleware.ErrorRecoveryMiddleware) // Catch panics
	r.Use(middleware.CORSMiddleware)          // Handle CORS

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Document endpoints
	api.HandleFunc("/documents", h.CreateDocument).Methods("POST")
	api.HandleFunc("/documents", h.ListDocuments).Methods("GET")
	api.HandleFunc("/documents/{id}", h.GetDocument).Methods("GET")
	api.HandleFunc("/documents/{id}", h.UpdateDocument).Methods("PUT")
	api.HandleFunc("/documents/{id}", h.DeleteDocument).Methods("DELETE")

	// Embedding endpoints
	api.HandleFunc("/documents/{id}/embed", h.GenerateEmbeddings).Methods("POST")
	api.HandleFunc("/search", h.SemanticSearch).Methods("POST")

	// RAG endpoints
	api.HandleFunc("/documents/{id}/summarize", h.SummarizeDocument).Methods("POST")
	api.HandleFunc("/documents/{id}/query", h.QueryDocument).Methods("POST")

	// AI/RAG endpoints
	api.HandleFunc("/search", h.SemanticSearch).Methods("POST")
	api.HandleFunc("/ai/query", h.QueryWithRAG).Methods("POST")
	api.HandleFunc("/ai/summarize/{id}", h.SummarizeDocument).Methods("POST")
	api.HandleFunc("/ai/query/{id}", h.QueryDocument).Methods("POST")

	// Knowledge graph endpoints
	api.HandleFunc("/graph", h.GetKnowledgeGraph).Methods("GET")
	api.HandleFunc("/graph/generate", h.GenerateKnowledgeGraph).Methods("POST")
	api.HandleFunc("/graph/nodes/{id}", h.GetGraphNode).Methods("GET")

	// Health check endpoint
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// WebSocket routes
	r.HandleFunc("/ws/document/{id}", h.HandleDocumentWebSocket)
	r.HandleFunc("/ws/updates", h.HandleUpdatesWebSocket)

	// Serve HTML files
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/static/index.html")
	})
	r.HandleFunc("/editor.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/static/editor.html")
	})
	r.HandleFunc("/graph.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/static/graph.html")
	})

	// Serve static assets
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	return r
}
