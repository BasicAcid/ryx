package ca

import (
	"encoding/json"
	"log"
	"math"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/discovery"
	"github.com/BasicAcid/ryx/internal/spatial"
	"github.com/BasicAcid/ryx/internal/types"
)

// NetworkManager handles CA grid connectivity between nodes
type NetworkManager struct {
	nodeID    string
	caEngine  *Engine
	discovery *discovery.Service
	spatial   *spatial.SpatialConfig

	// Communication interfaces
	commService types.CommunicationService

	// Connection tracking
	connectedGrids map[string]*GridConnection // nodeID -> connection info
	mu             sync.RWMutex

	// Update tracking
	lastBoundaryUpdate map[string]int // nodeID -> last generation received
	updateMu           sync.RWMutex
}

// GridConnection represents a connection to another CA grid
type GridConnection struct {
	NodeID     string            `json:"node_id"`
	Address    string            `json:"address"`
	Port       int               `json:"port"`
	Distance   *spatial.Distance `json:"distance"`
	Direction  string            `json:"direction"` // "north", "south", "east", "west"
	LastUpdate time.Time         `json:"last_update"`
	Generation int               `json:"generation"` // Last generation received
}

// NewNetworkManager creates a CA network manager
func NewNetworkManager(nodeID string, engine *Engine, discovery *discovery.Service, spatialConfig *spatial.SpatialConfig) *NetworkManager {
	nm := &NetworkManager{
		nodeID:             nodeID,
		caEngine:           engine,
		discovery:          discovery,
		spatial:            spatialConfig,
		connectedGrids:     make(map[string]*GridConnection),
		lastBoundaryUpdate: make(map[string]int),
	}

	// Set boundary callback in CA engine
	engine.SetBoundaryCallback(nm.broadcastBoundaryStates)

	return nm
}

// SetCommunicationService sets the communication service for sending messages
func (nm *NetworkManager) SetCommunicationService(comm types.CommunicationService) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.commService = comm
}

// Start begins CA network operations
func (nm *NetworkManager) Start() {
	log.Printf("CA Network[%s]: Starting CA grid network manager", nm.nodeID)

	// Start periodic neighbor discovery for CA connections
	go nm.discoverCANeighbors()

	// Start periodic cleanup of stale connections
	go nm.cleanupStaleConnections()
}

// Stop shuts down CA network operations
func (nm *NetworkManager) Stop() {
	log.Printf("CA Network[%s]: Stopping CA grid network manager", nm.nodeID)

	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Clear all connections
	nm.connectedGrids = make(map[string]*GridConnection)
	nm.lastBoundaryUpdate = make(map[string]int)
}

// broadcastBoundaryStates sends boundary states to connected neighbors
func (nm *NetworkManager) broadcastBoundaryStates(boundaries *BoundaryStates) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if nm.commService == nil {
		return // No communication service available
	}

	// Convert BoundaryStates to CABoundaryMessage
	msg := &types.CABoundaryMessage{
		Type:       "ca_boundary",
		NodeID:     nm.nodeID,
		Generation: boundaries.Generation,
		North:      cellStatesToInt(boundaries.North),
		South:      cellStatesToInt(boundaries.South),
		East:       cellStatesToInt(boundaries.East),
		West:       cellStatesToInt(boundaries.West),
		Timestamp:  time.Now().Unix(),
	}

	// Create InfoMessage wrapper
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("CA Network[%s]: Failed to marshal boundary message: %v", nm.nodeID, err)
		return
	}

	infoMsg := &types.InfoMessage{
		ID:        generateMessageID(msgData),
		Type:      "ca_boundary",
		Content:   msgData,
		Energy:    1.0,
		TTL:       time.Now().Add(5 * time.Second).Unix(),
		Source:    nm.nodeID,
		Timestamp: time.Now().Unix(),
	}

	// Send to connected neighbors
	sent := 0
	for _, conn := range nm.connectedGrids {
		err := nm.commService.SendInfoMessage(conn.NodeID, conn.Address, conn.Port, infoMsg)
		if err != nil {
			log.Printf("CA Network[%s]: Failed to send boundary to %s: %v", nm.nodeID, conn.NodeID, err)
		} else {
			sent++
		}
	}

	if sent > 0 {
		log.Printf("CA Network[%s]: Broadcast boundary states (gen %d) to %d connected grids",
			nm.nodeID, boundaries.Generation, sent)
	}
}

// HandleBoundaryMessage processes incoming boundary state messages
func (nm *NetworkManager) HandleBoundaryMessage(msg *types.InfoMessage, fromNodeID string) {
	var boundaryMsg types.CABoundaryMessage
	if err := json.Unmarshal(msg.Content, &boundaryMsg); err != nil {
		log.Printf("CA Network[%s]: Failed to unmarshal boundary message: %v", nm.nodeID, err)
		return
	}

	// Validate message
	if boundaryMsg.NodeID != fromNodeID {
		log.Printf("CA Network[%s]: Node ID mismatch in boundary message", nm.nodeID)
		return
	}

	// Check for duplicate/old messages
	nm.updateMu.Lock()
	lastGen, exists := nm.lastBoundaryUpdate[fromNodeID]
	if exists && boundaryMsg.Generation <= lastGen {
		nm.updateMu.Unlock()
		return // Old or duplicate message
	}
	nm.lastBoundaryUpdate[fromNodeID] = boundaryMsg.Generation
	nm.updateMu.Unlock()

	// Convert to BoundaryStates
	boundaries := &BoundaryStates{
		North:      intToCellStates(boundaryMsg.North),
		South:      intToCellStates(boundaryMsg.South),
		East:       intToCellStates(boundaryMsg.East),
		West:       intToCellStates(boundaryMsg.West),
		NodeID:     boundaryMsg.NodeID,
		Generation: boundaryMsg.Generation,
	}

	// Update CA engine
	nm.caEngine.UpdateNeighborBoundary(fromNodeID, boundaries)

	// Update connection timestamp
	nm.mu.Lock()
	if conn, exists := nm.connectedGrids[fromNodeID]; exists {
		conn.LastUpdate = time.Now()
		conn.Generation = boundaryMsg.Generation
	}
	nm.mu.Unlock()

	log.Printf("CA Network[%s]: Updated boundary states from %s (gen %d)",
		nm.nodeID, fromNodeID, boundaryMsg.Generation)
}

// discoverCANeighbors periodically discovers and connects to nearby CA grids
func (nm *NetworkManager) discoverCANeighbors() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.updateCAConnections()
		}
	}
}

// updateCAConnections evaluates neighbors and creates/removes CA connections
func (nm *NetworkManager) updateCAConnections() {
	if nm.discovery == nil {
		return
	}

	// Get current spatial neighbors
	neighbors := nm.discovery.GetNeighborsWithDistance()

	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Track which neighbors are still active
	activeNeighbors := make(map[string]bool)

	for _, neighbor := range neighbors {
		activeNeighbors[neighbor.NodeID] = true

		// Check if we should connect to this neighbor
		if nm.shouldConnectToNeighbor(neighbor) {
			// Create or update connection
			if _, exists := nm.connectedGrids[neighbor.NodeID]; !exists {
				conn := &GridConnection{
					NodeID:     neighbor.NodeID,
					Address:    neighbor.Address,
					Port:       neighbor.Port,
					Distance:   neighbor.Distance,
					Direction:  nm.calculateDirection(neighbor),
					LastUpdate: time.Now(),
					Generation: 0,
				}
				nm.connectedGrids[neighbor.NodeID] = conn
				log.Printf("CA Network[%s]: Connected to CA grid %s (%s)",
					nm.nodeID, neighbor.NodeID, conn.Direction)
			}
		}
	}

	// Remove connections to neighbors that are no longer active
	for nodeID := range nm.connectedGrids {
		if !activeNeighbors[nodeID] {
			delete(nm.connectedGrids, nodeID)
			nm.caEngine.RemoveNeighborBoundary(nodeID)
			log.Printf("CA Network[%s]: Disconnected from CA grid %s", nm.nodeID, nodeID)
		}
	}
}

// shouldConnectToNeighbor determines if we should connect CA grids to this neighbor
func (nm *NetworkManager) shouldConnectToNeighbor(neighbor *discovery.Neighbor) bool {
	// For now, connect to all neighbors with valid distance
	// This can be enhanced with spatial positioning logic later
	return neighbor.Distance != nil && !isInfiniteDistance(neighbor.Distance)
}

// calculateDirection determines the spatial direction to a neighbor (simplified)
func (nm *NetworkManager) calculateDirection(neighbor *discovery.Neighbor) string {
	// Simple direction calculation - can be enhanced with actual spatial math
	// For now, assign directions cyclically
	directions := []string{"north", "south", "east", "west"}
	hash := simpleStringHash(neighbor.NodeID)
	return directions[hash%len(directions)]
}

// cleanupStaleConnections removes connections that haven't been updated recently
func (nm *NetworkManager) cleanupStaleConnections() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.mu.Lock()
			staleTimeout := time.Now().Add(-60 * time.Second) // 60 second timeout

			for nodeID, conn := range nm.connectedGrids {
				if conn.LastUpdate.Before(staleTimeout) {
					delete(nm.connectedGrids, nodeID)
					nm.caEngine.RemoveNeighborBoundary(nodeID)
					log.Printf("CA Network[%s]: Removed stale connection to %s", nm.nodeID, nodeID)
				}
			}
			nm.mu.Unlock()
		}
	}
}

// GetConnectedGrids returns current CA grid connections
func (nm *NetworkManager) GetConnectedGrids() map[string]*GridConnection {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Return copy to avoid concurrent access
	result := make(map[string]*GridConnection)
	for k, v := range nm.connectedGrids {
		connCopy := *v
		result[k] = &connCopy
	}
	return result
}

// HandleInfoMessage implements types.InfoMessageHandler for CA messages
func (nm *NetworkManager) HandleInfoMessage(msg *types.InfoMessage, fromNodeID string) error {
	switch msg.Type {
	case "ca_boundary":
		nm.HandleBoundaryMessage(msg, fromNodeID)
		return nil
	default:
		log.Printf("CA Network[%s]: Unknown CA message type: %s", nm.nodeID, msg.Type)
		return nil
	}
}

// Utility functions

func cellStatesToInt(states []CellState) []int {
	result := make([]int, len(states))
	for i, state := range states {
		result[i] = int(state)
	}
	return result
}

func intToCellStates(ints []int) []CellState {
	result := make([]CellState, len(ints))
	for i, val := range ints {
		result[i] = CellState(val)
	}
	return result
}

func isInfiniteDistance(distance *spatial.Distance) bool {
	return math.IsInf(distance.Value, 1) || distance.Value < 0 // Check for infinite or invalid distance
}

func simpleStringHash(s string) int {
	hash := 0
	for _, c := range s {
		hash = (hash * 31) + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func generateMessageID(data []byte) string {
	// Simple message ID generation - in real implementation, use proper hashing
	return time.Now().Format("20060102150405.000000")
}
