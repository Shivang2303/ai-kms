# Go Interface Best Practices - Consumer-Driven Design

## The Go Way: "Accept Interfaces, Return Structs"

This document explains why we moved interfaces to consumer packages.

---

## ❌ Wrong Approach (Java-style)

```go
// repository/interfaces.go
package repository

type DocumentRepository interface {
    Create(...) error
    GetByID(...) error
}

// repository/document_repo.go  
package repository

type DocumentRepositoryImpl struct { ... }
func (r *DocumentRepositoryImpl) Create(...) error { ... }

// services/embedding.go
package services

import "ai-kms/internal/repository"

type EmbeddingService struct {
    docRepo repository.DocumentRepository  // ❌ Import from repository package
}
```

**Problems:**
- Interface defined where it's implemented (provider-driven)
- Forces all consumers to use same interface
- Violates Interface Segregation Principle
- Creates tight coupling

---

## ✅ Correct Approach (Go idiom)

```go
// services/interfaces.go
package services

// Consumer (services) defines what IT needs
type DocumentRepository interface {
    Create(...) error
    GetByID(...) error
}

type EmbeddingService struct {
    docRepo DocumentRepository  // ✅ Interface from THIS package
}

// repository/document_repo.go
package repository

// Implementation doesn't know about interface
type DocumentRepositoryImpl struct { ... }

func NewDocumentRepository() *DocumentRepositoryImpl {  // ✅ Return concrete type
    return &DocumentRepositoryImpl{...}
}

// Automatically satisfies services.DocumentRepository!
func (r *DocumentRepositoryImpl) Create(...) error { ... }
func (r *DocumentRepositoryImpl) GetByID(...) error { ... }
```

**Benefits:**
- Consumer defines exactly what it needs
- No import of repository package for interfaces
- Implementation is decoupled
- Easy to create minimal mocks for testing

---

## Key Principles

### 1. Interfaces Belong to the Consumer

```
┌──────────────┐
│   api/       │  Defines: EmbeddingService interface
│   handlers   │  (only methods handlers use)
└──────┬───────┘
       │ uses
       ▼
┌──────────────┐
│  services/   │  Defines: DocumentRepository, EmbeddingRepository interfaces
│  embedding   │  (only methods services use)
└──────┬───────┘
       │ uses
       ▼
┌──────────────┐
│ repository/  │  NO interfaces here!
│ document_repo│  Just concrete implementations
└──────────────┘
```

### 2. Return Concrete Types, Accept Interfaces

```go
// ✅ GOOD
func NewDocumentRepository(db *gorm.DB) *DocumentRepositoryImpl {
    return &DocumentRepositoryImpl{db: db}
}

func NewEmbeddingService(docRepo DocumentRepository) *EmbeddingServiceImpl {
    //                            ^ Accept interface
    return &EmbeddingServiceImpl{docRepo: docRepo}
    //      ^ Return concrete type
}

// ❌ BAD
func NewDocumentRepository(db *gorm.DB) DocumentRepository {
    //                                   ^ Don't return interface
    return &DocumentRepositoryImpl{db: db}
}
```

### 3. Implicit Interface Satisfaction

```go
// No need for explicit declaration!
// If DocumentRepositoryImpl has these methods, it satisfies the interface

type DocumentRepository interface {
    Create(...)
    GetByID(...)
}

type DocumentRepositoryImpl struct { ... }

// These methods automatically satisfy the interface
func (r *DocumentRepositoryImpl) Create(...) { ... }
func (r *DocumentRepositoryImpl) GetByID(...) { ... }

// Compile-time check (optional but recommended):
var _ DocumentRepository = (*DocumentRepositoryImpl)(nil)
```

---

## Our Implementation

### File Structure

```
internal/
├── api/
│   ├── handlers.go
│   └── interfaces.go        ← EmbeddingService interface (handlers use it)
├── services/
│   ├── embedding.go
│   └── interfaces.go        ← Repository interfaces (services use them)
└── repository/
    ├── document_repo.go     ← Concrete implementation (no interface!)
    └── embedding_repo.go    ← Concrete implementation (no interface!)
```

### Example: Embedding Service

**services/interfaces.go** (Consumer defines interface):
```go
package services

type DocumentRepository interface {
    Create(ctx context.Context, doc *models.DocumentCreate) (*models.Document, error)
    GetByID(ctx context.Context, id string) (*models.Document, error)
}

type EmbeddingRepository interface {
    StoreEmbedding(ctx context.Context, embedding *models.Embedding) error
    SemanticSearch(ctx context.Context, queryEmbedding []float32, limit int) ([]*models.SearchResult, error)
}
```

**services/embedding.go** (Uses local interfaces):
```go
package services

type EmbeddingServiceImpl struct {
    docRepo DocumentRepository       // Interface from THIS package
    embRepo EmbeddingRepository      // Interface from THIS package
}

func NewEmbeddingService(
    docRepo DocumentRepository,      // Accept interface
    embRepo EmbeddingRepository,     // Accept interface
) *EmbeddingServiceImpl {            // Return concrete type
    return &EmbeddingServiceImpl{
        docRepo: docRepo,
        embRepo: embRepo,
    }
}
```

**repository/document_repo.go** (Doesn't know about interface):
```go
package repository

type DocumentRepositoryImpl struct {
    db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepositoryImpl {  // Return concrete
    return &DocumentRepositoryImpl{db: db}
}

// Methods automatically satisfy services.DocumentRepository interface
func (r *DocumentRepositoryImpl) Create(...) (*models.Document, error) { ... }
func (r *DocumentRepositoryImpl) GetByID(...) (*models.Document, error) { ... }
```

---

## Testing Benefits

### Before (Provider-Driven)
```go
// Had to import repository package just for interface
import "ai-kms/internal/repository"

type MockDocumentRepository struct {
    mock.Mock
}

func (m *MockDocumentRepository) Create(...) error { ... }
// Had to implement ALL methods from repository.DocumentRepository
// even if service only uses 2 of them!
```

### After (Consumer-Driven)
```go
// services/embedding_test.go
package services

// Define minimal mock in same package
type MockDocumentRepository struct {
    mock.Mock
}

// Only implement methods services.DocumentRepository needs
func (m *MockDocumentRepository) Create(...) error { ... }
func (m *MockDocumentRepository) GetByID(...) error { ... }

// That's it! No other methods required
```

---

## Common Pitfalls

### ❌ Pitfall 1: Returning Interfaces
```go
// DON'T
func NewService() Service {  // ❌
    return &ServiceImpl{...}
}

// DO
func NewService() *ServiceImpl {  // ✅
    return &ServiceImpl{...}
}
```

**Why?** Returning concrete types allows callers to access additional methods if needed, while still accepting as interface parameter.

### ❌ Pitfall 2: Big Interfaces
```go
// DON'T
type Repository interface {
    Create(...)
    Update(...)
    Delete(...)
    GetByID(...)
    List(...)
    Search(...)
    Paginate(...)
    // ... 20 more methods
}

// DO - Create small, focused interfaces
type Creator interface {
    Create(...)
}

type Getter interface {
    GetByID(...)
}
```

**Go Proverb**: "The bigger the interface, the weaker the abstraction"

### ❌ Pitfall 3: Premature Abstraction
```go
// DON'T create interfaces you don't need
type Logger interface {  // Only one implementation exists
    Log(string)
}

// DO - Just use the concrete type until you need abstraction
type Logger struct { ... }
```

**Rule**: Create interfaces when you have 2+ implementations OR for testing.

---

## References

- [Go Proverbs](https://go-proverbs.github.io/)
- [Effective Go: Interfaces](https://golang.org/doc/effective_go#interfaces)
- [Accept Interfaces, Return Structs](https://bryanftan.medium.com/accept-interfaces-return-structs-in-go-d4cab29a301b)
- [Interface Pollution in Go](https://medium.com/@cep21/preemptive-interface-anti-pattern-in-go-54c18ac0668a)

---

## Summary

✅ **DO:**
- Define interfaces in the consumer package
- Return concrete types from constructors
- Accept interfaces as parameters
- Keep interfaces small and focused

❌ **DON'T:**
- Define interfaces next to implementations
- Return interfaces from constructors
- Create interfaces without a consumer
- Make giant interfaces

**Result**: Loose coupling, easy testing, idiomatic Go code! 🎉
