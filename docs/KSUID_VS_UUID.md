# KSUID vs UUID - Why We Use KSUID

## Overview

This project uses **KSUID (K-Sortable Unique Identifier)** instead of UUID for all primary keys.

## KSUID Structure

```
KSUID: 27 characters (base62 encoded)
Example: 2BkYU9f3T6xYh8rQ1pKJVW4vZ9L

Structure:
[Timestamp: 32 bits][Random Payload: 128 bits]
```

## Why KSUID is Better for Our Use Case

### 1. **Time-Ordered / Naturally Sortable** ⭐
- **KSUID**: First 32 bits are timestamp (seconds since epoch)
  - `ORDER BY id DESC` automatically sorts by creation time
  - No need for separate `created_at` index
- **UUID**: Random, no time information
  - Requires `created_at` column + index for time-based sorting

```go
// KSUID - Natural time ordering
db.Order("id DESC") // ✅ Sorted by creation time automatically

// UUID - Need created_at
db.Order("created_at DESC") // ❌ Requires additional index
```

### 2. **Better Database Performance** 🚀
- **KSUID**: Sequential IDs reduce B-tree index fragmentation
  - New entries append to end of index
  - Better cache locality
  - Faster inserts and queries
- **UUID v4**: Random IDs cause index fragmentation
  - New entries scattered throughout B-tree
  - More disk I/O
  - Slower performance over time

**Benchmark**: KSUIDs can be 10-30% faster for writes on large datasets

### 3. **Shorter Representation**
- **KSUID**: 27 characters (base62)
  - Example: `2BkYU9f3T6xYh8rQ1pKJVW4vZ9L`
- **UUID**: 36 characters with dashes
  - Example: `550e8400-e29b-41d4-a716-446655440000`

**Savings**: 25% shorter in text form

### 4. **Distributed-Safe** 🌐
- Both KSUID and UUID avoid collisions without central coordination
- KSUID combines timestamp + 128-bit random payload
- Safe for distributed systems, microservices
- No need for database sequences or auto-increment

### 5. **Debugging & Tracing** 🔍
- **KSUID**: Timestamp embedded in ID
  - Can extract creation time from ID alone
  - Easy to correlate events by ID ranges
- **UUID**: Completely opaque
  - No information encoded

```go
import "github.com/segmentio/ksuid"

id := ksuid.Parse("2BkYU9f3T6xYh8rQ1pKJVW4vZ9L")
fmt.Println(id.Time()) // Prints creation timestamp
```

### 6. **Lexicographically Sortable**
- KSUID sorts correctly as strings
- Useful for range queries, partition keys
- Works well with time-series data

## When UUID Might Be Better

1. **Need UUIDv7**: If you want UUID with time-ordering (newer spec)
2. **External Systems**: If integrating with systems that expect UUID format
3. **Strong Randomness**: UUIDv4 for security tokens (though KSUID is also cryptographically random)

## Our Implementation

```go
// models/document.go
type Document struct {
    ID string `gorm:"type:char(27);primaryKey"`
    // ... other fields
}

// Auto-generate on create
func (d *Document) BeforeCreate(tx *gorm.DB) error {
    if d.ID == "" {
        d.ID = ksuid.New().String()
    }
    return nil
}
```

### Database Schema

```sql
-- char(27) is perfect size for KSUID
CREATE TABLE documents (
    id CHAR(27) PRIMARY KEY,
    -- No need for created_at index!
    created_at TIMESTAMP,
    ...
);

-- Queries automatically sorted by creation time
SELECT * FROM documents ORDER BY id DESC LIMIT 10;
```

## References

- [KSUID Spec](https://github.com/segmentio/ksuid)
- [UUID vs KSUID Performance](https://blog.kowalczyk.info/article/JyRZ/generating-good-unique-ids-in-go.html)
- [PostgreSQL Index Performance](https://www.cybertec-postgresql.com/en/random-uuid-values-performance/)

## Summary

For a knowledge management system with frequent writes and time-based queries:
- ✅ Use **KSUID** for better performance and natural time-ordering
- ❌ Avoid **UUID v4** due to index fragmentation
- 🤔 Consider **UUID v7** as alternative (but KSUID is simpler)
