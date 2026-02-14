import { LogEntry, LogLevel } from '@/types/logs';
import { ChevronRight, Copy, ChevronLeft } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { LogExport } from './LogExport';

interface LogViewerProps {
  logs: LogEntry[];
  isLoading?: boolean;
  isConnected: boolean;
  total?: number;
  hasMore?: boolean;
  currentOffset?: number;
  onPageChange?: (offset: number) => void;
  queryStats?: {
    queriedChunks: number;
    scannedLines: number;
    executionTime: number;
  };
}

export function LogViewer({ logs, isLoading, isConnected, queryStats }: LogViewerProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (!isConnected) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center max-w-md">
          <p className="text-muted-foreground">
            Click ⚙️ Settings to connect to your Go backend
          </p>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="h-12 w-12 border-4 border-primary/30 border-t-primary rounded-full animate-spin" />
          <p className="text-muted-foreground font-mono">Querying backend...</p>
        </div>
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <p className="text-xl text-muted-foreground">No logs found</p>
          <p className="text-sm text-muted-foreground mt-2">
            Try adjusting your query or use the Test Panel to inject test logs
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto scrollbar-thin flex flex-col">
      <div className="px-6 py-2 bg-muted/30 border-b border-border flex items-center justify-between">
        <div className="flex items-center gap-6 text-xs font-mono">
          {queryStats && (
            <>
              <span className="text-muted-foreground">
                Chunks: <span className="text-foreground">{queryStats.queriedChunks}</span>
              </span>
              <span className="text-muted-foreground">
                Lines scanned: <span className="text-foreground">{queryStats.scannedLines}</span>
              </span>
              <span className="text-muted-foreground">
                Execution: <span className="text-primary">{queryStats.executionTime}ms</span>
              </span>
            </>
          )}
          <span className="text-muted-foreground">
            Results: <span className="text-foreground">{logs.length}</span>
            {total !== undefined && <span className="text-muted-foreground"> / {total}</span>}
          </span>
        </div>
        <LogExport logs={logs} />
      </div>
      <div className="divide-y divide-border/50 flex-1">
        {logs.map((log, index) => (
          <LogLine
            key={log.id}
            log={log}
            isExpanded={expandedId === log.id}
            onToggle={() => setExpandedId(expandedId === log.id ? null : log.id)}
            delay={index * 20}
          />
        ))}
      </div>
      
      {/* Pagination Controls */}
      {(total !== undefined || hasMore) && (
        <div className="px-6 py-3 bg-muted/30 border-t border-border flex items-center justify-between">
          <div className="text-xs text-muted-foreground font-mono">
            {currentOffset !== undefined && total !== undefined ? (
              <>
                Page: <span className="text-foreground">{Math.floor(currentOffset / 100) + 1}</span>
                {' '} (Showing{' '}
                <span className="text-foreground">{currentOffset + 1}</span>
                {' '}-{' '}
                <span className="text-foreground">
                  {Math.min(currentOffset + logs.length, total)}
                </span>
                {' '}of{' '}
                <span className="text-foreground">{total}</span>)
              </>
            ) : null}
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => onPageChange?.(Math.max(0, (currentOffset || 0) - 100))}
              disabled={!currentOffset || currentOffset === 0}
              className="p-2 hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed rounded transition-colors"
              title="Previous page"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            
            <button
              onClick={() => onPageChange?.((currentOffset || 0) + 100)}
              disabled={!hasMore}
              className="p-2 hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed rounded transition-colors"
              title="Next page"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

interface LogLineProps {
  log: LogEntry;
  isExpanded: boolean;
  onToggle: () => void;
  delay: number;
}

function LogLine({ log, isExpanded, onToggle, delay }: LogLineProps) {
  const level = (log.labels?.level as LogLevel) || 'info';
  
  const levelStyles: Record<LogLevel, string> = {
    error: 'log-level-error',
    warn: 'log-level-warn',
    info: 'log-level-info',
    debug: 'log-level-debug',
  };

  const levelBadgeStyles: Record<LogLevel, string> = {
    error: 'bg-destructive/20 text-destructive border-destructive/30',
    warn: 'bg-warning/20 text-warning border-warning/30',
    info: 'bg-info/20 text-info border-info/30',
    debug: 'bg-muted text-muted-foreground border-border',
  };

  const copyLog = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(JSON.stringify(log, null, 2));
    toast.success('Log copied to clipboard');
  };

  const service = log.labels?.service || 'unknown';
  const env = log.labels?.env || log.labels?.environment || '';

  return (
    <div
      className={`${levelStyles[level]} cursor-pointer animate-log-appear`}
      style={{ animationDelay: `${delay}ms` }}
      onClick={onToggle}
    >
      <div className="flex items-start gap-3 py-2">
        <ChevronRight
          className={`h-4 w-4 mt-0.5 text-muted-foreground transition-transform flex-shrink-0 ${
            isExpanded ? 'rotate-90' : ''
          }`}
        />
        
        <span className="text-muted-foreground font-mono text-xs flex-shrink-0 w-[180px]">
          {log.timestamp}
        </span>

        <span
          className={`text-xs font-mono uppercase px-2 py-0.5 rounded border flex-shrink-0 w-[60px] text-center ${
            levelBadgeStyles[level]
          }`}
        >
          {level}
        </span>

        <span className="label-badge flex-shrink-0">{service}</span>
        {env && <span className="label-badge flex-shrink-0">{env}</span>}

        <span className="font-mono text-sm truncate flex-1">{log.message}</span>

        <button
          onClick={copyLog}
          className="opacity-0 group-hover:opacity-100 p-1 hover:bg-muted rounded transition-all flex-shrink-0"
        >
          <Copy className="h-3.5 w-3.5 text-muted-foreground" />
        </button>
      </div>

      {isExpanded && (
        <div className="ml-8 pb-3 pt-1 animate-fade-in">
          <div className="bg-muted/50 rounded-lg p-4 font-mono text-sm">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-muted-foreground text-xs mb-2">Labels</p>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(log.labels).map(([key, value]) => (
                    <span key={key} className="label-badge">
                      <span className="text-muted-foreground">{key}=</span>
                      <span className="text-primary">"{value}"</span>
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <p className="text-muted-foreground text-xs mb-2">Metadata</p>
                <div className="space-y-1 text-xs">
                  <p>
                    <span className="text-muted-foreground">ID: </span>
                    <span className="text-foreground">{log.id}</span>
                  </p>
                  <p>
                    <span className="text-muted-foreground">Timestamp: </span>
                    <span className="text-foreground">{log.timestamp}</span>
                  </p>
                </div>
              </div>
            </div>
            
            <div className="mt-4 pt-4 border-t border-border/50">
              <p className="text-muted-foreground text-xs mb-2">Full Message</p>
              <p className="text-foreground break-all">{log.message}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
