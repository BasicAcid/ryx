package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BasicAcid/ryx/internal/node"
)

func main() {
	// Command line flags
	port := flag.Int("port", 9001, "UDP port to listen on")
	httpPort := flag.Int("http-port", 8001, "HTTP API port")
	clusterID := flag.String("cluster-id", "default", "Cluster identifier")
	nodeID := flag.String("node-id", "", "Node identifier (auto-generated if empty)")
	flag.Parse()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start the node
	config := &node.Config{
		Port:      *port,
		HTTPPort:  *httpPort,
		ClusterID: *clusterID,
		NodeID:    *nodeID,
	}

	n, err := node.New(config)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	log.Printf("Starting ryx-node %s on UDP:%d HTTP:%d cluster:%s",
		n.ID(), *port, *httpPort, *clusterID)

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
