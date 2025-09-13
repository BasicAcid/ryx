package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/api"
	"github.com/BasicAcid/ryx/internal/communication"
	"github.com/BasicAcid/ryx/internal/diffusion"
	"github.com/BasicAcid/ryx/internal/discovery"
)

// Config holds node configuration
type Config struct {
	Port      int
	HTTPPort  int
	ClusterID string
	NodeID    string
}

// Node represents a single ryx node
type Node struct {
	id        string
	config    *Config
	discovery *discovery.Service
	comm      *communication.Service
	diffusion *diffusion.Service
	api       *api.Server
	mu        sync.RWMutex
	running   bool
}

// New creates a new node instance
func New(config *Config) (*Node, error) {
	// Generate node ID if not provided
	nodeID := config.NodeID
	if nodeID == "" {
		nodeID = generateNodeID()
	}

	node := &Node{
		id:     nodeID,
		config: config,
	}

	// Initialize services
	var err error

	node.discovery, err = discovery.New(config.Port, config.ClusterID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	node.comm, err = communication.New(config.Port, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication service: %w", err)
	}

	// Initialize diffusion service
	node.diffusion = diffusion.New(nodeID)

	node.api, err = api.New(config.HTTPPort, node)
	if err != nil {
		return nil, fmt.Errorf("failed to create API server: %w", err)
	}

	return node, nil
}

// ID returns the node identifier
func (n *Node) ID() string {
	return n.id
}

// Start begins node operations
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("node already running")
	}

	log.Printf("Node %s starting services...", n.id)

	// Start communication service first
	if err := n.comm.Start(ctx); err != nil {
		return fmt.Errorf("failed to start communication: %w", err)
	}

	// Start discovery service
	if err := n.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}

	// Start diffusion service
	if err := n.diffusion.Start(ctx); err != nil {
		return fmt.Errorf("failed to start diffusion: %w", err)
	}

	// Start HTTP API server
	if err := n.api.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	n.running = true
	log.Printf("Node %s started successfully", n.id)

	return nil
}

// Stop gracefully shuts down the node
func (n *Node) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return
	}

	log.Printf("Node %s stopping services...", n.id)

	// Stop services in reverse order
	if n.api != nil {
		n.api.Stop()
	}
	if n.diffusion != nil {
		n.diffusion.Stop()
	}
	if n.discovery != nil {
		n.discovery.Stop()
	}
	if n.comm != nil {
		n.comm.Stop()
	}

	n.running = false
	log.Printf("Node %s stopped", n.id)
}

// GetStatus returns current node status
func (n *Node) GetStatus() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	status := map[string]interface{}{
		"node_id":    n.id,
		"cluster_id": n.config.ClusterID,
		"port":       n.config.Port,
		"http_port":  n.config.HTTPPort,
		"running":    n.running,
		"uptime":     time.Since(time.Now()).String(), // TODO: track actual uptime
	}

	// Add service-specific status
	if n.discovery != nil {
		status["neighbors"] = n.discovery.GetNeighbors()
	}

	if n.diffusion != nil {
		status["diffusion"] = n.diffusion.GetStats()
	}

	return status
}

// GetDiffusionService returns the diffusion service for API access
func (n *Node) GetDiffusionService() *diffusion.Service {
	return n.diffusion
}

// generateNodeID creates a random node identifier
func generateNodeID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "node_" + hex.EncodeToString(bytes)
}
