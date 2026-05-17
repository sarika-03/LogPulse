import { useState, useEffect, useRef, useCallback } from 'react';
import { Radio, Pause, Play, Filter, X, Trash2 } from 'lucide-react';
import { LogEntry } from '@/types/logs';
import { logStreamClient, StreamMessage } from '@/lib/streamClient';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

interface LiveStreamProps {
  isConnected: boolean;
}

export function LiveStream({ isConnected }: LiveStreamProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [filter, setFilter] = useState<Record<string, string>>({});
  const [filterInput, setFilterInput] = useState('');
  const [showFilter, setShowFilter] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const maxLogs = 500;

  // Auto-scroll to bottom
  useEffect(() => {
    if (!isPaused && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, isPaused]);

  // Handle incoming logs
  const handleLog = useCallback((msg: StreamMessage) => {
    if (isPaused || !msg.data) return;

    const entry: LogEntry = {
      id: msg.data.id,
      timestamp: msg.data.timestamp,
      level: msg.data.level as any,
      message: msg.data.message,
      labels: msg.data.labels,
    };

    setLogs((prev) => {
      const updated = [...prev, entry];
      // Keep only last maxLogs entries
      return updated.slice(-maxLogs);
    });
  }, [isPaused]);

  // Start streaming
  const startStream = useCallback(() => {
    const config = apiClient.getConfig();
    if (!config) {
      toast.error('Backend not configured');
      return;
    }

    logStreamClient.connect(config, filter);
    setIsStreaming(true);
    toast.success('Live stream started');
  }, [filter]);

  // Stop streaming
  const stopStream = useCallback(() => {
    logStreamClient.disconnect();
    setIsStreaming(false);
    toast.info('Live stream stopped');
  }, []);

  // Subscribe to log events
  useEffect(() => {
    const unsubLog = logStreamClient.on('log', handleLog);
    const unsubConnected = logStreamClient.on('connected', () => {
      console.log('Stream connected');
      setIsStreaming(true);
      setIsReconnecting(false);
    });
    const unsubDisconnected = logStreamClient.on('disconnected', () => {
      setIsStreaming(false);
      setIsReconnecting(false);
    });
    const unsubReconnecting = logStreamClient.on('reconnecting', (msg) => {
      console.log('Stream reconnecting:', msg.message);
      setIsReconnecting(true);
      setIsStreaming(true); // Keep UI showing stream controls but in reconnecting state
    });
    const unsubReconnected = logStreamClient.on('reconnected', () => {
      console.log('Stream reconnected');
      setIsStreaming(true);
      setIsReconnecting(false);
      toast.success('Connection successfully restored');
    });

    return () => {
      unsubLog();
      unsubConnected();
      unsubDisconnected();
      unsubReconnecting();
      unsubReconnected();
    };
  }, [handleLog]);

  // Update filter
  const applyFilter = () => {
    try {
      // Parse filter like {service=\"api\", level=\"error\"}
      const match = filterInput.match(/\{(.*)\}/);
      if (match) {
        const pairs = match[1].split(',');
        const newFilter: Record<string, string> = {};
        pairs.forEach((pair) => {
          const [key, value] = pair.split('=').map((s) => s.trim().replace(/"/g, ''));
          if (key && value) {
            newFilter[key] = value;
          }
        });
        setFilter(newFilter);
        logStreamClient.updateFilter(newFilter);
        toast.success('Filter applied');
      } else if (filterInput.trim() === '') {
        setFilter({});
        logStreamClient.updateFilter({});
        toast.success('Filter cleared');
      }
    } catch {
      toast.error('Invalid filter syntax');
    }
  };

  const clearLogs = () => {
    setLogs([]);
  };

  const levelColors: Record<string, string> = {
    error: 'text-destructive border-l-destructive',
    warn: 'text-warning border-l-warning',
    info: 'text-info border-l-info',
    debug: 'text-muted-foreground border-l-muted-foreground',
  };

  if (!isConnected) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        <p className="text-sm">Connect to backend to use live streaming</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Controls */}
      <div className="p-4 border-b border-border flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          {!isStreaming ? (
            <button
              onClick={startStream}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-success text-success-foreground font-medium hover:opacity-90"
            >
              <Radio className="h-4 w-4" />
              Start Live Stream
            </button>
          ) : (
            <>
              <button
                onClick={stopStream}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-destructive text-destructive-foreground font-medium hover:opacity-90"
              >
                <X className="h-4 w-4" />
                Stop
              </button>
              <button
                onClick={() => setIsPaused(!isPaused)}
                className={`flex items-center gap-2 px-3 py-2 rounded-lg font-medium transition-colors ${
                  isPaused
                    ? 'bg-warning/20 text-warning'
                    : 'bg-muted text-muted-foreground hover:text-foreground'
                }`}
              >
                {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
                {isPaused ? 'Resume' : 'Pause'}
              </button>
            </>
          )}

          <button
            onClick={() => setShowFilter(!showFilter)}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg font-medium transition-colors ${
              showFilter || Object.keys(filter).length > 0
                ? 'bg-primary/20 text-primary'
                : 'bg-muted text-muted-foreground hover:text-foreground'
            }`}
          >
            <Filter className="h-4 w-4" />
            Filter
            {Object.keys(filter).length > 0 && (
              <span className="bg-primary text-primary-foreground text-xs px-1.5 rounded-full">
                {Object.keys(filter).length}
              </span>
            )}
          </button>

          <button
            onClick={clearLogs}
            className="flex items-center gap-2 px-3 py-2 rounded-lg bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <Trash2 className="h-4 w-4" />
            Clear
          </button>
        </div>

        <div className="flex items-center gap-4 text-sm">
          {isStreaming && (
            <div className="flex items-center gap-2">
              <div className={`h-2 w-2 rounded-full ${isReconnecting ? 'bg-warning animate-pulse' : isPaused ? 'bg-warning' : 'bg-success animate-pulse'}`} />
              <span className="text-muted-foreground font-mono">
                {isReconnecting ? 'Reconnecting...' : (isPaused ? 'Paused' : 'Streaming')}
              </span>
            </div>
          )}
          <span className="text-muted-foreground font-mono">{logs.length} logs</span>
        </div>
      </div>

      {/* Filter input */}
      {showFilter && (
        <div className="p-4 border-b border-border bg-muted/30 flex items-center gap-3">
          <input
            type="text"
            value={filterInput}
            onChange={(e) => setFilterInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && applyFilter()}
            placeholder='{service="api-gateway", level="error"}'
            className="query-input flex-1 text-sm py-2"
          />
          <button
            onClick={applyFilter}
            className="px-4 py-2 rounded-lg bg-primary text-primary-foreground font-medium text-sm hover:opacity-90"
          >
            Apply
          </button>
          {Object.keys(filter).length > 0 && (
            <button
              onClick={() => {
                setFilter({});
                setFilterInput('');
                logStreamClient.updateFilter({});
              }}
              className="px-3 py-2 rounded-lg bg-muted text-muted-foreground hover:text-foreground text-sm"
            >
              Clear
            </button>
          )}
        </div>
      )}

      {/* Active filters */}
      {Object.keys(filter).length > 0 && (
        <div className="px-4 py-2 border-b border-border flex items-center gap-2 flex-wrap">
          <span className="text-xs text-muted-foreground">Active filters:</span>
          {Object.entries(filter).map(([key, value]) => (
            <span key={key} className="label-badge">
              <span className="text-muted-foreground">{key}=</span>
              <span className="text-primary">"{value}"</span>
            </span>
          ))}
        </div>
      )}

      {/* Log stream */}
      <div className="flex-1 overflow-auto scrollbar-thin font-mono text-sm">
        {logs.length === 0 ? (
          <div className="h-full flex items-center justify-center text-muted-foreground">
            {isStreaming ? 'Waiting for logs...' : 'Start streaming to see live logs'}
          </div>
        ) : (
          <div className="divide-y divide-border/30">
            {logs.map((log) => (
              <div
                key={log.id}
                className={`px-4 py-1.5 border-l-2 hover:bg-muted/30 animate-log-appear ${
                  levelColors[log.level] || levelColors.info
                }`}
              >
                <span className="text-muted-foreground text-xs mr-3">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className={`uppercase text-xs mr-3 ${levelColors[log.level]?.split(' ')[0]}`}>
                  [{log.level}]
                </span>
                <span className="text-primary/70 mr-3">[{log.labels.service || 'unknown'}]</span>
                <span>{log.message}</span>
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>
    </div>
  );
}
