package config

import (
	"sync"
	"time"
)

// RuntimeParameters holds all configurable system parameters
type RuntimeParameters struct {
	mu sync.RWMutex

	// Energy and propagation parameters
	EnergyDecayRate      float64 `json:"energy_decay_rate"`
	EnergyDecayCritical  float64 `json:"energy_decay_critical"`
	EnergyDecayRoutine   float64 `json:"energy_decay_routine"`
	DefaultEnergyInfo    int     `json:"default_energy_info"`
	DefaultEnergyCompute int     `json:"default_energy_compute"`

	// TTL and cleanup parameters
	DefaultTTLSeconds      int `json:"default_ttl_seconds"`
	CleanupIntervalSeconds int `json:"cleanup_interval_seconds"`
	CleanupBatchSize       int `json:"cleanup_batch_size"`

	// Neighbor and discovery parameters
	MaxNeighbors      int           `json:"max_neighbors"`
	MinNeighbors      int           `json:"min_neighbors"`
	NeighborTimeout   time.Duration `json:"neighbor_timeout"`
	DiscoveryInterval time.Duration `json:"discovery_interval"`
	AnnounceInterval  time.Duration `json:"announce_interval"`

	// Communication parameters
	MessageTimeout time.Duration `json:"message_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryBackoff   time.Duration `json:"retry_backoff"`

	// Performance parameters
	MaxConcurrentTasks     int     `json:"max_concurrent_tasks"`
	LoadBalancingThreshold float64 `json:"load_balancing_threshold"`
	PerformanceWindow      int     `json:"performance_window"`

	// Self-modification parameters
	AdaptationEnabled    bool          `json:"adaptation_enabled"`
	LearningRate         float64       `json:"learning_rate"`
	AdaptationThreshold  float64       `json:"adaptation_threshold"`
	ModificationCooldown time.Duration `json:"modification_cooldown"`
}

// GetDefaults returns default runtime parameters for mission-critical systems
func GetDefaults() *RuntimeParameters {
	return &RuntimeParameters{
		// Energy defaults - conservative for reliability
		EnergyDecayRate:      1.0,
		EnergyDecayCritical:  0.5, // Critical messages travel further
		EnergyDecayRoutine:   1.5, // Routine messages decay faster
		DefaultEnergyInfo:    10,
		DefaultEnergyCompute: 3,

		// TTL defaults - 5 minutes standard, frequent cleanup
		DefaultTTLSeconds:      300,
		CleanupIntervalSeconds: 30,
		CleanupBatchSize:       100,

		// Neighbor defaults - maintain good connectivity
		MaxNeighbors:      8,
		MinNeighbors:      3,
		NeighborTimeout:   60 * time.Second,
		DiscoveryInterval: 10 * time.Second,
		AnnounceInterval:  5 * time.Second,

		// Communication defaults - reliable but responsive
		MessageTimeout: 5 * time.Second,
		RetryAttempts:  3,
		RetryBackoff:   time.Second,

		// Performance defaults - balanced throughput
		MaxConcurrentTasks:     10,
		LoadBalancingThreshold: 0.8,
		PerformanceWindow:      100,

		// Self-modification defaults - conservative learning
		AdaptationEnabled:    true,
		LearningRate:         0.1,
		AdaptationThreshold:  0.05,
		ModificationCooldown: 30 * time.Second,
	}
}

// Get returns a parameter value (thread-safe)
func (rp *RuntimeParameters) Get(param string) interface{} {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	switch param {
	case "energy_decay_rate":
		return rp.EnergyDecayRate
	case "energy_decay_critical":
		return rp.EnergyDecayCritical
	case "energy_decay_routine":
		return rp.EnergyDecayRoutine
	case "default_energy_info":
		return rp.DefaultEnergyInfo
	case "default_energy_compute":
		return rp.DefaultEnergyCompute
	case "default_ttl_seconds":
		return rp.DefaultTTLSeconds
	case "cleanup_interval_seconds":
		return rp.CleanupIntervalSeconds
	case "max_neighbors":
		return rp.MaxNeighbors
	case "min_neighbors":
		return rp.MinNeighbors
	case "adaptation_enabled":
		return rp.AdaptationEnabled
	case "learning_rate":
		return rp.LearningRate
	default:
		return nil
	}
}

// Set updates a parameter value (thread-safe)
func (rp *RuntimeParameters) Set(param string, value interface{}) bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	switch param {
	case "energy_decay_rate":
		if v, ok := value.(float64); ok {
			rp.EnergyDecayRate = v
			return true
		}
	case "energy_decay_critical":
		if v, ok := value.(float64); ok {
			rp.EnergyDecayCritical = v
			return true
		}
	case "energy_decay_routine":
		if v, ok := value.(float64); ok {
			rp.EnergyDecayRoutine = v
			return true
		}
	case "default_energy_info":
		if v, ok := value.(int); ok {
			rp.DefaultEnergyInfo = v
			return true
		}
	case "default_energy_compute":
		if v, ok := value.(int); ok {
			rp.DefaultEnergyCompute = v
			return true
		}
	case "default_ttl_seconds":
		if v, ok := value.(int); ok {
			rp.DefaultTTLSeconds = v
			return true
		}
	case "cleanup_interval_seconds":
		if v, ok := value.(int); ok {
			rp.CleanupIntervalSeconds = v
			return true
		}
	case "max_neighbors":
		if v, ok := value.(int); ok {
			rp.MaxNeighbors = v
			return true
		}
	case "min_neighbors":
		if v, ok := value.(int); ok {
			rp.MinNeighbors = v
			return true
		}
	case "adaptation_enabled":
		if v, ok := value.(bool); ok {
			rp.AdaptationEnabled = v
			return true
		}
	case "learning_rate":
		if v, ok := value.(float64); ok {
			rp.LearningRate = v
			return true
		}
	}
	return false
}

// GetFloat64 returns a float64 parameter with default fallback
func (rp *RuntimeParameters) GetFloat64(param string, defaultValue float64) float64 {
	if v := rp.Get(param); v != nil {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return defaultValue
}

// GetInt returns an int parameter with default fallback
func (rp *RuntimeParameters) GetInt(param string, defaultValue int) int {
	if v := rp.Get(param); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return defaultValue
}

// GetBool returns a bool parameter with default fallback
func (rp *RuntimeParameters) GetBool(param string, defaultValue bool) bool {
	if v := rp.Get(param); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetDuration returns a time.Duration parameter with default fallback
func (rp *RuntimeParameters) GetDuration(param string, defaultValue time.Duration) time.Duration {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	switch param {
	case "neighbor_timeout":
		return rp.NeighborTimeout
	case "discovery_interval":
		return rp.DiscoveryInterval
	case "announce_interval":
		return rp.AnnounceInterval
	case "message_timeout":
		return rp.MessageTimeout
	case "retry_backoff":
		return rp.RetryBackoff
	case "modification_cooldown":
		return rp.ModificationCooldown
	}
	return defaultValue
}

// UpdateBatch allows updating multiple parameters atomically
func (rp *RuntimeParameters) UpdateBatch(updates map[string]interface{}) map[string]bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	results := make(map[string]bool)
	for param, value := range updates {
		// Temporarily unlock to use Set method
		rp.mu.Unlock()
		success := rp.Set(param, value)
		rp.mu.Lock()
		results[param] = success
	}
	return results
}

// Clone returns a deep copy of the parameters
func (rp *RuntimeParameters) Clone() *RuntimeParameters {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	return &RuntimeParameters{
		EnergyDecayRate:        rp.EnergyDecayRate,
		EnergyDecayCritical:    rp.EnergyDecayCritical,
		EnergyDecayRoutine:     rp.EnergyDecayRoutine,
		DefaultEnergyInfo:      rp.DefaultEnergyInfo,
		DefaultEnergyCompute:   rp.DefaultEnergyCompute,
		DefaultTTLSeconds:      rp.DefaultTTLSeconds,
		CleanupIntervalSeconds: rp.CleanupIntervalSeconds,
		CleanupBatchSize:       rp.CleanupBatchSize,
		MaxNeighbors:           rp.MaxNeighbors,
		MinNeighbors:           rp.MinNeighbors,
		NeighborTimeout:        rp.NeighborTimeout,
		DiscoveryInterval:      rp.DiscoveryInterval,
		AnnounceInterval:       rp.AnnounceInterval,
		MessageTimeout:         rp.MessageTimeout,
		RetryAttempts:          rp.RetryAttempts,
		RetryBackoff:           rp.RetryBackoff,
		MaxConcurrentTasks:     rp.MaxConcurrentTasks,
		LoadBalancingThreshold: rp.LoadBalancingThreshold,
		PerformanceWindow:      rp.PerformanceWindow,
		AdaptationEnabled:      rp.AdaptationEnabled,
		LearningRate:           rp.LearningRate,
		AdaptationThreshold:    rp.AdaptationThreshold,
		ModificationCooldown:   rp.ModificationCooldown,
	}
}
