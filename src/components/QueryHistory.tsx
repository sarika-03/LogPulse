import { Clock, Trash2 } from 'lucide-react';

interface QueryHistoryProps {
  history: string[];
  clearHistory: () => void;
  onRunQuery: (query: string) => void;
}

export function QueryHistory({ history, clearHistory, onRunQuery }: QueryHistoryProps) {
  if (history.length === 0) return null;

  return (
    <div className="flex flex-col space-y-2 px-4 pb-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-slate-400 flex items-center gap-2">
          <Clock className="w-4 h-4" />
          Recent Queries
        </h3>
        <button
          onClick={clearHistory}
          className="text-xs text-slate-500 hover:text-red-400 transition-colors flex items-center gap-1"
          title="Clear History"
        >
          <Trash2 className="w-3 h-3" />
          Clear
        </button>
      </div>
      
      <div className="flex flex-col gap-1 max-h-[200px] overflow-y-auto pr-1">
        {history.map((query, index) => (
          <div
            key={index}
            onClick={() => onRunQuery(query)}
            className="text-sm px-3 py-2 bg-slate-800/50 hover:bg-slate-700/50 text-slate-300 rounded cursor-pointer truncate transition-colors border border-slate-700/50"
            title={query}
          >
            {query}
          </div>
        ))}
      </div>
    </div>
  );
}