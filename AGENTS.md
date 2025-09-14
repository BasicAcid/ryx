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