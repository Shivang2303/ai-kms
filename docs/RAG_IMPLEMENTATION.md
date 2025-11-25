# RAG (Retrieval Augmented Generation) - Implementation Guide

## What is RAG?

**RAG** combines the power of retrieval systems (semantic search) with large language models to provide accurate, grounded answers using your own knowledge base.

```
 User Question
      │
      ▼
 ┌─────────────┐
 │   Embed     │  Convert question to vector
 │   Query     │
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │  Semantic   │  Find relevant documents
 │   Search    │  (cosine similarity)
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │   Build     │  Create prompt with context
 │   Prompt    │
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │     LLM     │  Generate answer
 │  (ChatGPT)  │
 └──────┬──────┘
        │
        ▼
    Answer + Sources
```

---

## Why RAG?

### Problem: LLM Limitations

**Pure LLM** (without RAG):
- ❌ Hallucinates facts
- ❌ Doesn't know your data
- ❌ Can't cite sources
- ❌ Knowledge cutoff date

**Example**:
```
User: "What did our Q3 report say about revenue?"
LLM: "I don't have access to your Q3 report."
```

### Solution: RAG

**With RAG**:
- ✅ Grounded in real documents
- ✅ Uses your knowledge base
- ✅ Can cite exact sources
- ✅ Always up-to-date

**Example**:
```
User: "What did our Q3 report say about revenue?"
RAG: "According to Q3_Report.pdf, revenue increased 
      by 23% to $4.2M, driven by enterprise sales."
Source: [Document 1] Q3 Financial Report
```

---

## Our Implementation

### 1. RAG Service

**File**: `internal/services/rag_service.go`

```go
type RAGService struct {
    openaiClient *openai.Client
    embRepo      EmbeddingRepository
}

func (s *RAGService) QueryWithContext(
    ctx context.Context,
    query string,
    maxChunks int,
) (string, []*models.SearchResult, error) {
    // 1. Generate embedding for query
    queryEmbedding, err := s.openaiClient.CreateEmbeddings([]string{query})
    
    // 2. Semantic search
    results, err := s.embRepo.SemanticSearch(ctx, queryEmbedding, maxChunks)
    
    // 3. Build context
    context := buildContext(results)
    
    // 4. Build prompt
    prompt := buildRAGPrompt(query, context)
    
    // 5. Get answer from LLM
    answer, err := s.openaiClient.ChatCompletion(ctx, messages)
    
    return answer, results, nil
}
```

---

## API Endpoints

### 1. Semantic Search

**Endpoint**: `POST /api/search`

**Description**: Find documents similar to a query

```bash
curl -X POST http://localhost:8080/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "machine learning best practices",
    "limit": 5
  }'
```

**Response**:
```json
{
  "query": "machine learning best practices",
  "results": [
    {
      "document_id": "2Bk...",
      "title": "ML Engineering Guide",
      "chunk_text": "The key to successful ML projects...",
      "similarity": 0.92
    }
  ],
  "count": 5
}
```

### 2. RAG Query

**Endpoint**: `POST /api/ai/query`

**Description**: Ask a question and get an AI answer with sources

```bash
curl -X POST http://localhost:8080/api/ai/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "How do I deploy the application?",
    "max_chunks": 5
  }'
```

**Response**:
```json
{
  "query": "How do I deploy the application?",
  "answer": "Based on the documentation, you can deploy using Docker Compose locally or Kubernetes for production. The Dockerfile is multi-stage optimized...",
  "sources": [
    {
      "document_id": "2Bk...",
      "title": "Deployment Guide",
      "chunk_text": "To deploy with Docker Compose...",
      "similarity": 0.95
    }
  ]
}
```

### 3. Document Summarization

**Endpoint**: `POST /api/ai/summarize/:id`

**Description**: Generate a concise summary of a document

```bash
curl -X POST http://localhost:8080/api/ai/summarize/2BkYU9f... \
  -H "Content-Type: application/json" \
  -d '{
    "max_words": 150
  }'
```

**Response**:
```json
{
  "document_id": "2BkYU9f...",
  "summary": "This document outlines the deployment strategy for AI-KMS. It covers Docker containerization with  multi-stage builds, Kubernetes manifests for production, and infrastructure setup with PostgreSQL and Jaeger.",
  "max_words": 150
}
```

### 4. Query Specific Document

**Endpoint**: `POST /api/ai/query/:id`

**Description**: Ask questions about a specific document

```bash
curl -X POST http://localhost:8080/api/ai/query/2BkYU9f... \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What are the main prerequisites?"
  }'
```

**Response**:
```json
{
  "document_id": "2BkYU9f...",
  "query": "What are the main prerequisites?",
  "answer": "The main prerequisites are: 1) Go 1.21+, 2) PostgreSQL with pgvector, 3) OpenAI API key"
}
```

---

## How RAG Works Internally

### Step 1: Embed the Query

```
Query: "How do CRDTs work?"
   ↓
OpenAI Embeddings API
   ↓
Vector: [0.12, -0.34, 0.56, ..., 0.23]  (1536 dimensions)
```

### Step 2: Semantic Search

```sql
SELECT 
    d.id,
    d.title,
    e.chunk_text,
    1 - (e.embedding <=> $1::vector) as similarity
FROM embeddings e
JOIN documents d ON d.id = e.document_id
WHERE d.deleted_at IS NULL
ORDER BY e.embedding <=> $1::vector
LIMIT 5;
```

**Cosine Similarity** finds most relevant chunks:
- Query: "How do CRDTs work?"
- Matches: Documents about CRDTs, conflict resolution, Yjs

### Step 3: Build Context

```
Context:
[Document 1] CRDT_VS_OT.md:
CRDTs ensure convergence through commutativity...

[Document 2] WebSocket Collaboration:
Our system uses Yjs, a CRDT library...

[Document 3] Real-time Sync:
Sessions are managed with awareness state...
```

### Step 4: Build Prompt

```
Based on the following context, please answer the question:

Context:
[Document 1] CRDT_VS_OT.md:
CRDTs ensure convergence through commutativity...

[Document 2] WebSocket Collaboration:
Our system uses Yjs, a CRDT library...

Question: How do CRDTs work?

Answer:
```

### Step 5: LLM Generation

```
GPT-3.5-turbo processes the prompt and generates:

"CRDTs (Conflict-free Replicated Data Types) work by ensuring 
that operations are commutative - they can be applied in any 
order and produce the same result. According to Document 1, 
this is achieved through mathematical properties like 
commutativity and idempotence. Document 2 mentions that our 
system uses Yjs, which implements a list CRDT based on the 
RGA algorithm..."
```

---

## Code Example: Using RAG

```go
// Initialize RAG service
ragService := services.NewRAGService(openaiClient, embRepo)

// Query with context
answer, sources, err := ragService.QueryWithContext(
    ctx,
    "How do I add a new feature?",
    5, // max context chunks
)

// Answer includes:
// - Generated response from LLM
// - Source documents used
// - Similarity scores

fmt.Println("Answer:", answer)
for _, source := range sources {
    fmt.Printf("Source: %s (%.2f similarity)\n", 
        source.Title, source.Similarity)
}
```

---

## Advantages

### 1. Accuracy

**Without RAG**:
```
Q: "What's our embedding model?"
A: "I don't have access to your codebase."
```

**With RAG**:
```
Q: "What's our embedding model?"
A: "According to the code, you're using text-embedding-ada-002 
    from OpenAI. [Source: internal/openai/client.go]"
```

### 2. Reduced Hallucination

LLM can only answer using provided context:
- If context doesn't have the answer → "I don't have enough information"
- If context has the answer → Cite specific source

### 3. Cost Efficiency

- Don't need to fine-tune expensive models
- Use smaller, cheaper models (GPT-3.5 instead of GPT-4)
- Only send relevant context (not entire documents)

### 4. Privacy

- Your data stays in your database
- Only relevant chunks sent to API
- Can self-host the LLM for full privacy

---

## Performance Optimization

### 1. Caching

```go
// Cache embeddings
type EmbeddingCache struct {
    cache map[string][]float32
    mu    sync.RWMutex
}

func (c *EmbeddingCache) GetOrCreate(text string) []float32 {
    c.mu.RLock()
    if embedding, ok := c.cache[text]; ok {
        c.mu.RUnlock()
        return embedding
    }
    c.mu.RUnlock()
    
    // Generate and cache
    embedding := createEmbedding(text)
    c.mu.Lock()
    c.cache[text] = embedding
    c.mu.Unlock()
    return embedding
}
```

### 2. Chunk Size Optimization

```
Too small (100 chars):  Lost context, too many chunks
Too large (5000 chars): Irrelevant info, expensive
Optimal (500-1000):     Good context, efficient
```

### 3. Reranking

```go
// Step 1: Fast semantic search (get 20 results)
initialResults := semanticSearch(query, 20)

// Step 2: Rerank with cross-encoder (more accurate)
rankedResults := rerank(query, initialResults)

// Step 3: Use top 5 for context
topResults := rankedResults[:5]
```

---

## Advanced Features

### 1. Hybrid Search

Combine **semantic** + **keyword** search:

```sql
SELECT *,
    (0.7 * semantic_score + 0.3 * keyword_score) AS hybrid_score
FROM (
    SELECT 
        id,
        1 - (embedding <=> $1::vector) AS semantic_score,
        ts_rank(content_tsvector, to_tsquery($2)) AS keyword_score
    FROM documents
) scores
ORDER BY hybrid_score DESC
LIMIT 10;
```

### 2. Conversation Memory

```go
type ConversationRAG struct {
    ragService *RAGService
    history    []ChatMessage
}

func (c *ConversationRAG) Query(query string) string {
    // Include conversation history in prompt
    messages := append(c.history, ChatMessage{
        Role:    "user",
        Content: query,
    })
    
    answer := c.ragService.QueryWithContext(query, 5)
    
    // Update history
    c.history = append(c.history, 
        ChatMessage{Role: "user", Content: query},
        ChatMessage{Role: "assistant", Content: answer},
    )
    
    return answer
}
```

### 3. Source Citation

```go
func buildRAGPrompt(query, context string) string {
    return fmt.Sprintf(`Based on the context below, answer the question. 
IMPORTANT: Cite your sources using [Document N] notation.

Context:
%s

Question: %s

Answer (with citations):`, context, query)
}
```

---

## Limitations

### 1. Context Window

- GPT-3.5: 4096 tokens (~3000 words)
- GPT-4: 8192 tokens (~6000 words)
- Solution: Summarize or chunk context

### 2. Cost

```
Per query:
- Embedding: $0.0001 (text-embedding-ada-002)
- Completion: $0.002 (gpt-3.5-turbo, 1000 tokens)

1000 queries/day = ~$2-3/day
```

### 3. Latency

```
Total time: ~2-4 seconds
- Embedding: 200-500ms
- Search: 50-100ms
- LLM: 1-3s (depends on answer length)
```

---

## Best Practices

### 1. Chunk Strategically

```go
// Don't split mid-sentence
chunks := smartChunk(content, 800, 100) // size, overlap

// Preserve context
for i, chunk := range chunks {
    chunk.Metadata = map[string]string{
        "source": documentTitle,
        "section": sectionName,
        "chunk_index": fmt.Sprintf("%d/%d", i+1, len(chunks)),
    }
}
```

### 2. Filter Results

```go
// Only use high-similarity results
relevantChunks := []SearchResult{}
for _, result := range results {
    if result.Similarity > 0.7 {  // Threshold
        relevantChunks = append(relevantChunks, result)
    }
}
```

### 3. Prompt Engineering

```
Bad prompt:
"Answer this: [question]"

Good prompt:
"Based on the context, answer the question. If the context 
doesn't contain enough information, say so. Always cite which 
document you're using."
```

---

## Testing RAG

```bash
# 1. Add documents
curl -X POST http://localhost:8080/api/documents \
  -d '{"title":"ML Guide","content":"Machine learning best practices..."}'

# 2. Wait for embedding (check logs)

# 3. Test semantic search
curl -X POST http://localhost:8080/api/search \
  -d '{"query":"machine learning","limit":3}'

# 4. Test RAG
curl -X POST http://localhost:8080/api/ai/query \
  -d '{"query":"What are ML best practices?"}'
```

---

## Summary

**RAG = Retrieval + Generation**

1. **Retrieve**: Find relevant context using semantic search
2. **Augment**: Add context to the prompt
3. **Generate**: Let LLM answer based on your data

**Result**: Accurate, grounded AI answers using your knowledge base! 🎯

**Next Steps**: Add conversation memory, hybrid search, and citation tracking.
