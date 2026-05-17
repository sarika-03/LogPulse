import { useState, useEffect } from 'react';

const HISTORY_KEY = 'logpulse_query_history';
const MAX_HISTORY = 20;

export interface HistoryEntry {
  query: string;
  timestamp: number;
  runCount: number;
  pinned: boolean;
}

export function useQueryHistory() {
  const [history, setHistory] = useState<HistoryEntry[]>([]);

  useEffect(() => {
    const stored = localStorage.getItem(HISTORY_KEY);
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        // backward-compat: if old format was string[], convert it
        if (Array.isArray(parsed) && typeof parsed[0] === 'string') {
          const migrated: HistoryEntry[] = parsed.map((q: string) => ({
            query: q,
            timestamp: Date.now(),
            runCount: 1,
            pinned: false,
          }));
          setHistory(migrated);
          localStorage.setItem(HISTORY_KEY, JSON.stringify(migrated));
        } else {
          setHistory(parsed);
        }
      } catch (err) {
        console.error('Failed to parse query history', err);
      }
    }
  }, []);

  const addQueryToHistory = (query: string) => {
    if (!query || query.trim() === '') return;

    setHistory((prev) => {
      const existing = prev.find((e) => e.query === query);
      let updated: HistoryEntry[];

      if (existing) {
        // bump run count and move to top (preserve pin)
        updated = [
          { ...existing, runCount: existing.runCount + 1, timestamp: Date.now() },
          ...prev.filter((e) => e.query !== query),
        ];
      } else {
        const newEntry: HistoryEntry = {
          query,
          timestamp: Date.now(),
          runCount: 1,
          pinned: false,
        };
        // keep pinned entries, trim unpinned to MAX_HISTORY
        const pinned = prev.filter((e) => e.pinned);
        const unpinned = prev.filter((e) => !e.pinned);
        updated = [...pinned, newEntry, ...unpinned].slice(0, MAX_HISTORY);
      }

      localStorage.setItem(HISTORY_KEY, JSON.stringify(updated));
      return updated;
    });
  };

  const togglePin = (query: string) => {
    setHistory((prev) => {
      const updated = prev.map((e) =>
        e.query === query ? { ...e, pinned: !e.pinned } : e
      );
      // pinned entries float to top
      const sorted = [
        ...updated.filter((e) => e.pinned),
        ...updated.filter((e) => !e.pinned),
      ];
      localStorage.setItem(HISTORY_KEY, JSON.stringify(sorted));
      return sorted;
    });
  };

  const clearHistory = () => {
    localStorage.removeItem(HISTORY_KEY);
    setHistory([]);
  };

  const exportHistory = () => {
    const blob = new Blob([JSON.stringify(history, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logpulse-history-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return { history, addQueryToHistory, togglePin, clearHistory, exportHistory };
}