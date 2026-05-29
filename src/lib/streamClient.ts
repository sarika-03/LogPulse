import { BackendConfig } from '@/types/logs';

export interface StreamMessage {
  type: 'connected' | 'log' | 'filter_updated' | 'error' | 'reconnecting' | 'reconnected' | 'disconnected';
  data?: {
    id: string;
    timestamp: string;
    message: string;
    labels: Record<string, string>;
    level: string;
  };
  message?: string;
  filter?: Record<string, string>;
  attempt?: number;
}

export class LogStreamClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 2000;
  private listeners: Map<string, Set<(msg: StreamMessage) => void>> = new Map();
  private config: BackendConfig | null = null;
  private filter: Record<string, string> = {};
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.listeners.set('log', new Set());
    this.listeners.set('connected', new Set());
    this.listeners.set('disconnected', new Set());
    this.listeners.set('error', new Set());
    this.listeners.set('reconnecting', new Set());
    this.listeners.set('reconnected', new Set());
  }

  connect(config: BackendConfig, filter: Record<string, string> = {}): void {
    this.config = config;
    this.filter = filter;
    this.reconnectAttempts = 0;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.doConnect();
  }

  private doConnect(): void {
    if (!this.config) return;

    // Convert HTTP URL to WebSocket URL
    const wsUrl = this.config.apiUrl
      .replace('http://', 'ws://')
      .replace('https://', 'wss://')
      .replace(/\/$/, '');

    // Build query string for initial filter
    const params = new URLSearchParams();
    Object.entries(this.filter).forEach(([key, value]) => {
      params.set(key, value);
    });

    const queryString = params.toString();
    const fullUrl = `${wsUrl}/stream${queryString ? `?${queryString}` : ''}`;

    console.log('[LogStream] Connecting to:', fullUrl);

    try {
      this.ws = new WebSocket(fullUrl);

      this.ws.onopen = () => {
        console.log('[LogStream] Connected');
        if (this.reconnectAttempts > 0) {
          this.emit('reconnected', { type: 'reconnected', message: 'Connection restored' });
        } else {
          this.emit('connected', { type: 'connected', message: 'Connected to log stream' });
        }
        this.reconnectAttempts = 0;
      };

      this.ws.onmessage = (event) => {
        try {
          const msg: StreamMessage = JSON.parse(event.data);
          console.log('[LogStream] Message:', msg.type);
          
          if (msg.type === 'log' && msg.data) {
            this.emit('log', msg);
          } else if (msg.type === 'connected') {
            this.emit('connected', msg);
          } else if (msg.type === 'filter_updated') {
            console.log('[LogStream] Filter updated:', msg.filter);
          }
        } catch (err) {
          console.error('[LogStream] Parse error:', err);
        }
      };

      this.ws.onclose = (event) => {
        console.log('[LogStream] Disconnected:', event.code, event.reason);
        this.emit('disconnected', { type: 'disconnected', message: 'Disconnected' });
        if (this.config) {
          this.attemptReconnect();
        }
      };

      this.ws.onerror = (error) => {
        console.error('[LogStream] Error:', error);
        this.emit('error', { type: 'error', message: 'WebSocket error' });
      };
    } catch (err) {
      console.error('[LogStream] Connection failed:', err);
      this.attemptReconnect();
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[LogStream] Max reconnect attempts reached');
      this.emit('error', { type: 'error', message: 'Max reconnect attempts reached. Please refresh.' });
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    console.log(`[LogStream] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.emit('reconnecting', { 
      type: 'reconnecting', 
      message: `Reconnecting in ${delay / 1000}s...`,
      attempt: this.reconnectAttempts
    });

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.config) {
        this.doConnect();
      }
    }, delay);
  }

  updateFilter(filter: Record<string, string>): void {
    this.filter = filter;
    
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'filter',
        labels: filter,
      }));
    }
  }

  disconnect(): void {
    this.config = null; // Set to null before closing to prevent reconnection
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  on(event: string, callback: (msg: StreamMessage) => void): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);

    // Return unsubscribe function
    return () => {
      this.listeners.get(event)?.delete(callback);
    };
  }

  private emit(event: string, msg: StreamMessage): void {
    this.listeners.get(event)?.forEach((callback) => callback(msg));
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
export const logStreamClient = new LogStreamClient();
