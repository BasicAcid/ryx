#!/usr/bin/env python3
"""
Network Launcher and Controller for Robust Computing Nodes
Spawns multiple nodes and provides control interface
"""

import subprocess
import time
import socket
import json
import threading
import sys
import random
from dataclasses import dataclass, asdict
from typing import List, Dict, Optional

@dataclass
class Message:
    type: str
    sender_id: str
    data: dict
    energy: int = 0
    hops: int = 0

class NetworkController:
    def __init__(self, grid_size: int = 5, base_port: int = 9000):
        self.grid_size = grid_size
        self.base_port = base_port
        self.processes: List[subprocess.Popen] = []
        self.nodes_info: Dict[str, dict] = {}

        # Create node grid
        for y in range(grid_size):
            for x in range(grid_size):
                node_id = f"node_{x}_{y}"
                port = base_port + (y * grid_size + x)
                self.nodes_info[node_id] = {
                    "port": port,
                    "grid_x": x,
                    "grid_y": y,
                    "process": None
                }

    def start_network(self):
        """Start all nodes in the network"""
        print(f"Starting {self.grid_size}x{self.grid_size} network...")

        for node_id, info in self.nodes_info.items():
            cmd = [
                sys.executable, "robust_node.py",
                "--id", node_id,
                "--port", str(info["port"]),
                "--grid-x", str(info["grid_x"]),
                "--grid-y", str(info["grid_y"]),
                "--grid-size", str(self.grid_size),
                "--base-port", str(self.base_port)
            ]

            process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.processes.append(process)
            info["process"] = process

            print(f"Started {node_id} on port {info['port']}")
            time.sleep(0.1)  # Stagger startup

        print(f"Network started with {len(self.processes)} nodes")
        time.sleep(2)  # Let nodes discover neighbors

    def send_message_to_node(self, node_id: str, message: Message) -> bool:
        """Send a message to a specific node"""
        if node_id not in self.nodes_info:
            print(f"Node {node_id} not found")
            return False

        node_info = self.nodes_info[node_id]
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            data = json.dumps(asdict(message)).encode()
            sock.sendto(data, ("127.0.0.1", node_info["port"]))
            sock.close()
            return True
        except Exception as e:
            print(f"Failed to send message to {node_id}: {e}")
            return False

    def inject_information(self, node_id: str, info_id: str, content: str, energy: int = 10):
        """Inject information into a specific node"""
        message = Message(
            type="information",
            sender_id="controller",
            data={
                "info_id": info_id,
                "content": content
            },
            energy=energy
        )

        if self.send_message_to_node(node_id, message):
            print(f"Injected '{info_id}' into {node_id} with energy {energy}")
        else:
            print(f"Failed to inject information into {node_id}")

    def kill_node(self, node_id: str):
        """Simulate node failure"""
        message = Message(
            type="die",
            sender_id="controller",
            data={}
        )

        if self.send_message_to_node(node_id, message):
            print(f"Sent kill signal to {node_id}")
        else:
            print(f"Failed to kill {node_id}")

    def kill_random_nodes(self, percentage: float = 0.2):
        """Kill random percentage of nodes"""
        nodes_to_kill = random.sample(
            list(self.nodes_info.keys()),
            int(len(self.nodes_info) * percentage)
        )

        print(f"Killing {len(nodes_to_kill)} random nodes...")
        for node_id in nodes_to_kill:
            self.kill_node(node_id)
            time.sleep(0.1)

    def seed_information_randomly(self, count: int = 3):
        """Seed information at random nodes"""
        node_ids = list(self.nodes_info.keys())

        for i in range(count):
            node_id = random.choice(node_ids)
            info_id = f"info_{i}_{int(time.time())}"
            content = f"Information package {i} seeded at {time.strftime('%H:%M:%S')}"

            self.inject_information(node_id, info_id, content, energy=15)
            time.sleep(0.5)

    def get_network_status(self):
        """Get status from all nodes (simplified)"""
        print(f"\n=== Network Status (Grid: {self.grid_size}x{self.grid_size}) ===")
        print(f"Total nodes: {len(self.nodes_info)}")
        print(f"Running processes: {len([p for p in self.processes if p.poll() is None])}")

        # Simple grid visualization
        print("\nGrid Layout (. = node, X = failed):")
        for y in range(self.grid_size):
            row = ""
            for x in range(self.grid_size):
                node_id = f"node_{x}_{y}"
                process = self.nodes_info[node_id]["process"]
                if process and process.poll() is None:
                    row += ". "
                else:
                    row += "X "
            print(f"  {row}")

    def shutdown_network(self):
        """Shutdown all nodes"""
        print("Shutting down network...")

        for process in self.processes:
            if process.poll() is None:
                process.terminate()

        # Wait for graceful shutdown
        time.sleep(2)

        # Force kill if needed
        for process in self.processes:
            if process.poll() is None:
                process.kill()

        print("Network shutdown complete")

    def interactive_mode(self):
        """Interactive command interface"""
        print("\n=== Robust Computing Network Controller ===")
        print("Commands:")
        print("  status      - Show network status")
        print("  seed [n]    - Seed information at n random nodes (default: 3)")
        print("  inject <node> <info_id> <content> - Inject specific information")
        print("  kill <node> - Kill specific node")
        print("  killrand [p] - Kill random percentage of nodes (default: 0.2)")
        print("  restart     - Restart the network")
        print("  quit        - Shutdown and exit")
        print()

        while True:
            try:
                command = input("robust> ").strip().split()
                if not command:
                    continue

                cmd = command[0].lower()

                if cmd == "quit":
                    break
                elif cmd == "status":
                    self.get_network_status()
                elif cmd == "seed":
                    count = int(command[1]) if len(command) > 1 else 3
                    self.seed_information_randomly(count)
                elif cmd == "inject" and len(command) >= 4:
                    node_id, info_id = command[1], command[2]
                    content = " ".join(command[3:])
                    self.inject_information(node_id, info_id, content)
                elif cmd == "kill" and len(command) > 1:
                    self.kill_node(command[1])
                elif cmd == "killrand":
                    percentage = float(command[1]) if len(command) > 1 else 0.2
                    self.kill_random_nodes(percentage)
                elif cmd == "restart":
                    self.shutdown_network()
                    time.sleep(1)
                    self.start_network()
                else:
                    print("Unknown command or invalid arguments")

            except KeyboardInterrupt:
                break
            except Exception as e:
                print(f"Error: {e}")

def main():
    import argparse

    parser = argparse.ArgumentParser(description='Launch robust computing network')
    parser.add_argument('--grid-size', type=int, default=5, help='Grid size (default: 5)')
    parser.add_argument('--base-port', type=int, default=9000, help='Base port (default: 9000)')
    parser.add_argument('--no-interactive', action='store_true', help='Run without interactive mode')

    args = parser.parse_args()

    controller = NetworkController(args.grid_size, args.base_port)

    try:
        controller.start_network()

        if not args.no_interactive:
            controller.interactive_mode()
        else:
            # Run some automated tests
            print("Running automated demonstration...")
            time.sleep(3)

            controller.seed_information_randomly(2)
            time.sleep(5)

            controller.get_network_status()
            time.sleep(3)

            controller.kill_random_nodes(0.3)
            time.sleep(5)

            controller.get_network_status()

            print("Demo complete. Ctrl+C to exit.")
            while True:
                time.sleep(1)

    except KeyboardInterrupt:
        print("\nReceived interrupt signal")
    finally:
        controller.shutdown_network()

if __name__ == "__main__":
    main()
