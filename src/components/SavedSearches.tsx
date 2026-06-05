import { useState, useEffect } from 'react';
import { Bookmark, Star, Trash2, Plus, Search, Clock, Play, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { toast } from 'sonner';

interface SavedSearch {
  id: string;
  name: string;
  query: string;
  timeRange: string;
  isFavorite: boolean;
  createdAt: string;
  lastUsed?: string;
  useCount: number;
}

interface SavedSearchesProps {
  onExecuteSearch: (query: string, timeRange: string) => void;
  currentQuery?: string;
  currentTimeRange?: string;
}

const STORAGE_KEY = 'lokiclone-saved-searches';

export function SavedSearches({ onExecuteSearch, currentQuery, currentTimeRange }: SavedSearchesProps) {
  const [searches, setSearches] = useState<SavedSearch[]>([]);
  const [isAdding, setIsAdding] = useState(false);
  const [newName, setNewName] = useState('');
  const [searchFilter, setSearchFilter] = useState('');

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        setSearches(JSON.parse(saved));
      } catch {
        setSearches([]);
      }
    }
  }, []);

  const saveToStorage = (newSearches: SavedSearch[]) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(newSearches));
    } catch {
      // Fail silently for quota or privacy mode errors
    }
    setSearches(newSearches);
  };

  const handleSaveSearch = () => {
    if (!newName.trim()) {
      toast.error('Please enter a name for your search');
      return;
    }
    if (!currentQuery) {
      toast.error('No query to save');
      return;
    }

    const newSearch: SavedSearch = {
      id: Date.now().toString(),
      name: newName.trim(),
      query: currentQuery,
      timeRange: currentTimeRange || '1h',
      isFavorite: false,
      createdAt: new Date().toISOString(),
      useCount: 0,
    };

    saveToStorage([newSearch, ...searches]);
    setNewName('');
    setIsAdding(false);
    toast.success('Search saved');
  };

  const handleExecute = (search: SavedSearch) => {
    const updated = searches.map((s) =>
      s.id === search.id
        ? { ...s, lastUsed: new Date().toISOString(), useCount: s.useCount + 1 }
        : s
    );
    saveToStorage(updated);
    onExecuteSearch(search.query, search.timeRange);
  };

  const handleToggleFavorite = (id: string) => {
    const updated = searches.map((s) =>
      s.id === id ? { ...s, isFavorite: !s.isFavorite } : s
    );
    saveToStorage(updated);
  };

  const handleDelete = (id: string) => {
    const updated = searches.filter((s) => s.id !== id);
    saveToStorage(updated);
    toast.success('Search deleted');
  };

  const filteredSearches = searches.filter(
    (s) =>
      s.name.toLowerCase().includes(searchFilter.toLowerCase()) ||
      s.query.toLowerCase().includes(searchFilter.toLowerCase())
  );

  const favoriteSearches = filteredSearches.filter((s) => s.isFavorite);
  const recentSearches = filteredSearches
    .filter((s) => !s.isFavorite)
    .sort((a, b) => (b.lastUsed || b.createdAt).localeCompare(a.lastUsed || a.createdAt));

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Bookmark className="h-4 w-4 text-primary" />
            <h3 className="font-medium">Saved Searches</h3>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsAdding(!isAdding)}
            className="h-7 px-2"
          >
            {isAdding ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          </Button>
        </div>

        {/* Save New Search Form */}
        {isAdding && (
          <div className="space-y-2 p-3 bg-muted rounded-lg mb-3">
            <p className="text-xs text-muted-foreground">Save current query:</p>
            <code className="text-xs font-mono text-primary block truncate">
              {currentQuery || 'No query entered'}
            </code>
            <div className="flex gap-2">
              <Input
                placeholder="Search name..."
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="h-8 text-sm"
                onKeyDown={(e) => e.key === 'Enter' && handleSaveSearch()}
              />
              <Button size="sm" onClick={handleSaveSearch} className="h-8">
                Save
              </Button>
            </div>
          </div>
        )}

        {/* Search Filter */}
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            placeholder="Filter searches..."
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="h-8 pl-8 text-sm"
          />
        </div>
      </div>

      {/* Searches List */}
      <div className="flex-1 overflow-auto scrollbar-thin">
        {searches.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full p-4 text-center">
            <Bookmark className="h-8 w-8 text-muted-foreground/30 mb-2" />
            <p className="text-sm text-muted-foreground">No saved searches yet</p>
            <p className="text-xs text-muted-foreground/70">
              Save your frequent queries for quick access
            </p>
          </div>
        ) : (
          <div className="p-2 space-y-4">
            {/* Favorites Section */}
            {favoriteSearches.length > 0 && (
              <div>
                <div className="flex items-center gap-1.5 px-2 mb-2">
                  <Star className="h-3 w-3 text-warning fill-warning" />
                  <span className="text-xs font-medium text-muted-foreground">Favorites</span>
                </div>
                <div className="space-y-1">
                  {favoriteSearches.map((search) => (
                    <SearchItem
                      key={search.id}
                      search={search}
                      onExecute={handleExecute}
                      onToggleFavorite={handleToggleFavorite}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* Recent Section */}
            {recentSearches.length > 0 && (
              <div>
                <div className="flex items-center gap-1.5 px-2 mb-2">
                  <Clock className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground">Recent</span>
                </div>
                <div className="space-y-1">
                  {recentSearches.map((search) => (
                    <SearchItem
                      key={search.id}
                      search={search}
                      onExecute={handleExecute}
                      onToggleFavorite={handleToggleFavorite}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

interface SearchItemProps {
  search: SavedSearch;
  onExecute: (search: SavedSearch) => void;
  onToggleFavorite: (id: string) => void;
  onDelete: (id: string) => void;
}

function SearchItem({ search, onExecute, onToggleFavorite, onDelete }: SearchItemProps) {
  return (
    <div className="group relative p-2 rounded-lg hover:bg-secondary/50 transition-colors">
      <div className="flex items-start gap-2">
        <button
          onClick={() => onToggleFavorite(search.id)}
          className="mt-0.5 text-muted-foreground hover:text-warning transition-colors"
        >
          <Star
            className={`h-3.5 w-3.5 ${
              search.isFavorite ? 'text-warning fill-warning' : ''
            }`}
          />
        </button>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium truncate">{search.name}</span>
            <span className="text-xs text-muted-foreground font-mono">{search.timeRange}</span>
          </div>
          <code className="text-xs font-mono text-muted-foreground block truncate">
            {search.query}
          </code>
          <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground/70">
            <span>Used {search.useCount}×</span>
            {search.lastUsed && (
              <span>
                Last: {new Date(search.lastUsed).toLocaleDateString()}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => onExecute(search)}
            className="p-1.5 rounded bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
            title="Run search"
          >
            <Play className="h-3 w-3" />
          </button>
          <button
            onClick={() => onDelete(search.id)}
            className="p-1.5 rounded bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors"
            title="Delete"
          >
            <Trash2 className="h-3 w-3" />
          </button>
        </div>
      </div>
    </div>
  );
}
