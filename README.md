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

### Phase 3A: Enhanced Cluster Management (Complete)
- **Large-scale clusters**: 50+ node clusters with smart resource management and parallel startup
- **Race condition fixes**: Thread-safe concurrent operations for production reliability
- **Performance optimization**: 32% faster startup with parallel node operations
- **Cluster profiles**: Predefined configurations (small/medium/large/huge) for different testing scenarios

### Phase 3B: Advanced Self-Modification (Complete)
- **Network-aware adaptation**: Latency and reliability-based energy decay modification
- **System load monitoring**: CPU/memory tracking with adaptive parameter adjustment
- **Fault pattern learning**: Exponential moving average fault tracking with adaptive routing
- **Performance-based topology**: Dynamic neighbor scoring and replacement for optimal network topology

### Phase 3C.1 & 3C.2: Spatial-Physical Computing (Complete)
- **Multi-modal coordinate systems**: GPS (fixed infrastructure), relative (vehicles), logical (cloud), none (development)
- **Hybrid neighbor selection**: 60% network performance + 40% spatial factors for intelligent topology
- **Zone-aware distribution**: 70% same-zone, 30% cross-zone neighbors for optimal redundancy
- **Physical fault isolation**: Barrier-aware routing (bulkheads, firewalls, zones) for mission-critical safety
- **Accurate distance calculations**: GPS Haversine (2715m NYC test), 3D Euclidean, logical zones
- **Spatial APIs**: Complete neighbor analysis, distance calculation, barrier management, and zone distribution

## Quick Start

### Spatial Computing Demo (Mission-Critical)

```bash
# Build both binaries
go build -o ryx-node ./cmd/ryx-node
go build -o ryx-cluster ./cmd/ryx-cluster

# Spaceship bridge node with compartment isolation
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
  --barriers "bulkhead:bridge:engine_bay:fault" --port 9010 --http-port 8010 &

# Smart city GPS node
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone datacenter_a \
  --port 9011 --http-port 8011 &

# Wait for neighbor discovery
sleep 5

# Test spatial neighbor analysis and zone distribution
curl -s localhost:8010/spatial/neighbors | jq '.zone_analysis'

# Test GPS distance calculation (NYC to Times Square ~2.7km)
curl -X POST localhost:8010/spatial/distance -H "Content-Type: application/json" -d '{
  "coord_system": "gps", "x": 40.758, "y": -73.985, "zone": "times_square"
}' | jq '.distance'

# Check barrier configuration for fault isolation
curl -s localhost:8010/spatial/barriers | jq '.'

# Stop nodes
killall ryx-node
```

### Automated Cluster Testing

```bash
# Start large spatial-aware cluster
./ryx-cluster -cmd start -profile huge  # 50 nodes with spatial awareness

# Test distributed computation across spatial cluster
curl -X POST localhost:8010/compute -H "Content-Type: application/json" \
  -d '{"type":"wordcount","data":"distributed spatial computing with ryx","energy":3}'

# Test spatial information diffusion
curl -X POST localhost:8010/inject -H "Content-Type: application/json" \
  -d '{"type":"critical","content":"Emergency spatial alert","energy":5,"ttl":3600}'

# Monitor spatial cluster status
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

## 🚀 Mission-Critical Use Cases

Ryx is designed for **mission-critical applications** where physical topology awareness and fault isolation are essential:

### Spaceship Core Systems
- **Compartment isolation**: Engine bay failures contained within bulkhead barriers
- **Bridge protection**: Command systems isolated from propulsion system failures  
- **Emergency response**: Critical messages cross barriers while routine traffic respects them
- **Autonomous operation**: Decades-long missions without ground control intervention

### Smart City Infrastructure
- **Geographic optimization**: Data centers select nearby neighbors for low-latency communication
- **Disaster resilience**: Physical damage contained to affected geographic areas
- **Load balancing**: Traffic routed to geographically optimal nodes
- **Fault isolation**: Infrastructure failures contained within physical zones

### Vehicle Systems
- **Fault containment**: Front-end collision damage doesn't disable rear systems
- **System isolation**: Critical driving systems protected from entertainment system failures
- **Redundancy**: Multiple physical zones ensure system availability
- **Real-time response**: Sub-millisecond critical system communication

### Industrial Control
- **Safety isolation**: Chemical plant incidents contained by physical zone boundaries
- **Control redundancy**: Control systems distributed across multiple physical locations
- **Maintenance zones**: Hot-swappable components without affecting distant systems
- **Emergency protocols**: Automatic isolation of damaged areas

## 📖 Usage

### Command Line Options

```bash
./ryx-node [options]

Basic Options:
  --port int         UDP port for node communication (default 9001)
  --http-port int    HTTP API port (default 8001)  
  --cluster-id str   Cluster identifier (default "default")
  --node-id str      Node identifier (auto-generated if empty)

Spatial Computing Options (Phase 3C):
  --coord-system str Coordinate system: gps, relative, logical, none (default "none")
  --x float         X coordinate (longitude for GPS, meters for relative)
  --y float         Y coordinate (latitude for GPS, meters for relative)  
  --z float         Z coordinate (altitude/height in meters)
  --zone str        Logical zone identifier (default "default")
  --barriers str    Comma-separated barriers (format: type:zoneA:zoneB:isolation)
```

### Example: 3-Node Spatial Cluster

```bash
# Terminal 1: Bridge node with spatial coordinates
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
  --barriers "bulkhead:bridge:engine_bay:fault" --port 9010 --http-port 8010

# Terminal 2: Another bridge node (same zone)  
./ryx-node --coord-system relative --x 16.8 --y -2.5 --z 3.2 --zone bridge \
  --barriers "bulkhead:bridge:engine_bay:fault" --port 9011 --http-port 8011

# Terminal 3: Engine bay node (different zone)
./ryx-node --coord-system relative --x 45.8 --y -8.5 --z 1.2 --zone engine_bay \
  --barriers "bulkhead:engine_bay:bridge:fault" --port 9012 --http-port 8012

# Check spatial cluster status with zone analysis
curl -s http://localhost:8010/spatial/neighbors | jq '.zone_analysis'
curl -s http://localhost:8010/status | jq '.spatial'
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