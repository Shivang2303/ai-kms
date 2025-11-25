# WebSocket Real-Time Collaboration

## Overview

This document explains the WebSocket-based real-time collaboration system for concurrent document editing.

---

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Client A   │     │  Client B   │     │  Client C   │
│ (Browser)   │     │ (Browser)   │     │ (Browser)   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ WebSocket         │ WebSocket         │ WebSocket
       └───────────────────┼───────────────────┘
                           │
                     ┌─────▼─────┐
                     │  Server   │
                     │ (Go)      │
                     └─────┬─────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
  ┌─────▼─────┐    ┌──────▼──────┐   ┌──────▼──────┐
  │ Session   │    │  Broadcast  │   │  Awareness  │
  │ Manager   │    │  Channels   │   │  State      │
  └───────────┘    └─────────────┘   └─────────────┘
```

---

## Key Components

### 1. Session Manager

**Purpose**: Manages all active WebSocket connections

**File**: `internal/services/collaboration/session_manager.go`

**Features**:
- Concurrent-safe session tracking
- Document-based "rooms" (multiple users per document)
- Automatic cleanup of inactive sessions
- Broadcast messaging to all users in a room

**Learning - Concurrency Patterns**:
```go
type SessionManager struct {
    documents map[string]map[*Session]bool  // Document rooms
    mu        sync.RWMutex                  // Concurrent access control
    
    register   chan *Session                // Registration channel
    unregister chan *Session                // Unregistration channel
    broadcast  chan *BroadcastMessage       // Broadcast channel
}
```

**Why channels?**
- Thread-safe communication
- Decouples senders from receivers
- Natural fit for Go's concurrency model

### 2. WebSocket Handler

**Purpose**: Upgrades HTTP connections to WebSocket

**File**: `internal/services/collaboration/websocket_handler.go`

**Flow**:
1. Client requests: `ws://localhost:8080/ws/document/doc123?user_id=alice`
2. Server upgrades HTTP connection to WebSocket
3. Creates session with unique ID
4. Registers with SessionManager
5. Starts read/write pumps (goroutines)

---

## Connection Flow

### Client Connects

```
1. HTTP Upgrade Request
   GET /ws/document/doc123?user_id=alice&user_name=Alice
   Upgrade: websocket
   
2. Server Response
   HTTP/1.1 101 Switching Protocols
   Upgrade: websocket
   
3. Session Created
   {
     "id": "2BkYU9f3T6xYh8rQ1pKJVW4vZ9L",
     "document_id": "doc123",
     "user_id": "alice",
     "user_name": "Alice",
     "connected_at": "2025-11-23T19:00:00Z"
   }
   
4. Join Notification Broadcast
   → Sent to all other users in document
   {
     "type": "join",
     "user": {
       "id": "alice",
       "name": "Alice"
     }
   }
   
5. Initial State Sent to New User
   {
     "type": "initial_state",
     "users": [
       {"id": "bob", "name": "Bob"},
       {"id": "charlie", "name": "Charlie"}
     ],
     "awareness": {...}
   }
```

### Client Sends Message

```
1. Client sends edit
   Binary message (Yjs update)
   
2. Server receives in ReadPump
   go session.ReadPump(ctx)
   
3. Broadcast to other clients
   sessionManager.Broadcast(docID, message, sender)
   
4. WritePump sends to each client
   go session.WritePump(ctx)
```

### Client Disconnects

```
1. Connection closed
   
2. Session unregistered
   sessionManager.unregister <- session
   
3. Leave notification broadcast
   {
     "type": "leave",
     "user": {"id": "alice", "name": "Alice"}
   }
   
4. Clean up resources
   - Close send channel
   - Remove from document room
   - Remove awareness state
```

---

## Concurrency Model

### Read/Write Pumps

**Problem**: WebSocket connections are **not** thread-safe

**Solution**: Separate goroutines for reading and writing

```go
// ReadPump: One goroutine reads from WebSocket
func (s *Session) ReadPump(ctx context.Context) {
    for {
        _, message, err := s.Conn.ReadMessage()
        if err != nil {
            break
        }
        
        // Broadcast to other users
        s.Manager.Broadcast(s.DocumentID, message, s)
    }
}

// WritePump: Another goroutine writes to WebSocket
func (s *Session) WritePump(ctx context.Context) {
    for {
        select {
        case message := <-s.Send:
            s.Conn.WriteMessage(websocket.BinaryMessage, message)
        case <-ticker.C:
            // Ping to keep connection alive
            s.Conn.WriteMessage(websocket.PingMessage, nil)
        }
    }
}
```

**Why separate pumps?**
- Reading doesn't block writing
- Writing doesn't block reading
- Ping/pong keep-alive works independently

---

## Awareness State

**Purpose**: Share user presence information (cursors, selections, names)

**Data Structure**:
```go
type AwarenessState struct {
    ClientID int                    `json:"client_id"`
    User     *UserInfo              `json:"user"`
    Cursor   *CursorPosition        `json:"cursor"`
    State    map[string]interface{} `json:"state"`
}

type CursorPosition struct {
    Line   int `json:"line"`
    Column int `json:"column"`
}
```

**Updates**:
```javascript
// Client-side (example)
awareness.setLocalState({
    user: { name: "Alice", color: "#ff6b6b" },
    cursor: { line: 42, column: 15 }
})

// Server broadcasts to all users in room
```

---

## Message Types

```go
const (
    MessageTypeSync          = 0  // Yjs sync (state vector)
    MessageTypeSyncUpdate    = 1  // Yjs updates (changes)
    MessageTypeAwareness     = 2  // User presence
    MessageTypeQueryAwareness = 3  // Request awareness
    
    MessageTypeJoin  = 10  // User joined
    MessageTypeLeave = 11  // User left
    MessageTypeError = 99  // Error
)
```

---

## Scalability Considerations

### Current Implementation (Single Server)

- All sessions in-memory
- Good for: 100s of concurrent users
- Limit: Single server RAM/CPU

### Future: Horizontal Scaling

**Option 1: Redis Pub/Sub**
```
Server A → Redis → Server B
   ↓                  ↓
 Users 1-100     Users 101-200
```

**Option 2**: Dedicated WebSocket Servers
```
Load Balancer
   ├─ WS Server 1 (sticky sessions)
   ├─ WS Server 2
   └─ WS Server 3
```

---

## Testing WebSocket

### Using websocat

```bash
# Install
brew install websocat

# Connect
websocat ws://localhost:8080/ws/document/doc123?user_id=alice&user_name=Alice

# Send message (paste JSON)
{"type": "test", "content": "hello"}
```

### Using JavaScript

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/document/doc123?user_id=alice&user_name=Alice');

ws.onopen = () => {
    console.log('Connected');
    ws.send(JSON.stringify({ type: 'test', content: 'hello' }));
};

ws.onmessage = (event) => {
    console.log('Received:', event.data);
};

ws.onerror = (error) => {
    console.error('Error:', error);
};

ws.onclose = () => {
    console.log('Disconnected');
};
```

---

## Cleanup & Resource Management

### Automatic Cleanup

```go
// Runs every 30 seconds
func (sm *SessionManager) cleanupLoop() {
    ticker := time.NewTicker(30 * time.Second)
    
    for {
        select {
        case <-ticker.C:
            sm.cleanup()  // Remove inactive sessions
        }
    }
}
```

**Removes sessions if**:
- No activity for > 5 minutes
- Connection closed
- Buffer full (client too slow)

### Graceful Shutdown

```go
// Shutdown sequence in main.go
1. Stop accepting new connections
2. Close all WebSocket connections
3. Wait for goroutines to finish
4. Exit clean
```

---

## Security Considerations

### Authentication

**Current** (Development):
```
?user_id=alice&user_name=Alice
```

**Production** (TODO):
```go
// Validate JWT token
token := r.Header.Get("Authorization")
claims, err := validateToken(token)
if err != nil {
    return unauthorized
}
userID := claims.UserID
```

### Rate Limiting

**TODO**: Limit messages per second per user
```go
type RateLimiter struct {
    requests map[string]int
    mu       sync.Mutex
}
```

### Message Validation

**TODO**: Validate message size and type
```go
if len(message) > maxMessageSize {
    return error
}
```

---

## Monitoring

### Metrics to Track

1. **Active Connections**: Current WebSocket count
2. **Messages/Second**: Throughput
3. **Average Latency**: Time from send to receive
4. **Error Rate**: Failed connections, timeouts
5. **Room Sizes**: Users per document

### With Jaeger Tracing

```go
ctx, span := middleware.StartSpan(ctx, "WebSocket.ProcessMessage")
span.SetAttributes(
    attribute.String("document.id", docID),
    attribute.Int("room.size", len(sessions)),
)
defer span.End()
```

---

## Troubleshooting

### Connection Drops

**Symptom**: Clients disconnect randomly

**Causes**:
- Network timeout (use ping/pong)
- Proxy timeout (configure keep-alive)
- Buffer overflow (slow client)

**Solutions**:
```go
// Increase ping frequency
ticker := time.NewTicker(30 * time.Second)

// Larger buffer
Send: make(chan []byte, 512)
```

### Messages Not Broadcasting

**Symptom**: User A sends but User B doesn't receive

**Debug**:
1. Check logs for registration
2. Verify users in same document room
3. Check WritePump is running
4. Test with single client first

---

## Next Steps (Phase 3)

- [ ] Yjs CRDT protocol implementation
- [ ] Conflict-free document merging
- [ ] Operational transformation
- [ ] Persistent session state
- [ ] Redis for horizontal scaling

---

## Summary

**WebSocket Server** enables real-time collaboration:
- ✅ Concurrent session management
- ✅ Broadcast messaging
- ✅ Awareness state (user presence)
- ✅ Graceful shutdown
- ✅ Automatic cleanup

**Connect**: `ws://localhost:8080/ws/document/:id`

**Ready for Phase 3**: Yjs CRDT integration! 🚀
