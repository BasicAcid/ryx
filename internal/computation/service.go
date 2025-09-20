package computation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/config"
	"github.com/BasicAcid/ryx/internal/types"
)

// queuedTask represents a task waiting for execution due to load constraints
type queuedTask struct {
	msg      *types.InfoMessage
	task     *types.ComputationTask
	executor types.TaskExecutor
	queued   time.Time
}

// Service manages distributed computation execution
type Service struct {
	nodeID    string
	executors map[string]types.TaskExecutor
	active    map[string]*types.ComputationResult // active computations
	completed map[string]*types.ComputationResult // completed computations
	diffusion types.InfoMessageHandler            // for result propagation
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Phase 3B: Advanced load-based optimization
	runtimeParams *config.RuntimeParameters
	behaviorMod   config.BehaviorModifier
	taskQueue     []*queuedTask
	queueMu       sync.RWMutex
}

// New creates a new computation service
func New(nodeID string) *Service {
	log.Printf("Creating computation service for node %s", nodeID)

	s := &Service{
		nodeID:    nodeID,
		executors: make(map[string]types.TaskExecutor),
		active:    make(map[string]*types.ComputationResult),
		completed: make(map[string]*types.ComputationResult),
	}

	// Register built-in executors
	s.registerExecutor(&WordCountExecutor{})

	return s
}

// NewWithConfig creates a new computation service with runtime configuration
func NewWithConfig(nodeID string, params *config.RuntimeParameters, behaviorMod config.BehaviorModifier) *Service {
	log.Printf("Creating computation service for node %s with adaptive behavior", nodeID)

	s := &Service{
		nodeID:    nodeID,
		executors: make(map[string]types.TaskExecutor),
		active:    make(map[string]*types.ComputationResult),
		completed: make(map[string]*types.ComputationResult),

		// Phase 3B: Advanced configuration
		runtimeParams: params,
		behaviorMod:   behaviorMod,
		taskQueue:     make([]*queuedTask, 0),
	}

	// Register built-in executors
	s.registerExecutor(&WordCountExecutor{})

	return s
}

// Start begins the computation service
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	log.Printf("Computation service starting for node %s", s.nodeID)

	// Start cleanup routine for completed computations
	go s.cleanupLoop()

	// Phase 3B: Start task queue processor
	go s.taskQueueLoop()

	return nil
}

// Stop shuts down the computation service
func (s *Service) Stop() {
	log.Printf("Computation service stopping for node %s", s.nodeID)
	if s.cancel != nil {
		s.cancel()
	}
}

// SetDiffusionService injects the diffusion service for result propagation
func (s *Service) SetDiffusionService(diffusion types.InfoMessageHandler) {
	s.diffusion = diffusion
}

// registerExecutor adds a task executor to the service
func (s *Service) registerExecutor(executor types.TaskExecutor) {
	s.executors[executor.GetTaskType()] = executor
	log.Printf("Registered executor for task type: %s", executor.GetTaskType())
}

// ExecuteTask processes a computational task from an InfoMessage
func (s *Service) ExecuteTask(msg *types.InfoMessage) error {
	if msg.Type != "task" {
		return fmt.Errorf("message type %s is not a computational task", msg.Type)
	}

	log.Printf("ExecuteTask: received task message id=%s", msg.ID)

	// Check if we've already processed this task
	s.mu.RLock()
	if _, exists := s.active[msg.ID]; exists {
		s.mu.RUnlock()
		log.Printf("Task %s already active, skipping", msg.ID)
		return nil
	}
	if _, exists := s.completed[msg.ID]; exists {
		s.mu.RUnlock()
		log.Printf("Task %s already completed, skipping", msg.ID)
		return nil
	}
	s.mu.RUnlock()

	// Parse task from message content
	var task types.ComputationTask
	if err := json.Unmarshal(msg.Content, &task); err != nil {
		return fmt.Errorf("failed to parse task: %w", err)
	}

	// Find appropriate executor
	executor, exists := s.executors[task.Type]
	if !exists {
		log.Printf("No executor available for task type: %s", task.Type)
		return nil // Not an error, just not supported on this node
	}

	// Phase 3B: Load-based scheduling decision
	if s.behaviorMod != nil {
		systemLoad := s.getCurrentSystemLoad()
		if !s.behaviorMod.ShouldExecuteTask(&task, systemLoad) {
			log.Printf("Task %s queued due to system load (%.2f)", msg.ID, systemLoad)
			s.queueTask(msg, &task, executor)
			return nil
		}
	}

	// Mark task as active
	s.mu.Lock()
	s.active[msg.ID] = &types.ComputationResult{
		TaskID:     msg.ID,
		TaskType:   task.Type,
		ExecutedBy: s.nodeID,
		Timestamp:  time.Now().Unix(),
	}
	s.mu.Unlock()

	// Execute task asynchronously
	go s.executeTaskAsync(msg.ID, &task, executor)

	return nil
}

// Phase 3B: Load-based scheduling methods

// getCurrentSystemLoad estimates current system load
func (s *Service) getCurrentSystemLoad() float64 {
	// Get active task count as a simple load metric
	s.mu.RLock()
	activeTasks := len(s.active)
	s.mu.RUnlock()

	// Calculate load based on active tasks vs capacity
	maxTasks := 10
	if s.runtimeParams != nil {
		maxTasks = s.runtimeParams.GetInt("max_concurrent_tasks", 10)
	}

	taskLoad := float64(activeTasks) / float64(maxTasks)

	// If we have advanced behavior modifier, get real system metrics
	if adaptiveMod, ok := s.behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		return adaptiveMod.GetSystemLoad()
	}

	return taskLoad
}

// queueTask adds a task to the execution queue
func (s *Service) queueTask(msg *types.InfoMessage, task *types.ComputationTask, executor types.TaskExecutor) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	qTask := &queuedTask{
		msg:      msg,
		task:     task,
		executor: executor,
		queued:   time.Now(),
	}

	// Insert task in priority order (critical tasks first)
	inserted := false
	for i, existing := range s.taskQueue {
		if s.getTaskPriority(task) > s.getTaskPriority(existing.task) {
			// Insert before this task
			s.taskQueue = append(s.taskQueue[:i], append([]*queuedTask{qTask}, s.taskQueue[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		s.taskQueue = append(s.taskQueue, qTask)
	}

	log.Printf("Task %s queued (position %d, type: %s)", msg.ID, len(s.taskQueue), task.Type)
}

// getTaskPriority returns numeric priority for task scheduling
func (s *Service) getTaskPriority(task *types.ComputationTask) int {
	if s.behaviorMod != nil {
		return s.behaviorMod.ModifyTaskPriority(task, 50) // Default priority 50
	}

	// Fallback priority assignment
	switch task.Type {
	case "safety", "emergency":
		return 100
	case "critical":
		return 80
	case "high":
		return 60
	case "normal":
		return 50
	case "background":
		return 10
	default:
		return 30
	}
}

// processTaskQueue processes queued tasks when load allows
func (s *Service) processTaskQueue() {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	if len(s.taskQueue) == 0 {
		return
	}

	systemLoad := s.getCurrentSystemLoad()

	// Process high-priority tasks even under moderate load
	for i := 0; i < len(s.taskQueue); i++ {
		qTask := s.taskQueue[i]

		shouldExecute := false
		if s.behaviorMod != nil {
			shouldExecute = s.behaviorMod.ShouldExecuteTask(qTask.task, systemLoad)
		} else {
			shouldExecute = systemLoad < 0.8 // Simple fallback
		}

		if shouldExecute {
			// Remove from queue
			s.taskQueue = append(s.taskQueue[:i], s.taskQueue[i+1:]...)
			i-- // Adjust index after removal

			// Mark as active
			s.mu.Lock()
			s.active[qTask.msg.ID] = &types.ComputationResult{
				TaskID:     qTask.msg.ID,
				TaskType:   qTask.task.Type,
				ExecutedBy: s.nodeID,
				Timestamp:  time.Now().Unix(),
			}
			s.mu.Unlock()

			log.Printf("Task %s dequeued and starting execution (waited %v)",
				qTask.msg.ID, time.Since(qTask.queued))

			// Execute asynchronously
			go s.executeTaskAsync(qTask.msg.ID, qTask.task, qTask.executor)

			// Only process one task per cycle to avoid overwhelming the system
			break
		}
	}
}

// taskQueueLoop periodically processes queued tasks
func (s *Service) taskQueueLoop() {
	ticker := time.NewTicker(5 * time.Second) // Check queue every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Task queue loop stopping for node %s", s.nodeID)
			return
		case <-ticker.C:
			s.processTaskQueue()

			// Update system metrics if we have advanced behavior modifier
			if adaptiveMod, ok := s.behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
				s.updateSystemMetrics(adaptiveMod)
			}
		}
	}
}

// updateSystemMetrics updates system performance metrics for adaptive behavior
func (s *Service) updateSystemMetrics(adaptiveMod *config.AdaptiveBehaviorModifier) {
	s.mu.RLock()
	activeTasks := len(s.active)
	s.mu.RUnlock()

	s.queueMu.RLock()
	queuedTasks := len(s.taskQueue)
	s.queueMu.RUnlock()

	// Simple CPU and memory usage estimation (would be better with real metrics)
	cpuUsage := float64(activeTasks) / 10.0 // Assume max 10 tasks = 100% CPU
	memoryUsage := cpuUsage * 0.8           // Assume similar memory usage pattern

	// Update metrics in behavior modifier
	adaptiveMod.UpdateSystemMetrics(cpuUsage, memoryUsage, activeTasks, queuedTasks)
}

// executeTaskAsync performs the actual task execution
func (s *Service) executeTaskAsync(taskID string, task *types.ComputationTask, executor types.TaskExecutor) {
	log.Printf("Starting execution of task %s (type: %s)", taskID, task.Type)
	startTime := time.Now()

	// Execute the computation
	result, err := executor.Execute(task)
	executionTime := time.Since(startTime).Milliseconds()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from active
	delete(s.active, taskID)

	if err != nil {
		log.Printf("Task %s execution failed: %v", taskID, err)
		return
	}

	// Update result with execution metadata
	result.TaskID = taskID
	result.ExecutedBy = s.nodeID
	result.ExecutionTime = executionTime
	result.Timestamp = time.Now().Unix()

	// Store completed result
	s.completed[taskID] = result

	log.Printf("Completed task %s in %dms", taskID, executionTime)

	// Propagate result through diffusion system if available (async to prevent deadlock)
	if s.diffusion != nil {
		go s.propagateResult(result)
	}
}

// propagateResult sends the computation result through the diffusion system
func (s *Service) propagateResult(result *types.ComputationResult) {
	// Serialize result
	resultData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to serialize result: %v", err)
		return
	}

	// Create result message with low energy (local propagation only)
	resultMsg := &types.InfoMessage{
		ID:        generateResultID(result),
		Type:      "result",
		Content:   resultData,
		Energy:    1, // Low energy for result sharing
		TTL:       time.Now().Add(5 * time.Minute).Unix(),
		Hops:      0,
		Source:    s.nodeID,
		Path:      []string{s.nodeID},
		Timestamp: time.Now().Unix(),
		Metadata: map[string]interface{}{
			"task_id":   result.TaskID,
			"task_type": result.TaskType,
			"executor":  result.ExecutedBy,
		},
	}

	// Send through diffusion
	err = s.diffusion.HandleInfoMessage(resultMsg, s.nodeID)
	if err != nil {
		log.Printf("Failed to propagate result: %v", err)
	} else {
		log.Printf("Propagated result for task %s", result.TaskID)
	}
}

// GetActiveComputations returns all currently active computations
func (s *Service) GetActiveComputations() map[string]*types.ComputationResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*types.ComputationResult)
	for k, v := range s.active {
		resultCopy := *v
		result[k] = &resultCopy
	}
	return result
}

// GetComputationResult returns a specific computation result
func (s *Service) GetComputationResult(taskID string) (*types.ComputationResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check completed first
	if result, exists := s.completed[taskID]; exists {
		resultCopy := *result
		return &resultCopy, true
	}

	// Check active
	if result, exists := s.active[taskID]; exists {
		resultCopy := *result
		return &resultCopy, true
	}

	return nil, false
}

// GetComputationStats returns computation statistics
func (s *Service) GetComputationStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"node_id":             s.nodeID,
		"active_tasks":        len(s.active),
		"completed_tasks":     len(s.completed),
		"available_executors": getExecutorTypes(s.executors),
	}
}

// cleanupLoop removes old completed computations
func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes completed computations older than 10 minutes
func (s *Service) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute).Unix()
	removed := 0

	for taskID, result := range s.completed {
		if result.Timestamp < cutoff {
			delete(s.completed, taskID)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("Cleaned up %d old computation results", removed)
	}
}

// generateResultID creates a unique ID for computation results
func generateResultID(result *types.ComputationResult) string {
	// Combine task ID and executor node for unique result ID
	data := fmt.Sprintf("%s:%s", result.TaskID, result.ExecutedBy)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// getExecutorTypes returns list of available executor types
func getExecutorTypes(executors map[string]types.TaskExecutor) []string {
	types := make([]string, 0, len(executors))
	for taskType := range executors {
		types = append(types, taskType)
	}
	return types
}

// WordCountExecutor implements word counting computation
type WordCountExecutor struct{}

// Execute performs word counting on the input data
func (e *WordCountExecutor) Execute(task *types.ComputationTask) (*types.ComputationResult, error) {
	// Get case sensitivity parameter
	caseSensitive := false
	if cs, ok := task.Parameters["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	// Process the text
	text := task.Data
	if !caseSensitive {
		text = strings.ToLower(text)
	}

	// Split into words and count
	words := strings.Fields(text)
	totalWords := len(words)
	wordCounts := make(map[string]int)

	for _, word := range words {
		// Remove basic punctuation
		word = strings.Trim(word, ".,!?;:\"'")
		if word != "" {
			wordCounts[word]++
		}
	}

	// Create result
	result := &types.ComputationResult{
		TaskType: "wordcount",
		Result: map[string]interface{}{
			"total_words":  totalWords,
			"unique_words": len(wordCounts),
			"word_counts":  wordCounts,
		},
	}

	return result, nil
}

// CanHandle returns true if this executor can handle the task type
func (e *WordCountExecutor) CanHandle(taskType string) bool {
	return taskType == "wordcount"
}

// GetTaskType returns the task type this executor handles
func (e *WordCountExecutor) GetTaskType() string {
	return "wordcount"
}
