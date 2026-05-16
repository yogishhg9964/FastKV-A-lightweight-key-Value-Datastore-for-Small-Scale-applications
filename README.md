# Hrux-DB: A Lightweight Distributed Key-Value Data Store

A high-performance, scalable distributed key-value data store designed for small-scale applications with support for multiple data structures, pub/sub messaging, and ACID transactions.

## Project Overview

Hrux-DB is a comprehensive key-value store system built with Go backend, React frontend UI, and microservice architecture for specialized use cases like flash sales.

### Key Features

- **Distributed Architecture**: Multi-node support with replication
- **Multiple Data Structures**: Key-Value pairs, Maps, Sets, Sorted Lists, Stacks, Queues
- **Pub/Sub Messaging**: Real-time event streaming and notifications
- **Transaction Support**: ACID-compliant transactions
- **Write-Ahead Logging (WAL)**: Durability and crash recovery
- **HTTP & TCP APIs**: Flexible client communication
- **Web UI Dashboard**: Interactive management interface
- **SDK Support**: JavaScript client library for easy integration

## Project Structure

```
Hrux-DB/
├── kv-distributed/          # Core Go backend
│   ├── cmd/
│   │   ├── server/          # TCP server
│   │   ├── client/          # CLI client
│   │   └── http-server/     # HTTP API server
│   ├── internal/
│   │   ├── api/             # API handlers and types
│   │   ├── datastructures/  # DS implementation
│   │   ├── indexing/        # Indexing service
│   │   ├── service/         # Core services (KV, Pub/Sub, Transactions)
│   │   └── storage/         # Storage engine and WAL
│   └── client/              # Go client library
│
├── kv-frontend/             # React + Vite UI
│   ├── src/
│   │   ├── components/      # UI components
│   │   ├── features/        # Feature pages (KeyValue, Map, Queue, etc.)
│   │   ├── api/             # API integration
│   │   └── App.jsx
│   └── vite.config.js
│
├── microservice-flashsale/  # Flash sale microservice
│   ├── server.js            # Express server
│   ├── load-test.js         # Performance testing
│   └── prove-all-features.js
│
├── sdk/                     # JavaScript SDK
│   └── hrux-client.js       # Client library for Node.js/Browser
│
└── Documentation/
    ├── hrux_db_research_paper.tex
    ├── manual_test_plan.md
    ├── metrics_methodology.md
    └── implementation plans
```

## Quick Start

### Prerequisites

- Go 1.18+
- Node.js 16+
- npm or yarn

### Backend Setup

```bash
cd kv-distributed

# Build the TCP server
go build -o server ./cmd/server

# Build the HTTP server
go build -o http-server ./cmd/http-server

# Build the CLI client
go build -o client ./cmd/client

# Run the server
./server
```

### Frontend Setup

```bash
cd kv-frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

### Flash Sale Microservice

```bash
cd microservice-flashsale

# Install dependencies
npm install

# Run the service
npm start

# Run load test
node load-test.js
```

## API Documentation

### HTTP Endpoints

The HTTP server runs on port 8080 (configurable) and provides RESTful endpoints for:

- `GET /kv/:key` - Get value
- `SET /kv/:key/:value` - Set value
- `DELETE /kv/:key` - Delete key
- `GET /datastructure/:type/:name` - Access data structures
- `POST /pubsub/subscribe` - Subscribe to topics
- `POST /pubsub/publish` - Publish messages
- `POST /transaction/begin` - Start transaction

### TCP Protocol

Binary protocol for high-performance connections. See `internal/api/types.go` for protocol specifications.

## Data Structures Supported

1. **Key-Value Pairs**: Basic get/set/delete operations
2. **Maps**: Nested key-value structures
3. **Sets**: Unique value collections
4. **Sorted Lists**: Ordered lists with scoring
5. **Stacks**: LIFO data structure
6. **Queues**: FIFO data structure

## Pub/Sub System

Real-time messaging with:
- Topic-based subscriptions
- Pattern matching support
- Message persistence options
- Multiple subscriber support

## Transaction Support

ACID-compliant transactions with:
- Isolation levels
- Rollback capability
- Conflict resolution
- Multi-key operations

## Replication

- Master-slave replication model
- Configurable replication factor
- Automatic failover
- Consistency guarantees

## Performance Features

- **Write-Ahead Logging**: Durability without full persistence
- **Indexing**: B-tree based indexing for fast lookups
- **Caching**: In-memory caching for frequently accessed keys
- **Connection Pooling**: Efficient resource utilization

## Testing

### Unit Tests
```bash
cd kv-distributed
go test ./...
```

### Load Testing
```bash
cd microservice-flashsale
node load-test.js
```

### Feature Verification
```bash
cd microservice-flashsale
node prove-all-features.js
```

## Configuration

### Environment Variables

```bash
# Backend
KV_PORT=6379              # Server port
KV_HTTP_PORT=8080         # HTTP server port
KV_DB_PATH=./data         # Data directory
KV_REPLICATION_FACTOR=3   # Replication copies
KV_WAL_ENABLED=true       # Enable WAL

# Frontend
VITE_API_URL=http://localhost:8080
```

## Client SDKs

### JavaScript/Node.js
```javascript
const HruxClient = require('./sdk/hrux-client.js');
const client = new HruxClient('localhost:6379');

// Key-Value operations
await client.set('key', 'value');
const value = await client.get('key');

// Pub/Sub
client.subscribe('topic', (msg) => {
  console.log('Received:', msg);
});
client.publish('topic', 'message');
```

### Go
```go
import "path/to/kv-distributed/client"

conn, err := client.Connect("localhost:6379")
value, err := conn.Get("key")
conn.Set("key", "value")
```

## Architecture Highlights

- **Microservices**: Modular services for different concerns
- **Pub/Sub Decoupling**: Event-driven architecture
- **Stateless HTTP API**: Horizontal scalability
- **Persistent Storage**: Durable data with WAL

## Production Considerations

- Configure appropriate replication factor for fault tolerance
- Enable WAL for data durability
- Set up monitoring and alerting
- Use connection pooling on clients
- Implement rate limiting for public endpoints
- Use SSL/TLS for secure communication

## Contributing

1. Create a feature branch
2. Make changes and test
3. Submit pull request

## Documentation

- `hrux_db_research_paper.tex` - Academic paper
- `manual_test_plan.md` - Testing procedures
- `metrics_methodology.md` - Performance metrics
- Implementation plans for specific features

## License

[Specify your license here]

## Authors

Developed as an 8th Semester Major Project at FAST.

## Support

For issues and questions, please open an issue on GitHub.

---

**Status**: Active Development
**Last Updated**: May 2026
