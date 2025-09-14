package config

import (
	"time"

	"github.com/BasicAcid/ryx/internal/types"
)

// BehaviorModifier defines the interface for runtime behavior modification
type BehaviorModifier interface {
	// Energy and propagation behavior
	ModifyEnergyDecay(msg *types.InfoMessage, currentDecay float64) float64
	ModifyTTL(msgType string, currentTTL time.Duration) time.Duration
	ModifyForwardingDecision(msg *types.InfoMessage, neighbor *types.Neighbor) bool

	// Neighbor selection behavior
	ModifyNeighborPriority(neighbor *types.Neighbor, currentPriority float64) float64
	ShouldAddNeighbor(candidate *types.Neighbor, currentNeighbors []*types.Neighbor) bool
	ShouldRemoveNeighbor(neighbor *types.Neighbor, reason string) bool

	// Communication behavior
	ModifyRetryPolicy(targetNode string, attempt int, baseDelay time.Duration) time.Duration
	ModifyMessageTimeout(msgType string, baseTimeout time.Duration) time.Duration

	// Task execution behavior
	ModifyTaskPriority(task *types.ComputationTask, currentPriority int) int
	ShouldExecuteTask(task *types.ComputationTask, systemLoad float64) bool

	// Cleanup and maintenance behavior
	ModifyCleanupInterval(currentInterval time.Duration, systemLoad float64) time.Duration
	ShouldCleanupMessage(msg *types.InfoMessage, systemMemoryUsage float64) bool
}

// DefaultBehaviorModifier provides default behavior (no modifications)
type DefaultBehaviorModifier struct {
	params *RuntimeParameters
}

// NewDefaultBehaviorModifier creates a default behavior modifier
func NewDefaultBehaviorModifier(params *RuntimeParameters) *DefaultBehaviorModifier {
	return &DefaultBehaviorModifier{
		params: params,
	}
}

// ModifyEnergyDecay applies energy decay based on message type and system parameters
func (d *DefaultBehaviorModifier) ModifyEnergyDecay(msg *types.InfoMessage, currentDecay float64) float64 {
	// Use message-type specific decay rates
	switch msg.Type {
	case "critical", "emergency", "safety":
		return d.params.GetFloat64("energy_decay_critical", 0.5)
	case "routine", "info", "log":
		return d.params.GetFloat64("energy_decay_routine", 1.5)
	default:
		return d.params.GetFloat64("energy_decay_rate", 1.0)
	}
}

// ModifyTTL adjusts TTL based on message type
func (d *DefaultBehaviorModifier) ModifyTTL(msgType string, currentTTL time.Duration) time.Duration {
	switch msgType {
	case "critical", "emergency":
		// Critical messages live longer
		return currentTTL * 3
	case "routine", "temp":
		// Routine messages have shorter TTL
		return currentTTL / 2
	default:
		return currentTTL
	}
}

// ModifyForwardingDecision determines if a message should be forwarded to a neighbor
func (d *DefaultBehaviorModifier) ModifyForwardingDecision(msg *types.InfoMessage, neighbor *types.Neighbor) bool {
	// Always forward critical messages
	if msg.Type == "critical" || msg.Type == "emergency" {
		return true
	}

	// Don't forward if neighbor is overloaded (placeholder - would need real metrics)
	// This is where adaptive behavior would go
	return true
}

// ModifyNeighborPriority adjusts neighbor priority based on performance
func (d *DefaultBehaviorModifier) ModifyNeighborPriority(neighbor *types.Neighbor, currentPriority float64) float64 {
	// This is where we'd implement adaptive neighbor selection
	// For now, return current priority unchanged
	return currentPriority
}

// ShouldAddNeighbor determines if a new neighbor should be added
func (d *DefaultBehaviorModifier) ShouldAddNeighbor(candidate *types.Neighbor, currentNeighbors []*types.Neighbor) bool {
	maxNeighbors := d.params.GetInt("max_neighbors", 8)
	return len(currentNeighbors) < maxNeighbors
}

// ShouldRemoveNeighbor determines if a neighbor should be removed
func (d *DefaultBehaviorModifier) ShouldRemoveNeighbor(neighbor *types.Neighbor, reason string) bool {
	// Remove neighbors that have been unresponsive
	if reason == "timeout" || reason == "unreachable" {
		return true
	}
	return false
}

// ModifyRetryPolicy adjusts retry timing based on target node performance
func (d *DefaultBehaviorModifier) ModifyRetryPolicy(targetNode string, attempt int, baseDelay time.Duration) time.Duration {
	// Exponential backoff with some jitter
	multiplier := 1 << uint(attempt) // 1, 2, 4, 8...
	return time.Duration(int64(baseDelay) * int64(multiplier))
}

// ModifyMessageTimeout adjusts timeout based on message type
func (d *DefaultBehaviorModifier) ModifyMessageTimeout(msgType string, baseTimeout time.Duration) time.Duration {
	switch msgType {
	case "critical", "emergency":
		// Critical messages get longer timeout
		return baseTimeout * 2
	case "routine":
		// Routine messages get shorter timeout
		return baseTimeout / 2
	default:
		return baseTimeout
	}
}

// ModifyTaskPriority adjusts task priority based on type and system state
func (d *DefaultBehaviorModifier) ModifyTaskPriority(task *types.ComputationTask, currentPriority int) int {
	switch task.Type {
	case "safety", "emergency":
		return 100 // Highest priority
	case "critical":
		return 80
	case "normal":
		return 50
	case "background":
		return 10
	default:
		return currentPriority
	}
}

// ShouldExecuteTask determines if a task should be executed now
func (d *DefaultBehaviorModifier) ShouldExecuteTask(task *types.ComputationTask, systemLoad float64) bool {
	loadThreshold := d.params.GetFloat64("load_balancing_threshold", 0.8)

	// Always execute critical tasks
	if task.Type == "critical" || task.Type == "emergency" || task.Type == "safety" {
		return true
	}

	// Don't execute low-priority tasks if system is overloaded
	if systemLoad > loadThreshold && (task.Type == "background" || task.Type == "routine") {
		return false
	}

	return true
}

// ModifyCleanupInterval adjusts cleanup frequency based on system load
func (d *DefaultBehaviorModifier) ModifyCleanupInterval(currentInterval time.Duration, systemLoad float64) time.Duration {
	if systemLoad > 0.9 {
		// Clean up more frequently when system is stressed
		return currentInterval / 2
	} else if systemLoad < 0.3 {
		// Clean up less frequently when system is idle
		return currentInterval * 2
	}
	return currentInterval
}

// ShouldCleanupMessage determines if a message should be cleaned up
func (d *DefaultBehaviorModifier) ShouldCleanupMessage(msg *types.InfoMessage, systemMemoryUsage float64) bool {
	// Never clean up critical messages prematurely
	if msg.Type == "critical" || msg.Type == "emergency" || msg.Type == "safety" {
		return false
	}

	// Clean up routine messages more aggressively if memory is tight
	if systemMemoryUsage > 0.8 && (msg.Type == "routine" || msg.Type == "temp") {
		return true
	}

	// Use default TTL-based cleanup
	return time.Now().Unix() > msg.TTL
}

// AdaptiveBehaviorModifier extends the default with learning capabilities
type AdaptiveBehaviorModifier struct {
	*DefaultBehaviorModifier

	// Performance tracking
	neighborPerformance map[string]float64
	messageSuccessRates map[string]float64
	adaptationEnabled   bool
	lastModification    time.Time
}

// NewAdaptiveBehaviorModifier creates an adaptive behavior modifier
func NewAdaptiveBehaviorModifier(params *RuntimeParameters) *AdaptiveBehaviorModifier {
	return &AdaptiveBehaviorModifier{
		DefaultBehaviorModifier: NewDefaultBehaviorModifier(params),
		neighborPerformance:     make(map[string]float64),
		messageSuccessRates:     make(map[string]float64),
		adaptationEnabled:       params.GetBool("adaptation_enabled", true),
		lastModification:        time.Now(),
	}
}

// ModifyNeighborPriority with adaptive learning
func (a *AdaptiveBehaviorModifier) ModifyNeighborPriority(neighbor *types.Neighbor, currentPriority float64) float64 {
	if !a.adaptationEnabled {
		return a.DefaultBehaviorModifier.ModifyNeighborPriority(neighbor, currentPriority)
	}

	// Adjust priority based on historical performance
	if performance, exists := a.neighborPerformance[neighbor.NodeID]; exists {
		learningRate := a.params.GetFloat64("learning_rate", 0.1)
		return currentPriority + (performance-0.5)*learningRate
	}

	return currentPriority
}

// RecordNeighborPerformance updates performance metrics for adaptive behavior
func (a *AdaptiveBehaviorModifier) RecordNeighborPerformance(nodeID string, latency time.Duration, success bool) {
	if !a.adaptationEnabled {
		return
	}

	// Convert performance to 0-1 scale (lower latency = higher performance)
	performance := 1.0 / (1.0 + float64(latency.Milliseconds())/1000.0)
	if !success {
		performance *= 0.5 // Penalty for failures
	}

	// Update with exponential moving average
	if existing, exists := a.neighborPerformance[nodeID]; exists {
		alpha := a.params.GetFloat64("learning_rate", 0.1)
		a.neighborPerformance[nodeID] = alpha*performance + (1-alpha)*existing
	} else {
		a.neighborPerformance[nodeID] = performance
	}
}
