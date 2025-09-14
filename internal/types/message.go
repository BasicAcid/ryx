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

// ComputationTask represents a computational task to be executed
type ComputationTask struct {
	Type       string                 `json:"type"`       // "wordcount", "search", "loganalysis"
	Data       string                 `json:"data"`       // Input data for computation
	Parameters map[string]interface{} `json:"parameters"` // Task-specific parameters
	Energy     int                    `json:"energy"`     // Propagation energy
	TTL        int                    `json:"ttl"`        // Time to live in seconds
}

// ComputationResult represents the result of a computational task
type ComputationResult struct {
	TaskID        string                 `json:"task_id"`        // SHA256 of original task
	TaskType      string                 `json:"task_type"`      // Type of computation performed
	Result        map[string]interface{} `json:"result"`         // Computation result data
	ExecutedBy    string                 `json:"executed_by"`    // Node that performed computation
	ExecutionTime int64                  `json:"execution_time"` // Milliseconds to complete
	Timestamp     int64                  `json:"timestamp"`      // When computation completed
}

// TaskExecutor defines the interface for executing computational tasks
type TaskExecutor interface {
	Execute(task *ComputationTask) (*ComputationResult, error)
	CanHandle(taskType string) bool
	GetTaskType() string
}

// ComputationService defines the interface for the computation service
type ComputationService interface {
	ExecuteTask(msg *InfoMessage) error
	GetActiveComputations() map[string]*ComputationResult
	GetComputationResult(taskID string) (*ComputationResult, bool)
	GetComputationStats() map[string]interface{}
}
