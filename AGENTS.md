# Agent Guidelines for Ryx Distributed Computing System

## Current System Status

**Ryx is now a mission-critical spatial-physical distributed computing system** with the following operational capabilities:

### Core Capabilities (Production Ready)
- **Spatial-aware distributed computing**: Multi-modal coordinate systems (GPS, relative, logical, none)
- **Hybrid neighbor selection**: 60% network performance + 40% spatial factors  
- **Zone-aware topology**: 70% same-zone, 30% cross-zone neighbors for optimal redundancy
- **Physical fault isolation**: Barrier-aware routing with compartment/zone isolation
- **Autonomous intelligence**: Runtime behavior adaptation based on network conditions and spatial factors
- **Large-scale operation**: 50+ node clusters with spatial awareness and race-condition-free operations
- **Mission-critical APIs**: Complete spatial neighbor analysis, distance calculation, and barrier management

### Deployment Scenarios
- **🚀 Spaceship core systems**: Compartment isolation (bridge/engine bay) with bulkhead barriers
- **🏙️ Smart city infrastructure**: GPS-based geographic optimization with fault isolation  
- **🚗 Vehicle systems**: Relative coordinates for fault containment (front/rear isolation)
- **☁️ Cloud deployments**: Logical zone awareness for availability regions
- **🏭 Industrial control**: Physical zone isolation for safety-critical systems

## Build/Lint/Test Commands
- **Build main binary**: `go build -o ryx-node ./cmd/ryx-node`
- **Build cluster tool**: `go build -o ryx-cluster ./cmd/ryx-cluster`
- **Run spatial node**: `./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --zone datacenter_a --port 9010 --http-port 8010`
- **Test spatial cluster**: `./ryx-cluster -cmd start -profile huge` (50 nodes with spatial awareness)
- **Test compilation**: `go build ./...`
- **Format code**: `go fmt ./...`
- **Run tests**: `go test ./...` (when tests are added)
- **Check dependencies**: `go mod tidy && go mod verify`

## Code Style Guidelines

### Project Structure
- **cmd/**: Main applications (`ryx-node`, `ryx-cluster`)
- **internal/**: Private application code (node, discovery, communication, api)
- **pkg/**: Public library code (protocol definitions)
- Use Go standard project layout

### Imports
- Standard library first, then third-party, then local
- Group imports with blank lines: stdlib, external, internal
- Use absolute imports: `github.com/BasicAcid/ryx/internal/node`

### Formatting & Style
- Use `go fmt` for consistent formatting
- Use `gofmt -s` for simplifications
- Line length: ~100 characters (Go convention)
- Use `goimports` for import management

### Types & Naming
- Use Go naming conventions: `CamelCase` for exported, `camelCase` for unexported
- Interface names: `ServiceProvider`, `StatusProvider`
- Struct names: `Node`, `Service`, `Config`
- Constants: `const MaxRetries = 3`
- Use descriptive names: `nodeID`, `clusterID`, `discoveryPort`

### Error Handling
- Always handle errors explicitly: `if err != nil { return err }`
- Wrap errors with context: `fmt.Errorf("failed to start node: %w", err)`
- Use standard error types where appropriate
- Log errors with structured context: `log.Printf("Node %s failed: %v", nodeID, err)`

### Concurrency
- Use goroutines for concurrent operations: `go s.messageLoop()`
- Use contexts for cancellation: `context.WithCancel(ctx)`
- **Always protect shared state with mutexes**: `sync.RWMutex` for read-heavy, `sync.Mutex` for mixed access
- Use channels for communication between goroutines
- Handle context cancellation in loops: `select { case <-ctx.Done(): return }`
- **Critical**: Maps are not thread-safe in Go - always use mutex protection for concurrent map access

### Network Programming
- Set appropriate timeouts on network operations
- Use UDP for node-to-node communication (fast, fault-tolerant)
- Use HTTP for control APIs (debuggable, standard)
- Handle network errors gracefully with retries
- Use JSON for message serialization (human-readable)

### Configuration
- Use flag package for CLI arguments
- Provide sensible defaults for all options
- Auto-generate IDs when not provided
- Validate configuration at startup

### Logging
- Use standard log package with structured messages
- Include node ID in log context: `log.Printf("Node %s: message", nodeID)`
- Log important state changes and network events
- Use appropriate log levels (when structured logging is added)

### Testing (Future)
- Test network behavior with realistic scenarios
- Mock network interfaces for unit tests
- Test failure scenarios (node deaths, network partitions)
- Use table-driven tests for multiple scenarios
- Test both success and failure paths

### UI/UX Guidelines
- **Minimize emoji usage**: Avoid emojis in code output, logs, and documentation
- Use clear, descriptive text instead of visual symbols
- Prefer professional terminal output for production systems
- Use consistent formatting for status messages and progress indicators

### Large-Scale Cluster Guidelines (Phase 3A+)
- **Race condition prevention**: Always use mutex protection for shared data structures
- **Parallel operations**: Batch concurrent operations to avoid resource exhaustion
- **Performance testing**: Measure startup times and resource usage for clusters 30+ nodes
- **Memory management**: Monitor memory usage during large cluster operations
- **Error handling**: Provide clear error messages for resource constraints and failures

### Spatial Computing Guidelines (Phase 3C+)

#### Multi-Modal Coordinate Systems
- **Support flexible coordinate systems**: GPS, relative, logical, none
- **Coordinate system selection**: Choose based on deployment environment
  - **GPS**: Fixed infrastructure (farms, data centers, smart cities)
  - **Relative**: Mobile platforms (ships, aircraft, vehicles) - coordinates relative to platform center
  - **Logical**: Cloud/virtual deployments (availability zones, racks)
  - **None**: Development/testing environments without spatial requirements

#### Spatial Configuration Patterns
```go
// Example coordinate system configurations
type SpatialConfig struct {
    CoordSystem string    // "gps", "relative", "logical", "none"
    X, Y, Z     *float64  // Physical/logical coordinates (nil if not applicable)
    Zone        string    // Logical zone identifier
    Barriers    []string  // Physical isolation boundaries
}
```

#### Command Line Interface for Spatial Nodes
```bash
# GPS-based deployment (smart city/farm)
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone sector_a

# Vehicle/ship deployment (relative coordinates)
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge

# Cloud deployment (logical zones)
./ryx-node --coord-system logical --zone us-east-1a --rack 42

# Development (no spatial awareness)
./ryx-node --coord-system none --zone dev_cluster
```

#### Spatial Discovery Implementation
- **Extend discovery messages**: Include coordinate information in neighbor announcements
- **Backward compatibility**: Nodes without coordinates continue working normally
- **Distance calculation**: Implement coordinate-system-aware distance functions
- **Physical topology emergence**: Spatial awareness builds through local neighbor interactions

#### Physical Fault Isolation Principles
- **Physical boundaries matter**: Network topology ≠ Physical topology for fault isolation
- **Zone-aware redundancy**: Critical data must be replicated across physical zones
- **Maintenance isolation**: Node removal/replacement should not affect physically distant nodes
- **Emergency partitioning**: System must isolate damaged physical areas automatically

#### Distance-Based Neighbor Selection (Phase 3C.2)
- **Hybrid scoring algorithm**: 60% network performance + 40% spatial factors
- **Zone-aware distribution**: Target 70% same-zone, 30% cross-zone neighbors for redundancy
- **Distance scoring by coordinate system**:
  - **GPS**: 0-1km = 1.0, 1-10km = 0.5-1.0, >50km = 0.0
  - **Relative**: 0-10m = 1.0, 10-100m = 0.5-1.0, >500m = 0.0  
  - **Logical**: Same zone = 1.0, different zone = 0.2
- **Spatial scoring components**: Zone affinity (+30%), distance penalty, system compatibility (+10%)

#### Spatial Testing Guidelines
- **Multi-zone testing**: Test fault isolation across different physical zones
- **Coordinate system validation**: Test all coordinate systems (GPS, relative, logical, none)
- **Distance calculation accuracy**: Verify GPS Haversine (~2.7km NYC test), Euclidean for relative
- **Hybrid neighbor selection**: Validate 60/40 network/spatial scoring split
- **Zone distribution**: Confirm 70/30 same-zone/cross-zone target ratios
- **Barrier respect**: Ensure physical barriers properly isolate network segments
- **Backward compatibility**: Non-spatial nodes must continue working unchanged

#### Mission-Critical Spatial Scenarios
- **Spaceship compartment isolation**: Engine bay failure should not affect bridge systems
  ```bash
  # Bridge nodes
  ./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
    --barriers "bulkhead:bridge:engine_bay:fault"
  
  # Engine bay nodes  
  ./ryx-node --coord-system relative --x 45.8 --y -8.5 --z 1.2 --zone engine_bay \
    --barriers "bulkhead:engine_bay:bridge:fault"
  ```

- **Vehicle fault containment**: Front-end collision should not disable rear systems
  ```bash
  # Front sensors
  ./ryx-node --coord-system relative --x 2.5 --y 0 --z 0.5 --zone front_sensors
  
  # Rear systems  
  ./ryx-node --coord-system relative --x -2.0 --y 0 --z 0.5 --zone rear_systems
  ```

- **Smart city infrastructure**: Geographic optimization with fault isolation
  ```bash
  # NYC data center
  ./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone datacenter_a
  
  # LA data center (distant backup)
  ./ryx-node --coord-system gps --x 34.0522 --y -118.2437 --z 15.0 --zone datacenter_c
  ```

#### Spatial API Design
- **RESTful spatial endpoints**: 
  - `GET /spatial/position` - Node's spatial configuration
  - `POST /spatial/position` - Update spatial configuration at runtime
  - `GET /spatial/neighbors` - Neighbors with distance and zone analysis
  - `GET /spatial/barriers` - Barrier configuration and status
  - `POST /spatial/distance` - Calculate distance to specified coordinates
- **Zone analysis data**: Same-zone/cross-zone counts, ratios, and target distributions
- **Distance calculation**: Coordinate-system-aware with proper units (meters, logical)
- **Barrier awareness**: Message-type routing (critical vs routine vs emergency)
- **Performance metrics**: Neighbor scoring, distance calculations, zone distribution

#### Implementation Best Practices
- **Coordinate validation**: Always validate coordinate bounds (GPS lat/lon, relative sanity checks)
- **Distance calculation optimization**: Cache results when appropriate, use efficient algorithms
- **Thread safety**: Protect spatial data structures with appropriate mutex patterns
- **Error handling**: Graceful degradation when spatial data unavailable or invalid
- **Memory management**: Bounded spatial data storage, automatic cleanup of stale neighbor data
- **API consistency**: Consistent JSON structure across all spatial endpoints
- **Testing coverage**: Test all coordinate systems, distance calculations, and neighbor selection scenarios

## Quick Reference: Spatial Commands

### Node Deployment Examples
```bash
# Spaceship bridge node with compartment isolation
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
  --barriers "bulkhead:bridge:engine_bay:fault" --port 9010 --http-port 8010

# Smart city GPS node
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone datacenter_a \
  --port 9011 --http-port 8011

# Vehicle system (relative to vehicle center)
./ryx-node --coord-system relative --x 2.5 --y 0 --z 0.5 --zone front_sensors \
  --port 9012 --http-port 8012

# Cloud deployment (logical zones)
./ryx-node --coord-system logical --zone us-east-1a --port 9013 --http-port 8013

# Development/testing (no spatial awareness)
./ryx-node --coord-system none --zone dev_cluster --port 9014 --http-port 8014
```

### API Testing Examples
```bash
# Check node's spatial configuration
curl -s localhost:8010/spatial/position | jq '.'

# Analyze spatial neighbors and zone distribution  
curl -s localhost:8010/spatial/neighbors | jq '.zone_analysis'

# Test GPS distance calculation (NYC to Times Square ~2.7km)
curl -X POST localhost:8010/spatial/distance -H "Content-Type: application/json" -d '{
  "coord_system": "gps", "x": 40.758, "y": -73.985, "zone": "times_square"
}' | jq '.distance'

# Check barrier configuration
curl -s localhost:8010/spatial/barriers | jq '.'

# Monitor node status with spatial information
curl -s localhost:8010/status | jq '.spatial'
```

### Validation Commands
```bash
# Test distance calculation accuracy
DISTANCE=$(curl -s -X POST localhost:8010/spatial/distance -H "Content-Type: application/json" -d '{
  "coord_system": "gps", "x": 40.758, "y": -73.985, "zone": "test"
}' | jq -r '.distance.value')
echo "Calculated distance: ${DISTANCE}m (expected ~2715m)"

# Verify zone-aware neighbor selection
SAME_ZONE=$(curl -s localhost:8010/spatial/neighbors | jq -r '.zone_analysis.same_zone_count')
CROSS_ZONE=$(curl -s localhost:8010/spatial/neighbors | jq -r '.zone_analysis.cross_zone_count')
echo "Same zone: $SAME_ZONE, Cross zone: $CROSS_ZONE (target 70/30 ratio)"
```