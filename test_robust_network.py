#!/usr/bin/env python3
"""
Test script to verify the robust computing network works correctly
"""

import subprocess
import time
import socket
import json
import sys
import os
import signal
from dataclasses import dataclass, asdict

@dataclass
class Message:
    type: str
    sender_id: str
    data: dict
    energy: int = 0
    hops: int = 0

def send_message_to_node(port: int, message: Message) -> bool:
    """Send a message to a node"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        data = json.dumps(asdict(message)).encode()
        sock.sendto(data, ("127.0.0.1", port))
        sock.close()
        return True
    except Exception as e:
        print(f"Failed to send message to port {port}: {e}")
        return False

def ping_node(port: int) -> dict:
    """Ping a node and wait for response"""
    try:
        # Create socket for receiving response
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.bind(("127.0.0.1", 0))  # Let OS choose port
        sock.settimeout(2.0)

        monitor_port = sock.getsockname()[1]

        # Send ping
        ping_msg = Message(
            type="ping",
            sender_id="tester",
            data={"response_port": monitor_port}
        )

        ping_data = json.dumps(asdict(ping_msg)).encode()
        sock.sendto(ping_data, ("127.0.0.1", port))

        # Wait for pong
        response_data, addr = sock.recvfrom(1024)
        response = json.loads(response_data.decode())

        sock.close()
        return response.get("data", {})

    except Exception as e:
        if sock:
            sock.close()
        return {"error": str(e), "alive": False}

def test_basic_functionality():
    """Test basic node functionality"""
    print("=== Testing Basic Node Functionality ===")

    # Start a small 3x3 network
    grid_size = 3
    base_port = 9100
    processes = []

    print(f"Starting {grid_size}x{grid_size} test network...")

    # Start nodes
    for y in range(grid_size):
        for x in range(grid_size):
            node_id = f"node_{x}_{y}"
            port = base_port + (y * grid_size + x)

            cmd = [
                sys.executable, "robust_node.py",
                "--id", node_id,
                "--port", str(port),
                "--grid-x", str(x),
                "--grid-y", str(y),
                "--grid-size", str(grid_size),
                "--base-port", str(base_port)
            ]

            process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            processes.append((process, node_id, port))
            time.sleep(0.1)

    print(f"Started {len(processes)} nodes")
    time.sleep(3)  # Let nodes discover neighbors

    try:
        # Test 1: Check if nodes are alive
        print("\nTest 1: Checking node connectivity...")
        alive_count = 0
        for process, node_id, port in processes:
            if process.poll() is None:  # Process is running
                response = ping_node(port)
                if response.get("alive", False):
                    alive_count += 1
                    print(f"  ✓ {node_id} is alive")
                else:
                    print(f"  ✗ {node_id} not responding")
            else:
                print(f"  ✗ {node_id} process died")

        print(f"Result: {alive_count}/{len(processes)} nodes alive")

        # Test 2: Inject information and check spreading
        print("\nTest 2: Testing information spreading...")

        # Inject into center node
        center_port = base_port + (1 * grid_size + 1)  # node_1_1
        info_msg = Message(
            type="information",
            sender_id="tester",
            data={
                "info_id": "test_info_1",
                "content": "Test information spreading"
            },
            energy=10
        )

        if send_message_to_node(center_port, info_msg):
            print("  ✓ Injected information into center node")
        else:
            print("  ✗ Failed to inject information")
            return False

        # Wait for spreading
        time.sleep(3)

        # Check how many nodes have the information
        nodes_with_info = 0
        for process, node_id, port in processes:
            if process.poll() is None:
                response = ping_node(port)
                info_count = response.get("information_count", 0)
                if info_count > 0:
                    nodes_with_info += 1
                    print(f"  ✓ {node_id} has {info_count} information packets")

        print(f"Result: {nodes_with_info}/{alive_count} nodes have information")

        # Test 3: Kill some nodes and check resilience
        print("\nTest 3: Testing fault tolerance...")

        # Kill corner nodes
        nodes_to_kill = [
            (0, base_port + (0 * grid_size + 0)),  # node_0_0
            (2, base_port + (0 * grid_size + 2)),  # node_2_0
        ]

        for i, (node_idx, port) in enumerate(nodes_to_kill):
            kill_msg = Message(
                type="die",
                sender_id="tester",
                data={}
            )
            if send_message_to_node(port, kill_msg):
                print(f"  ✓ Sent kill signal to node at port {port}")
            else:
                print(f"  ✗ Failed to kill node at port {port}")

        time.sleep(2)

        # Check remaining nodes
        remaining_alive = 0
        for process, node_id, port in processes:
            if process.poll() is None:
                response = ping_node(port)
                if response.get("alive", False):
                    remaining_alive += 1

        print(f"Result: {remaining_alive} nodes still alive after failures")

        # Test 4: Check if information still spreads
        print("\nTest 4: Testing information spreading after failures...")

        # Inject new information
        info_msg2 = Message(
            type="information",
            sender_id="tester",
            data={
                "info_id": "test_info_2",
                "content": "Post-failure information"
            },
            energy=8
        )

        # Inject into a different node
        alt_port = base_port + (2 * grid_size + 1)  # node_1_2
        if send_message_to_node(alt_port, info_msg2):
            print("  ✓ Injected post-failure information")
        else:
            print("  ✗ Failed to inject post-failure information")

        time.sleep(3)

        # Check spreading
        final_info_count = 0
        for process, node_id, port in processes:
            if process.poll() is None:
                response = ping_node(port)
                if response.get("alive", False) and response.get("information_count", 0) > 0:
                    final_info_count += 1

        print(f"Result: {final_info_count} nodes have information after recovery")

        # Summary
        print(f"\n=== Test Summary ===")
        print(f"Initial nodes: {len(processes)}")
        print(f"Nodes responding: {alive_count}")
        print(f"Information spread initially: {nodes_with_info}/{alive_count} nodes")
        print(f"Nodes surviving failures: {remaining_alive}")
        print(f"Information spread after failures: {final_info_count}")

        success = (alive_count >= len(processes) * 0.8 and
                  nodes_with_info >= 2 and
                  final_info_count >= 1)

        if success:
            print("✓ All tests PASSED - Network is working correctly!")
        else:
            print("✗ Some tests FAILED - Check the implementation")

        return success

    finally:
        # Cleanup
        print("\nCleaning up test processes...")
        for process, node_id, port in processes:
            if process.poll() is None:
                process.terminate()

        time.sleep(1)

        for process, node_id, port in processes:
            if process.poll() is None:
                process.kill()

def main():
    if len(sys.argv) > 1 and sys.argv[1] == "quick":
        # Quick test mode
        success = test_basic_functionality()
        sys.exit(0 if success else 1)
    else:
        print("Robust Computing Network Test")
        print("=============================")
        print()
        print("This script will test the basic functionality of the robust computing network.")
        print("It will start a small 3x3 network, test information spreading, and fault tolerance.")
        print()

        input("Press Enter to start the test...")

        success = test_basic_functionality()

        if success:
            print("\n🎉 Test completed successfully!")
            print("\nYou can now run the full network:")
            print("  python network_launcher.py --grid-size 5")
            print("  python network_monitor.py --grid-size 5  # (in another terminal)")
        else:
            print("\n❌ Test failed. Check the error messages above.")

if __name__ == "__main__":
    main()
