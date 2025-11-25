# Jaeger - Distributed Tracing Guide

## What is Jaeger?

**Jaeger** is an open-source distributed tracing system originally developed by Uber. It helps you monitor and troubleshoot microservices-based distributed systems.

### Key Concepts

- **Trace**: Complete journey of a request through your system
- **Span**: Individual operation within a trace
- **Context Propagation**: Passing trace information between services

---

## Architecture

```
┌─────────────┐
│  Your App   │
│  (AI-KMS)   │
└──────┬──────┘
       │ Sends spans
       ▼
┌─────────────────┐
│ OpenTelemetry   │ (Instrumentation)
│ SDK             │
└──────┬──────────┘
       │ Exports
       ▼
┌─────────────────┐
│ Jaeger Exporter │
└──────┬──────────┘
       │ HTTP/gRPC
       ▼
┌─────────────────┐
│ Jaeger Collector│
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│ Jaeger Storage  │ (In-Memory/Cassandra/Elasticsearch)
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│ Jaeger UI       │ http://localhost:16686
└─────────────────┘
```

---

## Running Jaeger Locally

### Option 1: Docker All-in-One (Easiest)

```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14250:14250 \
  -p 14268:14268 \
  -p 14269:14269 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

**Ports:**
- **16686**: Jaeger UI (web interface)
- **14268**: HTTP collector endpoint (we use this)
- **6831/6832**: UDP agent (for higher throughput)

### Option 2: Kubernetes/Production

For production, run separate components:
- Collector (receives traces)
- Query service (serves UI)
- Storage backend (Cassandra/Elasticsearch)

---

## Our Integration

### Configuration

```bash
# .env
JAEGER_ENDPOINT=http://localhost:14268/api/traces
```

### Initialization

```go
// cmd/server/main.go
jaegerShutdown, err := telemetry.InitJaeger("ai-kms", cfg.JaegerEndpoint)
if err != nil {
    log.Printf("⚠️  Failed to initialize Jaeger: %v", err)
}
defer jaegerShutdown(context.Background())
```

### Middleware Creates Root Spans

```go
// internal/middleware/tracing.go
func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Create root span for entire HTTP request
        ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
        defer span.End()
        
        // Add attributes (searchable in Jaeger UI)
        span.SetAttributes(
            attribute.String("http.method", r.Method),
            attribute.String("http.url", r.URL.Path),
            attribute.String("request.id", requestID),
        )
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## Using Jaeger UI

### 1. Open UI
```
http://localhost:16686
```

### 2. Search for Traces

**By Service**: Select "ai-kms"

**By Operation**: Select HTTP endpoints like "POST /api/documents"

**By Tags**: Search by `request.id`, `http.status_code`, etc.

### 3. Viewing a Trace

```
Trace Timeline:
┌─────────────────────────────────────────────┐
│ POST /api/documents (200ms)                 │ ← Root span
│   ├─ Create Document (50ms)                 │ ← Child span
│   ├─ Store Embedding (100ms)                │ ← Child span
│   │   ├─ OpenAI API Call (80ms)             │
│   │   └─ DB Insert (20ms)                   │
│   └─ Send Response (10ms)                   │
└─────────────────────────────────────────────┘
Total: 200ms
```

**What You See:**
- Time spent in each operation
- Parent-child relationships
- Errors (highlighted in red)
- Attributes/tags

### 4. Finding Errors

Search for:
- `error=true` tag
- HTTP status codes `>= 400`
- Operations with `otel.status_code=ERROR`

**Example Error Trace:**
```
GET /api/documents/invalid-id
  → Status: 404
  → Error: "document not found: invalid-id"
  → Stack trace in span details
```

---

## Creating Spans in Your Code

### Basic Span

```go
func (s *Service) DoWork(ctx context.Context) error {
    // Import middleware helper
    ctx, span := middleware.StartSpan(ctx, "Service.DoWork")
    defer span.End()
    
    // Your code...
    return nil
}
```

### Adding Attributes

```go
span := trace.SpanFromContext(ctx)
span.SetAttributes(
    attribute.String("document.id", docID),
    attribute.Int("chunk.count", len(chunks)),
)
```

### Recording Errors

```go
if err != nil {
    middleware.AddSpanError(ctx, err)  // Marks span as error
    return err
}
```

### Adding Events

```go
middleware.AddSpanEvent(ctx, "embedding_generated",
    attribute.Int("vector.dimensions", 1536),
)
```

---

## Example Trace Flow

### Request: Create Document
```
POST /api/documents
{
  "title": "My Note",
  "content": "AI is awesome"
}
```

### Jaeger Trace:
```
[Root] POST /api/documents (250ms)
  ├─ [Handler] CreateDocument (245ms)
  │   ├─ [Repo] Create Document (45ms)
  │   │   └─ [DB] INSERT documents (40ms)
  │   └─ [Service] Submit Embedding Job (5ms)
  │       └─ [Channel] Enqueue Job (1ms)
  └─ [Middleware] Write Response (5ms)

[Async] Worker Process Document (5000ms)
  ├─ [Service] Chunk Text (50ms)
  ├─ [OpenAI] Generate Embedding (4800ms)  ← Bottleneck!
  └─ [Repo] Store Embedding (150ms)
```

**Insight**: OpenAI API takes 4.8s - Maybe batch requests?

---

## Production Best Practices

### 1. Sampling

Don't trace every request in production:

```go
// Trace 10% of requests
sdktrace.WithSampler(
    sdktrace.ParentBased(
        sdktrace.TraceIDRatioBased(0.1),
    ),
)
```

### 2. Attribute Limits

Don't add huge attributes:
```go
// ❌ BAD - Giant document content
span.SetAttribute("document.content", doc.Content)

// ✅ GOOD - Just length
span.SetAttribute("document.content_length", len(doc.Content))
```

### 3. Span Naming

Use consistent, searchable names:
```go
// ✅ GOOD
"HTTP POST /api/documents"
"DB INSERT documents"
"OpenAI CreateEmbedding"

// ❌ BAD
"handle_request"
"database_stuff"
```

### 4. Storage

In-memory storage (default) is only for development.

Production options:
- **Cassandra**: High write throughput
- **Elasticsearch**: Rich querying
- **S3/GCS**: Long-term archival

---

## Troubleshooting

### Traces Not Appearing?

1. **Check Jaeger is running:**
   ```bash
   curl http://localhost:16686
   ```

2. **Check endpoint:**
   ```bash
   # In your app logs:
   ✓ Jaeger tracing initialized: http://localhost:14268/api/traces
   ```

3. **Check for errors:**
   ```go
   jaegerShutdown, err := telemetry.InitJaeger(...)
   if err != nil {
       log.Printf("Error: %v", err)
   }
   ```

### Missing Child Spans?

**Problem**: Context not propagated

```go
// ❌ BAD - New context loses trace info
newCtx := context.Background()
s.doWork(newCtx)

// ✅ GOOD - Pass context through
s.doWork(ctx)
```

---

## Comparison: Jaeger vs Alternatives

| Feature | Jaeger | Zipkin | Datadog | New Relic |
|---------|--------|--------|---------|-----------|
| **Cost** | Free | Free | Paid | Paid |
| **Setup** | Easy | Easy | SaaS | SaaS |
| **Sampling** | ✅ | ✅ | ✅ | ✅ |
| **Storage** | In-mem/Cassandra/ES | In-mem/MySQL/Cassandra | Proprietary | Proprietary |
| **OpenTelemetry** | ✅ Native | ✅ Compatible | ✅ Compatible | ✅ Compatible |

**Our Choice**: Jaeger because:
- Free and open-source
- Easy local development
- Industry standard
- Works with OpenTelemetry (vendor-neutral)

---

## Next Steps

1. **View your first trace:**
   - Start server: `./bin/server`
   - Create document: `POST /api/documents`
   - Open Jaeger UI: http://localhost:16686

2. **Add custom spans:**
   - In your services
   - In repository methods
   - In worker pool

3. **Production deployment:**
   - Set up persistent storage
   - Configure sampling rate
   - Add alerting on errors

---

## References

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Distributed Tracing Best Practices](https://opentelemetry.io/docs/concepts/sampling/)

---

## Summary

**Jaeger** is your window into the system's behavior:
- See exactly where time is spent
- Debug errors with full context
- Understand complex request flows
- Optimize performance bottlenecks

**Start Jaeger:**
```bash
docker run -d --name jaeger -p 16686:16686 -p 14268:14268 jaegertracing/all-in-one:latest
```

**View UI:**
```
http://localhost:16686
```

Happy tracing! 🔍
