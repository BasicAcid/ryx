package ca

import (
	"log"
	"sync"
	"time"
)

// CellState represents the state of a single cell in the CA grid
type CellState int

const (
	Dead  CellState = 0
	Alive CellState = 1
)

// BoundaryStates represents the edge states shared between neighboring CA grids
type BoundaryStates struct {
	// Edge arrays for 4 directions: North, South, East, West
	North []CellState `json:"north"` // Top edge (y=0)
	South []CellState `json:"south"` // Bottom edge (y=height-1)
	East  []CellState `json:"east"`  // Right edge (x=width-1)
	West  []CellState `json:"west"`  // Left edge (x=0)

	// Metadata
	NodeID     string `json:"node_id"`
	Generation int    `json:"generation"`
}

// Cell represents a single cell in the cellular automata grid
type Cell struct {
	State     CellState `json:"state"`
	NextState CellState `json:"next_state"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
}

// Grid represents a 2D cellular automata grid
type Grid struct {
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Cells      [][]Cell `json:"cells"`
	Generation int      `json:"generation"`
	mu         sync.RWMutex
}

// Engine manages cellular automata computation for a node
type Engine struct {
	nodeID        string
	grid          *Grid
	updateRate    time.Duration // Time between CA generations
	running       bool
	stopChan      chan bool
	mu            sync.RWMutex
	lastUpdate    time.Time
	totalUpdates  int64
	updateHistory []time.Time // For calculating updates per second

	// Phase 3: CA Grid Connectivity
	neighborBoundaries map[string]*BoundaryStates // Remote boundary states by node ID
	boundaryMu         sync.RWMutex               // Separate mutex for boundary state access
	boundaryCallback   func(*BoundaryStates)      // Callback to broadcast boundary states
}

// NewEngine creates a new CA engine for a node
func NewEngine(nodeID string, width, height int) *Engine {
	log.Printf("CA[%s]: Creating new cellular automata engine (%dx%d)", nodeID, width, height)

	engine := &Engine{
		nodeID:        nodeID,
		updateRate:    time.Second, // 1 generation per second by default
		stopChan:      make(chan bool),
		updateHistory: make([]time.Time, 0, 10), // Keep last 10 updates for rate calculation

		// Phase 3: Initialize boundary exchange
		neighborBoundaries: make(map[string]*BoundaryStates),
	}

	engine.grid = engine.createGrid(width, height)
	return engine
}

// createGrid initializes a new CA grid with given dimensions
func (e *Engine) createGrid(width, height int) *Grid {
	cells := make([][]Cell, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			cells[y][x] = Cell{
				State:     Dead,
				NextState: Dead,
				X:         x,
				Y:         y,
			}
		}
	}

	return &Grid{
		Width:      width,
		Height:     height,
		Cells:      cells,
		Generation: 0,
	}
}

// Start begins the CA update loop
func (e *Engine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	log.Printf("CA[%s]: Starting cellular automata engine", e.nodeID)
	go e.updateLoop()
}

// Stop stops the CA update loop
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	e.mu.Unlock()

	log.Printf("CA[%s]: Stopping cellular automata engine", e.nodeID)

	// Non-blocking send with timeout to prevent hanging
	select {
	case e.stopChan <- true:
		log.Printf("CA[%s]: Stop signal sent successfully", e.nodeID)
	case <-time.After(2 * time.Second):
		log.Printf("CA[%s]: Stop signal timeout - forcing shutdown", e.nodeID)
	}
}

// updateLoop runs the cellular automata update cycle
func (e *Engine) updateLoop() {
	ticker := time.NewTicker(e.updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateGeneration()
		case <-e.stopChan:
			log.Printf("CA[%s]: Update loop stopped", e.nodeID)
			return
		}
	}
}

// updateGeneration applies Conway's Game of Life rules to advance one generation
func (e *Engine) updateGeneration() {
	e.grid.mu.Lock()
	defer e.grid.mu.Unlock()

	// Calculate next states for all cells
	for y := 0; y < e.grid.Height; y++ {
		for x := 0; x < e.grid.Width; x++ {
			neighbors := e.countLiveNeighbors(x, y)
			current := e.grid.Cells[y][x].State

			// Conway's Game of Life rules
			switch {
			case current == Alive && (neighbors == 2 || neighbors == 3):
				e.grid.Cells[y][x].NextState = Alive // Survival
			case current == Dead && neighbors == 3:
				e.grid.Cells[y][x].NextState = Alive // Birth
			default:
				e.grid.Cells[y][x].NextState = Dead // Death or stays dead
			}
		}
	}

	// Apply next states to current states (synchronous update)
	for y := 0; y < e.grid.Height; y++ {
		for x := 0; x < e.grid.Width; x++ {
			e.grid.Cells[y][x].State = e.grid.Cells[y][x].NextState
		}
	}

	// Update generation counter and statistics
	e.grid.Generation++
	e.updateStats()

	// Phase 3: Boundary broadcasting temporarily disabled - will be re-implemented with proper lock ordering
	// if e.boundaryCallback != nil {
	//	boundaries := e.GetBoundaryStates()
	//	go e.boundaryCallback(boundaries)
	// }

	if e.grid.Generation%10 == 0 {
		log.Printf("CA[%s]: Generation %d completed", e.nodeID, e.grid.Generation)
	}
}

// countLiveNeighbors counts living neighbors (temporarily using wrap-around to avoid deadlock)
func (e *Engine) countLiveNeighbors(x, y int) int {
	count := 0

	// Check all 8 neighbors with wrap-around (original working logic)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue // Skip the cell itself
			}

			nx, ny := x+dx, y+dy

			// Handle boundaries with wrap-around (restore original logic)
			if nx < 0 {
				nx = e.grid.Width - 1
			} else if nx >= e.grid.Width {
				nx = 0
			}

			if ny < 0 {
				ny = e.grid.Height - 1
			} else if ny >= e.grid.Height {
				ny = 0
			}

			if e.grid.Cells[ny][nx].State == Alive {
				count++
			}
		}
	}

	return count
}

// getBoundaryNeighborState gets the state of a boundary neighbor from connected grids
// For now, just return Dead to avoid deadlock - we'll implement proper boundary exchange later
func (e *Engine) getBoundaryNeighborState(x, y, dx, dy int) CellState {
	// Temporarily disable boundary exchange to avoid deadlock
	// This will be re-enabled once we implement proper lock ordering
	return Dead
}

// updateStats updates performance statistics
func (e *Engine) updateStats() {
	now := time.Now()
	e.lastUpdate = now
	e.totalUpdates++

	// Keep sliding window of update times
	e.updateHistory = append(e.updateHistory, now)
	if len(e.updateHistory) > 10 {
		e.updateHistory = e.updateHistory[1:]
	}
}

// SetCell sets a specific cell state (for testing/initialization)
func (e *Engine) SetCell(x, y int, state CellState) error {
	e.grid.mu.Lock()
	defer e.grid.mu.Unlock()

	if x < 0 || x >= e.grid.Width || y < 0 || y >= e.grid.Height {
		return nil // Ignore out-of-bounds
	}

	e.grid.Cells[y][x].State = state
	return nil
}

// GetGrid returns a copy of the current grid state
func (e *Engine) GetGrid() *Grid {
	e.grid.mu.RLock()
	defer e.grid.mu.RUnlock()

	// Create a copy to avoid race conditions
	gridCopy := &Grid{
		Width:      e.grid.Width,
		Height:     e.grid.Height,
		Generation: e.grid.Generation,
		Cells:      make([][]Cell, e.grid.Height),
	}

	for y := 0; y < e.grid.Height; y++ {
		gridCopy.Cells[y] = make([]Cell, e.grid.Width)
		copy(gridCopy.Cells[y], e.grid.Cells[y])
	}

	return gridCopy
}

// GetStats returns CA engine statistics
func (e *Engine) GetStats() map[string]interface{} {
	// Acquire locks in consistent order: grid first, then engine
	var liveCells int
	var generation, width, height int
	var running bool

	// Get grid stats first
	if e.grid != nil {
		e.grid.mu.RLock()
		generation = e.grid.Generation
		width = e.grid.Width
		height = e.grid.Height
		for y := 0; y < e.grid.Height; y++ {
			for x := 0; x < e.grid.Width; x++ {
				if e.grid.Cells[y][x].State == Alive {
					liveCells++
				}
			}
		}
		e.grid.mu.RUnlock()
	}

	// Then get engine stats
	e.mu.RLock()
	running = e.running
	totalUpdates := e.totalUpdates
	lastUpdate := e.lastUpdate
	updateRate := e.updateRate

	// Calculate updates per second
	var updatesPerSecond float64
	if len(e.updateHistory) > 1 {
		duration := e.updateHistory[len(e.updateHistory)-1].Sub(e.updateHistory[0])
		if duration > 0 {
			updatesPerSecond = float64(len(e.updateHistory)-1) / duration.Seconds()
		}
	}
	e.mu.RUnlock()

	return map[string]interface{}{
		"node_id":            e.nodeID,
		"generation":         generation,
		"grid_width":         width,
		"grid_height":        height,
		"total_cells":        width * height,
		"live_cells":         liveCells,
		"dead_cells":         (width * height) - liveCells,
		"running":            running,
		"total_updates":      totalUpdates,
		"updates_per_second": updatesPerSecond,
		"last_update":        lastUpdate.Unix(),
		"update_rate_ms":     int(updateRate.Milliseconds()),
	}
}

// InitializePattern initializes the grid with a simple pattern for testing
func (e *Engine) InitializePattern(pattern string) {
	e.grid.mu.Lock()
	defer e.grid.mu.Unlock()

	// Clear grid first
	for y := 0; y < e.grid.Height; y++ {
		for x := 0; x < e.grid.Width; x++ {
			e.grid.Cells[y][x].State = Dead
		}
	}

	switch pattern {
	case "blinker":
		// Simple oscillating pattern
		if e.grid.Width >= 5 && e.grid.Height >= 3 {
			centerX, centerY := e.grid.Width/2, e.grid.Height/2
			e.grid.Cells[centerY][centerX-1].State = Alive
			e.grid.Cells[centerY][centerX].State = Alive
			e.grid.Cells[centerY][centerX+1].State = Alive
		}
	case "glider":
		// Famous glider pattern
		if e.grid.Width >= 5 && e.grid.Height >= 5 {
			e.grid.Cells[1][2].State = Alive
			e.grid.Cells[2][3].State = Alive
			e.grid.Cells[3][1].State = Alive
			e.grid.Cells[3][2].State = Alive
			e.grid.Cells[3][3].State = Alive
		}
	case "random":
		// Random initialization (10% alive)
		for y := 0; y < e.grid.Height; y++ {
			for x := 0; x < e.grid.Width; x++ {
				if time.Now().UnixNano()%10 == 0 {
					e.grid.Cells[y][x].State = Alive
				}
			}
		}
	}

	log.Printf("CA[%s]: Initialized with pattern '%s'", e.nodeID, pattern)
}

// Phase 3: CA Grid Connectivity Methods

// GetBoundaryStates extracts boundary states from current grid
func (e *Engine) GetBoundaryStates() *BoundaryStates {
	e.grid.mu.RLock()
	defer e.grid.mu.RUnlock()

	width, height := e.grid.Width, e.grid.Height
	boundaries := &BoundaryStates{
		North:      make([]CellState, width),
		South:      make([]CellState, width),
		East:       make([]CellState, height),
		West:       make([]CellState, height),
		NodeID:     e.nodeID,
		Generation: e.grid.Generation,
	}

	// Extract edge states
	for x := 0; x < width; x++ {
		boundaries.North[x] = e.grid.Cells[0][x].State        // Top edge
		boundaries.South[x] = e.grid.Cells[height-1][x].State // Bottom edge
	}

	for y := 0; y < height; y++ {
		boundaries.West[y] = e.grid.Cells[y][0].State       // Left edge
		boundaries.East[y] = e.grid.Cells[y][width-1].State // Right edge
	}

	return boundaries
}

// UpdateNeighborBoundary updates the boundary states from a neighboring node
func (e *Engine) UpdateNeighborBoundary(neighborNodeID string, boundaries *BoundaryStates) {
	e.boundaryMu.Lock()
	defer e.boundaryMu.Unlock()

	if boundaries != nil && boundaries.NodeID == neighborNodeID {
		e.neighborBoundaries[neighborNodeID] = boundaries
		log.Printf("CA[%s]: Updated boundary states from neighbor %s (gen %d)",
			e.nodeID, neighborNodeID, boundaries.Generation)
	}
}

// RemoveNeighborBoundary removes boundary states when a neighbor disconnects
func (e *Engine) RemoveNeighborBoundary(neighborNodeID string) {
	e.boundaryMu.Lock()
	defer e.boundaryMu.Unlock()

	if _, exists := e.neighborBoundaries[neighborNodeID]; exists {
		delete(e.neighborBoundaries, neighborNodeID)
		log.Printf("CA[%s]: Removed boundary states from disconnected neighbor %s", e.nodeID, neighborNodeID)
	}
}

// GetConnectedNeighbors returns list of neighbors with active boundary connections
func (e *Engine) GetConnectedNeighbors() []string {
	e.boundaryMu.RLock()
	defer e.boundaryMu.RUnlock()

	neighbors := make([]string, 0, len(e.neighborBoundaries))
	for nodeID := range e.neighborBoundaries {
		neighbors = append(neighbors, nodeID)
	}
	return neighbors
}

// SetBoundaryCallback sets the callback function for broadcasting boundary states
func (e *Engine) SetBoundaryCallback(callback func(*BoundaryStates)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.boundaryCallback = callback
}
