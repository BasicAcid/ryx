# Agent Guidelines for Ryx Distributed Computing System

## Build/Lint/Test Commands
- **Build main binary**: `go build -o ryx-node ./cmd/ryx-node`
- **Build cluster tool**: `go build -o ryx-cluster ./cmd/ryx-cluster`
- **Run single node**: `./ryx-node --port 9010 --http-port 8010`
- **Test large cluster**: `./ryx-cluster -cmd start -profile huge` (50 nodes)
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

#### Spatial Testing Guidelines
- **Multi-zone testing**: Test fault isolation across different physical zones
- **Coordinate system validation**: Test all coordinate systems (GPS, relative, logical, none)
- **Distance-based behavior**: Verify neighbor selection respects physical proximity
- **Barrier respect**: Ensure physical barriers properly isolate network segments

#### Mission-Critical Spatial Scenarios
- **Spaceship compartment isolation**: Engine bay failure should not affect bridge systems
- **Vehicle fault containment**: Front-end collision should not disable rear systems
- **Data center rack isolation**: Power failure in one rack should not cascade to others
- **Smart city zone management**: Infrastructure failure should be contained to affected zones

#### Spatial API Design
- **RESTful spatial endpoints**: `/spatial/position`, `/spatial/neighbors`, `/spatial/topology`
- **Coordinate system awareness**: APIs should respect the node's coordinate system
- **Distance-based filtering**: Allow filtering by physical/logical distance
- **Zone-based operations**: Support operations scoped to physical/logical zones