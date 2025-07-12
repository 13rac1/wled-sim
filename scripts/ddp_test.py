#!/usr/bin/env python3
"""
DDP Test Script for WLED Simulator

This script sends DDP (Distributed Display Protocol) packets to test
the WLED simulator's DDP functionality. It can set LEDs to various colors
and patterns.

Usage:
    python3 ddp_test.py [options]

Examples:
    python3 ddp_test.py --color blue
    python3 ddp_test.py --color red --host 192.168.1.100
    python3 ddp_test.py --pattern rainbow --leds 30
    python3 ddp_test.py --pattern cycle --delay 0.5
"""

import socket
import struct
import time
import argparse
import sys
from typing import List, Tuple


class DDPClient:
    """DDP (Distributed Display Protocol) client for sending LED data."""
    
    def __init__(self, host: str = "localhost", port: int = 4048):
        self.host = host
        self.port = port
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    
    def send_rgb_data(self, rgb_data: List[Tuple[int, int, int]], push: bool = True):
        """
        Send RGB data via DDP protocol.
        
        Args:
            rgb_data: List of (R, G, B) tuples for each LED
            push: Whether to set the push flag (display immediately)
        """
        num_leds = len(rgb_data)
        data_len = num_leds * 3
        
        # DDP header: 10 bytes
        # flags1: 0x41 (version 1, push flag) or 0x40 (version 1, no push)
        # flags2: 0x00 (no sequence number)
        # type: 0x01 (RGB data)
        # id: 0x01 (default output device)
        # offset: 0x00000000 (start at LED 0)
        # length: data_len (number of RGB bytes)
        
        flags1 = 0x41 if push else 0x40
        header = struct.pack('>BBBB L H', flags1, 0x00, 0x01, 0x01, 0, data_len)
        
        # Convert RGB tuples to bytes
        rgb_bytes = bytes([component for rgb in rgb_data for component in rgb])
        
        # Send UDP packet
        try:
            self.sock.sendto(header + rgb_bytes, (self.host, self.port))
            print(f"Sent DDP packet to {self.host}:{self.port} - {num_leds} LEDs")
        except Exception as e:
            print(f"Error sending DDP packet: {e}")
    
    def close(self):
        """Close the socket."""
        self.sock.close()


def create_solid_color(num_leds: int, color: Tuple[int, int, int]) -> List[Tuple[int, int, int]]:
    """Create a solid color pattern."""
    return [color] * num_leds


def create_rainbow(num_leds: int) -> List[Tuple[int, int, int]]:
    """Create a rainbow pattern."""
    colors = []
    for i in range(num_leds):
        # Create rainbow using HSV -> RGB conversion
        hue = (i * 360) // num_leds
        r, g, b = hsv_to_rgb(hue, 100, 100)
        colors.append((r, g, b))
    return colors


def create_gradient(num_leds: int, start_color: Tuple[int, int, int], end_color: Tuple[int, int, int]) -> List[Tuple[int, int, int]]:
    """Create a gradient between two colors."""
    colors = []
    for i in range(num_leds):
        ratio = i / (num_leds - 1) if num_leds > 1 else 0
        r = int(start_color[0] + (end_color[0] - start_color[0]) * ratio)
        g = int(start_color[1] + (end_color[1] - start_color[1]) * ratio)
        b = int(start_color[2] + (end_color[2] - start_color[2]) * ratio)
        colors.append((r, g, b))
    return colors


def hsv_to_rgb(h: float, s: float, v: float) -> Tuple[int, int, int]:
    """Convert HSV to RGB. H: 0-360, S: 0-100, V: 0-100"""
    h = h % 360
    s = s / 100.0
    v = v / 100.0
    
    c = v * s
    x = c * (1 - abs((h / 60) % 2 - 1))
    m = v - c
    
    if 0 <= h < 60:
        r, g, b = c, x, 0
    elif 60 <= h < 120:
        r, g, b = x, c, 0
    elif 120 <= h < 180:
        r, g, b = 0, c, x
    elif 180 <= h < 240:
        r, g, b = 0, x, c
    elif 240 <= h < 300:
        r, g, b = x, 0, c
    else:
        r, g, b = c, 0, x
    
    return (int((r + m) * 255), int((g + m) * 255), int((b + m) * 255))


def main():
    parser = argparse.ArgumentParser(description='DDP Test Script for WLED Simulator')
    parser.add_argument('--host', default='localhost', help='Target hostname/IP (default: localhost)')
    parser.add_argument('--port', type=int, default=4048, help='Target port (default: 4048)')
    parser.add_argument('--leds', type=int, default=20, help='Number of LEDs (default: 20)')
    parser.add_argument('--color', choices=['red', 'green', 'blue', 'white', 'yellow', 'cyan', 'magenta', 'orange'], 
                       help='Solid color to display')
    parser.add_argument('--pattern', choices=['rainbow', 'cycle', 'gradient', 'chase'], 
                       help='Pattern to display')
    parser.add_argument('--delay', type=float, default=1.0, help='Delay between updates for patterns (default: 1.0)')
    parser.add_argument('--iterations', type=int, default=10, help='Number of iterations for patterns (default: 10)')
    
    args = parser.parse_args()
    
    # Color definitions
    colors = {
        'red': (255, 0, 0),
        'green': (0, 255, 0),
        'blue': (0, 0, 255),
        'white': (255, 255, 255),
        'yellow': (255, 255, 0),
        'cyan': (0, 255, 255),
        'magenta': (255, 0, 255),
        'orange': (255, 165, 0)
    }
    
    client = DDPClient(args.host, args.port)
    
    try:
        if args.color:
            # Send solid color
            print(f"Setting {args.leds} LEDs to {args.color}")
            rgb_data = create_solid_color(args.leds, colors[args.color])
            client.send_rgb_data(rgb_data)
            
        elif args.pattern == 'rainbow':
            # Send rainbow pattern
            print(f"Displaying rainbow pattern on {args.leds} LEDs")
            rgb_data = create_rainbow(args.leds)
            client.send_rgb_data(rgb_data)
            
        elif args.pattern == 'gradient':
            # Send gradient pattern
            print(f"Displaying gradient pattern on {args.leds} LEDs")
            rgb_data = create_gradient(args.leds, (255, 0, 0), (0, 0, 255))  # Red to blue
            client.send_rgb_data(rgb_data)
            
        elif args.pattern == 'chase':
            # Chase pattern - light LEDs sequentially
            print(f"Running chase pattern on {args.leds} LEDs ({args.iterations} iterations)")
            for iteration in range(args.iterations):
                print(f"  Iteration {iteration + 1}/{args.iterations}")
                for led_index in range(args.leds):
                    # Create array with all LEDs off (black)
                    rgb_data = [(0, 0, 0)] * args.leds
                    # Light current LED in green
                    rgb_data[led_index] = (0, 255, 0)
                    
                    print(f"    Lighting LED {led_index}")
                    client.send_rgb_data(rgb_data)
                    time.sleep(args.delay)
                
                # Turn all LEDs off at the end of each iteration
                rgb_data = [(0, 0, 0)] * args.leds
                client.send_rgb_data(rgb_data)
                time.sleep(args.delay)
            
        elif args.pattern == 'cycle':
            # Cycle through colors
            print(f"Cycling through colors on {args.leds} LEDs ({args.iterations} iterations)")
            color_list = list(colors.keys())
            for i in range(args.iterations):
                for color_name in color_list:
                    print(f"  Setting to {color_name}")
                    rgb_data = create_solid_color(args.leds, colors[color_name])
                    client.send_rgb_data(rgb_data)
                    time.sleep(args.delay)
        else:
            # Default: set to blue
            print(f"Setting {args.leds} LEDs to blue (default)")
            rgb_data = create_solid_color(args.leds, colors['blue'])
            client.send_rgb_data(rgb_data)
            
    except KeyboardInterrupt:
        print("\nStopped by user")
    finally:
        client.close()


if __name__ == '__main__':
    main() 