# CRDT vs OT - Technical Deep Dive

## Why CRDT Over Operational Transformation?

### Executive Summary

**CRDT (Conflict-free Replicated Data Type)** is superior to **OT (Operational Transformation)** for modern distributed systems because it's **decentralized**, **simpler to implement correctly**, and **eventually consistent** without requiring a central server.

---

## Comparison: CRDT vs OT

| Aspect | CRDT | OT |
|--------|------|-----|
| **Architecture** | Peer-to-peer, decentralized | Requires central server |
| **Complexity** | Simpler (mathematical properties) | Complex (transform functions) |
| **Offline Support** | Excellent (merge when back online) | Poor (needs server) |
| **Convergence** | Guaranteed (by design) | Requires correct transforms |
| **Performance** | Better for large documents | Better for small changes |
| **Network** | Works with any topology | Star topology (hub-and-spoke) |
| **Correctness** | Mathematically proven | Easy to get wrong |

---

## The Problem They Solve

### Concurrent Editing Scenario

```
Initial state: "Hello"

User A: Insert "Beautiful " at position 0 → "Beautiful Hello"
User B: Insert " World" at position 5 → "Hello World"

Without CRDT/OT:
  A sees: "Beautiful Hello World"  ❌ Wrong!
  B sees: "Hello Beautiful World"  ❌ Wrong!

With CRDT:
  Both see: "Beautiful Hello World" ✅ Correct!
```

---

## Why OT Falls Short

### 1. Central Server Requirement

```
    User A ─────┐
                 ▼
    User B ───► Server (must order all operations)
                 ▲
    User C ─────┘
```

**Problems:**
- Single point of failure
- Server must process every operation
- Doesn't work offline
- Scalability bottleneck

### 2. Transform Function Complexity

OT requires writing correct transformation functions for every operation pair:

```javascript
// Example: Transform insert against insert
function transform(op1, op2) {
    if (op1.position < op2.position) {
        return op1;
    } else {
        return { ...op1, position: op1.position + op2.length };
    }
}
```

**Problems:**
- Must handle all operation combinations
- Edge cases are subtle and hard to test
- One bug breaks convergence for all users
- Google Wave failed partly due to OT complexity

### 3. TP1 and TP2 Properties

OT requires satisfying two properties:

**TP1** (Transformation Property 1):
```
transform(transform(a, b), transform(b, a)) = 
transform(transform(b, a), transform(a, b))
```

**TP2** (Transformation Property 2):
```
transform(a, transform(b, c)) = 
transform(transform(a, b), c)
```

**Problem**: Proving these hold for all operations is extremely difficult!

---

## How CRDTs Work

### Core Principle: Commutativity

CRDTs ensure that applying operations in **any order** produces the **same result**.

```
A + B = B + A  (mathematical commutativity)

Op1 ◦ Op2 = Op2 ◦ Op1  (operational commutativity)
```

### Types of CRDTs

#### 1. State-based CRDTs (CvRDT)

**Idea**: Send entire state, merge with local state

```
User A state: {...}  ──┐
                        ├─► Merge(stateA, stateB) → Final state
User B state: {...}  ──┘
```

**Pros**: Simple, works on any network
**Cons**: Large network overhead

#### 2. Operation-based CRDTs (CmRDT)

**Idea**: Send operations, apply in causal order

```
User A: Op1, Op2, Op3 → Broadcast
User B: Apply in causal order → Same result
```

**Pros**: Efficient network usage
**Cons**: Requires causal delivery

---

## Yjs CRDT Implementation

**Yjs** uses a **list CRDT** based on the **RGA (Replicated Growable Array)** algorithm.

### Internal Structure

Instead of positions (0, 1, 2...), Yjs uses **unique identifiers**:

```
Traditional array (positions):
[0]   [1]      [2]     [3]    [4]
 H     e        l       l      o

Yjs CRDT (unique IDs):
[A:0] [A:1]   [A:2]   [A:3]  [A:4]
  H     e       l       l      o

Insert "Beautiful " at position 0:
[B:0] [B:1] ... [B:10] [A:0] [A:1] [A:2] [A:3] [A:4]
  B     e   ...   l       H     e     l     l     o
```

### Key Components

#### 1. Unique Identifiers

Every character gets a **globally unique ID**:

```javascript
{
    client: "user-123",     // Who created it
    clock: 42,              // Logical timestamp
}
```

**Properties**:
- Immutable once created
- Globally unique across all clients
- Orders characters deterministically

#### 2. Lamport Timestamps (Logical Clocks)

Each client maintains a **logical clock**:

```
User A clock: 0 → 1 → 2 → 3
User B clock: 0 → 1 → 2 → 3

User A inserts "H": {client: A, clock: 1}
User B inserts "W": {client: B, clock: 1}

Merge:
  Compare (A, 1) vs (B, 1)
  → Use client ID as tiebreaker
  → Result: [A:1, B:1] or [B:1, A:1] (deterministic)
```

#### 3. Tombstones (Deleted Items)

Deleted items are **marked**, not removed:

```
Original: [A:0, B:1, C:2, D:3]
Delete "B": [A:0, B:1 (deleted), C:2, D:3]

Other clients see:
  [A:0, C:2, D:3]  (B hidden but still in structure)

Why?
  - Ensures all clients can reference B
  - Prevents divergence
  - Allows undo/redo
```

#### 4. Garbage Collection

Eventually, tombstones are removed:

```
After all clients acknowledge deletion:
  [A:0, B:1 (deleted), C:2, D:3]
    ↓
  [A:0, C:2, D:3]  (B actually removed)
```

---

## Synchronization Protocol

### 1. State Vector (Version Vector)

Each client tracks what it has seen from every client:

```javascript
User A state vector:
{
    "A": 5,  // Seen A's operations 0-4
    "B": 3,  // Seen B's operations 0-2
    "C": 0   // Seen nothing from C
}
```

### 2. Sync Protocol Steps

**Step 1: Exchange State Vectors**

```
Client A → Client B: 
  "I have: {A: 5, B: 3, C: 0}"

Client B → Client A:
  "I have: {A: 4, B: 5, C: 2}"
```

**Step 2: Calculate Missing Operations**

```
Client B analyzes A's vector:
  - A has A:0-4, but B has A:0-4 → Send A:5 to B
  - A has B:0-2, but B has B:0-4 → Send B:3,4 to A
  - A has C:0, but B has C:0-1 → Send C:0,1 to A
```

**Step 3: Send Missing Updates**

```
Client B → Client A:
  [Op(A:5), Op(B:3), Op(B:4), Op(C:0), Op(C:1)]

Client A → Client B:
  [Op(A:5)]
```

**Step 4: Apply Updates**

```
Both clients apply operations in causal order
  → Converge to same state
```

### 3. Causal Consistency

Operations are applied in **causal order**:

```
User A:
  1. Insert "H" at 0 → Op1 {clock: 1}
  2. Insert "i" at 1 → Op2 {clock: 2, depends: [1]}

User B receives:
  - Op2 arrives first
  - Waits for Op1 (dependency)
  - Applies Op1, then Op2
  → Correct order maintained
```

---

## Example: Concurrent Inserts

### Scenario

```
Initial: "Hello"

User A: Insert " World" after "Hello" → "Hello World"
User B: Insert "Beautiful " before "Hello" → "Beautiful Hello"
```

### Traditional (Position-based) ❌

```
User A: Insert " World" at position 5
User B: Insert "Beautiful " at position 0

A applies B's change:
  Position 5 + 10 (length of "Beautiful ") = position 15
  Result: "Beautiful Hello World" ❌ Wrong position!

B applies A's change:
  Position 5 (unchanged)
  Result: "Hello World" then "Beautiful " at 0
  Result: "Beautiful Hello" ❌ Missing " World"!
```

### CRDT (ID-based) ✅

```
Initial state:
  [A:0]H [A:1]e [A:2]l [A:3]l [A:4]o

User A inserts " World" after [A:4]:
  [A:0]H [A:1]e [A:2]l [A:3]l [A:4]o [C:0]  [C:1]W [C:2]o [C:3]r [C:4]l [C:5]d

User B inserts "Beautiful " before [A:0]:
  [B:0]B [B:1]e [B:2]a [B:3]u... [A:0]H [A:1]e [A:2]l [A:3]l [A:4]o

Merge (both clients):
  Sort by ID: B < A < C (assuming clockIDs tie-break by client ID)
  
  Result:
  [B:0]B [B:1]e [B:2]a [B:3]u [B:4]t [B:5]i [B:6]f [B:7]u [B:8]l [B:9]  
  [A:0]H [A:1]e [A:2]l [A:3]l [A:4]o 
  [C:0]  [C:1]W [C:2]o [C:3]r [C:4]l [C:5]d
  
  → "Beautiful Hello World" ✅ Correct!
```

### Why It Works

1. **Insertion points are IDs, not positions**
   - "After [A:4]" is immutable
   - Doesn't change when B inserts before

2. **Deterministic ordering**
   - All clients sort by same rule
   - IDs are globally unique
   - Convergence guaranteed

---

## CRDT Properties

### 1. Commutativity

```
Op1 then Op2 = Op2 then Op1
```

**Example**:
```
Insert "A" at 0 + Insert "B" at 0
  = [A, B] or [B, A]?

With CRDT IDs:
  [A:1] and [B:1] → Sort by client ID → Deterministic
```

### 2. Associativity

```
(Op1 + Op2) + Op3 = Op1 + (Op2 + Op3)
```

**Example**:
```
Merge in any order produces same result
```

### 3. Idempotence

```
Op1 + Op1 = Op1
```

**Example**:
```
Applying same operation twice has no effect
  → Safe for retransmission
```

---

## Performance Characteristics

### Memory Overhead

```
Traditional string: "Hello" = 5 bytes

CRDT structure:
  [
    {id: {client: "A", clock: 0}, value: "H"},  // ~50 bytes
    {id: {client: "A", clock: 1}, value: "e"},  // ~50 bytes
    ...
  ]
  
Total: ~250 bytes for "Hello"
```

**Optimization**: Yjs uses **run-length encoding**:
```
Instead of: [A:0]H [A:1]e [A:2]l [A:3]l [A:4]o
Store as: {id: A:0, content: "Hello"}  // Much more efficient!
```

### Network Bandwidth

**Full state**: Send entire document (large)
**Delta sync**: Send only differences (efficient)

```
Yjs delta sync:
  - State vector: ~10 bytes per client
  - Operations: ~30 bytes per operation
  - Compression: Binary encoding reduces further
```

---

## Real-world Use Cases

### Why Google Docs Uses OT

- **Started before CRDTs matured** (2006)
- **Central infrastructure** already exists
- **Fine-grained control** over conflict resolution
- **Rewriting is expensive**

### Why Modern Apps Use CRDTs

- **Figma**: Real-time collaboration on designs
- **Notion**: Block-based documents
- **Linear**: Issue tracking
- **Atom Teletype**: Pair programming

**Reasons**:
- Offline first
- Peer-to-peer sync
- Simpler to implement
- Better for distributed systems

---

## Trade-offs

### CRDT Wins

✅ Decentralized (no server bottleneck)
✅ Offline support
✅ Mathematically proven correctness
✅ Simpler implementation
✅ Works with any network topology

### OT Wins

✅ Lower memory overhead
✅ Finer control over conflicts
✅ Better for small, frequent changes
✅ Legacy applications

---

## Implementing CRDT (Yjs Example)

### Basic Usage

```javascript
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'

// Create document
const ydoc = new Y.Doc()

// Create shared text
const ytext = ydoc.getText('content')

// WebSocket sync provider
const provider = new WebsocketProvider(
  'ws://localhost:8080',
  'doc-123',
  ydoc
)

// Insert text
ytext.insert(0, 'Hello')

// Listen to changes
ytext.observe(event => {
  console.log('Changes:', event.changes)
})

// Automatically syncs with all connected clients!
```

### Integration with Our Go Backend

```go
// WebSocket handler receives Yjs binary updates
func (s *Session) ReadPump(ctx context.Context) {
    for {
        _, message, err := s.Conn.ReadMessage()
        
        // message contains Yjs CRDT update (binary)
        // Broadcast to all other clients in room
        s.Manager.Broadcast(s.DocumentID, message, s)
    }
}
```

**That's it!** Yjs handles all the CRDT logic.

---

## Summary

### CRDT Over OT Because:

1. **Decentralized** → No central server needed
2. **Simpler** → Fewer moving parts
3. **Proven** → Mathematical guarantees
4. **Offline-first** → Works without network
5. **Modern** → Better for distributed systems

### How CRDT Works:

1. **Unique IDs** for every character
2. **Logical clocks** for ordering
3. **State vectors** for sync
4. **Tombstones** for deletes
5. **Causal delivery** for consistency

### Synchronization:

1. Exchange **state vectors**
2. Calculate **missing operations**
3. Send **delta updates**
4. Apply in **causal order**
5. **Converge** to same state

---

## References

- [Yjs Documentation](https://docs.yjs.dev/)
- [CRDT Tech](https://crdt.tech/)
- [Conflict-free Replicated Data Types (Shapiro et al.)](https://hal.inria.fr/hal-00932836/document)
- [Real Differences between OT and CRDT](https://www.tiny.cloud/blog/real-time-collaboration-ot-vs-crdt/)
- [Why Figma Uses CRDTs](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/)

---

**Next**: Implement Yjs protocol in our WebSocket server! 🚀
