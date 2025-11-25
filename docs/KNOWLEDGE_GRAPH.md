# Knowledge Graph Implementation

## Overview

The knowledge graph enables discovery and navigation of relationships between documents, similar to Wikipedia's [[wiki links]] or Obsidian's bidirectional links.

---

## Core Concepts

### 1. Nodes = Documents

Each document is a node in the graph.

### 2. Edges = Links

Links represent relationships between documents:
- **Outgoing links**: Documents this one references (`[[Other Doc]]`)
- **Incoming links** (Backlinks): Documents that reference this one

### 3. Bidirectional Navigation

```
Document A
  → Links to: [[Document B]], [[Document C]]
  ← Linked from: [[Document D]]

Document B
  → Links to: [[Document C]]
  ← Linked from: [[Document A]], [[Document D]]
```

---

## Database Schema

### Link Model

```go
type Link struct {
    ID       string   // KSUID
    SourceID string   // Document that contains the link
    TargetID string   // Document being linked to
    LinkType string   // "reference", "related", "parent", "child"
    
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time
}
```

**Indexes**:
- `source_id` - Fast lookup of outgoing links
- `target_id` - Fast lookup of backlinks

---

## API Endpoints

### 1. Get Knowledge Graph

**Endpoint**: `GET /api/graph`

**Description**: Get the entire knowledge graph structure

```bash
curl http://localhost:8080/api/graph
```

**Response**:
```json
{
  "links": [
    {
      "id": "2Bk...",
      "source_id": "2Bk...",
      "target_id": "2Bk...",
      "link_type": "reference",
      "source": {
        "id": "2Bk...",
        "title": "CRDT vs OT"
      },
      "target": {
        "id": "2Bk...",
        "title": "WebSocket Collaboration"
      }
    }
  ],
  "stats": {
    "total_documents": 15,
    "total_links": 42,
    "avg_degree": 5.6,
    "clusters": 2
  }
}
```

### 2. Generate Knowledge Graph

**Endpoint**: `POST /api/graph/generate`

**Description**: Extract [[links]] from all documents and create graph

```bash
curl -X POST http://localhost:8080/api/graph/generate
```

**How it works**:
1. Reads all documents
2. Parses content for `[[wiki-style]]` links
3. Finds matching documents by title
4. Creates Link records

**Response**:
```json
{
  "message": "Knowledge graph generated",
  "links_created": 23
}
```

### 3. Get Graph Node

**Endpoint**: `GET /api/graph/nodes/:id`

**Description**: Get graph information for a specific document

```bash
curl http://localhost:8080/api/graph/nodes/2BkYU9f...
```

**Response**:
```json
{
  "node": {
    "id": "2BkYU9f...",
    "title": "CRDT vs OT",
    "outgoing_links": 3,
    "incoming_links": 5,
    "connected_nodes": ["2Bk...", "2Bk...", ...]
  },
  "outgoing_links": [
    {
      "id": "2Bk...",
      "target_id": "2Bk...",
      "link_type": "reference",
      "target": {
        "title": "WebSocket Collaboration"
      }
    }
  ],
  "incoming_links": [
    {
      "id": "2Bk...",
      "source_id": "2Bk...",
      "link_type": "reference",
      "source": {
        "title": "RAG Implementation"
      }
    }
  ]
}
```

---

## Wiki-Style Link Syntax

### Creating Links

```markdown
# My Document

This document discusses [[CRDT vs OT]] and [[WebSocket Collaboration]].

The [[Knowledge Graph]] enables discovery.
```

### Link Extraction

```go
func ExtractLinksFromContent(content string) []string {
    // Finds all [[text]] patterns
    // Returns: ["CRDT vs OT", "WebSocket Collaboration", "Knowledge Graph"]
}
```

---

## Graph Metrics

### Degree

**Degree** = Total connections (incoming + outgoing)

```
Document with 3 outgoing + 2 incoming = degree 5
```

**Average Degree** = Total links × 2 / Total documents

### Clusters

**Cluster** = Connected component in the graph

```
Cluster 1: Doc A ← → Doc B ← → Doc C
Cluster 2: Doc D ← → Doc E
```

If clusters > 1, the knowledge base has disconnected sections.

---

## Use Cases

### 1. Backlinks Panel

Show all documents that reference the current document:

```javascript
GET /api/graph/nodes/:id

// Display incoming_links
"Linked from:"
- Document A
- Document B
- Document C
```

### 2. Graph Visualization

```javascript
GET /api/graph

// Render with D3.js or similar
nodes.forEach(node => {
    drawCircle(node.id, node.title)
})

links.forEach(link => {
    drawLine(link.source_id, link.target_id)
})
```

### 3. Related Documents

Find documents one hop away:

```javascript
GET /api/graph/nodes/:id

// Show connected_nodes as suggestions
"Related:"
- Documents this links to
- Documents that link here
```

### 4. Orphaned Documents

Find documents with no connections:

```sql
SELECT d.* 
FROM documents d
LEFT JOIN links l ON (d.id = l.source_id OR d.id = l.target_id)
WHERE l.id IS NULL
```

---

## Implementation Details

### Repository Layer

**File**: `internal/repository/link_repo.go`

**Key Methods**:
- `CreateLink(sourceID, targetID, linkType)` - Add connection
- `GetOutgoingLinks(sourceID)` - Get links FROM document
- `GetIncomingLinks(targetID)` - Get links TO document (backlinks)
- `GetGraphNode(documentID)` - Get all graph info for document
- `GetGraphStats()` - Get overall statistics

### Database Queries

**Outgoing links**:
```sql
SELECT * FROM links 
WHERE source_id = $1 
AND deleted_at IS NULL
```

**Incoming links** (backlinks):
```sql
SELECT * FROM links 
WHERE target_id = $1 
AND deleted_at IS NULL
```

**Graph stats**:
```sql
SELECT 
    COUNT(DISTINCT d.id) as total_documents,
    COUNT(l.id) as total_links
FROM documents d
LEFT JOIN links l ON (d.id = l.source_id OR d.id = l.target_id)
WHERE d.deleted_at IS NULL
```

---

## Advanced Features

### 1. Link Types

Different relationships:
- `reference` - General citation
- `related` - Similar topic
- `parent` - Hierarchical (broader topic)
- `child` - Hierarchical (narrower topic)

```go
linkRepo.CreateLink(docA, docB, "parent")
linkRepo.CreateLink(docB, docC, "reference")
```

### 2. Link Strength

Track how strongly documents are related:

```go
type Link struct {
    //...
    Strength float64 // 0.0 to 1.0
}

// Based on:
// - Number of mentions
// - Semantic similarity
// - User interactions
```

### 3. Path Finding

Find connection between two documents:

```go
func FindPath(startID, endID string) []string {
    // BFS/DFS graph traversal
    // Returns: [docA, docB, docC, ...]
}
```

### 4. Community Detection

Group related documents:

```go
func DetectCommunities() [][]string {
    // Louvain algorithm or similar
    // Returns clusters of related docs
}
```

---

## Performance Optimization

### 1. Materialized Views

Pre-compute expensive queries:

```sql
CREATE MATERIALIZED VIEW document_stats AS
SELECT 
    d.id,
    COUNT(DISTINCT l_out.id) as outgoing,
    COUNT(DISTINCT l_in.id) as incoming
FROM documents d
LEFT JOIN links l_out ON d.id = l_out.source_id
LEFT JOIN links l_in ON d.id = l_in.target_id
GROUP BY d.id;

REFRESH MATERIALIZED VIEW document_stats;
```

### 2. Caching

```go
type GraphCache struct {
    nodeCache map[string]*GraphNode
    mu        sync.RWMutex
}

func (c *GraphCache) GetNode(id string) *GraphNode {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.nodeCache[id]
}
```

### 3. Incremental Updates

Don't regenerate entire graph - update only changed documents:

```go
func UpdateDocumentLinks(docID string) {
    // 1. Delete old links from this document
    db.Where("source_id = ?", docID).Delete(&Link{})
    
    // 2. Extract new links
    links := ExtractLinksFromContent(doc.Content)
    
    // 3. Create new link records
    for _, link := range links {
        CreateLink(docID, findDocByTitle(link), "reference")
    }
}
```

---

## Visualization Ideas

### 1. Force-Directed Graph

```javascript
// Using D3.js
const simulation = d3.forceSimulation(nodes)
    .force("link", d3.forceLink(links))
    .force("charge", d3.forceManyBody())
    .force("center", d3.forceCenter(width / 2, height / 2));
```

### 2. Hierarchical Tree

```javascript
// Using D3 tree layout
const tree = d3.tree()
    .size([height, width]);

const root = d3.hierarchy(data);
tree(root);
```

### 3. Circular Layout

```javascript
// Arrange nodes in a circle
nodes.forEach((node, i) => {
    const angle = (i / nodes.length) * 2 * Math.PI;
    node.x = centerX + radius * Math.cos(angle);
    node.y = centerY + radius * Math.sin(angle);
});
```

---

## Testing

```bash
# 1. Create documents
curl -X POST http://localhost:8080/api/documents \
  -d '{"title":"Doc A","content":"References [[Doc B]]"}'

curl -X POST http://localhost:8080/api/documents \
  -d '{"title":"Doc B","content":"Related to [[Doc C]]"}'

curl -X POST http://localhost:8080/api/documents \
  -d '{"title":"Doc C","content":"Mentions [[Doc A]]"}'

# 2. Generate graph
curl -X POST http://localhost:8080/api/graph/generate

# 3. View graph
curl http://localhost:8080/api/graph

# 4. View specific node
curl http://localhost:8080/api/graph/nodes/<doc_a_id>
```

---

## Summary

**Knowledge Graph = Network of Connected Knowledge**

**Key Features**:
- ✅ Bidirectional link tracking
- ✅ Wiki-style [[link]] syntax
- ✅ Graph statistics and analytics
- ✅ Backlinks for each document
- ✅ Graph visualization support

**Benefits**:
- Discover related content
- Navigate knowledge network
- Find connections between ideas
- Identify knowledge gaps
- Build second brain

**Next Steps**: Frontend graph visualization with D3.js! 📊
