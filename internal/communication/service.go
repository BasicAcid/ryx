package communication

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

// Message represents a communication message between nodes
type Message struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Data      map[string]interface{} `json:"data"`
	Energy    float64                `json:"energy"`
	Hops      int                    `json:"hops"`
	Timestamp int64                  `json:"timestamp"`
}

// Service handles inter-node communication
type Service struct {
	port             int
	nodeID           string
	conn             *net.UDPConn
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	diffusionService types.InfoMessageHandler

	// Phase 3B: Fault pattern learning
	behaviorMod config.BehaviorModifier

	// Phase 3: CA message handling
	caMessageHandler types.InfoMessageHandler
}

// New creates a new communication service
func New(port int, nodeID string) (*Service, error) {
	return &Service{
		port:   port,
		nodeID: nodeID,
	}, nil
}

// NewWithConfig creates a communication service with behavior modification
func NewWithConfig(port int, nodeID string, behaviorMod config.BehaviorModifier) (*Service, error) {
	return &Service{
		port:        port,
		nodeID:      nodeID,
		behaviorMod: behaviorMod,
	}, nil
}

// Start begins the communication service
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Listen on the main node port
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", s.port, err)
	}

	log.Printf("Communication service listening on port %d", s.port)

	// Start message handling loop
	go s.messageLoop()

	return nil
}

// Stop shuts down the communication service
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}

// SendMessage sends a message to a specific node
func (s *Service) SendMessage(address string, port int, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return fmt.Errorf("failed to resolve target address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendPing sends a ping message to check neighbor health
func (s *Service) SendPing(address string, port int) error {
	message := &Message{
		Type:      "ping",
		From:      s.nodeID,
		Data:      map[string]interface{}{"timestamp": time.Now().Unix()},
		Timestamp: time.Now().Unix(),
	}

	return s.SendMessage(address, port, message)
}

// SetDiffusionService injects the diffusion service for handling info messages
func (s *Service) SetDiffusionService(diffService types.InfoMessageHandler) {
	s.diffusionService = diffService
}

// SetCAMessageHandler sets the CA message handler for CA boundary messages
func (s *Service) SetCAMessageHandler(handler types.InfoMessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.caMessageHandler = handler
}

// SendInfoMessage sends an InfoMessage to a specific node with performance tracking
func (s *Service) SendInfoMessage(nodeID, address string, port int, infoMsg *types.InfoMessage) error {
	startTime := time.Now()

	// Convert InfoMessage to communication Message
	msg := &Message{
		Type:      "info",
		From:      s.nodeID,
		To:        nodeID,
		Data:      s.convertInfoMessageToData(infoMsg),
		Energy:    infoMsg.Energy,
		Hops:      infoMsg.Hops,
		Timestamp: time.Now().Unix(),
	}

	err := s.SendMessage(address, port, msg)
	latency := time.Since(startTime)

	// Phase 3B: Record performance metrics for adaptive behavior
	if s.behaviorMod != nil {
		if adaptiveMod, ok := s.behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
			if err != nil {
				// Record failure
				adaptiveMod.RecordCommunicationFailure(nodeID, infoMsg.Type, err.Error())
				adaptiveMod.RecordNeighborPerformance(nodeID, latency, false)
			} else {
				// Record success
				adaptiveMod.RecordCommunicationSuccess(nodeID)
				adaptiveMod.RecordNeighborPerformance(nodeID, latency, true)
			}
		}
	}

	return err
}

// messageLoop handles incoming messages
func (s *Service) messageLoop() {
	buffer := make([]byte, 4096)

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

			s.handleMessage(buffer[:n], addr)
		}
	}
}

// handleMessage processes incoming messages
func (s *Service) handleMessage(data []byte, addr *net.UDPAddr) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal message from %s: %v", addr, err)
		return
	}

	// Ignore messages from ourselves
	if msg.From == s.nodeID {
		return
	}

	log.Printf("Received %s message from %s", msg.Type, msg.From)

	switch msg.Type {
	case "ping":
		s.handlePing(&msg, addr)
	case "pong":
		s.handlePong(&msg, addr)
	case "info":
		s.handleInfo(&msg, addr)
	case "ca_boundary":
		s.handleCABoundary(&msg, addr)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handlePing responds to ping messages
func (s *Service) handlePing(msg *Message, addr *net.UDPAddr) {
	response := &Message{
		Type: "pong",
		From: s.nodeID,
		To:   msg.From,
		Data: map[string]interface{}{
			"timestamp":      time.Now().Unix(),
			"ping_timestamp": msg.Data["timestamp"],
		},
		Timestamp: time.Now().Unix(),
	}

	// Send response back
	if err := s.SendMessage(addr.IP.String(), s.port, response); err != nil {
		log.Printf("Failed to send pong to %s: %v", msg.From, err)
	}
}

// handlePong processes ping responses
func (s *Service) handlePong(msg *Message, addr *net.UDPAddr) {
	// Calculate round-trip time if we have the original timestamp
	if pingTime, ok := msg.Data["ping_timestamp"].(float64); ok {
		rtt := time.Now().Unix() - int64(pingTime)
		log.Printf("Pong from %s, RTT: %d seconds", msg.From, rtt)
	}
}

// handleInfo processes information messages and routes them to the diffusion service
func (s *Service) handleInfo(msg *Message, addr *net.UDPAddr) {
	log.Printf("Received info message from %s", msg.From)

	if s.diffusionService == nil {
		log.Printf("No diffusion service configured, ignoring info message")
		return
	}

	// Convert communication message to InfoMessage
	infoMsg, err := s.convertToInfoMessage(msg)
	if err != nil {
		log.Printf("Failed to convert message to InfoMessage: %v", err)
		return
	}

	// Route to diffusion service
	err = s.diffusionService.HandleInfoMessage(infoMsg, msg.From)
	if err != nil {
		log.Printf("Failed to handle info message: %v", err)
	}
}

// handleCABoundary processes CA boundary messages
func (s *Service) handleCABoundary(msg *Message, addr *net.UDPAddr) {
	log.Printf("Received CA boundary message from %s", msg.From)

	s.mu.RLock()
	handler := s.caMessageHandler
	s.mu.RUnlock()

	if handler == nil {
		log.Printf("No CA message handler configured, ignoring CA boundary message")
		return
	}

	// Convert communication message to InfoMessage
	infoMsg, err := s.convertToInfoMessage(msg)
	if err != nil {
		log.Printf("Failed to convert CA boundary message to InfoMessage: %v", err)
		return
	}

	// Route to CA message handler
	err = handler.HandleInfoMessage(infoMsg, msg.From)
	if err != nil {
		log.Printf("Failed to handle CA boundary message: %v", err)
	}
}

// convertToInfoMessage converts a communication Message to an InfoMessage
func (s *Service) convertToInfoMessage(msg *Message) (*types.InfoMessage, error) {
	data := msg.Data

	// Extract required fields from the message data
	id, ok := data["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	infoType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid type field")
	}

	contentStr, ok := data["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid content field")
	}
	content := []byte(contentStr)

	ttl, ok := data["ttl"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid ttl field")
	}

	source, ok := data["source"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid source field")
	}

	timestamp, ok := data["timestamp"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid timestamp field")
	}

	// Extract path
	pathInterface, ok := data["path"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid path field")
	}

	path := make([]string, len(pathInterface))
	for i, p := range pathInterface {
		pathStr, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("invalid path element")
		}
		path[i] = pathStr
	}

	// Extract metadata
	metadata := make(map[string]interface{})
	if metaInterface, ok := data["metadata"].(map[string]interface{}); ok {
		metadata = metaInterface
	}

	return &types.InfoMessage{
		ID:        id,
		Type:      infoType,
		Content:   content,
		Energy:    msg.Energy,
		TTL:       int64(ttl),
		Hops:      msg.Hops,
		Source:    source,
		Path:      path,
		Timestamp: int64(timestamp),
		Metadata:  metadata,
	}, nil
}

// convertInfoMessageToData converts an InfoMessage to map[string]interface{} for transmission
func (s *Service) convertInfoMessageToData(infoMsg *types.InfoMessage) map[string]interface{} {
	return map[string]interface{}{
		"id":        infoMsg.ID,
		"type":      infoMsg.Type,
		"content":   string(infoMsg.Content),
		"ttl":       infoMsg.TTL,
		"source":    infoMsg.Source,
		"path":      infoMsg.Path,
		"timestamp": infoMsg.Timestamp,
		"metadata":  infoMsg.Metadata,
	}
}
