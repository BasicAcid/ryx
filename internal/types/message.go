package types

// Phase 4A: Chemical Properties for Chemistry-Based Computing
// ChemicalProperties defines chemical behavior for messages
type ChemicalProperties struct {
	// Basic chemical properties
	Concentration float64 `json:"concentration"` // Chemical concentration (0.0-1.0)
	Reactivity    float64 `json:"reactivity"`    // Reaction probability (0.0-1.0)
	Catalyst      bool    `json:"catalyst"`      // Whether this message catalyzes reactions
	Inhibitor     bool    `json:"inhibitor"`     // Whether this message inhibits reactions

	// Reaction rules
	ReactionRules []*ReactionRule `json:"reaction_rules,omitempty"` // Rules for chemical reactions

	// Concentration gradient properties
	DiffusionRate  float64 `json:"diffusion_rate"`  // Rate of concentration diffusion
	SourceStrength float64 `json:"source_strength"` // Strength as concentration source

	// Chemical metadata
	ChemicalType string   `json:"chemical_type"` // "enzyme", "substrate", "product", "signal"
	AffinityTags []string `json:"affinity_tags"` // Tags for reaction matching
}

// ReactionRule defines how messages can react with each other
type ReactionRule struct {
	TargetType     string   `json:"target_type"`     // Message type this rule targets
	TargetTags     []string `json:"target_tags"`     // Affinity tags required for reaction
	ReactionType   string   `json:"reaction_type"`   // "combine", "transform", "catalyze", "inhibit"
	ProductType    string   `json:"product_type"`    // Resulting message type
	EnergyChange   float64  `json:"energy_change"`   // Energy released/consumed in reaction
	Probability    float64  `json:"probability"`     // Reaction probability (0.0-1.0)
	RequiredEnergy float64  `json:"required_energy"` // Minimum energy needed for reaction
}

// ConcentrationState tracks chemical concentrations at a node
type ConcentrationState struct {
	MessageCounts   map[string]int     `json:"message_counts"`   // Count of each message type
	TotalMessages   int                `json:"total_messages"`   // Total messages at node
	Concentrations  map[string]float64 `json:"concentrations"`   // Concentration by message type
	GradientVectors map[string]float64 `json:"gradient_vectors"` // Concentration gradients
	LastUpdate      int64              `json:"last_update"`      // Timestamp of last update
}

// ChemicalReaction represents a completed chemical reaction
type ChemicalReaction struct {
	ReactionID   string   `json:"reaction_id"`   // Unique reaction identifier
	ReactantIDs  []string `json:"reactant_ids"`  // IDs of reacting messages
	ProductID    string   `json:"product_id"`    // ID of resulting message
	ReactionType string   `json:"reaction_type"` // Type of reaction
	EnergyChange float64  `json:"energy_change"` // Energy released/consumed
	NodeID       string   `json:"node_id"`       // Node where reaction occurred
	Timestamp    int64    `json:"timestamp"`     // When reaction occurred
}

// InfoMessage represents information that diffuses through the network
// Phase 4A: Enhanced with continuous energy and chemical properties
type InfoMessage struct {
	ID        string                 `json:"id"`        // SHA256 hash of content
	Type      string                 `json:"type"`      // "text", "task", "result", etc.
	Content   []byte                 `json:"content"`   // Actual data
	Energy    float64                `json:"energy"`    // Continuous propagation fuel (decreases each hop)
	TTL       int64                  `json:"ttl"`       // Unix timestamp when expires
	Hops      int                    `json:"hops"`      // How far it's traveled
	Source    string                 `json:"source"`    // Original node ID
	Path      []string               `json:"path"`      // Nodes it has visited
	Timestamp int64                  `json:"timestamp"` // Creation time
	Metadata  map[string]interface{} `json:"metadata"`  // Extra data

	// Phase 4A: Chemical properties
	Chemical *ChemicalProperties `json:"chemical,omitempty"` // Chemical reaction properties
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
	Energy     float64                `json:"energy"`     // Continuous propagation energy
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
