package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DashboardServer struct {
	port         int
	startPort    int
	endPort      int
	httpTemplate *template.Template
}

type NodeInfo struct {
	ID             string  `json:"id"`
	Port           int     `json:"port"`
	HTTPPort       int     `json:"http_port"`
	Status         string  `json:"status"`
	Neighbors      int     `json:"neighbor_count"`
	Tasks          int     `json:"active_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	ChemEnergy     float64 `json:"chemistry_energy,omitempty"`
	Reachable      bool    `json:"reachable"`
	LastUpdate     string  `json:"last_update"`
}

type ClusterStatus struct {
	Nodes       []NodeInfo `json:"nodes"`
	TotalNodes  int        `json:"total_nodes"`
	ActiveNodes int        `json:"active_nodes"`
	Timestamp   string     `json:"timestamp"`
}

func NewDashboardServer(port, startPort, endPort int) (*DashboardServer, error) {
	tmpl, err := template.ParseFiles("cmd/ryx-dashboard/templates/dashboard.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &DashboardServer{
		port:         port,
		startPort:    startPort,
		endPort:      endPort,
		httpTemplate: tmpl,
	}, nil
}

func (ds *DashboardServer) discoverCluster() ClusterStatus {
	var nodes []NodeInfo
	activeCount := 0

	// Scan port range for active nodes
	for httpPort := ds.startPort; httpPort <= ds.endPort; httpPort++ {
		nodePort := httpPort + 1000 // HTTP port is UDP port - 1000

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/status", httpPort))

		node := NodeInfo{
			HTTPPort:  httpPort,
			Port:      nodePort,
			Reachable: false,
		}

		if err != nil {
			// Node not reachable
			nodes = append(nodes, node)
			continue
		}
		defer resp.Body.Close()

		// Parse node status
		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			nodes = append(nodes, node)
			continue
		}

		node.Reachable = true
		node.ID = getString(status, "node_id")
		node.Status = "healthy" // Default to healthy if reachable
		node.Neighbors = getInt(status, "neighbor_count")

		// Get computation statistics
		compResp, err := client.Get(fmt.Sprintf("http://localhost:%d/compute", httpPort))
		if err == nil {
			defer compResp.Body.Close()
			var compData map[string]interface{}
			if json.NewDecoder(compResp.Body).Decode(&compData) == nil {
				if stats, ok := compData["stats"].(map[string]interface{}); ok {
					node.Tasks = getInt(stats, "active_tasks")
					node.CompletedTasks = getInt(stats, "completed_tasks")
				}
			}
		}

		node.LastUpdate = time.Now().Format("15:04:05")

		// Try to get chemistry energy
		chemResp, err := client.Get(fmt.Sprintf("http://localhost:%d/chemistry/stats", httpPort))
		if err == nil {
			defer chemResp.Body.Close()
			var chemStats map[string]interface{}
			if json.NewDecoder(chemResp.Body).Decode(&chemStats) == nil {
				node.ChemEnergy = getFloat(chemStats, "total_energy")
			}
		}

		nodes = append(nodes, node)
		activeCount++
	}

	// Sort nodes by port for consistent display
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].HTTPPort < nodes[j].HTTPPort
	})

	return ClusterStatus{
		Nodes:       nodes,
		TotalNodes:  len(nodes),
		ActiveNodes: activeCount,
		Timestamp:   time.Now().Format("15:04:05"),
	}
}

func (ds *DashboardServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if err := ds.httpTemplate.Execute(w, nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (ds *DashboardServer) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	status := ds.discoverCluster()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (ds *DashboardServer) handleNodeProxy(w http.ResponseWriter, r *http.Request) {
	// Extract node HTTP port from URL path
	portStr := r.URL.Query().Get("port")
	if portStr == "" {
		http.Error(w, "Missing port parameter", http.StatusBadRequest)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port parameter", http.StatusBadRequest)
		return
	}

	// Extract the API path (everything after /node-proxy)
	apiPath := r.URL.Query().Get("path")
	if apiPath == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	// Proxy request to node
	client := &http.Client{Timeout: 5 * time.Second}
	targetURL := fmt.Sprintf("http://localhost:%d%s", port, apiPath)

	var resp *http.Response
	switch r.Method {
	case "GET":
		resp, err = client.Get(targetURL)
	case "POST":
		resp, err = client.Post(targetURL, r.Header.Get("Content-Type"), r.Body)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Node request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers and body
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Stream response body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func (ds *DashboardServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files from cmd/ryx-dashboard/static/
	http.StripPrefix("/static/", http.FileServer(http.Dir("cmd/ryx-dashboard/static/"))).ServeHTTP(w, r)
}

func (ds *DashboardServer) handleClusterTaskSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse task submission request
	var taskRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find first healthy node to submit task to
	status := ds.discoverCluster()
	var targetNode *NodeInfo
	for _, node := range status.Nodes {
		if node.Reachable {
			targetNode = &node
			break
		}
	}

	if targetNode == nil {
		http.Error(w, "No healthy nodes available", http.StatusServiceUnavailable)
		return
	}

	// Submit task to target node
	client := &http.Client{Timeout: 5 * time.Second}
	taskJSON, _ := json.Marshal(taskRequest)

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/compute", targetNode.HTTPPort),
		"application/json",
		strings.NewReader(string(taskJSON)),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit task: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Return the response from the node
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func (ds *DashboardServer) Start() error {
	http.HandleFunc("/", ds.handleDashboard)
	http.HandleFunc("/cluster/status", ds.handleClusterStatus)
	http.HandleFunc("/cluster/submit-task", ds.handleClusterTaskSubmit)
	http.HandleFunc("/node-proxy", ds.handleNodeProxy)
	http.HandleFunc("/static/", ds.handleStatic)

	log.Printf("Ryx Dashboard starting on http://localhost:%d", ds.port)
	log.Printf("Scanning for nodes on HTTP ports %d-%d", ds.startPort, ds.endPort)

	return http.ListenAndServe(fmt.Sprintf(":%d", ds.port), nil)
}

// Helper functions for safe type conversion
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

func main() {
	var (
		port      = flag.Int("port", 7000, "Dashboard HTTP port")
		startPort = flag.Int("start-port", 8010, "Start of HTTP port range to scan")
		endPort   = flag.Int("end-port", 8050, "End of HTTP port range to scan")
	)
	flag.Parse()

	dashboard, err := NewDashboardServer(*port, *startPort, *endPort)
	if err != nil {
		log.Fatalf("Failed to create dashboard: %v", err)
	}

	if err := dashboard.Start(); err != nil {
		log.Fatalf("Dashboard server failed: %v", err)
	}
}
