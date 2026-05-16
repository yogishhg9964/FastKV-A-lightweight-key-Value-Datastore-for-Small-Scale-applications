# Hrux-DB Knowledge Transfer (KT) Session

Welcome to the KT session for **Hrux-DB**. This document is designed to give you a comprehensive, low-level understanding of the system architecture, how the database engine is built from scratch, and how the various components interact.

Assume we are sitting in a conference room with a whiteboard. Let's break this system down layer by layer.

---

## 1. High-Level Architecture Overview

Hrux-DB is a **distributed, in-memory NoSQL database** written entirely in Go. It's conceptually similar to Redis. It supports basic Key-Value (KV) storage, Time-To-Live (TTL) expiration, advanced data structures (Sets, Maps, Queues, Stacks, Sorted Lists), and Master-Slave replication.

The system is highly modular and is divided into several clear "layers":
1. **Storage Layer:** Memory management and file persistence.
2. **Data Structures Layer:** Advanced types like Queues and Sets.
3. **Indexing Layer:** Prefix and Range queries.
4. **Service Layer:** The "Glue" that coordinates the engine.
5. **API / Network Layer:** RPC Server and Master-Slave replication.

Let's dive into the low-level implementation of each.

---

## 2. The Core Engine: How is Data Stored?

If you open `internal/storage/storage.go`, you'll see the heart of the database.

### In-Memory Storage
The entire database lives in RAM. The core data structure is simply a nested Go map:
```go
type Storage struct {
	mu       sync.RWMutex
	Buckets  map[string]map[string][]byte
	TTL      map[string]map[string]TTLInfo
	filename string
}
```
*   **Buckets**: Data is separated into logical "buckets" (like tables or namespaces). 
*   **Key/Value**: Inside each bucket, keys are `strings` and values are raw `[]byte`.
*   **Concurrency**: Go maps are *not* thread-safe. To prevent race conditions when multiple clients read/write simultaneously, the `Storage` struct wraps a `sync.RWMutex`. 
    *   Reads (`Get`, `List`) acquire a Read lock (`mu.RLock()`), allowing concurrent reads.
    *   Writes (`Put`, `Delete`) acquire an exclusive Write lock (`mu.Lock()`).

### Time-To-Live (TTL) Implementation
How do keys automatically expire?
We don't use a constant background scanner that deletes keys (which would spike CPU). Instead, Hrux-DB uses **Lazy Expiration**.
When a key is inserted with a TTL, an entry is added to the `TTL` map with a UNIX timestamp of its expiration time. 
Every time a `Get()` or `List()` operation is called, the system calls a helper function `checkTTL()`. If the current time is past the expiry timestamp, the system *deletes* the key on the spot and returns "Key not found". 

### Persistence (Durability)
Since RAM is volatile, how does data survive a restart?
Hrux-DB uses **Snapshotting** via Go's built-in `encoding/gob` package. The `SaveToFile()` function locks the entire database and streams the `Storage` struct into a binary file (default: `kvdata.gob`). When the server boots up (`main.go`), it attempts to `LoadFromFile()` and decode the gob file back into memory.

---

## 3. Advanced Data Structures

Just like Redis, Hrux-DB isn't just for KV pairs. It supports Lists, Sets, and Queues, implemented in `internal/datastructures/service.go`.

These are maintained in completely separate maps from the core KV store to avoid lock contention:
```go
type DataStructuresService struct {
	mu     sync.RWMutex
	Sets   map[string]Set          // Set map[string]struct{}
	Lists  map[string]SortedList   // SortedList []string
	Maps   map[string]Map          // Map map[string]interface{}
	Queues map[string]Queue        // Queue []string
	Stacks map[string]Stack        // Stack []string
}
```
*   **Sets**: Built using `map[string]struct{}`. An empty struct in Go allocates 0 bytes, making it the most memory-efficient way to check for uniqueness.
*   **Queues / Stacks**: Built using standard Go slices (`[]string`). 
    *   *Queue Pop* takes the first element and slices the array: `q[1:]`.
    *   *Stack Pop* takes the last element and slices the array: `s[:len(s)-1]`.
*   **Sorted Lists**: Whenever an item is added, it is appended to the slice, and `sort.Strings()` is immediately called. (Note: This is an $O(N \log N)$ operation on every insert, which is a known area for future optimization).

---

## 4. Master-Slave Replication

High availability is achieved through the RPC Server (`internal/api/server.go`).

When you boot the server, it exposes its methods via standard Go `net/rpc` over TCP. 
For replication, the `KVServer` struct maintains a list of connected slave nodes (`slaves []*rpc.Client`).

**How Write-Propagation works:**
Whenever a mutating operation occurs (e.g., `Put`, `Delete`, `Update`), the master node executes it locally first. Immediately after, it calls `s.replicateToSlaves("Put", req)`.
This function iterates through all registered slave TCP connections and fires asynchronous Goroutines (`go slave.Call(...)`) to replicate the command without blocking the client's response.

---

## 5. The "Two Servers" Architecture Anomaly

During your handover, you must be aware of an architectural split in `cmd/`:
1.  **`cmd/server/main.go`**: This is the **True Engine**. It wires up the `storage`, `datastructures`, `indexing`, and `api` packages via Dependency Injection and serves it over TCP RPC.
2.  **`cmd/http-server/main.go`**: This is currently a **Standalone HTTP REST Server**. If you read the code, you'll notice it completely bypasses the `internal/` packages! It defines its own global `data = make(map[string]map[string]Item)` and its own HTTP handlers (`/put`, `/get`). It is essentially a duplicated, lighter version of the DB engine designed specifically to interface with the React web dashboard (`kv-frontend`).

*Action Item for the next team:* The `http-server` should ideally be refactored to wrap the `KVService` from the `internal` packages (like the RPC server does) instead of maintaining its own global state. The RPC server actually has some `HTTPGet`, `HTTPPut` wrapper functions prepared for this exact bridge!

---

## Summary of the Request Flow

If a client sends an RPC request to `Put` a value:
1.  **Network**: Client connects to port `8080` via TCP RPC.
2.  **API Layer**: `KVServer.Put()` receives the `Request` struct.
3.  **Service Layer**: `KVService` routes it to `KVService.PutWithTTL()`.
4.  **Storage Layer**: `Storage.PutWithTTL()` acquires a Write Lock (`mu.Lock()`).
5.  **Memory**: The `[]byte` value is placed into `Buckets["bucketName"]["keyName"]`.
6.  **Replication**: `KVServer` asynchronously forwards the exact same `Request` to all slave nodes.
7.  **Response**: An "OK" is returned to the client.

## Final Handover Notes
*   **Strengths**: Zero external dependencies, highly educational, heavily concurrent (using `RWMutex`), and fast (all RAM).
*   **Weaknesses/Tech Debt**: 
    1. Snapshotting (`SaveToFile`) blocks the entire DB while writing to disk.
    2. `cmd/http-server` duplicates engine logic.
    3. `SortedList` sorts the entire array on every single insertion instead of using an insertion sort or a B-Tree.

I hope this helps your team hit the ground running!
