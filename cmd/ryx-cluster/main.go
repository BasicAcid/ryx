package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ClusterProfile defines predefined cluster configurations
type ClusterProfile struct {
	Name        string
	NodeCount   int
	BatchSize   int
	PortRange   [2]int // [start, end]
	Description string
}

// ClusterConfig holds the cluster configuration
type ClusterConfig struct {
	Nodes        int
	BasePort     int
	BaseHTTPPort int
	ClusterID    string
	NodeBinary   string
	PIDFile      string
	BatchSize    int
	Parallel     bool
	Profile      *ClusterProfile
}

// NodeInfo holds information about a running node
type NodeInfo struct {
	ID       string
	Port     int
	HTTPPort int
	PID      int
	Process  *exec.Cmd
}

// SerializableNodeInfo holds node info that can be JSON serialized
type SerializableNodeInfo struct {
	ID       string `json:"id"`
	Port     int    `json:"port"`
	HTTPPort int    `json:"http_port"`
	PID      int    `json:"pid"`
}

// Cluster manages multiple ryx-node instances
type Cluster struct {
	config  *ClusterConfig
	nodes   map[int]*NodeInfo
	nodesMx sync.RWMutex
	running bool
}

func main() {
	var (
		command      = flag.String("cmd", "help", "Command: start, stop, status, inject, help, chaos, benchmark")
		nodes        = flag.Int("nodes", 3, "Number of nodes to start")
		profile      = flag.String("profile", "", "Cluster profile: small, medium, large, huge")
		basePort     = flag.Int("base-port", 9010, "Base port for nodes")
		baseHTTPPort = flag.Int("base-http-port", 8010, "Base HTTP port for nodes")
		clusterID    = flag.String("cluster-id", "test", "Cluster identifier")
		content      = flag.String("content", "Hello Ryx Network", "Content to inject")
		energy       = flag.Int("energy", 5, "Energy for injected information")
		ttl          = flag.Int("ttl", 300, "TTL in seconds for injected information")
		nodeID       = flag.Int("node", 0, "Specific node ID (0-based) for injection")
		batchSize    = flag.Int("batch-size", 10, "Number of nodes to start in parallel")
		parallel     = flag.Bool("parallel", true, "Use parallel node operations")
	)
	flag.Parse()

	// Define cluster profiles
	profiles := map[string]*ClusterProfile{
		"small":  {Name: "small", NodeCount: 5, BatchSize: 3, PortRange: [2]int{9010, 9050}, Description: "Small cluster for basic testing"},
		"medium": {Name: "medium", NodeCount: 15, BatchSize: 5, PortRange: [2]int{9010, 9100}, Description: "Medium cluster for moderate testing"},
		"large":  {Name: "large", NodeCount: 30, BatchSize: 8, PortRange: [2]int{9010, 9200}, Description: "Large cluster for heavy testing"},
		"huge":   {Name: "huge", NodeCount: 50, BatchSize: 10, PortRange: [2]int{9010, 9300}, Description: "Huge cluster for maximum scale testing"},
	}

	// Apply profile if specified
	var selectedProfile *ClusterProfile
	if *profile != "" {
		if p, exists := profiles[*profile]; exists {
			selectedProfile = p
			*nodes = p.NodeCount
			*batchSize = p.BatchSize
			fmt.Printf("Using profile '%s': %s (%d nodes)\n", p.Name, p.Description, p.NodeCount)
		} else {
			log.Fatalf("Unknown profile: %s. Available profiles: small, medium, large, huge", *profile)
		}
	}

	config := &ClusterConfig{
		Nodes:        *nodes,
		BasePort:     *basePort,
		BaseHTTPPort: *baseHTTPPort,
		ClusterID:    *clusterID,
		NodeBinary:   "./ryx-node",
		PIDFile:      ".ryx-cluster.pids",
		BatchSize:    *batchSize,
		Parallel:     *parallel,
		Profile:      selectedProfile,
	}

	cluster := &Cluster{
		config: config,
		nodes:  make(map[int]*NodeInfo),
	}

	switch *command {
	case "start":
		err := cluster.Start()
		if err != nil {
			log.Fatalf("Failed to start cluster: %v", err)
		}
		fmt.Printf("Started %d-node ryx cluster\n", config.Nodes)
		cluster.PrintStatus()

	case "stop":
		err := cluster.Stop()
		if err != nil {
			log.Fatalf("Failed to stop cluster: %v", err)
		}
		fmt.Printf("Stopped ryx cluster\n")

	case "status":
		err := cluster.LoadFromPIDFile()
		if err != nil {
			log.Fatalf("Failed to load cluster info: %v", err)
		}
		cluster.PrintDetailedStatus()

	case "inject":
		err := cluster.LoadFromPIDFile()
		if err != nil {
			log.Fatalf("Failed to load cluster info: %v", err)
		}
		err = cluster.InjectInformation(*content, *energy, *ttl, *nodeID)
		if err != nil {
			log.Fatalf("Failed to inject information: %v", err)
		}
		fmt.Printf("Injected information into node %d\n", *nodeID)

	case "chaos":
		fmt.Printf("🌪️  Chaos engineering not yet implemented - coming in Phase 3A!\n")
		fmt.Printf("Planned features:\n")
		fmt.Printf("  - Node failures and restarts\n")
		fmt.Printf("  - Network partitions and delays\n")
		fmt.Printf("  - Resource constraints\n")
		fmt.Printf("Usage: ./ryx-cluster -cmd chaos -type node-kill -count 3\n")

	case "benchmark":
		fmt.Printf("Performance benchmarking not yet implemented - coming in Phase 3A!\n")
		fmt.Printf("Planned features:\n")
		fmt.Printf("  - Diffusion speed measurement\n")
		fmt.Printf("  - Computation throughput analysis\n")
		fmt.Printf("  - Network performance testing\n")
		fmt.Printf("Usage: ./ryx-cluster -cmd benchmark -type diffusion\n")

	case "help":
		cluster.PrintHelp()

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		cluster.PrintHelp()
		os.Exit(1)
	}
}

// Start launches all nodes in the cluster
func (c *Cluster) Start() error {
	// Check if cluster is already running
	c.nodesMx.RLock()
	nodeCount := len(c.nodes)
	c.nodesMx.RUnlock()
	if c.LoadFromPIDFile() == nil && nodeCount > 0 {
		return fmt.Errorf("cluster appears to be already running (found PID file)")
	}

	fmt.Printf("Starting %d-node ryx cluster", c.config.Nodes)
	if c.config.Profile != nil {
		fmt.Printf(" (profile: %s)", c.config.Profile.Name)
	}
	fmt.Printf("...\n")

	if c.config.Parallel && c.config.Nodes > 3 {
		return c.startNodesParallel()
	} else {
		return c.startNodesSequential()
	}

	c.running = true

	// Save PID file
	err := c.SavePIDFile()
	if err != nil {
		log.Printf("Warning: failed to save PID file: %v", err)
	}

	// Wait a moment for nodes to start up
	fmt.Printf("Waiting for nodes to start up...\n")
	time.Sleep(3 * time.Second)

	return nil
}

// startNodesSequential starts nodes one by one (original method)
func (c *Cluster) startNodesSequential() error {
	// Start each node
	for i := 0; i < c.config.Nodes; i++ {
		err := c.startSingleNode(i)
		if err != nil {
			return fmt.Errorf("failed to start node %d: %w", i, err)
		}
		// Brief pause between starts
		time.Sleep(100 * time.Millisecond)
	}

	c.running = true
	return c.saveAndWait()
}

// startNodesParallel starts nodes in parallel batches for faster large cluster startup
func (c *Cluster) startNodesParallel() error {
	batchSize := c.config.BatchSize
	fmt.Printf("  Using parallel startup with batch size: %d\n", batchSize)

	// Start nodes in parallel batches
	for batchStart := 0; batchStart < c.config.Nodes; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > c.config.Nodes {
			batchEnd = c.config.Nodes
		}

		fmt.Printf("  Starting batch %d-%d...\n", batchStart, batchEnd-1)

		// Start batch in parallel
		errChan := make(chan error, batchEnd-batchStart)
		for i := batchStart; i < batchEnd; i++ {
			go func(nodeID int) {
				errChan <- c.startSingleNode(nodeID)
			}(i)
		}

		// Wait for batch to complete
		for i := batchStart; i < batchEnd; i++ {
			if err := <-errChan; err != nil {
				return fmt.Errorf("failed to start node in batch: %w", err)
			}
		}

		// Brief pause between batches
		if batchEnd < c.config.Nodes {
			time.Sleep(200 * time.Millisecond)
		}
	}

	c.running = true
	return c.saveAndWait()
}

// startSingleNode starts a single node with the given ID
func (c *Cluster) startSingleNode(nodeID int) error {
	nodePort := c.config.BasePort + nodeID
	httpPort := c.config.BaseHTTPPort + nodeID

	fmt.Printf("  Starting node %d: UDP:%d HTTP:%d\n", nodeID, nodePort, httpPort)

	cmd := exec.Command(c.config.NodeBinary,
		"--port", strconv.Itoa(nodePort),
		"--http-port", strconv.Itoa(httpPort),
		"--cluster-id", c.config.ClusterID,
	)

	// Start the process
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start node %d: %w", nodeID, err)
	}

	nodeInfo := &NodeInfo{
		ID:       fmt.Sprintf("node_%d", nodeID),
		Port:     nodePort,
		HTTPPort: httpPort,
		PID:      cmd.Process.Pid,
		Process:  cmd,
	}

	c.nodesMx.Lock()
	c.nodes[nodeID] = nodeInfo
	c.nodesMx.Unlock()
	return nil
}

// saveAndWait saves PID file and waits for node startup
func (c *Cluster) saveAndWait() error {
	// Save PID file
	err := c.SavePIDFile()
	if err != nil {
		log.Printf("Warning: failed to save PID file: %v", err)
	}

	// Wait a moment for nodes to start up
	fmt.Printf("Waiting for nodes to start up...\n")
	startupTime := 3 * time.Second
	if c.config.Nodes > 20 {
		startupTime = 5 * time.Second // More time for large clusters
	}
	time.Sleep(startupTime)

	return nil
}

// Stop shuts down all nodes in the cluster
func (c *Cluster) Stop() error {
	// Load existing cluster info
	err := c.LoadFromPIDFile()
	if err != nil {
		return fmt.Errorf("no cluster found (PID file missing or invalid)")
	}

	c.nodesMx.RLock()
	nodeCount := len(c.nodes)
	nodesCopy := make(map[int]*NodeInfo)
	for k, v := range c.nodes {
		nodesCopy[k] = v
	}
	c.nodesMx.RUnlock()

	fmt.Printf("Stopping %d nodes...\n", nodeCount)

	// Stop each node
	for i, nodeInfo := range nodesCopy {
		fmt.Printf("  Stopping node %d (PID: %d)\n", i, nodeInfo.PID)

		if nodeInfo.Process != nil {
			// Try graceful shutdown first
			nodeInfo.Process.Process.Signal(syscall.SIGTERM)

			// Wait a moment for graceful shutdown
			done := make(chan error, 1)
			go func() {
				done <- nodeInfo.Process.Wait()
			}()

			select {
			case <-done:
				// Process exited gracefully
			case <-time.After(2 * time.Second):
				// Force kill if not graceful
				fmt.Printf("    Force killing node %d\n", i)
				nodeInfo.Process.Process.Kill()
			}
		} else {
			// Kill by PID if process not available
			process, err := os.FindProcess(nodeInfo.PID)
			if err == nil {
				process.Signal(syscall.SIGTERM)
			}
		}
	}

	// Remove PID file
	os.Remove(c.config.PIDFile)
	c.nodesMx.Lock()
	c.nodes = make(map[int]*NodeInfo)
	c.nodesMx.Unlock()
	c.running = false

	return nil
}

// PrintStatus shows basic cluster status
func (c *Cluster) PrintStatus() {
	fmt.Printf("\nCluster Status:\n")
	c.nodesMx.RLock()
	nodeCount := len(c.nodes)
	c.nodesMx.RUnlock()
	fmt.Printf("  Nodes: %d\n", nodeCount)
	fmt.Printf("  Cluster ID: %s\n", c.config.ClusterID)
	fmt.Printf("  Port range: %d-%d (UDP), %d-%d (HTTP)\n",
		c.config.BasePort, c.config.BasePort+c.config.Nodes-1,
		c.config.BaseHTTPPort, c.config.BaseHTTPPort+c.config.Nodes-1)

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  ./ryx-cluster -cmd status     # Detailed status\n")
	fmt.Printf("  ./ryx-cluster -cmd inject     # Inject test information\n")
	fmt.Printf("  ./ryx-cluster -cmd stop       # Stop cluster\n")
	fmt.Printf("\n")
}

// PrintDetailedStatus shows detailed status of all nodes
func (c *Cluster) PrintDetailedStatus() {
	fmt.Printf("\nDetailed Cluster Status:\n")

	c.nodesMx.RLock()
	nodeCount := len(c.nodes)
	nodesCopy := make(map[int]*NodeInfo)
	for k, v := range c.nodes {
		nodesCopy[k] = v
	}
	c.nodesMx.RUnlock()

	fmt.Printf("  Total nodes: %d\n", nodeCount)

	var totalNeighbors, totalMessages int

	for i, nodeInfo := range nodesCopy {
		fmt.Printf("\n  Node %d (PID: %d):\n", i, nodeInfo.PID)
		fmt.Printf("    UDP: %d, HTTP: %d\n", nodeInfo.Port, nodeInfo.HTTPPort)

		// Get node status via HTTP API
		status, err := c.getNodeStatus(nodeInfo.HTTPPort)
		if err != nil {
			fmt.Printf("    Status: ERROR - %v\n", err)
			continue
		}

		// Extract neighbor count
		neighbors := 0
		if neighborCount, ok := status["neighbor_count"].(float64); ok {
			neighbors = int(neighborCount)
			totalNeighbors += neighbors
		} else if neighborsData, ok := status["neighbors"].(map[string]interface{}); ok {
			// Fallback to map-based counting for backward compatibility
			neighbors = len(neighborsData)
			totalNeighbors += neighbors
		} else if neighborsArray, ok := status["neighbors"].([]interface{}); ok {
			// Handle array-based neighbors
			neighbors = len(neighborsArray)
			totalNeighbors += neighbors
		}

		// Extract diffusion info count
		messages := 0
		if diffusionData, ok := status["diffusion"].(map[string]interface{}); ok {
			if totalMsgs, ok := diffusionData["total_messages"].(float64); ok {
				messages = int(totalMsgs)
				totalMessages += messages
			}
		}

		fmt.Printf("    Status: RUNNING\n")
		fmt.Printf("    Neighbors: %d\n", neighbors)
		fmt.Printf("    Messages: %d\n", messages)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Average neighbors per node: %.1f\n", float64(totalNeighbors)/float64(nodeCount))
	fmt.Printf("  Total information messages: %d\n", totalMessages)
	fmt.Printf("\n")
}

// InjectInformation injects information into a specific node
func (c *Cluster) InjectInformation(content string, energy, ttl, targetNode int) error {
	c.nodesMx.RLock()
	nodeCount := len(c.nodes)
	if targetNode >= nodeCount {
		c.nodesMx.RUnlock()
		return fmt.Errorf("node %d does not exist (cluster has %d nodes)", targetNode, nodeCount)
	}
	nodeInfo := c.nodes[targetNode]
	c.nodesMx.RUnlock()

	fmt.Printf("Injecting information into node %d...\n", targetNode)
	fmt.Printf("  Content: %s\n", content)
	fmt.Printf("  Energy: %d\n", energy)
	fmt.Printf("  TTL: %d seconds\n", ttl)

	// Create injection request
	requestData := map[string]interface{}{
		"content": content,
		"energy":  energy,
		"ttl":     ttl,
	}

	requestJSON, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send injection request with timeout
	url := fmt.Sprintf("http://localhost:%d/inject", nodeInfo.HTTPPort)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second, // 5 second timeout for injection
	}

	resp, err := client.Post(url, "application/json", strings.NewReader(string(requestJSON)))
	if err != nil {
		return fmt.Errorf("failed to send injection request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("injection request failed with status: %s", resp.Status)
	}

	fmt.Printf("Information injected successfully!\n")

	// Wait a moment and show the diffusion progress
	fmt.Printf("Waiting for diffusion...\n")
	time.Sleep(2 * time.Second)

	fmt.Printf("\nDiffusion status:\n")

	c.nodesMx.RLock()
	nodesCopy2 := make(map[int]*NodeInfo)
	for k, v := range c.nodes {
		nodesCopy2[k] = v
	}
	c.nodesMx.RUnlock()

	for i, node := range nodesCopy2 {
		info, err := c.getNodeInfo(node.HTTPPort)
		if err != nil {
			fmt.Printf("  Node %d: ERROR getting info\n", i)
			continue
		}

		count := 0
		if infoData, ok := info["info"].(map[string]interface{}); ok {
			count = len(infoData)
		}

		if count > 0 {
			fmt.Printf("  Node %d: Has %d messages\n", i, count)
		} else {
			fmt.Printf("  Node %d: No messages yet\n", i)
		}
	}

	return nil
}

// getNodeStatus gets status from a node's HTTP API with timeout
func (c *Cluster) getNodeStatus(httpPort int) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://localhost:%d/status", httpPort)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second, // 3 second timeout
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	return status, err
}

// getNodeInfo gets info from a node's HTTP API with timeout
func (c *Cluster) getNodeInfo(httpPort int) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://localhost:%d/info", httpPort)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second, // 3 second timeout
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	return info, err
}

// SavePIDFile saves cluster information to PID file
func (c *Cluster) SavePIDFile() error {
	data := make(map[string]interface{})
	data["config"] = c.config

	// Convert NodeInfo to SerializableNodeInfo
	serializableNodes := make(map[string]SerializableNodeInfo)

	c.nodesMx.RLock()
	for i, node := range c.nodes {
		serializableNodes[fmt.Sprintf("%d", i)] = SerializableNodeInfo{
			ID:       node.ID,
			Port:     node.Port,
			HTTPPort: node.HTTPPort,
			PID:      node.PID,
		}
	}
	c.nodesMx.RUnlock()
	data["nodes"] = serializableNodes

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.config.PIDFile, jsonData, 0644)
}

// LoadFromPIDFile loads cluster information from PID file
func (c *Cluster) LoadFromPIDFile() error {
	data, err := os.ReadFile(c.config.PIDFile)
	if err != nil {
		return err
	}

	var clusterData map[string]interface{}
	err = json.Unmarshal(data, &clusterData)
	if err != nil {
		return err
	}

	// Load nodes from serializable format
	if nodesData, ok := clusterData["nodes"].(map[string]interface{}); ok {
		for key, nodeData := range nodesData {
			nodeIndex, _ := strconv.Atoi(key)
			if nodeMap, ok := nodeData.(map[string]interface{}); ok {
				nodeInfo := &NodeInfo{
					ID:       nodeMap["id"].(string),
					Port:     int(nodeMap["port"].(float64)),
					HTTPPort: int(nodeMap["http_port"].(float64)),
					PID:      int(nodeMap["pid"].(float64)),
					Process:  nil, // Process is not serialized, will be nil on load
				}
				c.nodesMx.Lock()
				c.nodes[nodeIndex] = nodeInfo
				c.nodesMx.Unlock()
			}
		}
	}

	return nil
}

// PrintHelp shows usage information
func (c *Cluster) PrintHelp() {
	fmt.Printf(`
ryx-cluster - Local cluster management for ryx distributed computing

COMMANDS:
  start      Start a local cluster
  stop       Stop the running cluster  
  status     Show detailed cluster status
  inject     Inject information into the cluster
  chaos      Chaos engineering (Phase 3A - coming soon)
  benchmark  Performance benchmarking (Phase 3A - coming soon)
  help       Show this help

FLAGS:
  -nodes N              Number of nodes (default: 3)
  -profile STRING       Cluster profile: small, medium, large, huge
  -batch-size N         Parallel startup batch size (default: 10)
  -parallel             Use parallel node operations (default: true)
  -base-port N          Base UDP port (default: 9010)  
  -base-http-port N     Base HTTP port (default: 8010)
  -cluster-id STRING    Cluster identifier (default: "test")
  -content STRING       Content to inject (default: "Hello Ryx Network")
  -energy N             Energy for diffusion (default: 5)
  -ttl N                TTL in seconds (default: 300)
  -node N               Target node for injection (default: 0)

EXAMPLES:

  BASIC USAGE:
  # Start a 5-node cluster
  ./ryx-cluster -cmd start -nodes 5

  # Use predefined cluster profiles (Phase 3A)
  ./ryx-cluster -cmd start -profile small    # 5 nodes, optimized for basic testing
  ./ryx-cluster -cmd start -profile medium   # 15 nodes, moderate resources
  ./ryx-cluster -cmd start -profile large    # 30 nodes, heavy testing
  ./ryx-cluster -cmd start -profile huge     # 50 nodes, maximum scale

  # Large cluster with custom parallel settings
  ./ryx-cluster -cmd start -nodes 25 -batch-size 8 -parallel

  # Show detailed status (neighbors, message counts)
  ./ryx-cluster -cmd status

  # Stop the cluster cleanly
  ./ryx-cluster -cmd stop

  INFORMATION INJECTION:
  # Inject unique content (creates new messages)
  ./ryx-cluster -cmd inject -content "Event A"
  ./ryx-cluster -cmd inject -content "Event B"  
  ./ryx-cluster -cmd inject -content "Event C"
  # Result: 3 messages stored (each has unique content)

  # Generate unique content with timestamps
  ./ryx-cluster -cmd inject -content "Log $(date +%s)"
  ./ryx-cluster -cmd inject -content "Log $(date +%s)" 
  # Result: 2 messages (timestamps make content unique)

  # Demonstrate deduplication (same content = same storage)
  ./ryx-cluster -cmd inject -content "Duplicate Test"
  ./ryx-cluster -cmd inject -content "Duplicate Test"
  # Result: 1 message (duplicate detected, not stored twice)

  # Inject into specific nodes
  ./ryx-cluster -cmd inject -node 0 -content "Node 0 data"
  ./ryx-cluster -cmd inject -node 1 -content "Node 1 data" 
  ./ryx-cluster -cmd inject -node 2 -content "Node 2 data"
  # Result: Each node stores its own message independently

UNDERSTANDING BEHAVIOR:
  Content-Addressable Storage: Same content = Same ID = Same storage slot
  • "Hello" → ID: abc123 (stored once)
  • "Hello" → ID: abc123 (duplicate, not stored again)
  • "World" → ID: def456 (different content, stored separately)

  Phase 2A Status: Nodes store information independently (no inter-node sharing yet)

QUICK WORKFLOW:
  1. Build: go build -o ryx-node ./cmd/ryx-node && go build -o ryx-cluster ./cmd/ryx-cluster
  2. Start: ./ryx-cluster -cmd start -nodes 3
  3. Test:  ./ryx-cluster -cmd inject -content "Event $(date +%s)"
  4. Check: ./ryx-cluster -cmd status
  5. Stop:  ./ryx-cluster -cmd stop

TESTING UNIQUE CONTENT:
  # Method 1: Timestamps
  for i in {1..3}; do ./ryx-cluster -cmd inject -content "Event $(date +%s)"; sleep 1; done

  # Method 2: Counters  
  for i in {1..3}; do ./ryx-cluster -cmd inject -content "Message $i"; done

  # Method 3: Random data
  ./ryx-cluster -cmd inject -content "Data $RANDOM"

`)
}
