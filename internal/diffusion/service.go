package diffusion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/types"
)

// Service manages information diffusion
type Service struct {
	nodeID  string
	storage map[string]*types.InfoMessage
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	comm    types.CommunicationService
	disc    types.DiscoveryService
}

// New creates a new diffusion service
func New(nodeID string) *Service {
	log.Printf("Creating new diffusion service for node %s", nodeID)
	return &Service{
		nodeID:  nodeID,
		storage: make(map[string]*types.InfoMessage),
	}
}

// Start begins the diffusion service
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	log.Printf("Diffusion service starting for node %s", s.nodeID)

	// Start cleanup routine
	go s.cleanupLoop()

	return nil
}

// Stop shuts down the diffusion service
func (s *Service) Stop() {
	log.Printf("Diffusion service stopping for node %s", s.nodeID)
	if s.cancel != nil {
		s.cancel()
	}
}

// InjectInfo creates and stores new information (simplified version)
func (s *Service) InjectInfo(infoType string, content []byte, energy int, ttl time.Duration) (*types.InfoMessage, error) {
	log.Printf("InjectInfo called: type=%s, content_len=%d, energy=%d, ttl=%v", infoType, len(content), energy, ttl)

	// Create content ID
	id := generateInfoID(content)
	log.Printf("Generated ID: %s", id)

	// Check if we already have this information
	s.mu.RLock()
	if existing, exists := s.storage[id]; exists {
		s.mu.RUnlock()
		log.Printf("Information already exists with ID %s", id)
		return existing, nil
	}
	s.mu.RUnlock()

	// Create new info message
	info := &types.InfoMessage{
		ID:        id,
		Type:      infoType,
		Content:   content,
		Energy:    energy,
		TTL:       time.Now().Add(ttl).Unix(),
		Hops:      0,
		Source:    s.nodeID,
		Path:      []string{s.nodeID},
		Timestamp: time.Now().Unix(),
		Metadata:  make(map[string]interface{}),
	}

	// Store locally
	s.mu.Lock()
	s.storage[id] = info
	s.mu.Unlock()

	log.Printf("Information injected successfully: id=%s", id)

	// Forward to neighbors if energy > 0 (Phase 2B inter-node diffusion)
	if info.Energy > 0 {
		log.Printf("InjectInfo: forwarding message id=%s with energy=%d", id, info.Energy)
		go s.forwardToNeighbors(info)
	} else {
		log.Printf("InjectInfo: message id=%s has no energy, not forwarding", id)
	}

	return info, nil
}

// HandleInfoMessage processes incoming information messages and forwards them
func (s *Service) HandleInfoMessage(msg *types.InfoMessage, fromNodeID string) error {
	log.Printf("HandleInfoMessage: received message id=%s from=%s energy=%d hops=%d",
		msg.ID, fromNodeID, msg.Energy, msg.Hops)

	// Check if we already have this message (deduplication)
	s.mu.RLock()
	if _, exists := s.storage[msg.ID]; exists {
		s.mu.RUnlock()
		log.Printf("HandleInfoMessage: message id=%s already exists, skipping", msg.ID)
		return nil
	}
	s.mu.RUnlock()

	// Store the message locally
	s.mu.Lock()
	s.storage[msg.ID] = msg
	s.mu.Unlock()

	log.Printf("HandleInfoMessage: stored message id=%s", msg.ID)

	// Forward to neighbors if energy > 0
	if msg.Energy > 0 {
		log.Printf("HandleInfoMessage: forwarding message id=%s with energy=%d", msg.ID, msg.Energy)
		go s.forwardToNeighbors(msg)
	} else {
		log.Printf("HandleInfoMessage: message id=%s has no energy, not forwarding", msg.ID)
	}

	return nil
}

// GetAllInfo returns all stored information
func (s *Service) GetAllInfo() map[string]*types.InfoMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("GetAllInfo: returning %d messages", len(s.storage))

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*types.InfoMessage)
	for k, v := range s.storage {
		info := *v
		result[k] = &info
	}
	return result
}

// GetInfo returns specific information by ID
func (s *Service) GetInfo(id string) (*types.InfoMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.storage[id]
	if !exists {
		return nil, false
	}

	// Return a copy
	infoCopy := *info
	return &infoCopy, true
}

// GetStats returns diffusion statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_messages": len(s.storage),
		"node_id":        s.nodeID,
	}
}

// cleanupLoop removes expired messages
func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Cleanup loop stopping for node %s", s.nodeID)
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes messages that have exceeded their TTL
func (s *Service) cleanup() {
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for id, info := range s.storage {
		if info.TTL < now {
			delete(s.storage, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("Cleaned up %d expired messages", removed)
	}
}

// SetCommunication injects the communication service for forwarding
func (s *Service) SetCommunication(comm types.CommunicationService) {
	s.comm = comm
}

// SetDiscovery injects the discovery service for finding neighbors
func (s *Service) SetDiscovery(disc types.DiscoveryService) {
	s.disc = disc
}

// forwardToNeighbors forwards a message to all eligible neighbors
func (s *Service) forwardToNeighbors(msg *types.InfoMessage) {
	if s.comm == nil || s.disc == nil {
		log.Printf("Services not injected, cannot forward message")
		return
	}

	neighbors := s.disc.GetNeighbors()
	log.Printf("forwardToNeighbors: found %d neighbors", len(neighbors))

	for _, neighbor := range neighbors {
		if s.shouldForward(msg, neighbor.NodeID) {
			// Create a forwarded copy with updated energy and path
			forwardedMsg := s.createForwardedMessage(msg, neighbor.NodeID)
			log.Printf("Forwarding message %s to %s (energy: %d→%d, hops: %d→%d)",
				msg.ID, neighbor.NodeID, msg.Energy, forwardedMsg.Energy, msg.Hops, forwardedMsg.Hops)

			err := s.comm.SendInfoMessage(neighbor.NodeID, neighbor.Address, neighbor.Port, forwardedMsg)
			if err != nil {
				log.Printf("Failed to forward message to %s: %v", neighbor.NodeID, err)
			}
		} else {
			log.Printf("Not forwarding message %s to %s (loop prevention or energy exhausted)", msg.ID, neighbor.NodeID)
		}
	}
}

// shouldForward determines if a message should be forwarded to a specific node
func (s *Service) shouldForward(msg *types.InfoMessage, targetNodeID string) bool {
	// Don't forward if energy is exhausted
	if msg.Energy <= 0 {
		return false
	}

	// Don't forward back to the source
	if msg.Source == targetNodeID {
		return false
	}

	// Don't forward to nodes already in the path (loop prevention)
	for _, nodeID := range msg.Path {
		if nodeID == targetNodeID {
			return false
		}
	}

	return true
}

// createForwardedMessage creates a new message for forwarding with updated energy and path
func (s *Service) createForwardedMessage(original *types.InfoMessage, targetNodeID string) *types.InfoMessage {
	// Create a deep copy
	forwarded := &types.InfoMessage{
		ID:        original.ID,
		Type:      original.Type,
		Content:   original.Content,
		Energy:    original.Energy - 1, // Decrease energy
		TTL:       original.TTL,
		Hops:      original.Hops + 1, // Increase hop count
		Source:    original.Source,   // Keep original source
		Timestamp: original.Timestamp,
		Metadata:  make(map[string]interface{}),
	}

	// Copy metadata
	for k, v := range original.Metadata {
		forwarded.Metadata[k] = v
	}

	// Update path with current node (the one doing the forwarding)
	forwarded.Path = make([]string, len(original.Path)+1)
	copy(forwarded.Path, original.Path)
	forwarded.Path[len(original.Path)] = s.nodeID

	return forwarded
}

// generateInfoID creates a content-addressable ID for information
func generateInfoID(content []byte) string {
	hash := sha256.Sum256(content)
	// Use first 8 bytes for shorter, readable IDs
	return hex.EncodeToString(hash[:8])
}
