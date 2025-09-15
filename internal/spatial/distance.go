package spatial

import (
	"fmt"
	"math"
)

// Distance represents a spatial distance between two nodes
type Distance struct {
	Value       float64          `json:"value"`        // Distance value
	Unit        string           `json:"unit"`         // Distance unit (meters, degrees, logical)
	CoordSystem CoordinateSystem `json:"coord_system"` // Which coordinate system was used
}

// CalculateDistance computes the distance between two spatial configurations
func CalculateDistance(from, to *SpatialConfig) (*Distance, error) {
	// Handle nil inputs
	if from == nil || to == nil {
		return nil, fmt.Errorf("cannot calculate distance with nil spatial config")
	}

	// Both nodes must use the same coordinate system
	if from.CoordSystem != to.CoordSystem {
		return nil, fmt.Errorf("cannot calculate distance between different coordinate systems: %s vs %s",
			from.CoordSystem, to.CoordSystem)
	}

	// Handle nodes without coordinates
	if !from.HasCoordinates() || !to.HasCoordinates() {
		return &Distance{
			Value:       math.Inf(1), // Infinite distance if no coordinates
			Unit:        "unknown",
			CoordSystem: from.CoordSystem,
		}, nil
	}

	switch from.CoordSystem {
	case CoordSystemGPS:
		return calculateGPSDistance(from, to)
	case CoordSystemRelative:
		return calculateEuclideanDistance(from, to, "meters")
	case CoordSystemLogical:
		return calculateLogicalDistance(from, to)
	case CoordSystemNone:
		return &Distance{
			Value:       math.Inf(1),
			Unit:        "none",
			CoordSystem: CoordSystemNone,
		}, nil
	default:
		return nil, fmt.Errorf("unknown coordinate system: %s", from.CoordSystem)
	}
}

// calculateGPSDistance computes distance between GPS coordinates using Haversine formula
func calculateGPSDistance(from, to *SpatialConfig) (*Distance, error) {
	// Get coordinates with defaults for missing values
	lat1 := getCoordOrDefault(from.Y, 0.0) // Latitude
	lon1 := getCoordOrDefault(from.X, 0.0) // Longitude
	lat2 := getCoordOrDefault(to.Y, 0.0)
	lon2 := getCoordOrDefault(to.X, 0.0)

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Earth radius in meters
	const earthRadius = 6371000
	distance := earthRadius * c

	// Add altitude difference if both Z coordinates are present
	if from.Z != nil && to.Z != nil {
		altDiff := *to.Z - *from.Z
		distance = math.Sqrt(distance*distance + altDiff*altDiff)
	}

	return &Distance{
		Value:       distance,
		Unit:        "meters",
		CoordSystem: CoordSystemGPS,
	}, nil
}

// calculateEuclideanDistance computes 3D Euclidean distance
func calculateEuclideanDistance(from, to *SpatialConfig, unit string) (*Distance, error) {
	x1 := getCoordOrDefault(from.X, 0.0)
	y1 := getCoordOrDefault(from.Y, 0.0)
	z1 := getCoordOrDefault(from.Z, 0.0)

	x2 := getCoordOrDefault(to.X, 0.0)
	y2 := getCoordOrDefault(to.Y, 0.0)
	z2 := getCoordOrDefault(to.Z, 0.0)

	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1

	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	return &Distance{
		Value:       distance,
		Unit:        unit,
		CoordSystem: from.CoordSystem,
	}, nil
}

// calculateLogicalDistance computes logical distance (same zone = 0, different zone = 1)
func calculateLogicalDistance(from, to *SpatialConfig) (*Distance, error) {
	// Logical distance based on zone membership
	distance := 0.0
	if from.Zone != to.Zone {
		distance = 1.0
	}

	// If both have coordinates, add small Euclidean component
	if from.HasCoordinates() && to.HasCoordinates() {
		euclidean, err := calculateEuclideanDistance(from, to, "logical")
		if err != nil {
			return nil, err
		}
		// Add small coordinate-based component (scaled down)
		distance += euclidean.Value * 0.01
	}

	return &Distance{
		Value:       distance,
		Unit:        "logical",
		CoordSystem: CoordSystemLogical,
	}, nil
}

// getCoordOrDefault returns the coordinate value or a default if nil
func getCoordOrDefault(coord *float64, defaultValue float64) float64 {
	if coord == nil {
		return defaultValue
	}
	return *coord
}

// IsInSameZone returns true if two nodes are in the same logical zone
func IsInSameZone(config1, config2 *SpatialConfig) bool {
	if config1 == nil || config2 == nil {
		return false
	}
	return config1.Zone != "" && config1.Zone == config2.Zone
}

// IsWithinDistance returns true if two nodes are within the specified distance
func IsWithinDistance(from, to *SpatialConfig, maxDistance float64) (bool, error) {
	distance, err := CalculateDistance(from, to)
	if err != nil {
		return false, err
	}

	// Handle infinite distance (nodes without coordinates)
	if math.IsInf(distance.Value, 1) {
		return false, nil
	}

	return distance.Value <= maxDistance, nil
}

// String returns a human-readable representation of the distance
func (d *Distance) String() string {
	if math.IsInf(d.Value, 1) {
		return fmt.Sprintf("∞ %s (%s)", d.Unit, d.CoordSystem)
	}
	return fmt.Sprintf("%.3f %s (%s)", d.Value, d.Unit, d.CoordSystem)
}
