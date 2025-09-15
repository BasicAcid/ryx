package config

import (
	"math"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/spatial"
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

// SystemMetrics tracks real-time system performance
type SystemMetrics struct {
	CPUUsage       float64          `json:"cpu_usage"`
	MemoryUsage    float64          `json:"memory_usage"`
	ActiveTasks    int              `json:"active_tasks"`
	MessageLoad    int              `json:"message_load"`
	NetworkLatency map[string]int64 `json:"network_latency"` // nodeID -> latency_ms
	Timestamp      time.Time        `json:"timestamp"`
}

// FaultPattern tracks failure patterns for adaptive routing
type FaultPattern struct {
	NodeID        string         `json:"node_id"`
	FailureTypes  map[string]int `json:"failure_types"` // message_type -> failure_count
	TotalFailures int            `json:"total_failures"`
	LastFailure   time.Time      `json:"last_failure"`
	RecoveryTests []time.Time    `json:"recovery_tests"`
	SuccessRate   float64        `json:"success_rate"`
}

// AdaptiveBehaviorModifier extends the default with learning capabilities
type AdaptiveBehaviorModifier struct {
	*DefaultBehaviorModifier

	// Performance tracking
	neighborPerformance map[string]float64
	messageSuccessRates map[string]float64
	adaptationEnabled   bool
	lastModification    time.Time

	// Advanced Phase 3B features
	systemMetrics       *SystemMetrics
	faultPatterns       map[string]*FaultPattern
	neighborLatency     map[string][]int64   // nodeID -> latency samples (sliding window)
	neighborReliability map[string]float64   // nodeID -> success rate
	loadHistory         []float64            // sliding window of system load
	adaptationHistory   map[string][]float64 // parameter -> historical values
	metricsLock         sync.RWMutex
}

// NewAdaptiveBehaviorModifier creates an adaptive behavior modifier
func NewAdaptiveBehaviorModifier(params *RuntimeParameters) *AdaptiveBehaviorModifier {
	return &AdaptiveBehaviorModifier{
		DefaultBehaviorModifier: NewDefaultBehaviorModifier(params),
		neighborPerformance:     make(map[string]float64),
		messageSuccessRates:     make(map[string]float64),
		adaptationEnabled:       params.GetBool("adaptation_enabled", true),
		lastModification:        time.Now(),

		// Advanced Phase 3B features
		systemMetrics:       &SystemMetrics{Timestamp: time.Now()},
		faultPatterns:       make(map[string]*FaultPattern),
		neighborLatency:     make(map[string][]int64),
		neighborReliability: make(map[string]float64),
		loadHistory:         make([]float64, 0, 100), // 100-sample sliding window
		adaptationHistory:   make(map[string][]float64),
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

	// Phase 3B: Record latency and reliability metrics
	a.recordLatencyMetric(nodeID, latency)
	a.updateReliabilityMetric(nodeID, success)
}

// Phase 3B: Advanced network-aware adaptation methods

// recordLatencyMetric adds latency sample to sliding window
func (a *AdaptiveBehaviorModifier) recordLatencyMetric(nodeID string, latency time.Duration) {
	a.metricsLock.Lock()
	defer a.metricsLock.Unlock()

	latencyMs := latency.Milliseconds()
	samples := a.neighborLatency[nodeID]

	// Maintain sliding window of last 20 samples
	if len(samples) >= 20 {
		samples = samples[1:]
	}
	samples = append(samples, latencyMs)
	a.neighborLatency[nodeID] = samples
}

// updateReliabilityMetric updates success rate using exponential moving average
func (a *AdaptiveBehaviorModifier) updateReliabilityMetric(nodeID string, success bool) {
	a.metricsLock.Lock()
	defer a.metricsLock.Unlock()

	successValue := 0.0
	if success {
		successValue = 1.0
	}

	if existing, exists := a.neighborReliability[nodeID]; exists {
		alpha := a.params.GetFloat64("learning_rate", 0.1)
		a.neighborReliability[nodeID] = alpha*successValue + (1-alpha)*existing
	} else {
		a.neighborReliability[nodeID] = successValue
	}
}

// getAverageLatency calculates average latency for a neighbor
func (a *AdaptiveBehaviorModifier) getAverageLatency(nodeID string) float64 {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	samples := a.neighborLatency[nodeID]
	if len(samples) == 0 {
		return 100.0 // Default 100ms if no data
	}

	var sum int64
	for _, sample := range samples {
		sum += sample
	}
	return float64(sum) / float64(len(samples))
}

// getSuccessRate returns success rate for a neighbor
func (a *AdaptiveBehaviorModifier) getSuccessRate(nodeID string) float64 {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	if rate, exists := a.neighborReliability[nodeID]; exists {
		return rate
	}
	return 0.95 // Default 95% success rate if no data
}

// Phase 3B: Enhanced network-aware energy decay with target neighbor awareness
func (a *AdaptiveBehaviorModifier) ModifyEnergyDecayForNeighbor(msg *types.InfoMessage, currentDecay float64, targetNeighbor string) float64 {
	if !a.adaptationEnabled {
		return a.DefaultBehaviorModifier.ModifyEnergyDecay(msg, currentDecay)
	}

	// Get base decay from default behavior
	baseDecay := a.DefaultBehaviorModifier.ModifyEnergyDecay(msg, currentDecay)

	// Phase 3B: Network-aware adaptation
	neighborLatency := a.getAverageLatency(targetNeighbor)
	neighborSuccessRate := a.getSuccessRate(targetNeighbor)

	// Calculate adaptive factors
	// Latency penalty: High latency neighbors get higher energy decay (max 2x)
	latencyPenalty := math.Min(neighborLatency/500.0, 2.0) // 500ms baseline, max 2x penalty

	// Reliability penalty: Unreliable neighbors get higher energy decay
	reliabilityPenalty := (1.0 - neighborSuccessRate) * 1.5 // Max 1.5x penalty for 0% success

	// Network adaptation factor
	networkFactor := 1.0 + latencyPenalty*0.3 + reliabilityPenalty*0.4

	// Apply adaptive energy decay
	adaptiveDecay := baseDecay * networkFactor

	// Ensure reasonable bounds (0.1 to 5.0)
	adaptiveDecay = math.Max(0.1, math.Min(5.0, adaptiveDecay))

	return adaptiveDecay
}

// Phase 3B: Fault pattern learning and adaptive routing

// RecordCommunicationFailure records a failure for fault pattern learning
func (a *AdaptiveBehaviorModifier) RecordCommunicationFailure(nodeID string, messageType string, failureReason string) {
	if !a.adaptationEnabled {
		return
	}

	a.metricsLock.Lock()
	defer a.metricsLock.Unlock()

	pattern, exists := a.faultPatterns[nodeID]
	if !exists {
		pattern = &FaultPattern{
			NodeID:       nodeID,
			FailureTypes: make(map[string]int),
			SuccessRate:  0.95, // Start with optimistic assumption
		}
		a.faultPatterns[nodeID] = pattern
	}

	// Record failure by message type
	pattern.FailureTypes[messageType]++
	pattern.TotalFailures++
	pattern.LastFailure = time.Now()

	// Update success rate with exponential moving average
	alpha := a.params.GetFloat64("learning_rate", 0.1)
	pattern.SuccessRate = (1-alpha)*pattern.SuccessRate + alpha*0.0 // Failure = 0.0
}

// RecordCommunicationSuccess records successful communication
func (a *AdaptiveBehaviorModifier) RecordCommunicationSuccess(nodeID string) {
	if !a.adaptationEnabled {
		return
	}

	a.metricsLock.Lock()
	defer a.metricsLock.Unlock()

	pattern, exists := a.faultPatterns[nodeID]
	if !exists {
		pattern = &FaultPattern{
			NodeID:       nodeID,
			FailureTypes: make(map[string]int),
			SuccessRate:  0.95,
		}
		a.faultPatterns[nodeID] = pattern
	}

	// Update success rate with exponential moving average
	alpha := a.params.GetFloat64("learning_rate", 0.1)
	pattern.SuccessRate = (1-alpha)*pattern.SuccessRate + alpha*1.0 // Success = 1.0
}

// Enhanced forwarding decision with fault pattern awareness
func (a *AdaptiveBehaviorModifier) ModifyForwardingDecision(msg *types.InfoMessage, neighbor *types.Neighbor) bool {
	// Use default behavior first
	if !a.DefaultBehaviorModifier.ModifyForwardingDecision(msg, neighbor) {
		return false
	}

	if !a.adaptationEnabled {
		return true
	}

	// Phase 3B: Fault-aware forwarding
	a.metricsLock.RLock()
	pattern, exists := a.faultPatterns[neighbor.NodeID]
	a.metricsLock.RUnlock()

	if !exists {
		return true // No failure history, proceed
	}

	// Critical messages always try (mission-critical requirement)
	if msg.Type == "critical" || msg.Type == "emergency" || msg.Type == "safety" {
		return true
	}

	// Check recent failure patterns for this message type
	recentFailures := pattern.FailureTypes[msg.Type]
	timeSinceLastFailure := time.Since(pattern.LastFailure)

	// Adaptive routing logic
	if recentFailures > 3 && timeSinceLastFailure < 5*time.Minute {
		// Route around consistently failing nodes for non-critical messages
		return false
	}

	// Check overall success rate
	if pattern.SuccessRate < 0.5 && timeSinceLastFailure < 2*time.Minute {
		// Skip unreliable neighbors for routine messages
		return msg.Type == "critical" || msg.Type == "emergency"
	}

	return true
}

// Phase 3B: System load monitoring and adaptive parameter tuning

// UpdateSystemMetrics updates current system performance metrics
func (a *AdaptiveBehaviorModifier) UpdateSystemMetrics(cpuUsage, memoryUsage float64, activeTasks, messageLoad int) {
	a.metricsLock.Lock()
	defer a.metricsLock.Unlock()

	a.systemMetrics.CPUUsage = cpuUsage
	a.systemMetrics.MemoryUsage = memoryUsage
	a.systemMetrics.ActiveTasks = activeTasks
	a.systemMetrics.MessageLoad = messageLoad
	a.systemMetrics.Timestamp = time.Now()

	// Update load history for trending analysis
	currentLoad := math.Max(cpuUsage, memoryUsage) // Overall system load
	if len(a.loadHistory) >= 100 {
		a.loadHistory = a.loadHistory[1:] // Remove oldest
	}
	a.loadHistory = append(a.loadHistory, currentLoad)
}

// GetSystemLoad returns current system load (0.0 to 1.0)
func (a *AdaptiveBehaviorModifier) GetSystemLoad() float64 {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	return math.Max(a.systemMetrics.CPUUsage, a.systemMetrics.MemoryUsage)
}

// GetLoadTrend returns load trend (-1.0 to 1.0, negative = decreasing load)
func (a *AdaptiveBehaviorModifier) GetLoadTrend() float64 {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	if len(a.loadHistory) < 10 {
		return 0.0 // Not enough data
	}

	// Simple trend calculation: compare recent average with older average
	recentCount := 5
	recent := a.loadHistory[len(a.loadHistory)-recentCount:]
	older := a.loadHistory[len(a.loadHistory)-2*recentCount : len(a.loadHistory)-recentCount]

	recentAvg := average(recent)
	olderAvg := average(older)

	// Normalize trend to -1.0 to 1.0 range
	trend := (recentAvg - olderAvg) / math.Max(olderAvg, 0.1)
	return math.Max(-1.0, math.Min(1.0, trend))
}

// Enhanced task execution decision with load-based scheduling
func (a *AdaptiveBehaviorModifier) ShouldExecuteTask(task *types.ComputationTask, systemLoad float64) bool {
	// Use default behavior first
	if !a.DefaultBehaviorModifier.ShouldExecuteTask(task, systemLoad) {
		return false
	}

	if !a.adaptationEnabled {
		return true
	}

	// Phase 3B: Advanced load-based scheduling
	currentLoad := a.GetSystemLoad()
	loadTrend := a.GetLoadTrend()

	// Mission-critical tasks always execute
	if task.Type == "critical" || task.Type == "emergency" || task.Type == "safety" {
		return true
	}

	// Adaptive scheduling based on load and trend
	loadThreshold := a.params.GetFloat64("load_balancing_threshold", 0.8)

	// Adjust threshold based on load trend
	if loadTrend > 0.5 { // Load increasing rapidly
		loadThreshold -= 0.1 // Be more conservative
	} else if loadTrend < -0.5 { // Load decreasing rapidly
		loadThreshold += 0.1 // Be more aggressive
	}

	// High-priority tasks get preference under moderate load
	if currentLoad < loadThreshold*1.2 && task.Type == "high" {
		return true
	}

	// Normal and background tasks wait if system is stressed
	if currentLoad > loadThreshold {
		return false
	}

	return true
}

// Adaptive cleanup interval based on system load and memory pressure
func (a *AdaptiveBehaviorModifier) ModifyCleanupInterval(currentInterval time.Duration, systemLoad float64) time.Duration {
	baseInterval := a.DefaultBehaviorModifier.ModifyCleanupInterval(currentInterval, systemLoad)

	if !a.adaptationEnabled {
		return baseInterval
	}

	// Phase 3B: Advanced cleanup adaptation
	memoryPressure := a.systemMetrics.MemoryUsage
	loadTrend := a.GetLoadTrend()

	// Base adaptation factor
	adaptationFactor := 1.0

	// Memory pressure factor
	if memoryPressure > 0.9 {
		adaptationFactor *= 0.3 // Clean 3x more frequently
	} else if memoryPressure > 0.8 {
		adaptationFactor *= 0.5 // Clean 2x more frequently
	} else if memoryPressure < 0.3 {
		adaptationFactor *= 2.0 // Clean half as frequently
	}

	// Load trend factor
	if loadTrend > 0.7 { // Load increasing rapidly
		adaptationFactor *= 0.7 // Clean more frequently to free resources
	}

	// Message load factor
	if a.systemMetrics.MessageLoad > 1000 {
		adaptationFactor *= 0.6 // High message volume needs frequent cleanup
	}

	adaptedInterval := time.Duration(float64(baseInterval) * adaptationFactor)

	// Ensure reasonable bounds (5 seconds to 10 minutes)
	minInterval := 5 * time.Second
	maxInterval := 10 * time.Minute
	return time.Duration(math.Max(float64(minInterval), math.Min(float64(maxInterval), float64(adaptedInterval))))
}

// Phase 3B: Performance-based neighbor selection and topology optimization

// CalculateNeighborScore computes a composite score for neighbor quality
func (a *AdaptiveBehaviorModifier) CalculateNeighborScore(neighborID string) float64 {
	if !a.adaptationEnabled {
		return 0.7 // Default neutral score
	}

	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	// Get metrics (with defaults for new neighbors)
	performance := a.neighborPerformance[neighborID]
	if performance == 0 {
		performance = 0.5 // Neutral starting point
	}

	latency := a.getAverageLatency(neighborID)
	if latency == 0 {
		latency = 100.0 // Default 100ms
	}

	reliability := a.getSuccessRate(neighborID)

	// Normalize latency to 0-1 scale (lower latency = higher score)
	// Assume 1000ms is very poor, 10ms is excellent
	latencyScore := math.Max(0.0, math.Min(1.0, 1.0-(latency-10.0)/990.0))

	// Composite score: performance (40%) + latency (30%) + reliability (30%)
	score := 0.4*performance + 0.3*latencyScore + 0.3*reliability

	// Ensure score is in valid range
	return math.Max(0.0, math.Min(1.0, score))
}

// Phase 3C.2: Spatial-aware neighbor scoring
// CalculateNeighborScoreWithSpatial computes score including spatial factors
func (a *AdaptiveBehaviorModifier) CalculateNeighborScoreWithSpatial(neighborID string, neighborSpatialConfig *spatial.SpatialConfig, distance *spatial.Distance, nodeSpatialConfig *spatial.SpatialConfig) float64 {
	// Start with base network performance score
	networkScore := a.CalculateNeighborScore(neighborID)

	// If no spatial configuration available, return network score only
	if nodeSpatialConfig == nil || nodeSpatialConfig.IsEmpty() || neighborSpatialConfig == nil || neighborSpatialConfig.IsEmpty() {
		return networkScore
	}

	// Calculate spatial score
	spatialScore := a.calculateSpatialScore(nodeSpatialConfig, neighborSpatialConfig, distance)

	// Hybrid scoring: 60% network performance + 40% spatial factors
	hybridScore := 0.6*networkScore + 0.4*spatialScore

	return math.Max(0.0, math.Min(1.0, hybridScore))
}

// calculateSpatialScore computes spatial component of neighbor score
func (a *AdaptiveBehaviorModifier) calculateSpatialScore(nodeConfig, neighborConfig *spatial.SpatialConfig, distance *spatial.Distance) float64 {
	score := 0.5 // Base neutral score

	// Zone affinity bonus (same zone gets higher score)
	if spatial.IsInSameZone(nodeConfig, neighborConfig) {
		score += 0.3 // +30% for same zone neighbors
	}

	// Distance penalty/bonus
	if distance != nil && !math.IsInf(distance.Value, 1) {
		// Normalize distance score based on coordinate system
		distanceScore := a.calculateDistanceScore(distance)
		score += 0.2 * distanceScore // Distance contributes up to 20%
	}

	// Coordinate system compatibility
	if nodeConfig.CoordSystem == neighborConfig.CoordSystem {
		score += 0.1 // +10% for same coordinate system
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// calculateDistanceScore converts distance to a 0-1 score (closer = higher score)
func (a *AdaptiveBehaviorModifier) calculateDistanceScore(distance *spatial.Distance) float64 {
	switch distance.CoordSystem {
	case spatial.CoordSystemGPS:
		// For GPS: 0-1km = 1.0, 1-10km = 0.5-1.0, >50km = 0.0
		if distance.Value <= 1000 { // 1km
			return 1.0
		} else if distance.Value <= 10000 { // 10km
			return 1.0 - (distance.Value-1000)/9000*0.5 // Linear decay from 1.0 to 0.5
		} else if distance.Value <= 50000 { // 50km
			return 0.5 - (distance.Value-10000)/40000*0.5 // Linear decay from 0.5 to 0.0
		}
		return 0.0

	case spatial.CoordSystemRelative:
		// For relative: 0-10m = 1.0, 10-100m = 0.5-1.0, >500m = 0.0
		if distance.Value <= 10 {
			return 1.0
		} else if distance.Value <= 100 {
			return 1.0 - (distance.Value-10)/90*0.5
		} else if distance.Value <= 500 {
			return 0.5 - (distance.Value-100)/400*0.5
		}
		return 0.0

	case spatial.CoordSystemLogical:
		// For logical: same zone = 1.0, different zone = 0.2
		if distance.Value == 0 {
			return 1.0 // Same zone
		}
		return 0.2 // Different zone

	default:
		return 0.5 // Neutral for unknown systems
	}
}

// ShouldAddNeighbor with performance-based evaluation
func (a *AdaptiveBehaviorModifier) ShouldAddNeighbor(candidate *types.Neighbor, currentNeighbors []*types.Neighbor) bool {
	// Use default behavior first
	if !a.DefaultBehaviorModifier.ShouldAddNeighbor(candidate, currentNeighbors) {
		return false
	}

	if !a.adaptationEnabled {
		return true
	}

	maxNeighbors := a.params.GetInt("max_neighbors", 8)

	// If we're not at capacity, add the neighbor
	if len(currentNeighbors) < maxNeighbors {
		return true
	}

	// Phase 3B: Performance-based replacement
	candidateScore := a.CalculateNeighborScore(candidate.NodeID)

	// Find the worst performing current neighbor
	worstScore := 1.0
	for _, neighbor := range currentNeighbors {
		score := a.CalculateNeighborScore(neighbor.NodeID)
		if score < worstScore {
			worstScore = score
		}
	}

	// Replace worst neighbor if candidate is significantly better
	improvementThreshold := 0.2 // Require 20% improvement
	return candidateScore > worstScore+improvementThreshold
}

// ShouldRemoveNeighbor with performance-based evaluation
func (a *AdaptiveBehaviorModifier) ShouldRemoveNeighbor(neighbor *types.Neighbor, reason string) bool {
	// Use default behavior first
	if a.DefaultBehaviorModifier.ShouldRemoveNeighbor(neighbor, reason) {
		return true
	}

	if !a.adaptationEnabled {
		return false
	}

	// Phase 3B: Performance-based removal
	score := a.CalculateNeighborScore(neighbor.NodeID)

	// Remove consistently poor performers
	if score < 0.3 && reason == "poor_performance" {
		return true
	}

	// Check fault patterns
	a.metricsLock.RLock()
	pattern, exists := a.faultPatterns[neighbor.NodeID]
	a.metricsLock.RUnlock()

	if exists {
		// Remove neighbors with very low success rate
		if pattern.SuccessRate < 0.2 {
			return true
		}

		// Remove neighbors with excessive recent failures
		if pattern.TotalFailures > 20 && time.Since(pattern.LastFailure) < 1*time.Minute {
			return true
		}
	}

	return false
}

// GetNeighborMetrics returns comprehensive metrics for a neighbor (for monitoring/debugging)
func (a *AdaptiveBehaviorModifier) GetNeighborMetrics(neighborID string) map[string]interface{} {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	metrics := map[string]interface{}{
		"node_id": neighborID,
		"score":   a.CalculateNeighborScore(neighborID),
	}

	if performance, exists := a.neighborPerformance[neighborID]; exists {
		metrics["performance"] = performance
	}

	if samples := a.neighborLatency[neighborID]; len(samples) > 0 {
		metrics["average_latency_ms"] = a.getAverageLatency(neighborID)
		metrics["latency_samples"] = len(samples)
	}

	if reliability, exists := a.neighborReliability[neighborID]; exists {
		metrics["reliability"] = reliability
	}

	if pattern, exists := a.faultPatterns[neighborID]; exists {
		metrics["total_failures"] = pattern.TotalFailures
		metrics["success_rate"] = pattern.SuccessRate
		metrics["last_failure"] = pattern.LastFailure
	}

	return metrics
}

// GetSystemMetrics returns current system performance metrics (for monitoring/debugging)
func (a *AdaptiveBehaviorModifier) GetSystemMetrics() map[string]interface{} {
	a.metricsLock.RLock()
	defer a.metricsLock.RUnlock()

	return map[string]interface{}{
		"cpu_usage":         a.systemMetrics.CPUUsage,
		"memory_usage":      a.systemMetrics.MemoryUsage,
		"active_tasks":      a.systemMetrics.ActiveTasks,
		"message_load":      a.systemMetrics.MessageLoad,
		"load_trend":        a.GetLoadTrend(),
		"timestamp":         a.systemMetrics.Timestamp,
		"load_history_size": len(a.loadHistory),
	}
}

// Helper function to calculate average of float64 slice
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
