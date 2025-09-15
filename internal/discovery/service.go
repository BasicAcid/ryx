package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/config"
	"github.com/BasicAcid/ryx/internal/types"
)

// Neighbor represents a discovered neighbor node
type Neighbor struct {
	NodeID    string    `json:"node_id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	ClusterID string    `json:"cluster_id"`
	LastSeen  time.Time `json:"last_seen"`
}

// AnnounceMessage is broadcast to discover neighbors
type AnnounceMessage struct {
	Type      string `json:"type"`
	NodeID    string `json:"node_id"`
	ClusterID string `json:"cluster_id"`
	Port      int    `json:"port"`
	Timestamp int64  `json:"timestamp"`
}

// Service handles neighbor discovery via UDP broadcasts
type Service struct {
	port      int
	clusterID string
	nodeID    string
	conn      *net.UDPConn
	neighbors map[string]*Neighbor
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Phase 3B: Performance-based neighbor selection
	runtimeParams *config.RuntimeParameters
	behaviorMod   config.BehaviorModifier
}

// New creates a new discovery service
func New(port int, clusterID, nodeID string) (*Service, error) {
	return &Service{
		port:      port,
		clusterID: clusterID,
		nodeID:    nodeID,
		neighbors: make(map[string]*Neighbor),
	}, nil
}

// NewWithConfig creates a discovery service with runtime configuration
func NewWithConfig(port int, clusterID, nodeID string, params *config.RuntimeParameters, behaviorMod config.BehaviorModifier) (*Service, error) {
	return &Service{
		port:      port,
		clusterID: clusterID,
		nodeID:    nodeID,
		neighbors: make(map[string]*Neighbor),

		// Phase 3B: Advanced configuration
		runtimeParams: params,
		behaviorMod:   behaviorMod,
	}, nil
}

// Start begins the discovery process
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Listen for broadcasts on our own discovery port
	discoveryPort := s.port + 1000
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", discoveryPort))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", discoveryPort, err)
	}

	log.Printf("Discovery service listening on port %d", discoveryPort)

	// Start listening for announcements
	go s.listenLoop()

	// Start periodic announcements
	go s.announceLoop()

	// Start cleanup routine
	go s.cleanupLoop()

	// Phase 3B: Start topology optimization routine
	go s.topologyOptimizationLoop()

	return nil
}

// Stop shuts down the discovery service
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}

// GetNeighbors returns current neighbors as a slice for diffusion service
func (s *Service) GetNeighbors() []*types.Neighbor {
	s.mu.RLock()
	defer s.mu.RUnlock()

	neighbors := make([]*types.Neighbor, 0, len(s.neighbors))
	for _, neighbor := range s.neighbors {
		neighbors = append(neighbors, &types.Neighbor{
			NodeID:    neighbor.NodeID,
			Address:   neighbor.Address,
			Port:      neighbor.Port,
			ClusterID: neighbor.ClusterID,
		})
	}
	return neighbors
}

// GetNeighborsMap returns current neighbors as a map for backward compatibility
func (s *Service) GetNeighborsMap() map[string]*Neighbor {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*Neighbor)
	for k, v := range s.neighbors {
		// Create a copy of the neighbor
		neighbor := *v
		result[k] = &neighbor
	}
	return result
}

// listenLoop listens for incoming announcements
func (s *Service) listenLoop() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, addr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				// Timeout is expected, continue
				continue
			}

			s.handleAnnouncement(buffer[:n], addr)
		}
	}
}

// handleAnnouncement processes incoming announcements
func (s *Service) handleAnnouncement(data []byte, addr *net.UDPAddr) {
	var msg AnnounceMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal announcement: %v", err)
		return
	}

	// Ignore our own announcements
	if msg.NodeID == s.nodeID {
		return
	}

	// Only accept nodes from same cluster
	if msg.ClusterID != s.clusterID {
		return
	}

	log.Printf("Discovered neighbor: %s at %s:%d", msg.NodeID, addr.IP, msg.Port)

	// Phase 3B: Use behavior modifier for neighbor addition decisions
	candidate := &types.Neighbor{
		NodeID:    msg.NodeID,
		Address:   addr.IP.String(),
		Port:      msg.Port,
		ClusterID: msg.ClusterID,
	}

	currentNeighbors := s.GetNeighbors()
	shouldAdd := true

	if s.behaviorMod != nil {
		shouldAdd = s.behaviorMod.ShouldAddNeighbor(candidate, currentNeighbors)
	}

	if shouldAdd {
		s.mu.Lock()

		// If we're at capacity and using advanced behavior, remove worst neighbor
		maxNeighbors := 8
		if s.runtimeParams != nil {
			maxNeighbors = s.runtimeParams.GetInt("max_neighbors", 8)
		}

		if len(s.neighbors) >= maxNeighbors && s.behaviorMod != nil {
			s.removeWorstNeighbor()
		}

		s.neighbors[msg.NodeID] = &Neighbor{
			NodeID:    msg.NodeID,
			Address:   addr.IP.String(),
			Port:      msg.Port,
			ClusterID: msg.ClusterID,
			LastSeen:  time.Now(),
		}
		s.mu.Unlock()

		log.Printf("Added neighbor %s (total: %d)", msg.NodeID, len(s.neighbors))
	} else {
		log.Printf("Neighbor %s rejected by behavior modifier", msg.NodeID)
	}
}

// announceLoop periodically broadcasts our presence
func (s *Service) announceLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial announcement immediately
	s.sendAnnouncement()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.sendAnnouncement()
		}
	}
}

// sendAnnouncement broadcasts our presence to a range of discovery ports
func (s *Service) sendAnnouncement() {
	msg := AnnounceMessage{
		Type:      "announce",
		NodeID:    s.nodeID,
		ClusterID: s.clusterID,
		Port:      s.port,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal announcement: %v", err)
		return
	}

	// Broadcast to a range of discovery ports (for local testing)
	// This allows nodes with different base ports to find each other
	basePort := 10000
	for i := 0; i < 20; i++ { // Try ports 10000-10019
		discoveryPort := basePort + i

		// Skip our own port to avoid self-messages
		if discoveryPort == s.port+1000 {
			continue
		}

		broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", discoveryPort))
		if err != nil {
			continue
		}

		conn, err := net.DialUDP("udp", nil, broadcastAddr)
		if err != nil {
			continue
		}

		conn.Write(data)
		conn.Close()
	}
}

// cleanupLoop removes stale neighbors
func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes neighbors not seen recently
func (s *Service) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-60 * time.Second) // 60 second timeout
	for nodeID, neighbor := range s.neighbors {
		if neighbor.LastSeen.Before(cutoff) {
			log.Printf("Removing stale neighbor: %s", nodeID)
			delete(s.neighbors, nodeID)
		}
	}
}

// Phase 3B: Performance-based topology optimization

// topologyOptimizationLoop periodically optimizes neighbor topology
func (s *Service) topologyOptimizationLoop() {
	ticker := time.NewTicker(60 * time.Second) // Optimize every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Topology optimization loop stopping for node %s", s.nodeID)
			return
		case <-ticker.C:
			s.optimizeTopology()
		}
	}
}

// optimizeTopology removes poor-performing neighbors and maintains optimal topology
func (s *Service) optimizeTopology() {
	if s.behaviorMod == nil {
		return // No optimization without behavior modifier
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	minNeighbors := 3
	if s.runtimeParams != nil {
		minNeighbors = s.runtimeParams.GetInt("min_neighbors", 3)
	}

	// Don't optimize if we're at minimum capacity
	if len(s.neighbors) <= minNeighbors {
		return
	}

	// Find neighbors to remove based on performance
	toRemove := make([]string, 0)

	for nodeID, neighbor := range s.neighbors {
		candidate := &types.Neighbor{
			NodeID:    neighbor.NodeID,
			Address:   neighbor.Address,
			Port:      neighbor.Port,
			ClusterID: neighbor.ClusterID,
		}

		if s.behaviorMod.ShouldRemoveNeighbor(candidate, "poor_performance") {
			toRemove = append(toRemove, nodeID)
		}
	}

	// Remove poor performers (but maintain minimum count)
	for _, nodeID := range toRemove {
		if len(s.neighbors) > minNeighbors {
			log.Printf("Removing poor-performing neighbor: %s", nodeID)
			delete(s.neighbors, nodeID)
		}
	}

	log.Printf("Topology optimization completed. Current neighbors: %d", len(s.neighbors))
}

// removeWorstNeighbor removes the lowest-scoring neighbor to make room for a better one
func (s *Service) removeWorstNeighbor() {
	if len(s.neighbors) == 0 {
		return
	}

	// Find neighbor with lowest score
	worstScore := 1.0
	worstNodeID := ""

	if adaptiveMod, ok := s.behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		for nodeID := range s.neighbors {
			score := adaptiveMod.CalculateNeighborScore(nodeID)
			if score < worstScore {
				worstScore = score
				worstNodeID = nodeID
			}
		}

		if worstNodeID != "" {
			log.Printf("Removing worst neighbor %s (score: %.3f) to make room", worstNodeID, worstScore)
			delete(s.neighbors, worstNodeID)
		}
	}
}
