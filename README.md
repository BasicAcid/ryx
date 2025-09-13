# Ryx - Distributed Computing System

A practical implementation of Dave Ackley's robust-first distributed computing model, built in Go for production deployment.

## ✨ Features

- **Effortless scaling**: Add nodes without configuration changes
- **Automatic discovery**: Nodes find each other via UDP broadcast
- **Fault tolerance**: No single points of failure, graceful degradation
- **Single binary**: Deploy anywhere with zero dependencies
- **HTTP API**: Monitor and control via REST endpoints

## 🚀 Quick Start

### Build and Run

```bash
# Build the node binary
go build -o ryx-node ./cmd/ryx-node

# Start your first node
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

| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/status` | GET | Detailed node status | Node info + neighbors |
| `/health` | GET | Health check | Simple status |
| `/ping` | GET | Connectivity test | Pong response |

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

### 🚧 Phase 2: Computation Engine (Next)
- Energy-based task diffusion
- Distributed computation execution  
- Result aggregation

### 📋 Phase 3: Development Tools
- `ryx-cluster` for local testing
- Chaos engineering capabilities
- Performance benchmarking

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