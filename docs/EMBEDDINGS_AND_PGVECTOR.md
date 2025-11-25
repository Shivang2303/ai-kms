# Embeddings & Vector Search - Technical Deep Dive

## What Are Embeddings?

**Embeddings** are dense vector representations of text that capture semantic meaning in a high-dimensional space.

### Simple Analogy
Think of embeddings as a way to convert text into coordinates in a multi-dimensional space where **similar meanings are close together**.

```
"cat" → [0.12, 0.45, -0.23, ... ] (1536 dimensions)
"dog" → [0.15, 0.48, -0.19, ... ] (close to "cat")
"car" → [0.89, -0.62, 0.34, ... ] (far from "cat")
```

---

## How Embeddings Work

### 1. Input Text
```
"Artificial intelligence is transforming healthcare"
```

### 2. Tokenization
```
["Artificial", "intelligence", "is", "transforming", "healthcare"]
```

### 3. Neural Network (OpenAI's text-embedding-ada-002)
```
Input tokens → Transformer layers → Vector output (1536 dimensions)
```

### 4. Output Embedding
```
[0.002, -0.015, 0.031, ..., -0.008] (1536 numbers)
```

---

## Why Embeddings are Powerful

### Traditional Keyword Search (Lexical)
```
Query: "AI in medicine"
Documents:
✗ "Machine learning transforms healthcare" - ❌ No match (different words!)
✓ "AI in medicine research" - ✅ Match (exact keywords)
```

### Semantic Search with Embeddings
```
Query embedding: AI in medicine → [0.12, 0.45, ...]
Documents:
✓ "Machine learning transforms healthcare" → [0.13, 0.46, ...] - ✅ Close! (similar meaning)
✓ "AI in medicine research" → [0.14, 0.47, ...] - ✅ Very close!
```

**Key Insight**: Embeddings capture **meaning**, not just words.

---

## pgvector: Vector Search in PostgreSQL

**pgvector** is a PostgreSQL extension that enables:
1. Storing vector embeddings efficiently
2. Performing similarity search using specialized indexes
3. Scaling to millions of vectors

### Architecture

```
┌─────────────────┐
│  Document Text  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  OpenAI API     │ Generate embedding
│  ada-002 model  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   float32[1536] │ Vector embedding
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   pgvector      │ Store in PostgreSQL
│   VECTOR(1536)  │
└─────────────────┘
```

### SQL Examples

#### Creating a Table with Vectors
```sql
CREATE TABLE embeddings (
    id CHAR(27) PRIMARY KEY,
    document_id CHAR(27),
    chunk_text TEXT,
    embedding vector(1536)  -- pgvector type!
);
```

#### Inserting Vectors
```sql
INSERT INTO embeddings (document_id, chunk_text, embedding)
VALUES (
    '2BkYU9f3T6xYh8rQ1pKJVW4vZ9L',
    'AI is transforming healthcare',
    '[0.002, -0.015, ..., -0.008]'  -- Vector as array
);
```

#### Creating an Index for Fast Search
```sql
-- IVFFlat index: partitions vector space for faster search
CREATE INDEX ON embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
```

### IVFFlat Index Explained

**IVF** = Inverted File with Flat compression

How it works:
1. **Clustering**: Divide vectors into ~100 clusters (lists)
2. **Search**: Only search closest clusters, not all vectors
3. **Trade-off**: ~95% accuracy, 10-100x faster

```
All Vectors (1M)
    ↓
Cluster into 100 groups
    ↓
Query vector → Find 3 closest clusters → Search only those
    ↓
Return top K results (~10)
```

---

## Similarity Metrics

### Cosine Similarity (We Use This!)

Measures the **angle** between two vectors (ignores magnitude).

```
Formula: similarity = (A · B) / (||A|| × ||B||)

Range: -1 (opposite) to +1 (identical)

Example:
A = [1, 2, 3]
B = [2, 4, 6]  (scaled version of A)
Cosine similarity = 1.0 (same direction!)
```

**pgvector operator**: `<=>` (cosine distance = 1 - cosine similarity)

```sql
-- Find most similar vectors (lowest distance)
SELECT * FROM embeddings
ORDER BY embedding <=> '[0.12, 0.45, ...]'
LIMIT 10;
```

### Other Metrics

| Metric | Operator | Use Case |
|--------|----------|----------|
| Cosine Distance | `<=>` | Text, direction matters |
| L2 Distance (Euclidean) | `<->` | Spatial data, magnitude matters |
| Inner Product | `<#>` | When vectors pre-normalized |

---

## Our Implementation

### 1. Text Chunking

Large documents are split into chunks:

```go
// services/embedding.go
func (s *EmbeddingServiceImpl) chunkText(text string, maxWords int) []string {
    words := strings.Fields(text)
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
```

**Why chunk?**
- OpenAI embedding models have token limits (~8191 tokens)
- Smaller chunks = more precise search results
- Can match specific paragraphs, not just whole documents

### 2. Embedding Generation

```go
// Call OpenAI API
embeddingVector, err := s.openaiClient.CreateEmbeddings([]string{chunk})

// Convert to pgvector format
vec := pgvector.NewVector(embeddingVector)

// Store in database
embedding := &models.Embedding{
    DocumentID: job.DocumentID,
    ChunkIndex: i,
    ChunkText:  chunk,
    Embedding:  vec,  // pgvector.Vector type
}
s.embRepo.StoreEmbedding(ctx, embedding)
```

### 3. Semantic Search

```go
// repository/embedding_repo.go
func (r *EmbeddingRepositoryImpl) SemanticSearch(
    ctx context.Context,
    queryEmbedding []float32,
    limit int,
) ([]*models.SearchResult, error) {
    vec := pgvector.NewVector(queryEmbedding)
    
    var results []*models.SearchResult
    
    // Use pgvector's cosine distance operator
    err := r.db.WithContext(ctx).Raw(`
        SELECT 
            e.document_id,
            d.title,
            e.chunk_text,
            1 - (e.embedding <=> ?) as score  -- Convert distance to similarity
        FROM embeddings e
        JOIN documents d ON d.id = e.document_id
        WHERE e.deleted_at IS NULL AND d.deleted_at IS NULL
        ORDER BY e.embedding <=> ?  -- Order by distance (ascending)
        LIMIT ?
    `, vec, vec, limit).Scan(&results).Error
    
    return results, err
}
```

**Key Points:**
- `<=>` returns **distance** (lower = more similar)
- `1 - distance` converts to **similarity score** (higher = more similar)
- Join with `documents` table to get titles
- Filter out soft-deleted records

---

## Performance Considerations

### Index Configuration

```sql
-- More lists = faster search but lower accuracy
CREATE INDEX ON embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);  -- Tune based on dataset size

-- Rule of thumb: lists = sqrt(total_rows)
-- 10K rows → lists = 100
-- 1M rows → lists = 1000
```

### Query Performance

| Vectors | Index | Search Time |
|---------|-------|-------------|
| 10K | None | ~100ms |
| 10K | IVFFlat | ~10ms |
| 1M | None | ~10s |
| 1M | IVFFlat | ~50ms |

### Storage Requirements

```
Vector size: 1536 dimensions × 4 bytes (float32) = 6KB per embedding

10K chunks × 6KB = 60MB
1M chunks × 6KB = 6GB
```

---

## Common Use Cases

### 1. Semantic Search (Our Implementation)
Find documents by meaning, not keywords.

```
Query: "how to debug memory leaks"
Results:
- "Troubleshooting memory issues in production"
- "Profiling heap allocations"
- "Detecting memory leaks with valgrind"
```

### 2. Document Similarity
Find related documents.

```
Document A → Embedding A
Find: Top 10 documents with closest embeddings
```

### 3. Recommendation Systems
```
User liked Document X → Embedding X
Recommend: Documents with similar embeddings
```

### 4. RAG (Retrieval Augmented Generation)
```
1. User question → Generate embedding
2. Find relevant chunks via semantic search
3. Feed chunks + question to LLM
4. LLM generates answer with context
```

---

## Limitations & Gotchas

### 1. Embeddings are NOT Perfect
- May group unrelated text if structurally similar
- Language/domain specific (train on technical docs for best results)
- OpenAI embeddings are black-box (can't explain why)

### 2. Index Accuracy
- IVFFlat is approximate (may miss some results)
- Use exact search for critical applications:
  ```sql
  SET ivfflat.probes = 10;  -- Search more clusters
  ```

### 3. Cold Start
- New documents need embeddings generated (async)
- Search only works after embedding generation completes

### 4. Cost
- OpenAI charges per token (~$0.0001 per 1000 tokens)
- 1M documents × 500 words avg = ~$50-100 for embeddings

---

## Best Practices

### 1. Chunk Strategically
```go
// Too small: Loses context
chunkSize := 50 words  // ❌

// Too large: Too generic
chunkSize := 5000 words  // ❌

// Just right: Paragraph-level
chunkSize := 200-500 words  // ✅
```

### 2. Use Hybrid Search
Combine semantic + keyword search for best results:

```sql
-- Semantic score
SELECT *, (1 - (embedding <=> ?)) AS semantic_score
FROM embeddings
WHERE chunk_text ILIKE '%keyword%'  -- Keyword filter
ORDER BY semantic_score DESC
LIMIT 10;
```

### 3. Monitor Performance
```sql
-- Check index usage
EXPLAIN ANALYZE
SELECT * FROM embeddings
ORDER BY embedding <=> '[...]'::vector
LIMIT 10;
```

### 4. Update Embeddings on Content Change
Our implementation does this automatically:

```go
// handlers.go - Update document
if update.Content != nil {
    job := services.EmbeddingJob{
        DocumentID: updated.ID,
        Content:    updated.Content,
    }
    h.embService.SubmitJob(job)  // Regenerate embeddings
}
```

---

## References

- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [IVFFlat Algorithm Paper](https://hal.inria.fr/inria-00514462/document)
- [Understanding Word Embeddings](https://arxiv.org/abs/1301.3781)

---

## Summary

**Embeddings** convert text to vectors that capture meaning.

**pgvector** enables efficient vector search in PostgreSQL.

**Our system**:
1. Chunks documents into smaller pieces
2. Generates embeddings via OpenAI
3. Stores vectors in pgvector
4. Enables semantic search with cosine similarity

This powers features like:
- Finding similar documents
- Semantic search (meaning-based, not keyword)
- RAG (feeding relevant context to LLMs)

**Next**: Implement semantic search endpoint and RAG chat! 🚀
