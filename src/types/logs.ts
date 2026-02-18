export type LogLevel = 'error' | 'warn' | 'info' | 'debug';

export interface LogLabel {
  key: string;
  value: string;
}

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  labels: Record<string, string>;
}

export interface LogQuery {
  labels: Record<string, string>;
  start?: string; // ISO timestamp
  end?: string;   // ISO timestamp
  limit?: number;
}

export interface IngestPayload {
  streams: Array<{
    labels: Record<string, string>;
    entries: Array<{
      ts: string;
      line: string;
    }>;
  }>;
}

export interface ChunkInfo {
  id: string;
  labels: Record<string, string>;
  startTime: string;
  endTime: string;
  size: number;
  entryCount: number;
}

export interface BackendHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  ingestionRate: number;
  storageUsed: number;
  chunksCount: number;
  uptime: number;
}

export interface BackendConfig {
  apiUrl: string;
  apiKey?: string;
}

export interface QueryResult {
  logs: LogEntry[];
  stats: {
    queriedChunks: number;
    scannedLines: number;
    executionTime: number;
  };
  total: number;      // Total matching logs
  hasMore: boolean;   // Whether more logs exist
  offset: number;     // Current offset
}

// Prometheus-compatible metrics format
export interface PrometheusMetrics {
  logpulse_ingested_bytes_total: number;
  logpulse_ingested_lines_total: number;
  logpulse_chunks_stored_total: number;
  logpulse_query_duration_seconds: number;
}
