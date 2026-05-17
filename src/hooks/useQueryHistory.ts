import { useState, useEffect } from 'react';

const HISTORY_KEY = 'logpulse_query_history';
const MAX_HISTORY = 20;

export function useQueryHistory() {
  const [history, setHistory] = useState<string[]>([]);

  // Load history from localStorage on initial render
  useEffect(() => {
    const storedHistory = localStorage.getItem(HISTORY_KEY);
    if (storedHistory) {
      try {
        setHistory(JSON.parse(storedHistory));
      } catch (error) {
        console.error("Failed to parse query history", error);
      }
    }
  }, []);

  // Add a new query to history
  const addQueryToHistory = (query: string) => {
    if (!query || query.trim() === '') return;

    setHistory((prevHistory) => {
      // Remove duplicate if it already exists, then add to top
      const filteredHistory = prevHistory.filter((q) => q !== query);
      const newHistory = [query, ...filteredHistory].slice(0, MAX_HISTORY);
      
      localStorage.setItem(HISTORY_KEY, JSON.stringify(newHistory));
      return newHistory;
    });
  };

  // Clear all history
  const clearHistory = () => {
    localStorage.removeItem(HISTORY_KEY);
    setHistory([]);
  };

  return { history, addQueryToHistory, clearHistory };
}