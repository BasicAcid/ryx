package types

// InfoMessage represents information that diffuses through the network
type InfoMessage struct {
	ID        string                 `json:"id"`        // SHA256 hash of content
	Type      string                 `json:"type"`      // "text", "task", "result", etc.
	Content   []byte                 `json:"content"`   // Actual data
	Energy    int                    `json:"energy"`    // Propagation fuel (decreases each hop)
	TTL       int64                  `json:"ttl"`       // Unix timestamp when expires
	Hops      int                    `json:"hops"`      // How far it's traveled
	Source    string                 `json:"source"`    // Original node ID
	Path      []string               `json:"path"`      // Nodes it has visited
	Timestamp int64                  `json:"timestamp"` // Creation time
	Metadata  map[string]interface{} `json:"metadata"`  // Extra data
}

// InfoMessageHandler defines the interface for handling info messages
type InfoMessageHandler interface {
	HandleInfoMessage(msg *InfoMessage, fromNodeID string) error
}

// CommunicationService defines the interface for sending messages between nodes
type CommunicationService interface {
	SendInfoMessage(nodeID, address string, port int, msg *InfoMessage) error
}

// DiscoveryService defines the interface for neighbor discovery
type DiscoveryService interface {
	GetNeighbors() []*Neighbor
}

// Neighbor represents a discovered neighbor node
type Neighbor struct {
	NodeID    string `json:"node_id"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	ClusterID string `json:"cluster_id"`
}
