package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/BasicAcid/ryx/internal/computation"
	"github.com/BasicAcid/ryx/internal/config"
	"github.com/BasicAcid/ryx/internal/diffusion"
	"github.com/BasicAcid/ryx/internal/discovery"
	"github.com/BasicAcid/ryx/internal/spatial"
	"github.com/BasicAcid/ryx/internal/topology"
	"github.com/BasicAcid/ryx/internal/types"
)

// NodeStatusProvider interface for getting node status
type NodeStatusProvider interface {
	GetStatus() map[string]interface{}
	ID() string
}

// DiffusionProvider interface for accessing diffusion service
type DiffusionProvider interface {
	GetDiffusionService() *diffusion.Service
}

// ComputationProvider interface for accessing computation service
type ComputationProvider interface {
	GetComputationService() *computation.Service
}

// ConfigurationProvider interface for runtime parameter modification
type ConfigurationProvider interface {
	GetRuntimeParameters() *config.RuntimeParameters
	GetBehaviorModifier() config.BehaviorModifier
	UpdateParameters(updates map[string]interface{}) map[string]bool
}

// DiscoveryProvider interface for accessing discovery service
type DiscoveryProvider interface {
	GetDiscoveryService() *discovery.Service
}

// SpatialProvider interface for accessing spatial configuration and barriers
type SpatialProvider interface {
	GetSpatialConfig() *spatial.SpatialConfig
	UpdateSpatialConfig(config *spatial.SpatialConfig) error
	GetBarrierManager() *spatial.BarrierManager
	CalculateDistanceTo(otherConfig *spatial.SpatialConfig) (*spatial.Distance, error)
	IsPathBlocked(to *spatial.SpatialConfig, messageType string) bool
}

// TopologyProvider interface for accessing network topology information
type TopologyProvider interface {
	GetTopologyMapper() *topology.TopologyMapper
}

// NodeProvider combines all interfaces
type NodeProvider interface {
	NodeStatusProvider
	DiffusionProvider
	ComputationProvider
	ConfigurationProvider
	DiscoveryProvider
	SpatialProvider
	TopologyProvider
}

// Server provides HTTP API for node control and status
type Server struct {
	port   int
	node   NodeProvider
	server *http.Server
}

// New creates a new API server
func New(port int, node NodeProvider) (*Server, error) {
	return &Server{
		port: port,
		node: node,
	}, nil
}

// Start begins the HTTP API server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ping", s.handlePing)

	// Diffusion endpoints
	mux.HandleFunc("/inject", s.handleInject)
	mux.HandleFunc("/info", s.handleInfo)
	mux.HandleFunc("/info/", s.handleInfoByID)

	// Computation endpoints (Phase 2C)
	mux.HandleFunc("/compute", s.handleCompute)
	mux.HandleFunc("/compute/", s.handleComputeByID)

	// Configuration endpoints (Phase 3B - Self-modification)
	mux.HandleFunc("/config", s.handleConfig)
	mux.HandleFunc("/config/", s.handleConfigParameter)

	// Phase 3B: Advanced adaptive algorithm monitoring
	mux.HandleFunc("/adaptive/neighbors", s.handleNeighborMetrics)
	mux.HandleFunc("/adaptive/faults", s.handleFaultPatterns)
	mux.HandleFunc("/adaptive/system", s.handleSystemMetrics)

	// Phase 3C.1: Spatial configuration endpoints
	mux.HandleFunc("/spatial/position", s.handleSpatialPosition)
	mux.HandleFunc("/spatial/neighbors", s.handleSpatialNeighbors)
	mux.HandleFunc("/spatial/barriers", s.handleSpatialBarriers)
	mux.HandleFunc("/spatial/distance", s.handleSpatialDistance)

	// Phase 4A: Chemistry endpoints
	mux.HandleFunc("/chemistry/concentrations", s.handleChemistryConcentrations)
	mux.HandleFunc("/chemistry/reactions", s.handleChemistryReactions)
	mux.HandleFunc("/chemistry/stats", s.handleChemistryStats)

	// Phase 4B-Alt: Unified metrics endpoint (Prometheus standard)
	mux.HandleFunc("/metrics", s.handleUnifiedMetrics)

	// Phase 3C.3a: Topology mapping endpoints
	mux.HandleFunc("/topology/map", s.handleTopologyMap)
	mux.HandleFunc("/topology/zones", s.handleTopologyZones)
	mux.HandleFunc("/topology/live", s.handleTopologyLive)

	// Dashboard removed - use external ryx-dashboard application

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.enableCORS(mux),
	}

	log.Printf("HTTP API server starting on port %d", s.port)

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the API server
func (s *Server) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}
}

// handleStatus returns detailed node status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.node.GetStatus()

	// Add neighbor_count field for cluster tool compatibility
	if neighborsValue, exists := status["neighbors"]; exists {
		switch neighbors := neighborsValue.(type) {
		case []interface{}:
			status["neighbor_count"] = len(neighbors)
		case []map[string]interface{}:
			status["neighbor_count"] = len(neighbors)
		case []*types.Neighbor:
			status["neighbor_count"] = len(neighbors)
		case []types.Neighbor:
			status["neighbor_count"] = len(neighbors)
		default:
			// Use reflection as fallback
			v := reflect.ValueOf(neighbors)
			if v.Kind() == reflect.Slice {
				status["neighbor_count"] = v.Len()
			} else {
				status["neighbor_count"] = 0
			}
		}
	} else {
		status["neighbor_count"] = 0
	}

	s.writeJSON(w, status)
}

// handleHealth returns simple health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"node_id":   s.node.ID(),
		"timestamp": time.Now().Unix(),
	}
	s.writeJSON(w, response)
}

// handlePing returns simple ping response
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"pong":      true,
		"node_id":   s.node.ID(),
		"timestamp": time.Now().Unix(),
	}
	s.writeJSON(w, response)
}

// handleInject handles information injection requests
func (s *Server) handleInject(w http.ResponseWriter, r *http.Request) {
	// Add panic recovery
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("PANIC in handleInject: %v", recovered)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	log.Printf("handleInject: received %s request", r.Method)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var request struct {
		Type    string  `json:"type"`
		Content string  `json:"content"`
		Energy  float64 `json:"energy"`
		TTL     int     `json:"ttl"` // TTL in seconds
	}

	log.Printf("handleInject: parsing request body")
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("handleInject: JSON decode error: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("handleInject: parsed request - type=%s, content=%s, energy=%d, ttl=%d",
		request.Type, request.Content, request.Energy, request.TTL)

	// Validate request
	if request.Type == "" {
		request.Type = "text" // Default type
	}
	if request.Energy <= 0 {
		request.Energy = 10 // Default energy
	}
	if request.TTL <= 0 {
		request.TTL = 300 // Default TTL: 5 minutes
	}

	log.Printf("handleInject: validated request - type=%s, energy=%d, ttl=%d",
		request.Type, request.Energy, request.TTL)

	// Check if node is nil
	if s.node == nil {
		log.Printf("handleInject: ERROR - node is nil")
		http.Error(w, "Node not initialized", http.StatusInternalServerError)
		return
	}

	// Get diffusion service with nil check
	log.Printf("handleInject: getting diffusion service")
	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleInject: ERROR - diffusion service is nil")
		http.Error(w, "Diffusion service not initialized", http.StatusInternalServerError)
		return
	}

	log.Printf("handleInject: diffusion service obtained, injecting info")

	// Inject information into diffusion service
	info, err := diffusionService.InjectInfo(
		request.Type,
		[]byte(request.Content),
		request.Energy,
		time.Duration(request.TTL)*time.Second,
	)

	if err != nil {
		log.Printf("handleInject: InjectInfo error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to inject info: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("handleInject: info injected successfully - id=%s", info.ID)

	// Return the created info message
	response := map[string]interface{}{
		"success": true,
		"info":    info,
		"message": "Information injected successfully",
	}

	log.Printf("handleInject: sending response")
	s.writeJSON(w, response)
	log.Printf("handleInject: completed successfully")
}

// handleInfo handles requests for all information
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("handleInfo: getting all info")

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleInfo: diffusion service is nil")
		http.Error(w, "Diffusion service not initialized", http.StatusInternalServerError)
		return
	}

	allInfo := diffusionService.GetAllInfo()

	response := map[string]interface{}{
		"count": len(allInfo),
		"info":  allInfo,
	}

	log.Printf("handleInfo: returning %d info messages", len(allInfo))
	s.writeJSON(w, response)
}

// handleInfoByID handles requests for specific information by ID
func (s *Server) handleInfoByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/info/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(path, "/info/")
	if id == "" {
		http.Error(w, "Missing info ID", http.StatusBadRequest)
		return
	}

	log.Printf("handleInfoByID: getting info id=%s", id)

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		http.Error(w, "Diffusion service not initialized", http.StatusInternalServerError)
		return
	}

	info, exists := diffusionService.GetInfo(id)
	if !exists {
		log.Printf("handleInfoByID: info id=%s not found", id)
		http.Error(w, "Information not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":   id,
		"info": info,
	}

	log.Printf("handleInfoByID: found info id=%s", id)
	s.writeJSON(w, response)
}

// handleCompute processes computational task injection and queries
func (s *Server) handleCompute(w http.ResponseWriter, r *http.Request) {
	// DISABLED: Return empty response to eliminate computation service calls
	switch r.Method {
	case http.MethodPost:
		// Return success without actually processing the task
		response := map[string]interface{}{
			"message": "Computational task disabled for debugging",
			"success": false,
		}
		s.writeJSON(w, response)
	case http.MethodGet:
		// Return empty task list
		response := map[string]interface{}{
			"active":    []interface{}{},
			"completed": []interface{}{},
			"node_id":   s.node.ID(),
		}
		s.writeJSON(w, response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleComputeInject injects a computational task into the network
func (s *Server) handleComputeInject(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleComputeInject: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	log.Printf("handleComputeInject: processing task injection request")

	var request types.ComputationTask
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("handleComputeInject: failed to decode request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.Type == "" {
		http.Error(w, "Task type is required", http.StatusBadRequest)
		return
	}
	if request.Data == "" {
		http.Error(w, "Task data is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.Energy == 0 {
		request.Energy = 3 // Default energy for wide distribution
	}
	if request.TTL == 0 {
		request.TTL = 300 // 5 minutes default
	}
	if request.Parameters == nil {
		request.Parameters = make(map[string]interface{})
	}

	// Serialize task for InfoMessage
	taskData, err := json.Marshal(request)
	if err != nil {
		log.Printf("handleComputeInject: failed to serialize task: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create and inject task via diffusion service
	diffusion := s.node.GetDiffusionService()
	if diffusion == nil {
		log.Printf("handleComputeInject: diffusion service not available")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	info, err := diffusion.InjectInfo("task", taskData, request.Energy, time.Duration(request.TTL)*time.Second)
	if err != nil {
		log.Printf("handleComputeInject: failed to inject task: %v", err)
		http.Error(w, "Failed to inject task", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Computational task injected successfully",
		"task": map[string]interface{}{
			"id":        info.ID,
			"type":      request.Type,
			"energy":    info.Energy,
			"ttl":       info.TTL,
			"timestamp": info.Timestamp,
		},
	}

	log.Printf("handleComputeInject: task injected successfully, id=%s", info.ID)
	s.writeJSON(w, response)
}

// handleComputeList lists all active and completed computations
func (s *Server) handleComputeList(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleComputeList: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	computation := s.node.GetComputationService()
	if computation == nil {
		http.Error(w, "Computation service not available", http.StatusServiceUnavailable)
		return
	}

	active := computation.GetActiveComputations()
	stats := computation.GetComputationStats()

	response := map[string]interface{}{
		"active_computations": active,
		"stats":               stats,
	}

	s.writeJSON(w, response)
}

// handleComputeByID handles requests for specific computation results
func (s *Server) handleComputeByID(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleComputeByID: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// DISABLED: Return not found for all computation task queries
	http.Error(w, "Computation service disabled for debugging", http.StatusServiceUnavailable)
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleConfig handles configuration requests (GET all params, POST to update multiple)
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w, r)
	case http.MethodPost:
		s.handleConfigUpdate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfigGet returns current configuration parameters
func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	params := s.node.GetRuntimeParameters()
	if params == nil {
		http.Error(w, "Configuration not available", http.StatusServiceUnavailable)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"parameters": map[string]interface{}{
			"energy_decay_rate":        params.EnergyDecayRate,
			"energy_decay_critical":    params.EnergyDecayCritical,
			"energy_decay_routine":     params.EnergyDecayRoutine,
			"default_energy_info":      params.DefaultEnergyInfo,
			"default_energy_compute":   params.DefaultEnergyCompute,
			"default_ttl_seconds":      params.DefaultTTLSeconds,
			"cleanup_interval_seconds": params.CleanupIntervalSeconds,
			"max_neighbors":            params.MaxNeighbors,
			"min_neighbors":            params.MinNeighbors,
			"adaptation_enabled":       params.AdaptationEnabled,
			"learning_rate":            params.LearningRate,
		},
	}

	s.writeJSON(w, response)
}

// handleConfigUpdate updates configuration parameters
func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	results := s.node.UpdateParameters(request)

	response := map[string]interface{}{
		"success": true,
		"results": results,
		"message": "Parameters updated",
	}

	s.writeJSON(w, response)
}

// handleConfigParameter handles individual parameter requests
func (s *Server) handleConfigParameter(w http.ResponseWriter, r *http.Request) {
	// Extract parameter name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/config/")
	if path == "" {
		http.Error(w, "Parameter name required", http.StatusBadRequest)
		return
	}

	params := s.node.GetRuntimeParameters()
	if params == nil {
		http.Error(w, "Configuration not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		value := params.Get(path)
		if value == nil {
			http.Error(w, "Parameter not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"parameter": path,
			"value":     value,
		}
		s.writeJSON(w, response)

	case http.MethodPut:
		var request struct {
			Value interface{} `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		success := params.Set(path, request.Value)
		if !success {
			http.Error(w, "Failed to set parameter", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"parameter": path,
			"value":     request.Value,
			"message":   "Parameter updated successfully",
		}
		s.writeJSON(w, response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// enableCORS adds CORS headers for web dashboard compatibility
func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Phase 3B: Advanced adaptive algorithm monitoring endpoints

// handleAdaptiveMetrics returns comprehensive metrics from the adaptive behavior modifier
func (s *Server) handleAdaptiveMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	behaviorMod := s.node.GetBehaviorModifier()
	if behaviorMod == nil {
		http.Error(w, "Behavior modifier not available", http.StatusServiceUnavailable)
		return
	}

	// Try to cast to AdaptiveBehaviorModifier for advanced metrics
	if adaptiveMod, ok := behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		systemMetrics := adaptiveMod.GetSystemMetrics()

		response := map[string]interface{}{
			"success": true,
			"metrics": map[string]interface{}{
				"system":             systemMetrics,
				"adaptation_enabled": true,
				"adaptive_features":  []string{"network_aware_energy", "load_based_scheduling", "fault_learning", "neighbor_optimization"},
				"timestamp":          time.Now().Unix(),
			},
		}
		s.writeJSON(w, response)
	} else {
		// Basic behavior modifier
		response := map[string]interface{}{
			"success": true,
			"metrics": map[string]interface{}{
				"adaptation_enabled": false,
				"behavior_type":      "default",
				"timestamp":          time.Now().Unix(),
			},
		}
		s.writeJSON(w, response)
	}
}

// handleNeighborMetrics returns performance metrics for all neighbors
func (s *Server) handleNeighborMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	behaviorMod := s.node.GetBehaviorModifier()
	if behaviorMod == nil {
		http.Error(w, "Behavior modifier not available", http.StatusServiceUnavailable)
		return
	}

	// Get current neighbors
	status := s.node.GetStatus()
	neighbors, ok := status["neighbors"]
	if !ok {
		neighbors = []interface{}{}
	}

	response := map[string]interface{}{
		"success":   true,
		"neighbors": neighbors,
	}

	// Add performance metrics if available
	if adaptiveMod, ok := behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		neighborMetrics := make(map[string]interface{})

		// Get metrics for each neighbor
		if neighborList, ok := neighbors.([]*types.Neighbor); ok {
			for _, neighbor := range neighborList {
				metrics := adaptiveMod.GetNeighborMetrics(neighbor.NodeID)
				neighborMetrics[neighbor.NodeID] = metrics
			}
		}

		response["neighbor_metrics"] = neighborMetrics
		response["advanced_metrics"] = true
	} else {
		response["advanced_metrics"] = false
	}

	s.writeJSON(w, response)
}

// handleFaultPatterns returns fault pattern learning data
func (s *Server) handleFaultPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	behaviorMod := s.node.GetBehaviorModifier()
	if behaviorMod == nil {
		http.Error(w, "Behavior modifier not available", http.StatusServiceUnavailable)
		return
	}

	response := map[string]interface{}{
		"success":                true,
		"fault_learning_enabled": false,
		"patterns":               make(map[string]interface{}),
	}

	// Add fault patterns if adaptive modifier is available
	if _, ok := behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		// Get fault patterns (this would require exposing the faultPatterns field)
		response["fault_learning_enabled"] = true
		response["message"] = "Fault pattern learning is active"

		// For now, just return that it's enabled
		// In a full implementation, we'd expose the fault patterns
	}

	s.writeJSON(w, response)
}

// handleSystemMetrics returns current system performance metrics
func (s *Server) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	behaviorMod := s.node.GetBehaviorModifier()
	if behaviorMod == nil {
		http.Error(w, "Behavior modifier not available", http.StatusServiceUnavailable)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"timestamp": time.Now().Unix(),
	}

	// Add system metrics if adaptive modifier is available
	if adaptiveMod, ok := behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
		systemMetrics := adaptiveMod.GetSystemMetrics()
		response["system_metrics"] = systemMetrics
		response["load_trend"] = adaptiveMod.GetLoadTrend()
		response["current_load"] = adaptiveMod.GetSystemLoad()
	} else {
		response["system_metrics"] = map[string]interface{}{
			"message": "Advanced system metrics not available with default behavior modifier",
		}
	}

	s.writeJSON(w, response)
}

// Phase 3C.1: Spatial configuration handlers

// handleSpatialPosition handles GET/POST requests for spatial position configuration
func (s *Server) handleSpatialPosition(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleGetSpatialPosition(w, r)
	case "POST":
		s.handleUpdateSpatialPosition(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetSpatialPosition returns the current spatial configuration
func (s *Server) handleGetSpatialPosition(w http.ResponseWriter, r *http.Request) {
	spatialConfig := s.node.GetSpatialConfig()

	response := map[string]interface{}{
		"spatial_config": spatialConfig,
		"description":    spatialConfig.String(),
		"has_coords":     spatialConfig.HasCoordinates(),
		"is_empty":       spatialConfig.IsEmpty(),
	}

	s.writeJSON(w, response)
}

// handleUpdateSpatialPosition updates the spatial configuration
func (s *Server) handleUpdateSpatialPosition(w http.ResponseWriter, r *http.Request) {
	var updateRequest struct {
		CoordSystem string   `json:"coord_system"`
		X           *float64 `json:"x,omitempty"`
		Y           *float64 `json:"y,omitempty"`
		Z           *float64 `json:"z,omitempty"`
		Zone        string   `json:"zone"`
		Barriers    []string `json:"barriers,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Create new spatial configuration
	newConfig, err := spatial.NewSpatialConfig(
		updateRequest.CoordSystem,
		updateRequest.X,
		updateRequest.Y,
		updateRequest.Z,
		updateRequest.Zone,
		updateRequest.Barriers,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid spatial configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Update node's spatial configuration
	if err := s.node.UpdateSpatialConfig(newConfig); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update spatial configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Spatial configuration updated successfully",
		"config":  newConfig,
	}

	s.writeJSON(w, response)
}

// handleSpatialNeighbors returns neighbors with distance information
func (s *Server) handleSpatialNeighbors(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Phase 3C.2: Get real spatial neighbors data from discovery service
	discoveryService := s.node.GetDiscoveryService()
	neighbors := discoveryService.GetNeighborsWithDistance()

	neighborsData := make([]map[string]interface{}, 0, len(neighbors))
	sameZoneCount := 0

	nodeSpatialConfig := s.node.GetSpatialConfig()

	for _, neighbor := range neighbors {
		neighborData := map[string]interface{}{
			"node_id":        neighbor.NodeID,
			"address":        neighbor.Address,
			"port":           neighbor.Port,
			"cluster_id":     neighbor.ClusterID,
			"last_seen":      neighbor.LastSeen,
			"spatial_config": neighbor.SpatialConfig,
			"distance":       neighbor.Distance,
		}

		// Add spatial analysis
		if neighbor.SpatialConfig != nil && nodeSpatialConfig != nil {
			sameZone := spatial.IsInSameZone(nodeSpatialConfig, neighbor.SpatialConfig)
			neighborData["same_zone"] = sameZone
			neighborData["path_blocked"] = s.node.IsPathBlocked(neighbor.SpatialConfig, "routine")

			if sameZone {
				sameZoneCount++
			}
		} else {
			neighborData["same_zone"] = false
			neighborData["path_blocked"] = false
		}

		neighborsData = append(neighborsData, neighborData)
	}

	// Zone-aware neighbor selection analysis
	sameZoneNeighbors := discoveryService.GetNeighborsInZone(nodeSpatialConfig.Zone)
	crossZoneNeighbors := discoveryService.GetNeighborsOutsideZone(nodeSpatialConfig.Zone)

	response := map[string]interface{}{
		"neighbors":              neighborsData,
		"neighbors_count":        len(neighbors),
		"current_spatial_config": nodeSpatialConfig,
		"zone_analysis": map[string]interface{}{
			"same_zone_count":   len(sameZoneNeighbors),
			"cross_zone_count":  len(crossZoneNeighbors),
			"same_zone_ratio":   float64(len(sameZoneNeighbors)) / float64(len(neighbors)),
			"target_same_zone":  0.7, // 70% target
			"target_cross_zone": 0.3, // 30% target
		},
	}

	s.writeJSON(w, response)
}

// handleSpatialBarriers returns barrier configuration and status
func (s *Server) handleSpatialBarriers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	barrierManager := s.node.GetBarrierManager()
	barriers := barrierManager.GetAllBarriers()

	// Convert barriers to JSON-serializable format
	barrierList := make([]map[string]interface{}, 0, len(barriers))
	for _, barrier := range barriers {
		barrierList = append(barrierList, map[string]interface{}{
			"id":          barrier.ID,
			"type":        barrier.Type,
			"description": barrier.Description,
			"isolation":   barrier.Isolation,
			"zone_a":      barrier.ZoneA,
			"zone_b":      barrier.ZoneB,
		})
	}

	response := map[string]interface{}{
		"barriers":            barrierList,
		"barriers_count":      len(barriers),
		"node_spatial_config": s.node.GetSpatialConfig(),
	}

	s.writeJSON(w, response)
}

// handleSpatialDistance calculates distance to specified coordinates
func (s *Server) handleSpatialDistance(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var distanceRequest struct {
		CoordSystem string   `json:"coord_system"`
		X           *float64 `json:"x,omitempty"`
		Y           *float64 `json:"y,omitempty"`
		Z           *float64 `json:"z,omitempty"`
		Zone        string   `json:"zone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&distanceRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Create target spatial configuration
	targetConfig, err := spatial.NewSpatialConfig(
		distanceRequest.CoordSystem,
		distanceRequest.X,
		distanceRequest.Y,
		distanceRequest.Z,
		distanceRequest.Zone,
		nil, // No barriers for distance calculation
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Calculate distance
	distance, err := s.node.CalculateDistanceTo(targetConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate distance: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if path is blocked
	pathBlocked := s.node.IsPathBlocked(targetConfig, "routine") // Use routine message type as default

	response := map[string]interface{}{
		"from_config":  s.node.GetSpatialConfig(),
		"to_config":    targetConfig,
		"distance":     distance,
		"same_zone":    spatial.IsInSameZone(s.node.GetSpatialConfig(), targetConfig),
		"path_blocked": pathBlocked,
		"message_type": "routine",
	}

	s.writeJSON(w, response)
}

// handleTopologyMap handles GET requests for complete network topology
func (s *Server) handleTopologyMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// DISABLED: Return empty topology to eliminate topology mapper calls
	emptyTopology := map[string]interface{}{
		"nodes":       []interface{}{},
		"connections": []interface{}{},
		"zones":       []interface{}{},
		"barriers":    []interface{}{},
		"metadata": map[string]interface{}{
			"node_count":       0,
			"connection_count": 0,
			"zone_count":       0,
			"barrier_count":    0,
			"generated_at":     0,
		},
	}
	s.writeJSON(w, emptyTopology)
}

// handleTopologyZones handles GET requests for zone-specific topology information
func (s *Server) handleTopologyZones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("handleTopologyZones: getting zone topology information")

	// DISABLED: Return empty zones to eliminate topology mapper calls
	emptyZones := map[string]interface{}{
		"zones":        []interface{}{},
		"zone_count":   0,
		"generated_at": 0,
	}
	s.writeJSON(w, emptyZones)
}

// handleTopologyLive handles WebSocket connections for live topology updates
func (s *Server) handleTopologyLive(w http.ResponseWriter, r *http.Request) {
	// For Phase 3C.3a, we'll implement a simple HTTP-based polling endpoint
	// WebSocket implementation can be added in a future enhancement

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("handleTopologyLive: providing live topology update")

	// DISABLED: Return empty live topology to eliminate topology mapper calls
	response := map[string]interface{}{
		"topology": map[string]interface{}{
			"nodes":       []interface{}{},
			"connections": []interface{}{},
			"zones":       []interface{}{},
			"barriers":    []interface{}{},
		},
		"live_update":           true,
		"poll_interval_seconds": 5,
		"supports_websocket":    false,
	}
	s.writeJSON(w, response)
}

// handleChemistryConcentrations returns current concentration state
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryConcentrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// DISABLED: Return empty chemistry state to eliminate chemistry engine calls
	emptyState := map[string]interface{}{
		"concentrations": make(map[string]float64),
		"last_update":    0,
		"node_id":        s.node.ID(),
	}
	s.writeJSON(w, emptyState)
}

// handleChemistryReactions returns reaction history
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryReactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// DISABLED: Return empty reactions to eliminate chemistry engine calls
	emptyReactions := map[string]interface{}{
		"reactions": []interface{}{},
		"count":     0,
	}
	s.writeJSON(w, emptyReactions)
}

// handleChemistryStats returns chemistry engine statistics
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// DISABLED: Return empty chemistry stats to eliminate chemistry engine calls
	emptyStats := map[string]interface{}{
		"node_id":             s.node.ID(),
		"total_energy":        0.0,
		"total_reactions":     0,
		"concentration_decay": 0.0,
		"reaction_threshold":  0.0,
		"diffusion_threshold": 0.0,
		"base_reaction_rate":  0.0,
		"last_update":         0,
		"total_messages":      0,
		"message_types":       0,
	}
	s.writeJSON(w, emptyStats)
}

// handleMetricsCluster returns comprehensive cluster health metrics
// Phase 4B-Alt: Enhanced observability for cluster behavior
func (s *Server) handleMetricsCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	discoveryService := s.node.GetDiscoveryService()
	if discoveryService == nil {
		log.Printf("handleMetricsCluster: discovery service not available")
		http.Error(w, "Discovery service not available", http.StatusServiceUnavailable)
		return
	}

	neighbors := discoveryService.GetNeighbors()

	// Calculate neighbor health metrics
	totalNeighbors := len(neighbors)
	healthyNeighbors := totalNeighbors // For now, assume all returned neighbors are healthy
	zoneDistribution := make(map[string]int)

	for range neighbors {
		// Note: types.Neighbor doesn't have zone info, will need to enhance this later
		// For now, just count neighbors
		zoneDistribution["unknown"]++
	}

	// Health percentage
	healthPercentage := float64(0)
	if totalNeighbors > 0 {
		healthPercentage = float64(healthyNeighbors) / float64(totalNeighbors) * 100
	}

	metrics := map[string]interface{}{
		"timestamp":         time.Now().Unix(),
		"neighbor_count":    totalNeighbors,
		"healthy_neighbors": healthyNeighbors,
		"health_percentage": healthPercentage,
		"zone_distribution": zoneDistribution,
		"node_id":           s.node.ID(),
		"note":              "Enhanced neighbor metrics require internal discovery service access",
	}

	log.Printf("handleMetricsCluster: returning cluster metrics for %d neighbors", totalNeighbors)
	s.writeJSON(w, metrics)
}

// handleMetricsChemistry returns detailed chemistry engine metrics
// Phase 4B-Alt: Chemistry-specific observability
func (s *Server) handleMetricsChemistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleMetricsChemistry: diffusion service not available")
		http.Error(w, "Diffusion service not available", http.StatusServiceUnavailable)
		return
	}

	chemEngine := diffusionService.GetChemistryEngine()
	if chemEngine == nil {
		log.Printf("handleMetricsChemistry: chemistry engine not available")
		http.Error(w, "Chemistry engine not available", http.StatusServiceUnavailable)
		return
	}

	concentrationState := chemEngine.GetConcentrationState()
	reactions := chemEngine.GetReactionHistory()
	stats := chemEngine.GetChemistryStats()

	// Calculate concentration metrics
	totalConcentration := float64(0)
	maxConcentration := float64(0)
	concentrationTypes := len(concentrationState.Concentrations)

	for _, concentration := range concentrationState.Concentrations {
		totalConcentration += concentration
		if concentration > maxConcentration {
			maxConcentration = concentration
		}
	}

	// Calculate reaction rate (reactions per minute)
	recentReactions := 0
	now := time.Now().Unix()
	for _, reaction := range reactions {
		if now-reaction.Timestamp < 60 { // Last minute
			recentReactions++
		}
	}

	metrics := map[string]interface{}{
		"timestamp":           time.Now().Unix(),
		"concentration_types": concentrationTypes,
		"total_concentration": totalConcentration,
		"max_concentration":   maxConcentration,
		"avg_concentration": func() float64 {
			if concentrationTypes > 0 {
				return totalConcentration / float64(concentrationTypes)
			}
			return 0
		}(),
		"total_reactions":      len(reactions),
		"recent_reaction_rate": recentReactions, // reactions per minute
		"total_messages":       concentrationState.TotalMessages,
		"message_types":        len(concentrationState.MessageCounts),
		"last_update":          concentrationState.LastUpdate,
		"concentrations":       concentrationState.Concentrations,
		"node_id":              s.node.ID(),
		"chemistry_stats":      stats,
	}

	log.Printf("handleMetricsChemistry: returning chemistry metrics - %d types, %d reactions", concentrationTypes, len(reactions))
	s.writeJSON(w, metrics)
}

// handleMetricsDiffusion returns message diffusion performance metrics
// Phase 4B-Alt: Diffusion behavior observability
func (s *Server) handleMetricsDiffusion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleMetricsDiffusion: diffusion service not available")
		http.Error(w, "Diffusion service not available", http.StatusServiceUnavailable)
		return
	}

	// Get stored messages for analysis
	storedMessagesMap := diffusionService.GetAllInfo()
	storedMessages := make([]*types.InfoMessage, 0, len(storedMessagesMap))
	for _, msg := range storedMessagesMap {
		storedMessages = append(storedMessages, msg)
	}

	// Calculate diffusion metrics
	totalMessages := len(storedMessages)
	totalEnergy := float64(0)
	energyDistribution := make(map[string]int) // Energy ranges
	typeDistribution := make(map[string]int)
	hopDistribution := make(map[int]int)

	var ages []int64
	now := time.Now().Unix()

	for _, msg := range storedMessages {
		totalEnergy += msg.Energy
		typeDistribution[msg.Type]++
		hopDistribution[msg.Hops]++

		age := now - msg.Timestamp
		ages = append(ages, age)

		// Energy distribution buckets
		switch {
		case msg.Energy == 0:
			energyDistribution["zero"]++
		case msg.Energy < 1:
			energyDistribution["low"]++
		case msg.Energy < 10:
			energyDistribution["medium"]++
		default:
			energyDistribution["high"]++
		}
	}

	// Calculate average energy
	avgEnergy := float64(0)
	if totalMessages > 0 {
		avgEnergy = totalEnergy / float64(totalMessages)
	}

	// Calculate average age
	avgAge := int64(0)
	if len(ages) > 0 {
		var totalAge int64
		for _, age := range ages {
			totalAge += age
		}
		avgAge = totalAge / int64(len(ages))
	}

	metrics := map[string]interface{}{
		"timestamp":           time.Now().Unix(),
		"total_messages":      totalMessages,
		"total_energy":        totalEnergy,
		"avg_energy":          avgEnergy,
		"avg_message_age_sec": avgAge,
		"energy_distribution": energyDistribution,
		"type_distribution":   typeDistribution,
		"hop_distribution":    hopDistribution,
		"message_types":       len(typeDistribution),
		"max_hops": func() int {
			max := 0
			for hops := range hopDistribution {
				if hops > max {
					max = hops
				}
			}
			return max
		}(),
		"node_id": s.node.ID(),
	}

	log.Printf("handleMetricsDiffusion: returning diffusion metrics for %d messages", totalMessages)
	s.writeJSON(w, metrics)
}

// handleMetricsSpatial returns spatial computing performance metrics
// Phase 4B-Alt: Spatial behavior observability
func (s *Server) handleMetricsSpatial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	spatialConfig := s.node.GetSpatialConfig()
	if spatialConfig == nil {
		// Return basic metrics for non-spatial nodes
		metrics := map[string]interface{}{
			"timestamp":       time.Now().Unix(),
			"spatial_enabled": false,
			"coord_system":    "none",
			"node_id":         s.node.ID(),
		}
		s.writeJSON(w, metrics)
		return
	}

	discoveryService := s.node.GetDiscoveryService()
	neighbors := []*types.Neighbor{}
	if discoveryService != nil {
		neighbors = discoveryService.GetNeighbors()
	}

	// Calculate spatial metrics - simplified for now since types.Neighbor doesn't include spatial info
	totalNeighbors := len(neighbors)
	sameZoneCount := 0
	crossZoneCount := 0
	var distances []float64
	zoneNeighbors := make(map[string]int)

	// Note: Current types.Neighbor doesn't include spatial information
	// We'll need to enhance this by accessing the internal discovery service neighbors
	// For now, provide basic metrics
	zoneNeighbors["needs_enhancement"] = totalNeighbors

	// Calculate distance statistics
	var avgDistance, minDistance, maxDistance float64
	if len(distances) > 0 {
		total := float64(0)
		minDistance = distances[0]
		maxDistance = distances[0]

		for _, dist := range distances {
			total += dist
			if dist < minDistance {
				minDistance = dist
			}
			if dist > maxDistance {
				maxDistance = dist
			}
		}
		avgDistance = total / float64(len(distances))
	}

	// Zone ratio calculation
	totalNeighborsForRatio := sameZoneCount + crossZoneCount
	sameZoneRatio := float64(0)
	crossZoneRatio := float64(0)
	if totalNeighborsForRatio > 0 {
		sameZoneRatio = float64(sameZoneCount) / float64(totalNeighborsForRatio) * 100
		crossZoneRatio = float64(crossZoneCount) / float64(totalNeighborsForRatio) * 100
	}

	metrics := map[string]interface{}{
		"timestamp":        time.Now().Unix(),
		"spatial_enabled":  true,
		"coord_system":     spatialConfig.CoordSystem,
		"has_coordinates":  spatialConfig.HasCoordinates(),
		"zone":             spatialConfig.Zone,
		"same_zone_count":  sameZoneCount,
		"cross_zone_count": crossZoneCount,
		"same_zone_ratio":  sameZoneRatio,
		"cross_zone_ratio": crossZoneRatio,
		"zone_neighbors":   zoneNeighbors,
		"avg_distance":     avgDistance,
		"min_distance":     minDistance,
		"max_distance":     maxDistance,
		"distance_samples": len(distances),
		"coordinates": map[string]interface{}{
			"x": spatialConfig.X,
			"y": spatialConfig.Y,
			"z": spatialConfig.Z,
		},
		"node_id": s.node.ID(),
	}

	log.Printf("handleMetricsSpatial: returning spatial metrics - zone ratio %.1f%%/%.1f%%", sameZoneRatio, crossZoneRatio)
	s.writeJSON(w, metrics)
}

// handleMetricsPerformance returns API and processing performance metrics
// Phase 4B-Alt: Performance observability
func (s *Server) handleMetricsPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic performance metrics (we'll enhance these as we add more tracking)
	startTime := time.Now()

	// Test basic API responsiveness
	statusData := s.node.GetStatus()
	apiLatency := time.Since(startTime).Nanoseconds()

	// Memory usage approximation (basic)
	diffusionService := s.node.GetDiffusionService()
	messageCount := 0
	if diffusionService != nil {
		allMessages := diffusionService.GetAllInfo()
		messageCount = len(allMessages)
	}

	discoveryService := s.node.GetDiscoveryService()
	neighborCount := 0
	if discoveryService != nil {
		neighborCount = len(discoveryService.GetNeighbors())
	}

	// Estimate memory usage (rough approximation)
	estimatedMemoryBytes := (messageCount * 1024) + (neighborCount * 256) // Rough estimates

	metrics := map[string]interface{}{
		"timestamp":              time.Now().Unix(),
		"api_response_time_ns":   apiLatency,
		"api_response_time_ms":   float64(apiLatency) / 1000000,
		"stored_messages":        messageCount,
		"neighbor_count":         neighborCount,
		"estimated_memory_bytes": estimatedMemoryBytes,
		"estimated_memory_kb":    estimatedMemoryBytes / 1024,
		"uptime_seconds":         time.Since(startTime).Seconds(), // This is just measurement time, we'll improve this
		"node_id":                s.node.ID(),
		"status_check":           statusData != nil,
	}

	log.Printf("handleMetricsPerformance: API latency %.2fms, %d messages, %d neighbors",
		float64(apiLatency)/1000000, messageCount, neighborCount)
	s.writeJSON(w, metrics)
}

// Helper function for distance calculation in spatial metrics
func (s *Server) calculateDistance(spatialConfig interface{}, x, y, z float64) float64 {
	// This is a simplified distance calculation for metrics
	// In a real implementation, we'd use the proper spatial distance calculation
	if config, ok := spatialConfig.(map[string]interface{}); ok {
		if myX, xOk := config["x"].(float64); xOk {
			if myY, yOk := config["y"].(float64); yOk {
				dx := x - myX
				dy := y - myY
				return math.Sqrt(dx*dx + dy*dy)
			}
		}
	}
	return 0
}

// handleUnifiedMetrics provides both Prometheus and JSON metrics via format detection
// Phase 4B-Alt: Unified metrics endpoint with smart format detection
func (s *Server) handleUnifiedMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Smart format detection
	format := r.URL.Query().Get("format")
	accept := r.Header.Get("Accept")

	// Default to Prometheus format (industry standard)
	if format == "json" || strings.Contains(accept, "application/json") {
		s.handleJSONMetrics(w, r)
	} else {
		s.handlePrometheusMetrics(w, r)
	}
}

// handlePrometheusMetrics returns all metrics in Prometheus time-series format
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// Set Prometheus content type
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Collect all metrics and convert to Prometheus format
	promMetrics := s.collectPrometheusMetrics()

	// Write Prometheus format response
	w.Write([]byte(promMetrics))
	log.Printf("handlePrometheusMetrics: returned %d bytes of metrics", len(promMetrics))
}

// handleJSONMetrics returns comprehensive JSON metrics combining all data sources
func (s *Server) handleJSONMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect all metrics from different sources
	allMetrics := s.collectAllJSONMetrics()

	log.Printf("handleJSONMetrics: returning comprehensive metrics")
	s.writeJSON(w, allMetrics)
}

// collectPrometheusMetrics gathers all metrics and formats them for Prometheus
func (s *Server) collectPrometheusMetrics() string {
	var metrics []string
	nodeID := s.node.ID()

	// Helper function to create base labels
	baseLabels := func(additionalLabels ...string) string {
		labels := []string{fmt.Sprintf("node_id=\"%s\"", nodeID)}
		labels = append(labels, additionalLabels...)
		return strings.Join(labels, ",")
	}

	// Cluster metrics
	if discoveryService := s.node.GetDiscoveryService(); discoveryService != nil {
		neighbors := discoveryService.GetNeighbors()
		totalNeighbors := len(neighbors)
		healthyNeighbors := totalNeighbors // Assume healthy for now

		healthPercentage := float64(0)
		if totalNeighbors > 0 {
			healthPercentage = float64(healthyNeighbors) / float64(totalNeighbors) * 100
		}

		metrics = append(metrics,
			"# HELP ryx_neighbors_total Total number of discovered neighbors",
			"# TYPE ryx_neighbors_total gauge",
			fmt.Sprintf("ryx_neighbors_total{%s} %d", baseLabels(), totalNeighbors),
			"",
			"# HELP ryx_neighbor_health_percentage Percentage of healthy neighbors",
			"# TYPE ryx_neighbor_health_percentage gauge",
			fmt.Sprintf("ryx_neighbor_health_percentage{%s} %.2f", baseLabels(), healthPercentage),
			"",
		)
	}

	// Diffusion metrics
	if diffusionService := s.node.GetDiffusionService(); diffusionService != nil {
		storedMessagesMap := diffusionService.GetAllInfo()
		totalMessages := len(storedMessagesMap)
		totalEnergy := float64(0)
		typeDistribution := make(map[string]int)
		maxHops := 0

		for _, msg := range storedMessagesMap {
			totalEnergy += msg.Energy
			typeDistribution[msg.Type]++
			if msg.Hops > maxHops {
				maxHops = msg.Hops
			}
		}

		avgEnergy := float64(0)
		if totalMessages > 0 {
			avgEnergy = totalEnergy / float64(totalMessages)
		}

		metrics = append(metrics,
			"# HELP ryx_messages_stored_total Total number of stored messages",
			"# TYPE ryx_messages_stored_total gauge",
			fmt.Sprintf("ryx_messages_stored_total{%s} %d", baseLabels(), totalMessages),
			"",
			"# HELP ryx_message_energy_total Total energy of all stored messages",
			"# TYPE ryx_message_energy_total gauge",
			fmt.Sprintf("ryx_message_energy_total{%s} %.2f", baseLabels(), totalEnergy),
			"",
			"# HELP ryx_message_energy_avg Average energy per message",
			"# TYPE ryx_message_energy_avg gauge",
			fmt.Sprintf("ryx_message_energy_avg{%s} %.2f", baseLabels(), avgEnergy),
			"",
			"# HELP ryx_message_hops_max Maximum hops seen in stored messages",
			"# TYPE ryx_message_hops_max gauge",
			fmt.Sprintf("ryx_message_hops_max{%s} %d", baseLabels(), maxHops),
			"",
		)

		// Per-type message counts
		if len(typeDistribution) > 0 {
			metrics = append(metrics,
				"# HELP ryx_messages_by_type_total Number of messages by type",
				"# TYPE ryx_messages_by_type_total gauge",
			)
			for msgType, count := range typeDistribution {
				labels := baseLabels(fmt.Sprintf("message_type=\"%s\"", msgType))
				metrics = append(metrics, fmt.Sprintf("ryx_messages_by_type_total{%s} %d", labels, count))
			}
			metrics = append(metrics, "")
		}
	}

	// Chemistry metrics
	if diffusionService := s.node.GetDiffusionService(); diffusionService != nil {
		if chemEngine := diffusionService.GetChemistryEngine(); chemEngine != nil {
			concentrationState := chemEngine.GetConcentrationState()
			reactions := chemEngine.GetReactionHistory()

			totalConcentration := float64(0)
			for _, concentration := range concentrationState.Concentrations {
				totalConcentration += concentration
			}

			metrics = append(metrics,
				"# HELP ryx_chemistry_total_concentration Sum of all chemical concentrations",
				"# TYPE ryx_chemistry_total_concentration gauge",
				fmt.Sprintf("ryx_chemistry_total_concentration{%s} %.4f", baseLabels(), totalConcentration),
				"",
				"# HELP ryx_chemistry_reactions_total Total number of chemical reactions",
				"# TYPE ryx_chemistry_reactions_total counter",
				fmt.Sprintf("ryx_chemistry_reactions_total{%s} %d", baseLabels(), len(reactions)),
				"",
				"# HELP ryx_chemistry_message_types Number of different chemical message types",
				"# TYPE ryx_chemistry_message_types gauge",
				fmt.Sprintf("ryx_chemistry_message_types{%s} %d", baseLabels(), len(concentrationState.Concentrations)),
				"",
			)

			// Per-chemical concentrations
			if len(concentrationState.Concentrations) > 0 {
				metrics = append(metrics,
					"# HELP ryx_chemistry_concentration Chemical concentration by type",
					"# TYPE ryx_chemistry_concentration gauge",
				)
				for chemType, concentration := range concentrationState.Concentrations {
					labels := baseLabels(fmt.Sprintf("chemical_type=\"%s\"", chemType))
					metrics = append(metrics, fmt.Sprintf("ryx_chemistry_concentration{%s} %.4f", labels, concentration))
				}
				metrics = append(metrics, "")
			}
		}
	}

	// Spatial metrics
	if spatialConfig := s.node.GetSpatialConfig(); spatialConfig != nil {
		spatialEnabled := 1
		hasCoords := 0
		if spatialConfig.HasCoordinates() {
			hasCoords = 1
		}

		coordLabels := baseLabels(
			fmt.Sprintf("coord_system=\"%s\"", spatialConfig.CoordSystem),
			fmt.Sprintf("zone=\"%s\"", spatialConfig.Zone),
		)

		metrics = append(metrics,
			"# HELP ryx_spatial_enabled Whether spatial computing is enabled (1=enabled, 0=disabled)",
			"# TYPE ryx_spatial_enabled gauge",
			fmt.Sprintf("ryx_spatial_enabled{%s} %d", coordLabels, spatialEnabled),
			"",
			"# HELP ryx_spatial_has_coordinates Whether node has coordinate information",
			"# TYPE ryx_spatial_has_coordinates gauge",
			fmt.Sprintf("ryx_spatial_has_coordinates{%s} %d", coordLabels, hasCoords),
			"",
		)

		if spatialConfig.HasCoordinates() && spatialConfig.X != nil && spatialConfig.Y != nil {
			metrics = append(metrics,
				"# HELP ryx_spatial_coordinate_x Node X coordinate",
				"# TYPE ryx_spatial_coordinate_x gauge",
				fmt.Sprintf("ryx_spatial_coordinate_x{%s} %.6f", coordLabels, *spatialConfig.X),
				"",
				"# HELP ryx_spatial_coordinate_y Node Y coordinate",
				"# TYPE ryx_spatial_coordinate_y gauge",
				fmt.Sprintf("ryx_spatial_coordinate_y{%s} %.6f", coordLabels, *spatialConfig.Y),
				"",
			)
			if spatialConfig.Z != nil {
				metrics = append(metrics,
					"# HELP ryx_spatial_coordinate_z Node Z coordinate",
					"# TYPE ryx_spatial_coordinate_z gauge",
					fmt.Sprintf("ryx_spatial_coordinate_z{%s} %.6f", coordLabels, *spatialConfig.Z),
					"",
				)
			}
		}
	}

	// Performance metrics
	startTime := time.Now()
	statusData := s.node.GetStatus()
	apiLatency := time.Since(startTime).Nanoseconds()

	metrics = append(metrics,
		"# HELP ryx_api_response_time_milliseconds API response time in milliseconds",
		"# TYPE ryx_api_response_time_milliseconds gauge",
		fmt.Sprintf("ryx_api_response_time_milliseconds{%s,endpoint=\"status\"} %.3f", baseLabels(), float64(apiLatency)/1000000),
		"",
		"# HELP ryx_status_check_success Whether status check succeeded (1=success, 0=failure)",
		"# TYPE ryx_status_check_success gauge",
	)

	statusSuccess := 0
	if statusData != nil {
		statusSuccess = 1
	}
	metrics = append(metrics,
		fmt.Sprintf("ryx_status_check_success{%s} %d", baseLabels(), statusSuccess),
		"",
	)

	return strings.Join(metrics, "\n")
}

// collectAllJSONMetrics gathers comprehensive metrics from all sources
func (s *Server) collectAllJSONMetrics() map[string]interface{} {
	response := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"node_id":   s.node.ID(),
	}

	// Cluster metrics
	if discoveryService := s.node.GetDiscoveryService(); discoveryService != nil {
		neighbors := discoveryService.GetNeighbors()
		totalNeighbors := len(neighbors)
		healthyNeighbors := totalNeighbors // Assume healthy for now

		healthPercentage := float64(0)
		if totalNeighbors > 0 {
			healthPercentage = float64(healthyNeighbors) / float64(totalNeighbors) * 100
		}

		response["cluster"] = map[string]interface{}{
			"neighbor_count":    totalNeighbors,
			"healthy_neighbors": healthyNeighbors,
			"health_percentage": healthPercentage,
			"zone_distribution": map[string]int{"unknown": totalNeighbors},
		}
	}

	// Diffusion metrics
	if diffusionService := s.node.GetDiffusionService(); diffusionService != nil {
		storedMessagesMap := diffusionService.GetAllInfo()
		totalMessages := len(storedMessagesMap)
		totalEnergy := float64(0)
		typeDistribution := make(map[string]int)
		hopDistribution := make(map[int]int)
		maxHops := 0

		var ages []int64
		now := time.Now().Unix()

		for _, msg := range storedMessagesMap {
			totalEnergy += msg.Energy
			typeDistribution[msg.Type]++
			hopDistribution[msg.Hops]++

			if msg.Hops > maxHops {
				maxHops = msg.Hops
			}

			age := now - msg.Timestamp
			ages = append(ages, age)
		}

		avgEnergy := float64(0)
		if totalMessages > 0 {
			avgEnergy = totalEnergy / float64(totalMessages)
		}

		avgAge := int64(0)
		if len(ages) > 0 {
			var totalAge int64
			for _, age := range ages {
				totalAge += age
			}
			avgAge = totalAge / int64(len(ages))
		}

		response["diffusion"] = map[string]interface{}{
			"total_messages":      totalMessages,
			"total_energy":        totalEnergy,
			"avg_energy":          avgEnergy,
			"avg_message_age_sec": avgAge,
			"type_distribution":   typeDistribution,
			"hop_distribution":    hopDistribution,
			"message_types":       len(typeDistribution),
			"max_hops":            maxHops,
		}
	}

	// Chemistry metrics
	if diffusionService := s.node.GetDiffusionService(); diffusionService != nil {
		if chemEngine := diffusionService.GetChemistryEngine(); chemEngine != nil {
			concentrationState := chemEngine.GetConcentrationState()
			reactions := chemEngine.GetReactionHistory()
			stats := chemEngine.GetChemistryStats()

			totalConcentration := float64(0)
			maxConcentration := float64(0)
			for _, concentration := range concentrationState.Concentrations {
				totalConcentration += concentration
				if concentration > maxConcentration {
					maxConcentration = concentration
				}
			}

			// Calculate reaction rate (reactions per minute)
			recentReactions := 0
			now := time.Now().Unix()
			for _, reaction := range reactions {
				if now-reaction.Timestamp < 60 { // Last minute
					recentReactions++
				}
			}

			avgConcentration := float64(0)
			concentrationTypes := len(concentrationState.Concentrations)
			if concentrationTypes > 0 {
				avgConcentration = totalConcentration / float64(concentrationTypes)
			}

			response["chemistry"] = map[string]interface{}{
				"concentration_types":  concentrationTypes,
				"total_concentration":  totalConcentration,
				"max_concentration":    maxConcentration,
				"avg_concentration":    avgConcentration,
				"total_reactions":      len(reactions),
				"recent_reaction_rate": recentReactions,
				"total_messages":       concentrationState.TotalMessages,
				"message_types":        len(concentrationState.MessageCounts),
				"last_update":          concentrationState.LastUpdate,
				"concentrations":       concentrationState.Concentrations,
				"chemistry_stats":      stats,
			}
		}
	}

	// Spatial metrics
	if spatialConfig := s.node.GetSpatialConfig(); spatialConfig != nil {
		hasCoords := spatialConfig.HasCoordinates()

		spatialData := map[string]interface{}{
			"spatial_enabled":  true,
			"coord_system":     spatialConfig.CoordSystem,
			"has_coordinates":  hasCoords,
			"zone":             spatialConfig.Zone,
			"same_zone_count":  0,
			"cross_zone_count": 0,
			"same_zone_ratio":  0.0,
			"cross_zone_ratio": 0.0,
			"zone_neighbors":   map[string]int{"needs_enhancement": 0},
			"avg_distance":     0.0,
			"min_distance":     0.0,
			"max_distance":     0.0,
			"distance_samples": 0,
		}

		if hasCoords && spatialConfig.X != nil && spatialConfig.Y != nil {
			spatialData["coordinates"] = map[string]interface{}{
				"x": *spatialConfig.X,
				"y": *spatialConfig.Y,
				"z": func() interface{} {
					if spatialConfig.Z != nil {
						return *spatialConfig.Z
					}
					return 0.0
				}(),
			}
		}

		response["spatial"] = spatialData
	} else {
		response["spatial"] = map[string]interface{}{
			"spatial_enabled": false,
			"coord_system":    "none",
		}
	}

	// Performance metrics
	startTime := time.Now()
	statusData := s.node.GetStatus()
	apiLatency := time.Since(startTime).Nanoseconds()

	// Basic resource estimation
	messageCount := 0
	if diffusionService := s.node.GetDiffusionService(); diffusionService != nil {
		allMessages := diffusionService.GetAllInfo()
		messageCount = len(allMessages)
	}

	neighborCount := 0
	if discoveryService := s.node.GetDiscoveryService(); discoveryService != nil {
		neighborCount = len(discoveryService.GetNeighbors())
	}

	estimatedMemoryBytes := (messageCount * 1024) + (neighborCount * 256)

	response["performance"] = map[string]interface{}{
		"api_response_time_ns":   apiLatency,
		"api_response_time_ms":   float64(apiLatency) / 1000000,
		"stored_messages":        messageCount,
		"neighbor_count":         neighborCount,
		"estimated_memory_bytes": estimatedMemoryBytes,
		"estimated_memory_kb":    estimatedMemoryBytes / 1024,
		"status_check":           statusData != nil,
	}

	// Adaptive metrics (legacy Phase 3B data)
	if behaviorMod := s.node.GetBehaviorModifier(); behaviorMod != nil {
		if adaptiveMod, ok := behaviorMod.(*config.AdaptiveBehaviorModifier); ok {
			systemMetrics := adaptiveMod.GetSystemMetrics()
			response["adaptive"] = map[string]interface{}{
				"system":             systemMetrics,
				"adaptation_enabled": true,
				"type":               "AdaptiveBehaviorModifier",
			}
		} else {
			response["adaptive"] = map[string]interface{}{
				"adaptation_enabled": false,
				"type":               "DefaultBehaviorModifier",
			}
		}
	}

	return response
}

// Dashboard functionality removed - use external ryx-dashboard application
