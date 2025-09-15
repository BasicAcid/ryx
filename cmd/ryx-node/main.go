package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/BasicAcid/ryx/internal/node"
	"github.com/BasicAcid/ryx/internal/spatial"
)

func main() {
	// Command line flags
	port := flag.Int("port", 9001, "UDP port to listen on")
	httpPort := flag.Int("http-port", 8001, "HTTP API port")
	clusterID := flag.String("cluster-id", "default", "Cluster identifier")
	nodeID := flag.String("node-id", "", "Node identifier (auto-generated if empty)")

	// Phase 3C.1: Spatial configuration flags
	coordSystem := flag.String("coord-system", "none", "Coordinate system: gps, relative, logical, none")
	x := flag.Float64("x", 0, "X coordinate (longitude for GPS, meters for relative)")
	y := flag.Float64("y", 0, "Y coordinate (latitude for GPS, meters for relative)")
	z := flag.Float64("z", 0, "Z coordinate (altitude/height in meters)")
	zone := flag.String("zone", "default", "Logical zone identifier")
	barriers := flag.String("barriers", "", "Comma-separated list of barriers (format: type:zoneA:zoneB:isolation)")

	flag.Parse()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse spatial configuration from CLI flags
	spatialConfig, err := parseSpatialConfig(*coordSystem, *x, *y, *z, *zone, *barriers)
	if err != nil {
		log.Fatalf("Invalid spatial configuration: %v", err)
	}

	// Create and start the node
	config := &node.Config{
		Port:          *port,
		HTTPPort:      *httpPort,
		ClusterID:     *clusterID,
		NodeID:        *nodeID,
		SpatialConfig: spatialConfig,
	}

	n, err := node.New(config)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	log.Printf("Starting ryx-node %s on UDP:%d HTTP:%d cluster:%s spatial:%s",
		n.ID(), *port, *httpPort, *clusterID, spatialConfig.String())

	// Start the node in a goroutine
	if err := n.Start(ctx); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
	n.Stop()
	log.Println("Shutdown complete")
}

// parseSpatialConfig creates a spatial configuration from CLI arguments
func parseSpatialConfig(coordSystem string, x, y, z float64, zone, barriers string) (*spatial.SpatialConfig, error) {
	// Handle coordinate values - only set if not zero or if coord system requires them
	var xPtr, yPtr, zPtr *float64

	if coordSystem != "none" {
		// For non-none systems, include coordinates if they're non-zero or if it's GPS/relative
		if x != 0 || coordSystem == "gps" || coordSystem == "relative" {
			xPtr = &x
		}
		if y != 0 || coordSystem == "gps" || coordSystem == "relative" {
			yPtr = &y
		}
		if z != 0 || coordSystem == "gps" || coordSystem == "relative" {
			zPtr = &z
		}
	}

	// Parse barriers list
	var barrierList []string
	if barriers != "" {
		barrierList = strings.Split(barriers, ",")
		// Trim whitespace from each barrier
		for i, barrier := range barrierList {
			barrierList[i] = strings.TrimSpace(barrier)
		}
	}

	return spatial.NewSpatialConfig(coordSystem, xPtr, yPtr, zPtr, zone, barrierList)
}
