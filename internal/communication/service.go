package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Message represents a communication message between nodes
type Message struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Data      map[string]interface{} `json:"data"`
	Energy    int                    `json:"energy"`
	Hops      int                    `json:"hops"`
	Timestamp int64                  `json:"timestamp"`
}

// Service handles inter-node communication
type Service struct {
	port   int
	nodeID string
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// New creates a new communication service
func New(port int, nodeID string) (*Service, error) {
	return &Service{
		port:   port,
		nodeID: nodeID,
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

// handleInfo processes information messages
func (s *Service) handleInfo(msg *Message, addr *net.UDPAddr) {
	log.Printf("Info from %s: %v", msg.From, msg.Data)
	// TODO: Implement information diffusion logic
}
