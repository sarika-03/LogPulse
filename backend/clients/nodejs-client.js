#!/usr/bin/env node

/**
 * LogPulse WebSocket Client
 * 
 * A simple Node.js client to receive and display streaming logs from LogPulse.
 * 
 * Usage:
 *   node client.js [--host localhost] [--port 8080] [--service api] [--level error]
 *   
 * Examples:
 *   node client.js                                    # All logs
 *   node client.js --service api-gateway              # Filter by service
 *   node client.js --service api --level error        # Multiple filters
 */

const WebSocket = require('ws');
const { URL } = require('url');

class LogPulseClient {
    constructor(host = 'localhost', port = 8080, filters = {}) {
        this.host = host;
        this.port = port;
        this.filters = filters;
        this.ws = null;
        this.messageCount = 0;
        this.startTime = Date.now();
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 2000;
    }

    /**
     * Connect to the WebSocket stream
     */
    connect() {
        const url = this.buildUrl();
        console.log(`[${new Date().toISOString()}] Connecting to ${url}`);

        try {
            this.ws = new WebSocket(url);

            this.ws.on('open', () => this.onOpen());
            this.ws.on('message', (data) => this.onMessage(data));
            this.ws.on('error', (err) => this.onError(err));
            this.ws.on('close', () => this.onClose());
        } catch (err) {
            console.error(`[ERROR] Connection failed: ${err.message}`);
            this.scheduleReconnect();
        }
    }

    /**
     * Build WebSocket URL with query parameters
     */
    buildUrl() {
        const url = new URL(`ws://${this.host}:${this.port}/stream`);
        
        // Add filter parameters
        for (const [key, value] of Object.entries(this.filters)) {
            url.searchParams.append(key, value);
        }
        
        return url.toString();
    }

    /**
     * Handle WebSocket open event
     */
    onOpen() {
        console.log('[✓] Connected to LogPulse stream');
        console.log('[Info] Waiting for logs...');
        
        // Reset reconnect attempts on successful connection
        this.reconnectAttempts = 0;
        
        // Send initial filter if provided
        if (Object.keys(this.filters).length > 0) {
            const filterMsg = {
                type: 'filter',
                labels: this.filters
            };
            this.ws.send(JSON.stringify(filterMsg));
        }
    }

    /**
     * Handle incoming WebSocket messages
     */
    onMessage(data) {
        try {
            const msg = JSON.parse(data);

            switch (msg.type) {
                case 'connected':
                    console.log(`[Info] Server message: ${msg.message}`);
                    if (Object.keys(msg.filter || {}).length > 0) {
                        console.log(`[Info] Active filter: ${JSON.stringify(msg.filter)}`);
                    }
                    break;

                case 'log':
                    this.onLogMessage(msg.data);
                    break;

                case 'filter_updated':
                    console.log(`[Info] Filter updated: ${JSON.stringify(msg.filter)}`);
                    break;

                default:
                    console.log(`[?] Unknown message type: ${msg.type}`);
            }
        } catch (err) {
            console.error(`[ERROR] Failed to parse message: ${err.message}`);
        }
    }

    /**
     * Handle log message
     */
    onLogMessage(logData) {
        this.messageCount++;
        
        const timestamp = logData.timestamp || new Date().toISOString();
        const level = logData.level || 'INFO';
        const service = logData.labels?.service || 'unknown';
        const message = logData.message || '';
        
        // Format and display the log
        const levelColor = this.getLevelColor(level);
        const logId = logData.id || '???';
        
        console.log(
            `[${timestamp}] ${levelColor}${level.padEnd(5)}${'\x1b[0m'} ` +
            `[${service.padEnd(15)}] ${message.substring(0, 100)}`
        );

        // Log metadata every 50 messages
        if (this.messageCount % 50 === 0) {
            const elapsed = (Date.now() - this.startTime) / 1000;
            const rate = (this.messageCount / elapsed).toFixed(2);
            console.log(`[Stats] Received ${this.messageCount} logs at ${rate} msgs/sec`);
        }
    }

    /**
     * Get ANSI color code for log level
     */
    getLevelColor(level) {
        const colors = {
            'error': '\x1b[31m',    // Red
            'warn': '\x1b[33m',     // Yellow
            'warning': '\x1b[33m',  // Yellow
            'info': '\x1b[32m',     // Green
            'debug': '\x1b[36m',    // Cyan
            'trace': '\x1b[90m'     // Gray
        };
        return colors[level.toLowerCase()] || '\x1b[0m';
    }

    /**
     * Handle WebSocket error
     */
    onError(err) {
        console.error(`[ERROR] WebSocket error: ${err.message}`);
    }

    /**
     * Handle WebSocket close
     */
    onClose() {
        console.log(`[✗] Disconnected from LogPulse stream`);
        console.log(`[Stats] Total logs received: ${this.messageCount}`);
        this.scheduleReconnect();
    }

    /**
     * Schedule reconnection with exponential backoff
     */
    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error(`[ERROR] Max reconnection attempts reached. Giving up.`);
            process.exit(1);
        }

        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        
        console.log(`[Info] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
        
        setTimeout(() => this.connect(), delay);
    }

    /**
     * Send a filter update to the server
     */
    updateFilter(newFilters) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('[ERROR] Not connected to server');
            return;
        }

        const msg = {
            type: 'filter',
            labels: newFilters
        };

        this.ws.send(JSON.stringify(msg));
        this.filters = newFilters;
        console.log(`[Info] Filter update sent: ${JSON.stringify(newFilters)}`);
    }

    /**
     * Gracefully close the connection
     */
    close() {
        if (this.ws) {
            this.ws.close();
        }
    }
}

// Parse command line arguments
function parseArgs() {
    const args = process.argv.slice(2);
    const config = {
        host: 'localhost',
        port: 8080,
        filters: {}
    };

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        
        if (arg === '--host' && i + 1 < args.length) {
            config.host = args[++i];
        } else if (arg === '--port' && i + 1 < args.length) {
            config.port = parseInt(args[++i], 10);
        } else if (arg.startsWith('--')) {
            const key = arg.substring(2);
            if (i + 1 < args.length) {
                config.filters[key] = args[++i];
            }
        }
    }

    return config;
}

// Main
if (require.main === module) {
    const config = parseArgs();
    
    console.log(`
╔════════════════════════════════════════╗
║       LogPulse WebSocket Client        ║
╚════════════════════════════════════════╝
`);

    const client = new LogPulseClient(config.host, config.port, config.filters);
    client.connect();

    // Handle graceful shutdown
    process.on('SIGINT', () => {
        console.log('\n[Info] Shutting down gracefully...');
        client.close();
        setTimeout(() => process.exit(0), 1000);
    });
}

module.exports = LogPulseClient;
