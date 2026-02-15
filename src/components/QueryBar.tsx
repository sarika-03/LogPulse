import { Search, Play, Clock, RefreshCw, Loader2 } from 'lucide-react';
import { useState } from 'react';

// Enhanced with query validation and error feedback (Issue #16)

const timeRanges = [
  { label: 'Last 5 minutes', value: '5m', duration: 5 },
  { label: 'Last 15 minutes', value: '15m', duration: 15 },
  { label: 'Last 1 hour', value: '1h', duration: 60 },
  { label: 'Last 6 hours', value: '6h', duration: 360 },
];

interface QueryBarProps {
  onQuery: (query: string, timeRange: string) => void;
  onRefresh: () => void;
  isLoading: boolean;
  isConnected: boolean;
}

export function QueryBar({ onQuery, onRefresh, isLoading, isConnected }: QueryBarProps) {
  const [query, setQuery] = useState('{service="api-gateway"}');
  const [selectedRange, setSelectedRange] = useState('1h');
    const [queryError, setQueryError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {

      // Simple query validation
  const validateQuery = (q: string): string | null => {
    if (!q || q.trim() === '') {
      return 'Query cannot be empty';
    }
    if (!q.includes('{') || !q.includes('}')) {
      return 'Invalid LogQL syntax: Query must contain { } braces';
    }
    return null;
  };
    e.preventDefault();
        
    // Validate query before submission
    const error = validateQuery(query);
    if (error) {
      setQueryError(error);
      return;
    }
    setQueryError(null);
    if (isConnected) {
      onQuery(query, selectedRange);
    }
  };

  return (
    <div className="glass-panel border-b border-border px-6 py-4">
      <form onSubmit={handleSubmit} className="flex items-center gap-4">
        <div className="flex-1 relative">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder='{service="api-gateway", env="prod"}'
            className="query-input w-full pl-12 pr-4 text-foreground bg-muted border-border hover:border-primary/50 focus:border-primary"
            disabled={!isConnected}
            autoComplete="off"
          />
        </div>
                {queryError && (
          <div className="text-sm text-red-500 mt-2 flex items-center gap-2">
            <span>⚠</span>
            <span>{queryError}</span>
          </div>
        )}

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 bg-muted rounded-lg p-1">
            {timeRanges.map((range) => (
              <button
                key={range.value}
                type="button"
                onClick={() => setSelectedRange(range.value)}
                className={`px-3 py-1.5 text-sm font-mono rounded transition-all ${
                  selectedRange === range.value
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {range.value}
              </button>
            ))}
          </div>

          <button
            type="button"
            onClick={onRefresh}
            disabled={!isConnected || isLoading}
            className="p-2 rounded-lg bg-muted text-muted-foreground hover:text-foreground border border-border transition-all hover:border-primary/50 disabled:opacity-50"
          >
            <RefreshCw className={`h-5 w-5 ${isLoading ? 'animate-spin' : ''}`} />
          </button>

          <button
            type="submit"
            disabled={!isConnected || isLoading}
            className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground font-semibold transition-all hover:opacity-90 glow-primary disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Play className="h-4 w-4" />
            )}
            Run Query
          </button>
        </div>
      </form>

      <div className="flex items-center gap-2 mt-3 text-xs">
        <Clock className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">Query syntax:</span>
        <code className="font-mono text-primary bg-primary/10 px-2 py-0.5 rounded">
          {'{'}label="value", label2="value2"{'}'}
        </code>
        <span className="text-muted-foreground">|</span>
        <span className="text-muted-foreground">Labels from your Go backend</span>
      </div>
    </div>
  );
}
