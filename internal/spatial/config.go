package spatial

import (
	"fmt"
	"strings"
)

// CoordinateSystem defines the type of coordinate system used by a node
type CoordinateSystem string

const (
	// GPS coordinates for fixed infrastructure (farms, data centers, smart cities)
	CoordSystemGPS CoordinateSystem = "gps"

	// Relative coordinates for vehicles (ships, aircraft, cars) - relative to platform center
	CoordSystemRelative CoordinateSystem = "relative"

	// Logical coordinates for cloud/virtual deployments (availability zones, racks)
	CoordSystemLogical CoordinateSystem = "logical"

	// No spatial awareness - backward compatibility for development/testing
	CoordSystemNone CoordinateSystem = "none"
)

// SpatialConfig represents the spatial configuration of a node
type SpatialConfig struct {
	// Coordinate system type
	CoordSystem CoordinateSystem `json:"coord_system"`

	// Physical/logical coordinates (meaning depends on CoordSystem)
	// nil values indicate coordinate not specified
	X *float64 `json:"x,omitempty"`
	Y *float64 `json:"y,omitempty"`
	Z *float64 `json:"z,omitempty"`

	// Logical zone identifier (always present)
	Zone string `json:"zone"`

	// Physical isolation boundaries this node respects
	Barriers []string `json:"barriers,omitempty"`
}

// NewSpatialConfig creates a new spatial configuration with validation
func NewSpatialConfig(coordSystem string, x, y, z *float64, zone string, barriers []string) (*SpatialConfig, error) {
	cs := CoordinateSystem(strings.ToLower(coordSystem))

	config := &SpatialConfig{
		CoordSystem: cs,
		X:           x,
		Y:           y,
		Z:           z,
		Zone:        zone,
		Barriers:    barriers,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if the spatial configuration is valid
func (sc *SpatialConfig) Validate() error {
	switch sc.CoordSystem {
	case CoordSystemGPS:
		return sc.validateGPS()
	case CoordSystemRelative:
		return sc.validateRelative()
	case CoordSystemLogical:
		return sc.validateLogical()
	case CoordSystemNone:
		return sc.validateNone()
	default:
		return fmt.Errorf("unknown coordinate system: %s (valid: gps, relative, logical, none)", sc.CoordSystem)
	}
}

// validateGPS validates GPS coordinate system configuration
func (sc *SpatialConfig) validateGPS() error {
	if sc.X != nil && (*sc.X < -180 || *sc.X > 180) {
		return fmt.Errorf("GPS longitude X must be between -180 and 180, got %f", *sc.X)
	}
	if sc.Y != nil && (*sc.Y < -90 || *sc.Y > 90) {
		return fmt.Errorf("GPS latitude Y must be between -90 and 90, got %f", *sc.Y)
	}
	// Z coordinate can be any altitude value
	return nil
}

// validateRelative validates relative coordinate system configuration
func (sc *SpatialConfig) validateRelative() error {
	// Relative coordinates can be any values - they're relative to platform center
	// No specific validation needed beyond basic sanity checks
	if sc.X != nil && (*sc.X < -1000000 || *sc.X > 1000000) {
		return fmt.Errorf("relative X coordinate seems unrealistic: %f", *sc.X)
	}
	if sc.Y != nil && (*sc.Y < -1000000 || *sc.Y > 1000000) {
		return fmt.Errorf("relative Y coordinate seems unrealistic: %f", *sc.Y)
	}
	if sc.Z != nil && (*sc.Z < -100000 || *sc.Z > 100000) {
		return fmt.Errorf("relative Z coordinate seems unrealistic: %f", *sc.Z)
	}
	return nil
}

// validateLogical validates logical coordinate system configuration
func (sc *SpatialConfig) validateLogical() error {
	// Logical systems primarily use zones, coordinates are optional
	if sc.Zone == "" {
		return fmt.Errorf("logical coordinate system requires a zone identifier")
	}
	return nil
}

// validateNone validates no coordinate system configuration
func (sc *SpatialConfig) validateNone() error {
	// No coordinates should be specified for 'none' system
	if sc.X != nil || sc.Y != nil || sc.Z != nil {
		return fmt.Errorf("coordinate system 'none' should not have X, Y, Z coordinates specified")
	}
	return nil
}

// HasCoordinates returns true if the node has explicit coordinates
func (sc *SpatialConfig) HasCoordinates() bool {
	return sc.X != nil || sc.Y != nil || sc.Z != nil
}

// IsEmpty returns true if this is an empty/default spatial configuration
func (sc *SpatialConfig) IsEmpty() bool {
	return sc.CoordSystem == CoordSystemNone &&
		sc.Zone == "" &&
		!sc.HasCoordinates() &&
		len(sc.Barriers) == 0
}

// String returns a human-readable representation of the spatial config
func (sc *SpatialConfig) String() string {
	if sc.IsEmpty() {
		return "spatial: none"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("system=%s", sc.CoordSystem))

	if sc.HasCoordinates() {
		coords := "coords=("
		if sc.X != nil {
			coords += fmt.Sprintf("%.3f", *sc.X)
		} else {
			coords += "nil"
		}
		coords += ","
		if sc.Y != nil {
			coords += fmt.Sprintf("%.3f", *sc.Y)
		} else {
			coords += "nil"
		}
		coords += ","
		if sc.Z != nil {
			coords += fmt.Sprintf("%.3f", *sc.Z)
		} else {
			coords += "nil"
		}
		coords += ")"
		parts = append(parts, coords)
	}

	if sc.Zone != "" {
		parts = append(parts, fmt.Sprintf("zone=%s", sc.Zone))
	}

	if len(sc.Barriers) > 0 {
		parts = append(parts, fmt.Sprintf("barriers=%v", sc.Barriers))
	}

	return "spatial: " + strings.Join(parts, " ")
}
