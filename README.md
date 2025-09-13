# Ryx - Distributed Computing System

A practical implementation of Dave Ackley's robust-first distributed computing model, built in Go for production deployment.

## ✨ Features

### Phase 1: Core Foundation ✅
- **Effortless scaling**: Add nodes without configuration changes
- **Automatic discovery**: Nodes find each other via UDP broadcast
- **Single binary**: Deploy anywhere with zero dependencies
- **HTTP API**: Monitor and control via REST endpoints

### Phase 2A: Information Diffusion ✅
- **Information injection**: Seed data into the network via HTTP API
- **Content-addressable storage**: SHA256-based message deduplication
- **TTL-based cleanup**: Automatic memory management prevents leaks
- **Cluster management**: `ryx-cluster` tool for easy local testing
- **Comprehensive logging**: Detailed operation tracking for debugging

## 🚀 Quick Start

### Phase 2A: Automated Cluster Testing

```bash
# Build both binaries
go build -o ryx-node ./cmd/ryx-node
go build -o ryx-cluster ./cmd/ryx-cluster

# Start a 3-node cluster
./ryx-cluster -cmd start -nodes 3

# Inject information into the network
./ryx-cluster -cmd inject -content "Hello Ryx Network" -energy 5

# Check cluster status
./ryx-cluster -cmd status

# Stop the cluster
./ryx-cluster -cmd stop
```

### Manual Node Testing

```bash
# Start individual nodes manually
./ryx-node --port 9010 --http-port 8010

# In another terminal, start a second node
./ryx-node --port 9011 --http-port 8011

# Watch them discover each other automatically!
```

### Check Status

```bash
# View node status and neighbors
curl http://localhost:8010/status | jq

# Simple health check
curl http://localhost:8010/health

# Test connectivity
curl http://localhost:8010/ping
```

## 📖 Usage

### Command Line Options

```bash
./ryx-node [options]

Options:
  --port int         UDP port for node communication (default 9001)
  --http-port int    HTTP API port (default 8001)  
  --cluster-id str   Cluster identifier (default "default")
  --node-id str      Node identifier (auto-generated if empty)
```

### Example: 3-Node Cluster

```bash
# Terminal 1
./ryx-node --port 9010 --http-port 8010

# Terminal 2  
./ryx-node --port 9011 --http-port 8011

# Terminal 3
./ryx-node --port 9012 --http-port 8012

# Check cluster status
curl -s http://localhost:8010/status | jq '.neighbors | length'  # Should show 2
curl -s http://localhost:8011/status | jq '.neighbors | length'  # Should show 2  
curl -s http://localhost:8012/status | jq '.neighbors | length'  # Should show 2
```

## 🔌 HTTP API

Each node exposes a REST API on its HTTP port:

### Endpoints

#### Core Endpoints
| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/status` | GET | Detailed node status | Node info + neighbors + diffusion |
| `/health` | GET | Health check | Simple status |
| `/ping` | GET | Connectivity test | Pong response |

#### Phase 2A: Information Diffusion
| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/inject` | POST | Inject information into network | Success + info details |
| `/info` | GET | List all stored information | Count + info list |
| `/info/{id}` | GET | Get specific information by ID | Info details |

### API Examples

```bash
# Get full node status
curl http://localhost:8010/status
{
  "node_id": "node_f4960d50",
  "cluster_id": "default", 
  "port": 9010,
  "http_port": 8010,
  "running": true,
  "uptime": "171ns",
  "neighbors": {
    "node_dc2dd334": {
      "node_id": "node_dc2dd334",
      "address": "127.0.0.1",
      "port": 9011,
      "cluster_id": "default",
      "last_seen": "2025-09-13T12:17:24.056121552+02:00"
    }
  }
}

# Health check
curl http://localhost:8010/health
{
  "status": "healthy",
  "node_id": "node_f4960d50", 
  "timestamp": 1757758332
}

# Inject information (Phase 2A)
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello Network","energy":5,"ttl":300}'
{
  "success": true,
  "info": {
    "id": "9f86d081884c7d65",
    "type": "text",
    "content": "SGVsbG8gTmV0d29yaw==",
    "energy": 5,
    "ttl": 1757760123,
    "hops": 0,
    "source": "node_f4960d50",
    "path": ["node_f4960d50"],
    "timestamp": 1757759823
  },
  "message": "Information injected successfully"
}

# List all information
curl http://localhost:8010/info
{
  "count": 1,
  "info": {
    "9f86d081884c7d65": { ... }
  }
}
```

## 🏗️ Architecture

### Core Components

- **Node Discovery**: UDP broadcast for automatic neighbor detection
- **Communication**: UDP messaging between neighbors
- **HTTP API**: REST endpoints for monitoring and control
- **Health Monitoring**: Automatic neighbor health tracking

### Network Topology

```
┌─────────────┐     UDP      ┌─────────────┐     UDP      ┌─────────────┐
│    Node A   │◄──────────►│    Node B   │◄──────────►│    Node C   │
│  Port 9010  │             │  Port 9011  │             │  Port 9012  │ 
│  HTTP 8010  │             │  HTTP 8011  │             │  HTTP 8012  │
└─────────────┘             └─────────────┘             └─────────────┘
      ▲                           ▲                           ▲
      │ HTTP API                  │ HTTP API                  │ HTTP API
      ▼                           ▼                           ▼
 ┌──────────┐               ┌──────────┐               ┌──────────┐
 │  Client  │               │  Client  │               │  Client  │
 └──────────┘               └──────────┘               └──────────┘
```

### Discovery Process

1. Each node broadcasts its presence every 5 seconds
2. Nodes listen on discovery ports (base_port + 1000)
3. Announcements are sent to port range 10000-10019
4. Neighbors are tracked with last-seen timestamps
5. Stale neighbors are cleaned up after 60 seconds

## 🧠 Key Concepts

### **Content-Addressable Storage**

Ryx uses **SHA256-based content addressing** where each piece of information gets a unique ID based on its content. This is fundamental to how the system works:

```bash
# Same content always produces the same ID
"Hello World" → ID: 9f86d081884c7d65
"Hello World" → ID: 9f86d081884c7d65  # Same ID, duplicate detected

# Different content produces different IDs  
"Hello Network" → ID: a1b2c3d4e5f6789a
"Hello Universe" → ID: b2c3d4e5f6789ab1
```

**Why This Design?**
- ✅ **Deduplication**: Identical information is stored only once
- ✅ **Data Integrity**: Content hash verifies data hasn't been corrupted
- ✅ **Loop Prevention**: Critical for distributed diffusion algorithms
- ✅ **Memory Efficiency**: Same content uses same storage across all nodes

### **Understanding Message Behavior**

#### **Expected: Unique Content Creates New Messages**
```bash
./ryx-cluster -cmd inject -content "Event A"    # Creates message 1
./ryx-cluster -cmd inject -content "Event B"    # Creates message 2
./ryx-cluster -cmd inject -content "Event C"    # Creates message 3
# Result: 3 different messages stored
```

#### **Expected: Duplicate Content is Deduplicated**
```bash
./ryx-cluster -cmd inject -content "Log Entry"           # Creates message 1
./ryx-cluster -cmd inject -content "Log Entry"           # Duplicate, not stored
./ryx-cluster -cmd inject -content "Log Entry" -energy 10 # Still duplicate
# Result: Only 1 message stored (deduplication working correctly)
```

#### **Generating Unique Content for Testing**
```bash
# Use timestamps for unique content
./ryx-cluster -cmd inject -content "Event $(date +%s)"

# Use counters
for i in {1..5}; do
  ./ryx-cluster -cmd inject -content "Message $i"
done

# Use random data
./ryx-cluster -cmd inject -content "Data $RANDOM"
```

### **Phase 2A vs Phase 2B Behavior**

**Current Phase 2A**: Information Storage Foundation
- ✅ Each node stores information independently
- ✅ Content-addressable deduplication within each node
- ✅ No inter-node message sharing (that's Phase 2B)

**Future Phase 2B**: Inter-Node Diffusion
- ⏳ Messages will spread between neighbor nodes
- ⏳ Energy decay will limit propagation distance
- ⏳ Loop prevention using path tracking

## 🛠️ Development

### Project Structure

```
ryx/
├── cmd/
│   └── ryx-node/           # Main node binary
├── internal/
│   ├── node/               # Core node logic
│   ├── discovery/          # Neighbor discovery
│   ├── communication/      # UDP messaging
│   └── api/                # HTTP API server
├── go.mod                  # Go module
├── ROADMAP.md             # Development roadmap
└── README.md              # This file
```

### Building

```bash
# Build main binary
go build -o ryx-node ./cmd/ryx-node

# Build all packages  
go build ./...

# Format code
go fmt ./...

# Check dependencies
go mod tidy
```

### Testing Discovery

```bash
# Start multiple nodes and watch logs
./ryx-node --port 9010 --http-port 8010 &
./ryx-node --port 9011 --http-port 8011 &
./ryx-node --port 9012 --http-port 8012 &

# You should see logs like:
# "Discovered neighbor: node_abc123 at 127.0.0.1:9011"

# Clean up
pkill ryx-node
```

## 🗺️ Roadmap

### ✅ Phase 1: Core Foundation (Complete)
- Node discovery and communication
- HTTP API for monitoring
- Graceful startup/shutdown

### ✅ Phase 2A: Information Diffusion (Complete)
- Information injection via HTTP API
- Content-addressable storage with SHA256 IDs
- TTL-based automatic cleanup
- `ryx-cluster` tool for easy local testing
- Comprehensive logging and error handling

### 🚧 Phase 2B: Computation Engine (Next)
- Energy-based task diffusion between nodes
- Distributed computation execution  
- Result aggregation through neighbor consensus

### 📋 Phase 3: Advanced Development Tools
- Large-scale cluster simulation
- Chaos engineering capabilities
- Performance benchmarking and visualization

### 🎯 Phase 4: Production Ready
- Multi-machine deployment
- Monitoring and observability
- Container deployment

See [ROADMAP.md](ROADMAP.md) for complete development plan.

## 🤝 Contributing

This project implements Dave Ackley's robust computing principles:

- **Local-only communication**: Nodes only talk to immediate neighbors
- **Energy-based diffusion**: Information spreads with decay over time
- **No global coordination**: System self-organizes through local interactions  
- **Automatic fault tolerance**: Work routes around failures naturally

See [AGENTS.md](AGENTS.md) for coding guidelines.

## 📜 License

MIT License - see LICENSE file for details.

---

**Status**: Phase 1 complete ✅ - Ready for Phase 2 development