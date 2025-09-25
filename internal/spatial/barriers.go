package spatial

import (
	"fmt"
	"sync"
)

// BarrierType defines the type of physical barrier
type BarrierType string

const (
	// Physical barriers
	BarrierFirewall BarrierType = "firewall" // Network firewall/security boundary
	BarrierBulkhead BarrierType = "bulkhead" // Physical wall/compartment separator
	BarrierZone     BarrierType = "zone"     // Zone boundary (maintenance, security, etc.)
	BarrierDistance BarrierType = "distance" // Distance-based isolation

	// Isolation types
	IsolationFault       = "fault"       // Fault isolation boundary
	IsolationMaintenance = "maintenance" // Maintenance isolation
	IsolationSecurity    = "security"    // Security boundary
	IsolationEmergency   = "emergency"   // Emergency isolation
)

// PhysicalBarrier represents a barrier that can block communication or movement
type PhysicalBarrier struct {
	ID          string      `json:"id"`          // Unique barrier identifier
	Type        BarrierType `json:"type"`        // Barrier type
	Description string      `json:"description"` // Human-readable description
	Isolation   string      `json:"isolation"`   // Type of isolation provided

	// Zone-based barriers (most common)
	ZoneA string `json:"zone_a,omitempty"` // First zone blocked
	ZoneB string `json:"zone_b,omitempty"` // Second zone blocked

	// Coordinate-based barriers (future use)
	Geometry []float64 `json:"geometry,omitempty"` // Barrier coordinates/geometry
}

// NewZoneBarrier creates a barrier between two zones
func NewZoneBarrier(id, zoneA, zoneB string, barrierType BarrierType, isolation string) *PhysicalBarrier {
	return &PhysicalBarrier{
		ID:          id,
		Type:        barrierType,
		Description: fmt.Sprintf("%s barrier between %s and %s", barrierType, zoneA, zoneB),
		Isolation:   isolation,
		ZoneA:       zoneA,
		ZoneB:       zoneB,
	}
}

// BlocksPath returns true if this barrier blocks communication between two spatial configs
func (b *PhysicalBarrier) BlocksPath(from, to *SpatialConfig) bool {
	if from == nil || to == nil {
		return false
	}

	// Same zone nodes are never blocked by barriers
	if IsInSameZone(from, to) {
		return false
	}

	switch b.Type {
	case BarrierZone:
		return b.blocksZonePath(from, to)
	case BarrierFirewall:
		return b.blocksFirewallPath(from, to)
	case BarrierBulkhead:
		return b.blocksBulkheadPath(from, to)
	case BarrierDistance:
		return b.blocksDistancePath(from, to)
	default:
		return false
	}
}

// blocksZonePath checks if zone barrier blocks path
func (b *PhysicalBarrier) blocksZonePath(from, to *SpatialConfig) bool {
	// Zone barriers are disabled in simplified model
	return false
}

// blocksFirewallPath checks if firewall blocks path
func (b *PhysicalBarrier) blocksFirewallPath(from, to *SpatialConfig) bool {
	// Firewall blocks based on zone boundary (similar to zone barrier)
	return b.blocksZonePath(from, to)
}

// blocksBulkheadPath checks if bulkhead blocks path
func (b *PhysicalBarrier) blocksBulkheadPath(from, to *SpatialConfig) bool {
	// Bulkhead provides physical isolation between zones
	return b.blocksZonePath(from, to)
}

// blocksDistancePath checks if distance barrier blocks path
func (b *PhysicalBarrier) blocksDistancePath(from, to *SpatialConfig) bool {
	// Distance barriers block based on physical distance
	// This would require coordinate-based geometry calculations (future feature)
	return false
}

// ShouldRespectBarrier returns true if a message type should respect this barrier
func (b *PhysicalBarrier) ShouldRespectBarrier(messageType string) bool {
	switch messageType {
	case "emergency", "critical":
		// Emergency/critical messages can cross most barriers
		return b.Isolation == IsolationSecurity // Only respect security barriers
	case "routine":
		// Routine messages respect all barriers
		return true
	case "maintenance":
		// Maintenance messages respect fault barriers but not maintenance barriers
		return b.Isolation != IsolationMaintenance
	default:
		// Default: respect all barriers
		return true
	}
}

// BarrierManager manages a collection of physical barriers
type BarrierManager struct {
	barriers map[string]*PhysicalBarrier
	mu       sync.RWMutex
}

// NewBarrierManager creates a new barrier manager
func NewBarrierManager() *BarrierManager {
	return &BarrierManager{
		barriers: make(map[string]*PhysicalBarrier),
	}
}

// AddBarrier adds a barrier to the manager (DISABLED FOR DEADLOCK DEBUGGING)
func (bm *BarrierManager) AddBarrier(barrier *PhysicalBarrier) {
	// DISABLED: No-op to eliminate barrier management
}

// RemoveBarrier removes a barrier from the manager
func (bm *BarrierManager) RemoveBarrier(barrierID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	delete(bm.barriers, barrierID)
}

// GetBarrier retrieves a barrier by ID
func (bm *BarrierManager) GetBarrier(barrierID string) *PhysicalBarrier {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.barriers[barrierID]
}

// GetAllBarriers returns all barriers (DISABLED FOR DEADLOCK DEBUGGING)
func (bm *BarrierManager) GetAllBarriers() map[string]*PhysicalBarrier {
	// DISABLED: Return empty map to eliminate any potential deadlock source
	return make(map[string]*PhysicalBarrier)
}

// FilterBarriers returns barriers that affect the given zones
func (bm *BarrierManager) FilterBarriers(zones []string) []*PhysicalBarrier {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var result []*PhysicalBarrier

	zoneSet := make(map[string]bool)
	for _, zone := range zones {
		zoneSet[zone] = true
	}

	for _, barrier := range bm.barriers {
		if zoneSet[barrier.ZoneA] || zoneSet[barrier.ZoneB] {
			result = append(result, barrier)
		}
	}

	return result
}

// PathBlocked returns true if any barrier blocks the path for the given message type (DISABLED FOR DEADLOCK DEBUGGING)
func (bm *BarrierManager) PathBlocked(from, to *SpatialConfig, messageType string) bool {
	// DISABLED: Always return false to eliminate barrier checking
	return false
}

// GetBlockingBarriers returns all barriers that would block the given path
func (bm *BarrierManager) GetBlockingBarriers(from, to *SpatialConfig, messageType string) []*PhysicalBarrier {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var blocking []*PhysicalBarrier

	for _, barrier := range bm.barriers {
		if barrier.ShouldRespectBarrier(messageType) && barrier.BlocksPath(from, to) {
			blocking = append(blocking, barrier)
		}
	}

	return blocking
}

// LoadBarriersFromConfig loads barriers from node barrier configuration (DISABLED FOR DEADLOCK DEBUGGING)
func (bm *BarrierManager) LoadBarriersFromConfig(nodeConfig *SpatialConfig) {
	// DISABLED: No-op to eliminate barrier configuration loading
}

// String returns a human-readable representation of the barrier
func (b *PhysicalBarrier) String() string {
	return fmt.Sprintf("%s barrier '%s' (%s isolation) between %s and %s",
		b.Type, b.ID, b.Isolation, b.ZoneA, b.ZoneB)
}
