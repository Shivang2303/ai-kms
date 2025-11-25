# AI Knowledge Management System - Setup Guide

## Prerequisites

- Go 1.24+
- PostgreSQL 12+ (with superuser access to install extensions)
- OpenAI API key
- (Optional) Docker for easy PostgreSQL setup

## Step 1: Database Setup

### Option A: Using Docker (Recommended)

```bash
# Run PostgreSQL with pgvector extension
docker run -d \
  --name ai-kms-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=ai_kms \
  -p 5432:5432 \
  ankane/pgvector
```

### Option B: Manual PostgreSQL Setup

```bash
# Install pgvector extension (Ubuntu/Debian)
sudo apt-get install postgresql-14-pgvector

# Or on MacOS
brew install pgvector

# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE ai_kms;

# Connect to the database
\c ai_kms

# Enable pgvector extension
CREATE EXTENSION vector;
```

## Step 2: Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your settings
nano .env
```

Required configuration:
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=ai_kms

# OpenAI (REQUIRED)
OPENAI_API_KEY=sk-your-api-key-here

# Server
SERVER_HOST=localhost
SERVER_PORT=8080

# Worker Pool (optional, has defaults)
EMBEDDING_WORKERS=5
EMBEDDING_QUEUE_SIZE=100
```

## Step 3: Install Dependencies

```bash
# Download Go dependencies
go mod download

# Verify installation
go mod verify
```

## Step 4: Build the Server

```bash
# Build binary
go build -o bin/server ./cmd/server

# Or use the Makefile
make build
```

## Step 5: Run the Server

```bash
# Run the server
./bin/server

# Or directly with go run
go run cmd/server/main.go
```

You should see:
```
🚀 Starting AI Knowledge Management System...
✓ Database connected and migrated successfully
✓ OpenAI client initialized
🔧 Starting embedding worker pool with 5 workers
  Worker 0 started
  Worker 1 started
  ...
✓ Embedding worker pool started
🌐 Server listening on http://localhost:8080
```

## Step 6: Test the API

### Create a Document

```bash
curl -X POST http://localhost:8080/api/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Note",
    "content": "This is a test document for the AI KMS system. It will automatically generate embeddings using OpenAI.",
    "format": "markdown"
  }'
```

Response:
```json
{
  "id": "2BkYU9f3T6xYh8rQ1pKJVW4vZ9L",
  "title": "My First Note",
  "content": "This is a test document...",
  "format": "markdown",
  "metadata": {},
  "created_at": "2025-11-23T12:00:00Z",
  "updated_at": "2025-11-23T12:00:00Z"
}
```

The document will be automatically submitted to the embedding worker pool! Check the logs:
```
Worker 2 processing document 2BkYU9f3T6xYh8rQ1pKJVW4vZ9L
Worker 2 completed document 2BkYU9f3T6xYh8rQ1pKJVW4vZ9L
Generated 1 embeddings for document 2BkYU9f3T6xYh8rQ1pKJVW4vZ9L
```

### List Documents

```bash
curl http://localhost:8080/api/documents?limit=10&offset=0
```

### Get a Document

```bash
curl http://localhost:8080/api/documents/2BkYU9f3T6xYh8rQ1pKJVW4vZ9L
```

### Update a Document

```bash
curl -X PUT http://localhost:8080/api/documents/2BkYU9f3T6xYh8rQ1pKJVW4vZ9L \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Updated content - embeddings will regenerate automatically!"
  }'
```

### Soft Delete

```bash
curl -X DELETE http://localhost:8080/api/documents/2BkYU9f3T6xYh8rQ1pKJVW4vZ9L
```

### Hard Delete (Permanent)

```bash
curl -X DELETE "http://localhost:8080/api/documents/2BkYU9f3T6xYh8rQ1pKJVW4vZ9L?hard=true"
```

## What's Working Now (Phase 1 Complete ✅)

1. **CRUD Operations**: Create, Read, Update, Delete documents
2. **KSUID IDs**: Time-ordered, database-friendly identifiers
3. **Soft Deletes**: Documents are marked as deleted, not removed
4. **GORM**: Clean ORM abstraction over PostgreSQL
5. **Embedding Worker Pool**: Concurrent embedding generation using goroutines
6. **Auto-Embedding**: Documents automatically get embeddings on create/update
7. **pgvector Integration**: Embeddings stored as vectors in PostgreSQL

## Next Steps (Phase 2+)

- Real-time collaboration with Yjs/WebSocket
- Semantic search with vector similarity
- Knowledge graph with bi-directional links
- AI chat and summarization

## Troubleshooting

### Database Connection Error
```
❌ Failed to connect to database
```
**Solution**: Ensure PostgreSQL is running and credentials in `.env` are correct

### pgvector Extension Not Found
```
failed to enable pgvector extension
```
**Solution**: Install pgvector extension (see Step 1)

### OpenAI API Error
```
OPENAI_API_KEY is required
```
**Solution**: Set your OpenAI API key in `.env`

### Worker Pool Not Processing
Check the logs for:
- Worker startup messages
- Queue length (should be > 0 after creating documents)
- OpenAI API errors (rate limits, invalid key)

## Learning Resources

- **KSUID vs UUID**: See `docs/KSUID_VS_UUID.md`
- **Worker Pool Pattern**: See comments in `internal/services/embedding.go`
- **GORM Guide**: https://gorm.io/docs/
- **pgvector**: https://github.com/pgvector/pgvector

## Development

```bash
# Run with auto-reload (install air first: go install github.com/air-verse/air@latest)
air

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run
```

Enjoy building your AI Knowledge Management System! 🚀
