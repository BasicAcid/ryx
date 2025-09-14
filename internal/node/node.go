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
	"github.com/BasicAcid/ryx/internal/computation"
	"github.com/BasicAcid/ryx/internal/config"
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
	id          string
	config      *Config
	discovery   *discovery.Service
	comm        *communication.Service
	diffusion   *diffusion.Service
	computation *computation.Service
	api         *api.Server

	// Self-modification components
	runtimeParams *config.RuntimeParameters
	behaviorMod   config.BehaviorModifier

	mu      sync.RWMutex
	running bool
}

// New creates a new node instance
func New(cfg *Config) (*Node, error) {
	// Generate node ID if not provided
	nodeID := cfg.NodeID
	if nodeID == "" {
		nodeID = generateNodeID()
	}

	// Initialize runtime parameters and behavior modifier
	runtimeParams := config.GetDefaults()
	behaviorMod := config.NewAdaptiveBehaviorModifier(runtimeParams)

	node := &Node{
		id:            nodeID,
		config:        cfg,
		runtimeParams: runtimeParams,
		behaviorMod:   behaviorMod,
	}

	// Initialize services
	var err error

	node.discovery, err = discovery.New(cfg.Port, cfg.ClusterID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	node.comm, err = communication.New(cfg.Port, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication service: %w", err)
	}

	// Initialize diffusion service with configuration
	node.diffusion = diffusion.NewWithConfig(nodeID, runtimeParams, behaviorMod)

	// Initialize computation service
	node.computation = computation.New(nodeID)

	node.api, err = api.New(cfg.HTTPPort, node)
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

	// Start computation service
	if err := n.computation.Start(ctx); err != nil {
		return fmt.Errorf("failed to start computation: %w", err)
	}

	// Wire up service dependencies for Phase 2B inter-node diffusion
	n.diffusion.SetCommunication(n.comm)
	n.diffusion.SetDiscovery(n.discovery)
	n.comm.SetDiffusionService(n.diffusion)

	// Wire up Phase 2C computation integration
	n.diffusion.SetComputationService(n.computation)
	n.computation.SetDiffusionService(n.diffusion)

	log.Printf("Node %s: Phase 2B diffusion and Phase 2C computation services wired up", n.id)

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
	if n.computation != nil {
		n.computation.Stop()
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

	if n.computation != nil {
		status["computation"] = n.computation.GetComputationStats()
	}

	return status
}

// GetDiffusionService returns the diffusion service for API access
func (n *Node) GetDiffusionService() *diffusion.Service {
	return n.diffusion
}

// GetComputationService returns the computation service for API access
func (n *Node) GetComputationService() *computation.Service {
	return n.computation
}

// GetRuntimeParameters returns the runtime parameters for API access
func (n *Node) GetRuntimeParameters() *config.RuntimeParameters {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.runtimeParams
}

// GetBehaviorModifier returns the behavior modifier for API access
func (n *Node) GetBehaviorModifier() config.BehaviorModifier {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.behaviorMod
}

// UpdateParameters updates multiple runtime parameters atomically
func (n *Node) UpdateParameters(updates map[string]interface{}) map[string]bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.runtimeParams == nil {
		// Return all failures if parameters not available
		results := make(map[string]bool)
		for param := range updates {
			results[param] = false
		}
		return results
	}

	return n.runtimeParams.UpdateBatch(updates)
}

// generateNodeID creates a random node identifier
func generateNodeID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "node_" + hex.EncodeToString(bytes)
}
