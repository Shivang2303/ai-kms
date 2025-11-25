# AI Knowledge Management System

An Obsidian-like knowledge management system with AI capabilities, built with Go, PostgreSQL (pgvector), and OpenAI.

## Features

- Document management (Markdown, JSON, Text)
- Semantic search using pgvector embeddings
- RAG (Retrieval Augmented Generation) for summarization and Q&A
- Knowledge graph generation and visualization
- Real-time collaborative editing via WebSocket
- OpenAI integration for LLM features

## Prerequisites

- Go 1.24+
- PostgreSQL 12+ with pgvector extension
- OpenAI API key

## Setup

1. Clone the repository
2. Copy `.env.example` to `.env` and configure:
   ```bash
   cp .env.example .env
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Set up PostgreSQL with pgvector:
   ```sql
   CREATE EXTENSION vector;
   ```

5. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

6. Open `http://localhost:8080` in your browser

## Project Structure

- `cmd/server/` - Application entry point
- `internal/api/` - HTTP handlers and WebSocket handlers
- `internal/db/` - Database connection and migrations
- `internal/models/` - Data models
- `internal/openai/` - OpenAI API client
- `internal/config/` - Configuration management
- `web/static/` - Frontend files (HTML, CSS, JS)

## API Endpoints

- `POST /api/documents` - Create document
- `GET /api/documents` - List documents
- `GET /api/documents/:id` - Get document
- `PUT /api/documents/:id` - Update document
- `DELETE /api/documents/:id` - Delete document
- `POST /api/documents/:id/embed` - Generate embeddings
- `POST /api/documents/:id/summarize` - Summarize document
- `POST /api/documents/:id/query` - RAG Q&A
- `GET /api/graph` - Get knowledge graph
- `POST /api/graph/generate` - Generate knowledge graph
- `WS /ws/document/:id` - WebSocket for live editing
- `WS /ws/updates` - WebSocket for system updates

## Development

The boilerplate provides the foundation. You'll need to implement:

1. Document CRUD operations in handlers
2. Embedding generation and storage
3. RAG service for summarization and queries
4. Knowledge graph generation logic
5. WebSocket message broadcasting for collaboration
6. Frontend graph visualization

## License

MIT

