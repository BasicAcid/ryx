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
	"github.com/BasicAcid/ryx/internal/spatial"
	"github.com/BasicAcid/ryx/internal/topology"
)

// Config holds node configuration
type Config struct {
	Port      int
	HTTPPort  int
	ClusterID string
	NodeID    string

	// Phase 3C.1: Spatial configuration
	SpatialConfig *spatial.SpatialConfig
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

	// Phase 3C.1: Spatial awareness
	barrierManager *spatial.BarrierManager

	// Phase 3C.3a: Topology mapping
	topologyMapper *topology.TopologyMapper

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

	// Phase 3C.1: Initialize spatial configuration with defaults if not provided
	spatialConfig := cfg.SpatialConfig
	if spatialConfig == nil {
		// Default to no spatial awareness for backward compatibility
		spatialConfig = &spatial.SpatialConfig{
			CoordSystem: spatial.CoordSystemNone,
			Zone:        "default",
		}
	}

	// Initialize barrier manager and load barriers from config
	barrierManager := spatial.NewBarrierManager()
	barrierManager.LoadBarriersFromConfig(spatialConfig)

	node := &Node{
		id:             nodeID,
		config:         cfg,
		runtimeParams:  runtimeParams,
		behaviorMod:    behaviorMod,
		barrierManager: barrierManager,
	}

	// Phase 3B: Initialize services with advanced behavior modification
	var err error

	// Phase 3C.2: Initialize discovery service with spatial awareness
	node.discovery, err = discovery.NewWithSpatialConfig(cfg.Port, cfg.ClusterID, nodeID, runtimeParams, behaviorMod, spatialConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	// Initialize communication service with fault pattern learning
	node.comm, err = communication.NewWithConfig(cfg.Port, nodeID, behaviorMod)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication service: %w", err)
	}

	// Initialize diffusion service with network-aware adaptation
	node.diffusion = diffusion.NewWithConfig(nodeID, runtimeParams, behaviorMod)

	// Initialize computation service with load-based optimization
	node.computation = computation.NewWithConfig(nodeID, runtimeParams, behaviorMod)

	// Phase 3C.3a: Initialize topology mapper
	node.topologyMapper = topology.NewTopologyMapper(node)

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

// GetStatus returns current node status (lock-free to prevent deadlocks)
func (n *Node) GetStatus() map[string]interface{} {
	// Build status without holding node mutex to prevent deadlocks
	// If we can respond to HTTP requests, the node is effectively "running"
	status := map[string]interface{}{
		"node_id":    n.id,                            // immutable after creation
		"cluster_id": n.config.ClusterID,              // immutable after creation
		"port":       n.config.Port,                   // immutable after creation
		"http_port":  n.config.HTTPPort,               // immutable after creation
		"running":    true,                            // if we can respond, we're running
		"uptime":     time.Since(time.Now()).String(), // TODO: track actual uptime
	}

	// Add service-specific status (each service has its own mutex protection)
	if n.discovery != nil {
		neighbors := n.discovery.GetNeighbors()
		if neighbors != nil {
			status["neighbors"] = neighbors
		} else {
			status["neighbors"] = []interface{}{}
		}
	} else {
		status["neighbors"] = []interface{}{}
	}

	if n.diffusion != nil {
		if diffusionStats := n.diffusion.GetStats(); diffusionStats != nil {
			status["diffusion"] = diffusionStats
		}
	}

	if n.computation != nil {
		if computationStats := n.computation.GetComputationStats(); computationStats != nil {
			status["computation"] = computationStats
		}
	}

	// Phase 3C.1: Add spatial configuration to status
	spatialConfig := n.GetSpatialConfig()
	if spatialConfig != nil && !spatialConfig.IsEmpty() {
		status["spatial"] = map[string]interface{}{
			"coord_system": spatialConfig.CoordSystem,
			"x":            spatialConfig.X,
			"y":            spatialConfig.Y,
			"z":            spatialConfig.Z,
			"zone":         spatialConfig.Zone,
			"barriers":     spatialConfig.Barriers,
			"has_coords":   spatialConfig.HasCoordinates(),
		}

		// Add barrier manager status
		if n.barrierManager != nil {
			barriers := n.barrierManager.GetAllBarriers()
			status["barriers_count"] = len(barriers)
		} else {
			status["barriers_count"] = 0
		}
	} else {
		status["spatial"] = map[string]interface{}{
			"coord_system": "none",
			"x":            nil,
			"y":            nil,
			"z":            nil,
			"zone":         "default",
			"barriers":     nil,
			"has_coords":   false,
		}
		status["barriers_count"] = 0
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

// Phase 3C.1: Spatial configuration methods

// GetSpatialConfig returns the node's spatial configuration
func (n *Node) GetSpatialConfig() *spatial.SpatialConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.config.SpatialConfig != nil {
		return n.config.SpatialConfig
	}

	// Return default spatial config for backward compatibility
	return &spatial.SpatialConfig{
		CoordSystem: spatial.CoordSystemNone,
		Zone:        "default",
	}
}

// UpdateSpatialConfig updates the node's spatial configuration
func (n *Node) UpdateSpatialConfig(newConfig *spatial.SpatialConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Validate the new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid spatial configuration: %w", err)
	}

	n.config.SpatialConfig = newConfig

	// Reload barriers from the new configuration
	n.barrierManager.LoadBarriersFromConfig(newConfig)

	log.Printf("Node %s spatial configuration updated: %s", n.id, newConfig.String())
	return nil
}

// GetBarrierManager returns the barrier manager for spatial isolation
func (n *Node) GetBarrierManager() *spatial.BarrierManager {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.barrierManager
}

// CalculateDistanceTo calculates distance to another node's spatial config
func (n *Node) CalculateDistanceTo(otherConfig *spatial.SpatialConfig) (*spatial.Distance, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return spatial.CalculateDistance(n.GetSpatialConfig(), otherConfig)
}

// IsPathBlocked returns true if communication path is blocked by barriers
func (n *Node) IsPathBlocked(to *spatial.SpatialConfig, messageType string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.barrierManager.PathBlocked(n.GetSpatialConfig(), to, messageType)
}

// GetDiscoveryService returns the discovery service for API access
func (n *Node) GetDiscoveryService() *discovery.Service {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.discovery
}

// GetNodeID returns the node identifier
func (n *Node) GetNodeID() string {
	return n.id
}

// GetClusterID returns the cluster identifier
func (n *Node) GetClusterID() string {
	if n.config == nil {
		return ""
	}
	return n.config.ClusterID
}

// GetTopologyMapper returns the topology mapper for API access
func (n *Node) GetTopologyMapper() *topology.TopologyMapper {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.topologyMapper
}

// generateNodeID creates a random node identifier
func generateNodeID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "node_" + hex.EncodeToString(bytes)
}
