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

// NodeProvider combines all interfaces
type NodeProvider interface {
	NodeStatusProvider
	DiffusionProvider
	ComputationProvider
	ConfigurationProvider
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
		Type    string `json:"type"`
		Content string `json:"content"`
		Energy  int    `json:"energy"`
		TTL     int    `json:"ttl"` // TTL in seconds
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
