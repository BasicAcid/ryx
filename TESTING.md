# Ryx Phase 2A Testing Guide

## 🎯 Phase 2A Implementation Status

### ✅ **Completed Features**
1. **Information Diffusion Foundation** - Complete service architecture
2. **HTTP API Endpoints** - `/inject`, `/info`, `/info/{id}` with comprehensive logging
3. **Content-Addressable Storage** - SHA256-based message deduplication
4. **TTL-Based Cleanup** - Automatic memory management
5. **ryx-cluster Tool** - Complete cluster management and testing automation

### 🔧 **Architecture Improvements**
1. **Enhanced Error Handling** - Panic recovery and detailed logging in API handlers
2. **Nil Pointer Safety** - Comprehensive nil checks throughout the codebase
3. **Comprehensive Logging** - Step-by-step operation logging for debugging
4. **Clean Integration** - Proper service lifecycle management

## 🧠 Key Concepts

### **Content-Addressable Storage**
Ryx uses **SHA256-based content addressing** where each piece of information gets a unique ID based on its content:

```bash
# Same content = Same ID = Same storage slot
"Hello World" → ID: 9f86d081884c7d65
"Hello World" → ID: 9f86d081884c7d65  # Duplicate detected, not stored again

# Different content = Different ID = Different storage slot  
"Hello Network" → ID: a1b2c3d4e5f6789a
"Hello Universe" → ID: b2c3d4e5f6789ab1
```

**Why This Matters:**
- ✅ **Deduplication**: Prevents storing identical information multiple times
- ✅ **Memory Efficiency**: Same content uses same storage regardless of source
- ✅ **Data Integrity**: Content hash verifies data hasn't been corrupted
- ✅ **Loop Prevention**: Critical for Phase 2B inter-node diffusion

### **Expected Behavior Patterns**

#### **Unique Content (Creates New Messages)**
```bash
./ryx-cluster -cmd inject -content "Message A"     # Count: 1
./ryx-cluster -cmd inject -content "Message B"     # Count: 2  
./ryx-cluster -cmd inject -content "Message C"     # Count: 3
```

#### **Duplicate Content (No New Storage)**
```bash
./ryx-cluster -cmd inject -content "Hello"         # Count: 1 (new)
./ryx-cluster -cmd inject -content "Hello"         # Count: 1 (duplicate)
./ryx-cluster -cmd inject -content "Hello" -energy 10  # Count: 1 (still duplicate)
```

#### **Generating Unique Content for Testing**
```bash
# Using timestamps
./ryx-cluster -cmd inject -content "Log entry $(date +%s)"

# Using random numbers  
./ryx-cluster -cmd inject -content "Event $RANDOM"

# Using counters
for i in {1..5}; do 
  ./ryx-cluster -cmd inject -content "Message $i"
done
```

### **Information Message Structure**
Each stored message contains rich metadata:
```json
{
  "id": "9f86d081884c7d65",        // SHA256 hash of content (first 8 bytes)
  "type": "text",                  // Message type
  "content": "SGVsbG8gV29ybGQ=",    // Base64-encoded content  
  "energy": 5,                     // Propagation fuel (for Phase 2B)
  "ttl": 1757760123,              // Expiration timestamp
  "hops": 0,                       // Distance traveled (for Phase 2B)
  "source": "node_abc123",         // Originating node
  "path": ["node_abc123"],         // Nodes visited (for loop prevention)
  "timestamp": 1757759823,         // Creation time
  "metadata": {}                   // Extensible data
}
```

## 🧪 Testing Protocol

### **Test 1: Build Verification**
```bash
cd /home/david/Workspace/ryx

# Build both binaries
go build -o ryx-node ./cmd/ryx-node
go build -o ryx-cluster ./cmd/ryx-cluster

# Verify binaries exist
ls -la ryx-node ryx-cluster

# Test help output
./ryx-cluster -cmd help
```

### **Test 2: Single Node Information Storage**
```bash
# Start single node
./ryx-node --port 9010 --http-port 8010 &

# Wait for startup
sleep 3

# Test basic API
curl http://localhost:8010/ping
curl http://localhost:8010/status

# Test information injection
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello World","energy":5}'

# Verify information is stored
curl http://localhost:8010/info

# Stop node
pkill ryx-node
```

### **Test 3: Automated Cluster Testing**
```bash
# Start 3-node cluster using ryx-cluster tool
./ryx-cluster -cmd start -nodes 3

# Check cluster status
./ryx-cluster -cmd status

# Inject information
./ryx-cluster -cmd inject -content "Test Diffusion" -energy 5

# Stop cluster
./ryx-cluster -cmd stop
```

### **Test 4: Multi-Node Neighbor Discovery**
```bash
# Start cluster
./ryx-cluster -cmd start -nodes 3

# Wait for neighbor discovery (10 seconds)
sleep 10

# Check each node's neighbors via HTTP API
curl http://localhost:8010/status | jq '.neighbors'
curl http://localhost:8011/status | jq '.neighbors'  
curl http://localhost:8012/status | jq '.neighbors'

# Clean up
./ryx-cluster -cmd stop
```

### **Test 5: Information Storage Persistence**
```bash
# Start cluster
./ryx-cluster -cmd start -nodes 3

# Inject multiple messages
curl -X POST http://localhost:8010/inject -d '{"content":"Message 1","energy":3}'
curl -X POST http://localhost:8011/inject -d '{"content":"Message 2","energy":4}'
curl -X POST http://localhost:8012/inject -d '{"content":"Message 3","energy":2}'

# Check storage on all nodes
curl http://localhost:8010/info | jq '.count'
curl http://localhost:8011/info | jq '.count'
curl http://localhost:8012/info | jq '.count'

# Clean up
./ryx-cluster -cmd stop
```

### **Test 6: Content-Addressable Storage Behavior**
```bash
# Start cluster
./ryx-cluster -cmd start -nodes 3

# Test 6a: Unique Content Creates New Messages
echo "=== Testing Unique Content ==="
./ryx-cluster -cmd inject -content "Message A" -node 0
./ryx-cluster -cmd inject -content "Message B" -node 0  
./ryx-cluster -cmd inject -content "Message C" -node 0
./ryx-cluster -cmd status
# Expected: Node 0 should have 3 messages

# Test 6b: Duplicate Content Detection
echo "=== Testing Duplicate Detection ==="
./ryx-cluster -cmd inject -content "Duplicate Test" -node 0
curl http://localhost:8010/info | jq '.count'  # Should be 4
./ryx-cluster -cmd inject -content "Duplicate Test" -node 0  # Same content
curl http://localhost:8010/info | jq '.count'  # Should still be 4
./ryx-cluster -cmd inject -content "Duplicate Test" -energy 10 -node 0  # Different energy, same content
curl http://localhost:8010/info | jq '.count'  # Should still be 4

# Test 6c: Verify Content IDs
echo "=== Testing Content Addressing ==="
CONTENT_ID=$(curl -s http://localhost:8010/info | jq -r '.info | keys[0]')
echo "Message ID: $CONTENT_ID"
curl http://localhost:8010/info/$CONTENT_ID | jq '.info.content' | base64 -d
# Should display the actual message content

# Test 6d: Generate Unique Test Data
echo "=== Testing Unique Data Generation ==="
for i in {1..3}; do
  ./ryx-cluster -cmd inject -content "Timestamp: $(date +%s.%N)" -node 0
  sleep 1
done
./ryx-cluster -cmd status
# Expected: Node 0 should have increased message count

# Clean up  
./ryx-cluster -cmd stop
```

### **Test 7: Phase 2B Inter-Node Diffusion**
```bash
# Start cluster and wait for neighbor discovery
./ryx-cluster -cmd start -nodes 3
sleep 8

# Inject message with energy on one node - should spread to all nodes
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"Phase 2B Test","energy":3}'

# Wait for propagation
sleep 3

# Verify message reached all nodes
echo "=== Node 0 (Original) ==="
curl -s http://localhost:8010/info | jq '.count'  # Should be 1

echo "=== Node 1 (Neighbor) ==="  
curl -s http://localhost:8011/info | jq '.count'  # Should be 1

echo "=== Node 2 (Neighbor) ==="
curl -s http://localhost:8012/info | jq '.count'  # Should be 1

# Check energy decay and path tracking
MESSAGE_ID=$(curl -s http://localhost:8010/info | jq -r '.info | keys[0]')

echo "=== Energy Decay Analysis ==="
echo "Original node:"
curl -s http://localhost:8010/info/$MESSAGE_ID | jq '{energy: .info.energy, hops: .info.hops, path: .info.path}'

echo "Forwarded nodes:"
curl -s http://localhost:8011/info/$MESSAGE_ID | jq '{energy: .info.energy, hops: .info.hops, path: .info.path}'
curl -s http://localhost:8012/info/$MESSAGE_ID | jq '{energy: .info.energy, hops: .info.hops, path: .info.path}'

# Expected results:
# Original: energy=3, hops=0, path=["node_X"] 
# Forwarded: energy=2, hops=1, path=["node_X", "node_X"]

# Clean up
./ryx-cluster -cmd stop
```

### **Test 8: Energy Exhaustion**
```bash
# Start cluster
./ryx-cluster -cmd start -nodes 3
sleep 8

# Inject message with low energy
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"Low Energy Test","energy":1}'

sleep 3

# Check propagation - should reach neighbors but not propagate further
MESSAGE_ID=$(curl -s http://localhost:8010/info | jq -r '.info | keys[0]')

echo "=== Energy Exhaustion Test ==="
echo "Original node (energy=1):"
curl -s http://localhost:8010/info/$MESSAGE_ID | jq '.info.energy'

echo "Neighbor nodes (energy=0, no further propagation):"
curl -s http://localhost:8011/info/$MESSAGE_ID | jq '.info.energy'
curl -s http://localhost:8012/info/$MESSAGE_ID | jq '.info.energy'

# Expected: All nodes have the message, but forwarded copies have energy=0

# Clean up
./ryx-cluster -cmd stop
```

### **Test 9: Multi-Node Content Distribution (Legacy)**
```bash
# Start cluster
./ryx-cluster -cmd start -nodes 3

# Inject same content into different nodes - Phase 2B will deduplicate
./ryx-cluster -cmd inject -content "Shared Message" -node 0
sleep 3  # Wait for diffusion

./ryx-cluster -cmd inject -content "Shared Message" -node 1
sleep 3  # This should be deduplicated

# Check message counts - all nodes should have 1 message (deduplicated)
echo "=== Deduplication Test ==="
curl http://localhost:8010/info | jq '.count'  # Should be 1
curl http://localhost:8011/info | jq '.count'  # Should be 1  
curl http://localhost:8012/info | jq '.count'  # Should be 1

# Verify all have same content ID
ID_0=$(curl -s http://localhost:8010/info | jq -r '.info | keys[0]')
ID_1=$(curl -s http://localhost:8011/info | jq -r '.info | keys[0]')
ID_2=$(curl -s http://localhost:8012/info | jq -r '.info | keys[0]')
echo "Node 0 ID: $ID_0"
echo "Node 1 ID: $ID_1" 
echo "Node 2 ID: $ID_2"
# All IDs should be identical

# Clean up
./ryx-cluster -cmd stop
```

## Expected Results

### **Phase 2B Diffusion Results**

#### Test 7: Basic Inter-Node Diffusion
- **Message Count**: All 3 nodes should have count=1 (message spread to all)
- **Energy Decay**: Original energy=3, forwarded energy=2  
- **Hop Tracking**: Original hops=0, forwarded hops=1
- **Path Tracking**: Original path=["node_A"], forwarded path=["node_A", "node_A"]

#### Test 8: Energy Exhaustion
- **Propagation**: Message reaches immediate neighbors only
- **Energy States**: Original=1, neighbors=0 (no further forwarding)
- **Network Coverage**: Limited by energy, demonstrates controlled propagation

#### Test 9: Network-Wide Deduplication
- **Behavior**: Second injection of same content ignored (already exists)
- **Consistency**: All nodes have identical message ID and content
- **Efficiency**: No duplicate storage or propagation

### **Successful Test Results**

#### **Test 1: Build**
- Both `ryx-node` and `ryx-cluster` binaries should build without errors
- Help output should show comprehensive usage information

#### **Test 2: Single Node**
```json
// /ping response
{"node_id":"node_abc123","pong":true,"timestamp":123456789}

// /status response  
{
  "node_id":"node_abc123",
  "cluster_id":"default", 
  "port":9010,
  "http_port":8010,
  "running":true,
  "neighbors":{},
  "diffusion":{"total_messages":0,"node_id":"node_abc123"}
}

// After injection - /info response
{
  "count":1,
  "info":{
    "9f86d081884c7d65":{
      "id":"9f86d081884c7d65",
      "type":"text",
      "content":"SGVsbG8gV29ybGQ=", // Base64: "Hello World"
      "energy":5,
      "ttl":123456789,
      "hops":0,
      "source":"node_abc123",
      "path":["node_abc123"],
      "timestamp":123456789,
      "metadata":{}
    }
  }
}
```

#### **Test 3: Cluster Tool**
```
🚀 Starting 3-node ryx cluster...
  Starting node 0: UDP:9010 HTTP:8010
  Starting node 1: UDP:9011 HTTP:8011  
  Starting node 2: UDP:9012 HTTP:8012
⏳ Waiting for nodes to start up...
✅ Started 3-node ryx cluster

📊 Cluster Status:
  Nodes: 3
  Cluster ID: test
  Port range: 9010-9012 (UDP), 8010-8012 (HTTP)
```

#### **Test 4: Neighbor Discovery**
- Each node should discover 2 neighbors (the other 2 nodes)
- `/status` should show `"neighbors": {...}` with 2 entries each

#### **Test 5: Information Storage**
- Each node should store exactly 1 message (the one injected into it)
- All nodes should show `"count": 1` in their `/info` response

#### **Test 6: Content-Addressable Storage**
- **Unique Content**: Each unique message increases the count
  - 3 different messages → count increases to 3
- **Duplicate Detection**: Same content doesn't create new storage
  - Same message injected multiple times → count stays the same
  - Different energy/TTL with same content → still considered duplicate
- **Content IDs**: Identical content produces identical SHA256 IDs
  - Content "Hello" → Always generates same ID (e.g., `9f86d081884c7d65`)
- **Unique Generation**: Timestamp-based content creates unique messages
  - Each timestamp injection → count increases

#### **Test 7: Multi-Node Content Distribution**
- **Independent Storage**: Each node stores messages independently (Phase 2A)
  - Same content injected into 3 nodes → each node has count: 1
- **Identical IDs**: Same content produces same ID across all nodes
  - All nodes should show identical message ID for identical content
- **Content Verification**: Base64 decoding shows original message
  - `echo "SGVsbG8=" | base64 -d` → `Hello`

## 🔧 Understanding Common Behaviors

### **"Why doesn't my message count increase?"**
**Likely Cause**: You're injecting the same content repeatedly.

**Solution**: Use unique content for testing:
```bash
# Instead of this (creates duplicates):
./ryx-cluster -cmd inject -content "test"  # count: 1
./ryx-cluster -cmd inject -content "test"  # count: still 1

# Do this (creates unique messages):
./ryx-cluster -cmd inject -content "test-$(date +%s)"  # count: 1
./ryx-cluster -cmd inject -content "test-$(date +%s)"  # count: 2
```

### **"Are my nodes communicating with each other?"**
**Phase 2A Status**: Nodes discover neighbors but don't share information yet.

**Current Behavior**:
- ✅ Neighbor discovery works (nodes find each other)  
- ✅ Information storage works (each node stores independently)
- ⏳ Inter-node diffusion comes in Phase 2B

**Verification**: Check neighbor counts in status - should show 2 neighbors per node in 3-node cluster.

## 🐛 Troubleshooting

### **Common Issues**

#### **Build Errors**
```bash
# If build fails, check Go module
go mod tidy
go build ./...
```

#### **Node Startup Issues**
```bash
# Check for port conflicts
netstat -tuln | grep :901
netstat -tuln | grep :801

# Use different ports if needed
./ryx-cluster -cmd start -base-port 9020 -base-http-port 8020
```

#### **API Connection Issues**  
```bash
# Verify node is listening
curl http://localhost:8010/ping
# Should return: {"node_id":"...","pong":true,"timestamp":...}

# Check logs for errors (nodes log to stdout)
```

#### **Information Injection Failures**
```bash
# Check detailed logs - nodes will show step-by-step injection logging
# Look for "handleInject:" prefixed log messages

# Verify JSON format
curl -X POST http://localhost:8010/inject \
  -H "Content-Type: application/json" \
  -d '{"content":"test message","energy":5,"ttl":300}' \
  -v
```

## 🎯 Phase 2A Success Criteria

### **Core Functionality ✅**
- [x] Information injection works without crashes
- [x] Content-addressable storage with SHA256 IDs  
- [x] TTL-based automatic cleanup
- [x] HTTP API endpoints function correctly
- [x] Multi-node cluster startup and management

### **Developer Experience ✅**  
- [x] `ryx-cluster` tool for easy testing
- [x] Comprehensive logging for debugging
- [x] Clear error messages and recovery
- [x] Automated cluster management

### **Foundation Ready ✅**
- [x] Service architecture supports future diffusion logic
- [x] Clean interfaces between components
- [x] Extensible message format
- [x] Scalable storage and cleanup mechanisms

## 🚀 Next Phase: Phase 2A+ (Future)

**To complete full information diffusion**, the next step would be to:

1. **Add Inter-Node Message Forwarding** - Connect diffusion service to communication service
2. **Implement Energy Decay Logic** - Messages lose energy as they hop between nodes  
3. **Add Loop Prevention** - Track message paths to prevent infinite cycles
4. **Enhance ryx-cluster** - Add real-time diffusion visualization

**Current implementation provides the complete foundation** for these features with robust error handling, comprehensive testing tools, and clean architecture.

---

**Status**: Phase 2A Foundation ✅ **COMPLETE** - Ready for testing and validation!