#!/usr/bin/env python3
"""
Network Monitor - Real-time visualization of the robust computing network
"""

import socket
import json
import time
import threading
import sys
from dataclasses import dataclass, asdict
from typing import Dict, List
import os

@dataclass
class Message:
    type: str
    sender_id: str
    data: dict
    energy: int = 0
    hops: int = 0

class NetworkMonitor:
    def __init__(self, grid_size: int = 5, base_port: int = 9000):
        self.grid_size = grid_size
        self.base_port = base_port
        self.nodes_status: Dict[str, dict] = {}
        self.running = True

        # Socket for sending pings and receiving pongs
        self.monitor_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.monitor_socket.bind(("127.0.0.1", base_port - 1))  # Use port just below base_port
        self.monitor_socket.settimeout(0.1)

        # Initialize nodes info
        for y in range(grid_size):
            for x in range(grid_size):
                node_id = f"node_{x}_{y}"
                port = base_port + (y * grid_size + x)
                self.nodes_status[node_id] = {
                    "port": port,
                    "grid_x": x,
                    "grid_y": y,
                    "alive": False,
                    "information": {},
                    "information_count": 0,
                    "generation": 0,
                    "last_seen": 0,
                    "response_time": 0
                }

        # Start response listener
        self.listen_thread = threading.Thread(target=self._listen_for_responses, daemon=True)
        self.listen_thread.start()

    def _listen_for_responses(self):
        """Listen for pong responses from nodes"""
        while self.running:
            try:
                data, addr = self.monitor_socket.recvfrom(2048)
                message_dict = json.loads(data.decode())
                message = Message(**message_dict)

                if message.type == "pong":
                    self._handle_pong_response(message, addr)

            except socket.timeout:
                continue
            except Exception as e:
                if self.running:
                    pass  # Ignore errors when shutting down

    def _handle_pong_response(self, message: Message, addr):
        """Handle pong response from a node"""
        node_id = message.sender_id
        if node_id in self.nodes_status:
            node_status = self.nodes_status[node_id]
            data = message.data

            # Update node status from pong response
            node_status["alive"] = data.get("alive", False)
            node_status["information"] = data.get("information", {})
            node_status["information_count"] = data.get("information_count", 0)
            node_status["generation"] = data.get("generation", 0)
            node_status["last_seen"] = time.time()
            node_status["neighbors_count"] = data.get("neighbors", 0)

    def ping_all_nodes(self):
        """Send ping to all nodes"""
        ping_message = Message(
            type="ping",
            sender_id="monitor",
            data={"timestamp": time.time()}
        )

        # Reset alive status before pinging
        for node_status in self.nodes_status.values():
            node_status["alive"] = False

        # Send pings to all nodes
        for node_id, node_info in self.nodes_status.items():
            try:
                data = json.dumps(asdict(ping_message)).encode()
                self.monitor_socket.sendto(data, ("127.0.0.1", node_info["port"]))
            except Exception as e:
                pass  # Node might be dead

    def monitor_loop(self):
        """Main monitoring loop"""
        while self.running:
            # Ping all nodes
            self.ping_all_nodes()

            # Wait a bit for responses
            time.sleep(0.5)

            # Display current state
            self.display_network_state()

            time.sleep(1.5)  # Update every 2 seconds total

    def display_network_state(self):
        """Display the current network state"""
        # Clear screen (works on most terminals)
        os.system('clear' if os.name == 'posix' else 'cls')

        print("=" * 70)
        print(f"ROBUST COMPUTING NETWORK MONITOR - Grid {self.grid_size}x{self.grid_size}")
        print("=" * 70)
        print(f"Time: {time.strftime('%Y-%m-%d %H:%M:%S')}")
        print()

        # Count alive nodes and information
        alive_nodes = [n for n in self.nodes_status.values() if n["alive"]]
        alive_count = len(alive_nodes)
        total_count = len(self.nodes_status)
        total_info = sum(n["information_count"] for n in alive_nodes)

        print(f"Network Status: {alive_count}/{total_count} nodes alive ({alive_count/total_count*100:.1f}%)")
        print(f"Total Information Packets: {total_info}")
        print()

        # Display grid
        print("Network Grid:")
        print("  Legend: ● = alive, ○ = dead, ◆ = alive with info")
        print()

        for y in range(self.grid_size):
            row = f"  {y} "
            for x in range(self.grid_size):
                node_id = f"node_{x}_{y}"
                node = self.nodes_status[node_id]

                if node["alive"]:
                    if node["information_count"] > 0:
                        row += "◆ "  # Alive with information
                    else:
                        row += "● "  # Alive without information
                else:
                    row += "○ "     # Dead or unreachable
            print(row)

        # Column numbers
        print("    " + " ".join(str(i) for i in range(self.grid_size)))
        print()

        # Show detailed information about nodes with data
        nodes_with_info = [(nid, n) for nid, n in self.nodes_status.items()
                          if n["alive"] and n["information_count"] > 0]

        if nodes_with_info:
            print("Nodes with Information:")
            for node_id, node in nodes_with_info[:10]:  # Show first 10
                info_list = list(node["information"].keys())[:3]  # Show first 3 info items
                info_str = ", ".join(info_list)
                if len(node["information"]) > 3:
                    info_str += f" (+{len(node['information'])-3} more)"
                print(f"  {node_id}: {node['information_count']} items [{info_str}]")

            if len(nodes_with_info) > 10:
                print(f"  ... and {len(nodes_with_info)-10} more nodes")
        else:
            print("No nodes currently have information")

        print()

        # Show some statistics
        if alive_nodes:
            avg_generation = sum(n["generation"] for n in alive_nodes) / len(alive_nodes)
            max_info = max(n["information_count"] for n in alive_nodes)
            print(f"Statistics:")
            print(f"  Average Generation: {avg_generation:.1f}")
            print(f"  Max Info per Node: {max_info}")
            print(f"  Information Spread: {len(nodes_with_info)}/{alive_count} nodes have data")

        print()
        print("Commands: Ctrl+C to exit monitor")
        print("Use network_launcher.py in another terminal to control the network")

    def start(self):
        """Start monitoring"""
        try:
            print("Starting network monitor...")
            print("Waiting for nodes to respond...")
            time.sleep(2)  # Give nodes time to start
            self.monitor_loop()
        except KeyboardInterrupt:
            print("\nMonitor stopping...")
            self.running = False
        finally:
            self.monitor_socket.close()

def main():
    import argparse

    parser = argparse.ArgumentParser(description='Monitor robust computing network')
    parser.add_argument('--grid-size', type=int, default=5, help='Grid size (default: 5)')
    parser.add_argument('--base-port', type=int, default=9000, help='Base port (default: 9000)')

    args = parser.parse_args()

    print(f"Starting network monitor for {args.grid_size}x{args.grid_size} grid on port {args.base_port}")
    print("Make sure the network is running first with network_launcher.py")
    print()

    monitor = NetworkMonitor(args.grid_size, args.base_port)
    monitor.start()

if __name__ == "__main__":
    main()
