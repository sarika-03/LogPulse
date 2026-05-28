import { useState } from 'react';
import { Clock, Trash2, Pin, Download, Search } from 'lucide-react';
import { HistoryEntry } from '@/hooks/useQueryHistory';

interface QueryHistoryProps {
  history: HistoryEntry[];
  clearHistory: () => void;
  togglePin: (query: string) => void;
  exportHistory: () => void;
  onRunQuery: (query: string) => void;
}

function timeAgo(ts: number): string {
  const diff = Math.floor((Date.now() - ts) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export function QueryHistory({
  history,
  clearHistory,
  togglePin,
  exportHistory,
  onRunQuery,
}: QueryHistoryProps) {
  const [filter, setFilter] = useState('');

  if (history.length === 0) return null;

  const filtered = filter.trim()
    ? history.filter((e) =>
        e.query.toLowerCase().includes(filter.toLowerCase())
      )
    : history;

  return (
    <div className="flex flex-col space-y-2 px-4 pb-4">
      {/* header row */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-slate-400 flex items-center gap-2">
          <Clock className="w-4 h-4" />
          Recent Queries
        </h3>
        <div className="flex items-center gap-2">
          <button
            onClick={exportHistory}
            className="text-xs text-slate-500 hover:text-blue-400 transition-colors"
            title="Export history as JSON"
          >
            <Download className="w-3 h-3" />
          </button>
          <button
            onClick={clearHistory}
            className="text-xs text-slate-500 hover:text-red-400 transition-colors flex items-center gap-1"
            title="Clear history"
          >
            <Trash2 className="w-3 h-3" />
            Clear
          </button>
        </div>
      </div>

      {/* filter input */}
      <div className="relative">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-slate-500 pointer-events-none" />
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter queries…"
          className="w-full pl-7 pr-3 py-1.5 text-xs rounded bg-slate-800/60 border border-slate-700/50 text-slate-300 placeholder-slate-600 focus:outline-none focus:border-slate-500"
        />
      </div>

      {/* list */}
      <div className="flex flex-col gap-1 max-h-[240px] overflow-y-auto pr-1">
        {filtered.length === 0 && (
          <p className="text-xs text-slate-600 px-1 py-2">No matches</p>
        )}
        {filtered.map((entry, index) => (
          <div
            key={index}
            className="group relative flex flex-col px-3 py-2 bg-slate-800/50 hover:bg-slate-700/50 text-slate-300 rounded cursor-pointer transition-colors border border-slate-700/50"
            onClick={() => onRunQuery(entry.query)}
            title={entry.query}
          >
            {/* query text */}
            <span className="text-sm truncate pr-6">{entry.query}</span>

            {/* meta row */}
            <div className="flex items-center gap-2 mt-1">
              <span className="text-[11px] text-slate-500">
                {timeAgo(entry.timestamp)}
              </span>
              {entry.runCount > 1 && (
                <span className="text-[11px] bg-slate-700 text-slate-400 rounded px-1.5 py-0.5">
                  ×{entry.runCount}
                </span>
              )}
              {entry.pinned && (
                <span className="text-[11px] text-amber-500">pinned</span>
              )}
            </div>

            {/* pin button — visible on hover */}
            <button
              onClick={(e) => {
                e.stopPropagation();
                togglePin(entry.query);
              }}
              className={`absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity p-0.5 rounded ${
                entry.pinned
                  ? 'text-amber-400'
                  : 'text-slate-600 hover:text-slate-300'
              }`}
              title={entry.pinned ? 'Unpin' : 'Pin to top'}
            >
              <Pin className="w-3 h-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}