#!/usr/bin/env python3
"""
LogPulse WebSocket Client (Python)

A simple Python client to receive and display streaming logs from LogPulse.

Installation:
    pip install websockets

Usage:
    python3 client.py [--host localhost] [--port 8080] [--service api] [--level error]

Examples:
    python3 client.py                                  # All logs
    python3 client.py --service api-gateway            # Filter by service
    python3 client.py --service api --level error      # Multiple filters
    python3 client.py --host 10.0.0.1 --port 8080      # Custom host/port
"""

import asyncio
import json
import sys
import argparse
from datetime import datetime
from typing import Dict, Optional
import signal

try:
    import websockets
    from websockets.client import WebSocketClientProtocol
except ImportError:
    print("ERROR: websockets library not installed")
    print("Install with: pip install websockets")
    sys.exit(1)


class LogPulseClient:
    """WebSocket client for LogPulse log streaming"""

    def __init__(self, host: str = 'localhost', port: int = 8080, filters: Optional[Dict] = None):
        self.host = host
        self.port = port
        self.filters = filters or {}
        self.ws: Optional[WebSocketClientProtocol] = None
        self.message_count = 0
        self.running = True
        self.reconnect_attempts = 0
        self.max_reconnect_attempts = 5
        self.reconnect_delay = 2.0

    def build_url(self) -> str:
        """Build WebSocket URL with query parameters"""
        url = f"ws://{self.host}:{self.port}/stream"
        
        if self.filters:
            params = "&".join(f"{k}={v}" for k, v in self.filters.items())
            url += f"?{params}"
        
        return url

    async def connect(self):
        """Connect to the WebSocket stream"""
        url = self.build_url()
        self._log(f"Connecting to {url}", level="INFO")

        try:
            async with websockets.connect(url, ping_interval=20, ping_timeout=10) as websocket:
                self.ws = websocket
                self.reconnect_attempts = 0
                self._log("Connected to LogPulse stream", level="OK")
                self._log("Waiting for logs...", level="INFO")
                
                # Send initial filter if provided
                if self.filters:
                    filter_msg = {
                        "type": "filter",
                        "labels": self.filters
                    }
                    await websocket.send(json.dumps(filter_msg))
                
                # Listen for messages
                while self.running:
                    try:
                        message = await asyncio.wait_for(websocket.recv(), timeout=60)
                        await self.on_message(message)
                    except asyncio.TimeoutError:
                        # Send ping to keep connection alive
                        continue
                    except websockets.exceptions.ConnectionClosed:
                        break

        except Exception as e:
            self._log(f"Connection error: {e}", level="ERROR")
            await self.schedule_reconnect()

    async def on_message(self, data: str):
        """Handle incoming WebSocket message"""
        try:
            msg = json.loads(data)
            
            if msg.get('type') == 'connected':
                self._log(f"Server: {msg.get('message')}", level="INFO")
                if msg.get('filter'):
                    self._log(f"Active filter: {msg['filter']}", level="INFO")
            
            elif msg.get('type') == 'log':
                await self.on_log_message(msg.get('data', {}))
            
            elif msg.get('type') == 'filter_updated':
                self._log(f"Filter updated: {msg.get('filter')}", level="INFO")
            
            else:
                self._log(f"Unknown message type: {msg.get('type')}", level="WARN")

        except json.JSONDecodeError as e:
            self._log(f"Failed to parse message: {e}", level="ERROR")

    async def on_log_message(self, log_data: Dict):
        """Handle log message"""
        self.message_count += 1
        
        timestamp = log_data.get('timestamp', datetime.now().isoformat())
        level = log_data.get('level', 'INFO').upper()
        service = log_data.get('labels', {}).get('service', 'unknown')
        message = log_data.get('message', '')
        
        # Format output with colors
        color = self._get_level_color(level)
        reset = '\033[0m'
        
        output = (
            f"[{timestamp}] {color}{level:5}{reset} "
            f"[{service:15}] {message[:100]}"
        )
        print(output)
        
        # Log stats every 50 messages
        if self.message_count % 50 == 0:
            self._log(
                f"Received {self.message_count} logs total",
                level="STATS"
            )

    def _get_level_color(self, level: str) -> str:
        """Get ANSI color code for log level"""
        colors = {
            'ERROR': '\033[31m',    # Red
            'WARN': '\033[33m',     # Yellow
            'WARNING': '\033[33m',  # Yellow
            'INFO': '\033[32m',     # Green
            'DEBUG': '\033[36m',    # Cyan
            'TRACE': '\033[90m'     # Gray
        }
        return colors.get(level.upper(), '\033[0m')

    async def schedule_reconnect(self):
        """Schedule reconnection with exponential backoff"""
        if self.reconnect_attempts >= self.max_reconnect_attempts:
            self._log("Max reconnection attempts reached. Giving up.", level="ERROR")
            self.running = False
            return

        self.reconnect_attempts += 1
        delay = self.reconnect_delay * (2 ** (self.reconnect_attempts - 1))
        
        self._log(
            f"Reconnecting in {delay:.1f}s (attempt {self.reconnect_attempts}/{self.max_reconnect_attempts})...",
            level="INFO"
        )
        
        await asyncio.sleep(delay)
        if self.running:
            await self.connect()

    async def update_filter(self, new_filters: Dict):
        """Send a filter update to the server"""
        if not self.ws or self.ws.closed:
            self._log("Not connected to server", level="ERROR")
            return

        msg = {
            "type": "filter",
            "labels": new_filters
        }
        
        await self.ws.send(json.dumps(msg))
        self.filters = new_filters
        self._log(f"Filter update sent: {new_filters}", level="INFO")

    async def close(self):
        """Gracefully close the connection"""
        self.running = False
        if self.ws:
            await self.ws.close()

    def _log(self, message: str, level: str = "INFO"):
        """Log a message with timestamp"""
        timestamp = datetime.now().isoformat(timespec='seconds')
        
        colors = {
            'ERROR': '\033[31m',
            'WARN': '\033[33m',
            'INFO': '\033[36m',
            'OK': '\033[32m',
            'STATS': '\033[35m'
        }
        
        color = colors.get(level, '\033[0m')
        reset = '\033[0m'
        
        print(f"[{timestamp}] {color}[{level:5}]{reset} {message}")


async def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(
        description='LogPulse WebSocket Client',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python3 client.py                          # All logs
  python3 client.py --service api-gateway    # Filter by service
  python3 client.py --level error            # Filter by level
  python3 client.py --service api --env prod # Multiple filters
        """
    )
    
    parser.add_argument('--host', default='localhost', help='Server host (default: localhost)')
    parser.add_argument('--port', type=int, default=8080, help='Server port (default: 8080)')
    
    # Parse known args to allow arbitrary filters
    args, unknown = parser.parse_known_args()
    
    # Build filters from remaining arguments
    filters = {}
    for i, arg in enumerate(unknown):
        if arg.startswith('--'):
            key = arg[2:]
            if i + 1 < len(unknown) and not unknown[i + 1].startswith('--'):
                filters[key] = unknown[i + 1]
    
    # Print header
    print("""
╔════════════════════════════════════════╗
║    LogPulse WebSocket Client (Python)   ║
╚════════════════════════════════════════╝
""")
    
    client = LogPulseClient(args.host, args.port, filters)
    
    # Handle signals
    def handle_signal(sig, frame):
        print("\n[Info] Shutting down gracefully...")
        asyncio.create_task(client.close())
    
    signal.signal(signal.SIGINT, handle_signal)
    
    try:
        await client.connect()
    except KeyboardInterrupt:
        await client.close()
    finally:
        if client.message_count > 0:
            print(f"\n[Stats] Total logs received: {client.message_count}")


if __name__ == "__main__":
    asyncio.run(main())
