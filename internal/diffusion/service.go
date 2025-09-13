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
	return info, nil
}

// HandleInfoMessage processes incoming information messages
func (s *Service) HandleInfoMessage(msg *types.InfoMessage, fromNodeID string) error {
	log.Printf("HandleInfoMessage: received message id=%s from=%s", msg.ID, fromNodeID)

	// Simple storage for now (no diffusion logic yet)
	s.mu.Lock()
	s.storage[msg.ID] = msg
	s.mu.Unlock()

	log.Printf("HandleInfoMessage: stored message id=%s", msg.ID)
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

// generateInfoID creates a content-addressable ID for information
func generateInfoID(content []byte) string {
	hash := sha256.Sum256(content)
	// Use first 8 bytes for shorter, readable IDs
	return hex.EncodeToString(hash[:8])
}
