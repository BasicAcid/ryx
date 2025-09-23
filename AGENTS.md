# Agent Guidelines for Ryx Cellular Automata Computing System

## Current System Status

**Ryx is transitioning from distributed computing to cellular automata computing** following Dave Ackley's robust computing vision. The system has excellent spatial substrate infrastructure that serves as the perfect foundation for CA computation.

### Foundation Infrastructure (Ready for CA)
- **Spatial substrate**: Multi-modal coordinate systems (GPS, relative, logical, none)
- **Zone-aware topology**: Neighbor discovery with spatial awareness
- **Physical fault isolation**: Barrier system for CA boundary conditions
- **Runtime parameters**: Configurable system behavior for CA rule adaptation
- **Large-scale clusters**: 50+ node spatial topology for CA grid networks

### CA Conversion Status
- **Message diffusion**: Currently disabled - will be replaced with CA pattern propagation
- **Chemistry engine**: Temporarily disabled - will become CA update rules
- **Computation service**: Will be replaced with emergent CA computation
- **Spatial APIs**: Will be simplified for CA monitoring only

## Current Development Phase: Cellular Automata Conversion

**Strategic Focus**: Convert existing spatial infrastructure into cellular automata computing substrate where computation emerges from local CA rules rather than explicit programming.

**CA Conversion Goals**:
- Replace message diffusion with CA pattern propagation between neighboring node grids
- Convert chemistry reaction rules into cellular automata update rules
- Implement cellular grids within each spatial node
- Enable emergent computation through CA evolution
- Maintain spatial fault isolation as CA boundary conditions

## Build/Lint/Test Commands
- **Build CA node**: `go build -o ryx-node ./cmd/ryx-node`
- **Build cluster tool**: `go build -o ryx-cluster ./cmd/ryx-cluster`
- **Run spatial CA node**: `./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --zone datacenter_a --port 9010 --http-port 8010`
- **Test spatial CA cluster**: `./ryx-cluster -cmd start -profile huge` (50 nodes with spatial CA substrate)
- **Monitor spatial status**: `curl -s localhost:8010/spatial/neighbors | jq '.zone_analysis'`
- **Check CA grid status**: `curl -s localhost:8010/status | jq '.spatial'` (CA grid monitoring to be added)
- **Test compilation**: `go build ./...`
- **Format code**: `go fmt ./...`
- **Run tests**: `go test ./...` (CA-specific tests to be added)
- **Check dependencies**: `go mod tidy && go mod verify`

## Code Style Guidelines

### Project Structure
- **cmd/**: Main applications (`ryx-node`, `ryx-cluster`)
- **internal/**: Private application code (node, discovery, communication, spatial, ca)
- **internal/ca/**: Cellular automata engine (to be added)
- **internal/spatial/**: Spatial substrate infrastructure (CA foundation)
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
- **Minimize emoji usage**: Avoid emojis in code output, logs, and documentation unless specifically requested
- **Use emojis sparingly**: Only use emojis when they add genuine value and the user explicitly requests them
- Use clear, descriptive text instead of visual symbols
- Prefer professional terminal output for production systems
- Use consistent formatting for status messages and progress indicators
- **Status indicators**: Use text-based markers like [DONE], [TODO], [TARGET], [VALIDATED] instead of emoji symbols

### Cellular Automata Guidelines

#### CA Design Principles
- **Local computation only**: CA rules operate on immediate cell neighbors
- **No global coordination**: Computation emerges from local interactions
- **Spatial locality**: CA patterns propagate through spatial substrate
- **Deterministic rules**: CA update functions must be predictable and repeatable
- **Bounded state space**: Cell states must have finite, well-defined domains

#### CA Implementation Patterns
```go
// Example CA cell and grid structures
type CACell struct {
    State    int       // Current cell state
    NextState int      // State for next update cycle
    X, Y     int       // Position in grid
    Energy   float64   // Energy level (inherited from chemistry)
}

type CAGrid struct {
    Width, Height int
    Cells        [][]CACell
    UpdateRules  []CARule
    Generation   int
}

type CARule struct {
    Pattern      [][]int    // Neighborhood pattern to match
    NewState     int        // State to transition to
    Probability  float64    // Rule application probability
}
```

### Large-Scale CA Cluster Guidelines
- **Race condition prevention**: Always use mutex protection for CA grid access
- **Synchronous updates**: Ensure all cells update simultaneously per generation
- **Performance testing**: Measure CA update rates and memory usage for large grids
- **Memory management**: Bounded CA grid sizes, efficient state representation
- **Error handling**: Graceful degradation when CA rules fail or produce invalid states

### Spatial CA Substrate Guidelines

#### Multi-Modal Coordinate Systems (CA Substrate)
- **Support flexible coordinate systems**: GPS, relative, logical, none - each becomes CA grid placement
- **Coordinate system selection**: Choose based on CA deployment environment
  - **GPS**: Fixed CA grids (research facilities, distributed installations)
  - **Relative**: Mobile CA grids (vehicles, ships, aircraft) - CA grids relative to platform center
  - **Logical**: Virtual CA grids (cloud deployments, logical zones)
  - **None**: Development/testing CA grids without spatial constraints

#### Spatial Configuration Patterns (CA Substrate)
```go
// Example CA substrate configurations
type SpatialConfig struct {
    CoordSystem string    // "gps", "relative", "logical", "none"
    X, Y, Z     *float64  // Physical/logical coordinates for CA grid placement
    Zone        string    // Logical zone identifier for CA region
    Barriers    []string  // Physical isolation boundaries (CA boundary conditions)
}

// CA-specific configuration
type CAConfig struct {
    GridWidth   int       // CA grid width (cells)
    GridHeight  int       // CA grid height (cells)
    UpdateRate  int       // CA generations per second
    Rules       []CARule  // Local CA update rules
    StateCount  int       // Number of possible cell states
}
```

#### Command Line Interface for CA Nodes
```bash
# GPS-based CA deployment (research facility)
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone sector_a \
  --ca-grid-size 32x32 --ca-update-rate 10

# Vehicle/ship CA deployment (relative coordinates)
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
  --ca-grid-size 16x16 --ca-update-rate 5

# Cloud CA deployment (logical zones)
./ryx-node --coord-system logical --zone us-east-1a \
  --ca-grid-size 64x64 --ca-update-rate 20

# Development CA grid (no spatial constraints)
./ryx-node --coord-system none --zone dev_cluster \
  --ca-grid-size 8x8 --ca-update-rate 1
```

#### CA Grid Discovery Implementation
- **Extend discovery messages**: Include CA grid information in neighbor announcements
- **CA grid topology**: Build network of connected CA grids through spatial discovery
- **Boundary exchange**: CA grids share edge states with spatially adjacent neighbors
- **CA network emergence**: Connected CA computation emerges through local grid interactions

#### Physical Fault Isolation for CA
- **Physical boundaries become CA boundaries**: Barriers define CA grid isolation
- **Zone-aware CA redundancy**: CA patterns replicated across physical zones
- **CA grid isolation**: Node removal/replacement preserves CA computation in distant grids
- **Emergency CA partitioning**: Damaged areas automatically isolated via barrier boundaries

#### Distance-Based CA Grid Neighbor Selection
- **CA grid connectivity**: Determine which CA grids can exchange boundary states
- **Zone-aware CA distribution**: Target 70% same-zone, 30% cross-zone CA connections
- **CA interaction distance by coordinate system**:
  - **GPS**: 0-1km = direct CA boundary exchange, 1-10km = pattern sync, >50km = isolated
  - **Relative**: 0-10m = direct CA coupling, 10-100m = loose coupling, >500m = isolated
  - **Logical**: Same zone = tight CA coupling, different zone = loose pattern exchange
- **CA coupling strength**: Distance determines frequency and type of CA state exchange

#### CA Testing Guidelines
- **Multi-zone CA testing**: Test CA pattern isolation across different physical zones
- **Coordinate system validation**: Test all coordinate systems with CA grids
- **CA grid connectivity**: Verify spatial distance determines CA coupling strength
- **CA boundary exchange**: Validate edge state synchronization between neighboring grids
- **Zone distribution**: Confirm 70/30 same-zone/cross-zone CA connections
- **Barrier respect**: Ensure physical barriers properly isolate CA grid networks
- **CA rule validation**: Test CA update rules produce expected pattern evolution

#### Mission-Critical CA Computing Scenarios
- **Spaceship compartment CA isolation**: Engine bay CA grid failure should not affect bridge CA computation
  ```bash
  # Bridge CA grid
  ./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge \
    --barriers "bulkhead:bridge:engine_bay:fault" --ca-grid-size 16x16
  
  # Engine bay CA grid
  ./ryx-node --coord-system relative --x 45.8 --y -8.5 --z 1.2 --zone engine_bay \
    --barriers "bulkhead:engine_bay:bridge:fault" --ca-grid-size 16x16
  ```

- **Vehicle CA fault containment**: Front-end collision should not disable rear CA systems
  ```bash
  # Front sensor CA grid
  ./ryx-node --coord-system relative --x 2.5 --y 0 --z 0.5 --zone front_sensors \
    --ca-grid-size 8x8
  
  # Rear system CA grid
  ./ryx-node --coord-system relative --x -2.0 --y 0 --z 0.5 --zone rear_systems \
    --ca-grid-size 8x8
  ```

- **Research facility CA infrastructure**: Geographic CA computation with fault isolation
  ```bash
  # Primary research CA grid
  ./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone lab_a \
    --ca-grid-size 64x64
  
  # Backup research CA grid (distant)
  ./ryx-node --coord-system gps --x 34.0522 --y -118.2437 --z 15.0 --zone lab_c \
    --ca-grid-size 64x64
  ```

#### CA Monitoring API Design (Minimal)
- **Read-only CA endpoints**:
  - `GET /ca/grid` - Current CA grid state and configuration
  - `GET /ca/stats` - CA update statistics (generations, patterns, performance)
  - `GET /spatial/position` - Node's spatial configuration (CA grid placement)
  - `GET /spatial/neighbors` - Connected CA grids with coupling strength
  - `GET /spatial/barriers` - Barrier configuration (CA boundary conditions)
- **CA metrics**: Grid size, update rate, generation count, pattern complexity
- **Spatial substrate**: Distance calculations for CA grid connectivity
- **No external control**: CA computation proceeds autonomously without API injection
- **Performance monitoring**: CA update rates, memory usage, pattern evolution tracking

#### CA Implementation Best Practices
- **Coordinate validation**: Always validate spatial coordinates for CA grid placement
- **CA grid bounds**: Ensure grid dimensions are reasonable and memory-bounded
- **Thread safety**: Protect CA grid access with appropriate mutex patterns
- **Synchronous updates**: All CA cells must update simultaneously per generation
- **Error handling**: Graceful degradation when CA rules fail or produce invalid states
- **Memory management**: Bounded CA grid sizes, efficient cell state representation
- **CA rule validation**: Ensure CA update rules are deterministic and well-defined
- **Performance optimization**: Efficient CA update algorithms, minimal memory allocation
- **Testing coverage**: Test CA rules, grid connectivity, and spatial substrate integration

## Quick Reference: CA Computing Commands

### CA Node Deployment Examples
```bash
# Research facility CA grid with spatial placement
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --z 10.5 --zone lab_a \
  --ca-grid-size 32x32 --ca-update-rate 10 --port 9010 --http-port 8010

# Vehicle CA grid (relative coordinates)
./ryx-node --coord-system relative --x 2.5 --y 0 --z 0.5 --zone front_sensors \
  --ca-grid-size 16x16 --ca-update-rate 5 --port 9012 --http-port 8012

# Cloud CA deployment (logical zones)
./ryx-node --coord-system logical --zone us-east-1a \
  --ca-grid-size 64x64 --ca-update-rate 20 --port 9013 --http-port 8013

# Development CA grid (no spatial constraints)
./ryx-node --coord-system none --zone dev_cluster \
  --ca-grid-size 8x8 --ca-update-rate 1 --port 9014 --http-port 8014
```

### CA Monitoring Examples
```bash
# Check CA grid status and configuration
curl -s localhost:8010/ca/grid | jq '.'

# Monitor CA update statistics
curl -s localhost:8010/ca/stats | jq '.'

# Check spatial substrate (CA grid placement)
curl -s localhost:8010/spatial/position | jq '.'

# Analyze connected CA grids and coupling strength
curl -s localhost:8010/spatial/neighbors | jq '.zone_analysis'

# Check barrier configuration (CA boundary conditions)
curl -s localhost:8010/spatial/barriers | jq '.'

# Monitor overall node status with CA information
curl -s localhost:8010/status | jq '.ca'
```

### CA Validation Commands
```bash
# Verify CA grid is running and updating
GENERATION=$(curl -s localhost:8010/ca/stats | jq -r '.generation')
sleep 1
NEW_GENERATION=$(curl -s localhost:8010/ca/stats | jq -r '.generation')
echo "CA generations: $GENERATION -> $NEW_GENERATION (should increase)"

# Verify spatial CA grid connectivity
CONNECTED_GRIDS=$(curl -s localhost:8010/spatial/neighbors | jq -r '.zone_analysis.same_zone_count')
echo "Connected CA grids: $CONNECTED_GRIDS (target 70/30 zone distribution)"

# Check CA performance metrics
UPDATE_RATE=$(curl -s localhost:8010/ca/stats | jq -r '.updates_per_second')
echo "CA update rate: $UPDATE_RATE updates/sec"
```