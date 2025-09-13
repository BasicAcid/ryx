package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BasicAcid/ryx/internal/diffusion"
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

// NodeProvider combines both interfaces
type NodeProvider interface {
	NodeStatusProvider
	DiffusionProvider
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

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
