# Ryx - Distributed Computing System

A practical implementation of Dave Ackley's robust-first distributed computing model, built in Go for production deployment.

## Features

### Phase 1: Core Foundation (Complete)
- **Effortless scaling**: Add nodes without configuration changes
- **Automatic discovery**: Nodes find each other via UDP broadcast
- **Single binary**: Deploy anywhere with zero dependencies
- **HTTP API**: Monitor and control via REST endpoints

### Phase 2A: Information Storage (Complete)
- **Information injection**: Seed data into the network via HTTP API
- **Content-addressable storage**: SHA256-based message deduplication
- **TTL-based cleanup**: Automatic memory management prevents leaks
- **Cluster management**: `ryx-cluster` tool for easy local testing
- **Comprehensive logging**: Detailed operation tracking for debugging

### Phase 2B: Inter-Node Diffusion (Complete)
- **Message forwarding**: Information automatically spreads between neighbors
- **Energy decay**: Messages lose energy with each hop, limiting propagation distance
- **Loop prevention**: Path tracking prevents infinite message cycles
- **Hop tracking**: Full propagation history maintained for analysis

### Phase 2C: Distributed Computation (Complete)
- **Task injection**: Inject computational tasks via HTTP API (`/compute`)
- **Energy-based task distribution**: Tasks spread through network using existing diffusion system
- **Local computation execution**: WordCount executor processes tasks on each node
- **Automatic consensus**: Identical results achieve consensus through content-addressable storage
- **Result aggregation**: Query computation results across the network

### Phase 2 Enhancement: Self-Modification (Complete)
- **Runtime parameter system**: 20+ configurable system parameters for autonomous adaptation
- **Message-type aware behavior**: Critical messages live 3× longer, routine messages 2× shorter
- **Behavior modification API**: HTTP endpoints for runtime system configuration and tuning
- **Adaptive algorithms**: Performance tracking and learning hooks for autonomous optimization
- **Thread-safe modification**: Concurrent parameter updates with mutex protection
- **Mission-critical foundation**: Self-modification capabilities for decades-long autonomous operation

## Quick Start

### Automated Cluster Testing

```bash
# Build both binaries
go build -o ryx-node ./cmd/ryx-node
go build -o ryx-cluster ./cmd/ryx-cluster

# Start a 3-node cluster
./ryx-cluster -cmd start -nodes 3

# Inject information that spreads across all nodes (Phase 2B)
./ryx-cluster -cmd inject -content "Hello Ryx Network" -energy 3

# Inject computational task that executes across cluster (Phase 2C)
curl -X POST localhost:8010/compute \
  -H "Content-Type: application/json" \
  -d '{"type":"wordcount","data":"distributed computing with ryx","energy":2}'

# Test self-modification: Critical vs routine messages (Phase 2 Enhancement)
curl -X POST localhost:8010/inject -H "Content-Type: application/json" \
  -d '{"type":"critical","content":"Emergency alert","energy":5,"ttl":3600}'
curl -X POST localhost:8010/inject -H "Content-Type: application/json" \
  -d '{"type":"routine","content":"Status update","energy":5,"ttl":3600}'

# Runtime system configuration (Phase 2 Enhancement)  
curl -X GET localhost:8010/config                    # View all parameters
curl -X PUT localhost:8010/config/energy_decay_rate -H "Content-Type: application/json" \
  -d '{"value": 0.5}'                                # Modify energy decay

# Check cluster status and results
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

## HTTP API

Each node exposes a REST API on its HTTP port:

### Endpoints

#### Core Endpoints
| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/status` | GET | Detailed node status | Node info + neighbors + diffusion |
| `/health` | GET | Health check | Simple status |
| `/ping` | GET | Connectivity test | Pong response |

#### Information Diffusion
| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/inject` | POST | Inject information that spreads across network | Success + info details |
| `/info` | GET | List all stored information | Count + info list |
| `/info/{id}` | GET | Get specific information by ID | Info details |

#### Distributed Computation (Phase 2C)
| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/compute` | POST | Inject computational task that executes across network | Task details |
| `/compute` | GET | List active and completed computations | Computation status |
| `/compute/{id}` | GET | Get specific computation result | Result details |

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

# Inject information that spreads across cluster
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello Network","energy":3,"ttl":300}'
{
  "success": true,
  "info": {
    "id": "9f86d081884c7d65",
    "type": "text",
    "content": "SGVsbG8gTmV0d29yaw==",
    "energy": 3,
    "ttl": 1757760123,
    "hops": 0,
    "source": "node_f4960d50",
    "path": ["node_f4960d50"],
    "timestamp": 1757759823
  },
  "message": "Information injected successfully"
}

# Check message on different nodes - shows energy decay and path tracking
curl http://localhost:8011/info/9f86d081884c7d65
{
  "info": {
    "id": "9f86d081884c7d65",
    "energy": 2,         # Energy decreased by 1
    "hops": 1,           # One hop from original node
    "path": ["node_f4960d50", "node_f4960d50"]  # Propagation path
    ...
  }
}

# Inject computational task (Phase 2C)
curl -X POST http://localhost:8010/compute \
  -H "Content-Type: application/json" \
  -d '{"type":"wordcount","data":"hello world ryx","energy":2}'
{
  "success": true,
  "message": "Computational task injected successfully",
  "task": {
    "id": "abc123def456",
    "type": "wordcount",
    "energy": 2,
    "timestamp": 1757760000
  }
}

# Check computation results on different nodes
curl http://localhost:8011/compute/abc123def456
{
  "task_id": "abc123def456",
  "result": {
    "task_type": "wordcount",
    "result": {
      "total_words": 3,
      "unique_words": 3,
      "word_counts": {"hello": 1, "world": 1, "ryx": 1}
    },
    "executed_by": "node_abc123",
    "execution_time": 5
  }
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

## Architecture

### Core Components

- **Node Discovery**: UDP broadcast for automatic neighbor detection
- **Communication**: UDP messaging between neighbors with info forwarding
- **Information Diffusion**: Energy-based message propagation with loop prevention
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

## Key Concepts

### Content-Addressable Storage

Ryx uses SHA256-based content addressing where each piece of information gets a unique ID based on its content:

```bash
# Same content always produces the same ID
"Hello World" → ID: 9f86d081884c7d65
"Hello World" → ID: 9f86d081884c7d65  # Same ID, duplicate detected

# Different content produces different IDs  
"Hello Network" → ID: a1b2c3d4e5f6789a
"Hello Universe" → ID: b2c3d4e5f6789ab1
```

**Benefits:**
- **Deduplication**: Identical information is stored only once
- **Data Integrity**: Content hash verifies data hasn't been corrupted
- **Loop Prevention**: Critical for distributed diffusion algorithms
- **Memory Efficiency**: Same content uses same storage across all nodes

### Information Diffusion

Messages spread through the network with energy-based propagation:

```bash
# Energy decreases with each hop
Original Node:  energy=3, hops=0, path=["node_A"]
Neighbor Nodes: energy=2, hops=1, path=["node_A", "node_A"]
```

**Key Features:**
- **Energy Decay**: Messages lose 1 energy per hop, stopping when energy reaches 0
- **Path Tracking**: Full propagation history prevents infinite loops
- **Automatic Forwarding**: New information spreads without manual intervention

### Testing Diffusion Behavior

#### Unique Content Creates New Messages
```bash
./ryx-cluster -cmd inject -content "Event A"    # Creates message 1, spreads to all nodes
./ryx-cluster -cmd inject -content "Event B"    # Creates message 2, spreads to all nodes
./ryx-cluster -cmd inject -content "Event C"    # Creates message 3, spreads to all nodes
# Result: 3 different messages on each node
```

#### Duplicate Content is Deduplicated
```bash
./ryx-cluster -cmd inject -content "Log Entry"           # Creates message, spreads once
./ryx-cluster -cmd inject -content "Log Entry"           # Duplicate, ignored
./ryx-cluster -cmd inject -content "Log Entry" -energy 10 # Still duplicate, ignored
# Result: Only 1 message across all nodes
```

#### Energy Limits Propagation Distance
```bash
./ryx-cluster -cmd inject -content "Low energy" -energy 1
# Result: Message reaches immediate neighbors only (energy=0 stops further spread)

./ryx-cluster -cmd inject -content "High energy" -energy 5  
# Result: Message can travel up to 5 hops through the network
```

## 🛠️ Development

### Project Structure

```
ryx/
├── cmd/
│   ├── ryx-node/           # Main node binary
│   └── ryx-cluster/        # Cluster management tool
├── internal/
│   ├── node/               # Core node logic
│   ├── discovery/          # Neighbor discovery  
│   ├── communication/      # UDP messaging
│   ├── diffusion/          # Information diffusion
│   ├── computation/        # Distributed computation
│   ├── api/                # HTTP API server
│   └── types/              # Shared data structures
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

### Phase 1: Core Foundation (Complete)
- Node discovery and communication
- HTTP API for monitoring
- Graceful startup/shutdown

### Phase 2A: Information Storage (Complete)
- Information injection via HTTP API
- Content-addressable storage with SHA256 IDs
- TTL-based automatic cleanup
- `ryx-cluster` tool for easy local testing
- Comprehensive logging and error handling

### Phase 2B: Inter-Node Diffusion (Complete)
- Energy-based message forwarding between neighbors
- Automatic loop prevention using path tracking
- Energy decay limiting propagation distance
- Hop counting for diffusion analysis

### Phase 2C: Distributed Computation (Complete)
- Distributed task execution with automatic result collection
- Energy-based task distribution using existing diffusion system  
- Automatic consensus through content-addressable storage
- WordCount executor with configurable parameters
- HTTP API for task injection and result queries (`/compute`)

### Phase 3: Advanced Development Tooling (Next)
- Large-scale cluster simulation (50+ nodes)
- Advanced chaos engineering and fault testing
- Performance benchmarking and metrics collection
- Automated testing scenarios and regression tests
- Network topology visualization and monitoring

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

**Status**: Phase 2C complete - Distributed computation with automatic consensus operational