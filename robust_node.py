#!/usr/bin/env python3
"""
Robust Computing Node - Dave Ackley inspired distributed computing
Each node communicates only with immediate neighbors via UDP sockets
"""

import socket
import json
import time
import threading
import random
import sys
import argparse
from dataclasses import dataclass, asdict
from typing import Dict, List, Set, Optional
import logging

@dataclass
class Message:
    type: str
    sender_id: str
    data: dict
    energy: int = 0
    hops: int = 0

@dataclass
class NodeInfo:
    node_id: str
    host: str
    port: int
    grid_x: int
    grid_y: int

class RobustNode:
    def __init__(self, node_id: str, port: int, grid_x: int, grid_y: int, grid_size: int = 10):
        self.node_id = node_id
        self.port = port
        self.host = "127.0.0.1"
        self.grid_x = grid_x
        self.grid_y = grid_y
        self.grid_size = grid_size

        # Node state
        self.alive = True
        self.neighbors: Dict[str, NodeInfo] = {}
        self.information: Dict[str, dict] = {}  # Information this node has
        self.generation = 0

        # Network
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket.bind((self.host, self.port))
        self.socket.settimeout(0.1)  # Non-blocking with timeout

        # Threads
        self.running = True
        self.listen_thread = threading.Thread(target=self._listen_loop, daemon=True)
        self.process_thread = threading.Thread(target=self._process_loop, daemon=True)

        # Logging
        logging.basicConfig(level=logging.INFO,
                          format=f'[{self.node_id}] %(asctime)s - %(message)s')
        self.logger = logging.getLogger(self.node_id)

        self.logger.info(f"Node {node_id} starting at {self.host}:{self.port} position ({grid_x},{grid_y})")

    def start(self):
        """Start the node's processing threads"""
        self.listen_thread.start()
        self.process_thread.start()

    def discover_neighbors(self, base_port: int):
        """Discover immediate neighbors in the grid"""
        # Calculate neighbor positions (8-connected grid)
        neighbor_positions = [
            (self.grid_x + dx, self.grid_y + dy)
            for dx in [-1, 0, 1]
            for dy in [-1, 0, 1]
            if dx != 0 or dy != 0  # Exclude self
        ]

        # Filter valid positions
        valid_neighbors = [
            (x, y) for x, y in neighbor_positions
            if 0 <= x < self.grid_size and 0 <= y < self.grid_size
        ]

        # Convert to ports and create neighbor info
        for x, y in valid_neighbors:
            neighbor_port = base_port + (y * self.grid_size + x)
            neighbor_id = f"node_{x}_{y}"

            if neighbor_id != self.node_id:  # Don't add self
                self.neighbors[neighbor_id] = NodeInfo(
                    node_id=neighbor_id,
                    host=self.host,
                    port=neighbor_port,
                    grid_x=x,
                    grid_y=y
                )

        self.logger.info(f"Discovered {len(self.neighbors)} neighbors: {list(self.neighbors.keys())}")

    def _listen_loop(self):
        """Listen for incoming messages"""
        while self.running:
            try:
                data, addr = self.socket.recvfrom(1024)
                message_dict = json.loads(data.decode())
                message = Message(**message_dict)
                self._handle_message(message, addr)
            except socket.timeout:
                continue
            except Exception as e:
                if self.running:  # Only log if we're supposed to be running
                    self.logger.error(f"Error in listen loop: {e}")

    def _handle_message(self, message: Message, sender_addr):
        """Handle incoming message based on type"""
        if message.type == "ping":
            self._handle_ping(message, sender_addr)
        elif message.type == "information":
            self._handle_information(message)
        elif message.type == "heartbeat":
            self._handle_heartbeat(message)
        elif message.type == "die":
            self._handle_die(message)

    def _handle_ping(self, message: Message, sender_addr):
        """Respond to ping with detailed status"""
        response = Message(
            type="pong",
            sender_id=self.node_id,
            data={
                "alive": self.alive,
                "grid_x": self.grid_x,
                "grid_y": self.grid_y,
                "information_count": len(self.information),
                "information": {
                    info_id: {
                        "energy": info["energy"],
                        "hops": info["hops"]
                    }
                    for info_id, info in self.information.items()
                },
                "generation": self.generation,
                "neighbors": len(self.neighbors)
            }
        )

        # Send response back to sender
        try:
            response_data = json.dumps(asdict(response)).encode()
            self.socket.sendto(response_data, sender_addr)
        except Exception as e:
            self.logger.error(f"Failed to send pong response: {e}")

    def _handle_information(self, message: Message):
        """Handle information spreading"""
        if not self.alive:
            return

        info_id = message.data.get("info_id")
        if info_id and info_id not in self.information and message.energy > 0:
            # Accept the information
            self.information[info_id] = {
                "content": message.data.get("content"),
                "energy": message.energy - 1,
                "received_from": message.sender_id,
                "hops": message.hops + 1
            }
            self.logger.info(f"Received information '{info_id}' with energy {message.energy-1}")

    def _handle_heartbeat(self, message: Message):
        """Handle heartbeat from neighbor"""
        pass  # Could track neighbor liveness here

    def _handle_die(self, message: Message):
        """Handle command to simulate node failure"""
        self.logger.info("Received DIE command - simulating node failure")
        self.alive = False
        # Also clear any information when we "die"
        self.information.clear()

    def _process_loop(self):
        """Main processing loop - spread information, decay energy, etc."""
        while self.running:
            if self.alive:
                self._spread_information()
                self._decay_information()
                self.generation += 1
            time.sleep(1.0)  # Process every second

    def _spread_information(self):
        """Spread information to neighbors (key Ackley principle)"""
        for info_id, info in list(self.information.items()):
            if info["energy"] > 0:
                # Create message to spread
                message = Message(
                    type="information",
                    sender_id=self.node_id,
                    data={
                        "info_id": info_id,
                        "content": info["content"]
                    },
                    energy=info["energy"],
                    hops=info["hops"]
                )

                # Send to all neighbors (alive or not - let the network decide)
                for neighbor_id in self.neighbors:
                    self._send_to_node(neighbor_id, message)

    def _decay_information(self):
        """Decay information energy over time"""
        for info_id in list(self.information.keys()):
            self.information[info_id]["energy"] -= 1

            # Remove information with no energy
            if self.information[info_id]["energy"] <= 0:
                self.logger.info(f"Information '{info_id}' decayed away")
                del self.information[info_id]

    def _send_to_node(self, target_node_id: str, message: Message):
        """Send message to a specific node"""
        if target_node_id in self.neighbors:
            neighbor = self.neighbors[target_node_id]
            try:
                data = json.dumps(asdict(message)).encode()
                self.socket.sendto(data, (neighbor.host, neighbor.port))
            except Exception as e:
                # Don't log connection errors - neighbor might be dead
                pass

    def inject_information(self, info_id: str, content: str, energy: int = 10):
        """Inject new information into the network"""
        if self.alive:
            self.information[info_id] = {
                "content": content,
                "energy": energy,
                "received_from": "self",
                "hops": 0
            }
            self.logger.info(f"Injected information '{info_id}' with energy {energy}")

    def get_status(self) -> dict:
        """Get current node status"""
        return {
            "node_id": self.node_id,
            "alive": self.alive,
            "position": (self.grid_x, self.grid_y),
            "neighbors": len(self.neighbors),
            "information_count": len(self.information),
            "information": {
                info_id: {
                    "energy": info["energy"],
                    "hops": info["hops"]
                }
                for info_id, info in self.information.items()
            },
            "generation": self.generation
        }

    def simulate_failure(self):
        """Simulate node failure"""
        self.logger.info("Simulating node failure")
        self.alive = False
        self.information.clear()

    def revive(self):
        """Revive failed node"""
        self.logger.info("Node reviving")
        self.alive = True

    def shutdown(self):
        """Graceful shutdown"""
        self.logger.info("Shutting down")
        self.running = False
        self.socket.close()

def main():
    parser = argparse.ArgumentParser(description='Run a robust computing node')
    parser.add_argument('--id', type=str, required=True, help='Node ID')
    parser.add_argument('--port', type=int, required=True, help='Port number')
    parser.add_argument('--grid-x', type=int, required=True, help='Grid X position')
    parser.add_argument('--grid-y', type=int, required=True, help='Grid Y position')
    parser.add_argument('--grid-size', type=int, default=5, help='Grid size (default: 5)')
    parser.add_argument('--base-port', type=int, default=9000, help='Base port for neighbor discovery')

    args = parser.parse_args()

    # Create and start node
    node = RobustNode(args.id, args.port, args.grid_x, args.grid_y, args.grid_size)
    node.discover_neighbors(args.base_port)
    node.start()

    try:
        # Keep the node running
        while True:
            time.sleep(1)

            # Print status every 10 seconds if we have information
            if node.generation % 10 == 0 and len(node.information) > 0:
                status = node.get_status()
                print(f"Status: alive={status['alive']}, info_count={status['information_count']}")

    except KeyboardInterrupt:
        print(f"\nShutting down node {args.id}")
        node.shutdown()

if __name__ == "__main__":
    main()
