package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	mux.HandleFunc("/adaptive/metrics", s.handleAdaptiveMetrics)
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

	// Phase 3C.3a: Topology mapping endpoints
	mux.HandleFunc("/topology/map", s.handleTopologyMap)
	mux.HandleFunc("/topology/zones", s.handleTopologyZones)
	mux.HandleFunc("/topology/live", s.handleTopologyLive)

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
	switch r.Method {
	case http.MethodPost:
		s.handleComputeInject(w, r)
	case http.MethodGet:
		s.handleComputeList(w, r)
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

	// Extract computation ID from path
	path := strings.TrimPrefix(r.URL.Path, "/compute/")
	if path == "" {
		http.Error(w, "Computation ID required", http.StatusBadRequest)
		return
	}

	taskID := strings.Split(path, "/")[0]

	computation := s.node.GetComputationService()
	if computation == nil {
		http.Error(w, "Computation service not available", http.StatusServiceUnavailable)
		return
	}

	result, exists := computation.GetComputationResult(taskID)
	if !exists {
		http.Error(w, "Computation not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"task_id": taskID,
		"result":  result,
	}

	s.writeJSON(w, response)
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

	log.Printf("handleTopologyMap: getting network topology")

	topologyMapper := s.node.GetTopologyMapper()
	if topologyMapper == nil {
		log.Printf("handleTopologyMap: topology mapper not available")
		http.Error(w, "Topology mapper not available", http.StatusServiceUnavailable)
		return
	}

	topology, err := topologyMapper.GetCurrentTopology()
	if err != nil {
		log.Printf("handleTopologyMap: failed to get topology: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get topology: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("handleTopologyMap: topology generated - %d nodes, %d connections, %d zones, %d barriers",
		topology.Metadata.NodeCount,
		topology.Metadata.ConnectionCount,
		topology.Metadata.ZoneCount,
		topology.Metadata.BarrierCount)

	s.writeJSON(w, topology)
}

// handleTopologyZones handles GET requests for zone-specific topology information
func (s *Server) handleTopologyZones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("handleTopologyZones: getting zone topology information")

	topologyMapper := s.node.GetTopologyMapper()
	if topologyMapper == nil {
		log.Printf("handleTopologyZones: topology mapper not available")
		http.Error(w, "Topology mapper not available", http.StatusServiceUnavailable)
		return
	}

	// Check if specific zone requested
	zoneID := r.URL.Query().Get("zone")
	if zoneID != "" {
		log.Printf("handleTopologyZones: getting topology for zone: %s", zoneID)

		zone, err := topologyMapper.GetZoneTopology(zoneID)
		if err != nil {
			log.Printf("handleTopologyZones: failed to get zone topology: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get zone topology: %v", err), http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"zone":      zone,
			"requested": zoneID,
		}
		s.writeJSON(w, response)
		return
	}

	// Get all zones
	topology, err := topologyMapper.GetCurrentTopology()
	if err != nil {
		log.Printf("handleTopologyZones: failed to get topology: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get topology: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"zones":        topology.Zones,
		"zone_count":   len(topology.Zones),
		"generated_at": topology.Metadata.GeneratedAt,
	}

	log.Printf("handleTopologyZones: returning %d zones", len(topology.Zones))
	s.writeJSON(w, response)
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

	topologyMapper := s.node.GetTopologyMapper()
	if topologyMapper == nil {
		log.Printf("handleTopologyLive: topology mapper not available")
		http.Error(w, "Topology mapper not available", http.StatusServiceUnavailable)
		return
	}

	topology, err := topologyMapper.GetCurrentTopology()
	if err != nil {
		log.Printf("handleTopologyLive: failed to get topology: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get topology: %v", err), http.StatusInternalServerError)
		return
	}

	// Add live update metadata
	response := map[string]interface{}{
		"topology":              topology,
		"live_update":           true,
		"poll_interval_seconds": 5,     // Recommend polling every 5 seconds
		"supports_websocket":    false, // Future enhancement
	}

	log.Printf("handleTopologyLive: live topology update provided")
	s.writeJSON(w, response)
}

// handleChemistryConcentrations returns current concentration state
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryConcentrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleChemistryConcentrations: diffusion service not available")
		http.Error(w, "Diffusion service not available", http.StatusServiceUnavailable)
		return
	}

	chemEngine := diffusionService.GetChemistryEngine()
	if chemEngine == nil {
		log.Printf("handleChemistryConcentrations: chemistry engine not available")
		http.Error(w, "Chemistry engine not available", http.StatusServiceUnavailable)
		return
	}

	concentrationState := chemEngine.GetConcentrationState()
	log.Printf("handleChemistryConcentrations: returning concentration state")
	s.writeJSON(w, concentrationState)
}

// handleChemistryReactions returns reaction history
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryReactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleChemistryReactions: diffusion service not available")
		http.Error(w, "Diffusion service not available", http.StatusServiceUnavailable)
		return
	}

	chemEngine := diffusionService.GetChemistryEngine()
	if chemEngine == nil {
		log.Printf("handleChemistryReactions: chemistry engine not available")
		http.Error(w, "Chemistry engine not available", http.StatusServiceUnavailable)
		return
	}

	reactions := chemEngine.GetReactionHistory()
	response := map[string]interface{}{
		"reactions": reactions,
		"count":     len(reactions),
	}

	log.Printf("handleChemistryReactions: returning %d reactions", len(reactions))
	s.writeJSON(w, response)
}

// handleChemistryStats returns chemistry engine statistics
// Phase 4A: Chemistry monitoring
func (s *Server) handleChemistryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	diffusionService := s.node.GetDiffusionService()
	if diffusionService == nil {
		log.Printf("handleChemistryStats: diffusion service not available")
		http.Error(w, "Diffusion service not available", http.StatusServiceUnavailable)
		return
	}

	chemEngine := diffusionService.GetChemistryEngine()
	if chemEngine == nil {
		log.Printf("handleChemistryStats: chemistry engine not available")
		http.Error(w, "Chemistry engine not available", http.StatusServiceUnavailable)
		return
	}

	stats := chemEngine.GetChemistryStats()
	log.Printf("handleChemistryStats: returning chemistry statistics")
	s.writeJSON(w, stats)
}
