package topology

import (
	"fmt"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/discovery"
	"github.com/BasicAcid/ryx/internal/spatial"
)

// NodeProvider interface for accessing node information
type NodeProvider interface {
	GetSpatialConfig() *spatial.SpatialConfig
	GetDiscoveryService() *discovery.Service
	GetBarrierManager() *spatial.BarrierManager
	GetNodeID() string
	GetClusterID() string
}

// TopologyMapper generates real-time network topology maps
type TopologyMapper struct {
	node           NodeProvider
	discovery      *discovery.Service
	barrierManager *spatial.BarrierManager
	mu             sync.RWMutex
	lastUpdate     time.Time
}

// NewTopologyMapper creates a new topology mapper
func NewTopologyMapper(node NodeProvider) *TopologyMapper {
	return &TopologyMapper{
		node:           node,
		discovery:      node.GetDiscoveryService(),
		barrierManager: node.GetBarrierManager(),
		lastUpdate:     time.Now(),
	}
}

// NetworkTopology represents complete spatial network layout
type NetworkTopology struct {
	Nodes       []*TopologyNode    `json:"nodes"`
	Connections []*TopologyLink    `json:"connections"`
	Barriers    []*TopologyBarrier `json:"barriers"`
	Zones       []*TopologyZone    `json:"zones"`
	Metadata    *TopologyMetadata  `json:"metadata"`
}

// TopologyNode represents a node in the topology
type TopologyNode struct {
	NodeID        string                 `json:"node_id"`
	ClusterID     string                 `json:"cluster_id"`
	SpatialConfig *spatial.SpatialConfig `json:"spatial_config"`
	Status        string                 `json:"status"` // "active", "degraded", "failed"
	Connections   int                    `json:"connections"`
	LastSeen      time.Time              `json:"last_seen"`
	Address       string                 `json:"address"`
	Port          int                    `json:"port"`
}

// TopologyLink represents connection between nodes
type TopologyLink struct {
	FromNodeID string            `json:"from_node_id"`
	ToNodeID   string            `json:"to_node_id"`
	Distance   *spatial.Distance `json:"distance,omitempty"`
	Quality    float64           `json:"quality"` // 0.0-1.0 connection quality
	Latency    *time.Duration    `json:"latency,omitempty"`
	Blocked    bool              `json:"blocked"`   // Blocked by barriers
	SameZone   bool              `json:"same_zone"` // Same logical zone
}

// TopologyBarrier represents a physical barrier in the topology
type TopologyBarrier struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	ZonesAffected []string `json:"zones_affected"`
	IsolationType string   `json:"isolation_type"`
	Active        bool     `json:"active"`
}

// TopologyZone represents a logical zone in the topology
type TopologyZone struct {
	ID          string   `json:"id"`
	NodeCount   int      `json:"node_count"`
	Nodes       []string `json:"nodes"`
	Connections int      `json:"connections"` // Connections within zone
	CrossZone   int      `json:"cross_zone"`  // Connections to other zones
}

// TopologyMetadata contains metadata about the topology
type TopologyMetadata struct {
	GeneratedAt     time.Time `json:"generated_at"`
	GeneratedBy     string    `json:"generated_by"`
	NodeCount       int       `json:"node_count"`
	ConnectionCount int       `json:"connection_count"`
	ZoneCount       int       `json:"zone_count"`
	BarrierCount    int       `json:"barrier_count"`
	ClusterID       string    `json:"cluster_id"`
}

// GetCurrentTopology generates a complete topology snapshot
func (tm *TopologyMapper) GetCurrentTopology() (*NetworkTopology, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Get current node's spatial config
	nodeSpatialConfig := tm.node.GetSpatialConfig()
	if nodeSpatialConfig == nil {
		return nil, fmt.Errorf("node spatial configuration not available")
	}

	// Get all neighbors from discovery service (with spatial information)
	neighbors := tm.discovery.GetNeighborsWithDistance()

	// Build topology nodes
	nodes := tm.buildTopologyNodes(neighbors)

	// Build connections between nodes
	connections := tm.buildTopologyConnections(nodes, nodeSpatialConfig)

	// Build barrier information
	barriers := tm.buildTopologyBarriers()

	// Build zone information
	zones := tm.buildTopologyZones(nodes)

	// Build metadata
	metadata := tm.buildTopologyMetadata(nodes, connections, zones, barriers)

	topology := &NetworkTopology{
		Nodes:       nodes,
		Connections: connections,
		Barriers:    barriers,
		Zones:       zones,
		Metadata:    metadata,
	}

	tm.lastUpdate = time.Now()
	return topology, nil
}

// buildTopologyNodes creates topology nodes from discovery neighbors
func (tm *TopologyMapper) buildTopologyNodes(neighbors []*discovery.Neighbor) []*TopologyNode {
	// Include current node
	nodes := []*TopologyNode{
		{
			NodeID:        tm.node.GetNodeID(),
			ClusterID:     tm.node.GetClusterID(),
			SpatialConfig: tm.node.GetSpatialConfig(),
			Status:        "active", // Current node is always active
			Connections:   len(neighbors),
			LastSeen:      time.Now(),
			Address:       "localhost", // Current node
			Port:          0,           // Not applicable for current node
		},
	}

	// Add neighbor nodes
	for _, neighbor := range neighbors {
		status := "active"

		// Check if node seems degraded based on last seen time
		if time.Since(neighbor.LastSeen) > 30*time.Second {
			status = "degraded"
		}
		if time.Since(neighbor.LastSeen) > 60*time.Second {
			status = "failed"
		}

		node := &TopologyNode{
			NodeID:        neighbor.NodeID,
			ClusterID:     neighbor.ClusterID,
			SpatialConfig: neighbor.SpatialConfig,
			Status:        status,
			Connections:   1, // At least connected to current node
			LastSeen:      neighbor.LastSeen,
			Address:       neighbor.Address,
			Port:          neighbor.Port,
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// buildTopologyConnections creates connections between nodes
func (tm *TopologyMapper) buildTopologyConnections(nodes []*TopologyNode, nodeSpatialConfig *spatial.SpatialConfig) []*TopologyLink {
	var connections []*TopologyLink
	currentNodeID := tm.node.GetNodeID()

	// For now, we only know direct connections from current node to neighbors
	// In a full mesh topology mapper, we'd need to gather connection info from all nodes
	for _, node := range nodes {
		if node.NodeID == currentNodeID {
			continue // Skip current node
		}

		// Calculate distance if both nodes have spatial configuration
		var distance *spatial.Distance
		if nodeSpatialConfig != nil && node.SpatialConfig != nil {
			if dist, err := spatial.CalculateDistance(nodeSpatialConfig, node.SpatialConfig); err == nil {
				distance = dist
			}
		}

		// Check if path is blocked by barriers
		blocked := false
		if nodeSpatialConfig != nil && node.SpatialConfig != nil {
			blocked = tm.barrierManager.PathBlocked(nodeSpatialConfig, node.SpatialConfig, "routine")
		}

		// Check if nodes are in same zone
		sameZone := false
		if nodeSpatialConfig != nil && node.SpatialConfig != nil {
			sameZone = spatial.IsInSameZone(nodeSpatialConfig, node.SpatialConfig)
		}

		// Calculate connection quality based on node status and last seen
		quality := tm.calculateConnectionQuality(node)

		connection := &TopologyLink{
			FromNodeID: currentNodeID,
			ToNodeID:   node.NodeID,
			Distance:   distance,
			Quality:    quality,
			Latency:    nil, // TODO: Implement latency measurement
			Blocked:    blocked,
			SameZone:   sameZone,
		}

		connections = append(connections, connection)
	}

	return connections
}

// calculateConnectionQuality estimates connection quality based on node status
func (tm *TopologyMapper) calculateConnectionQuality(node *TopologyNode) float64 {
	switch node.Status {
	case "active":
		// Base quality on how recently we've seen the node
		timeSinceLastSeen := time.Since(node.LastSeen)
		if timeSinceLastSeen < 5*time.Second {
			return 1.0
		} else if timeSinceLastSeen < 15*time.Second {
			return 0.8
		} else if timeSinceLastSeen < 30*time.Second {
			return 0.6
		}
		return 0.4
	case "degraded":
		return 0.3
	case "failed":
		return 0.1
	default:
		return 0.5
	}
}

// buildTopologyBarriers creates barrier information from barrier manager
func (tm *TopologyMapper) buildTopologyBarriers() []*TopologyBarrier {
	var barriers []*TopologyBarrier

	allBarriers := tm.barrierManager.GetAllBarriers()
	for _, barrier := range allBarriers {
		zonesAffected := []string{}
		if barrier.ZoneA != "" {
			zonesAffected = append(zonesAffected, barrier.ZoneA)
		}
		if barrier.ZoneB != "" {
			zonesAffected = append(zonesAffected, barrier.ZoneB)
		}

		topologyBarrier := &TopologyBarrier{
			ID:            barrier.ID,
			Type:          string(barrier.Type),
			Description:   barrier.Description,
			ZonesAffected: zonesAffected,
			IsolationType: barrier.Isolation,
			Active:        true, // TODO: Implement barrier status checking
		}

		barriers = append(barriers, topologyBarrier)
	}

	return barriers
}

// buildTopologyZones creates zone information from nodes
func (tm *TopologyMapper) buildTopologyZones(nodes []*TopologyNode) []*TopologyZone {
	zoneMap := make(map[string]*TopologyZone)
	zoneNodes := make(map[string][]string)

	// Group nodes by zone
	for _, node := range nodes {
		if node.SpatialConfig == nil || node.SpatialConfig.Zone == "" {
			continue
		}

		zoneID := node.SpatialConfig.Zone
		if _, exists := zoneMap[zoneID]; !exists {
			zoneMap[zoneID] = &TopologyZone{
				ID:          zoneID,
				NodeCount:   0,
				Nodes:       []string{},
				Connections: 0,
				CrossZone:   0,
			}
			zoneNodes[zoneID] = []string{}
		}

		zoneMap[zoneID].NodeCount++
		zoneMap[zoneID].Nodes = append(zoneMap[zoneID].Nodes, node.NodeID)
		zoneNodes[zoneID] = append(zoneNodes[zoneID], node.NodeID)
	}

	// Calculate zone connection statistics
	// This is simplified - in reality we'd need connection info from all nodes
	for _, zone := range zoneMap {
		// For now, assume each node has at least one connection within its zone
		zone.Connections = zone.NodeCount - 1
		if zone.Connections < 0 {
			zone.Connections = 0
		}

		// Estimate cross-zone connections
		zone.CrossZone = zone.NodeCount // Simplified estimate
	}

	// Convert map to slice
	var zones []*TopologyZone
	for _, zone := range zoneMap {
		zones = append(zones, zone)
	}

	return zones
}

// buildTopologyMetadata creates metadata for the topology
func (tm *TopologyMapper) buildTopologyMetadata(nodes []*TopologyNode, connections []*TopologyLink, zones []*TopologyZone, barriers []*TopologyBarrier) *TopologyMetadata {
	return &TopologyMetadata{
		GeneratedAt:     time.Now(),
		GeneratedBy:     tm.node.GetNodeID(),
		NodeCount:       len(nodes),
		ConnectionCount: len(connections),
		ZoneCount:       len(zones),
		BarrierCount:    len(barriers),
		ClusterID:       tm.node.GetClusterID(),
	}
}

// GetZoneTopology returns topology information for a specific zone
func (tm *TopologyMapper) GetZoneTopology(zoneID string) (*TopologyZone, error) {
	topology, err := tm.GetCurrentTopology()
	if err != nil {
		return nil, err
	}

	for _, zone := range topology.Zones {
		if zone.ID == zoneID {
			return zone, nil
		}
	}

	return nil, fmt.Errorf("zone not found: %s", zoneID)
}

// GetLiveTopologyUpdates returns a channel for live topology updates
// This would be used for WebSocket-based live updates in the future
func (tm *TopologyMapper) GetLiveTopologyUpdates() <-chan *NetworkTopology {
	updates := make(chan *NetworkTopology, 10)

	// For now, just return a closed channel
	// In a real implementation, this would provide periodic topology updates
	close(updates)

	return updates
}

// String returns a human-readable representation of the topology
func (nt *NetworkTopology) String() string {
	return fmt.Sprintf("Topology: %d nodes, %d connections, %d zones, %d barriers (generated by %s at %s)",
		nt.Metadata.NodeCount,
		nt.Metadata.ConnectionCount,
		nt.Metadata.ZoneCount,
		nt.Metadata.BarrierCount,
		nt.Metadata.GeneratedBy,
		nt.Metadata.GeneratedAt.Format(time.RFC3339))
}
